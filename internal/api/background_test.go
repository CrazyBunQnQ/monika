package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBackgroundTaskManagerStartStop(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewBackgroundTaskManager(filepath.Join(tmpDir, "logs"))
	defer mgr.Cleanup()

	id, err := mgr.Start("echo hello", ".")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty task ID")
	}

	tasks := mgr.List()
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Status != BgTaskRunning {
		t.Fatalf("expected status running, got %s", tasks[0].Status)
	}

	time.Sleep(2 * time.Second)

	tasks = mgr.List()
	if tasks[0].Status != BgTaskExited {
		t.Fatalf("expected status exited, got %s", tasks[0].Status)
	}

	logs, err := mgr.Logs(id, 10)
	if err != nil {
		t.Fatalf("Logs failed: %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("expected at least 1 log line")
	}
	if !strings.Contains(logs[0], "hello") {
		t.Fatalf("expected log to contain 'hello', got %s", logs[0])
	}
}

func TestBackgroundTaskManagerStop(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewBackgroundTaskManager(filepath.Join(tmpDir, "logs"))
	defer mgr.Cleanup()

	id, err := mgr.Start("ping -n 30 127.0.0.1", ".")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	tasks := mgr.List()
	if tasks[0].Status != BgTaskRunning {
		t.Fatalf("expected running, got %s", tasks[0].Status)
	}

	err = mgr.Stop(id)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	tasks = mgr.List()
	if tasks[0].Status != BgTaskStopped {
		t.Fatalf("expected stopped, got %s", tasks[0].Status)
	}

	err = mgr.Stop("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent task")
	}
}

func TestBackgroundTaskManagerLogsNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewBackgroundTaskManager(filepath.Join(tmpDir, "logs"))
	defer mgr.Cleanup()

	_, err := mgr.Logs("nonexistent", 10)
	if err == nil {
		t.Fatal("expected error for non-existent task")
	}
}

func TestBackgroundTaskManagerLogLines(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewBackgroundTaskManager(filepath.Join(tmpDir, "logs"))
	defer mgr.Cleanup()

	id, err := mgr.Start("echo line1 && echo line2 && echo line3", ".")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(2 * time.Second)

	// Read all lines from start
	lines, err := mgr.LogLines(id, 0, 100)
	if err != nil {
		t.Fatalf("LogLines failed: %v", err)
	}
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "line1") {
		t.Fatalf("expected first line to contain 'line1', got %s", lines[0])
	}

	// Read last 2 lines
	lines, err = mgr.LogLines(id, -2, 100)
	if err != nil {
		t.Fatalf("LogLines tail failed: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines from tail, got %d", len(lines))
	}

	// Read with limit
	lines, err = mgr.LogLines(id, 0, 1)
	if err != nil {
		t.Fatalf("LogLines with limit failed: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line with limit=1, got %d", len(lines))
	}
}

func TestBackgroundTaskManagerSubscribe(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewBackgroundTaskManager(filepath.Join(tmpDir, "logs"))
	defer mgr.Cleanup()

	ch := mgr.Subscribe()

	id, err := mgr.Start("echo test-subscribe", ".")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	select {
	case ev := <-ch:
		if ev.Type != BgEventStarted {
			t.Fatalf("expected BgEventStarted, got %s", ev.Type)
		}
		if ev.TaskID != id {
			t.Fatalf("expected task ID %s, got %s", id, ev.TaskID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for started event")
	}

	timeout := time.After(5 * time.Second)
	for {
		select {
		case ev := <-ch:
			if ev.Type == BgEventExited {
				if ev.TaskID != id {
					t.Fatalf("expected task ID %s, got %s", id, ev.TaskID)
				}
				return
			}
			if ev.Type == BgEventLogUpdate {
				if ev.LineCount <= 0 {
					t.Fatal("expected positive line_count in log_update event")
				}
			}
		case <-timeout:
			t.Fatal("timed out waiting for exited event")
		}
	}
}

func TestCleanOldLogs(t *testing.T) {
	tmpDir := t.TempDir()
	logDir := filepath.Join(tmpDir, "logs")
	os.MkdirAll(logDir, 0755)

	// Create a fake old log file
	oldFile := filepath.Join(logDir, "old.log")
	os.WriteFile(oldFile, []byte("old log"), 0644)

	// Set modtime to 8 days ago
	oldTime := time.Now().Add(-8 * 24 * time.Hour)
	os.Chtimes(oldFile, oldTime, oldTime)

	// Create a recent file
	newFile := filepath.Join(logDir, "new.log")
	os.WriteFile(newFile, []byte("new log"), 0644)

	mgr := NewBackgroundTaskManager(logDir)
	mgr.CleanOldLogs()

	// Old file should be deleted
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Fatal("expected old log file to be deleted")
	}
	// New file should remain
	if _, err := os.Stat(newFile); err != nil {
		t.Fatal("expected new log file to remain")
	}
}
