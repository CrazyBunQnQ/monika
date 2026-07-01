package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"monika/internal/memory"
	"monika/internal/tool"
)

type memoryGraphTool struct{ store *memory.KBStore }

func NewMemoryGraph(store *memory.KBStore) tool.Tool { return &memoryGraphTool{store} }

func (t *memoryGraphTool) Name() string { return "memory_graph" }

func (t *memoryGraphTool) Description() string {
	return "Query the memory knowledge graph. Traverse links from a seed memory, find entities, or discover related memories. " +
		"Use 'traverse' to walk the link graph from a seed path. Use 'entity' to find all memories mentioning a file/function/concept. " +
		"Use 'neighbors' to find memories sharing entities with a seed."
}

func (t *memoryGraphTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action":   map[string]any{"type": "string", "description": "'traverse' (walk link graph), 'entity' (find by entity name), or 'neighbors' (co-occurring entities). Default: 'traverse'."},
			"seed":     map[string]any{"type": "string", "description": "Seed memory path (for traverse) or entity name (for entity/neighbors)."},
			"scope":    map[string]any{"type": "string", "description": "'global', 'project', or 'auto' (default)."},
			"hops":     map[string]any{"type": "integer", "description": "Max graph traversal depth (default 2, max 3)."},
			"relation": map[string]any{"type": "string", "description": "Filter by relation type: 'related', 'causes', 'fixes', 'supersedes', 'part-of'."},
		},
		"required": []string{"seed"},
	}
}

func (t *memoryGraphTool) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var p struct {
		Action   string `json:"action"`
		Seed     string `json:"seed"`
		Scope    string `json:"scope"`
		Hops     int    `json:"hops"`
		Relation string `json:"relation"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	if p.Action == "" {
		p.Action = "traverse"
	}
	if p.Scope == "" {
		p.Scope = "auto"
	}
	if p.Hops <= 0 {
		p.Hops = 2
	}
	if p.Hops > 3 {
		p.Hops = 3
	}

	var sb strings.Builder

	switch p.Action {
	case "traverse":
		scopes := t.resolveScopes(p.Scope)
		for _, scope := range scopes {
			nodes, err := t.store.GraphTraverse(scope, p.Seed, p.Hops, p.Relation)
			if err != nil || len(nodes) == 0 {
				continue
			}
			fmt.Fprintf(&sb, "Graph traversal from %s (scope: %s, %d nodes):\n\n", p.Seed, scope, len(nodes))
			for _, n := range nodes {
				indent := strings.Repeat("  ", n.Depth)
				fmt.Fprintf(&sb, "%s%d. **%s** [%s]\n", indent, n.Depth, n.File.Title, n.File.Path)
				if len(n.File.Tags) > 0 {
					fmt.Fprintf(&sb, "%s   tags: %s\n", indent, strings.Join(n.File.Tags, ", "))
				}
				if n.File.Snippet != "" {
					fmt.Fprintf(&sb, "%s   snippet: %s\n", indent, n.File.Snippet)
				}
				fmt.Fprintf(&sb, "%s   path: %s\n", indent, strings.Join(n.Path, " → "))
			}
			break
		}
		if sb.Len() == 0 {
			return tool.ExecutionResult{Content: "No graph nodes found from the seed."}, nil
		}

	case "entity":
		scopes := t.resolveScopes(p.Scope)
		total := 0
		for _, scope := range scopes {
			results, err := t.store.QueryByEntity(scope, p.Seed)
			if err != nil || len(results) == 0 {
				continue
			}
			total += len(results)
			fmt.Fprintf(&sb, "Scope %s — %d memories mentioning '%s':\n\n", scope, len(results), p.Seed)
			for i, r := range results {
				fmt.Fprintf(&sb, "%d. **%s** [%s] path: %s\n", i+1, r.Title, r.Category, r.Path)
				if len(r.Tags) > 0 {
					fmt.Fprintf(&sb, "   tags: %s\n", strings.Join(r.Tags, ", "))
				}
			}
			sb.WriteString("\n")
		}
		if total == 0 {
			return tool.ExecutionResult{Content: fmt.Sprintf("No memories found mentioning entity '%s'.", p.Seed)}, nil
		}

	case "neighbors":
		scopes := t.resolveScopes(p.Scope)
		for _, scope := range scopes {
			nb, err := t.store.EntityNeighborhood(scope, p.Seed, 1)
			if err != nil || len(nb) == 0 {
				continue
			}
			fmt.Fprintf(&sb, "Entity neighborhood for '%s' (scope: %s):\n\n", p.Seed, scope)
			for memPath, ents := range nb {
				fmt.Fprintf(&sb, "**%s**: %s\n", memPath, strings.Join(ents, ", "))
			}
			break
		}
		if sb.Len() == 0 {
			return tool.ExecutionResult{Content: fmt.Sprintf("No neighborhood found for entity '%s'.", p.Seed)}, nil
		}

	default:
		return tool.ExecutionResult{Content: "Unknown action: " + p.Action + ". Use 'traverse', 'entity', or 'neighbors'."}, nil
	}

	return tool.ExecutionResult{Content: sb.String()}, nil
}

func (t *memoryGraphTool) resolveScopes(scope string) []string {
	switch scope {
	case memory.ScopeGlobal:
		return []string{memory.ScopeGlobal}
	case memory.ScopeProject:
		return []string{memory.ScopeProject}
	default:
		return []string{memory.ScopeProject, memory.ScopeGlobal}
	}
}
