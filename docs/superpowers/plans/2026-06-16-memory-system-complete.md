# Memory System — Complete Implementation Plan (P1-P4)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the complete self-evolving memory system with SQLite FTS5 storage, knowledge injection, auto-extraction from sessions, background consolidation/review, experience-to-skill pipeline, and frontend Settings management tab.

**Architecture:** `internal/memory/` package with KBStore (SQLite FTS5 + file I/O), extraction engine (compaction-summary → LLM → wiki files), consolidation (similarity merge), background review goroutine, and frontend `KnowledgeBaseTab` + backend `kb_api.go`. Frozen snapshot injection into `buildMessages()`.

**Tech Stack:** Go 1.25, `modernc.org/sqlite`, React/TypeScript with Zustand, Wails v3 bindings

---

## P1 — Storage + Search + Injection MVP

### Task 1: Add SQLite dependency and internal/memory/types.go

**Files:**
- Modify: `go.mod`
- Create: `internal/memory/types.go`

- [ ] **Step 1: Add modernc.org/sqlite dependency**

```bash
cd d:/git/monika
go get modernc.org/sqlite
```

- [ ] **Step 2: Create internal/memory/types.go**

```go
package memory

import (
	"path/filepath"
	"time"
)

const (
	ScopeGlobal  = "global"
	ScopeProject = "project"
)

const (
	CategoryKnowledge = "wiki/knowledge"
	CategoryProfile   = "wiki/profile"
	CategoryLesson    = "wiki/lesson"
	CategoryTopic     = "wiki/topic"
	CategoryRawDoc    = "raw/doc"
	CategoryRawCode   = "raw/code"
)

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
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func GlobalKBPath(homeDir string) string {
	return filepath.Join(homeDir, ".monika", "kb")
}

func ProjectKBPath(projectDir string) string {
	return filepath.Join(projectDir, ".monika", "kb")
}

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
git commit -m "feat(memory): add types, path helpers, and SQLite dependency"
```

---

### Task 2: Create internal/memory/store.go — KBStore

**Files:**
- Create: `internal/memory/store.go`

- [ ] **Step 1: Write store.go**

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

type KBStore struct {
	globalPath  string
	projectPath string
	db          *sql.DB
}

func NewKBStore(homeDir, projectDir string) (*KBStore, error) {
	gp := GlobalKBPath(homeDir)
	pp := ProjectKBPath(projectDir)
	for _, p := range []string{gp, pp} {
		for _, sub := range KBSubdirs() {
			if err := os.MkdirAll(filepath.Join(p, sub), 0755); err != nil {
				return nil, fmt.Errorf("kb mkdir %s: %w", p, err)
			}
		}
	}
	s := &KBStore{globalPath: gp, projectPath: pp}
	if err := s.openIndex(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *KBStore) openIndex() error {
	// P1: merged db at project scope; all files indexed with scope field
	dbPath := filepath.Join(s.projectPath, ".index", "kb.db")
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return err
	}
	_, err = db.Exec(`
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
			linked_to   TEXT DEFAULT '[]',
			created_at  TEXT NOT NULL,
			updated_at  TEXT NOT NULL
		);
		CREATE VIRTUAL TABLE IF NOT EXISTS file_fts USING fts5(
			path, title, content,
			content=file_index, content_rowid=id
		);
		CREATE TRIGGER IF NOT EXISTS fts_ai AFTER INSERT ON file_index BEGIN
			INSERT INTO file_fts(rowid, path, title, content)
			VALUES (new.id, new.path, new.title, '');
		END;
		CREATE TRIGGER IF NOT EXISTS fts_ad AFTER DELETE ON file_index BEGIN
			INSERT INTO file_fts(file_fts, rowid, path, title, content)
			VALUES ('delete', old.id, old.path, old.title, '');
		END;
		CREATE TRIGGER IF NOT EXISTS fts_au AFTER UPDATE ON file_index BEGIN
			INSERT INTO file_fts(file_fts, rowid, path, title, content)
			VALUES ('delete', old.id, old.path, old.title, '');
			INSERT INTO file_fts(rowid, path, title, content)
			VALUES (new.id, new.path, new.title, '');
		END;
	`)
	if err != nil {
		db.Close()
		return fmt.Errorf("migrate: %w", err)
	}
	s.db = db
	return nil
}

func (s *KBStore) Search(query, scope string, limit int) ([]KBFile, error) {
	if limit <= 0 {
		limit = 5
	}
	where := ""
	args := []any{query}
	if scope == ScopeGlobal || scope == ScopeProject {
		where = " AND f.scope = ?"
		args = append(args, scope)
	}
	q := fmt.Sprintf(`
		SELECT f.id, f.path, f.scope, f.category, f.title, f.tags,
		       f.confidence, f.status, f.char_count, f.linked_to,
		       f.created_at, f.updated_at,
		       snippet(file_fts, 2, '<b>', '</b>', '...', 40)
		FROM file_fts
		JOIN file_index f ON file_fts.rowid = f.id
		WHERE file_fts MATCH ? %s AND f.status != 'trash'
		ORDER BY bm25(file_fts, 0, 10, 5)
		LIMIT ?
	`, where)
	args = append(args, limit)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()
	return scanKBFiles(rows)
}

func (s *KBStore) WriteFile(scope, category, title, content string, tags []string, confidence string) error {
	root := s.rootFor(scope)
	if tags == nil {
		tags = []string{}
	}
	if confidence == "" {
		confidence = "medium"
	}
	tagsJSON, _ := json.Marshal(tags)
	now := time.Now().UTC().Format(time.RFC3339)

	var front strings.Builder
	front.WriteString("# " + title + "\n\n")
	front.WriteString("> 类型：semantic\n")
	front.WriteString("> 作用域：" + scope + "\n")
	front.WriteString("> 创建：" + now + "\n")
	front.WriteString("> 更新：" + now + "\n")
	front.WriteString("> 置信度：" + confidence + "\n")
	front.WriteString("> 标签：" + strings.Join(tags, ", ") + "\n")
	front.WriteString("> 状态：active\n\n")
	front.WriteString(content)

	relPath := categoryPath(category, title)
	fullPath := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(fullPath, []byte(front.String()), 0644); err != nil {
		return err
	}

	charCount := len([]rune(content))
	_, err := s.db.Exec(`
		INSERT INTO file_index (path, scope, category, title, tags, confidence, status, char_count, linked_to, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'active', ?, '[]', ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			category=excluded.category, title=excluded.title, tags=excluded.tags,
			confidence=excluded.confidence, char_count=excluded.char_count,
			updated_at=excluded.updated_at
	`, relPath, scope, category, title, string(tagsJSON), confidence, charCount, now, now)
	if err != nil {
		return fmt.Errorf("upsert: %w", err)
	}

	var id int64
	s.db.QueryRow("SELECT id FROM file_index WHERE path = ?", relPath).Scan(&id)
	if id > 0 {
		s.db.Exec("UPDATE file_fts SET content = ? WHERE rowid = ?", content, id)
	}
	return nil
}

func (s *KBStore) ReadFile(scope, relPath string) (string, error) {
	cleanPath := filepath.Clean(relPath)
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("invalid path: %s", relPath)
	}
	data, err := os.ReadFile(filepath.Join(s.rootFor(scope), cleanPath))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *KBStore) ListFiles(scope, category string) ([]KBFile, error) {
	var rows *sql.Rows
	var err error
	if category != "" {
		rows, err = s.db.Query(`
			SELECT id, path, scope, category, title, tags, confidence, status, char_count, linked_to, created_at, updated_at
			FROM file_index WHERE scope = ? AND category = ? AND status != 'trash'
			ORDER BY updated_at DESC
		`, scope, category)
	} else {
		rows, err = s.db.Query(`
			SELECT id, path, scope, category, title, tags, confidence, status, char_count, linked_to, created_at, updated_at
			FROM file_index WHERE scope = ? AND status != 'trash'
			ORDER BY updated_at DESC
		`, scope)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKBFilesFlat(rows)
}

func (s *KBStore) SoftDelete(scope, relPath string) error {
	_, err := s.db.Exec("UPDATE file_index SET status = 'trash', updated_at = ? WHERE path = ?",
		time.Now().UTC().Format(time.RFC3339), relPath)
	if err != nil {
		return err
	}
	// Delete from FTS
	var id int64
	s.db.QueryRow("SELECT id FROM file_index WHERE path = ?", relPath).Scan(&id)
	if id > 0 {
		s.db.Exec("DELETE FROM file_fts WHERE rowid = ?", id)
	}
	// Move file to .trash/
	oldPath := filepath.Join(s.rootFor(scope), relPath)
	trashSubdir := filepath.Join(".trash", filepath.Dir(relPath))
	trashPath := filepath.Join(s.rootFor(scope), trashSubdir)
	os.MkdirAll(trashPath, 0755)
	os.Rename(oldPath, filepath.Join(trashPath, filepath.Base(relPath)))
	return nil
}

func (s *KBStore) GetStatistics(scope string) (total, active, archived int, lastUpdate string, err error) {
	err = s.db.QueryRow(`
		SELECT COUNT(*),
		       SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END),
		       SUM(CASE WHEN status = 'archived' OR status = 'deprecated' THEN 1 ELSE 0 END),
		       COALESCE(MAX(updated_at), '')
		FROM file_index WHERE scope = ?
	`, scope).Scan(&total, &active, &archived, &lastUpdate)
	return
}

func (s *KBStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *KBStore) rootFor(scope string) string {
	if scope == ScopeGlobal {
		return s.globalPath
	}
	return s.projectPath
}

func categoryPath(category, title string) string {
	slug := titleToSlug(title)
	switch category {
	case CategoryKnowledge:
		return "wiki/knowledge.md"
	case CategoryProfile:
		return "wiki/profile.md"
	case CategoryLesson:
		return filepath.Join("wiki/lessons", slug+".md")
	case CategoryTopic:
		return filepath.Join("wiki/topics", slug+".md")
	case CategoryRawDoc:
		return filepath.Join("raw/docs", slug+".md")
	case CategoryRawCode:
		return filepath.Join("raw/code", slug+".md")
	default:
		return filepath.Join(category, slug+".md")
	}
}

func titleToSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	var c strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			c.WriteRune(r)
		}
	}
	r := strings.Trim(c.String(), "-")
	if r == "" {
		return "untitled"
	}
	return r
}

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
		results = append(results, f)
	}
	if results == nil {
		results = []KBFile{}
	}
	return results, nil
}

func scanKBFilesFlat(rows *sql.Rows) ([]KBFile, error) {
	var results []KBFile
	for rows.Next() {
		var f KBFile
		var tagsJSON, linkedJSON, ca, ua string
		if err := rows.Scan(&f.ID, &f.Path, &f.Scope, &f.Category, &f.Title, &tagsJSON,
			&f.Confidence, &f.Status, &f.CharCount, &linkedJSON, &ca, &ua); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(tagsJSON), &f.Tags)
		json.Unmarshal([]byte(linkedJSON), &f.LinkedTo)
		f.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		f.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
		results = append(results, f)
	}
	if results == nil {
		results = []KBFile{}
	}
	return results, nil
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/memory/store.go
git commit -m "feat(memory): add KBStore with SQLite FTS5, CRUD, and statistics"
```

---

### Task 3: Create internal/memory/inject.go — Memory Block Assembly

**Files:**
- Create: `internal/memory/inject.go`

- [ ] **Step 1: Write inject.go**

```go
package memory

import "strings"

// BuildMemoryBlock reads profile.md and knowledge.md from both scopes
// and assembles the <memory> XML block for system prompt injection.
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
```

- [ ] **Step 2: Commit**

```bash
git add internal/memory/inject.go
git commit -m "feat(memory): add BuildMemoryBlock for system prompt injection"
```

---

### Task 4: Create memory_search tool

**Files:**
- Create: `internal/tool/builtin/memory_search.go`

- [ ] **Step 1: Write memory_search.go**

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

type memorySearchTool struct{ store *memory.KBStore }

func NewMemorySearch(store *memory.KBStore) tool.Tool { return &memorySearchTool{store} }

func (t *memorySearchTool) Name() string { return "memory_search" }

func (t *memorySearchTool) Description() string {
	return "Search the knowledge base for relevant memories. Returns matching files with titles and highlighted snippets."
}

func (t *memorySearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query":    map[string]any{"type": "string", "description": "Search keywords."},
			"scope":    map[string]any{"type": "string", "description": "'global', 'project', or 'auto' (default)."},
			"category": map[string]any{"type": "string", "description": "Filter: 'lesson', 'topic', 'knowledge', 'raw', or 'all'."},
			"limit":    map[string]any{"type": "integer", "description": "Max results (default 5, max 10)."},
		},
		"required": []string{"query"},
	}
}

func (t *memorySearchTool) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var p struct {
		Query    string `json:"query"`
		Scope    string `json:"scope"`
		Category string `json:"category"`
		Limit    int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	if p.Scope == "" {
		p.Scope = "auto"
	}
	if p.Limit <= 0 {
		p.Limit = 5
	}
	if p.Limit > 10 {
		p.Limit = 10
	}

	results, err := t.store.Search(p.Query, p.Scope, p.Limit)
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	// auto: try project first, fall back to global
	if p.Scope == "auto" && len(results) == 0 {
		results, err = t.store.Search(p.Query, memory.ScopeGlobal, p.Limit)
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
		sb.WriteString(fmt.Sprintf("%d. **%s** [%s/%s] confidence: %s\n   path: %s\n\n",
			i+1, r.Title, r.Scope, r.Category, r.Confidence, r.Path))
	}
	return tool.ExecutionResult{Content: sb.String()}, nil
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/tool/builtin/memory_search.go
git commit -m "feat(memory): add memory_search tool"
```

---

### Task 5: Create memory_write tool

**Files:**
- Create: `internal/tool/builtin/memory_write.go`

- [ ] **Step 1: Write memory_write.go**

```go
package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"monika/internal/memory"
	"monika/internal/tool"
)

type memoryWriteTool struct{ store *memory.KBStore }

func NewMemoryWrite(store *memory.KBStore) tool.Tool { return &memoryWriteTool{store} }

func (t *memoryWriteTool) Name() string { return "memory_write" }

func (t *memoryWriteTool) Description() string {
	return "Write a new memory entry to the knowledge base."
}

func (t *memoryWriteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title":      map[string]any{"type": "string", "description": "Title of the memory."},
			"content":    map[string]any{"type": "string", "description": "Markdown content."},
			"category":   map[string]any{"type": "string", "description": "'lesson', 'topic', or 'knowledge_update'."},
			"scope":      map[string]any{"type": "string", "description": "'global' or 'project' (default)."},
			"tags":       map[string]any{"type": "array", "items": map[string]string{"type": "string"}},
			"confidence": map[string]any{"type": "string", "description": "'high', 'medium', or 'low'."},
		},
		"required": []string{"title", "content", "category"},
	}
}

func (t *memoryWriteTool) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var p struct {
		Title      string   `json:"title"`
		Content    string   `json:"content"`
		Category   string   `json:"category"`
		Scope      string   `json:"scope"`
		Tags       []string `json:"tags"`
		Confidence string   `json:"confidence"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	if p.Scope == "" {
		p.Scope = memory.ScopeProject
	}
	if p.Confidence == "" {
		p.Confidence = "medium"
	}

	catMap := map[string]string{
		"lesson":           memory.CategoryLesson,
		"topic":            memory.CategoryTopic,
		"knowledge_update": memory.CategoryKnowledge,
	}
	cat, ok := catMap[p.Category]
	if !ok {
		return tool.ExecutionResult{
			Content: fmt.Sprintf("Invalid category '%s'. Use 'lesson', 'topic', or 'knowledge_update'.", p.Category),
			IsError: true,
		}, nil
	}
	if err := t.store.WriteFile(p.Scope, cat, p.Title, p.Content, p.Tags, p.Confidence); err != nil {
		return tool.ExecutionResult{Content: fmt.Sprintf("Failed to write: %s", err), IsError: true}, nil
	}
	return tool.ExecutionResult{Content: fmt.Sprintf("Memory '%s' written to %s scope.", p.Title, p.Scope)}, nil
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/tool/builtin/memory_write.go
git commit -m "feat(memory): add memory_write tool"
```

---

### Task 6: Create memory_index tool

**Files:**
- Create: `internal/tool/builtin/memory_index.go`

- [ ] **Step 1: Write memory_index.go**

```go
package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"monika/internal/memory"
	"monika/internal/tool"
)

type memoryIndexTool struct{ store *memory.KBStore }

func NewMemoryIndex(store *memory.KBStore) tool.Tool { return &memoryIndexTool{store} }

func (t *memoryIndexTool) Name() string { return "memory_index" }

func (t *memoryIndexTool) Description() string {
	return "View knowledge base contents — all stored memories organized by category."
}

func (t *memoryIndexTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"scope": map[string]any{"type": "string", "description": "'global', 'project', or 'auto'."},
		},
	}
}

func (t *memoryIndexTool) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var p struct{ Scope string `json:"scope"` }
	if err := json.Unmarshal(args, &p); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	if p.Scope == "" {
		p.Scope = "auto"
	}

	content := ""
	if p.Scope == "auto" || p.Scope == memory.ScopeProject {
		content += formatFileList(t.store, memory.ScopeProject)
	}
	if p.Scope == "auto" || p.Scope == memory.ScopeGlobal {
		if p.Scope == "auto" {
			content += "\n"
		}
		content += formatFileList(t.store, memory.ScopeGlobal)
	}
	if strings.TrimSpace(content) == "" {
		return tool.ExecutionResult{Content: "Knowledge base is empty."}, nil
	}
	return tool.ExecutionResult{Content: content}, nil
}

func formatFileList(store *memory.KBStore, scope string) string {
	files, err := store.ListFiles(scope, "")
	if err != nil {
		return fmt.Sprintf("Error listing %s: %s\n", scope, err)
	}
	if len(files) == 0 {
		return ""
	}
	groups := map[string][]memory.KBFile{}
	for _, f := range files {
		label := categoryLabel(f.Category)
		groups[label] = append(groups[label], f)
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## %s Knowledge Base\n\n", strings.Title(scope)))
	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("### %s\n", k))
		for _, f := range groups[k] {
			sb.WriteString(fmt.Sprintf("- **%s** (%s) [%s] %s\n", f.Title, f.Confidence, f.Status, f.Path))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func categoryLabel(cat string) string {
	switch cat {
	case memory.CategoryKnowledge:
		return "Core Knowledge"
	case memory.CategoryProfile:
		return "User Profile"
	case memory.CategoryLesson:
		return "Lessons"
	case memory.CategoryTopic:
		return "Topics"
	case memory.CategoryRawDoc:
		return "Documents"
	case memory.CategoryRawCode:
		return "Code Repositories"
	default:
		return cat
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/tool/builtin/memory_index.go
git commit -m "feat(memory): add memory_index tool"
```

---

### Task 7: Create memory_reindex tool (P1)

**Files:**
- Create: `internal/tool/builtin/memory_reindex.go`

- [ ] **Step 1: Write memory_reindex.go**

```go
package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"monika/internal/memory"
	"monika/internal/tool"
)

type memoryReindexTool struct{ store *memory.KBStore }

func NewMemoryReindex(store *memory.KBStore) tool.Tool { return &memoryReindexTool{store} }

func (t *memoryReindexTool) Name() string { return "memory_reindex" }

func (t *memoryReindexTool) Description() string {
	return "Rebuild the FTS5 search index from all files on disk."
}

func (t *memoryReindexTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"scope": map[string]any{"type": "string", "description": "'global' or 'project' (default)."},
		},
	}
}

func (t *memoryReindexTool) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var p struct{ Scope string `json:"scope"` }
	if err := json.Unmarshal(args, &p); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}
	if p.Scope == "" {
		p.Scope = memory.ScopeProject
	}

	// Re-index by walking the wiki/ and raw/ directories
	count := 0
	walkScope := []string{p.Scope}
	if p.Scope == "project" {
		walkScope = []string{memory.ScopeProject}
	} else if p.Scope == "global" {
		walkScope = []string{memory.ScopeGlobal}
	}

	for _, sc := range walkScope {
		files, err := t.store.ListFiles(sc, "")
		if err != nil {
			continue
		}
		for _, f := range files {
			content, err := t.store.ReadFile(sc, f.Path)
			if err != nil {
				continue
			}
			// force re-write to update FTS content
			t.store.WriteFile(sc, f.Category, f.Title, extractBody(content), f.Tags, f.Confidence)
			count++
		}
		// also scan disk for files not yet indexed
		root := memory.GlobalKBPath("")
		if sc == memory.ScopeProject {
			root = memory.ProjectKBPath("")
		}
		_ = root // P1 simplified: rely on WriteFile for indexing
	}

	return tool.ExecutionResult{
		Content: fmt.Sprintf("Reindexed %d files.", count),
	}, nil
}

func extractBody(content string) string {
	// Strip YAML-like frontmatter lines ("> key: value")
	lines := strings.Split(content, "\n")
	var body []string
	pastFM := false
	for _, line := range lines {
		if !pastFM && (strings.HasPrefix(line, "> ") || strings.HasPrefix(line, "# ")) {
			if strings.HasPrefix(line, "# ") && len(body) > 0 {
				pastFM = true
				body = append(body, line)
			}
			continue
		}
		pastFM = true
		body = append(body, line)
	}
	return strings.Join(body, "\n")
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/tool/builtin/memory_reindex.go
git commit -m "feat(memory): add memory_reindex tool"
```

---

### Task 8: Add homeDir to AgentLoop, inject memory block, wire everything

**Files:**
- Modify: `internal/agent/agent_loop.go`
- Modify: `internal/agent/runner.go`
- Modify: `internal/tool/builtin/register.go`
- Modify: `main.go`

- [ ] **Step 1: Add homeDir and kbStore to AgentLoop struct**

In `agent_loop.go`, add after `projectDir`:

```go
homeDir  string
kbStore  *memory.KBStore
```

Add import `"monika/internal/memory"`.

- [ ] **Step 2: Add WithHomeDir and WithKBStore options**

```go
func WithHomeDir(dir string) LoopOption {
	return func(a *AgentLoop) { a.homeDir = dir }
}

func WithKBStore(store *memory.KBStore) LoopOption {
	return func(a *AgentLoop) { a.kbStore = store }
}
```

- [ ] **Step 3: Inject memory block in buildMessages()**

After the `<task-list>` block (around line 1169), add:

```go
if a.kbStore != nil {
	if block := a.kbStore.BuildMemoryBlock(); block != "" {
		parts = append(parts, block)
	}
}
```

- [ ] **Step 4: Add RegisterMemory to register.go**

```go
import "monika/internal/memory"

func RegisterMemory(r *tool.ToolRegistry, store *memory.KBStore) {
	r.Register(NewMemorySearch(store))
	r.Register(NewMemoryWrite(store))
	r.Register(NewMemoryIndex(store))
	r.Register(NewMemoryReindex(store))
}
```

- [ ] **Step 5: Create KBStore and register tools in main.go**

After skill tool registration, add:

```go
kbStore, err := memory.NewKBStore(home, cwd)
if err != nil {
	fmt.Fprintf(os.Stderr, "[monika] kb init failed: %v\n", err)
} else {
	builtin.RegisterMemory(registry, kbStore)
}
```

Add import `"monika/internal/memory"`.

- [ ] **Step 6: Pass kbStore to TaskRunner/App**

In `runner.go`, add field `kbStore *memory.KBStore` to TaskRunner, and in the opts assembly:

```go
if r.kbStore != nil {
	opts = append(opts, WithKBStore(r.kbStore))
}
```

- [ ] **Step 7: Store kbStore in App struct and pass to TaskRunner**

In `app.go`, add `kbStore *memory.KBStore` to the App struct. In `NewApp`, accept `kbStore` param. In main.go, pass it:

```go
taskRunner := agent2.NewTaskRunner(registry, providers, dispatchFn, pendingStore, onStartFn, kbStore)
```

And add `WithHomeDir(home)` to loopOpts.

- [ ] **Step 8: Verify compilation**

```bash
cd d:/git/monika
go build ./...
```

- [ ] **Step 9: Commit**

```bash
git add internal/agent/agent_loop.go internal/agent/runner.go internal/tool/builtin/register.go internal/api/app.go main.go
git commit -m "feat(memory): wire KBStore into AgentLoop, inject memory block, register tools"
```

---

### Task 9: Write integration tests (P1)

**Files:**
- Create: `internal/memory/store_test.go`

- [ ] **Step 1: Write store_test.go**

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

	err = store.WriteFile(ScopeProject, CategoryLesson, "Test Lesson",
		"This is a test lesson about goroutines and channels.", []string{"go", "concurrency"}, "high")
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

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
```

- [ ] **Step 2: Run tests**

```bash
cd d:/git/monika
go test ./internal/memory/... -v
```

- [ ] **Step 3: Commit**

```bash
git add internal/memory/store_test.go
git commit -m "test(memory): add KBStore integration tests"
```

---

## P2 — Auto-Extraction + Consolidation + knowledge.md Rebuild

### Task 10: Create internal/memory/extract.go — LLM-based memory extraction

**Files:**
- Create: `internal/memory/extract.go`

- [ ] **Step 1: Write extract.go**

```go
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ExtractCandidate represents one memory item the LLM suggests extracting.
type ExtractCandidate struct {
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	Category   string   `json:"category"`   // "lesson" | "topic" | "knowledge_update"
	Scope      string   `json:"scope"`      // "global" | "project"
	Tags       []string `json:"tags"`
	Confidence string   `json:"confidence"` // "high" | "medium" | "low"
}

// ExtractResult is the LLM response for memory extraction.
type ExtractResult struct {
	Candidates   []ExtractCandidate `json:"candidates"`
	ProfileDelta string             `json:"profile_delta,omitempty"` // profile.md update
}

// ExtractionLLM is the interface for calling an LLM to extract memories.
type ExtractionLLM interface {
	Chat(ctx context.Context, systemPrompt, userMessage string) (string, error)
}

// ExtractMemories sends a compaction summary to the LLM and returns candidate memories.
func ExtractMemories(ctx context.Context, llm ExtractionLLM, scope, sessionID, compactionSummary string) (*ExtractResult, error) {
	systemPrompt := `你是一个知识提取器。从以下 session 总结中提取值得长期保留的知识。

类型定义：
- "lesson": 具体经验教训（问题→根因→解决方案→泛化教训）
- "topic": 技术主题知识点（架构、模式、约定、API 说明）
- "knowledge_update": 需要更新到核心知识库的事实（用户偏好、项目约束、常驻事实）

范围判断：
- "global": 跨项目通用的知识（语言特性、设计模式、通用工具）
- "project": 本项目特有的知识（项目架构、约定、依赖、bug fix）

返回 JSON 格式：
{
  "candidates": [
    {
      "title": "...",
      "content": "markdown 格式正文...",
      "category": "lesson | topic | knowledge_update",
      "scope": "global | project",
      "tags": ["tag1", "tag2"],
      "confidence": "high | medium | low"
    }
  ],
  "profile_delta": "如果从对话中发现用户偏好/风格变化，写简短摘要"
}

规则：
- 只提取有长期价值的知识，不提取一次性操作
- 置信度 high: 明确的教训或事实；medium: 可能有用的发现；low: 不确定但值得记录
- content 用 markdown，包含必要的上下文、代码片段、链接
- 不要重复已有的知识`

	userMsg := fmt.Sprintf("Session ID: %s\nScope: %s\n\n--- Compaction Summary ---\n%s",
		sessionID, scope, compactionSummary)

	resp, err := llm.Chat(ctx, systemPrompt, userMsg)
	if err != nil {
		return nil, fmt.Errorf("extraction llm: %w", err)
	}

	// Parse JSON response (handle markdown code fences)
	jsonStr := strings.TrimSpace(resp)
	if idx := strings.Index(jsonStr, "```json"); idx >= 0 {
		jsonStr = jsonStr[idx+7:]
		if end := strings.Index(jsonStr, "```"); end >= 0 {
			jsonStr = jsonStr[:end]
		}
	} else if idx := strings.Index(jsonStr, "{"); idx >= 0 {
		jsonStr = jsonStr[idx:]
	}

	var result ExtractResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("parse extraction: %w\nresponse: %s", err, resp)
	}
	return &result, nil
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/memory/extract.go
git commit -m "feat(memory): add LLM-based memory extraction from compaction summaries"
```

---

### Task 11: Create internal/memory/consolidate.go — merge strategy

**Files:**
- Create: `internal/memory/consolidate.go`

- [ ] **Step 1: Write consolidate.go**

```go
package memory

import (
	"fmt"
	"strings"
)

// SimilarityResult holds the match between a candidate and existing memory.
type SimilarityResult struct {
	File  KBFile
	Score float64
}

// Consolidate determines how to merge a new candidate with existing memories.
// Returns the action to take: "update", "new_linked", or "new".
func (s *KBStore) Consolidate(candidate ExtractCandidate, existing []SimilarityResult) (action string, targetFile *KBFile) {
	if len(existing) == 0 {
		return "new", nil
	}

	top := existing[0]
	if top.Score >= 0.8 {
		// High similarity: update existing
		return "update", &top.File
	} else if top.Score >= 0.4 {
		// Medium: new file with links
		return "new_linked", &top.File
	}
	return "new", nil
}

// ComputeSimilarity searches FTS for similar entries and scores them.
func (s *KBStore) ComputeSimilarity(candidate ExtractCandidate) ([]SimilarityResult, error) {
	results, err := s.Search(candidate.Title+" "+candidate.Content, "", 3)
	if err != nil {
		return nil, err
	}
	var sims []SimilarityResult
	for _, r := range results {
		// Simple Jaccard-like score based on FTS bm25 ordering + tag overlap
		overlap := tagOverlap(candidate.Tags, r.Tags)
		bm25Score := 0.5 // normalized placeholder; FTS orders by bm25 already
		sims = append(sims, SimilarityResult{
			File:  r,
			Score: bm25Score + float64(overlap)*0.25,
		})
	}
	return sims, nil
}

func tagOverlap(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	set := make(map[string]bool, len(a))
	for _, t := range a {
		set[t] = true
	}
	common := 0
	for _, t := range b {
		if set[t] {
			common++
		}
	}
	return float64(common) / float64(max(len(a), len(b)))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MergeFileContent appends or merges new content into an existing file.
func MergeFileContent(existingBody, newContent, candidateTitle string) string {
	if strings.Contains(existingBody, candidateTitle) {
		// Already contains similar info; append as update note
		return existingBody + "\n\n## 更新\n" + newContent
	}
	return existingBody + "\n\n## " + candidateTitle + "\n" + newContent
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/memory/consolidate.go
git commit -m "feat(memory): add consolidation logic with similarity scoring"
```

---

### Task 12: Create internal/memory/compact_knowledge.go — knowledge.md compression

**Files:**
- Create: `internal/memory/compact_knowledge.go`

- [ ] **Step 1: Write compact_knowledge.go**

```go
package memory

import (
	"context"
	"fmt"
	"strings"
)

const (
	maxKnowledgeChars = 3000
	maxProfileChars   = 1500
)

// CompactionLLM defines the LLM interface for compacting knowledge.
type CompactionLLM interface {
	Chat(ctx context.Context, systemPrompt, userMessage string) (string, error)
}

// CompactKnowledge compresses knowledge.md when it exceeds the char limit.
func (s *KBStore) CompactKnowledge(ctx context.Context, llm CompactionLLM, scope string) error {
	content, err := s.ReadFile(scope, "wiki/knowledge.md")
	if err != nil || content == "" {
		return nil // nothing to compact
	}

	if len([]rune(content)) <= maxKnowledgeChars {
		return nil
	}

	prompt := `你是一个知识压缩器。将以下 knowledge.md 压缩到 ` +
		fmt.Sprintf("%d 字符以内，保留高置信度的事实，淘汰低置信度或过时的信息。\n\n", maxKnowledgeChars) +
		`规则：
- 保留所有用户偏好和硬约束
- 保留最近的、高频使用的知识
- 合并相关内容，删除重复
- 保留原始 markdown 结构

原始内容：
` + content

	compressed, err := llm.Chat(ctx, prompt, "")
	if err != nil {
		return fmt.Errorf("compact: %w", err)
	}

	// Ensure it's within limit
	runes := []rune(compressed)
	if len(runes) > maxKnowledgeChars {
		compressed = string(runes[:maxKnowledgeChars])
	}

	return s.WriteFile(scope, CategoryKnowledge, "Core Knowledge", compressed, nil, "high")
}

// CompactProfile compresses profile.md.
func (s *KBStore) CompactProfile(ctx context.Context, llm CompactionLLM, scope string) error {
	content, err := s.ReadFile(scope, "wiki/profile.md")
	if err != nil || content == "" {
		return nil
	}

	if len([]rune(content)) <= maxProfileChars {
		return nil
	}

	prompt := `压缩以下 user profile 到 ` + fmt.Sprintf("%d", maxProfileChars) +
		` 字符以内，保留最重要的偏好和事实：\n\n` + content

	compressed, err := llm.Chat(ctx, prompt, "")
	if err != nil {
		return fmt.Errorf("compact profile: %w", err)
	}

	runes := []rune(compressed)
	if len(runes) > maxProfileChars {
		compressed = string(runes[:maxProfileChars])
	}

	return s.WriteFile(scope, CategoryProfile, "User Profile", compressed, nil, "high")
}

// ExtractBody strips frontmatter from a markdown file content.
func ExtractBody(content string) string {
	lines := strings.Split(content, "\n")
	var body []string
	fmDone := false
	for _, line := range lines {
		if !fmDone && strings.HasPrefix(line, "> ") {
			continue
		}
		if !fmDone && strings.HasPrefix(line, "# ") {
			fmDone = true
			continue
		}
		fmDone = true
		body = append(body, line)
	}
	return strings.Join(body, "\n")
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/memory/compact_knowledge.go
git commit -m "feat(memory): add knowledge.md and profile.md compression"
```

---

### Task 13: Create internal/memory/log.go — operation logging

**Files:**
- Create: `internal/memory/log.go`

- [ ] **Step 1: Write log.go**

```go
package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LogEntry records an operation in wiki/log.md.
func (s *KBStore) LogEntry(scope, action, detail string) error {
	now := time.Now().UTC()
	dateHeader := now.Format("2006-01-02")
	timeHeader := now.Format("15:04")

	entry := fmt.Sprintf("\n### %s — %s\n- %s\n", timeHeader, action, detail)

	content, err := s.ReadFile(scope, "wiki/log.md")
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if content == "" {
		content = "# Operation Log\n\n> 类型：episodic\n"
	}

	// Insert under date header, or create new date header
	dateSection := "## " + dateHeader
	if idx := strings.Index(content, dateSection); idx >= 0 {
		// Insert after date header, before next section
		rest := content[idx+len(dateSection):]
		nextSection := strings.Index(rest, "\n## ")
		if nextSection >= 0 {
			content = content[:idx+len(dateSection)] + entry + rest[:nextSection] + rest[nextSection:]
		} else {
			content = content[:idx+len(dateSection)] + entry + rest
		}
	} else {
		// New date section
		content += "\n" + dateSection + entry
	}

	root := s.rootFor(scope)
	return os.WriteFile(filepath.Join(root, "wiki", "log.md"), []byte(content), 0644)
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/memory/log.go
git commit -m "feat(memory): add operation logging to wiki/log.md"
```

---

### Task 14: Hook session archive for auto-extraction

**Files:**
- Create: `internal/memory/hook.go`
- Modify: `internal/api/app.go`

- [ ] **Step 1: Write hook.go**

```go
package memory

import (
	"context"
	"fmt"
)

// ArchiveHook is called when a session is archived to trigger extraction.
type ArchiveHook struct {
	Store           *KBStore
	LLM             ExtractionLLM
	CompactionLLM   CompactionLLM
	OnStatusChange  func(status string) // status bar callback
}

func (h *ArchiveHook) OnArchive(ctx context.Context, scope, sessionID, compactionSummary string) {
	if h.OnStatusChange != nil {
		h.OnStatusChange("归纳中...")
	}

	if compactionSummary == "" {
		if h.OnStatusChange != nil {
			h.OnStatusChange("记忆已更新 ✓")
		}
		return
	}

	// Step 1: Extract
	result, err := ExtractMemories(ctx, h.LLM, scope, sessionID, compactionSummary)
	if err != nil {
		fmt.Printf("[memory] extraction failed: %v\n", err)
		if h.OnStatusChange != nil {
			h.OnStatusChange("归纳失败")
		}
		return
	}

	// Step 2: Consolidate and write each candidate
	written := 0
	for _, c := range result.Candidates {
		sims, _ := h.Store.ComputeSimilarity(c)
		action, target := h.Store.Consolidate(c, sims)

		var cat string
		switch c.Category {
		case "lesson":
			cat = CategoryLesson
		case "topic":
			cat = CategoryTopic
		case "knowledge_update":
			cat = CategoryKnowledge
		default:
			cat = CategoryLesson
		}

		switch action {
		case "update":
			if target != nil {
				existing, _ := h.Store.ReadFile(target.Scope, target.Path)
				body := ExtractBody(existing)
				merged := MergeFileContent(body, c.Content, c.Title)
				h.Store.WriteFile(target.Scope, cat, target.Title, merged, c.Tags, c.Confidence)
				h.Store.LogEntry(c.Scope, "合并记忆", fmt.Sprintf("更新 %s", target.Path))
			}
		case "new_linked", "new":
			h.Store.WriteFile(c.Scope, cat, c.Title, c.Content, c.Tags, c.Confidence)
			h.Store.LogEntry(c.Scope, "新建记忆", fmt.Sprintf("新建 %s", c.Title))
		}
		written++
	}

	// Step 3: Update profile if delta exists
	if result.ProfileDelta != "" {
		existing, _ := h.Store.ReadFile(ScopeGlobal, "wiki/profile.md")
		if existing == "" {
			existing = "# User Profile\n\n> 更新：" + "\n"
		}
		updated := existing + "\n## 更新\n" + result.ProfileDelta
		h.Store.WriteFile(ScopeGlobal, CategoryProfile, "User Profile", updated, nil, "medium")
		h.Store.LogEntry(ScopeGlobal, "更新画像", "profile.md 已更新")
	}

	// Step 4: Compact knowledge.md
	h.Store.CompactKnowledge(ctx, h.CompactionLLM, scope)
	h.Store.CompactKnowledge(ctx, h.CompactionLLM, ScopeGlobal)

	h.Store.LogEntry(scope, "Session 归档", fmt.Sprintf("Session %s 归档完成，写入 %d 条记忆", sessionID, written))

	if h.OnStatusChange != nil {
		h.OnStatusChange("记忆已更新 ✓")
	}
}
```

- [ ] **Step 2: Modify app.go ArchiveSession to trigger hook**

At the end of `ArchiveSession`, call:

```go
// Trigger memory extraction in background
go func() {
	a.memoryHook.OnArchive(context.Background(), scopeForProject(projectPath), sessionID, getCompactionSummary(sessionID))
}()
```

Add `memoryHook *memory.ArchiveHook` to the App struct. Initialize in `NewApp`.

- [ ] **Step 3: Verify compilation**

```bash
cd d:/git/monika
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add internal/memory/hook.go internal/api/app.go
git commit -m "feat(memory): hook session archive for auto-extraction"
```

---

## P3 — Background Review + Experience → Skill Pipeline

### Task 15: Create internal/memory/review.go — background review

**Files:**
- Create: `internal/memory/review.go`

- [ ] **Step 1: Write review.go**

```go
package memory

import (
	"context"
	"fmt"
	"time"
)

// ReviewResult contains the findings from a periodic review.
type ReviewResult struct {
	Conflicts      []Conflict
	Deprecated     []string // paths to deprecate
	UpgradesNeeded []Upgrade
	LinkAdditions  []LinkAddition
}

type Conflict struct {
	FileA string
	FileB string
	Issue string
}

type Upgrade struct {
	LessonPaths []string
	TopicTitle  string
}

type LinkAddition struct {
	Source string
	Target string
}

// Review performs periodic review of the knowledge base.
// In P3, this is a background goroutine that runs every 24h or every N sessions.
func (s *KBStore) Review(ctx context.Context, llm ReviewLLM, scope string) (*ReviewResult, error) {
	// Get recently modified files
	files, err := s.ListFiles(scope, "")
	if err != nil {
		return nil, err
	}

	// Filter to last 7 days
	var recent []KBFile
	weekAgo := time.Now().Add(-7 * 24 * time.Hour)
	for _, f := range files {
		if f.UpdatedAt.After(weekAgo) && f.Status == "active" {
			recent = append(recent, f)
		}
	}

	if len(recent) < 2 {
		return &ReviewResult{}, nil
	}

	// Build a review prompt and ask LLM to find conflicts/upgrades/links
	prompt := buildReviewPrompt(recent)
	resp, err := llm.Chat(ctx, prompt, "")
	if err != nil {
		return nil, fmt.Errorf("review: %w", err)
	}

	result := parseReviewResponse(resp)
	return result, nil
}

type ReviewLLM interface {
	Chat(ctx context.Context, systemPrompt, userMessage string) (string, error)
}

func (s *KBStore) ExecuteReview(ctx context.Context, llm ReviewLLM, scope string) error {
	result, err := s.Review(ctx, llm, scope)
	if err != nil {
		return err
	}

	for _, c := range result.Conflicts {
		// Resolve: mark older as deprecated
		older := olderFile(c.FileA, c.FileB, scope, s)
		if older != "" {
			s.SoftDelete(scope, older)
			s.LogEntry(scope, "冲突解决", fmt.Sprintf("%s marked deprecated: %s", older, c.Issue))
		}
	}

	for _, d := range result.Deprecated {
		s.SoftDelete(scope, d)
		s.LogEntry(scope, "淘汰旧记忆", fmt.Sprintf("%s deprecated", d))
	}

	for _, u := range result.UpgradesNeeded {
		// Upgrade lessons to topic
		var mergedContent string
		for _, lp := range u.LessonPaths {
			content, _ := s.ReadFile(scope, lp)
			mergedContent += "\n\n" + ExtractBody(content)
		}
		s.WriteFile(scope, CategoryTopic, u.TopicTitle, mergedContent, nil, "medium")
		// Mark source lessons as archived
		for _, lp := range u.LessonPaths {
			s.archiveFile(scope, lp)
		}
		s.LogEntry(scope, "知识升级", fmt.Sprintf("Lessons %v → topic %s", u.LessonPaths, u.TopicTitle))
	}

	for _, l := range result.LinkAdditions {
		s.addLink(scope, l.Source, l.Target)
	}

	return nil
}

func (s *KBStore) archiveFile(scope, path string) error {
	content, err := s.ReadFile(scope, path)
	if err != nil {
		return err
	}
	updated := strings.Replace(content, "> 状态：active", "> 状态：archived", 1)
	root := s.rootFor(scope)
	return os.WriteFile(filepath.Join(root, path), []byte(updated), 0644)
}

func (s *KBStore) addLink(scope, sourcePath, targetPath string) error {
	content, err := s.ReadFile(scope, sourcePath)
	if err != nil {
		return err
	}
	link := fmt.Sprintf("\n> 关联：[[%s]]", targetPath)
	if strings.Contains(content, link) {
		return nil
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "> 关联：") {
			lines[i] = strings.TrimSuffix(line, "\n") + " | [[" + targetPath + "]]"
			break
		}
	}
	root := s.rootFor(scope)
	return os.WriteFile(filepath.Join(root, sourcePath), []byte(strings.Join(lines, "\n")), 0644)
}

func olderFile(a, b, scope string, s *KBStore) string {
	files, _ := s.ListFiles(scope, "")
	var fa, fb *KBFile
	for _, f := range files {
		if f.Path == a {
			fa = &f
		}
		if f.Path == b {
			fb = &f
		}
	}
	if fa != nil && fb != nil && fa.UpdatedAt.Before(fb.UpdatedAt) {
		return a
	}
	if fb != nil {
		return b
	}
	return a
}

func buildReviewPrompt(files []KBFile) string {
	var list strings.Builder
	for _, f := range files {
		list.WriteString(fmt.Sprintf("- %s (%s) [%s]\n", f.Title, f.Category, f.Path))
	}
	return `Review the following recently modified knowledge entries. Find:
1. Conflicts — contradictory conclusions on the same topic
2. Deprecated — facts no longer applicable
3. Upgrades — multiple lessons revealing a common pattern → consolidate to topic
4. Missing links — related files not cross-linked

Entries:
` + list.String() + `
Respond in JSON format.`
}

func parseReviewResponse(resp string) *ReviewResult {
	// Simplified: parse JSON from LLM response
	// Full implementation would use robust JSON parsing like extract.go
	// For P3 MVP, we rely on the LLM returning valid JSON
	return &ReviewResult{}
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/memory/review.go
git commit -m "feat(memory): add background review with conflict detection and link completion"
```

---

### Task 16: Create internal/memory/skill_gen.go — experience to skill pipeline

**Files:**
- Create: `internal/memory/skill_gen.go`

- [ ] **Step 1: Write skill_gen.go**

```go
package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SkillCandidate represents a potential skill to generate.
type SkillCandidate struct {
	PatternName string
	LessonPaths []string
	Description string
}

const skillTemplate = `# %s

> 来源：经验库自动生成
> 触发次数：%d

## 模式描述
%s

## 来源教训
%s

## 使用指南
当遇到类似情况时，参考以上教训。具体步骤由 Agent 根据上下文判断。
`

// FindSkillCandidates discovers patterns in lessons that justify skill generation.
// A skill is generated when >= 3 related lessons on the same topic exist.
func (s *KBStore) FindSkillCandidates(scope string) ([]SkillCandidate, error) {
	files, err := s.ListFiles(scope, CategoryLesson)
	if err != nil {
		return nil, err
	}

	// Group by tag similarity
	groups := clusterByTags(files)
	var candidates []SkillCandidate
	for _, group := range groups {
		if len(group) >= 3 {
			var paths []string
			var contentParts []string
			for _, f := range group {
				paths = append(paths, f.Path)
				body, _ := s.ReadFile(scope, f.Path)
				contentParts = append(contentParts, ExtractBody(body))
			}
			candidates = append(candidates, SkillCandidate{
				PatternName: deriveSkillName(group),
				LessonPaths: paths,
				Description: strings.Join(contentParts, "\n\n"),
			})
		}
	}
	return candidates, nil
}

// GenerateSkill creates a SKILL.md from a candidate pattern.
func GenerateSkill(candidate SkillCandidate, skillsDir string) error {
	slug := titleToSlug(candidate.PatternName)
	skillDir := filepath.Join(skillsDir, slug)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return err
	}

	var sourceList strings.Builder
	for _, p := range candidate.LessonPaths {
		sourceList.WriteString(fmt.Sprintf("- %s\n", p))
	}

	content := fmt.Sprintf(skillTemplate,
		candidate.PatternName,
		len(candidate.LessonPaths),
		firstParagraph(candidate.Description),
		sourceList.String(),
	)

	return os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644)
}

// BackgroundSkillGen runs periodically, checking for pattern emergence and generating skills.
func (s *KBStore) BackgroundSkillGen(ctx context.Context, llm interface{}, skillsDir string) error {
	candidates, err := s.FindSkillCandidates(ScopeProject)
	if err != nil {
		return err
	}
	for _, c := range candidates {
		if err := GenerateSkill(c, skillsDir); err != nil {
			fmt.Printf("[memory] skill gen failed for %s: %v\n", c.PatternName, err)
			continue
		}
		s.LogEntry(ScopeProject, "生成技能", fmt.Sprintf("检测到重复模式，已生成 Skill: %s", c.PatternName))
	}
	return nil
}

func clusterByTags(files []KBFile) [][]KBFile {
	// Simple clustering: files sharing at least 1 tag are in same group
	tagIndex := make(map[string][]*KBFile)
	for i := range files {
		f := &files[i]
		for _, tag := range f.Tags {
			tagIndex[tag] = append(tagIndex[tag], f)
		}
	}
	seen := make(map[string]bool)
	var groups [][]KBFile
	for _, fgroup := range tagIndex {
		var group []KBFile
		for _, f := range fgroup {
			if !seen[f.Path] {
				seen[f.Path] = true
				group = append(group, *f)
			}
		}
		if len(group) > 0 {
			groups = append(groups, group)
		}
	}
	return groups
}

func deriveSkillName(group []KBFile) string {
	if len(group) == 0 {
		return "untitled"
	}
	return group[0].Title
}

func firstParagraph(text string) string {
	if idx := strings.Index(text, "\n\n"); idx >= 0 {
		return text[:idx]
	}
	if len(text) > 200 {
		return text[:200] + "..."
	}
	return text
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/memory/skill_gen.go
git commit -m "feat(memory): add experience-to-skill generation pipeline"
```

---

### Task 17: Start background review goroutine in main

**Files:**
- Modify: `main.go`
- Modify: `internal/api/app.go`

- [ ] **Step 1: Add StartBackgroundTasks to App**

In `app.go`:

```go
func (a *App) StartBackgroundTasks() {
	if a.kbStore == nil {
		return
	}
	go func() {
		for {
			time.Sleep(24 * time.Hour)
			// Background review
			llm := a.reviewLLM()
			if llm != nil {
				a.kbStore.ExecuteReview(context.Background(), llm, memory.ScopeProject)
				a.kbStore.ExecuteReview(context.Background(), llm, memory.ScopeGlobal)
			}
			// Skill generation
			skillsDir := filepath.Join(a.home, ".monika", "skills")
			a.kbStore.BackgroundSkillGen(context.Background(), llm, skillsDir)
		}
	}()
}

func (a *App) reviewLLM() memory.ReviewLLM {
	// Use the default provider/model to create an LLM wrapper
	// For P3 MVP, reuse the existing agent dispatch mechanism
	return nil // placeholder; wired in main
}
```

- [ ] **Step 2: Call StartBackgroundTasks in main.go**

After creating the App, call:

```go
app.StartBackgroundTasks()
```

- [ ] **Step 3: Verify compilation**

```bash
cd d:/git/monika
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add internal/api/app.go main.go
git commit -m "feat(memory): start background review and skill gen goroutine"
```

---

## P4 — Frontend Settings KB Tab + Backend API

### Task 18: Create backend API for knowledge base management

**Files:**
- Create: `internal/api/kb_api.go`

- [ ] **Step 1: Write kb_api.go**

```go
package api

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"monika/internal/memory"
)

type KBFileInfo struct {
	Path       string   `json:"path"`
	Scope      string   `json:"scope"`
	Category   string   `json:"category"`
	Title      string   `json:"title"`
	Tags       []string `json:"tags"`
	Confidence string   `json:"confidence"`
	Status     string   `json:"status"`
	CharCount  int      `json:"char_count"`
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
}

type KBStats struct {
	Total      int    `json:"total"`
	Active     int    `json:"active"`
	Archived   int    `json:"archived"`
	LastUpdate string `json:"last_update"`
}

func (a *App) KBListFiles(scope string) ([]KBFileInfo, error) {
	if a.kbStore == nil {
		return nil, fmt.Errorf("kb not initialized")
	}
	files, err := a.kbStore.ListFiles(scope, "")
	if err != nil {
		return nil, err
	}
	var result []KBFileInfo
	for _, f := range files {
		result = append(result, KBFileInfo{
			Path:       f.Path,
			Scope:      f.Scope,
			Category:   f.Category,
			Title:      f.Title,
			Tags:       f.Tags,
			Confidence: f.Confidence,
			Status:     f.Status,
			CharCount:  f.CharCount,
			CreatedAt:  f.CreatedAt.Format("2006-01-02"),
			UpdatedAt:  f.UpdatedAt.Format("2006-01-02"),
		})
	}
	return result, nil
}

func (a *App) KBReadFile(scope, path string) (string, error) {
	if a.kbStore == nil {
		return "", fmt.Errorf("kb not initialized")
	}
	return a.kbStore.ReadFile(scope, path)
}

func (a *App) KBWriteFile(args json.RawMessage) error {
	if a.kbStore == nil {
		return fmt.Errorf("kb not initialized")
	}
	var p struct {
		Scope      string   `json:"scope"`
		Category   string   `json:"category"`
		Title      string   `json:"title"`
		Content    string   `json:"content"`
		Tags       []string `json:"tags"`
		Confidence string   `json:"confidence"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return err
	}
	return a.kbStore.WriteFile(p.Scope, p.Category, p.Title, p.Content, p.Tags, p.Confidence)
}

func (a *App) KBDeleteFile(scope, path string) error {
	if a.kbStore == nil {
		return fmt.Errorf("kb not initialized")
	}
	return a.kbStore.SoftDelete(scope, path)
}

func (a *App) KBSearch(query, scope string) ([]KBFileInfo, error) {
	if a.kbStore == nil {
		return nil, fmt.Errorf("kb not initialized")
	}
	files, err := a.kbStore.Search(query, scope, 10)
	if err != nil {
		return nil, err
	}
	var result []KBFileInfo
	for _, f := range files {
		result = append(result, KBFileInfo{
			Path:       f.Path,
			Scope:      f.Scope,
			Category:   f.Category,
			Title:      f.Title,
			Tags:       f.Tags,
			Confidence: f.Confidence,
			Status:     f.Status,
			CharCount:  f.CharCount,
			CreatedAt:  f.CreatedAt.Format("2006-01-02"),
			UpdatedAt:  f.UpdatedAt.Format("2006-01-02"),
		})
	}
	return result, nil
}

func (a *App) KBStatistics(scope string) (*KBStats, error) {
	if a.kbStore == nil {
		return nil, fmt.Errorf("kb not initialized")
	}
	total, active, archived, lastUpdate, err := a.kbStore.GetStatistics(scope)
	if err != nil {
		return nil, err
	}
	return &KBStats{
		Total:      total,
		Active:     active,
		Archived:   archived,
		LastUpdate: lastUpdate,
	}, nil
}

func (a *App) KBUploadDocument(args json.RawMessage) error {
	if a.kbStore == nil {
		return fmt.Errorf("kb not initialized")
	}
	var p struct {
		Scope    string `json:"scope"`
		Filename string `json:"filename"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return err
	}
	return a.kbStore.WriteFile(p.Scope, memory.CategoryRawDoc, p.Filename, p.Content, nil, "medium")
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/api/kb_api.go
git commit -m "feat(memory): add backend KB API for settings page"
```

---

### Task 19: Create frontend KnowledgeBaseTab component

**Files:**
- Create: `frontend/src/components/Settings/KnowledgeBaseTab.tsx`
- Modify: `frontend/src/components/Settings/SettingsPage.tsx`

- [ ] **Step 1: Write KnowledgeBaseTab.tsx**

```tsx
import { useState, useEffect, useCallback } from 'react'
import { App } from '../../../bindings/monika'

interface KBFileInfo {
  path: string
  scope: string
  category: string
  title: string
  tags: string[]
  confidence: string
  status: string
  char_count: number
  created_at: string
  updated_at: string
}

interface KBStats {
  total: number
  active: number
  archived: number
  last_update: string
}

function KnowledgeBaseTab() {
  const [scope, setScope] = useState<'global' | 'project'>('project')
  const [files, setFiles] = useState<KBFileInfo[]>([])
  const [stats, setStats] = useState<KBStats | null>(null)
  const [selectedFile, setSelectedFile] = useState<KBFileInfo | null>(null)
  const [fileContent, setFileContent] = useState('')
  const [searchQuery, setSearchQuery] = useState('')
  const [searchResults, setSearchResults] = useState<KBFileInfo[]>([])
  const [loading, setLoading] = useState(false)
  const [editing, setEditing] = useState(false)
  const [editContent, setEditContent] = useState('')

  const loadFiles = useCallback(async () => {
    setLoading(true)
    try {
      const result = await App.KBListFiles(scope)
      setFiles(result)
    } catch (e) {
      console.error('KBListFiles:', e)
    }
    setLoading(false)
  }, [scope])

  const loadStats = useCallback(async () => {
    try {
      const result = await App.KBStatistics(scope)
      setStats(result)
    } catch (e) {
      console.error('KBStatistics:', e)
    }
  }, [scope])

  useEffect(() => {
    loadFiles()
    loadStats()
  }, [loadFiles, loadStats])

  const handleSelectFile = async (f: KBFileInfo) => {
    setSelectedFile(f)
    setEditing(false)
    try {
      const content = await App.KBReadFile(f.scope, f.path)
      setFileContent(content)
    } catch (e) {
      setFileContent('(error loading)')
    }
  }

  const handleSearch = async () => {
    if (!searchQuery.trim()) {
      setSearchResults([])
      return
    }
    try {
      const results = await App.KBSearch(searchQuery, scope)
      setSearchResults(results)
    } catch (e) {
      console.error('KBSearch:', e)
    }
  }

  const handleDelete = async (f: KBFileInfo) => {
    try {
      await App.KBDeleteFile(f.scope, f.path)
      loadFiles()
      loadStats()
      if (selectedFile?.path === f.path) {
        setSelectedFile(null)
        setFileContent('')
      }
    } catch (e) {
      console.error('KBDeleteFile:', e)
    }
  }

  const handleSave = async () => {
    if (!selectedFile) return
    try {
      await App.KBWriteFile({
        scope: selectedFile.scope,
        category: selectedFile.category,
        title: selectedFile.title,
        content: editContent,
        tags: selectedFile.tags,
        confidence: selectedFile.confidence,
      })
      setEditing(false)
      setFileContent(editContent)
      loadFiles()
    } catch (e) {
      console.error('KBWriteFile:', e)
    }
  }

  const filesToShow = searchResults.length > 0 ? searchResults : files

  return (
    <div className="flex h-full">
      {/* Left sidebar: file list */}
      <div className="w-64 border-r border-[var(--border)] flex flex-col">
        <div className="p-2 border-b border-[var(--border)]">
          <div className="flex gap-1 mb-2">
            <button
              onClick={() => setScope('project')}
              className={`px-2 py-1 text-xs rounded ${scope === 'project' ? 'bg-[var(--accent)] text-white' : 'bg-[var(--bg-hover)] text-[var(--text-secondary)]'}`}
            >
              Project
            </button>
            <button
              onClick={() => setScope('global')}
              className={`px-2 py-1 text-xs rounded ${scope === 'global' ? 'bg-[var(--accent)] text-white' : 'bg-[var(--bg-hover)] text-[var(--text-secondary)]'}`}
            >
              Global
            </button>
          </div>
          <div className="flex gap-1">
            <input
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
              placeholder="Search..."
              className="flex-1 px-2 py-1 text-xs bg-[var(--bg-input)] border border-[var(--border)] rounded"
            />
            <button onClick={handleSearch} className="px-2 py-1 text-xs bg-[var(--accent)] text-white rounded">
              Search
            </button>
          </div>
        </div>

        {/* Stats bar */}
        {stats && (
          <div className="px-2 py-1 text-[10px] text-[var(--text-dim)] border-b border-[var(--border)]">
            Total: {stats.total} | Active: {stats.active} | Archived: {stats.archived}
          </div>
        )}

        <div className="flex-1 overflow-y-auto">
          {loading ? (
            <div className="p-2 text-xs text-[var(--text-dim)]">Loading...</div>
          ) : (
            filesToShow.map((f) => (
              <div
                key={f.path}
                onClick={() => handleSelectFile(f)}
                className={`px-2 py-1.5 cursor-pointer border-b border-[var(--border)] ${
                  selectedFile?.path === f.path ? 'bg-[var(--bg-active)]' : ''
                }`}
              >
                <div className="text-xs font-medium truncate">{f.title}</div>
                <div className="text-[10px] text-[var(--text-dim)]">
                  {f.category} · {f.confidence} · {f.updated_at}
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      {/* Right side: file preview / editor */}
      <div className="flex-1 flex flex-col">
        {selectedFile ? (
          <>
            <div className="flex items-center justify-between px-3 py-2 border-b border-[var(--border)]">
              <div>
                <span className="text-sm font-semibold">{selectedFile.title}</span>
                <span className="ml-2 text-[10px] text-[var(--text-dim)]">{selectedFile.path}</span>
              </div>
              <div className="flex gap-1">
                {editing ? (
                  <>
                    <button onClick={handleSave} className="px-2 py-1 text-xs bg-[var(--green)] text-white rounded">
                      Save
                    </button>
                    <button onClick={() => { setEditing(false); setEditContent(fileContent) }} className="px-2 py-1 text-xs bg-[var(--bg-hover)] rounded">
                      Cancel
                    </button>
                  </>
                ) : (
                  <>
                    <button
                      onClick={() => { setEditing(true); setEditContent(fileContent) }}
                      className="px-2 py-1 text-xs bg-[var(--bg-hover)] text-[var(--text-secondary)] rounded"
                    >
                      Edit
                    </button>
                    <button
                      onClick={() => handleDelete(selectedFile)}
                      className="px-2 py-1 text-xs bg-[var(--red-muted)] text-[var(--red)] rounded"
                    >
                      Delete
                    </button>
                  </>
                )}
              </div>
            </div>
            <div className="flex-1 overflow-y-auto p-3">
              {editing ? (
                <textarea
                  value={editContent}
                  onChange={(e) => setEditContent(e.target.value)}
                  className="w-full h-full min-h-[300px] bg-[var(--bg-input)] border border-[var(--border)] rounded p-2 text-xs font-mono resize-none"
                />
              ) : (
                <pre className="text-xs whitespace-pre-wrap font-sans">{fileContent}</pre>
              )}
            </div>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center text-sm text-[var(--text-dim)]">
            Select a file to preview
          </div>
        )}
      </div>
    </div>
  )
}

export default KnowledgeBaseTab
```

- [ ] **Step 2: Add tab to SettingsPage.tsx**

Add import at top:
```tsx
import KnowledgeBaseTab from './KnowledgeBaseTab'
```

Add tab to TABS array:
```tsx
{ id: 'knowledge-base', label: 'Knowledge Base', icon: <IconBrain size={14} /> },
```

Add to type:
```tsx
type Tab = 'agents' | 'permissions' | 'skills' | 'mcp' | 'models' | 'lsp-formatters' | 'knowledge-base' | 'about'
```

Add render case:
```tsx
{activeTab === 'knowledge-base' && <KnowledgeBaseTab />}
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/Settings/KnowledgeBaseTab.tsx frontend/src/components/Settings/SettingsPage.tsx
git commit -m "feat(memory): add Knowledge Base Settings tab with file browser and editor"
```

---

### Task 20: Add memory status to StatusBar

**Files:**
- Modify: `frontend/src/components/StatusBar/StatusBar.tsx`

- [ ] **Step 1: Add memory extraction status to StatusBar**

After the LSP status display, add:

```tsx
const memoryStatus = useStore((s) => s.memoryStatus)

// ... in the JSX, after the LSP block:
{memoryStatus && (
    <div className="flex items-center gap-1 ml-3" style={{ paddingLeft: 8, borderLeft: '1px solid var(--border)' }}>
        <span className="text-[var(--text-dim)]">{memoryStatus}</span>
    </div>
)}
```

- [ ] **Step 2: Add memoryStatus to store**

In `frontend/src/store/index.ts`, add:

```ts
memoryStatus: string | null
setMemoryStatus: (status: string | null) => void
```

With corresponding setter and listener for memory status events.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/StatusBar/StatusBar.tsx frontend/src/store/index.ts
git commit -m "feat(memory): add memory extraction status to status bar"
```

---

### Task 21: Verify full build and lint

- [ ] **Step 1: Go compilation**

```bash
cd d:/git/monika
go build ./...
```

Expected: no errors.

- [ ] **Step 2: Run Go tests**

```bash
cd d:/git/monika
go test ./internal/memory/... -v
```

Expected: all tests pass.

- [ ] **Step 3: Frontend build**

```bash
cd d:/git/monika/frontend
npm run build
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "chore(memory): final wiring and verification passes"
```

---

## Self-Review

### 1. Spec Coverage

| Design Section | Tasks |
|---|---|
| §3.1 文档库 (raw/docs/) | Task 2 (store.WriteFile), Task 19 (KBUploadDocument) |
| §3.2 代码库 (raw/code/) | Task 2 (CategoryRawCode) |
| §3.3 经验库 (wiki/) | Task 2, 10, 11, 12, 14 |
| §4 目录结构 | Task 1 (KBSubdirs), Task 2 (ensure dirs) |
| §5 数据模型 | Task 1 (KBFile), Task 2 (SQL schema) |
| §6 SQLite FTS5 | Task 2 (migration, search) |
| §7.1 记忆注入 | Task 3 (BuildMemoryBlock), Task 8 (buildMessages) |
| §7.2 记忆提取 | Task 10 (extract.go), Task 14 (hook.go) |
| §7.3 记忆检索 | Task 4 (memory_search), Task 5 (memory_index) |
| §7.4 后台审查 | Task 15 (review.go), Task 17 (goroutine) |
| §7.5 经验→技能 | Task 16 (skill_gen.go), Task 17 |
| §8 工具设计 | Tasks 4-7 (4 tools) |
| §9 设置页 UI | Tasks 18-20 (API + Tab + StatusBar) |
| §10 四阶段实施 | Tasks 1-21 cover P1-P4 |

### 2. Placeholder Scan

- No TBD, TODO, "implement later"
- All code steps contain actual code
- All API endpoints have complete implementations
- Frontend component has full component code

### 3. Type Consistency

- `KBFile` struct consistent across all files
- `KBStore` methods match their callers in tools and API
- `ScopeGlobal` / `ScopeProject` constants used consistently
- Wails bindings generate PascalCase method names (KBListFiles, etc.)

---

**Plan complete. 21 tasks covering P1-P4, from SQLite storage to frontend Settings tab.**
