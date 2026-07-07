package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"monika/internal/agent"
	"monika/internal/config"
	"monika/internal/dap"
	"monika/internal/lsp"
	"monika/internal/memory"
	"monika/internal/tool"
	"monika/pkg/engine"
)

// TSQueryFunc calls the frontend tree-sitter service.
// Returns (nil, nil) when tree-sitter is unavailable.
type TSQueryFunc func(ctx context.Context, method string, params map[string]any) (json.RawMessage, error)

func RegisterDefaults(r *tool.ToolRegistry, projectDir, homeDir string, tsQuery TSQueryFunc, media MediaCaller) error {
	r.Register(NewFileRead(projectDir, homeDir, tsQuery))
	r.Register(NewFileWrite(projectDir))
	r.Register(NewFileEdit(projectDir))
	r.Register(NewFilePatch(projectDir))
	r.Register(NewFileList(projectDir))
	r.Register(NewGlob(projectDir))
	r.Register(NewGrep(projectDir, tsQuery))
	sh, err := NewBash(projectDir)
	if err != nil {
		return err
	}
	r.Register(sh)
	bt, err := NewBackgroundTask()
	if err != nil {
		return err
	}
	r.Register(bt)
	r.Register(NewImageUnderstand(media))
	r.Register(NewVideoUnderstand(media))
	r.Register(NewPdfUnderstand(media))
	r.Register(NewAudioUnderstand(media))
	return nil

}

// RegisterLSP registers the LSP tool for code intelligence.
func RegisterLSP(r *tool.ToolRegistry, projectDir string, lspServers map[string]lsp.ServerConfig, formatters map[string]lsp.FormatterConfig) error {
	t, err := NewLSPTool(projectDir, lspServers, formatters)
	if err != nil {
		return err
	}
	r.Register(t)
	return nil
}

// WireLSPHooks connects LSP diagnostics and symbols hooks to file tools.
// Must be called after both RegisterDefaults and RegisterLSP.
func WireLSPHooks(r *tool.ToolRegistry) {
	lspTool, ok := r.Get("lsp")
	if !ok {
		return
	}

	diagFunc := func(ctx context.Context, filePath string) string {
		if filePath == "" {
			return "\n\n--- LSP Diagnostics ---\n(empty file path)"
		}

		// Skip diagnostics entirely for file types with no LSP server configured.
		if checker, ok := lspTool.(interface {
			HasServerForFile(string) bool
		}); ok {
			if !checker.HasServerForFile(filePath) {
				return ""
			}
		}

		// Wait for LSP server to be ready before querying diagnostics.
		ready := false
		if checker, ok := lspTool.(interface {
			ReadyForFile(context.Context, string) bool
		}); ok {
			deadline := time.Now().Add(5 * time.Second)
			for !checker.ReadyForFile(ctx, filePath) {
				if time.Now().After(deadline) {
					return "\n\n--- LSP Diagnostics ---\nLSP server not ready (timeout)"
				}
				select {
				case <-ctx.Done():
					return "\n\n--- LSP Diagnostics ---\nLSP: context cancelled"
				case <-time.After(300 * time.Millisecond):
				}
			}
			ready = true
		}
		if !ready {
			return "\n\n--- LSP Diagnostics ---\nLSP server not available"
		}
		// Optional: format file via LSP before diagnostics
		if formatter, ok := lspTool.(interface {
			FormatContent(context.Context, string) (string, error)
		}); ok {
			formatter.FormatContent(ctx, filePath)
		}
		// Send didSave to trigger LSP re-analysis (like OMP's notifyFileSaved)
		if saver, ok := lspTool.(interface {
			NotifySavedForFile(context.Context, string) error
		}); ok {
			saver.NotifySavedForFile(ctx, filePath)
		}

		// Run diagnostics. The action internally waits for the server
		// to publish updated diagnostics before returning.
		diagArgs, _ := json.Marshal(map[string]string{"action": "diagnostics", "file": filePath})
		diagResult, err := lspTool.Execute(ctx, diagArgs)
		if err != nil {
			return fmt.Sprintf("\n\n--- LSP Diagnostics ---\n(error running diagnostics: %s)", err)
		}
		if diagResult.IsError {
			return fmt.Sprintf("\n\n--- LSP Diagnostics ---\n(error: %s)", diagResult.Content)
		}

		if diagResult.Content == "" {
			return "\n\n--- LSP Diagnostics ---\nLSP: no diagnostics returned"
		}

		// Always respond — never silently return empty (OMP principle).
		hasError := strings.Contains(diagResult.Content, "Error")
		hasWarning := strings.Contains(diagResult.Content, "Warning")

		var sb strings.Builder
		sb.WriteString("\n\n--- LSP Diagnostics ---\n")
		sb.WriteString(diagResult.Content)

		if hasError {
			caArgs, _ := json.Marshal(map[string]string{"action": "code_actions", "file": filePath})
			caResult, caErr := lspTool.Execute(ctx, caArgs)
			if caErr == nil && caResult.Content != "" && !strings.Contains(caResult.Content, "No code") {
				sb.WriteString("\n--- Available Code Actions ---\n")
				sb.WriteString(caResult.Content)
			}
		} else if !hasWarning {
			sb.WriteString("\n(no issues)")
		}

		return sb.String()
	}

	symFunc := func(ctx context.Context, filePath string) string {
		if filePath == "" {
			return ""
		}
		symArgs, _ := json.Marshal(map[string]string{"action": "symbols", "file": filePath})
		symResult, err := lspTool.Execute(ctx, symArgs)
		if err != nil || symResult.IsError {
			return ""
		}
		if symResult.Content == "" || strings.Contains(symResult.Content, "No symbols") {
			return ""
		}
		return "\n\n--- LSP Symbol Outline ---\n" + symResult.Content
	}

	if t, ok := r.Get("file_edit"); ok {
		if fe, ok := t.(interface{ SetDiagFunc(LSPDiagFunc) }); ok {
			fe.SetDiagFunc(diagFunc)
		}
	}
	if t, ok := r.Get("patch"); ok {
		if fp, ok := t.(interface{ SetDiagFunc(LSPDiagFunc) }); ok {
			fp.SetDiagFunc(diagFunc)
		}
	}
	if t, ok := r.Get("file_write"); ok {
		if fw, ok := t.(interface{ SetDiagFunc(LSPDiagFunc) }); ok {
			fw.SetDiagFunc(diagFunc)
		}
	}
	if t, ok := r.Get("file_read"); ok {
		if fr, ok := t.(interface{ SetSymFunc(LSPDiagFunc) }); ok {
			fr.SetSymFunc(symFunc)
		}
	}
}

// RegisterAskUser registers the ask_user tool for user interaction.
func RegisterAskUser(r *tool.ToolRegistry) {
	r.Register(NewAskUser())
}

// RegisterTasks registers the three task planning tools.
// Called separately after TaskStore is created in main.
func RegisterTasks(r *tool.ToolRegistry, store tool.TaskStore) {
	r.Register(NewTaskCreate(store))
	r.Register(NewTaskAppend(store))
	r.Register(NewTaskUpdate(store))
	r.Register(NewTaskList(store))
}

// RegisterSpawnAgent registers the SpawnAgent tool for dispatching subtasks to other agents.
// Called after AgentRegistry and TaskRunner are created in main.
func RegisterSpawnAgent(r *tool.ToolRegistry, registry *agent.AgentRegistry, dispatchFn func(ctx context.Context, task agent.SubTask) <-chan agent.Event, pendingStore func(parentID, childID string)) {
	r.Register(NewSpawnAgent(registry, dispatchFn, pendingStore))
}

// RegisterAgentManagement registers list_agents, create_agent, and delete_agent tools.
// The callbacks are typically wired to App methods.
func RegisterAgentManagement(
	r *tool.ToolRegistry,
	saveFn func(json.RawMessage) error,
	deleteFn func(json.RawMessage) error,
	listFn func() []AgentInfo,
	checkFn func(name string) (isCustom bool, exists bool),
) {
	r.Register(NewAgentListTool(listFn))
	r.Register(NewAgentCreateTool(saveFn))
	r.Register(NewAgentDeleteTool(deleteFn, checkFn))
}

func RegisterSkillTool(r *tool.ToolRegistry, skEng engine.SkillEngine, home string, getCwd func() string, cfg *config.Config) {
	r.Register(NewSkillTool(skEng, home, getCwd, cfg))
}

// RegisterSkillSearchTool registers the skill_search tool for fuzzy discovery.
func RegisterSkillSearchTool(r *tool.ToolRegistry, skEng engine.SkillEngine, home string, getCwd func() string, cfg *config.Config) {
	r.Register(NewSkillSearchTool(skEng, home, getCwd, cfg))
}

// RegisterSkillManagement registers install_skill and uninstall_skill tools.
// The installFn and uninstallFn callbacks are typically wired to App methods.
func RegisterSkillManagement(r *tool.ToolRegistry, installFn func(url string, scope string) ([]string, error), uninstallFn func(name string) error) {
	r.Register(NewSkillInstallTool(installFn))
	r.Register(NewSkillUninstallTool(uninstallFn))
}

// RegisterMCPManagement registers install_mcp_server, uninstall_mcp_server, and list_mcp_servers tools.
// The callbacks are typically wired to App methods.
func RegisterMCPManagement(
	r *tool.ToolRegistry,
	saveFn func(json.RawMessage) error,
	deleteFn func(json.RawMessage) error,
	reconnectFn func(json.RawMessage) ([]string, error),
	listFn func() []MCPServerInfo,
) {
	r.Register(NewMCPInstallTool(saveFn, reconnectFn))
	r.Register(NewMCPUninstallTool(deleteFn))
	r.Register(NewMCPListTool(listFn))
}

// RegisterMCPSearchTool registers the mcp_search tool for fuzzy discovery.
func RegisterMCPSearchTool(r *tool.ToolRegistry, mcpRegistry *engine.MCPRegistry) {
	r.Register(NewMCPSearchTool(mcpRegistry))
}

// RegisterLSPListTool registers the lsp_list tool for querying language server status.
func RegisterLSPListTool(r *tool.ToolRegistry) {
	r.Register(NewLSPListTool(r))
}

// RegisterDebug registers the DAP debug tool for LLM-driven debugging.
func RegisterDebug(r *tool.ToolRegistry, manager *dap.DapManager) {
	r.Register(NewDebugTool(manager))
}

// RegisterDatabase registers db_schema and db_query tools for database interaction.
func RegisterDatabase(r *tool.ToolRegistry, dbMgr DBQuerier) {
	if dbMgr == nil {
		return
	}
	r.Register(NewDBSchema(dbMgr))
	r.Register(NewDBQuery(dbMgr))
}

// RegisterMemory registers the memory search/write/index/reindex tools.
func RegisterMemory(r *tool.ToolRegistry, store *memory.KBStore) {
	r.Register(NewMemorySearch(store))
	r.Register(NewMemoryRead(store))
	r.Register(NewMemoryUpdate(store))
	r.Register(NewMemoryWrite(store))
	r.Register(NewMemoryIndex(store))
	r.Register(NewMemoryReindex(store))
	r.Register(NewMemoryGraph(store))
}
