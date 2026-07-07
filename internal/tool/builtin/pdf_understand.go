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

const maxPDFBytes = 50 * 1024 * 1024 // 50 MB

type pdfUnderstand struct {
	media MediaCaller
}

func NewPdfUnderstand(media MediaCaller) tool.Tool {
	return &pdfUnderstand{media: media}
}

func (p *pdfUnderstand) Name() string { return "pdf_understand" }

func (p *pdfUnderstand) Description() string {
	return `Analyze a PDF document on the local filesystem and provide a detailed summary of its contents.

Use this when the user:
- references a PDF by path and wants to know what's in it
- asks a specific question about the content of a PDF document
- wants a summary, extraction, or analysis of text, tables, or figures in a PDF

The tool reads the file, sends its bytes (base64-encoded) plus the prompt to a
multimodal model, and returns the model's reply as plain text.

Limits: max 50 MB per PDF.`
}

func (p *pdfUnderstand) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filePath": map[string]any{
				"type":        "string",
				"description": "Absolute path to the PDF file.",
			},
			"question": map[string]any{
				"type":        "string",
				"description": "Specific question about the PDF. If omitted, returns a general summary.",
			},
		},
		"required": []string{"filePath"},
	}
}

func (p *pdfUnderstand) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var in struct {
		FilePath string `json:"filePath"`
		Question string `json:"question"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return tool.ExecutionResult{Content: "invalid arguments: " + err.Error(), IsError: true}, nil
	}
	if in.FilePath == "" {
		return tool.ExecutionResult{Content: "filePath is required", IsError: true}, nil
	}

	safe, err := resolveMediaPath(in.FilePath)
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	info, err := os.Stat(safe)
	if err != nil {
		return tool.ExecutionResult{Content: "cannot stat pdf: " + err.Error(), IsError: true}, nil
	}
	if info.Size() > maxPDFBytes {
		return tool.ExecutionResult{
			Content: fmt.Sprintf("pdf too large: %d bytes (max %d)", info.Size(), maxPDFBytes),
			IsError: true,
		}, nil
	}

	data, err := os.ReadFile(safe)
	if err != nil {
		return tool.ExecutionResult{Content: "cannot read pdf: " + err.Error(), IsError: true}, nil
	}
	if len(data) < 4 || string(data[0:4]) != "%PDF" {
		return tool.ExecutionResult{Content: "unsupported file format (not a valid PDF)", IsError: true}, nil
	}

	prompt := in.Question
	if prompt == "" {
		prompt = "Analyze this PDF document and provide a detailed summary of its contents."
	}

	if p.media == nil {
		return tool.ExecutionResult{Content: "media provider not configured", IsError: true}, nil
	}

	attachments := []engine.AttachmentRef{{
		URL:      "data:application/pdf;base64," + base64.StdEncoding.EncodeToString(data),
		MimeType: "application/pdf",
	}}

	resp, usage, err := p.media(ctx, prompt, attachments)
	if err != nil {
		return tool.ExecutionResult{Content: "media call failed: " + err.Error(), IsError: true}, nil
	}
	if resp == "" {
		resp = "(model returned no summary)"
	}

	result := pdfResult{
		FilePath: in.FilePath,
		FileName: filepath.Base(safe),
		Question: in.Question,
		Summary:  resp,
	}
	out, _ := json.Marshal(result)
	return tool.ExecutionResult{Content: string(out), Usage: usage}, nil
}

type pdfResult struct {
	FilePath string `json:"filePath"`
	FileName string `json:"fileName"`
	Question string `json:"question,omitempty"`
	Summary  string `json:"summary"`
}
