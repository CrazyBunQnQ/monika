package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"monika/internal/tool"
)

type agentDeleteTool struct {
	deleteFn func(json.RawMessage) error
	checkFn  func(name string) (isCustom bool, exists bool)
}

func NewAgentDeleteTool(deleteFn func(json.RawMessage) error, checkFn func(name string) (isCustom bool, exists bool)) tool.Tool {
	return &agentDeleteTool{deleteFn: deleteFn, checkFn: checkFn}
}

func (t *agentDeleteTool) Name() string { return "delete_agent" }

func (t *agentDeleteTool) Description() string {
	return `Delete a custom agent by name. Builtin agents (general, explore, plan, compaction) cannot be deleted.

Use this tool when the user asks to remove a custom agent they previously created.`
}

func (t *agentDeleteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "The name of the custom agent to delete",
			},
		},
		"required": []string{"name"},
	}
}

func (t *agentDeleteTool) Execute(_ context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var params struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return tool.ExecutionResult{}, fmt.Errorf("delete_agent: invalid args: %w", err)
	}
	if params.Name == "" {
		return tool.ExecutionResult{Content: "Error: name is required", IsError: true}, nil
	}

	isCustom, exists := t.checkFn(params.Name)
	if !exists {
		return tool.ExecutionResult{Content: fmt.Sprintf("Agent %q not found.", params.Name), IsError: true}, nil
	}
	if !isCustom {
		return tool.ExecutionResult{Content: fmt.Sprintf("Cannot delete builtin agent %q. Builtin agents cannot be removed.", params.Name), IsError: true}, nil
	}

	payload, _ := json.Marshal(map[string]string{"name": params.Name})
	if err := t.deleteFn(payload); err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("Failed to delete agent: %s", err), IsError: true}, nil
	}

	return tool.ExecutionResult{
		Content: fmt.Sprintf("Agent %q deleted successfully.", params.Name),
	}, nil
}
