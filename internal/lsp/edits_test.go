package lsp

import (
	"os"
	"path/filepath"
	"testing"
)

func pos(line, char int) Position {
	return Position{Line: line, Character: char}
}

func rng(startLine, startChar, endLine, endChar int) Range {
	return Range{Start: pos(startLine, startChar), End: pos(endLine, endChar)}
}

func TestApplyTextEditsToString_Empty(t *testing.T) {
	result, err := ApplyTextEditsToString("hello", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello" {
		t.Fatalf("expected 'hello', got %q", result)
	}
}

func TestApplyTextEditsToString_SingleLineInsert(t *testing.T) {
	content := "hello world"
	edits := []TextEdit{
		{Range: rng(0, 5, 0, 5), NewText: " beautiful"},
	}
	result, err := ApplyTextEditsToString(content, edits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello beautiful world" {
		t.Fatalf("expected 'hello beautiful world', got %q", result)
	}
}

func TestApplyTextEditsToString_SingleLineReplace(t *testing.T) {
	content := "hello world"
	edits := []TextEdit{
		{Range: rng(0, 6, 0, 11), NewText: "Go"},
	}
	result, err := ApplyTextEditsToString(content, edits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello Go" {
		t.Fatalf("expected 'hello Go', got %q", result)
	}
}

func TestApplyTextEditsToString_SingleLineDelete(t *testing.T) {
	content := "hello world"
	edits := []TextEdit{
		{Range: rng(0, 5, 0, 11), NewText: ""},
	}
	result, err := ApplyTextEditsToString(content, edits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello" {
		t.Fatalf("expected 'hello', got %q", result)
	}
}

func TestApplyTextEditsToString_MultiLineEdit(t *testing.T) {
	content := "line1\nline2\nline3"
	edits := []TextEdit{
		{Range: rng(0, 5, 1, 0), NewText: ""},
	}
	result, err := ApplyTextEditsToString(content, edits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "line1line2\nline3" {
		t.Fatalf("expected 'line1line2\\nline3', got %q", result)
	}
}

func TestApplyTextEditsToString_MultipleEdits(t *testing.T) {
	content := "foo bar baz"
	edits := []TextEdit{
		{Range: rng(0, 0, 0, 3), NewText: "ONE"},
		{Range: rng(0, 4, 0, 7), NewText: "TWO"},
		{Range: rng(0, 8, 0, 11), NewText: "THREE"},
	}
	result, err := ApplyTextEditsToString(content, edits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ONE TWO THREE" {
		t.Fatalf("expected 'ONE TWO THREE', got %q", result)
	}
}

func TestApplyTextEditsToString_OverlappingRejected(t *testing.T) {
	content := "hello world"
	edits := []TextEdit{
		{Range: rng(0, 0, 0, 7), NewText: "X"},
		{Range: rng(0, 5, 0, 11), NewText: "Y"},
	}
	_, err := ApplyTextEditsToString(content, edits)
	if err == nil {
		t.Fatal("expected error for overlapping edits")
	}
}

func TestRangesOverlap_NoOverlap(t *testing.T) {
	cases := []struct {
		a, b Range
	}{
		{rng(0, 0, 0, 3), rng(0, 3, 0, 6)},  // adjacent, no overlap
		{rng(0, 0, 0, 5), rng(0, 6, 0, 10)},  // gap between
		{rng(0, 0, 0, 5), rng(1, 0, 1, 5)},   // different lines
	}
	for _, c := range cases {
		if rangesOverlap(c.a, c.b) {
			t.Errorf("expected no overlap for %v and %v", c.a, c.b)
		}
	}
}

func TestRangesOverlap_Yes(t *testing.T) {
	cases := []struct {
		a, b Range
	}{
		{rng(0, 0, 0, 6), rng(0, 3, 0, 9)},   // partial overlap same line
		{rng(0, 0, 1, 0), rng(0, 5, 0, 10)},   // multi-line overlaps single-line
		{rng(0, 2, 0, 4), rng(0, 0, 0, 6)},    // contained within
	}
	for _, c := range cases {
		if !rangesOverlap(c.a, c.b) {
			t.Errorf("expected overlap for %v and %v", c.a, c.b)
		}
	}
}

func TestApplyTextEditsToString_AppendsNewLine(t *testing.T) {
	content := "line1"
	edits := []TextEdit{
		{Range: rng(0, 5, 0, 5), NewText: "\nline2"},
	}
	result, err := ApplyTextEditsToString(content, edits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "line1\nline2" {
		t.Fatalf("expected 'line1\\nline2', got %q", result)
	}
}

func TestApplyWorkspaceEdit_ChangesMap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello world"), 0o644)

	edit := WorkspaceEdit{
		Changes: map[string][]TextEdit{
			"file:///" + filepath.ToSlash(path): {
				{Range: rng(0, 6, 0, 11), NewText: "Go"},
			},
		},
	}

	applied, err := ApplyWorkspaceEdit(edit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if applied != 1 {
		t.Fatalf("expected 1 applied, got %d", applied)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "hello Go" {
		t.Fatalf("expected 'hello Go', got %q", string(data))
	}
}

func TestApplyWorkspaceEdit_DocumentChanges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("foo bar"), 0o644)

	edit := WorkspaceEdit{
		DocumentChanges: []DocumentChange{
			{
				TextDocument: &TextDocumentEdit{
					TextDocument: OptionalVersionedTextDocumentIdentifier{
					URI: "file:///" + filepath.ToSlash(path),
				},
					Edits: []TextEdit{
						{Range: rng(0, 0, 0, 3), NewText: "ONE"},
					},
				},
			},
		},
	}

	applied, err := ApplyWorkspaceEdit(edit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if applied != 1 {
		t.Fatalf("expected 1 applied, got %d", applied)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "ONE bar" {
		t.Fatalf("expected 'ONE bar', got %q", string(data))
	}
}

func TestApplyWorkspaceEdit_CreateFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.txt")
	uri := "file:///" + filepath.ToSlash(dir) + "/new.txt"

	edit := WorkspaceEdit{
		DocumentChanges: []DocumentChange{
			{CreateFile: &CreateFile{URI: uri}},
		},
	}

	applied, err := ApplyWorkspaceEdit(edit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if applied != 1 {
		t.Fatalf("expected 1 applied, got %d", applied)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "" {
		t.Fatalf("expected empty file, got %q", string(data))
	}
}

func TestApplyWorkspaceEdit_DeleteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "del.txt")
	os.WriteFile(path, []byte("bye"), 0o644)
	uri := "file:///" + filepath.ToSlash(path)

	edit := WorkspaceEdit{
		DocumentChanges: []DocumentChange{
			{DeleteFile: &DeleteFile{URI: uri}},
		},
	}

	applied, err := ApplyWorkspaceEdit(edit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if applied != 1 {
		t.Fatalf("expected 1 applied, got %d", applied)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("expected file to be deleted")
	}
}

func TestFlattenWorkspaceTextEdits(t *testing.T) {
	edit := WorkspaceEdit{
		Changes: map[string][]TextEdit{
			"file:///a.go": {{Range: rng(0, 0, 0, 1), NewText: "x"}},
			"file:///b.go": {{Range: rng(0, 0, 0, 1), NewText: "y"}},
		},
	}

	flat := FlattenWorkspaceTextEdits(edit)
	if len(flat) != 2 {
		t.Fatalf("expected 2 URIs, got %d", len(flat))
	}
}
