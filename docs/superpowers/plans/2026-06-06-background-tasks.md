# Background Tasks Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Support long-running shell commands (dev servers, watchers) by running them in the background instead of blocking the agent loop, with a TASKS panel for monitoring and control.

**Architecture:** Extend the existing `bash` tool with an `action` parameter (`run`/`background`/`stop`/`logs`). A new `BackgroundTaskManager` on the Go side manages process lifecycles, captures logs in a ring buffer, and emits events to the frontend. A new TASKS dockview panel lists background tasks, and the existing PREVIEW panel shows task details with real-time logs.

**Tech Stack:** Go (os/exec, sync), Wails v3 events, React + Zustand + dockview, Tailwind CSS v4

---

### Task 1: BackgroundTaskManager — core types and ring buffer

**Files:**
- Create: `internal/api/background.go`

- [ ] **Step 1: Write the BackgroundTask type and RingBuffer**

Create `internal/api/background.go` with the core types:

```go
package api

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
)

type BgTaskStatus string

const (
	BgTaskRunning BgTaskStatus = "running"
	BgTaskStopped BgTaskStatus = "stopped"
	BgTaskExited  BgTaskStatus = "exited"
)

type BgTaskEventType string

const (
	BgTaskStarted BgTaskEventType = "started"
	BgTaskLog     BgTaskEventType = "log"
	BgTaskStopped BgTaskEventType = "stopped"
	BgTaskExited  BgTaskEventType = "exited"
)

type BgTaskEvent struct {
	Type     BgTaskEventType `json:"type"`
	TaskID   string          `json:"task_id"`
	Command  string          `json:"command,omitempty"`
	WorkDir  string          `json:"work_dir,omitempty"`
	PID      int             `json:"pid,omitempty"`
	Status   BgTaskStatus    `json:"status,omitempty"`
	ExitCode int             `json:"exit_code,omitempty"`
	LogLine  string          `json:"log_line,omitempty"`
}

type BgTaskInfo struct {
	ID        string       `json:"id"`
	Command   string       `json:"command"`
	WorkDir   string       `json:"work_dir"`
	PID       int          `json:"pid"`
	Status    BgTaskStatus `json:"status"`
	ExitCode  int          `json:"exit_code"`
	StartedAt time.Time    `json:"started_at"`
}

const ringBufferSize = 500

type ringBuffer struct {
	lines []string
	head  int
	count int
}

func newRingBuffer(size int) *ringBuffer {
	return &ringBuffer{lines: make([]string, size)}
}

func (r *ringBuffer) Write(line string) {
	r.lines[r.head] = line
	r.head = (r.head + 1) % len(r.lines)
	if r.count < len(r.lines) {
		r.count++
	}
}

func (r *ringBuffer) LastN(n int) []string {
	if n > r.count {
		n = r.count
	}
	result := make([]string, n)
	start := (r.head - n + len(r.lines)) % len(r.lines)
	for i := 0; i < n; i++ {
		result[i] = r.lines[(start+i)%len(r.lines)]
	}
	return result
}
```

- [ ] **Step 2: Write the BackgroundTaskManager struct**

Add to the same file:

```go
type BackgroundTaskManager struct {
	mu      sync.Mutex
	tasks   map[string]*bgTask
	subscribers []chan BgTaskEvent
	subMu   sync.Mutex
}

type bgTask struct {
	info   BgTaskInfo
	logs   *ringBuffer
	cmd    *exec.Cmd
	cancel context.CancelFunc
}

func NewBackgroundTaskManager() *BackgroundTaskManager {
	return &BackgroundTaskManager{
		tasks: make(map[string]*bgTask),
	}
}
```

Add `"context"` to the imports.

- [ ] **Step 3: Write the Start method**

```go
func (m *BackgroundTaskManager) Start(command, workdir, shell, shellArg string) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, shell, shellArg, command)
	cmd.Dir = workdir
	hideWindow(cmd)

	id := uuid.New().String()
	logs := newRingBuffer(ringBufferSize)

	task := &bgTask{
		info: BgTaskInfo{
			ID:        id,
			Command:   command,
			WorkDir:   workdir,
			Status:    BgTaskRunning,
			StartedAt: time.Now(),
		},
		logs:   logs,
		cmd:    cmd,
		cancel: cancel,
	}

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		cancel()
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	task.info.PID = cmd.Process.Pid

	m.mu.Lock()
	m.tasks[id] = task
	m.mu.Unlock()

	m.emit(BgTaskEvent{
		Type:    BgTaskStarted,
		TaskID:  id,
		Command: command,
		WorkDir: workdir,
		PID:     task.info.PID,
		Status:  BgTaskRunning,
	})

	go m.readLogs(id, stdout, stderr)
	go m.waitExit(id)

	return id, nil
}
```

- [ ] **Step 4: Write the readLogs, waitExit, and helper methods**

```go
func (m *BackgroundTaskManager) readLogs(taskID string, stdout, stderr io.ReadCloser) {
	var wg sync.WaitGroup
	wg.Add(2)
	readStream := func(r io.ReadCloser) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			m.mu.Lock()
			if t, ok := m.tasks[taskID]; ok {
				t.logs.Write(line)
			}
			m.mu.Unlock()
			m.emit(BgTaskEvent{Type: BgTaskLog, TaskID: taskID, LogLine: line})
		}
	}
	go readStream(stdout)
	go readStream(stderr)
	wg.Wait()
}

func (m *BackgroundTaskManager) waitExit(taskID string) {
	m.mu.Lock()
	t, ok := m.tasks[taskID]
	m.mu.Unlock()
	if !ok {
		return
	}
	err := t.cmd.Wait()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	m.mu.Lock()
	t.info.Status = BgTaskExited
	t.info.ExitCode = exitCode
	m.mu.Unlock()

	m.emit(BgTaskEvent{
		Type:     BgTaskExited,
		TaskID:   taskID,
		Status:   BgTaskExited,
		ExitCode: exitCode,
	})
}

func (m *BackgroundTaskManager) emit(ev BgTaskEvent) {
	m.subMu.Lock()
	defer m.subMu.Unlock()
	for _, ch := range m.subscribers {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (m *BackgroundTaskManager) Subscribe() <-chan BgTaskEvent {
	ch := make(chan BgTaskEvent, 256)
	m.subMu.Lock()
	m.subscribers = append(m.subscribers, ch)
	m.subMu.Unlock()
	return ch
}
```

Add `"bufio"`, `"io"` to the imports.

- [ ] **Step 5: Write Stop, Logs, List, and Cleanup methods**

```go
func (m *BackgroundTaskManager) Stop(taskID string) error {
	m.mu.Lock()
	t, ok := m.tasks[taskID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("task %q not found", taskID)
	}
	if t.info.Status != BgTaskRunning {
		m.mu.Unlock()
		return fmt.Errorf("task %q is not running", taskID)
	}
	m.mu.Unlock()

	if runtime.GOOS == "windows" {
		kill := exec.Command("taskkill", "/PID", fmt.Sprintf("%d", t.info.PID), "/T", "/F")
		kill.Run()
	} else {
		t.cmd.Process.Signal(os.Interrupt)
	}

	t.cancel()
	t.info.Status = BgTaskStopped

	m.emit(BgTaskEvent{Type: BgTaskStopped, TaskID: taskID, Status: BgTaskStopped})
	return nil
}

func (m *BackgroundTaskManager) Logs(taskID string, lines int) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("task %q not found", taskID)
	}
	return t.logs.LastN(lines), nil
}

func (m *BackgroundTaskManager) List() []BgTaskInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]BgTaskInfo, 0, len(m.tasks))
	for _, t := range m.tasks {
		result = append(result, t.info)
	}
	return result
}

func (m *BackgroundTaskManager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.tasks {
		if t.info.Status == BgTaskRunning {
			t.cancel()
			if runtime.GOOS == "windows" {
				exec.Command("taskkill", "/PID", fmt.Sprintf("%d", t.info.PID), "/T", "/F").Run()
			} else {
				t.cmd.Process.Signal(os.Interrupt)
			}
		}
	}
}
```

- [ ] **Step 6: Verify it compiles**

Run: `cd d:/git/monika && go build ./internal/api/...`
Expected: compiles without errors

- [ ] **Step 7: Commit**

```bash
git add internal/api/background.go
git commit -m "feat: add BackgroundTaskManager with process lifecycle and ring buffer"
```

---

### Task 2: Wire BackgroundTaskManager into App

**Files:**
- Modify: `internal/api/app.go:44-86` (App struct)
- Modify: `internal/api/app.go:88-112` (NewApp)
- Modify: `internal/api/app.go:283-291` (ServiceShutdown)

- [ ] **Step 1: Add bgTaskMgr field to App struct**

In the App struct (around line 83, after `tsBridge`), add:

```go
	bgTaskMgr *BackgroundTaskManager
```

- [ ] **Step 2: Initialize in NewApp**

In the `NewApp` function return value, add:

```go
		bgTaskMgr: NewBackgroundTaskManager(),
```

- [ ] **Step 3: Add shutdown cleanup in ServiceShutdown**

In `ServiceShutdown`, before `a.eventBus.Close()`, add:

```go
	a.bgTaskMgr.Cleanup()
```

- [ ] **Step 4: Add Wails-exposed API methods for frontend**

Add new methods to `App`:

```go
func (a *App) ListBgTasks() []BgTaskInfo {
	return a.bgTaskMgr.List()
}

func (a *App) StopBgTask(taskID string) error {
	return a.bgTaskMgr.Stop(taskID)
}

func (a *App) GetBgTaskLogs(taskID string) ([]string, error) {
	return a.bgTaskMgr.Logs(taskID, 100)
}
```

- [ ] **Step 5: Add event forwarding in ServiceStartup**

In `ServiceStartup`, after the existing auto-check goroutine, add a goroutine that subscribes to background task events and forwards them to the frontend:

```go
	go func() {
		for ev := range a.bgTaskMgr.Subscribe() {
			se := StreamEvent{
				Type: "bg_task",
				Seq:  a.eventSeq.Add(1),
			}
			data, _ := json.Marshal(ev)
			se.Content = string(data)
			application.Get().Event.Emit("stream", se)
		}
	}()
```

- [ ] **Step 6: Verify it compiles**

Run: `cd d:/git/monika && go build ./internal/api/...`
Expected: compiles without errors

- [ ] **Step 7: Commit**

```bash
git add internal/api/app.go
git commit -m "feat: wire BackgroundTaskManager into App with lifecycle and event forwarding"
```

---

### Task 3: Extend bash tool with background action

**Files:**
- Modify: `internal/tool/builtin/bash.go`
- Modify: `internal/tool/builtin/register.go:20-34` (RegisterDefaults)

- [ ] **Step 1: Add BgManager interface and field to bashTool**

At the top of `bash.go`, add an interface for the background manager (avoids direct api package import):

```go
type BgManager interface {
	Start(command, workdir, shell, shellArg string) (string, error)
	Stop(taskID string) error
	Logs(taskID string, lines int) ([]string, error)
}
```

Add field to `bashTool`:

```go
type bashTool struct {
	projectDir string
	shell      string
	shellArg   string
	bgMgr      BgManager
}
```

- [ ] **Step 2: Update NewBash and add SetBgManager**

```go
func NewBash(projectDir string) (tool.Tool, error) {
	shell, shellArg := resolveShell()
	if shell == "" {
		return nil, fmt.Errorf("no shell found on system")
	}
	return &bashTool{
		projectDir: projectDir,
		shell:      shell,
		shellArg:   shellArg,
	}, nil
}

func (b *bashTool) SetBgManager(mgr BgManager) {
	b.bgMgr = mgr
}
```

- [ ] **Step 3: Update Parameters to include action and task_id**

Replace the `Parameters()` method body:

```go
func (b *bashTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The command to execute",
			},
			"workdir": map[string]any{
				"type":        "string",
				"description": "The working directory. Defaults to the project directory.",
			},
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"run", "background", "stop", "logs"},
				"description": "Action mode: 'run' (default, wait for completion), 'background' (run in background, return task_id), 'stop' (stop a background task), 'logs' (get recent logs of a background task).",
			},
			"task_id": map[string]any{
				"type":        "string",
				"description": "Background task ID. Required when action is 'stop' or 'logs'.",
			},
		},
		"required": []string{"command"},
	}
}
```

- [ ] **Step 4: Update Description to mention background mode**

Append to the existing description string (before the closing backtick), add:

```

# Background Tasks

When a command is expected to run for a long time (dev servers, watchers, file watchers, build watchers, etc.), use action='background' instead of blocking. The command runs in the background and you get a task_id back. You can check logs with action='logs' and stop with action='stop'. Do NOT block waiting for long-running commands.`
```

- [ ] **Step 5: Update Execute to handle actions**

Replace the `Execute` method. The workdir validation stays the same; after validation, branch on `action`:

```go
func (b *bashTool) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var params struct {
		Command string `json:"command"`
		Workdir string `json:"workdir"`
		Action  string `json:"action"`
		TaskID  string `json:"task_id"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	switch params.Action {
	case "stop":
		return b.executeStop(params)
	case "logs":
		return b.executeLogs(params)
	case "background":
		return b.executeBackground(ctx, params)
	default:
		return b.executeRun(ctx, params)
	}
}
```

Then add the four action methods. `executeRun` is the existing logic (lines 123-186 of the original), moved into its own method. The other three are new:

```go
func (b *bashTool) executeBackground(ctx context.Context, params struct{ Command, Workdir, Action, TaskID string }) (tool.ExecutionResult, error) {
	if b.bgMgr == nil {
		return tool.ExecutionResult{Content: "background mode not available", IsError: true}, nil
	}
	workdir := b.projectDir
	if params.Workdir != "" {
		workdir = params.Workdir
	}
	taskID, err := b.bgMgr.Start(params.Command, workdir, b.shell, b.shellArg)
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	return tool.ExecutionResult{Content: fmt.Sprintf("Background task started.\ntask_id: %s", taskID)}, nil
}

func (b *bashTool) executeStop(params struct{ Command, Workdir, Action, TaskID string }) (tool.ExecutionResult, error) {
	if b.bgMgr == nil {
		return tool.ExecutionResult{Content: "background mode not available", IsError: true}, nil
	}
	if params.TaskID == "" {
		return tool.ExecutionResult{Content: "task_id is required for stop action", IsError: true}, nil
	}
	if err := b.bgMgr.Stop(params.TaskID); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	return tool.ExecutionResult{Content: fmt.Sprintf("Task %s stopped.", params.TaskID)}, nil
}

func (b *bashTool) executeLogs(params struct{ Command, Workdir, Action, TaskID string }) (tool.ExecutionResult, error) {
	if b.bgMgr == nil {
		return tool.ExecutionResult{Content: "background mode not available", IsError: true}, nil
	}
	if params.TaskID == "" {
		return tool.ExecutionResult{Content: "task_id is required for logs action", IsError: true}, nil
	}
	logs, err := b.bgMgr.Logs(params.TaskID, 50)
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	if len(logs) == 0 {
		return tool.ExecutionResult{Content: "No logs yet."}, nil
	}
	return tool.ExecutionResult{Content: strings.Join(logs, "\n")}, nil
}
```

For `executeRun`, extract the existing logic (workdir validation + exec + truncation) into this method. The workdir validation block stays as-is but uses the params struct.

- [ ] **Step 6: Wire in register.go**

In `RegisterDefaults`, after `r.Register(sh)`, add:

```go
	// Background manager will be wired later via SetBgManager after App is created.
```

Then in `main.go`, after `appService` is created (around line 293), add:

```go
	if bashTool, ok := registry.Get("bash"); ok {
		if bt, ok := bashTool.(interface{ SetBgManager(BgManager) }); ok {
			bt.SetBgManager(appService.bgTaskMgr)
		}
	}
```

But since `main.go` can't import the `BgManager` type from `builtin` package directly, we need to export the interface. Add the `BgManager` interface to `internal/tool/builtin/bash.go` (already done in step 1). In `main.go`, use the concrete type assertion:

```go
	if bashTool, ok := registry.Get("bash"); ok {
		if setter, ok := bashTool.(interface{ SetBgManager(builtin.BgManager) }); ok {
			setter.SetBgManager(appService.BgTaskManager())
		}
	}
```

Add a getter to `App`:

```go
func (a *App) BgTaskManager() *BackgroundTaskManager {
	return a.bgTaskMgr
}
```

Note: `*BackgroundTaskManager` satisfies the `BgManager` interface since it has the same methods.

- [ ] **Step 7: Verify it compiles**

Run: `cd d:/git/monika && go build ./...`
Expected: compiles without errors

- [ ] **Step 8: Commit**

```bash
git add internal/tool/builtin/bash.go internal/tool/builtin/register.go main.go internal/api/app.go
git commit -m "feat: extend bash tool with background/stop/logs actions"
```

---

### Task 4: Frontend store — background tasks slice

**Files:**
- Modify: `frontend/src/store/index.ts`

- [ ] **Step 1: Add background task types and state**

Near the top of the store (after existing type definitions), add:

```typescript
interface BgTaskInfo {
  id: string
  command: string
  work_dir: string
  pid: number
  status: 'running' | 'stopped' | 'exited'
  exit_code: number
  started_at: string
}
```

In the state interface, add:

```typescript
  bgTasks: BgTaskInfo[]
  selectedBgTaskId: string | null
  bgTaskLogs: Record<string, string[]>
```

In the initial state, add:

```typescript
  bgTasks: [] as BgTaskInfo[],
  selectedBgTaskId: null as string | null,
  bgTaskLogs: {} as Record<string, string[]>,
```

- [ ] **Step 2: Add store actions**

Add actions to the store:

```typescript
  selectBgTask: (id: string | null) => set({ selectedBgTaskId: id }),
  updateBgTask: (info: BgTaskInfo) => set((state) => {
    const idx = state.bgTasks.findIndex(t => t.id === info.id)
    if (idx >= 0) {
      const tasks = [...state.bgTasks]
      tasks[idx] = info
      return { bgTasks: tasks }
    }
    return { bgTasks: [...state.bgTasks, info] }
  }),
  appendBgTaskLog: (taskId: string, line: string) => set((state) => ({
    bgTaskLogs: { ...state.bgTaskLogs, [taskId]: [...(state.bgTaskLogs[taskId] || []), line].slice(-500) },
  })),
```

- [ ] **Step 3: Handle bg_task stream events**

In the `Events.On('stream', ...)` handler, add a case for `bg_task` events:

```typescript
if (data.type === 'bg_task') {
  try {
    const ev = typeof data.content === 'string' ? JSON.parse(data.content) : data.content
    const store = get()
    switch (ev.type) {
      case 'started':
        store.updateBgTask({
          id: ev.task_id,
          command: ev.command,
          work_dir: ev.work_dir,
          pid: ev.pid,
          status: 'running',
          exit_code: 0,
          started_at: new Date().toISOString(),
        })
        break
      case 'log':
        store.appendBgTaskLog(ev.task_id, ev.log_line)
        break
      case 'stopped':
      case 'exited': {
        const task = store.bgTasks.find(t => t.id === ev.task_id)
        if (task) {
          store.updateBgTask({ ...task, status: ev.status, exit_code: ev.exit_code || 0 })
        }
        break
      }
    }
  } catch { /* ignore parse errors */ }
  return
}
```

- [ ] **Step 4: Verify TypeScript compiles**

Run: `cd d:/git/monika/frontend && npx tsc --noEmit`
Expected: no errors related to the new code

- [ ] **Step 5: Commit**

```bash
git add frontend/src/store/index.ts
git commit -m "feat: add background tasks state to Zustand store"
```

---

### Task 5: TASKS panel component

**Files:**
- Create: `frontend/src/components/Tasks/TasksPanel.tsx`

- [ ] **Step 1: Create the TasksPanel component**

```tsx
import { IDockviewPanelProps } from 'dockview'
import { useStore } from '../../store'

export default function TasksPanel({}: IDockviewPanelProps) {
  const bgTasks = useStore((s) => s.bgTasks)
  const selectedBgTaskId = useStore((s) => s.selectedBgTaskId)
  const selectBgTask = useStore((s) => s.selectBgTask)

  return (
    <div className="flex flex-col h-full">
      <div className="px-3 py-2 text-xs font-semibold text-[var(--text-muted)] tracking-wider border-b border-[var(--border)]">
        TASKS
      </div>
      <div className="flex-1 overflow-auto">
        {bgTasks.length === 0 && (
          <div className="px-3 py-4 text-xs text-[var(--text-muted)]">No background tasks</div>
        )}
        {bgTasks.map((task) => (
          <div
            key={task.id}
            onClick={() => selectBgTask(task.id)}
            className={`px-3 py-2 cursor-pointer border-b border-[var(--border)] hover:bg-[var(--bg-hover)] ${
              selectedBgTaskId === task.id ? 'bg-[var(--bg-active)]' : ''
            }`}
          >
            <div className="flex items-center gap-2">
              <span
                className={`w-2 h-2 rounded-full flex-shrink-0 ${
                  task.status === 'running'
                    ? 'bg-green-500'
                    : task.status === 'stopped'
                      ? 'bg-red-500'
                      : 'bg-gray-500'
                }`}
              />
              <span className="text-sm text-[var(--text)] truncate">{task.command}</span>
            </div>
            <div className="text-xs text-[var(--text-muted)] mt-0.5">
              PID {task.pid} · {task.status}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Register in App.tsx**

In `frontend/src/App.tsx`:

Add import:
```tsx
import TasksPanel from './components/Tasks/TasksPanel'
```

Add to `components` map:
```tsx
const components: Record<string, React.FunctionComponent<IDockviewPanelProps>> = {
  chat: ChatArea,
  preview: PreviewPanel,
  files: FileTree,
  changes: ChangesList,
  session: SessionList,
  tasks: TasksPanel,
}
```

- [ ] **Step 3: Add to default layout**

In `frontend/src/components/Panel/defaultLayout.ts`, add a `tasks` panel alongside `changes`. In the branch that contains `files-group` and `changes-group`, add a new leaf:

```typescript
{
  type: 'leaf',
  size: FS_CH_W,
  data: { id: 'tasks-group', views: ['tasks'], activeView: 'tasks' },
},
```

And add to the panels map:

```typescript
tasks: {
  id: 'tasks',
  contentComponent: 'tasks',
  tabComponent: 'session-tab',
  title: 'TASKS',
  renderer: 'always',
},
```

Adjust sizes: change `FS_CH_W` split from 50/50 to three-way. Replace the branch with three leaves each at `FS_CH_W / 1.5` or approximately 177.

- [ ] **Step 4: Verify frontend compiles**

Run: `cd d:/git/monika/frontend && npx tsc --noEmit`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/Tasks/TasksPanel.tsx frontend/src/App.tsx frontend/src/components/Panel/defaultLayout.ts
git commit -m "feat: add TASKS panel component and register in dockview layout"
```

---

### Task 6: PREVIEW panel — background task detail view

**Files:**
- Modify: `frontend/src/components/Preview/PreviewPanel.tsx`

- [ ] **Step 1: Read the current PreviewPanel to understand its structure**

Read the file to understand how it currently handles file preview mode. We'll add a parallel view for background task details.

- [ ] **Step 2: Add background task detail view**

When `selectedBgTaskId` is set and the user clicks a task in the TASKS panel, the PREVIEW panel should show the task's details and logs. Add a condition at the top of the PreviewPanel render: if `selectedBgTaskId` is set, render the task detail view instead of the file preview.

The task detail view should show:
- Header: command + work_dir + status badge + PID
- Stop button (only when running)
- Log output in a scrollable terminal-style `<pre>` element
- Footer: started_at + exit_code (if exited)

```tsx
// Inside PreviewPanel, at the start of the render function:
const selectedBgTaskId = useStore((s) => s.selectedBgTaskId)
const bgTasks = useStore((s) => s.bgTasks)
const bgTaskLogs = useStore((s) => s.bgTaskLogs)
const stopBgTask = useStore((s) => s.stopBgTask)

if (selectedBgTaskId) {
  const task = bgTasks.find(t => t.id === selectedBgTaskId)
  const logs = bgTaskLogs[selectedBgTaskId] || []
  if (!task) return null

  return (
    <div className="flex flex-col h-full">
      <div className="px-4 py-3 border-b border-[var(--border)]">
        <div className="text-sm font-mono text-[var(--text)] break-all">{task.command}</div>
        <div className="text-xs text-[var(--text-muted)] mt-1">{task.work_dir}</div>
        <div className="flex items-center gap-3 mt-2">
          <span className={`text-xs px-1.5 py-0.5 rounded ${
            task.status === 'running' ? 'bg-green-900 text-green-300' :
            task.status === 'stopped' ? 'bg-red-900 text-red-300' :
            'bg-gray-800 text-gray-300'
          }`}>{task.status}</span>
          <span className="text-xs text-[var(--text-muted)]">PID {task.pid}</span>
          {task.status === 'running' && (
            <button
              onClick={() => stopBgTask(task.id)}
              className="text-xs px-2 py-0.5 bg-red-600 text-white rounded hover:bg-red-700"
            >Stop</button>
          )}
        </div>
      </div>
      <div className="flex-1 overflow-auto p-4">
        <pre className="text-xs font-mono text-[var(--text)] whitespace-pre-wrap">{logs.join('\n')}</pre>
      </div>
      <div className="px-4 py-2 border-t border-[var(--border)] text-xs text-[var(--text-muted)]">
        Started: {new Date(task.started_at).toLocaleTimeString()}
        {task.exit_code > 0 && ` · Exit code: ${task.exit_code}`}
      </div>
    </div>
  )
}
```

- [ ] **Step 3: Add stopBgTask action to store**

In `frontend/src/store/index.ts`, add:

```typescript
  stopBgTask: async (taskId: string) => {
    try {
      const { StopBgTask } = await import('../../bindings/monika')
      await StopBgTask(taskId)
    } catch (e) {
      console.error('[monika] failed to stop bg task:', e)
    }
  },
```

- [ ] **Step 4: Verify frontend compiles**

Run: `cd d:/git/monika/frontend && npx tsc --noEmit`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/Preview/PreviewPanel.tsx frontend/src/store/index.ts
git commit -m "feat: add background task detail view in PREVIEW panel"
```

---

### Task 7: Regenerate Wails bindings and verify end-to-end

**Files:**
- Modified: `frontend/bindings/monika/` (auto-generated)

- [ ] **Step 1: Regenerate Wails bindings**

Run: `cd d:/git/monika && wails3 generate bindings -f "..." -ts`

This generates TypeScript bindings for the new `ListBgTasks`, `StopBgTask`, and `GetBgTaskLogs` methods on `App`.

- [ ] **Step 2: Verify the full project builds**

Run: `cd d:/git/monika && go build ./...`
Expected: compiles without errors

Run: `cd d:/git/monika/frontend && npx tsc --noEmit`
Expected: no TypeScript errors

- [ ] **Step 3: Commit**

```bash
git add frontend/bindings/
git commit -m "chore: regenerate Wails bindings for background task APIs"
```

---

### Task 8: Integration test and final commit

**Files:**
- None new (verification only)

- [ ] **Step 1: Manual smoke test**

1. Run `wails3 dev`
2. In the chat, ask the AI to "start a dev server" or "run npm run dev"
3. Verify: AI uses `action='background'`, task appears in TASKS panel
4. Click the task in TASKS → PREVIEW shows logs in real-time
5. Click Stop → task stops
6. Verify logs are still viewable after task exits

- [ ] **Step 2: Final commit if any fixes needed**

```bash
git add -A
git commit -m "fix: integration fixes from smoke test"
```
