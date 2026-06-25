package agent

import (
	"context"
	"testing"
	"time"

	"monika/internal/tool"
	"monika/pkg/engine"
)

func TestTaskRunner_Dispatch_AgentNotFound(t *testing.T) {
	registry := NewAgentRegistry([]Agent{
		{Name: "general", SystemPrompt: "test"},
	})
	runner := NewTaskRunner(registry, nil, nil, nil, nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	task := SubTask{
		ID:     "test-1",
		Type:   TaskSubtask,
		Agent:  "nonexistent",
		Prompt: "do something",
	}
	ch := runner.Dispatch(ctx, task, nil)

	var gotError bool
	for ev := range ch {
		if ev.Type == EventError && ev.Content == `agent "nonexistent" not found` {
			gotError = true
		}
	}
	if !gotError {
		t.Error("expected error for nonexistent agent")
	}
}

func TestTaskRunner_Dispatch_Cancellation(t *testing.T) {
	registry := NewAgentRegistry([]Agent{
		{Name: "general", SystemPrompt: "test"},
	})
	waitCh := make(chan struct{})
	prov := staticProvider(nil, nil)
	prov.streamFn = func(ctx context.Context, req engine.ChatRequest) (<-chan engine.ChatEvent, error) {
		evCh := make(chan engine.ChatEvent)
		go func() {
			<-waitCh
			close(evCh)
		}()
		return evCh, nil
	}
	runner := NewTaskRunner(registry, prov, nil, tool.NewRegistry(), nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	task := SubTask{
		ID:     "test-2",
		Type:   TaskSubtask,
		Agent:  "general",
		Prompt: "do something",
	}
	ch := runner.Dispatch(ctx, task, nil)

	cancel()
	close(waitCh)

	// Channel should close without hanging (cancellation propagates correctly)
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("dispatch channel did not close after cancellation")
	}
}
