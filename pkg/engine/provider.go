package engine

import (
	"context"
	"encoding/json"
	"strings"
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
	Attachments      []AttachmentRef `json:"attachments,omitempty"`
	ReasoningContent string          `json:"reasoning_content"`
	ToolCalls        []ToolCall      `json:"tool_calls,omitempty"`
	ToolCallID       string          `json:"tool_call_id,omitempty"`
	Name             string          `json:"name,omitempty"`
	TokenUsage       *Usage          `json:"token_usage,omitempty"`
	QuotedMessages   []QuotedMessage `json:"quoted_messages,omitempty"`
}

// AttachmentRef is a reference to a media attachment (image, audio, or PDF)
// attached to a chat message. URL may be either a data URL
// (e.g. data:image/jpeg;base64,...) or an https URL. Detail hints at the
// model's processing resolution: "auto" | "low" | "high" (images only).
type AttachmentRef struct {
	URL      string `json:"url,omitempty"`
	Detail   string `json:"detail,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}

// MarshalJSON renders the message content either as a plain string (the
// historical behavior, used for text-only messages) or as an OpenAI-style
// multipart array when one or more attachments are present. This keeps the wire
// format backward-compatible: messages without Attachments still serialize to
// "role":"user","content":"hello", exactly as before.
func (m ChatMessage) MarshalJSON() ([]byte, error) {
	type alias ChatMessage
	if len(m.Attachments) == 0 {
		return json.Marshal(alias(m))
	}
	parts := make([]map[string]any, 0, 1+len(m.Attachments))
	if m.Content != "" {
		parts = append(parts, map[string]any{"type": "text", "text": m.Content})
	}
	for _, att := range m.Attachments {
		part := attachmentPart(att)
		if part != nil {
			parts = append(parts, part)
		}
	}
	// Marshal the alias to get all ChatMessage fields, then override content
	raw, err := json.Marshal(alias(m))
	if err != nil {
		return nil, err
	}
	result := make(map[string]any)
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	result["content"] = parts
	return json.Marshal(result)
}

// attachmentPart renders a single AttachmentRef into the OpenAI-style content
// part expected by multipart requests, dispatching on MIME type.
func attachmentPart(att AttachmentRef) map[string]any {
	mime := att.MimeType
	switch {
	case strings.HasPrefix(mime, "image/"):
		entry := map[string]any{
			"type":      "image_url",
			"image_url": map[string]any{"url": att.URL},
		}
		if att.Detail != "" {
			entry["image_url"].(map[string]any)["detail"] = att.Detail
		}
		return entry
	case strings.HasPrefix(mime, "audio/"):
		// format is the subtype, e.g. "mp3" from "audio/mp3".
		format := strings.TrimPrefix(mime, "audio/")
		switch format {
		case "mp3", "wav", "flac", "ogg":
			// supported by OpenAI input_audio format
		default:
			// Unsupported audio format (e.g. mp4/m4a/aac): fall back to
			// image_url with the data URL. Most providers can still process
			// the audio bytes from a data URL even though the OpenAI
			// input_audio format doesn't accept this codec.
			return map[string]any{
				"type":      "image_url",
				"image_url": map[string]any{"url": att.URL},
			}
		}
		data := att.URL
		// For data URLs, extract just the base64 payload after the comma.
		if idx := strings.Index(att.URL, ","); idx >= 0 {
			data = att.URL[idx+1:]
		}
		return map[string]any{
			"type":        "input_audio",
			"input_audio": map[string]any{"data": data, "format": format},
		}
	case mime == "application/pdf":
		return map[string]any{
			"type": "file",
			"file": map[string]any{"file_data": att.URL},
		}
	default:
		// Unknown MIME type: skip this attachment.
		return nil
	}
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
