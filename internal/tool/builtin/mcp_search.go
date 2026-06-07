package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"monika/internal/tool"
	"monika/pkg/engine"
)

type mcpSearchTool struct {
	mcpRegistry *engine.MCPRegistry
}

func NewMCPSearchTool(mcpRegistry *engine.MCPRegistry) tool.Tool {
	return &mcpSearchTool{mcpRegistry: mcpRegistry}
}

func (t *mcpSearchTool) Name() string { return "mcp_search" }

func (t *mcpSearchTool) Description() string {
	return `Fuzzy search connected MCP tools by name, server, or capability description.

Use this tool to discover MCP tools available for the current task. Each tool is shown with its server ID (prefix) and description. Once identified, call the tool directly by its full prefixed name.

Parameters:
- query (optional): search keywords for matching tool name, server ID, or description. Empty to list all tools.`
}

func (t *mcpSearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search keywords for matching tool name, server ID, or description. Empty to list all tools.",
			},
		},
	}
}

func (t *mcpSearchTool) Execute(_ context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return tool.ExecutionResult{}, fmt.Errorf("mcp_search: invalid args: %w", err)
	}

	tools := t.mcpRegistry.GetTools()
	servers := t.mcpRegistry.GetServers()
	serverNames := make(map[string]string, len(servers))
	for _, s := range servers {
		name := s.ID
		if s.Name != "" {
			name = s.Name
		}
		serverNames[s.ID] = name
	}

	query := strings.ToLower(strings.TrimSpace(params.Query))
	type match struct {
		PrefixedName string
		ServerID     string
		ServerName   string
		Description  string
		Annotations  engine.MCPAnnotations
	}

	var matches []match
	for _, t := range tools {
		serverName := serverNames[t.ServerID]
		if query == "" ||
			strings.Contains(strings.ToLower(t.Name), query) ||
			strings.Contains(strings.ToLower(t.ServerID), query) ||
			strings.Contains(strings.ToLower(serverName), query) ||
			strings.Contains(strings.ToLower(t.Description), query) {
			matches = append(matches, match{
				PrefixedName: t.Name,
				ServerID:     t.ServerID,
				ServerName:   serverName,
				Description:  t.Description,
				Annotations:  t.Annotations,
			})
		}
	}

	if len(matches) == 0 {
		if query != "" {
			return tool.ExecutionResult{Content: fmt.Sprintf("No MCP tools found matching %q.", params.Query)}, nil
		}
		return tool.ExecutionResult{Content: "No MCP tools available."}, nil
	}

	// Group by server for readability
	byServer := make(map[string][]match)
	for _, m := range matches {
		byServer[m.ServerID] = append(byServer[m.ServerID], m)
	}

	var lines []string
	for _, srv := range servers {
		serverMatches := byServer[srv.ID]
		if len(serverMatches) == 0 {
			continue
		}
		displayName := serverNames[srv.ID]
		if srv.Instructions != "" {
			lines = append(lines, fmt.Sprintf("### %s\n%s", displayName, srv.Instructions))
		} else {
			lines = append(lines, fmt.Sprintf("### %s", displayName))
		}
		for _, m := range serverMatches {
			annTags := annotationTags(m.Annotations)
			desc := m.Description
			if desc == "" {
				desc = "(no description)"
			}
			lines = append(lines, fmt.Sprintf("- **%s**%s: %s", m.PrefixedName, annTags, desc))
		}
		lines = append(lines, "")
	}

	return tool.ExecutionResult{
		Content: fmt.Sprintf("Found %d MCP tool(s):\n%s", len(matches), strings.Join(lines, "\n")),
	}, nil
}

func annotationTags(a engine.MCPAnnotations) string {
	var tags []string
	if a.ReadOnly {
		tags = append(tags, "read-only")
	}
	if a.Destructive {
		tags = append(tags, "destructive")
	}
	if a.Idempotent {
		tags = append(tags, "idempotent")
	}
	if len(tags) == 0 {
		return ""
	}
	return " [" + strings.Join(tags, ", ") + "]"
}
