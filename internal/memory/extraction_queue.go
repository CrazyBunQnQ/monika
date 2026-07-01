package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ExtractionItem struct {
	ID         string `json:"id"`
	SessionID  string `json:"session_id"`
	Transcript string `json:"transcript"`
	CreatedAt  string `json:"created_at"`
}

type ExtractionQueue struct {
	mu    sync.Mutex
	path  string
	items []ExtractionItem
}

func NewExtractionQueue(dir string) (*ExtractionQueue, error) {
	q := &ExtractionQueue{
		path: filepath.Join(dir, ".monika", "extraction_queue.json"),
	}
	if err := q.load(); err != nil {
		return nil, err
	}
	return q, nil
}

func (q *ExtractionQueue) EnqueueOrReplace(item ExtractionItem) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if item.CreatedAt == "" {
		item.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	filtered := q.items[:0]
	for _, existing := range q.items {
		if existing.SessionID != item.SessionID {
			filtered = append(filtered, existing)
		}
	}
	q.items = append(filtered, item)
	return q.saveLocked()
}

func (q *ExtractionQueue) Dequeue() (ExtractionItem, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return ExtractionItem{}, false
	}
	item := q.items[0]
	q.items = q.items[1:]
	_ = q.saveLocked()
	return item, true
}

func (q *ExtractionQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

func (q *ExtractionQueue) load() error {
	data, err := os.ReadFile(q.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read extraction queue: %w", err)
	}
	return json.Unmarshal(data, &q.items)
}

func (q *ExtractionQueue) saveLocked() error {
	data, err := json.MarshalIndent(q.items, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal extraction queue: %w", err)
	}
	dir := filepath.Dir(q.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create extraction queue dir: %w", err)
	}
	tmpPath := q.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write extraction queue: %w", err)
	}
	if err := os.Rename(tmpPath, q.path); err != nil {
		return fmt.Errorf("rename extraction queue: %w", err)
	}
	return nil
}
