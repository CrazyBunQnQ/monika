package memory

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type KBStore struct {
	globalPath  string
	projectPath string
	globalDB    *sql.DB
	projectDB   *sql.DB
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

	gdb, err := openDB(filepath.Join(gp, ".index", "kb.db"))
	if err != nil {
		return nil, fmt.Errorf("open global db: %w", err)
	}
	s.globalDB = gdb

	pdb, err := openDB(filepath.Join(pp, ".index", "kb.db"))
	if err != nil {
		gdb.Close()
		return nil, fmt.Errorf("open project db: %w", err)
	}
	s.projectDB = pdb

	s.autoReindex(ScopeGlobal)
	s.autoReindex(ScopeProject)

	return s, nil
}

func openDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, err
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS file_index (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			path        TEXT NOT NULL UNIQUE,
			scope       TEXT NOT NULL DEFAULT 'project',
			category    TEXT NOT NULL,
			title       TEXT NOT NULL,
			content     TEXT DEFAULT '',
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
			VALUES (new.id, new.path, new.title, new.content);
		END;
		CREATE TRIGGER IF NOT EXISTS fts_ad AFTER DELETE ON file_index BEGIN
			INSERT INTO file_fts(file_fts, rowid, path, title, content)
			VALUES ('delete', old.id, old.path, old.title, old.content);
		END;
		CREATE TRIGGER IF NOT EXISTS fts_au AFTER UPDATE ON file_index BEGIN
			INSERT INTO file_fts(file_fts, rowid, path, title, content)
			VALUES ('delete', old.id, old.path, old.title, old.content);
			INSERT INTO file_fts(rowid, path, title, content)
			VALUES (new.id, new.path, new.title, new.content);
		END;
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

func (s *KBStore) dbFor(scope string) *sql.DB {
	if scope == ScopeGlobal {
		return s.globalDB
	}
	return s.projectDB
}

// autoReindex 如果 DB 索引为空但磁盘上有 .md 文件，从磁盘重建索引。
func (s *KBStore) autoReindex(scope string) {
	var count int
	s.dbFor(scope).QueryRow("SELECT COUNT(*) FROM file_index").Scan(&count)
	if count > 0 {
		return
	}
	s.ReindexFromDisk(scope)
}

// sanitizeFTSQuery 转义 FTS5 查询中的特殊字符。
// FTS5 的 MATCH 操作符将 *, (), :, "", AND/OR/NOT 等视为语法元素。
// 将每个词用双引号包裹，使其成为短语查询，避免语法错误。
func sanitizeFTSQuery(query string) string {
	words := strings.Fields(query)
	for i, w := range words {
		w = strings.ReplaceAll(w, `"`, `""`)
		words[i] = `"` + w + `"`
	}
	return strings.Join(words, " ")
}

func containsCJK(s string) bool {
	for _, r := range s {
		if (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
			(r >= 0x3400 && r <= 0x4DBF) || // CJK Extension A
			(r >= 0x3040 && r <= 0x30FF) { // Hiragana + Katakana
			return true
		}
	}
	return false
}

func (s *KBStore) Search(query, scope string, limit int) ([]KBFile, error) {
	if limit <= 0 {
		limit = 5
	}

	if scope == ScopeAuto {
		return s.searchAuto(query, limit)
	}
	return s.searchSingle(query, scope, limit)
}

// searchAuto 合并搜索 project 和 global 两个 DB，project 结果优先。
func (s *KBStore) searchAuto(query string, limit int) ([]KBFile, error) {
	proj, err := s.searchSingle(query, ScopeProject, limit)
	if err != nil {
		return nil, err
	}
	glob, err := s.searchSingle(query, ScopeGlobal, limit)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	var merged []KBFile
	for _, f := range append(proj, glob...) {
		key := f.Scope + "/" + f.Path
		if seen[key] {
			continue
		}
		seen[key] = true
		merged = append(merged, f)
	}
	if len(merged) > limit {
		merged = merged[:limit]
	}
	return merged, nil
}

func (s *KBStore) searchSingle(query, scope string, limit int) ([]KBFile, error) {
	if containsCJK(query) {
		return s.searchLike(query, scope, limit)
	}
	return s.searchFTS(query, scope, limit)
}

// searchFTS 使用 FTS5 全文搜索（适用于英文等以空格分词的语言）。
func (s *KBStore) searchFTS(query, scope string, limit int) ([]KBFile, error) {
	args := []any{sanitizeFTSQuery(query), limit}
	q := `
		SELECT f.id, f.path, f.scope, f.category, f.title, f.tags,
		       f.confidence, f.status, f.char_count, f.linked_to,
		       f.created_at, f.updated_at,
		       snippet(file_fts, 2, '<b>', '</b>', '...', 40)
		FROM file_fts
		JOIN file_index f ON file_fts.rowid = f.id
		WHERE file_fts MATCH ? AND f.status != 'trash'
		ORDER BY bm25(file_fts, 0, 10, 5)
		LIMIT ?
	`

	rows, err := s.dbFor(scope).Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()
	return scanKBFiles(rows)
}

// searchLike 使用 LIKE 子串匹配搜索（适用于中文等 FTS5 unicode61 无法正确分词的语言）。
// 将查询按空白拆分为多个词，每个词分别 LIKE 匹配，用 AND 组合。
func (s *KBStore) searchLike(query, scope string, limit int) ([]KBFile, error) {
	words := strings.Fields(query)
	if len(words) == 0 {
		return nil, nil
	}

	var conditions []string
	var args []any
	for _, w := range words {
		likePattern := "%" + likeEscape(w) + "%"
		conditions = append(conditions, `(f.title LIKE ? ESCAPE '\' OR f.content LIKE ? ESCAPE '\')`)
		args = append(args, likePattern, likePattern)
	}
	args = append(args, limit)

	rows, err := s.dbFor(scope).Query(`
		SELECT f.id, f.path, f.scope, f.category, f.title, f.tags,
		       f.confidence, f.status, f.char_count, f.linked_to,
		       f.created_at, f.updated_at,
		       substr(f.content, 1, 200)
		FROM file_index f
		WHERE f.status != 'trash'
		  AND (`+strings.Join(conditions, " AND ")+`)
		ORDER BY f.updated_at DESC
		LIMIT ?
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()
	return scanKBFiles(rows)
}

func likeEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
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
	_, err := s.dbFor(scope).Exec(`
		INSERT INTO file_index (path, scope, category, title, content, tags, confidence, status, char_count, linked_to, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'active', ?, '[]', ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			category=excluded.category, title=excluded.title, content=excluded.content, tags=excluded.tags,
			confidence=excluded.confidence, status='active', char_count=excluded.char_count,
			updated_at=excluded.updated_at
	`, relPath, scope, category, title, content, string(tagsJSON), confidence, charCount, now, now)
	if err != nil {
		return fmt.Errorf("upsert: %w", err)
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

func (s *KBStore) ListFiles(scope, category string) ([]KBFile, error) {
	var rows *sql.Rows
	var err error
	if category != "" {
		rows, err = s.dbFor(scope).Query(`
			SELECT id, path, scope, category, title, tags, confidence, status, char_count, linked_to, created_at, updated_at
			FROM file_index WHERE category = ? AND status != 'trash'
			ORDER BY updated_at DESC
		`, category)
	} else {
		rows, err = s.dbFor(scope).Query(`
			SELECT id, path, scope, category, title, tags, confidence, status, char_count, linked_to, created_at, updated_at
			FROM file_index WHERE status != 'trash'
			ORDER BY updated_at DESC
		`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKBFilesFlat(rows)
}

func (s *KBStore) SoftDelete(scope, relPath string) error {
	db := s.dbFor(scope)
	_, err := db.Exec("UPDATE file_index SET status = 'trash', updated_at = ? WHERE path = ?",
		time.Now().UTC().Format(time.RFC3339), relPath)
	if err != nil {
		return err
	}
	var id int64
	if err := db.QueryRow("SELECT id FROM file_index WHERE path = ?", relPath).Scan(&id); err != nil {
		return fmt.Errorf("soft-delete scan id: %w", err)
	}
	if _, err := db.Exec("DELETE FROM file_fts WHERE rowid = ?", id); err != nil {
		return fmt.Errorf("soft-delete fts: %w", err)
	}
	oldPath := filepath.Join(s.rootFor(scope), relPath)
	trashSubdir := filepath.Join(".trash", filepath.Dir(relPath))
	trashPath := filepath.Join(s.rootFor(scope), trashSubdir)
	if err := os.MkdirAll(trashPath, 0755); err != nil {
		return fmt.Errorf("soft-delete mkdir: %w", err)
	}
	if err := os.Rename(oldPath, filepath.Join(trashPath, filepath.Base(relPath))); err != nil {
		return fmt.Errorf("soft-delete rename: %w", err)
	}
	return nil
}

// setLinkedTo 把某文件的出链列表写入 DB linked_to 列。
// 与 addLink 写入 markdown 的「> 关联：[[...]]」行保持同步，让搜索结果能直接返回依赖关系。
func (s *KBStore) setLinkedTo(scope, relPath string, links []string) error {
	if links == nil {
		links = []string{}
	}
	linksJSON, err := json.Marshal(links)
	if err != nil {
		return fmt.Errorf("marshal linked_to: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.dbFor(scope).Exec(
		"UPDATE file_index SET linked_to = ?, updated_at = ? WHERE path = ?",
		string(linksJSON), now, relPath)
	if err != nil {
		return fmt.Errorf("set linked_to: %w", err)
	}
	return nil
}

func (s *KBStore) SetFileStatus(scope, relPath, status string) error {
	root := s.rootFor(scope)
	fullPath := filepath.Join(root, relPath)
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := s.dbFor(scope).Exec("UPDATE file_index SET status = ?, updated_at = ? WHERE path = ?", status, now, relPath)
	if err != nil {
		return fmt.Errorf("set status: %w", err)
	}

	// Also update the file's frontmatter
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil // file may not exist on disk, DB update is enough
	}
	content := string(data)
	updated := strings.Replace(content, "> 状态：active", "> 状态："+status, 1)
	updated = strings.Replace(updated, "> 状态：archived", "> 状态："+status, 1)
	return os.WriteFile(fullPath, []byte(updated), 0644)
}

func (s *KBStore) GetStatistics(scope string) (total, active, archived int, lastUpdate string, err error) {
	err = s.dbFor(scope).QueryRow(`
		SELECT COUNT(*),
		       SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END),
		       SUM(CASE WHEN status = 'archived' OR status = 'deprecated' THEN 1 ELSE 0 END),
		       COALESCE(MAX(updated_at), '')
		FROM file_index WHERE status != 'trash'
	`).Scan(&total, &active, &archived, &lastUpdate)
	return
}

func (s *KBStore) Close() error {
	var err error
	if s.globalDB != nil {
		err = s.globalDB.Close()
	}
	if s.projectDB != nil {
		if perr := s.projectDB.Close(); perr != nil && err == nil {
			err = perr
		}
	}
	return err
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
		return path.Join("wiki/lessons", slug+".md")
	case CategoryTopic:
		return path.Join("wiki/topics", slug+".md")
	case CategoryRawDoc:
		return path.Join("raw/docs", slug+".md")
	case CategoryRawCode:
		return path.Join("raw/code", slug+".md")
	default:
		return path.Join(category, slug+".md")
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
		h := sha256.Sum256([]byte(title))
		return "untitled-" + hex.EncodeToString(h[:3])
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
		// Ignore unmarshal/parse errors: tags/linked_to are internally written as
		// valid JSON and timestamps always use RFC3339, so failures here are
		// recoverable per-row corruption, not fatal DB errors.
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

func scanKBFilesFlat(rows *sql.Rows) ([]KBFile, error) {
	var results []KBFile
	for rows.Next() {
		var f KBFile
		var tagsJSON, linkedJSON, ca, ua string
		if err := rows.Scan(&f.ID, &f.Path, &f.Scope, &f.Category, &f.Title, &tagsJSON,
			&f.Confidence, &f.Status, &f.CharCount, &linkedJSON, &ca, &ua); err != nil {
			return nil, err
		}
		// Ignore unmarshal/parse errors: tags/linked_to are internally written as
		// valid JSON and timestamps always use RFC3339, so failures here are
		// recoverable per-row corruption, not fatal DB errors.
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

// ReindexFromDisk 遍历磁盘上指定 scope 的所有 .md 文件，
// 解析 frontmatter 并重新写入 FTS 索引。
// 与基于 ListFiles 的旧逻辑不同，此方法直接读取文件系统，
// 即使 DB 索引完全为空也能正常工作。
func (s *KBStore) ReindexFromDisk(scope string) (int, error) {
	root := s.rootFor(scope)
	count := 0

	err := filepath.Walk(root, func(fullPath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".index" || name == ".trash" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}

		relPath, err := filepath.Rel(root, fullPath)
		if err != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)

		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil
		}
		content := string(data)

		title := parseFMTitle(content)
		category := pathToCategory(relPath)
		tags := parseFMTags(content)
		confidence := parseFMField(content, "置信度")
		links := parseFMLinks(content)
		body := extractMDBody(content)

		if err := s.WriteFile(scope, category, title, body, tags, confidence); err != nil {
			return nil
		}
		// 回填 linked_to：WriteFile 总是写 '[]'，这里把正文里解析到的链接同步进 DB，
		// 否则历史文件里的「> 关联：[[...]]」对搜索不可见。
		if len(links) > 0 {
			if err := s.setLinkedTo(scope, relPath, links); err != nil {
				return nil
			}
		}
		count++
		return nil
	})
	return count, err
}

func pathToCategory(relPath string) string {
	dir := filepath.ToSlash(filepath.Dir(relPath))
	switch {
	case relPath == "wiki/knowledge.md":
		return CategoryKnowledge
	case relPath == "wiki/profile.md":
		return CategoryProfile
	case strings.HasPrefix(dir, "wiki/lessons"):
		return CategoryLesson
	case strings.HasPrefix(dir, "wiki/topics"):
		return CategoryTopic
	case strings.HasPrefix(dir, "raw/docs"):
		return CategoryRawDoc
	case strings.HasPrefix(dir, "raw/code"):
		return CategoryRawCode
	default:
		return dir
	}
}

func parseFMTitle(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[2:])
		}
	}
	return "Untitled"
}

func parseFMField(content, field string) string {
	prefix := "> " + field + "："
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(line[len(prefix):])
		}
	}
	return ""
}

func parseFMTags(content string) []string {
	raw := parseFMField(content, "标签")
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			tags = append(tags, p)
		}
	}
	return tags
}

// parseFMLinks 从 frontmatter 的「> 关联：[[a]] | [[b]]」行抽取所有 [[...]] 链接。
// addLink 写入文件时把链接追加到该行，parseFMLinks 与之对称，用于回填 DB linked_to 列。
func parseFMLinks(content string) []string {
	raw := parseFMField(content, "关联")
	if raw == "" {
		return []string{}
	}
	links := make([]string, 0, 4)
	for _, part := range strings.Split(raw, "|") {
		part = strings.TrimSpace(part)
		part = strings.TrimPrefix(part, "[[")
		part = strings.TrimSuffix(part, "]]")
		part = strings.TrimSpace(part)
		if part != "" {
			links = append(links, part)
		}
	}
	return links
}

func extractMDBody(content string) string {
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
