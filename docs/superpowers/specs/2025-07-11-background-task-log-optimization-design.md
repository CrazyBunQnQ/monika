# Background Task Log Optimization Design

**Date**: 2025-07-11
**Status**: Approved (brainstorming complete)

## Problem

Background task logs are currently streamed via Wails events line-by-line into the Zustand store. Each line triggers:
1. `appendBgTaskLog()` — full array spread + `.slice(-500)` on every line
2. React re-render of the entire `<pre><AnsiText>` element (up to 500 lines)
3. `AnsiText` re-parsing the entire joined string each time

This causes significant UI lag when viewing high-frequency log output (e.g., dev servers, build watchers).

## Solution

Replace the in-memory ring buffer + per-line event push with file-based log storage and on-demand line-range reading, using the same displayCount + IntersectionObserver lazy-loading pattern as chat messages.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Data transport | Hybrid push (A) | Lightweight notification events; content read on demand |
| File retention | Keep files (A) | Users can view history after task exits; 7-day cleanup |
| Rendering | displayCount + IntersectionObserver (B) | Proven pattern in chat; no new dependencies |
| Architecture | Pure file + line-range API (A) | Lowest memory; supports unlimited log size |

## Architecture

### Data Flow

```
子进程 stdout
  → mvdan/sh onLine callback
  → stripANSI + write to .monika/temp/logs/<taskId>.log (Fprintln, O_APPEND)
  → lineCount++
  → emit BgTaskEvent{Type: "log_update", TaskID, LineCount}
  → app.go goroutine → Wails Event "stream"
  → frontend store updateBgTaskLineCount(taskId, count) — single number assignment
  → [if TASK panel visible] debounce 300ms → call BgTaskLogLines API for last 200 lines
  → update bgTaskLogCache → visibleLines slice → <AnsiText> render last 50 lines
  → user scrolls up → IntersectionObserver → displayCount += 50
  → if beyond cache → call API for earlier 200 lines → prepend to cache
```

## Backend Changes

### File Storage

- **Path**: `<projectDir>/.monika/temp/logs/<taskId>.log`
- **Write mode**: `O_APPEND|O_CREATE|O_WRONLY`, line-buffered via `Fprintln`
- **Retention**: files persist after task exits; cleanup at app startup deletes logs older than 7 days

### `background.go` Changes

**Remove**:
- `ringBuffer` struct and all related code (`newRingBuffer`, `Write`, `LastN`)
- `defaultRingBufferSize` constant

**`bgTask` struct changes**:
- Remove `ringBuf *ringBuffer`
- Add `logFile *os.File` — file handle for appending
- Add `lineCount int` — total lines written (protected by mutex, accessed atomically)

**`BackgroundTaskManager` changes**:
- Add `logDir string` field — set to `<projectDir>/.monika/temp/logs/` at init
- `NewBackgroundTaskManager(logDir string)` — takes log directory parameter

**`Start()` changes**:
- Create log directory if not exists
- Open log file for the task: `<logDir>/<taskID>.log`
- `onLine` callback: `stripANSI(line)` → `Fprintln(logFile, line)` → `lineCount++` → emit event

**New `LogLines(taskID string, offset, limit int) ([]string, error)` method**:
- `offset >= 0`: absolute line number from file start
- `offset < 0`: from tail (e.g., offset=-200 means 200th line from end)
- Implementation: open file read-only, `bufio.Scanner` scan line by line, skip to offset, return next `limit` lines
- Each call opens a fresh read-only fd, closes after reading

**`Logs(taskID string, lines int) ([]string, error)` — keep for backward compat**:
- Internal delegate: `LogLines(taskID, -lines, lines)`
- `BgManager` interface unchanged

**`Stop()` changes**:
- Close `logFile` after cancelling process

**`Cleanup()` changes**:
- Close all open log files
- Stop all running tasks

**Startup cleanup** — new method `CleanOldLogs()`:
- Scan `logDir` for `.log` files
- Delete files older than 7 days (based on file modtime)
- Called during `ServiceStartup`

### `background.go` Event Changes

**`BgTaskEvent` struct**:
- Remove `LogLine string`
- Add `LineCount int json:"line_count"`

**Event type constant**:
- `BgEventLog BgTaskEventType = "log"` → `BgEventLogUpdate BgTaskEventType = "log_update"`

**Event payload**:
```json
{"type": "log_update", "task_id": "xxx", "line_count": 1234}
```

### `app.go` Changes

**New API method**:
```go
func (a *App) BgTaskLogLines(taskID string, offset, limit int) ([]string, error)
```
Delegates to `bgTaskMgr.LogLines(taskID, offset, limit)`.

**Existing API — unchanged signatures**:
- `GetBgTaskLogs(taskID)` — still returns last 100 lines (internal impl changes to file read)
- `ListBgTasks()`, `StopBgTask()`, `StartBgTask()` — unchanged

**Event goroutine** (`app.go:270-280`): unchanged structure, just forwards the modified event payload.

**`ServiceStartup()`**: call `bgTaskMgr.CleanOldLogs()` after initialization.

### `background_task` Tool — No Changes

The `background_task` tool's `logs` action calls `BgManager.Logs(taskID, lines)`. Since the interface signature is unchanged and the internal implementation switches to file-backed reading, no tool code changes are needed.

## Frontend Changes

### Store (`store/index.ts`)

**Remove**:
- `bgTaskLogs: Record<string, string[]>`
- `appendBgTaskLog(taskId, line)`

**Add state**:
```ts
bgTaskLineCounts: Record<string, number>   // task_id → total line count
bgTaskLogCache: Record<string, { offset: number; lines: string[] }>  // task_id → cached line range
bgTaskDisplayCount: Record<string, number>  // task_id → visible line count
```

**Add actions**:
```ts
updateBgTaskLineCount(taskId: string, count: number)
setBgTaskLogCache(taskId: string, offset: number, lines: string[])
loadBgTaskLogs(taskId: string, offset: number, limit: number)  // async, calls Wails API
loadMoreBgTaskLines(taskId: string)  // IntersectionObserver trigger
```

**Constants**:
```ts
const BG_LOG_INITIAL_DISPLAY = 50   // default render last 50 lines
const BG_LOG_LOAD_MORE = 50          // load 50 more on scroll up
const BG_LOG_FETCH_BATCH = 200       // API reads 200 lines per call
```

**Event handler change** (`store/index.ts:2359-2388`):
```ts
case 'log_update':
    store.updateBgTaskLineCount(ev.task_id, ev.line_count)
    break
```
Single number assignment, zero array operations.

### `PreviewPanel.tsx`

**Remove**:
- `bgTaskLogs` selector
- `bgLogs` array derivation
- `appendBgTaskLog` usage

**Add**:
- Selectors for `bgTaskLineCounts[selectedBgTaskId]`, `bgTaskLogCache[selectedBgTaskId]`, `bgTaskDisplayCount[selectedBgTaskId]`
- `useEffect`: when `selectedBgTaskId` changes or `lineCount` changes (debounced 300ms), call API to fetch tail 200 lines
- IntersectionObserver: top sentinel triggers `loadMoreBgTaskLines()`, displayCount += 50, fetch more if beyond cache
- Scroll position preservation: reuse chat's `useLayoutEffect` + `prevScrollHeight` pattern
- `sentinelRef` for the observer target

**Visible lines derivation**:
```tsx
const visibleLines = useMemo(() => {
    const cache = bgTaskLogCache[selectedBgTaskId]
    if (!cache) return []
    const count = bgTaskDisplayCount[selectedBgTaskId] || BG_LOG_INITIAL_DISPLAY
    return cache.lines.slice(cache.lines.length - count)
}, [bgTaskLogCache, bgTaskDisplayCount, selectedBgTaskId])
```

**Render structure**:
```tsx
<div ref={bgLogRef} className="flex-1 overflow-auto">
    {hasMore && <div ref={sentinelRef} style={{ height: 1 }} />}
    <pre className="p-4 text-xs font-mono text-[#abb2bf] whitespace-pre-wrap leading-relaxed">
        <AnsiText text={visibleLines.join('\n')} />
    </pre>
</div>
```

**Search filter**: Ctrl+F search operates on currently cached lines only. If search term is not found, show hint "仅在当前可见范围搜索".

### Auto-scroll behavior

- When near bottom (scrollTop + clientHeight >= scrollHeight - 50px), treat as "following" mode
- In following mode, new log updates auto-scroll to bottom
- When user scrolls up away from bottom, stop auto-scrolling (same as chat `stickToBottom` pattern)

## Edge Cases

1. **TASK panel not visible**: `log_update` events still update `bgTaskLineCounts` (zero-cost number), but no API calls. First load happens when panel becomes visible.
2. **Switching tasks**: clear previous task's `bgTaskLogCache` and `bgTaskDisplayCount`, load new task's tail 200 lines.
3. **Task exited, viewing history**: file persists on disk, API reads normally.
4. **Empty or missing log file**: `LogLines` returns empty array + `lineCount: 0`, frontend shows empty state.
5. **Concurrent read/write**: writes use `O_APPEND` (atomic), reads use independent read-only fd. Safe.
6. **`background_task` tool `logs` action**: `BgManager.Logs()` interface unchanged, internal reads from file. Zero tool code changes.

## Files to Modify

| File | Change |
|------|--------|
| `internal/api/background.go` | Replace ringBuffer with file I/O; add `LogLines()`; modify events; add `CleanOldLogs()` |
| `internal/api/app.go` | Add `BgTaskLogLines` API; pass logDir to manager; call `CleanOldLogs()` at startup |
| `frontend/src/store/index.ts` | Replace `bgTaskLogs` with line-count + cache state; new actions; event handler change |
| `frontend/src/components/Preview/PreviewPanel.tsx` | Replace full-render with lazy-load pattern; add IntersectionObserver + debounce |

## Files NOT Modified

| File | Reason |
|------|--------|
| `internal/tool/builtin/background_task.go` | `BgManager.Logs()` interface unchanged |
| `internal/tool/builtin/bash.go` | `BgManager` interface unchanged |
| `main.go` | `BgTaskManager()` accessor unchanged |
