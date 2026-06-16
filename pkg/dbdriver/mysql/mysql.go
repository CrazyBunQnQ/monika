package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"monika/pkg/dbdriver"

	_ "github.com/go-sql-driver/mysql"
)

func init() {
	dbdriver.Register(&Driver{})
}

type Driver struct{}

func (d *Driver) ID() string { return "mysql" }

func (d *Driver) Open(dsn string) (dbdriver.Connection, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("mysql: open: %w", err)
	}
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("mysql: ping: %w", err)
	}
	return &Conn{db: db}, nil
}

type Conn struct {
	db *sql.DB
}

func (c *Conn) Query(ctx context.Context, query string) (*dbdriver.QueryResult, error) {
	if err := dbdriver.ValidateReadOnlySQL(query); err != nil {
		return nil, fmt.Errorf("mysql: %w", err)
	}
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("mysql: query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("mysql: columns: %w", err)
	}

	var allRows [][]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("mysql: scan: %w", err)
		}
		row := make([]any, len(cols))
		for i, v := range vals {
			row[i] = sanitizeValue(v)
		}
		allRows = append(allRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql: rows: %w", err)
	}

	return &dbdriver.QueryResult{
		Columns: cols,
		Rows:    allRows,
		Tag:     "SELECT",
	}, nil
}

func (c *Conn) Schema(ctx context.Context, filter string) (*dbdriver.SchemaResult, error) {
	q := `
		SELECT c.TABLE_NAME, c.COLUMN_NAME, c.DATA_TYPE,
		       c.IS_NULLABLE,
		       CASE WHEN pk.COLUMN_NAME IS NOT NULL THEN '1' ELSE '0' END AS IS_PK
		FROM information_schema.COLUMNS c
		LEFT JOIN (
			SELECT kcu.TABLE_SCHEMA, kcu.TABLE_NAME, kcu.COLUMN_NAME
			FROM information_schema.TABLE_CONSTRAINTS tc
			JOIN information_schema.KEY_COLUMN_USAGE kcu
				ON tc.CONSTRAINT_NAME = kcu.CONSTRAINT_NAME
				AND tc.TABLE_SCHEMA = kcu.TABLE_SCHEMA
			WHERE tc.CONSTRAINT_TYPE = 'PRIMARY KEY'
		) pk ON pk.TABLE_SCHEMA = c.TABLE_SCHEMA
			AND pk.TABLE_NAME = c.TABLE_NAME
			AND pk.COLUMN_NAME = c.COLUMN_NAME
		WHERE c.TABLE_SCHEMA = DATABASE()
	`
	args := []any{}
	if filter != "" {
		q += ` AND c.TABLE_NAME LIKE ?`
		args = append(args, filter)
	}
	q += ` ORDER BY c.TABLE_NAME, c.ORDINAL_POSITION`

	rows, err := c.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("mysql: schema: %w", err)
	}
	defer rows.Close()

	tableMap := make(map[string]*dbdriver.TableInfo)
	var tableOrder []string

	for rows.Next() {
		var tableName, colName, dataType, isNullable, isPK string
		if err := rows.Scan(&tableName, &colName, &dataType, &isNullable, &isPK); err != nil {
			return nil, fmt.Errorf("mysql: schema scan: %w", err)
		}
		t, ok := tableMap[tableName]
		if !ok {
			t = &dbdriver.TableInfo{Name: tableName}
			tableMap[tableName] = t
			tableOrder = append(tableOrder, tableName)
		}
		t.Columns = append(t.Columns, dbdriver.ColumnInfo{
			Name:     colName,
			Type:     dataType,
			Nullable: strings.EqualFold(isNullable, "YES"),
			PK:       isPK == "1",
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql: schema rows: %w", err)
	}

	tables := make([]dbdriver.TableInfo, 0, len(tableOrder))
	for _, name := range tableOrder {
		tables = append(tables, *tableMap[name])
	}

	return &dbdriver.SchemaResult{Tables: tables}, nil
}

func (c *Conn) Close() error {
	return c.db.Close()
}

func sanitizeValue(v any) any {
	switch val := v.(type) {
	case []byte:
		return string(val)
	default:
		return val
	}
}
