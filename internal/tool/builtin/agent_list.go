package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"monika/internal/tool"
)

type AgentInfo struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Model       string            `json:"model"`
	Provider    string            `json:"provider"`
	Hidden      bool              `json:"hidden"`
	Disabled    bool              `json:"disabled"`
	IsCustom    bool              `json:"isCustom"`
	Source      string            `json:"source"`
	Permission  map[string]string `json:"permission"`
}

type agentListTool struct {
	listFn func() []AgentInfo
}

func NewAgentListTool(listFn func() []AgentInfo) tool.Tool {
	return &agentListTool{listFn: listFn}
}

func (t *agentListTool) Name() string { return "list_agents" }

func (t *agentListTool) Description() string {
	return `List all registered agents (builtin and custom) with their status.

Use this tool to discover available agents before using spawn_agent.`
}

func (t *agentListTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (t *agentListTool) Execute(_ context.Context, _ json.RawMessage) (tool.ExecutionResult, error) {
	agents := t.listFn()
	if len(agents) == 0 {
		return tool.ExecutionResult{Content: "No agents registered."}, nil
	}

	var lines []string
	for _, a := range agents {
		tags := a.Source
		if a.IsCustom {
			tags = "custom"
		}
		status := ""
		if a.Disabled {
			status = " [disabled]"
		} else if a.Hidden {
			status = " [hidden]"
		}
		desc := a.Description
		if desc == "" {
			desc = "(no description)"
		}
		detail := ""
		if a.Model != "" {
			detail = fmt.Sprintf(" model=%s", a.Model)
		}
		lines = append(lines, fmt.Sprintf("- %s (%s%s)%s%s", a.Name, tags, detail, status, desc))
	}

	return tool.ExecutionResult{
		Content: fmt.Sprintf("Registered agents (%d):\n%s", len(agents), strings.Join(lines, "\n")),
	}, nil
}
