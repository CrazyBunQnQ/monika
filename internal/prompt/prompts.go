package prompt

import "strings"

type PromptSet struct {
	Identity         string
	ToolUsage        string
	Planning         string
	CodeQuality      string
	ResponseStyle    string
	SafetyBoundaries string
	Remember         string
	MaxSteps         string
}

func ForModel(modelID string) string {
	m := strings.ToLower(modelID)
	switch {
	case strings.Contains(m, "claude"), strings.Contains(m, "anthropic"):
		return "anthropic"
	case strings.Contains(m, "gpt"), strings.Contains(m, "o1-"), strings.Contains(m, "o1_"), strings.Contains(m, "o3-"), strings.Contains(m, "o3_"), strings.Contains(m, "o4-"), strings.Contains(m, "o4_"):
		return "gpt"
	case strings.Contains(m, "gemini"):
		return "gemini"
	case strings.Contains(m, "deepseek"), strings.Contains(m, "qwen"), strings.Contains(m, "kimi"):
		return "deepseek"
	default:
		return "default"
	}
}

func Get(modelID string) PromptSet {
	switch ForModel(modelID) {
	case "anthropic":
		return anthropicPrompt
	case "gpt":
		return gptPrompt
	case "gemini":
		return geminiPrompt
	case "deepseek":
		return deepseekPrompt
	default:
		return defaultPrompt
	}
}
