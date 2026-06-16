# Database Auto-Discovery & Query — Design Spec

**Date**: 2026-06-10
**Status**: Approved

## Summary

Enable Monika's Agent to automatically discover project databases, query their schemas, and execute read-only queries during coding assistance. The Agent uses the project's own database drivers (via lightweight bridge scripts) rather than bundling separate Go drivers.

## Motivation

When an Agent writes SQL, ORM models, or migration code, it currently has no visibility into the project's actual database. It guesses schema from code alone. By giving the Agent direct read access to the database, it can:

- Reference real table structures when generating code
- Query sample data to validate assumptions
- Understand relationships across tables
- Produce accurate migration scripts

## Architecture

```
┌─────────────────────────────────────────────────────┐
│  Monika (Go)                                        │
│                                                     │
│  ┌──────────────┐   ┌──────────────┐               │
│  │ dbdiscovery  │──▶│ databases.json│ (cache)       │
│  │ scan project │   └──────┬───────┘               │
│  └──────────────┘          │                        │
│                            ▼                        │
│  ┌──────────────────────────────────────────────┐   │
│  │ DBManager                                    │   │
│  │  - connection lifecycle                      │   │
│  │  - schema summary → system prompt            │   │
│  │  - db_schema / db_query tool execution       │   │
│  └──────────┬───────────────────┬───────────────┘   │
│             │                   │                    │
│    ┌────────▼────────┐  ┌──────▼──────────┐        │
│    │ Bridge process   │  │ Go native driver │        │
│    │ (long-lived)     │  │ (Go project      │        │
│    │ node/python/ruby │  │  fallback)       │        │
│    └────────┬────────┘  └──────┬──────────┘        │
│             │                   │                    │
└─────────────┼───────────────────┼────────────────────┘
              ▼                   ▼
         Project drivers     Compiled-in drivers
         (node_modules/      (statically linked)
          venv/ Gems)
```

## Module 1: Auto-Discovery — `internal/dbdiscovery/`

### Files

```
internal/dbdiscovery/
├── discover.go       # Discoverer interface + registry + Scan logic
├── env.go            # .env / .env.local → DATABASE_URL / REDIS_URL
├── docker.go         # docker-compose.yml → database services
├── rails.go          # config/database.yml
├── prisma.go         # prisma/schema.prisma
├── django.go         # settings.py → DATABASES dict
├── nodeorm.go        # typeorm/knex/sequelize config files
└── runtime.go        # runtime detection (package.json → node, etc.)
```

### Discoverer Interface

```go
type Discoverer interface {
    Name() string
    Scan(projectDir string) ([]DiscoveredDB, error)
}

type DiscoveredDB struct {
    Name        string // "docker-compose/postgres"
    Driver      string // "postgres", "mysql", "sqlite", "mongo", "redis"
    DSN         string // connection string
    Source      string // source file path
    RuntimeHint string // "node" / "python" / "ruby" / "" (empty = Go)
}
```

### Discovery Flow

1. Detect project runtime via `Runtime.Detect(projectDir)`
2. Run all Discoverers in priority order, collect results
3. Deduplicate (same DSN from different sources → merge)
4. Write to `.monika/databases.json`

### Trigger Points

- Project first opened (no `databases.json` exists)
- User clicks "Rescan" in Settings UI
- Project config file changes (watch `.env`, `docker-compose.yml`, etc.)

### Cache File — `.monika/databases.json`

```json
{
  "scanned_at": "2026-06-10T10:00:00Z",
  "runtime": "node",
  "connections": [
    {
      "name": "docker-compose/postgres",
      "driver": "postgres",
      "dsn": "postgresql://app:secret@localhost:5432/app_dev",
      "source": "docker-compose.yml",
      "status": "available"
    }
  ]
}
```

Users can edit this file directly to adjust connections. Relative paths in DSN (for SQLite) resolve relative to `projectDir`.

## Module 2: Bridge Process — `internal/dbbridge/`

### Files

```
internal/dbbridge/
├── bridge.go         # BridgeManager: process management + JSON IPC
├── protocol.go       # request/response protocol types
└── scripts/
    ├── node.js       # Node.js bridge — uses project node_modules drivers
    ├── python.py     # Python bridge — uses project venv drivers
    └── ruby.rb       # Ruby bridge — uses project Gem drivers
```

### IPC Protocol

Communication via stdin/stdout JSON, one message per line:

```jsonl
// Requests (Go → Bridge)
{"id":1,"action":"open","driver":"postgres","dsn":"postgresql://..."}
{"id":2,"action":"query","conn":"main-db","query":"SELECT * FROM users LIMIT 5"}
{"id":3,"action":"schema","conn":"main-db","filter":""}
{"id":4,"action":"close","conn":"main-db"}

// Responses (Bridge → Go)
{"id":1,"status":"ok","conn":"main-db"}
{"id":2,"status":"ok","columns":["id","email","name"],"rows":[[1,"a@b.c","Alice"]],"tag":"SELECT 3"}
{"id":3,"status":"ok","tables":[{"name":"users","columns":[{"name":"id","type":"serial","pk":true},{"name":"email","type":"varchar(255)","unique":true}]}]}
{"id":4,"status":"ok"}
```

### Bridge Script Structure

Each bridge script (~60 lines) follows the same pattern:

1. Read JSON lines from stdin
2. `open`: load the project's driver, connect using DSN
3. `query`: execute read-only query, return columns + rows
4. `schema`: introspect tables/columns/indexes/foreign keys
5. `close`: close connection

### Read-Only Enforcement

Enforced **inside the bridge script** before execution:

- **SQL (postgres/mysql/sqlite)**: Parse first keyword, only allow `SELECT`, `SHOW`, `DESCRIBE`, `EXPLAIN`, `WITH` (validate WITH body starts with SELECT)
- **Redis**: Only allow read commands: `GET`, `MGET`, `KEYS`, `TYPE`, `SCAN`, `HGET`, `HGETALL`, `LRANGE`, `SMEMBERS`, `ZCARD`, `ZSCORE`, etc.
- **MongoDB**: Only allow `find`, `aggregate`, `count`, `distinct`, `listCollections`

### Lifecycle

- **Start**: When project opens and databases are discovered
- **Keep-alive**: Heartbeat ping every 30s, auto-restart on crash (max 3 retries)
- **Shutdown**: SIGTERM on project switch or app exit
- **Error**: If project driver not installed, return clear message (e.g., "PostgreSQL driver not found. Run: npm install pg")

### Runtime Detection Priority

```
package.json       → node bridge
requirements.txt   → python bridge
pyproject.toml     → python bridge
Pipfile            → python bridge
Gemfile            → ruby bridge
go.mod (only)      → Go native driver fallback
```

## Module 3: Go Native Driver Fallback — `pkg/dbdriver/`

For pure Go projects with no script runtime.

```
pkg/dbdriver/
├── driver.go        # Driver/Connection interfaces + Registry
├── postgres/
│   └── postgres.go  # pgx driver, init() self-registers
├── mysql/
│   └── mysql.go     # go-sql-driver/mysql
├── sqlite/
│   └── sqlite.go    # modernc.org/sqlite (pure Go, no CGO)
├── mongo/
│   └── mongo.go     # mongo-driver
└── redis/
    └── redis.go     # go-redis
```

### Driver Interface

```go
type Driver interface {
    ID() string
    Open(dsn string) (Connection, error)
}

type Connection interface {
    Query(ctx context.Context, query string, args ...any) (*QueryResult, error)
    Schema(ctx context.Context, filter string) (*SchemaResult, error)
    Close() error
}

type QueryResult struct {
    Columns []string
    Rows    [][]any
    Tag     string
}
```

Self-registration pattern (same as `pkg/engine`):

```go
func init() { dbdriver.Register(&PostgresDriver{}) }
```

Activation in `main.go`:

```go
import (
    _ "monika/pkg/dbdriver/postgres"
    _ "monika/pkg/dbdriver/mysql"
    _ "monika/pkg/dbdriver/sqlite"
    _ "monika/pkg/dbdriver/mongo"
    _ "monika/pkg/dbdriver/redis"
)
```

### Selection Logic

```
Project has Node.js/Python/Ruby → Bridge (primary, uses project drivers)
Project is pure Go             → Go native drivers (fallback)
Neither available              → Mark connection "unavailable", Agent prompts user
```

## Module 4: Tools — `internal/tool/builtin/`

### db_schema (read-only, auto-allowed)

```go
Name:        "db_schema"
Description: "Query database schema (tables, columns, indexes, foreign keys) for the project's databases."
Parameters: {
    "connection": { "type": "string", "description": "Connection name (optional, defaults to first)" },
    "filter":     { "type": "string", "description": "Table name pattern (optional, e.g. 'users%')" }
}
```

Classification: **read operation** in permission pipeline → auto-allowed in Auto mode.

### db_query (read-only queries, permission pipeline)

```go
Name:        "db_query"
Description: "Execute a read-only SQL query against a project database. Only SELECT/SHOW/DESCRIBE/EXPLAIN allowed."
Parameters: {
    "connection": { "type": "string", "description": "Connection name (optional, defaults to first)" },
    "query":      { "type": "string", "description": "SQL query to execute (read-only)" }
}
```

Classification: **read operation** (read-only enforced at bridge level, no mutations possible).

### Registration

Add `RegisterDatabase(r *tool.ToolRegistry, dbMgr *DBManager)` in `register.go`.

## Module 5: Schema Injection into System Prompt

`DBManager.SchemaSummary()` generates a compact schema overview injected into the agent's system prompt:

```
## Connected Databases (read-only access)

### main-db (PostgreSQL) — from docker-compose.yml
Tables: users(id serial PK, email varchar(255) UNIQUE, name varchar(100), created_at timestamp),
        orders(id serial PK, user_id int FK→users.id, total decimal(10,2), status varchar(20)),
        order_items(id serial PK, order_id int FK→orders.id, product varchar(200), qty int)

### cache (Redis) — from .env
Key patterns: session:*, cart:*
```

### Refresh Triggers

- Project first opened (lazy: on first agent message)
- User modifies database config in Settings UI
- User clicks "Refresh Schema" in Settings
- Not on every message (avoid unnecessary queries)

## Module 6: Settings UI

New **Databases** tab in the Settings panel:

- List of discovered connections (name, driver, source file, status indicator)
- "Test Connection" button per connection
- "Rescan Project" button to re-run discovery
- Manual add/remove connection form
- Status indicators: connected (green) / unavailable (red) / driver missing (yellow)

## DBManager — `internal/api/db_manager.go`

```go
type DBManager struct {
    mu         sync.RWMutex
    conns      map[string]*DBConnection   // key = connection name
    projectDir string
    bridge     *dbbridge.BridgeManager
    drivers    *dbdriver.Registry
}

func (m *DBManager) Init(entries []DiscoveredDB) error
func (m *DBManager) Query(ctx context.Context, connName, query string) (*QueryResult, error)
func (m *DBManager) Schema(ctx context.Context, connName, filter string) (*SchemaResult, error)
func (m *DBManager) SchemaSummary() string
func (m *DBManager) CloseAll()
func (m *DBManager) ListConnections() []ConnectionInfo
func (m *DBManager) TestConnection(connName string) error
```

Lazy connection: `Init` stores entries, actual connection happens on first `Query`/`Schema` call.

## Edge Cases

| Scenario | Handling |
|----------|----------|
| No databases in project | No bridge started, tools return "no connections available" |
| Database unreachable | Mark unavailable, Agent suggests checking connection |
| Driver not installed in project | Bridge returns install hint (e.g. `npm install pg`), Agent relays |
| Docker containers not running | Connection fails, Agent suggests `docker-compose up` |
| Multiple runtimes detected | Primary runtime selected by priority (node > python > ruby) |
| Production DSN discovered | Read-only mode enforced; UI shows warning badge |
| SQLite relative path | Resolved relative to projectDir |
| Bridge process crash | Auto-restart up to 3 times, then mark unavailable |

## Supported Databases

| Database | Bridge Driver (Node) | Bridge Driver (Python) | Bridge Driver (Ruby) | Go Fallback |
|----------|---------------------|----------------------|---------------------|-------------|
| PostgreSQL | `pg` | `psycopg2` | `pg` | `pgx` |
| MySQL | `mysql2` | `pymysql` | `mysql2` | `go-sql-driver/mysql` |
| SQLite | `better-sqlite3` | `sqlite3` (stdlib) | `sqlite3-ruby` | `modernc.org/sqlite` |
| MongoDB | `mongodb` | `pymongo` | `mongo` | `mongo-driver` |
| Redis | `ioredis` | `redis` | `redis` | `go-redis` |

## Future Extensions

- ClickHouse, CockroachDB: Add new Discoverer + bridge driver mapping + Go driver
- Database diagram visualization: Use schema data to render ER diagrams
- Migration generation: Agent can diff schema between two connections
- Connection encryption: Encrypt DSN passwords in `databases.json`
