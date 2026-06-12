# Session Worktree Binding Design

## Summary

Support session-level git worktree binding, enabling multiple sessions to work in parallel on different branches within the same project. Each session can be bound to a git worktree; all tool operations (file_read, file_edit, bash, etc.) execute within the bound worktree directory.

## Approach

Option A (session-level binding): Add a `WorktreePath` field to the `Session` struct. Resolve the effective working directory at tool execution time by checking `session.WorktreePath` first, falling back to `projectPath()`. No new abstractions — the session is the single source of truth for worktree binding.

Key design decisions:

- **Forward verification**: When switching to a session or before tool execution, actively check if the bound worktree directory exists. No reverse notification or broadcast — each session only cares about its own worktree.
- **Deleted worktree handling**: A non-modal banner in the chat area offers two options: Rebuild (re-run `git worktree add`) or Revert to Project Root (clear `WorktreePath`). If the user ignores the banner, the session silently falls back to `ProjectDir`.
- **Worktree indicator**: Displayed as a chip in the input toolbar next to `ModelPicker`. Not shown in the tab title or title bar (branch names can be too long).
- **Worktree manager UI**: A modal dialog similar to a file manager, where users can browse, create, delete, and attach worktrees. "Create Worktree" is part of this dialog, not a standalone menu item.

## Data Model

### Session Extension

```go
// internal/api/session_manager.go
type Session struct {
    // ...existing fields...
    WorktreePath string `json:"worktree_path,omitempty"`
}
```

- Empty string means unbound — session uses `ProjectDir` directly.
- Persisted to JSON, backward compatible via `omitempty`.

### SessionInfo Extension

```go
// internal/api/types.go
type SessionInfo struct {
    // ...existing fields...
    WorktreePath   string `json:"worktree_path,omitempty"`
    WorktreeBranch string `json:"worktree_branch,omitempty"`
}
```

### Frontend Store Extension

```typescript
// store/index.ts
sessionWorktrees: Record<string, string>  // sessionId → worktreePath
setSessionWorktree: (sessionId: string, path: string) => void
worktreeBanner: { sessionId: string; deletedPath: string } | null
showWorktreeBanner: (sessionId: string, path: string) => void
dismissWorktreeBanner: () => void
```

## Working Directory Resolution

### Core Logic

```go
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
    // Worktree deleted: fall back to ProjectDir.
    // The banner is shown via frontend on next session switch, not here.
    return a.projectPath()
}
```

### Affected Code Paths

| Path | Current | After |
|------|---------|-------|
| `SendMessage()` | `WithProjectDir(projectPath)` | `WithProjectDir(resolveWorkingDir(sessionID))` |
| `ReadFile()`, `WriteFile()`, `CreateDir()`, `Rename()`, `DeleteItem()`, `CopyItem()`, `DuplicateItem()` | `a.projectPath()` | Use `resolveWorkingDir` or per-call resolution |
| `RunShellCommand()` | Uses `projectPath` param | Uses `resolveWorkingDir` |
| LSP operations | Resolved per file | Resolved per session |

## Forward Verification

When the user switches to a session (or when the frontend first opens a session tab), the frontend calls `App.VerifyWorktree(sessionID)`:

```go
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

type WorktreeVerifyResult struct {
    Deleted bool   `json:"deleted"`
    Path    string `json:"path"`
}
```

The frontend shows the banner when `Deleted == true`. The same check also happens inside `resolveWorkingDir()` as a safety net before every tool execution.

## Worktree Management API

New file: `internal/api/worktree.go`

### CreateWorktree

```
CreateWorktree(sessionID string) (worktreePath string, error)
```

1. Load session, read `ProjectDir`.
2. Resolve source branch: `git -C <projectDir> rev-parse --abbrev-ref HEAD`.
3. Generate worktree name: `<branch>-<sessionID[:8]>`.
4. Run: `git -C <projectDir> worktree add .worktrees/<name> <branch>`.
   - If the branch doesn't exist yet, create it first: `git -C <projectDir> branch <branch> HEAD`.
5. Set `session.WorktreePath = <projectDir>/.worktrees/<name>`.
6. Save session.

### AttachWorktree

```
AttachWorktree(sessionID string, worktreePath string) error
```

1. Verify `worktreePath` exists and is a valid git worktree.
2. Load session, set `session.WorktreePath = worktreePath`.
3. Save session.

### DetachWorktree

```
DetachWorktree(sessionID string) error
```

1. Load session.
2. Set `session.WorktreePath = ""`.
3. Save session.
4. Does NOT delete the worktree from disk.

### DeleteWorktree

```
DeleteWorktree(worktreePath string) error
```

1. Run `git -C <projectDir> worktree remove <worktreePath>`.
2. Run `git -C <projectDir> worktree prune`.
3. Does NOT modify any session — the next forward verification will detect the deletion.

### ListWorktrees

```
ListWorktrees() ([]WorktreeInfo, error)
```

Delegates to the existing `git worktree list --porcelain` parsing in `OpenProject()`. Extracts that parsing into a shared helper.

### RebuildWorktree

```
RebuildWorktree(sessionID string) error
```

1. Load session. If `WorktreePath` is empty, error.
2. Extract branch name from the path (last component, strip session suffix).
3. Run `git -C <projectDir> worktree add .worktrees/<name> <branch>`.
4. Returns error if the branch no longer exists (frontend displays a message suggesting manual action).

## AgentLoop Changes

Minimal: the `resolveWorkingDir()` call happens before the loop is created, so `AgentLoop.projectDir` is already set to the correct worktree path. No internal agent changes needed.

If we want an extra safety check, `runStreaming()` can call `os.Stat(a.projectDir)` before each tool execution and emit a `worktree_deleted` event type, but this is optional — the `resolveWorkingDir` already handles it.

## Frontend: Worktree Indicator

### WorktreeChip

Located in `ChatInput.tsx` toolbar, between `ModelPicker` and the token counter.

```
[PermissionModePicker] [ModelPicker] [🌳 feature-branch] [1.2K / 128K] ... [Send]
```

- Hidden when `WorktreePath` is empty.
- Shows branch name extracted from the worktree path.
- Click opens the WorktreeManager dialog with the current worktree pre-selected.
- Style: similar to `LabelChip`, using a neutral/accent background.

### Session Right-Click Context Menu

New component: `SessionContextMenu.tsx`

Menu items:
- "Manage Worktree..." → opens `WorktreeManager` dialog (always visible).
- "Detach Worktree" → calls `App.DetachWorktree(sessionId)` (only when bound).

## Frontend: Worktree Manager

### WorktreeManager Dialog

New component: `WorktreeManager.tsx`

A modal dialog modeled after a file manager:

**Header**:
- Title: "Worktrees"
- Close button

**List Area**:
- Each row: branch name | path | binding status
- Binding status: "Bound to <session name>" (with session switcher link), "Unbound", "This session" (highlighted, current)
- Multi-column layout: Branch | Path | Status

**Actions** (bottom bar):
- `+ New` — expands an inline creation form:
  - Branch name text input
  - "Create" button → calls `App.CreateWorktree(sessionId)`
  - On success: closes creation form, refreshes list, binds to current session
- `Attach` — binds the selected worktree to the current session (disabled if already bound to current session)
- `Delete` — calls `App.DeleteWorktree(worktreePath)` with confirmation dialog
  - Shows warning if the worktree is bound to other sessions
- `Detach` — calls `App.DetachWorktree(sessionId)` (only when current session is bound)

**Empty State**:
- "No worktrees. Create one to work on a different branch."

## Frontend: Worktree Deleted Banner

### WorktreeBanner

New component: `WorktreeBanner.tsx`

Displayed at the top of the chat message area in `ChatArea.tsx`, above the message list but below the tab bar.

```
⚠️ Worktree "D:/project/.worktrees/feature-branch-a1b2c3d4" no longer exists.
[Rebuild] [Revert to Project Root] [✕]
```

**Behavior**:
- **Rebuild**: calls `App.RebuildWorktree(sessionId)`
  - Success → dismisses banner, refreshes worktree indicator.
  - Failure (e.g. branch deleted) → shows inline error: "Failed to rebuild: branch was deleted. Please create a new worktree or attach to an existing one." with a "Manage" button that opens `WorktreeManager`.
- **Revert to Project Root**: calls `App.DetachWorktree(sessionId)`, dismisses banner.
- **Dismiss (✕)**: hides the banner but does NOT clear `WorktreePath`. The session will fall back to `ProjectDir` in `resolveWorkingDir()`. The banner will re-appear on next session switch if the worktree is still deleted.
- Non-modal, non-blocking. User can continue chatting while the banner is visible.

## Edge Cases

| Scenario | Behavior |
|----------|----------|
| Worktree path is invalid | `CreateWorktree` validates the path is under `.worktrees/`. `AttachWorktree` verifies the path is a valid git worktree. |
| Branch already exists | `git worktree add` will fail. Return error with the git error message. |
| Concurrent access | SessionManager has a mutex. Two sessions bound to different worktrees operate in parallel without conflict. |
| Worktree bound to other sessions | `AttachWorktree` allows multiple sessions to share a worktree. `DeleteWorktree` warns if the worktree is bound to other sessions. |
| Session deleted | Only the session JSON is deleted. The worktree on disk remains. User cleans up manually via WorktreeManager. |
| Project switched | `OpenProject()` re-loads worktree list. Bound sessions keep their absolute `WorktreePath` unchanged. |
| App restart | `WorktreePath` persists in session JSON. On next session access, forward verification detects if the worktree was manually deleted while the app was closed. |

## Files Changed

| File | Change |
|------|--------|
| `internal/api/session_manager.go` | Add `WorktreePath` field to `Session` struct |
| `internal/api/types.go` | Add `WorktreePath`, `WorktreeBranch` to `SessionInfo`; add `WorktreeVerifyResult` type |
| `internal/api/worktree.go` | New: `CreateWorktree`, `AttachWorktree`, `DetachWorktree`, `DeleteWorktree`, `ListWorktrees`, `RebuildWorktree`, `VerifyWorktree` |
| `internal/api/app.go` | Add `resolveWorkingDir()` helper; update `SendMessage()`, file operations to use it; wire worktree API methods; extract shared `git worktree list` parsing |
| `frontend/src/store/index.ts` | Add `sessionWorktrees`, `worktreeBanner` state and actions |
| `frontend/src/components/Chat/ChatInput.tsx` | Add `WorktreeChip` in toolbar |
| `frontend/src/components/Chat/ChatArea.tsx` | Add `WorktreeBanner` above message list; call `VerifyWorktree` on mount/session switch |
| `frontend/src/components/Chat/WorktreeChip.tsx` | New: worktree indicator chip |
| `frontend/src/components/Chat/WorktreeManager.tsx` | New: worktree management modal dialog |
| `frontend/src/components/Chat/WorktreeBanner.tsx` | New: deleted worktree banner |
| `frontend/src/components/Chat/SessionContextMenu.tsx` | New: session right-click context menu |

## Out of Scope

- Automatic worktree cleanup / garbage collection
- Merging between worktrees
- Worktree diff visualization
- Displaying worktree name in session list title or tab title
