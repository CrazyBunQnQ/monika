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
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	_ "modernc.org/sqlite"
)

type KBStore struct {
	mu          sync.RWMutex
	globalPath  string
	projectPath string
	globalDB    *sql.DB
	projectDB   *sql.DB
}

func NewKBStore(homeDir, projectDir string) (*KBStore, error) {
	gp := GlobalKBPath(homeDir)
	for _, sub := range KBSubdirs() {
		if err := os.MkdirAll(filepath.Join(gp, sub), 0755); err != nil {
			return nil, fmt.Errorf("kb mkdir %s: %w", gp, err)
		}
	}
	s := &KBStore{globalPath: gp}

	gdb, err := openDB(filepath.Join(gp, ".index", "kb.db"))
	if err != nil {
		return nil, fmt.Errorf("open global db: %w", err)
	}
	s.globalDB = gdb

	s.autoReindex(ScopeGlobal)

	if projectDir != "" {
		if err := s.initProjectKB(projectDir); err != nil {
			gdb.Close()
			return nil, err
		}
	}

	return s, nil
}

// initProjectKB 在 projectDir 下创建知识库目录结构和 SQLite 索引。
// 调用方负责加锁。
func (s *KBStore) initProjectKB(projectDir string) error {
	pp := ProjectKBPath(projectDir)
	for _, sub := range KBSubdirs() {
		if err := os.MkdirAll(filepath.Join(pp, sub), 0755); err != nil {
			return fmt.Errorf("kb mkdir %s: %w", pp, err)
		}
	}
	pdb, err := openDB(filepath.Join(pp, ".index", "kb.db"))
	if err != nil {
		return fmt.Errorf("open project db: %w", err)
	}
	s.projectPath = pp
	s.projectDB = pdb
	s.autoReindex(ScopeProject)
	return nil
}

// SetProjectDir 切换项目知识库到新的项目目录。
// 关闭旧的 project DB，打开新项目的 KB。
func (s *KBStore) SetProjectDir(projectDir string) error {
	if projectDir != "" {
		newPP := ProjectKBPath(projectDir)
		s.mu.RLock()
		same := s.projectDB != nil && s.projectPath == newPP
		s.mu.RUnlock()
		if same {
			return nil
		}

		for _, sub := range KBSubdirs() {
			if err := os.MkdirAll(filepath.Join(newPP, sub), 0755); err != nil {
				return fmt.Errorf("kb mkdir %s: %w", newPP, err)
			}
		}
		pdb, err := openDB(filepath.Join(newPP, ".index", "kb.db"))
		if err != nil {
			return fmt.Errorf("open project db: %w", err)
		}

		s.mu.Lock()
		oldDB := s.projectDB
		s.projectDB = pdb
		s.projectPath = newPP
		s.mu.Unlock()

		if oldDB != nil {
			oldDB.Close()
		}
		s.autoReindex(ScopeProject)
	} else {
		s.mu.Lock()
		oldDB := s.projectDB
		s.projectDB = nil
		s.projectPath = ""
		s.mu.Unlock()
		if oldDB != nil {
			oldDB.Close()
		}
	}
	return nil
}

func (s *KBStore) dbFor(scope string) *sql.DB {
	if scope == ScopeGlobal {
		return s.globalDB
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.projectDB
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

// autoReindex 如果 DB 索引为空但磁盘上有 .md 文件，从磁盘重建索引。
func (s *KBStore) autoReindex(scope string) {
	var count int
	s.dbFor(scope).QueryRow("SELECT COUNT(*) FROM file_index").Scan(&count)
	if count > 0 {
		return
	}
	s.ReindexFromDisk(scope)
}

// buildFTSQuery 构建安全的 FTS5 短语查询。
// 将每个词用双引号包裹（内部 " 转义为 ""），避免 *, (), :, AND/OR/NOT 等
// 被 FTS5 当作语法元素；词间用 operator（"AND" 或 "OR"）连接。
func buildFTSQuery(query, operator string) string {
	words := strings.Fields(query)
	quoted := make([]string, 0, len(words))
	for _, w := range words {
		w = strings.ReplaceAll(w, `"`, `""`)
		quoted = append(quoted, `"`+w+`"`)
	}
	return strings.Join(quoted, " "+operator+" ")
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
	return s.SearchHybrid(query, scope, limit)
}

// SearchHybrid performs multi-channel lexical search (FTS5 + LIKE) on a single
// scope and reranks the candidate pool. The wider-than-requested pool gives the
// reranker room to improve ordering over raw BM25 output.
//
// Graceful degradation: when no embedding provider is configured (the current
// default), this is pure lexical search + rerank. The TODO marks where semantic
// search will be folded in once a provider is wired.
func (s *KBStore) SearchHybrid(query, scope string, limit int) ([]KBFile, error) {
	if limit <= 0 {
		limit = 5
	}

	// Candidate pool is 2x the requested limit so reranking can reorder rather
	// than just truncate. Capped to keep latency bounded on large matches.
	poolSize := limit * 2
	if poolSize > 20 {
		poolSize = 20
	}

	candidates, err := s.searchSingle(query, scope, poolSize)
	if err != nil {
		return nil, err
	}

	// TODO(p3-2): when an EmbeddingProvider is configured on the store, run
	// semantic search (cosineSimilarity over the embeddings table) and merge
	// its top-k into the candidate pool before reranking. Intentionally
	// skipped on the lexical-only path to avoid touching the DB when no
	// provider is present.

	if len(candidates) > limit {
		candidates = rerankCandidates(query, candidates, limit)
	}
	return candidates, nil
}

// searchAuto 合并搜索 project 和 global 两个 DB，然后用 rerankCandidates 统一排序。
// project 不再天然优先——rerank 会基于 tag/title 重叠和 BM25 排名决定最终顺序。
func (s *KBStore) searchAuto(query string, limit int) ([]KBFile, error) {
	// 两个 scope 都拉更宽的候选池（2x limit），给统一 rerank 留出重排空间。
	poolSize := limit * 2
	if poolSize > 20 {
		poolSize = 20
	}

	proj, err := s.searchSingle(query, ScopeProject, poolSize)
	if err != nil {
		return nil, err
	}
	glob, err := s.searchSingle(query, ScopeGlobal, poolSize)
	if err != nil {
		return nil, err
	}

	merged := mergeSearchResults(proj, glob, poolSize)
	if len(merged) > limit {
		merged = rerankCandidates(query, merged, limit)
	}
	return merged, nil
}

func (s *KBStore) searchSingle(query, scope string, limit int) ([]KBFile, error) {
	if s.dbFor(scope) == nil {
		return nil, nil
	}
	// FTS5 始终执行：即使查询含 CJK，也能命中被 FTS5 索引的英文/拉丁词。
	ftsResults, err := s.searchFTS(query, scope, limit)
	if err != nil {
		return nil, err
	}
	// CJK 查询额外跑 LIKE：FTS5 的 unicode61 分词器无法正确切分 CJK，
	// 用子串匹配补召回，再与 FTS 结果合并去重。
	if containsCJK(query) {
		likeResults, err := s.searchLike(query, scope, limit)
		if err != nil {
			return nil, err
		}
		return mergeSearchResults(ftsResults, likeResults, limit), nil
	}
	return ftsResults, nil
}

// mergeSearchResults 合并两组结果，按 Scope + "/" + Path 去重，并截断到 limit。
func mergeSearchResults(a, b []KBFile, limit int) []KBFile {
	seen := make(map[string]bool)
	merged := make([]KBFile, 0, len(a)+len(b))
	for _, f := range append(a, b...) {
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
	return merged
}

// searchFTS 使用 FTS5 全文搜索（适用于英文等以空格分词的语言）。
// 策略：先尝试 AND（所有词都命中，精度高）；若 AND 无结果再回退到 OR（任一词命中，召回高）。
func (s *KBStore) searchFTS(query, scope string, limit int) ([]KBFile, error) {
	if results, err := s.execFTS(buildFTSQuery(query, "AND"), scope, limit); err == nil && len(results) > 0 {
		return results, nil
	}
	return s.execFTS(buildFTSQuery(query, "OR"), scope, limit)
}

// execFTS 执行给定的（已转义为短语）FTS5 MATCH 查询。
func (s *KBStore) execFTS(query, scope string, limit int) ([]KBFile, error) {
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
	rows, err := s.dbFor(scope).Query(q, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()
	return scanKBFiles(rows)
}

// isCJKRune reports whether r is a CJK ideograph or Japanese kana.
func isCJKRune(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
		(r >= 0x3400 && r <= 0x4DBF) || // CJK Extension A
		(r >= 0x3040 && r <= 0x30FF) // Hiragana + Katakana
}

// splitSearchTerms extracts search terms from a query string. For
// whitespace-delimited words (English, etc.) it uses strings.Fields. For CJK
// text (which has no word boundaries), it additionally extracts 2-character
// bi-grams so that LIKE matching can find relevant memories. Without bi-grams,
// the entire CJK sentence becomes one LIKE pattern and almost never matches.
func splitSearchTerms(query string) []string {
	var terms []string
	seen := make(map[string]bool)
	add := func(s string) {
		if !seen[s] {
			seen[s] = true
			terms = append(terms, s)
		}
	}
	for _, f := range strings.Fields(query) {
		if containsCJK(f) {
			runes := []rune(f)
			added := false
			for i := 0; i+1 < len(runes); i++ {
				if isCJKRune(runes[i]) && isCJKRune(runes[i+1]) {
					add(string(runes[i : i+2]))
					added = true
				}
			}
			if !added && len(runes) > 0 && isCJKRune(runes[0]) {
				add(f)
			}
		} else if len(f) > 1 {
			add(f)
		}
	}
	return terms
}

// searchLike 使用 LIKE 子串匹配搜索（适用于中文等 FTS5 unicode61 无法正确分词的语言）。
// CJK 查询通过 bi-gram 提取后用 OR 组合，任何 bi-gram 命中即召回。
func (s *KBStore) searchLike(query, scope string, limit int) ([]KBFile, error) {
	terms := splitSearchTerms(query)
	if len(terms) == 0 {
		return nil, nil
	}

	var conditions []string
	var args []any
	for _, w := range terms {
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
		  AND (`+strings.Join(conditions, " OR ")+`)
		ORDER BY f.updated_at DESC
		LIMIT ?
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()
	results, err := scanKBFiles(rows)
	if err != nil {
		return nil, err
	}
	for i, f := range results {
		if f.Snippet == "" {
			continue
		}
		if pos := findMatchPosition(f.Snippet, query); pos >= 0 {
			results[i].Snippet = buildContextualSnippet(f.Snippet, pos, 120)
		}
	}
	return results, nil
}

// findMatchPosition returns the byte index of the first occurrence of any query
// word (length > 1) in s, case-insensitive. Returns -1 when no query word is
// present. Used to center LIKE-search snippets on the actual match.
func findMatchPosition(s, query string) int {
	sLower := strings.ToLower(s)
	minPos := -1
	for _, w := range splitSearchTerms(strings.ToLower(query)) {
		if len(w) <= 1 {
			continue
		}
		if pos := strings.Index(sLower, w); pos >= 0 && (minPos == -1 || pos < minPos) {
			minPos = pos
		}
	}
	return minPos
}

// buildContextualSnippet returns a window of approximately windowLen bytes
// centered on pos, with ellipses marking truncation. pos must be a valid byte
// offset into s. Start/end are snapped to rune boundaries to avoid splitting
// multibyte characters.
func buildContextualSnippet(s string, pos, windowLen int) string {
	if pos < 0 || windowLen <= 0 || pos >= len(s) {
		return s
	}
	start := pos - windowLen/2
	if start < 0 {
		start = 0
	}
	end := start + windowLen
	if end > len(s) {
		end = len(s)
	}
	for start < len(s) && !utf8.RuneStart(s[start]) {
		start--
	}
	for end > start && end < len(s) && !utf8.RuneStart(s[end]) {
		end--
	}
	out := s[start:end]
	if start > 0 {
		out = "…" + out
	}
	if end < len(s) {
		out += "…"
	}
	return out
}

func likeEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

func (s *KBStore) WriteFile(scope, category, title, content string, tags []string, confidence string) error {
	if tags == nil {
		tags = []string{}
	}
	tags = normalizeTags(tags)
	if confidence == "" {
		confidence = "medium"
	}

	// Dedup check: lexical signals only (no LLM). Runs on every write, so it
	// must stay cheap. Rejects near-duplicates; guides caller to memory_update.
	queryText := title + " " + content
	if existing, _ := s.Search(queryText, scope, 3); len(existing) > 0 {
		for _, e := range existing {
			if sim := computeWriteSimilarity(title, tags, e); sim >= 0.75 {
				return fmt.Errorf("similar memory already exists: %s (path: %s, similarity: %.2f). Use memory_update to merge instead",
					e.Title, e.Path, sim)
			} else if sim >= 0.5 && detectContradiction(title, content, e.Title) {
				_ = s.markConflict(scope, title, e.Path)
			}
		}
	}

	return s.writeFileUnchecked(scope, category, title, content, tags, confidence)
}

// writeFileUnchecked writes the memory without running the dedup check.
// Used by bulkImport/ReindexFromDisk where the file already exists on disk
// and must be indexed as-is rather than rejected as a near-duplicate.
func (s *KBStore) writeFileUnchecked(scope, category, title, content string, tags []string, confidence string) error {
	root := s.rootFor(scope)
	if tags == nil {
		tags = []string{}
	}
	tags = normalizeTags(tags)
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

// computeWriteSimilarity returns a 0-1 lexical similarity for write-time dedup.
// Title keyword overlap weighted higher than tags: two memories with the same
// title word-set are almost always the same fact regardless of body wording.
func computeWriteSimilarity(title string, tags []string, existing KBFile) float64 {
	tagSim := tagOverlap(tags, existing.Tags)
	titleSim := keywordOverlap(title, existing.Title)
	return tagSim*0.4 + titleSim*0.6
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
	// file_index.path 以正斜杠存储（categoryPath 用 path.Join），Windows 上
	// filepath.Clean 会产生反斜杠导致 WHERE path = ? 匹配不到行，需归一化。
	dbPath := filepath.ToSlash(cleanPath)
	title := extractTitleFromContent(content)
	_, err := s.dbFor(scope).Exec(`
		UPDATE file_index SET content = ?, title = ?, char_count = ?, updated_at = ?
		WHERE path = ?
	`, content, title, charCount, now, dbPath)
	if err != nil {
		return fmt.Errorf("update index: %w", err)
	}
	return nil
}

// extractTitleFromContent 从 markdown 内容中提取标题（第一个 # 开头的行）。
// 供 UpdateFile 刷新 DB title 列使用，避免 search/index 显示旧标题。
func extractTitleFromContent(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

func (s *KBStore) ListFiles(scope, category string) ([]KBFile, error) {
	db := s.dbFor(scope)
	if db == nil {
		return nil, nil
	}
	var rows *sql.Rows
	var err error
	if category != "" {
		rows, err = db.Query(`
			SELECT id, path, scope, category, title, tags, confidence, status, char_count, linked_to, created_at, updated_at
			FROM file_index WHERE category = ? AND status != 'trash'
			ORDER BY updated_at DESC
		`, category)
	} else {
		rows, err = db.Query(`
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

// BuildIndex generates a compact one-line-per-entry index of saved memories.
// Sorted by updated_at DESC. Returns empty string if no memories.
// The index is intended for inclusion in the system prompt so the LLM can
// discover existing memories and proactively memory_read relevant ones.
func (s *KBStore) BuildIndex(scope string, limit int) (string, error) {
	files, err := s.ListFiles(scope, "")
	if err != nil || len(files) == 0 {
		return "", nil
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].UpdatedAt.After(files[j].UpdatedAt)
	})
	if limit > 0 && len(files) > limit {
		files = files[:limit]
	}

	var b strings.Builder
	for i, f := range files {
		fmt.Fprintf(&b, "%d. [%s] %s (%s)", i+1, categoryLabel(f.Category), f.Title, f.Path)
		if len(f.Tags) > 0 {
			fmt.Fprintf(&b, " tags: %s", strings.Join(f.Tags, ", "))
		}
		b.WriteString("\n")
	}
	return b.String(), nil
}

// categoryLabel collapses a category path like "wiki/lesson" or "raw/doc"
// to its trailing segment ("lesson", "doc") for compact index rendering.
func categoryLabel(category string) string {
	if i := strings.LastIndex(category, "/"); i >= 0 {
		return category[i+1:]
	}
	return category
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.projectPath == "" {
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
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
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

		if err := s.writeFileUnchecked(scope, category, title, body, tags, confidence); err != nil {
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
	i := 0
	if i < len(lines) && strings.HasPrefix(lines[i], "# ") {
		i++
	}
	for i < len(lines) {
		if lines[i] == "" || strings.HasPrefix(lines[i], "> ") {
			i++
		} else {
			break
		}
	}
	return strings.Join(lines[i:], "\n")
}
