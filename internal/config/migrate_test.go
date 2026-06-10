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
