package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"monika/internal/memory"
	"monika/internal/tool"
)

type memoryReadTool struct{ store *memory.KBStore }

func NewMemoryRead(store *memory.KBStore) tool.Tool { return &memoryReadTool{store} }

func (t *memoryReadTool) Name() string { return "memory_read" }

func (t *memoryReadTool) Description() string {
	return "Read a single memory's full content by path. Use after memory_search to get complete details."
}

func (t *memoryReadTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":  map[string]any{"type": "string", "description": "Path of the memory file (from memory_search or memory_index results)."},
			"scope": map[string]any{"type": "string", "description": "'global' or 'project' (default)."},
		},
		"required": []string{"path"},
	}
}

func (t *memoryReadTool) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var p struct {
		Path  string `json:"path"`
		Scope string `json:"scope"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	if p.Scope == "" {
		p.Scope = memory.ScopeProject
	}

	content, err := t.store.ReadFile(p.Scope, p.Path)
	if err != nil {
		return tool.ExecutionResult{
			Content: fmt.Sprintf("Memory not found at path '%s' (scope: %s): %s", p.Path, p.Scope, err),
			IsError: true,
		}, nil
	}
	return tool.ExecutionResult{Content: content}, nil
}
