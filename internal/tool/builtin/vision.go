package builtin

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"monika/internal/tool"
	"monika/pkg/engine"
)

// MediaCaller runs a single-turn vision call against a configured provider
// and returns the full assistant text plus the token usage emitted by the
// provider. It is injected from main.go (where the provider engines are
// constructed) so that the builtin package stays free of any reverse
// dependency on bootstrap/config wiring.
//
// ctx must carry the provider ID via tool.WithProvider / WithModel so that the
// caller can resolve the right engine at invocation time. This keeps the tool
// session-agnostic: whichever session invokes video/image_understand uses its
// own provider.
//
// The returned Usage is the LAST usage chunk the provider emitted on the
// stream (vision calls produce a single non-streaming usage event). Callers
// should fold this into the tool's ExecutionResult so the agent loop can
// surface it through the same EventUsage channel the chat path uses; without
// that propagation vision calls are invisible to budget display and
// compaction decisions.
type MediaCaller func(ctx context.Context, prompt string, attachments []engine.AttachmentRef) (string, *engine.Usage, error)

// NewDefaultMediaCaller returns a MediaCaller backed by a static map of
// provider engines. Callers that already maintain a provider map (main.go
// uses bootstrap.Result.Providers) pass this directly to RegisterDefaults.
func NewDefaultMediaCaller(providers map[string]engine.ProviderEngine) MediaCaller {
	return func(ctx context.Context, prompt string, attachments []engine.AttachmentRef) (string, *engine.Usage, error) {
		providerID := tool.ProviderFromContext(ctx)
		model := tool.ModelFromContext(ctx)

		if providerID == "" {
			return "", nil, errors.New("vision: no provider in context")
		}
		prov, ok := providers[providerID]
		if !ok {
			return "", nil, fmt.Errorf("vision: provider %q not available", providerID)
		}

		msgs := []engine.ChatMessage{{
			Role:        "user",
			Content:     prompt,
			Attachments: attachments,
		}}
		req := engine.ChatRequest{
			Provider: providerID,
			Model:    model,
			Messages: msgs,
		}

		evCh, err := prov.StreamChat(ctx, req)
		if err != nil {
			return "", nil, fmt.Errorf("vision: %w", err)
		}

		var sb strings.Builder
		var lastUsage *engine.Usage
		for ev := range evCh {
			switch ev.Kind {
			case engine.EventContentDelta:
				sb.WriteString(ev.Text)
			case engine.EventUsage:
				u := ev.Usage
				lastUsage = &u
			case engine.EventError:
				if ev.Error.Message != "" {
					return "", lastUsage, fmt.Errorf("vision: %s", ev.Error.Message)
				}
				return "", lastUsage, errors.New("vision: provider returned an error")
			}
		}
		return stripThinkTags(sb.String()), lastUsage, nil
	}
}

var thinkTagRe = regexp.MustCompile(`(?s)<think\b[^>]*>.*?</think\s*>`)

// stripThinkTags removes <think>...</think> reasoning blocks that some models
// (e.g. DeepSeek-R1) embed in the content stream. Handles both closed tags and
// unclosed <think> tags (strips from <think> to end of string).
func stripThinkTags(s string) string {
	s = thinkTagRe.ReplaceAllString(s, "")
	if idx := strings.Index(s, "<think"); idx >= 0 {
		closeIdx := strings.Index(s[idx:], "</think")
		if closeIdx >= 0 {
			s = s[:idx] + s[idx+closeIdx+len("</think"):]
		} else {
			s = s[:idx]
		}
	}
	return strings.TrimSpace(s)
}
