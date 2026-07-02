package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"monika/internal/tool"
)

type agentCreateTool struct {
	saveFn func(json.RawMessage) error
}

func NewAgentCreateTool(saveFn func(json.RawMessage) error) tool.Tool {
	return &agentCreateTool{saveFn: saveFn}
}

func (t *agentCreateTool) Name() string { return "create_agent" }

func (t *agentCreateTool) Description() string {
	return `Create or update a custom agent. Once created, the agent can be dispatched via spawn_agent.

Use this tool when the user wants to add a new specialized agent or modify an existing custom agent's configuration.
Builtin agents (general, explore, plan) can be updated but not deleted.`
}

func (t *agentCreateTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Unique agent name (snake_case). Used as the subagent_type in spawn_agent.",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Short description of what this agent does.",
			},
			"system_prompt": map[string]any{
				"type":        "string",
				"description": "The system prompt that defines the agent's behavior and instructions.",
			},
			"model": map[string]any{
				"type":        "string",
				"description": "Model ID in 'providerID/modelID' format (e.g. 'deepseek/deepseek-chat'). Empty inherits the default.",
			},
		},
		"required": []string{"name", "system_prompt"},
	}
}

func (t *agentCreateTool) Execute(_ context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var params struct {
		Name         string `json:"name"`
		Description  string `json:"description"`
		SystemPrompt string `json:"system_prompt"`
		Model        string `json:"model"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return tool.ExecutionResult{}, fmt.Errorf("create_agent: invalid args: %w", err)
	}
	if params.Name == "" {
		return tool.ExecutionResult{Content: "Error: name is required", IsError: true}, nil
	}
	if params.SystemPrompt == "" {
		return tool.ExecutionResult{Content: "Error: system_prompt is required", IsError: true}, nil
	}

	entry := map[string]any{
		"name":         params.Name,
		"description":  params.Description,
		"systemPrompt": params.SystemPrompt,
		"model":        params.Model,
		"isCustom":     true,
		"source":       "custom",
	}
	payload, _ := json.Marshal(entry)

	if err := t.saveFn(payload); err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("Failed to create agent: %s", err), IsError: true}, nil
	}

	return tool.ExecutionResult{
		Content: fmt.Sprintf("Agent %q created successfully. Use spawn_agent with subagent_type=%q to dispatch it.", params.Name, params.Name),
	}, nil
}
