# LSP Integration

Monika includes a built-in [Language Server Protocol](https://microsoft.github.io/language-server-protocol/) client that gives the AI agent code intelligence — diagnostics, go-to-definition, find references, rename, and more — using the same underlying technology as VS Code and other IDEs.

## How It Works

1. When a project is opened, Monika scans the project root for marker files (e.g. `go.mod`, `package.json`, `Cargo.toml`).
2. Marker files are matched against 40+ built-in language server configurations.
3. Monika checks whether the corresponding language server binary is available (PATH, `node_modules/.bin`, venv, etc.).
4. When the agent needs to analyze a file, the matching language server is started on demand (stdio mode).
5. Idle servers are automatically stopped after 5 minutes to free resources.

## Built-in Languages

| Language | Server | Marker Files |
|----------|--------|--------------|
| Go | `gopls` | `go.mod`, `go.work` |
| TypeScript / JavaScript | `typescript-language-server` | `tsconfig.json`, `package.json` |
| Rust | `rust-analyzer` | `Cargo.toml` |
| C / C++ | `clangd` | `compile_commands.json`, `CMakeLists.txt` |
| Python | `pyright` / `basedpyright` / `ruff` / `pylsp` | `pyproject.toml`, `requirements.txt` |
| Java | `jdtls` | `pom.xml`, `build.gradle` |
| Kotlin | `kotlin-lsp` | `build.gradle.kts` |
| C# | `omnisharp` | `*.sln`, `*.csproj` |
| Swift | `sourcekit-lsp` | `Package.swift` |
| Ruby | `ruby-lsp` / `solargraph` | `Gemfile` |
| Scala | `metals` | `build.sbt` |
| Haskell | `hls` | `stack.yaml`, `cabal.project` |
| OCaml | `ocamllsp` | `dune-project` |
| Elixir | `elixir-ls` | `mix.exs` |
| Erlang | `erlang_ls` | `rebar.config` |
| Gleam | `gleam` | `gleam.toml` |
| Zig | `zls` | `build.zig` |
| Lua | `lua-language-server` | `.luarc.json` |
| PHP | `intelephense` / `phpactor` | `composer.json` |
| Dart | `dart` | `pubspec.yaml` |
| HTML | `vscode-html-language-server` | `package.json` |
| CSS / SCSS / Less | `vscode-css-language-server` | `package.json` |
| JSON | `vscode-json-language-server` | `package.json` |
| YAML | `yaml-language-server` | `.git` |
| Bash | `bash-language-server` | `.git` |
| Vue | `vue-language-server` | `vue.config.js` |
| Svelte | `svelteserver` | `svelte.config.js` |
| Astro | `astro-ls` | `astro.config.mjs` |
| Tailwind CSS | `tailwindcss-language-server` | `tailwind.config.js` |
| GraphQL | `graphql-lsp` | `.graphqlrc` |
| Prisma | `prisma-language-server` | `schema.prisma` |
| Terraform | `terraform-ls` | `.terraform` |
| Dockerfile | `docker-langserver` | `Dockerfile` |
| LaTeX | `texlab` | `.latexmkrc` |
| Markdown | `marksman` | `.marksman.toml` |
| Nix | `nixd` | `flake.nix` |
| Biome (JS/TS) | `biome` | `biome.json` |
| ESLint | `vscode-eslint-language-server` | `.eslintrc` |
| Deno | `deno` (LSP mode) | `deno.json` |

> Full source: [`internal/lsp/defaults.go`](../internal/lsp/defaults.go).

## Installing Language Servers

Language servers must be installed separately — Monika does not bundle any. Common installations:

```bash
# TypeScript / JavaScript
npm install -g typescript-language-server typescript

# Go
go install golang.org/x/tools/gopls@latest

# Python (pyright)
npm install -g pyright

# Python (ruff)
pip install ruff

# Rust
rustup component add rust-analyzer

# C/C++
# See the official clangd installation guide

# Lua
npm install -g lua-language-server
```

Make sure the binary is discoverable via `PATH`. Monika also searches `node_modules/.bin` and Python venv directories.

## Agent LSP Actions

The agent uses the built-in `lsp` tool to invoke these actions:

| Action | Description |
|--------|-------------|
| `diagnostics` | Get errors and warnings for a file |
| `definition` | Go to definition (function, type, variable) |
| `type_definition` | Go to type definition |
| `implementation` | Find all implementations of an interface / abstract type |
| `references` | Find all references to a symbol |
| `hover` | Get hover info (type, signature, docs) |
| `symbols` | Get document symbol outline |
| `code_actions` | List available code actions (quick fixes, refactoring) |
| `execute_code_action` | Execute a specific code action |
| `rename` | Rename a symbol across the workspace |
| `status` | Show configured and running LSP servers |

### Parameters

```jsonc
{
  "action": "diagnostics",      // required — action name
  "file": "/path/to/file.go",   // absolute file path (not needed for status)
  "line": 10,                   // 0-based line (for definition, hover, etc.)
  "character": 5,               // 0-based column
  "end_line": 15,               // range end line (for code_actions)
  "end_character": 20,          // range end column
  "new_name": "newVar",         // new name for rename
  "action_title": "Add import"  // code action title for execute_code_action
}
```

## Custom Configuration

Create `.monika/lsp.json` in the project root to override built-in configs or add new servers:

```jsonc
{
  // Override a built-in server
  "gopls": {
    "command": "/usr/local/bin/gopls",
    "args": ["serve"],
    "fileTypes": [".go"],
    "rootMarkers": ["go.mod"],
    "settings": {
      "gopls": {
        "staticcheck": true,
        "goimportsLocalPrefix": "github.com/myorg"
      }
    }
  },

  // Disable a built-in server
  "typescript-language-server": {
    "disabled": true
  },

  // Add a custom server
  "my-custom-lsp": {
    "command": "my-lsp-server",
    "args": ["--stdio"],
    "fileTypes": [".custom"],
    "rootMarkers": [".customrc"]
  }
}
```

### Config Fields

| Field | Type | Description |
|-------|------|-------------|
| `command` | string | Executable name or absolute path |
| `args` | string[] | Launch arguments |
| `fileTypes` | string[] | Associated file extensions (e.g. `.go`, `.ts`) |
| `rootMarkers` | string[] | Project root marker files (glob patterns supported) |
| `initOptions` | object | Initialization options for the LSP `initialize` request |
| `settings` | object | Configuration sent via `workspace/didChangeConfiguration` |
| `disabled` | boolean | Set to `true` to disable this server |

## Binary Discovery Order

Monika searches for language server binaries in this order:

1. If `command` is an absolute path, use it directly
2. Project's `node_modules/.bin/`
3. Python virtual environments (`.venv/bin/`, `venv/bin/`, or `Scripts/` on Windows)
4. System `PATH`

## Automatic Behavior

- **On-demand startup** — language servers are started only when the agent actually requests file analysis.
- **Auto-reconnect** — dead server processes are detected and restarted automatically.
- **Idle timeout** — servers with no activity for 5 minutes are shut down.
- **Content sync** — file content is synced to the language server before each agent request.

## Debugging

If LSP features aren't working as expected:

- Debug log: `lsp_debug.log` in the same directory as the Monika executable.
- Use the `status` action to check server states:

```
lsp action=status
```
