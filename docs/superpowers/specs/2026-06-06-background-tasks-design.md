# Background Tasks Design

## Summary

Support long-running commands (dev servers, watchers, etc.) by running them in the background instead of blocking the agent loop. The AI agent autonomously decides when to background a command via the `action` parameter on the `bash` tool.

## Approach

Extend the existing `bash` tool with an `action` parameter to support background execution, lifecycle management, and log retrieval. Add a `BackgroundTaskManager` on the Go side and a TASKS panel + PREVIEW detail view on the frontend.

## Bash Tool Extension

### New Parameters

| Parameter | Type   | Required | Description                                          |
|-----------|--------|----------|------------------------------------------------------|
| `command` | string | yes      | Command to execute                                   |
| `workdir` | string | no       | Working directory                                    |
| `action`  | string | no       | `run` (default) / `background` / `stop` / `logs`    |
| `task_id` | string | no       | Target task ID when action is `stop` or `logs`       |

### Behavior

- **`action="run"`** (or omitted): executes as today, blocks until completion.
- **`action="background"`**: starts the command without waiting. Returns `task_id` and `pid`. Agent can use the `task_id` for subsequent operations.
- **`action="stop"`**: sends a termination signal to the process identified by `task_id`. Windows: `taskkill`, Unix: `SIGTERM`. Returns stop confirmation.
- **`action="logs"`**: returns the latest N log lines (default 50) for the given `task_id`.

### AI Judgment

System prompt guidance tells the agent: "When a command is expected to run for a long time (dev servers, watch processes, serve commands, etc.), use `action='background'` instead of blocking." The agent decides autonomously; no user intervention needed.

## Go: BackgroundTaskManager

New file: `internal/api/background.go`

### Structure

```go
type BackgroundTaskManager struct {
    tasks map[string]*BackgroundTask
    mu    sync.Mutex
}

type BackgroundTask struct {
    ID        string
    Command   string
    WorkDir   string
    PID       int
    Status    string       // "running" / "stopped" / "exited"
    ExitCode  int
    StartedAt time.Time
    LogBuffer *RingBuffer  // retains last N lines (default 500)
}
```

### Methods

| Method                                   | Description                                              |
|------------------------------------------|----------------------------------------------------------|
| `Start(command, workdir string)`         | Start process, return taskID. Uses `exec.CommandStart`.  |
| `Stop(taskID string)`                    | Terminate process by taskID. Platform-specific signal.   |
| `Logs(taskID string, lines int)`         | Return last N lines from the ring buffer.                |
| `List()`                                 | Return snapshot of all tasks.                            |
| `Subscribe() <-chan TaskEvent`           | Emit events for frontend: start/log/stop/exit.           |

### Key Behaviors

1. `Start` launches via `exec.CommandStart`. Stdout/stderr are tee'd into the ring buffer.
2. On process exit, status becomes `exited`, exit code is captured, and a `TaskEvent` is emitted.
3. `Stop` sends `Process.Signal` (Unix) or `taskkill` (Windows).
4. Manager is attached to the `App` struct. The bash tool accesses it via context.
5. All running processes are cleaned up on application exit.

### Task Cleanup

- User manually stops a task.
- Process exits on its own (exited status retained in list for inspection).
- Application shutdown kills all running processes.

## Frontend: TASKS Panel + PREVIEW Detail

### TASKS Panel

New dockview panel positioned to the right of FILES.

| Element       | Description                                          |
|---------------|------------------------------------------------------|
| Title         | `TASKS`                                              |
| Task list     | Each row: status icon + command name + duration      |
| Status icons  | Running (green), Stopped (red), Exited (gray)        |
| Click action  | Selects task, shows detail in PREVIEW panel          |

### PREVIEW Panel: Background Task Detail

When a background task is selected, PREVIEW switches to task detail view:

| Section      | Content                                               |
|--------------|-------------------------------------------------------|
| Header       | Command text + working directory + status + PID       |
| Action bar   | Stop button (enabled when running)                    |
| Log area     | Real-time scrolling stdout/stderr output, terminal style |
| Footer       | Start time + exit code (if exited)                    |

### Data Flow

1. Go `BackgroundTaskManager.Subscribe()` emits `TaskEvent` (start / log / stop / exit).
2. `App` forwards via Wails EventBus as `StreamEvent{Type: "bg_task", ...}`.
3. Zustand store adds `backgroundTasks` slice, listens for events, updates state.
4. TASKS panel reads task list from store.
5. PREVIEW panel reads `selectedBgTaskId` to show corresponding task logs.

### Integration with Existing Layout

- TASKS panel registered in `App.tsx` dockview components map, same level as FILES.
- PREVIEW panel adds a view mode: `file` / `bg_task`. Switches based on what the user clicks.

## Files Changed

| File                              | Change                                                |
|-----------------------------------|-------------------------------------------------------|
| `internal/api/background.go`      | New: BackgroundTaskManager + BackgroundTask types     |
| `internal/tool/builtin/bash.go`   | Extend: add `action`, `task_id` params + background logic |
| `internal/api/app.go`             | Wire BackgroundTaskManager into App, expose API methods |
| `internal/api/types.go`           | New: BackgroundTaskInfo, TaskEvent types               |
| `frontend/src/store/index.ts`     | New: backgroundTasks slice + event handling            |
| `frontend/src/components/Tasks/`  | New: TASKS panel component                             |
| `frontend/src/components/Preview/`| Extend: add bg_task view mode                          |
| `frontend/src/App.tsx`            | Register TASKS panel in dockview                       |
