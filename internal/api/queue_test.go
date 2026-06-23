package api

import (
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
