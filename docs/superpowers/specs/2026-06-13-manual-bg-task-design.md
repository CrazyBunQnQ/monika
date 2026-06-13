# Manual Background Task Design

## Summary

Allow users to manually type and run shell commands as background tasks from the TASKS panel, instead of only relying on the AI agent to start them via the bash tool.

## Approach

Add a `StartBgTask` API method on the Go `App`, a `startBgTask` action in the Zustand store, and an inline input box at the top of the TASKS panel. All existing infrastructure — `BackgroundTaskManager`, event streaming, log buffering, stop/logs API — is reused without modification.

## Go: StartBgTask API

New method on `App` in `internal/api/app.go`, placed alongside existing `ListBgTasks` / `StopBgTask` / `GetBgTaskLogs`:

```go
func (a *App) StartBgTask(command string) (string, error) {
    return a.bgTaskMgr.Start(command, a.projectPath())
}
```

Workdir is always the project root (`a.projectPath()`). The `BackgroundTaskManager.Start` method already handles process lifecycle, event emission (`bg_task` stream events), and log buffering.

## Frontend Store

New action in `frontend/src/store/index.ts`:

```ts
startBgTask: async (command: string) => {
    await Call.ByName('monika/internal/api.App.StartBgTask', command)
}
```

No manual state insertion needed — the existing `Events.On('stream', ...)` handler already listens for `bg_task` events and calls `updateBgTask` / `appendBgTaskLog` to populate the store.

## Frontend: TasksPanel Input

In `frontend/src/components/Tasks/TasksPanel.tsx`, add an `<input>` between the "TASKS" header and the task list. On Enter, calls `startBgTask` and clears the input.

Styling matches the existing `ChatInput` terminal-style (dark background, monospace font, matching border/colors).

## Files Changed

| File | Change |
|------|--------|
| `internal/api/app.go` | Add `StartBgTask` method |
| `frontend/src/store/index.ts` | Add `startBgTask` action |
| `frontend/src/components/Tasks/TasksPanel.tsx` | Add input box at top of panel |
