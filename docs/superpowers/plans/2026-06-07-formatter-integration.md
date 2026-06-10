# Formatter Integration & Config Consolidation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Integrate CLI formatters into Monika's file editing pipeline with CLI-first fallback-to-LSP strategy, consolidate `lsp.json` into `config.json`, and add frontend settings UI for LSP and formatter configuration.

**Architecture:** `FormatterConfig` lives in the `internal/lsp` package to avoid import cycles. `Config` gains `LSP LSPConfig` and `Formatters map[string]lsp.FormatterConfig`. On `Load()`, `lsp.json` is migrated into `config.json.lsp.servers` then renamed to `.migrated`. `Manager` gains a `formatters` field used in `WriteThrough` to try CLI formatter first then fallback to LSP. Four new `App` API methods (Get/Save with scope `"global"` | `"project"`) expose config to the frontend. Settings page gains LSP + Formatters tabs with Global/Project scope toggle.

**Tech Stack:** Go 1.22+, Wails v3, React/TypeScript, zustand, gopkg.in/yaml.v3

---

## File Structure

### Created files

| File | Responsibility |
|------|---------------|
| `internal/config/migrate.go` | lsp.json → config.json migration logic |
| `internal/config/migrate_test.go` | Migration tests |
| `internal/lsp/formatter.go` | FormatterConfig, unmarshalers, extToLang, ResolveFormatter, RunCLIFormatter |
| `internal/lsp/formatter_test.go` | Formatter unit tests |
| `frontend/src/components/Settings/LspFormattersTab.tsx` | LSP + Formatters settings UI with scope toggle |

### Modified files

| File | Change summary |
|------|---------------|
| `internal/config/config.go` | Add `LSPConfig`, import `lsp.FormatterConfig`, update `Config`, `merge()`, call migration in `Load()` |
| `internal/config/config_test.go` | Tests for LSP/formatter merge behavior |
| `internal/lsp/defaults.go` | Add yaml tags to `ServerConfig` |
| `internal/lsp/config.go` | Add `ResolveServersFromConfig(workdir, userServers)` |
| `internal/lsp/manager.go` | Add `formatters` field, update `WriteThrough` Step 2 |
| `internal/lsp/tool.go` | `NewLSPTool` accepts formatters config, pass to Manager |
| `internal/tool/builtin/lsp.go` | `NewLSPTool` signature changes |
| `internal/tool/builtin/register.go` | `RegisterLSP` accepts `*config.Config` |
| `main.go` | Pass `Config` to `RegisterLSP` |
| `internal/api/app.go` | Add `GetLSPConfig`, `SaveLSPConfig`, `GetFormatterConfig`, `SaveFormatterConfig`, update project-open path |
| `frontend/src/store/index.ts` | New types, state, actions for settings scope/LSP/formatters |
| `frontend/src/components/Settings/SettingsPage.tsx` | Add LSP + Formatters tab |

---

### Task 1: FormatterConfig + extToLang + ResolveFormatter + RunCLIFormatter

**Files:**
- Create: `internal/lsp/formatter.go`
- Create: `internal/lsp/formatter_test.go`
- Modify: `internal/config/config.go` (use `lsp.FormatterConfig` in Config struct)
- Modify: `internal/config/config_test.go` (update formatter tests)

- [ ] **Step 1: Write failing test for FormatterConfig**

Create `internal/lsp/formatter_test.go`:

```go
package lsp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

type formatterCfg = FormatterConfig

func TestFormatterConfigUnmarshalJSON_Shorthand(t *testing.T) {
	var fc FormatterConfig
	if err := json.Unmarshal([]byte(`"lsp"`), &fc); err != nil {
		t.Fatal(err)
	}
	if fc.Ref != "lsp" {
		t.Fatalf("Ref = %q, want \"lsp\"", fc.Ref)
	}
	if fc.Command != "" {
		t.Fatalf("Command = %q, want empty", fc.Command)
	}
}

func TestFormatterConfigUnmarshalJSON_Object(t *testing.T) {
	var fc FormatterConfig
	if err := json.Unmarshal([]byte(`{"command":"black","args":["--line-length","100"]}`), &fc); err != nil {
		t.Fatal(err)
	}
	if fc.Ref != "" {
		t.Fatalf("Ref = %q, want empty", fc.Ref)
	}
	if fc.Command != "black" {
		t.Fatalf("Command = %q, want \"black\"", fc.Command)
	}
	if len(fc.Args) != 2 || fc.Args[0] != "--line-length" || fc.Args[1] != "100" {
		t.Fatalf("Args = %v, want [--line-length 100]", fc.Args)
	}
}

func TestFormatterConfigMarshalJSON_Shorthand(t *testing.T) {
	fc := FormatterConfig{Ref: "lsp"}
	data, err := json.Marshal(fc)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `"lsp"` {
		t.Fatalf("got %s, want \"lsp\"", string(data))
	}
}

func TestFormatterConfigMarshalJSON_Object(t *testing.T) {
	fc := FormatterConfig{Command: "black", Args: []string{"--line-length", "100"}}
	data, err := json.Marshal(fc)
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"command":"black","args":["--line-length","100"]}`
	if string(data) != expected {
		t.Fatalf("got %s, want %s", string(data), expected)
	}
}

func TestFormatterConfigUnmarshalYAML_Shorthand(t *testing.T) {
	var fc FormatterConfig
	if err := yaml.Unmarshal([]byte(`lsp`), &fc); err != nil {
		t.Fatal(err)
	}
	if fc.Ref != "lsp" {
		t.Fatalf("Ref = %q, want \"lsp\"", fc.Ref)
	}
	if fc.Command != "" {
		t.Fatalf("Command = %q, want empty", fc.Command)
	}
}

func TestFormatterConfigUnmarshalYAML_Object(t *testing.T) {
	var fc FormatterConfig
	if err := yaml.Unmarshal([]byte(`command: black
args:
  - "--line-length"
  - "100"
`), &fc); err != nil {
		t.Fatal(err)
	}
	if fc.Ref != "" {
		t.Fatalf("Ref = %q, want empty", fc.Ref)
	}
	if fc.Command != "black" {
		t.Fatalf("Command = %q, want \"black\"", fc.Command)
	}
	if len(fc.Args) != 2 || fc.Args[0] != "--line-length" || fc.Args[1] != "100" {
		t.Fatalf("Args = %v, want [--line-length 100]", fc.Args)
	}
}

func TestResolveFormatter_Found(t *testing.T) {
	formatters := map[string]FormatterConfig{
		"python": {Command: "black", Args: []string{"--line-length", "100"}},
	}
	cmd, args, found := ResolveFormatter(formatters, "/home/user/main.py")
	if !found {
		t.Fatal("expected found")
	}
	if cmd != "black" {
		t.Fatalf("cmd = %q, want \"black\"", cmd)
	}
	if len(args) != 1 || args[0] != "--line-length" {
		t.Fatalf("args = %v", args)
	}
}

func TestResolveFormatter_RefLSP(t *testing.T) {
	formatters := map[string]FormatterConfig{
		"go": {Ref: "lsp"},
	}
	_, _, found := ResolveFormatter(formatters, "main.go")
	if found {
		t.Fatal("expected not found (lsp shorthand)")
	}
}

func TestResolveFormatter_NotFound(t *testing.T) {
	formatters := map[string]FormatterConfig{
		"python": {Command: "black"},
	}
	_, _, found := ResolveFormatter(formatters, "main.rs")
	if found {
		t.Fatal("expected not found")
	}
}

func TestResolveFormatter_EmptyFormatters(t *testing.T) {
	_, _, found := ResolveFormatter(nil, "main.go")
	if found {
		t.Fatal("expected not found (nil map)")
	}
}

func TestResolveFormatter_UnknownExtension(t *testing.T) {
	formatters := map[string]FormatterConfig{
		"go": {Command: "gofmt"},
	}
	_, _, found := ResolveFormatter(formatters, "Dockerfile")
	if found {
		t.Fatal("expected not found")
	}
}

func TestRunCLIFormatter_ExecutesAndReturnsContent(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("skipping CLI formatter test in CI (no formatter installed)")
	}
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "test.py")
	if err := os.WriteFile(filePath, []byte("x=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Use a simple in-place formatter (cat/type doesn't format, skip real formatter test)
	// Instead, test that a command that just reads the file works
	content, err := RunCLIFormatter(context.Background(), "echo", []string{"hello"}, filePath)
	if err != nil {
		t.Skipf("echo not available: %v", err)
	}
	// The echo command writes to stdout, but RunCLIFormatter reads file from disk
	// after the command runs, so content should still be "x=1\n"
	if content != "x=1\n" {
		t.Fatalf("content = %q", content)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd d:/git/monika && go test ./internal/lsp/ -run "TestFormatterConfig|TestResolveFormatter|TestRunCLIFormatter" -v
```
Expected: FAIL with "undefined: FormatterConfig", "undefined: ResolveFormatter", etc.

- [ ] **Step 3: Implement FormatterConfig, extToLang, ResolveFormatter, RunCLIFormatter**

Create `internal/lsp/formatter.go`:

```go
package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type FormatterConfig struct {
	Command string   `yaml:"command" json:"command"`
	Args    []string `yaml:"args,omitempty" json:"args,omitempty"`
	Ref     string   `yaml:"-" json:"-"`
}

func (f *FormatterConfig) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		f.Ref = s
		return nil
	}
	type alias FormatterConfig
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*f = FormatterConfig(a)
	return nil
}

func (f FormatterConfig) MarshalJSON() ([]byte, error) {
	if f.Ref != "" {
		return json.Marshal(f.Ref)
	}
	type alias FormatterConfig
	return json.Marshal(alias(f))
}

func (f *FormatterConfig) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		f.Ref = value.Value
		return nil
	}
	type alias FormatterConfig
	var a alias
	if err := value.Decode(&a); err != nil {
		return err
	}
	*f = FormatterConfig(a)
	return nil
}

var extToLang = map[string]string{
	".go":    "go",
	".mod":   "go",
	".sum":   "go",
	".py":    "python",
	".pyi":   "python",
	".ts":    "typescript",
	".tsx":   "typescript",
	".js":    "javascript",
	".jsx":   "javascript",
	".mjs":   "javascript",
	".cjs":   "javascript",
	".rs":    "rust",
	".lua":   "lua",
	".sh":    "shell",
	".bash":  "shell",
	".zsh":   "shell",
	".c":     "c",
	".h":     "c",
	".cpp":   "cpp",
	".cc":    "cpp",
	".cxx":   "cpp",
	".hpp":   "cpp",
	".hxx":   "cpp",
	".java":  "java",
	".rb":    "ruby",
	".rake":  "ruby",
	".gemspec": "ruby",
	".php":   "php",
	".swift": "swift",
	".kt":    "kotlin",
	".kts":   "kotlin",
	".cs":    "csharp",
	".scss":  "scss",
	".sass":  "scss",
	".css":   "css",
	".less":  "css",
	".html":  "html",
	".htm":   "html",
	".json":  "json",
	".jsonc": "json",
	".yaml":  "yaml",
	".yml":   "yaml",
	".md":    "markdown",
	".mdx":   "markdown",
}

func ResolveFormatter(formatters map[string]FormatterConfig, filePath string) (command string, args []string, found bool) {
	if formatters == nil {
		return "", nil, false
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == "" {
		return "", nil, false
	}
	lang, ok := extToLang[ext]
	if !ok {
		return "", nil, false
	}
	cfg, ok := formatters[lang]
	if !ok || cfg.Ref == "lsp" {
		return "", nil, false
	}
	return cfg.Command, cfg.Args, true
}

func RunCLIFormatter(ctx context.Context, command string, args []string, filePath string) (string, error) {
	fullArgs := append(args, filePath)
	cmd := exec.CommandContext(ctx, command, fullArgs...)
	cmd.Dir = filepath.Dir(filePath)

	// Resolve command path (search node_modules/.bin, venv, PATH)
	resolved := ResolveCommand(command, cmd.Dir)
	if resolved != command {
		cmd.Path = resolved
		// Rebuild with resolved path
		cmd.Args = append([]string{resolved}, fullArgs...)
	}

	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("formatter %q: %w", command, err)
	}
	var stderrBuf strings.Builder
	if stderr != nil {
		// drain stderr
		go func() {
			buf := make([]byte, 1024)
			for {
				n, _ := stderr.Read(buf)
				if n == 0 {
					break
				}
				stderrBuf.Write(buf[:n])
			}
		}()
	}
	if err := cmd.Wait(); err != nil {
		if stderrBuf.Len() > 0 {
			return "", fmt.Errorf("formatter %q exited: %v: %s", command, err, strings.TrimSpace(stderrBuf.String()))
		}
		return "", fmt.Errorf("formatter %q: %w", command, err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("formatter %q: read output: %w", command, err)
	}
	return string(data), nil
}
```

- [ ] **Step 4: Update Config struct in config.go to use lsp.FormatterConfig**

In `internal/config/config.go`, update the `Formatters` field:

```go
	Formatters map[string]lsp.FormatterConfig `yaml:"formatters" json:"formatters"`
```

This replaces what was placeholder in the struct. Make sure `"monika/internal/lsp"` is in imports.

- [ ] **Step 5: Update config_test.go formatter test types**

In `internal/config/config_test.go`, change all `FormatterConfig` references to `lsp.FormatterConfig` and add `"monika/internal/lsp"` to imports:

```go
import (
	"monika/internal/lsp"
	...
)
```

Update the formatter-related tests to use `lsp.FormatterConfig{}` instead of `FormatterConfig{}`.

- [ ] **Step 6: Run all tests**

```bash
cd d:/git/monika && go test ./internal/lsp/ -run "TestFormatterConfig|TestResolveFormatter|TestRunCLIFormatter" -v
```
Expected: 12 PASS (1 skip)

```bash
cd d:/git/monika && go test ./internal/config/ -v
```
Expected: ALL PASS

- [ ] **Step 7: Commit**

```bash
git add internal/lsp/formatter.go internal/lsp/formatter_test.go internal/config/config.go internal/config/config_test.go
git commit -m "feat: add FormatterConfig, extToLang, ResolveFormatter, RunCLIFormatter"
```

---

### Task 2: LSPConfig + merge() update + ServerConfig yaml tags

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/lsp/defaults.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write failing test for LSP config merge**

Add to `internal/config/config_test.go`:

```go
func TestLoadMergesLSPServers(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	project := filepath.Join(tmp, "project")
	mustWrite(t, filepath.Join(home, ".monika", "config.json"), []byte(`{
  "lsp": {
    "servers": {
      "gopls": { "command": "gopls", "fileTypes": [".go"], "disabled": true }
    }
  }
}`))
	mustWrite(t, filepath.Join(project, ".monika", "config.json"), []byte(`{
  "lsp": {
    "servers": {
      "pyright": { "command": "pyright-langserver", "fileTypes": [".py"] }
    }
  }
}`))
	cfg, err := Load(Options{HomeDir: home, ProjectDir: project})
	if err != nil {
		t.Fatal(err)
	}
	gopls, ok := cfg.LSP.Servers["gopls"]
	if !ok {
		t.Fatal("expected gopls in LSP servers")
	}
	if !gopls.Disabled {
		t.Fatal("expected gopls to be disabled")
	}
	pyright, ok := cfg.LSP.Servers["pyright"]
	if !ok {
		t.Fatal("expected pyright in LSP servers")
	}
	if pyright.Command != "pyright-langserver" {
		t.Fatalf("pyright command = %q", pyright.Command)
	}
}

func TestLoadMergesFormatters(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	project := filepath.Join(tmp, "project")
	mustWrite(t, filepath.Join(home, ".monika", "config.json"), []byte(`{
  "formatters": { "go": "lsp" }
}`))
	mustWrite(t, filepath.Join(project, ".monika", "config.json"), []byte(`{
  "formatters": { "python": { "command": "black", "args": [] } }
}`))
	cfg, err := Load(Options{HomeDir: home, ProjectDir: project})
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Formatters) != 2 {
		t.Fatalf("expected 2 formatters, got %d", len(cfg.Formatters))
	}
	goFmt, ok := cfg.Formatters["go"]
	if !ok {
		t.Fatal("expected go formatter")
	}
	if goFmt.Ref != "lsp" {
		t.Fatalf("go ref = %q, want \"lsp\"", goFmt.Ref)
	}
	pyFmt, ok := cfg.Formatters["python"]
	if !ok {
		t.Fatal("expected python formatter")
	}
	if pyFmt.Command != "black" {
		t.Fatalf("python command = %q", pyFmt.Command)
	}
}

func TestLoadProjectFormattersOverrideGlobal(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	project := filepath.Join(tmp, "project")
	mustWrite(t, filepath.Join(home, ".monika", "config.json"), []byte(`{
  "formatters": { "go": "lsp", "python": { "command": "black" } }
}`))
	mustWrite(t, filepath.Join(project, ".monika", "config.json"), []byte(`{
  "formatters": { "go": { "command": "gofmt", "args": ["-w"] } }
}`))
	cfg, err := Load(Options{HomeDir: home, ProjectDir: project})
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Formatters) != 2 {
		t.Fatalf("expected 2 formatters, got %d", len(cfg.Formatters))
	}
	goFmt := cfg.Formatters["go"]
	if goFmt.Command != "gofmt" {
		t.Fatalf("go command = %q, want \"gofmt\"", goFmt.Command)
	}
}
```

- [ ] **Step 2: Add yaml tags to ServerConfig**

In `internal/lsp/defaults.go`, update `ServerConfig` struct:

```go
type ServerConfig struct {
	Command     string         `yaml:"command" json:"command"`
	Args        []string       `yaml:"args,omitempty" json:"args,omitempty"`
	FileTypes   []string       `yaml:"fileTypes" json:"fileTypes"`
	RootMarkers []string       `yaml:"rootMarkers" json:"rootMarkers"`
	InitOptions map[string]any `yaml:"initOptions,omitempty" json:"initOptions,omitempty"`
	Settings    map[string]any `yaml:"settings,omitempty" json:"settings,omitempty"`
	Disabled    bool           `yaml:"disabled,omitempty" json:"disabled,omitempty"`
}
```

- [ ] **Step 3: Add LSPConfig struct and merge logic**

In `internal/config/config.go`, add after the `ToolsConfig` block (before `func Load`):

```go
type LSPConfig struct {
	Servers map[string]lsp.ServerConfig `yaml:"servers" json:"servers"`
}
```

Update the `Config` struct to add `LSP` and `Formatters` fields (after `Tools`):

```go
	LSP        LSPConfig                     `yaml:"lsp" json:"lsp"`
	Formatters map[string]lsp.FormatterConfig `yaml:"formatters" json:"formatters"`
```

In the `merge()` function, add before the closing `}`:

```go
	if len(src.LSP.Servers) > 0 {
		if dst.LSP.Servers == nil {
			dst.LSP.Servers = make(map[string]lsp.ServerConfig, len(src.LSP.Servers))
		}
		for name, srv := range src.LSP.Servers {
			dst.LSP.Servers[name] = srv
		}
	}
	if len(src.Formatters) > 0 {
		if dst.Formatters == nil {
			dst.Formatters = make(map[string]lsp.FormatterConfig, len(src.Formatters))
		}
		for lang, fmt := range src.Formatters {
			dst.Formatters[lang] = fmt
		}
	}
```

- [ ] **Step 4: Run all config tests**

```bash
cd d:/git/monika && go test ./internal/config/ -v
```
Expected: ALL PASS (existing + new)

- [ ] **Step 5: Run lsp tests**

```bash
cd d:/git/monika && go test ./internal/lsp/ -v
```
Expected: ALL PASS (edits_test.go still passes, formatter_test.go passes)

- [ ] **Step 6: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go internal/lsp/defaults.go
git commit -m "feat: add LSPConfig, update merge(), add yaml tags to ServerConfig"
```

---

### Task 3: lsp.json migration

**Files:**
- Create: `internal/config/migrate.go`
- Create: `internal/config/migrate_test.go`

- [ ] **Step 1: Write failing test for migration**

Create `internal/config/migrate_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateLSPJSON_MovesServersToConfig(t *testing.T) {
	tmp := t.TempDir()
	project := filepath.Join(tmp, "project")
	monikaDir := filepath.Join(project, ".monika")
	os.MkdirAll(monikaDir, 0o755)

	lspJSON := filepath.Join(monikaDir, "lsp.json")
	mustWrite(t, lspJSON, []byte(`{
  "gopls": { "command": "gopls", "fileTypes": [".go"] },
  "pyright": { "command": "pyright-langserver", "fileTypes": [".py"] }
}`))

	configJSON := filepath.Join(monikaDir, "config.json")
	mustWrite(t, configJSON, []byte(`{"model_provider":"openai"}`))

	if err := migrateLSPJSON(project); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(Options{ProjectDir: project})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ModelProvider != "openai" {
		t.Fatalf("model_provider = %q", cfg.ModelProvider)
	}
	if _, ok := cfg.LSP.Servers["gopls"]; !ok {
		t.Fatal("gopls not found in config")
	}
	if _, ok := cfg.LSP.Servers["pyright"]; !ok {
		t.Fatal("pyright not found in config")
	}

	if _, err := os.Stat(lspJSON); !os.IsNotExist(err) {
		t.Fatal("lsp.json should have been renamed")
	}
	if _, err := os.Stat(lspJSON + ".migrated"); err != nil {
		t.Fatal("lsp.json.migrated should exist")
	}
}

func TestMigrateLSPJSON_NoLSPJsonDoesNothing(t *testing.T) {
	tmp := t.TempDir()
	project := filepath.Join(tmp, "project")
	monikaDir := filepath.Join(project, ".monika")
	os.MkdirAll(monikaDir, 0o755)

	configJSON := filepath.Join(monikaDir, "config.json")
	mustWrite(t, configJSON, []byte(`{"model_provider":"openai"}`))

	if err := migrateLSPJSON(project); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(Options{ProjectDir: project})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ModelProvider != "openai" {
		t.Fatalf("model_provider = %q", cfg.ModelProvider)
	}
	if len(cfg.LSP.Servers) != 0 {
		t.Fatalf("expected 0 LSP servers, got %d", len(cfg.LSP.Servers))
	}
}

func TestMigrateLSPJSON_NoMonikaDir(t *testing.T) {
	tmp := t.TempDir()
	project := filepath.Join(tmp, "project")
	if err := migrateLSPJSON(project); err != nil {
		t.Fatal(err)
	}
}

func TestMigrateLSPJSON_NoExistingConfigCreatesOne(t *testing.T) {
	tmp := t.TempDir()
	project := filepath.Join(tmp, "project")
	monikaDir := filepath.Join(project, ".monika")
	os.MkdirAll(monikaDir, 0o755)

	lspJSON := filepath.Join(monikaDir, "lsp.json")
	mustWrite(t, lspJSON, []byte(`{
  "gopls": { "command": "gopls", "fileTypes": [".go"] }
}`))

	if err := migrateLSPJSON(project); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(Options{ProjectDir: project})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.LSP.Servers["gopls"]; !ok {
		t.Fatal("gopls not found in config")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd d:/git/monika && go test ./internal/config/ -run "TestMigrateLSPJSON" -v
```
Expected: FAIL with "undefined: migrateLSPJSON"

- [ ] **Step 3: Implement migrateLSPJSON**

Create `internal/config/migrate.go`:

```go
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func migrateLSPJSON(projectDir string) error {
	lspPath := filepath.Join(projectDir, ".monika", "lsp.json")
	if _, err := os.Stat(lspPath); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	data, err := os.ReadFile(lspPath)
	if err != nil {
		return fmt.Errorf("migrate lsp.json: %w", err)
	}

	var servers map[string]lsp.ServerConfig
	if err := json.Unmarshal(data, &servers); err != nil {
		return fmt.Errorf("migrate lsp.json: %w", err)
	}

	configPath := filepath.Join(projectDir, ".monika", "config.json")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			os.MkdirAll(filepath.Dir(configPath), 0o755)
			configData = []byte("{}")
		} else {
			return fmt.Errorf("migrate lsp.json: %w", err)
		}
	}

	var cfgMap map[string]json.RawMessage
	if err := json.Unmarshal(configData, &cfgMap); err != nil {
		return fmt.Errorf("migrate lsp.json: %w", err)
	}

	serversData, err := json.Marshal(servers)
	if err != nil {
		return fmt.Errorf("migrate lsp.json: %w", err)
	}
	cfgMap["lsp"] = json.RawMessage(fmt.Sprintf(`{"servers":%s}`, serversData))

	out, err := json.MarshalIndent(cfgMap, "", "  ")
	if err != nil {
		return fmt.Errorf("migrate lsp.json: %w", err)
	}

	tmpPath := configPath + ".tmp"
	if err := os.WriteFile(tmpPath, out, 0o600); err != nil {
		return fmt.Errorf("migrate lsp.json: %w", err)
	}
	if err := os.Rename(tmpPath, configPath); err != nil {
		return fmt.Errorf("migrate lsp.json: %w", err)
	}

	if err := os.Rename(lspPath, lspPath+".migrated"); err != nil {
		return fmt.Errorf("migrate lsp.json: %w", err)
	}

	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd d:/git/monika && go test ./internal/config/ -run "TestMigrateLSPJSON" -v
```
Expected: 4 PASS

- [ ] **Step 5: Wire migration into Load()**

In `internal/config/config.go`, `Load()` function, add after the project-level merge block (after the closing `}` of the project block, before `return cfg, nil`):

```go
	if opts.ProjectDir != "" {
		if err := migrateLSPJSON(opts.ProjectDir); err != nil {
			fmt.Fprintf(os.Stderr, "[monika] lsp.json migration: %v\n", err)
		}
	}
```

Add `"fmt"` to imports in config.go if not already present.

- [ ] **Step 6: Run all config tests**

```bash
cd d:/git/monika && go test ./internal/config/ -v
```
Expected: ALL PASS

- [ ] **Step 7: Commit**

```bash
git add internal/config/migrate.go internal/config/migrate_test.go internal/config/config.go
git commit -m "feat: add lsp.json to config.json migration"
```

---

### Task 4: ResolveServersFromConfig + Manager formatters field

**Files:**
- Modify: `internal/lsp/config.go`
- Modify: `internal/lsp/manager.go`

- [ ] **Step 1: Add ResolveServersFromConfig**

In `internal/lsp/config.go`, add new function after `ResolveServers`:

```go
// ResolveServersFromConfig returns the merged list of server configs:
// defaults filtered by root markers present in workdir,
// then overridden by the provided user server configs (from config.json).
func ResolveServersFromConfig(workdir string, userServers map[string]ServerConfig) map[string]ServerConfig {
	result := make(map[string]ServerConfig)
	for name, cfg := range DefaultServers {
		if cfg.Disabled {
			continue
		}
		if !hasRootMarker(workdir, cfg.RootMarkers) {
			continue
		}
		if !binaryAvailable(cfg.Command, workdir) {
			continue
		}
		result[name] = cfg
	}
	for name, cfg := range userServers {
		if cfg.Disabled {
			delete(result, name)
			continue
		}
		result[name] = cfg
	}
	return result
}
```

- [ ] **Step 2: Run lsp tests**

```bash
cd d:/git/monika && go test ./internal/lsp/ -v
```
Expected: ALL PASS (no regression)

- [ ] **Step 3: Add formatters field to Manager**

In `internal/lsp/manager.go`, update the `Manager` struct (add after `servers` field):

```go
	formatters  map[string]FormatterConfig
```

Update `NewManager` signature:

```go
func NewManager(workdir string, lspServers map[string]ServerConfig, formatters map[string]FormatterConfig) *Manager {
	return &Manager{
		workdir:     workdir,
		servers:     ResolveServersFromConfig(workdir, lspServers),
		formatters:  formatters,
		clients:     make(map[string]*managedClient),
		openFiles:   make(map[string]*openFile),
		clientLocks: make(map[string]*sync.Mutex),
		fileLocks:   make(map[string]*sync.Mutex),
	}
}
```

- [ ] **Step 4: Update all callers of NewManager**

In `internal/lsp/tool.go`, update `NewLSPTool`:

```go
func NewLSPTool(projectDir string, lspServers map[string]ServerConfig, formatters map[string]FormatterConfig) (tool.Tool, error) {
	m := NewManager(projectDir, lspServers, formatters)
	m.Start()
	return &LSPTool{manager: m}, nil
}
```

In `internal/tool/builtin/lsp.go`:

```go
package builtin

import (
	"monika/internal/lsp"
	"monika/internal/tool"
)

func NewLSPTool(projectDir string, lspServers map[string]lsp.ServerConfig, formatters map[string]lsp.FormatterConfig) (tool.Tool, error) {
	return lsp.NewLSPTool(projectDir, lspServers, formatters)
}
```

In `internal/tool/builtin/register.go`, update `RegisterLSP`:

```go
func RegisterLSP(r *tool.ToolRegistry, projectDir string, lspServers map[string]lsp.ServerConfig, formatters map[string]lsp.FormatterConfig) error {
	t, err := NewLSPTool(projectDir, lspServers, formatters)
	if err != nil {
		return err
	}
	r.Register(t)
	return nil
}
```

- [ ] **Step 5: Update main.go callers**

In `main.go`, find the `RegisterLSP` call and update:

```go
	builtin.RegisterLSP(registry, cwd, pr.Config.LSP.Servers, pr.Config.Formatters)
```

Find the project-open path in `internal/api/app.go` (around line 368):

```go
	_ = builtin.RegisterLSP(a.registry, path, a.cfg.LSP.Servers, a.cfg.Formatters)
```

- [ ] **Step 6: Build check**

```bash
cd d:/git/monika && go build ./...
```
Expected: No errors

- [ ] **Step 7: Run all tests**

```bash
cd d:/git/monika && go test ./... -short
```
Expected: ALL PASS

- [ ] **Step 8: Commit**

```bash
git add internal/lsp/config.go internal/lsp/manager.go internal/lsp/tool.go internal/tool/builtin/lsp.go internal/tool/builtin/register.go main.go internal/api/app.go
git commit -m "feat: add ResolveServersFromConfig, pass formatters config through to Manager"
```

---

### Task 5: WriteThrough pipeline change (CLI first, LSP fallback)

**Files:**
- Modify: `internal/lsp/manager.go`

- [ ] **Step 1: Update WriteThrough Step 2**

In `internal/lsp/manager.go`, replace the Step 2 block (lines 588-596):

Current:
```go
	// Step 2: Optional format
	if opts.FormatOnWrite {
		formatted, err := m.FormatContent(ctx, client, filePath)
		if err == nil && formatted != content {
			// Content changed — re-sync with formatted content
			finalContent = formatted
			_ = m.SyncContentFromMemory(ctx, client, filePath, formatted, serverName)
		}
	}
```

New:
```go
	// Step 2: Optional format — try CLI formatter first, fallback to LSP
	if opts.FormatOnWrite {
		var formatted string
		var err error
		cmd, cmdArgs, found := ResolveFormatter(m.formatters, filePath)
		if found {
			formatted, err = RunCLIFormatter(ctx, cmd, cmdArgs, filePath)
			if err != nil {
				lspLog("WriteThrough: CLI formatter failed: %v, falling back to LSP", err)
				formatted, err = m.FormatContent(ctx, client, filePath)
			}
		} else {
			formatted, err = m.FormatContent(ctx, client, filePath)
		}
		if err == nil && formatted != content {
			finalContent = formatted
			_ = m.SyncContentFromMemory(ctx, client, filePath, formatted, serverName)
		}
	}
```

- [ ] **Step 2: Build check**

```bash
cd d:/git/monika && go build ./internal/lsp/...
```
Expected: No errors

- [ ] **Step 3: Run lsp tests**

```bash
cd d:/git/monika && go test ./internal/lsp/ -v
```
Expected: ALL PASS

- [ ] **Step 4: Full build**

```bash
cd d:/git/monika && go build ./...
```
Expected: No errors

- [ ] **Step 5: Commit**

```bash
git add internal/lsp/manager.go
git commit -m "feat: WriteThrough uses CLI formatters with LSP fallback"
```

---

### Task 6: API layer — scope helpers and Get/Save methods

**Files:**
- Modify: `internal/api/app.go`

- [ ] **Step 1: Add scope config read/write helpers**

In `internal/api/app.go`, add two helper methods:

```go
func (a *App) configPathForScope(scope string) string {
	if scope == "project" {
		pp := a.projectPath()
		if pp == "" {
			return ""
		}
		return filepath.Join(pp, ".monika", "config.json")
	}
	return filepath.Join(a.home, ".monika", "config.json")
}

func (a *App) readConfigForScope(scope string) (config2.Config, error) {
	configPath := a.configPathForScope(scope)
	if configPath == "" {
		return config2.Config{}, fmt.Errorf("no project path for scope %q", scope)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return config2.Config{}, nil
		}
		return config2.Config{}, err
	}
	var cfg config2.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return config2.Config{}, fmt.Errorf("%s: %w", configPath, err)
	}
	return cfg, nil
}

func (a *App) writeConfigForScope(scope string, updateFn func(*config2.Config)) error {
	configPath := a.configPathForScope(scope)
	if configPath == "" {
		return fmt.Errorf("no project path for scope %q", scope)
	}
	cfg, err := a.readConfigForScope(scope)
	if err != nil {
		return err
	}
	updateFn(&cfg)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	os.MkdirAll(filepath.Dir(configPath), 0o755)
	tmp := configPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, configPath)
}
```

- [ ] **Step 2: Add GetLSPConfig and SaveLSPConfig**

```go
func (a *App) GetLSPConfig(scope string) map[string]lsp.ServerConfig {
	cfg, err := a.readConfigForScope(scope)
	if err != nil {
		return nil
	}
	if cfg.LSP.Servers == nil {
		return map[string]lsp.ServerConfig{}
	}
	return cfg.LSP.Servers
}

func (a *App) SaveLSPConfig(scope string, servers map[string]lsp.ServerConfig) error {
	if scope != "global" && scope != "project" {
		return fmt.Errorf("invalid scope %q (must be \"global\" or \"project\")", scope)
	}
	if err := a.writeConfigForScope(scope, func(cfg *config2.Config) {
		cfg.LSP.Servers = servers
	}); err != nil {
		return err
	}
	// Update in-memory merged config
	a.mu.Lock()
	if a.cfg.LSP.Servers == nil {
		a.cfg.LSP.Servers = make(map[string]lsp.ServerConfig)
	}
	for name, srv := range servers {
		a.cfg.LSP.Servers[name] = srv
	}
	a.mu.Unlock()
	return nil
}
```

- [ ] **Step 3: Add GetFormatterConfig and SaveFormatterConfig**

```go
func (a *App) GetFormatterConfig(scope string) map[string]lsp.FormatterConfig {
	cfg, err := a.readConfigForScope(scope)
	if err != nil {
		return nil
	}
	if cfg.Formatters == nil {
		return map[string]lsp.FormatterConfig{}
	}
	return cfg.Formatters
}

func (a *App) SaveFormatterConfig(scope string, formatters map[string]lsp.FormatterConfig) error {
	if scope != "global" && scope != "project" {
		return fmt.Errorf("invalid scope %q (must be \"global\" or \"project\")", scope)
	}
	if err := a.writeConfigForScope(scope, func(cfg *config2.Config) {
		cfg.Formatters = formatters
	}); err != nil {
		return err
	}
	// Update in-memory merged config
	a.mu.Lock()
	if a.cfg.Formatters == nil {
		a.cfg.Formatters = make(map[string]lsp.FormatterConfig)
	}
	for lang, fmt := range formatters {
		a.cfg.Formatters[lang] = fmt
	}
	a.mu.Unlock()
	return nil
}
```

- [ ] **Step 4: Build and test**

```bash
cd d:/git/monika && go build ./...
```
Expected: No errors

```bash
cd d:/git/monika && go vet ./internal/api/...
```
Expected: No errors

- [ ] **Step 5: Commit**

```bash
git add internal/api/app.go
git commit -m "feat: add Get/Save LSP and Formatter config API with scope support"
```

---

### Task 7: Frontend store types and actions

**Files:**
- Modify: `frontend/src/store/index.ts`

- [ ] **Step 1: Add type definitions**

Add after the `LSPServerStatus` interface (which already exists):

```typescript
export interface FormatterEntry {
    command: string
    args?: string[]
    ref?: string  // "lsp" shorthand
}

export type SettingsScope = 'global' | 'project'
```

- [ ] **Step 2: Add state fields**

Add to the store state interface (after existing `settingsOpen`):

```typescript
    settingsScope: SettingsScope
    lspConfigServers: Record<string, { command: string; args: string[]; fileTypes: string[]; rootMarkers: string[]; initOptions?: Record<string, any>; settings?: Record<string, any>; disabled?: boolean }>
    formatterConfig: Record<string, FormatterEntry>
```

- [ ] **Step 3: Add initial state**

Add to the initial state object:

```typescript
    settingsScope: 'global' as SettingsScope,
    lspConfigServers: {},
    formatterConfig: {},
```

- [ ] **Step 4: Add actions**

Add to the store actions interface:

```typescript
    setSettingsScope: (scope: SettingsScope) => void
    loadLSPConfig: (scope: SettingsScope) => Promise<void>
    saveLSPConfig: (scope: SettingsScope, servers: Record<string, any>) => Promise<void>
    loadFormatterConfig: (scope: SettingsScope) => Promise<void>
    saveFormatterConfig: (scope: SettingsScope, formatters: Record<string, FormatterEntry>) => Promise<void>
```

- [ ] **Step 5: Add action implementations**

Add to the `set` object:

```typescript
    setSettingsScope: (scope) => set({ settingsScope: scope }),

    loadLSPConfig: async (scope) => {
        try {
            const servers = await Call.ByName('monika/internal/api.App.GetLSPConfig', scope)
            set({ lspConfigServers: servers || {} })
        } catch { set({ lspConfigServers: {} }) }
    },

    saveLSPConfig: async (scope, servers) => {
        await Call.ByName('monika/internal/api.App.SaveLSPConfig', scope, servers)
        set({ lspConfigServers: servers })
    },

    loadFormatterConfig: async (scope) => {
        try {
            const formatters = await Call.ByName('monika/internal/api.App.GetFormatterConfig', scope)
            // Normalize: convert string "lsp" to { ref: "lsp", command: "" }
            const normalized: Record<string, FormatterEntry> = {}
            for (const [lang, cfg] of Object.entries(formatters || {})) {
                if (typeof cfg === 'string') {
                    normalized[lang] = { command: '', ref: cfg }
                } else {
                    normalized[lang] = cfg as FormatterEntry
                }
            }
            set({ formatterConfig: normalized })
        } catch { set({ formatterConfig: {} }) }
    },

    saveFormatterConfig: async (scope, formatters) => {
        // Convert { ref: "lsp" } back to "lsp" string, objects stay as-is
        const payload: Record<string, any> = {}
        for (const [lang, cfg] of Object.entries(formatters)) {
            if (cfg.ref) {
                payload[lang] = cfg.ref
            } else {
                payload[lang] = cfg
            }
        }
        await Call.ByName('monika/internal/api.App.SaveFormatterConfig', scope, payload)
        set({ formatterConfig: formatters })
    },
```

- [ ] **Step 6: Add resetProjectState cleanup**

In the `resetProjectState` action, add:

```typescript
        lspConfigServers: {},
        formatterConfig: {},
        settingsScope: 'global',
```

- [ ] **Step 7: Verify TypeScript compiles**

```bash
cd d:/git/monika/frontend && npm run build 2>&1 | head -20
```
If using a different build command, check package.json.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/store/index.ts
git commit -m "feat: add store types and actions for LSP/formatter config with scope"
```

---

### Task 8: Frontend LspFormattersTab component

**Files:**
- Create: `frontend/src/components/Settings/LspFormattersTab.tsx`

- [ ] **Step 1: Create LSP server card + formatter card component**

Create `frontend/src/components/Settings/LspFormattersTab.tsx`:

```tsx
import { useState, useEffect, useCallback } from 'react'
import { useStore } from '../../store'
import Modal, { ModalHeader, ModalBody, ModalFooter, ModalButton } from '../ui/Modal'
import { IconServer, IconTrash, IconPlus, IconEdit, IconCheck, IconRefresh } from '../Icons'

// --- LSP Server Card ---

function LspServerCard({ name, srv, onDelete, onEdit }: {
    name: string
    srv: { command: string; args?: string[]; fileTypes: string[]; rootMarkers?: string[]; disabled?: boolean }
    onDelete: () => void
    onEdit: () => void
}) {
    return (
        <div className="rounded-lg px-4 py-3 w-full relative group/card" style={{ background: 'var(--bg-card)' }}>
            <div className="flex items-start gap-3">
                <div className="mt-0.5 shrink-0" style={{ color: 'var(--text-dim)' }}>
                    <IconServer size={16} />
                </div>
                <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-0.5">
                        <span className="font-mono text-[14px] font-semibold text-[var(--text-primary)]">{name}</span>
                        {srv.disabled && (
                            <span className="text-[10px] px-1 py-0.5 rounded-sm font-medium bg-[var(--bg-input)] text-[var(--red)]">
                                disabled
                            </span>
                        )}
                    </div>
                    <div className="text-[11px] text-[var(--text-dim)] font-mono">
                        {srv.command} {(srv.args || []).join(' ')}
                    </div>
                    <div className="flex flex-wrap gap-1 mt-1">
                        {(srv.fileTypes || []).map(ft => (
                            <span key={ft} className="text-[10px] px-1.5 py-0.5 rounded-sm" style={{ background: 'var(--bg-input)', color: 'var(--text-dim)' }}>
                                {ft}
                            </span>
                        ))}
                    </div>
                </div>
                <div className="opacity-0 group-hover/card:opacity-100 transition-opacity flex gap-1 shrink-0">
                    <button onClick={onEdit} title="Edit" className="inline-flex items-center text-[var(--text-dim)] hover:text-[var(--accent)] text-[11px] px-1.5 py-0.5 cursor-pointer bg-transparent border-none rounded transition-colors">
                        <IconEdit size={13} />
                    </button>
                    <button onClick={onDelete} className="inline-flex items-center text-[var(--text-dim)] hover:text-[var(--red)] text-[11px] px-1.5 py-0.5 cursor-pointer bg-transparent border-none rounded transition-colors">
                        <IconTrash size={13} />
                    </button>
                </div>
            </div>
        </div>
    )
}

// --- Formatter Card ---

function FormatterCard({ lang, cfg, onEdit, onDelete }: {
    lang: string
    cfg: { command: string; args?: string[]; ref?: string }
    onEdit: () => void
    onDelete: () => void
}) {
    const isLsp = cfg.ref === 'lsp' || cfg.command === 'lsp'
    return (
        <div className="rounded-lg px-4 py-3 w-full relative group/card" style={{ background: 'var(--bg-card)' }}>
            <div className="flex items-start gap-3">
                <div className="mt-0.5 shrink-0 font-mono text-[12px] font-semibold" style={{ color: 'var(--accent)' }}>
                    {lang}
                </div>
                <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-0.5">
                        {isLsp ? (
                            <span className="text-[12px] text-[var(--text-dim)]">Use LSP formatting</span>
                        ) : (
                            <span className="text-[12px] text-[var(--text-primary)] font-mono">
                                {cfg.command} {(cfg.args || []).join(' ')}
                            </span>
                        )}
                    </div>
                </div>
                <div className="opacity-0 group-hover/card:opacity-100 transition-opacity flex gap-1 shrink-0">
                    <button onClick={onEdit} title="Edit" className="inline-flex items-center text-[var(--text-dim)] hover:text-[var(--accent)] text-[11px] px-1.5 py-0.5 cursor-pointer bg-transparent border-none rounded transition-colors">
                        <IconEdit size={13} />
                    </button>
                    <button onClick={onDelete} className="inline-flex items-center text-[var(--text-dim)] hover:text-[var(--red)] text-[11px] px-1.5 py-0.5 cursor-pointer bg-transparent border-none rounded transition-colors">
                        <IconTrash size={13} />
                    </button>
                </div>
            </div>
        </div>
    )
}

// --- Language dropdown options ---

const LANGUAGES = [
    'go', 'python', 'typescript', 'javascript', 'rust', 'lua',
    'shell', 'c', 'cpp', 'java', 'ruby', 'php',
    'swift', 'kotlin', 'csharp', 'scss', 'css', 'html',
    'json', 'yaml', 'markdown',
]

// --- Main Tab Component ---

export default function LspFormattersTab() {
    const scope = useStore(s => s.settingsScope)
    const setScope = useStore(s => s.setSettingsScope)
    const lspServers = useStore(s => s.lspConfigServers)
    const formatterConfig = useStore(s => s.formatterConfig)
    const loadLSPConfig = useStore(s => s.loadLSPConfig)
    const saveLSPConfig = useStore(s => s.saveLSPConfig)
    const loadFormatterConfig = useStore(s => s.loadFormatterConfig)
    const saveFormatterConfig = useStore(s => s.saveFormatterConfig)
    const loadLSPStatus = useStore(s => s.loadLSPStatus)

    const [subtab, setSubtab] = useState<'lsp' | 'formatters'>('lsp')

    // LSP modal state
    const [lspModal, setLspModal] = useState(false)
    const [editingLsp, setEditingLsp] = useState<string | null>(null)
    const [lspName, setLspName] = useState('')
    const [lspCommand, setLspCommand] = useState('')
    const [lspArgs, setLspArgs] = useState('')
    const [lspFileTypes, setLspFileTypes] = useState('')
    const [lspRootMarkers, setLspRootMarkers] = useState('')
    const [lspDisabled, setLspDisabled] = useState(false)

    // Formatter modal state
    const [fmtModal, setFmtModal] = useState(false)
    const [editingLang, setEditingLang] = useState<string | null>(null)
    const [fmtLang, setFmtLang] = useState('')
    const [fmtCommand, setFmtCommand] = useState('')
    const [fmtArgs, setFmtArgs] = useState('')

    useEffect(() => {
        loadLSPConfig(scope)
        loadFormatterConfig(scope)
    }, [scope])

    // --- LSP handlers ---

    const openLspAdd = () => {
        setEditingLsp(null)
        setLspName('')
        setLspCommand('')
        setLspArgs('')
        setLspFileTypes('')
        setLspRootMarkers('')
        setLspDisabled(false)
        setLspModal(true)
    }

    const openLspEdit = (name: string, srv: any) => {
        setEditingLsp(name)
        setLspName(name)
        setLspCommand(srv.command || '')
        setLspArgs((srv.args || []).join(' '))
        setLspFileTypes((srv.fileTypes || []).join(','))
        setLspRootMarkers((srv.rootMarkers || []).join(','))
        setLspDisabled(srv.disabled || false)
        setLspModal(true)
    }

    const handleLspSave = () => {
        const args = lspArgs.trim() ? lspArgs.split(/\s+/) : []
        const fileTypes = lspFileTypes.split(',').map(s => s.trim()).filter(Boolean)
        const rootMarkers = lspRootMarkers.split(',').map(s => s.trim()).filter(Boolean)
        const updated = {
            ...lspServers,
            [lspName]: {
                command: lspCommand.trim(),
                args,
                fileTypes,
                rootMarkers,
                disabled: lspDisabled,
            },
        }
        saveLSPConfig(scope, updated)
        setLspModal(false)
    }

    const handleLspDelete = (name: string) => {
        const updated = { ...lspServers }
        delete updated[name]
        saveLSPConfig(scope, updated)
    }

    // --- Formatter handlers ---

    const openFmtAdd = () => {
        setEditingLang(null)
        setFmtLang('')
        setFmtCommand('')
        setFmtArgs('')
        setFmtModal(true)
    }

    const openFmtEdit = (lang: string, cfg: any) => {
        setEditingLang(lang)
        setFmtLang(lang)
        setFmtCommand(cfg.command || cfg.ref || '')
        setFmtArgs((cfg.args || []).join(' '))
        setFmtModal(true)
    }

    const handleFmtSave = () => {
        const cmd = fmtCommand.trim()
        const args = fmtArgs.trim() ? fmtArgs.split(/\s+/) : []
        const entry: any = cmd === 'lsp' ? { ref: 'lsp' } : { command: cmd, args }
        const updated = { ...formatterConfig, [fmtLang]: entry }
        saveFormatterConfig(scope, updated)
        setFmtModal(false)
    }

    const handleFmtDelete = (lang: string) => {
        const updated = { ...formatterConfig }
        delete updated[lang]
        saveFormatterConfig(scope, updated)
    }

    return (
        <div>
            {/* Scope Selector */}
            <div className="flex items-center gap-2 mb-4">
                <span className="text-[11px] text-[var(--text-dim)]">Scope:</span>
                {(['global', 'project'] as const).map(s => (
                    <button
                        key={s}
                        onClick={() => setScope(s)}
                        className={`text-[11px] px-2 py-0.5 rounded-sm border cursor-pointer transition-colors ${
                            scope === s
                                ? 'bg-[var(--accent)] text-white border-[var(--accent)]'
                                : 'bg-transparent text-[var(--text-dim)] border-[var(--border)] hover:border-[var(--text-dim)]'
                        }`}
                    >
                        {s === 'global' ? 'Global' : 'Project'}
                    </button>
                ))}
                <button
                    onClick={() => loadLSPStatus()}
                    title="Refresh LSP status"
                    className="ml-auto inline-flex items-center text-[var(--text-dim)] hover:text-[var(--accent)] text-[11px] px-1.5 py-0.5 cursor-pointer bg-transparent border-none rounded transition-colors"
                >
                    <IconRefresh size={12} />
                </button>
            </div>

            {/* Subtabs */}
            <div className="flex gap-2 mb-4 border-b border-[var(--border)]">
                {(['lsp', 'formatters'] as const).map(st => (
                    <button
                        key={st}
                        onClick={() => setSubtab(st)}
                        className={`text-[13px] px-3 py-1.5 cursor-pointer bg-transparent border-none border-b-2 transition-colors ${
                            subtab === st
                                ? 'text-[var(--text-primary)] border-[var(--accent)] font-medium'
                                : 'text-[var(--text-dim)] border-transparent hover:text-[var(--text-primary)]'
                        }`}
                    >
                        {st === 'lsp' ? 'LSP Servers' : 'Formatters'}
                    </button>
                ))}
            </div>

            {/* LSP Subtab */}
            {subtab === 'lsp' && (
                <div>
                    <div className="flex items-center justify-between mb-3">
                        <span className="text-[11px] text-[var(--text-dim)]">
                            {Object.keys(lspServers).length} server{Object.keys(lspServers).length !== 1 ? 's' : ''}
                        </span>
                        <button
                            onClick={openLspAdd}
                            className="inline-flex items-center gap-1 text-[11px] px-2 py-1 rounded-sm cursor-pointer bg-[var(--accent)] text-white border-none hover:opacity-90 transition-opacity"
                        >
                            <IconPlus size={12} /> Add Server
                        </button>
                    </div>
                    <div className="flex flex-col gap-2">
                        {Object.entries(lspServers).map(([name, srv]) => (
                            <LspServerCard
                                key={name}
                                name={name}
                                srv={srv}
                                onEdit={() => openLspEdit(name, srv)}
                                onDelete={() => handleLspDelete(name)}
                            />
                        ))}
                        {Object.keys(lspServers).length === 0 && (
                            <div className="text-[12px] text-[var(--text-dim)] py-4 text-center">
                                No LSP servers configured. Add one to enable code intelligence for a language.
                            </div>
                        )}
                    </div>
                </div>
            )}

            {/* Formatters Subtab */}
            {subtab === 'formatters' && (
                <div>
                    <div className="flex items-center justify-between mb-3">
                        <span className="text-[11px] text-[var(--text-dim)]">
                            {Object.keys(formatterConfig).length} formatter{Object.keys(formatterConfig).length !== 1 ? 's' : ''}
                        </span>
                        <button
                            onClick={openFmtAdd}
                            className="inline-flex items-center gap-1 text-[11px] px-2 py-1 rounded-sm cursor-pointer bg-[var(--accent)] text-white border-none hover:opacity-90 transition-opacity"
                        >
                            <IconPlus size={12} /> Add Formatter
                        </button>
                    </div>
                    <div className="flex flex-col gap-2">
                        {Object.entries(formatterConfig).map(([lang, cfg]) => (
                            <FormatterCard
                                key={lang}
                                lang={lang}
                                cfg={cfg}
                                onEdit={() => openFmtEdit(lang, cfg)}
                                onDelete={() => handleFmtDelete(lang)}
                            />
                        ))}
                        {Object.keys(formatterConfig).length === 0 && (
                            <div className="text-[12px] text-[var(--text-dim)] py-4 text-center">
                                No formatters configured. Files will use LSP formatting when available.
                            </div>
                        )}
                    </div>
                </div>
            )}

            {/* LSP Add/Edit Modal */}
            {lspModal && (
                <Modal onClose={() => setLspModal(false)} wide>
                    <ModalHeader>{editingLsp ? `Edit ${editingLsp}` : 'Add LSP Server'}</ModalHeader>
                    <ModalBody>
                        <div className="flex flex-col gap-3">
                            <label className="flex flex-col gap-1 text-[12px]">
                                Server Name
                                <input
                                    value={lspName}
                                    onChange={e => setLspName(e.target.value)}
                                    className="bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1 text-[13px] text-[var(--text-primary)]"
                                    placeholder="e.g. gopls"
                                    disabled={!!editingLsp}
                                />
                            </label>
                            <label className="flex flex-col gap-1 text-[12px]">
                                Command
                                <input
                                    value={lspCommand}
                                    onChange={e => setLspCommand(e.target.value)}
                                    className="bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1 text-[13px] text-[var(--text-primary)]"
                                    placeholder="e.g. gopls"
                                />
                            </label>
                            <label className="flex flex-col gap-1 text-[12px]">
                                Arguments (space-separated)
                                <input
                                    value={lspArgs}
                                    onChange={e => setLspArgs(e.target.value)}
                                    className="bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1 text-[13px] text-[var(--text-primary)]"
                                    placeholder="e.g. serve"
                                />
                            </label>
                            <label className="flex flex-col gap-1 text-[12px]">
                                File Types (comma-separated)
                                <input
                                    value={lspFileTypes}
                                    onChange={e => setLspFileTypes(e.target.value)}
                                    className="bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1 text-[13px] text-[var(--text-primary)]"
                                    placeholder="e.g. .go, .mod, .sum"
                                />
                            </label>
                            <label className="flex flex-col gap-1 text-[12px]">
                                Root Markers (comma-separated)
                                <input
                                    value={lspRootMarkers}
                                    onChange={e => setLspRootMarkers(e.target.value)}
                                    className="bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1 text-[13px] text-[var(--text-primary)]"
                                    placeholder="e.g. go.mod, go.sum"
                                />
                            </label>
                            <label className="flex items-center gap-2 text-[12px]">
                                <input
                                    type="checkbox"
                                    checked={lspDisabled}
                                    onChange={e => setLspDisabled(e.target.checked)}
                                />
                                Disabled
                            </label>
                        </div>
                    </ModalBody>
                    <ModalFooter>
                        <ModalButton onClick={() => setLspModal(false)}>Cancel</ModalButton>
                        <ModalButton primary onClick={handleLspSave} disabled={!lspName.trim() || !lspCommand.trim()}>Save</ModalButton>
                    </ModalFooter>
                </Modal>
            )}

            {/* Formatter Add/Edit Modal */}
            {fmtModal && (
                <Modal onClose={() => setFmtModal(false)}>
                    <ModalHeader>{editingLang ? `Edit ${editingLang} Formatter` : 'Add Formatter'}</ModalHeader>
                    <ModalBody>
                        <div className="flex flex-col gap-3">
                            <label className="flex flex-col gap-1 text-[12px]">
                                Language
                                <select
                                    value={fmtLang}
                                    onChange={e => setFmtLang(e.target.value)}
                                    className="bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1 text-[13px] text-[var(--text-primary)]"
                                    disabled={!!editingLang}
                                >
                                    <option value="">Select language...</option>
                                    {LANGUAGES.map(l => (
                                        <option key={l} value={l}>{l}</option>
                                    ))}
                                </select>
                            </label>
                            <label className="flex flex-col gap-1 text-[12px]">
                                Command (or type "lsp" for LSP formatting)
                                <input
                                    value={fmtCommand}
                                    onChange={e => setFmtCommand(e.target.value)}
                                    className="bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1 text-[13px] text-[var(--text-primary)]"
                                    placeholder="e.g. black, prettier, or lsp"
                                />
                            </label>
                            <label className="flex flex-col gap-1 text-[12px]">
                                Arguments (space-separated)
                                <input
                                    value={fmtArgs}
                                    onChange={e => setFmtArgs(e.target.value)}
                                    className="bg-[var(--bg-input)] border border-[var(--border)] rounded px-2 py-1 text-[13px] text-[var(--text-primary)]"
                                    placeholder="e.g. --write"
                                />
                            </label>
                            <div className="text-[10px] text-[var(--text-dim)]">
                                Type <span className="font-mono">lsp</span> in the command field to use LSP formatting for this language.
                            </div>
                        </div>
                    </ModalBody>
                    <ModalFooter>
                        <ModalButton onClick={() => setFmtModal(false)}>Cancel</ModalButton>
                        <ModalButton primary onClick={handleFmtSave} disabled={!fmtLang.trim() || !fmtCommand.trim()}>Save</ModalButton>
                    </ModalFooter>
                </Modal>
            )}
        </div>
    )
}
```

- [ ] **Step 2: Run frontend build**

```bash
cd d:/git/monika/frontend && npm run build 2>&1 | tail -30
```
Fix any TypeScript errors if present.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/Settings/LspFormattersTab.tsx
git commit -m "feat: add LSP and Formatter settings UI with scope toggle"
```

---

### Task 9: SettingsPage integration

**Files:**
- Modify: `frontend/src/components/Settings/SettingsPage.tsx`

- [ ] **Step 1: Add new tab to SettingsPage**

In `frontend/src/components/Settings/SettingsPage.tsx`:

Add import:
```tsx
import LspFormattersTab from './LspFormattersTab'
```

Update the `Tab` type and `TABS` array:
```tsx
type Tab = 'agents' | 'permissions' | 'skills' | 'mcp' | 'models' | 'lsp-formatters' | 'about'

const TABS: { id: Tab; label: string; icon: React.ReactNode }[] = [
  { id: 'models', label: 'Providers', icon: <IconDatabase size={14} /> },
  { id: 'agents', label: 'Agents', icon: <IconBot size={14} /> },
  { id: 'permissions', label: 'Permissions', icon: <IconShield size={14} /> },
  { id: 'skills', label: 'Skills', icon: <IconStar size={14} /> },
  { id: 'mcp', label: 'MCP', icon: <IconPlug size={14} /> },
  { id: 'lsp-formatters', label: 'LSP & Format', icon: <IconServer size={14} /> },
  { id: 'about', label: 'About', icon: <IconInfo size={14} /> },
]
```

Add the tab content:
```tsx
          {activeTab === 'lsp-formatters' && <LspFormattersTab />}
```

- [ ] **Step 2: Run frontend build**

```bash
cd d:/git/monika/frontend && npm run build 2>&1 | tail -30
```
Fix any errors.

- [ ] **Step 3: Full Go build**

```bash
cd d:/git/monika && go build ./...
```
Expected: No errors

- [ ] **Step 4: Run all tests**

```bash
cd d:/git/monika && go test ./... -short
```
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/Settings/SettingsPage.tsx
git commit -m "feat: integrate LSP & Formatters settings tab"
```

---

## Self-Review

### Spec coverage

| Spec requirement | Task |
|-----------------|------|
| `FormatterConfig` custom `UnmarshalJSON` + `UnmarshalYAML` | Task 1 |
| `Config.LSP` + `Config.Formatters` fields | Task 1, Task 2 |
| Two-level config merge (global + project) | Task 2 (merge) |
| lsp.json migration to config.json | Task 3 |
| Extension-to-language mapping (~20 languages) | Task 1 |
| `ResolveFormatter` | Task 1 |
| `RunCLIFormatter` | Task 1 |
| WriteThrough Step 2: CLI first, LSP fallback | Task 5 |
| Config injection path (main → RegisterLSP → Manager) | Task 4 |
| LSP servers from config (ResolveServersFromConfig) | Task 4 |
| API: `GetLSPConfig(scope)`, `SaveLSPConfig(scope)` | Task 6 |
| API: `GetFormatterConfig(scope)`, `SaveFormatterConfig(scope)` | Task 6 |
| Frontend scope selector (Global / Project) | Task 8 |
| Frontend LSP server list + add/edit/delete | Task 8 |
| Frontend Formatter list + add/edit/delete | Task 8 |
| SettingsPage tab integration | Task 9 |
| Empty formatters map → LSP formatting (existing behavior) | Task 5 (fallback logic) |
| Formatter command not found → error → fallback LSP | Task 5 |
| Project has lsp.json but no .monika/ dir → migration creates it | Task 3 |
| Formatter takes too long → context cancellation | Task 1 (exec.CommandContext) |

### Placeholder scan

No TBDs, TODOs, "add appropriate error handling", or "similar to Task N" patterns found.

### Type consistency

- `FormatterConfig` defined in `internal/lsp` package → used by `config.Config.Formatters map[string]lsp.FormatterConfig` → used by `Manager.formatters map[string]FormatterConfig` → used by `ResolveFormatter`
- `ServerConfig` already in `internal/lsp` → used by `LSPConfig.Servers map[string]lsp.ServerConfig` → passed through to `ResolveServersFromConfig`
- `lsp.LSPServerStatus` returned by `GetLSPStatus()` — existing, unchanged
- Frontend types match Go types: `LSPServerStatus` already exists, `FormatterEntry` added

---

**Plan complete and saved to `docs/superpowers/plans/2026-06-07-formatter-integration.md`. Two execution options:**

1. **Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

2. **Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
