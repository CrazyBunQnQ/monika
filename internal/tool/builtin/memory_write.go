package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"monika/internal/memory"
	"monika/internal/tool"
)

type memoryWriteTool struct{ store *memory.KBStore }

func NewMemoryWrite(store *memory.KBStore) tool.Tool { return &memoryWriteTool{store} }

func (t *memoryWriteTool) Name() string { return "memory_write" }

func (t *memoryWriteTool) Description() string {
	return "Create a NEW memory entry. If a similar memory may already exist, use memory_search first then memory_update."
}

func (t *memoryWriteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title":      map[string]any{"type": "string", "description": "Title of the memory."},
			"content":    map[string]any{"type": "string", "description": "Markdown content."},
			"category":   map[string]any{"type": "string", "description": "'lesson', 'topic', or 'knowledge_update'."},
			"scope":      map[string]any{"type": "string", "description": "'global' or 'project' (default)."},
			"tags":       map[string]any{"type": "array", "items": map[string]string{"type": "string"}},
			"confidence": map[string]any{"type": "string", "description": "'high', 'medium', or 'low'."},
		},
		"required": []string{"title", "content", "category"},
	}
}

func (t *memoryWriteTool) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var p struct {
		Title      string   `json:"title"`
		Content    string   `json:"content"`
		Category   string   `json:"category"`
		Scope      string   `json:"scope"`
		Tags       []string `json:"tags"`
		Confidence string   `json:"confidence"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	if p.Scope == "" {
		p.Scope = memory.ScopeProject
	}
	if p.Confidence == "" {
		p.Confidence = "medium"
	}

	catMap := map[string]string{
		"lesson":           memory.CategoryLesson,
		"topic":            memory.CategoryTopic,
		"knowledge_update": memory.CategoryKnowledge,
		"knowledge":        memory.CategoryKnowledge,
		"profile":          memory.CategoryProfile,
	}
	cat, ok := catMap[p.Category]
	if !ok {
		return tool.ExecutionResult{
			Content: fmt.Sprintf("Invalid category '%s'. Use 'lesson', 'topic', or 'knowledge_update'.", p.Category),
			IsError: true,
		}, nil
	}
	if err := t.store.WriteFile(p.Scope, cat, p.Title, p.Content, p.Tags, p.Confidence); err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("Failed to write: %s", err), IsError: true}, nil
	}
	return tool.ExecutionResult{Content: fmt.Sprintf("Memory '%s' written to %s scope. If you intended to update an existing memory, use memory_update instead.", p.Title, p.Scope)}, nil
}
