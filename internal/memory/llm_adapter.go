package memory

import (
	"context"
)

// GoLLMAdapter wraps a Go function as an ExtractionLLM/CompactionLLM/ReviewLLM.
type GoLLMAdapter struct {
	ChatFn func(ctx context.Context, systemPrompt, userMessage string) (string, error)
}

func (a *GoLLMAdapter) Chat(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	return a.ChatFn(ctx, systemPrompt, userMessage)
}
