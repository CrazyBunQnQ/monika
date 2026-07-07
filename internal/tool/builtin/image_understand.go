package builtin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"monika/internal/tool"
	"monika/pkg/engine"
)

const maxImageBytes = 20 * 1024 * 1024 // 20 MB

type imageUnderstand struct {
	media MediaCaller
}

func NewImageUnderstand(media MediaCaller) tool.Tool {
	return &imageUnderstand{media: media}
}

func (i *imageUnderstand) Name() string { return "image_understand" }

func (i *imageUnderstand) Description() string {
	return `Analyze an image file (PNG, JPEG, WebP, GIF) on the local filesystem and answer a question about it.

Use this when the user:
- pastes a screenshot, diagram, photo, or UI mock-up and asks for a description or analysis
- references an image by path and wants to know what's in it
- asks a specific question about the visual content of an image

The tool reads the file, sends its bytes (base64-encoded) plus the prompt to a
multimodal vision model, and returns the model's reply as plain text. There
is no separate transcription step — one round-trip per call.

Limits: max 20 MB per image. Supported formats: png, jpg, jpeg, webp, gif (static only).`
}

func (i *imageUnderstand) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filePath": map[string]any{
				"type":        "string",
				"description": "Absolute path to the image file.",
			},
			"question": map[string]any{
				"type":        "string",
				"description": "Specific question about the image. If omitted, returns a general description.",
			},
			"detail": map[string]any{
				"type":        "string",
				"enum":        []string{"auto", "low", "high"},
				"description": "Image processing detail level. 'high' uses more tokens but reads fine text better. Default: auto.",
			},
		},
		"required": []string{"filePath"},
	}
}

func (i *imageUnderstand) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var p struct {
		FilePath string `json:"filePath"`
		Question string `json:"question"`
		Detail   string `json:"detail"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return tool.ExecutionResult{Content: "invalid arguments: " + err.Error(), IsError: true}, nil
	}
	if p.FilePath == "" {
		return tool.ExecutionResult{Content: "filePath is required", IsError: true}, nil
	}

	safe, err := resolveMediaPath(p.FilePath)
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	// Stat first so a multi-GB file never reaches the read buffer.
	// Reading before size-checking turned image_understand into a
	// trivial memory-exhaustion vector — a 4GB hostile image would
	// get os.ReadFile'd in full before being rejected.
	info, err := os.Stat(safe)
	if err != nil {
		return tool.ExecutionResult{Content: "cannot stat image: " + err.Error(), IsError: true}, nil
	}
	if info.Size() > maxImageBytes {
		return tool.ExecutionResult{
			Content: fmt.Sprintf("image too large: %d bytes (max %d)", info.Size(), maxImageBytes),
			IsError: true,
		}, nil
	}

	data, err := os.ReadFile(safe)
	if err != nil {
		return tool.ExecutionResult{Content: "cannot read image: " + err.Error(), IsError: true}, nil
	}
	if len(data) == 0 {
		return tool.ExecutionResult{Content: "image file is empty", IsError: true}, nil
	}

	mime := detectImageMime(safe, data)
	if mime == "" {
		return tool.ExecutionResult{
			Content: "unsupported image format (supported: png, jpg, webp, gif)",
			IsError: true,
		}, nil
	}

	prompt := p.Question
	if prompt == "" {
		prompt = "Describe this image in detail."
	}
	detail := p.Detail
	if detail == "" {
		detail = "auto"
	}

	if i.media == nil {
		return tool.ExecutionResult{Content: "vision provider not configured", IsError: true}, nil
	}

	images := []engine.AttachmentRef{{
		URL:      "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data),
		Detail:   detail,
		MimeType: mime,
	}}

	resp, usage, err := i.media(ctx, prompt, images)
	if err != nil {
		return tool.ExecutionResult{Content: "vision call failed: " + err.Error(), IsError: true}, nil
	}
	if resp == "" {
		resp = "(model returned no description)"
	}

	// Wrap in the same JSON shape MediaToolBlock parses, so the frontend
	// can render the description and the original image preview uniformly
	// for both image and video tools. The thumbnail is intentionally
	// NOT included in the JSON — image_understand returns to the LLM
	// only the textual summary, mirroring the video_understand shape
	// after commit 5e7093c. The frontend fetches the inline preview
	// image via /__media__?path=... on demand when the card renders.
	result := imageResult{
		FilePath: p.FilePath,
		FileName: filepath.Base(safe),
		MimeType: mime,
		Size:     info.Size(),
		Question: p.Question,
		Summary:  resp,
	}
	out, _ := json.Marshal(result)
	return tool.ExecutionResult{Content: string(out), Usage: usage}, nil
}

type imageResult struct {
	FilePath string `json:"filePath"`
	FileName string `json:"fileName"`
	MimeType string `json:"mimeType"`
	Size     int64  `json:"size"`
	Question string `json:"question,omitempty"`
	Summary  string `json:"summary"`
}

// detectImageMime returns the MIME type from magic bytes only. An
// extension alone is not enough — a renamed file with arbitrary
// content must not be passed to the vision model. Empty string
// means the format is not supported.
func detectImageMime(path string, data []byte) string {
	_ = path
	if len(data) < 8 {
		return ""
	}
	if data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
		return "image/png"
	}
	if data[0] == 0xFF && data[1] == 0xD8 {
		return "image/jpeg"
	}
	if string(data[0:4]) == "GIF8" {
		return "image/gif"
	}
	if string(data[0:4]) == "RIFF" && len(data) >= 12 && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}
	return ""
}
