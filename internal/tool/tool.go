package tool

import (
	"context"
	"encoding/json"
	"sync"
)

type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Execute(ctx context.Context, args json.RawMessage) (ExecutionResult, error)
}

type ExecutionResult struct {
	Content     string
	IsError     bool
	DiffLines   []string
	Conflicts   bool   // merge conflict markers in file
	Conflict    bool   // user has unsaved edits — AI edit blocked
	DiskContent string `json:"diskContent,omitempty"` // file on disk (with user edits)
	AiContent   string `json:"aiContent,omitempty"`   // what AI wants to write
	// Usage is set by tools that talk to an LLM on their own (e.g.
	// image/video_understand routing through MediaCaller). The agent
	// loop reads this to emit EventUsage so the tokens count toward
	// the conversation budget and compaction decisions.
	Usage any `json:"usage,omitempty"`
}

type ToolRegistry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

func NewRegistry() *ToolRegistry {
	return &ToolRegistry{tools: make(map[string]Tool)}
}

func (r *ToolRegistry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

func (r *ToolRegistry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

func (r *ToolRegistry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

func (r *ToolRegistry) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
}
