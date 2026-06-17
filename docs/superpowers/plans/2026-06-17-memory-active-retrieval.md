# 记忆主动检索与自更新机制 — 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 LLM 在任务生命周期中主动检索和使用记忆，通过工具闭环实现记忆的读/写/更新，删除脆弱的后台归档提取机制。

**Architecture:** 补全 memory 工具集（memory_read / memory_update / memory_search snippet 修复），将 system prompt 中冻结注入的 profile/knowledge 改为 LLM 主动检索，删除 ArchiveHook 整套后台提取链路改为 LLM 自主写入。

**Tech Stack:** Go 1.25, Wails v3, React/TypeScript, SQLite (modernc.org/sqlite), FTS5

**Spec:** `docs/superpowers/specs/2026-06-17-memory-active-retrieval-design.md`

---

## 文件结构

| 文件 | 职责 | 改动类型 |
|------|------|----------|
| `internal/memory/types.go` | KBFile 数据结构 | 修改：加 Snippet 字段 |
| `internal/memory/store.go` | KBStore 存储层 | 修改：scanKBFiles 赋值 Snippet；新增 UpdateFile 方法 |
| `internal/memory/compact_knowledge.go` | 字符上限常量 | 修改：导出常量 |
| `internal/memory/hook.go` | ArchiveHook 后台提取 | **删除** |
| `internal/memory/inject.go` | BuildMemoryBlock | 不改（保留函数，不再被调用） |
| `internal/tool/builtin/memory_search.go` | 搜索工具 | 修改：输出加 snippet |
| `internal/tool/builtin/memory_read.go` | 读取工具 | **新增** |
| `internal/tool/builtin/memory_update.go` | 更新工具 | **新增** |
| `internal/tool/builtin/memory_write.go` | 写入工具 | 修改：catMap 补 profile + 描述收窄 |
| `internal/tool/builtin/register.go` | 工具注册 | 修改：注册新工具 |
| `main.go` | 入口 | 修改：prompt 替换 + 删 ArchiveHook 构造 |
| `internal/agent/agent_loop.go` | 消息构建 | 修改：移除 BuildMemoryBlock 注入 |
| `internal/api/app.go` | App 层 | 修改：删 memory hook 相关代码 |
| `frontend/src/components/Chat/ChatInput.tsx` | 前端 | 修改：删 /memory 命令 |

---

## Task 1: KBFile.Snippet 字段修复

**Files:**
- Modify: `internal/memory/types.go:25-38`
- Modify: `internal/memory/store.go:473-495`
- Test: `internal/memory/store_test.go`

- [ ] **Step 1: 写失败测试 — 验证 Search 结果包含 Snippet**

在 `internal/memory/store_test.go` 末尾追加：

```go
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
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/memory/ -run TestKBStoreSearchReturnsSnippet -v`
Expected: FAIL — Snippet 为空（字段未赋值）

- [ ] **Step 3: 给 KBFile 加 Snippet 字段**

在 `internal/memory/types.go` 的 `KBFile` 结构体中，在 `LinkedTo` 字段之后添加：

```go
type KBFile struct {
	ID         int64     `json:"id"`
	Path       string    `json:"path"`
	Scope      string    `json:"scope"`
	Category   string    `json:"category"`
	Title      string    `json:"title"`
	Tags       []string  `json:"tags"`
	Confidence string    `json:"confidence"`
	Status     string    `json:"status"`
	CharCount  int       `json:"char_count"`
	LinkedTo   []string  `json:"linked_to,omitempty"`
	Snippet    string    `json:"snippet,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
```

- [ ] **Step 4: scanKBFiles 赋值 Snippet**

在 `internal/memory/store.go` 的 `scanKBFiles` 函数（约第 473-495 行），在 `f.CreatedAt, _ = ...` 之后、`results = append(...)` 之前，添加一行：

```go
func scanKBFiles(rows *sql.Rows) ([]KBFile, error) {
	var results []KBFile
	for rows.Next() {
		var f KBFile
		var tagsJSON, linkedJSON, ca, ua, snippet string
		if err := rows.Scan(&f.ID, &f.Path, &f.Scope, &f.Category, &f.Title, &tagsJSON,
			&f.Confidence, &f.Status, &f.CharCount, &linkedJSON, &ca, &ua, &snippet); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(tagsJSON), &f.Tags)
		json.Unmarshal([]byte(linkedJSON), &f.LinkedTo)
		f.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		f.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
		f.Snippet = snippet
		results = append(results, f)
	}
	if results == nil {
		results = []KBFile{}
	}
	return results, nil
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./internal/memory/ -run TestKBStoreSearchReturnsSnippet -v`
Expected: PASS

- [ ] **Step 6: 运行全部 memory 测试确保无回归**

Run: `go test ./internal/memory/ -v`
Expected: 全部 PASS

- [ ] **Step 7: 提交**

```bash
git add internal/memory/types.go internal/memory/store.go internal/memory/store_test.go
git commit -m "feat(memory): 修复 scanKBFiles 丢弃 snippet 的 bug

KBFile 结构体新增 Snippet 字段，scanKBFiles 将已查出的 SQL snippet
赋值给该字段。memory_search 工具此前只返回标题和标签，LLM 无法判断
记忆相关性。"
```

---

## Task 2: 导出字符上限常量

**Files:**
- Modify: `internal/memory/compact_knowledge.go:9-12`
- Test: 无需单独测试（常量重命名，编译即验证）

- [ ] **Step 1: 导出常量**

将 `internal/memory/compact_knowledge.go` 第 9-12 行的常量从私有改为导出：

```go
const (
	MaxKnowledgeChars = 3000
	MaxProfileChars   = 1500
)
```

- [ ] **Step 2: 全局替换引用**

在同文件内将所有 `maxKnowledgeChars` 替换为 `MaxKnowledgeChars`，`maxProfileChars` 替换为 `MaxProfileChars`。涉及行：
- 第 24 行：`if len([]rune(content)) <= MaxKnowledgeChars {`
- 第 29 行：`fmt.Sprintf("%d 字符以内...`, MaxKnowledgeChars)`
- 第 45 行：`if len(runes) > MaxKnowledgeChars {`
- 第 46 行：`compressed = string(runes[:MaxKnowledgeChars])`
- 第 58 行：`if len([]rune(content)) <= MaxProfileChars {`
- 第 62 行：`fmt.Sprintf("%d", MaxProfileChars)`
- 第 71 行：`if len(runes) > MaxProfileChars {`
- 第 72 行：`compressed = string(runes[:MaxProfileChars])`

- [ ] **Step 3: 编译确认**

Run: `go build ./internal/memory/`
Expected: 编译通过，无错误

- [ ] **Step 4: 提交**

```bash
git add internal/memory/compact_knowledge.go
git commit -m "refactor(memory): 导出 MaxKnowledgeChars/MaxProfileChars 常量

供 memory_update 工具检查字符上限使用。"
```

---

## Task 3: KBStore.UpdateFile 方法

**Files:**
- Modify: `internal/memory/store.go`（新增方法）
- Test: `internal/memory/store_test.go`

- [ ] **Step 1: 写失败测试**

在 `internal/memory/store_test.go` 末尾追加：

```go
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
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/memory/ -run TestKBStoreUpdateFile -v`
Expected: FAIL — `UpdateFile` 未定义

- [ ] **Step 3: 实现 UpdateFile 方法**

在 `internal/memory/store.go` 的 `ReadFile` 方法之后（约第 313 行之后）添加：

```go
// UpdateFile 按路径覆盖写入已有记忆，刷新索引和时间戳。
// 调用前应通过 ReadFile 确认路径存在。content 为完整的文件内容（含 frontmatter）。
func (s *KBStore) UpdateFile(scope, relPath, content string) error {
	cleanPath := filepath.Clean(relPath)
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("invalid path: %s", relPath)
	}
	fullPath := filepath.Join(s.rootFor(scope), cleanPath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("memory not found at path: %s", relPath)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return err
	}
	charCount := len([]rune(content))
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.dbFor(scope).Exec(`
		UPDATE file_index SET content = ?, char_count = ?, updated_at = ?
		WHERE path = ?
	`, content, charCount, now, cleanPath)
	if err != nil {
		return fmt.Errorf("update index: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/memory/ -run TestKBStoreUpdateFile -v`
Expected: PASS

- [ ] **Step 5: 运行全部 memory 测试**

Run: `go test ./internal/memory/ -v`
Expected: 全部 PASS

- [ ] **Step 6: 提交**

```bash
git add internal/memory/store.go internal/memory/store_test.go
git commit -m "feat(memory): 新增 KBStore.UpdateFile 按路径更新记忆

供 memory_update 工具使用：按 path 覆盖写文件 + 刷新索引 + 更新时间戳。
含路径校验（防穿越）和存在性检查。"
```

---

## Task 4: memory_search 输出加 snippet

**Files:**
- Modify: `internal/tool/builtin/memory_search.go:64-77`
- Test: `internal/tool/builtin/memory_search_test.go`（新建）

- [ ] **Step 1: 写失败测试**

新建 `internal/tool/builtin/memory_search_test.go`：

```go
package builtin

import (
	"context"
	"strings"
	"testing"

	"monika/internal/memory"
)

func TestMemorySearchOutputContainsSnippet(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := memory.NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	err = store.WriteFile(memory.ScopeProject, memory.CategoryLesson, "CORS Fix",
		"Wails v3 dev mode CORS configuration issue.", []string{"cors"}, "high")
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	tool := NewMemorySearch(store)
	args := []byte(`{"query": "CORS"}`)
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "snippet:") {
		t.Errorf("output should contain snippet line, got: %s", result.Content)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/tool/builtin/ -run TestMemorySearchOutputContainsSnippet -v`
Expected: FAIL — 输出不含 "snippet:"

- [ ] **Step 3: 修改 memory_search 输出格式**

在 `internal/tool/builtin/memory_search.go` 的 `Execute` 方法中，修改输出循环（约第 66-76 行），在 `sb.WriteString("\n")` 之前添加 snippet 行：

```go
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d matching memories:\n\n", len(results)))
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. **%s** [%s/%s] confidence: %s\n   path: %s | chars: %d\n",
			i+1, r.Title, r.Scope, r.Category, r.Confidence, r.Path, r.CharCount))
		if len(r.Tags) > 0 {
			sb.WriteString(fmt.Sprintf("   tags: %s\n", strings.Join(r.Tags, ", ")))
		}
		if len(r.LinkedTo) > 0 {
			sb.WriteString(fmt.Sprintf("   links: %s\n", strings.Join(r.LinkedTo, ", ")))
		}
		if r.Snippet != "" {
			sb.WriteString(fmt.Sprintf("   snippet: %s\n", r.Snippet))
		}
		sb.WriteString("\n")
	}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/tool/builtin/ -run TestMemorySearchOutputContainsSnippet -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/tool/builtin/memory_search.go internal/tool/builtin/memory_search_test.go
git commit -m "feat(memory_search): 输出结果增加 snippet 行

让 LLM 在搜索时看到高亮片段，能判断记忆相关性，无需先调 memory_read。"
```

---

## Task 5: memory_read 工具

**Files:**
- Create: `internal/tool/builtin/memory_read.go`
- Test: `internal/tool/builtin/memory_read_test.go`（新建）

- [ ] **Step 1: 写失败测试**

新建 `internal/tool/builtin/memory_read_test.go`：

```go
package builtin

import (
	"context"
	"strings"
	"testing"

	"monika/internal/memory"
)

func TestMemoryReadExistingFile(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := memory.NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	err = store.WriteFile(memory.ScopeProject, memory.CategoryLesson, "Read Test",
		"Full content of this memory.", []string{"test"}, "high")
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	results, err := store.Search("Read Test", memory.ScopeProject, 1)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	path := results[0].Path

	tool := NewMemoryRead(store)
	args := []byte(`{"path": "` + path + `"}`)
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "Full content of this memory.") {
		t.Errorf("expected full content, got: %s", result.Content)
	}
}

func TestMemoryReadNonExistent(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := memory.NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	tool := NewMemoryRead(store)
	args := []byte(`{"path": "wiki/lessons/nonexistent.md"}`)
	result, _ := tool.Execute(context.Background(), args)
	if !result.IsError {
		t.Error("expected error for nonexistent path")
	}
	if !strings.Contains(result.Content, "not found") {
		t.Errorf("error should mention 'not found', got: %s", result.Content)
	}
}

func TestMemoryReadPathTraversal(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := memory.NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	tool := NewMemoryRead(store)
	args := []byte(`{"path": "../../etc/passwd"}`)
	result, _ := tool.Execute(context.Background(), args)
	if !result.IsError {
		t.Error("expected error for path traversal")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/tool/builtin/ -run TestMemoryRead -v`
Expected: FAIL — `NewMemoryRead` 未定义

- [ ] **Step 3: 实现 memory_read 工具**

新建 `internal/tool/builtin/memory_read.go`：

```go
package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"monika/internal/memory"
	"monika/internal/tool"
)

type memoryReadTool struct{ store *memory.KBStore }

func NewMemoryRead(store *memory.KBStore) tool.Tool { return &memoryReadTool{store} }

func (t *memoryReadTool) Name() string { return "memory_read" }

func (t *memoryReadTool) Description() string {
	return "Read a single memory's full content by path. Use after memory_search to get complete details."
}

func (t *memoryReadTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":  map[string]any{"type": "string", "description": "Path of the memory file (from memory_search or memory_index results)."},
			"scope": map[string]any{"type": "string", "description": "'global' or 'project' (default)."},
		},
		"required": []string{"path"},
	}
}

func (t *memoryReadTool) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var p struct {
		Path  string `json:"path"`
		Scope string `json:"scope"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	if p.Scope == "" {
		p.Scope = memory.ScopeProject
	}

	content, err := t.store.ReadFile(p.Scope, p.Path)
	if err != nil {
		return tool.ExecutionResult{
			Content: fmt.Sprintf("Memory not found at path '%s' (scope: %s): %s", p.Path, p.Scope, err),
			IsError: true,
		}, nil
	}
	return tool.ExecutionResult{Content: content}, nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/tool/builtin/ -run TestMemoryRead -v`
Expected: PASS（3 个测试用例全部通过）

- [ ] **Step 5: 提交**

```bash
git add internal/tool/builtin/memory_read.go internal/tool/builtin/memory_read_test.go
git commit -m "feat(tools): 新增 memory_read 工具

按 path 读取单条记忆全文，供 LLM 在 memory_search 后查看完整内容。
复用 KBStore.ReadFile，含路径校验。"
```

---

## Task 6: memory_update 工具

**Files:**
- Create: `internal/tool/builtin/memory_update.go`
- Test: `internal/tool/builtin/memory_update_test.go`（新建）

- [ ] **Step 1: 写失败测试**

新建 `internal/tool/builtin/memory_update_test.go`：

```go
package builtin

import (
	"context"
	"strings"
	"testing"

	"monika/internal/memory"
)

func TestMemoryUpdateExistingFile(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := memory.NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	err = store.WriteFile(memory.ScopeProject, memory.CategoryLesson, "Update Me",
		"Original content.", []string{"test"}, "high")
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	results, err := store.Search("Update Me", memory.ScopeProject, 1)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	path := results[0].Path

	tool := NewMemoryUpdate(store)
	args := []byte(`{"path": "` + path + `", "content": "# Update Me\n\nMerged and updated content."}`)
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "updated") {
		t.Errorf("expected success message with 'updated', got: %s", result.Content)
	}

	// 验证实际写入
	read, _ := store.ReadFile(memory.ScopeProject, path)
	if !strings.Contains(read, "Merged and updated content.") {
		t.Errorf("content not actually written, got: %s", read)
	}
}

func TestMemoryUpdateNonExistent(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := memory.NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	tool := NewMemoryUpdate(store)
	args := []byte(`{"path": "wiki/lessons/nonexistent.md", "content": "test"}`)
	result, _ := tool.Execute(context.Background(), args)
	if !result.IsError {
		t.Error("expected error for nonexistent path")
	}
	if !strings.Contains(result.Content, "memory_write") {
		t.Errorf("error should guide to memory_write, got: %s", result.Content)
	}
}

func TestMemoryUpdateProfileOverflowWarning(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := memory.NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	// 先创建 profile.md
	err = store.WriteFile(memory.ScopeGlobal, memory.CategoryProfile, "User Profile",
		"Short profile.", nil, "medium")
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// 生成超长内容（超过 1500 字符）
	longContent := strings.Repeat("a", 2000)

	tool := NewMemoryUpdate(store)
	args := []byte(`{"path": "wiki/profile.md", "scope": "global", "content": "` + longContent + `"}`)
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// 超限时应写入成功但返回警告
	if result.IsError {
		t.Fatalf("should not be error, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "exceeds") && !strings.Contains(result.Content, "overflow") && !strings.Contains(result.Content, "limit") {
		t.Errorf("should warn about overflow, got: %s", result.Content)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/tool/builtin/ -run TestMemoryUpdate -v`
Expected: FAIL — `NewMemoryUpdate` 未定义

- [ ] **Step 3: 实现 memory_update 工具**

新建 `internal/tool/builtin/memory_update.go`：

```go
package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"monika/internal/memory"
	"monika/internal/tool"
)

type memoryUpdateTool struct{ store *memory.KBStore }

func NewMemoryUpdate(store *memory.KBStore) tool.Tool { return &memoryUpdateTool{store} }

func (t *memoryUpdateTool) Name() string { return "memory_update" }

func (t *memoryUpdateTool) Description() string {
	return "Update an existing memory by path with merged content. LLM should memory_read first, merge new insight, then pass the full merged content. Check for overflow warnings on profile/knowledge files."
}

func (t *memoryUpdateTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":    map[string]any{"type": "string", "description": "Path of the memory to update (from memory_search/memory_index)."},
			"content": map[string]any{"type": "string", "description": "Full merged content (including frontmatter) to overwrite the file."},
			"scope":   map[string]any{"type": "string", "description": "'global' or 'project' (default)."},
		},
		"required": []string{"path", "content"},
	}
}

func (t *memoryUpdateTool) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var p struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Scope   string `json:"scope"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	if p.Scope == "" {
		p.Scope = memory.ScopeProject
	}

	if err := t.store.UpdateFile(p.Scope, p.Path, p.Content); err != nil {
		return tool.ExecutionResult{
			Content: fmt.Sprintf("Failed to update '%s': %s. Use memory_write to create a new memory.", p.Path, err),
			IsError: true,
		}, nil
	}

	msg := fmt.Sprintf("Memory '%s' updated in %s scope.", p.Path, p.Scope)

	// 字符上限检查
	if warn := checkCharLimit(p.Path, p.Content); warn != "" {
		msg += "\n" + warn
	}

	return tool.ExecutionResult{Content: msg}, nil
}

// checkCharLimit 检查 profile/knowledge 文件是否超出字符上限。
// 返回警告字符串（超限时）或空字符串（未超限）。
func checkCharLimit(path, content string) string {
	charCount := len([]rune(content))
	lowerPath := strings.ToLower(path)

	if strings.Contains(lowerPath, "profile.md") && charCount > memory.MaxProfileChars {
		return fmt.Sprintf("⚠️ Content exceeds profile limit (%d/%d chars). Written successfully, but please read it back and trim to fit the limit using memory_update.", charCount, memory.MaxProfileChars)
	}
	if strings.Contains(lowerPath, "knowledge.md") && charCount > memory.MaxKnowledgeChars {
		return fmt.Sprintf("⚠️ Content exceeds knowledge limit (%d/%d chars). Written successfully, but please read it back and trim to fit the limit using memory_update.", charCount, memory.MaxKnowledgeChars)
	}
	return ""
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/tool/builtin/ -run TestMemoryUpdate -v`
Expected: PASS（3 个测试用例全部通过）

- [ ] **Step 5: 提交**

```bash
git add internal/tool/builtin/memory_update.go internal/tool/builtin/memory_update_test.go
git commit -m "feat(tools): 新增 memory_update 工具

按 path 覆盖写已有记忆，含字符上限警告（profile 1500/knowledge 3000）。
超限时不阻止写入，而是警告 LLM 自行精简。"
```

---

## Task 7: memory_write 补 profile 映射 + 语义收窄

**Files:**
- Modify: `internal/tool/builtin/memory_write.go`
- Test: `internal/tool/builtin/memory_write_test.go`（新建）

- [ ] **Step 1: 写失败测试**

新建 `internal/tool/builtin/memory_write_test.go`：

```go
package builtin

import (
	"context"
	"strings"
	"testing"

	"monika/internal/memory"
)

func TestMemoryWriteProfileCategory(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := memory.NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	tool := NewMemoryWrite(store)
	args := []byte(`{"title": "User Profile", "content": "Test profile.", "category": "profile", "scope": "global"}`)
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}

	// 验证 profile.md 确实被写入
	content, err := store.ReadFile(memory.ScopeGlobal, "wiki/profile.md")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(content, "Test profile.") {
		t.Errorf("profile not written, got: %s", content)
	}
}

func TestMemoryWriteReturnMentionsUpdate(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()

	store, err := memory.NewKBStore(homeDir, projectDir)
	if err != nil {
		t.Fatalf("NewKBStore: %v", err)
	}
	defer store.Close()

	tool := NewMemoryWrite(store)
	args := []byte(`{"title": "Test", "content": "Test.", "category": "lesson"}`)
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(result.Content, "memory_update") {
		t.Errorf("return message should mention memory_update for existing memories, got: %s", result.Content)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/tool/builtin/ -run TestMemoryWrite -v`
Expected: FAIL — profile category 无效（catMap 缺映射）

- [ ] **Step 3: 修改 memory_write**

在 `internal/tool/builtin/memory_write.go` 中做三处修改：

**3a. Description 改为（第 19 行）：**

```go
func (t *memoryWriteTool) Description() string {
	return "Create a NEW memory entry. If a similar memory may already exist, use memory_search first then memory_update."
}
```

**3b. catMap 补 profile 映射（第 56-60 行）：**

```go
	catMap := map[string]string{
		"lesson":           memory.CategoryLesson,
		"topic":            memory.CategoryTopic,
		"knowledge_update": memory.CategoryKnowledge,
		"profile":          memory.CategoryProfile,
	}
```

**3c. 返回信息追加 update 提示（第 71 行）：**

```go
	return tool.ExecutionResult{Content: fmt.Sprintf("Memory '%s' written to %s scope. If you intended to update an existing memory, use memory_update instead.", p.Title, p.Scope)}, nil
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/tool/builtin/ -run TestMemoryWrite -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/tool/builtin/memory_write.go internal/tool/builtin/memory_write_test.go
git commit -m "feat(memory_write): 补 profile 映射 + 语义收窄为'新建'

catMap 补 profile 映射让 LLM 能写 profile.md。Description 明确为
'create NEW'，返回提示引导 LLM 在更新场景使用 memory_update。"
```

---

## Task 8: 注册新工具

**Files:**
- Modify: `internal/tool/builtin/register.go:261`

- [ ] **Step 1: 注册 memory_read 和 memory_update**

在 `internal/tool/builtin/register.go` 的 `NewMemorySearch(store)` 注册行之后（约第 261 行），添加：

```go
	r.Register(NewMemorySearch(store))
	r.Register(NewMemoryRead(store))
	r.Register(NewMemoryUpdate(store))
```

- [ ] **Step 2: 编译确认**

Run: `go build ./internal/tool/builtin/`
Expected: 编译通过

- [ ] **Step 3: 提交**

```bash
git add internal/tool/builtin/register.go
git commit -m "feat(tools): 注册 memory_read 和 memory_update 工具"
```

---

## Task 9: 替换 system prompt（Knowledge Base 区块）

**Files:**
- Modify: `main.go:236-255`

- [ ] **Step 1: 替换 Knowledge Base 区块**

在 `main.go` 中，找到第 236-255 行的 Knowledge Base 区块（从 `` `## Knowledge Base (Memory)` `` 开始到 `` `knowledge: user preferences, project constraints, persistent facts` `` 结束），替换为：

```go
		`## Knowledge Base (Memory)

You have access to a self-evolving knowledge base that persists across sessions.
Use memory_search/memory_read to look up relevant knowledge on demand.

**MEMORY USAGE — mandatory task lifecycle:**

Every task MUST follow this closed loop. Skipping steps degrades quality over time.

1. **BEFORE acting** — memory_search(query) to check for relevant past experience
   (similar problems, conventions, user preferences). If results look relevant,
   memory_read(path) for full content. Apply what you find.
2. **DURING** — execute the task normally.
3. **AFTER completing** — if you learned something worth keeping (a bug root cause,
   a working pattern, a user preference, a project convention):
   - memory_search first to check if a similar memory already exists
   - exists → memory_read full content → merge new insight → memory_update(path, merged)
   - not exists → memory_write(title, content, category)
   
   For profile (wiki/profile.md) and core knowledge (wiki/knowledge.md), use
   memory_update with the specific path. These files have character limits
   (profile: 1500, knowledge: 3000) — the tool warns on overflow, then you
   must read back and trim.

**Also:** Before any web search, MCP tool, or asking the user, you MUST first call
memory_search — only fall back to external sources if no relevant memory exists.

**Tools:**
- memory_search(query, scope?, category?, limit?) — search; returns title + snippet
- memory_read(path, scope?) — read a single memory's full content
- memory_write(title, content, category, scope?, tags?, confidence?) — create NEW memory
- memory_update(path, content, scope?) — overwrite existing memory with merged content
- memory_index(scope?) — list all memories by category

**Memory types:** lessons (bugs/causes/solutions), topics (architecture/patterns),
knowledge (preferences/constraints/persistent facts).`
```

- [ ] **Step 2: 编译确认**

Run: `go build .`
Expected: 编译通过

- [ ] **Step 3: 提交**

```bash
git add main.go
git commit -m "feat(prompt): 替换记忆区块为任务生命周期闭环指令

将'web search 前先搜'单点触发扩展为 BEFORE/DURING/AFTER 全周期闭环。
明确 profile/knowledge 通过 memory_update 主动写入。"
```

---

## Task 10: 移除 BuildMemoryBlock 在 system prompt 中的注入

**Files:**
- Modify: `internal/agent/agent_loop.go:1181-1185`

- [ ] **Step 1: 删除 BuildMemoryBlock 注入逻辑**

在 `internal/agent/agent_loop.go` 的 `buildMessages` 方法中，找到第 1181-1185 行：

```go
		if a.kbStore != nil {
			if block := a.kbStore.BuildMemoryBlock(); block != "" {
				parts = append(parts, block)
			}
		}
```

删除整段。profile/knowledge 不再注入 system prompt，改为 LLM 通过工具主动检索。

- [ ] **Step 2: 编译确认**

Run: `go build ./internal/agent/`
Expected: 编译通过

- [ ] **Step 3: 提交**

```bash
git add internal/agent/agent_loop.go
git commit -m "refactor(agent): 移除 BuildMemoryBlock 在 system prompt 中的注入

profile/knowledge 不再冻结注入，改由 LLM 通过 memory_search/memory_read
主动检索。system prompt 完全稳定，跨 session prefix cache 始终命中。"
```

---

## Task 11: 删除 ArchiveHook 后台提取链路

**Files:**
- Delete: `internal/memory/hook.go`
- Modify: `internal/api/app.go`（删 memoryHook 相关代码）
- Modify: `main.go`（删 ArchiveHook 构造）

- [ ] **Step 1: 删除 hook.go**

```bash
rm internal/memory/hook.go
```

- [ ] **Step 2: 删除 app.go 中的 memory hook 相关代码**

在 `internal/api/app.go` 中：

**2a.** 删除 `memoryHook` 字段（第 77 行附近）：
```go
	memoryHook      *memory.ArchiveHook
```

**2b.** 删除 ArchiveSession 内的注释死代码（第 604-609 行附近）：
```go
	// 会话归档自动触发记忆提取 — 暂不支持
	// if a.memoryHook != nil {
	// 	summary := extractCompactionSummary(s)
	// 	scope := memory.ScopeProject
	// 	go a.memoryHook.OnArchive(context.Background(), scope, sessionID, summary)
	// }
```

**2c.** 删除 `extractCompactionSummary` 函数（第 615-640 行附近，整个函数）。

**2d.** 删除 `SetMemoryHook` 方法（第 643-645 行）：
```go
func (a *App) SetMemoryHook(hook *memory.ArchiveHook) {
	a.memoryHook = hook
}
```

**2e.** 删除 `TriggerMemorySummarize` 方法（第 647-668 行，整个方法）。

**2f.** 检查是否还需要 `import "monika/internal/memory"`：grep 该文件内是否还有其他 memory 包引用。如果 app.go 内还有其他地方用 memory 包（如 kb_api.go 中的 store），则保留 import。如果只有已删除的代码用到，则删除 import。

- [ ] **Step 3: 删除 main.go 中的 ArchiveHook 构造**

在 `main.go` 中找到 ArchiveHook 构造和 SetMemoryHook 调用（第 392-397 行附近）：

```go
		hook := &memory.ArchiveHook{
			Store:          kbStore,
			LLM:            extractionAdapter,
			CompactionLLM:  compactionAdapter,
			OnStatusChange: func(status string) { ... },
		}
		appService.SetMemoryHook(hook)
```

删除整段。同时检查 main.go 中 `extractionAdapter` / `compactionAdapter` 是否还有其他引用——如果只被这段代码使用，一并删除其构造代码。

- [ ] **Step 4: 编译确认**

Run: `go build ./...`
Expected: 编译通过。如果报错（如 extractionAdapter 未使用、import 多余），逐个修复。

- [ ] **Step 5: 运行测试确认无回归**

Run: `go test ./internal/memory/ ./internal/api/ -v`
Expected: 全部 PASS

- [ ] **Step 6: 提交**

```bash
git add -A
git commit -m "refactor(memory): 删除 ArchiveHook 后台提取链路

删除 hook.go 整文件、app.go 中 memoryHook/SetMemoryHook/
TriggerMemorySummarize/extractCompactionSummary、main.go 中 ArchiveHook
构造。记忆更新完全由 LLM 通过 memory_write/memory_update 工具自主完成。"
```

---

## Task 12: 删除前端 /memory 命令

**Files:**
- Modify: `frontend/src/components/Chat/ChatInput.tsx:688-701`

- [ ] **Step 1: 删除 /memory 命令处理**

在 `frontend/src/components/Chat/ChatInput.tsx` 中，删除第 688-701 行的 `/memory` 命令处理块：

```typescript
        if (resolved === '/memory') {
            if (!projectPath || !activeSessionId) return
            setValue('')
            const store = useStore.getState()
            store.appendToSession(activeSessionId, [{ id: crypto.randomUUID(), role: 'assistant', content: '⏳ 正在从当前会话提取记忆...' }])
            App.TriggerMemorySummarize(projectPath, activeSessionId).then(() => {
                const s = useStore.getState()
                s.appendToSession(activeSessionId, [{ id: crypto.randomUUID(), role: 'assistant', content: '✅ 记忆提取完成，请打开设置 → 知识库查看。' }])
            }).catch((err: unknown) => {
                const s = useStore.getState()
                s.appendToSession(activeSessionId, [{ id: crypto.randomUUID(), role: 'assistant', content: `❌ 记忆提取失败: ${String(err)}` }])
            })
            return
        }
```

- [ ] **Step 2: 检查并删除 TriggerMemorySummarize 的绑定声明**

在 frontend bindings 中搜索 `TriggerMemorySummarize` 的声明（通常在自动生成的 `frontend/bindings/` 目录或 `.wails` 类型文件中），如果存在则删除。由于 Wails 绑定是自动生成的，后端方法删除后重新 `wails generate` 即可，此处只需删除前端调用。

- [ ] **Step 3: 前端类型检查**

Run: `cd frontend && npx tsc --noEmit`
Expected: 无类型错误（如果 TriggerMemorySummarize 在 App 类型中仍有声明但已被删除的后端方法，需要重新生成 wails 绑定）

- [ ] **Step 4: 提交**

```bash
git add frontend/src/components/Chat/ChatInput.tsx
git commit -m "refactor(frontend): 删除 /memory 命令处理

记忆提取不再需要手动触发，LLM 通过工具自主管理记忆。"
```

---

## Task 13: 全量验证

- [ ] **Step 1: Go 全量编译和测试**

Run: `go build ./... && go test ./... -v`
Expected: 全部编译通过，全部测试 PASS

- [ ] **Step 2: Go vet**

Run: `go vet ./...`
Expected: 无警告

- [ ] **Step 3: 前端编译**

Run: `cd frontend && npm run build`
Expected: 编译通过

- [ ] **Step 4: 确认无残留引用**

Run: `grep -r "ArchiveHook\|SetMemoryHook\|TriggerMemorySummarize\|extractCompactionSummary\|OnArchive\|memoryHook" --include="*.go" .`
Expected: 无输出（或仅有 spec/plan 文档中的引用）

- [ ] **Step 5: 提交（如有修复）**

```bash
git add -A
git commit -m "test: 全量验证通过 — 编译/测试/vet/前端构建均无错误"
```

---

## 自审记录

**1. Spec coverage 检查：**
- §4.1.1 KBFile.Snippet → Task 1 ✅
- §4.1.2 memory_search snippet 输出 → Task 4 ✅
- §4.1.3 memory_read → Task 5 ✅
- §4.1.4 memory_update（含字符上限） → Task 6 + Task 2（常量导出） ✅
- §4.1.5 scope 默认值 → Task 5/6 代码中实现（project 默认） ✅
- §4.1.6 memory_write catMap + 语义收窄 → Task 7 ✅
- §4.1.7 工具注册 → Task 8 ✅
- §4.2 Prompt 闭环 → Task 9 ✅
- §4.3 移除 BuildMemoryBlock 注入 → Task 10 ✅
- §4.4 删除 ArchiveHook → Task 11 ✅
- 前端 /memory 删除 → Task 12 ✅

**2. Placeholder scan：** 无 TBD/TODO，每步都有完整代码。

**3. Type consistency：** `NewMemoryRead` / `NewMemoryUpdate` 在 Task 5/6 定义，Task 8 引用名称一致。`UpdateFile` 在 Task 3 定义，Task 6 引用一致。`MaxKnowledgeChars` / `MaxProfileChars` 在 Task 2 导出，Task 6 引用一致。
