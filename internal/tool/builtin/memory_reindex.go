package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"monika/internal/memory"
	"monika/internal/tool"
)

type memoryReindexTool struct{ store *memory.KBStore }

func NewMemoryReindex(store *memory.KBStore) tool.Tool { return &memoryReindexTool{store} }

func (t *memoryReindexTool) Name() string { return "memory_reindex" }

func (t *memoryReindexTool) Description() string {
	return "Rebuild the FTS5 search index from all files on disk."
}

func (t *memoryReindexTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"scope": map[string]any{"type": "string", "description": "'global' or 'project' (default)."},
		},
	}
}

func (t *memoryReindexTool) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var p struct {
		Scope string `json:"scope"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	if p.Scope == "" {
		p.Scope = memory.ScopeProject
	}

	count, err := t.store.ReindexFromDisk(p.Scope)
	if err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("Reindex failed: %v", err), IsError: true}, nil
	}

	return tool.ExecutionResult{
		Content: fmt.Sprintf("Reindexed %d files.", count),
	}, nil
}
