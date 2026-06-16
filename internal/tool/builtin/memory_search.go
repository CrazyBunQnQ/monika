package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"monika/internal/memory"
	"monika/internal/tool"
)

type memorySearchTool struct{ store *memory.KBStore }

func NewMemorySearch(store *memory.KBStore) tool.Tool { return &memorySearchTool{store} }

func (t *memorySearchTool) Name() string { return "memory_search" }

func (t *memorySearchTool) Description() string {
	return "Search the knowledge base for relevant memories. Returns matching files with titles and highlighted snippets."
}

func (t *memorySearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query":    map[string]any{"type": "string", "description": "Search keywords."},
			"scope":    map[string]any{"type": "string", "description": "'global', 'project', or 'auto' (default)."},
			"category": map[string]any{"type": "string", "description": "Filter: 'lesson', 'topic', 'knowledge', 'raw', or 'all'."},
			"limit":    map[string]any{"type": "integer", "description": "Max results (default 5, max 10)."},
		},
		"required": []string{"query"},
	}
}

func (t *memorySearchTool) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var p struct {
		Query    string `json:"query"`
		Scope    string `json:"scope"`
		Category string `json:"category"`
		Limit    int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	if p.Scope == "" {
		p.Scope = "auto"
	}
	if p.Limit <= 0 {
		p.Limit = 5
	}
	if p.Limit > 10 {
		p.Limit = 10
	}

	results, err := t.store.Search(p.Query, p.Scope, p.Limit)
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	if len(results) == 0 {
		return tool.ExecutionResult{Content: "No matching memories found."}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d matching memories:\n\n", len(results)))
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. **%s** [%s/%s] confidence: %s\n   path: %s | chars: %d\n",
			i+1, r.Title, r.Scope, r.Category, r.Confidence, r.Path, r.CharCount))
		if len(r.Tags) > 0 {
			sb.WriteString(fmt.Sprintf("   tags: %s\n", strings.Join(r.Tags, ", ")))
		}
		if len(r.LinkedTo) > 0 {
			sb.WriteString(fmt.Sprintf("   links: %s\n", strings.Join(r.LinkedTo, ", ")))
		}
		sb.WriteString("\n")
	}
	return tool.ExecutionResult{Content: sb.String()}, nil
}
