package lsp

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// ApplyTextEditsToString applies text edits to content and returns the new content.
// Edits are applied in reverse order (bottom-to-top) to preserve offsets.
// Returns an error if any edits have overlapping ranges.
func ApplyTextEditsToString(content string, edits []TextEdit) (string, error) {
	if len(edits) == 0 {
		return content, nil
	}

	sorted := make([]TextEdit, len(edits))
	copy(sorted, edits)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Range.Start.Line != sorted[j].Range.Start.Line {
			return sorted[i].Range.Start.Line < sorted[j].Range.Start.Line
		}
		return sorted[i].Range.Start.Character < sorted[j].Range.Start.Character
	})

	for i := 1; i < len(sorted); i++ {
		if rangesOverlap(sorted[i-1].Range, sorted[i].Range) {
			return "", fmt.Errorf("lsp: overlapping text edits at %v and %v", sorted[i-1].Range, sorted[i].Range)
		}
	}

	lines := strings.Split(content, "\n")

	for i := len(sorted) - 1; i >= 0; i-- {
		edit := sorted[i]
		lines = applyEditToLines(lines, edit)
	}

	return strings.Join(lines, "\n"), nil
}

func applyEditToLines(lines []string, edit TextEdit) []string {
	startLine := edit.Range.Start.Line
	startChar := edit.Range.Start.Character
	endLine := edit.Range.End.Line
	endChar := edit.Range.End.Character

	if startLine >= len(lines) {
		return lines
	}

	var before, after string
	if startLine < len(lines) {
		line := lines[startLine]
		if startChar <= len(line) {
			before = line[:startChar]
		} else {
			before = line
		}
	}

	if endLine < len(lines) {
		line := lines[endLine]
		if endChar <= len(line) {
			after = line[endChar:]
		}
	}

	newText := before + edit.NewText + after
	newLines := strings.Split(newText, "\n")

	replacementCount := endLine - startLine + 1
	if endLine >= len(lines) {
		replacementCount = len(lines) - startLine
	}

	result := make([]string, 0, len(lines)-replacementCount+len(newLines))
	result = append(result, lines[:startLine]...)
	result = append(result, newLines...)
	remainingStart := endLine + 1
	if remainingStart < len(lines) {
		result = append(result, lines[remainingStart:]...)
	}

	return result
}

func rangesOverlap(a, b Range) bool {
	if a.End.Line < b.Start.Line {
		return false
	}
	if a.Start.Line > b.End.Line {
		return false
	}
	if a.End.Line == b.Start.Line && a.End.Character <= b.Start.Character {
		return false
	}
	if a.Start.Line == b.End.Line && a.Start.Character >= b.End.Character {
		return false
	}
	return true
}

// ApplyWorkspaceEdit applies a WorkspaceEdit to the filesystem.
// For text edits, it reads the file, applies edits, and writes back.
// For resource operations, it creates/renames/deletes files.
func ApplyWorkspaceEdit(edit WorkspaceEdit) (int, error) {
	applied := 0

	if len(edit.DocumentChanges) > 0 {
		for _, change := range edit.DocumentChanges {
			switch {
			case change.TextDocument != nil:
				uri := change.TextDocument.TextDocument.URI
				path := uriToPath(uri)
				content, err := os.ReadFile(path)
				if err != nil {
					return applied, fmt.Errorf("lsp: read %s: %w", path, err)
				}
				newContent, err := ApplyTextEditsToString(string(content), change.TextDocument.Edits)
				if err != nil {
					return applied, fmt.Errorf("lsp: apply edits to %s: %w", path, err)
				}
				if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
					return applied, fmt.Errorf("lsp: write %s: %w", path, err)
				}
				applied++
			case change.CreateFile != nil:
				path := uriToPath(change.CreateFile.URI)
				if _, err := os.Stat(path); err == nil && !change.CreateFile.Options.Overwrite {
					if !change.CreateFile.Options.IgnoreIfExists {
						return applied, fmt.Errorf("lsp: create %s: already exists", path)
					}
					continue
				}
				if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
					return applied, fmt.Errorf("lsp: create %s: %w", path, err)
				}
				applied++
			case change.RenameFile != nil:
				oldPath := uriToPath(change.RenameFile.OldURI)
				newPath := uriToPath(change.RenameFile.NewURI)
				if err := os.Rename(oldPath, newPath); err != nil {
					return applied, fmt.Errorf("lsp: rename %s -> %s: %w", oldPath, newPath, err)
				}
				applied++
			case change.DeleteFile != nil:
				path := uriToPath(change.DeleteFile.URI)
				if err := os.Remove(path); err != nil {
					if !change.DeleteFile.Options.IgnoreIfNotExists {
						return applied, fmt.Errorf("lsp: delete %s: %w", path, err)
					}
				}
				applied++
			}
		}
		return applied, nil
	}

	for uri, edits := range edit.Changes {
		path := uriToPath(uri)
		content, err := os.ReadFile(path)
		if err != nil {
			return applied, fmt.Errorf("lsp: read %s: %w", path, err)
		}
		newContent, err := ApplyTextEditsToString(string(content), edits)
		if err != nil {
			return applied, fmt.Errorf("lsp: apply edits to %s: %w", path, err)
		}
		if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
			return applied, fmt.Errorf("lsp: write %s: %w", path, err)
		}
		applied++
	}

	return applied, nil
}

// FlattenWorkspaceTextEdits extracts all TextEdits from a WorkspaceEdit
// into a flat map of URI -> []TextEdit.
func FlattenWorkspaceTextEdits(edit WorkspaceEdit) map[string][]TextEdit {
	result := make(map[string][]TextEdit)

	if len(edit.DocumentChanges) > 0 {
		for _, change := range edit.DocumentChanges {
			if change.TextDocument != nil {
				uri := change.TextDocument.TextDocument.URI
				result[uri] = append(result[uri], change.TextDocument.Edits...)
			}
		}
		return result
	}

	for uri, edits := range edit.Changes {
		result[uri] = edits
	}
	return result
}
