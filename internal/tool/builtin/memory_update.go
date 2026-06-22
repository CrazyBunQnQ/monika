package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"monika/internal/memory"
	"monika/internal/tool"
)

type memoryUpdateTool struct{ store *memory.KBStore }

func NewMemoryUpdate(store *memory.KBStore) tool.Tool { return &memoryUpdateTool{store} }

func (t *memoryUpdateTool) Name() string { return "memory_update" }

func (t *memoryUpdateTool) Description() string {
	return "Update an existing memory by path with merged content. LLM should memory_read first, merge new insight, then pass the full merged content. Check for overflow warnings on profile/knowledge files."
}

func (t *memoryUpdateTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":    map[string]any{"type": "string", "description": "Path of the memory to update (from memory_search/memory_index)."},
			"content": map[string]any{"type": "string", "description": "Full merged content (including frontmatter) to overwrite the file."},
			"scope":   map[string]any{"type": "string", "description": "'global' or 'project' (default)."},
		},
		"required": []string{"path", "content"},
	}
}

func (t *memoryUpdateTool) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var p struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Scope   string `json:"scope"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	if p.Scope == "" {
		p.Scope = memory.ScopeProject
	}

	if err := t.store.UpdateFile(p.Scope, p.Path, p.Content); err != nil {
		return tool.ExecutionResult{
			Content: fmt.Sprintf("Failed to update '%s': %s. Use memory_write to create a new memory.", p.Path, err),
			IsError: true,
		}, nil
	}

	msg := fmt.Sprintf("Memory '%s' updated in %s scope.", p.Path, p.Scope)

	// 字符上限检查
	if warn := checkCharLimit(p.Path, p.Content); warn != "" {
		msg += "\n" + warn
	}

	return tool.ExecutionResult{Content: msg}, nil
}

// checkCharLimit 检查 profile/knowledge 文件是否超出字符上限。
// 返回警告字符串（超限时）或空字符串（未超限）。
func checkCharLimit(path, content string) string {
	charCount := len([]rune(content))
	lowerPath := strings.ToLower(path)

	if strings.Contains(lowerPath, "profile.md") && charCount > memory.MaxProfileChars {
		return fmt.Sprintf("⚠️ Content exceeds profile limit (%d/%d chars). Written successfully, but please read it back and trim to fit the limit using memory_update.", charCount, memory.MaxProfileChars)
	}
	if strings.Contains(lowerPath, "knowledge.md") && charCount > memory.MaxKnowledgeChars {
		return fmt.Sprintf("⚠️ Content exceeds knowledge limit (%d/%d chars). Written successfully, but please read it back and trim to fit the limit using memory_update.", charCount, memory.MaxKnowledgeChars)
	}
	return ""
}
