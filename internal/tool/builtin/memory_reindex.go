package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

	count := 0
	walkScope := []string{p.Scope}
	if p.Scope == "project" {
		walkScope = []string{memory.ScopeProject}
	} else if p.Scope == "global" {
		walkScope = []string{memory.ScopeGlobal}
	}

	for _, sc := range walkScope {
		files, err := t.store.ListFiles(sc, "")
		if err != nil {
			continue
		}
		for _, f := range files {
			content, err := t.store.ReadFile(sc, f.Path)
			if err != nil {
				continue
			}
			t.store.WriteFile(sc, f.Category, f.Title, extractBody(content), f.Tags, f.Confidence)
			count++
		}
	}

	return tool.ExecutionResult{
		Content: fmt.Sprintf("Reindexed %d files.", count),
	}, nil
}

func extractBody(content string) string {
	lines := strings.Split(content, "\n")
	var body []string
	pastFM := false
	for _, line := range lines {
		if !pastFM && (strings.HasPrefix(line, "> ") || strings.HasPrefix(line, "# ")) {
			if strings.HasPrefix(line, "# ") && len(body) > 0 {
				pastFM = true
				body = append(body, line)
			}
			continue
		}
		pastFM = true
		body = append(body, line)
	}
	return strings.Join(body, "\n")
}
