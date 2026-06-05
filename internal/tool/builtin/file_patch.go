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
	return "Applies a search/replace patch to a file. Finds the exact occurrence of `search` in the file content and replaces it with `replace`. The `search` string must match exactly one location in the file (whitespace-sensitive). Fails if `search` is not found or matches multiple locations. The `replace` text is taken from the message content when not provided in JSON args — use this for large code blocks."
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
				"description": "The replacement text. If not provided in JSON args, the natural code from the assistant's message content is used instead.",
			},
		},
		"required": []string{"filePath", "search"},
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

	// replace from message content (natural code) takes priority over JSON args
	if content := tool.MessageContentFromContext(ctx); content != "" {
		params.Replace = content
	}

	if params.Search == "" {
		return tool.ExecutionResult{Content: "search must not be empty", IsError: true}, nil
	}

	safePath, err := resolveToolPath(params.FilePath, tool.ProjectDirOrDefault(ctx, f.projectDir))
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
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

	hadCRLF := strings.Contains(rawContent, "\r\n")
	content := strings.ReplaceAll(rawContent, "\r\n", "\n")
	normalizedSearch := strings.ReplaceAll(search, "\r\n", "\n")
	normalizedReplace := strings.ReplaceAll(replace, "\r\n", "\n")

	count := strings.Count(content, normalizedSearch)
	if count == 0 {
		snippet := normalizedSearch
		if len(snippet) > 80 {
			snippet = snippet[:77] + "..."
		}
		return tool.ExecutionResult{
			Content: fmt.Sprintf("search string not found in file: %q", snippet),
			IsError: true,
		}, nil
	}
	if count > 1 {
		snippet := normalizedSearch
		if len(snippet) > 80 {
			snippet = snippet[:77] + "..."
		}
		return tool.ExecutionResult{
			Content: fmt.Sprintf("search string matched %d locations (must be unique): %q", count, snippet),
			IsError: true,
		}, nil
	}

	newContent := strings.Replace(content, normalizedSearch, normalizedReplace, 1)

	if hadCRLF {
		newContent = strings.ReplaceAll(newContent, "\n", "\r\n")
	}
	if err := os.WriteFile(path, []byte(newContent), info.Mode()); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	diffLines := computeDiff(path, content, newContent)

	// Count lines changed
	oldLines := len(splitLines(normalizedSearch))
	newLines := len(splitLines(normalizedReplace))

	resultText := fmt.Sprintf("Patched %d -> %d lines in %s", oldLines, newLines, path)
	balanceWarn := checkBracketBalance(normalizedReplace)
	if balanceWarn != "" {
		resultText += "\n⚠ " + balanceWarn
	}
	return tool.ExecutionResult{
		Content:   resultText,
		DiffLines: diffLines,
	}, nil
}
