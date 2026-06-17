package builtin

import (
	"context"
	"strings"
	"testing"

	"monika/internal/memory"
)

func TestMemoryUpdateExistingFile(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := memory.NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	err = store.WriteFile(memory.ScopeProject, memory.CategoryLesson, "Update Me",
		"Original content.", []string{"test"}, "high")
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	results, err := store.Search("Update Me", memory.ScopeProject, 1)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	path := results[0].Path

	tool := NewMemoryUpdate(store)
	args := []byte(`{"path": "` + path + `", "content": "# Update Me\n\nMerged and updated content."}`)
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "updated") {
		t.Errorf("expected success message with 'updated', got: %s", result.Content)
	}

	// 验证实际写入
	read, _ := store.ReadFile(memory.ScopeProject, path)
	if !strings.Contains(read, "Merged and updated content.") {
		t.Errorf("content not actually written, got: %s", read)
	}
}

func TestMemoryUpdateNonExistent(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := memory.NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	tool := NewMemoryUpdate(store)
	args := []byte(`{"path": "wiki/lessons/nonexistent.md", "content": "test"}`)
	result, _ := tool.Execute(context.Background(), args)
	if !result.IsError {
		t.Error("expected error for nonexistent path")
	}
	if !strings.Contains(result.Content, "memory_write") {
		t.Errorf("error should guide to memory_write, got: %s", result.Content)
	}
}

func TestMemoryUpdateProfileOverflowWarning(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := memory.NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	// 先创建 profile.md
	err = store.WriteFile(memory.ScopeGlobal, memory.CategoryProfile, "User Profile",
		"Short profile.", nil, "medium")
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// 生成超长内容（超过 1500 字符）
	longContent := strings.Repeat("a", 2000)

	tool := NewMemoryUpdate(store)
	args := []byte(`{"path": "wiki/profile.md", "scope": "global", "content": "` + longContent + `"}`)
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// 超限时应写入成功但返回警告
	if result.IsError {
		t.Fatalf("should not be error, got: %s", result.Content)
	}
	if !strings.Contains(strings.ToLower(result.Content), "exceeds") && !strings.Contains(strings.ToLower(result.Content), "limit") {
		t.Errorf("should warn about overflow, got: %s", result.Content)
	}
}
