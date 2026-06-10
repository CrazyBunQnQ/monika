# HISTORY Tab Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a HISTORY sub-tab to the existing CHANGES panel, displaying a graphical git log with local and remote commits.

**Architecture:** The existing `ChangesList` component is refactored into a `GitPanel` container with two internal tabs (CHANGES / HISTORY). A new Go backend method `GitLog` parses `git log --graph --all` output into structured `CommitInfo` objects. Frontend state is added to the Zustand store following the existing `changeStats` pattern.

**Tech Stack:** Go (git CLI via `os/exec`), React 18, Zustand, Tailwind CSS v4, dockview, Wails 3 bindings

---

## File Structure

| File | Responsibility |
|------|---------------|
| `internal/api/types.go` | Add `CommitInfo` struct |
| `internal/api/app.go` | Add `GitLog` method — executes git CLI, parses output |
| `frontend/bindings/monika/...` | Auto-regenerated — do NOT edit manually |
| `frontend/src/store/index.ts` | Add `commitHistory` state, `loadCommitHistory` action |
| `frontend/src/components/ChangesList/ChangesList.tsx` | Refactor into GitPanel with tab switching + HISTORY view |

---

### Task 1: Add `CommitInfo` type to Go backend

**Files:**
- Modify: `internal/api/types.go:101` (after `ChangeStat` struct)

- [ ] **Step 1: Add CommitInfo struct after ChangeStat**

Insert after line 101 (`}` closing `ChangeStat`):

```go

type CommitInfo struct {
	Hash      string `json:"hash"`
	Author    string `json:"author"`
	Date      string `json:"date"`
	Message   string `json:"message"`
	Refs      string `json:"refs"`
	GraphLine string `json:"graph_line"`
}
```

- [ ] **Step 2: Verify the file compiles**

Run: `go build ./internal/api/...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/api/types.go
git commit -m "feat: add CommitInfo type for git log history"
```

---

### Task 2: Implement `GitLog` backend method

**Files:**
- Modify: `internal/api/app.go:1562` (after `ListBranches` method)

- [ ] **Step 1: Add GitLog method after ListBranches**

Insert after line 1562 (`}` closing `ListBranches`):

```go
// GitLog returns recent git commits with graph topology for the given project.
// It executes git log --graph --all with structured output, parsing each line
// into graph prefix, hash, author, date, message, and ref decorations.
func (a *App) GitLog(projectPath string) ([]CommitInfo, error) {
	cmd := command("git", "log", "--graph", "--all", "--no-color",
		"--pretty=format:%x00%H%x00%h%x00%an%x00%ar%x00%s%x00%D",
		"-200")
	cmd.Dir = projectPath
	out, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[monika] GitLog git log failed: %v\n", err)
		return nil, err
	}

	var commits []CommitInfo
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Split graph prefix from structured data at the NUL byte.
		nulIdx := strings.IndexByte(line, 0x00)
		if nulIdx < 0 {
			continue
		}
		graphPrefix := line[:nulIdx]
		rest := line[nulIdx+1:]

		parts := strings.SplitN(rest, "\x00", 6)
		if len(parts) < 5 {
			continue
		}

		refs := ""
		if len(parts) >= 6 {
			refs = parts[5]
		}

		commits = append(commits, CommitInfo{
			GraphLine: graphPrefix,
			Hash:      parts[2], // short hash
			Author:    parts[3],
			Date:      parts[4],
			Message:   parts[5],
			Refs:      refs,
		})
	}
	return commits, nil
}
```

Note: The format string uses `%x00` (NUL byte) as delimiter. Fields are: full-hash, short-hash, author-name, relative-date, subject, ref-names. The graph prefix is before the first NUL byte.

- [ ] **Step 2: Verify the file compiles**

Run: `go build ./internal/api/...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/api/app.go
git commit -m "feat: add GitLog method to return structured commit history"
```

---

### Task 3: Regenerate Wails bindings

**Files:**
- Modify: `frontend/bindings/monika/...` (auto-generated)

- [ ] **Step 1: Run Wails binding generator**

Run: `wails3 generate bindings -f "..." -ts`

- [ ] **Step 2: Verify the CommitInfo type appears in generated bindings**

Run: `grep -r "CommitInfo" frontend/bindings/monika/`
Expected: should find the new type in the generated TS files

- [ ] **Step 3: Commit**

```bash
git add frontend/bindings/
git commit -m "chore: regenerate Wails bindings for CommitInfo"
```

---

### Task 4: Add `commitHistory` state and `loadCommitHistory` action to Zustand store

**Files:**
- Modify: `frontend/src/store/index.ts`
  - Line 6: add `CommitInfo` to import
  - Line 170: add `commitHistory` to `AppState` interface
  - Line 250: add `loadCommitHistory` method signature to `AppState` interface
  - Line 317: add `commitHistory` initial state
  - After line 1082: add `loadCommitHistory` and `commitHistory` action implementations

- [ ] **Step 1: Add CommitInfo to the import from bindings**

At `frontend/src/store/index.ts:6`, change:

```typescript
import type { RecentProject, BranchInfo, ModelInfo, ProviderInfo, ChangeStat, SessionInfo } from '../../bindings/monika'
```

to:

```typescript
import type { RecentProject, BranchInfo, ModelInfo, ProviderInfo, ChangeStat, SessionInfo, CommitInfo } from '../../bindings/monika'
```

- [ ] **Step 2: Add commitHistory to AppState interface**

After the `changeStats` property at line 170, insert:

```typescript
    commitHistory: { commits: CommitInfo[]; loading: boolean; error: string }
```

- [ ] **Step 3: Add loadCommitHistory method signature to AppState interface**

After the `setChangeStats` method signature around line 249, insert:

```typescript
    loadCommitHistory: () => void
```

- [ ] **Step 4: Add commitHistory initial state**

After `changeStats: { stats: [], loading: false, error: '' },` at line 317, insert:

```typescript
    commitHistory: { commits: [], loading: false, error: '' },
```

- [ ] **Step 5: Add loadCommitHistory action implementation**

After the `setChangeStats` action at line 1082, insert:

```typescript

    loadCommitHistory: async () => {
        const { projectPath } = get()
        if (!projectPath) return
        set({ commitHistory: { commits: [], loading: true, error: '' } })
        try {
            const result = await App.GitLog(projectPath)
            const commits = Array.isArray(result) ? result : []
            set({ commitHistory: { commits, loading: false, error: '' } })
        } catch {
            set({ commitHistory: { commits: [], loading: false, error: 'Failed to load history' } })
        }
    },
```

- [ ] **Step 6: Add commitHistory to resetProjectState**

In the `resetProjectState` method, after `changeStats: { stats: [], loading: false, error: '' },` add:

```typescript
            commitHistory: { commits: [], loading: false, error: '' },
```

- [ ] **Step 7: Verify TypeScript compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: no errors

- [ ] **Step 8: Commit**

```bash
git add frontend/src/store/index.ts
git commit -m "feat: add commitHistory state and loadCommitHistory action to store"
```

---

### Task 5: Refactor ChangesList into GitPanel with tab switching

**Files:**
- Modify: `frontend/src/components/ChangesList/ChangesList.tsx` (full rewrite)

This is the largest task. The component is restructured to:
1. Add a tab bar at the top: [ CHANGES | HISTORY ]
2. Render the existing changes list when CHANGES tab is active
3. Render a new commit history list when HISTORY tab is active

- [ ] **Step 1: Rewrite ChangesList.tsx as GitPanel**

Replace the entire file content with:

```tsx
import { useState, useEffect, useRef } from 'react'
import { createPortal } from 'react-dom'
import { IDockviewPanelProps } from 'dockview'
import { App as MonikaApp } from '../../../bindings/monika'
import type { ChangeStat } from '../../../bindings/monika'
import { useStore } from '../../store'
import { IconFile } from '../Icons'

function ChangesList(_props: IDockviewPanelProps) {
  const [activeTab, setActiveTab] = useState<'changes' | 'history'>('changes')

  return (
    <div
      className="flex flex-col h-full"
      style={{ background: 'var(--bg-sidebar)' }}
    >
      {/* Tab bar */}
      <div
        className="flex items-center gap-0 select-none shrink-0"
        style={{
          fontFamily: 'var(--font-sans)',
          background: 'var(--bg-sidebar)',
          borderBottom: '1px solid var(--border)',
        }}
      >
        <TabButton
          label="CHANGES"
          active={activeTab === 'changes'}
          onClick={() => setActiveTab('changes')}
        />
        <TabButton
          label="HISTORY"
          active={activeTab === 'history'}
          onClick={() => setActiveTab('history')}
        />
      </div>

      {/* Tab content */}
      {activeTab === 'changes' ? <ChangesTab /> : <HistoryTab />}
    </div>
  )
}

function TabButton({ label, active, onClick }: { label: string; active: boolean; onClick: () => void }) {
  return (
    <button
      className="text-[12px] px-3 py-[5px] cursor-pointer transition-colors"
      style={{
        fontFamily: 'var(--font-sans)',
        color: active ? 'var(--text-primary)' : 'var(--text-dim)',
        background: 'transparent',
        border: 'none',
        borderBottom: active ? '2px solid var(--accent)' : '2px solid transparent',
        fontWeight: active ? 600 : 400,
      }}
      onClick={onClick}
    >
      {label}
    </button>
  )
}

function ChangesTab() {
  const projectPath = useStore((s) => s.projectPath)
  const changes = useStore((s) => s.changeStats)
  const setPreviewDiff = useStore((s) => s.setPreviewDiff)
  const setPreviewFile = useStore((s) => s.setPreviewFile)
  const setRevealFilePath = useStore((s) => s.setRevealFilePath)
  const selectedPath = useStore((s) => s.preview.mode === 'diff' ? s.preview.filePath : null)
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number; path: string } | null>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const contextMenuJustOpened = useRef(false)

  useEffect(() => {
    if (!contextMenu) return
    contextMenuJustOpened.current = true
    const onClick = () => {
      if (contextMenuJustOpened.current) {
        contextMenuJustOpened.current = false
        return
      }
      setContextMenu(null)
    }
    window.addEventListener('click', onClick)
    return () => window.removeEventListener('click', onClick)
  }, [contextMenu])

  const handleClick = async (stat: ChangeStat) => {
    try {
      const result = await MonikaApp.GetFileDiff(projectPath, stat.path)
      const fileName = stat.path.split('/').pop() || stat.path
      if (result && result.lines) {
        setPreviewDiff(stat.path, fileName, result.lines)
      }
    } catch {
      // ignore
    }
  }

  const handleViewSource = async (path: string) => {
    try {
      const fileName = path.split('/').pop() || path
      const result = await MonikaApp.ReadFile(projectPath, path)
      setPreviewFile(path, fileName, result?.content || '')
      setRevealFilePath(path)
    } catch {
      // ignore
    }
  }

  const handleContextMenu = (e: React.MouseEvent, path: string) => {
    e.preventDefault()
    e.stopPropagation()
    setContextMenu({ x: e.clientX, y: e.clientY, path })
  }

  const renderContextMenu = () => {
    if (!contextMenu) return null
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
        <div
          className="flex items-center gap-2.5 px-3 py-[5px] cursor-pointer transition-colors rounded-sm mx-1"
          style={{ color: 'var(--text-secondary)' }}
          onMouseEnter={(e) => { e.currentTarget.style.background = 'var(--bg-hover)'; e.currentTarget.style.color = 'var(--text-primary)' }}
          onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent'; e.currentTarget.style.color = 'var(--text-secondary)' }}
          onClick={() => { setContextMenu(null); handleViewSource(contextMenu.path) }}
        >
          <span className="flex-shrink-0 flex items-center" style={{ opacity: 0.7, width: 14 }}><IconFile size={14} /></span>
          <span>View Source File</span>
        </div>
      </div>,
      document.body
    )
  }

  const basenameFn = (p: string) => p.split('/').pop() || p

  return (
    <>
      <div className="flex-1 overflow-y-auto" style={{ padding: '0 8px' }}>
        {changes.loading && changes.stats.length === 0 ? (
          <div className="py-4 text-[12px] text-[var(--text-dim)] px-1">Loading...</div>
        ) : changes.error && changes.stats.length === 0 ? (
          <div className="py-4 text-[12px] text-[var(--red)] px-1">{changes.error}</div>
        ) : changes.stats.length === 0 ? (
          <div className="py-4 text-[12px] text-[var(--text-dim)] px-1">No changes</div>
        ) : (
          changes.stats.map((stat) => {
            const active = selectedPath === stat.path
            return (
              <div
                key={stat.path}
                className="flex items-center gap-1 cursor-pointer text-[13px] leading-[26px] rounded-md transition-colors duration-100 mx-1 px-[6px]"
                style={{
                  color: active ? 'var(--text-primary)' : 'var(--text-secondary)',
                  background: active ? 'var(--bg-active)' : 'transparent',
                }}
                onMouseEnter={(e) => {
                  if (!active) e.currentTarget.style.background = 'rgba(255,255,255,0.04)'
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = active ? 'var(--bg-active)' : 'transparent'
                }}
                onClick={() => handleClick(stat)}
                onContextMenu={(e) => handleContextMenu(e, stat.path)}
                title={stat.path}
              >
                <span className="truncate flex-1">{basenameFn(stat.path)}</span>
                {stat.added > 0 && (
                  <span className="text-[11px] flex-shrink-0" style={{ color: 'var(--green)' }}>
                    +{stat.added}
                  </span>
                )}
                {stat.deleted > 0 && (
                  <span className="text-[11px] flex-shrink-0 ml-0.5" style={{ color: 'var(--red)' }}>
                    -{stat.deleted}
                  </span>
                )}
              </div>
            )
          })
        )}
      </div>
      {renderContextMenu()}
    </>
  )
}

function HistoryTab() {
  const commitHistory = useStore((s) => s.commitHistory)
  const loadCommitHistory = useStore((s) => s.loadCommitHistory)

  useEffect(() => {
    loadCommitHistory()
  }, [loadCommitHistory])

  if (commitHistory.loading) {
    return (
      <div className="flex-1 overflow-y-auto" style={{ padding: '8px' }}>
        <div className="py-4 text-[12px] text-[var(--text-dim)] px-1">Loading...</div>
      </div>
    )
  }

  if (commitHistory.error) {
    return (
      <div className="flex-1 overflow-y-auto" style={{ padding: '8px' }}>
        <div className="py-4 text-[12px] text-[var(--red)] px-1">{commitHistory.error}</div>
      </div>
    )
  }

  if (commitHistory.commits.length === 0) {
    return (
      <div className="flex-1 overflow-y-auto" style={{ padding: '8px' }}>
        <div className="py-4 text-[12px] text-[var(--text-dim)] px-1">No commits</div>
      </div>
    )
  }

  return (
    <div className="flex-1 overflow-y-auto" style={{ padding: '0 4px' }}>
      {commitHistory.commits.map((commit, idx) => (
        <CommitRow key={commit.Hash + '-' + idx} commit={commit} />
      ))}
    </div>
  )
}

function CommitRow({ commit }: { commit: { hash: string; author: string; date: string; message: string; refs: string; graph_line: string } }) {
  return (
    <div
      className="flex items-start gap-1 text-[12px] leading-[22px] rounded-sm transition-colors duration-100 mx-1 px-[4px] cursor-default"
      style={{
        fontFamily: 'var(--font-sans)',
        color: 'var(--text-secondary)',
      }}
      onMouseEnter={(e) => { e.currentTarget.style.background = 'rgba(255,255,255,0.04)' }}
      onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent' }}
    >
      {/* Graph column */}
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

      {/* Hash */}
      <span
        className="flex-shrink-0"
        style={{ color: 'var(--accent)', fontFamily: 'var(--font-mono)', fontSize: '11px' }}
      >
        {commit.hash}
      </span>

      {/* Refs */}
      {commit.refs && <RefTags refs={commit.refs} />}

      {/* Message + author + date */}
      <span className="truncate min-w-0" style={{ color: 'var(--text-primary)' }}>
        {commit.message}
      </span>
      <span className="flex-shrink-0" style={{ color: 'var(--text-dim)', fontSize: '11px' }}>
        {commit.author} · {commit.date}
      </span>
    </div>
  )
}

function RefTags({ refs }: { refs: string }) {
  const parts = refs.split(',').map((s) => s.trim()).filter(Boolean)
  return (
    <span className="flex items-center gap-1 flex-shrink-0">
      {parts.map((ref, i) => {
        let color = 'var(--text-dim)'
        let bg = 'var(--bg-sidebar)'
        if (ref.startsWith('tag:')) {
          color = '#f0c040'
          bg = 'rgba(240,192,64,0.1)'
        } else if (ref.startsWith('HEAD')) {
          color = 'var(--accent)'
          bg = 'rgba(100,150,255,0.1)'
        } else if (ref.startsWith('origin/')) {
          color = 'var(--green)'
          bg = 'rgba(80,200,120,0.1)'
        }
        return (
          <span
            key={i}
            style={{
              color,
              background: bg,
              borderRadius: '3px',
              padding: '0 4px',
              fontSize: '10px',
              lineHeight: '18px',
              whiteSpace: 'nowrap',
            }}
          >
            {ref}
          </span>
        )
      })}
    </span>
  )
}

export default ChangesList
```

- [ ] **Step 2: Verify TypeScript compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: no errors. If `CommitInfo` type is not found in bindings, check Task 3 was completed.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/ChangesList/ChangesList.tsx
git commit -m "feat: refactor ChangesList into GitPanel with HISTORY tab"
```

---

### Task 6: Full build verification

- [ ] **Step 1: Build the Go backend**

Run: `go build .`
Expected: compiles without errors

- [ ] **Step 2: Build the frontend**

Run: `cd frontend && npm run build`
Expected: builds without errors

- [ ] **Step 3: Run Go tests**

Run: `go test ./...`
Expected: all tests pass

- [ ] **Step 4: Commit any fixups if needed**
