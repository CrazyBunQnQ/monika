# Session Message Queue Manager — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a per-session message queue that lets users send messages while the agent is busy, with modify/reorder/cancel operations and auto-execution.

**Architecture:** Queue embedded in SessionManager (Approach A). `QueuedMessage` items stored as a `Queue []QueuedMessage` field on `Session`, persisted via existing session JSON. SendMessage enqueues when busy instead of rejecting. Auto-drain after agent loop completion. Sidebar panel UI for queue management.

**Tech Stack:** Go 1.25+ (backend), React 18 + TypeScript + Tailwind CSS v4 + Zustand (frontend), Wails v3 (IPC)

**Spec:** `docs/superpowers/specs/2026-06-23-message-queue-manager-design.md`

---

## File Structure

**Create:**
- `internal/api/queue_test.go` — unit tests for queue logic
- `frontend/src/components/QueuePanel/QueuePanel.tsx` — sidebar panel component
- `frontend/src/components/QueuePanel/QueueItem.tsx` — single queue item component

**Modify:**
- `internal/api/types.go` — add `QueuedMessage` struct
- `internal/api/session_manager.go` — add `Queue`/`QueuePaused` fields + queue helper methods
- `internal/api/app.go` — refactor SendMessage, add auto-drain, add 8 new API methods, add events, add recovery
- `frontend/src/store/index.ts` — add queue state + actions + event listeners
- `frontend/src/components/Chat/ChatArea.tsx` — remove busy guard, handle enqueue response
- `frontend/src/App.tsx` — register QueuePanel in dockview components map

---

## Task 1: Data Model — QueuedMessage + Session Fields

**Files:**
- Modify: `internal/api/types.go` (append after line 265)
- Modify: `internal/api/session_manager.go:25-45` (Session struct)
- Create: `internal/api/queue_test.go`

- [ ] **Step 1: Add QueuedMessage struct to types.go**

Add to `internal/api/types.go` after the last type (after line 265):

```go
// QueuedMessage represents a chat message waiting in a session's queue.
type QueuedMessage struct {
	ID         string `json:"id"`
	Text       string `json:"text"`
	ProviderID string `json:"provider_id"`
	Model      string `json:"model"`
	Status     string `json:"status"`          // "queued" | "executing" | "error"
	Error      string `json:"error,omitempty"`
	CreatedAt  int64  `json:"created_at"`
}
```

- [ ] **Step 2: Add Queue fields to Session struct**

In `internal/api/session_manager.go`, add two fields to the `Session` struct (after `WorktreePath` at line 44):

```go
	WorktreePath    string               `json:"worktree_path,omitempty"`
	Queue           []QueuedMessage      `json:"queue,omitempty"`
	QueuePaused     bool                 `json:"queue_paused,omitempty"`
```

- [ ] **Step 3: Write test for queue save/load round-trip**

Create `internal/api/queue_test.go`:

```go
package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionQueueSaveLoad(t *testing.T) {
	dir := t.TempDir()
	sm := NewSessionManager(dir, dir)

	s, err := sm.New("test-model", "test-provider")
	if err != nil {
		t.Fatal(err)
	}

	s.Queue = []QueuedMessage{
		{ID: "q1", Text: "hello", ProviderID: "p", Model: "m", Status: "queued", CreatedAt: 1},
		{ID: "q2", Text: "world", ProviderID: "p", Model: "m", Status: "queued", CreatedAt: 2},
	}
	s.QueuePaused = true

	if err := sm.Save(s); err != nil {
		t.Fatal(err)
	}

	loaded, err := sm.Load(s.ID)
	if err != nil {
		t.Fatal(err)
	}

	if len(loaded.Queue) != 2 {
		t.Fatalf("expected 2 queue items, got %d", len(loaded.Queue))
	}
	if loaded.Queue[0].ID != "q1" || loaded.Queue[0].Text != "hello" {
		t.Errorf("unexpected first item: %+v", loaded.Queue[0])
	}
	if !loaded.QueuePaused {
		t.Error("expected QueuePaused=true")
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/api/ -run TestSessionQueueSaveLoad -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/api/types.go internal/api/session_manager.go internal/api/queue_test.go
git commit -m "feat: add QueuedMessage struct and Queue fields to Session"
```

---

## Task 2: SessionManager Queue Helper Methods

**Files:**
- Modify: `internal/api/session_manager.go` (append after line 222)
- Modify: `internal/api/queue_test.go` (add tests)

- [ ] **Step 1: Write failing tests for queue helper methods**

Add to `internal/api/queue_test.go`:

```go
func TestSessionQueueHelpers(t *testing.T) {
	dir := t.TempDir()
	sm := NewSessionManager(dir, dir)

	s, err := sm.New("m", "p")
	if err != nil {
		t.Fatal(err)
	}

	// Enqueue
	sm.EnqueueQueueItem(s, QueuedMessage{ID: "q1", Text: "first", Status: "queued", CreatedAt: 1})
	sm.EnqueueQueueItem(s, QueuedMessage{ID: "q2", Text: "second", Status: "queued", CreatedAt: 2})
	if len(s.Queue) != 2 {
		t.Fatalf("expected 2 items, got %d", len(s.Queue))
	}

	// Find
	idx := sm.FindQueueItem(s, "q2")
	if idx != 1 {
		t.Fatalf("expected index 1, got %d", idx)
	}

	// Update
	sm.UpdateQueueItem(s, "q1", func(item *QueuedMessage) {
		item.Text = "edited"
	})
	if s.Queue[0].Text != "edited" {
		t.Errorf("expected edited text, got %s", s.Queue[0].Text)
	}

	// Remove
	sm.RemoveQueueItem(s, "q1")
	if len(s.Queue) != 1 || s.Queue[0].ID != "q2" {
		t.Errorf("expected only q2 remaining, got %+v", s.Queue)
	}

	// Reorder
	sm.EnqueueQueueItem(s, QueuedMessage{ID: "q3", Text: "third", Status: "queued", CreatedAt: 3})
	sm.ReorderQueue(s, []string{"q3", "q2"})
	if s.Queue[0].ID != "q3" || s.Queue[1].ID != "q2" {
		t.Errorf("reorder failed: %+v", s.Queue)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/api/ -run TestSessionQueueHelpers -v`
Expected: FAIL — `sm.EnqueueQueueItem undefined`

- [ ] **Step 3: Implement queue helper methods**

Add to `internal/api/session_manager.go` after line 222 (after `Unlock` method):

```go
func (sm *SessionManager) EnqueueQueueItem(s *Session, item QueuedMessage) {
	s.Queue = append(s.Queue, item)
}

func (sm *SessionManager) FindQueueItem(s *Session, itemID string) int {
	for i, item := range s.Queue {
		if item.ID == itemID {
			return i
		}
	}
	return -1
}

func (sm *SessionManager) UpdateQueueItem(s *Session, itemID string, fn func(*QueuedMessage)) {
	for i := range s.Queue {
		if s.Queue[i].ID == itemID {
			fn(&s.Queue[i])
			return
		}
	}
}

func (sm *SessionManager) RemoveQueueItem(s *Session, itemID string) {
	idx := sm.FindQueueItem(s, itemID)
	if idx >= 0 {
		s.Queue = append(s.Queue[:idx], s.Queue[idx+1:]...)
	}
}

func (sm *SessionManager) ReorderQueue(s *Session, itemIDs []string) {
	itemMap := make(map[string]QueuedMessage)
	for _, item := range s.Queue {
		itemMap[item.ID] = item
	}
	var reordered []QueuedMessage
	for _, id := range itemIDs {
		if item, ok := itemMap[id]; ok {
			reordered = append(reordered, item)
		}
	}
	s.Queue = reordered
}

func (sm *SessionManager) NextQueuedItem(s *Session) *QueuedMessage {
	for i := range s.Queue {
		if s.Queue[i].Status == "queued" {
			return &s.Queue[i]
		}
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/api/ -run TestSessionQueueHelpers -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/api/session_manager.go internal/api/queue_test.go
git commit -m "feat: add queue helper methods to SessionManager"
```

---

## Task 3: Refactor SendMessage + Enqueue When Busy

**Files:**
- Modify: `internal/api/app.go:666-816` (SendMessage function)
- Modify: `internal/api/queue_test.go` (add enqueue test)

This task extracts the core execution logic into a private method `startAgentLoop`, then modifies `SendMessage` to enqueue when busy instead of returning an error.

- [ ] **Step 1: Write failing test for enqueue-when-busy**

Add to `internal/api/queue_test.go`:

```go
func TestEnqueueWhenBusy(t *testing.T) {
	dir := t.TempDir()
	sm := NewSessionManager(dir, dir)

	s, err := sm.New("m", "p")
	if err != nil {
		t.Fatal(err)
	}
	s.Status = StatusGenerating
	if err := sm.Save(s); err != nil {
		t.Fatal(err)
	}

	// Simulate: session is generating, enqueue a message
	item := QueuedMessage{
		ID:         generateID(),
		Text:       "queued msg",
		ProviderID: "p",
		Model:      "m",
		Status:     "queued",
		CreatedAt:  time.Now().Unix(),
	}

	sm.Lock()
	loaded, _ := sm.Load(s.ID)
	sm.EnqueueQueueItem(loaded, item)
	sm.Save(loaded)
	sm.Unlock()

	reloaded, _ := sm.Load(s.ID)
	if len(reloaded.Queue) != 1 || reloaded.Queue[0].Text != "queued msg" {
		t.Errorf("expected 1 queued item, got %+v", reloaded.Queue)
	}
}
```

Add `"time"` to the imports in `queue_test.go`.

- [ ] **Step 2: Run test to verify it passes (uses existing helpers)**

Run: `go test ./internal/api/ -run TestEnqueueWhenBusy -v`
Expected: PASS

- [ ] **Step 3: Extract `startAgentLoop` from SendMessage**

In `internal/api/app.go`, extract the execution portion of `SendMessage` (lines 706-813, from provider lookup through the goroutine) into a new method. The new method signature:

```go
func (a *App) startAgentLoop(sm *SessionManager, s *Session, sessionID, text, providerID, model, queueItemID string)
```

This method assumes:
- `cancelFuncs[sessionID]` is already set by the caller
- The caller holds no lock (locks are taken inside as needed)

Move lines 706-813 into `startAgentLoop`. At the end of the goroutine (around current line 797-805), after saving the session status, add auto-drain call:

```go
		// Auto-drain: if message came from queue, remove it and check for next
		if queueItemID != "" {
			sm.RemoveQueueItem(s, queueItemID)
		}
		sm.SetStatus(s, StatusPending)
		sm.Save(s)
		sm.Unlock()

		// Auto-drain check (implemented in Task 4)
		a.drainQueue(sm, sessionID)
```

Note: the `ctx.Err()` check for cancelled vs normal completion stays. For cancelled context (user cancelled), skip the drain. For normal completion, drain.

Updated goroutine cleanup section:

```go
		sm.Lock()
		if queueItemID != "" {
			sm.RemoveQueueItem(s, queueItemID)
		}
		if ctx.Err() != nil {
			sm.SetStatus(s, StatusPending)
		} else {
			sm.SetStatus(s, StatusPending)
		}
		sm.Save(s)
		sm.Unlock()

		if ctx.Err() == nil {
			a.handleAgentEvent(sessionID, model, agent2.Event{
				Type:    agent2.EventSessionUpdated,
				Content: s.Title,
			})
			// Auto-drain only on normal completion
			a.drainQueue(sm, sessionID)
		}
```

- [ ] **Step 4: Rewrite SendMessage with enqueue branch**

Replace `SendMessage` (lines 666-816) with:

```go
func (a *App) SendMessage(projectPath, sessionID, text, providerID, model string) error {
	sm := a.getSessionManager(projectPath)
	sm.Lock()

	s, err := sm.Load(sessionID)
	if err != nil {
		sm.Unlock()
		return err
	}

	// Check if session is busy
	a.cancelMu.Lock()
	_, busy := a.cancelFuncs[sessionID]
	a.cancelMu.Unlock()

	if busy {
		// Enqueue the message instead of rejecting
		item := QueuedMessage{
			ID:         generateID(),
			Text:       text,
			ProviderID: providerID,
			Model:      model,
			Status:     "queued",
			CreatedAt:  time.Now().Unix(),
		}
		sm.EnqueueQueueItem(s, item)
		if err := sm.Save(s); err != nil {
			sm.Unlock()
			return err
		}
		sm.Unlock()

		// Notify frontend
		a.emitQueueUpdated(sessionID, s.Queue)
		return nil
	}

	// Not busy — set up cancel func and execute
	ctx, cancel := context.WithCancel(a.ctx)
	a.cancelMu.Lock()
	a.cancelFuncs[sessionID] = cancel
	a.cancelMu.Unlock()
	sm.Unlock()

	a.startAgentLoop(sm, s, sessionID, text, providerID, model, "")
	return nil
}
```

- [ ] **Step 5: Add placeholder `drainQueue` method (implemented in Task 4)**

Add temporarily to `internal/api/app.go`:

```go
func (a *App) drainQueue(sm *SessionManager, sessionID string) {
	// Implemented in Task 4
}
```

Also add placeholder:

```go
func (a *App) emitQueueUpdated(sessionID string, queue []QueuedMessage) {
	// Implemented in Task 6
}
```

- [ ] **Step 6: Verify build compiles**

Run: `go build ./internal/api/`
Expected: No errors

- [ ] **Step 7: Run existing tests**

Run: `go test ./internal/api/ -run TestSessionQueue -v`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add internal/api/app.go internal/api/queue_test.go
git commit -m "feat: refactor SendMessage, enqueue messages when session is busy"
```

---

## Task 4: Auto-Drain Mechanism

**Files:**
- Modify: `internal/api/app.go` (drainQueue method)
- Modify: `internal/api/queue_test.go` (add drain test)

- [ ] **Step 1: Write failing test for drainQueue logic**

Add to `internal/api/queue_test.go`:

```go
func TestNextQueuedItem(t *testing.T) {
	dir := t.TempDir()
	sm := NewSessionManager(dir, dir)

	s, _ := sm.New("m", "p")
	s.Queue = []QueuedMessage{
		{ID: "q1", Text: "first", Status: "executing", CreatedAt: 1},
		{ID: "q2", Text: "second", Status: "queued", CreatedAt: 2},
		{ID: "q3", Text: "third", Status: "error", CreatedAt: 3},
		{ID: "q4", Text: "fourth", Status: "queued", CreatedAt: 4},
	}

	next := sm.NextQueuedItem(s)
	if next == nil || next.ID != "q2" {
		t.Errorf("expected q2, got %+v", next)
	}

	// Mark q2 as executing, next should be q4
	sm.UpdateQueueItem(s, "q2", func(item *QueuedMessage) { item.Status = "executing" })
	next = sm.NextQueuedItem(s)
	if next == nil || next.ID != "q4" {
		t.Errorf("expected q4, got %+v", next)
	}

	// No queued items
	sm.UpdateQueueItem(s, "q4", func(item *QueuedMessage) { item.Status = "executing" })
	next = sm.NextQueuedItem(s)
	if next != nil {
		t.Errorf("expected nil, got %+v", next)
	}
}
```

- [ ] **Step 2: Run test to verify it passes (helper already exists)**

Run: `go test ./internal/api/ -run TestNextQueuedItem -v`
Expected: PASS

- [ ] **Step 3: Implement drainQueue**

Replace the placeholder `drainQueue` in `internal/api/app.go` with:

```go
func (a *App) drainQueue(sm *SessionManager, sessionID string) {
	sm.Lock()
	defer sm.Unlock()

	s, err := sm.Load(sessionID)
	if err != nil {
		return
	}

	// Don't drain if paused
	if s.QueuePaused {
		return
	}

	// Find next queued item
	item := sm.NextQueuedItem(s)
	if item == nil {
		return
	}

	// Check not busy
	a.cancelMu.Lock()
	_, busy := a.cancelFuncs[sessionID]
	a.cancelMu.Unlock()
	if busy {
		return
	}

	// Mark as executing
	sm.UpdateQueueItem(s, item.ID, func(qi *QueuedMessage) {
		qi.Status = "executing"
	})
	sm.Save(s)

	// Notify frontend that this item is starting
	a.emitQueueItemStarted(sessionID, *item)

	// Set up cancel func
	ctx, cancel := context.WithCancel(a.ctx)
	a.cancelMu.Lock()
	a.cancelFuncs[sessionID] = cancel
	a.cancelMu.Unlock()

	// Start agent loop for this queued message
	a.startAgentLoop(sm, s, sessionID, item.Text, item.ProviderID, item.Model, item.ID)
}
```

- [ ] **Step 4: Add placeholder `emitQueueItemStarted`**

```go
func (a *App) emitQueueItemStarted(sessionID string, item QueuedMessage) {
	// Implemented in Task 6
}
```

- [ ] **Step 5: Verify build compiles**

Run: `go build ./internal/api/`
Expected: No errors

- [ ] **Step 6: Commit**

```bash
git add internal/api/app.go internal/api/queue_test.go
git commit -m "feat: add auto-drain mechanism for message queue"
```

---

## Task 5: Queue API Methods

**Files:**
- Modify: `internal/api/app.go` (add 8 new exported methods)

- [ ] **Step 1: Implement all queue API methods**

Add these methods to `internal/api/app.go`:

```go
func (a *App) GetQueue(projectPath, sessionID string) ([]QueuedMessage, error) {
	sm := a.getSessionManager(projectPath)
	sm.Lock()
	defer sm.Unlock()
	s, err := sm.Load(sessionID)
	if err != nil {
		return nil, err
	}
	return s.Queue, nil
}

func (a *App) EditQueueItem(projectPath, sessionID, itemID, newText string) error {
	sm := a.getSessionManager(projectPath)
	sm.Lock()
	defer sm.Unlock()
	s, err := sm.Load(sessionID)
	if err != nil {
		return err
	}
	idx := sm.FindQueueItem(s, itemID)
	if idx < 0 {
		return fmt.Errorf("queue item %s not found", itemID)
	}
	if s.Queue[idx].Status == "executing" {
		return fmt.Errorf("cannot edit an executing message")
	}
	sm.UpdateQueueItem(s, itemID, func(item *QueuedMessage) {
		item.Text = newText
	})
	if err := sm.Save(s); err != nil {
		return err
	}
	a.emitQueueUpdated(sessionID, s.Queue)
	return nil
}

func (a *App) ReorderQueue(projectPath, sessionID string, itemIDs []string) error {
	sm := a.getSessionManager(projectPath)
	sm.Lock()
	defer sm.Unlock()
	s, err := sm.Load(sessionID)
	if err != nil {
		return err
	}
	sm.ReorderQueue(s, itemIDs)
	if err := sm.Save(s); err != nil {
		return err
	}
	a.emitQueueUpdated(sessionID, s.Queue)
	return nil
}

func (a *App) CancelQueueItem(projectPath, sessionID, itemID string) error {
	sm := a.getSessionManager(projectPath)
	sm.Lock()
	s, err := sm.Load(sessionID)
	if err != nil {
		sm.Unlock()
		return err
	}
	idx := sm.FindQueueItem(s, itemID)
	if idx < 0 {
		sm.Unlock()
		return fmt.Errorf("queue item %s not found", itemID)
	}

	if s.Queue[idx].Status == "executing" {
		// Cancel current generation + pause queue
		sm.UpdateQueueItem(s, itemID, func(item *QueuedMessage) {
			item.Status = "error"
			item.Error = "cancelled by user"
		})
		s.QueuePaused = true
		if err := sm.Save(s); err != nil {
			sm.Unlock()
			return err
		}
		sm.Unlock()

		// Cancel the running agent loop
		a.cancelMu.Lock()
		cancel, ok := a.cancelFuncs[sessionID]
		a.cancelMu.Unlock()
		if ok {
			cancel()
		}

		a.emitQueueError(sessionID, itemID, "cancelled by user")
		return nil
	}

	// Simple removal for queued/error items
	sm.RemoveQueueItem(s, itemID)
	if err := sm.Save(s); err != nil {
		sm.Unlock()
		return err
	}
	sm.Unlock()
	a.emitQueueUpdated(sessionID, s.Queue)
	return nil
}

func (a *App) PauseQueue(projectPath, sessionID string) error {
	sm := a.getSessionManager(projectPath)
	sm.Lock()
	defer sm.Unlock()
	s, err := sm.Load(sessionID)
	if err != nil {
		return err
	}
	s.QueuePaused = true
	if err := sm.Save(s); err != nil {
		return err
	}
	a.emitQueueUpdated(sessionID, s.Queue)
	return nil
}

func (a *App) ResumeQueue(projectPath, sessionID string) error {
	sm := a.getSessionManager(projectPath)
	sm.Lock()
	s, err := sm.Load(sessionID)
	if err != nil {
		sm.Unlock()
		return err
	}
	s.QueuePaused = false
	if err := sm.Save(s); err != nil {
		sm.Unlock()
		return err
	}
	sm.Unlock()

	// If idle and has queued items, trigger drain
	a.cancelMu.Lock()
	_, busy := a.cancelFuncs[sessionID]
	a.cancelMu.Unlock()
	if !busy {
		a.drainQueue(sm, sessionID)
	}
	return nil
}

func (a *App) RetryQueueItem(projectPath, sessionID, itemID string) error {
	sm := a.getSessionManager(projectPath)
	sm.Lock()
	s, err := sm.Load(sessionID)
	if err != nil {
		sm.Unlock()
		return err
	}
	idx := sm.FindQueueItem(s, itemID)
	if idx < 0 {
		sm.Unlock()
		return fmt.Errorf("queue item %s not found", itemID)
	}
	if s.Queue[idx].Status != "error" {
		sm.Unlock()
		return fmt.Errorf("can only retry failed items")
	}
	sm.UpdateQueueItem(s, itemID, func(item *QueuedMessage) {
		item.Status = "queued"
		item.Error = ""
	})
	s.QueuePaused = false
	if err := sm.Save(s); err != nil {
		sm.Unlock()
		return err
	}
	sm.Unlock()

	// Trigger drain if idle
	a.cancelMu.Lock()
	_, busy := a.cancelFuncs[sessionID]
	a.cancelMu.Unlock()
	if !busy {
		a.drainQueue(sm, sessionID)
	}
	return nil
}

func (a *App) SkipQueueItem(projectPath, sessionID, itemID string) error {
	sm := a.getSessionManager(projectPath)
	sm.Lock()
	s, err := sm.Load(sessionID)
	if err != nil {
		sm.Unlock()
		return err
	}
	sm.RemoveQueueItem(s, itemID)
	s.QueuePaused = false
	if err := sm.Save(s); err != nil {
		sm.Unlock()
		return err
	}
	sm.Unlock()

	// Trigger drain if idle
	a.cancelMu.Lock()
	_, busy := a.cancelFuncs[sessionID]
	a.cancelMu.Unlock()
	if !busy {
		a.drainQueue(sm, sessionID)
	}
	return nil
}
```

- [ ] **Step 2: Verify build compiles**

Run: `go build ./internal/api/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/api/app.go
git commit -m "feat: add queue management API methods (edit, reorder, cancel, pause, resume, retry, skip)"
```

---

## Task 6: Queue Events

**Files:**
- Modify: `internal/api/app.go` (implement event emitters)

- [ ] **Step 1: Implement event emitter methods**

Replace the three placeholder methods in `internal/api/app.go`:

```go
func (a *App) emitQueueUpdated(sessionID string, queue []QueuedMessage) {
	se := StreamEvent{
		SessionID: sessionID,
		Type:      "queue_updated",
		Seq:       a.eventSeq.Add(1),
	}
	se.Content = queueToJSON(queue)
	application.Get().Event.Emit("stream", se)
}

func (a *App) emitQueueItemStarted(sessionID string, item QueuedMessage) {
	se := StreamEvent{
		SessionID: sessionID,
		Type:      "queue_item_started",
		Seq:       a.eventSeq.Add(1),
	}
	se.Content = queueItemToJSON(item)
	application.Get().Event.Emit("stream", se)
}

func (a *App) emitQueueError(sessionID, itemID, errorMsg string) {
	se := StreamEvent{
		SessionID: sessionID,
		Type:      "queue_error",
		Seq:       a.eventSeq.Add(1),
	}
	se.Content = fmt.Sprintf(`{"item_id":%q,"error":%q}`, itemID, errorMsg)
	application.Get().Event.Emit("stream", se)
}

func queueToJSON(queue []QueuedMessage) string {
	data, err := json.Marshal(queue)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func queueItemToJSON(item QueuedMessage) string {
	data, err := json.Marshal(item)
	if err != nil {
		return "{}"
	}
	return string(data)
}
```

- [ ] **Step 2: Add error handling in startAgentLoop goroutine**

In the `startAgentLoop` goroutine (inside the event processing loop), after `for ev := range events`, add error detection. After the loop ends, check if an error event was emitted:

Find the event processing loop in `startAgentLoop` and add tracking:

```go
		var hadError bool
		events := loop.Run(ctx, conv, text)
		for ev := range events {
			select {
			case <-ctx.Done():
				_ = ctx.Err()
			default:
			}
			if ev.Type == agent2.EventError {
				hadError = true
			}
			a.handleAgentEvent(sessionID, model, ev)
		}
```

Then in the cleanup section, add error-pause logic before the normal save:

```go
		if hadError && queueItemID != "" {
			// Error during queued message execution — pause queue
			sm.Lock()
			sm.UpdateQueueItem(s, queueItemID, func(item *QueuedMessage) {
				item.Status = "error"
			})
			s.QueuePaused = true
			sm.Save(s)
			sm.Unlock()
			// Notify frontend
			a.emitQueueError(sessionID, queueItemID, "execution failed")
			// Do not drain — queue is paused
			return
		}
```

This block goes BEFORE the normal completion save + drain logic.

- [ ] **Step 3: Verify build compiles**

Run: `go build ./internal/api/`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add internal/api/app.go
git commit -m "feat: emit queue events (queue_updated, queue_item_started, queue_error)"
```

---

## Task 7: Restart Recovery

**Files:**
- Modify: `internal/api/app.go:1556-1574` (resetStaleSessions)

- [ ] **Step 1: Add queue recovery to resetStaleSessions**

In `internal/api/app.go`, modify `resetStaleSessions` to also recover queue state. Replace the function body:

```go
func (a *App) resetStaleSessions(projectPath string) {
	sm := a.getSessionManager(projectPath)
	sessions, err := sm.List()
	if err != nil {
		return
	}
	for _, info := range sessions {
		s, err := sm.Load(info.ID)
		if err != nil {
			continue
		}
		needsSave := false

		// Reset stale generating status
		if s.Status == StatusGenerating {
			s.Status = StatusPending
			needsSave = true
		}

		// Recover queue: reset "executing" items to "queued"
		for i := range s.Queue {
			if s.Queue[i].Status == "executing" {
				s.Queue[i].Status = "queued"
				needsSave = true
			}
		}

		if needsSave {
			sm.Lock()
			sm.Save(s)
			sm.Unlock()
		}

		// Auto-trigger queue if not paused and has queued items
		if !s.QueuePaused && s.Status != StatusGenerating {
			for _, item := range s.Queue {
				if item.Status == "queued" {
					go a.drainQueue(sm, info.ID)
					break
				}
			}
		}
	}
}
```

- [ ] **Step 2: Verify build compiles**

Run: `go build ./internal/api/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/api/app.go
git commit -m "feat: add queue recovery on app restart (reset executing items, auto-trigger)"
```

---

## Task 8: Regenerate Wails Bindings

**Files:**
- Generated: `frontend/bindings/monika/index.ts`

- [ ] **Step 1: Regenerate bindings**

Run:
```bash
wails3 generate bindings -ts
```

Then copy the barrel index:
```bash
node -e "require('fs').copyFileSync('build/barrel_index.ts','frontend/bindings/monika/index.ts')"
```

- [ ] **Step 2: Verify new methods appear in bindings**

Run: `grep -c "Queue" frontend/bindings/monika/index.ts`
Expected: A number > 0 (should find GetQueue, EditQueueItem, etc.)

- [ ] **Step 3: Commit**

```bash
git add frontend/bindings/
git commit -m "chore: regenerate Wails bindings for queue API methods"
```

---

## Task 9: Frontend Store — Types, State, Actions

**Files:**
- Modify: `frontend/src/store/index.ts`

- [ ] **Step 1: Add QueuedMessage type and state fields**

In `frontend/src/store/index.ts`, add the type after the `Message` interface (around line 89):

```ts
interface QueuedMessage {
    id: string
    text: string
    provider_id: string
    model: string
    status: 'queued' | 'executing' | 'error'
    error?: string
    created_at: number
}
```

Add state fields to `AppState` interface (after `bgTaskLogs` around line 236):

```ts
    sessionQueues: Record<string, QueuedMessage[]>
    queuePaused: Record<string, boolean>
```

Add action signatures to `AppState` (after existing actions):

```ts
    setQueue: (sessionId: string, items: QueuedMessage[]) => void
    updateQueueItem: (sessionId: string, itemId: string, changes: Partial<QueuedMessage>) => void
    removeQueueItem: (sessionId: string, itemId: string) => void
    reorderQueue: (sessionId: string, itemIds: string[]) => void
    toggleQueuePause: (sessionId: string, paused: boolean) => void
```

- [ ] **Step 2: Add initial state values**

In the `create()` function's initial state object, add:

```ts
    sessionQueues: {},
    queuePaused: {},
```

- [ ] **Step 3: Implement actions**

Add to the actions section of the store:

```ts
    setQueue: (sessionId, items) => set((state) => ({
        sessionQueues: { ...state.sessionQueues, [sessionId]: items },
    })),

    updateQueueItem: (sessionId, itemId, changes) => set((state) => {
        const queue = state.sessionQueues[sessionId] || []
        return {
            sessionQueues: {
                ...state.sessionQueues,
                [sessionId]: queue.map((item) =>
                    item.id === itemId ? { ...item, ...changes } : item
                ),
            },
        }
    }),

    removeQueueItem: (sessionId, itemId) => set((state) => {
        const queue = state.sessionQueues[sessionId] || []
        return {
            sessionQueues: {
                ...state.sessionQueues,
                [sessionId]: queue.filter((item) => item.id !== itemId),
            },
        }
    }),

    reorderQueue: (sessionId, itemIds) => set((state) => {
        const queue = state.sessionQueues[sessionId] || []
        const map = new Map(queue.map((item) => [item.id, item]))
        return {
            sessionQueues: {
                ...state.sessionQueues,
                [sessionId]: itemIds.map((id) => map.get(id)!).filter(Boolean),
            },
        }
    }),

    toggleQueuePause: (sessionId, paused) => set((state) => ({
        queuePaused: { ...state.queuePaused, [sessionId]: paused },
    })),
```

- [ ] **Step 4: Verify TypeScript compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 5: Commit**

```bash
git add frontend/src/store/index.ts
git commit -m "feat: add queue state and actions to Zustand store"
```

---

## Task 10: Frontend Event Listeners

**Files:**
- Modify: `frontend/src/store/index.ts` (in `setupWailsEvents` function)

- [ ] **Step 1: Add queue event handlers to setupWailsEvents**

Find the `setupWailsEvents` function in `frontend/src/store/index.ts` (search for `Events.On('stream'`). In the `processEvent` function's switch/if chain, add cases for queue events:

```ts
            case 'queue_updated': {
                try {
                    const items = data.content ? JSON.parse(data.content) : []
                    get().setQueue(data.session_id, items)
                } catch {}
                break
            }
            case 'queue_item_started': {
                try {
                    const item = data.content ? JSON.parse(data.content) : null
                    if (item) {
                        // Update item status to executing
                        get().updateQueueItem(data.session_id, item.id, { status: 'executing' })
                        // Add user message + assistant placeholder to chat
                        const userMsg = {
                            id: crypto.randomUUID(),
                            role: 'user' as const,
                            content: item.text,
                        }
                        const assistantMsg = {
                            id: crypto.randomUUID(),
                            role: 'assistant' as const,
                            content: '',
                            startedAt: Date.now(),
                        }
                        get().appendToSession(data.session_id, [userMsg, assistantMsg])
                        get().addGeneratingSession(data.session_id)
                    }
                } catch {}
                break
            }
            case 'queue_error': {
                try {
                    const info = data.content ? JSON.parse(data.content) : null
                    if (info) {
                        get().updateQueueItem(data.session_id, info.item_id, {
                            status: 'error',
                            error: info.error,
                        })
                        get().toggleQueuePause(data.session_id, true)
                    }
                } catch {}
                break
            }
```

Add these cases alongside the existing event type checks (e.g., after `case 'done':` or similar event routing).

- [ ] **Step 2: Verify TypeScript compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/store/index.ts
git commit -m "feat: add queue event listeners (queue_updated, queue_item_started, queue_error)"
```

---

## Task 11: Frontend ChatArea — Remove Busy Guard

**Files:**
- Modify: `frontend/src/components/Chat/ChatArea.tsx:180-210`

- [ ] **Step 1: Modify handleSend to allow sending while busy**

In `frontend/src/components/Chat/ChatArea.tsx`, replace the `handleSend` function (lines 180-210):

```tsx
    const handleSend = async (text: string) => {
        if (!text.trim()) return

        if (!projectPath || !sessionId) return

        if (!selectedProvider || !selectedModel) {
            useStore.getState().addMessage({ id: crypto.randomUUID(), role: 'error', content: 'No provider or model selected. Please choose a model from the toolbar.' })
            return
        }

        const store = useStore.getState()
        store.setMsgFilter('all')

        const isBusy = generatingSessionIds.includes(sessionId)

        if (isBusy) {
            // Message will be queued by backend — don't add optimistic UI
            try {
                await App.SendMessage(projectPath, sessionId, text, selectedProvider, selectedModel)
            } catch (err) {
                useStore.getState().addMessage({ id: crypto.randomUUID(), role: 'error', content: String(err) })
            }
        } else {
            // Not busy — add optimistic UI and send
            const userMsg = { id: crypto.randomUUID(), role: 'user' as const, content: text }
            const assistantMsg = { id: crypto.randomUUID(), role: 'assistant' as const, content: '', startedAt: Date.now() }
            store.appendToSession(sessionId, [userMsg, assistantMsg])
            store.addGeneratingSession(sessionId)

            try {
                await App.SendMessage(projectPath, sessionId, text, selectedProvider, selectedModel)
            } catch (err) {
                useStore.getState().addMessage({ id: crypto.randomUUID(), role: 'error', content: String(err) })
                store.removeGeneratingSession(sessionId)
                const currentMsgs = useStore.getState().sessionMessages[sessionId] || []
                useStore.getState().setMessages(currentMsgs.filter(m => m.id !== assistantMsg.id))
            }
        }
    }
```

- [ ] **Step 2: Verify TypeScript compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/Chat/ChatArea.tsx
git commit -m "feat: remove busy guard in ChatArea, allow queueing when generating"
```

---

## Task 12: QueueItem Component

**Files:**
- Create: `frontend/src/components/QueuePanel/QueueItem.tsx`

- [ ] **Step 1: Create QueueItem component**

Create `frontend/src/components/QueuePanel/QueueItem.tsx`:

```tsx
import { useState } from 'react'
import { useStore } from '../../store'

interface QueueItemProps {
    item: {
        id: string
        text: string
        provider_id: string
        model: string
        status: 'queued' | 'executing' | 'error'
        error?: string
        created_at: number
    }
    sessionId: string
    projectPath: string
    onDragStart: () => void
    onDragOver: (e: React.DragEvent) => void
    onDrop: () => void
}

export function QueueItem({ item, sessionId, projectPath, onDragStart, onDragOver, onDrop }: QueueItemProps) {
    const [editing, setEditing] = useState(false)
    const [editText, setEditText] = useState(item.text)
    const removeQueueItem = useStore((s) => s.removeQueueItem)

    const statusColor =
        item.status === 'executing' ? 'text-blue-400' :
        item.status === 'error' ? 'text-red-400' :
        'text-yellow-400'

    const statusIcon =
        item.status === 'executing' ? '🔄' :
        item.status === 'error' ? '❌' :
        '⏳'

    const handleSaveEdit = async () => {
        try {
            const { App } = await import('../../bindings/monika')
            await App.EditQueueItem(projectPath, sessionId, item.id, editText)
            setEditing(false)
        } catch (err) {
            console.error('Failed to edit queue item:', err)
        }
    }

    const handleCancel = async () => {
        try {
            const { App } = await import('../../bindings/monika')
            await App.CancelQueueItem(projectPath, sessionId, item.id)
            removeQueueItem(sessionId, item.id)
        } catch (err) {
            console.error('Failed to cancel queue item:', err)
        }
    }

    const handleRetry = async () => {
        try {
            const { App } = await import('../../bindings/monika')
            await App.RetryQueueItem(projectPath, sessionId, item.id)
        } catch (err) {
            console.error('Failed to retry queue item:', err)
        }
    }

    const handleSkip = async () => {
        try {
            const { App } = await import('../../bindings/monika')
            await App.SkipQueueItem(projectPath, sessionId, item.id)
            removeQueueItem(sessionId, item.id)
        } catch (err) {
            console.error('Failed to skip queue item:', err)
        }
    }

    const canEdit = item.status === 'queued' || item.status === 'error'
    const canDrag = item.status !== 'executing'

    return (
        <div
            className="flex items-start gap-2 rounded border border-zinc-700 bg-zinc-800/50 p-2 text-sm"
            draggable={canDrag}
            onDragStart={onDragStart}
            onDragOver={onDragOver}
            onDrop={onDrop}
        >
            {canDrag && <span className="cursor-grab text-zinc-500 select-none">⠿</span>}
            <span className={statusColor}>{statusIcon}</span>
            <div className="flex-1 min-w-0">
                {editing ? (
                    <div className="flex flex-col gap-1">
                        <textarea
                            className="w-full rounded bg-zinc-900 p-1 text-xs text-zinc-200 border border-zinc-600"
                            value={editText}
                            onChange={(e) => setEditText(e.target.value)}
                            rows={2}
                        />
                        <div className="flex gap-2">
                            <button className="text-xs text-green-400 hover:underline" onClick={handleSaveEdit}>保存</button>
                            <button className="text-xs text-zinc-400 hover:underline" onClick={() => { setEditText(item.text); setEditing(false) }}>取消</button>
                        </div>
                    </div>
                ) : (
                    <>
                        <p className="text-zinc-300 truncate">{item.text}</p>
                        {item.status === 'error' && item.error && (
                            <p className="text-xs text-red-400 mt-1">{item.error}</p>
                        )}
                        <div className="flex gap-2 mt-1">
                            {canEdit && (
                                <button className="text-xs text-blue-400 hover:underline" onClick={() => setEditing(true)}>编辑</button>
                            )}
                            {item.status === 'error' && (
                                <>
                                    <button className="text-xs text-green-400 hover:underline" onClick={handleRetry}>重试</button>
                                    <button className="text-xs text-yellow-400 hover:underline" onClick={handleSkip}>跳过</button>
                                </>
                            )}
                            <button className="text-xs text-red-400 hover:underline" onClick={handleCancel}>取消</button>
                        </div>
                    </>
                )}
            </div>
        </div>
    )
}
```

- [ ] **Step 2: Verify TypeScript compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/QueuePanel/QueueItem.tsx
git commit -m "feat: add QueueItem component with edit/cancel/retry/skip/drag"
```

---

## Task 13: QueuePanel Component

**Files:**
- Create: `frontend/src/components/QueuePanel/QueuePanel.tsx`

- [ ] **Step 1: Create QueuePanel component**

Create `frontend/src/components/QueuePanel/QueuePanel.tsx`:

```tsx
import { useState } from 'react'
import { useStore } from '../../store'
import { QueueItem } from './QueueItem'

const MAX_VISIBLE = 10

export function QueuePanel() {
    const projectPath = useStore((s) => s.projectPath)
    const activeSessionId = useStore((s) => s.activeSessionId)
    const sessionQueues = useStore((s) => s.sessionQueues)
    const queuePaused = useStore((s) => s.queuePaused)
    const reorderQueue = useStore((s) => s.reorderQueue)
    const toggleQueuePause = useStore((s) => s.toggleQueuePause)

    const [showAll, setShowAll] = useState(false)
    const [dragIndex, setDragIndex] = useState<number | null>(null)

    const queue = sessionQueues[activeSessionId] || []
    const paused = queuePaused[activeSessionId] || false
    const visibleItems = showAll ? queue : queue.slice(0, MAX_VISIBLE)

    const handlePauseToggle = async () => {
        try {
            const { App } = await import('../../bindings/monika')
            if (paused) {
                await App.ResumeQueue(projectPath, activeSessionId)
                toggleQueuePause(activeSessionId, false)
            } else {
                await App.PauseQueue(projectPath, activeSessionId)
                toggleQueuePause(activeSessionId, true)
            }
        } catch (err) {
            console.error('Failed to toggle pause:', err)
        }
    }

    const handleDragStart = (index: number) => () => {
        setDragIndex(index)
    }

    const handleDragOver = (e: React.DragEvent) => {
        e.preventDefault()
    }

    const handleDrop = (index: number) => () => {
        if (dragIndex === null || dragIndex === index) return
        const newOrder = [...queue]
        const [moved] = newOrder.splice(dragIndex, 1)
        newOrder.splice(index, 0, moved)
        const itemIds = newOrder.map((item) => item.id)

        reorderQueue(activeSessionId, itemIds)
        setDragIndex(null)

        // Persist to backend
        import('../../bindings/monika').then(({ App }) => {
            App.ReorderQueue(projectPath, activeSessionId, itemIds)
        })
    }

    if (queue.length === 0) {
        return (
            <div className="flex flex-col h-full p-3 text-zinc-500 text-sm">
                <h3 className="text-zinc-400 font-medium mb-2">消息队列</h3>
                <p className="text-xs">队列为空</p>
            </div>
        )
    }

    return (
        <div className="flex flex-col h-full p-3">
            <div className="flex items-center justify-between mb-2">
                <h3 className="text-zinc-400 font-medium text-sm">消息队列 ({queue.length})</h3>
                <button
                    className={`text-xs px-2 py-1 rounded ${paused ? 'bg-green-600 text-white' : 'bg-zinc-700 text-zinc-300'} hover:opacity-80`}
                    onClick={handlePauseToggle}
                >
                    {paused ? '▶ 恢复' : '⏸ 暂停'}
                </button>
            </div>
            <div className="flex-1 overflow-y-auto space-y-2">
                {visibleItems.map((item, index) => (
                    <QueueItem
                        key={item.id}
                        item={item}
                        sessionId={activeSessionId}
                        projectPath={projectPath}
                        onDragStart={handleDragStart(index)}
                        onDragOver={handleDragOver}
                        onDrop={handleDrop(index)}
                    />
                ))}
            </div>
            {queue.length > MAX_VISIBLE && (
                <button
                    className="text-xs text-blue-400 hover:underline mt-2 text-center"
                    onClick={() => setShowAll(!showAll)}
                >
                    {showAll ? '收起' : `查看全部 (${queue.length})`}
                </button>
            )}
        </div>
    )
}
```

- [ ] **Step 2: Verify TypeScript compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/QueuePanel/QueuePanel.tsx
git commit -m "feat: add QueuePanel component with drag-reorder, pause/resume, overflow"
```

---

## Task 14: Register QueuePanel in App.tsx

**Files:**
- Modify: `frontend/src/App.tsx:20-26`

- [ ] **Step 1: Import QueuePanel**

In `frontend/src/App.tsx`, add import after line 8:

```tsx
import { QueuePanel } from './components/QueuePanel/QueuePanel'
```

- [ ] **Step 2: Register in components map**

In `frontend/src/App.tsx`, add to the `components` map (line 20-26):

```tsx
const components: Record<string, React.FunctionComponent<IDockviewPanelProps>> = {
    chat: ChatArea,
    preview: PreviewPanel,
    files: FileTree,
    changes: ChangesList,
    session: SessionList,
    queue: QueuePanel,
}
```

- [ ] **Step 3: Verify TypeScript compiles**

Run: `cd frontend && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add frontend/src/App.tsx
git commit -m "feat: register QueuePanel in dockview components"
```

---

## Task 15: Integration Test — Manual Verification

- [ ] **Step 1: Start dev mode**

Run: `wails3 dev`

- [ ] **Step 2: Test basic queue flow**

1. Open a project, create a new session
2. Send a message
3. While the agent is generating, send another message
4. Verify: the second message appears in the QueuePanel sidebar (not rejected)
5. Wait for the first message to complete
6. Verify: the queued message automatically starts executing
7. Verify: the user message appears in chat when the queued item starts

- [ ] **Step 3: Test edit/reorder/cancel**

1. While agent is busy, queue 3 messages
2. Edit the text of a queued message → verify it updates
3. Drag to reorder → verify order changes
4. Cancel a queued message → verify it's removed

- [ ] **Step 4: Test pause/resume**

1. Queue messages while busy
2. Click pause button
3. Wait for current message to complete
4. Verify: next message does NOT auto-start
5. Click resume
6. Verify: next message starts executing

- [ ] **Step 5: Test error handling**

1. Queue a message with an invalid provider (simulate error)
2. Verify: queue pauses on error
3. Click retry → verify item resets to queued and queue resumes
4. Queue another failing message
5. Click skip → verify item is removed and queue continues

- [ ] **Step 6: Test cancel executing**

1. While a queued message is executing, click cancel
2. Verify: generation stops, queue pauses
3. Verify: user must manually resume

- [ ] **Step 7: Test restart recovery**

1. Queue several messages
2. Close the app while messages are queued
3. Reopen the app
4. Verify: queued messages are still present
5. Verify: if not paused, first message auto-starts
