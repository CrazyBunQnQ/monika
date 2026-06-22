package builtin

import (
	"context"
	"strings"
	"testing"

	"monika/internal/memory"
)

func TestMemoryReadExistingFile(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := memory.NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	err = store.WriteFile(memory.ScopeProject, memory.CategoryLesson, "Read Test",
		"Full content of this memory.", []string{"test"}, "high")
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	results, err := store.Search("Read Test", memory.ScopeProject, 1)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	path := results[0].Path

	tool := NewMemoryRead(store)
	args := []byte(`{"path": "` + path + `"}`)
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "Full content of this memory.") {
		t.Errorf("expected full content, got: %s", result.Content)
	}
}

func TestMemoryReadNonExistent(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := memory.NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	tool := NewMemoryRead(store)
	args := []byte(`{"path": "wiki/lessons/nonexistent.md"}`)
	result, _ := tool.Execute(context.Background(), args)
	if !result.IsError {
		t.Error("expected error for nonexistent path")
	}
	if !strings.Contains(result.Content, "not found") {
		t.Errorf("error should mention 'not found', got: %s", result.Content)
	}
}

func TestMemoryReadPathTraversal(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := memory.NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	tool := NewMemoryRead(store)
	args := []byte(`{"path": "../../etc/passwd"}`)
	result, _ := tool.Execute(context.Background(), args)
	if !result.IsError {
		t.Error("expected error for path traversal")
	}
}
