package dbbridge

type Request struct {
	ID     int    `json:"id"`
	Action string `json:"action"`
	Driver string `json:"driver,omitempty"`
	DSN    string `json:"dsn,omitempty"`
	Conn   string `json:"conn,omitempty"`
	Query  string `json:"query,omitempty"`
	Filter string `json:"filter,omitempty"`
}

type Response struct {
	ID      int           `json:"id"`
	Status  string        `json:"status"`
	Error   string        `json:"error,omitempty"`
	Conn    string        `json:"conn,omitempty"`
	Columns []string      `json:"columns,omitempty"`
	Rows    [][]any       `json:"rows,omitempty"`
	Tag     string        `json:"tag,omitempty"`
	Tables  []TableSchema `json:"tables,omitempty"`
}

type TableSchema struct {
	Name    string         `json:"name"`
	Columns []ColumnSchema `json:"columns"`
}

type ColumnSchema struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	PK       bool   `json:"pk"`
}
