// Smoke test for video_understand + image_understand end-to-end.
// Runs against the internal OpenAI-compatible endpoint.
//
// Usage:
//
//	go run ./cmd/smoketest --video=/tmp/sample.mp4
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	oaip "monika/internal/engines/provider/openai"
	"monika/internal/tool"
	"monika/internal/tool/builtin"
	"monika/pkg/engine"
)

const (
	defaultBaseURL = "http://192.168.0.8:8000/v1"
	defaultAPIKey  = "sk-luoan-security-ai-llm"
	defaultModel   = "LuoAnMax"
	defaultVideo   = "/tmp/sample.mp4"
)

func main() {
	videoPath := flag.String("video", defaultVideo, "path to a sample mp4 file")
	baseURL := flag.String("base", defaultBaseURL, "OpenAI-compatible base URL")
	apiKey := flag.String("key", defaultAPIKey, "API key")
	model := flag.String("model", defaultModel, "model name")
	flag.Parse()

	if _, err := os.Stat(*videoPath); err != nil {
		fmt.Fprintf(os.Stderr, "video not found: %v\n", err)
		os.Exit(1)
	}
	abs, err := filepath.Abs(*videoPath)
	if err != nil {
		fatal(err)
	}

	// Single-provider engine map.
	provider := newOpenAIProvider(*baseURL, *apiKey, *model)
	providers := map[string]engine.ProviderEngine{"smoke": provider}
	media := builtin.NewDefaultMediaCaller(providers)

	dir := filepath.Dir(abs)

	// ---------- video_understand ----------
	fmt.Println("=== video_understand ===")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctx = tool.WithProjectDir(ctx, dir)
	ctx = tool.WithProvider(ctx, "smoke")
	ctx = tool.WithModel(ctx, *model)

	v := builtin.NewVideoUnderstand(media)
	args, _ := json.Marshal(map[string]any{
		"filePath":      abs,
		"question":      "What does this test video show? Be brief.",
		"frameInterval": 1,
		"maxFrames":     3,
	})
	start := time.Now()
	res, err := v.Execute(ctx, json.RawMessage(args))
	elapsed := time.Since(start)
	if err != nil {
		fatal(fmt.Errorf("video_understand.Execute: %w", err))
	}
	if res.IsError {
		fatal(fmt.Errorf("video_understand returned error: %s", res.Content))
	}
	fmt.Printf("elapsed: %s\n", elapsed)
	fmt.Printf("output: %s\n", truncate(res.Content, 2000))

	var parsed map[string]any
	if err := json.Unmarshal([]byte(res.Content), &parsed); err != nil {
		fatal(fmt.Errorf("parse video JSON: %w", err))
	}
	reportVideo("video_understand", parsed, res.Usage != nil)

	// ---------- image_understand ----------
	fmt.Println()
	fmt.Println("=== image_understand ===")
	imgPath, err := extractFirstFrame(*videoPath)
	if err != nil {
		fatal(fmt.Errorf("extract frame: %w", err))
	}
	defer func() {
		os.Remove(imgPath)
		os.Remove(filepath.Dir(imgPath))
	}()
	fmt.Printf("frame file: %s\n", imgPath)

	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel2()
	ctx2 = tool.WithProjectDir(ctx2, dir)
	ctx2 = tool.WithProvider(ctx2, "smoke")
	ctx2 = tool.WithModel(ctx2, *model)

	img := builtin.NewImageUnderstand(media)
	imgArgs, _ := json.Marshal(map[string]any{
		"filePath": imgPath,
		"question": "Describe this image in one sentence.",
	})
	start = time.Now()
	res2, err := img.Execute(ctx2, json.RawMessage(imgArgs))
	elapsed = time.Since(start)
	if err != nil {
		fatal(fmt.Errorf("image_understand.Execute: %w", err))
	}
	if res2.IsError {
		fatal(fmt.Errorf("image_understand returned error: %s", res2.Content))
	}
	fmt.Printf("elapsed: %s\n", elapsed)
	fmt.Printf("output: %s\n", truncate(res2.Content, 2000))

	var parsedImg map[string]any
	if err := json.Unmarshal([]byte(res2.Content), &parsedImg); err != nil {
		fatal(fmt.Errorf("parse image JSON: %w", err))
	}
	reportImage("image_understand", parsedImg, res2.Usage != nil)

	fmt.Println()
	fmt.Println("=== ALL SMOKE TESTS PASSED ===")
}

func reportVideo(name string, p map[string]any, hasUsage bool) {
	fmt.Printf("%s fields:\n", name)
	for _, k := range []string{"filePath", "fileName", "mimeType", "duration_seconds", "frame_count", "summary"} {
		v, ok := p[k]
		fmt.Printf("  %-20s = %v (present=%v)\n", k, truncate(fmt.Sprintf("%v", v), 80), ok)
	}
	if ts, ok := p["timeline"].([]any); ok {
		fmt.Printf("  timeline             = %d entries\n", len(ts))
	}
	if km, ok := p["key_moments"].([]any); ok {
		fmt.Printf("  key_moments          = %d entries\n", len(km))
	}
	fmt.Printf("  usage event captured = %v\n", hasUsage)
}

func reportImage(name string, p map[string]any, hasUsage bool) {
	fmt.Printf("%s fields:\n", name)
	for _, k := range []string{"filePath", "fileName", "mimeType", "size", "summary"} {
		v, ok := p[k]
		fmt.Printf("  %-20s = %v (present=%v)\n", k, truncate(fmt.Sprintf("%v", v), 80), ok)
	}
	fmt.Printf("  usage event captured = %v\n", hasUsage)
}

func extractFirstFrame(videoPath string) (string, error) {
	// Place the frame inside the same directory as the source video
	// so image_understand's project-scoped resolveToolPath accepts it
	// (the tool refuses paths outside the project directory).
	dir := filepath.Join(filepath.Dir(videoPath), ".smoke-frame-tmp")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	out := filepath.Join(dir, "frame.jpg")
	ffmpeg := os.Getenv("FFMPEG_BIN")
	if ffmpeg == "" {
		ffmpeg = "/d/ffmpeg/bin/ffmpeg"
	}
	cmd := exec.Command(ffmpeg, "-y", "-ss", "0.5", "-i", videoPath, "-frames:v", "1",
		"-q:v", "5", "-vf", "scale=512:-1", out)
	if outb, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ffmpeg: %w: %s", err, string(outb))
	}
	return out, nil
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "FATAL: %v\n", err)
	os.Exit(1)
}

// newOpenAIProvider wires the existing internal OpenAI engine to the
// smoke-test's settings. We bypass bootstrap so the test stays a
// single self-contained binary — no config files, no provider map.
func newOpenAIProvider(baseURL, apiKey, model string) engine.ProviderEngine {
	p := &oaip.OpenAIProvider{}
	_ = p.Init(context.Background(), map[string]any{
		"base_url": baseURL,
		"api_key":  apiKey,
		"models": []map[string]any{
			{"id": model, "name": model},
		},
	})
	return p
}
