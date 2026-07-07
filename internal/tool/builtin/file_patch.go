package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"monika/internal/tool"
)

type filePatch struct {
	projectDir string
	diagFunc   LSPDiagFunc
}

func NewFilePatch(projectDir string) tool.Tool {
	return &filePatch{projectDir: projectDir}
}

func (f *filePatch) SetDiagFunc(fn LSPDiagFunc) { f.diagFunc = fn }

func (f *filePatch) Name() string { return "patch" }

func (f *filePatch) Description() string {
	return "Applies a search/replace patch to a file. Finds the exact occurrence of `search` in the file content and replaces it with `replace`. The `search` string must match exactly one location in the file. Line endings (CRLF/LF) are normalized automatically — provide search and replace with any line ending style. Fails if `search` is not found or matches multiple locations. WARNING: do NOT call patch on the same file in parallel — LSP diagnostics triggered by one edit can lock the file and cause the other to fail. Serialize multiple edits to the same file."
}

func (f *filePatch) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filePath": map[string]any{
				"type":        "string",
				"description": "The absolute path to the file to modify",
			},
			"search": map[string]any{
				"type":        "string",
				"description": "The exact text to search for in the file. Must be unique in the file (exactly one match).",
			},
			"replace": map[string]any{
				"type":        "string",
				"description": "The replacement text. Use empty string to delete the search text.",
			},
		},
		"required": []string{"filePath", "search", "replace"},
	}
}

func (f *filePatch) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var params struct {
		FilePath string `json:"filePath"`
		Search   string `json:"search"`
		Replace  string `json:"replace"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	if params.Search == "" {
		return tool.ExecutionResult{Content: "search must not be empty", IsError: true}, nil
	}

	safePath, err := resolveToolPath(params.FilePath, tool.ProjectDirOrDefault(ctx, f.projectDir))
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	if tool.IsFileDirty(safePath) {
		diskData, _ := os.ReadFile(safePath)
		diskContent := string(diskData)
		aiContent, _, err := patchContent(diskContent, params.Search, params.Replace)
		if err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
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

	result, err := f.patchFile(safePath, params.Search, params.Replace)
	if err != nil {
		return result, err
	}
	if !result.IsError && f.diagFunc != nil {
		result.Content += f.diagFunc(ctx, safePath)
	}
	return result, nil
}

func (f *filePatch) patchFile(path, search, replace string) (tool.ExecutionResult, error) {
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

	newContent, partialMatch, err := patchContent(rawContent, search, replace)
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	if err := os.WriteFile(path, []byte(newContent), info.Mode()); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	diffLines := computeDiff(path, rawContent, newContent)

	oldLines := len(splitLines(search))
	newLines := len(splitLines(replace))

	resultText := fmt.Sprintf("Patched %d -> %d lines in %s", oldLines, newLines, path)
	if partialMatch {
		resultText += "\n⚠ search matched mid-line (not at line boundary) — verify correctness"
	}
	balanceWarn := checkBracketBalanceDelta(rawContent, newContent)
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

// normalizeLineEndings converts all CRLF in s to LF, then if toCRLF is true,
// converts all LF to CRLF. This lets callers match the file's line ending
// style without normalizing the file content itself (which would corrupt
// mixed-ending files on the CRLF→LF→CRLF round-trip).
func normalizeLineEndings(s string, toCRLF bool) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if toCRLF {
		s = strings.ReplaceAll(s, "\n", "\r\n")
	}
	return s
}

// patchContent performs a validated search/replace on content. It:
//   - normalizes line endings in search/replace to match content's style
//     (no round-trip corruption of the file content itself)
//   - validates that search matches exactly once (rejects 0 or >1 matches)
//   - detects partial-line matches (search starts/ends mid-line)
//
// Both the normal and the conflict (dirty-file) paths share this function,
// ensuring identical validation.
func patchContent(content, search, replace string) (newContent string, partialMatch bool, err error) {
	isCRLF := strings.Contains(content, "\r\n")
	normalizedSearch := normalizeLineEndings(search, isCRLF)
	normalizedReplace := normalizeLineEndings(replace, isCRLF)

	count := strings.Count(content, normalizedSearch)
	if count == 0 {
		snippet := normalizedSearch
		if len(snippet) > 80 {
			snippet = snippet[:77] + "..."
		}
		return "", false, fmt.Errorf("search string not found in file: %q", snippet)
	}
	if count > 1 {
		snippet := normalizedSearch
		if len(snippet) > 80 {
			snippet = snippet[:77] + "..."
		}
		return "", false, fmt.Errorf("search string matched %d locations (must be unique): %q", count, snippet)
	}

	newContent = strings.Replace(content, normalizedSearch, normalizedReplace, 1)

	matchIdx := strings.Index(content, normalizedSearch)
	if matchIdx > 0 && content[matchIdx-1] != '\n' && content[matchIdx-1] != '\r' {
		partialMatch = true
	} else if endIdx := matchIdx + len(normalizedSearch); endIdx < len(content) && content[endIdx] != '\n' && content[endIdx] != '\r' {
		partialMatch = true
	}

	return newContent, partialMatch, nil
}
