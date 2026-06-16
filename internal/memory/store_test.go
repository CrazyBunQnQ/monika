package memory

import (
	"strings"
	"testing"
)

func TestKBStoreWriteAndSearch(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	err = store.WriteFile(ScopeProject, CategoryLesson, "Test Lesson",
		"This is a test lesson about goroutines and channels.", []string{"go", "concurrency"}, "high")
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	results, err := store.Search("goroutines channels", ScopeProject, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Title != "Test Lesson" {
		t.Errorf("got '%s'", results[0].Title)
	}
}

func TestKBStoreBuildMemoryBlock(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	store.WriteFile(ScopeGlobal, CategoryProfile, "User Profile", "Test profile.", nil, "high")
	store.WriteFile(ScopeGlobal, CategoryKnowledge, "Core", "Global knowledge.", nil, "high")
	store.WriteFile(ScopeProject, CategoryKnowledge, "Proj", "Project knowledge.", nil, "high")

	block := store.BuildMemoryBlock()
	if block == "" {
		t.Fatal("expected non-empty block")
	}
	if !strings.Contains(block, "<global_memory>") {
		t.Error("missing <global_memory>")
	}
	if !strings.Contains(block, "<project_memory>") {
		t.Error("missing <project_memory>")
	}
}

func TestKBStoreSoftDelete(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	store.WriteFile(ScopeProject, CategoryTopic, "To Delete",
		"Will be deleted.", nil, "low")
	store.SoftDelete(ScopeProject, "wiki/topics/to-delete.md")

	results, _ := store.Search("deleted", ScopeProject, 5)
	if len(results) != 0 {
		t.Errorf("expected 0 results after soft delete, got %d", len(results))
	}
}
