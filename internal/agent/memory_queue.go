package agent

import (
	"context"
	"sync"
)

// MemoryQueue receives notifications about memory changes from memory tools,
// so the change takes effect immediately in the current AgentLoop run.
// Pending notes are drained by buildEntryPrefix and prepended to the next
// user message as <memory-update> blocks. This does NOT touch the system
// prompt, preserving the DeepSeek prefix cache.
type MemoryQueue interface {
	QueueMemory(note string)
	DrainPending() []string
}

type memoryQueueImpl struct {
	pending []string
	mu      sync.Mutex
}

func NewMemoryQueue() MemoryQueue {
	return &memoryQueueImpl{}
}

func (q *memoryQueueImpl) QueueMemory(note string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.pending = append(q.pending, note)
}

func (q *memoryQueueImpl) DrainPending() []string {
	q.mu.Lock()
	defer q.mu.Unlock()
	notes := q.pending
	q.pending = nil
	return notes
}

type memoryQueueKey struct{}

func WithMemoryQueueInContext(ctx context.Context, q MemoryQueue) context.Context {
	return context.WithValue(ctx, memoryQueueKey{}, q)
}

func MemoryQueueFromContext(ctx context.Context) (MemoryQueue, bool) {
	q, ok := ctx.Value(memoryQueueKey{}).(MemoryQueue)
	return q, ok && q != nil
}
