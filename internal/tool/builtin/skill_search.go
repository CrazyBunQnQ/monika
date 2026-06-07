package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"monika/internal/config"
	"monika/internal/tool"
	"monika/pkg/engine"
)

type skillSearchTool struct {
	skEng  engine.SkillEngine
	home   string
	getCwd func() string
	cfg    *config.Config
}

func NewSkillSearchTool(skEng engine.SkillEngine, home string, getCwd func() string, cfg *config.Config) tool.Tool {
	return &skillSearchTool{skEng: skEng, home: home, getCwd: getCwd, cfg: cfg}
}

func (t *skillSearchTool) Name() string { return "skill_search" }

func (t *skillSearchTool) Description() string {
	return `Fuzzy search installed skills by name or description.

Use this tool when you want to find a skill relevant to the current task. Returns matching skill names and descriptions. Once identified, load the skill with the **skill** tool.

Parameters:
- query (optional): search keywords; returns all skills if empty`
}

func (t *skillSearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search keywords for matching skill name or description. Empty to list all skills.",
			},
		},
	}
}

func (t *skillSearchTool) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return tool.ExecutionResult{}, fmt.Errorf("skill_search: invalid args: %w", err)
	}

	skills, err := t.skEng.Discover(ctx, t.home, t.getCwd(), t.cfg.Skill.Paths)
	if err != nil {
		return tool.ExecutionResult{}, fmt.Errorf("skill_search: discovery failed: %w", err)
	}

	query := strings.ToLower(strings.TrimSpace(params.Query))
	matches := skills
	if query != "" {
		matches = nil
		for _, s := range skills {
			if s.Enabled != nil && !*s.Enabled {
				continue
			}
			if strings.Contains(strings.ToLower(s.Name), query) ||
				strings.Contains(strings.ToLower(s.Description), query) {
				matches = append(matches, s)
			}
		}
	}

	if len(matches) == 0 {
		if query != "" {
			return tool.ExecutionResult{Content: fmt.Sprintf("No skills found matching %q.", params.Query)}, nil
		}
		return tool.ExecutionResult{Content: "No skills installed."}, nil
	}

	var lines []string
	for _, s := range matches {
		if s.Enabled != nil && !*s.Enabled {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s: %s", s.Name, s.Description))
	}

	return tool.ExecutionResult{
		Content: fmt.Sprintf("Found %d skill(s):\n%s", len(lines), strings.Join(lines, "\n")),
	}, nil
}
