package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"monika/internal/tool"
)

func TestFileRead(t *testing.T) {
	dir := t.TempDir()
	f := NewFileRead(dir, "", nil)

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

	f := NewFileRead(dir, "", nil)
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

	f := NewFileRead(dir, "", nil)
	args, _ := json.Marshal(map[string]any{"filePath": path, "offset": 1, "limit": 100})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	resultLines := strings.Count(result.Content, "\n") + 1
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

	f := NewFileRead(dir, "", nil)
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
	f := NewFileRead(dir, "", nil)
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
	subDir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "deep.tsx"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "deep.go"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	f := NewGlob(dir)

	// test 1: simple *.go
	args, _ := json.Marshal(map[string]any{"pattern": "*.go"})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}

	// test 2: recursive **/*.go
	args, _ = json.Marshal(map[string]any{"pattern": "**/*.go"})
	result, err = f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "sub"+string(filepath.Separator)+"deep.go") {
		t.Fatalf("recursive glob should find deep.go, got: %s", result.Content)
	}

	// test 3: recursive **/*.tsx
	args, _ = json.Marshal(map[string]any{"pattern": "**/*.tsx"})
	result, err = f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "sub"+string(filepath.Separator)+"deep.tsx") {
		t.Fatalf("recursive glob should find deep.tsx, got: %s", result.Content)
	}

	// test 4: wildcard match containing pattern **/*deep*
	args, _ = json.Marshal(map[string]any{"pattern": "**/*deep*"})
	result, err = f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "deep.tsx") || !strings.Contains(result.Content, "deep.go") {
		t.Fatalf("wildcard glob should find both deep files, got: %s", result.Content)
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
	if err := RegisterDefaults(r, t.TempDir(), "", nil); err != nil {
		t.Fatal(err)
	}
	expected := []string{"file_read", "file_write", "file_edit", "file_list", "glob", "grep", "bash", "background_task"}
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

	anchor := lineHash("hello world") + ":1"

	f := NewFileEdit(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath":   path,
		"anchor":     anchor,
		"new_string": "goodbye world",
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

func TestFileEditMultiLineReplace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("line1\nline2\nline3\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	anchor := lineHash("line1") + ":1"

	f := NewFileEdit(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath":   path,
		"anchor":     anchor,
		"new_string": "alpha\nbeta",
		"line_count": 2,
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

func TestFileEditInsertAfterLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("line1\nline3\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	anchor := lineHash("line1") + ":1"

	f := NewFileEdit(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath":   path,
		"anchor":     anchor,
		"new_string": "line2",
		"line_count": 0,
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
	if string(data) != "line1\nline2\nline3\n" {
		t.Fatalf("content = %q", string(data))
	}
}

func TestFileEditHashMismatchFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := NewFileEdit(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath":   path,
		"anchor":     "badhash:1",
		"new_string": "replaced",
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for hash mismatch")
	}
}

func TestFileEditOutsideProject(t *testing.T) {
	dir := t.TempDir()
	f := NewFileEdit(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath":   "/etc/passwd",
		"anchor":     "abc123:1",
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

	anchor := lineHash("echo hello") + ":1"
	f := NewFileEdit(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath":   path,
		"anchor":     anchor,
		"new_string": "echo goodbye",
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
	// Use escaped hex to avoid triggering our own conflict marker detection
	conflict := "before\n\x3c\x3c\x3c\x3c\x3c\x3c\x3c HEAD\nfoo\n=======\nbar\n\x3e\x3e\x3e\x3e\x3e\x3e\x3e branch\nafter"
	os.WriteFile(path, []byte(conflict), 0o644)

	anchor := lineHash("before") + ":1"
	args, _ := json.Marshal(map[string]any{
		"filePath":   path,
		"anchor":     anchor,
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

func TestFileEditAnchorVerification(t *testing.T) {
	dir := t.TempDir()
	f := NewFileEdit(dir)
	path := filepath.Join(dir, "anchor.txt")
	os.WriteFile(path, []byte("alpha\nbeta\ngamma\ndelta"), 0o644)

	anchor := lineHash("gamma") + ":3"
	args, _ := json.Marshal(map[string]any{
		"filePath":   path,
		"anchor":     anchor,
		"new_string": "GAMMA",
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
	want := "alpha\nbeta\nGAMMA\ndelta"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
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
	f := NewFileRead(dir, "", nil)
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

func TestFileReadSkillName(t *testing.T) {
	dir := t.TempDir()
	homeDir := t.TempDir()

	// Create a global skill in homeDir/.monika/skills/test-skill/SKILL.md
	skillDir := filepath.Join(homeDir, ".monika", "skills", "test-skill")
	os.MkdirAll(skillDir, 0o755)
	skillContent := "# Test Skill\nThis is a test."
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatal(err)
	}

	f := NewFileRead(dir, homeDir, nil)
	args, _ := json.Marshal(map[string]any{"skillName": "test-skill"})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "# Test Skill") {
		t.Fatalf("expected skill content, got: %s", result.Content)
	}
}

func TestFileReadSkillNameNotFound(t *testing.T) {
	dir := t.TempDir()
	f := NewFileRead(dir, "", nil)
	args, _ := json.Marshal(map[string]any{"skillName": "nonexistent-skill"})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for nonexistent skill")
	}
}

func TestFileReadSkillNamePreferProject(t *testing.T) {
	dir := t.TempDir()
	homeDir := t.TempDir()

	// Create global skill
	globalSkillDir := filepath.Join(homeDir, ".monika", "skills", "my-skill")
	os.MkdirAll(globalSkillDir, 0o755)
	if err := os.WriteFile(filepath.Join(globalSkillDir, "SKILL.md"), []byte("global"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create project skill (should be preferred)
	projSkillDir := filepath.Join(dir, ".monika", "skills", "my-skill")
	os.MkdirAll(projSkillDir, 0o755)
	if err := os.WriteFile(filepath.Join(projSkillDir, "SKILL.md"), []byte("project"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := NewFileRead(dir, homeDir, nil)
	args, _ := json.Marshal(map[string]any{"skillName": "my-skill"})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "project") {
		t.Fatalf("expected project skill content, got: %s", result.Content)
	}
}

func TestFileReadMutuallyExclusiveOrBothEmpty(t *testing.T) {
	dir := t.TempDir()
	f := NewFileRead(dir, "", nil)

	// Both empty
	args, _ := json.Marshal(map[string]any{})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for both empty")
	}
	if !strings.Contains(result.Content, "filePath") || !strings.Contains(result.Content, "skillName") {
		t.Fatalf("expected error mentioning both params, got: %s", result.Content)
	}

	// Both provided
	args2, _ := json.Marshal(map[string]any{"filePath": "/some/path", "skillName": "some-skill"})
	result2, err2 := f.Execute(context.Background(), args2)
	if err2 != nil {
		t.Fatal(err2)
	}
	if !result2.IsError {
		t.Fatal("expected error for both provided")
	}
	if !strings.Contains(result2.Content, "mutually exclusive") {
		t.Fatalf("expected error mentioning mutual exclusivity, got: %s", result2.Content)
	}
}

func TestFileEditLineBeyondFile(t *testing.T) {
	dir := t.TempDir()
	f := NewFileEdit(dir)
	path := filepath.Join(dir, "short.txt")
	os.WriteFile(path, []byte("only\n"), 0o644)

	args, _ := json.Marshal(map[string]any{
		"filePath":   path,
		"anchor":     lineHash("only") + ":100",
		"new_string": "x",
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for line beyond file")
	}
}

func TestFileEditInvalidAnchor(t *testing.T) {
	dir := t.TempDir()
	f := NewFileEdit(dir)
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello\n"), 0o644)

	args, _ := json.Marshal(map[string]any{
		"filePath":   path,
		"anchor":     "no-colon-here",
		"new_string": "x",
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected error for invalid anchor")
	}
}

func TestPatch(t *testing.T) {
	dir := t.TempDir()
	f := NewFilePatch(dir)
	if f.Name() != "patch" {
		t.Fatalf("name = %q", f.Name())
	}
	if f.Description() == "" {
		t.Fatal("description empty")
	}
}

func TestPatchBasicReplace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello world\n"), 0o644)

	f := NewFilePatch(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath": path,
		"search":   "hello",
		"replace":  "goodbye",
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "goodbye world\n" {
		t.Fatalf("content = %q", string(data))
	}
}

func TestPatchNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello world\n"), 0o644)

	f := NewFilePatch(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath": path,
		"search":   "nonexistent",
		"replace":  "x",
	})
	result, _ := f.Execute(context.Background(), args)
	if !result.IsError {
		t.Fatal("expected error for not found")
	}
}

func TestPatchMultipleMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("foo\nfoo\n"), 0o644)

	f := NewFilePatch(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath": path,
		"search":   "foo",
		"replace":  "bar",
	})
	result, _ := f.Execute(context.Background(), args)
	if !result.IsError {
		t.Fatal("expected error for multiple matches")
	}
}

func TestPatchEmptySearch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello\n"), 0o644)

	f := NewFilePatch(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath": path,
		"search":   "",
		"replace":  "x",
	})
	result, _ := f.Execute(context.Background(), args)
	if !result.IsError {
		t.Fatal("expected error for empty search")
	}
}

func TestPatchCRLFPreserved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("line1\r\nline2\r\nline3\r\n"), 0o644)

	f := NewFilePatch(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath": path,
		"search":   "line2",
		"replace":  "replaced",
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	data, _ := os.ReadFile(path)
	expected := "line1\r\nreplaced\r\nline3\r\n"
	if string(data) != expected {
		t.Fatalf("content = %q, want %q", string(data), expected)
	}
}

func TestPatchMixedLineEndings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	original := "line1\r\nline2\nline3\r\n"
	os.WriteFile(path, []byte(original), 0o644)

	f := NewFilePatch(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath": path,
		"search":   "line2",
		"replace":  "replaced",
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "line2") {
		t.Fatalf("line2 should have been replaced: %q", string(data))
	}
	if strings.Count(string(data), "\r\n") != 2 {
		t.Fatalf("CRLF count changed: got %q", string(data))
	}
	if !strings.Contains(string(data), "replaced\nline3") {
		t.Fatalf("LF-only line ending should be preserved: %q", string(data))
	}
}

func TestPatchConflictPathValidation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("foo\nfoo\nbar\n"), 0o644)

	tool.SetFileDirty(path, true)
	defer tool.SetFileDirty(path, false)

	f := NewFilePatch(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath": path,
		"search":   "foo",
		"replace":  "baz",
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatalf("expected error for multiple matches in dirty file, got: %+v", result)
	}
	if result.AiContent != "" {
		t.Fatalf("should not produce AiContent for ambiguous match, got: %q", result.AiContent)
	}
}

func TestPatchConflictPathSingleMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("alpha\nbeta\n"), 0o644)

	tool.SetFileDirty(path, true)
	defer tool.SetFileDirty(path, false)

	f := NewFilePatch(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath": path,
		"search":   "alpha",
		"replace":  "gamma",
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Conflict {
		t.Fatalf("expected conflict result, got: %+v", result)
	}
	if result.AiContent == "" {
		t.Fatal("expected AiContent for valid single match in dirty file")
	}
	if !strings.Contains(result.AiContent, "gamma") {
		t.Fatalf("AiContent should contain replacement: %q", result.AiContent)
	}
}

func TestPatchPartialMatchWarning(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("max_value := 1\n"), 0o644)

	f := NewFilePatch(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath": path,
		"search":   "x_value",
		"replace":  "y_value",
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "partial") && !strings.Contains(result.Content, "mid-line") {
		t.Fatalf("expected partial-match warning, got: %s", result.Content)
	}
}

func TestPatchConflictMarkers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "\x3c\x3c\x3c\x3c\x3c\x3c\x3c HEAD\nfoo\n=======\nbar\n\x3e\x3e\x3e\x3e\x3e\x3e\x3e branch\n"
	os.WriteFile(path, []byte(content), 0o644)

	f := NewFilePatch(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath": path,
		"search":   "foo",
		"replace":  "baz",
	})
	result, _ := f.Execute(context.Background(), args)
	if !result.IsError {
		t.Fatal("expected error for conflict markers")
	}
}

func TestPatchMultiLineReplace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("func old() {\n\treturn nil\n}\n"), 0o644)

	f := NewFilePatch(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath": path,
		"search":   "func old() {\n\treturn nil\n}",
		"replace":  "func new() {\n\treturn 42\n}",
	})
	result, err := f.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	data, _ := os.ReadFile(path)
	expected := "func new() {\n\treturn 42\n}\n"
	if string(data) != expected {
		t.Fatalf("content = %q, want %q", string(data), expected)
	}
}

func TestPatchOutsideProject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello\n"), 0o644)

	f := NewFilePatch(dir)
	args, _ := json.Marshal(map[string]any{
		"filePath": "/etc/passwd",
		"search":   "root",
		"replace":  "hacked",
	})
	result, _ := f.Execute(context.Background(), args)
	if !result.IsError {
		t.Fatal("expected error for path outside project")
	}
}

func TestPatchContentNotFound(t *testing.T) {
	_, _, err := patchContent("hello world", "nonexistent", "x")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestPatchContentMultipleMatch(t *testing.T) {
	_, _, err := patchContent("foo\nfoo\n", "foo", "bar")
	if err == nil {
		t.Fatal("expected error for multiple matches")
	}
}

func TestPatchContentPartialMatch(t *testing.T) {
	content := "max_value := 1"
	newContent, partial, err := patchContent(content, "x_value", "y_value")
	if err != nil {
		t.Fatal(err)
	}
	if !partial {
		t.Fatal("expected partial match")
	}
	if newContent != "may_value := 1" {
		t.Fatalf("content = %q", newContent)
	}
}

func TestPatchContentNoPartialForFullLine(t *testing.T) {
	content := "max_value := 1\n"
	newContent, partial, err := patchContent(content, "max_value := 1", "replaced")
	if err != nil {
		t.Fatal(err)
	}
	if partial {
		t.Fatal("expected no partial match for full line")
	}
	if newContent != "replaced\n" {
		t.Fatalf("content = %q", newContent)
	}
}

func TestApplyEditMixedCRLF(t *testing.T) {
	content := "line1\r\nline2\nline3\r\n"
	newString := "replaced"
	result, err := applyEditToContent(content, lineHash("line2")+":2", newString, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "replaced") {
		t.Fatalf("replacement missing: %q", result)
	}
	if !strings.Contains(result, "\r\n") {
		t.Fatalf("CRLF lines should be preserved: %q", result)
	}
	if !strings.Contains(result, "replaced\nline3") {
		t.Fatalf("LF-only line should be preserved: %q", result)
	}
}

func TestHasConflictMarkers(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"real conflict", "\x3c\x3c\x3c\x3c\x3c\x3c\x3c HEAD\nfoo\n=======\nbar\n\x3e\x3e\x3e\x3e\x3e\x3e\x3e branch", true},
		{"only start", "\x3c\x3c\x3c\x3c\x3c\x3c\x3c HEAD\nfoo\n", false},
		{"only end", "bar\n\x3e\x3e\x3e\x3e\x3e\x3e\x3e branch", false},
		{"markdown equals", "Title\n=======", false},
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
