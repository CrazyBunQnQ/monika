package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"monika/internal/tool"
)

type lspListTool struct {
	registry *tool.ToolRegistry
}

func NewLSPListTool(registry *tool.ToolRegistry) tool.Tool {
	return &lspListTool{registry: registry}
}

func (t *lspListTool) Name() string { return "lsp_list" }

func (t *lspListTool) Description() string {
	return "List available language servers and their status. Use this tool when you need to confirm which language servers are active."
}

func (t *lspListTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (t *lspListTool) Execute(ctx context.Context, _ json.RawMessage) (tool.ExecutionResult, error) {
	lspTool, ok := t.registry.Get("lsp")
	if !ok {
		return tool.ExecutionResult{Content: "LSP tool not available."}, nil
	}

	statusCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	statusArgs, _ := json.Marshal(map[string]string{"action": "status"})
	result, err := lspTool.Execute(statusCtx, statusArgs)
	if err != nil {
		return tool.ExecutionResult{}, fmt.Errorf("lsp_list: %w", err)
	}
	if result.IsError {
		return tool.ExecutionResult{Content: fmt.Sprintf("LSP status error: %s", result.Content)}, nil
	}
	if result.Content == "" {
		return tool.ExecutionResult{Content: "No LSP servers active."}, nil
	}

	return tool.ExecutionResult{
		Content: result.Content,
	}, nil
}
