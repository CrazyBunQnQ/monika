package builtin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"monika/internal/tool"
	"monika/pkg/engine"
)

const maxImageBytes = 20 * 1024 * 1024 // 20 MB

type imageUnderstand struct {
	projectDir string
	vision     VisionCaller
}

func NewImageUnderstand(projectDir string, vision VisionCaller) tool.Tool {
	return &imageUnderstand{projectDir: projectDir, vision: vision}
}

func (i *imageUnderstand) Name() string { return "image_understand" }

func (i *imageUnderstand) Description() string {
	return `Analyze an image file (PNG, JPEG, WebP, GIF) inside the project directory and answer a question about it.

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
				"description": "Absolute path to the image file inside the project directory.",
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

	safe, err := resolveToolPath(p.FilePath, tool.ProjectDirOrDefault(ctx, i.projectDir))
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	data, err := os.ReadFile(safe)
	if err != nil {
		return tool.ExecutionResult{Content: "cannot read image: " + err.Error(), IsError: true}, nil
	}
	if len(data) == 0 {
		return tool.ExecutionResult{Content: "image file is empty", IsError: true}, nil
	}
	if len(data) > maxImageBytes {
		return tool.ExecutionResult{
			Content: fmt.Sprintf("image too large: %d bytes (max %d)", len(data), maxImageBytes),
			IsError: true,
		}, nil
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

	if i.vision == nil {
		return tool.ExecutionResult{Content: "vision provider not configured", IsError: true}, nil
	}

	images := []engine.ImageRef{{
		URL:    "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data),
		Detail: detail,
	}}

	resp, err := i.vision(ctx, prompt, images)
	if err != nil {
		return tool.ExecutionResult{Content: "vision call failed: " + err.Error(), IsError: true}, nil
	}
	if resp == "" {
		resp = "(model returned no description)"
	}

	// Wrap in the same JSON shape MediaToolBlock parses, so the frontend
	// can render the description and the original image preview uniformly
	// for both image and video tools. Thumbnails carries the original
	// image as a single-entry list so the preview button has something
	// to click; it is dropped from the LLM-facing JSON in a follow-up
	// patch via GetMediaThumbnails.
	result := imageResult{
		FilePath:  p.FilePath,
		FileName:  filepath.Base(safe),
		MimeType:  mime,
		Size:      int64(len(data)),
		Question:  p.Question,
		Summary:   resp,
		Thumbnail: "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data),
	}
	out, _ := json.Marshal(result)
	return tool.ExecutionResult{Content: string(out)}, nil
}

type imageResult struct {
	FilePath  string `json:"filePath"`
	FileName  string `json:"fileName"`
	MimeType  string `json:"mimeType"`
	Size      int64  `json:"size"`
	Question  string `json:"question,omitempty"`
	Summary   string `json:"summary"`
	Thumbnail string `json:"thumbnail,omitempty"`
}

// detectImageMime returns the MIME type from the file extension, falling back
// to a magic-byte sniff. Empty string means the format is not supported.
func detectImageMime(path string, data []byte) string {
	switch ext := strings.ToLower(filepath.Ext(path)); ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	}
	if len(data) >= 8 {
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
	}
	return ""
}
