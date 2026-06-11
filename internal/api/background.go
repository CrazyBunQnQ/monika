package api

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"

	"monika/internal/tool/builtin"
)

// BgTaskStatus represents the current state of a background task.
type BgTaskStatus string

const (
	BgTaskRunning BgTaskStatus = "running"
	BgTaskStopped BgTaskStatus = "stopped"
	BgTaskExited  BgTaskStatus = "exited"
)

// BgTaskEventType represents the type of a background task event.
type BgTaskEventType string

const (
	BgEventStarted BgTaskEventType = "started"
	BgEventLog     BgTaskEventType = "log"
	BgEventStopped BgTaskEventType = "stopped"
	BgEventExited  BgTaskEventType = "exited"
)

// BgTaskEvent is emitted for background task lifecycle and log events.
type BgTaskEvent struct {
	Type     BgTaskEventType `json:"type"`
	TaskID   string          `json:"task_id"`
	Command  string          `json:"command,omitempty"`
	WorkDir  string          `json:"work_dir,omitempty"`
	PID      int             `json:"pid,omitempty"`
	Status   BgTaskStatus    `json:"status,omitempty"`
	ExitCode int             `json:"exit_code,omitempty"`
	LogLine  string          `json:"log_line,omitempty"`
}

// BgTaskInfo is a snapshot of a background task's current state.
type BgTaskInfo struct {
	ID        string       `json:"id"`
	Command   string       `json:"command"`
	WorkDir   string       `json:"work_dir"`
	PID       int          `json:"pid"`
	Status    BgTaskStatus `json:"status"`
	ExitCode  int          `json:"exit_code,omitempty"`
	StartedAt time.Time    `json:"started_at"`
}

// ringBuffer is a fixed-size circular buffer for log lines.
type ringBuffer struct {
	mu    sync.Mutex
	buf   []string
	size  int
	head  int // next write position
	count int // number of items written (up to size)
}

func newRingBuffer(size int) *ringBuffer {
	return &ringBuffer{
		buf:  make([]string, size),
		size: size,
	}
}

func (rb *ringBuffer) Write(line string) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.buf[rb.head] = line
	rb.head = (rb.head + 1) % rb.size
	if rb.count < rb.size {
		rb.count++
	}
}

func (rb *ringBuffer) LastN(n int) []string {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	if n > rb.count {
		n = rb.count
	}
	if n == 0 {
		return nil
	}
	result := make([]string, 0, n)
	start := (rb.head - n + rb.size) % rb.size
	for i := 0; i < n; i++ {
		idx := (start + i) % rb.size
		result = append(result, rb.buf[idx])
	}
	return result
}

// bgTask holds internal state for a background task.
type bgTask struct {
	info    BgTaskInfo
	ringBuf *ringBuffer
	cancel  context.CancelFunc
}

const (
	defaultRingBufferSize  = 500
	bgSubscriberBufferSize = 256
)

// BackgroundTaskManager manages background processes with log buffering and event broadcasting.
type BackgroundTaskManager struct {
	mu          sync.Mutex
	tasks       map[string]*bgTask
	subscribers map[chan BgTaskEvent]struct{}
	engine      *builtin.ShellEngine
}

// NewBackgroundTaskManager creates a new BackgroundTaskManager.
func NewBackgroundTaskManager() *BackgroundTaskManager {
	return &BackgroundTaskManager{
		tasks:       make(map[string]*bgTask),
		subscribers: make(map[chan BgTaskEvent]struct{}),
		engine:      builtin.NewShellEngine(),
	}
}

// Start begins a background process and returns its task ID.
func (m *BackgroundTaskManager) Start(command, workdir string) (string, error) {
	id := uuid.New().String()
	ringBuf := newRingBuffer(defaultRingBufferSize)

	ctx, cancel := context.WithCancel(context.Background())

	onLine := func(line string) {
		line = stripANSI(line)
		ringBuf.Write(line)
		m.emit(BgTaskEvent{
			Type:    BgEventLog,
			TaskID:  id,
			LogLine: line,
		})
	}

	bgCancel, exitCh, err := m.engine.StartBackground(ctx, command, workdir, os.Environ(), onLine)
	if err != nil {
		cancel()
		return "", fmt.Errorf("start process: %w", err)
	}

	task := &bgTask{
		info: BgTaskInfo{
			ID:        id,
			Command:   command,
			WorkDir:   workdir,
			Status:    BgTaskRunning,
			StartedAt: time.Now(),
		},
		ringBuf: ringBuf,
		cancel: func() {
			bgCancel()
			cancel()
		},
	}

	m.mu.Lock()
	m.tasks[id] = task
	m.mu.Unlock()

	m.emit(BgTaskEvent{
		Type:    BgEventStarted,
		TaskID:  id,
		Command: command,
		WorkDir: workdir,
		Status:  BgTaskRunning,
	})

	go m.waitExit(id, exitCh)

	return id, nil
}

// Stop kills a running background task.
func (m *BackgroundTaskManager) Stop(taskID string) error {
	m.mu.Lock()
	task, ok := m.tasks[taskID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("task %s not found", taskID)
	}
	if task.info.Status != BgTaskRunning {
		m.mu.Unlock()
		return fmt.Errorf("task %s is not running", taskID)
	}
	task.info.Status = BgTaskStopped
	m.mu.Unlock()

	// Cancel the context — mvdan/sh will propagate to child processes.
	task.cancel()

	task.cancel()

	m.emit(BgTaskEvent{
		Type:    BgEventStopped,
		TaskID:  taskID,
		Command: task.info.Command,
		WorkDir: task.info.WorkDir,
		PID:     task.info.PID,
		Status:  BgTaskStopped,
	})

	return nil
}

// Logs returns the last N log lines from a task's ring buffer.
func (m *BackgroundTaskManager) Logs(taskID string, lines int) ([]string, error) {
	m.mu.Lock()
	task, ok := m.tasks[taskID]
	m.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("task %s not found", taskID)
	}
	return task.ringBuf.LastN(lines), nil
}

// List returns a snapshot of all background tasks.
func (m *BackgroundTaskManager) List() []BgTaskInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]BgTaskInfo, 0, len(m.tasks))
	for _, task := range m.tasks {
		result = append(result, task.info)
	}
	return result
}

// Subscribe returns a channel that receives background task events.
func (m *BackgroundTaskManager) Subscribe() <-chan BgTaskEvent {
	ch := make(chan BgTaskEvent, bgSubscriberBufferSize)
	m.mu.Lock()
	m.subscribers[ch] = struct{}{}
	m.mu.Unlock()
	return ch
}

// Cleanup stops all running tasks and closes subscriber channels.
func (m *BackgroundTaskManager) Cleanup() {
	m.mu.Lock()
	ids := make([]string, 0)
	for id, task := range m.tasks {
		if task.info.Status == BgTaskRunning {
			ids = append(ids, id)
		}
	}
	m.mu.Unlock()

	for _, id := range ids {
		m.Stop(id)
	}

	m.mu.Lock()
	for ch := range m.subscribers {
		close(ch)
		delete(m.subscribers, ch)
	}
	m.mu.Unlock()
}

func (m *BackgroundTaskManager) waitExit(taskID string, exitCh <-chan int) {
	code := <-exitCh

	m.mu.Lock()
	task, ok := m.tasks[taskID]
	if !ok {
		m.mu.Unlock()
		return
	}
	task.info.ExitCode = code
	if task.info.Status == BgTaskRunning {
		task.info.Status = BgTaskExited
	}
	m.mu.Unlock()

	m.emit(BgTaskEvent{
		Type:     BgEventExited,
		TaskID:   taskID,
		PID:      task.info.PID,
		Status:   task.info.Status,
		ExitCode: task.info.ExitCode,
	})
}

func (m *BackgroundTaskManager) emit(ev BgTaskEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for ch := range m.subscribers {
		select {
		case ch <- ev:
		default:
		}
	}
}

// unused: kept for reference — the old readLogs pattern is no longer needed
// since mvdan/sh streams lines via the onLine callback.
var _ = bufio.Scanner{}
var _ = io.ReadCloser(nil)
