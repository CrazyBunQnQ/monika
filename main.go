package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"monika/internal/agent"
	"monika/internal/api"
	"monika/internal/bootstrap"
	config2 "monika/internal/config"
	"monika/internal/dap"
	"monika/internal/dbdiscovery"
	"monika/internal/memory"
	"monika/internal/permission"
	"monika/internal/prompt"
	"monika/internal/tool"
	"monika/internal/tool/builtin"
	"monika/internal/update"
	engine2 "monika/pkg/engine"
	"monika/pkg/modelsdev"

	_ "monika/internal/engines/mcp"
	_ "monika/internal/engines/provider/openai"
	_ "monika/internal/engines/skill"

	_ "monika/pkg/dbdriver/mongo"
	_ "monika/pkg/dbdriver/mysql"
	_ "monika/pkg/dbdriver/postgres"
	_ "monika/pkg/dbdriver/redis"
	_ "monika/pkg/dbdriver/sqlite"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed frontend/dist
var embeddedAssets embed.FS

//go:embed winres/icon.png
var iconPNG []byte

func main() {
	// Clean up leftover from previous update.
	if exe, err := os.Executable(); err == nil {
		update.CleanupOld(filepath.Dir(exe))
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot determine home directory:", err)
		os.Exit(1)
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot determine working directory:", err)
		os.Exit(1)
	}
	workspaceRoot := memory.ResolveWorkspaceRoot(cwd)

	// Refresh models.dev catalog (background, non-blocking).
	go func() {
		if err := modelsdev.Refresh(home); err != nil {
			fmt.Fprintf(os.Stderr, "[monika] models.dev refresh: %v\n", err)
		}
	}()

	ctx := context.Background()
	pr, err := bootstrap.InitProvider(ctx, home, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Sync models.dev model limits into config.
	// New models from models.dev are added with Enabled=false.
	syncModelsDev(home, &pr.Config)

	registry := tool.NewRegistry()

	// Create a standalone tsBridge for tree-sitter IPC (uses application.Get(), no app ref needed).
	tsBridge := api.NewTSBridge()
	tsQueryFn := tsBridge.QueryFunc()
	builtin.RegisterDefaults(registry, cwd, home, builtin.TSQueryFunc(tsQueryFn))
	builtin.RegisterLSP(registry, cwd, pr.Config.LSP.Servers, pr.Config.Formatters)
	builtin.WireLSPHooks(registry)

	taskStore := builtin.NewTaskStore(nil)
	builtin.RegisterTasks(registry, taskStore)

	var dbMgr *api.DBManager
	cache, err := dbdiscovery.LoadCache(workspaceRoot)
	if err != nil {
		cache, _ = dbdiscovery.Scan(workspaceRoot)
	}
	if cache != nil && len(cache.Connections) > 0 {
		dbMgr = api.NewDBManager(workspaceRoot)
		dbMgr.Init(cache)
		dbMgr.StartSchemaBackground()
		builtin.RegisterDatabase(registry, dbMgr)
	}

	// Create MCP registry and connect servers asynchronously
	mcpRegistry := engine2.NewMCPRegistry()
	mcpEng, err := engine2.EngineByID("mcp")
	if err == nil {
		if mcp, ok := mcpEng.(engine2.MCPEngine); ok {
			_ = mcp.Init(ctx, nil)
			if len(pr.Config.MCP.Servers) > 0 {
				go func() {
					for _, srv := range pr.Config.MCP.Servers {
						cfg := engine2.MCPServerConfig{
							ID: srv.ID, Type: srv.Type, Command: srv.Command,
							Args: srv.Args, Env: srv.Env, URL: srv.URL, Headers: srv.Headers,
						}
						mcpCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						conn, err := mcp.ConnectServer(mcpCtx, cfg)
						cancel()
						if err != nil {
							fmt.Fprintf(os.Stderr, "[monika] MCP server %q connect failed: %v\n", srv.ID, err)
							continue
						}
						tools, err := conn.ListTools(context.Background())
						if err != nil {
							fmt.Fprintf(os.Stderr, "[monika] MCP server %q list tools: %v\n", srv.ID, err)
							continue
						}
						meta := conn.ServerMeta()
						mcpRegistry.AddServer(meta, conn, tools)
						fmt.Fprintf(os.Stderr, "[monika] MCP server %q connected (%d tools)\n", srv.ID, len(tools))
					}
				}()
			}
		}
	}

	// Initialize skill engine
	var skEngine engine2.SkillEngine
	skillEng, err := engine2.EngineByID("skill")
	if err == nil {
		if sk, ok := skillEng.(engine2.SkillEngine); ok {
			skEngine = sk
		}
	}

	// Register skill tool for on-demand skill loading
	var appGetProjectPath func() string
	getCwd := func() string {
		if appGetProjectPath != nil {
			return appGetProjectPath()
		}
		return cwd
	}
	if skEngine != nil {
		builtin.RegisterSkillTool(registry, skEngine, home, getCwd, &pr.Config)
		builtin.RegisterSkillSearchTool(registry, skEngine, home, getCwd, &pr.Config)
	}

	kbStore, err := memory.NewKBStore(home, workspaceRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[monika] kb init failed: %v\n", err)
	} else {
		builtin.RegisterMemory(registry, kbStore)
	}

	// Register skill management tools (install/uninstall) with deferred App binding
	var skillInstallFn func(url string, scope string) ([]string, error)
	var skillUninstallFn func(name string) error
	builtin.RegisterSkillManagement(registry,
		func(url string, scope string) ([]string, error) {
			if skillInstallFn == nil {
				return nil, fmt.Errorf("skill installation not available yet")
			}
			return skillInstallFn(url, scope)
		},
		func(name string) error {
			if skillUninstallFn == nil {
				return fmt.Errorf("skill uninstallation not available yet")
			}
			return skillUninstallFn(name)
		},
	)

	application.RegisterEvent[api.StreamEvent]("stream")
	application.RegisterEvent[api.TSRequest]("ts:request")
	application.RegisterEvent[update.UpdateInfo]("update-available")
	application.RegisterEvent[string]("branch-changed")
	application.RegisterEvent[[]api.NotificationData]("tray-notifications-changed")

	ps := agent.PromptForModel(pr.Model)
	shellPath, _ := builtin.ResolveShell()
	shellName := filepath.Base(shellPath)
	if shellName == "" {
		shellName = "sh"
	}
	shellName += " (mvdan/sh)"
	systemParts := []string{
		fmt.Sprintf("OS Version: %s\nWorking directory: {{WorkingDirectory}}\nShell: %s", runtime.GOOS, shellName),
		ps.Identity,
		`## Knowledge Base (Memory)

You have a self-evolving knowledge base that persists across sessions. It contains
past lessons (bugs, root causes, solutions), topics (architecture, patterns), and
core knowledge (your preferences, project conventions, persistent facts).

After completing a task, if you learned something worth keeping, save it with
memory_write or memory_update. Use memory_search to find relevant past experience.
Tools: memory_search, memory_read, memory_write, memory_update, memory_index.
Memory types: lessons (bugs/causes/solutions), topics (architecture/patterns),
knowledge (preferences/constraints/persistent facts).`,
		ps.ToolUsage,
		ps.Planning,
		ps.CodeQuality,
		ps.ResponseStyle,
		ps.SafetyBoundaries,
		ps.Remember,
	}
	if p := loadSystemPrompt(cwd); p != "" {
		wrapped := `<project_rules>
The content below is your PROJECT RULES from AGENTS.md. These rules are NON-NEGOTIABLE — they represent the project's architectural decisions, coding conventions, and hard constraints. You MUST follow them as strictly as the rules above. Violating project rules is as serious as violating core safety boundaries.

` + p + `
</project_rules>`
		systemParts = append(systemParts, wrapped)
	}
	dynamicCapabilities := `
## Dynamic Capabilities

Skills, MCP tools, and LSP servers are discoverable at runtime — their availability
changes as you install or configure them. Always search before assuming:

- **skill_search(query)** — fuzzy search installed skills by name or description
- **mcp_search(query)** — fuzzy search MCP tools by name, server, or capability
- **lsp_list** — list currently available language servers

	Once you identify the right skill or tool, load it with **skill** or call the MCP tool directly.`

	systemParts = append(systemParts, strings.TrimSpace(dynamicCapabilities))

	systemPrompt := strings.Join(systemParts, "\n\n")

	// One-time {{WorkingDirectory}} replacement at startup so the system
	// prompt stays fully static (better for prompt caching).
	normalized := strings.ReplaceAll(cwd, "\\", "/")
	systemPrompt = strings.ReplaceAll(systemPrompt, "{{WorkingDirectory}}", normalized)

	// Append a compact memory index to the (static) system prompt so the LLM
	// can discover existing memories and proactively memory_read relevant ones.
	// Computed once at App startup; new memories written mid-session are handled
	// by the memory queue which injects <memory-update> blocks into user messages.
	if kbStore != nil {
		memIndex, _ := kbStore.BuildIndex(memory.ScopeAuto, 50)
		if memIndex != "" {
			systemPrompt += "\n\n# Memory Index\n\nSaved memories from previous sessions. Use memory_read(path) when one looks relevant.\n\n" + memIndex
		}
	}
	loopOpts := []agent.LoopOption{
		agent.WithProjectDir(cwd),
		agent.WithModel(pr.Model),
		agent.WithSystemPrompt(systemPrompt),
		agent.WithHomeDir(home),
	}
	// Memory auto-recall + immediate-write visibility.
	// memQueue is always wired (memory_write queues regardless of kbStore);
	// memSearchFn requires kbStore to perform a real search.
	memQueue := agent.NewMemoryQueue()
	loopOpts = append(loopOpts, agent.WithMemQueue(memQueue))

	// Database schema availability hint (one-shot, per-message for now).
	if dbMgr != nil && dbMgr.DefaultConnection() != "" {
		loopOpts = append(loopOpts, agent.WithDBSchemaNote(
			"This project has connected databases. Use db_schema to inspect their structure."))
	}
	if kbStore != nil {
		memSearchFn := func(query string) string {
			results, err := kbStore.Search(query, memory.ScopeAuto, 3)
			if err != nil || len(results) == 0 {
				return ""
			}
			var b strings.Builder
			for _, r := range results {
				fmt.Fprintf(&b, "- **%s** [%s] path: %s\n  snippet: %s\n",
					r.Title, r.Category, r.Path, r.Snippet)
			}
			return b.String()
		}
		loopOpts = append(loopOpts, agent.WithMemSearchFn(memSearchFn))
	}

	// Wire permission pipeline
	rules, _ := permission.LoadRules(home, cwd)
	hardRuleEngine := permission.NewHardRuleEngine(rules, cwd)
	pipeline := permission.NewPipeline(permission.Auto, hardRuleEngine, nil)
	pipeline.SetProject(home, cwd)
	loopOpts = append(loopOpts, agent.WithPermissionPipeline(pipeline))

	// MCP registry
	loopOpts = append(loopOpts, agent.WithMCPRegistry(mcpRegistry))

	// Register runtime discovery tools
	builtin.RegisterMCPSearchTool(registry, mcpRegistry)
	builtin.RegisterLSPListTool(registry)

	// Build agent registry with builtin agents
	exploreBasePrompt := systemPrompt + "\n\n" + prompt.ExplorePrompt
	planBasePrompt := systemPrompt + "\n\n" + prompt.PlanPrompt

	agentRegistry := agent.NewAgentRegistry([]agent.Agent{
		{
			Name:         "general",
			Description:  "General-purpose agent for research and multi-step tasks",
			SystemPrompt: systemPrompt,
		},
		{
			Name:         "explore",
			Description:  "Fast agent specialized for exploring codebases",
			SystemPrompt: exploreBasePrompt,
		},
		{
			Name:         "plan",
			Description:  "Read-only planning agent that analyzes and plans before implementation",
			SystemPrompt: planBasePrompt,
		},
		{
			Name:         "compaction",
			Description:  "Internal — conversation summarizer",
			SystemPrompt: agent.CompactionPrompt,
			Hidden:       true,
		},
	})
	agentRegistry.MergeConfig(pr.Config.Agents)

	// Resolve default provider engine for task runner
	defaultProvider, ok := pr.Providers[pr.Config.ModelProvider]
	if !ok {
		for _, p := range pr.Providers {
			defaultProvider = p
			break
		}
	}

	// Create task runner for subagent dispatch.
	var appService *api.App
	taskRunner := agent.NewTaskRunner(agentRegistry, defaultProvider, pr.Providers, registry,
		func(task agent.SubTask, agentName string) {
			if appService != nil {
				appService.SaveChildSession(task.SessionID, &agent.ChildSession{
					Agent:    agentName,
					Title:    task.Description,
					ParentID: task.ParentID,
					Messages: []engine2.ChatMessage{
						{Role: "user", Content: task.Prompt},
					},
				})
			}
		},
		func(task agent.SubTask, child *agent.ChildSession) {
			if appService != nil {
				appService.SaveChildSession(task.SessionID, child)
				appService.SaveChildSessionToDisk(task.SessionID, child)
			}
		})

	builtin.RegisterAskUser(registry)

	// Register SpawnAgent tool
	builtin.RegisterSpawnAgent(registry, agentRegistry,
		func(ctx context.Context, task agent.SubTask) <-chan agent.Event {
			return taskRunner.Dispatch(ctx, task, nil)
		},
		func(parentID, childID string) {
			if appService != nil {
				appService.PendingChildSession(parentID, childID)
			}
		})

	var taskStoreAccessor api.TaskStoreAccessor
	if accessor, ok := taskStore.(api.TaskStoreAccessor); ok {
		taskStoreAccessor = accessor
	}

	appService = api.NewApp(home, cwd, pr.Config, pr.Providers, pr.Model, registry, loopOpts, taskStoreAccessor, agentRegistry, taskRunner, mcpRegistry, kbStore)

	appService.InitTSBridge(tsBridge)
	// Background memory maintenance: decay (archive/delete stale) + review (conflict/upgrade detection).
	if kbStore != nil {
		policy := memory.DefaultDecayPolicy()
		go func() {
			// Run decay on startup for both scopes if 24h+ since last run.
			for _, scope := range []string{memory.ScopeProject, memory.ScopeGlobal} {
				if time.Since(kbStore.LastDecayTime(scope)) >= 24*time.Hour {
					archived, deleted, _ := kbStore.RunDecay(scope, policy)
					if archived+deleted > 0 {
						fmt.Fprintf(os.Stderr, "[monika] memory decay (%s): %d archived, %d deleted\n", scope, archived, deleted)
					}
					kbStore.SetLastDecayTime(scope)
				}
			}
			// Then re-check every 24 hours.
			ticker := time.NewTicker(24 * time.Hour)
			defer ticker.Stop()
			for range ticker.C {
				for _, scope := range []string{memory.ScopeProject, memory.ScopeGlobal} {
					archived, deleted, _ := kbStore.RunDecay(scope, policy)
					if archived+deleted > 0 {
						fmt.Fprintf(os.Stderr, "[monika] memory decay (%s): %d archived, %d deleted\n", scope, archived, deleted)
					}
					kbStore.SetLastDecayTime(scope)
				}
			}
		}()

		// Periodic review uses the default provider, wrapped as a ReviewLLM.
		if defaultProvider != nil {
			reviewLLM := &memory.GoLLMAdapter{
				ChatFn: func(ctx context.Context, systemPrompt, userMessage string) (string, error) {
					msgs := []engine2.ChatMessage{{Role: "system", Content: systemPrompt}}
					if userMessage != "" {
						msgs = append(msgs, engine2.ChatMessage{Role: "user", Content: userMessage})
					}
					events, err := defaultProvider.StreamChat(ctx, engine2.ChatRequest{
						Provider: defaultProvider.ID(),
						Model:    pr.Model,
						Messages: msgs,
					})
					if err != nil {
						return "", err
					}
					var sb strings.Builder
					for ev := range events {
						if ev.Kind == engine2.EventContentDelta && ev.Text != "" {
							sb.WriteString(ev.Text)
						}
					}
					return sb.String(), nil
				},
			}
			go func() {
				time.Sleep(10 * time.Second) // let the app settle before the first review
				for _, scope := range []string{memory.ScopeProject, memory.ScopeGlobal} {
					kbStore.AutoReviewIfNeeded(context.Background(), reviewLLM, scope)
				}
			}()
		}
	}
	appGetProjectPath = appService.GetProjectPath
	if dbMgr != nil {
		appService.SetDBManager(dbMgr)
	}

	// Wire DAP debugger
	dapManager := dap.NewDapManager(cwd)
	appService.SetDapManager(dapManager)
	builtin.RegisterDebug(registry, dapManager)

	// Wire background task manager to bash and background_task tools
	for _, name := range []string{"bash", "background_task"} {
		if t, ok := registry.Get(name); ok {
			if setter, ok := t.(interface{ SetBgManager(builtin.BgManager) }); ok {
				setter.SetBgManager(appService.BgTaskManager())
			}
		}
	}

	// Wire skill management callbacks to App
	skillInstallFn = func(url string, scope string) ([]string, error) {
		args, _ := json.Marshal(map[string]string{"url": url, "scope": scope})
		return appService.InstallSkillFromURL(args)
	}
	skillUninstallFn = func(name string) error {
		args, _ := json.Marshal(map[string]string{"Name": name})
		return appService.UninstallSkill(args)
	}

	// Wire MCP management callbacks to App
	mcpSaveFn := func(args json.RawMessage) error {
		return appService.SaveMCPServer(args)
	}
	mcpDeleteFn := func(args json.RawMessage) error {
		return appService.DeleteMCPServer(args)
	}
	mcpReconnectFn := func(args json.RawMessage) ([]string, error) {
		return appService.ReconnectMCPServer(args)
	}
	mcpListFn := func() []builtin.MCPServerInfo {
		servers := appService.ListMCPServers()
		result := make([]builtin.MCPServerInfo, len(servers))
		for i, s := range servers {
			result[i] = builtin.MCPServerInfo{
				ID: s.ID, Type: s.Type, Command: s.Command,
				Args: s.Args, Env: s.Env, URL: s.URL, Status: s.Status,
			}
		}
		return result
	}
	builtin.RegisterMCPManagement(registry, mcpSaveFn, mcpDeleteFn, mcpReconnectFn, mcpListFn)

	pipeline.SetConfirmUI(appService)
	appService.SetPipeline(pipeline)

	// Wire task change callback so TaskStore mutations push events to the frontend
	builtin.SetTaskStoreCallback(taskStore, func(sessionID string, tasks []tool.Task) {
		taskItems := make([]agent.TaskItem, len(tasks))
		for i, t := range tasks {
			taskItems[i] = agent.TaskItem{
				ID: t.ID, Subject: t.Subject, Description: t.Description,
				Status: t.Status, BlockedBy: t.BlockedBy,
			}
		}
		appService.EmitTaskEvent(sessionID, taskItems)
	})

	appService.AppendLoopOption(agent.WithAskUserFunc(func(ctx context.Context, args tool.AskUserArgs) (string, error) {
		sessionID := tool.SessionIDFromContext(ctx)
		return appService.AskUser(ctx, sessionID, args.Question, args.Title, args.Options)
	}))

	assets, err := fs.Sub(embeddedAssets, "frontend/dist")
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to extract embedded assets:", err)
		os.Exit(1)
	}

	app := application.New(application.Options{
		Name:        "monika",
		Description: "Agentic coding editor",
		Services: []application.Service{
			application.NewService(appService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
			Middleware: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.HasPrefix(r.URL.Path, "/__local__/") {
						relPath := strings.TrimPrefix(r.URL.Path, "/__local__/")
						relPath = strings.TrimPrefix(relPath, "/")
						pp := appService.GetProjectPath()
						if pp == "" {
							http.NotFound(w, r)
							return
						}
						absPath := filepath.Join(pp, filepath.FromSlash(relPath))
						absPath = filepath.Clean(absPath)
						if !strings.HasPrefix(absPath, filepath.Clean(pp)) {
							http.NotFound(w, r)
							return
						}
						http.ServeFile(w, r, absPath)
						return
					}
					next.ServeHTTP(w, r)
				})
			},
		},
	})

	mainWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Monika",
		Width:            1400,
		Height:           900,
		MinWidth:         900,
		MinHeight:        600,
		Frameless:        true,
		StartState:       application.WindowStateMaximised,
		Hidden:           true,
		BackgroundColour: application.NewRGB(24, 24, 30),
	})

	trayMgr := api.NewTrayManager(app, mainWindow, iconPNG)
	if err := trayMgr.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "[monika] tray init failed: %v\n", err)
	} else {
		// Only intercept close if tray is active
		mainWindow.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
			mainWindow.Hide()
			e.Cancel()
		})
		appService.SetTrayManager(trayMgr)
	}

	// Show the main window once WebView2 runtime is initialised.
	// Hidden: true prevents a title-bar flash on startup.
	mainWindow.OnWindowEvent(events.Common.WindowRuntimeReady, func(e *application.WindowEvent) {
		mainWindow.Show()
	})

	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func syncModelsDev(home string, cfg *config2.Config) {
	catalog, err := modelsdev.Catalog(home)
	if err != nil {
		return
	}

	type modelInfo struct {
		Context  int64
		Output   int64
		Provider string
	}
	modelIndex := make(map[string]modelInfo, 4096)
	for providerID, p := range catalog {
		for modelID, md := range p.Models {
			if md.Limit.Context > 0 {
				modelIndex[modelID] = modelInfo{
					Context:  md.Limit.Context,
					Output:   md.Limit.Output,
					Provider: providerID,
				}
			}
		}
	}

	changed := false

	// Enrich already-configured providers from models.dev: fill missing limits and
	// auto-append new catalog models (disabled). Providers themselves are never auto-added —
	// users must explicitly add providers through the Settings UI.
	if len(cfg.ModelProviders) > 0 {
		for key, pc := range cfg.ModelProviders {
			existingIDs := make(map[string]bool, len(pc.Models))
			for i := range pc.Models {
				m := &pc.Models[i]
				existingIDs[m.ID] = true
				if info, ok := modelIndex[m.ID]; ok {
					if m.ContextLimit == 0 && info.Context > 0 {
						m.ContextLimit = info.Context
						changed = true
					}
					if m.OutputLimit == 0 && info.Output > 0 {
						m.OutputLimit = info.Output
						changed = true
					}
					if m.DisplayName == "" {
						m.DisplayName = m.ID
						changed = true
					}
				}
			}

			// Auto-detect models.dev provider if not set.
			if pc.ModelsDevProvider == "" {
				for modelID, info := range modelIndex {
					if existingIDs[modelID] {
						pc.ModelsDevProvider = info.Provider
						changed = true
						break
					}
				}
				if pc.ModelsDevProvider == "" {
					normalized := normalizeID(key)
					for pID := range catalog {
						if normalizeID(pID) == normalized {
							pc.ModelsDevProvider = pID
							changed = true
							break
						}
					}
				}
			}

			// Auto-append new catalog models for this provider (disabled by default).
			// Users opt in by enabling them in the Settings UI.
			if devProv, ok := catalog[pc.ModelsDevProvider]; ok {
				newIDs := make([]string, 0, 16)
				for modelID, md := range devProv.Models {
					if !existingIDs[modelID] && md.Limit.Context > 0 {
						newIDs = append(newIDs, modelID)
					}
				}
				sort.Strings(newIDs)
				for _, modelID := range newIDs {
					md := devProv.Models[modelID]
					name := md.Name
					if name == "" {
						name = modelID
					}
					pc.Models = append(pc.Models, config2.ModelEntry{
						ID:           modelID,
						DisplayName:  name,
						ContextLimit: md.Limit.Context,
						OutputLimit:  md.Limit.Output,
						Enabled:      false,
					})
					existingIDs[modelID] = true
					changed = true
				}
			}

			cfg.ModelProviders[key] = pc
		}
	}

	if !changed {
		return
	}

	// Write back updated config.
	configPath := filepath.Join(home, ".monika", "config.json")
	_ = os.MkdirAll(filepath.Dir(configPath), 0755)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(configPath, data, 0600)
}

func loadSystemPrompt(projectDir string) string {
	paths := []string{
		filepath.Join(projectDir, "AGENTS.md"),
		filepath.Join(projectDir, ".monika", "AGENTS.md"),
	}
	for _, p := range paths {
		if data, err := os.ReadFile(p); err == nil {
			return string(data)
		}
	}

	return ""
}

func normalizeID(id string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(id) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}
