package agent

import (
	"context"
	"fmt"
	"strings"

	"monika/internal/tool"
	"monika/pkg/engine"
)


// ChildSession holds the result of a completed child agent run.
type ChildSession struct {
	Messages   []engine.ChatMessage
	Agent      string
	ParentID   string
	Title      string
	TokenCount int64
}

type TaskRunner struct {
	registry   *AgentRegistry
	provider   engine.ProviderEngine            // default / fallback provider
	providers  map[string]engine.ProviderEngine // all available providers, keyed by ID
	tools      *tool.ToolRegistry
	onStart    func(task SubTask, agentName string) // called before child runs
	onComplete func(task SubTask, child *ChildSession)
}

func NewTaskRunner(registry *AgentRegistry, provider engine.ProviderEngine, providers map[string]engine.ProviderEngine, tools *tool.ToolRegistry, onStart func(task SubTask, agentName string), onComplete func(task SubTask, child *ChildSession)) *TaskRunner {
	return &TaskRunner{
		registry:   registry,
		provider:   provider,
		providers:  providers,
		tools:      tools,
		onStart:    onStart,
		onComplete: onComplete,
	}
}

func (r *TaskRunner) Dispatch(ctx context.Context, task SubTask, parent *AgentLoop) <-chan Event {
	resultCh := make(chan Event, 128)

	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				resultCh <- Event{Type: EventError, Content: fmt.Sprintf("child panic: %v", rec)}
			}
			close(resultCh)
		}()


		ag, ok := r.registry.Get(task.Agent)
		if !ok {
			resultCh <- Event{Type: EventError, Content: fmt.Sprintf("agent %q not found", task.Agent)}
			return
		}

		// Notify before running so the frontend can open the tab immediately
		if r.onStart != nil {
			r.onStart(task, ag.Name)
		}

		opts := []LoopOption{
			WithAgent(ag),
			WithParent(parent),
			WithSessionID(task.SessionID),
		}
		// Resolve projectDir: task explicit > parent inheritance
		if task.ProjectDir != "" {
			opts = append(opts, WithProjectDir(task.ProjectDir))
		} else if parent != nil && parent.projectDir != "" {
			opts = append(opts, WithProjectDir(parent.projectDir))
		}
		// Replace {{WorkingDirectory}} in the agent system prompt with the actual project directory.
		if ag.SystemPrompt != "" {
			resolvedDir := task.ProjectDir
			if resolvedDir == "" && parent != nil {
				resolvedDir = parent.projectDir
			}
			if resolvedDir != "" {
				normalizedDir := strings.ReplaceAll(resolvedDir, "\\", "/")
				opts = append(opts, WithSystemPrompt(strings.ReplaceAll(ag.SystemPrompt, "{{WorkingDirectory}}", normalizedDir)))
			}
		}
		// Resolve model: agent explicit > task override > parent inheritance
		if ag.Model == "" {
			if task.Model != "" {
				opts = append(opts, WithModel(task.Model))
			} else if parent != nil && parent.model != "" {
				opts = append(opts, WithModel(parent.model))
			}
		}
		// Resolve provider: agent explicit > task override > parent inheritance
		if ag.Provider == "" {
			if task.Provider != "" {
				opts = append(opts, WithProvider(task.Provider))
			} else if parent != nil && parent.providerID != "" {
				opts = append(opts, WithProvider(parent.providerID))
			}
		}
		// Resolve context/output limits: parent > task explicit
		if parent != nil && parent.modelContextLimit > 0 {
			opts = append(opts, WithContextLimit(parent.modelContextLimit), WithOutputLimit(parent.modelOutputLimit))
		} else if task.ContextLimit > 0 {
			opts = append(opts, WithContextLimit(task.ContextLimit), WithOutputLimit(task.OutputLimit))
		}
		// Resolve provider engine: task explicit > parent inheritance > default
		provEng := r.provider
		resolvedProvider := task.Provider
		if resolvedProvider == "" && parent != nil {
			resolvedProvider = parent.providerID
		}
		if resolvedProvider != "" {
			if p, ok := r.providers[resolvedProvider]; ok {
				provEng = p
			}
		}
		child := NewLoop(provEng, r.tools, opts...)

		var childConv *Conversation
		if len(task.Messages) > 0 {
			// Compaction: use pre-built messages (head with truncated tool outputs)
			childConv = &Conversation{
				ID:       task.SessionID,
				Messages: task.Messages,
			}
		} else {
			childConv = &Conversation{ID: task.SessionID}
		}

		childCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		for ev := range child.Run(childCtx, childConv, task.Prompt) {
			select {
			case resultCh <- ev:
			case <-ctx.Done():
				return
			}
		}

		// Notify completion so caller can persist the child session
		if r.onComplete != nil && len(childConv.Messages) > 0 {
			r.onComplete(task, &ChildSession{
				Messages:   childConv.Messages,
				Agent:      ag.Name,
				ParentID:   task.ParentID,
				Title:      task.Description,
				TokenCount: childConv.TokenCount,
			})
		}
	}()

	return resultCh
}
