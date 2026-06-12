package api

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

// App.VerifyWorktree delegates to the standalone VerifyWorktree function.
func (a *App) VerifyWorktree(sessionID string) *WorktreeVerifyResult {
	sm := a.getSessionManagerForSession(sessionID)
	if sm == nil {
		return nil
	}
	return VerifyWorktree(sm, sessionID)
}

// ListWorktrees returns all worktrees for the current project with binding info.
func (a *App) ListWorktrees() ([]WorktreeInfo, error) {
	projectDir := a.projectPath()
	worktrees, err := listGitWorktrees(projectDir)
	if err != nil {
		return nil, err
	}
	// Collect all session bindings to annotate worktrees
	bindingMap := make(map[string][]SessionRef) // worktreePath -> sessions
	for _, sm := range a.sessions {
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
	cmd := command("git", "-C", projectDir, "worktree", "list", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}
	return parseWorktreeList(string(out))
}

// parseWorktreeList parses "git worktree list --porcelain" output.
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
			// refs/heads/branch-name -> branch-name
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

// AttachWorktree binds a session to an existing worktree path.
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

// DetachWorktree unbinds a session from its worktree (does not delete the worktree on disk).
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

// DeleteWorktree removes a worktree from disk via git worktree remove.
func (a *App) DeleteWorktree(worktreePath string) error {
	projectDir := a.projectPath()
	// git worktree remove <path>
	cmd := command("git", "-C", projectDir, "worktree", "remove", worktreePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree remove failed: %w\n%s", err, string(out))
	}
	// git worktree prune
	_ = command("git", "-C", projectDir, "worktree", "prune").Run()
	return nil
}

// RebuildWorktree re-creates a deleted worktree for a session.
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

	cmd := command("git", "-C", projectDir, "worktree", "add", s.WorktreePath, branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree add failed: %w\n%s", err, string(out))
	}
	return nil
}

// CreateWorktree creates a new git worktree and binds it to a session.
func (a *App) CreateWorktree(sessionID, branch string) (string, error) {
	sm := a.getSessionManagerForSession(sessionID)
	if sm == nil {
		return "", fmt.Errorf("no session manager for session %s", sessionID)
	}

	projectDir := a.projectPath()
	resolvedDir := a.resolveWorkingDir(sessionID)

	// If branch is empty, resolve from source context
	if branch == "" {
		cmd := command("git", "-C", resolvedDir, "rev-parse", "--abbrev-ref", "HEAD")
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
	_ = os.MkdirAll(filepath.Join(projectDir, ".worktrees"), 0755)

	// Run git worktree add
	cmd := command("git", "-C", projectDir, "worktree", "add", wtPath, branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		// If branch doesn't exist locally, create it at HEAD
		createCmd := command("git", "-C", projectDir, "branch", branch, "HEAD")
		if createOut, createErr := createCmd.CombinedOutput(); createErr != nil {
			return "", fmt.Errorf("branch %q not found and could not be created: %w\n%s", branch, createErr, string(createOut))
		}
		// Retry worktree add
		if out2, err2 := command("git", "-C", projectDir, "worktree", "add", wtPath, branch).CombinedOutput(); err2 != nil {
			return "", fmt.Errorf("git worktree add failed: %w\n%s", err2, string(out2))
		}
		_ = out // suppress unused warning
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
