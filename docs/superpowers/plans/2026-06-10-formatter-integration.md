# Formatter Integration & Config Consolidation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Integrate CLI formatters into Monika's file editing pipeline with CLI-first fallback-to-LSP strategy, consolidate `lsp.json` into `config.json`, and add frontend settings UI for LSP and formatter configuration.

**Architecture:** Config structs gain `LSP` and `Formatters` fields. On `Load()`, `lsp.json` is migrated into `config.json.lsp.servers`. Manager gains a `formatters` field used in `WriteThrough` to try CLI formatter first then fallback to LSP. Four new `App` API methods (Get/Save with scope) expose config to the frontend. Settings page gains LSP + Formatters tabs with Global/Project scope toggle.

**Tech Stack:** Go 1.22+, Wails v3, React/TypeScript, zustand, gopkg.in/yaml.v3

---

## File Structure

### Created files

| File | Responsibility |
|------|--------------|
| `internal/config/migrate.go` | lsp.json → config.json migration logic |
| `internal/config/migrate_test.go` | Migration tests |
| `internal/lsp/formatter.go` | `extToLang` map, `ResolveFormatter`, `RunCLIFormatter` |
| `internal/lsp/formatter_test.go` | Formatter unit tests |
| `frontend/src/components/Settings/LspFormattersTab.tsx` | LSP + Formatters settings UI with scope toggle |

### Modified files

| File | Change summary |
|------|---------------|
| `internal/config/config.go` | Add `LSPConfig`, `FormatterConfig`, update `Config`, `merge()`, `Load()` calls migration |
| `internal/config/config_test.go` | Tests for new structs + merge behavior |
| `internal/lsp/defaults.go` | Add yaml tags to `ServerConfig` |
| `internal/lsp/config.go` | Add `ResolveServersFromConfig(workdir, userServers)` |
| `internal/lsp/manager.go` | Add `formatters` field, update `WriteThrough` Step 2 |
| `internal/lsp/tool.go` | `NewLSPTool` accepts formatters config, pass to Manager |
| `internal/tool/builtin/lsp.go` | `NewLSPTool` signature change |
| `internal/tool/builtin/register.go` | `RegisterLSP` accepts `*config.Config` |
| `main.go` | Pass `Config` to `RegisterLSP` |
| `internal/api/app.go` | Add `GetLSPConfig`, `SaveLSPConfig`, `GetFormatterConfig`, `SaveFormatterConfig`, update project-open path |
| `frontend/src/store/index.ts` | New types, state, actions for settings scope/LSP/formatters |
| `frontend/src/components/Settings/SettingsPage.tsx` | Add LSP + Formatters tab |

---

### Task 1: FormatterConfig struct with custom unmarshalers

**Files:**
- Modify: `internal/config/config.go`
- Create: `internal/config/config_test.go` (add test functions)

- [ ] **Step 1: Write the failing test for FormatterConfig**

Insert at end of `internal/config/config_test.go`:

```go
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
```

Add imports at top of config_test.go:
```go
import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd d:/git/monika && go test ./internal/config/ -run "TestFormatterConfig" -v
```
Expected: FAIL with "undefined: FormatterConfig"

- [ ] **Step 3: Add FormatterConfig struct and unmarshalers**

In `internal/config/config.go`, add after the `ToolsConfig` block (after line 86):

```go
type LSPConfig struct {
	Servers map[string]lsp.ServerConfig `yaml:"servers" json:"servers"`
}

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
```

Add `"monika/internal/lsp"` to imports in config.go.

Update `Config` struct to include new fields (add after `Tools` field, line 25):
```go
	LSP       LSPConfig                 `yaml:"lsp" json:"lsp"`
	Formatters map[string]FormatterConfig `yaml:"formatters" json:"formatters"`
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd d:/git/monika && go test ./internal/config/ -run "TestFormatterConfig" -v
```
Expected: 6 PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat: add FormatterConfig with string shorthand unmarshalers"
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

- [ ] **Step 3: Add merge logic for LSP and Formatters**

In `internal/config/config.go`, `merge()` function, add before the closing `}` (before line 255):

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
			dst.Formatters = make(map[string]FormatterConfig, len(src.Formatters))
		}
		for lang, fmt := range src.Formatters {
			dst.Formatters[lang] = fmt
		}
	}
```

- [ ] **Step 4: Run tests**

```bash
cd d:/git/monika && go test ./internal/config/ -run "TestLoadMergesLSP|TestLoadMergesFormatter|TestLoadProjectFormatter" -v
```
Expected: 3 PASS

- [ ] **Step 5: Run existing config tests to check no regression**

```bash
cd d:/git/monika && go test ./internal/config/ -v
```
Expected: ALL PASS

- [ ] **Step 6: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go internal/lsp/defaults.go
git commit -m "feat: add LSPConfig to Config, update merge(), add yaml tags to ServerConfig"
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
	"encoding/json"
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

	// Read merged config
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

	// Check lsp.json was renamed
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
	// Should not create .monika dir if neither lsp.json nor config.json exist
}
```

- [ ] **Step 2: Run test to verify it fails**

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
			// No config.json yet, create one
			os.MkdirAll(filepath.Dir(configPath), 0o755)
			configData = []byte("{}")
		} else {
			return fmt.Errorf("migrate lsp.json: %w", err)
		}
	}

	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(configData, &cfg); err != nil {
		return fmt.Errorf("migrate lsp.json: %w", err)
	}

	serversData, _ := json.Marshal(servers)
	cfg["lsp"] = json.RawMessage(fmt.Sprintf(`{"servers":%s}`, serversData))

	out, err := json.MarshalIndent(cfg, "", "  ")
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
Expected: 3 PASS

- [ ] **Step 5: Wire migration into Load()**

In `internal/config/config.go`, `Load()` function, add after the project-level merge block (after line 118, before `return cfg, nil`):

```go
	if opts.ProjectDir != "" {
		if err := migrateLSPJSON(opts.ProjectDir); err != nil {
			fmt.Fprintf(os.Stderr, "[monika] lsp.json migration: %v\n", err)
		}
	}
```

Add `"fmt"` to imports in config.go.

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

### Task 4: Extension-to-language mapping + ResolveFormatter

**Files:**
- Create: `internal/lsp/formatter.go`
- Create: `internal/lsp/formatter_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/lsp/formatter_test.go`:

```go
package lsp

import (
	"testing"
)

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
```

The test imports config for FormatterConfig type. Update to use the local package definition:
```go
package lsp

import (
	"testing"
)

// formatterCfg is a local test alias. In production, ResolveFormatter accepts
// FormatterConfig from the config package. We define it inline for tests.
type formatterCfg struct {
	Command string
	Args    []string
	Ref     string
}
```

Wait — the actual `FormatterConfig` is defined in the `config` package. The `lsp` package will import `config`. But to avoid an import cycle (`config` already imports `lsp` for `ServerConfig`), we should define the resolver to accept a minimal interface or keep the formatter config type self-contained.

**Design decision**: The `lsp` package cannot import `config` (would create a cycle since `config` imports `lsp` for `LSPConfig.Servers`). So `ResolveFormatter` should accept `map[string]FormatterConfig` where `FormatterConfig` is defined locally in the `lsp` package, OR accept an interface.

**Solution**: Move `FormatterConfig` to the `lsp` package. This avoids the cycle since `config` already imports `lsp`. Update the `config` package to use `lsp.FormatterConfig` instead of its own.

This changes the plan: `FormatterConfig` lives in `internal/lsp/formatter.go`, not `internal/config/config.go`.

Let me re-plan this properly. The config package struct becomes:

```go
Formatters map[string]lsp.FormatterConfig `yaml:"formatters" json:"formatters"`
```

This requires `config` to import `lsp`, which it already does (for `LSPConfig`). No cycle.

And the `lsp` package defines `FormatterConfig` locally. The unmarshalers stay in the `lsp` package.

Updated files:
- `internal/lsp/formatter.go` — defines `FormatterConfig`, unmarshalers, `extToLang`, `ResolveFormatter`, `RunCLIFormatter`
- `internal/config/config.go` — uses `lsp.FormatterConfig`, drops its own definition

Let me rewrite Task 1 and Task 4 accordingly.

Actually, I realize that `FormatterConfig.UnmarshalJSON` requires `encoding/json` and the type needs to be in the same package for the custom unmarshaler to work. Since `config.Load()` uses `json.Unmarshal` on the `Config` struct, and `Config.Formatters` is `map[string]lsp.FormatterConfig`, the custom `UnmarshalJSON` on `lsp.FormatterConfig` will be called automatically by the json package.

So the plan is:
1. Define `FormatterConfig` in `internal/lsp/formatter.go` with custom unmarshalers
2. `config.Config.Formatters` becomes `map[string]lsp.FormatterConfig`
3. `ResolveFormatter` lives in the same file, accepting `map[string]FormatterConfig`
4. Tests go in `internal/lsp/formatter_test.go`

This is cleaner. Let me finalize the plan with this adjustment.

---

Let me restart the plan for clarity with this adjusted architecture. Actually, I already wrote the full plan for tasks 5-13. Let me just adjust task 1 and task 4 to account for FormatterConfig living in the lsp package.

Wait, but FormatterConfig.UnmarshalYAML needs `gopkg.in/yaml.v3`, and the `lsp` package currently doesn't depend on yaml. That's fine — we add the import.

But actually, the custom YAML unmarshaler requires the type to be in the package being unmarshaled. Since `Config` is in the `config` package and uses `yaml.Unmarshal`, and `Config.Formatters` is `map[string]lsp.FormatterConfig`, the yaml.v3 library should call `UnmarshalYAML` on `lsp.FormatterConfig` if it implements the interface. Let me verify.

yaml.v3 uses interface check: `type Unmarshaler interface { UnmarshalYAML(value *yaml.Node) error }`

If `lsp.FormatterConfig` implements this interface, yaml.v3 should call it during unmarshal. Yes, this works.

OK, final plan with adjusted architecture:

**Task 1**: Define `FormatterConfig` in `internal/lsp/formatter.go` with unmarshalers. Update `config.Config.Formatters` to use `lsp.FormatterConfig`. Tests in `internal/lsp/formatter_test.go`.
**Task 2**: LSPConfig + merge() + ServerConfig yaml tags (unchanged).
**Task 3**: Migration (unchanged).
**Task 4**: extToLang + ResolveFormatter + RunCLIFormatter (same file, add to formatter.go).

Tasks 1 and 4 can be combined into a single task since they're in the same file.
