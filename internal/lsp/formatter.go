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
	".go":      "go",
	".mod":     "go",
	".sum":     "go",
	".py":      "python",
	".pyi":     "python",
	".ts":      "typescript",
	".tsx":     "typescript",
	".js":      "javascript",
	".jsx":     "javascript",
	".mjs":     "javascript",
	".cjs":     "javascript",
	".rs":      "rust",
	".lua":     "lua",
	".sh":      "shell",
	".bash":    "shell",
	".zsh":     "shell",
	".c":       "c",
	".h":       "c",
	".cpp":     "cpp",
	".cc":      "cpp",
	".cxx":     "cpp",
	".hpp":     "cpp",
	".hxx":     "cpp",
	".java":    "java",
	".rb":      "ruby",
	".rake":    "ruby",
	".gemspec": "ruby",
	".php":     "php",
	".swift":   "swift",
	".kt":      "kotlin",
	".kts":     "kotlin",
	".cs":      "csharp",
	".scss":    "scss",
	".sass":    "scss",
	".css":     "css",
	".less":    "css",
	".html":    "html",
	".htm":     "html",
	".json":    "json",
	".jsonc":   "json",
	".yaml":    "yaml",
	".yml":     "yaml",
	".md":      "markdown",
	".mdx":     "markdown",
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

	resolved := ResolveCommand(command, cmd.Dir)
	if resolved != command {
		cmd.Path = resolved
		cmd.Args = append([]string{resolved}, fullArgs...)
	}

	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
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
