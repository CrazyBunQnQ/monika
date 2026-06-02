# LSP 集成

Monika 内置 [Language Server Protocol](https://microsoft.github.io/language-server-protocol/) 客户端，为 AI Agent 提供代码智能能力——诊断、跳转定义、查找引用、重命名等，与 VS Code 等 IDE 底层使用相同的技术。

[English](lsp.md)

## 工作原理

1. 打开项目时，Monika 扫描项目根目录下的标记文件（如 `go.mod`、`package.json`、`Cargo.toml`）
2. 根据标记文件自动匹配内置的 40+ 语言服务器配置
3. 检查对应语言服务器的可执行文件是否可用（PATH、`node_modules/.bin`、venv 等）
4. 当 Agent 需要分析某个文件时，按需启动对应语言服务器（stdio 模式）
5. 空闲 5 分钟后自动关闭语言服务器，释放资源

## 内置支持的语言

| 语言 | 服务器 | 标记文件 |
|------|--------|----------|
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
| Deno | `deno`（LSP 模式） | `deno.json` |

> 完整列表见源码 [`internal/lsp/defaults.go`](../internal/lsp/defaults.go)。

## 安装语言服务器

语言服务器需自行安装，Monika 不捆绑任何语言服务器。常见安装方式：

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
# 参考 clangd 官方安装指南

# Lua
npm install -g lua-language-server
```

安装完成后，确保可执行文件在 `PATH` 中可找到。Monika 也会搜索 `node_modules/.bin` 和 Python venv 目录。

## Agent 可用的 LSP 操作

Agent 通过内置的 `lsp` 工具调用以下操作：

| 操作 | 说明 |
|------|------|
| `diagnostics` | 获取文件的错误和警告 |
| `definition` | 跳转到定义（函数、类型、变量） |
| `type_definition` | 跳转到类型定义 |
| `implementation` | 查找接口/抽象类型的所有实现 |
| `references` | 查找符号的所有引用 |
| `hover` | 获取悬停信息（类型、签名、文档） |
| `symbols` | 获取文件符号大纲 |
| `code_actions` | 列出可用的代码操作（快速修复、重构） |
| `execute_code_action` | 执行指定的代码操作 |
| `rename` | 跨工作区重命名符号 |
| `status` | 查看已配置和运行中的 LSP 服务器状态 |

### 参数说明

```jsonc
{
  "action": "diagnostics",      // 必填，操作名
  "file": "/path/to/file.go",   // 文件绝对路径（status 操作除外）
  "line": 10,                   // 0-based 行号（definition、hover 等需要）
  "character": 5,               // 0-based 列号
  "end_line": 15,               // 范围结束行（code_actions）
  "end_character": 20,          // 范围结束列
  "new_name": "newVar",         // rename 的新名称
  "action_title": "Add import"  // execute_code_action 的操作标题
}
```

## 自定义配置

在项目根目录创建 `.monika/lsp.json`，可以覆盖内置配置或添加新的语言服务器：

```jsonc
{
  // 覆盖内置配置
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

  // 禁用某个内置服务器
  "typescript-language-server": {
    "disabled": true
  },

  // 添加自定义语言服务器
  "my-custom-lsp": {
    "command": "my-lsp-server",
    "args": ["--stdio"],
    "fileTypes": [".custom"],
    "rootMarkers": [".customrc"]
  }
}
```

### 配置字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `command` | string | 可执行文件名或绝对路径 |
| `args` | string[] | 启动参数 |
| `fileTypes` | string[] | 关联的文件扩展名（如 `.go`、`.ts`） |
| `rootMarkers` | string[] | 项目根目录标记文件（支持 glob 模式） |
| `initOptions` | object | LSP initialize 请求的初始化选项 |
| `settings` | object | `workspace/didChangeConfiguration` 的配置 |
| `disabled` | boolean | 设为 `true` 禁用该服务器 |

## 可执行文件查找顺序

Monika 按以下顺序查找语言服务器可执行文件：

1. 如果 `command` 是绝对路径，直接使用
2. 项目的 `node_modules/.bin/` 目录
3. Python 虚拟环境（`.venv/bin/`、`venv/bin/`，Windows 上为 `Scripts/`）
4. 系统 `PATH`

## 自动行为

- **按需启动** — 只有 Agent 实际请求分析文件时，才启动对应的语言服务器
- **自动重连** — 检测到语言服务器进程已死时，自动清理并重新启动
- **空闲超时** — 5 分钟无活动后自动关闭语言服务器
- **内容同步** — 每次 Agent 请求前，自动将最新文件内容同步给语言服务器

## 调试

如果 LSP 功能异常：

- 调试日志：Monika 可执行文件同目录的 `lsp_debug.log`
- 使用 `status` 操作查看服务器运行状态：

```
lsp action=status
```
