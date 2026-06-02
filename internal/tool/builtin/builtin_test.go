package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"monika/internal/tool"
)

func TestFileRead(t *testing.T) {
	dir := t.TempDir()
	f := NewFileRead(dir, nil)

	if f.Name() != "file_read" {
		t.Fatalf("name = %q", f.Name())
	}
	if f.Description() == "" {
		t.Fatal("description empty")
	}
	params := f.Parameters()
	if _, ok := params["properties"]; !ok {
		t.Fatal("missing properties")
	}
}

func TestFileReadReadsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("line1\nline2\nline3\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := NewFileRead(dir, nil)
	args, _ := json.Marshal(map[string]any{"filePath": path, "offset": 1, "limit": 200})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Content, "line1") || !strings.Contains(result.Content, "line2") || !strings.Contains(result.Content, "line3") {
		t.Fatalf("content = %q", result.Content)
	}
}

func TestFileReadExplicitLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.txt")
	var lines []string
	for i := 1; i <= 300; i++ {
		lines = append(lines, fmt.Sprintf("line%d", i))
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	f := NewFileRead(dir, nil)
	args, _ := json.Marshal(map[string]any{"filePath": path, "offset": 1, "limit": 100})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	resultLines := strings.Count(result.Content, "\n") + 1
	// 100 numbered lines + footer hint lines
	if resultLines < 100 {
		t.Fatalf("expected at least 100 lines with explicit limit, got %d", resultLines)
	}
}

func TestFileReadWithOffsetLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("line1\nline2\nline3\nline4\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := NewFileRead(dir, nil)
	args, _ := json.Marshal(map[string]any{"filePath": path, "offset": 2, "limit": 2})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Content, "line2") || !strings.Contains(result.Content, "line3") {
		t.Fatalf("content = %q", result.Content)
	}
}

func TestFileReadOutsideProject(t *testing.T) {
	dir := t.TempDir()
	f := NewFileRead(dir, nil)
	args, _ := json.Marshal(map[string]any{"filePath": "/etc/passwd"})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for path outside project")
	}
}

func TestFileWriteCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.txt")

	f := NewFileWrite(dir)
	args, _ := json.Marshal(map[string]any{"filePath": path, "content": "hello world"})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world" {
		t.Fatalf("content = %q", string(data))
	}
}

func TestFileWriteOutsideProject(t *testing.T) {
	dir := t.TempDir()
	f := NewFileWrite(dir)
	args, _ := json.Marshal(map[string]any{"filePath": "/etc/hosts", "content": "bad"})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for path outside project")
	}
}

func TestFileList(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	f := NewFileList(dir)
	args, _ := json.Marshal(map[string]any{"dirPath": dir})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.Content == "" {
		t.Fatal("empty output")
	}
}

func TestGlob(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.go"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "c.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	f := NewGlob(dir)
	args, _ := json.Marshal(map[string]any{"pattern": "*.go"})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
}

func TestGrep(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := NewGrep(dir, nil)
	args, _ := json.Marshal(map[string]any{"pattern": "func.main"})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.Content == "" {
		t.Fatal("empty output, expected matches")
	}
}

func TestGrepInvalidRegex(t *testing.T) {
	dir := t.TempDir()
	f := NewGrep(dir, nil)
	args, _ := json.Marshal(map[string]any{"pattern": "["})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for invalid regex")
	}
}

func TestBashResolveShell(t *testing.T) {
	shell, _ := resolveShell()
	if shell == "" {
		t.Fatal("no shell found on system")
	}
}

func TestBashExecute(t *testing.T) {
	sh, err := NewBash(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if sh.Name() != "bash" {
		t.Fatalf("name = %q", sh.Name())
	}

	args, _ := json.Marshal(map[string]any{"command": "echo hello"})
	result, err := sh.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError && result.Content != "hello" {
		t.Fatalf("content = %q", result.Content)
	}
}

func TestRegisterDefaultsAll(t *testing.T) {
	r := tool.NewRegistry()
	if err := RegisterDefaults(r, t.TempDir(), nil); err != nil {
		t.Fatal(err)
	}
	expected := []string{"file_read", "file_write", "file_edit", "file_edit_hunks", "file_list", "glob", "grep", "bash"}
	for _, name := range expected {
		if _, ok := r.Get(name); !ok {
			t.Fatalf("tool %q not registered", name)
		}
	}
}

func TestFileEdit(t *testing.T) {
	dir := t.TempDir()
	f := NewFileEdit(dir)

	if f.Name() != "file_edit" {
		t.Fatalf("name = %q", f.Name())
	}
	if f.Description() == "" {
		t.Fatal("description empty")
	}
	params := f.Parameters()
	if _, ok := params["properties"]; !ok {
		t.Fatal("missing properties")
	}
}

func TestFileEditSingleReplace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("hello world\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := NewFileEdit(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath":   path,
		"old_string": "hello",
		"new_string": "goodbye",
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "goodbye world\n" {
		t.Fatalf("content = %q", string(data))
	}
}

func TestFileEditReplaceAll(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("foo bar foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := NewFileEdit(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath":    path,
		"old_string":  "foo",
		"new_string":  "baz",
		"replace_all": true,
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "baz bar baz\n" {
		t.Fatalf("content = %q", string(data))
	}
}

func TestFileEditNonUniqueFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("foo bar foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := NewFileEdit(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath":   path,
		"old_string": "foo",
		"new_string": "baz",
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for non-unique old_string")
	}
}

func TestFileEditNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := NewFileEdit(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath":   path,
		"old_string": "nonexistent",
		"new_string": "replaced",
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for old_string not found")
	}
}

func TestFileEditOutsideProject(t *testing.T) {
	dir := t.TempDir()
	f := NewFileEdit(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath":   "/etc/passwd",
		"old_string": "a",
		"new_string": "b",
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for path outside project")
	}
}

func TestFileEditMultiLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("line1\nline2\nline3\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := NewFileEdit(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath":   path,
		"old_string": "line1\nline2",
		"new_string": "alpha\nbeta",
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "alpha\nbeta\nline3\n" {
		t.Fatalf("content = %q", string(data))
	}
}

func TestFileEditCRLFNormalization(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	// File uses CRLF line endings
	if err := os.WriteFile(path, []byte("line1\r\nline2\r\nline3\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := NewFileEdit(dir)
	// old_string uses LF (as the AI would provide)
	args, _ := json.Marshal(map[string]any{
		"filePath":   path,
		"old_string": "line1\nline2",
		"new_string": "alpha\nbeta",
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "alpha\r\nbeta\r\nline3\r\n" {
		t.Fatalf("content = %q", string(data))
	}
}

func TestFileEditPreservesPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.sh")
	if err := os.WriteFile(path, []byte("echo hello\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	origInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	f := NewFileEdit(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath":   path,
		"old_string": "hello",
		"new_string": "goodbye",
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}

	newInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if origInfo.Mode() != newInfo.Mode() {
		t.Fatalf("permissions changed: %o -> %o", origInfo.Mode(), newInfo.Mode())
	}
}

func TestFileEditConflictMarkers(t *testing.T) {
	dir := t.TempDir()
	f := NewFileEdit(dir)
	path := filepath.Join(dir, "conflict.txt")
	os.WriteFile(path, []byte("before\n<<<<<<< HEAD\nfoo\n=======\nbar\n>>>>>>> branch\nafter"), 0o644)

	args, _ := json.Marshal(map[string]any{
		"filePath":   path,
		"old_string": "foo",
		"new_string": "baz",
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for conflict markers")
	}
	if !result.Conflicts {
		t.Fatal("expected Conflicts=true")
	}
}

func TestFileEditFuzzyWhitespace(t *testing.T) {
	dir := t.TempDir()
	f := NewFileEdit(dir)
	path := filepath.Join(dir, "fuzzy.txt")
	// Content with blank line boundaries so wsFuzzyScanner finds separate segments
	os.WriteFile(path, []byte("line1\n\n  line2  \t line3\n\nline4"), 0o644)

	// Use different whitespace — should still match via fuzzy
	args, _ := json.Marshal(map[string]any{
		"filePath":   path,
		"old_string": "line2 line3",
		"new_string": "replaced",
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success but got: %s", result.Content)
	}

	data, _ := os.ReadFile(path)
	got := string(data)
	// Fuzzy match preserves leading whitespace from the original segment
	want := "line1\n\n  replaced\n\nline4"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFileEditAnchorVerification(t *testing.T) {
	dir := t.TempDir()
	f := NewFileEdit(dir)
	path := filepath.Join(dir, "anchor.txt")
	os.WriteFile(path, []byte("alpha\nbeta\ngamma\ndelta"), 0o644)

	// Compute anchor hash for line 2 ("beta")
	h := fnv.New32a()
	h.Write([]byte("beta"))
	hash := fmt.Sprintf("%08x", h.Sum32())
	anchor := fmt.Sprintf("%s:2", hash)

	args, _ := json.Marshal(map[string]any{
		"filePath":   path,
		"old_string": "gamma",
		"new_string": "GAMMA",
		"anchor":     anchor,
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success but got: %s", result.Content)
	}

	data, _ := os.ReadFile(path)
	got := string(data)
	if !strings.Contains(got, "GAMMA") {
		t.Fatalf("expected GAMMA in %q", got)
	}
}

func TestPatchEdit(t *testing.T) {
	dir := t.TempDir()
	p := NewPatchEdit(dir)
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("aaa\nbbb\nccc\nddd\neee"), 0o644)

	hunks := "@@ -2,3 +2,3 @@\n bbb\n-ccc\n+CCC\n ddd\n"
	args, _ := json.Marshal(map[string]any{
		"filePath": path,
		"hunks":    hunks,
	})
	result, err := p.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success but got: %s", result.Content)
	}

	data, _ := os.ReadFile(path)
	got := string(data)
	want := "aaa\nbbb\nCCC\nddd\neee"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPatchEditMultipleHunks(t *testing.T) {
	dir := t.TempDir()
	p := NewPatchEdit(dir)
	path := filepath.Join(dir, "multi.txt")
	os.WriteFile(path, []byte("line1\nline2\nline3\nline4\nline5\nline6\nline7"), 0o644)

	hunks := "@@ -2,3 +2,3 @@\n line2\n-line3\n+LINE3\n line4\n@@ -5,3 +5,3 @@\n line5\n-line6\n+LINE6\n line7\n"
	args, _ := json.Marshal(map[string]any{
		"filePath": path,
		"hunks":    hunks,
	})
	result, err := p.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success but got: %s", result.Content)
	}

	data, _ := os.ReadFile(path)
	got := string(data)
	want := "line1\nline2\nLINE3\nline4\nline5\nLINE6\nline7"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPatchEditRejectsConflicts(t *testing.T) {
	dir := t.TempDir()
	p := NewPatchEdit(dir)
	path := filepath.Join(dir, "conflict.txt")
	os.WriteFile(path, []byte("before\n<<<<<<< HEAD\nfoo\n=======\nbar\n>>>>>>> branch\nafter"), 0o644)

	args, _ := json.Marshal(map[string]any{
		"filePath": path,
		"hunks":    "@@ -1,3 +1,3 @@\n before\n-foo\n+baz\n bar",
	})
	result, err := p.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for conflict markers")
	}
	if !result.Conflicts {
		t.Fatal("expected Conflicts=true")
	}
}

func TestFileListTree(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "a", "b"), 0o755)
	os.WriteFile(filepath.Join(dir, "a", "b", "c.txt"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "a", "d.go"), []byte("x"), 0o644)

	f := NewFileList(dir)
	args, _ := json.Marshal(map[string]any{
		"dirPath": dir,
		"tree":    true,
		"depth":   3,
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success but got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "a/") {
		t.Fatalf("expected tree to contain 'a/', got: %s", result.Content)
	}
}

func TestFileReadRanges(t *testing.T) {
	dir := t.TempDir()
	f := NewFileRead(dir, nil)
	path := filepath.Join(dir, "ranges.txt")
	var lines []string
	for i := 1; i <= 20; i++ {
		lines = append(lines, fmt.Sprintf("line %d", i))
	}
	os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)

	args, _ := json.Marshal(map[string]any{
		"filePath": path,
		"ranges":   "3-5,18-20",
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success but got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "line 3") || !strings.Contains(result.Content, "line 19") {
		t.Fatalf("expected ranges in output, got: %s", result.Content)
	}
	if strings.Contains(result.Content, "line 10") {
		t.Fatalf("should not contain middle lines, got: %s", result.Content)
	}
}

func TestFileEditFuzzyReplaceAll(t *testing.T) {
	dir := t.TempDir()
	f := NewFileEdit(dir)
	path := filepath.Join(dir, "fuzzy_all.txt")
	os.WriteFile(path, []byte("aaa\n\n  bbb   ccc\n\nxxx\n\n bbb\tccc \n\nzzz"), 0o644)

	args, _ := json.Marshal(map[string]any{
		"filePath":    path,
		"old_string":  "bbb ccc",
		"new_string":  "REPLACED",
		"replace_all": true,
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success but got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "2 occurrence") {
		t.Fatalf("expected 2 occurrences, got: %s", result.Content)
	}

	data, _ := os.ReadFile(path)
	got := string(data)
	if strings.Count(got, "REPLACED") != 2 {
		t.Fatalf("expected 2 REPLACED in %q", got)
	}
}

func TestFileEditFuzzyAmbiguousFails(t *testing.T) {
	dir := t.TempDir()
	f := NewFileEdit(dir)
	path := filepath.Join(dir, "fuzzy_amb.txt")
	os.WriteFile(path, []byte("aaa\n\n  bbb   ccc\n\nxxx\n\n bbb\tccc \n\nzzz"), 0o644)

	args, _ := json.Marshal(map[string]any{
		"filePath":   path,
		"old_string": "bbb ccc",
		"new_string": "REPLACED",
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatalf("expected ambiguity error but got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "2 times") {
		t.Fatalf("expected ambiguity message, got: %s", result.Content)
	}
}

func TestHasConflictMarkers(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"real conflict", "<<<<<<< HEAD\nfoo\n=======\nbar\n>>>>>>> branch", true},
		{"only start", "<<<<<<< HEAD\nfoo\n", false},
		{"only end", "bar\n>>>>>>> branch", false},
		{"markdown equals", "Title\n=======", false},
		{"markdown equals with arrow in code", "Title\n=======\n```\n>>>>>>> foo\n```", false},
		{"clean", "no markers here", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasConflictMarkers(tt.content); got != tt.want {
				t.Errorf("hasConflictMarkers(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
