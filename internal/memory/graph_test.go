package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func tempKBStore(t *testing.T) (*KBStore, string) {
	t.Helper()
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	projectDir := filepath.Join(dir, "project")
	os.MkdirAll(home, 0755)
	os.MkdirAll(projectDir, 0755)
	s, err := NewKBStore(home, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s, home
}

func TestWriteFileAutoLinks(t *testing.T) {
	s, _ := tempKBStore(t)

	s.writeFileUnchecked(ScopeProject, CategoryLesson, "Error Wrapping Pattern",
		"Always wrap errors with fmt.Errorf and %w for chain", []string{"go", "error-handling"}, "high")

	err := s.WriteFile(ScopeProject, CategoryLesson, "Sentinel Error Values",
		"Define sentinel errors with errors.Is for comparison", []string{"go", "error-handling"}, "medium")
	if err != nil {
		t.Fatalf("WriteFile linked: %v", err)
	}

	results, err := s.Search("Go error wrapping", ScopeProject, 5)
	if err != nil || len(results) < 2 {
		t.Fatalf("expected 2+ results, got %d (err=%v)", len(results), err)
	}

	var foundLink bool
	for _, r := range results {
		if len(r.LinkedTo) > 0 || len(r.Backlinks) > 0 {
			foundLink = true
			break
		}
	}
	if !foundLink {
		t.Error("expected at least one link or backlink between related memories")
	}
}

func TestGraphTraverse(t *testing.T) {
	s, _ := tempKBStore(t)

	s.writeFileUnchecked(ScopeProject, CategoryLesson, "Root Lesson", "root content", nil, "medium")
	rootPath := "wiki/lessons/root-lesson.md"

	s.addTypedLink(ScopeProject, rootPath, "wiki/lessons/child-a.md", "related")

	s.writeFileUnchecked(ScopeProject, CategoryLesson, "Child A", "child", nil, "medium")
	s.writeFileUnchecked(ScopeProject, CategoryLesson, "Grandchild", "grandchild", nil, "medium")
	s.addTypedLink(ScopeProject, "wiki/lessons/child-a.md", "wiki/lessons/grandchild.md", "causes")

	nodes, err := s.GraphTraverse(ScopeProject, rootPath, 2, "")
	if err != nil {
		t.Fatalf("GraphTraverse: %v", err)
	}
	if len(nodes) == 0 {
		t.Fatal("expected graph nodes, got 0")
	}
}

func TestExtractEntities(t *testing.T) {
	content := "Some text mentioning store.go and graph.go files.\n" +
		"Also references `KBStore` and `GraphTraverse` identifiers.\n" +
		"```go\n" +
		"func ProcessData(input string) error {\n" +
		"	result := ComputeResult(input)\n" +
		"	return nil\n" +
		"}\n" +
		"```\n"

	entities := extractEntities(content)

	types := make(map[string]int)
	for _, e := range entities {
		types[e.Type]++
	}

	if types["file"] < 2 {
		t.Errorf("expected >=2 file entities, got %d", types["file"])
	}
	if types["function"] < 1 {
		t.Errorf("expected >=1 function entity, got %d", types["function"])
	}
}

func TestEntityIndexOnWrite(t *testing.T) {
	s, _ := tempKBStore(t)

	s.writeFileUnchecked(ScopeProject, CategoryLesson, "Entity Test",
		"See store.go and main.go for details", nil, "medium")

	results, err := s.QueryByEntity(ScopeProject, "store.go")
	if err != nil {
		t.Fatalf("QueryByEntity: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected to find memory mentioning store.go")
	}
}

func TestSoftDeleteRemovesEdges(t *testing.T) {
	s, _ := tempKBStore(t)

	s.writeFileUnchecked(ScopeProject, CategoryLesson, "Source", "content", nil, "medium")
	s.writeFileUnchecked(ScopeProject, CategoryLesson, "Target", "content", nil, "medium")

	srcPath := "wiki/lessons/source.md"
	tgtPath := "wiki/lessons/target.md"
	s.addTypedLink(ScopeProject, srcPath, tgtPath, "related")

	backlinks, _ := s.getBacklinks(ScopeProject, tgtPath)
	if len(backlinks) != 1 {
		t.Fatalf("expected 1 backlink, got %d", len(backlinks))
	}

	s.SoftDelete(ScopeProject, srcPath)

	backlinks, _ = s.getBacklinks(ScopeProject, tgtPath)
	if len(backlinks) != 0 {
		t.Errorf("expected 0 backlinks after delete, got %d", len(backlinks))
	}
}

func TestLinkByTitle(t *testing.T) {
	s, _ := tempKBStore(t)

	s.writeFileUnchecked(ScopeProject, CategoryLesson, "Go Errors",
		"error handling in go", []string{"go"}, "high")

	err := s.LinkByTitle(ScopeProject, CategoryLesson, "New Lesson", "Go Errors", "related")
	if err != nil {
		t.Fatalf("LinkByTitle: %v", err)
	}

	newPath := "wiki/lessons/new-lesson.md"
	edges, _ := s.getEdges(ScopeProject, newPath)
	if len(edges) == 0 {
		t.Error("expected edge from LinkByTitle")
	}
}
