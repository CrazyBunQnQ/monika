package api

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"

	"monika/internal/tool/builtin"
)

type BgTaskStatus = builtin.BgTaskStatus

const (
	BgTaskRunning = builtin.BgTaskRunning
	BgTaskStopped = builtin.BgTaskStopped
	BgTaskExited  = builtin.BgTaskExited
)

type BgTaskEventType string

const (
	BgEventStarted   BgTaskEventType = "started"
	BgEventLogUpdate BgTaskEventType = "log_update"
	BgEventStopped   BgTaskEventType = "stopped"
	BgEventExited    BgTaskEventType = "exited"
)

type BgTaskEvent struct {
	Type      BgTaskEventType `json:"type"`
	TaskID    string          `json:"task_id"`
	Command   string          `json:"command,omitempty"`
	WorkDir   string          `json:"work_dir,omitempty"`
	PID       int             `json:"pid,omitempty"`
	Status    BgTaskStatus    `json:"status,omitempty"`
	ExitCode  int             `json:"exit_code,omitempty"`
	LineCount int             `json:"line_count,omitempty"`
}

type BgTaskInfo = builtin.BgTaskInfo

type bgTask struct {
	mu        sync.Mutex
	info      BgTaskInfo
	logFile   *os.File
	logPath   string
	lineCount int
	cancel    context.CancelFunc
}

const (
	bgSubscriberBufferSize = 256
	bgLogRetention         = 7 * 24 * time.Hour
)

type BackgroundTaskManager struct {
	mu          sync.Mutex
	tasks       map[string]*bgTask
	subscribers map[chan BgTaskEvent]struct{}
	engine      *builtin.ShellEngine
	logDir      string
}

func NewBackgroundTaskManager(logDir string) *BackgroundTaskManager {
	return &BackgroundTaskManager{
		tasks:       make(map[string]*bgTask),
		subscribers: make(map[chan BgTaskEvent]struct{}),
		engine:      builtin.NewShellEngine(),
		logDir:      logDir,
	}
}

func (m *BackgroundTaskManager) SetLogDir(dir string) {
	m.mu.Lock()
	m.logDir = dir
	m.mu.Unlock()
	os.MkdirAll(dir, 0755)
}

func (m *BackgroundTaskManager) Start(command, workdir string) (string, error) {
	id := uuid.New().String()

	if err := os.MkdirAll(m.logDir, 0755); err != nil {
		return "", fmt.Errorf("create log dir: %w", err)
	}

	logPath := filepath.Join(m.logDir, id+".log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return "", fmt.Errorf("create log file: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	task := &bgTask{
		info: BgTaskInfo{
			ID:        id,
			Command:   command,
			WorkDir:   workdir,
			Status:    BgTaskRunning,
			StartedAt: time.Now(),
		},
		logFile: logFile,
		logPath: logPath,
	}

	onLine := func(line string) {
		line = stripANSI(line)
		fmt.Fprintln(logFile, line)
		task.mu.Lock()
		task.lineCount++
		count := task.lineCount
		task.mu.Unlock()
		m.emit(BgTaskEvent{
			Type:      BgEventLogUpdate,
			TaskID:    id,
			LineCount: count,
		})
	}

	bgCancel, exitCh, err := m.engine.StartBackground(ctx, command, workdir, os.Environ(), onLine)
	if err != nil {
		cancel()
		logFile.Close()
		return "", fmt.Errorf("start process: %w", err)
	}

	task.cancel = func() {
		bgCancel()
		cancel()
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

func (m *BackgroundTaskManager) Logs(taskID string, lines int) ([]string, error) {
	return m.LogLines(taskID, -lines, lines)
}

func (m *BackgroundTaskManager) LogLines(taskID string, offset, limit int) ([]string, error) {
	m.mu.Lock()
	task, ok := m.tasks[taskID]
	m.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	f, err := os.Open(task.logPath)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()

	task.mu.Lock()
	totalLines := task.lineCount
	task.mu.Unlock()

	var skipCount int
	if offset < 0 {
		skipCount = totalLines + offset
		if skipCount < 0 {
			skipCount = 0
		}
	} else {
		skipCount = offset
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	result := make([]string, 0, limit)
	lineIdx := 0
	count := 0

	for scanner.Scan() {
		if lineIdx < skipCount {
			lineIdx++
			continue
		}
		if count >= limit {
			break
		}
		result = append(result, scanner.Text())
		count++
		lineIdx++
	}

	return result, nil
}

func (m *BackgroundTaskManager) List() []BgTaskInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]BgTaskInfo, 0, len(m.tasks))
	for _, task := range m.tasks {
		result = append(result, task.info)
	}
	return result
}

func (m *BackgroundTaskManager) Subscribe() <-chan BgTaskEvent {
	ch := make(chan BgTaskEvent, bgSubscriberBufferSize)
	m.mu.Lock()
	m.subscribers[ch] = struct{}{}
	m.mu.Unlock()
	return ch
}

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
	for _, task := range m.tasks {
		if task.logFile != nil {
			task.logFile.Close()
		}
	}
	for ch := range m.subscribers {
		close(ch)
		delete(m.subscribers, ch)
	}
	m.mu.Unlock()
}

func (m *BackgroundTaskManager) CleanOldLogs() {
	entries, err := os.ReadDir(m.logDir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-bgLogRetention)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".log" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(m.logDir, entry.Name()))
		}
	}
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
