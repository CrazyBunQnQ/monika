package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"monika/internal/tool"
)

type lspTool struct {
	manager *Manager
}

func NewLSPTool(projectDir string) (tool.Tool, error) {
	m := NewManager(projectDir)
	m.Start()
	return &lspTool{manager: m}, nil
}

func (t *lspTool) Manager() *Manager { return t.manager }

func (t *lspTool) Name() string { return "lsp" }

func (t *lspTool) Description() string {
	return `Language Server Protocol client. Provides code intelligence via LSP servers.

Actions:
- diagnostics: Get diagnostics (errors, warnings) for a file
- definition: Go to definition at a position in a file
- type_definition: Go to type definition at a position
- implementation: Find implementations of an interface or type at a position
- references: Find all references to a symbol at a position
- hover: Get hover information (type docs) at a position
- symbols: Get document symbols (outline) for a file
- code_actions: List available code actions for a range in a file
- rename: Rename a symbol at a position across the workspace
- status: Show configured and running LSP servers

The file path must be absolute or relative to the project directory.
Line and character are 0-based (same as LSP protocol).`
}

func (t *lspTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"diagnostics", "definition", "type_definition", "implementation", "references", "hover", "symbols", "code_actions", "rename", "status"},
				"description": "The LSP action to perform",
			},
			"file": map[string]any{
				"type":        "string",
				"description": "Absolute file path (required for all actions except status)",
			},
			"line": map[string]any{
				"type":        "integer",
				"description": "0-based line number (required for definition, type_definition, implementation, references, hover, rename)",
			},
			"character": map[string]any{
				"type":        "integer",
				"description": "0-based character offset (required for definition, type_definition, implementation, references, hover, rename)",
			},
			"end_line": map[string]any{
				"type":        "integer",
				"description": "0-based end line for range-based actions (code_actions)",
			},
			"end_character": map[string]any{
				"type":        "integer",
				"description": "0-based end character for range-based actions (code_actions)",
			},
			"new_name": map[string]any{
				"type":        "string",
				"description": "New name for the symbol (required for rename action)",
			},
		},
		"required": []string{"action"},
	}
}

func (t *lspTool) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var params struct {
		Action       string `json:"action"`
		File         string `json:"file"`
		Line         int    `json:"line"`
		Character    int    `json:"character"`
		EndLine      int    `json:"end_line"`
		EndCharacter int    `json:"end_character"`
		NewName      string `json:"new_name"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	if params.Action == "status" {
		return tool.ExecutionResult{Content: t.manager.Status()}, nil
	}

	if params.File == "" {
		return tool.ExecutionResult{Content: "file is required for this action", IsError: true}, nil
	}

	filePath := params.File
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(t.manager.workdir, filePath)
	}

	client, _, err := t.manager.ClientForFile(ctx, filePath)
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	uri := fileToURI(filePath)

	_, _, err = t.manager.EnsureFileOpen(ctx, client, filePath)
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	pos := Position{Line: params.Line, Character: params.Character}

	switch params.Action {
	case "diagnostics":
		select {
		case <-ctx.Done():
			return tool.ExecutionResult{Content: ctx.Err().Error(), IsError: true}, nil
		default:
		}
		diags := client.Diagnostics(uri)
		return tool.ExecutionResult{Content: FormatDiagnostics(uri, diags)}, nil

	case "definition":
		locs, err := client.Definition(ctx, uri, pos)
		if err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
		}
		return tool.ExecutionResult{Content: FormatLocations(locs)}, nil

	case "type_definition":
		locs, err := client.TypeDefinition(ctx, uri, pos)
		if err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
		}
		return tool.ExecutionResult{Content: FormatLocations(locs)}, nil

	case "implementation":
		locs, err := client.Implementation(ctx, uri, pos)
		if err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
		}
		return tool.ExecutionResult{Content: FormatLocations(locs)}, nil

	case "references":
		locs, err := client.References(ctx, uri, pos)
		if err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
		}
		return tool.ExecutionResult{Content: FormatLocations(locs)}, nil

	case "hover":
		hover, err := client.Hover(ctx, uri, pos)
		if err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
		}
		if hover == nil {
			return tool.ExecutionResult{Content: "No hover information available."}, nil
		}
		return tool.ExecutionResult{Content: hover.Contents.Value}, nil

	case "symbols":
		syms, err := client.DocumentSymbols(ctx, uri)
		if err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
		}
		return tool.ExecutionResult{Content: FormatSymbols(syms, "")}, nil

	case "code_actions":
		r := Range{
			Start: pos,
			End:   Position{Line: params.EndLine, Character: params.EndCharacter},
		}
		if r.End.Line == 0 && r.End.Character == 0 {
			r.End = r.Start
		}
		diags := client.Diagnostics(uri)
		actions, err := client.CodeActions(ctx, uri, r, diags)
		if err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
		}
		return tool.ExecutionResult{Content: formatCodeActions(actions)}, nil

	case "rename":
		if params.NewName == "" {
			return tool.ExecutionResult{Content: "new_name is required for rename action", IsError: true}, nil
		}
		edit, err := client.Rename(ctx, uri, pos, params.NewName)
		if err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
		}
		if edit == nil {
			return tool.ExecutionResult{Content: "Rename is not available at this position."}, nil
		}
		applied, err := ApplyWorkspaceEdit(*edit)
		if err != nil {
			return tool.ExecutionResult{Content: fmt.Sprintf("Rename planned but apply failed: %v", err), IsError: true}, nil
		}

		// Sync changed files with LSP servers
		for fileURI := range FlattenWorkspaceTextEdits(*edit) {
			changedPath := uriToPath(fileURI)
			_ = t.manager.SyncContent(ctx, client, changedPath)
		}

		return tool.ExecutionResult{Content: fmt.Sprintf("Rename applied: %d file(s) modified.", applied)}, nil

	default:
		return tool.ExecutionResult{
			Content: fmt.Sprintf("unknown action: %s (supported: diagnostics, definition, type_definition, implementation, references, hover, symbols, code_actions, rename, status)", params.Action),
			IsError: true,
		}, nil
	}
}

func formatCodeActions(actions []CodeAction) string {
	if len(actions) == 0 {
		return "No code actions available."
	}

	// Group by kind
	groups := make(map[string][]CodeAction)
	var kinds []string
	for _, a := range actions {
		k := string(a.Kind)
		if k == "" {
			k = "other"
		}
		if _, ok := groups[k]; !ok {
			kinds = append(kinds, k)
		}
		groups[k] = append(groups[k], a)
	}
	sort.Strings(kinds)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Code actions (%d):\n", len(actions)))
	for _, kind := range kinds {
		sb.WriteString(fmt.Sprintf("\n[%s]\n", kind))
		for i, a := range groups[kind] {
			sb.WriteString(fmt.Sprintf("  %d. %s", i+1, a.Title))
			if a.Disabled != nil {
				sb.WriteString(fmt.Sprintf(" (disabled: %s)", a.Disabled.Reason))
			}
			if a.Edit != nil {
				flat := FlattenWorkspaceTextEdits(*a.Edit)
				sb.WriteString(fmt.Sprintf(" — %d file(s) affected", len(flat)))
			}
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func extToLanguageID(ext string) string {
	m := map[string]string{
		".go": "go", ".ts": "typescript", ".tsx": "typescriptreact",
		".js": "javascript", ".jsx": "javascriptreact", ".mjs": "javascript",
		".cjs": "javascript", ".rs": "rust", ".c": "c", ".cpp": "cpp",
		".cc": "cpp", ".cxx": "cpp", ".h": "c", ".hpp": "cpp",
		".hxx": "cpp", ".py": "python", ".pyi": "python",
		".java": "java", ".kt": "kotlin", ".kts": "kotlin",
		".cs": "csharp", ".swift": "swift", ".rb": "ruby",
		".scala": "scala", ".sbt": "scala", ".hs": "haskell",
		".ml": "ocaml", ".mli": "ocaml", ".ex": "elixir", ".exs": "elixir",
		".erl": "erlang", ".gleam": "gleam", ".zig": "zig",
		".html": "html", ".htm": "html", ".css": "css", ".scss": "scss",
		".less": "less", ".json": "json", ".jsonc": "jsonc",
		".yaml": "yaml", ".yml": "yaml", ".vue": "vue", ".svelte": "svelte",
		".astro": "astro", ".lua": "lua", ".php": "php",
		".dart": "dart", ".sh": "shellscript", ".bash": "shellscript",
		".zsh": "shellscript", ".tf": "terraform",
		".md": "markdown", ".tex": "latex", ".graphql": "graphql",
		".prisma": "prisma", ".vim": "vim", ".nix": "nix",
		".odin": "odin", ".tla": "tla",
	}
	if id, ok := m[ext]; ok {
		return id
	}
	return strings.TrimPrefix(ext, ".")
}
