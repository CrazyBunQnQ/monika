# Session Message Queue Manager — Design Spec

**Date:** 2026-06-23
**Status:** Draft
**Approach:** A — Embed queue in SessionManager

## Problem

When a session is generating, new messages are hard-rejected at both the frontend (`generatingSessionIds` check in `ChatArea.tsx:190`) and backend (`cancelFuncs` check in `app.go:678`). Users cannot queue follow-up messages while the agent is working. This forces a strict send-wait-send workflow with no way to plan ahead or batch messages.

## Solution

Add a per-session message queue. Messages sent while the agent is busy are queued instead of rejected. The queue auto-drains as the agent completes each message. Users can modify, reorder, and cancel queued messages through a dedicated UI.

## Requirements Summary

| Item | Decision |
|---|---|
| Execution mode | Hybrid — auto-execute by default, user can pause for manual control |
| Persistence | Queue saved to disk as part of session JSON, survives app restart |
| UI layout | Sidebar panel (up to 10 items) + overflow full management page |
| Queue scope | Chat messages only (shell commands and compaction still reject when busy) |
| Error handling | Pause queue on failure, user decides: retry / skip / edit-then-retry |
| Queue operations | Modify text, drag-to-reorder, cancel |

## Architecture: Approach A (Embed in SessionManager)

The queue is stored as a field on the existing `Session` struct and persisted alongside the session JSON. This reuses the existing `SessionManager` Lock/Load/Save pattern with minimal new infrastructure.

## 1. Data Model

### `QueuedMessage` (new struct in `internal/api/types.go`)

```go
type QueuedMessage struct {
    ID         string `json:"id"`                       // UUID
    Text       string `json:"text"`                     // message text (editable)
    ProviderID string `json:"provider_id"`              // provider selected at send time
    Model      string `json:"model"`                    // model selected at send time
    Status     string `json:"status"`                   // "queued" | "executing" | "error"
    Error      string `json:"error,omitempty"`          // failure reason
    CreatedAt  int64  `json:"created_at"`               // unix timestamp for ordering reference
}
```

### `Session` changes (`internal/api/session_manager.go`)

```go
type Session struct {
    ...existing fields...
    Queue       []QueuedMessage `json:"queue,omitempty"`
    QueuePaused bool            `json:"queue_paused,omitempty"`
}
```

- Queue items persist with the session JSON via existing Load/Save.
- `QueuePaused` marks whether auto-execution is paused.
- The currently executing message has `Status="executing"`; on completion it is removed from the queue and enters `Messages` history (existing behavior).

## 2. Backend Queue Logic

### `SendMessage` flow change (`internal/api/app.go`)

Current behavior: if `cancelFuncs[sessionID]` exists, return error `"session is already generating"`.

New behavior:

```
SendMessage(projectPath, sessionID, text, providerID, model):
  1. Lock session, Load
  2. If session NOT generating:
     → Execute immediately (existing logic unchanged)
  3. If session IS generating:
     → Create QueuedMessage{ID: uuid, Text, ProviderID, Model, Status: "queued", CreatedAt: now}
     → Append to s.Queue
     → Save session
     → Emit "queue_updated" event
     → Return nil (success — message queued, no error)
```

### Auto-drain mechanism

In the agent loop completion goroutine (current `app.go:797` area), after persisting messages/tokens/status:

```
goroutine cleanup:
  1. Save messages, tokens, status (existing logic)
  2. If current message came from queue → remove that QueuedMessage from Queue
  3. Check QueuePaused:
     - false AND Queue has "queued" items → take first, mark "executing", start new agent loop
     - true OR Queue empty → set StatusPending, done
```

This creates chained execution: each completion triggers the next until the queue is empty or paused.

### Pause / Resume

- `PauseQueue(sessionID)`: set `QueuePaused = true`, Save.
- `ResumeQueue(sessionID)`: set `QueuePaused = false`, Save. If session is idle and queue has queued items → immediately trigger execution of the first item.

### Error handling

```
Agent loop error:
  1. Find Queue item with Status="executing"
  2. Set Status="error", Error=<error message>
  3. Set QueuePaused = true
  4. Emit "queue_error" event

User recovery options:
  - Retry:   RetryQueueItem (resets Status to "queued" AND resumes queue)
  - Skip:    SkipQueueItem (removes item AND resumes queue)
  - Edit:    EditQueueItem (change text), then RetryQueueItem
```

## 3. Wails API Bindings

### New methods (`internal/api/app.go`)

| Method | Description |
|---|---|
| `GetQueue(projectPath, sessionID) []QueuedMessage` | Return current queue |
| `EditQueueItem(projectPath, sessionID, itemID, newText)` | Edit a queued message's text |
| `ReorderQueue(projectPath, sessionID, itemIDs [])` | Reorder queue by given ID sequence |
| `CancelQueueItem(projectPath, sessionID, itemID)` | Cancel/remove a queue item (behavior depends on status — see Cancel Behavior below) |
| `PauseQueue(projectPath, sessionID)` | Pause auto-execution |
| `ResumeQueue(projectPath, sessionID)` | Resume auto-execution (triggers if idle) |
| `RetryQueueItem(projectPath, sessionID, itemID)` | Reset failed item to "queued" and resume queue |
| `SkipQueueItem(projectPath, sessionID, itemID)` | Remove failed item and resume queue |

All methods follow the `sm.Lock() → Load → modify Queue → Save → Unlock` pattern for thread safety.

### Modified existing methods

`SendMessage` — only the busy branch changes (reject → enqueue). All other logic unchanged.

### Events emitted via Wails EventEmit

| Event | Trigger |
|---|---|
| `queue_updated` | Queue changed (item added, removed, reordered, status changed) |
| `queue_item_started` | A queue item begins executing (event payload includes message text, provider, model so frontend can add it to chat) |
| `queue_error` | A queue item failed, queue paused |

## 4. Cancel Behavior

Two distinct cancel scenarios:

| Scenario | Behavior |
|---|---|
| **Cancel executing message** | Stop agent loop (CancelGeneration) + set item Status="error", Error="cancelled by user" + **pause entire queue** (`QueuePaused = true`) + emit `queue_error`. User must manually ResumeQueue to continue. |
| **Cancel queued message** | Remove item from Queue. Queue continues normally — no pause. |

`CancelQueueItem` branches on item Status:

```
CancelQueueItem(sessionID, itemID):
  If item.Status == "executing":
    → CancelGeneration(sessionID)
    → item.Status = "error", Error = "cancelled by user"
    → QueuePaused = true
    → Emit "queue_error"
  If item.Status == "queued" or "error":
    → Remove from Queue
    → Emit "queue_updated"
```

## 5. Frontend Design

### New components

**`QueuePanel`** (dockview sidebar panel)
- Shows current session's queue, max 10 items visible
- Each item: text preview, status badge (queued / executing / error), drag handle, edit/cancel buttons
- Footer: pause/resume toggle button + queue count
- "View all" link when items exceed 10

**`QueueItem`** (reusable single-item component)
- Click to enter inline edit mode
- Drag handle for reorder (HTML5 drag or @dnd-kit)
- Error state: additional retry/skip buttons

**`QueueFullPage`** (full management view)
- Full list with all queue items
- Bulk operations (cancel all, retry all failed)
- Opened via dockview panel or modal when queue exceeds 10 items

### Zustand store additions (`frontend/src/store/index.ts`)

```ts
// New state
sessionQueues: Record<string, QueuedMessage[]>
queuePaused: Record<string, boolean>

// New actions
setQueue(sessionId, items)
updateQueueItem(sessionId, itemId, changes)
removeQueueItem(sessionId, itemId)
reorderQueue(sessionId, itemIds)
toggleQueuePause(sessionId)
```

### Event listeners (in `setupWailsEvents()`)

| Event | Frontend action |
|---|---|
| `queue_updated` | Update `sessionQueues[sid]` |
| `queue_item_started` | Update corresponding item status to executing, add user message + assistant placeholder to chat (payload includes text/provider/model) |
| `queue_error` | Update item status to error, set `queuePaused[sid] = true` |

### Send message flow change (`ChatArea.tsx`)

Remove the frontend busy guard (`generatingSessionIds.includes(sessionId)` check at line 190). Always call `SendMessage`. The backend decides: execute immediately or enqueue. If enqueued, the `queue_updated` event refreshes the QueuePanel UI.

The optimistic UI append (user message + assistant placeholder) should only happen for immediately-executed messages. For queued messages, the QueuePanel shows the item instead — no chat placeholder is added until the item actually starts executing.

## 6. Edge Cases & Recovery

### App restart recovery

Existing `resetStaleSessions` (`app.go:1556`) resets `StatusGenerating` → `StatusPending`. Add:

- Scan all sessions' `Queue` fields
- Reset any `Status="executing"` items to `"queued"` (crash recovery — incomplete execution)
- If session is idle AND `QueuePaused=false` AND has `"queued"` items → auto-trigger execution of first item

### Concurrency

- The busy branch of `SendMessage` (enqueue) and all queue operation methods go through `sm.Lock() → Load → Save → Unlock`, ensuring serialization.
- Two rapid `SendMessage` calls to the same busy session: both serialize via Lock, both enqueue correctly, no race condition.

### Operation permissions by status

| Operation | queued | executing | error |
|---|---|---|---|
| Edit text | yes | no | yes |
| Drag reorder | yes | no (fixed position) | yes |
| Cancel/remove | yes (simple removal) | yes (cancel gen + pause queue) | yes (simple removal) |
| Retry | — | — | yes |

## 7. Testing Strategy

### Go backend

- Unit test `Session` queue save/load round-trip
- Unit test `SendMessage` enqueue behavior when busy (mock agent loop)
- Unit test auto-drain chain (mock agent loop completing → next item starts)
- Unit test error → pause → retry/skip flows
- Unit test restart recovery (executing → queued reset)
- Unit test concurrent SendMessage (two goroutines, same busy session)

### Frontend

- Manual testing via dev mode (`wails3 dev`)
- Verify QueuePanel renders, drag-reorder works, inline edit works
- Verify event-driven updates (queue_updated, queue_error)
- Verify overflow → full page transition at >10 items
