# Background Task Log Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace in-memory ring buffer log storage with file-based storage and lazy-loading to eliminate UI lag when viewing high-frequency background task logs.

**Architecture:** Backend writes log lines to `<projectDir>/.monika/temp/logs/<taskId>.log` files and emits lightweight `log_update` events (task_id + line_count only). Frontend reads log content on demand via a new `BgTaskLogLines` API with offset/limit pagination, using the same displayCount + IntersectionObserver lazy-rendering pattern as chat messages.

**Tech Stack:** Go (backend), React + TypeScript + Zustand (frontend), Wails v3 (IPC), bufio.Scanner (file reading)

**Spec:** `docs/superpowers/specs/2025-07-11-background-task-log-optimization-design.md`

---

## File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `internal/api/background.go` | Modify | Replace ringBuffer with file I/O; add `LogLines()`; modify events; add `CleanOldLogs()` |
| `internal/api/background_test.go` | Modify | Update tests for new constructor signature; remove ringBuffer tests; add LogLines tests |
| `internal/api/app.go` | Modify | Add `BgTaskLogLines` API; pass logDir to manager; call `CleanOldLogs()` at startup |
| `frontend/src/store/index.ts` | Modify | Replace `bgTaskLogs` with line-count + cache state; new actions; event handler change |
| `frontend/src/components/Preview/PreviewPanel.tsx` | Modify | Replace full-render with lazy-load pattern; add IntersectionObserver + debounce |

---

## Task 1: Backend — Replace ringBuffer with file-based logging

**Files:**
- Modify: `internal/api/background.go`
- Modify: `internal/api/background_test.go`

- [ ] **Step 1: Update background_test.go — change constructor calls and remove ringBuffer test**

Replace the entire `background_test.go` content:

```go
package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBackgroundTaskManagerStartStop(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewBackgroundTaskManager(filepath.Join(tmpDir, "logs"))
	defer mgr.Cleanup()

	id, err := mgr.Start("echo hello", ".")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty task ID")
	}

	tasks := mgr.List()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Status != BgTaskRunning {
		t.Fatalf("expected status running, got %s", tasks[0].Status)
	}

	time.Sleep(2 * time.Second)

	tasks = mgr.List()
	if tasks[0].Status != BgTaskExited {
		t.Fatalf("expected status exited, got %s", tasks[0].Status)
	}

	logs, err := mgr.Logs(id, 10)
	if err != nil {
		t.Fatalf("Logs failed: %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("expected at least 1 log line")
	}
	if !strings.Contains(logs[0], "hello") {
		t.Fatalf("expected log to contain 'hello', got %s", logs[0])
	}
}

func TestBackgroundTaskManagerStop(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewBackgroundTaskManager(filepath.Join(tmpDir, "logs"))
	defer mgr.Cleanup()

	id, err := mgr.Start("ping -n 30 127.0.0.1", ".")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	tasks := mgr.List()
	if tasks[0].Status != BgTaskRunning {
		t.Fatalf("expected running, got %s", tasks[0].Status)
	}

	err = mgr.Stop(id)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	tasks = mgr.List()
	if tasks[0].Status != BgTaskStopped {
		t.Fatalf("expected stopped, got %s", tasks[0].Status)
	}

	err = mgr.Stop("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent task")
	}
}

func TestBackgroundTaskManagerLogsNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewBackgroundTaskManager(filepath.Join(tmpDir, "logs"))
	defer mgr.Cleanup()

	_, err := mgr.Logs("nonexistent", 10)
	if err == nil {
		t.Fatal("expected error for non-existent task")
	}
}

func TestBackgroundTaskManagerLogLines(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewBackgroundTaskManager(filepath.Join(tmpDir, "logs"))
	defer mgr.Cleanup()

	id, err := mgr.Start("echo line1 && echo line2 && echo line3", ".")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(2 * time.Second)

	// Read all lines from start
	lines, err := mgr.LogLines(id, 0, 100)
	if err != nil {
		t.Fatalf("LogLines failed: %v", err)
	}
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "line1") {
		t.Fatalf("expected first line to contain 'line1', got %s", lines[0])
	}

	// Read last 2 lines
	lines, err = mgr.LogLines(id, -2, 100)
	if err != nil {
		t.Fatalf("LogLines tail failed: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines from tail, got %d", len(lines))
	}

	// Read with limit
	lines, err = mgr.LogLines(id, 0, 1)
	if err != nil {
		t.Fatalf("LogLines with limit failed: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line with limit=1, got %d", len(lines))
	}
}

func TestBackgroundTaskManagerSubscribe(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewBackgroundTaskManager(filepath.Join(tmpDir, "logs"))
	defer mgr.Cleanup()

	ch := mgr.Subscribe()

	id, err := mgr.Start("echo test-subscribe", ".")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	select {
	case ev := <-ch:
		if ev.Type != BgEventStarted {
			t.Fatalf("expected BgEventStarted, got %s", ev.Type)
		}
		if ev.TaskID != id {
			t.Fatalf("expected task ID %s, got %s", id, ev.TaskID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for started event")
	}

	timeout := time.After(5 * time.Second)
	for {
		select {
		case ev := <-ch:
			if ev.Type == BgEventExited {
				if ev.TaskID != id {
					t.Fatalf("expected task ID %s, got %s", id, ev.TaskID)
				}
				return
			}
			if ev.Type == BgEventLogUpdate {
				if ev.LineCount <= 0 {
					t.Fatal("expected positive line_count in log_update event")
				}
			}
		case <-timeout:
			t.Fatal("timed out waiting for exited event")
		}
	}
}

func TestCleanOldLogs(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "logs")
	os.MkdirAll(logDir, 0755)

	// Create a fake old log file
	oldFile := filepath.Join(logDir, "old.log")
	os.WriteFile(oldFile, []byte("old log"), 0644)

	// Set modtime to 8 days ago
	oldTime := time.Now().Add(-8 * 24 * time.Hour)
	os.Chtimes(oldFile, oldTime, oldTime)

	// Create a recent file
	newFile := filepath.Join(logDir, "new.log")
	os.WriteFile(newFile, []byte("new log"), 0644)

	mgr := NewBackgroundTaskManager(logDir)
	mgr.CleanOldLogs()

	// Old file should be deleted
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Fatal("expected old log file to be deleted")
	}
	// New file should remain
	if _, err := os.Stat(newFile); err != nil {
		t.Fatal("expected new log file to remain")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd d:/git/monika && go test ./internal/api/ -run TestBackgroundTask -v -count=1`
Expected: FAIL — `NewBackgroundTaskManager` takes 0 args but tests pass 1; `BgEventLogUpdate` undefined; `LogLines` undefined; `CleanOldLogs` undefined

- [ ] **Step 3: Rewrite background.go with file-based logging**

Replace the entire content of `internal/api/background.go`:

```go
package api

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"

	"monika/internal/tool/builtin"
)

type BgTaskStatus = builtin.BgTaskStatus

const (
	BgTaskRunning = builtin.BgTaskRunning
	BgTaskStopped = builtin.BgTaskStopped
	BgTaskExited  = builtin.BgTaskExited
)

type BgTaskEventType string

const (
	BgEventStarted  BgTaskEventType = "started"
	BgEventLogUpdate BgTaskEventType = "log_update"
	BgEventStopped  BgTaskEventType = "stopped"
	BgEventExited   BgTaskEventType = "exited"
)

type BgTaskEvent struct {
	Type      BgTaskEventType `json:"type"`
	TaskID    string          `json:"task_id"`
	Command   string          `json:"command,omitempty"`
	WorkDir   string          `json:"work_dir,omitempty"`
	PID       int             `json:"pid,omitempty"`
	Status    BgTaskStatus    `json:"status,omitempty"`
	ExitCode  int             `json:"exit_code,omitempty"`
	LineCount int             `json:"line_count,omitempty"`
}

type BgTaskInfo = builtin.BgTaskInfo

type bgTask struct {
	mu        sync.Mutex
	info      BgTaskInfo
	logFile   *os.File
	logPath   string
	lineCount int
	cancel    context.CancelFunc
}

const (
	bgSubscriberBufferSize = 256
	bgLogRetention         = 7 * 24 * time.Hour
)

type BackgroundTaskManager struct {
	mu          sync.Mutex
	tasks       map[string]*bgTask
	subscribers map[chan BgTaskEvent]struct{}
	engine      *builtin.ShellEngine
	logDir      string
}

func NewBackgroundTaskManager(logDir string) *BackgroundTaskManager {
	return &BackgroundTaskManager{
		tasks:       make(map[string]*bgTask),
		subscribers: make(map[chan BgTaskEvent]struct{}),
		engine:      builtin.NewShellEngine(),
		logDir:      logDir,
	}
}

func (m *BackgroundTaskManager) Start(command, workdir string) (string, error) {
	id := uuid.New().String()

	if err := os.MkdirAll(m.logDir, 0755); err != nil {
		return "", fmt.Errorf("create log dir: %w", err)
	}

	logPath := filepath.Join(m.logDir, id+".log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return "", fmt.Errorf("create log file: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	task := &bgTask{
		info: BgTaskInfo{
			ID:        id,
			Command:   command,
			WorkDir:   workdir,
			Status:    BgTaskRunning,
			StartedAt: time.Now(),
		},
		logFile: logFile,
		logPath: logPath,
	}

	onLine := func(line string) {
		line = stripANSI(line)
		fmt.Fprintln(logFile, line)
		task.mu.Lock()
		task.lineCount++
		count := task.lineCount
		task.mu.Unlock()
		m.emit(BgTaskEvent{
			Type:      BgEventLogUpdate,
			TaskID:    id,
			LineCount: count,
		})
	}

	bgCancel, exitCh, err := m.engine.StartBackground(ctx, command, workdir, os.Environ(), onLine)
	if err != nil {
		cancel()
		logFile.Close()
		return "", fmt.Errorf("start process: %w", err)
	}

	task.cancel = func() {
		bgCancel()
		cancel()
	}

	m.mu.Lock()
	m.tasks[id] = task
	m.mu.Unlock()

	m.emit(BgTaskEvent{
		Type:    BgEventStarted,
		TaskID:  id,
		Command: command,
		WorkDir: workdir,
		Status:  BgTaskRunning,
	})

	go m.waitExit(id, exitCh)

	return id, nil
}

func (m *BackgroundTaskManager) Stop(taskID string) error {
	m.mu.Lock()
	task, ok := m.tasks[taskID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("task %s not found", taskID)
	}
	if task.info.Status != BgTaskRunning {
		m.mu.Unlock()
		return fmt.Errorf("task %s is not running", taskID)
	}
	task.info.Status = BgTaskStopped
	m.mu.Unlock()

	task.cancel()

	m.emit(BgTaskEvent{
		Type:    BgEventStopped,
		TaskID:  taskID,
		Command: task.info.Command,
		WorkDir: task.info.WorkDir,
		PID:     task.info.PID,
		Status:  BgTaskStopped,
	})

	return nil
}

func (m *BackgroundTaskManager) Logs(taskID string, lines int) ([]string, error) {
	return m.LogLines(taskID, -lines, lines)
}

func (m *BackgroundTaskManager) LogLines(taskID string, offset, limit int) ([]string, error) {
	m.mu.Lock()
	task, ok := m.tasks[taskID]
	m.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	f, err := os.Open(task.logPath)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()

	task.mu.Lock()
	totalLines := task.lineCount
	task.mu.Unlock()

	var skipCount int
	if offset < 0 {
		skipCount = totalLines + offset
		if skipCount < 0 {
			skipCount = 0
		}
	} else {
		skipCount = offset
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	result := make([]string, 0, limit)
	lineIdx := 0
	count := 0

	for scanner.Scan() {
		if lineIdx < skipCount {
			lineIdx++
			continue
		}
		if count >= limit {
			break
		}
		result = append(result, scanner.Text())
		count++
		lineIdx++
	}

	return result, nil
}

func (m *BackgroundTaskManager) List() []BgTaskInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]BgTaskInfo, 0, len(m.tasks))
	for _, task := range m.tasks {
		result = append(result, task.info)
	}
	return result
}

func (m *BackgroundTaskManager) Subscribe() <-chan BgTaskEvent {
	ch := make(chan BgTaskEvent, bgSubscriberBufferSize)
	m.mu.Lock()
	m.subscribers[ch] = struct{}{}
	m.mu.Unlock()
	return ch
}

func (m *BackgroundTaskManager) Cleanup() {
	m.mu.Lock()
	ids := make([]string, 0)
	for id, task := range m.tasks {
		if task.info.Status == BgTaskRunning {
			ids = append(ids, id)
		}
	}
	m.mu.Unlock()

	for _, id := range ids {
		m.Stop(id)
	}

	m.mu.Lock()
	for _, task := range m.tasks {
		if task.logFile != nil {
			task.logFile.Close()
		}
	}
	for ch := range m.subscribers {
		close(ch)
		delete(m.subscribers, ch)
	}
	m.mu.Unlock()
}

func (m *BackgroundTaskManager) CleanOldLogs() {
	entries, err := os.ReadDir(m.logDir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-bgLogRetention)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".log" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(m.logDir, entry.Name()))
		}
	}
}

func (m *BackgroundTaskManager) waitExit(taskID string, exitCh <-chan int) {
	code := <-exitCh

	m.mu.Lock()
	task, ok := m.tasks[taskID]
	if !ok {
		m.mu.Unlock()
		return
	}
	task.info.ExitCode = code
	if task.info.Status == BgTaskRunning {
		task.info.Status = BgTaskExited
	}
	m.mu.Unlock()

	m.emit(BgTaskEvent{
		Type:     BgEventExited,
		TaskID:   taskID,
		PID:      task.info.PID,
		Status:   task.info.Status,
		ExitCode: task.info.ExitCode,
	})
}

func (m *BackgroundTaskManager) emit(ev BgTaskEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for ch := range m.subscribers {
		select {
		case ch <- ev:
		default:
		}
	}
}
```

Also add the `SetLogDir` method (used by `app.go` when project opens):

```go
func (m *BackgroundTaskManager) SetLogDir(dir string) {
	m.mu.Lock()
	m.logDir = dir
	m.mu.Unlock()
	os.MkdirAll(dir, 0755)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd d:/git/monika && go test ./internal/api/ -run "TestBackgroundTask|TestCleanOldLogs" -v -count=1`
Expected: PASS for all tests

- [ ] **Step 5: Commit**

```bash
cd d:/git/monika && git add internal/api/background.go internal/api/background_test.go && git commit -m "refactor: replace ringBuffer with file-based background task log storage"
```

---

## Task 2: Backend — Adapt app.go for new manager + new API

**Files:**
- Modify: `internal/api/app.go`

- [ ] **Step 1: Update NewBackgroundTaskManager call in app.go**

In `app.go`, find line ~127 (the `NewApp` function). Change the constructor to pass empty string (logDir will be set when project opens via `SetLogDir`):

```go
// Before (line 127):
bgTaskMgr:         NewBackgroundTaskManager(),

// After:
bgTaskMgr:         NewBackgroundTaskManager(""),
```

- [ ] **Step 2: Set logDir when project opens**

In `app.go`, find the `OpenProject` method. After the project is registered (search for where `a.projects[path]` is set or the project path is stored), add:

```go
a.bgTaskMgr.SetLogDir(filepath.Join(path, ".monika", "temp", "logs"))
```

Use grep to find the exact location: search for `a.projects[` in `app.go` to find where the project path variable is available.

- [ ] **Step 3: Add BgTaskLogLines API method**

Add after the existing `GetBgTaskLogs` method (around line 329-331 in `app.go`):

```go
func (a *App) BgTaskLogLines(taskID string, offset, limit int) ([]string, error) {
	return a.bgTaskMgr.LogLines(taskID, offset, limit)
}
```

- [ ] **Step 4: Call CleanOldLogs at startup**

In `app.go` `ServiceStartup` method (around line 246), after the bg task event goroutine (around line 280), add:

```go
go a.bgTaskMgr.CleanOldLogs()
```

- [ ] **Step 5: Verify backend compiles**

Run: `cd d:/git/monika && go build ./internal/api/`
Expected: No errors

- [ ] **Step 6: Commit**

```bash
cd d:/git/monika && git add internal/api/app.go internal/api/background.go && git commit -m "feat: add BgTaskLogLines API and file-based log directory management"
```

---

## Task 3: Regenerate Wails bindings

**Files:**
- Regenerate: `frontend/bindings/monika/`

- [ ] **Step 1: Generate bindings**

Run:
```bash
cd d:/git/monika && wails3 generate bindings -ts
```

- [ ] **Step 2: Copy barrel index**

Run:
```bash
cd d:/git/monika && node -e "require('fs').copyFileSync('build/barrel_index.ts','frontend/bindings/monika/index.ts')"
```

- [ ] **Step 3: Verify BgTaskLogLines appears in bindings**

Run: `grep -r "BgTaskLogLines" frontend/bindings/`
Expected: Should appear in the generated bindings

- [ ] **Step 4: Commit**

```bash
cd d:/git/monika && git add frontend/bindings/ && git commit -m "chore: regenerate wails bindings for BgTaskLogLines"
```

Note: `frontend/bindings/` is gitignored, so this may be a no-op for git. That's expected.

---

## Task 4: Frontend — Store changes

**Files:**
- Modify: `frontend/src/store/index.ts`

- [ ] **Step 1: Replace bgTaskLogs state with new state fields**

In `frontend/src/store/index.ts`, find the state interface (around line 245-247):

```ts
// Before:
bgTasks: BgTaskInfo[]
selectedBgTaskId: string | null
bgTaskLogs: Record<string, string[]>
```

Replace with:

```ts
bgTasks: BgTaskInfo[]
selectedBgTaskId: string | null
bgTaskLineCounts: Record<string, number>
bgTaskLogCache: Record<string, { offset: number; lines: string[] }>
bgTaskDisplayCount: Record<string, number>
```

- [ ] **Step 2: Replace appendBgTaskLog with new actions in the interface**

Find `appendBgTaskLog` in the interface (around line 380):

```ts
// Before:
appendBgTaskLog: (taskId: string, line: string) => void
```

Replace with:

```ts
updateBgTaskLineCount: (taskId: string, count: number) => void
setBgTaskLogCache: (taskId: string, offset: number, lines: string[]) => void
loadBgTaskLogs: (taskId: string, offset: number, limit: number) => Promise<void>
loadMoreBgTaskLines: (taskId: string) => void
```

- [ ] **Step 3: Update initial state values**

Find the initial state (around line 477-479):

```ts
// Before:
bgTasks: [] as BgTaskInfo[],
bgTaskLogs: {} as Record<string, string[]>,
```

Replace with:

```ts
bgTasks: [] as BgTaskInfo[],
bgTaskLineCounts: {} as Record<string, number>,
bgTaskLogCache: {} as Record<string, { offset: number; lines: string[] }>,
bgTaskDisplayCount: {} as Record<string, number>,
```

- [ ] **Step 4: Add constants**

Near the existing `INITIAL_DISPLAY_COUNT` constants (around line 403-404), add:

```ts
const BG_LOG_INITIAL_DISPLAY = 50
const BG_LOG_LOAD_MORE = 50
const BG_LOG_FETCH_BATCH = 200
```

- [ ] **Step 5: Replace appendBgTaskLog implementation with new actions**

Find the `appendBgTaskLog` implementation (around line 785-787):

```ts
// Before:
appendBgTaskLog: (taskId, line) => set((state) => ({
    bgTaskLogs: { ...state.bgTaskLogs, [taskId]: [...(state.bgTaskLogs[taskId] || []), line].slice(-500) },
})),
```

Replace with:

```ts
updateBgTaskLineCount: (taskId, count) => set((state) => ({
    bgTaskLineCounts: { ...state.bgTaskLineCounts, [taskId]: count },
})),

setBgTaskLogCache: (taskId, offset, lines) => set((state) => ({
    bgTaskLogCache: { ...state.bgTaskLogCache, [taskId]: { offset, lines } },
})),

loadBgTaskLogs: async (taskId, offset, limit) => {
    try {
        const lines = await Call.ByName('monika/internal/api.App.BgTaskLogLines', taskId, offset, limit)
        const store = useStore.getState()
        const existing = store.bgTaskLogCache[taskId]
        let mergedLines: string[]
        let mergedOffset: number

        if (existing && offset < existing.offset) {
            // Prepending older lines
            mergedLines = [...(lines as string[]), ...existing.lines]
            mergedOffset = offset
        } else if (existing && offset >= existing.offset + existing.lines.length) {
            // Replacing with newer range (e.g., tail refresh)
            mergedLines = lines as string[]
            mergedOffset = offset
        } else {
            // Initial load or overlapping
            mergedLines = lines as string[]
            mergedOffset = offset
        }

        store.setBgTaskLogCache(taskId, mergedOffset, mergedLines)
    } catch (e) {
        console.error('[monika] failed to load bg task logs:', e)
    }
},

loadMoreBgTaskLines: (taskId) => {
    const store = useStore.getState()
    const current = store.bgTaskDisplayCount[taskId] || BG_LOG_INITIAL_DISPLAY
    const next = current + BG_LOG_LOAD_MORE
    const cache = store.bgTaskLogCache[taskId]

    if (cache && next > cache.lines.length) {
        // Need to fetch older lines
        const newOffset = cache.offset - BG_LOG_FETCH_BATCH
        store.loadBgTaskLogs(taskId, newOffset, BG_LOG_FETCH_BATCH)
    }

    set((state) => ({
        bgTaskDisplayCount: { ...state.bgTaskDisplayCount, [taskId]: next },
    }))
},
```

- [ ] **Step 6: Update event handler for log_update events**

Find the event handler (around line 2375-2376):

```ts
// Before:
case 'log':
    store.appendBgTaskLog(ev.task_id, ev.log_line)
    break
```

Replace with:

```ts
case 'log_update':
    store.updateBgTaskLineCount(ev.task_id, ev.line_count)
    break
```

- [ ] **Step 7: Update selectBgTask to reset cache**

Find `selectBgTask` (around line 775):

```ts
// Before:
selectBgTask: (id) => set({ selectedBgTaskId: id, preview: { mode: 'task', filePath: null, fileName: null, fileContent: null, diffLines: null, conflictAiContent: null, conflictActive: false, commitDetail: null, commitFiles: null, commitHash: null } }),
```

Replace with:

```ts
selectBgTask: (id) => set((state) => ({
    selectedBgTaskId: id,
    bgTaskDisplayCount: id ? { ...state.bgTaskDisplayCount, [id]: BG_LOG_INITIAL_DISPLAY } : state.bgTaskDisplayCount,
    preview: { mode: 'task', filePath: null, fileName: null, fileContent: null, diffLines: null, conflictAiContent: null, conflictActive: false, commitDetail: null, commitFiles: null, commitHash: null },
})),
```

- [ ] **Step 8: Verify frontend compiles**

Run: `cd d:/git/monika/frontend && npx tsc --noEmit 2>&1 | head -30`
Expected: May have errors in PreviewPanel.tsx (will fix in Task 5), but store/index.ts should be clean

- [ ] **Step 9: Commit**

```bash
cd d:/git/monika && git add frontend/src/store/index.ts && git commit -m "feat: replace bgTaskLogs with file-backed lazy-loading store state"
```

---

## Task 5: Frontend — PreviewPanel lazy-load rendering

**Files:**
- Modify: `frontend/src/components/Preview/PreviewPanel.tsx`

- [ ] **Step 1: Replace store selectors**

Find the selectors around lines 617-618:

```tsx
// Before:
const bgTasks = useStore((s) => s.bgTasks)
const bgTaskLogs = useStore((s) => s.bgTaskLogs)
```

Replace with:

```tsx
const bgTasks = useStore((s) => s.bgTasks)
const bgTaskLineCounts = useStore((s) => s.bgTaskLineCounts)
const bgTaskLogCache = useStore((s) => s.bgTaskLogCache)
const bgTaskDisplayCount = useStore((s) => s.bgTaskDisplayCount)
```

- [ ] **Step 2: Add sentinel ref and replace derived values**

Find around lines 737-738 and 629:

```tsx
// Before (line 629):
const bgLogRef = useRef<HTMLDivElement>(null)

// Before (line 737-738):
const bgTask = selectedBgTaskId ? bgTasks.find(t => t.id === selectedBgTaskId) : null
const bgLogs = selectedBgTaskId ? (bgTaskLogs[selectedBgTaskId] || []) : []
```

Add a sentinel ref after `bgLogRef`:

```tsx
const bgLogRef = useRef<HTMLDivElement>(null)
const bgSentinelRef = useRef<HTMLDivElement>(null)
const bgPrevScrollHeightRef = useRef(0)
const bgStickToBottomRef = useRef(true)
```

Replace derived values:

```tsx
const bgTask = selectedBgTaskId ? bgTasks.find(t => t.id === selectedBgTaskId) : null
const bgLineCount = selectedBgTaskId ? (bgTaskLineCounts[selectedBgTaskId] || 0) : 0
const bgCache = selectedBgTaskId ? bgTaskLogCache[selectedBgTaskId] : undefined
const bgDisplayCount = selectedBgTaskId ? (bgTaskDisplayCount[selectedBgTaskId] || 50) : 50

const bgLogs = useMemo(() => {
    if (!bgCache) return []
    return bgCache.lines.slice(Math.max(0, bgCache.lines.length - bgDisplayCount))
}, [bgCache, bgDisplayCount])

const bgHasMore = bgCache ? (bgCache.offset > 0 || bgDisplayCount < bgCache.lines.length) : false
```

- [ ] **Step 3: Replace auto-scroll effect with lazy-load + debounce logic**

Find the auto-scroll effect (around lines 891-898):

```tsx
// Before:
// Auto-scroll background task logs to bottom when new output arrives
const bgLogsLen = bgLogs.length
const bgTaskStatus = bgTask?.status
useEffect(() => {
    const el = bgLogRef.current
    if (!el || !showTask) return
    el.scrollTop = el.scrollHeight
}, [bgLogsLen, bgTaskStatus, showTask, selectedBgTaskId])
```

Replace with:

```tsx
// Initial load and periodic refresh when task is selected
const bgTaskStatus = bgTask?.status
const showTask2 = showTask
useEffect(() => {
    if (!selectedBgTaskId || !showTask2) return
    const store = useStore.getState()
    store.loadBgTaskLogs(selectedBgTaskId, -200, 200)
    const timer = setTimeout(() => {
        const el = bgLogRef.current
        if (el) el.scrollTop = el.scrollHeight
    }, 100)
    return () => clearTimeout(timer)
}, [selectedBgTaskId, showTask2])

// Debounced refresh when line count changes
const bgLineCountRef = useRef(bgLineCount)
const bgDebounceRef = useRef<ReturnType<typeof setTimeout>>()
useEffect(() => {
    if (!showTask || !selectedBgTaskId) return
    if (bgLineCount === bgLineCountRef.current) return
    bgLineCountRef.current = bgLineCount

    clearTimeout(bgDebounceRef.current)
    bgDebounceRef.current = setTimeout(() => {
        const store = useStore.getState()
        store.loadBgTaskLogs(selectedBgTaskId, -200, 200)
        if (bgStickToBottomRef.current) {
            requestAnimationFrame(() => {
                const el = bgLogRef.current
                if (el) el.scrollTop = el.scrollHeight
            })
        }
    }, 300)

    return () => clearTimeout(bgDebounceRef.current)
}, [bgLineCount, showTask, selectedBgTaskId])

// IntersectionObserver for lazy loading older lines on scroll to top
useEffect(() => {
    const el = bgSentinelRef.current
    const scrollEl = bgLogRef.current
    if (!el || !scrollEl || !bgHasMore) return

    const observer = new IntersectionObserver(
        ([entry]) => {
            if (entry.isIntersecting) {
                bgPrevScrollHeightRef.current = scrollEl.scrollHeight
                useStore.getState().loadMoreBgTaskLines(selectedBgTaskId!)
            }
        },
        { root: scrollEl, threshold: 0 }
    )
    observer.observe(el)
    return () => observer.disconnect()
}, [bgHasMore, selectedBgTaskId])

// Restore scroll position after prepending older messages
useLayoutEffect(() => {
    if (bgPrevScrollHeightRef.current > 0) {
        const scrollEl = bgLogRef.current
        if (scrollEl) {
            const delta = scrollEl.scrollHeight - bgPrevScrollHeightRef.current
            if (delta > 0) {
                scrollEl.scrollTop += delta
            }
        }
        bgPrevScrollHeightRef.current = 0
    }
}, [bgDisplayCount, bgCache?.lines.length])

// Track stick-to-bottom state
useEffect(() => {
    const el = bgLogRef.current
    if (!el) return
    const onScroll = () => {
        const nearBottom = el.scrollTop + el.clientHeight >= el.scrollHeight - 50
        bgStickToBottomRef.current = nearBottom
    }
    el.addEventListener('scroll', onScroll)
    return () => el.removeEventListener('scroll', onScroll)
}, [showTask, selectedBgTaskId])
```

Make sure `useLayoutEffect` is imported. Check existing imports — if not present, add it to the React import.

- [ ] **Step 4: Update search filter count display**

Find the search count display (around line 1392):

```tsx
// Before:
{bgLogs.filter(l => stripAnsi(l).toLowerCase().includes(bgSearchQuery.toLowerCase())).length} / {bgLogs.length}
```

This stays the same since `bgLogs` is now derived from cache. No change needed.

- [ ] **Step 5: Update log output render area with sentinel**

Find the log output area (around lines 1403-1405):

```tsx
// Before:
<div ref={bgLogRef} className="flex-1 overflow-auto" style={{ background: '#080a0e' }}>
    <pre className="p-4 text-xs font-mono text-[#abb2bf] whitespace-pre-wrap leading-relaxed"><AnsiText text={(bgSearchQuery ? bgLogs.filter(l => stripAnsi(l).toLowerCase().includes(bgSearchQuery.toLowerCase())) : bgLogs).join('\n')} /></pre>
</div>
```

Replace with:

```tsx
<div ref={bgLogRef} className="flex-1 overflow-auto" style={{ background: '#080a0e' }}>
    {bgHasMore && <div ref={bgSentinelRef} style={{ height: 1 }} />}
    <pre className="p-4 text-xs font-mono text-[#abb2bf] whitespace-pre-wrap leading-relaxed"><AnsiText text={(bgSearchQuery ? bgLogs.filter(l => stripAnsi(l).toLowerCase().includes(bgSearchQuery.toLowerCase())) : bgLogs).join('\n')} /></pre>
</div>
```

- [ ] **Step 6: Remove stale references to old variables**

Search for any remaining references to `bgLogsLen` in the file and remove them. The old effect that used `bgLogsLen` is gone now.

Also check for `bgTaskLogs` references — there should be none left after step 1.

- [ ] **Step 7: Verify frontend compiles**

Run: `cd d:/git/monika/frontend && npx tsc --noEmit 2>&1 | head -30`
Expected: No errors

- [ ] **Step 8: Commit**

```bash
cd d:/git/monika && git add frontend/src/components/Preview/PreviewPanel.tsx && git commit -m "feat: implement lazy-loading background task log viewer with IntersectionObserver"
```

---

## Task 6: Integration testing and cleanup

- [ ] **Step 1: Run Go tests**

Run: `cd d:/git/monika && go test ./internal/api/ -v -count=1 -run "TestBackgroundTask|TestCleanOldLogs"`
Expected: All PASS

- [ ] **Step 2: Run go vet**

Run: `cd d:/git/monika && go vet ./internal/api/`
Expected: No issues

- [ ] **Step 3: Build the full project**

Run: `cd d:/git/monika && go build .`
Expected: No errors

- [ ] **Step 4: Manual test**

Start dev mode: `cd d:/git/monika && wails3 dev`

Test scenarios:
1. Start a background task (e.g., a dev server via agent or manually)
2. Verify logs appear in the TASK panel without lag
3. Scroll up — verify older lines load
4. Switch between tasks — verify cache resets
5. Stop a task — verify logs still viewable
6. Restart app — verify old logs cleaned up (only if >7 days old)

- [ ] **Step 5: Final commit if any fixes were made**

```bash
cd d:/git/monika && git add -A && git commit -m "fix: integration fixes for background task log optimization"
```
