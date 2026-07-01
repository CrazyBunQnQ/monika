package memory

import (
	"context"
	"fmt"
	"strings"

	"monika/pkg/engine"
)

// ProviderExtractionLLM wraps a ProviderEngine to implement ExtractionLLM.
type ProviderExtractionLLM struct {
	Provider engine.ProviderEngine
	Model    string
}

// Chat calls StreamChat and collects all content deltas into a single string.
func (p *ProviderExtractionLLM) Chat(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	messages := []engine.ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	}
	events, err := p.Provider.StreamChat(ctx, engine.ChatRequest{
		Provider: p.Provider.ID(),
		Model:    p.Model,
		Messages: messages,
	})
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for ev := range events {
		switch ev.Kind {
		case engine.EventContentDelta:
			sb.WriteString(ev.Text)
		case engine.EventError:
			return sb.String(), fmt.Errorf("extraction LLM error: %s", ev.Error.Message)
		}
	}
	return sb.String(), nil
}
