package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type Entity struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Scope string `json:"scope"`
}

var (
	filePathRe  = regexp.MustCompile(`[\w\-./]+\.(go|ts|tsx|js|jsx|py|rs|java|c|cpp|h|hpp|md|yaml|yml|json|toml|sql|sh|rb|php|vue|svelte)`)
	backtickRe  = regexp.MustCompile("`[^`\n]{2,60}`")
	funcCallRe  = regexp.MustCompile(`\b([A-Z]\w*)\(`)
	codeFenceRe = regexp.MustCompile("(?s)```\\w*\\n(.*?)```")
)

func ensureEntitiesTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS entities (
			id    INTEGER PRIMARY KEY AUTOINCREMENT,
			name  TEXT NOT NULL,
			type  TEXT NOT NULL,
			scope TEXT NOT NULL DEFAULT 'project',
			UNIQUE(name, type, scope)
		);
		CREATE TABLE IF NOT EXISTS memory_entities (
			memory_id INTEGER NOT NULL,
			entity_id INTEGER NOT NULL,
			PRIMARY KEY(memory_id, entity_id)
		);
		CREATE INDEX IF NOT EXISTS idx_me_memory ON memory_entities(memory_id);
		CREATE INDEX IF NOT EXISTS idx_me_entity ON memory_entities(entity_id);
	`)
	return err
}

func extractEntities(content string) []Entity {
	seen := make(map[string]bool)
	var entities []Entity

	add := func(name, typ string) {
		name = strings.TrimSpace(name)
		if len(name) < 2 || len(name) > 120 {
			return
		}
		key := typ + ":" + name
		if seen[key] {
			return
		}
		seen[key] = true
		entities = append(entities, Entity{Name: name, Type: typ})
	}

	for _, m := range filePathRe.FindAllString(content, -1) {
		add(m, "file")
	}

	fenced := codeFenceRe.FindAllStringSubmatch(content, -1)
	for _, m := range fenced {
		for _, line := range strings.Split(m[1], "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
				continue
			}
			for _, fm := range funcCallRe.FindAllStringSubmatch(line, -1) {
				add(fm[1], "function")
			}
		}
	}

	for _, m := range backtickRe.FindAllString(content, -1) {
		inner := strings.Trim(m, "`")
		if strings.ContainsAny(inner, " \t") {
			add(inner, "concept")
		} else if len(inner) > 2 {
			add(inner, "identifier")
		}
	}

	return entities
}

func (s *KBStore) indexEntities(scope, relPath string, content string) error {
	db := s.dbFor(scope)
	if db == nil {
		return nil
	}

	var memoryID int64
	if err := db.QueryRow("SELECT id FROM file_index WHERE path = ?", relPath).Scan(&memoryID); err != nil {
		return nil
	}

	if _, err := db.Exec("DELETE FROM memory_entities WHERE memory_id = ?", memoryID); err != nil {
		return fmt.Errorf("clear entities: %w", err)
	}

	tags, _ := s.getTagsForPath(scope, relPath)
	for _, tag := range tags {
		s.upsertEntity(db, scope, tag, "tag", memoryID)
	}

	entities := extractEntities(content)
	for _, e := range entities {
		s.upsertEntity(db, scope, e.Name, e.Type, memoryID)
	}

	return nil
}

func (s *KBStore) upsertEntity(db *sql.DB, scope, name, typ string, memoryID int64) {
	var entityID int64
	err := db.QueryRow(
		`INSERT INTO entities (name, type, scope) VALUES (?, ?, ?)
		 ON CONFLICT(name, type, scope) DO UPDATE SET name=name
		 RETURNING id`,
		name, typ, scope).Scan(&entityID)
	if err != nil {
		return
	}
	db.Exec("INSERT OR IGNORE INTO memory_entities (memory_id, entity_id) VALUES (?, ?)", memoryID, entityID)
}

func (s *KBStore) getTagsForPath(scope, relPath string) ([]string, error) {
	db := s.dbFor(scope)
	if db == nil {
		return nil, nil
	}
	var tagsJSON string
	if err := db.QueryRow("SELECT tags FROM file_index WHERE path = ?", relPath).Scan(&tagsJSON); err != nil {
		return nil, nil
	}
	var tags []string
	json.Unmarshal([]byte(tagsJSON), &tags)
	return tags, nil
}

func (s *KBStore) removeEntitiesForPath(scope string, memoryID int64) error {
	db := s.dbFor(scope)
	if db == nil {
		return nil
	}
	_, err := db.Exec("DELETE FROM memory_entities WHERE memory_id = ?", memoryID)
	return err
}

func (s *KBStore) QueryByEntity(scope, entityName string) ([]KBFile, error) {
	db := s.dbFor(scope)
	if db == nil {
		return nil, nil
	}
	rows, err := db.Query(`
		SELECT f.id, f.path, f.scope, f.category, f.title, f.tags,
		       f.confidence, f.status, f.char_count, f.linked_to,
		       f.created_at, f.updated_at, ''
		FROM file_index f
		JOIN memory_entities me ON f.id = me.memory_id
		JOIN entities e ON me.entity_id = e.id
		WHERE e.name = ? AND f.scope = ? AND f.status != 'trash'
		ORDER BY f.updated_at DESC
	`, entityName, scope)
	if err != nil {
		return nil, fmt.Errorf("query by entity: %w", err)
	}
	defer rows.Close()
	return scanKBFiles(rows)
}

func (s *KBStore) getEntitiesForPath(scope, relPath string) ([]string, error) {
	db := s.dbFor(scope)
	if db == nil {
		return nil, nil
	}
	rows, err := db.Query(`
		SELECT e.name FROM entities e
		JOIN memory_entities me ON e.id = me.entity_id
		JOIN file_index f ON me.memory_id = f.id
		WHERE f.path = ?
	`, relPath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	return names, nil
}

func (s *KBStore) AllEntities(scope string) ([]Entity, error) {
	db := s.dbFor(scope)
	if db == nil {
		return nil, nil
	}
	rows, err := db.Query(`
		SELECT e.id, e.name, e.type, e.scope
		FROM entities e
		WHERE e.scope = ?
		ORDER BY e.name
	`, scope)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entities []Entity
	for rows.Next() {
		var e Entity
		if err := rows.Scan(&e.ID, &e.Name, &e.Type, &e.Scope); err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	return entities, nil
}

func (s *KBStore) EntityNeighborhood(scope, entityName string, depth int) (map[string][]string, error) {
	if depth <= 0 {
		depth = 1
	}
	result := make(map[string][]string)

	seedMems, err := s.QueryByEntity(scope, entityName)
	if err != nil {
		return nil, err
	}

	visited := make(map[string]bool)
	for _, m := range seedMems {
		ents, _ := s.getEntitiesForPath(scope, m.Path)
		for _, e := range ents {
			if e != entityName && !visited[e] {
				visited[e] = true
				related, _ := s.QueryByEntity(scope, e)
				for _, r := range related {
					result[e] = append(result[e], r.Title)
				}
			}
		}
		result[m.Path] = ents
	}

	return result, nil
}
