package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractionQueueEnqueueAndDequeue(t *testing.T) {
	dir := t.TempDir()
	q, err := NewExtractionQueue(dir)
	if err != nil {
		t.Fatalf("NewExtractionQueue: %v", err)
	}

	if q.Len() != 0 {
		t.Fatalf("expected empty queue, got Len=%d", q.Len())
	}

	items := []ExtractionItem{
		{ID: "a", SessionID: "s1", Transcript: "t1"},
		{ID: "b", SessionID: "s2", Transcript: "t2"},
	}
	for _, item := range items {
		if err := q.EnqueueOrReplace(item); err != nil {
			t.Fatalf("EnqueueOrReplace: %v", err)
		}
	}
	if q.Len() != 2 {
		t.Fatalf("expected Len=2, got %d", q.Len())
	}

	first, ok := q.Dequeue()
	if !ok || first.SessionID != "s1" {
		t.Fatalf("expected s1 first, got %+v ok=%v", first, ok)
	}
	if first.Status != "processing" {
		t.Fatalf("expected status=processing after Dequeue, got %s", first.Status)
	}
	second, ok := q.Dequeue()
	if !ok || second.SessionID != "s2" {
		t.Fatalf("expected s2 second, got %+v ok=%v", second, ok)
	}
	_, ok = q.Dequeue()
	if ok {
		t.Fatal("expected Dequeue to return false on empty queue")
	}
}

func TestExtractionQueueReplaceSameSession(t *testing.T) {
	dir := t.TempDir()
	q, _ := NewExtractionQueue(dir)

	if err := q.EnqueueOrReplace(ExtractionItem{ID: "a", SessionID: "s1", Transcript: "old"}); err != nil {
		t.Fatalf("first enqueue: %v", err)
	}
	if err := q.EnqueueOrReplace(ExtractionItem{ID: "b", SessionID: "s1", Transcript: "new"}); err != nil {
		t.Fatalf("second enqueue: %v", err)
	}
	if q.Len() != 1 {
		t.Fatalf("expected Len=1 after replace, got %d", q.Len())
	}
	item, ok := q.Dequeue()
	if !ok || item.Transcript != "new" {
		t.Fatalf("expected 'new' transcript, got %+v", item)
	}
}

func TestExtractionQueueDifferentSessionsNotReplaced(t *testing.T) {
	dir := t.TempDir()
	q, _ := NewExtractionQueue(dir)

	q.EnqueueOrReplace(ExtractionItem{SessionID: "s1", Transcript: "t1"})
	q.EnqueueOrReplace(ExtractionItem{SessionID: "s2", Transcript: "t2"})
	q.EnqueueOrReplace(ExtractionItem{SessionID: "s1", Transcript: "t1-updated"})

	if q.Len() != 2 {
		t.Fatalf("expected Len=2, got %d", q.Len())
	}
}

func TestExtractionQueuePersistAndReload(t *testing.T) {
	dir := t.TempDir()
	q1, _ := NewExtractionQueue(dir)
	q1.EnqueueOrReplace(ExtractionItem{SessionID: "s1", Transcript: "t1"})
	q1.EnqueueOrReplace(ExtractionItem{SessionID: "s2", Transcript: "t2"})

	queuePath := filepath.Join(dir, ".monika", "extraction_queue.json")
	if _, err := os.Stat(queuePath); err != nil {
		t.Fatalf("queue file not written: %v", err)
	}

	q2, err := NewExtractionQueue(dir)
	if err != nil {
		t.Fatalf("reload NewExtractionQueue: %v", err)
	}
	if q2.Len() != 2 {
		t.Fatalf("expected Len=2 after reload, got %d", q2.Len())
	}
	item, _ := q2.Dequeue()
	if item.SessionID != "s1" {
		t.Fatalf("expected s1 after reload, got %s", item.SessionID)
	}
}

func TestExtractionQueueAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	q, _ := NewExtractionQueue(dir)
	q.EnqueueOrReplace(ExtractionItem{SessionID: "s1", Transcript: "t1"})

	tmpPath := filepath.Join(dir, ".monika", "extraction_queue.json.tmp")
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Fatalf("tmp file should not exist after save, got err=%v", err)
	}
}

func TestExtractionQueueCompleteRemovesItem(t *testing.T) {
	dir := t.TempDir()
	q, _ := NewExtractionQueue(dir)
	q.EnqueueOrReplace(ExtractionItem{ID: "a", SessionID: "s1", Transcript: "t1"})
	q.EnqueueOrReplace(ExtractionItem{ID: "b", SessionID: "s2", Transcript: "t2"})

	item, _ := q.Dequeue()
	q.Complete(item.ID)

	if q.Len() != 1 {
		t.Fatalf("expected Len=1 after Complete, got %d", q.Len())
	}
	next, _ := q.Dequeue()
	if next.SessionID != "s2" {
		t.Fatalf("expected s2 remaining, got %s", next.SessionID)
	}
}

func TestExtractionQueueCrashRecovery(t *testing.T) {
	dir := t.TempDir()
	q1, _ := NewExtractionQueue(dir)
	q1.EnqueueOrReplace(ExtractionItem{ID: "a", SessionID: "s1", Transcript: "t1"})

	item, _ := q1.Dequeue()
	if item.Status != "processing" {
		t.Fatalf("expected processing status, got %s", item.Status)
	}

	q2, err := NewExtractionQueue(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if q2.Len() != 1 {
		t.Fatalf("expected Len=1 after crash recovery, got %d", q2.Len())
	}
	recovered, _ := q2.Dequeue()
	if recovered.SessionID != "s1" {
		t.Fatalf("expected s1 recovered, got %s", recovered.SessionID)
	}
}

func TestExtractionQueueLenSkipsProcessing(t *testing.T) {
	dir := t.TempDir()
	q, _ := NewExtractionQueue(dir)
	q.EnqueueOrReplace(ExtractionItem{ID: "a", SessionID: "s1", Transcript: "t1"})
	q.EnqueueOrReplace(ExtractionItem{ID: "b", SessionID: "s2", Transcript: "t2"})

	q.Dequeue()
	if q.Len() != 1 {
		t.Fatalf("expected Len=1 (one processing), got %d", q.Len())
	}
}
