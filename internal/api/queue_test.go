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
