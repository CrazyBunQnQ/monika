package api

import (
	"strings"
	"testing"
	"time"
)

func TestBackgroundTaskManagerStartStop(t *testing.T) {
	mgr := NewBackgroundTaskManager()
	defer mgr.Cleanup()

	shell, shellArg := "cmd", "/C"
	id, err := mgr.Start("echo hello", ".", shell, shellArg)
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
	if tasks[0].ID != id {
		t.Fatalf("expected task ID %s, got %s", id, tasks[0].ID)
	}
	if tasks[0].Status != BgTaskRunning {
		t.Fatalf("expected status running, got %s", tasks[0].Status)
	}
	if tasks[0].Command != "echo hello" {
		t.Fatalf("expected command 'echo hello', got %s", tasks[0].Command)
	}

	// Wait for the process to finish (echo is fast)
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
	mgr := NewBackgroundTaskManager()
	defer mgr.Cleanup()

	// Start a long-running process: ping with high count
	shell, shellArg := "cmd", "/C"
	id, err := mgr.Start("ping -n 30 127.0.0.1", ".", shell, shellArg)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Give it a moment to start
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

	// Stop non-existent task
	err = mgr.Stop("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent task")
	}
}

func TestBackgroundTaskManagerLogsNonExistent(t *testing.T) {
	mgr := NewBackgroundTaskManager()
	defer mgr.Cleanup()

	_, err := mgr.Logs("nonexistent", 10)
	if err == nil {
		t.Fatal("expected error for non-existent task")
	}
}

func TestRingBuffer(t *testing.T) {
	rb := newRingBuffer(3)

	rb.Write("a")
	rb.Write("b")
	rb.Write("c")

	lines := rb.LastN(3)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0] != "a" || lines[1] != "b" || lines[2] != "c" {
		t.Fatalf("expected [a b c], got %v", lines)
	}

	// Overflow: write one more
	rb.Write("d")
	lines = rb.LastN(3)
	if lines[0] != "b" || lines[1] != "c" || lines[2] != "d" {
		t.Fatalf("expected [b c d], got %v", lines)
	}

	// Request more than available
	lines = rb.LastN(10)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	// Empty buffer
	rb2 := newRingBuffer(5)
	lines = rb2.LastN(5)
	if len(lines) != 0 {
		t.Fatalf("expected 0 lines from empty buffer, got %d", len(lines))
	}
}

func TestBackgroundTaskManagerSubscribe(t *testing.T) {
	mgr := NewBackgroundTaskManager()
	defer mgr.Cleanup()

	ch := mgr.Subscribe()

	shell, shellArg := "cmd", "/C"
	id, err := mgr.Start("echo test-subscribe", ".", shell, shellArg)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Should receive a started event
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

	// Wait for exit
	select {
	case ev := <-ch:
		if ev.Type != BgEventLog {
			// might get log lines first, that's fine
		}
	case <-time.After(5 * time.Second):
	}

	// Wait for exit event
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
		case <-timeout:
			t.Fatal("timed out waiting for exited event")
		}
	}
}
