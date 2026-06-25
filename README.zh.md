<p align="center">
  <h1 align="center">Monika</h1>
  <p align="center">
    <strong>开源的、AI 原生桌面编程编辑器。</strong><br>
    不是嫁接到 IDE 上的聊天侧边栏——而是一个 AI Agent 拥有整个项目一等访问权限的编辑器。
  </p>
</p>

<p align="center">
  <a href="README.md">English</a>
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

## Monika 是什么？

Monika 是一个**开源的 AI 编程编辑器**，运行在桌面端。基于 [Wails v3](https://wails.io)（Go 后端 + React 前端）构建，它让自主的 AI Agent 通过多面板 GUI 直接、一等地访问你的代码库——文件树、Monaco 编辑器、语言服务器、Git 以及完整的工具链。

你不再需要在聊天窗口和编辑器之间来回复制粘贴。Monika 中的 Agent 在**你工作的同一个环境里**读取文件、执行搜索、运行命令、应用修改、解决冲突。它感知整个项目，并直接对其操作。

**不绑定任何模型提供商。** 自带 API Key 即可——任何兼容 OpenAI 的端点都能接入（DeepSeek、OpenAI、Claude、Gemini、本地模型……）。模型上下文窗口和输出限制自动从 [models.dev](https://models.dev) 获取。

> ### 为什么叫 "Monika"？
> 名字来自《心跳文学部》（Doki Doki Literature Club）。游戏中的 Monika 是唯一一个觉醒了自我意识的角色——她意识到自己身处程序之中，能够读取文件、改写周围的世界。
>
> 这正是我们对 AI 编程 Agent 的期望：不只是待在聊天框里回答问题，而是感知整个项目并直接操作。一个打破对话与代码库之间"第四面墙"的 Agent。
>
> 我们相信下一代 IDE 将以 AI 为核心驱动，而非作为侧边栏的附加功能。Monika 是我们迈向这一愿景的尝试。

---

## 为什么选择 Monika？

如今大多数"AI 编程"工具分为两类：**闭源编辑器**（Cursor、Windsurf）将你锁定在它们的云服务和模型选择中；**终端 Agent / IDE 插件**（Aider、OpenCode、Continue）虽然强大，但缺乏统一的可视化工作区。

Monika 与众不同：

| | Monika | Cursor / Windsurf | Aider / OpenCode | Continue (插件) |
|---|---|---|---|---|
| **开源** | 是 | 否 | 是 | 是 |
| **桌面 GUI** | 是 | 是 | 终端 | IDE 面板 |
| **AI 原生（非插件）** | 是 | 是 | 是 | 否 |
| **自带模型** | 任意 OpenAI 兼容 | 有限 | 是 | 是 |
| **编辑器内冲突解决** | 是 | 部分 | 否 | 否 |
| **LSP 原生编辑** | Monaco + LSP | VS Code 分支 | 外部 | 宿主 IDE |
| **独立可执行文件** | 是 | 否 | 是 | 否 |

**核心差异化：**

- **AI 原生，而非 AI 附加。** Agent 循环、工具系统和编辑器是一体设计的——Agent 在你使用的同一个 Monaco 编辑器中编辑文件，共享同一套 LSP 诊断。
- **带冲突检测的交互式编辑。** 当 AI 编辑的文件在磁盘上已发生变化，你会看到并排的冲突解决 UI——绝不静默覆盖你的工作。
- **不绑定提供商。** 没有厂商锁定。指向任意 OpenAI 兼容 API 即可。
- **通过 Skills 和 MCP 扩展。** 从 GitHub 仓库加载可复用的 Agent 技能，或通过 Model Context Protocol 连接数据库、浏览器和 Web 搜索。

---

## 功能一览

### AI Agent 循环

实时流式响应、工具调用卡片、Token 用量追踪。当对话超出模型上下文窗口时，Agent 自动压缩上下文——通过单独的 LLM 调用摘要历史消息，保持长会话的连贯性。

### 全项目工具链

Agent 通过丰富的内置工具直接操作你的项目：

| 分类 | 工具 | 功能 |
|------|------|------|
| **文件** | `file_read`, `file_write`, `file_edit`, `patch`, `file_list` | 精确读取、创建、编辑和修补文件 |
| **搜索** | `glob`, `grep` | 按模式发现文件、按正则搜索内容 |
| **Shell** | `bash`, `background_task` | 同步执行命令或作为可追踪的后台任务 |
| **代码智能** | `lsp`, `lsp_list` | 诊断、跳转定义、引用、重命名、悬停、补全 ([文档](docs/lsp.zh.md)) |
| **数据库** | `db_schema`, `db_query` | 浏览表、列、外键；执行只读 SQL/Redis 查询 |
| **记忆** | `memory_search`, `memory_read`, `memory_write`, `memory_update` | 持久化知识库——跨会话回忆教训和主题 |
| **调试** | `debug` | 启动/附加、断点、单步、检查变量、计算表达式 |
| **协作** | `ask_user`, `spawn_agent` | 提问澄清、扇出并行子 Agent |
| **任务跟踪** | `task_create`, `task_append`, `task_update`, `task_list` | 多步骤工作的结构化待办列表 |
| **可扩展** | `skill`, `install_skill`, `mcp_search`, `install_mcp_server`, … | 运行时发现和安装 Skills 与 MCP 服务器 |

### Monaco 编辑器，LSP 原生

内置编辑器是 [Monaco](https://microsoft.github.io/monaco-editor/)（VS Code 背后的引擎），连接真实的 Language Server Protocol 服务器。Agent 看到的诊断、补全和符号信息与你完全一致。即使没有配置 LSP，tree-sitter 装饰也能提供语法高亮。

### 交互式编辑与冲突解决

当 AI 提出文件修改时，修改会落入一个可编辑的预览面板。如果文件在 Agent 上次读取后已在磁盘上变化，**冲突 UI** 让你协调 AI 修改与你的修改——绝不静默覆盖。脏文件跟踪在各会话间清晰标记未保存的工作。

### 多标签会话

并发会话标签页，每个拥有独立上下文。会话以 JSON 持久化，下次启动时恢复。

### 多面板工作区

会话列表、聊天、文件树 + 编辑器、后台任务控制台、状态栏，全在一个窗口。三种布局模式——专注聊天、分屏、纯文件——可拖拽分隔条自由调节。

### Shell 模式

专用的输入模式用于内联运行 Shell 命令，支持 ANSI 渲染、Tab 补全、历史导航和后台任务日志实时流。

### Git 集成

文件变更追踪、Diff 查看、暂存、提交与推送、提交历史、本地/远程分支列表、创建和切换分支、Worktree 感知的分支管理。

### 数据库集成

打开项目时自动从 `.env` 文件和 `docker-compose.yml` 发现数据库连接。支持 5 种原生驱动（PostgreSQL、MySQL、SQLite、Redis、MongoDB），对缺少纯 Go 实现的驱动通过桥接模式调用 Node.js/Python。Agent 可以浏览数据库架构（表、列、外键）并执行只读查询（SELECT、SHOW、EXPLAIN）——在工具层、驱动层和权限层三层强制只读。

### 知识库与记忆

持久化、自我演进的知识库，跨会话存活。存储教训（bug 和根因）、主题（架构和模式）以及原始笔记。全文搜索（FTS5）让 Agent 在遇到类似问题时回忆过往经验。

### Skills & MCP

- **Skills** — 支持 [SKILL.md](https://github.com) 标准。从 GitHub 仓库自动发现和加载可复用的 Agent 工作流。
- **MCP** — [Model Context Protocol](https://modelcontextprotocol.io) 支持，通过 stdio JSON-RPC 扩展 Agent 能力（数据库、浏览器、Web 搜索等）。

### 调试器

通过 DAP（Debug Adapter Protocol）支持启动和调试程序——设置断点、检查变量、单步执行、计算表达式，全部在内置调试器 UI 中完成。

### 子 Agent 并发

内置 TaskRunner 以并行 goroutine 调度子 Agent——适合大规模代码搜索和多文件重构。

### 权限安全

每个工具调用经过权限管线——硬性规则 + 安全模型双重验证，防止未授权或破坏性操作。

---

## 安装

从 [Releases](https://github.com/RedTeaLab/monika/releases) 下载对应平台的安装包，或从源码构建：

### 前置依赖

| 平台 | 依赖 |
|------|------|
| **macOS** | Go 1.25+, Node.js 18+, Xcode Command Line Tools |
| **Windows** | Go 1.25+, Node.js 18+, WebView2 |

### 安装 Wails v3 CLI

```bash
# 安装与项目匹配的 CLI 版本 (v3.0.0-alpha.78)
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha.78
```

### 从源码构建

```bash
git clone https://github.com/RedTeaLab/monika.git
cd monika

# 1. 安装前端依赖
cd frontend && npm install && cd ..

# 2. 生成 Wails 绑定 (Go 类型 → TypeScript)
wails3 generate bindings -ts
node -e "require('fs').copyFileSync('build/barrel_index.ts','frontend/bindings/monika/index.ts')"

# 3a. 开发模式 (热重载)
wails3 dev

# 3b. 或构建独立可执行文件
cd frontend && npm run build && cd ..
go build -o monika .
# macOS: ./monika
# Windows: .\monika.exe
```

### 配置模型提供商

启动 Monika，打开 **设置 → 模型提供商** 交互式配置。也可以手动创建 `~/.monika/config.yaml`：

```yaml
model_provider: deepseek
model: deepseek-chat
model_providers:
  deepseek:
    name: deepseek
    base_url: https://api.deepseek.com
    api_key: sk-xxx
    wire_api: openai
```

---

## 支持的模型提供商

任何兼容 OpenAI API 的端点都能开箱即用——只需设置 `wire_api: openai` 并将 `base_url` 指向对应 API：

| 提供商 | `base_url` | 默认模型 |
|--------|------------|----------|
| DeepSeek | `https://api.deepseek.com` | `deepseek-chat` |
| OpenAI | `https://api.openai.com` | `gpt-4o` |
| Anthropic Claude | via OpenAI 兼容端点 | `claude-sonnet-4-5` |
| Google Gemini | via OpenAI 兼容端点 | `gemini-2.0-flash` |
| 本地 (Ollama, LM Studio, …) | `http://localhost:xxxx` | — |

---

## 架构

```
monika/
├── main.go                # Wails 入口，嵌入前端，连接所有服务
├── frontend/              # React 18 + TypeScript + Tailwind CSS v4
│   └── src/
│       ├── App.tsx        # 根组件，dockview 面板布局
│       ├── store/         # 单一 Zustand Store，全部应用状态
│       └── components/    # UI 组件
├── internal/
│   ├── agent/             # Agent 循环、流式传输、上下文压缩、子 Agent 调度
│   ├── api/               # Wails 服务: App, SessionManager, FileService, EventBus, DBManager
│   ├── bootstrap/         # 从配置初始化 Provider
│   ├── config/            # YAML/JSON 配置加载 (~/.monika/ + .monika/)
│   ├── dap/               # Debug Adapter Protocol 客户端
│   ├── dbbridge/          # Node.js/Python 数据库驱动桥接脚本
│   ├── dbdiscovery/       # 从 .env、docker-compose 自动发现数据库
│   ├── engines/           # Provider 适配器 + Skill + MCP 引擎
│   ├── lsp/               # Language Server Protocol 客户端 + LSP 工具
│   ├── memory/            # 持久化知识库 (lessons, topics, raw)
│   ├── permission/        # 工具权限管线 (硬性规则 + 安全模型)
│   ├── platform/          # 平台相关工具 (通知、托盘等)
│   ├── prompt/            # System Prompt 构建
│   ├── tool/              # 工具接口 + 注册表 + 内置工具
│   ├── update/            # 自更新逻辑
│   └── version/           # 版本信息 (构建时通过 ldflags 注入)
└── pkg/
    ├── dbdriver/          # 数据库驱动接口 + 5 个原生驱动 (PostgreSQL, MySQL, SQLite, Redis, MongoDB)
    ├── engine/            # 公共 Engine 接口 + 注册表
    ├── openai/            # OpenAI 兼容 SSE 流式客户端
    ├── modelsdev/         # models.dev 模型目录获取
    └── gitutil/           # Git 工具函数
```

### Engine 模式

每个引擎实现 `pkg/engine.Engine` 接口，通过 `init()` + `engine.Register()` 自注册。Provider 引擎额外实现 `StreamChat` 和 `ListModels`。配置中的 `wire_api` 字段决定使用哪个引擎适配器。

### Tool 模式

工具实现 `Name()` / `Description()` / `Parameters()` / `Execute()` 接口，通过可组合的注册函数（`RegisterDefaults`, `RegisterTasks`, `RegisterSpawnAgent` 等）灵活组合。

---

## 开发

```bash
# 测试
go test ./...

# 静态分析
go vet ./...

# 格式化
gofmt -w .

# 前端构建
cd frontend && npm run build

# 重新生成 Wails 绑定 (修改 Go API 类型后)
wails3 generate bindings -ts
node -e "require('fs').copyFileSync('build/barrel_index.ts','frontend/bindings/monika/index.ts')"

# 依赖整理
go mod tidy
```

---

## 贡献

欢迎提交 Issue 和 Pull Request。开发细节和架构决策请参考 [AGENTS.md](AGENTS.md)。

## License

[MIT License](LICENSE) © 2025 RedTeaLab

第三方组件：

| 组件 | 协议 |
|------|------|
| [Wails](https://wails.io) | MIT |
| [Monaco Editor](https://microsoft.github.io/monaco-editor/) | MIT |
| [tree-sitter](https://tree-sitter.github.io/) | MIT |
| [dockview](https://dockview.dev) | MIT |
| [React](https://react.dev) | MIT |
| [zustand](https://zustand.docs.pmnd.rs) | MIT |
| [LXGW WenKai](https://github.com/lxgw/LxgwWenKai) | SIL OFL 1.1 |
| [Maple Mono NF](https://github.com/subframe7536/Maple-font) | SIL OFL 1.1 |

## 致谢

Monika 站在巨人的肩膀上。这些项目塑造了我们的思考，在很多情况下为我们今天看到的功能提供了直接灵感：

- **[VS Code](https://github.com/microsoft/vscode)** — 定义了现代开发体验的编辑器。
- **[oh-my-pi](https://github.com/can1357/oh-my-pi)** — 一个能力出众的终端 AI Agent，拥有深度 LSP 集成和子 Agent 编排，向我们展示了 Agent 表面可以做到什么。
- **[OpenCode](https://github.com/sst/opencode)** — 一个开源 AI 编程 Agent，其清晰的架构和提供商无关的设计启发了我们的方案。

开源建立在开源之上——我们很高兴成为这一传统的一部分。
