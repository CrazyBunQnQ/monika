package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type MemoryEdge struct {
	Target   string `json:"target"`
	Relation string `json:"relation,omitempty"`
}

type GraphNode struct {
	File  KBFile
	Depth int
	Path  []string
}

func ensureEdgesTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS memory_edges (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			source_path TEXT NOT NULL,
			target_path TEXT NOT NULL,
			relation    TEXT NOT NULL DEFAULT 'related',
			created_at  TEXT NOT NULL,
			UNIQUE(source_path, target_path, relation)
		);
		CREATE INDEX IF NOT EXISTS idx_edges_source ON memory_edges(source_path);
		CREATE INDEX IF NOT EXISTS idx_edges_target ON memory_edges(target_path);
	`)
	return err
}

func (s *KBStore) addTypedLink(scope, sourcePath, targetPath, relation string) error {
	db := s.dbFor(scope)
	if db == nil {
		return fmt.Errorf("scope %q has no database initialized", scope)
	}
	if relation == "" {
		relation = "related"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(
		`INSERT INTO memory_edges (source_path, target_path, relation, created_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(source_path, target_path, relation) DO NOTHING`,
		sourcePath, targetPath, relation, now)
	if err != nil {
		return fmt.Errorf("add edge: %w", err)
	}
	linkedTo, _ := s.getLinkedPaths(scope, sourcePath)
	_ = s.setLinkedTo(scope, sourcePath, linkedTo)
	return nil
}

func (s *KBStore) LinkByTitle(scope, sourceCategory, sourceTitle, targetTitle, relation string) error {
	sourcePath := categoryPath(sourceCategory, sourceTitle)
	results, err := s.Search(targetTitle, scope, 3)
	if err != nil || len(results) == 0 {
		return nil
	}
	for _, r := range results {
		titleSim := keywordOverlap(targetTitle, r.Title)
		if titleSim >= 0.5 || strings.Contains(strings.ToLower(r.Title), strings.ToLower(targetTitle)) {
			return s.addTypedLink(scope, sourcePath, r.Path, relation)
		}
	}
	return nil
}

func (s *KBStore) getLinkedPaths(scope, sourcePath string) ([]string, error) {
	db := s.dbFor(scope)
	if db == nil {
		return nil, nil
	}
	rows, err := db.Query(
		`SELECT DISTINCT target_path FROM memory_edges WHERE source_path = ?`,
		sourcePath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return paths, nil
}

func (s *KBStore) getBacklinks(scope, targetPath string) ([]string, error) {
	db := s.dbFor(scope)
	if db == nil {
		return nil, nil
	}
	rows, err := db.Query(
		`SELECT DISTINCT source_path FROM memory_edges WHERE target_path = ?`,
		targetPath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return paths, nil
}

func (s *KBStore) getEdges(scope, path string) ([]MemoryEdge, error) {
	db := s.dbFor(scope)
	if db == nil {
		return nil, nil
	}
	rows, err := db.Query(
		`SELECT target_path, relation FROM memory_edges WHERE source_path = ?`,
		path)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var edges []MemoryEdge
	for rows.Next() {
		var e MemoryEdge
		if err := rows.Scan(&e.Target, &e.Relation); err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}
	return edges, nil
}

func (s *KBStore) removeEdgesForPath(scope, path string) error {
	db := s.dbFor(scope)
	if db == nil {
		return nil
	}
	_, err := db.Exec(
		`DELETE FROM memory_edges WHERE source_path = ? OR target_path = ?`,
		path, path)
	return err
}

func (s *KBStore) fillBacklinks(scope string, results []KBFile) {
	for i := range results {
		bl, err := s.getBacklinks(scope, results[i].Path)
		if err == nil && len(bl) > 0 {
			results[i].Backlinks = bl
		}
	}
}

func (s *KBStore) recordAccess(scope string, results []KBFile) {
	db := s.dbFor(scope)
	if db == nil || len(results) == 0 {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := db.Begin()
	if err != nil {
		return
	}
	stmt, err := tx.Prepare(`UPDATE file_index SET access_count = access_count + 1, last_accessed = ? WHERE id = ?`)
	if err != nil {
		tx.Rollback()
		return
	}
	defer stmt.Close()
	for _, f := range results {
		if f.Scope != scope {
			continue
		}
		_, _ = stmt.Exec(now, f.ID)
	}
	_ = tx.Commit()
}

func (s *KBStore) GraphTraverse(scope, seedPath string, maxHops int, relationFilter string) ([]GraphNode, error) {
	db := s.dbFor(scope)
	if db == nil {
		return nil, fmt.Errorf("scope %q has no database initialized", scope)
	}
	if maxHops <= 0 {
		maxHops = 2
	}

	visited := map[string]bool{seedPath: true}
	queue := []GraphNode{{File: KBFile{}, Depth: 0, Path: []string{seedPath}}}
	var results []GraphNode

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		currentPath := seedPath
		if current.Depth > 0 {
			currentPath = current.File.Path
		}

		edges, _ := s.getEdges(scope, currentPath)
		for _, edge := range edges {
			if relationFilter != "" && edge.Relation != relationFilter {
				continue
			}
			if visited[edge.Target] {
				continue
			}
			visited[edge.Target] = true

			var f KBFile
			row := db.QueryRow(
				`SELECT id, path, scope, category, title, tags, confidence, status, char_count, linked_to, created_at, updated_at, access_count, last_accessed
				 FROM file_index WHERE path = ? AND status != 'trash'`,
				edge.Target)
			var tagsJSON, linkedJSON, ca, ua string
			if err := row.Scan(&f.ID, &f.Path, &f.Scope, &f.Category, &f.Title, &tagsJSON,
				&f.Confidence, &f.Status, &f.CharCount, &linkedJSON, &ca, &ua, &f.AccessCount, new(string)); err != nil {
				continue
			}
			json.Unmarshal([]byte(tagsJSON), &f.Tags)
			json.Unmarshal([]byte(linkedJSON), &f.LinkedTo)
			f.CreatedAt, _ = time.Parse(time.RFC3339, ca)
			f.UpdatedAt, _ = time.Parse(time.RFC3339, ua)

			node := GraphNode{
				File:  f,
				Depth: current.Depth + 1,
				Path:  append(append([]string{}, current.Path...), edge.Target),
			}
			results = append(results, node)
			if node.Depth < maxHops {
				queue = append(queue, node)
			}
		}
	}
	return results, nil
}
