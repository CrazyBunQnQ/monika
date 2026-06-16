# Database Auto-Discovery & Query — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enable Monika's Agent to automatically discover project databases and query their schemas/data during coding assistance.

**Architecture:** Discovery layer scans project config files (.env, docker-compose.yml, etc.) to find database connections. A bridge process uses the project's own runtime (Node.js/Python/Ruby) and drivers to execute read-only queries. Go native drivers serve as fallback for pure Go projects. Two builtin tools (`db_schema`, `db_query`) expose this to the Agent, with schema summaries injected into the system prompt.

**Tech Stack:** Go 1.25+, Node.js bridge scripts, Python bridge scripts, pgx/go-redis/mongo-driver/modernc-sqlite/go-sql-driver for Go fallback

**Spec:** `docs/superpowers/specs/2026-06-10-database-auto-discovery-design.md`

---

## File Structure

### New Files

```
internal/dbdiscovery/
├── discover.go              # Discoverer interface, Registry, Scan()
├── env.go                   # .env parser
├── docker.go                # docker-compose.yml parser
├── runtime.go               # runtime detection

internal/dbbridge/
├── bridge.go                # BridgeManager: process lifecycle + JSON IPC
├── protocol.go              # request/response types
└── scripts/
    ├── node.js              # Node.js bridge script
    └── python.py            # Python bridge script

pkg/dbdriver/
├── driver.go                # Driver/Connection interfaces + Registry
├── postgres/postgres.go     # pgx driver
├── mysql/mysql.go           # go-sql-driver
├── sqlite/sqlite.go         # modernc.org/sqlite
├── mongo/mongo.go           # mongo-driver
└── redis/redis.go           # go-redis

internal/api/db_manager.go   # DBManager: ties discovery + bridge + drivers
internal/tool/builtin/
├── db_schema.go             # db_schema tool
└── db_query.go              # db_query tool
```

### Modified Files

```
internal/tool/builtin/register.go   # add RegisterDatabase()
internal/permission/pipeline.go     # add db_schema/db_query to readOps
internal/prompt/default.go          # add database prompt section
main.go                             # wire DBManager + imports
```

---

## Task 1: pkg/dbdriver — Driver Interface & Registry

**Files:**
- Create: `pkg/dbdriver/driver.go`

This is the foundation. All other modules depend on these types.

- [ ] **Step 1: Create driver.go with interfaces and registry**

```go
package dbdriver

import (
	"context"
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
```

- [ ] **Step 2: Run go vet to verify**

Run: `go vet ./pkg/dbdriver/...`
Expected: no errors (note: needs `import "fmt"` added)

- [ ] **Step 3: Commit**

```bash
git add pkg/dbdriver/driver.go
git commit -m "feat(db): add dbdriver interface and registry"
```

---

## Task 2: pkg/dbdriver — PostgreSQL Driver

**Files:**
- Create: `pkg/dbdriver/postgres/postgres.go`

- [ ] **Step 1: Install pgx dependency**

Run: `go get github.com/jackc/pgx/v5`

- [ ] **Step 2: Create postgres driver**

```go
package postgres

import (
	"context"
	"fmt"
	"strings"

	"monika/pkg/dbdriver"

	"github.com/jackc/pgx/v5/pgxpool"
)

func init() {
	dbdriver.Register(&PostgresDriver{})
}

type PostgresDriver struct{}

func (d *PostgresDriver) ID() string { return "postgres" }

func (d *PostgresDriver) Open(dsn string) (dbdriver.Connection, error) {
	ctx := context.Background()
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse config: %w", err)
	}
	poolCfg.MinConns = 1
	poolCfg.MaxConns = 3
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}
	return &PgConn{pool: pool}, nil
}

type PgConn struct {
	pool *pgxpool.Pool
}

func (c *PgConn) Query(ctx context.Context, query string) (*dbdriver.QueryResult, error) {
	rows, err := c.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols := make([]string, len(rows.FieldDescriptions()))
	for i, fd := range rows.FieldDescriptions() {
		cols[i] = fd.Name
	}

	var resultRows [][]any
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		resultRows = append(resultRows, vals)
	}

	tag := ""
	if rows.CommandTag().RowsAffected() > 0 {
		tag = fmt.Sprintf("SELECT %d", rows.CommandTag().RowsAffected())
	}

	return &dbdriver.QueryResult{Columns: cols, Rows: resultRows, Tag: tag}, nil
}

func (c *PgConn) Schema(ctx context.Context, filter string) (*dbdriver.SchemaResult, error) {
	q := `SELECT table_name, column_name, data_type, is_nullable,
	      CASE WHEN pk.column_name IS NOT NULL THEN true ELSE false END as is_pk
	      FROM information_schema.columns c
	      LEFT JOIN (
	          SELECT ku.column_name, ku.table_name
	          FROM information_schema.table_constraints tc
	          JOIN information_schema.key_column_usage ku
	            ON tc.constraint_name = ku.constraint_name AND tc.table_name = ku.table_name
	          WHERE tc.constraint_type = 'PRIMARY KEY'
	      ) pk ON c.column_name = pk.column_name AND c.table_name = pk.table_name
	      WHERE c.table_schema = 'public'`
	args := []any{}
	if filter != "" {
		q += ` AND c.table_name LIKE $1`
		args = append(args, filter)
	}
	q += ` ORDER BY c.table_name, c.ordinal_position`

	rows, err := c.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tableMap := map[string][]dbdriver.ColumnInfo{}
	for rows.Next() {
		var tableName, colName, dataType, isNullable string
		var isPK bool
		if err := rows.Scan(&tableName, &colName, &dataType, &isNullable, &isPK); err != nil {
			return nil, err
		}
		tableMap[tableName] = append(tableMap[tableName], dbdriver.ColumnInfo{
			Name:     colName,
			Type:     dataType,
			Nullable: strings.EqualFold(isNullable, "YES"),
			PK:       isPK,
		})
	}

	var tables []dbdriver.TableInfo
	for name, cols := range tableMap {
		tables = append(tables, dbdriver.TableInfo{Name: name, Columns: cols})
	}
	return &dbdriver.SchemaResult{Tables: tables}, nil
}

func (c *PgConn) Close() error {
	c.pool.Close()
	return nil
}
```

- [ ] **Step 3: Run go vet**

Run: `go vet ./pkg/dbdriver/postgres/...`

- [ ] **Step 4: Commit**

```bash
git add pkg/dbdriver/postgres/
git commit -m "feat(db): add PostgreSQL driver via pgx"
```

---

## Task 3: pkg/dbdriver — Remaining Drivers (MySQL, SQLite, Redis, Mongo)

**Files:**
- Create: `pkg/dbdriver/mysql/mysql.go`
- Create: `pkg/dbdriver/sqlite/sqlite.go`
- Create: `pkg/dbdriver/redis/redis.go`
- Create: `pkg/dbdriver/mongo/mongo.go`

Follow the same pattern as Task 2 for each driver. Each driver:
1. `go get` the dependency
2. Implement `Driver` interface with `init()` self-registration
3. Implement `Connection` interface with `Query()`, `Schema()`, `Close()`
4. `go vet` to verify
5. Commit

- [ ] **Step 1: MySQL driver** — `go get github.com/go-sql-driver/mysql`, implement using `database/sql`

- [ ] **Step 2: SQLite driver** — `go get modernc.org/sqlite`, implement using `database/sql`

- [ ] **Step 3: Redis driver** — `go get github.com/redis/go-redis/v9`, implement with `INFO`, `KEYS`, `TYPE`, `HGETALL` etc. for schema; `GET`/`LRANGE` etc. for query

- [ ] **Step 4: MongoDB driver** — `go get go.mongodb.org/mongo-driver/v2/mongo`, implement with `ListCollections` for schema; `Find` for query

- [ ] **Step 5: Commit all drivers**

```bash
git add pkg/dbdriver/
git commit -m "feat(db): add MySQL, SQLite, Redis, MongoDB drivers"
```

---

## Task 4: dbdiscovery — Auto-Discovery Framework

**Files:**
- Create: `internal/dbdiscovery/discover.go`
- Create: `internal/dbdiscovery/runtime.go`

- [ ] **Step 1: Create discover.go with Discoverer interface and Scan logic**

```go
package dbdiscovery

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Discoverer interface {
	Name() string
	Scan(projectDir string) ([]DiscoveredDB, error)
}

type DiscoveredDB struct {
	Name        string `json:"name"`
	Driver      string `json:"driver"`
	DSN         string `json:"dsn"`
	Source      string `json:"source"`
	RuntimeHint string `json:"runtime_hint,omitempty"`
}

type CacheFile struct {
	ScannedAt   time.Time     `json:"scanned_at"`
	Runtime     string        `json:"runtime"`
	Connections []DiscoveredDB `json:"connections"`
}

var (
	discMu       sync.RWMutex
	discoverers  []Discoverer
)

func RegisterDiscoverer(d Discoverer) {
	discMu.Lock()
	defer discMu.Unlock()
	discoverers = append(discoverers, d)
}

func Scan(projectDir string) (*CacheFile, error) {
	runtime := DetectRuntime(projectDir)

	var all []DiscoveredDB
	discMu.RLock()
	defer discMu.RUnlock()
	for _, d := range discoverers {
		results, err := d.Scan(projectDir)
		if err != nil {
			continue
		}
		all = append(all, results...)
	}

	all = deduplicate(all)
	for i := range all {
		if all[i].RuntimeHint == "" {
			all[i].RuntimeHint = runtime
		}
	}

	cache := &CacheFile{
		ScannedAt:   time.Now(),
		Runtime:     runtime,
		Connections: all,
	}

	cachePath := filepath.Join(projectDir, ".monika", "databases.json")
	_ = os.MkdirAll(filepath.Dir(cachePath), 0755)
	data, _ := json.MarshalIndent(cache, "", "  ")
	_ = os.WriteFile(cachePath, data, 0600)

	return cache, nil
}

func LoadCache(projectDir string) (*CacheFile, error) {
	data, err := os.ReadFile(filepath.Join(projectDir, ".monika", "databases.json"))
	if err != nil {
		return nil, err
	}
	var cache CacheFile
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	return &cache, nil
}

func deduplicate(conns []DiscoveredDB) []DiscoveredDB {
	seen := map[string]bool{}
	var result []DiscoveredDB
	for _, c := range conns {
		key := c.Driver + ":" + c.DSN
		if !seen[key] {
			seen[key] = true
			result = append(result, c)
		}
	}
	return result
}
```

- [ ] **Step 2: Create runtime.go**

```go
package dbdiscovery

import (
	"os"
	"path/filepath"
)

func DetectRuntime(projectDir string) string {
	if fileExists(projectDir, "package.json") {
		return "node"
	}
	if fileExists(projectDir, "requirements.txt") || fileExists(projectDir, "pyproject.toml") || fileExists(projectDir, "Pipfile") {
		return "python"
	}
	if fileExists(projectDir, "Gemfile") {
		return "ruby"
	}
	return ""
}

func fileExists(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}
```

- [ ] **Step 3: Run go vet**

Run: `go vet ./internal/dbdiscovery/...`

- [ ] **Step 4: Commit**

```bash
git add internal/dbdiscovery/
git commit -m "feat(db): add auto-discovery framework and runtime detection"
```

---

## Task 5: dbdiscovery — .env and Docker Compose Discoverers

**Files:**
- Create: `internal/dbdiscovery/env.go`
- Create: `internal/dbdiscovery/docker.go`

- [ ] **Step 1: Create env.go — parse .env files for DATABASE_URL, REDIS_URL, MONGO_URL**

Parse standard env var patterns:
- `DATABASE_URL=postgresql://...` → driver "postgres"
- `DATABASE_URL=mysql://...` → driver "mysql"
- `DATABASE_URL=sqlite:///...` → driver "sqlite"
- `REDIS_URL=redis://...` → driver "redis"
- `MONGODB_URI=mongodb://...` → driver "mongo"
- Also check `REDIS_URL`, `MONGO_URL`, `DB_HOST`+`DB_PORT`+`DB_USER`+`DB_PASSWORD`+`DB_NAME` combinations

Implement the `Discoverer` interface, register via `init()`.

- [ ] **Step 2: Create docker.go — parse docker-compose.yml for database services**

Scan for services whose image contains `postgres`, `mysql`, `mongo`, `redis`, `sqlite`. Build DSN from `environment` vars (`POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, etc.) and `ports` mapping.

Implement the `Discoverer` interface, register via `init()`.

- [ ] **Step 3: Run go vet**

Run: `go vet ./internal/dbdiscovery/...`

- [ ] **Step 4: Commit**

```bash
git add internal/dbdiscovery/
git commit -m "feat(db): add .env and docker-compose discoverers"
```

---

## Task 6: dbbridge — Bridge Protocol & Manager

**Files:**
- Create: `internal/dbbridge/protocol.go`
- Create: `internal/dbbridge/bridge.go`

- [ ] **Step 1: Create protocol.go with request/response types**

```go
package dbbridge

type Request struct {
	ID     int    `json:"id"`
	Action string `json:"action"` // "open", "query", "schema", "close", "ping"
	Driver string `json:"driver,omitempty"`
	DSN    string `json:"dsn,omitempty"`
	Conn   string `json:"conn,omitempty"`
	Query  string `json:"query,omitempty"`
	Filter string `json:"filter,omitempty"`
}

type Response struct {
	ID       int           `json:"id"`
	Status   string        `json:"status"` // "ok" or "error"
	Error    string        `json:"error,omitempty"`
	Conn     string        `json:"conn,omitempty"`
	Columns  []string      `json:"columns,omitempty"`
	Rows     [][]any       `json:"rows,omitempty"`
	Tag      string        `json:"tag,omitempty"`
	Tables   []TableSchema `json:"tables,omitempty"`
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
```

- [ ] **Step 2: Create bridge.go with BridgeManager**

BridgeManager manages a long-lived bridge subprocess:
1. Detects runtime (node/python) and selects bridge script
2. Starts the subprocess with the project directory as CWD
3. Communicates via stdin/stdout JSON lines
4. Handles heartbeat, crash recovery (max 3 retries)
5. Embeds bridge scripts via `//go:embed scripts/*`

Key methods:
- `Start(ctx context.Context, projectDir, runtime string) error`
- `Send(req Request) (Response, error)` — sends request, waits for response with matching ID
- `Stop()` — sends SIGTERM, waits for exit
- `IsRunning() bool`

- [ ] **Step 3: Run go vet**

Run: `go vet ./internal/dbbridge/...`

- [ ] **Step 4: Commit**

```bash
git add internal/dbbridge/
git commit -m "feat(db): add bridge protocol and process manager"
```

---

## Task 7: dbbridge — Node.js Bridge Script

**Files:**
- Create: `internal/dbbridge/scripts/node.js`

- [ ] **Step 1: Write the Node.js bridge script (~80 lines)**

The script:
1. Reads JSON lines from stdin
2. On `open`: tries to `require()` the appropriate driver (`pg`, `mysql2`, `better-sqlite3`, `mongodb`, `ioredis`) and connects
3. On `query`: enforces read-only (only SELECT/SHOW/DESCRIBE/EXPLAIN allowed for SQL), executes, returns JSON result
4. On `schema`: queries `information_schema` (SQL) or equivalent for table/column info
5. On `close`: closes connection
6. On `ping`: responds with `{"status":"ok"}`

- [ ] **Step 2: Test manually**

Run: `echo '{"id":1,"action":"ping"}' | node internal/dbbridge/scripts/node.js`
Expected: `{"id":1,"status":"ok"}`

- [ ] **Step 3: Commit**

```bash
git add internal/dbbridge/scripts/node.js
git commit -m "feat(db): add Node.js bridge script"
```

---

## Task 8: dbbridge — Python Bridge Script

**Files:**
- Create: `internal/dbbridge/scripts/python.py`

- [ ] **Step 1: Write the Python bridge script (~80 lines)**

Same protocol as Node.js, using Python drivers: `psycopg2`/`pymysql`/`sqlite3`/`pymongo`/`redis`.

- [ ] **Step 2: Test manually**

Run: `echo '{"id":1,"action":"ping"}' | python3 internal/dbbridge/scripts/python.py`
Expected: `{"id":1,"status":"ok"}`

- [ ] **Step 3: Commit**

```bash
git add internal/dbbridge/scripts/python.py
git commit -m "feat(db): add Python bridge script"
```

---

## Task 9: DBManager — Central Coordinator

**Files:**
- Create: `internal/api/db_manager.go`

- [ ] **Step 1: Create db_manager.go**

DBManager is the central coordinator that:
1. Holds discovered connections from `dbdiscovery`
2. Manages bridge process and Go driver fallback
3. Routes `Query()` and `Schema()` calls to bridge or Go driver
4. Generates `SchemaSummary()` string for system prompt injection
5. Manages lazy connection lifecycle

Key struct:

```go
type DBManager struct {
    mu         sync.RWMutex
    projectDir string
    conns      map[string]*managedConn  // key = connection name
    bridge     *dbbridge.BridgeManager
    runtime    string
    schemaOnce sync.Once
    schemaCache string
}

type managedConn struct {
    db       dbdriver.Connection
    info     dbdiscovery.DiscoveredDB
    ready    bool
    lastErr  error
}
```

Methods:
- `NewDBManager(projectDir string) *DBManager`
- `Init(cache *dbdiscovery.CacheFile)` — stores entries, determines execution strategy per connection
- `Query(ctx context.Context, connName, query string) (*dbdriver.QueryResult, error)` — lazy connect, then delegate
- `Schema(ctx context.Context, connName, filter string) (*dbdriver.SchemaResult, error)` — lazy connect, then delegate
- `SchemaSummary() string` — one-time fetch, cached
- `ListConnections() []ConnectionInfo` — for Settings UI
- `TestConnection(ctx context.Context, connName string) error` — try connecting
- `CloseAll()` — close all connections + stop bridge

Connection strategy per entry:
- If runtime is "node"/"python"/"ruby" → use bridge
- If runtime is "" (Go project) → use `dbdriver.DriverByID(driver).Open(dsn)`
- If driver not found in Go registry → mark unavailable

- [ ] **Step 2: Run go vet**

Run: `go vet ./internal/api/...`

- [ ] **Step 3: Commit**

```bash
git add internal/api/db_manager.go
git commit -m "feat(db): add DBManager central coordinator"
```

---

## Task 10: Builtin Tools — db_schema and db_query

**Files:**
- Create: `internal/tool/builtin/db_schema.go`
- Create: `internal/tool/builtin/db_query.go`
- Modify: `internal/tool/builtin/register.go`

- [ ] **Step 1: Create db_schema.go**

Implement `tool.Tool` interface:
- `Name()` → `"db_schema"`
- `Description()` → explains schema querying
- `Parameters()` → `{ "connection": {"type":"string"}, "filter": {"type":"string"} }`
- `Execute()` → calls `dbMgr.Schema(ctx, connName, filter)`, formats result as readable text

The tool struct holds a reference to `*DBManager`.

- [ ] **Step 2: Create db_query.go**

Same pattern:
- `Name()` → `"db_query"`
- `Parameters()` → `{ "connection": {"type":"string"}, "query": {"type":"string"} }`
- `Execute()` → calls `dbMgr.Query(ctx, connName, query)`, formats as table

- [ ] **Step 3: Add RegisterDatabase to register.go**

```go
func RegisterDatabase(r *tool.ToolRegistry, dbMgr *DBManager) {
    if dbMgr == nil {
        return
    }
    r.Register(NewDBSchema(dbMgr))
    r.Register(NewDBQuery(dbMgr))
}
```

Note: `DBManager` type is from `internal/api`, so use an interface or concrete type depending on import structure. Prefer defining a small interface in the builtin package:

```go
type DBQuerier interface {
    Query(ctx context.Context, connName, query string) (*dbdriver.QueryResult, error)
    Schema(ctx context.Context, connName, filter string) (*dbdriver.SchemaResult, error)
    ListConnections() []string
}
```

- [ ] **Step 4: Run go vet**

Run: `go vet ./internal/tool/builtin/...`

- [ ] **Step 5: Commit**

```bash
git add internal/tool/builtin/db_schema.go internal/tool/builtin/db_query.go internal/tool/builtin/register.go
git commit -m "feat(db): add db_schema and db_query builtin tools"
```

---

## Task 11: Permission Pipeline — Classify db tools as read ops

**Files:**
- Modify: `internal/permission/pipeline.go`

- [ ] **Step 1: Add db_schema and db_query to readOps map**

In `pipeline.go`, add to the `readOps` map:

```go
var readOps = map[string]bool{
    "file_read": true,
    "grep":      true,
    "glob":      true,
    "file_list": true,
    "skill":     true,
    "ask_user":  true,
    "db_schema": true,
    "db_query":  true,
}
```

Both tools are read-only by design (enforced at bridge/driver level), so they auto-allow in Auto mode.

- [ ] **Step 2: Run go vet**

Run: `go vet ./internal/permission/...`

- [ ] **Step 3: Commit**

```bash
git add internal/permission/pipeline.go
git commit -m "feat(db): classify db_schema and db_query as read operations"
```

---

## Task 12: System Prompt — Inject Database Schema

**Files:**
- Modify: `internal/prompt/default.go` (add database usage instructions)
- Modify: `main.go` (wire DBManager + schema injection)

- [ ] **Step 1: Add database section to the tool usage prompt in default.go**

Add instructions for when/how to use `db_schema` and `db_query` in the `defaultToolUsage` prompt section.

- [ ] **Step 2: Wire DBManager in main.go**

1. After project opens, call `dbdiscovery.LoadCache()` or `dbdiscovery.Scan()`
2. Create `DBManager` with project dir
3. Call `dbMgr.Init(cache)` to store connections
4. Pass `dbMgr` to `RegisterDatabase()`
5. In system prompt assembly, call `dbMgr.SchemaSummary()` and append
6. Add `_ "monika/pkg/dbdriver/postgres"` etc. imports
7. In `ServiceShutdown`, call `dbMgr.CloseAll()`

- [ ] **Step 3: Run go vet**

Run: `go vet ./...`

- [ ] **Step 4: Commit**

```bash
git add main.go internal/prompt/default.go
git commit -m "feat(db): wire DBManager into main + inject schema into system prompt"
```

---

## Task 13: Frontend — Settings UI (Databases Tab)

**Files:**
- Create: `frontend/src/components/Settings/DatabasesTab.tsx`
- Modify: `frontend/src/components/Settings/Settings.tsx` (add tab)

- [ ] **Step 1: Create DatabasesTab.tsx**

Component showing:
- List of discovered connections (name, driver, source, status badge)
- "Test Connection" button per row
- "Rescan Project" button
- Manual add form (name, driver dropdown, DSN input)

Uses Wails bindings to call `App.ListDatabaseConnections()`, `App.TestDatabaseConnection(name)`, `App.RescanDatabases()`.

- [ ] **Step 2: Add tab to Settings component**

Add "Databases" tab alongside existing tabs.

- [ ] **Step 3: Regenerate Wails bindings**

Run: `wails3 generate bindings -ts`

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/Settings/
git commit -m "feat(db): add Databases settings tab"
```

---

## Task 14: Integration — App API Methods for Frontend

**Files:**
- Modify: `internal/api/app.go`

- [ ] **Step 1: Add App methods for database management**

```go
func (a *App) ListDatabaseConnections() []ConnectionInfo
func (a *App) TestDatabaseConnection(args json.RawMessage) error
func (a *App) RescanDatabases() ([]ConnectionInfo, error)
```

These delegate to `DBManager`.

- [ ] **Step 2: Run go vet + build frontend**

Run: `go vet ./internal/api/...`

- [ ] **Step 3: Commit**

```bash
git add internal/api/app.go
git commit -m "feat(db): add database API methods for frontend"
```

---

## Task 15: Build & Integration Test

- [ ] **Step 1: Full build**

```bash
cd frontend && npm run build && cd ..
go build -o monika .
```

- [ ] **Step 2: Manual smoke test**

Launch `./monika`, open a project with a `.env` containing `DATABASE_URL`, verify:
1. `.monika/databases.json` is created
2. Settings > Databases shows the connection
3. Agent can use `db_schema` and `db_query` tools

- [ ] **Step 3: Final commit**

```bash
git add -A
git commit -m "feat(db): database auto-discovery and query — complete integration"
```
