# Formatter Integration & Config Consolidation Design

## Overview

Integrate CLI formatters (prettier, black, rustfmt, etc.) into Monika's file editing pipeline, providing VS Code-level formatting experience. Simultaneously consolidate `lsp.json` into `config.json` and add frontend settings UI for both LSP and formatter configuration.

## Config Structure

### New Fields in `config.json`

```json
{
  "model_provider": "...",
  "model_providers": { ... },
  "agents": [ ... ],
  "skill": { ... },
  "mcp": { ... },
  "tools": { ... },

  "lsp": {
    "servers": {
      "gopls": {
        "command": "gopls",
        "args": ["serve"],
        "fileTypes": [".go", ".mod", ".sum"],
        "rootMarkers": ["go.mod"]
      },
      "typescript-language-server": {
        "command": "typescript-language-server",
        "args": ["--stdio"],
        "fileTypes": [".ts", ".tsx", ".js", ".jsx"],
        "rootMarkers": ["tsconfig.json", "package.json"]
      }
    }
  },

  "formatters": {
    "python": { "command": "black", "args": ["--line-length", "100"] },
    "typescript": { "command": "npx", "args": ["prettier", "--write"] },
    "go": "lsp"
  }
}
```

### Go Structs

```go
type Config struct {
    ModelProvider  string                    `yaml:"model_provider" json:"model_provider"`
    Model          string                    `yaml:"model" json:"model"`
    ModelProviders map[string]ProviderConfig `yaml:"model_providers" json:"model_providers"`
    Agents         []AgentEntry              `yaml:"agents" json:"agents"`
    Skill          SkillConfig               `yaml:"skill" json:"skill"`
    MCP            MCPConfig                 `yaml:"mcp" json:"mcp"`
    Tools          ToolsConfig               `yaml:"tools" json:"tools"`
    LSP            LSPConfig                 `yaml:"lsp" json:"lsp"`
    Formatters     map[string]FormatterConfig `yaml:"formatters" json:"formatters"`
}

type LSPConfig struct {
    Servers map[string]lsp.ServerConfig `yaml:"servers" json:"servers"`
}

type FormatterConfig struct {
    Command string   `yaml:"command" json:"command"`
    Args    []string `yaml:"args,omitempty" json:"args,omitempty"`
    Ref     string   `yaml:"-" json:"-"` // internal: "lsp" string shorthand
}
```

`FormatterConfig` implements custom `UnmarshalJSON`:
- String value `"lsp"` → sets `Ref = "lsp"`, Command empty
- Object `{ "command": "black", "args": [...] }` → normal struct deserialization

### Two-level Config

Both `lsp` and `formatters` support global + project-level override:
- Global: `~/.monika/config.json`
- Project: `.monika/config.json`
- Merge strategy: project-level entries override global entries with same key (maps are merged, project wins on conflict)

### lsp.json Migration

On `Load()`:
1. Load `config.json` as usual (global then project)
2. Check if `.monika/lsp.json` exists (project level)
3. If exists, merge its contents into `cfg.LSP.Servers`
4. Write merged config back to `.monika/config.json`
5. Rename `lsp.json` → `lsp.json.migrated` (safe, not deleted)
6. Only runs once per project (after migration, lsp.json no longer exists)

## Formatter Execution

### New file: `internal/lsp/formatter.go`

**Extension-to-language mapping table** (~20 languages):
```
.go → "go", .py → "python", .ts/.tsx → "typescript",
.js/.jsx → "javascript", .rs → "rust", .lua → "lua",
.sh/.bash → "shell", .c/.h → "c", .cpp/.hpp → "cpp",
.java → "java", .rb → "ruby", .php → "php",
.swift → "swift", .kt → "kotlin", .cs → "csharp",
.scss/.sass → "scss", .css → "css", .html → "html",
.json → "json", .yaml/.yml → "yaml", .md → "markdown"
```

**`ResolveFormatter(formatters map[string]FormatterConfig, filePath string) (command string, args []string, found bool)`**
1. Extract file extension from `filePath`
2. Map extension to language name
3. Look up `formatters[language]`
4. If not found or `config.Ref == "lsp"` → return `(found=false)`
5. Otherwise return `(config.Command, config.Args, true)`

**`RunCLIFormatter(ctx context.Context, command string, args []string, filePath string) (string, error)`**
1. Build full arg list: `append(args, filePath)`
2. Execute command with `exec.CommandContext(ctx, command, fullArgs...)`
3. Set working directory to the file's parent directory
4. Capture stderr for error reporting
5. If exit code != 0, return error
6. Read file from disk (formatter writes in-place), return formatted content

## WriteThrough Pipeline Change

Current Step 2 in `WriteThrough` (`internal/lsp/manager.go:588`):
```go
if opts.FormatOnWrite {
    formatted, err := m.FormatContent(ctx, client, filePath)
    // ...
}
```

New Step 2:
```go
if opts.FormatOnWrite {
    cmd, args, found := ResolveFormatter(m.formatters, filePath)
    if found {
        formatted, err := RunCLIFormatter(ctx, cmd, args, filePath)
        if err != nil {
            // CLI failed, fallback to LSP
            formatted, err = m.FormatContent(ctx, client, filePath)
        }
        // use formatted content
    } else {
        formatted, err = m.FormatContent(ctx, client, filePath)
        // use formatted content
    }
}
```

Config injection path:
- `main.go` creates `Config`, passes to `RegisterLSP` or `WireLSPHooks`
- `Manager` stores `formatters` from config
- `WireLSPHooks` closure in `register.go` accesses config through the manager

## LSP Startup Change

`Manager` initialization currently reads `.monika/lsp.json` separately.
After consolidation, `Manager` receives LSP server config from `Config.LSP.Servers`.
The `defaults.go` `DefaultServers` map remains as built-in fallback — user config overrides defaults.

## Frontend Settings UI

### Scope Selector

Settings page has a scope toggle at the top: **Global** | **Project**
- Determines whether edits write to `~/.monika/config.json` or `.monika/config.json`
- Both scopes display merged values, but project-level entries are visually marked (e.g., highlight or badge)
- Switch to "Project" scope → shows project-level config, edits save to `.monika/config.json`

### LSP Configuration Section

- List of configured LSP servers
- Each entry shows: server name, command, file types, root markers
- Add / Edit / Delete actions
- On save: writes to `config.json` at the selected scope, in `lsp.servers` field
- Editing triggers Go-side `SaveLSPConfig(scope, servers)` API call

### Formatter Configuration Section

- List of configured formatters grouped by language
- Each entry shows: language name, command, args
- Add: language dropdown + command input + args input (comma-separated or tag-style)
- Edit / Delete actions
- Special value: typing "lsp" in command field sets the shorthand
- On save: writes to `config.json` at the selected scope, in `formatters` field
- Editing triggers Go-side `SaveFormatterConfig(scope, formatters)` API call

### API Surface

New Go methods on `App` (registered as Wails services):

```go
func (a *App) GetLSPConfig(scope string) map[string]ServerConfig
func (a *App) SaveLSPConfig(scope string, servers map[string]ServerConfig) error
func (a *App) GetFormatterConfig(scope string) map[string]FormatterConfig
func (a *App) SaveFormatterConfig(scope string, formatters map[string]FormatterConfig) error
```

`scope` is `"global"` or `"project"`.

## Files Changed

| File | Change |
|------|--------|
| `internal/config/config.go` | Add `LSPConfig`, `FormatterConfig` structs, migration logic in `Load()` |
| `internal/lsp/formatter.go` | New file: `ResolveFormatter`, `RunCLIFormatter`, extension mapping |
| `internal/lsp/manager.go` | WriteThrough pipeline change, accept formatters config, LSP startup from config |
| `internal/lsp/tool.go` | Pass formatters through if needed |
| `internal/tool/builtin/register.go` | `WireLSPHooks` accepts config, passes formatters to manager |
| `main.go` | Wire new config fields to LSP manager |
| `internal/api/types.go` | New binding types for LSP/Formatter config |
| `internal/api/app.go` | New `GetLSPConfig`, `SaveLSPConfig`, `GetFormatterConfig`, `SaveFormatterConfig` methods |
| `frontend/bindings/monika/` | Auto-regenerated from Go types |
| `frontend/src/store/index.ts` | Settings state for LSP and formatters |
| `frontend/src/components/` | New settings components for LSP and formatter config |

## Edge Cases

- Formatter command not found on system: `RunCLIFormatter` returns error → fallback LSP, no user-visible crash
- No LSP server for file type and no CLI formatter configured: no formatting, file saved as-is (current behavior)
- Project has `lsp.json` but no `.monika/` dir: migration creates `.monika/` if needed
- Empty `formatters` map: all files use LSP formatting when available (current behavior preserved)
- Formatter modifies file in place but returns non-zero: treat as error, fallback to LSP
- Formatter takes too long: respect context cancellation from WriteThrough
