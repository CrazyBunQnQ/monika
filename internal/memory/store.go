package memory

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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
	dbPath := filepath.Join(s.projectPath, ".index", "kb.db")
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000")
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
	var id int64
	if err := s.db.QueryRow("SELECT id FROM file_index WHERE path = ?", relPath).Scan(&id); err != nil {
		return fmt.Errorf("soft-delete scan id: %w", err)
	}
	if _, err := s.db.Exec("DELETE FROM file_fts WHERE rowid = ?", id); err != nil {
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
