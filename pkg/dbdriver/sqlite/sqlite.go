package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"monika/pkg/dbdriver"

	_ "modernc.org/sqlite"
)

func init() {
	dbdriver.Register(&Driver{})
}

type Driver struct{}

func (d *Driver) ID() string { return "sqlite" }

func (d *Driver) Open(dsn string) (dbdriver.Connection, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open: %w", err)
	}
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite: ping: %w", err)
	}
	return &Conn{db: db}, nil
}

type Conn struct {
	db *sql.DB
}

func (c *Conn) Query(ctx context.Context, query string) (*dbdriver.QueryResult, error) {
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sqlite: query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("sqlite: columns: %w", err)
	}

	var allRows [][]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("sqlite: scan: %w", err)
		}
		row := make([]any, len(cols))
		for i, v := range vals {
			row[i] = sanitizeValue(v)
		}
		allRows = append(allRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: rows: %w", err)
	}

	return &dbdriver.QueryResult{
		Columns: cols,
		Rows:    allRows,
		Tag:     "SELECT",
	}, nil
}

func (c *Conn) Schema(ctx context.Context, filter string) (*dbdriver.SchemaResult, error) {
	q := `SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`
	args := []any{}
	if filter != "" {
		q += ` AND name LIKE ?`
		args = append(args, filter)
	}
	q += ` ORDER BY name`

	rows, err := c.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite: schema tables: %w", err)
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("sqlite: schema scan tables: %w", err)
		}
		tableNames = append(tableNames, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: schema rows: %w", err)
	}

	pkMap, err := c.getPrimaryKeys(ctx)
	if err != nil {
		return nil, err
	}

	tables := make([]dbdriver.TableInfo, 0, len(tableNames))
	for _, name := range tableNames {
		cols, err := c.getTableColumns(ctx, name, pkMap[name])
		if err != nil {
			return nil, err
		}
		tables = append(tables, dbdriver.TableInfo{Name: name, Columns: cols})
	}

	return &dbdriver.SchemaResult{Tables: tables}, nil
}

func (c *Conn) getTableColumns(ctx context.Context, table string, pkSet map[string]bool) ([]dbdriver.ColumnInfo, error) {
	rows, err := c.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", quoteIdent(table)))
	if err != nil {
		return nil, fmt.Errorf("sqlite: pragma table_info: %w", err)
	}
	defer rows.Close()

	var cols []dbdriver.ColumnInfo
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull int
		var dfltValue any
		var pk int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
			return nil, fmt.Errorf("sqlite: pragma scan: %w", err)
		}
		cols = append(cols, dbdriver.ColumnInfo{
			Name:     name,
			Type:     dataType,
			Nullable: notNull == 0,
			PK:       pk > 0 || pkSet[name],
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: pragma rows: %w", err)
	}
	return cols, nil
}

func (c *Conn) getPrimaryKeys(ctx context.Context) (map[string]map[string]bool, error) {
	rows, err := c.db.QueryContext(ctx, `
		SELECT m.name, il.name
		FROM sqlite_master m
		JOIN pragma_index_list(m.name) il
		WHERE il.origin = 'c'
	`)
	if err != nil {
		return nil, nil
	}
	defer rows.Close()

	result := make(map[string]map[string]bool)
	for rows.Next() {
		var table, indexName string
		if err := rows.Scan(&table, &indexName); err != nil {
			return nil, fmt.Errorf("sqlite: index list scan: %w", err)
		}
		colRows, err := c.db.QueryContext(ctx, fmt.Sprintf("PRAGMA index_xinfo(%s)", quoteIdent(indexName)))
		if err != nil {
			continue
		}
		for colRows.Next() {
			var rank, cid int
			var name *string
			var seqno int
			var collation *string
			var desc, cond int
			if err := colRows.Scan(&rank, &seqno, &cid, &name, &collation, &desc, &cond); err != nil {
				colRows.Close()
				continue
			}
			if name != nil && *name != "" {
				if result[table] == nil {
					result[table] = make(map[string]bool)
				}
				result[table][*name] = true
			}
		}
		colRows.Close()
	}
	return result, nil
}

func (c *Conn) Close() error {
	return c.db.Close()
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func sanitizeValue(v any) any {
	switch val := v.(type) {
	case []byte:
		return string(val)
	default:
		return val
	}
}
