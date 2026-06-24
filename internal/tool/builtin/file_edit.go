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

// LSPDiagFunc runs LSP diagnostics on a file after modification.
// Returns additional content to append to the tool result, or empty string.
type LSPDiagFunc func(ctx context.Context, filePath string) string

type fileEdit struct {
	projectDir string
	diagFunc   LSPDiagFunc
}

func NewFileEdit(projectDir string) tool.Tool {
	return &fileEdit{projectDir: projectDir}
}

// SetDiagFunc sets the optional LSP diagnostics hook.
func (f *fileEdit) SetDiagFunc(fn LSPDiagFunc) { f.diagFunc = fn }

func (f *fileEdit) Name() string { return "file_edit" }

func (f *fileEdit) Description() string {
	return "Replaces lines in a file using line-number positioning with hash verification. The anchor (from file_read output, format 'hash:lineNumber') identifies the starting line and verifies it has not changed. line_count specifies how many lines to replace (default 1). Set line_count to 0 to insert new_string after the anchor line without replacing anything. The new_string parameter is REQUIRED. Refuses to edit files containing merge conflict markers. WARNING: do NOT call file_edit on the same file in parallel — LSP diagnostics triggered by one edit can lock the file and cause the other to fail. Serialize multiple edits to the same file."
}

func (f *fileEdit) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filePath": map[string]any{
				"type":        "string",
				"description": "The absolute path to the file to modify",
			},
			"anchor": map[string]any{
				"type":        "string",
				"description": "Content hash and line number from file_read output. Format: 'hash:lineNumber' (e.g. 'a1b2c3:42'). The hash is verified against the current line content to ensure the file has not changed.",
			},
			"new_string": map[string]any{
				"type":        "string",
				"description": "The replacement text. REQUIRED — must always be provided explicitly.",
			},
			"line_count": map[string]any{
				"type":        "integer",
				"description": "Number of lines to replace starting from the anchor line. Default is 1. Set to 0 to insert new_string after the anchor line without deleting any lines.",
			},
		},
		"required": []string{"filePath", "anchor", "new_string"},
	}
}

func (f *fileEdit) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var params struct {
		FilePath  string `json:"filePath"`
		Anchor    string `json:"anchor"`
		NewString string `json:"new_string"`
		LineCount *int   `json:"line_count"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	if params.Anchor == "" {
		return tool.ExecutionResult{Content: "anchor must not be empty", IsError: true}, nil
	}

	lineCount := 1
	if params.LineCount != nil {
		if *params.LineCount < 0 {
			return tool.ExecutionResult{Content: "line_count must be >= 0", IsError: true}, nil
		}
		lineCount = *params.LineCount
	}

	safePath, err := f.resolvePath(ctx, params.FilePath)
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	if tool.IsFileDirty(safePath) {
		diskData, _ := os.ReadFile(safePath)
		diskContent := string(diskData)
		aiContent, applyErr := applyEditToContent(diskContent, params.Anchor, params.NewString, lineCount)
		if applyErr != nil {
			return tool.ExecutionResult{Content: applyErr.Error(), IsError: true}, nil
		}
		diff := computeDiff(safePath, diskContent, aiContent)
		return tool.ExecutionResult{
			Content:     fmt.Sprintf("⚠ %s has unsaved user edits. Choose Accept AI or Keep Mine in preview.", safePath),
			Conflict:    true,
			DiskContent: diskContent,
			AiContent:   aiContent,
			DiffLines:   diff,
		}, nil
	}

	result, err := f.editFile(safePath, params.Anchor, params.NewString, lineCount)
	if err != nil {
		return result, err
	}
	if !result.IsError && f.diagFunc != nil {
		result.Content += f.diagFunc(ctx, safePath)
	}
	return result, nil
}

func (f *fileEdit) resolvePath(ctx context.Context, p string) (string, error) {
	return resolveToolPath(p, tool.ProjectDirOrDefault(ctx, f.projectDir))
}

func (f *fileEdit) editFile(path, anchor, newString string, lineCount int) (tool.ExecutionResult, error) {
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
			Content:   "file contains unresolved merge conflict markers. Please resolve conflicts before editing.",
			IsError:   true,
			Conflicts: true,
		}, nil
	}

	content := strings.ReplaceAll(rawContent, "\r\n", "\n")
	newString = strings.ReplaceAll(newString, "\r\n", "\n")

	// Parse anchor: "hash:lineNumber"
	colonIdx := strings.LastIndex(anchor, ":")
	if colonIdx < 0 {
		return tool.ExecutionResult{Content: fmt.Sprintf("invalid anchor format %q, expected 'hash:lineNumber'", anchor), IsError: true}, nil
	}
	expectedHash := anchor[:colonIdx]
	lineNumStr := anchor[colonIdx+1:]
	lineNum, err := strconv.Atoi(lineNumStr)
	if err != nil || lineNum < 1 {
		return tool.ExecutionResult{Content: fmt.Sprintf("invalid anchor line number %q", lineNumStr), IsError: true}, nil
	}

	lines := strings.Split(content, "\n")
	if lineNum > len(lines) {
		return tool.ExecutionResult{
			Content: fmt.Sprintf("anchor line %d is beyond file length (%d lines)", lineNum, len(lines)),
			IsError: true,
		}, nil
	}

	// Verify hash of the anchor line
	actualHash := lineHash(lines[lineNum-1])
	if actualHash != expectedHash {
		snippet := strings.TrimSpace(lines[lineNum-1])
		if len(snippet) > 60 {
			snippet = snippet[:57] + "..."
		}
		return tool.ExecutionResult{
			Content: fmt.Sprintf("anchor hash mismatch at line %d: expected %s but got %s (%q). Re-read the file to get the current content.", lineNum, expectedHash, actualHash, snippet),
			IsError: true,
		}, nil
	}

	// Also verify that all lines to be replaced (if lineCount > 0) still exist
	if lineCount > 0 && lineNum+lineCount-1 > len(lines) {
		return tool.ExecutionResult{
			Content: fmt.Sprintf("line_count %d starting at line %d exceeds file length (%d lines)", lineCount, lineNum, len(lines)),
			IsError: true,
		}, nil
	}

	rawBefore := content
	var applyErr error
	content, applyErr = applyEditToContent(rawContent, anchor, newString, lineCount)
	if applyErr != nil {
		return tool.ExecutionResult{Content: applyErr.Error(), IsError: true}, nil
	}
	if err := os.WriteFile(path, []byte(content), info.Mode()); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	diffLines := computeDiff(path, rawBefore, content)

	action := "Replaced"
	detail := fmt.Sprintf("%d line(s)", lineCount)
	if lineCount == 0 {
		action = "Inserted after"
		detail = fmt.Sprintf("line %d", lineNum)
	}
	balanceWarn := checkBracketBalanceDelta(rawBefore, content)
	resultText := fmt.Sprintf("%s %s in %s", action, detail, path)
	if balanceWarn != "" {
		resultText += "\n⚠ " + balanceWarn
	}
	if len(diffLines) > 0 {
		resultText += "\n" + strings.Join(diffLines, "\n")
	}
	return tool.ExecutionResult{
		Content:   resultText,
		DiffLines: diffLines,
	}, nil
}

// applyEditToContent applies a line-based edit to the given content string
// and returns the new content. It does NOT write to disk.
func applyEditToContent(content, anchor, newString string, lineCount int) (string, error) {
	hadCRLF := strings.Contains(content, "\r\n")
	content = strings.ReplaceAll(content, "\r\n", "\n")
	newString = strings.ReplaceAll(newString, "\r\n", "\n")

	colonIdx := strings.LastIndex(anchor, ":")
	if colonIdx < 0 {
		return "", fmt.Errorf("invalid anchor format")
	}
	lineNum, err := strconv.Atoi(anchor[colonIdx+1:])
	if err != nil || lineNum < 1 {
		return "", fmt.Errorf("invalid anchor line number")
	}

	lines := strings.Split(content, "\n")
	if lineNum > len(lines) {
		return "", fmt.Errorf("anchor line beyond file length")
	}

	var result []string
	if lineCount == 0 {
		before := lines[:lineNum]
		after := lines[lineNum:]
		insertLines := splitLines(newString)
		result = make([]string, 0, len(before)+len(insertLines)+len(after))
		result = append(result, before...)
		result = append(result, insertLines...)
		result = append(result, after...)
	} else {
		before := lines[:lineNum-1]
		after := lines[lineNum+lineCount-1:]
		replacementLines := splitLines(newString)
		result = make([]string, 0, len(before)+len(replacementLines)+len(after))
		result = append(result, before...)
		result = append(result, replacementLines...)
		result = append(result, after...)
	}

	content = strings.Join(result, "\n")
	if hadCRLF {
		content = strings.ReplaceAll(content, "\n", "\r\n")
	}
	return content, nil
}

// checkBracketBalanceDelta detects whether an edit changed the bracket
// balance of the entire file. Unlike checking new_string in isolation
// (which false-positives on valid partial edits), this compares the net
// bracket count of the whole file before and after. A non-zero delta
// strongly suggests broken structure — e.g. a '}' was removed without
// its matching '{'.
func checkBracketBalanceDelta(oldContent, newContent string) string {
	o := countBrackets(oldContent)
	n := countBrackets(newContent)
	var warns []string
	if o.parens != n.parens {
		warns = append(warns, fmt.Sprintf("() %d->%d", o.parens, n.parens))
	}
	if o.braces != n.braces {
		warns = append(warns, fmt.Sprintf("{} %d->%d", o.braces, n.braces))
	}
	if o.brackets != n.brackets {
		warns = append(warns, fmt.Sprintf("[] %d->%d", o.brackets, n.brackets))
	}
	if len(warns) == 0 {
		return ""
	}
	return "bracket balance changed: " + strings.Join(warns, ", ") + " -- verify structure"
}

type bracketCounts struct{ parens, braces, brackets int }

func countBrackets(s string) bracketCounts {
	var c bracketCounts
	for _, ch := range s {
		switch ch {
		case '(':
			c.parens++
		case ')':
			c.parens--
		case '{':
			c.braces++
		case '}':
			c.braces--
		case '[':
			c.brackets++
		case ']':
			c.brackets--
		}
	}
	return c
}

// splitLines splits s by newline, removing the trailing empty element
// that strings.Split produces when s ends with \n.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	if strings.HasSuffix(s, "\n") {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func hasConflictMarkers(content string) bool {
	return strings.Contains(content, "\x3c\x3c\x3c\x3c\x3c\x3c\x3c") &&
		strings.Contains(content, "\x3e\x3e\x3e\x3e\x3e\x3e\x3e")
}
