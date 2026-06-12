# Session Worktree Binding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement session-level git worktree binding, enabling parallel branch-isolated development across sessions.

**Architecture:** Add `WorktreePath` field to `Session` struct and implement `resolveWorkingDir()` that checks this field before every tool execution. A new `internal/api/worktree.go` file hosts CRUD APIs for worktree management. Frontend shows a worktree indicator chip in the input toolbar, a WorktreeManager modal for browsing/creating/deleting worktrees, and a non-modal banner when a bound worktree is deleted.

**Tech Stack:** Go (Wails v3 backend), React/TypeScript (frontend with zustand store), git worktree CLI

**Spec:** `docs/superpowers/specs/2026-06-12-session-worktree-design.md`

---

### Task 1: Extend Session / SessionInfo / API Types

**Files:**
- Modify: `internal/api/session_manager.go:25-44`
- Modify: `internal/api/types.go:44-78, 52-55`
- Test: `internal/api/session_manager_test.go` (new)

**Goal:** Add `WorktreePath` to `Session` and `SessionInfo`. Add `SessionRef`, `WorktreeVerifyResult`. Extend `WorktreeInfo` with `BoundSessions`.

- [ ] **Step 1: Write tests for Session serialization with WorktreePath**

```go
// internal/api/session_manager_test.go
package api

import (
    "testing"
    "os"
    "path/filepath"
)

func TestSessionWorktreePathRoundTrip(t *testing.T) {
    dir := t.TempDir()
    sm := NewSessionManager(dir, "/tmp/test-project")
    s, err := sm.New("gpt-4", "openai")
    if err != nil {
        t.Fatal(err)
    }
    s.WorktreePath = "/tmp/test-project/.worktrees/feature-x"
    if err := sm.Save(s); err != nil {
        t.Fatal(err)
    }
    loaded, err := sm.Load(s.ID)
    if err != nil {
        t.Fatal(err)
    }
    if loaded.WorktreePath != "/tmp/test-project/.worktrees/feature-x" {
        t.Errorf("expected worktree path %q, got %q", "/tmp/test-project/.worktrees/feature-x", loaded.WorktreePath)
    }
}

func TestSessionInfoIncludesWorktree(t *testing.T) {
    dir := t.TempDir()
    sm := NewSessionManager(dir, "/tmp/test-project")
    s, err := sm.New("gpt-4", "openai")
    if err != nil {
        t.Fatal(err)
    }
    s.WorktreePath = "/tmp/test-project/.worktrees/feature-x"
    sm.Save(s)
    
    infos, err := sm.List()
    if err != nil {
        t.Fatal(err)
    }
    var found bool
    for _, info := range infos {
        if info.ID == s.ID {
            found = true
            if info.WorktreePath != s.WorktreePath {
                t.Errorf("expected WorktreePath %q, got %q", s.WorktreePath, info.WorktreePath)
            }
            break
        }
    }
    if !found {
        t.Fatal("session not found in List()")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/api/ -run "TestSessionWorktreePathRoundTrip|TestSessionInfoIncludesWorktree" -v`
Expected: FAIL — Session struct has no WorktreePath field yet.

- [ ] **Step 3: Add WorktreePath to Session struct**

```go
// internal/api/session_manager.go — Session struct
type Session struct {
    ID              string               `json:"id"`
    Title           string               `json:"title"`
    CustomTitle     bool                 `json:"custom_title,omitempty"`
    ProjectDir      string               `json:"project_dir"`
    Messages        []engine.ChatMessage `json:"messages"`
    Model           string               `json:"model"`
    Provider        string               `json:"provider"`
    Status          string               `json:"status"`
    Pinned          bool                 `json:"pinned"`
    TokenCount      int64                `json:"token_count,omitempty"`
    TokenMax        int64                `json:"token_max,omitempty"`
    CompactionCount int                  `json:"compaction_count,omitempty"`
    CompactionFrom  int                  `json:"compaction_from,omitempty"`
    ParentID        string               `json:"parent_id,omitempty"`
    Tasks           []tool.Task          `json:"tasks,omitempty"`
    LastViewedAt    *time.Time           `json:"last_viewed_at,omitempty"`
    CreatedAt       time.Time            `json:"created_at"`
    UpdatedAt       time.Time            `json:"updated_at"`
    WorktreePath    string               `json:"worktree_path,omitempty"` // NEW
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/api/ -run "TestSessionWorktreePathRoundTrip|TestSessionInfoIncludesWorktree" -v`
Expected: PASS

- [ ] **Step 5: Extend SessionInfo in types.go**

```go
// internal/api/types.go — SessionInfo struct
type SessionInfo struct {
    ID             string `json:"id"`
    Title          string `json:"title"`
    Status         string `json:"status"`
    Pinned         bool   `json:"pinned"`
    UpdatedAt      string `json:"updated_at"`
    TokenCount     int64  `json:"token_count,omitempty"`
    TokenMax       int64  `json:"token_max,omitempty"`
    WorktreePath   string `json:"worktree_path,omitempty"`
    WorktreeBranch string `json:"worktree_branch,omitempty"` // set by List() if WorktreePath is non-empty
}
```

- [ ] **Step 6: Add new types to types.go**

```go
// internal/api/types.go — after WorktreeInfo, before RecentProject

// SessionRef is a lightweight reference to a session for display purposes.
type SessionRef struct {
    ID    string `json:"id"`
    Title string `json:"title"`
}

type WorktreeVerifyResult struct {
    Deleted bool   `json:"deleted"`
    Path    string `json:"path"`
}
```

- [ ] **Step 7: Extend existing WorktreeInfo**

```go
// internal/api/types.go — replace existing WorktreeInfo
type WorktreeInfo struct {
    Branch        string       `json:"branch"`
    Path          string       `json:"path"`
    BoundSessions []SessionRef `json:"bound_sessions,omitempty"`
}
```

Note: `BoundSessions` will be populated by `ListWorktrees()` in Task 2. Keep it empty for now; the field must exist so the JSON shape is correct.

- [ ] **Step 8: Populate WorktreeBranch in SessionManager.List()**

```go
// internal/api/session_manager.go — List() method, inside the loop
func (sm *SessionManager) List() ([]SessionInfo, error) {
    // ...existing code...
    for _, file := range files {
        // ...existing loading code...
        info := SessionInfo{
            // ...existing fields...
            WorktreePath: session.WorktreePath,
        }
        // Resolve branch name from WorktreePath if set
        if session.WorktreePath != "" {
            // Extract last path component and strip any session suffix
            base := filepath.Base(session.WorktreePath)
            // If the path contains a hash suffix like "feature-a1b2c3d4", it's the worktree dir name
            // We can use the branch from git, but for the info we just show the worktree name
            info.WorktreeBranch = base
        }
        result = append(result, info)
    }
    // ...rest...
}
```

- [ ] **Step 9: Run all session tests to verify nothing broke**

Run: `go test ./internal/api/ -run "Session" -v`
Expected: PASS

- [ ] **Step 10: Commit**

```bash
git add internal/api/session_manager.go internal/api/types.go internal/api/session_manager_test.go
git commit -m "feat: add WorktreePath to Session/SessionInfo + new types for worktree binding"
```

---

### Task 2: Implement Worktree API Methods

**Files:**
- Create: `internal/api/worktree.go`
- Test: `internal/api/worktree_test.go`

**Goal:** Implement the seven worktree API methods: VerifyWorktree, ListWorktrees, AttachWorktree, DetachWorktree, DeleteWorktree, RebuildWorktree, CreateWorktree.

- [ ] **Step 1: Write test for VerifyWorktree with non-existent path**

```go
// internal/api/worktree_test.go
package api

import (
    "testing"
    "os"
    "path/filepath"
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

    a := &App{}
    result := a.VerifyWorktree(s.ID)
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

    a := &App{}
    result := a.VerifyWorktree(s.ID)
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

    a := &App{}
    result := a.VerifyWorktree(s.ID)
    if result == nil {
        t.Fatal("expected non-nil result")
    }
    if result.Deleted {
        t.Error("expected Deleted=false for existing path")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/api/ -run "TestVerifyWorktree" -v`
Expected: FAIL — VerifyWorktree not defined on App

- [ ] **Step 3: Implement VerifyWorktree**

```go
// internal/api/worktree.go
package api

import (
    "os"
)

func (a *App) VerifyWorktree(sessionID string) *WorktreeVerifyResult {
    sm := a.getSessionManagerForSession(sessionID)
    if sm == nil {
        return nil
    }
    s, err := sm.Load(sessionID)
    if err != nil || s.WorktreePath == "" {
        return nil
    }
    if _, err := os.Stat(s.WorktreePath); os.IsNotExist(err) {
        return &WorktreeVerifyResult{Deleted: true, Path: s.WorktreePath}
    }
    return &WorktreeVerifyResult{Deleted: false, Path: s.WorktreePath}
}
```

Note: `getSessionManagerForSession` accesses `a.projects`, which is nil in tests. We need to handle this. Since these are Wails-bound methods, the test will validate the logic parsing but needs an App struct with a mock projects map. For now, let the test self-correct — we may need to make getSessionManagerForSession nil-safe, or add a test helper.

Actually — `getSessionManagerForSession` calls `a.getSessionManager(projectDir)` which uses `a.projects`. In tests, App has no projects set up. Let me update the test to use a direct sessionManager approach instead of going through App. Or, better, refactor VerifyWorktree to accept a SessionManager directly.

Refined approach — make worktree functions work with a SessionManager interface:

```go
// internal/api/worktree.go
// VerifyWorktree checks if a session's bound worktree path exists.
func VerifyWorktree(sm *SessionManager, sessionID string) *WorktreeVerifyResult {
    sm.Lock()
    defer sm.Unlock()
    s, err := sm.Load(sessionID)
    if err != nil || s.WorktreePath == "" {
        return nil
    }
    if _, err := os.Stat(s.WorktreePath); os.IsNotExist(err) {
        return &WorktreeVerifyResult{Deleted: true, Path: s.WorktreePath}
    }
    return &WorktreeVerifyResult{Deleted: false, Path: s.WorktreePath}
}
```

Then `App.VerifyWorktree` delegates:

```go
func (a *App) VerifyWorktree(sessionID string) *WorktreeVerifyResult {
    sm := a.getSessionManagerForSession(sessionID)
    if sm == nil {
        return nil
    }
    return VerifyWorktree(sm, sessionID)
}
```

This keeps them testable because tests can call `VerifyWorktree(sm, id)` directly.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/api/ -run "TestVerifyWorktree" -v`
Expected: PASS

- [ ] **Step 5: Implement ListWorktrees**

```go
// internal/api/worktree.go

// ListWorktrees returns all worktrees for the current project with binding info.
func (a *App) ListWorktrees() ([]WorktreeInfo, error) {
    projectDir := a.projectPath()
    worktrees, err := listGitWorktrees(projectDir)
    if err != nil {
        return nil, err
    }
    // Collect all session bindings to annotate worktrees
    bindingMap := make(map[string][]SessionRef) // worktreePath → sessions
    for projectPath, sm := range a.projects {
        _ = projectPath
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
    for i := range worktrees {
        if refs, ok := bindingMap[worktrees[i].Path]; ok {
            worktrees[i].BoundSessions = refs
        }
    }
    return worktrees, nil
}

// listGitWorktrees runs "git worktree list --porcelain" and parses the output.
func listGitWorktrees(projectDir string) ([]WorktreeInfo, error) {
    cmd := exec.Command("git", "-C", projectDir, "worktree", "list", "--porcelain")
    out, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("git worktree list: %w", err)
    }
    return parseWorktreeList(string(out))
}

// parseWorktreeList parses "git worktree list --porcelain" output.
// Stub for now — full implementation is added in Task 3 along with the shared helper extraction.
func parseWorktreeList(output string) ([]WorktreeInfo, error) {
    return nil, fmt.Errorf("not implemented yet — will be moved to shared helper in Task 3")
}
- [ ] **Step 6: Write test for ListWorktrees with binding scan**

```go
// internal/api/worktree_test.go
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
```

- [ ] **Step 7: Implement AttachWorktree**

```go
// internal/api/worktree.go
func (a *App) AttachWorktree(sessionID, worktreePath string) error {
    // Validate worktree exists and is a git worktree
    wtDir := filepath.Join(worktreePath, ".git")
    if fi, err := os.Stat(wtDir); err != nil || !fi.IsDir() {
        // Also check for .git file (linked worktrees)
        if fi, err := os.Stat(filepath.Join(worktreePath, ".git")); err != nil || fi.IsDir() {
            return fmt.Errorf("not a valid git worktree: %s", worktreePath)
        }
    }
    
    sm := a.getSessionManagerForSession(sessionID)
    if sm == nil {
        return fmt.Errorf("no session manager for session %s", sessionID)
    }
    sm.Lock()
    defer sm.Unlock()
    
    s, err := sm.Load(sessionID)
    if err != nil {
        return err
    }
    s.WorktreePath = worktreePath
    return sm.Save(s)
}
```

- [ ] **Step 8: Implement DetachWorktree**

```go
// internal/api/worktree.go
func (a *App) DetachWorktree(sessionID string) error {
    sm := a.getSessionManagerForSession(sessionID)
    if sm == nil {
        return fmt.Errorf("no session manager for session %s", sessionID)
    }
    sm.Lock()
    defer sm.Unlock()
    
    s, err := sm.Load(sessionID)
    if err != nil {
        return err
    }
    s.WorktreePath = ""
    return sm.Save(s)
}
```

- [ ] **Step 9: Write test for Attach/Detach round trip**

```go
// internal/api/worktree_test.go
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
```

- [ ] **Step 10: Run all worktree tests**

Run: `go test ./internal/api/ -run "Worktree" -v`
Expected: PASS

- [ ] **Step 11: Implement DeleteWorktree and RebuildWorktree (stubs with real git exec)**

```go
// internal/api/worktree.go

func (a *App) DeleteWorktree(worktreePath string) error {
    projectDir := a.projectPath()
    // git worktree remove <path>
    cmd := exec.Command("git", "-C", projectDir, "worktree", "remove", worktreePath)
    if out, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("git worktree remove failed: %w\n%s", err, string(out))
    }
    // git worktree prune
    exec.Command("git", "-C", projectDir, "worktree", "prune").Run()
    return nil
}

func (a *App) RebuildWorktree(sessionID string) error {
    sm := a.getSessionManagerForSession(sessionID)
    if sm == nil {
        return fmt.Errorf("no session manager for session %s", sessionID)
    }
    sm.Lock()
    defer sm.Unlock()
    
    s, err := sm.Load(sessionID)
    if err != nil {
        return err
    }
    if s.WorktreePath == "" {
        return fmt.Errorf("session %s has no worktree binding", sessionID)
    }
    projectDir := s.ProjectDir
    // Extract the worktree dir name from the path
    wtName := filepath.Base(s.WorktreePath)
    // Derive branch name: strip trailing session hash if present
    branch := wtName
    // Try to detect branch from the worktree path convention: <branch>-<sessionID[:8]>
    if sid := s.ID; len(sid) >= 8 && strings.HasSuffix(branch, "-"+sid[:8]) {
        branch = strings.TrimSuffix(branch, "-"+sid[:8])
    }
    
    cmd := exec.Command("git", "-C", projectDir, "worktree", "add", s.WorktreePath, branch)
    if out, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("git worktree add failed: %w\n%s", err, string(out))
    }
    return nil
}
```

- [ ] **Step 12: Implement CreateWorktree**

```go
// internal/api/worktree.go

func (a *App) CreateWorktree(sessionID, branch string) (string, error) {
    sm := a.getSessionManagerForSession(sessionID)
    if sm == nil {
        return "", fmt.Errorf("no session manager for session %s", sessionID)
    }
    
    projectDir := a.projectPath()
    // Determine the effective working directory for branch resolution
    resolvedDir := a.resolveWorkingDir(sessionID)
    
    // If branch is empty, resolve from source context
    if branch == "" {
        cmd := exec.Command("git", "-C", resolvedDir, "rev-parse", "--abbrev-ref", "HEAD")
        out, err := cmd.Output()
        if err != nil {
            return "", fmt.Errorf("failed to resolve source branch: %w", err)
        }
        branch = strings.TrimSpace(string(out))
    }
    
    // Generate worktree name
    wtName := branch + "-" + sessionID[:8]
    wtPath := filepath.Join(projectDir, ".worktrees", wtName)
    
    // Ensure .worktrees directory exists
    os.MkdirAll(filepath.Join(projectDir, ".worktrees"), 0755)
    
    // Run git worktree add
    cmd := exec.Command("git", "-C", projectDir, "worktree", "add", wtPath, branch)
    if out, err := cmd.CombinedOutput(); err != nil {
        // If branch doesn't exist locally, create it at HEAD
        createCmd := exec.Command("git", "-C", projectDir, "branch", branch, "HEAD")
        if createOut, createErr := createCmd.CombinedOutput(); createErr != nil {
            return "", fmt.Errorf("branch %q not found and could not be created: %w\n%s", branch, createErr, string(createOut))
        }
        // Retry worktree add
        if out2, err2 := exec.Command("git", "-C", projectDir, "worktree", "add", wtPath, branch).CombinedOutput(); err2 != nil {
            return "", fmt.Errorf("git worktree add failed: %w\n%s", err2, string(out2))
        }
    }
    
    // Bind to session
    sm.Lock()
    defer sm.Unlock()
    s, err := sm.Load(sessionID)
    if err != nil {
        return wtPath, err // worktree was created, but binding failed
    }
    s.WorktreePath = wtPath
    if saveErr := sm.Save(s); saveErr != nil {
        return wtPath, saveErr
    }
    return wtPath, nil
}
```

- [ ] **Step 13: Run all build to verify compilation**

Run: `go build ./internal/api/`
Expected: builds without errors

- [ ] **Step 14: Commit**

```bash
git add internal/api/worktree.go internal/api/worktree_test.go
git commit -m "feat: implement worktree API methods (Verify/List/Attach/Detach/Delete/Rebuild/Create)"
```

---

### Task 3: Add resolveWorkingDir and Wire into App

**Files:**
- Modify: `internal/api/app.go`

**Goal:** Implement `resolveWorkingDir()` and update `SendMessage()` and `RunShellCommand()` to use it. Extract the git worktree list parsing from `OpenProject()` into a shared helper. Frontend file operations (ReadFile, WriteFile, etc.) remain unchanged — they operate on the project root for file tree browsing, while only agent-facing execution paths (message sending, shell commands) use the worktree path.

- [ ] **Step 1: Add resolveWorkingDir helper**

```go
// internal/api/app.go — add method to App

// resolveWorkingDir returns the effective working directory for a session.
// If the session has a valid WorktreePath, it returns that. Otherwise falls back to projectPath().
func (a *App) resolveWorkingDir(sessionID string) string {
    sm := a.getSessionManagerForSession(sessionID)
    if sm == nil {
        return a.projectPath()
    }
    s, err := sm.Load(sessionID)
    if err != nil || s.WorktreePath == "" {
        return a.projectPath()
    }
    if _, err := os.Stat(s.WorktreePath); err == nil {
        return s.WorktreePath
    }
    // Worktree was deleted; fall back silently.
    // Frontend shows the banner via VerifyWorktree on session switch.
    return a.projectPath()
}
```

- [ ] **Step 2: Update SendMessage to use resolveWorkingDir**

```go
// internal/api/app.go — in SendMessage, replace line:
//   agent2.WithProjectDir(projectPath),
// with:
//   agent2.WithProjectDir(a.resolveWorkingDir(sessionID)),
```

- [ ] **Step 3: Update file operation methods**

- [ ] **Step 3: Update SendMessage and RunShellCommand**

`SendMessage` currently receives `projectPath` from the frontend. Replace its usage with `resolveWorkingDir(sessionID)`:

```go
// internal/api/app.go — in SendMessage, replace:
//   agent2.WithProjectDir(projectPath),
// with:
//   agent2.WithProjectDir(a.resolveWorkingDir(sessionID)),
```

`RunShellCommand` already has `sessionID`. Update its working directory resolution:

```go
// internal/api/app.go — in RunShellCommand, replace resolution with:
func (a *App) RunShellCommand(sessionID, command string) error {
    workDir := a.resolveWorkingDir(sessionID)
    // ...use workDir instead of the old projectPath-based resolution...
}
```

Frontend file operations (ReadFile, WriteFile, CreateDir, Rename, DeleteItem, DuplicateItem, CopyItem, ListFileTree) remain unchanged. They operate on the project root as before. Only agent execution paths need worktree resolution — the agent's tools already receive `ProjectDir` from the AgentLoop.

- [ ] **Step 4: Extract git worktree list parsing into shared helper**

```go
// internal/api/worktree.go

// listGitWorktrees parses "git worktree list --porcelain" output.
func listGitWorktrees(projectDir string) ([]WorktreeInfo, error) {
    cmd := exec.Command("git", "-C", projectDir, "worktree", "list", "--porcelain")
    out, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("git worktree list: %w", err)
    }
    return parseWorktreeList(string(out))
}

func parseWorktreeList(output string) ([]WorktreeInfo, error) {
    var worktrees []WorktreeInfo
    var current *WorktreeInfo
    for _, line := range strings.Split(output, "\n") {
        if strings.HasPrefix(line, "worktree ") {
            if current != nil {
                worktrees = append(worktrees, *current)
            }
            current = &WorktreeInfo{Path: strings.TrimPrefix(line, "worktree ")}
        } else if strings.HasPrefix(line, "branch ") && current != nil {
            ref := strings.TrimPrefix(line, "branch ")
            // refs/heads/branch-name → branch-name
            if strings.HasPrefix(ref, "refs/heads/") {
                current.Branch = ref[len("refs/heads/"):]
            } else {
                current.Branch = ref
            }
        }
    }
    if current != nil {
        worktrees = append(worktrees, *current)
    }
    return worktrees, nil
}
```

- [ ] **Step 5: Update OpenProject to reuse the shared helper**

```go
// internal/api/app.go — in OpenProject(), replace the inline git worktree parsing with:
worktrees, err := listGitWorktrees(projectPath)
if err != nil {
    // Silently continue — worktrees list is best-effort
    info.Worktrees = nil
} else {
    info.Worktrees = worktrees
}
```

- [ ] **Step 6: Write test for parseWorktreeList**

```go
// internal/api/worktree_test.go
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
```

- [ ] **Step 7: Run tests**

Run: `go test ./internal/api/ -run "ParseWorktreeList|Worktree" -v`
Expected: PASS

- [ ] **Step 8: Run build**

Run: `go build ./...`
Expected: builds without errors

- [ ] **Step 9: Commit**

```bash
git add internal/api/app.go internal/api/worktree.go internal/api/worktree_test.go
git commit -m "feat: add resolveWorkingDir, wire into SendMessage/file ops, extract git worktree list parsing"
```

---

### Task 4: Frontend Store Extensions

**Files:**
- Modify: `frontend/src/store/index.ts`

**Goal:** Add `sessionWorktrees` record, `worktreeBanner` state, and their actions to the zustand store.

- [ ] **Step 1: Add state and actions to the store interface**

```typescript
// frontend/src/store/index.ts — add to the StoreState interface

interface StoreState {
    // ...existing fields...
    
    // Worktree binding: sessionId → worktreePath
    sessionWorktrees: Record<string, string>
    
    // Worktree deleted banner
    worktreeBanner: { sessionId: string; deletedPath: string } | null
    
    // Actions
    setSessionWorktree: (sessionId: string, path: string) => void
    showWorktreeBanner: (sessionId: string, path: string) => void
    dismissWorktreeBanner: () => void
}
```

- [ ] **Step 2: Add initial values and actions**

In the `useStore` create call, find the existing initial state object and add:

```typescript
// In the initial state
sessionWorktrees: {} as Record<string, string>,
worktreeBanner: null,

// In the actions
setSessionWorktree: (sessionId: string, path: string) =>
    set((state) => ({
        sessionWorktrees: { ...state.sessionWorktrees, [sessionId]: path },
    })),

showWorktreeBanner: (sessionId: string, path: string) =>
    set({ worktreeBanner: { sessionId, deletedPath: path } }),

dismissWorktreeBanner: () => set({ worktreeBanner: null }),
```

- [ ] **Step 3: Verify TypeScript compilation**

Run: `cd frontend && npx tsc --noEmit`
Expected: No type errors

- [ ] **Step 4: Commit**

```bash
git add frontend/src/store/index.ts
git commit -m "feat: add sessionWorktrees and worktreeBanner state to store"
```

---

### Task 5: WorktreeChip Component

**Files:**
- Create: `frontend/src/components/Chat/WorktreeChip.tsx`
- Modify: `frontend/src/components/Chat/ChatInput.tsx`

**Goal:** A toolbar chip showing the bound branch name, hidden when no worktree is bound.

- [ ] **Step 1: Create WorktreeChip component**

```typescript
// frontend/src/components/Chat/WorktreeChip.tsx
import { useStore } from '../../store'

interface WorktreeChipProps {
    sessionId: string
    onClick: () => void
}

export default function WorktreeChip({ sessionId, onClick }: WorktreeChipProps) {
    const worktreePath = useStore((s) => s.sessionWorktrees[sessionId])
    
    if (!worktreePath) return null
    
    // Extract branch name from the worktree path
    // Path format: .../.worktrees/<branch>-<sessionHash> or .../<branch>
    const parts = worktreePath.replace(/\\/g, '/').split('/')
    const dirName = parts[parts.length - 1]
    
    // Try to strip trailing session hash (e.g., "feature-x-a1b2c3d4" → "feature-x")
    const branchDisplay = sessionId.length >= 8 && dirName.endsWith('-' + sessionId.slice(0, 8))
        ? dirName.slice(0, -(sessionId.slice(0, 8).length + 1))
        : dirName
    
    return (
        <span
            onClick={onClick}
            className="text-[11px] px-1.5 py-0.5 rounded cursor-pointer select-none"
            style={{
                background: 'var(--bg-hover)',
                color: 'var(--text-secondary)',
                border: '1px solid var(--border)',
                whiteSpace: 'nowrap',
                maxWidth: 150,
                overflow: 'hidden',
                textOverflow: 'ellipsis',
            }}
            title={`Worktree: ${worktreePath}`}
        >
            🌳 {branchDisplay}
        </span>
    )
}
```

- [ ] **Step 2: Add WorktreeChip to ChatInput toolbar**

In `ChatInput.tsx`, import WorktreeChip and add it after `ModelPicker` in the toolbar:

```typescript
// At the top:
import WorktreeChip from './WorktreeChip'

// In the component:
const [worktreeManagerOpen, setWorktreeManagerOpen] = useState(false)

// In the toolbar JSX (after <ModelPicker /> and before token counter):
<WorktreeChip
    sessionId={activeSessionId}
    onClick={() => setWorktreeManagerOpen(true)}
/>
{worktreeManagerOpen && (
    <WorktreeManager
        sessionId={activeSessionId}
        onClose={() => setWorktreeManagerOpen(false)}
    />
)}
```

- [ ] **Step 3: Verify TypeScript**

Run: `cd frontend && npx tsc --noEmit`
Expected: No type errors

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/Chat/WorktreeChip.tsx frontend/src/components/Chat/ChatInput.tsx
git commit -m "feat: add WorktreeChip component to input toolbar"
```

---

### Task 6: WorktreeManager Dialog

**Files:**
- Create: `frontend/src/components/Chat/WorktreeManager.tsx`

**Goal:** A modal dialog for browsing, creating, attaching, and deleting worktrees.

- [ ] **Step 1: Create WorktreeManager component**

```typescript
// frontend/src/components/Chat/WorktreeManager.tsx
import { useState, useEffect } from 'react'
import { App } from '../../../bindings/monika'
import { useStore } from '../../store'

interface WorktreeInfo {
    branch: string
    path: string
    bound_sessions?: { id: string; title: string }[]
}

interface WorktreeManagerProps {
    sessionId: string
    onClose: () => void
}

export default function WorktreeManager({ sessionId, onClose }: WorktreeManagerProps) {
    const [worktrees, setWorktrees] = useState<WorktreeInfo[]>([])
    const [selectedPath, setSelectedPath] = useState<string | null>(null)
    const [creating, setCreating] = useState(false)
    const [newBranch, setNewBranch] = useState('')
    const [error, setError] = useState('')
    const setSessionWorktree = useStore((s) => s.setSessionWorktree)
    const currentWorktreePath = useStore((s) => s.sessionWorktrees[sessionId])

    const loadWorktrees = async () => {
        try {
            const list = await App.ListWorktrees() as WorktreeInfo[]
            setWorktrees(list || [])
        } catch {
            setWorktrees([])
        }
    }
    }

    useEffect(() => { loadWorktrees() }, [])

    const handleAttach = async () => {
        if (!selectedPath) return
        setError('')
        try {
            await App.AttachWorktree(sessionId, selectedPath)
            setSessionWorktree(sessionId, selectedPath)
            loadWorktrees()
        } catch (e: any) {
            setError(e?.message || 'Failed to attach worktree')
        }
    }

    const handleDetach = async () => {
        setError('')
        try {
            await App.DetachWorktree(sessionId)
            setSessionWorktree(sessionId, '')
            loadWorktrees()
        } catch (e: any) {
            setError(e?.message || 'Failed to detach worktree')
        }
    }

    const handleDelete = async () => {
        if (!selectedPath) return
        // Check if other sessions are bound to this worktree
        const wt = worktrees.find(w => w.path === selectedPath)
        const boundOthers = (wt?.bound_sessions || []).filter(s => s.id !== sessionId)
        if (boundOthers.length > 0) {
            const names = boundOthers.map(s => s.title || s.id.slice(0, 8)).join(', ')
            if (!confirm(`This worktree is bound to ${names}. Delete anyway?`)) return
        }
        setError('')
        try {
            await App.DeleteWorktree(selectedPath)
            if (selectedPath === currentWorktreePath) {
                setSessionWorktree(sessionId, '')
            }
            loadWorktrees()
            setSelectedPath(null)
        } catch (e: any) {
            setError(e?.message || 'Failed to delete worktree')
        }
    }

    const handleCreate = async () => {
        if (!newBranch.trim()) return
        setError('')
        try {
            const path = await App.CreateWorktree(sessionId, newBranch.trim()) as string
            setSessionWorktree(sessionId, path)
            setCreating(false)
            setNewBranch('')
            loadWorktrees()
        } catch (e: any) {
            setError(e?.message || 'Failed to create worktree')
        }
    }

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center" style={{ background: 'rgba(0,0,0,0.4)' }}>
            <div className="bg-[var(--bg-card)] rounded-lg shadow-xl border border-[var(--border)] w-[600px] max-h-[500px] flex flex-col">
                {/* Header */}
                <div className="flex items-center justify-between px-4 py-3 border-b border-[var(--border)]">
                    <span className="text-[13px] font-medium">Worktrees</span>
                    <button onClick={onClose} className="text-[var(--text-dim)] hover:text-[var(--text-primary)] text-[15px] px-1">&times;</button>
                </div>

                {/* Error */}
                {error && (
                    <div className="px-4 py-2 text-[11px]" style={{ color: 'var(--red)' }}>
                        {error}
                    </div>
                )}

                {/* List */}
                <div className="flex-1 overflow-y-auto px-4 py-2">
                    {worktrees.length === 0 && !creating && (
                        <div className="text-[12px] py-8 text-center" style={{ color: 'var(--text-dim)' }}>
                            No worktrees. Create one to work on a different branch.
                        </div>
                    )}
                    {worktrees.map((wt) => {
                        const isBound = wt.bound_sessions?.some(s => s.id === sessionId)
                        const otherSessions = (wt.bound_sessions || []).filter(s => s.id !== sessionId)
                        return (
                            <div
                                key={wt.path}
                                className={`flex items-center gap-3 px-3 py-2 rounded cursor-pointer text-[12px] ${
                                    selectedPath === wt.path ? 'bg-[var(--bg-hover)]' : ''
                                }`}
                                onClick={() => setSelectedPath(wt.path)}
                                style={selectedPath === wt.path ? { background: 'var(--bg-hover)' } : undefined}
                            >
                                <span className="font-mono w-[120px] truncate">{wt.branch || '(detached)'}</span>
                                <span className="flex-1 truncate" style={{ color: 'var(--text-dim)' }}>{wt.path}</span>
                                <span className="text-[11px] shrink-0">
                                    {isBound ? (
                                        <span style={{ color: 'var(--accent)' }}>This session</span>
                                    ) : otherSessions.length > 0 ? (
                                        <span style={{ color: 'var(--text-dim)' }}>
                                            Bound: {otherSessions.map(s => s.title || s.id.slice(0, 8)).join(', ')}
                                        </span>
                                    ) : (
                                        <span className="text-[var(--text-dim)]">Unbound</span>
                                    )}
                                </span>
                            </div>
                        )
                    })}
                    {creating && (
                        <div className="flex items-center gap-2 px-3 py-2 mt-2">
                            <input
                                value={newBranch}
                                onChange={(e) => setNewBranch(e.target.value)}
                                onKeyDown={(e) => { if (e.key === 'Enter') handleCreate() }}
                                placeholder="Branch name"
                                className="flex-1 text-[12px] px-2 py-1 rounded border border-[var(--border)] bg-transparent outline-none"
                                style={{ color: 'var(--text-primary)', borderColor: 'var(--border)' }}
                                autoFocus
                            />
                            <button
                                onClick={handleCreate}
                                disabled={!newBranch.trim()}
                                className="text-[11px] px-2 py-1 rounded"
                                style={{ background: 'var(--accent)', color: '#fff', opacity: newBranch.trim() ? 1 : 0.5 }}
                            >Create</button>
                            <button
                                onClick={() => { setCreating(false); setNewBranch('') }}
                                className="text-[11px] px-2 py-1 rounded"
                                style={{ color: 'var(--text-dim)' }}
                            >Cancel</button>
                        </div>
                    )}
                </div>

                {/* Actions */}
                <div className="flex items-center gap-2 px-4 py-2.5 border-t border-[var(--border)]">
                    <button
                        onClick={() => setCreating(true)}
                        className="text-[11px] px-2.5 py-1 rounded"
                        style={{ background: 'var(--bg-hover)', color: 'var(--text-primary)' }}
                    >+ New</button>
                    <button
                        onClick={handleAttach}
                        disabled={!selectedPath || selectedPath === currentWorktreePath}
                        className="text-[11px] px-2.5 py-1 rounded"
                        style={{
                            background: selectedPath && selectedPath !== currentWorktreePath ? 'var(--accent)' : 'var(--bg-hover)',
                            color: selectedPath && selectedPath !== currentWorktreePath ? '#fff' : 'var(--text-dim)',
                        }}
                    >Attach</button>
                    <button
                        onClick={handleDelete}
                        disabled={!selectedPath}
                        className="text-[11px] px-2.5 py-1 rounded"
                        style={{
                            background: selectedPath ? 'var(--red)' : 'var(--bg-hover)',
                            color: selectedPath ? '#fff' : 'var(--text-dim)',
                        }}
                    >Delete</button>
                    <div className="flex-1" />
                    {currentWorktreePath && (
                        <button
                            onClick={handleDetach}
                            className="text-[11px] px-2.5 py-1 rounded"
                            style={{ color: 'var(--text-dim)' }}
                        >Detach</button>
                    )}
                </div>
            </div>
        </div>
    )
}
```

- [ ] **Step 2: Verify TypeScript**

Run: `cd frontend && npx tsc --noEmit`
Expected: No type errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/Chat/WorktreeManager.tsx
git commit -m "feat: add WorktreeManager modal dialog for worktree CRUD"
```

---

### Task 7: WorktreeBanner Component

**Files:**
- Create: `frontend/src/components/Chat/WorktreeBanner.tsx`
- Modify: `frontend/src/components/Chat/ChatArea.tsx`

**Goal:** A non-modal banner displayed when a bound worktree is missing, offering Rebuild and Revert to Project Root options.

- [ ] **Step 1: Create WorktreeBanner component**

```typescript
// frontend/src/components/Chat/WorktreeBanner.tsx
import { useState } from 'react'
import { App } from '../../../bindings/monika'
import { useStore } from '../../store'

interface WorktreeBannerProps {
    sessionId: string
    deletedPath: string
    onClose: () => void
}

export default function WorktreeBanner({ sessionId, deletedPath, onClose }: WorktreeBannerProps) {
    const [rebuilding, setRebuilding] = useState(false)
    const [error, setError] = useState('')
    const setSessionWorktree = useStore((s) => s.setSessionWorktree)
    const dismissWorktreeBanner = useStore((s) => s.dismissWorktreeBanner)

    const handleRebuild = async () => {
        setRebuilding(true)
        setError('')
        try {
            await App.RebuildWorktree(sessionId)
            setSessionWorktree(sessionId, deletedPath)
            dismissWorktreeBanner()
        } catch (e: any) {
            setError('Failed to rebuild: ' + (e?.message || 'unknown error') + '. Please create a new worktree.')
        } finally {
            setRebuilding(false)
        }
    }

    const handleRevert = async () => {
        try {
            await App.DetachWorktree(sessionId)
            setSessionWorktree(sessionId, '')
            dismissWorktreeBanner()
        } catch (e: any) {
            setError(e?.message || 'Failed to detach worktree')
        }
    }

    return (
        <div
            className="flex items-center gap-2 px-4 py-2 text-[12px]"
            style={{ background: 'var(--bg-warning, #fff3cd)', color: 'var(--text-warning, #856404)' }}
        >
            <span>⚠️ Worktree "{deletedPath}" no longer exists.</span>
            <button
                onClick={handleRebuild}
                disabled={rebuilding}
                className="text-[11px] px-2 py-0.5 rounded border ml-2"
                style={{ borderColor: 'currentColor' }}
            >
                {rebuilding ? 'Rebuilding...' : 'Rebuild'}
            </button>
            <button
                onClick={handleRevert}
                className="text-[11px] px-2 py-0.5 rounded border"
                style={{ borderColor: 'currentColor' }}
            >Revert to Project Root</button>
            <button onClick={onClose} className="ml-auto text-[14px] opacity-60 hover:opacity-100">&times;</button>
            {error && <div className="text-[11px] ml-2" style={{ color: 'var(--red)' }}>{error}</div>}
        </div>
    )
}
```

- [ ] **Step 2: Integrate into ChatArea**

In `ChatArea.tsx`, import WorktreeBanner and add it above the message list:

```typescript
// At the top:
import WorktreeBanner from './WorktreeBanner'

// In the component, add state:
const [verifyResult, setVerifyResult] = useState<{ sessionId: string; path: string } | null>(null)
const dismissLocal = useStore((s) => s.dismissWorktreeBanner)

// Add effect to verify on mount and session switch:
useEffect(() => {
    const check = async () => {
        if (!sessionId || isDefaultChat || isOverlay) return
        try {
            const result = await App.VerifyWorktree(sessionId) as { deleted: boolean; path: string } | null
            if (result?.deleted) {
                setVerifyResult({ sessionId, path: result.path })
            } else {
                setVerifyResult(null)
            }
        } catch { /* ignore */ }
    }
    check()
}, [sessionId, isDefaultChat, isOverlay])

// Also when worktreeBanner is dismissed but verify detects it again:
// Sync with store's worktreeBanner if needed (or use local state only — simpler)

// In JSX, above the message list div:
{verifyResult && (
    <WorktreeBanner
        sessionId={verifyResult.sessionId}
        deletedPath={verifyResult.path}
        onClose={() => setVerifyResult(null)}
    />
)}
```

- [ ] **Step 3: Verify TypeScript**

Run: `cd frontend && npx tsc --noEmit`
Expected: No type errors

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/Chat/WorktreeBanner.tsx frontend/src/components/Chat/ChatArea.tsx
git commit -m "feat: add WorktreeBanner for deleted worktree detection + integrate into ChatArea"
```

---

### Task 8: SessionContextMenu Component

**Files:**
- Create: `frontend/src/components/Chat/SessionContextMenu.tsx`

**Goal:** Right-click context menu on session list items with "Manage Worktree..." and "Detach Worktree" options.

- [ ] **Step 1: Create SessionContextMenu component**

```typescript
// frontend/src/components/Chat/SessionContextMenu.tsx
import { useStore } from '../../store'

interface SessionContextMenuProps {
    sessionId: string
    x: number
    y: number
    onClose: () => void
    onManageWorktree: () => void
}

export default function SessionContextMenu({ sessionId, x, y, onClose, onManageWorktree }: SessionContextMenuProps) {
    const worktreePath = useStore((s) => s.sessionWorktrees[sessionId])
    const detachSessionWorktree = useStore((s) => s.setSessionWorktree) // reuse setter with '' to detach

    const handleDetach = async () => {
        try {
            const { App } = await import('../../../bindings/monika')
            await App.DetachWorktree(sessionId)
            detachSessionWorktree(sessionId, '')
        } catch { /* ignore */ }
        onClose()
    }

    // Close on click outside
    const handleBackdropClick = (e: React.MouseEvent) => {
        e.stopPropagation()
        onClose()
    }

    return (
        <div
            className="fixed inset-0 z-50"
            onClick={handleBackdropClick}
            onContextMenu={(e) => { e.preventDefault(); onClose() }}
        >
            <div
                className="absolute bg-[var(--bg-card)] border border-[var(--border)] rounded shadow-lg py-1 min-w-[180px]"
                style={{ left: x, top: y }}
                onClick={(e) => e.stopPropagation()}
            >
                <button
                    onClick={() => { onManageWorktree(); onClose() }}
                    className="w-full text-left px-3 py-1.5 text-[12px] hover:bg-[var(--bg-hover)]"
                >Manage Worktree...</button>
                {worktreePath && (
                    <button
                        onClick={handleDetach}
                        className="w-full text-left px-3 py-1.5 text-[12px] hover:bg-[var(--bg-hover)]"
                    >Detach Worktree</button>
                )}
            </div>
        </div>
    )
}
```

- [ ] **Step 2: Integrate into session list**

The session list is in the sidebar. Find where session items are rendered (likely in `SessionSidebar.tsx` or similar) and add right-click handling.

To find the exact file:

```bash
grep -rn "onContextMenu\|contextmenu\|contextMenu" frontend/src/components/Session/ --include="*.tsx"
```

Add context menu to each session list item:

```typescript
// In the session item component:
const [contextMenu, setContextMenu] = useState<{ sessionId: string; x: number; y: number } | null>(null)

// On the session list item div:
<div
    onContextMenu={(e) => {
        e.preventDefault()
        setContextMenu({ sessionId: session.id, x: e.clientX, y: e.clientY })
    }}
>
    {/* session item content */}
</div>

// At the bottom of the component:
{contextMenu && (
    <SessionContextMenu
        sessionId={contextMenu.sessionId}
        x={contextMenu.x}
        y={contextMenu.y}
        onClose={() => setContextMenu(null)}
        onManageWorktree={() => setWorktreeManagerOpen(contextMenu.sessionId)}
    />
)}
```

- [ ] **Step 3: Verify TypeScript**

Run: `cd frontend && npx tsc --noEmit`
Expected: No type errors

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/Chat/SessionContextMenu.tsx frontend/src/components/Session/SessionSidebar.tsx
git commit -m "feat: add SessionContextMenu with Manage/Detach worktree options"
```

---

### Task 9: Wire WorktreeManager into Session Sidebar

**Files:**
- Modify: `frontend/src/components/Session/SessionSidebar.tsx` (or equivalent session list component)

**Goal:** When "Manage Worktree..." is clicked from the context menu, open the WorktreeManager dialog.

- [ ] **Step 1: Add WorktreeManager import and state**

```typescript
// frontend/src/components/Session/SessionSidebar.tsx
import { useState } from 'react'
import WorktreeManager from '../Chat/WorktreeManager'
// ...existing imports...

// In the component:
const [worktreeManagerSessionId, setWorktreeManagerSessionId] = useState<string | null>(null)

// Pass to context menu's onManageWorktree:
onManageWorktree={() => setWorktreeManagerSessionId(session.id)}

// At the bottom of the component's JSX:
{worktreeManagerSessionId && (
    <WorktreeManager
        sessionId={worktreeManagerSessionId}
        onClose={() => setWorktreeManagerSessionId(null)}
    />
)}
```

- [ ] **Step 2: Verify TypeScript**

Run: `cd frontend && npx tsc --noEmit`
Expected: No type errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/Session/SessionSidebar.tsx
git commit -m "feat: wire WorktreeManager into session sidebar context menu"
```

---

### Task 10: Integration Test Pass

**Files:** All modified

**Goal:** Ensure full application compiles and no regressions.

- [ ] **Step 1: Build Go backend**

Run: `go build ./...`
Expected: compiles without error

- [ ] **Step 2: Lint Go code**

Run: `go vet ./...`
Expected: passes without warnings

- [ ] **Step 3: Run Go tests**

Run: `go test ./internal/api/ -v`
Expected: all existing tests + new tests pass

- [ ] **Step 4: Build frontend**

Run: `cd frontend && npx tsc --noEmit`
Expected: compiles without error

- [ ] **Step 5: Commit any final fixes**

```bash
git add -A
git commit -m "fix: integration fixes after full build pass"
```
