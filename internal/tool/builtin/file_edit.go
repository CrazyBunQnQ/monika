package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"strconv"
	"strings"

	"monika/internal/tool"
)

type fileEdit struct {
	projectDir string
}

func NewFileEdit(projectDir string) tool.Tool {
	return &fileEdit{projectDir: projectDir}
}

func (f *fileEdit) Name() string { return "file_edit" }
func (f *fileEdit) Description() string {
	return "Performs exact string replacements in a file. The edit will fail if old_string is not unique in the file. Use replace_all to replace every occurrence. Supports anchor verification (pass 'lineHash:lineNumber' to verify context before editing). Falls back to whitespace-insensitive matching if exact match fails. Refuses to edit files containing merge conflict markers (<<<<<<<, =======, >>>>>>>). Prefer file_edit_hunks for multi-region edits."
}

func (f *fileEdit) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filePath": map[string]any{
				"type":        "string",
				"description": "The absolute path to the file to modify",
			},
			"old_string": map[string]any{
				"type":        "string",
				"description": "The text to replace",
			},
			"new_string": map[string]any{
				"type":        "string",
				"description": "The text to replace it with",
			},
			"replace_all": map[string]any{
				"type":        "boolean",
				"description": "Replace all occurrences of old_string (default false)",
			},
			"anchor": map[string]any{
				"type":        "string",
				"description": "FNV-1 hash of the surrounding context line before old_string, for verification. Format: 'lineHash:lineNumber' (e.g. 'a1b2c3d4:42').",
			},
		},
		"required": []string{"filePath", "old_string", "new_string"},
	}
}

func (f *fileEdit) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var params struct {
		FilePath   string `json:"filePath"`
		OldString  string `json:"old_string"`
		NewString  string `json:"new_string"`
		ReplaceAll bool   `json:"replace_all"`
		Anchor     string `json:"anchor"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	if params.OldString == "" {
		return tool.ExecutionResult{Content: "old_string must not be empty", IsError: true}, nil
	}
	if params.OldString == params.NewString {
		return tool.ExecutionResult{Content: "new_string must be different from old_string", IsError: true}, nil
	}

	safePath, err := f.resolvePath(ctx, params.FilePath)
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	return f.editFile(safePath, params.OldString, params.NewString, params.Anchor, params.ReplaceAll)
}

func (f *fileEdit) resolvePath(ctx context.Context, p string) (string, error) {
	return resolveToolPath(p, tool.ProjectDirOrDefault(ctx, f.projectDir))
}

func (f *fileEdit) editFile(path, oldString, newString, anchor string, replaceAll bool) (tool.ExecutionResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	rawContent := string(data)

	// Conflict marker detection
	if hasConflictMarkers(rawContent) {
		return tool.ExecutionResult{
			Content:   "file contains unresolved merge conflict markers (<<<<<<<, =======, >>>>>>>). Please resolve conflicts before editing.",
			IsError:   true,
			Conflicts: true,
		}, nil
	}

	hadCRLF := strings.Contains(rawContent, "\r\n")

	content := strings.ReplaceAll(rawContent, "\r\n", "\n")
	oldString = strings.ReplaceAll(oldString, "\r\n", "\n")
	newString = strings.ReplaceAll(newString, "\r\n", "\n")

	// Verify anchor hash if provided
	if anchor != "" {
		if err := verifyAnchor(content, oldString, anchor); err != nil {
			return tool.ExecutionResult{Content: fmt.Sprintf("anchor verification failed: %s", err), IsError: true}, nil
		}
	}

	count := strings.Count(content, oldString)
	fuzzyMatch := false

	if count == 0 {
		// Fuzzy whitespace: try matching after normalizing runs of whitespace
		fuzzyOld := normalizeWhitespace(oldString)
		fuzzyCount := 0
		fuzzyIdx := -1
		scanner := &wsFuzzyScanner{content: content}
		for {
			idx, seg := scanner.next()
			if idx < 0 {
				break
			}
			if normalizeWhitespace(seg) == fuzzyOld {
				fuzzyCount++
				if fuzzyIdx < 0 {
					fuzzyIdx = idx
				}
				if fuzzyCount == 1 {
					// Replace the matched segment
					content = content[:idx] + newString + content[idx+len(seg):]
					// After replacement, restart scan (positions shifted)
					scanner = &wsFuzzyScanner{content: content}
					if replaceAll {
						fuzzyCount = 0
						fuzzyIdx = -1
						continue
					}
					break
				}
			}
		}
		if fuzzyCount == 0 {
			snippet := oldString
			if len(snippet) > 80 {
				snippet = snippet[:80] + "..."
			}
			return tool.ExecutionResult{Content: fmt.Sprintf("old_string not found in file: %q", snippet), IsError: true}, nil
		}
		count = fuzzyCount
		fuzzyMatch = true
	}

	if !fuzzyMatch {
		if count > 1 && !replaceAll {
			return tool.ExecutionResult{
				Content: fmt.Sprintf("old_string found %d times in file. Use replace_all to replace all occurrences, or provide a larger string with more surrounding context to make it unique.", count),
				IsError: true,
			}, nil
		}

		result := strings.ReplaceAll(content, oldString, newString)
		content = result
	}

	if hadCRLF {
		content = strings.ReplaceAll(content, "\n", "\r\n")
	}
	if err := os.WriteFile(path, []byte(content), info.Mode()); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	diffLines := computeDiff(path, strings.ReplaceAll(rawContent, "\r\n", "\n"), content)

	suffix := ""
	if fuzzyMatch {
		suffix = " (fuzzy whitespace match)"
	}
	return tool.ExecutionResult{
		Content:   fmt.Sprintf("Replaced %d occurrence(s) in %s%s", count, path, suffix),
		DiffLines: diffLines,
	}, nil
}

// normalizeWhitespace collapses runs of whitespace (spaces, tabs, newlines) into single spaces
// and trims leading/trailing whitespace for fuzzy matching.
func normalizeWhitespace(s string) string {
	var b strings.Builder
	prevSpace := true // trim leading
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		b.WriteRune(r)
		prevSpace = false
	}
	result := b.String()
	if prevSpace && len(result) > 0 {
		result = result[:len(result)-1]
	}
	return result
}

// wsFuzzyScanner slides a window over content to find segments matching a fuzzy pattern.
// It uses the length of the original old_string to determine window size.
type wsFuzzyScanner struct {
	content string
	pos     int
}

func (s *wsFuzzyScanner) next() (int, string) {
	if s.pos >= len(s.content) {
		return -1, ""
	}
	// Advance to next non-whitespace character
	for s.pos < len(s.content) && (s.content[s.pos] == ' ' || s.content[s.pos] == '\t' || s.content[s.pos] == '\n' || s.content[s.pos] == '\r') {
		s.pos++
	}
	if s.pos >= len(s.content) {
		return -1, ""
	}
	start := s.pos
	// Find the end of the current line-stripped segment
	// We look for a chunk bounded by blank lines (paragraph boundary)
	end := s.pos
	for end < len(s.content) {
		if end+1 < len(s.content) && s.content[end] == '\n' && (s.content[end+1] == '\n' || s.content[end+1] == '\r') {
			break
		}
		end++
	}
	s.pos = end
	return start, s.content[start:end]
}

// verifyAnchor checks that the context line at the given line number matches the expected FNV-1 hash.
// anchor format: "hexHash:lineNumber" (1-indexed).
func verifyAnchor(content, oldString, anchor string) error {
	parts := strings.SplitN(anchor, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid anchor format %q, expected 'hash:lineNumber'", anchor)
	}
	expectedHash := parts[0]
	lineNum, err := strconv.Atoi(parts[1])
	if err != nil || lineNum < 1 {
		return fmt.Errorf("invalid line number in anchor %q", parts[1])
	}

	lines := strings.Split(content, "\n")
	if lineNum > len(lines) {
		return fmt.Errorf("anchor line %d exceeds file length %d", lineNum, len(lines))
	}

	// Find the position of old_string to determine which line it starts at
	oldString = strings.ReplaceAll(oldString, "\r\n", "\n")
	idx := strings.Index(content, oldString)
	if idx < 0 {
		return nil // Can't verify without finding old_string; let main logic handle
	}
	anchorLine := lineNum - 1 // 0-indexed
	actualLine := strings.TrimSpace(lines[anchorLine])
	h := fnv.New32a()
	h.Write([]byte(actualLine))
	actualHash := fmt.Sprintf("%08x", h.Sum32())

	if actualHash != expectedHash {
		snippet := actualLine
		if len(snippet) > 60 {
			snippet = snippet[:57] + "..."
		}
		return fmt.Errorf("expected hash %s at line %d but got %s (%q)", expectedHash, lineNum, actualHash, snippet)
	}
	return nil
}

// hasConflictMarkers returns true if content contains git merge conflict markers.
func hasConflictMarkers(content string) bool {
	return (strings.Contains(content, "<<<<<<<") || strings.Contains(content, "=======")) && strings.Contains(content, ">>>>>>>")
}
