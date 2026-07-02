package agent

import (
	"monika/internal/prompt"
	"testing"
)

func TestPromptConstantsNotEmpty(t *testing.T) {
	ps := prompt.Get("")
	constants := map[string]string{
		"Identity":         ps.Identity,
		"ToolUsage":        ps.ToolUsage,
		"Planning":         ps.Planning,
		"CodeQuality":      ps.CodeQuality,
		"ResponseStyle":    ps.ResponseStyle,
		"SafetyBoundaries": ps.SafetyBoundaries,
		"Remember":         ps.Remember,
	}
	for name, value := range constants {
		if value == "" {
			t.Errorf("%s is empty", name)
		}
	}
}

func TestPromptTokenBudget(t *testing.T) {
	ps := prompt.Get("")
	total := len(ps.Identity) +
		len(ps.ToolUsage) +
		len(ps.Planning) +
		len(ps.CodeQuality) +
		len(ps.ResponseStyle) +
		len(ps.SafetyBoundaries) +
		len(ps.Remember)

	const maxChars = 18000
	if total > maxChars {
		t.Errorf("total prompt size %d chars exceeds budget %d", total, maxChars)
	}
	t.Logf("total prompt size: %d chars (~%d estimated tokens)", total, total/4)
}
