<p align="center">
  <h1 align="center">Monika</h1>
  <p align="center">
    <strong>The open-source, AI-native desktop code editor.</strong><br>
    Not a chat sidebar bolted onto an IDE — an editor where the AI agent has first-class access to your entire project.
  </p>
</p>

<p align="center">
  <a href="README.zh.md">中文</a>
</p>

<p align="center">
  <a href="https://github.com/RedTeaLab/monika/actions"><img src="https://img.shields.io/github/actions/workflow/status/RedTeaLab/monika/release.yml?style=flat&colorA=222222&colorB=3FB950" alt="Build"></a>
  <a href="https://github.com/RedTeaLab/monika/releases"><img src="https://img.shields.io/github/v/release/RedTeaLab/monika?style=flat&colorA=222222&colorB=F0883E" alt="Release"></a>
  <a href="https://github.com/RedTeaLab/monika/blob/main/LICENSE"><img src="https://img.shields.io/github/license/RedTeaLab/monika?style=flat&colorA=222222&colorB=58A6FF" alt="License"></a>
  <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-00ADD8?style=flat&colorA=222222&logo=go&logoColor=white" alt="Go"></a>
  <a href="https://react.dev"><img src="https://img.shields.io/badge/React-61DAFB?style=flat&colorA=222222&logo=react&logoColor=black" alt="React"></a>
  <a href="https://wails.io"><img src="https://img.shields.io/badge/Wails_v3-F7CC42?style=flat&colorA=222222" alt="Wails"></a>
</p>

![Monika Screenshot](screen/PixPin_2026-06-07_16-40-22.png)

---

## What is Monika?

Monika is an **open-source AI coding editor** for the desktop. Built on [Wails v3](https://wails.io) (Go backend + React frontend), it gives an autonomous AI agent direct, first-class access to your codebase — the file tree, the Monaco editor, language servers, git, and a full tool belt — through a multi-panel GUI.

Instead of copy-pasting between a chat window and your editor, the agent in Monika reads files, runs searches, executes commands, applies edits, and resolves conflicts **inside the same environment you work in**. It perceives the whole project and acts on it directly.

**Provider-agnostic.** Bring your own API key — any OpenAI-compatible endpoint works (DeepSeek, OpenAI, Claude, Gemini, local models, …). Context windows and output limits are auto-fetched from [models.dev](https://models.dev).

> ### Why "Monika"?
> The name comes from *Doki Doki Literature Club*. In the game, Monika is the one character who becomes self-aware — she realizes she's inside a program, reads its files, and rewrites the world around her.
>
> That's exactly what we want an AI coding agent to do: not just sit in a chat box, but perceive the entire project and act on it. An agent that breaks the fourth wall between conversation and codebase.
>
> We believe the next generation of IDEs will be driven by AI at the core, not bolted on as a sidebar. Monika is our attempt to build that future.

---

## Why Monika?

Most "AI coding" tools today fall into two camps: **closed-source editors** (Cursor, Windsurf) that lock you into their cloud and model choices, or **terminal agents / IDE plugins** (Aider, OpenCode, Continue) that are powerful but lack a cohesive visual workspace.

Monika is different:

| | Monika | Cursor / Windsurf | Aider / OpenCode | Continue (plugin) |
|---|---|---|---|---|
| **Open source** | Yes | No | Yes | Yes |
| **Desktop GUI** | Yes | Yes | Terminal | IDE panel |
| **AI-native (not a plugin)** | Yes | Yes | Yes | No |
| **Bring your own model** | Any OpenAI-compatible | Limited | Yes | Yes |
| **In-editor conflict resolution** | Yes | Partial | No | No |
| **LSP-native editing** | Monaco + LSP | VS Code fork | External | Host IDE |
| **Self-contained binary** | Yes | No | Yes | No |

**Key differentiators:**

- **AI-native, not AI-bolted-on.** The agent loop, tool system, and editor are designed together — the agent edits files in the same Monaco editor you use, with the same LSP diagnostics.
- **Interactive editing with conflict detection.** When the AI edits a file that has changed on disk, you get a side-by-side conflict resolution UI — never silently overwrite work.
- **Provider-agnostic.** No vendor lock-in. Point at any OpenAI-compatible API.
- **Extensible via Skills & MCP.** Load reusable agent skills from GitHub repos, or connect databases, browsers, and web search through Model Context Protocol.

---

## Features

### AI Agent Loop

Real-time streaming responses, tool-call cards, and token-usage tracking. The agent automatically compacts context when a conversation exceeds the model's window — summarizing older messages via a separate LLM call so long sessions stay coherent.

### Full-Project Tool Belt

The agent operates on your project directly through a rich set of built-in tools:

| Category | Tools | What they do |
|----------|-------|--------------|
| **Files** | `file_read`, `file_write`, `file_edit`, `patch`, `file_list` | Read, create, edit, and patch files with precision |
| **Search** | `glob`, `grep` | Discover files by pattern and search content by regex |
| **Shell** | `bash`, `background_task` | Run commands synchronously or as trackable background tasks |
| **Code intel** | `lsp`, `lsp_list` | Diagnostics, go-to-definition, references, rename, hover, completions ([docs](docs/lsp.md)) |
| **Collaboration** | `ask_user`, `spawn_agent` | Ask clarifying questions, fan out parallel sub-agents |
| **Task tracking** | `task_create`, `task_append`, `task_update`, `task_list` | Structured todo lists for multi-step work |
| **Extensibility** | `skill`, `install_skill`, `mcp_search`, `install_mcp_server`, … | Discover and install Skills & MCP servers at runtime |

### Monaco Editor, LSP-Native

The built-in editor is [Monaco](https://microsoft.github.io/monaco-editor/) (the engine behind VS Code) wired to real Language Server Protocol servers. The agent sees the same diagnostics, completions, and symbol info you do. Tree-sitter decorations add syntax highlighting even where no LSP is configured.

### Interactive Editing & Conflict Resolution

When the AI proposes a file edit, it lands in an editable preview panel. If the file changed on disk since the agent last read it, a **conflict UI** lets you reconcile AI changes with your own — nothing is silently overwritten. Dirty-file tracking keeps a clear indicator of unsaved work across sessions.

### Multi-Tab Sessions

Up to 8 concurrent session tabs, each with independent context. Sessions persist as JSON and restore on the next launch.

### Multi-Panel Workspace

Session list, chat, file tree + editor, background-task console, and status bar in one window. Three layout modes — chat-focused, split, and files-only — with a draggable divider.

### Shell Mode

A dedicated input mode for running shell commands inline, with ANSI rendering, tab-completion, history navigation, and live background-task log streaming.

### Git Integration

Change tracking, diffs, local/remote branch listing, branch create/switch, and worktree-aware branch management.

### Skills & MCP

- **Skills** — Supports the [SKILL.md](https://github.com) standard. Auto-discovers and loads reusable agent workflows from GitHub repos.
- **MCP** — [Model Context Protocol](https://modelcontextprotocol.io) support extends the agent with databases, browsers, web search, and more via stdio JSON-RPC.

### Concurrent Sub-Agents

A built-in TaskRunner dispatches up to 4 concurrent child agents (semaphore-gated) — ideal for large-scale code search and multi-file refactors.

### Permission Safety

Every tool call passes through a permission pipeline — hard rules plus a security model — preventing unauthorized or destructive operations.

---

## Quick Start

Download a prebuilt binary from [Releases](https://github.com/RedTeaLab/monika/releases), or build from source:

### Prerequisites

| Platform | Requirements |
|----------|-------------|
| **macOS** | Go 1.25+, Node.js 18+, Xcode Command Line Tools |
| **Windows** | Go 1.25+, Node.js 18+, WebView2 |

### Install Wails v3 CLI

```bash
# Install the CLI version matching this project (v3.0.0-alpha.78)
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha.78
```

### Build from Source

```bash
git clone https://github.com/RedTeaLab/monika.git
cd monika

# 1. Install frontend dependencies
cd frontend && npm install && cd ..

# 2. Generate Wails bindings (Go types → TypeScript)
wails3 generate bindings -ts
node -e "require('fs').copyFileSync('build/barrel_index.ts','frontend/bindings/monika/index.ts')"

# 3a. Dev mode (hot reload)
wails3 dev

# 3b. Or build standalone
cd frontend && npm run build && cd ..
go build -o monika .
# macOS: ./monika
# Windows: .\monika.exe
```

### Configure Provider

First launch guides you through setup, or create `~/.monika/config.yaml` manually:

```yaml
model_provider: deepseek
model: deepseek-chat
model_providers:
  deepseek:
    name: deepseek
    base_url: https://api.deepseek.com
    api_key: sk-xxx
```

---

## Features

### Multi-Panel GUI

Session list, chat area, file tree with CodeMirror 6 editor, console, and status bar — all in one window. Three layout modes (chat, split, files-only) with a draggable divider.

### Multi-Tab Sessions

Up to 8 concurrent session tabs with independent message caching. Sessions are automatically persisted as JSON and restored on next startup.

### Streaming Agent Loop

Real-time text deltas, tool execution cards, and token usage tracking. The agent handles context compaction automatically — when the conversation exceeds the model limit, it summarizes older messages with a separate LLM call.

### Tool Calling

The agent can manipulate your project directly:

| Tool | Description |
|------|-------------|
| `file_read` | Read files with precision (offset/limit) |
| `file_write` | Create or overwrite files |
| `file_edit` | Exact string replacement |
| `file_list` | List directory contents |
| `glob` | Glob pattern file discovery |
| `grep` | Regex search across files |
| `bash` | Execute shell commands (cross-platform) |
| `lsp` | Language Server Protocol — diagnostics, go-to-definition, references, rename, etc. ([docs](docs/lsp.md)) |
| `db_schema` | Browse database schema (tables, columns, foreign keys) |
| `db_query` | Execute read-only SQL/Redis queries (SELECT, SHOW, EXPLAIN) |

### Git Integration

File change tracking, diff viewing, local/remote branch listing, branch creation and switching, worktree-aware branch management.

### Skills & MCP

- **Skills** — Supports the [SKILL.md](https://github.com) standard, auto-discovers and loads skills from GitHub repos
- **MCP** — Model Context Protocol, extends agent capabilities (databases, browser, web search, etc.) via stdio JSON-RPC transport

### Concurrent Sub-Agents

Built-in TaskRunner dispatches up to 4 concurrent child agents via semaphore, ideal for large-scale code search and multi-file modification tasks.

### Permission Safety

Every tool call goes through a complete permission pipeline — hard rules and security model double validation to prevent unauthorized operations.

## Supported Providers

Any OpenAI-compatible endpoint works out of the box:

| Provider | Engine ID | Default Model |
|----------|-----------|---------------|
| DeepSeek | `deepseek` | `deepseek-chat` |
| OpenAI | `openai` | `gpt-4o` |
| Anthropic Claude | via OpenAI-compatible API | `claude-sonnet-4-5` |
| Google Gemini | via OpenAI-compatible API | `gemini-2.0-flash` |
| Local (Ollama, LM Studio, …) | any OpenAI-compatible endpoint | — |
| Custom | any OpenAI-compatible endpoint | — |

---

## Architecture

```
monika/
├── main.go                # Wails entry point, embeds frontend, wires all services
├── frontend/              # React 18 + TypeScript + Tailwind CSS v4
│   └── src/
│       ├── App.tsx        # Root component, dockview panel layout
│       ├── store/         # Single Zustand store, all app state
│       └── components/    # UI components
├── internal/
│   ├── agent/             # Agent loop, streaming, compaction, sub-agent dispatch
│   ├── api/               # Wails services: App, SessionManager, FileService, EventBus
│   ├── bootstrap/         # Provider initialization
│   ├── config/            # YAML/JSON config loader (~/.monika/ + .monika/)
│   ├── dbbridge/          # Bridge scripts for Node.js/Python database drivers
│   ├── dbdiscovery/       # Database auto-discovery from .env, docker-compose
│   ├── engines/           # Provider adapters + Skill + MCP engines
│   ├── lsp/               # Language Server Protocol client + LSP tool
│   ├── permission/        # Tool permission pipeline
│   └── tool/              # Tool interface + registry + builtin tools
└── pkg/
    ├── dbdriver/          # Database driver interface + 5 native drivers (PostgreSQL, MySQL, SQLite, Redis, MongoDB)
    ├── engine/            # Public Engine interface + registry
    ├── openai/            # OpenAI-compatible SSE streaming client
    ├── modelsdev/         # models.dev catalog fetcher
    └── gitutil/           # Git utility functions
```

### Engine Pattern

Every engine implements the `pkg/engine.Engine` interface and self-registers via `init()` + `engine.Register()`. Provider engines additionally implement `StreamChat` and `ListModels`. The `wire_api` config field selects which adapter to use.

### Tool Pattern

Tools implement `Name()` / `Description()` / `Parameters()` / `Execute()` and are composed via combinable registration functions (`RegisterDefaults`, `RegisterTasks`, `RegisterSpawnAgent`, …).

---

## Development

```bash
# Run tests
go test ./...

# Static analysis
go vet ./...

# Format
gofmt -w .

# Build frontend
cd frontend && npm run build

# Regenerate Wails bindings (after changing Go API types)
wails3 generate bindings -ts
node -e "require('fs').copyFileSync('build/barrel_index.ts','frontend/bindings/monika/index.ts')"

# Tidy dependencies
go mod tidy
```

---

## Contributing

Issues and pull requests are welcome. See [AGENTS.md](AGENTS.md) for development details and architecture decisions.

## License

[MIT License](LICENSE) © 2025 RedTeaLab

Third-party components:

| Component | License |
|-----------|---------|
| [Wails](https://wails.io) | MIT |
| [Monaco Editor](https://microsoft.github.io/monaco-editor/) | MIT |
| [tree-sitter](https://tree-sitter.github.io/) | MIT |
| [dockview](https://dockview.dev) | MIT |
| [React](https://react.dev) | MIT |
| [zustand](https://zustand.docs.pmnd.rs) | MIT |
| [LXGW WenKai](https://github.com/lxgw/LxgwWenKai) | SIL OFL 1.1 |
| [Maple Mono NF](https://github.com/subframe7536/Maple-font) | SIL OFL 1.1 |

## Acknowledgments

Monika stands on the shoulders of giants. These projects shaped our thinking and, in many cases, provided direct inspiration:

- **[VS Code](https://github.com/microsoft/vscode)** — The editor that defined the modern development experience.
- **[oh-my-pi](https://github.com/can1357/oh-my-pi)** — A remarkably capable terminal AI agent with deep LSP integration and sub-agent orchestration that showed us what an agent surface could be.
- **[OpenCode](https://github.com/sst/opencode)** — An open-source AI coding agent whose clean architecture and provider-agnostic approach informed our design.

Open source builds on open source — and we're glad to be part of that tradition.
