package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKBStoreWriteAndSearch(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	err = store.WriteFile(ScopeProject, CategoryLesson, "Test Lesson",
		"This is a test lesson about goroutines and channels.", []string{"go", "concurrency"}, "high")
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err = store.WriteFile(ScopeProject, CategoryTopic, "VSCode MCP 配置指南",
		"VSCode + Claude Code 通过 MCP 操作 Unreal Engine 配置指南", []string{"vscode", "mcp", "unreal"}, "high")
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// 英文多词搜索（走 FTS5）
	results, err := store.Search("goroutines channels", ScopeProject, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Title != "Test Lesson" {
		t.Errorf("got '%s'", results[0].Title)
	}

	// 中文多词搜索（走 searchLike，拆分后 AND 组合）
	results, err = store.Search("unreal engine mcp vscode 配置", ScopeProject, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for Chinese multi-word, got %d", len(results))
	}
	if results[0].Title != "VSCode MCP 配置指南" {
		t.Errorf("got '%s'", results[0].Title)
	}
}

func TestKBStoreBuildMemoryBlock(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	store.WriteFile(ScopeGlobal, CategoryProfile, "User Profile", "Test profile.", nil, "high")
	store.WriteFile(ScopeGlobal, CategoryKnowledge, "Core", "Global knowledge.", nil, "high")
	store.WriteFile(ScopeProject, CategoryKnowledge, "Proj", "Project knowledge.", nil, "high")

	block := store.BuildMemoryBlock()
	if block == "" {
		t.Fatal("expected non-empty block")
	}
	if !strings.Contains(block, "<global_memory>") {
		t.Error("missing <global_memory>")
	}
	if !strings.Contains(block, "<project_memory>") {
		t.Error("missing <project_memory>")
	}
}

func TestKBStoreSoftDelete(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	store.WriteFile(ScopeProject, CategoryTopic, "To Delete",
		"Will be deleted.", nil, "low")
	store.SoftDelete(ScopeProject, "wiki/topics/to-delete.md")

	results, _ := store.Search("deleted", ScopeProject, 5)
	if len(results) != 0 {
		t.Errorf("expected 0 results after soft delete, got %d", len(results))
	}
}

func TestKBStoreSearchWithSpecialChars(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	err = store.WriteFile(ScopeProject, CategoryLesson, "Go Concurrency",
		"Using goroutines with channels for concurrent programming.", []string{"go"}, "high")
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// 包含 FTS5 特殊字符的查询不应导致语法错误
	queries := []string{
		"goroutines*",
		"(channels)",
		"goroutines: channels",
		"func (s *Store)",
		`"goroutines"`,
		"goroutines AND channels",
	}
	for _, q := range queries {
		results, err := store.Search(q, ScopeProject, 5)
		if err != nil {
			t.Errorf("Search(%q): %v", q, err)
		}
		// 至少不应报错；部分查询可能返回 0 结果（因为转义后分词不同）
		_ = results
	}
}

func writeDiskFile(t *testing.T, store *KBStore, scope, relPath, content string) {
	t.Helper()
	root := store.rootFor(scope)
	full := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestReindexFromDisk(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	// 直接写文件到磁盘，模拟 WriteFile 的输出格式，但不经过 DB
	knowledgeContent := "# Global Knowledge\n\n" +
		"> 类型：semantic\n" +
		"> 作用域：global\n" +
		"> 创建：2026-01-01T00:00:00Z\n" +
		"> 更新：2026-01-01T00:00:00Z\n" +
		"> 置信度：high\n" +
		"> 标签：\n" +
		"> 状态：active\n\n" +
		"用户偏好中文交流，使用 Go 和 React。"

	writeDiskFile(t, store, ScopeGlobal, "wiki/knowledge.md", knowledgeContent)

	// DB 索引为空时搜索应返回 0 结果
	results, _ := store.Search("用户偏好", ScopeGlobal, 5)
	if len(results) != 0 {
		t.Fatalf("before reindex: expected 0, got %d", len(results))
	}

	// Reindex 从磁盘重建索引
	count, err := store.ReindexFromDisk(ScopeGlobal)
	if err != nil {
		t.Fatalf("ReindexFromDisk: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 reindexed, got %d", count)
	}

	// 搜索现在应返回结果
	results, err = store.Search("用户偏好", ScopeGlobal, 5)
	if err != nil {
		t.Fatalf("Search after reindex: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("after reindex: expected 1, got %d", len(results))
	}
	if results[0].Title != "Global Knowledge" {
		t.Errorf("got title '%s'", results[0].Title)
	}
}
func TestKBStoreSearchReturnsSnippet(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	err = store.WriteFile(ScopeProject, CategoryLesson, "Snippet Test",
		"This lesson talks about goroutine leak detection patterns.", []string{"go"}, "high")
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	results, err := store.Search("goroutine leak", ScopeProject, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Snippet == "" {
		t.Errorf("expected non-empty Snippet, got empty")
	}
}
func TestKBStoreUpdateFile(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	err = store.WriteFile(ScopeProject, CategoryLesson, "Update Target",
		"Original content.", []string{"test"}, "high")
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// 读取实际路径
	results, err := store.Search("Update Target", ScopeProject, 1)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	path := results[0].Path

	// 正常更新
	err = store.UpdateFile(ScopeProject, path, "# Update Target\n\nUpdated content here.")
	if err != nil {
		t.Fatalf("UpdateFile: %v", err)
	}

	content, err := store.ReadFile(ScopeProject, path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(content, "Updated content here.") {
		t.Errorf("content not updated, got: %s", content)
	}

	// 验证 DB 索引和 FTS 也已刷新（UpdateFile 的核心功能）
	searchResults, err := store.Search("Updated content here.", ScopeProject, 1)
	if err != nil {
		t.Fatalf("Search after update: %v", err)
	}
	if len(searchResults) != 1 {
		t.Fatalf("expected 1 search result after update, got %d", len(searchResults))
	}
	if searchResults[0].Path != path {
		t.Errorf("search returned wrong path: got %s, want %s", searchResults[0].Path, path)
	}

	// 不存在的路径
	err = store.UpdateFile(ScopeProject, "wiki/lessons/nonexistent.md", "content")
	if err == nil {
		t.Error("expected error for nonexistent path, got nil")
	}

	// 路径穿越
	err = store.UpdateFile(ScopeProject, "../../etc/passwd", "content")
	if err == nil {
		t.Error("expected error for path traversal, got nil")
	}
}
