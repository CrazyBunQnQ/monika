package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyWorktree_Deleted(t *testing.T) {
	dir := t.TempDir()
	sm := NewSessionManager(dir, "/tmp/test-project")
	s, err := sm.New("gpt-4", "openai")
	if err != nil {
		t.Fatal(err)
	}
	s.WorktreePath = filepath.Join(dir, "nonexistent-worktree")
	sm.Save(s)

	result := VerifyWorktree(sm, s.ID)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Deleted {
		t.Error("expected Deleted=true for nonexistent path")
	}
	if result.Path != s.WorktreePath {
		t.Errorf("expected path %q, got %q", s.WorktreePath, result.Path)
	}
}

func TestVerifyWorktree_NoBinding(t *testing.T) {
	dir := t.TempDir()
	sm := NewSessionManager(dir, "/tmp/test-project")
	s, err := sm.New("gpt-4", "openai")
	if err != nil {
		t.Fatal(err)
	}
	// WorktreePath is empty by default
	sm.Save(s)

	result := VerifyWorktree(sm, s.ID)
	if result != nil {
		t.Error("expected nil result for unbound session")
	}
}

func TestVerifyWorktree_Exists(t *testing.T) {
	dir := t.TempDir()
	sm := NewSessionManager(dir, "/tmp/test-project")
	s, err := sm.New("gpt-4", "openai")
	if err != nil {
		t.Fatal(err)
	}
	// Create an actual directory
	wtPath := filepath.Join(dir, "existing-worktree")
	os.MkdirAll(wtPath, 0755)
	s.WorktreePath = wtPath
	sm.Save(s)

	result := VerifyWorktree(sm, s.ID)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Deleted {
		t.Error("expected Deleted=false for existing path")
	}
}

func TestListWorktreesScanBindings(t *testing.T) {
	// Test the bindingMap logic directly
	sm1 := NewSessionManager(t.TempDir(), "/proj")
	s1, _ := sm1.New("gpt-4", "openai")
	s1.Title = "Session A"
	s1.WorktreePath = "/proj/.worktrees/feature-x"
	sm1.Save(s1)

	sm2 := NewSessionManager(t.TempDir(), "/proj")
	s2, _ := sm2.New("gpt-4", "openai")
	s2.Title = "Session B"
	s2.WorktreePath = "/proj/.worktrees/feature-x" // same worktree
	sm2.Save(s2)

	// Build binding map manually to test the logic
	bindingMap := make(map[string][]SessionRef)
	for _, sm := range []*SessionManager{sm1, sm2} {
		infos, _ := sm.List()
		for _, si := range infos {
			if si.WorktreePath != "" {
				bindingMap[si.WorktreePath] = append(bindingMap[si.WorktreePath], SessionRef{
					ID:    si.ID,
					Title: si.Title,
				})
			}
		}
	}
	refs, ok := bindingMap["/proj/.worktrees/feature-x"]
	if !ok {
		t.Fatal("expected feature-x in binding map")
	}
	if len(refs) != 2 {
		t.Errorf("expected 2 sessions bound, got %d", len(refs))
	}
}

func TestAttachDetachRoundTrip(t *testing.T) {
	dir := t.TempDir()
	sm := NewSessionManager(dir, "/tmp/project")
	s, _ := sm.New("gpt-4", "openai")

	// Create a fake worktree dir
	wtPath := filepath.Join(dir, "my-worktree")
	os.MkdirAll(filepath.Join(wtPath, ".git"), 0755) // minimal git marker

	// Simulate Attach
	s.WorktreePath = wtPath
	sm.Save(s)

	loaded, _ := sm.Load(s.ID)
	if loaded.WorktreePath != wtPath {
		t.Fatalf("attach failed: %q != %q", loaded.WorktreePath, wtPath)
	}

	// Simulate Detach
	s.WorktreePath = ""
	sm.Save(s)

	loaded2, _ := sm.Load(s.ID)
	if loaded2.WorktreePath != "" {
		t.Fatal("detach failed: worktree path not cleared")
	}
}

func TestParseWorktreeList(t *testing.T) {
	output := `worktree D:/project/main
HEAD 1234567abc
branch refs/heads/main

worktree D:/project/.worktrees/feature-x
HEAD 89abcdef
branch refs/heads/feature-x

worktree D:/project/.worktrees/fix-bug-a1b2c3d4
detached
`
	worktrees, err := parseWorktreeList(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(worktrees) != 3 {
		t.Fatalf("expected 3 worktrees, got %d", len(worktrees))
	}
	if worktrees[0].Path != "D:/project/main" {
		t.Errorf("expected path %q, got %q", "D:/project/main", worktrees[0].Path)
	}
	if worktrees[0].Branch != "main" {
		t.Errorf("expected branch 'main', got %q", worktrees[0].Branch)
	}
	if worktrees[1].Branch != "feature-x" {
		t.Errorf("expected branch 'feature-x', got %q", worktrees[1].Branch)
	}
	if worktrees[2].Branch != "" {
		t.Errorf("expected empty branch for detached, got %q", worktrees[2].Branch)
	}
}
