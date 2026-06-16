package dbdiscovery

import (
	"os"
	"path/filepath"
)

func DetectRuntime(projectDir string) string {
	if fileExists(projectDir, "package.json") {
		return "node"
	}
	if fileExists(projectDir, "requirements.txt") || fileExists(projectDir, "pyproject.toml") || fileExists(projectDir, "Pipfile") {
		return "python"
	}
	if fileExists(projectDir, "Gemfile") {
		return "ruby"
	}
	return ""
}

func fileExists(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}
