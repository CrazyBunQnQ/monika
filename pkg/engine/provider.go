package engine

import (
	"context"
	"encoding/json"
)

type ChatRequest struct {
	Provider string
	Model    string
	Messages []ChatMessage
	Tools    []ToolDef
}

type ChatMessage struct {
	Role             string          `json:"role"`
	Content          string          `json:"content"`
	Images           []ImageRef      `json:"images,omitempty"`
	ReasoningContent string          `json:"reasoning_content"`
	ToolCalls        []ToolCall      `json:"tool_calls,omitempty"`
	ToolCallID       string          `json:"tool_call_id,omitempty"`
	Name             string          `json:"name,omitempty"`
	TokenUsage       *Usage          `json:"token_usage,omitempty"`
	QuotedMessages   []QuotedMessage `json:"quoted_messages,omitempty"`
}

// ImageRef is a reference to an image attached to a chat message.
// URL may be either a data URL (data:image/jpeg;base64,...) or an https URL.
// Detail hints at the model's processing resolution: "auto" | "low" | "high".
type ImageRef struct {
	URL    string `json:"url,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// MarshalJSON renders the message content either as a plain string (the
// historical behavior, used for text-only messages) or as an OpenAI-style
// multipart array when one or more images are attached. This keeps the wire
// format backward-compatible: messages without Images still serialize to
// "role":"user","content":"hello", exactly as before.
func (m ChatMessage) MarshalJSON() ([]byte, error) {
	type alias ChatMessage // avoid recursion
	if len(m.Images) == 0 {
		return json.Marshal(alias(m))
	}
	parts := make([]map[string]any, 0, 1+len(m.Images))
	if m.Content != "" {
		parts = append(parts, map[string]any{"type": "text", "text": m.Content})
	}
	for _, img := range m.Images {
		entry := map[string]any{
			"type":      "image_url",
			"image_url": map[string]any{"url": img.URL},
		}
		if img.Detail != "" {
			entry["image_url"].(map[string]any)["detail"] = img.Detail
		}
		parts = append(parts, entry)
	}
	return json.Marshal(map[string]any{
		"role":              m.Role,
		"content":           parts,
		"reasoning_content": m.ReasoningContent,
		"tool_calls":        m.ToolCalls,
		"tool_call_id":      m.ToolCallID,
		"name":              m.Name,
		"token_usage":       m.TokenUsage,
		"quoted_messages":   m.QuotedMessages,
	})
}

// QuotedMessage is a snapshot of a referenced message used for quoting/forwarding.
type QuotedMessage struct {
	ID      string `json:"id"`
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatEvent struct {
	Kind             EventKind
	Text             string
	ReasoningContent string
	ToolCall         *ToolCall
	Usage            Usage
	Error            ProviderError
	RetryAttempt     int    // current retry attempt (1-based, for EventRetrying)
	RetryMax         int    // total retry attempts (for EventRetrying)
	RetryReason      string // reason for retrying (for EventRetrying)
}

type EventKind int

const (
	EventContentDelta EventKind = iota
	EventToolCallStart
	EventToolCallDelta
	EventToolCallEnd
	EventUsage
	EventError
	EventMessageEnd
	EventRetrying
)

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolCallFunc `json:"function"`
}

type ToolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Usage struct {
	InputTokens      int64 `json:"input_tokens"`
	OutputTokens     int64 `json:"output_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
	ReasoningTokens  int64 `json:"reasoning_tokens"`
	CacheReadTokens  int64 `json:"cache_read_tokens"`
	CacheWriteTokens int64 `json:"cache_write_tokens"`
}

type ProviderError struct {
	Code    string
	Message string
}

type Model struct {
	ID          string
	DisplayName string
}

type ToolDef struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type ProviderEngine interface {
	Engine
	StreamChat(ctx context.Context, req ChatRequest) (<-chan ChatEvent, error)
	ListModels(ctx context.Context) ([]Model, error)
}
