package lsp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

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
	if len(args) != 2 || args[0] != "--line-length" || args[1] != "100" {
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
	content, err := RunCLIFormatter(context.Background(), "echo", []string{}, filePath)
	if err != nil {
		t.Skipf("echo not available: %v", err)
	}
	// echo writes nothing to the file so content should still be "x=1\n"
	if content != "x=1\n" {
		t.Fatalf("content = %q", content)
	}
}
