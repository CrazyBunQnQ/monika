# HISTORY Right-Click Menu & CHANGES Commit Workflow

**Date:** 2026-06-22
**Status:** Approved (design)

## Problem

The CHANGES panel currently only lists unstaged files for diff viewing — there is no way to stage, commit, or push. The HISTORY panel is read-only: clicking a commit row does nothing, and there is no right-click menu to inspect or act on commits.

Together these gaps force users to leave the app for routine git operations (commit, checkout, revert, cherry-pick, reset, tag, branch-from-commit).

## Goal

Turn the CHANGES + HISTORY tabs into a **complete git client** within the existing dockview layout:

1. **CHANGES** gains an Unstaged / Staged / Commit-message three-section layout with multi-select staging and commit (+push).
2. **HISTORY** gains a right-click context menu with the full set of git operations (view details, copy, checkout, tag, branch, revert, cherry-pick, reset, amend).
3. **Preview panel** gains a `mode: 'commit'` that renders a `[file-list | diff]` split-view for inspecting a commit's changes.

## Decisions

| Decision | Choice |
|----------|--------|
| Scope | **Complete git client** — includes history-rewriting operations (revert, cherry-pick, reset) |
| Design merge | CHANGES commit workflow and HISTORY context menu designed together — they share backend git infrastructure, danger-confirmation UI, and post-operation refresh |
| Layout (CHANGES) | Three-section vertical: Unstaged → Staged → Commit message + buttons |
| Staging model | Multi-select via checkboxes; Stage / Unstage buttons move files between sections |
| Commit actions | `Commit` and `Commit & Push` (Push disabled if no remote) |
| Commit-detail view | Reuse Preview panel with new `mode: 'commit'` — left file-list, right diff (split-view, dockview maximize for full-screen) |
| Target user | Developer daily use — git terminology used directly, no hand-holding, confirmation only for destructive ops |
| File diff location | Single-file diff from CHANGES opens in Preview panel (`mode: 'diff'`) — Unstaged shows working-tree diff, Staged shows cached diff |
| Implementation phasing | Phase 1 = commit workflow + read-only/navigation menu; Phase 2 = history-rewriting menu items |

## Architecture

### Preview panel mode extension

```typescript
interface PreviewState {
    mode: 'file' | 'diff' | 'task' | 'commit' | null
    // existing fields unchanged
}
```

`mode: 'commit'` renders a `CommitSplitView` — left column is the commit's file list, right column is the selected file's diff. Reuses existing `DiffView` rendering and dockview maximize for full-screen.

`mode: 'staging'` is **not** needed — commit message input lives in the CHANGES tab bottom section.

### Data flow

```
HISTORY CommitRow onClick
  → store.setPreviewCommit(hash)
  → backend GitShow(path, hash)
  → PreviewPanel mode='commit'
  → CommitSplitView
      ├─ left:  CommitFileList (reuses ChangeStat row rendering)
      └─ right: DiffView (reuses existing diffLines rendering)

CHANGES file row onClick
  → Unstaged: GetFileDiff (working-tree diff)
  → Staged:   GetStagedFileDiff (cached diff)
  → PreviewPanel mode='diff'
```

## Phase 1 — Commit Workflow + Read-Only Menu

### CHANGES tab layout

```
┌─────────────────────────────┐
│  CHANGES  │  HISTORY         │  ← tab bar
├─────────────────────────────┤
│  Unstaged (3)     [Stage ▶]  │
│  ▢ file-a.ts      +12 -3     │
│  ▢ src/b.go       +45 -1     │
│  ▢ README.md      +2         │
├─────────────────────────────┤
│  Staged (2)      [◀ Unstage] │
│  ✓ staged-1.ts    +8 -2      │
│  ✓ staged-2.go    +30        │
├─────────────────────────────┤
│  ┌─────────────────────────┐│
│  │ commit message...       ││  ← textarea
│  └─────────────────────────┘│
│     [Commit]  [Commit & Push]│
└─────────────────────────────┘
```

- Three sections scroll independently; commit-message area pinned at bottom.
- File rows: checkbox (multi-select) + filename (truncate) + +/- stats.
- Click checkbox → toggle selection; click filename row (non-checkbox) → Preview panel shows diff.
- Double-click file row → quick stage/unstage.
- Commit message textarea: `Ctrl+Enter` = Commit, `Ctrl+Shift+Enter` = Commit & Push.

### Backend API (Phase 1)

```go
// ChangeStat gains Status + Staged fields
type ChangeStat struct {
    Path    string `json:"path"`
    Added   int    `json:"added"`
    Deleted int    `json:"deleted"`
    Status  string `json:"status"`  // "modified"|"added"|"deleted"|"renamed"|"untracked"|"unmerged"
    Staged  bool   `json:"staged"`
}

// Staging
StageFiles(projectPath string, paths []string) error        // git add -- <paths...>
UnstageFiles(projectPath string, paths []string) error      // git reset HEAD -- <paths...>

// Committing
Commit(projectPath string, message string) error            // git commit -m <msg>
CommitAndPush(projectPath string, message string) error     // git commit -m <msg> && git push (uses upstream tracking branch)

// Staged diff
GetStagedFileDiff(projectPath, filePath string) (*DiffResult, error)  // git diff --cached -- <file>

// Commit detail
GitShow(projectPath, hash string) (*CommitDetail, error)    // git show --numstat + metadata
type CommitDetail struct {
    Hash    string       `json:"hash"`
    Author  string       `json:"author"`
    Date    string       `json:"date"`
    Message string       `json:"message"`
    Files   []ChangeStat `json:"files"`
}
GetCommitFileDiff(projectPath, hash, filePath string) (*DiffResult, error)  // git diff <hash>^ <hash> -- <file> (for root commit: git show <hash> -- <file>)

// Navigation operations
CheckoutCommit(projectPath, hash string) error              // git checkout <hash>
CreateTag(projectPath, hash, tagName string) error          // git tag <name> <hash>
CreateBranchAt(projectPath, hash, branchName string) error  // git checkout -b <name> <hash>
```

### HISTORY right-click menu (Phase 1)

| Menu item | Action |
|-----------|--------|
| View Details | Opens commit split-view in Preview (also triggered by left-click) |
| ─────────── | |
| Copy Hash | Copies full hash to clipboard |
| Copy Message | Copies commit message to clipboard |
| ─────────── | |
| Checkout Commit | Detached HEAD at commit (confirmation dialog) |
| Create Tag... | Input dialog for tag name → `git tag` |
| Create Branch at Commit... | Input dialog for branch name → `git checkout -b` |

### Empty states

- Unstaged empty: "No unstaged changes"
- Staged empty: "No staged changes" (Commit button disabled)
- Both empty: "Working tree clean"

## Phase 2 — History-Rewriting Menu

### HISTORY right-click menu (Phase 2 additions)

| Menu item | Action |
|-----------|--------|
| ─────────── | |
| Revert Commit | Creates reverse commit (`git revert --no-edit`), confirmation dialog |
| Cherry-pick Commit | Applies commit to current branch (`git cherry-pick`), confirmation dialog |
| ─────────── | |
| Reset to This Commit ▶ | Submenu: Soft / Mixed / Hard |
| Amend Message... | Input dialog, only visible on HEAD commit (`git commit --amend -m`) |

### Backend API (Phase 2)

```go
RevertCommit(projectPath, hash string) error                // git revert --no-edit <hash>
CherryPickCommit(projectPath, hash string) error            // git cherry-pick <hash>
ResetToCommit(projectPath, hash, mode string) error        // mode: "soft"|"mixed"|"hard"
AmendMessage(projectPath, message string) error            // git commit --amend -m <msg> (HEAD only)
```

## Danger Confirmation

Reuses existing `ConfirmModal` component for destructive operations:

- **Checkout Commit**: warns about detached HEAD state
- **Reset --hard**: warns that ALL uncommitted changes will be permanently discarded
- **Reset --soft/mixed**: lightweight warning
- **Revert / Cherry-pick**: warns that a new commit will be created and conflicts may occur

Example (Reset --hard):

```
Reset to a1b2c3d (Hard)?

This will permanently discard ALL uncommitted changes
and move HEAD to this commit. This cannot be undone.
                    [Cancel]  [Reset Hard]
```

## Error Handling

| Scenario | Handling |
|----------|----------|
| Commit with empty Staged | Frontend blocks — Commit button disabled |
| Commit & Push with no remote | Backend returns error, Toast: "No remote configured" |
| Stage/Unstage nonexistent file | Backend returns stderr, Toast shows git error |
| Checkout/Reset with dirty working tree | Backend detects `git status --porcelain` non-empty, returns error: "Working tree not clean, commit or stash first" |
| Revert/Cherry-pick conflict | Backend returns conflict info, Toast: "Conflict occurred, resolve manually" — no auto-resolve; CHANGES tab shows conflict files for manual resolution |
| Amend on non-HEAD commit | Frontend hides the menu item — only shown for HEAD commit |
| Illegal/duplicate tag or branch name | Backend returns stderr, Toast shows reason |
| Push network failure | Toast: "Push failed: <error>" — local commit is NOT rolled back |

All new backend methods return `error` containing git stderr (consistent with existing `GitLog` convention). Frontend does not parse error codes — displays backend error text via Toast.

## Post-Operation Refresh

All write operations (commit, stage/unstage, checkout, tag, branch, revert, cherry-pick, reset, amend) emit the existing `commit-history-changed` event after completion. This triggers:

- HISTORY list refresh (`loadCommitHistory`)
- CHANGES list refresh (`loadChangeStats`)

## Testing

### Backend unit tests

Each new method tested with a temporary git repo: `git init` in temp dir → create commits → call method → assert git state. Key edge cases:

- Empty staged commit
- Root commit diff (no `hash^`)
- Illegal branch/tag names
- Detached HEAD operations
- Dirty working tree blocking

### Frontend tests

- CHANGES tab: stage/unstage flow, multi-select, commit message input, button disabled states
- Context menu: correct item visibility (Amend only on HEAD), confirmation dialog triggers

### Manual verification

Phase 1: stage single file → Staged section → unstage → back; multi-select stage; commit → HISTORY refresh; commit & push → remote updated; click HISTORY commit → Preview split-view; right-click copy hash → clipboard correct; checkout commit → detached HEAD in TitleBar.

Phase 2: revert → new reverse commit; reset --hard → working tree clean (confirmation first); cherry-pick across branches.

## Component Structure

```
ChangesList.tsx
├── ChangesTab
│   ├── UnstagedSection    (header + checkbox file rows + Stage button)
│   ├── StagedSection      (header + checkbox file rows + Unstage button)
│   └── CommitSection      (textarea + Commit / Commit & Push buttons)
└── HistoryTab
    ├── CommitRow          (onClick → setPreviewCommit, onContextMenu → menu)
    └── CommitContextMenu  (portal-rendered, reuses SessionContextMenu pattern)

PreviewPanel.tsx
└── mode='commit'
    └── CommitSplitView
        ├── CommitFileList  (left, reuses ChangeStat row rendering)
        └── DiffView        (right, reuses existing diff rendering)
```
