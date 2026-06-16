package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"monika/internal/tool"
)

type dbQuery struct {
	dbMgr DBQuerier
}

func NewDBQuery(dbMgr DBQuerier) tool.Tool {
	return &dbQuery{dbMgr: dbMgr}
}

func (d *dbQuery) Name() string { return "db_query" }
func (d *dbQuery) Description() string {
	return "Execute a SQL query on a database connection and return results as a formatted table. Use db_schema first to understand the available tables and columns."
}
func (d *dbQuery) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"connection": map[string]any{
				"type":        "string",
				"description": "The database connection name. If omitted, uses the default connection.",
			},
			"query": map[string]any{
				"type":        "string",
				"description": "The SQL query to execute.",
			},
		},
		"required": []string{"query"},
	}
}

func (d *dbQuery) Execute(ctx context.Context, args json.RawMessage) (tool.ExecutionResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var params struct {
		Connection string `json:"connection"`
		Query      string `json:"query"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	if params.Query == "" {
		return tool.ExecutionResult{Content: "query is required", IsError: true}, nil
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

	qr, err := d.dbMgr.Query(ctx, connName, params.Query)
	if err != nil {
		return tool.ExecutionResult{Content: err.Error(), IsError: true}, nil
	}

	if len(qr.Rows) == 0 {
		return tool.ExecutionResult{Content: fmt.Sprintf("%s (0 rows)", qr.Tag)}, nil
	}

	colWidths := make([]int, len(qr.Columns))
	for i, col := range qr.Columns {
		colWidths[i] = utf8.RuneCountInString(col)
	}
	for _, row := range qr.Rows {
		for i, val := range row {
			s := fmt.Sprintf("%v", val)
			w := utf8.RuneCountInString(s)
			if w > colWidths[i] {
				colWidths[i] = w
			}
		}
	}

	pad := func(s string, w int) string {
		rw := utf8.RuneCountInString(s)
		if rw >= w {
			return s
		}
		return s + strings.Repeat(" ", w-rw)
	}

	var buf strings.Builder
	parts := make([]string, len(qr.Columns))
	for i, col := range qr.Columns {
		parts[i] = pad(col, colWidths[i])
	}
	buf.WriteString(strings.Join(parts, " | "))
	buf.WriteString("\n")

	sep := make([]string, len(qr.Columns))
	for i, w := range colWidths {
		sep[i] = strings.Repeat("-", w)
	}
	buf.WriteString(strings.Join(sep, "-+-"))
	buf.WriteString("\n")

	for _, row := range qr.Rows {
		for i, val := range row {
			parts[i] = pad(fmt.Sprintf("%v", val), colWidths[i])
		}
		buf.WriteString(strings.Join(parts, " | "))
		buf.WriteString("\n")
	}

	buf.WriteString(fmt.Sprintf("\n%s (%d rows)", qr.Tag, len(qr.Rows)))
	return tool.ExecutionResult{Content: buf.String()}, nil
}
