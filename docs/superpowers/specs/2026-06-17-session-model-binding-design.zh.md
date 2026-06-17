# 会话级模型绑定（Session-Scoped Model Binding）

**日期：** 2026-06-17
**状态：** 已通过（设计）

## 问题

当前模型/Provider 选择器是**全局的**。用户打开多个会话并在它们之间切换时，下拉框不能正确反映该会话正在（或应当）使用的模型：

- 会话 A 用 GLM 创建，会话 B 用 DeepSeek 创建。
- 切到 B 并选择 DeepSeek，改变的是**全局**选择器。
- 切回 A 时，下拉框还停在 DeepSeek —— 在 A 里发消息就会用 DeepSeek 而不是 GLM。

后端 `Session` 结构体其实**已经**按会话持久化了 `Model` + `Provider`（`internal/api/session_manager.go:30-31`），`NewSession` 在创建时也会设置它们。缺陷完全在前端：`selectedProvider`/`selectedModel` 是全局单值，而 `switchSessionTab` 恢复了消息/token，却**没有**恢复模型。

## 目标

每个会话绑定自己的 provider + 模型。切换激活会话时，下拉框恢复该会话绑定的模型。修改下拉框只影响当前会话，绝不动全局默认。

## 决策

| 决策点 | 选择 |
|----------|------|
| 下拉框改动的范围 | 只绑定到**当前激活会话**；全局默认不变 |
| 新建会话的默认模型 | 使用**全局默认**（`~/.monika/config.json` 的 `model_provider`/`model`） |
| 架构 | **单一数据源** —— `selected*` 是当前会话绑定的派生视图，而不是独立的全局状态 |

## 架构（方案 B）

### 后端

无需改 schema —— `Session.Model` / `Session.Provider` 已存在。

**新增 RPC —— `SetSessionModel`**（`internal/api/app.go`）：
```go
func (a *App) SetSessionModel(projectPath, sessionID, providerID, model string) error
```
- 加载会话，设置 `s.Provider` / `s.Model`，保存。复用 `SetSessionPinned` / `RenameSession` 已有的 `getSessionManager` + 加锁模式。

**`SendMessage` 持久化本次使用的模型**（`internal/api/app.go`，约 752 行）：
- 保存代码块当前设置了 `s.Messages`、`s.TokenCount`……，但**漏掉了** `s.Model`/`s.Provider`。补上：
  ```go
  s.Provider = providerID
  s.Model = model
  ```
  （这两个值已经是函数参数）。这样磁盘上的会话记录会与实际使用的模型保持同步，包括下次发送前通过下拉框做的中途换模型。

其他后端无需改动。`GetDefaultModel` / `SetDefaultModel` / `PersistSelection` 仍然是全局默认机制，保持不变。

### 前端

**把当前全局 `selected*` 混在一起的两个概念拆开：**

| 状态 | 含义 | 使用者 |
|-------|---------|-----------|
| `sessionBindings: Record<sessionId, { provider: string; model: string }>` | **数据源** —— 每个会话绑定的模型，加载会话时填充。 | — |
| `selectedProvider` / `selectedModel` | **派生** —— 等于 `sessionBindings[activeSessionId]`，缺失时回退到 `default*`。 | ModelPicker、ChatArea、ChatInput（发送/压缩） |
| `defaultProvider` / `defaultModel` | 全局默认（供新会话用）。启动时从 `GetDefaultModel` 加载一次。 | SessionList（新建会话）、ModelsTab（设置页） |

`selected*` 不再是独立的全局状态 —— 它永远是当前会话绑定的映射（或没有绑定时回退到全局默认）。

**加载会话时填充 `sessionBindings`。** 凡是从后端加载会话、且能拿到 `session.model` / `session.provider` 的地方，都写入 `sessionBindings[id]`：
- `openSessionTab`（`store/index.ts:777`）
- `restoreSessionTabs`（`store/index.ts:998`）
- `loadSessionList`（`store/index.ts:1054`）
- `pushSubagentOverlay`（`store/index.ts:943`）

用一个辅助函数保持 DRY：
```ts
const applySessionBinding = (id: string, provider?: string, model?: string) => { ... }
```

**切换 tab 时恢复。** `switchSessionTab`（`store/index.ts:905`）已经恢复了 `messages`、`tokenCount`、`tokenMax`。扩展它，从 `sessionBindings[id]` 恢复 `selectedProvider`/`selectedModel`（回退 `default*`）。`setSelectedModel` 里已有的 token-max 恢复逻辑也必须在这里执行，让 token 进度条匹配恢复后模型的上下文窗口。

**下拉框改动不再动全局默认。** `ModelPicker.onSelect`（`ModelPicker.tsx:110`）当前调用 `setSelectedProvider` / `setSelectedModel`。把它改指向一个新的 action，该 action：
1. 更新 `sessionBindings[activeSessionId]` 和 `selected*`。
2. 调用 `App.SetSessionModel(projectPath, activeSessionId, provider, model)`。
3. **不**调用 `SetDefaultModel`。

**生成中禁用下拉框。** `ModelPicker` 读取 `generatingSessionIds`，当当前会话正在生成时禁用 provider/模型选择器。当前这一轮的模型在发送时就已固定（`WithModel` 已固化在运行中的 loop 里），所以生成中途切换本来也无法影响本轮 —— 禁用是为了避免误导用户。生成完成后下拉框恢复可用。

**新建会话继续用全局默认。** `SessionList.handleNewSession`（`SessionList.tsx:80`）从 `selectedProvider`/`selectedModel` 改为 `defaultProvider`/`defaultModel`。

**设置页继续管全局默认。** `ModelsTab`（`ModelsTab.tsx:215`）不变 —— 仍是 `SetDefaultModel`，只是现在读写 `default*`。

### Bindings 重新生成

因为 `SetSessionModel` 只用于 Wails 的 Go API，添加后需重新生成 bindings：
```bash
wails3 generate bindings -ts
node -e "require('fs').copyFileSync('build/barrel_index.ts','frontend/bindings/monika/index.ts')"
```

## 行为流程

| 流程 | 行为 |
|------|----------|
| 切到 tab B | `selected*` ← `sessionBindings[B]`（或 `default*`）；token 进度条更新为 B 模型的上下文窗口 |
| 打开/加载会话 | `sessionBindings[id]` ← `session.{provider,model}` |
| 在当前会话改下拉框 | 更新 `sessionBindings[active]` + `selected*`；通过 `SetSessionModel` 持久化；全局默认不变 |
| 生成中改模型 | 当前会话处于 `generatingSessionIds` 时下拉框禁用；生成完成后恢复 |
| 新建会话 | 用 `default*` 创建；创建后它的绑定 = 默认值 |
| 发送消息 | 用 `selected*`（= 当前会话绑定）；`SendMessage` 同时把 model/provider 写回会话 JSON |
| 设置 → 改默认 | 只更新 `default*`（`SetDefaultModel`） |

## 边界情况

- **历史会话**：创建时 `NewSession` 总会设置 model/provider → 加载即得到正确绑定，无需迁移。
- **model/provider 为空的会话**（遗留/极端）：`selected*` 回退到 `default*`；第一次发送会通过 `SendMessage` 的持久化修复回填模型。
- **子会话**（`sub_` / `call_compact_`）：自带模型。绑定恢复用其存储的模型，缺失则回退。不改变它们 spawn 时的继承逻辑。
- **Provider/模型失效**（例如用户删了某个 provider）：`loadProviders` / `loadModelsForProvider` 里已有的回退逻辑会选一个有效的首个选项；此处不变。

## 不在范围内

- 改变子会话/子 agent 在 spawn 时继承模型的方式。
- 按会话设置 temperature 等其他 agent 参数。
- 在会话 tab/列表上加一个模型徽标可视化（可作为后续工作）。
