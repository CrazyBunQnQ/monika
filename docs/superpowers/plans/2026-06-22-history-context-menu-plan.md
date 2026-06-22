# HISTORY Context Menu & CHANGES Commit Workflow — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn CHANGES + HISTORY tabs into a complete git client: three-section staging/commit layout in CHANGES, right-click context menu with full git operations in HISTORY, and commit-detail split-view in Preview panel.

**Architecture:** Backend extends `FileService` and `App` with git operations (stage/unstage, commit, checkout, tag, revert, cherry-pick, reset). Frontend rewrites `ChangesTab` into three sections (Unstaged/Staged/Commit), adds `onClick` and `onContextMenu` to `CommitRow`, extends `PreviewPanel` with `mode: 'commit'` split-view, and reuses existing `ConfirmModal` for danger confirmations.

**Tech Stack:** Go (backend, uses `os/exec` for git commands), React/TypeScript (frontend, Zustand store, dockview panels), Wails auto-generated bindings connect frontend to backend.

**Phases:**
- **Phase 1 (Tasks 1–11):** CHANGES staging + commit, HISTORY read-only menu (view/copy/checkout/tag/branch), commit split-view
- **Phase 2 (Tasks 12–14):** HISTORY writing operations (revert/cherry-pick/reset/amend) + danger confirmations

---

## File Structure

```
internal/api/
  types.go              — extend ChangeStat, add CommitDetail
  file_service.go       — rewrite ListChangeStats, add StageFiles/UnstageFiles/GetStagedDiff
  app.go                — add Commit, CommitAndPush, GitShow, GetCommitFileDiff,
                          CheckoutCommit, CreateTag, CreateBranchAt (Phase 1),
                          RevertCommit, CherryPickCommit, ResetToCommit, AmendMessage (Phase 2)

frontend/src/
  store/index.ts        — extend PreviewState, add stage/unstage/commit store actions,
                          add feedback state for inline messages
  components/ChangesList/
    ChangesList.tsx     — rewrite ChangesTab, add HistoryTab context menu + onClick,
                          add CommitContextMenu component
  components/Preview/
    PreviewPanel.tsx    — add mode='commit' rendering with CommitSplitView
```

---

### Task 1: Extend Backend Types

**Files:**
- Modify: `internal/api/types.go`

- [ ] **Step 1: Add Status and Staged fields to ChangeStat**

```go
// Find ChangeStat at ~line 112 and update:
type ChangeStat struct {
    Path    string `json:"path"`
    Added   int    `json:"added"`
    Deleted int    `json:"deleted"`
    Status  string `json:"status"`  // "modified"|"added"|"deleted"|"renamed"|"untracked"|"unmerged"
    Staged  bool   `json:"staged"`
}
```

- [ ] **Step 2: Add CommitDetail type after DiffResult**

```go
// Add after DiffResult (~line 111):
type CommitDetail struct {
    Hash    string       `json:"hash"`
    Author  string       `json:"author"`
    Date    string       `json:"date"`
    Message string       `json:"message"`
    Files   []ChangeStat `json:"files"`
}
```

- [ ] **Step 3: Build to verify types compile**

Run: `cd d:/git/monika && go build ./...`

Expected: Compiles, but ChangeStat consumers (ListChangeStats) may show field assignment errors since existing code doesn't set Status/Staged. Fix in Task 2.

---

### Task 2: Rewrite ListChangeStats for Staged/Unstaged Distinction

**Files:**
- Modify: `internal/api/file_service.go:558-627`

- [ ] **Step 1: Replace ListChangeStats implementation**

Replace the current `ListChangeStats` method (lines 558–627) with:

```go
func (f *FileService) ListChangeStats() ([]ChangeStat, error) {
    stats := make([]ChangeStat, 0)

    // Run three git commands in parallel to get full picture:
    // 1. git diff --numstat (unstaged tracked changes)
    // 2. git diff --cached --numstat (staged changes)
    // 3. git status --porcelain -uall (untracked + unmerged)

    // 1. Unstaged tracked changes
    unstagedCmd := command("git", "diff", "--numstat")
    unstagedCmd.Dir = f.projectDir
    if out, err := unstagedCmd.Output(); err == nil {
        for _, line := range strings.Split(string(out), "\n") {
            if line == "" { continue }
            fields := strings.Fields(line)
            if len(fields) < 3 { continue }
            added, _ := strconv.Atoi(fields[0])
            deleted, _ := strconv.Atoi(fields[1])
            if added == 0 && deleted == 0 && fields[0] == "-" && fields[1] == "-" { continue }
            stats = append(stats, ChangeStat{
                Path: fields[2], Added: added, Deleted: deleted,
                Status: "modified", Staged: false,
            })
        }
    }

    // 2. Staged changes
    stagedCmd := command("git", "diff", "--cached", "--numstat")
    stagedCmd.Dir = f.projectDir
    if out, err := stagedCmd.Output(); err == nil {
        for _, line := range strings.Split(string(out), "\n") {
            if line == "" { continue }
            fields := strings.Fields(line)
            if len(fields) < 3 { continue }
            added, _ := strconv.Atoi(fields[0])
            deleted, _ := strconv.Atoi(fields[1])
            if added == 0 && deleted == 0 && fields[0] == "-" && fields[1] == "-" { continue }
            stats = append(stats, ChangeStat{
                Path: fields[2], Added: added, Deleted: deleted,
                Status: "modified", Staged: true,
            })
        }
    }

    // 3. Untracked and unmerged files via status --porcelain
    statusCmd := command("git", "status", "--porcelain", "-uall")
    statusCmd.Dir = f.projectDir
    if statusOut, err := statusCmd.Output(); err == nil {
        for _, line := range strings.Split(string(statusOut), "\n") {
            if len(line) < 3 { continue }
            xy := line[0:2]
            filename := strings.TrimSpace(line[3:])
            if idx := strings.Index(filename, " -> "); idx >= 0 {
                filename = filename[idx+4:]
            }
            if filename == "" { continue }

            // Skip directories
            absPath := filepath.Join(f.projectDir, filename)
            if info, err2 := os.Stat(absPath); err2 != nil || info.IsDir() { continue }

            // Handle unmerged (conflicts)
            if xy[0] == 'U' || xy[1] == 'U' {
                stats = append(stats, ChangeStat{
                    Path: filename, Added: 0, Deleted: 0,
                    Status: "unmerged", Staged: false,
                })
                continue
            }

            // Handle untracked
            if xy == "??" {
                data, err2 := os.ReadFile(absPath)
                total := 0
                if err2 == nil {
                    total = len(strings.Split(string(data), "\n"))
                }
                stats = append(stats, ChangeStat{
                    Path: filename, Added: total, Deleted: 0,
                    Status: "untracked", Staged: false,
                })
            }
        }
    }

    return stats, nil
}
```

- [ ] **Step 2: Verify build**

Run: `cd d:/git/monika && go build ./...`

---

### Task 3: Add StageFiles and UnstageFiles

**Files:**
- Modify: `internal/api/file_service.go` (add two methods)

- [ ] **Step 1: Add UnstageFiles to FileService**

Add after `ListChangeStats`:

```go
func (f *FileService) UnstageFiles(paths []string) error {
    if len(paths) == 0 { return nil }
    args := []string{"reset", "HEAD", "--"}
    args = append(args, paths...)
    cmd := command("git", args...)
    cmd.Dir = f.projectDir
    out, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("failed to unstage: %s. %s", err.Error(), strings.TrimSpace(string(out)))
    }
    return nil
}
```

- [ ] **Step 2: Add UnstageFiles to App**

Add to `app.go`:

```go
func (a *App) UnstageFiles(projectPath string, paths []string) error {
    fs := a.getFileService(projectPath)
    return fs.UnstageFiles(paths)
}
```

- [ ] **Step 3: Add StageFiles to FileService**

```go
func (f *FileService) StageFiles(paths []string) error {
    if len(paths) == 0 { return nil }
    args := []string{"add", "--"}
    args = append(args, paths...)
    cmd := command("git", args...)
    cmd.Dir = f.projectDir
    out, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("failed to stage: %s. %s", err.Error(), strings.TrimSpace(string(out)))
    }
    return nil
}
```

- [ ] **Step 4: Add StageFiles to App**

```go
func (a *App) StageFiles(projectPath string, paths []string) error {
    fs := a.getFileService(projectPath)
    return fs.StageFiles(paths)
}
```

- [ ] **Step 5: Verify build**

Run: `cd d:/git/monika && go build ./...`

---

### Task 4: Add Commit and CommitAndPush

**Files:**
- Modify: `internal/api/app.go`

- [ ] **Step 1: Add Commit method**

```go
func (a *App) Commit(projectPath string, message string) error {
    if strings.TrimSpace(message) == "" {
        return fmt.Errorf("commit message must not be empty")
    }
    cmd := command("git", "commit", "-m", message)
    cmd.Dir = projectPath
    out, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("commit failed: %s. %s", err.Error(), strings.TrimSpace(string(out)))
    }
    a.emitCommitHistoryChangedIfChanged()
    return nil
}
```

- [ ] **Step 2: Add CommitAndPush method**

```go
func (a *App) CommitAndPush(projectPath string, message string) error {
    if err := a.Commit(projectPath, message); err != nil {
        return err
    }
    cmd := command("git", "push")
    cmd.Dir = projectPath
    out, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("push failed (commit succeeded): %s. %s", err.Error(), strings.TrimSpace(string(out)))
    }
    return nil
}
```

- [ ] **Step 3: Verify build**

Run: `cd d:/git/monika && go build ./...`

---

### Task 5: Add GetStagedFileDiff

**Files:**
- Modify: `internal/api/file_service.go`
- Modify: `internal/api/app.go`

- [ ] **Step 1: Add GetStagedDiff to FileService**

```go
func (f *FileService) GetStagedDiff(filePath string) (DiffResult, error) {
    // Get staged version from git index
    stagedCmd := command("git", "show", ":"+filePath)
    stagedCmd.Dir = f.projectDir
    stagedOut, stagedErr := stagedCmd.Output()
    stagedContent := ""
    if stagedErr == nil {
        stagedContent = string(stagedOut)
    }

    // Get HEAD version
    headCmd := command("git", "show", "HEAD:"+filePath)
    headCmd.Dir = f.projectDir
    headOut, headErr := headCmd.Output()
    headContent := ""
    if headErr == nil {
        headContent = string(headOut)
    }

    lines := computeUnifiedDiff(filePath, headContent, stagedContent)
    return DiffResult{
        FilePath: filePath,
        Lines:    lines,
        Old:      headContent,
        New:      stagedContent,
    }, nil
}
```

- [ ] **Step 2: Add GetStagedFileDiff to App**

```go
func (a *App) GetStagedFileDiff(projectPath, filePath string) (*DiffResult, error) {
    fs := a.getFileService(projectPath)
    dr, err := fs.GetStagedDiff(filePath)
    if err != nil {
        return nil, err
    }
    return &dr, nil
}
```

- [ ] **Step 3: Verify build**

Run: `cd d:/git/monika && go build ./...`

---

### Task 6: Add GitShow and GetCommitFileDiff

**Files:**
- Modify: `internal/api/app.go`

- [ ] **Step 1: Add GitShow method**

```go
func (a *App) GitShow(projectPath, hash string) (*CommitDetail, error) {
    // Use --numstat to get file list
    cmd := command("git", "show", "--numstat", "--pretty=format:%H%n%an%n%ar%n%s", "--no-color", hash)
    cmd.Dir = projectPath
    out, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("git show failed: %s. %s", err.Error(), string(out))
    }

    lines := strings.Split(strings.TrimSpace(string(out)), "\n")
    if len(lines) < 4 {
        return nil, fmt.Errorf("unexpected git show output")
    }

    detail := &CommitDetail{
        Hash:    lines[0],
        Author:  lines[1],
        Date:    lines[2],
        Message: lines[3],
        Files:   make([]ChangeStat, 0),
    }

    // Parse numstat lines (start from index 4, skip empty line separator)
    for i := 4; i < len(lines); i++ {
        line := lines[i]
        if line == "" { continue }
        fields := strings.Fields(line)
        if len(fields) < 3 { continue }
        added, _ := strconv.Atoi(fields[0])
        deleted, _ := strconv.Atoi(fields[1])
        path := fields[2]
        if added == 0 && deleted == 0 && fields[0] == "-" && fields[1] == "-" { continue }
        detail.Files = append(detail.Files, ChangeStat{
            Path: path, Added: added, Deleted: deleted,
        })
    }

    return detail, nil
}
```

Add `strconv` import to `app.go` if not already present:
```go
import (
    // ...existing imports...
    "strconv"
    // ...
)
```

- [ ] **Step 2: Add GetCommitFileDiff method**

```go
func (a *App) GetCommitFileDiff(projectPath, hash, filePath string) (*DiffResult, error) {
    // Get old version: parent of the commit
    oldCmd := command("git", "show", hash+"^:"+filePath)
    oldCmd.Dir = projectPath
    oldOut, oldErr := oldCmd.Output()
    oldContent := ""
    if oldErr == nil {
        oldContent = string(oldOut)
    }
    // For root commit (no parent), oldContent stays empty

    // Get new version: this commit's version
    newCmd := command("git", "show", hash+":"+filePath)
    newCmd.Dir = projectPath
    newOut, newErr := newCmd.Output()
    newContent := ""
    if newErr == nil {
        newContent = string(newOut)
    }

    lines := computeUnifiedDiff(filePath, oldContent, newContent)
    return &DiffResult{
        FilePath: filePath,
        Lines:    lines,
        Old:      oldContent,
        New:      newContent,
    }, nil
}
```

- [ ] **Step 3: Verify build**

Run: `cd d:/git/monika && go build ./...`

---

### Task 7: Add Phase 1 Navigation Operations (Checkout, Tag, Branch)

**Files:**
- Modify: `internal/api/app.go`

- [ ] **Step 1: Add CheckoutCommit**

```go
func (a *App) CheckoutCommit(projectPath, hash string) error {
    // Guard: check for unresolved merge conflicts
    if files := hasUnmergedFiles(projectPath); len(files) > 0 {
        return fmt.Errorf("UNMERGED_FILES:%s", strings.Join(files, ","))
    }

    // Auto-stash if working tree has tracked changes
    stashed, err := autoStash(projectPath)
    if err != nil {
        return err
    }

    cmd := command("git", "checkout", hash)
    cmd.Dir = projectPath
    out, err := cmd.CombinedOutput()
    if err != nil {
        if stashed {
            _ = autoStashPop(projectPath)
        }
        return fmt.Errorf("checkout failed: %s. %s", err.Error(), strings.TrimSpace(string(out)))
    }

    a.setProjectBranch(projectPath, hash[:7])
    if stashed {
        _ = autoStashPop(projectPath)
    }
    return nil
}
```

- [ ] **Step 2: Add CreateTag**

```go
func (a *App) CreateTag(projectPath, hash, tagName string) error {
    if tagName == "" {
        return fmt.Errorf("tag name must not be empty")
    }
    cmd := command("git", "tag", tagName, hash)
    cmd.Dir = projectPath
    out, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("failed to create tag: %s. %s", err.Error(), strings.TrimSpace(string(out)))
    }
    return nil
}
```

- [ ] **Step 3: Add CreateBranchAt**

```go
func (a *App) CreateBranchAt(projectPath, hash, branchName string) error {
    if err := validateBranchName(branchName); err != nil {
        return err
    }
    cmd := command("git", "branch", branchName, hash)
    cmd.Dir = projectPath
    out, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("failed to create branch: %s. %s", err.Error(), strings.TrimSpace(string(out)))
    }
    return nil
}
```

- [ ] **Step 4: Verify build**

Run: `cd d:/git/monika && go build ./...`

---

### Task 8: Frontend Store Changes

**Files:**
- Modify: `frontend/src/store/index.ts`

- [ ] **Step 1: Extend PreviewState type**

Find `interface PreviewState` (~line 96) and update `mode` union:

```typescript
interface PreviewState {
    mode: 'file' | 'diff' | 'task' | 'commit' | null
    filePath: string | null
    fileName: string | null
    fileContent: string | null
    diffLines: string[] | null
    conflictAiContent?: string | null
    conflictActive?: boolean
    commitDetail?: CommitDetail | null       // added
    commitFiles?: ChangeStat[] | null        // added
    commitHash?: string | null               // added
}
```

- [ ] **Step 2: Add store state fields**

Find the initial state (~line 391) and add after `changeStats`:

```typescript
changeStats: { stats: [], loading: false, error: '' },
commitHistory: { commits: [], loading: false, error: '' },
feedback: { message: '', type: 'info' as 'info' | 'error' | 'success' },  // added
```

Add the type for feedback (~line 192 area):

```typescript
feedback: { message: string; type: 'info' | 'error' | 'success' }
```

- [ ] **Step 3: Add store actions**

Add to the `AppState` type and implementation:

```typescript
// In type definition:
stageFiles: (paths: string[]) => Promise<void>
unstageFiles: (paths: string[]) => Promise<void>
commitChanges: (message: string, push: boolean) => Promise<void>
setPreviewCommit: (hash: string) => Promise<void>
setCommitFileDiff: (filePath: string) => Promise<void>
clearFeedback: () => void

// In set() call (after existing store methods):
stageFiles: async (paths) => {
    const { projectPath, loadChangeStats } = get()
    if (!projectPath) return
    try {
        await App.StageFiles(projectPath, paths)
        await loadChangeStats()
        set({ feedback: { message: `${paths.length} file(s) staged`, type: 'success' } })
    } catch (err: any) {
        set({ feedback: { message: err?.message || 'Failed to stage', type: 'error' } })
    }
},

unstageFiles: async (paths) => {
    const { projectPath, loadChangeStats } = get()
    if (!projectPath) return
    try {
        await App.UnstageFiles(projectPath, paths)
        await loadChangeStats()
        set({ feedback: { message: `${paths.length} file(s) unstaged`, type: 'success' } })
    } catch (err: any) {
        set({ feedback: { message: err?.message || 'Failed to unstage', type: 'error' } })
    }
},

commitChanges: async (message, push) => {
    const { projectPath, loadChangeStats } = get()
    if (!projectPath) return
    try {
        if (push) {
            await App.CommitAndPush(projectPath, message)
        } else {
            await App.Commit(projectPath, message)
        }
        await loadChangeStats()
        get().loadCommitHistory()
        set({ feedback: { message: push ? 'Committed & pushed' : 'Committed', type: 'success' } })
    } catch (err: any) {
        set({ feedback: { message: err?.message || 'Commit failed', type: 'error' } })
    }
},

setPreviewCommit: async (hash) => {
    const { projectPath } = get()
    if (!projectPath) return
    try {
        const detail = await App.GitShow(projectPath, hash)
        set({
            preview: {
                mode: 'commit',
                filePath: null,
                fileName: detail.message,
                fileContent: null,
                diffLines: null,
                commitDetail: detail,
                commitFiles: detail.files || [],
                commitHash: hash,
                conflictAiContent: null,
                conflictActive: false,
            },
            selectedBgTaskId: null,
        })
    } catch {
        set({ feedback: { message: 'Failed to load commit details', type: 'error' } })
    }
},

setCommitFileDiff: async (filePath) => {
    const { projectPath, preview } = get()
    if (!projectPath || !preview.commitHash) return
    try {
        const result = await App.GetCommitFileDiff(projectPath, preview.commitHash, filePath)
        set({
            preview: {
                ...preview,
                filePath: filePath,
                fileName: filePath.split('/').pop() || filePath,
                diffLines: result.lines || [],
            },
        })
    } catch {
        set({ feedback: { message: 'Failed to load file diff', type: 'error' } })
    }
},

clearFeedback: () => set({ feedback: { message: '', type: 'info' } }),
```

Also add to the `AppState` type interface:

```typescript
stageFiles: (paths: string[]) => Promise<void>
unstageFiles: (paths: string[]) => Promise<void>
commitChanges: (message: string, push: boolean) => Promise<void>
setPreviewCommit: (hash: string) => Promise<void>
setCommitFileDiff: (filePath: string) => Promise<void>
clearFeedback: () => void
```

- [ ] **Step 4: Add loadChangeStats to the type/store interface if not present**

The `useChangeWatcher` hook calls `setChangeStats` directly. We need a `loadChangeStats` method. Check if it exists; if not:

```typescript
// Add to type:
loadChangeStats: () => Promise<void>

// Add implementation near loadCommitHistory:
loadChangeStats: async () => {
    const { projectPath } = get()
    if (!projectPath) return
    set((s) => ({ changeStats: { ...s.changeStats, loading: true, error: '' } }))
    try {
        const stats = await App.ListChangeStats(projectPath)
        set({ changeStats: { stats: Array.isArray(stats) ? stats : [], loading: false, error: '' } })
    } catch {
        set((s) => ({ changeStats: { ...s.changeStats, loading: false, error: 'Failed to load changes' } }))
    }
},
```

- [ ] **Step 5: Clear preview state on reset**

In the clear/reset methods, reset commit fields too. Find `clearPreview`:

```typescript
clearPreview: () => set({
    preview: {
        mode: null, filePath: null, fileName: null, fileContent: null,
        diffLines: null, conflictAiContent: null, conflictActive: false,
        commitDetail: null, commitFiles: null, commitHash: null,
    },
}),
```

- [ ] **Step 6: Verify TypeScript compiles**

Run: `cd d:/git/monika/frontend && npx tsc --noEmit`

Expected: may have errors about missing App methods (StageFiles etc.) — those will be auto-generated by Wails when backend compiles. Ignore for now.

---

### Task 9: Rewrite ChangesTab with Three-Section Layout

**Files:**
- Modify: `frontend/src/components/ChangesList/ChangesList.tsx`

- [ ] **Step 1: Import additional dependencies**

At the top of the file, add imports:

```typescript
import { useState, useEffect, useRef, useMemo, useCallback } from 'react'
import { createPortal } from 'react-dom'
```

(These are already there — just verify.)

- [ ] **Step 2: Write the new ChangesTab component**

Replace the existing `ChangesTab` function (lines 103–243) with:

```tsx
function ChangesTab({ effectivePath }: { effectivePath: string }) {
    const changes = useStore((s) => s.changeStats)
    const setPreviewDiff = useStore((s) => s.setPreviewDiff)
    const setPreviewFile = useStore((s) => s.setPreviewFile)
    const stageFiles = useStore((s) => s.stageFiles)
    const unstageFiles = useStore((s) => s.unstageFiles)
    const commitChanges = useStore((s) => s.commitChanges)
    const feedback = useStore((s) => s.feedback)
    const clearFeedback = useStore((s) => s.clearFeedback)
    const loadChangeStats = useStore((s) => s.loadChangeStats)
    const [selectedUnstaged, setSelectedUnstaged] = useState<Set<string>>(new Set())
    const [selectedStaged, setSelectedStaged] = useState<Set<string>>(new Set())
    const [commitMsg, setCommitMsg] = useState('')
    const [committing, setCommitting] = useState(false)

    // Clear feedback after 4s
    useEffect(() => {
        if (!feedback.message) return
        const t = setTimeout(() => clearFeedback(), 4000)
        return () => clearTimeout(t)
    }, [feedback.message])

    const unstaged = changes.stats.filter(s => !s.staged)
    const staged = changes.stats.filter(s => s.staged)

    const toggleUnstaged = useCallback((path: string) => {
        setSelectedUnstaged(prev => {
            const next = new Set(prev)
            next.has(path) ? next.delete(path) : next.add(path)
            return next
        })
    }, [])

    const toggleStaged = useCallback((path: string) => {
        setSelectedStaged(prev => {
            const next = new Set(prev)
            next.has(path) ? next.delete(path) : next.add(path)
            return next
        })
    }, [])

    const handleStage = async () => {
        if (selectedUnstaged.size === 0) return
        await stageFiles(Array.from(selectedUnstaged))
        setSelectedUnstaged(new Set())
    }

    const handleUnstage = async () => {
        if (selectedStaged.size === 0) return
        await unstageFiles(Array.from(selectedStaged))
        setSelectedStaged(new Set())
    }

    const handleCommit = async (push: boolean) => {
        if (!commitMsg.trim() || staged.length === 0) return
        setCommitting(true)
        await commitChanges(commitMsg, push)
        setCommitting(false)
        setCommitMsg('')
    }

    const handleFileClick = async (stat: ChangeStat) => {
        try {
            if (stat.staged) {
                const result = await MonikaApp.GetStagedFileDiff(effectivePath, stat.path)
                const fileName = stat.path.split('/').pop() || stat.path
                if (result && result.lines) {
                    setPreviewDiff(stat.path, fileName, result.lines)
                }
            } else {
                const result = await MonikaApp.GetFileDiff(effectivePath, stat.path)
                const fileName = stat.path.split('/').pop() || stat.path
                if (result && result.lines) {
                    setPreviewDiff(stat.path, fileName, result.lines)
                }
            }
        } catch {
            // ignore
        }
    }

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && e.ctrlKey && e.shiftKey) {
            e.preventDefault()
            handleCommit(true)
        } else if (e.key === 'Enter' && e.ctrlKey) {
            e.preventDefault()
            handleCommit(false)
        }
    }

    const basenameFn = (p: string) => p.split('/').pop() || p

    return (
        <div className="flex flex-col flex-1 overflow-hidden">
            {/* Feedback bar */}
            {feedback.message && (
                <div
                    className="shrink-0 px-2 py-1 text-[11px]"
                    style={{
                        color: feedback.type === 'error' ? 'var(--red)' :
                               feedback.type === 'success' ? 'var(--green)' : 'var(--text-dim)',
                        background: feedback.type === 'error' ? 'rgba(255,50,50,0.1)' :
                                    feedback.type === 'success' ? 'rgba(0,200,100,0.1)' : 'transparent',
                    }}
                >
                    {feedback.message}
                </div>
            )}

            {/* Unstaged section */}
            <div className="flex-1 overflow-y-auto" style={{ minHeight: 0 }}>
                <div className="flex items-center justify-between px-2 py-1 text-[11px] text-[var(--text-dim)] sticky top-0" style={{ background: 'var(--bg-sidebar)' }}>
                    <span>Unstaged ({unstaged.length})</span>
                    {selectedUnstaged.size > 0 && (
                        <button
                            className="px-2 py-0.5 rounded text-[11px] cursor-pointer"
                            style={{ background: 'var(--bg-active)', color: 'var(--accent)', border: 'none' }}
                            onClick={handleStage}
                        >
                            Stage ▶
                        </button>
                    )}
                </div>
                {unstaged.length === 0 ? (
                    <div className="py-2 text-[11px] text-[var(--text-dim)] px-2">
                        {changes.loading ? 'Loading...' : 'No unstaged changes'}
                    </div>
                ) : (
                    unstaged.map((stat) => {
                        const selected = selectedUnstaged.has(stat.path)
                        return (
                            <div
                                key={'un-'+stat.path}
                                className="flex items-center gap-1 cursor-pointer text-[12px] leading-[24px] rounded-md transition-colors mx-1 px-[6px]"
                                style={{
                                    color: 'var(--text-secondary)',
                                    background: selected ? 'var(--bg-active)' : 'transparent',
                                }}
                                onDoubleClick={() => handleStage()}
                            >
                                <input
                                    type="checkbox"
                                    checked={selected}
                                    onChange={() => toggleUnstaged(stat.path)}
                                    onClick={(e) => e.stopPropagation()}
                                    style={{ accentColor: 'var(--accent)', cursor: 'pointer' }}
                                />
                                <span
                                    className="truncate flex-1"
                                    onClick={() => handleFileClick(stat)}
                                    title={stat.path}
                                >
                                    {basenameFn(stat.path)}
                                </span>
                                {stat.added > 0 && (
                                    <span className="text-[11px] flex-shrink-0" style={{ color: 'var(--green)' }}>+{stat.added}</span>
                                )}
                                {stat.deleted > 0 && (
                                    <span className="text-[11px] flex-shrink-0 ml-0.5" style={{ color: 'var(--red)' }}>-{stat.deleted}</span>
                                )}
                            </div>
                        )
                    })
                )}

                {/* Divider */}
                <div style={{ height: '1px', background: 'var(--border)', margin: '4px 0' }} />

                {/* Staged section */}
                <div className="flex items-center justify-between px-2 py-1 text-[11px] text-[var(--text-dim)] sticky top-0" style={{ background: 'var(--bg-sidebar)', zIndex: 1 }}>
                    <span>Staged ({staged.length})</span>
                    {selectedStaged.size > 0 && (
                        <button
                            className="px-2 py-0.5 rounded text-[11px] cursor-pointer"
                            style={{ background: 'var(--bg-active)', color: 'var(--accent)', border: 'none' }}
                            onClick={handleUnstage}
                        >
                            ◀ Unstage
                        </button>
                    )}
                </div>
                {staged.length === 0 ? (
                    <div className="py-2 text-[11px] text-[var(--text-dim)] px-2">
                        No staged changes
                    </div>
                ) : (
                    staged.map((stat) => {
                        const selected = selectedStaged.has(stat.path)
                        return (
                            <div
                                key={'st-'+stat.path}
                                className="flex items-center gap-1 cursor-pointer text-[12px] leading-[24px] rounded-md transition-colors mx-1 px-[6px]"
                                style={{
                                    color: 'var(--text-secondary)',
                                    background: selected ? 'var(--bg-active)' : 'transparent',
                                }}
                                onDoubleClick={() => handleUnstage()}
                            >
                                <input
                                    type="checkbox"
                                    checked={selected}
                                    onChange={() => toggleStaged(stat.path)}
                                    onClick={(e) => e.stopPropagation()}
                                    style={{ accentColor: 'var(--accent)', cursor: 'pointer' }}
                                />
                                <span
                                    className="truncate flex-1"
                                    onClick={() => handleFileClick(stat)}
                                    title={stat.path}
                                >
                                    {basenameFn(stat.path)}
                                </span>
                                {stat.added > 0 && (
                                    <span className="text-[11px] flex-shrink-0" style={{ color: 'var(--green)' }}>+{stat.added}</span>
                                )}
                                {stat.deleted > 0 && (
                                    <span className="text-[11px] flex-shrink-0 ml-0.5" style={{ color: 'var(--red)' }}>-{stat.deleted}</span>
                                )}
                            </div>
                        )
                    })
                )}
            </div>

            {/* Commit section */}
            <div
                className="shrink-0 border-t border-[var(--border)]"
                style={{ padding: '8px' }}
            >
                <textarea
                    className="w-full rounded-md text-[12px]"
                    style={{
                        background: 'var(--bg-input)',
                        border: '1px solid var(--border)',
                        color: 'var(--text-primary)',
                        fontFamily: 'var(--font-sans)',
                        padding: '6px 8px',
                        resize: 'none',
                        height: '50px',
                        outline: 'none',
                    }}
                    placeholder="Commit message..."
                    value={commitMsg}
                    onChange={(e) => setCommitMsg(e.target.value)}
                    onKeyDown={handleKeyDown}
                    disabled={committing}
                />
                <div className="flex gap-2 mt-2">
                    <button
                        className="flex-1 px-3 py-1.5 rounded-md text-[12px] font-medium cursor-pointer transition-colors"
                        style={{
                            background: staged.length > 0 && commitMsg.trim() ? 'var(--accent)' : 'var(--bg-hover)',
                            color: staged.length > 0 && commitMsg.trim() ? '#fff' : 'var(--text-dim)',
                            border: 'none',
                            opacity: staged.length > 0 && commitMsg.trim() ? 1 : 0.5,
                        }}
                        onClick={() => handleCommit(false)}
                        disabled={committing || staged.length === 0 || !commitMsg.trim()}
                    >
                        {committing ? 'Committing...' : 'Commit'}
                    </button>
                    <button
                        className="flex-1 px-3 py-1.5 rounded-md text-[12px] font-medium cursor-pointer transition-colors"
                        style={{
                            background: staged.length > 0 && commitMsg.trim() ? 'var(--accent)' : 'var(--bg-hover)',
                            color: staged.length > 0 && commitMsg.trim() ? '#fff' : 'var(--text-dim)',
                            border: 'none',
                            opacity: staged.length > 0 && commitMsg.trim() ? 1 : 0.5,
                        }}
                        onClick={() => handleCommit(true)}
                        disabled={committing || staged.length === 0 || !commitMsg.trim()}
                    >
                        {committing ? 'Pushing...' : 'Commit & Push'}
                    </button>
                </div>
            </div>
        </div>
    )
}
```

- [ ] **Step 2: Verify TypeScript compiles**

Run: `cd d:/git/monika/frontend && npx tsc --noEmit`

---

### Task 10: Add CommitRow onClick + Context Menu

**Files:**
- Modify: `frontend/src/components/ChangesList/ChangesList.tsx`

- [ ] **Step 1: Update CommitRow with onClick and onContextMenu**

Replace the existing `CommitRow` function (lines 288–334) with:

```tsx
function CommitRow({ commit, onClick, onContextMenu }: { commit: CommitInfo; onClick: () => void; onContextMenu: (e: React.MouseEvent) => void }) {
    return (
        <div
            className="flex items-center gap-1 text-[12px] leading-[22px] rounded-md transition-colors duration-150 px-1 cursor-pointer hover:bg-[var(--bg-hover)]"
            style={{
                fontFamily: 'var(--font-sans)',
                color: 'var(--text-secondary)',
            }}
            onClick={onClick}
            onContextMenu={onContextMenu}
        >
            <span
                className="flex-shrink-0 select-none"
                style={{
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--text-dim)',
                    whiteSpace: 'pre',
                    lineHeight: '22px',
                    fontSize: '11px',
                }}
            >
                {commit.graph_line}
            </span>
            <span
                className="flex-shrink-0"
                style={{ color: 'var(--accent)', fontFamily: 'var(--font-mono)', fontSize: '11px', width: '8ch' }}
            >
                {commit.hash.slice(0, 7)}
            </span>
            {commit.refs && <RefTags refs={commit.refs} />}
            <span className="truncate flex-1 min-w-0" style={{ color: 'var(--text-primary)' }}>
                {commit.message}
            </span>
            <span className="flex-shrink-0 flex items-center" style={{ color: 'var(--text-dim)', fontSize: '11px', width: '25ch' }}>
                <span>{commit.author}</span>
                <span className="ml-auto">{commit.date}</span>
            </span>
        </div>
    )
}
```

- [ ] **Step 2: Update HistoryTab with state and handlers**

Replace the existing `HistoryTab` function (lines 245–286) with:

```tsx
function HistoryTab({ active, effectivePath }: { active: boolean; effectivePath: string }) {
    const commitHistory = useStore((s) => s.commitHistory)
    const loadCommitHistory = useStore((s) => s.loadCommitHistory)
    const setPreviewCommit = useStore((s) => s.setPreviewCommit)
    const [contextMenu, setContextMenu] = useState<{ x: number; y: number; commit: CommitInfo } | null>(null)
    const menuRef = useRef<HTMLDivElement>(null)
    const menuJustOpened = useRef(false)

    useEffect(() => {
        if (!active) return
        loadCommitHistory(effectivePath)
    }, [active, effectivePath])

    useEffect(() => {
        if (!contextMenu) return
        menuJustOpened.current = true
        const onClick = () => {
            if (menuJustOpened.current) { menuJustOpened.current = false; return }
            setContextMenu(null)
        }
        window.addEventListener('click', onClick)
        return () => window.removeEventListener('click', onClick)
    }, [contextMenu])

    const handleClick = (commit: CommitInfo) => {
        setPreviewCommit(commit.hash)
    }

    const handleContextMenu = (e: React.MouseEvent, commit: CommitInfo) => {
        e.preventDefault()
        e.stopPropagation()
        setContextMenu({ x: e.clientX, y: e.clientY, commit })
    }

    const handleCopyHash = (commit: CommitInfo) => {
        navigator.clipboard.writeText(commit.hash)
        setContextMenu(null)
    }

    const handleCopyMessage = (commit: CommitInfo) => {
        navigator.clipboard.writeText(commit.message)
        setContextMenu(null)
    }

    const renderContextMenu = () => {
        if (!contextMenu) return null
        const c = contextMenu.commit
        return createPortal(
            <div
                ref={menuRef}
                className="fixed"
                style={{
                    left: contextMenu.x,
                    top: contextMenu.y,
                    zIndex: 2000,
                    background: 'var(--bg-elevated)',
                    border: '1px solid var(--border)',
                    borderRadius: 'var(--radius-md)',
                    padding: '4px 0',
                    minWidth: '200px',
                    boxShadow: '0 8px 24px rgba(0,0,0,0.4)',
                    fontSize: '12px',
                    fontFamily: 'var(--font-sans)',
                }}
                onClick={(e) => e.stopPropagation()}
            >
                {/* View Details */}
                <ContextMenuItem icon="◆" label="View Details" onClick={() => { setContextMenu(null); handleClick(c) }} />
                <ContextMenuDivider />
                {/* Copy Hash */}
                <ContextMenuItem label="Copy Hash" onClick={() => handleCopyHash(c)} />
                {/* Copy Message */}
                <ContextMenuItem label="Copy Message" onClick={() => handleCopyMessage(c)} />
                <ContextMenuDivider />
                {/* Checkout Commit */}
                <ContextMenuItem label="Checkout Commit..." onClick={() => handleCheckoutCommit(c)} />
                {/* Create Tag */}
                <ContextMenuItem label="Create Tag..." onClick={() => handleCreateTag(c)} />
                {/* Create Branch at Commit */}
                <ContextMenuItem label="Create Branch at Commit..." onClick={() => handleCreateBranch(c)} />
                {/* Phase 2 items (omit for now) */}
            </div>,
            document.body
        )
    }

    if (commitHistory.loading && commitHistory.commits.length === 0) {
        return <div className="flex-1 overflow-y-auto" style={{ padding: '8px' }}><div className="py-4 text-[12px] text-[var(--text-dim)] px-1">Loading...</div></div>
    }
    if (commitHistory.error && commitHistory.commits.length === 0) {
        return <div className="flex-1 overflow-y-auto" style={{ padding: '8px' }}><div className="py-4 text-[12px] text-[var(--red)] px-1">{commitHistory.error}</div></div>
    }
    if (commitHistory.commits.length === 0) {
        return <div className="flex-1 overflow-y-auto" style={{ padding: '8px' }}><div className="py-4 text-[12px] text-[var(--text-dim)] px-1">No commits</div></div>
    }
    return (
        <>
            <div className="flex-1 overflow-y-auto" style={{ padding: '0 8px' }}>
                {commitHistory.commits.map((commit, idx) => (
                    <CommitRow
                        key={commit.hash + '-' + idx}
                        commit={commit}
                        onClick={() => handleClick(commit)}
                        onContextMenu={(e) => handleContextMenu(e, commit)}
                    />
                ))}
            </div>
            {renderContextMenu()}
        </>
    )
}
```

- [ ] **Step 3: Add ContextMenu helper components and action handlers (stubs for Phase 1)**

Add at the end of the file (or before the export):

```tsx
function ContextMenuItem({ icon, label, onClick }: { icon?: string; label: string; onClick: () => void }) {
    return (
        <div
            className="flex items-center gap-2.5 px-3 py-[5px] cursor-pointer transition-colors rounded-sm mx-1"
            style={{ color: 'var(--text-secondary)' }}
            onMouseEnter={(e) => { e.currentTarget.style.background = 'var(--bg-hover)'; e.currentTarget.style.color = 'var(--text-primary)' }}
            onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent'; e.currentTarget.style.color = 'var(--text-secondary)' }}
            onClick={onClick}
        >
            {icon && <span className="flex-shrink-0 flex items-center" style={{ opacity: 0.7, width: 14 }}>{icon}</span>}
            <span>{label}</span>
        </div>
    )
}

function ContextMenuDivider() {
    return <div style={{ height: '1px', background: 'var(--border)', margin: '4px 0' }} />
}
```

- [ ] **Step 4: Add action handler stubs inside HistoryTab**

Add these functions inside `HistoryTab` (used by the context menu):

```typescript
const handleCheckoutCommit = async (c: CommitInfo) => {
    setContextMenu(null)
    try {
        await MonikaApp.CheckoutCommit(effectivePath, c.hash)
        loadCommitHistory()
    } catch (err: any) {
        // error shown via feedback or inline
    }
}

const handleCreateTag = async (c: CommitInfo) => {
    setContextMenu(null)
    const name = prompt('Tag name:')
    if (!name) return
    try {
        await MonikaApp.CreateTag(effectivePath, c.hash, name)
    } catch (err: any) {
        // error shown via feedback
    }
}

const handleCreateBranch = async (c: CommitInfo) => {
    setContextMenu(null)
    const name = prompt('Branch name:')
    if (!name) return
    try {
        await MonikaApp.CreateBranchAt(effectivePath, c.hash, name)
    } catch (err: any) {
        // error shown via feedback
    }
}
```

- [ ] **Step 5: Verify TypeScript compiles**

Run: `cd d:/git/monika/frontend && npx tsc --noEmit`

---

### Task 11: Add Commit Split-View to PreviewPanel

**Files:**
- Modify: `frontend/src/components/Preview/PreviewPanel.tsx`

- [ ] **Step 1: Extract store bindings for commit mode**

Locate the existing PreviewPanel function (~line 606) and find where `preview` is destructured. Add commit-specific bindings:

```typescript
// Near the top of PreviewPanel, find the useStore call and add:
const commitDetail = useStore((s) => s.preview.commitDetail)
const commitFiles = useStore((s) => s.preview.commitFiles)
const commitHash = useStore((s) => s.preview.commitHash)
const setCommitFileDiff = useStore((s) => s.setCommitFileDiff)
```

Add a new state variable after existing state declarations:
```typescript
const showCommit = preview.mode === 'commit' && commitFiles
const [selectedCommitFile, setSelectedCommitFile] = useState<string | null>(null)
```

- [ ] **Step 2: Add commit-split-view rendering**

Find where `showDiff`, `showFile`, `showEmpty` are rendered (~lines 1808–1819). Add the commit mode before `showEmpty`:

```tsx
{
    showCommit && commitFiles && (
        <div className="flex-1 flex overflow-hidden">
            {/* Left: file list */}
            <div
                className="flex-shrink-0 overflow-y-auto border-r border-[var(--border)]"
                style={{ width: '200px', background: 'var(--bg-sidebar)' }}
            >
                <div className="px-2 pt-2 pb-1 text-[11px] text-[var(--text-dim)]" style={{ fontFamily: 'var(--font-sans)' }}>
                    {commitFiles.length} file{commitFiles.length !== 1 ? 's' : ''} changed
                </div>
                {commitFiles.map((f) => (
                    <div
                        key={f.path}
                        className="flex items-center gap-1 px-2 py-1 cursor-pointer text-[12px] leading-[20px] hover:bg-[var(--bg-hover)] truncate"
                        style={{
                            color: selectedCommitFile === f.path ? 'var(--text-primary)' : 'var(--text-secondary)',
                            background: selectedCommitFile === f.path ? 'var(--bg-active)' : 'transparent',
                        }}
                        onClick={() => {
                            setSelectedCommitFile(f.path)
                            setCommitFileDiff(f.path)
                        }}
                    >
                        <span className="truncate">{f.path.split('/').pop()}</span>
                        <span className="ml-auto text-[10px]" style={{ color: 'var(--green)' }}>+{f.added}</span>
                        {f.deleted > 0 && <span className="text-[10px]" style={{ color: 'var(--red)' }}>-{f.deleted}</span>}
                    </div>
                ))}
            </div>
            {/* Right: diff */}
            <div className="flex-1 flex flex-col overflow-hidden">
                {preview.diffLines ? (
                    <DiffView
                        lines={preview.diffLines}
                        fileName={preview.fileName || ''}
                        conflictActive={false}
                    />
                ) : (
                    <div className="flex-1 flex items-center justify-center">
                        <div className="text-[13px] text-[var(--text-dim)] select-none">
                            Select a file to view diff
                        </div>
                    </div>
                )}
            </div>
        </div>
    )
}
```

- [ ] **Step 3: Add commit metadata header above the split-view**

After the commit-split-view div but before the file list, show commit metadata:

```tsx
{
    showCommit && commitDetail && (
        <div className="flex flex-col flex-1 overflow-hidden">
            {/* Commit header */}
            <div
                className="flex items-center gap-2 px-3 py-2 shrink-0 border-b border-[var(--border)]"
                style={{ background: 'var(--bg-sidebar)' }}
            >
                <span className="font-mono text-[12px]" style={{ color: 'var(--accent)' }}>
                    {commitHash?.slice(0, 7)}
                </span>
                <span className="flex-1 text-[13px] font-medium truncate" style={{ color: 'var(--text-primary)' }}>
                    {commitDetail.message}
                </span>
                <span className="text-[11px]" style={{ color: 'var(--text-dim)' }}>
                    {commitDetail.author} · {commitDetail.date}
                </span>
            </div>
            {/* Split view */}
            ... (wrap the commit split-view content above here)
        </div>
    )
}
```

- [ ] **Step 4: Update the rendering logic**

Replace the earlier commit-split-view with the full wrapped version (combining steps 2 and 3):

```tsx
{
    showCommit && commitFiles && commitDetail && (
        <div className="flex flex-col flex-1 overflow-hidden">
            <div
                className="flex items-center gap-2 px-3 py-2 shrink-0 border-b border-[var(--border)]"
                style={{ background: 'var(--bg-sidebar)' }}
            >
                <span className="font-mono text-[12px]" style={{ color: 'var(--accent)' }}>
                    {commitHash?.slice(0, 7)}
                </span>
                <span className="flex-1 text-[13px] font-medium truncate" style={{ color: 'var(--text-primary)' }}>
                    {commitDetail.message}
                </span>
                <span className="text-[11px]" style={{ color: 'var(--text-dim)' }}>
                    {commitDetail.author} · {commitDetail.date}
                </span>
            </div>
            <div className="flex-1 flex overflow-hidden">
                <div
                    className="flex-shrink-0 overflow-y-auto border-r border-[var(--border)]"
                    style={{ width: '200px', background: 'var(--bg-sidebar)' }}
                >
                    <div className="px-2 pt-2 pb-1 text-[11px] text-[var(--text-dim)]" style={{ fontFamily: 'var(--font-sans)' }}>
                        {commitFiles.length} file{commitFiles.length !== 1 ? 's' : ''} changed
                    </div>
                    {commitFiles.map((f) => (
                        <div
                            key={f.path}
                            className="flex items-center gap-1 px-2 py-1 cursor-pointer text-[12px] leading-[20px] hover:bg-[var(--bg-hover)] truncate"
                            style={{
                                color: selectedCommitFile === f.path ? 'var(--text-primary)' : 'var(--text-secondary)',
                                background: selectedCommitFile === f.path ? 'var(--bg-active)' : 'transparent',
                            }}
                            onClick={() => {
                                setSelectedCommitFile(f.path)
                                setCommitFileDiff(f.path)
                            }}
                        >
                            <span className="truncate">{f.path.split('/').pop()}</span>
                            <span className="ml-auto text-[10px]" style={{ color: 'var(--green)' }}>+{f.added}</span>
                            {f.deleted > 0 && <span className="text-[10px]" style={{ color: 'var(--red)' }}>-{f.deleted}</span>}
                        </div>
                    ))}
                </div>
                <div className="flex-1 flex flex-col overflow-hidden">
                    {preview.diffLines ? (
                        <DiffView
                            lines={preview.diffLines}
                            fileName={preview.fileName || ''}
                            conflictActive={false}
                        />
                    ) : (
                        <div className="flex-1 flex items-center justify-center">
                            <div className="text-[13px] text-[var(--text-dim)] select-none">
                                Select a file to view diff
                            </div>
                        </div>
                    )}
                </div>
            </div>
        </div>
    )
}
```

- [ ] **Step 5: Verify TypeScript compiles**

Run: `cd d:/git/monika/frontend && npx tsc --noEmit`

---

### Task 12: Phase 2 — Backend History-Rewriting Operations

**Files:**
- Modify: `internal/api/app.go`

- [ ] **Step 1: Add RevertCommit**

```go
func (a *App) RevertCommit(projectPath, hash string) error {
    cmd := command("git", "revert", "--no-edit", hash)
    cmd.Dir = projectPath
    out, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("revert failed: %s. %s", err.Error(), strings.TrimSpace(string(out)))
    }
    a.emitCommitHistoryChangedIfChanged()
    return nil
}
```

- [ ] **Step 2: Add CherryPickCommit**

```go
func (a *App) CherryPickCommit(projectPath, hash string) error {
    cmd := command("git", "cherry-pick", hash)
    cmd.Dir = projectPath
    out, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("cherry-pick failed: %s. %s", err.Error(), strings.TrimSpace(string(out)))
    }
    a.emitCommitHistoryChangedIfChanged()
    return nil
}
```

- [ ] **Step 3: Add ResetToCommit**

```go
func (a *App) ResetToCommit(projectPath, hash, mode string) error {
    switch mode {
    case "soft", "mixed", "hard":
        // ok
    default:
        return fmt.Errorf("invalid reset mode: %s (use soft, mixed, or hard)", mode)
    }
    cmd := command("git", "reset", "--"+mode, hash)
    cmd.Dir = projectPath
    out, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("reset failed: %s. %s", err.Error(), strings.TrimSpace(string(out)))
    }
    a.emitCommitHistoryChangedIfChanged()
    return nil
}
```

- [ ] **Step 4: Add AmendMessage**

```go
func (a *App) AmendMessage(projectPath, message string) error {
    if strings.TrimSpace(message) == "" {
        return fmt.Errorf("commit message must not be empty")
    }
    cmd := command("git", "commit", "--amend", "-m", message)
    cmd.Dir = projectPath
    out, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("amend failed: %s. %s", err.Error(), strings.TrimSpace(string(out)))
    }
    a.emitCommitHistoryChangedIfChanged()
    return nil
}
```

- [ ] **Step 5: Verify build**

Run: `cd d:/git/monika && go build ./...`

---

### Task 13: Phase 2 — Frontend Menu Items + Confirm Modals

**Files:**
- Modify: `frontend/src/components/ChangesList/ChangesList.tsx`

- [ ] **Step 1: Add ConfirmModal import**

```typescript
import ConfirmModal from '../Chat/ConfirmModal'
```

- [ ] **Step 2: Add confirm modal state in HistoryTab**

Add inside `HistoryTab` function:

```typescript
const [confirmModal, setConfirmModal] = useState<{
    title: string; message: string; confirmLabel: string; variant: 'danger' | 'primary';
    onConfirm: () => Promise<void>;
} | null>(null)
```

- [ ] **Step 3: Add handler functions inside HistoryTab**

```typescript
const handleRevertCommit = (c: CommitInfo) => {
    setContextMenu(null)
    setConfirmModal({
        title: `Revert ${c.hash.slice(0, 7)}?`,
        message: `This will create a new commit that reverses "${c.message}".`,
        confirmLabel: 'Revert',
        variant: 'danger',
        onConfirm: async () => {
            await MonikaApp.RevertCommit(effectivePath, c.hash)
            loadCommitHistory()
            setConfirmModal(null)
        },
    })
}

const handleCherryPick = (c: CommitInfo) => {
    setContextMenu(null)
    setConfirmModal({
        title: 'Cherry-pick Commit?',
        message: `Apply "${c.hash.slice(0, 7)}: ${c.message}" to the current branch.`,
        confirmLabel: 'Cherry-pick',
        variant: 'primary',
        onConfirm: async () => {
            await MonikaApp.CherryPickCommit(effectivePath, c.hash)
            loadCommitHistory()
            setConfirmModal(null)
        },
    })
}

const handleReset = (c: CommitInfo, mode: 'soft' | 'mixed' | 'hard') => {
    setContextMenu(null)
    const descriptions = {
        soft: 'HEAD only — staged changes are kept.',
        mixed: 'HEAD + index — unstaged changes are kept (default).',
        hard: 'ALL changes will be permanently discarded.',
    }
    setConfirmModal({
        title: `Reset to ${c.hash.slice(0, 7)} (${mode})?`,
        message: descriptions[mode] + (mode === 'hard' ? ' This cannot be undone.' : ''),
        confirmLabel: `Reset ${mode}`,
        variant: mode === 'hard' ? 'danger' : 'primary',
        onConfirm: async () => {
            await MonikaApp.ResetToCommit(effectivePath, c.hash, mode)
            loadCommitHistory()
            setConfirmModal(null)
        },
    })
}

const handleAmendMessage = (c: CommitInfo) => {
    setContextMenu(null)
    const newMsg = prompt('New commit message:', c.message)
    if (!newMsg) return
    MonikaApp.AmendMessage(effectivePath, newMsg).then(() => {
        loadCommitHistory()
    }).catch(() => {
        // ignore for now
    })
}
```

- [ ] **Step 4: Add Phase 2 menu items to renderContextMenu**

Inside `renderContextMenu`, add after the Phase 1 menu items and another divider:

```tsx
{/* Phase 2 items */}
<ContextMenuDivider />
<ContextMenuItem label="Revert Commit" onClick={() => handleRevertCommit(c)} />
<ContextMenuItem label="Cherry-pick Commit" onClick={() => handleCherryPick(c)} />
{/* Reset submenu — for simplicity, use separate items */}
<ContextMenuItem label="Reset to Commit (Soft)" onClick={() => handleReset(c, 'soft')} />
<ContextMenuItem label="Reset to Commit (Mixed)" onClick={() => handleReset(c, 'mixed')} />
<ContextMenuItem variant="danger" label="Reset to Commit (Hard)" onClick={() => handleReset(c, 'hard')} />
{/* Amend — only show for HEAD */}
{/* Detect HEAD by checking if the first commit in list: */}
{c.hash === commitHistory.commits[0]?.hash && (
    <>
        <ContextMenuDivider />
        <ContextMenuItem label="Amend Message..." onClick={() => handleAmendMessage(c)} />
    </>
)}
```

- [ ] **Step 5: Add the confirm modal render**

At the end of HistoryTab's return, add:

```tsx
{confirmModal && (
    <ConfirmModal
        title={confirmModal.title}
        message={confirmModal.message}
        confirmLabel={confirmModal.confirmLabel}
        variant={confirmModal.variant}
        onConfirm={confirmModal.onConfirm}
        onCancel={() => setConfirmModal(null)}
    />
)}
```

- [ ] **Step 6: Update ContextMenuItem to support variant**

```tsx
function ContextMenuItem({ icon, label, onClick, variant }: { icon?: string; label: string; onClick: () => void; variant?: 'danger' }) {
    const color = variant === 'danger' ? 'var(--red)' : 'var(--text-secondary)'
    const hoverColor = variant === 'danger' ? 'var(--red)' : 'var(--text-primary)'
    return (
        <div
            className="flex items-center gap-2.5 px-3 py-[5px] cursor-pointer transition-colors rounded-sm mx-1"
            style={{ color }}
            onMouseEnter={(e) => { e.currentTarget.style.background = 'var(--bg-hover)'; e.currentTarget.style.color = hoverColor }}
            onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent'; e.currentTarget.style.color = color }}
            onClick={onClick}
        >
            {icon && <span className="flex-shrink-0 flex items-center" style={{ opacity: 0.7, width: 14 }}>{icon}</span>}
            <span>{label}</span>
        </div>
    )
}
```

- [ ] **Step 7: Verify TypeScript compiles**

Run: `cd d:/git/monika/frontend && npx tsc --noEmit`

---

### Task 14: Rebuild + Test

**Files:**
- (all of the above)

- [ ] **Step 1: Full backend build**

Run: `cd d:/git/monika && go build ./...`

- [ ] **Step 2: Full Wails build (generates frontend bindings)**

Run: `cd d:/git/monika && wails build`

This will regenerate `frontend/bindings/monika/` with all new backend methods.

- [ ] **Step 3: Frontend typecheck after bindings generation**

Run: `cd d:/git/monika/frontend && npx tsc --noEmit`

Expected: No errors. Fix any TypeScript issues.

- [ ] **Step 4: Manual smoke test**

1. Open the app in a git project
2. Verify CHANGES tab shows Unstaged/Staged/Commit sections
3. Stage a file → moves to Staged section
4. Write a commit message → Commit → HISTORY refreshes
5. Click HISTORY commit → Preview shows file list + diff
6. Right-click HISTORY commit → context menu appears
7. Copy Hash → clipboard
8. Checkout commit → detached HEAD
9. Create Tag → tag created
10. Revert → confirm modal → commit reversed
11. Reset --hard → confirm modal → reset applied
