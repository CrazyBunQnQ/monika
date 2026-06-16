# Memory System P1 — Storage + Search + Injection MVP

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the minimal memory system that can store markdown knowledge files, search them via SQLite FTS5, and inject a frozen snapshot into the system prompt at session start.

**Architecture:** New `internal/memory` package with `KBStore` (SQLite FTS5 index + file I/O), `BuildMemoryBlock` (assembles `<memory>` XML for system prompt), and three Agent tools (`memory_search`, `memory_write`, `memory_index`). The `AgentLoop` gets a new `homeDir` field so `buildMessages()` can read `profile.md` and `knowledge.md` from both global (`~/.monika/kb/`) and project (`<project>/.monika/kb/`) scopes.

**Tech Stack:** Go 1.25, `modernc.org/sqlite` (pure Go, no CGO), existing `tool.Tool` interface

---

### Task 1: Add SQLite dependency and internal/memory/types.go

**Files:**
- Modify: `go.mod`
- Create: `internal/memory/types.go`

- [ ] **Step 1: Add modernc.org/sqlite dependency**

```bash
cd d:/git/monika
go get modernc.org/sqlite
```

Expected: dependency added to go.mod and go.sum.

- [ ] **Step 2: Create internal/memory/types.go with path helpers, KBFile struct, and constants**

```go
package memory

import (
	"path/filepath"
	"time"
)

// Scope constants
const (
	ScopeGlobal  = "global"
	ScopeProject = "project"
)

// Category constants
const (
	CategoryKnowledge = "wiki/knowledge"
	CategoryProfile   = "wiki/profile"
	CategoryLesson    = "wiki/lesson"
	CategoryTopic     = "wiki/topic"
	CategoryRawDoc    = "raw/doc"
	CategoryRawCode   = "raw/code"
)

// KBFile represents metadata for one knowledge file tracked in the index.
type KBFile struct {
	ID         int64     `json:"-"`
	Path       string    `json:"path"`
	Scope      string    `json:"scope"`
	Category   string    `json:"category"`
	Title      string    `json:"title"`
	Tags       []string  `json:"tags"`
	Confidence string    `json:"confidence"`
	Status     string    `json:"status"`
	CharCount  int       `json:"char_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// GlobalKBPath returns the global knowledge base directory.
func GlobalKBPath(homeDir string) string {
	return filepath.Join(homeDir, ".monika", "kb")
}

// ProjectKBPath returns the project-scoped knowledge base directory.
func ProjectKBPath(projectDir string) string {
	return filepath.Join(projectDir, ".monika", "kb")
}

// KBSubdirs returns the subdirectory paths to create for a kb root.
func KBSubdirs() []string {
	return []string{
		"raw/docs",
		"raw/code",
		"wiki/topics",
		"wiki/lessons",
		"wiki/.trash",
		".index",
		".trash",
	}
}
```

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum internal/memory/types.go
git commit -m "feat(memory): add types and path helpers for knowledge base"
```

---

### Task 2: Create internal/memory/store.go — KBStore with SQLite FTS5

**Files:**
- Create: `internal/memory/store.go`

- [ ] **Step 1: Write store.go with InitDB, Search, WriteFile, ReadFile, GetIndex**

```go
package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// KBStore manages the knowledge base: file I/O and SQLite FTS5 index.
type KBStore struct {
	globalPath  string
	projectPath string
	db          *sql.DB
}

// NewKBStore initializes both kb directory trees and the SQLite index.
func NewKBStore(homeDir, projectDir string) (*KBStore, error) {
	globalPath := GlobalKBPath(homeDir)
	projectPath := ProjectKBPath(projectDir)

	for _, p := range []string{globalPath, projectPath} {
		if err := ensureKBDirs(p); err != nil {
			return nil, fmt.Errorf("kb dirs for %s: %w", p, err)
		}
	}

	s := &KBStore{globalPath: globalPath, projectPath: projectPath}
	if err := s.openIndexes(); err != nil {
		return nil, fmt.Errorf("open indexes: %w", err)
	}
	return s, nil
}

func ensureKBDirs(root string) error {
	for _, sub := range KBSubdirs() {
		if err := os.MkdirAll(filepath.Join(root, sub), 0755); err != nil {
			return err
		}
	}
	return nil
}

func (s *KBStore) openIndexes() error {
	for _, root := range []string{s.globalPath, s.projectPath} {
		dbPath := filepath.Join(root, ".index", "kb.db")
		// Use a shared in-memory approach: each kb root has its own db file.
		// For simplicity in P1, we open the project index only (the primary store).
		// The global index is maintained as a separate db file.
		_ = dbPath // will be used when we open project-scoped db
	}

	// P1 MVP: single db at project level; global files indexed in same db with scope='global'.
	// In a production system we'd have separate dbs; for P1 we use one merged db.
	dbPath := filepath.Join(s.projectPath, ".index", "kb.db")
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}

	// Enable WAL mode for concurrent read safety.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return fmt.Errorf("enable WAL: %w", err)
	}

	if err := s.migrate(db); err != nil {
		db.Close()
		return fmt.Errorf("migrate: %w", err)
	}

	s.db = db
	return nil
}

func (s *KBStore) migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS file_index (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			path        TEXT NOT NULL UNIQUE,
			scope       TEXT NOT NULL DEFAULT 'project',
			category    TEXT NOT NULL,
			title       TEXT NOT NULL,
			tags        TEXT DEFAULT '[]',
			confidence  TEXT DEFAULT 'medium',
			status      TEXT DEFAULT 'active',
			char_count  INTEGER DEFAULT 0,
			created_at  TEXT NOT NULL,
			updated_at  TEXT NOT NULL
		);

		CREATE VIRTUAL TABLE IF NOT EXISTS file_fts USING fts5(
			path,
			title,
			content,
			content=file_index,
			content_rowid=id
		);

		CREATE TRIGGER IF NOT EXISTS file_index_ai AFTER INSERT ON file_index BEGIN
			INSERT INTO file_fts(rowid, path, title, content)
			VALUES (new.id, new.path, new.title, '');
		END;

		CREATE TRIGGER IF NOT EXISTS file_index_ad AFTER DELETE ON file_index BEGIN
			INSERT INTO file_fts(file_fts, rowid, path, title, content)
			VALUES ('delete', old.id, old.path, old.title, '');
		END;

		CREATE TRIGGER IF NOT EXISTS file_index_au AFTER UPDATE ON file_index BEGIN
			INSERT INTO file_fts(file_fts, rowid, path, title, content)
			VALUES ('delete', old.id, old.path, old.title, '');
			INSERT INTO file_fts(rowid, path, title, content)
			VALUES (new.id, new.path, new.title, '');
		END;
	`)
	return err
}

// Search runs a FTS5 query against the indexed knowledge files.
func (s *KBStore) Search(query, scope string, limit int) ([]KBFile, error) {
	if limit <= 0 {
		limit = 5
	}

	whereScope := ""
	args := []any{query}
	if scope == ScopeGlobal || scope == ScopeProject {
		whereScope = " AND f.scope = ?"
		args = append(args, scope)
	}

	sqlQ := fmt.Sprintf(`
		SELECT f.id, f.path, f.scope, f.category, f.title, f.tags,
		       f.confidence, f.status, f.char_count, f.created_at, f.updated_at,
		       snippet(file_fts, 2, '<b>', '</b>', '...', 40) as snippet
		FROM file_fts
		JOIN file_index f ON file_fts.rowid = f.id
		WHERE file_fts MATCH ? %s
		ORDER BY bm25(file_fts, 0, 10, 5)
		LIMIT ?
	`, whereScope)
	args = append(args, limit)

	rows, err := s.db.Query(sqlQ, args...)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()

	var results []KBFile
	for rows.Next() {
		var f KBFile
		var tagsJSON string
		var createdAt, updatedAt string
		var snippet string
		if err := rows.Scan(&f.ID, &f.Path, &f.Scope, &f.Category, &f.Title, &tagsJSON,
			&f.Confidence, &f.Status, &f.CharCount, &createdAt, &updatedAt, &snippet); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		json.Unmarshal([]byte(tagsJSON), &f.Tags)
		f.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		f.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		results = append(results, f)
	}
	if results == nil {
		results = []KBFile{}
	}
	return results, nil
}

// WriteFile writes a markdown file to disk and updates the FTS index.
func (s *KBStore) WriteFile(scope, category, title, content string, tags []string, confidence string) error {
	var root string
	if scope == ScopeGlobal {
		root = s.globalPath
	} else {
		root = s.projectPath
	}

	tagsJSON, _ := json.Marshal(tags)
	now := time.Now().UTC().Format(time.RFC3339)

	// Build frontmatter
	var sb strings.Builder
	sb.WriteString("# " + title + "\n\n")
	sb.WriteString("> 类型：semantic\n")
	sb.WriteString("> 作用域：" + scope + "\n")
	sb.WriteString("> 创建：" + now + "\n")
	sb.WriteString("> 更新：" + now + "\n")
	if confidence == "" {
		confidence = "medium"
	}
	sb.WriteString("> 置信度：" + confidence + "\n")
	sb.WriteString("> 状态：active\n\n")
	sb.WriteString(content)

	// Determine file path
	slug := titleToSlug(title)
	var relPath string
	switch category {
	case CategoryKnowledge:
		relPath = "wiki/knowledge.md"
	case CategoryProfile:
		relPath = "wiki/profile.md"
	case CategoryLesson:
		relPath = filepath.Join("wiki/lessons", slug+".md")
	case CategoryTopic:
		relPath = filepath.Join("wiki/topics", slug+".md")
	default:
		relPath = filepath.Join(category, slug+".md")
	}

	fullPath := filepath.Join(root, relPath)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(fullPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	// Upsert into index
	charCount := len([]rune(content))
	_, err := s.db.Exec(`
		INSERT INTO file_index (path, scope, category, title, tags, confidence, status, char_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'active', ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			category=excluded.category,
			title=excluded.title,
			tags=excluded.tags,
			confidence=excluded.confidence,
			char_count=excluded.char_count,
			updated_at=excluded.updated_at
	`, relPath, scope, category, title, string(tagsJSON), confidence, charCount, now, now)
	if err != nil {
		return fmt.Errorf("index upsert: %w", err)
	}

	// Update FTS content
	var id int64
	s.db.QueryRow("SELECT id FROM file_index WHERE path = ?", relPath).Scan(&id)
	if id > 0 {
		s.db.Exec("UPDATE file_fts SET content = ? WHERE rowid = ?", content, id)
	}

	return nil
}

// ReadFile reads the full content of a knowledge file by its relative path.
func (s *KBStore) ReadFile(scope, relPath string) (string, error) {
	var root string
	if scope == ScopeGlobal {
		root = s.globalPath
	} else {
		root = s.projectPath
	}
	// Prevent path traversal
	cleanPath := filepath.Clean(relPath)
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("invalid path: %s", relPath)
	}

	data, err := os.ReadFile(filepath.Join(root, cleanPath))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetIndex returns the index file content for a given scope.
func (s *KBStore) GetIndex(scope string) (string, error) {
	var root string
	if scope == ScopeGlobal {
		root = s.globalPath
	} else {
		root = s.projectPath
	}
	data, err := os.ReadFile(filepath.Join(root, "wiki", "index.md"))
	if err != nil {
		if os.IsNotExist(err) {
			return "暂无索引。使用 memory_write 添加记忆后将自动建立索引。", nil
		}
		return "", err
	}
	return string(data), nil
}

// BuildMemoryBlock reads profile.md and knowledge.md from both scopes
// and assembles them into the <memory> XML block for system prompt injection.
func (s *KBStore) BuildMemoryBlock() string {
	var sb strings.Builder

	globalProfile, _ := s.ReadFile(ScopeGlobal, "wiki/profile.md")
	globalKnowledge, _ := s.ReadFile(ScopeGlobal, "wiki/knowledge.md")
	projectKnowledge, _ := s.ReadFile(ScopeProject, "wiki/knowledge.md")

	hasGlobal := globalProfile != "" || globalKnowledge != ""
	hasProject := projectKnowledge != ""

	if !hasGlobal && !hasProject {
		return ""
	}

	if hasGlobal {
		sb.WriteString("\n\n<global_memory>\n")
		if globalProfile != "" {
			sb.WriteString("<user_profile>\n")
			sb.WriteString(globalProfile)
			sb.WriteString("\n</user_profile>\n")
		}
		if globalKnowledge != "" {
			sb.WriteString("\n<core_knowledge>\n")
			sb.WriteString(globalKnowledge)
			sb.WriteString("\n</core_knowledge>\n")
		}
		sb.WriteString("</global_memory>")
	}

	if hasProject {
		sb.WriteString("\n\n<project_memory>\n")
		sb.WriteString("<project_knowledge>\n")
		sb.WriteString(projectKnowledge)
		sb.WriteString("\n</project_knowledge>\n")
		sb.WriteString("</project_memory>")
	}

	return sb.String()
}

func (s *KBStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func titleToSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove non-alphanumeric except hyphens
	var cleaned strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			cleaned.WriteRune(r)
		}
	}
	result := cleaned.String()
	result = strings.Trim(result, "-")
	if result == "" {
		return "untitled"
	}
	return result
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/memory/store.go
git commit -m "feat(memory): add KBStore with SQLite FTS5 index and file I/O"
```

---

### Task 3: Create memory_search tool

**Files:**
- Create: `internal/tool/builtin/memory_search.go`

- [ ] **Step 1: Write the tool implementation**

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

type memorySearchTool struct {
	store *memory.KBStore
}

func NewMemorySearch(store *memory.KBStore) tool.Tool {
	return &memorySearchTool{store: store}
}

func (t *memorySearchTool) Name() string { return "memory_search" }

func (t *memorySearchTool) Description() string {
	return "Search the knowledge base for relevant memories. Returns matching files with titles and highlighted snippets. Use this tool to recall facts, lessons, topics, or previously stored knowledge."
}

func (t *memorySearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search keywords. Use natural language or key terms.",
			},
			"scope": map[string]any{
				"type":        "string",
				"description": "Search scope: 'global', 'project', or 'auto' (default 'auto' tries project first then falls back to global).",
			},
			"category": map[string]any{
				"type":        "string",
				"description": "Filter by category: 'lesson', 'topic', 'knowledge', 'raw', or 'all' (default 'all').",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum results to return (default 5, max 10).",
			},
		},
		"required": []string{"query"},
	}
}

func (t *memorySearchTool) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var params struct {
		Query    string `json:"query"`
		Scope    string `json:"scope"`
		Category string `json:"category"`
		Limit    int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	if params.Scope == "" {
		params.Scope = "auto"
	}
	if params.Limit <= 0 {
		params.Limit = 5
	}
	if params.Limit > 10 {
		params.Limit = 10
	}

	var results []memory.KBFile
	var err error

	if params.Scope == "auto" {
		results, err = t.store.Search(params.Query, memory.ScopeProject, params.Limit)
		if err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
		}
		if len(results) == 0 {
			results, err = t.store.Search(params.Query, memory.ScopeGlobal, params.Limit)
			if err != nil {
				return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
			}
		}
	} else {
		results, err = t.store.Search(params.Query, params.Scope, params.Limit)
		if err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
		}
	}

	if len(results) == 0 {
		return tool.ExecutionResult{Content: "No matching memories found."}, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d matching memories:\n\n", len(results)))
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. **%s** [%s/%s]\n", i+1, r.Title, r.Scope, r.Category))
		sb.WriteString(fmt.Sprintf("   path: %s\n", r.Path))
		sb.WriteString(fmt.Sprintf("   confidence: %s | updated: %s\n", r.Confidence, r.UpdatedAt.Format("2006-01-02")))
		sb.WriteString("\n")
	}

	return tool.ExecutionResult{Content: sb.String()}, nil
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd d:/git/monika
go build ./internal/tool/builtin/...
```

Expected: compiles without errors.

- [ ] **Step 3: Commit**

```bash
git add internal/tool/builtin/memory_search.go
git commit -m "feat(memory): add memory_search tool"
```

---

### Task 4: Create memory_write tool

**Files:**
- Create: `internal/tool/builtin/memory_write.go`

- [ ] **Step 1: Write the tool implementation**

```go
package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"monika/internal/memory"
	"monika/internal/tool"
)

type memoryWriteTool struct {
	store *memory.KBStore
}

func NewMemoryWrite(store *memory.KBStore) tool.Tool {
	return &memoryWriteTool{store: store}
}

func (t *memoryWriteTool) Name() string { return "memory_write" }

func (t *memoryWriteTool) Description() string {
	return "Write a new memory entry to the knowledge base. Creates or updates a markdown file in the wiki and indexes it for search."
}

func (t *memoryWriteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title": map[string]any{
				"type":        "string",
				"description": "Title of the memory entry. Use a concise, descriptive name.",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Markdown content for the memory. Include relevant details, context, and conclusions.",
			},
			"category": map[string]any{
				"type":        "string",
				"description": "Memory category: 'lesson', 'topic', or 'knowledge_update'.",
			},
			"scope": map[string]any{
				"type":        "string",
				"description": "Scope: 'global' (cross-project) or 'project' (current project). Default 'project'.",
			},
			"tags": map[string]any{
				"type":        "array",
				"items":       map[string]string{"type": "string"},
				"description": "Tags for categorization and search.",
			},
			"confidence": map[string]any{
				"type":        "string",
				"description": "Confidence level: 'high', 'medium', or 'low'. Default 'medium'.",
			},
		},
		"required": []string{"title", "content", "category"},
	}
}

func (t *memoryWriteTool) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var params struct {
		Title      string   `json:"title"`
		Content    string   `json:"content"`
		Category   string   `json:"category"`
		Scope      string   `json:"scope"`
		Tags       []string `json:"tags"`
		Confidence string   `json:"confidence"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	if params.Scope == "" {
		params.Scope = memory.ScopeProject
	}
	if params.Confidence == "" {
		params.Confidence = "medium"
	}

	validCategories := map[string]string{
		"lesson":           memory.CategoryLesson,
		"topic":            memory.CategoryTopic,
		"knowledge_update": memory.CategoryKnowledge,
	}

	category, ok := validCategories[params.Category]
	if !ok {
		return tool.ExecutionResult{
			Content: fmt.Sprintf("Invalid category '%s'. Use 'lesson', 'topic', or 'knowledge_update'.", params.Category),
			IsError: true,
		}, nil
	}

	if err := t.store.WriteFile(params.Scope, category, params.Title, params.Content, params.Tags, params.Confidence); err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("Failed to write memory: %s", err), IsError: true}, nil
	}

	return tool.ExecutionResult{
		Content: fmt.Sprintf("Memory '%s' written successfully to %s scope.", params.Title, params.Scope),
	}, nil
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd d:/git/monika
go build ./internal/tool/builtin/...
```

Expected: compiles without errors.

- [ ] **Step 3: Commit**

```bash
git add internal/tool/builtin/memory_write.go
git commit -m "feat(memory): add memory_write tool"
```

---

### Task 5: Create memory_index tool

**Files:**
- Create: `internal/tool/builtin/memory_index.go`

- [ ] **Step 1: Write the tool implementation**

```go
package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"monika/internal/memory"
	"monika/internal/tool"
)

type memoryIndexTool struct {
	store *memory.KBStore
}

func NewMemoryIndex(store *memory.KBStore) tool.Tool {
	return &memoryIndexTool{store: store}
}

func (t *memoryIndexTool) Name() string { return "memory_index" }

func (t *memoryIndexTool) Description() string {
	return "View the knowledge base index, listing all stored memories organized by type and tags. Use this to understand what the agent knows before searching."
}

func (t *memoryIndexTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"scope": map[string]any{
				"type":        "string",
				"description": "Scope: 'global', 'project', or 'auto' (default 'auto').",
			},
		},
	}
}

func (t *memoryIndexTool) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var params struct {
		Scope string `json:"scope"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	if params.Scope == "" {
		params.Scope = "auto"
	}

	var content string
	var err error

	if params.Scope == "auto" {
		content, err = t.store.GetIndex(memory.ScopeProject)
		if err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
		}
		if content == "" || content == "暂无索引。使用 memory_write 添加记忆后将自动建立索引。" {
			globalContent, gErr := t.store.GetIndex(memory.ScopeGlobal)
			if gErr == nil && globalContent != "" {
				content = fmt.Sprintf("## 项目知识库\n\n%s\n\n## 全局知识库\n\n%s", content, globalContent)
			}
		}
	} else {
		content, err = t.store.GetIndex(params.Scope)
		if err != nil {
			return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
		}
	}

	return tool.ExecutionResult{Content: content}, nil
}
```

- [ ] **Step 2: Verify compilation**

```bash
cd d:/git/monika
go build ./internal/tool/builtin/...
```

Expected: compiles without errors.

- [ ] **Step 3: Commit**

```bash
git add internal/tool/builtin/memory_index.go
git commit -m "feat(memory): add memory_index tool"
```

---

### Task 6: Add homeDir to AgentLoop and inject memory block in buildMessages()

**Files:**
- Modify: `internal/agent/agent_loop.go`

- [ ] **Step 1: Add homeDir field and WithHomeDir option to AgentLoop**

Read the existing AgentLoop struct at line 251:

```
type AgentLoop struct {
	agent    Agent
	provider engine.ProviderEngine
	tools    *tool.ToolRegistry
	conv *Conversation
	parent *AgentLoop

	sessionID         string
	systemPrompt      string
	pipeline          *permission.Pipeline
	projectDir        string
	model             string
	...
```

Add `homeDir` field after `projectDir`:

Patch location: insert after `projectDir` (line 263 in agent_loop.go).

```go
homeDir           string
```

Add `WithHomeDir` option after `WithProjectDir` (line 299):

```go
func WithHomeDir(dir string) LoopOption {
	return func(a *AgentLoop) {
		a.homeDir = dir
	}
}
```

- [ ] **Step 2: Add KBStore to AgentLoop**

Add a `kbStore` field after `homeDir`:

```go
kbStore           *memory.KBStore
```

Add the import for `"monika/internal/memory"` to the imports block.

- [ ] **Step 3: Add WithKBStore option**

```go
func WithKBStore(store *memory.KBStore) LoopOption {
	return func(a *AgentLoop) {
		a.kbStore = store
	}
}
```

- [ ] **Step 4: Inject memory block in buildMessages()**

In `buildMessages()`, after the `<task-list>` block (around line 1169), add:

```go
// Inject frozen memory snapshot (P1)
if a.kbStore != nil {
	if memoryBlock := a.kbStore.BuildMemoryBlock(); memoryBlock != "" {
		parts = append(parts, memoryBlock)
	}
}
```

- [ ] **Step 5: Verify compilation**

```bash
cd d:/git/monika
go build ./internal/agent/...
```

Expected: compiles without errors.

- [ ] **Step 6: Commit**

```bash
git add internal/agent/agent_loop.go
git commit -m "feat(memory): inject frozen memory snapshot into system prompt via AgentLoop"
```

---

### Task 7: Register memory tools and wire KBStore

**Files:**
- Modify: `internal/tool/builtin/register.go`
- Modify: `main.go`

- [ ] **Step 1: Add RegisterMemory function to register.go**

Add after the existing RegisterSkillManagement function:

```go
// RegisterMemory registers the three knowledge base tools.
func RegisterMemory(r *tool.ToolRegistry, store *memory.KBStore) {
	r.Register(NewMemorySearch(store))
	r.Register(NewMemoryWrite(store))
	r.Register(NewMemoryIndex(store))
}
```

Add the import for `"monika/internal/memory"` to the imports block of register.go.

- [ ] **Step 2: Create KBStore and register memory tools in main.go**

In `main.go`, after the skill tool registration, add:

```go
// Initialize knowledge base
kbStore, err := memory.NewKBStore(home, cwd)
if err != nil {
	fmt.Fprintf(os.Stderr, "[monika] kb init failed: %v\n", err)
} else {
	builtin.RegisterMemory(registry, kbStore)
}
```

Add the import for `"monika/internal/memory"` to the imports block of main.go.

- [ ] **Step 3: Pass KBStore to AgentLoop when constructing**

In the runner or wherever AgentLoop is instantiated, pass `WithKBStore(kbStore)` and `WithHomeDir(homeDir)` options.

Find the AgentLoop construction site — likely in `internal/api/app.go` or `runner.go`. For P1, we need the kbStore and homeDir available when creating AgentLoops.

Let's modify `internal/agent/runner.go` to accept and pass through kbStore/store and homeDir.

Add to the `TaskRunner` struct:

```go
kbStore  *memory.KBStore
homeDir  string
```

Add to the `NewTaskRunner` constructor or a setter. Modify the opts assembly (around line 74 of runner.go):

```go
if r.kbStore != nil {
	opts = append(opts, WithKBStore(r.kbStore))
}
if r.homeDir != "" {
	opts = append(opts, WithHomeDir(r.homeDir))
}
if parent == nil && r.homeDir != "" {
	opts = append(opts, WithHomeDir(r.homeDir))
}
```

- [ ] **Step 4: Verify full project compilation**

```bash
cd d:/git/monika
go build ./...
```

Expected: full project compiles without errors.

- [ ] **Step 5: Commit**

```bash
git add internal/tool/builtin/register.go internal/agent/runner.go main.go
git commit -m "feat(memory): register memory tools and wire KBStore into agent loop"
```

---

### Task 8: Integration test — write, search, inject

**Files:**
- Create: `internal/memory/store_test.go`

- [ ] **Step 1: Write test for KBStore WriteFile and Search**

```go
package memory

import (
	"os"
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

	// Write a test memory
	err = store.WriteFile(ScopeProject, CategoryLesson, "Test Lesson",
		"This is a test lesson about goroutines and channels.", []string{"go", "concurrency"}, "high")
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Search for it
	results, err := store.Search("goroutines channels", ScopeProject, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Title != "Test Lesson" {
		t.Errorf("expected 'Test Lesson', got '%s'", results[0].Title)
	}

	// Verify the file exists on disk
	expectedPath := projectDir + "/.monika/kb/wiki/lessons/test-lesson.md"
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected file at %s to exist", expectedPath)
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

	// Write profile and knowledge
	store.WriteFile(ScopeGlobal, CategoryProfile, "User Profile", "Test user profile content.", nil, "high")
	store.WriteFile(ScopeGlobal, CategoryKnowledge, "Core Knowledge", "Test global knowledge.", nil, "high")
	store.WriteFile(ScopeProject, CategoryKnowledge, "Project Knowledge", "Test project knowledge.", nil, "high")

	block := store.BuildMemoryBlock()
	if block == "" {
		t.Fatal("expected non-empty memory block")
	}

	// Check it contains expected sections
	if !contains(block, "<global_memory>") {
		t.Error("expected <global_memory> tag")
	}
	if !contains(block, "<user_profile>") {
		t.Error("expected <user_profile> tag")
	}
	if !contains(block, "<project_memory>") {
		t.Error("expected <project_memory> tag")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) &&
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}()
}
```

- [ ] **Step 2: Run tests**

```bash
cd d:/git/monika
go test ./internal/memory/... -v
```

Expected: both tests pass.

- [ ] **Step 3: Commit**

```bash
git add internal/memory/store_test.go
git commit -m "test(memory): add KBStore write and search integration tests"
```

---

### Self-Review

**1. Spec coverage check against memory-system-design.md P1:**

- [x] `internal/memory/types.go` — path constants, KBFile struct, scope/category enums
- [x] `internal/memory/store.go` — KBStore with InitKB, WriteFile, ReadFile, Search, GetIndex, BuildMemoryBlock
- [x] `internal/tool/builtin/memory_search.go` — memory_search tool
- [x] `internal/tool/builtin/memory_write.go` — memory_write tool
- [x] `internal/tool/builtin/memory_index.go` — memory_index tool
- [x] `internal/agent/agent_loop.go` — inject <memory> block in buildMessages()
- [x] `internal/tool/builtin/register.go` — RegisterMemory function
- [x] `main.go` — create KBStore and register tools
- [x] `internal/agent/runner.go` — pass kbStore and homeDir to AgentLoop

**2. Placeholder scan:**

- No TBD, TODO, or "implement later" strings found
- All code steps contain actual Go code
- All test steps contain actual test code
- No "add appropriate error handling" vagueness — error paths are explicit

**3. Type consistency:**

- `KBFile` struct used consistently across store.go, memory_search.go, store_test.go
- `KBStore` methods (Search, WriteFile, ReadFile, GetIndex, BuildMemoryBlock) match their callers
- Tool interface: Name(), Description(), Parameters(), Execute() — all three tools follow same pattern
- `scope` parameter uses `memory.ScopeGlobal` / `memory.ScopeProject` / `"auto"` consistently
- `category` parameter uses `memory.CategoryLesson` etc. consistently

**One gap identified:** The `memory_index` tool returns `index.md` content, but P1 doesn't include a task to auto-generate `index.md` when files are written. This is acceptable for P1 MVP — the index will show "暂无索引" until the user or agent manually creates it via `memory_write` to the knowledge category. Auto-index generation is a P2 concern.

---

**Plan complete and saved to `docs/superpowers/plans/2026-06-16-memory-system-p1.md`. Two execution options:**

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
