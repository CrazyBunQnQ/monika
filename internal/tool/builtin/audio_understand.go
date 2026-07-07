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

const maxAudioBytes = 25 * 1024 * 1024 // 25 MB

type audioUnderstand struct {
	media MediaCaller
}

func NewAudioUnderstand(media MediaCaller) tool.Tool {
	return &audioUnderstand{media: media}
}

func (a *audioUnderstand) Name() string { return "audio_understand" }

func (a *audioUnderstand) Description() string {
	return `Transcribe and analyze an audio file (MP3, WAV, FLAC, OGG, M4A, AAC) on the local filesystem.

Use this when the user:
- references an audio file by path and wants a transcription or summary
- asks a specific question about the content of an audio recording
- wants to know what is said in a voice memo, podcast clip, or meeting recording

The tool reads the file, sends its bytes (base64-encoded) plus the prompt to a
multimodal model, and returns the model's reply as plain text.

Limits: max 25 MB per audio file. Supported formats: mp3, wav, flac, ogg, m4a, aac.`
}

func (a *audioUnderstand) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filePath": map[string]any{
				"type":        "string",
				"description": "Absolute path to the audio file.",
			},
			"question": map[string]any{
				"type":        "string",
				"description": "Specific question about the audio. If omitted, returns a transcription and analysis.",
			},
			"language": map[string]any{
				"type":        "string",
				"description": "Spoken language hint (e.g. \"en\", \"es\") to improve transcription accuracy.",
			},
		},
		"required": []string{"filePath"},
	}
}

func (a *audioUnderstand) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var in struct {
		FilePath string `json:"filePath"`
		Question string `json:"question"`
		Language string `json:"language"`
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
		return tool.ExecutionResult{Content: "cannot stat audio: " + err.Error(), IsError: true}, nil
	}
	if info.Size() > maxAudioBytes {
		return tool.ExecutionResult{
			Content: fmt.Sprintf("audio too large: %d bytes (max %d)", info.Size(), maxAudioBytes),
			IsError: true,
		}, nil
	}

	data, err := os.ReadFile(safe)
	if err != nil {
		return tool.ExecutionResult{Content: "cannot read audio: " + err.Error(), IsError: true}, nil
	}

	mime := detectAudioMime(data)
	if mime == "" {
		return tool.ExecutionResult{
			Content: "unsupported audio format (supported: mp3, wav, flac, ogg, m4a, aac)",
			IsError: true,
		}, nil
	}

	// Best-effort duration detection via ffprobe. If ffprobe is missing or
	// fails, duration is simply omitted from the result (graceful degradation).
	duration, _ := ffprobeDuration(ctx, safe)

	prompt := in.Question
	if prompt == "" {
		prompt = "Transcribe and analyze this audio."
	}
	if in.Language != "" {
		prompt = fmt.Sprintf("Language: %s. %s", in.Language, prompt)
	}

	if a.media == nil {
		return tool.ExecutionResult{Content: "media provider not configured", IsError: true}, nil
	}

	attachments := []engine.AttachmentRef{{
		URL:      "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data),
		MimeType: mime,
	}}

	resp, usage, err := a.media(ctx, prompt, attachments)
	if err != nil {
		return tool.ExecutionResult{Content: "media call failed: " + err.Error(), IsError: true}, nil
	}
	if resp == "" {
		resp = "(model returned no transcription)"
	}

	result := audioResult{
		FilePath:        in.FilePath,
		FileName:        filepath.Base(safe),
		DurationSeconds: duration,
		Question:        in.Question,
		Summary:         resp,
	}
	out, _ := json.Marshal(result)
	return tool.ExecutionResult{Content: string(out), Usage: usage}, nil
}

type audioResult struct {
	FilePath        string  `json:"filePath"`
	FileName        string  `json:"fileName"`
	DurationSeconds float64 `json:"duration_seconds,omitempty"`
	Question        string  `json:"question,omitempty"`
	Summary         string  `json:"summary"`
}

// detectAudioMime returns the MIME type from magic bytes only. An extension
// alone is not enough — a renamed file with arbitrary content must not be
// passed to the model. Empty string means the format is not supported.
func detectAudioMime(data []byte) string {
	if len(data) < 4 {
		return ""
	}
	// MP3: ID3 tag or MPEG audio frame sync.
	if string(data[0:3]) == "ID3" || (data[0] == 0xFF && (data[1]&0xE0) == 0xE0) {
		return "audio/mp3"
	}
	// WAV: RIFF....WAVE container.
	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WAVE" {
		return "audio/wav"
	}
	// FLAC native stream marker.
	if string(data[0:4]) == "fLaC" {
		return "audio/flac"
	}
	// OGG container.
	if string(data[0:4]) == "OggS" {
		return "audio/ogg"
	}
	// M4A: ISO base media with ftyp box.
	if len(data) >= 12 && string(data[4:8]) == "ftyp" && string(data[8:12]) == "M4A " {
		return "audio/mp4"
	}
	// AAC: ADTS frame sync (0xFFF0 mask).
	if len(data) >= 2 && data[0] == 0xFF && (data[1]&0xF6) == 0xF0 {
		return "audio/aac"
	}
	return ""
}
