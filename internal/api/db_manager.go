package api

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"monika/internal/dbbridge"
	"monika/internal/dbdiscovery"
	"monika/pkg/dbdriver"
)

type DBManager struct {
	mu         sync.RWMutex
	projectDir string
	conns      map[string]*managedConn
	bridge     *dbbridge.BridgeManager
	runtime    string

	schemaCache string
	schemaMu    sync.RWMutex
}

type managedConn struct {
	info    dbdiscovery.DiscoveredDB
	dbConn  dbdriver.Connection
	useBridge bool
	ready   bool
	lastErr error
}

type ConnectionInfo struct {
	Name   string `json:"name"`
	Driver string `json:"driver"`
	Source string `json:"source"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func NewDBManager(projectDir string) *DBManager {
	return &DBManager{
		projectDir: projectDir,
		conns:      make(map[string]*managedConn),
	}
}

func (m *DBManager) Init(cache *dbdiscovery.CacheFile) {
	if cache == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.runtime = cache.Runtime

	for i := range cache.Connections {
		entry := cache.Connections[i]
		name := entry.Name
		if name == "" {
			name = entry.Driver
		}

		mc := &managedConn{
			info: entry,
		}

		switch m.runtime {
		case "node", "python":
			mc.useBridge = true
		default:
			drv, err := dbdriver.DriverByID(entry.Driver)
			if err != nil {
				mc.lastErr = err
				mc.ready = false
			} else {
				dsn := m.resolveDSN(entry.DSN)
				conn, err := drv.Open(dsn)
				if err != nil {
					mc.lastErr = err
				} else {
					mc.dbConn = conn
					mc.ready = true
				}
			}
		}

		m.conns[name] = mc
	}

	needsBridge := false
	for _, mc := range m.conns {
		if mc.useBridge {
			needsBridge = true
			break
		}
	}

	if needsBridge {
		m.bridge = dbbridge.NewBridgeManager()
		m.bridge.OnRestart = m.resetConnectionStates
		ctx := context.Background()
		if err := m.bridge.Start(ctx, m.projectDir, m.runtime); err != nil {
			for _, mc := range m.conns {
				if mc.useBridge {
					mc.lastErr = fmt.Errorf("bridge start: %w", err)
				}
			}
		}
	}
}

func (m *DBManager) Query(ctx context.Context, connName, query string) (*dbdriver.QueryResult, error) {
	mc, err := m.getConnection(ctx, connName)
	if err != nil {
		return nil, err
	}

	if mc.useBridge {
		return m.queryBridge(mc, query)
	}
	return mc.dbConn.Query(ctx, query)
}

func (m *DBManager) Schema(ctx context.Context, connName, filter string) (*dbdriver.SchemaResult, error) {
	mc, err := m.getConnection(ctx, connName)
	if err != nil {
		return nil, err
	}

	if mc.useBridge {
		return m.schemaBridge(mc, filter)
	}
	return mc.dbConn.Schema(ctx, filter)
}

func (m *DBManager) SchemaSummary() string {
	m.schemaMu.RLock()
	if m.schemaCache != "" {
		m.schemaMu.RUnlock()
		return m.schemaCache
	}
	m.schemaMu.RUnlock()

	var summary strings.Builder
	m.schemaMu.Lock()
	if m.schemaCache != "" {
		m.schemaMu.Unlock()
		return m.schemaCache
	}
	m.schemaMu.Unlock()

	m.mu.RLock()
	names := make([]string, 0, len(m.conns))
	for name := range m.conns {
		names = append(names, name)
	}
	conns := make(map[string]*managedConn, len(m.conns))
	for k, v := range m.conns {
		conns[k] = v
	}
	m.mu.RUnlock()

	for _, name := range names {
		mc := conns[name]
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		sr, err := m.Schema(ctx, name, "")
		cancel()
		if err != nil {
			fmt.Fprintf(&summary, "## %s (%s)\n  Error: %v\n\n", name, mc.info.Driver, err)
			continue
		}

		fmt.Fprintf(&summary, "## %s (%s)\n", name, mc.info.Driver)
		for _, t := range sr.Tables {
			fmt.Fprintf(&summary, "### %s\n", t.Name)
			for _, c := range t.Columns {
				pk := ""
				if c.PK {
					pk = " [PK]"
				}
				null := ""
				if c.Nullable {
					null = "?"
				}
				fmt.Fprintf(&summary, "  - %s %s%s%s\n", c.Name, c.Type, null, pk)
			}
		}
		summary.WriteString("\n")
	}

	result := summary.String()

	m.schemaMu.Lock()
	m.schemaCache = result
	m.schemaMu.Unlock()

	return result
}

func (m *DBManager) ListConnections() []ConnectionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]ConnectionInfo, 0, len(m.conns))
	for name, mc := range m.conns {
		status := "available"
		errMsg := ""

		if mc.lastErr != nil {
			status = "error"
			errMsg = mc.lastErr.Error()
		} else if mc.ready {
			status = "connected"
		}

		if mc.useBridge && m.bridge != nil && !m.bridge.IsRunning() && mc.lastErr == nil {
			status = "unavailable"
		}

		result = append(result, ConnectionInfo{
			Name:   name,
			Driver: mc.info.Driver,
			Source: mc.info.Source,
			Status: status,
			Error:  errMsg,
		})
	}
	return result
}

func (m *DBManager) TestConnection(ctx context.Context, connName string) error {
	mc, err := m.getConnection(ctx, connName)
	if err != nil {
		return err
	}
	if mc.useBridge {
		if m.bridge == nil || !m.bridge.IsRunning() {
			return fmt.Errorf("dbmanager: bridge not running for connection %q", connName)
		}
		return nil
	}
	if mc.dbConn == nil {
		return fmt.Errorf("dbmanager: connection %q has no underlying connection", connName)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	switch mc.info.Driver {
	case "redis":
		_, err = mc.dbConn.Query(ctx, "PING")
	case "mongo":
		_, err = mc.dbConn.Query(ctx, `{"ping": 1}`)
	default:
		_, err = mc.dbConn.Query(ctx, "SELECT 1")
	}
	return err
}

func (m *DBManager) DefaultConnection() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.conns) == 0 {
		return ""
	}
	names := make([]string, 0, len(m.conns))
	for name := range m.conns {
		names = append(names, name)
	}
	sort.Strings(names)
	return names[0]
}

func (m *DBManager) ListConnectionNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.conns))
	for name := range m.conns {
		names = append(names, name)
	}
	return names
}

func (m *DBManager) resetConnectionStates() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, mc := range m.conns {
		mc.ready = false
		mc.dbConn = nil
		mc.lastErr = nil
	}

	m.schemaMu.Lock()
	m.schemaCache = ""
	m.schemaMu.Unlock()
}

func (m *DBManager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, mc := range m.conns {
		if mc.dbConn != nil {
			mc.dbConn.Close()
			mc.ready = false
		}
		delete(m.conns, name)
	}

	if m.bridge != nil {
		m.bridge.Stop()
		m.bridge = nil
	}

	m.schemaMu.Lock()
	m.schemaCache = ""
	m.schemaMu.Unlock()
}

func (m *DBManager) getConnection(ctx context.Context, connName string) (*managedConn, error) {
	m.mu.RLock()
	mc, ok := m.conns[connName]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("dbmanager: unknown connection %q", connName)
	}

	if mc.ready {
		return mc, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if mc.ready {
		return mc, nil
	}

	if !mc.useBridge {
		if mc.lastErr != nil {
			return nil, fmt.Errorf("dbmanager: connection %q unavailable: %w", connName, mc.lastErr)
		}

		drv, err := dbdriver.DriverByID(mc.info.Driver)
		if err != nil {
			return nil, fmt.Errorf("dbmanager: driver %q not found: %w", mc.info.Driver, err)
		}

		dsn := m.resolveDSN(mc.info.DSN)
		conn, err := drv.Open(dsn)
		if err != nil {
			mc.lastErr = err
			return nil, fmt.Errorf("dbmanager: connect %q: %w", connName, err)
		}

		mc.dbConn = conn
		mc.ready = true
		return mc, nil
	}

	if m.bridge == nil || !m.bridge.IsRunning() {
		return nil, fmt.Errorf("dbmanager: bridge not running for connection %q", connName)
	}

	resp, err := m.bridge.Send(dbbridge.Request{
		ID:     m.bridge.NextID(),
		Action: "open",
		Driver: mc.info.Driver,
		DSN:    m.resolveDSN(mc.info.DSN),
		Conn:   connName,
	})
	if err != nil {
		mc.lastErr = err
		return nil, fmt.Errorf("dbmanager: bridge connect %q: %w", connName, err)
	}
	if resp.Status != "ok" {
		mc.lastErr = fmt.Errorf("bridge: %s", resp.Error)
		return nil, mc.lastErr
	}

	mc.ready = true
	return mc, nil
}

func (m *DBManager) queryBridge(mc *managedConn, query string) (*dbdriver.QueryResult, error) {
	if m.bridge == nil {
		return nil, fmt.Errorf("dbmanager: bridge not initialized")
	}

	resp, err := m.bridge.Send(dbbridge.Request{
		ID:     m.bridge.NextID(),
		Action: "query",
		Conn:   mc.info.Name,
		Query:  query,
	})
	if err != nil {
		return nil, fmt.Errorf("dbmanager: bridge query: %w", err)
	}
	if resp.Status != "ok" {
		return nil, fmt.Errorf("bridge: %s", resp.Error)
	}

	return &dbdriver.QueryResult{
		Columns: resp.Columns,
		Rows:    resp.Rows,
		Tag:     resp.Tag,
	}, nil
}

func (m *DBManager) schemaBridge(mc *managedConn, filter string) (*dbdriver.SchemaResult, error) {
	if m.bridge == nil {
		return nil, fmt.Errorf("dbmanager: bridge not initialized")
	}

	resp, err := m.bridge.Send(dbbridge.Request{
		ID:     m.bridge.NextID(),
		Action: "schema",
		Conn:   mc.info.Name,
		Filter: filter,
	})
	if err != nil {
		return nil, fmt.Errorf("dbmanager: bridge schema: %w", err)
	}
	if resp.Status != "ok" {
		return nil, fmt.Errorf("bridge: %s", resp.Error)
	}

	tables := make([]dbdriver.TableInfo, len(resp.Tables))
	for i, t := range resp.Tables {
		cols := make([]dbdriver.ColumnInfo, len(t.Columns))
		for j, c := range t.Columns {
			cols[j] = dbdriver.ColumnInfo{
				Name:     c.Name,
				Type:     c.Type,
				Nullable: c.Nullable,
				PK:       c.PK,
			}
		}
		tables[i] = dbdriver.TableInfo{
			Name:    t.Name,
			Columns: cols,
		}
	}

	return &dbdriver.SchemaResult{Tables: tables}, nil
}

func (m *DBManager) resolveDSN(dsn string) string {
	if dsn == "" {
		return dsn
	}
	if strings.Contains(dsn, "://") || strings.Contains(dsn, "@") || strings.HasPrefix(dsn, "/") {
		return dsn
	}
	if strings.Contains(dsn, "=") {
		return dsn
	}
	return filepath.Join(m.projectDir, dsn)
}

func (m *DBManager) Reset(cache *dbdiscovery.CacheFile) {
	m.CloseAll()

	m.mu.Lock()
	m.conns = make(map[string]*managedConn)
	m.mu.Unlock()

	m.Init(cache)
}
