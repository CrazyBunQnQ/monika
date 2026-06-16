# Changelog

## v0.0.20 (2026-06-16)

### New Features

#### Database System

Monika now has a full-featured database integration system, allowing the agent to explore schemas and run read-only queries directly.

- **Auto-discovery**: scans `.env` files and `docker-compose.yml` for database connection strings at project open, with runtime detection for locally running databases
- **5 native drivers**: PostgreSQL (pgx), MySQL (go-sql-driver), SQLite (modernc.org/sqlite), Redis (go-redis), MongoDB (mongo-driver)
- **Bridge pattern**: for drivers without native Go support, spawns Node.js/Python subprocesses via bridge scripts with a JSON protocol
- **Two new tools**:
  - `db_schema` — browse tables, columns, foreign keys, and indexes
  - `db_query` — execute read-only SQL queries (SELECT, SHOW, EXPLAIN) and Redis commands
- **Databases settings tab**: manage connections, test connectivity, view schemas in the UI
- **System prompt integration**: discovered database schemas are automatically injected into the agent's context

#### macOS Support

- File explorer now works on macOS: reveals files in Finder (`open -R`), lists volumes under `/Volumes`
- Cross-platform LSP process spawning (extracted `SysProcAttr` to platform-specific files)
- File dialog navigation fixed for macOS root paths
- Home directory detection for file dialog initial path
- Build instructions and documentation updated for macOS

### Security Hardening (Database)

- SQL read-only validation: blocks multi-statement injection, CTE bypass (`WITH ... INSERT/UPDATE/DELETE`), and `INTO OUTFILE`
- MongoDB `$where` injection blocked
- Redis `KEYS` command removed (replaced with safe alternatives)
- Read-only enforcement at three layers: tool-level, driver-level (readonly transaction mode), permission-level (hard rule)
- Bridge script SQL injection hardening
- Query timeout enforcement on all database operations

### Bug Fixes

- Resolve DSN key-value format and MySQL Go-DSN parsing issues
- Preserve Redis DSN and PostgreSQL `sslmode=prefer`
- Fix AB-BA deadlock in schema cache and TOCTOU race conditions
- Fix bridge crash recovery and stderr capture
- Fix MySQL URL-to-DSN conversion and password encoding
- Fix async schema cancel/restart and reconnect logic
- Fix Unicode alignment in terminal output
- Add Redis PING to allowlist, close pool on reconnect

### Documentation

- Updated AGENTS.md, README.md, README.zh.md with database feature details and macOS build instructions
- Added database auto-discovery design spec and implementation plan
