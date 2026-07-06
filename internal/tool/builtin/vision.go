package builtin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"monika/internal/tool"
	"monika/pkg/engine"
)

// VisionCaller runs a single-turn vision call against a configured provider
// and returns the full assistant text. It is injected from main.go (where the
// provider engines are constructed) so that the builtin package stays free of
// any reverse dependency on bootstrap/config wiring.
//
// ctx must carry the provider ID via tool.WithProvider / WithModel so that the
// caller can resolve the right engine at invocation time. This keeps the tool
// session-agnostic: whichever session invokes video/image_understand uses its
// own provider.
type VisionCaller func(ctx context.Context, prompt string, images []engine.ImageRef) (string, error)

// NewDefaultVisionCaller returns a VisionCaller backed by a static map of
// provider engines. Callers that already maintain a provider map (main.go
// uses bootstrap.Result.Providers) pass this directly to RegisterDefaults.
func NewDefaultVisionCaller(providers map[string]engine.ProviderEngine) VisionCaller {
	return func(ctx context.Context, prompt string, images []engine.ImageRef) (string, error) {
		providerID := tool.ProviderFromContext(ctx)
		model := tool.ModelFromContext(ctx)

		if providerID == "" {
			return "", errors.New("vision: no provider in context")
		}
		prov, ok := providers[providerID]
		if !ok {
			return "", fmt.Errorf("vision: provider %q not available", providerID)
		}

		msgs := []engine.ChatMessage{{
			Role:    "user",
			Content: prompt,
			Images:  images,
		}}
		req := engine.ChatRequest{
			Provider: providerID,
			Model:    model,
			Messages: msgs,
		}

		evCh, err := prov.StreamChat(ctx, req)
		if err != nil {
			return "", fmt.Errorf("vision: %w", err)
		}

		var sb strings.Builder
		for ev := range evCh {
			switch ev.Kind {
			case engine.EventContentDelta:
				sb.WriteString(ev.Text)
			case engine.EventError:
				if ev.Error.Message != "" {
					return "", fmt.Errorf("vision: %s", ev.Error.Message)
				}
				return "", errors.New("vision: provider returned an error")
			}
		}
		return sb.String(), nil
	}
}
