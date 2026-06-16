package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"monika/internal/tool"
	"monika/pkg/dbdriver"
)

type DBQuerier interface {
	Schema(ctx context.Context, connName, filter string) (*dbdriver.SchemaResult, error)
	Query(ctx context.Context, connName, query string) (*dbdriver.QueryResult, error)
	DefaultConnection() string
	ListConnectionNames() []string
}

type dbSchema struct {
	dbMgr DBQuerier
}

func NewDBSchema(dbMgr DBQuerier) tool.Tool {
	return &dbSchema{dbMgr: dbMgr}
}

func (d *dbSchema) Name() string { return "db_schema" }
func (d *dbSchema) Description() string {
	return "Query the database schema to discover tables, columns, types, and primary keys. Use this before writing SQL queries with db_query to understand the available structure."
}
func (d *dbSchema) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"connection": map[string]any{
				"type":        "string",
				"description": "The database connection name. If omitted, uses the default connection.",
			},
			"filter": map[string]any{
				"type":        "string",
				"description": "Filter tables by name (substring match). If omitted, returns all tables.",
			},
		},
	}
}

func (d *dbSchema) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	var params struct {
		Connection string `json:"connection"`
		Filter     string `json:"filter"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	connName := params.Connection
	if connName == "" {
		connName = d.dbMgr.DefaultConnection()
	}
	if connName == "" {
		names := d.dbMgr.ListConnectionNames()
		if len(names) == 0 {
			return tool.ExecutionResult{Content: "no database connections available", IsError: true}, nil
		}
		return tool.ExecutionResult{
			Content: fmt.Sprintf("no default connection. available: %s", strings.Join(names, ", ")),
			IsError: true,
		}, nil
	}

	sr, err := d.dbMgr.Schema(ctx, connName, params.Filter)
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	if len(sr.Tables) == 0 {
		return tool.ExecutionResult{Content: "no tables found"}, nil
	}

	var buf strings.Builder
	for _, t := range sr.Tables {
		fmt.Fprintf(&buf, "### %s\n", t.Name)
		for _, c := range t.Columns {
			pk := ""
			if c.PK {
				pk = " [PK]"
			}
			null := ""
			if c.Nullable {
				null = "?"
			}
			fmt.Fprintf(&buf, "  - %s %s%s%s\n", c.Name, c.Type, null, pk)
		}
	}
	return tool.ExecutionResult{Content: strings.TrimRight(buf.String(), "\n")}, nil
}
