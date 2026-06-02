package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"monika/internal/tool"
)

type patchEdit struct {
	projectDir string
}

func NewPatchEdit(projectDir string) tool.Tool {
	return &patchEdit{projectDir: projectDir}
}

func (p *patchEdit) Name() string { return "file_edit_hunks" }
func (p *patchEdit) Description() string {
	return "Apply a patch to a file using unified diff hunk format. Each hunk has a header '@@ -oldStart,oldCount +newStart,newCount @@' followed by context/remove/add lines. More robust than file_edit for multi-region changes."
}
func (p *patchEdit) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filePath": map[string]any{
				"type":        "string",
				"description": "The absolute path to the file to modify",
			},
			"hunks": map[string]any{
				"type":        "string",
				"description": "Unified diff hunks to apply. Format: '@@ -oldStart,oldCount +newStart,newCount @@' followed by ' ' (context), '-' (remove), '+' (add) lines.",
			},
		},
		"required": []string{"filePath", "hunks"},
	}
}

func (p *patchEdit) resolvePath(ctx context.Context, path string) (string, error) {
	return resolveToolPath(path, tool.ProjectDirOrDefault(ctx, p.projectDir))
}

func (p *patchEdit) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var params struct {
		FilePath string `json:"filePath"`
		Hunks    string `json:"hunks"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	safePath, err := p.resolvePath(ctx, params.FilePath)
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	return applyHunks(safePath, params.Hunks)
}

type hunk struct {
	oldStart int
	oldCount int
	newStart int
	newCount int
	lines    []hunkLine
}

type hunkLine struct {
	op   byte // ' ', '-', '+'
	text string
}

func applyHunks(path, patch string) (tool.ExecutionResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	rawContent := string(data)

	if hasConflictMarkers(rawContent) {
		return tool.ExecutionResult{
			Content:   "file contains unresolved merge conflict markers. Resolve conflicts first.",
			IsError:   true,
			Conflicts: true,
		}, nil
	}

	hadCRLF := strings.Contains(rawContent, "\r\n")
	content := strings.ReplaceAll(rawContent, "\r\n", "\n")

	hunks, err := parseHunks(patch)
	if err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("parse error: %s", err), IsError: true}, nil
	}
	if len(hunks) == 0 {
		return tool.ExecutionResult{Content: "no hunks found in patch", IsError: true}, nil
	}

	lines := strings.Split(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	// Apply hunks in reverse order to preserve line positions
	for i := len(hunks) - 1; i >= 0; i-- {
		h := hunks[i]
		start := h.oldStart - 1 // 0-indexed
		if start < 0 {
			start = 0
		}

		// Verify context lines and build replacement in a single pass
		var replacement []string
		fileIdx := 0
		for _, hl := range h.lines {
			switch hl.op {
			case ' ':
				idx := start + fileIdx
				if idx >= len(lines) {
					return tool.ExecutionResult{
						Content: fmt.Sprintf("hunk %d: context line %d exceeds file (file has %d lines)", i+1, idx+1, len(lines)),
						IsError: true,
					}, nil
				}
				if strings.TrimSpace(lines[idx]) != strings.TrimSpace(hl.text) {
					return tool.ExecutionResult{
						Content: fmt.Sprintf("hunk %d: context mismatch at line %d\n  expected: %s\n  actual:   %s", i+1, idx+1, truncateLine(hl.text, 80), truncateLine(lines[idx], 80)),
						IsError: true,
					}, nil
				}
				replacement = append(replacement, lines[idx])
				fileIdx++
			case '-':
				fileIdx++
			case '+':
				replacement = append(replacement, hl.text)
			}
		}

		// Replace the section: lines[start..start+oldCount-1] → replacement
		end := start + h.oldCount
		if end > len(lines) {
			end = len(lines)
		}
		newLines := make([]string, 0, len(lines)-h.oldCount+len(replacement))
		newLines = append(newLines, lines[:start]...)
		newLines = append(newLines, replacement...)
		newLines = append(newLines, lines[end:]...)
		lines = newLines
	}

	result := strings.Join(lines, "\n")
	if hadCRLF {
		result = strings.ReplaceAll(result, "\n", "\r\n")
	}
	if err := os.WriteFile(path, []byte(result), info.Mode()); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	diffLines := computeDiff(path, strings.ReplaceAll(rawContent, "\r\n", "\n"), strings.ReplaceAll(result, "\r\n", "\n"))
	return tool.ExecutionResult{
		Content:   fmt.Sprintf("Applied %d hunk(s) to %s", len(hunks), path),
		DiffLines: diffLines,
	}, nil
}

func parseHunks(patch string) ([]hunk, error) {
	patch = strings.ReplaceAll(patch, "\r\n", "\n")
	lines := strings.Split(patch, "\n")

	var hunks []hunk
	var current *hunk

	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			if current != nil {
				hunks = append(hunks, *current)
			}
			h, err := parseHunkHeader(line)
			if err != nil {
				return nil, err
			}
			current = &h
			continue
		}
		if current == nil {
			continue
		}
		if len(line) == 0 {
			continue
		}
		op := line[0]
		switch op {
		case ' ', '-', '+':
			current.lines = append(current.lines, hunkLine{op: op, text: line[1:]})
		case '\\':
			// "\ No newline at end of file" — ignore
		default:
			// Skip non-diff lines (e.g., "--- a/file", "+++ b/file")
		}
	}
	if current != nil {
		hunks = append(hunks, *current)
	}
	return hunks, nil
}

func parseHunkHeader(line string) (hunk, error) {
	// @@ -oldStart,oldCount +newStart,newCount @@
	s := line
	atIdx := strings.Index(s, "@@")
	if atIdx < 0 {
		return hunk{}, fmt.Errorf("invalid hunk header: %s", line)
	}
	rest := s[atIdx+2:]
	atIdx2 := strings.Index(rest, "@@")
	if atIdx2 < 0 {
		return hunk{}, fmt.Errorf("invalid hunk header: %s", line)
	}
	body := strings.TrimSpace(rest[:atIdx2])

	// Parse -oldStart,oldCount +newStart,newCount
	parts := strings.Fields(body)
	if len(parts) < 2 {
		return hunk{}, fmt.Errorf("invalid hunk header body: %s", body)
	}

	oldPart := strings.TrimPrefix(parts[0], "-")
	newPart := strings.TrimPrefix(parts[1], "+")

	os, oc := parseRange(oldPart)
	ns, nc := parseRange(newPart)

	return hunk{oldStart: os, oldCount: oc, newStart: ns, newCount: nc}, nil
}

func parseRange(s string) (start, count int) {
	parts := strings.SplitN(s, ",", 2)
	start, _ = strconv.Atoi(parts[0])
	if len(parts) == 2 {
		count, _ = strconv.Atoi(parts[1])
	} else {
		count = 1
	}
	return
}
