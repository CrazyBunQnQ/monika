package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"monika/internal/memory"
	"monika/internal/tool"
)

type memoryIndexTool struct{ store *memory.KBStore }

func NewMemoryIndex(store *memory.KBStore) tool.Tool { return &memoryIndexTool{store} }

func (t *memoryIndexTool) Name() string { return "memory_index" }

func (t *memoryIndexTool) Description() string {
	return "View knowledge base contents — all stored memories organized by category."
}

func (t *memoryIndexTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"scope": map[string]any{"type": "string", "description": "'global', 'project', or 'auto'."},
		},
	}
}

func (t *memoryIndexTool) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var p struct {
		Scope string `json:"scope"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	if p.Scope == "" {
		p.Scope = "auto"
	}

	content := ""
	if p.Scope == "auto" || p.Scope == memory.ScopeProject {
		content += formatFileList(t.store, memory.ScopeProject)
	}
	if p.Scope == "auto" || p.Scope == memory.ScopeGlobal {
		if p.Scope == "auto" {
			content += "\n"
		}
		content += formatFileList(t.store, memory.ScopeGlobal)
	}
	if strings.TrimSpace(content) == "" {
		return tool.ExecutionResult{Content: "Knowledge base is empty."}, nil
	}
	return tool.ExecutionResult{Content: content}, nil
}

func formatFileList(store *memory.KBStore, scope string) string {
	files, err := store.ListFiles(scope, "")
	if err != nil {
		return fmt.Sprintf("Error listing %s: %s\n", scope, err)
	}
	if len(files) == 0 {
		return ""
	}
	groups := map[string][]memory.KBFile{}
	for _, f := range files {
		label := categoryLabel(f.Category)
		groups[label] = append(groups[label], f)
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## %s Knowledge Base\n\n", strings.Title(scope)))
	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("### %s\n", k))
		for _, f := range groups[k] {
			sb.WriteString(fmt.Sprintf("- **%s** (%s) [%s] %s\n", f.Title, f.Confidence, f.Status, f.Path))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func categoryLabel(cat string) string {
	switch cat {
	case memory.CategoryKnowledge:
		return "Core Knowledge"
	case memory.CategoryProfile:
		return "User Profile"
	case memory.CategoryLesson:
		return "Lessons"
	case memory.CategoryTopic:
		return "Topics"
	case memory.CategoryRawDoc:
		return "Documents"
	case memory.CategoryRawCode:
		return "Code Repositories"
	default:
		return cat
	}
}
