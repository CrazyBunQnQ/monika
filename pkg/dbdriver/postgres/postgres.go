package postgres

import (
	"context"
	"fmt"
	"strings"

	"monika/pkg/dbdriver"

	"github.com/jackc/pgx/v5/pgxpool"
)

func init() {
	dbdriver.Register(&Driver{})
}

type Driver struct{}

func (d *Driver) ID() string { return "postgres" }

func (d *Driver) Open(dsn string) (dbdriver.Connection, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse config: %w", err)
	}
	config.MinConns = 1
	config.MaxConns = 3

	ctx := context.Background()
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("postgres: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}
	return &Conn{pool: pool}, nil
}

type Conn struct {
	pool *pgxpool.Pool
}

func (c *Conn) Query(ctx context.Context, query string) (*dbdriver.QueryResult, error) {
	if err := dbdriver.ValidateReadOnlySQL(query); err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}
	rows, err := c.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgres: query: %w", err)
	}
	defer rows.Close()

	fds := rows.FieldDescriptions()
	cols := make([]string, len(fds))
	for i, fd := range fds {
		cols[i] = string(fd.Name)
	}

	var allRows [][]any
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("postgres: scan: %w", err)
		}
		allRows = append(allRows, vals)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: rows: %w", err)
	}

	return &dbdriver.QueryResult{
		Columns: cols,
		Rows:    allRows,
		Tag:     rows.CommandTag().String(),
	}, nil
}

func (c *Conn) Schema(ctx context.Context, filter string) (*dbdriver.SchemaResult, error) {
	q := `
		SELECT c.table_name, c.column_name, c.data_type,
		       c.is_nullable,
		       CASE WHEN pk.column_name IS NOT NULL THEN true ELSE false END AS is_pk
		FROM information_schema.columns c
		LEFT JOIN (
			SELECT kcu.table_schema, kcu.table_name, kcu.column_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu
				ON tc.constraint_name = kcu.constraint_name
				AND tc.table_schema = kcu.table_schema
			WHERE tc.constraint_type = 'PRIMARY KEY'
		) pk ON pk.table_schema = c.table_schema
			AND pk.table_name = c.table_name
			AND pk.column_name = c.column_name
		WHERE c.table_schema = 'public'
	`
	args := []any{}
	if filter != "" {
		q += ` AND c.table_name LIKE ` + fmt.Sprintf("$%d", len(args)+1)
		args = append(args, filter)
	}
	q += ` ORDER BY c.table_name, c.ordinal_position`

	rows, err := c.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres: schema: %w", err)
	}
	defer rows.Close()

	tableMap := make(map[string]*dbdriver.TableInfo)
	var tableOrder []string

	for rows.Next() {
		var tableName, colName, dataType, isNullable string
		var isPK bool
		if err := rows.Scan(&tableName, &colName, &dataType, &isNullable, &isPK); err != nil {
			return nil, fmt.Errorf("postgres: schema scan: %w", err)
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
			PK:       isPK,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: schema rows: %w", err)
	}

	tables := make([]dbdriver.TableInfo, 0, len(tableOrder))
	for _, name := range tableOrder {
		tables = append(tables, *tableMap[name])
	}

	return &dbdriver.SchemaResult{Tables: tables}, nil
}

func (c *Conn) Close() error {
	c.pool.Close()
	return nil
}
