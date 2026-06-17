package builtin

import (
	"context"
	"strings"
	"testing"

	"monika/internal/memory"
)

func TestMemoryWriteProfileCategory(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := memory.NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	tool := NewMemoryWrite(store)
	args := []byte(`{"title": "User Profile", "content": "Test profile.", "category": "profile", "scope": "global"}`)
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}

	// 验证 profile.md 确实被写入
	content, err := store.ReadFile(memory.ScopeGlobal, "wiki/profile.md")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(content, "Test profile.") {
		t.Errorf("profile not written, got: %s", content)
	}
}

func TestMemoryWriteReturnMentionsUpdate(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := memory.NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	tool := NewMemoryWrite(store)
	args := []byte(`{"title": "Test", "content": "Test.", "category": "lesson"}`)
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result.Content, "memory_update") {
		t.Errorf("return message should mention memory_update for existing memories, got: %s", result.Content)
	}
}
