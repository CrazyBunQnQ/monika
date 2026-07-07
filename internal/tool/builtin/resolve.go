package builtin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// resolveToolPath validates that p is an absolute path inside projectDir.
func resolveToolPath(p, projectDir string) (string, error) {
	if !filepath.IsAbs(p) {
		return "", fmt.Errorf("filePath must be absolute")
	}
	absPath, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	absProject, err := filepath.Abs(projectDir)
	if err != nil {
		return "", err
	}
	// Resolve symlinks so that different representations of the same
	// directory (e.g. via symlinks or junctions) are treated as equal.
	if real, err := filepath.EvalSymlinks(absProject); err == nil {
		absProject = real
	}
	if real, err := filepath.EvalSymlinks(absPath); err == nil {
		absPath = real
	}
	rel, err := filepath.Rel(absProject, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path %s is outside project directory", p)
	}
	return absPath, nil
}

var mediaExtensions = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true,
	".mp4": true, ".mov": true, ".webm": true, ".mkv": true, ".avi": true, ".m4v": true,
	".pdf": true,
	".mp3": true, ".wav": true, ".flac": true, ".ogg": true, ".m4a": true, ".aac": true,
}

// resolveMediaPath validates that p is an absolute path to an existing file
// with a media extension. Unlike resolveToolPath, it does NOT restrict to the
// project directory — media files may come from anywhere on disk (desktop,
// downloads, etc.) since these tools are read-only.
func resolveMediaPath(p string) (string, error) {
	if !filepath.IsAbs(p) {
		return "", fmt.Errorf("filePath must be absolute")
	}
	ext := strings.ToLower(filepath.Ext(p))
	if !mediaExtensions[ext] {
		return "", fmt.Errorf("unsupported file type: %s (supported: png, jpg, jpeg, webp, gif, mp4, mov, webm, mkv, avi, m4v, pdf, mp3, wav, flac, ogg, m4a, aac)", ext)
	}
	absPath, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("cannot access file: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file: %s", p)
	}
	return absPath, nil
}
