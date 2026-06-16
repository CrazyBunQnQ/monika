package dbdriver

import (
	"context"
	"fmt"
	"sync"
)

type Driver interface {
	ID() string
	Open(dsn string) (Connection, error)
}

type Connection interface {
	Query(ctx context.Context, query string) (*QueryResult, error)
	Schema(ctx context.Context, filter string) (*SchemaResult, error)
	Close() error
}

type QueryResult struct {
	Columns []string `json:"columns"`
	Rows    [][]any  `json:"rows"`
	Tag     string   `json:"tag"`
}

type SchemaResult struct {
	Tables []TableInfo `json:"tables"`
}

type TableInfo struct {
	Name    string       `json:"name"`
	Columns []ColumnInfo `json:"columns"`
}

type ColumnInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	PK       bool   `json:"pk"`
}

var (
	driversMu sync.RWMutex
	drivers   = make(map[string]Driver)
)

func Register(d Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	drivers[d.ID()] = d
}

func DriverByID(id string) (Driver, error) {
	driversMu.RLock()
	defer driversMu.RUnlock()
	d, ok := drivers[id]
	if !ok {
		return nil, fmt.Errorf("dbdriver: unknown driver %q", id)
	}
	return d, nil
}
