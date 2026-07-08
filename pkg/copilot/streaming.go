package copilot

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"monika/pkg/engine"
	oaiclient "monika/pkg/openai"
)

// Option configures StreamChat.
type Option func(*streamConfig)

type streamConfig struct {
	editorVersion string
	refreshToken  string
	onRefresh     TokenRefreshCallback
	hasVision     bool
	integrationID string
}

// WithEditorVersion sets the Editor-Version header.
func WithEditorVersion(v string) Option { return func(c *streamConfig) { c.editorVersion = v } }

// WithRefreshToken enables automatic token refresh on 401.
func WithRefreshToken(rt string) Option { return func(c *streamConfig) { c.refreshToken = rt } }

// WithRefreshCallback registers a callback for token refresh persistence.
func WithRefreshCallback(cb TokenRefreshCallback) Option {
	return func(c *streamConfig) { c.onRefresh = cb }
}

// WithVision controls the Copilot-Vision-Request header.
func WithVision(v bool) Option { return func(c *streamConfig) { c.hasVision = v } }

// WithIntegrationID sets the Copilot-Integration-Id header.
func WithIntegrationID(id string) Option { return func(c *streamConfig) { c.integrationID = id } }

var refreshMu sync.Mutex

// StreamChat sends a streaming chat request to the Copilot API.
func StreamChat(
	ctx context.Context,
	baseURL, token, model string,
	messages []engine.ChatMessage,
	tools []engine.ToolDef,
	opts ...Option,
) (<-chan engine.ChatEvent, error) {
	cfg := &streamConfig{}
	for _, o := range opts {
		o(cfg)
	}

	bodyJSON, err := oaiclient.BuildChatRequest(model, messages, tools, true)
	if err != nil {
		return nil, err
	}

	setHeaders := func(req *http.Request, authToken string) {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+authToken)
		if cfg.editorVersion != "" {
			req.Header.Set("Editor-Version", cfg.editorVersion)
		}
		if cfg.hasVision {
			req.Header.Set("Copilot-Vision-Request", "true")
		}
		if cfg.integrationID != "" {
			req.Header.Set("Copilot-Integration-Id", cfg.integrationID)
		}
	}

	doRequest := func(authToken string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(bodyJSON))
		if err != nil {
			return nil, err
		}
		setHeaders(req, authToken)
		return oaiclient.HTTPClientFor(baseURL).Do(req)
	}

	// First attempt
	resp, err := doRequest(token)
	if err != nil {
		if !oaiclient.RetryableHTTPError(err) {
			return nil, err
		}
	} else if resp.StatusCode == http.StatusUnauthorized && cfg.refreshToken != "" {
		resp.Body.Close()
		newToken, refreshErr := doRefresh(cfg)
		if refreshErr == nil {
			resp, err = doRequest(newToken)
		} else {
			return nil, fmt.Errorf("copilot: token expired and refresh failed: %w", refreshErr)
		}
	}

	if err == nil && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode < 500 {
			return nil, fmt.Errorf("copilot API returned %d: %s", resp.StatusCode, string(respBody))
		}
	} else if err == nil {
		ch := make(chan engine.ChatEvent, 128)
		go func() {
			defer close(ch)
			if err := oaiclient.ParseSSEStreamInGoroutine(ctx, resp, ch); err != nil {
				if ctx.Err() != nil {
					return
				}
				oaiclient.SendError(ch, err)
			}
		}()
		return ch, nil
	}

	// Retry path
	ch := make(chan engine.ChatEvent, 128)
	go func() {
		defer close(ch)
		const maxAttempts = 10
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			if attempt > 1 {
				delay := time.Duration(500*(1<<(attempt-2))) * time.Millisecond
				if delay > 30*time.Second {
					delay = 30 * time.Second
				}
				select {
				case <-time.After(delay):
				case <-ctx.Done():
					return
				}
			}

			req, _ := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(bodyJSON))
			setHeaders(req, token)

			r, e := oaiclient.HTTPClientFor(baseURL).Do(req)
			if e != nil {
				if oaiclient.RetryableHTTPError(e) && attempt < maxAttempts {
					continue
				}
				oaiclient.SendError(ch, e)
				return
			}
			if r.StatusCode != http.StatusOK {
				rBody, _ := io.ReadAll(r.Body)
				r.Body.Close()
				if r.StatusCode >= 500 && attempt < maxAttempts {
					continue
				}
				oaiclient.SendError(ch, fmt.Errorf("copilot API returned %d: %s", r.StatusCode, string(rBody)))
				return
			}
			if err := oaiclient.ParseSSEStreamInGoroutine(ctx, r, ch); err != nil {
				if ctx.Err() != nil {
					return
				}
				oaiclient.SendError(ch, err)
				return
			}
			return
		}
		oaiclient.SendError(ch, fmt.Errorf("copilot: stream failed after %d attempts", maxAttempts))
	}()
	return ch, nil
}

func doRefresh(cfg *streamConfig) (string, error) {
	refreshMu.Lock()
	defer refreshMu.Unlock()

	resp, err := RefreshToken(context.Background(), cfg.refreshToken)
	if err != nil {
		return "", err
	}
	expiresAt := time.Now().Unix() + int64(resp.ExpiresIn)
	if cfg.onRefresh != nil {
		cfg.onRefresh(resp.AccessToken, resp.RefreshToken, expiresAt)
	}
	return resp.AccessToken, nil
}

// DetectVision checks if any message has image attachments.
func DetectVision(messages []engine.ChatMessage) bool {
	for _, msg := range messages {
		for _, att := range msg.Attachments {
			if strings.HasPrefix(att.MimeType, "image/") {
				return true
			}
		}
	}
	return false
}
