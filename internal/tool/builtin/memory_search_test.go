package builtin

import (
	"context"
	"strings"
	"testing"

	"monika/internal/memory"
)

func TestMemorySearchOutputContainsSnippet(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := memory.NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	err = store.WriteFile(memory.ScopeProject, memory.CategoryLesson, "CORS Fix",
		"Wails v3 dev mode CORS configuration issue.", []string{"cors"}, "high")
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	tool := NewMemorySearch(store)
	args := []byte(`{"query": "CORS"}`)
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "snippet:") {
		t.Errorf("output should contain snippet line, got: %s", result.Content)
	}
}
