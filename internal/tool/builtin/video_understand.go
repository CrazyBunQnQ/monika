package builtin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"monika/internal/tool"
	"monika/pkg/engine"
)

const (
	maxVideoBytes  = 500 * 1024 * 1024 // 500 MB
	maxVideoFrames = 128
	maxVideoSecs   = 60 * 60 // 1 hour
)

type videoUnderstand struct {
	projectDir string
	vision     VisionCaller
}

func NewVideoUnderstand(projectDir string, vision VisionCaller) tool.Tool {
	return &videoUnderstand{projectDir: projectDir, vision: vision}
}

func (v *videoUnderstand) Name() string { return "video_understand" }

func (v *videoUnderstand) Description() string {
	return `Analyze a video file (mp4, mov, webm, mkv, avi) inside the project directory.

The tool samples N frames at fixed time intervals, sends them to a multimodal
vision model, and returns a structured analysis as JSON with:
  - summary: short overall description
  - duration_seconds: total video length
  - frame_count: number of frames actually sampled
  - timeline: array of {t, what} entries describing what happens at each sample
  - key_moments: array of {t, title, description} for notable events

Use when the user references a video and wants a summary, asks what happens in
it, or asks about a specific scene.

Prerequisites: ffmpeg and ffprobe must be on PATH.
Limits: max 500 MB, max 60 min duration, max 128 sampled frames per call.`
}

func (v *videoUnderstand) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filePath": map[string]any{
				"type":        "string",
				"description": "Absolute path to the video file inside the project directory.",
			},
			"question": map[string]any{
				"type":        "string",
				"description": "Specific question about the video (optional). If omitted, returns a general summary.",
			},
			"frameInterval": map[string]any{
				"type":        "integer",
				"description": "Seconds between sampled frames. Default 10, range 1-60.",
			},
			"maxFrames": map[string]any{
				"type":        "integer",
				"description": "Maximum frames to sample. Default 32, max 128.",
			},
			"startTime": map[string]any{
				"type":        "number",
				"description": "Start time in seconds (default 0).",
			},
			"endTime": map[string]any{
				"type":        "number",
				"description": "End time in seconds (default full duration).",
			},
		},
		"required": []string{"filePath"},
	}
}

type videoResult struct {
	FilePath        string         `json:"filePath"`
	FileName        string         `json:"fileName"`
	MimeType        string         `json:"mimeType"`
	DurationSeconds float64        `json:"duration_seconds"`
	FrameCount      int            `json:"frame_count"`
	FrameInterval   float64        `json:"frame_interval_seconds"`
	Question        string         `json:"question,omitempty"`
	Summary         string         `json:"summary"`
	Timeline        []timelineItem `json:"timeline"`
	KeyMoments      []keyMoment    `json:"key_moments"`
	// Thumbnails are NOT included here — the LLM has no use for base64
	// JPEGs in the tool result and they'd bloat the conversation by
	// 50-100k tokens per call. The frontend fetches them via
	// App.GetMediaThumbnails separately when the user opens the card.
	Error string `json:"error,omitempty"`
}

type timelineItem struct {
	T    float64 `json:"t"`
	What string  `json:"what"`
}

type keyMoment struct {
	T           float64 `json:"t"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
}

// mediaThumbnail is the wire shape returned by App.GetMediaThumbnails.
type mediaThumbnail struct {
	T   float64 `json:"t"`
	URL string  `json:"url"`
}

func (v *videoUnderstand) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	out, err := v.runAnalysis(ctx, args, nil)
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	return tool.ExecutionResult{Content: out}, nil
}

// ExecuteStreaming returns a channel of progress events so the chat UI
// can render "Sampling frames... 5/32..." in real time instead of a
// frozen spinner. The channel is closed when the analysis completes;
// the agent loop accumulates the EventTextDelta content as the final
// tool result, so the structured JSON still surfaces as the tool output.
func (v *videoUnderstand) ExecuteStreaming(ctx context.Context, args json.RawMessage) (<-chan agent.Event, error) {
	ch := make(chan agent.Event, 32)
	go func() {
		defer close(ch)
		progress := func(msg string) {
			select {
			case ch <- agent.Event{Type: agent.EventTextDelta, Content: msg}:
			case <-ctx.Done():
			}
		}
		out, err := v.runAnalysis(ctx, args, progress)
		if err != nil {
			select {
			case ch <- agent.Event{Type: agent.EventError, Content: err.Error()}:
			case <-ctx.Done():
			}
			return
		}
		// Push the final JSON as one last EventTextDelta so it lands in
		// the tool output. The frontend's MediaToolBlock parses it.
		select {
		case ch <- agent.Event{Type: agent.EventTextDelta, Content: "\n\n" + out}:
		case <-ctx.Done():
		}
	}()
	return ch, nil
}

func (v *videoUnderstand) runAnalysis(ctx context.Context, args json.RawMessage, progress func(string)) (string, error) {
	var p struct {
		FilePath      string  `json:"filePath"`
		Question      string  `json:"question"`
		FrameInterval float64 `json:"frameInterval"`
		MaxFrames     int     `json:"maxFrames"`
		StartTime     float64 `json:"startTime"`
		EndTime       float64 `json:"endTime"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if p.FilePath == "" {
		return "", fmt.Errorf("filePath is required")
	}

	safe, err := resolveToolPath(p.FilePath, tool.ProjectDirOrDefault(ctx, v.projectDir))
	if err != nil {
		return "", err
	}

	info, err := os.Stat(safe)
	if err != nil {
		return "", fmt.Errorf("cannot stat video: %w", err)
	}
	if info.Size() > maxVideoBytes {
		return "", fmt.Errorf("video too large: %d bytes (max %d)", info.Size(), maxVideoBytes)
	}

	// Apply defaults and clamp to sane bounds.
	if p.FrameInterval <= 0 {
		p.FrameInterval = 10
	}
	if p.FrameInterval < 1 {
		p.FrameInterval = 1
	}
	if p.FrameInterval > 60 {
		p.FrameInterval = 60
	}
	if p.MaxFrames <= 0 {
		p.MaxFrames = 32
	}
	if p.MaxFrames > maxVideoFrames {
		p.MaxFrames = maxVideoFrames
	}

	if progress != nil {
		progress(fmt.Sprintf("📹 Reading video metadata (%s)...\n", formatBytes(info.Size())))
	}
	duration, err := ffprobeDuration(ctx, safe)
	if err != nil {
		return "", fmt.Errorf("ffprobe failed (is ffmpeg installed?): %w", err)
	}
	if duration <= 0 {
		return "", fmt.Errorf("video has no detectable duration")
	}
	if duration > maxVideoSecs {
		return "", fmt.Errorf("video too long: %.0fs (max %ds)", duration, maxVideoSecs)
	}

	start := p.StartTime
	if start < 0 {
		start = 0
	}
	end := p.EndTime
	if end <= 0 || end > duration {
		end = duration
	}
	if end <= start {
		return "", fmt.Errorf("endTime (%.2f) must be greater than startTime (%.2f)", end, start)
	}

	timestamps := sampleTimestamps(start, end, p.FrameInterval, p.MaxFrames)
	if len(timestamps) == 0 {
		return "", fmt.Errorf("no frames to sample in the given range")
	}

	if v.vision == nil {
		return "", fmt.Errorf("vision provider not configured")
	}

	if progress != nil {
		progress(fmt.Sprintf("🖼 Sampling %d frames from %s of video (%.0fs total)...\n",
			len(timestamps), formatDuration(end-start), duration))
	}

	tmpDir, err := os.MkdirTemp("", "monika-video-*")
	if err != nil {
		return "", fmt.Errorf("cannot create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	frames, err := extractFrames(ctx, safe, tmpDir, timestamps)
	if err != nil {
		return "", fmt.Errorf("ffmpeg frame extraction failed: %w", err)
	}

	if progress != nil {
		progress(fmt.Sprintf("🤖 Sending %d frames to vision model...\n", len(frames)))
	}

	prompt := buildVideoPrompt(duration, len(frames), p.FrameInterval, p.Question)
	resp, err := v.vision(ctx, prompt, frames)
	if err != nil {
		return "", fmt.Errorf("vision call failed: %w", err)
	}

	result := videoResult{
		FilePath:        p.FilePath,
		FileName:        filepath.Base(safe),
		MimeType:        "video/mp4",
		DurationSeconds: duration,
		FrameCount:      len(frames),
		FrameInterval:   p.FrameInterval,
		Question:        p.Question,
	}
	if parsed := parseModelJSON(resp); parsed != nil {
		if s, ok := parsed["summary"].(string); ok {
			result.Summary = s
		}
		if arr, ok := parsed["timeline"].([]any); ok {
			result.Timeline = parseTimeline(arr)
		}
		if arr, ok := parsed["key_moments"].([]any); ok {
			result.KeyMoments = parseKeyMoments(arr)
		}
	}
	if result.Summary == "" {
		result.Summary = strings.TrimSpace(resp)
		if result.Summary == "" {
			result.Summary = "(model returned no description)"
		}
	}

	if progress != nil {
		progress("✅ Done.\n")
	}

	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("encode result: %w", err)
	}
	return string(out), nil
}

func formatBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	}
	if n < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(n)/1024/1024)
	}
	return fmt.Sprintf("%.2f GB", float64(n)/1024/1024/1024)
}

func formatDuration(seconds float64) string {
	if seconds >= 60 {
		m := int(seconds / 60)
		s := seconds - float64(m*60)
		return fmt.Sprintf("%dm%.0fs", m, s)
	}
	return fmt.Sprintf("%.0fs", seconds)
}

// sampleTimestamps returns a deterministic, evenly-spaced set of timestamps
// (in seconds) between start and end, capped to maxN entries.
func sampleTimestamps(start, end, interval float64, maxN int) []float64 {
	span := end - start
	if span <= 0 {
		return nil
	}
	n := int(span/interval) + 1
	if n > maxN {
		n = maxN
	}
	if n < 1 {
		n = 1
	}
	out := make([]float64, n)
	if n == 1 {
		out[0] = start + span/2
		return out
	}
	step := span / float64(n-1)
	for i := 0; i < n; i++ {
		out[i] = start + step*float64(i)
	}
	return out
}

// ffprobeDuration runs `ffprobe -v error -show_entries format=duration -of json`
// on path and returns the duration in seconds. Returns an error if ffprobe is
// missing or the video is unreadable.
func ffprobeDuration(parentCtx context.Context, path string) (float64, error) {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return 0, fmt.Errorf("ffprobe not found on PATH: %w", err)
	}
	ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "json",
		path,
	).Output()
	if err != nil {
		return 0, err
	}
	var parsed struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		return 0, err
	}
	if parsed.Format.Duration == "" {
		return 0, fmt.Errorf("no duration in ffprobe output: %s", string(out))
	}
	return strconv.ParseFloat(parsed.Format.Duration, 64)
}

// extractFrames runs ffmpeg once per timestamp to grab a single frame,
// re-encoded to a 768px-wide JPEG. The per-frame timeout is derived
// from the parent context so that agent cancellation propagates to
// ffmpeg instead of leaving zombie processes running for up to a
// minute per frame.
//
// Returns the base64-encoded images ready to embed in a multimodal
// message. The on-disk paths are intentionally not returned — the
// thumbnails are now fetched lazily via App.GetMediaThumbnails
// rather than embedded in the tool result.
func extractFrames(parentCtx context.Context, videoPath, tmpDir string, timestamps []float64) ([]engine.ImageRef, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("ffmpeg not found on PATH: %w", err)
	}
	images := make([]engine.ImageRef, 0, len(timestamps))
	for i, t := range timestamps {
		framePath := filepath.Join(tmpDir, fmt.Sprintf("frame-%03d.jpg", i))
		ctx, cancel := context.WithTimeout(parentCtx, 60*time.Second)
		cmd := exec.CommandContext(ctx, "ffmpeg",
			"-ss", strconv.FormatFloat(t, 'f', 3, 64),
			"-i", videoPath,
			"-frames:v", "1",
			"-q:v", "5",
			"-vf", "scale=768:-1",
			"-y",
			framePath,
		)
		out, err := cmd.CombinedOutput()
		cancel()
		if err != nil {
			return nil, fmt.Errorf("ffmpeg at t=%.2fs: %w: %s", t, err, string(out))
		}
		data, err := os.ReadFile(framePath)
		if err != nil {
			return nil, fmt.Errorf("read frame %d: %w", i, err)
		}
		images = append(images, engine.ImageRef{
			URL:    "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(data),
			Detail: "low",
		})
	}
	return images, nil
}

// buildVideoPrompt is the prompt sent to the multimodal model. It instructs
// the model to return structured JSON so the frontend can render a rich card.
func buildVideoPrompt(durationSec float64, frameCount int, intervalSec float64, question string) string {
	q := ""
	if question != "" {
		q = "\n\nUser question: " + question
	}
	return fmt.Sprintf(
		"These are %d sampled key frames from a %.0f-second video (one frame every %.0f seconds, in order). "+
			"Analyze the visual content and reply with ONLY a single JSON object — no markdown, no prose — using this exact shape:\n"+
			`{"summary":"<1-3 sentences overall>","timeline":[{"t":<seconds>,"what":"<one short clause>"}],"key_moments":[{"t":<seconds>,"title":"<short>","description":"<one sentence>"}]}`+
			"%s",
		frameCount, durationSec, intervalSec, q,
	)
}

// parseModelJSON extracts the first balanced JSON object from a model reply.
// Models occasionally wrap the JSON in markdown fences or add leading prose,
// so we try a permissive scan rather than a strict json.Unmarshal on the
// whole string.
func parseModelJSON(s string) map[string]any {
	s = strings.TrimSpace(s)
	// Strip ```json ... ``` fences if present.
	if strings.HasPrefix(s, "```") {
		if i := strings.Index(s, "\n"); i >= 0 {
			s = s[i+1:]
		}
		if j := strings.LastIndex(s, "```"); j >= 0 {
			s = s[:j]
		}
		s = strings.TrimSpace(s)
	}
	// Try the whole string first. We allocate a fresh map per attempt:
	// json.Unmarshal may leave a destination map partially populated
	// even when it returns an error (see encoding/json docs), so reusing
	// the same `direct` across attempts would merge fields from a
	// failed first parse with the second parse's result.
	if m, err := unmarshalIntoFreshMap([]byte(s)); err == nil {
		return m
	}
	// Fallback: find the first '{' and last '}'.
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end <= start {
		return nil
	}
	candidate := s[start : end+1]
	m, err := unmarshalIntoFreshMap([]byte(candidate))
	if err != nil {
		return nil
	}
	return m
}

func unmarshalIntoFreshMap(data []byte) (map[string]any, error) {
	m := make(map[string]any)
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func parseTimeline(arr []any) []timelineItem {
	out := make([]timelineItem, 0, len(arr))
	for _, v := range arr {
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		item := timelineItem{}
		if t, ok := m["t"].(float64); ok {
			item.T = t
		}
		if w, ok := m["what"].(string); ok {
			item.What = w
		}
		if item.What != "" || item.T != 0 {
			out = append(out, item)
		}
	}
	return out
}

func parseKeyMoments(arr []any) []keyMoment {
	out := make([]keyMoment, 0, len(arr))
	for _, v := range arr {
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		km := keyMoment{}
		if t, ok := m["t"].(float64); ok {
			km.T = t
		}
		if s, ok := m["title"].(string); ok {
			km.Title = s
		}
		if s, ok := m["description"].(string); ok {
			km.Description = s
		}
		if km.Title != "" || km.Description != "" {
			out = append(out, km)
		}
	}
	return out
}

// ExtractMediaThumbnails runs ffmpeg to produce up to maxN evenly-spaced
// JPEGs of the given video, returning them as base64 data URLs paired
// with their timestamp. Exposed (capitalized) so the API layer can wire
// it to App.GetMediaThumbnails; the tool's main flow does not embed
// thumbnails in the LLM-facing result.
func ExtractMediaThumbnails(videoPath string, maxN int) ([]mediaThumbnail, error) {
	return extractThumbnails(context.Background(), videoPath, maxN)
}

func extractThumbnails(parentCtx context.Context, videoPath string, maxN int) ([]mediaThumbnail, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("ffmpeg not found on PATH: %w", err)
	}
	if maxN <= 0 {
		maxN = 8
	}
	if maxN > 32 {
		maxN = 32
	}
	duration, err := ffprobeDuration(parentCtx, videoPath)
	if err != nil {
		return nil, err
	}
	if duration <= 0 || maxN == 1 {
		ts := duration / 2
		frame, err := snapshotFrame(videoPath, ts)
		if err != nil {
			return nil, err
		}
		return []mediaThumbnail{{T: ts, URL: frame}}, nil
	}
	out := make([]mediaThumbnail, 0, maxN)
	step := duration / float64(maxN-1)
	for i := 0; i < maxN; i++ {
		ts := step * float64(i)
		frame, err := snapshotFrame(videoPath, ts)
		if err != nil {
			return nil, err
		}
		out = append(out, mediaThumbnail{T: ts, URL: frame})
	}
	return out, nil
}

func snapshotFrame(videoPath string, t float64) (string, error) {
	tmpDir, err := os.MkdirTemp("", "monika-thumb-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)
	framePath := filepath.Join(tmpDir, "thumb.jpg")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "ffmpeg",
		"-ss", strconv.FormatFloat(t, 'f', 3, 64),
		"-i", videoPath,
		"-frames:v", "1",
		"-q:v", "5",
		"-vf", "scale=320:-1",
		"-y",
		framePath,
	).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg at t=%.2fs: %w: %s", t, err, string(out))
	}
	data, err := os.ReadFile(framePath)
	if err != nil {
		return "", err
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(data), nil
}
