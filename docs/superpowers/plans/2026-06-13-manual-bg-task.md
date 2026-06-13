# Manual Background Task Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow users to manually type and run shell commands as background tasks from the TASKS panel.

**Architecture:** Add a `StartBgTask` Go API method on `App`, a `startBgTask` store action, and an inline input box in `TasksPanel.tsx`. All existing `BackgroundTaskManager` infrastructure is reused without changes.

**Tech Stack:** Go (Wails v3 API), React + Zustand + Tailwind CSS v4

---

### Task 1: Add StartBgTask Go API method

**Files:**
- Modify: `internal/api/app.go`

- [ ] **Step 1: Add StartBgTask method**

Insert after `StopBgTask` (after line 305). The method delegates to the existing `BackgroundTaskManager.Start` with `projectPath()` as workdir.

```go
func (a *App) StartBgTask(command string) (string, error) {
	return a.bgTaskMgr.Start(command, a.projectPath())
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/api/`

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/api/app.go
git commit -m "feat: add StartBgTask API method for manual background task creation"
```

---

### Task 2: Add startBgTask store action

**Files:**
- Modify: `frontend/src/store/index.ts`

- [ ] **Step 1: Add startBgTask to the store type interface**

Insert after `stopBgTask` declaration in the store type (after line 339):

```ts
startBgTask: (command: string) => Promise<void>
```

- [ ] **Step 2: Add startBgTask implementation**

Insert after `stopBgTask` implementation (after line 680):

```ts
startBgTask: async (command: string) => {
    await Call.ByName('monika/internal/api.App.StartBgTask', command)
},
```

- [ ] **Step 3: Verify TypeScript compilation**

Run: `cd frontend && npx tsc --noEmit`

Expected: no errors related to store.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/store/index.ts
git commit -m "feat: add startBgTask store action"
```

---

### Task 3: Add inline input box to TasksPanel

**Files:**
- Modify: `frontend/src/components/Tasks/TasksPanel.tsx`

- [ ] **Step 1: Import startBgTask from store**

Change the destructured store hooks at the top of the component:

```tsx
export default function TasksPanel({ }: IDockviewPanelProps) {
    const bgTasks = useStore((s) => s.bgTasks)
    const selectedBgTaskId = useStore((s) => s.selectedBgTaskId)
    const selectBgTask = useStore((s) => s.selectBgTask)
    const startBgTask = useStore((s) => s.startBgTask)
```

- [ ] **Step 2: Add input element after the TASKS header**

Insert after the "TASKS" header div (after line 13) and before the task list div (before line 14):

```tsx
            <div className="px-2 py-1.5 border-b border-[var(--border)]">
                <input
                    className="w-full bg-[var(--bg-input)] text-[var(--text)] text-xs px-2 py-1 rounded outline-none border border-[var(--border)] focus:border-[var(--accent)] placeholder:text-[var(--text-muted)]"
                    placeholder="Run command..."
                    onKeyDown={(e) => {
                        if (e.key === 'Enter' && e.currentTarget.value.trim()) {
                            startBgTask(e.currentTarget.value.trim())
                            e.currentTarget.value = ''
                        }
                    }}
                />
            </div>
```

- [ ] **Step 3: Verify TypeScript compilation**

Run: `cd frontend && npx tsc --noEmit`

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/Tasks/TasksPanel.tsx
git commit -m "feat: add inline command input to TASKS panel"
```
