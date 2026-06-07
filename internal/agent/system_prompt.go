package agent

import "monika/internal/prompt"

func PromptForModel(model string) prompt.PromptSet {
	return prompt.Get(model)
}
