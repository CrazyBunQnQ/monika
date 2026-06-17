# 记忆主动检索与自更新机制 — 设计文档

> 日期：2026-06-17
> 作用域：internal/memory, internal/tool/builtin, internal/agent, internal/api, main.go
> 前置文档：`docs/memory-system-design.md`

---

## 一、问题陈述

### 1.1 现状缺陷

记忆系统已实现三层架构（存储/管理/进化层），但存在两个影响实用性的核心缺陷：

**缺陷 A — 记忆检索触发范围过窄**

`main.go:242-243` 的 system prompt 约束：

```
Before using any web search, MCP tools, or asking the user, you MUST first call
memory_search(query) to check the local knowledge base.
```

该约束实测有效——LLM 在 web 搜索前确实会调 memory_search。但触发条件仅限"web 搜索 / MCP 工具 / 问用户前"，**正常编码任务（grep / 读文件 / 编辑）根本不命中**。大量已存储的 lessons/topics 在最该被使用的时刻未被检索。

**缺陷 B — profile.md / knowledge.md 无更新机制**

这两个被冻结注入到 system prompt 的核心文件，其唯一更新入口是 `ArchiveHook.OnArchive`（`hook.go:15`），而 `OnArchive` 仅由前端 `/memory` 命令通过 `TriggerMemorySummarize`（`app.go:647`）触发。`app.go:604-609` 的归档触发代码被注释。

**后果**：用户不主动发 `/memory`，profile/knowledge 永不更新。

### 1.2 目标

1. 将记忆检索从"单点触发"扩展为"任务生命周期闭环"，让 LLM 在正常编码任务中主动使用记忆
2. 为 profile/knowledge 建立可靠的后台更新机制，不依赖用户手动命令
3. 全程保护 prefix cache——profile/knowledge 移出 system prompt，改用 hook 机制注入

---

## 二、方案选型（已确认）

| 决策点 | 选定方案 | 理由 |
|--------|----------|------|
| 检索驱动方式 | **Prompt 扩展触发范围**（不搞系统自动注入） | 现有 prompt 约束证明有效，问题在范围窄而非不遵从 |
| 查重责任 | **LLM 自主查重**（先 search 再决定 update/write） | 合并质量 LLM 语义理解 > bm25 规则；工具保持简单 |
| profile/knowledge 注入 | **移出 system prompt，改为 session 级 hook 注入** | system prompt 完全稳定 → 跨 session cache 命中 |
| profile/knowledge 更新 | **复用懒归档机制**（1h idle 自动触发） | 无需新建调度器，覆盖崩溃场景 |
| 更新时机 | **session 归档时**（显式 + 懒归档） | 缓存友好，异步不影响当前会话 |

---

## 三、架构变更总览

```
变更前：
┌─────────────────────────────────────────────────┐
│ system prompt (每轮重建，prefix cache 易失效)     │
│   ├── agent 基础提示                              │
│   ├── ── BuildMemoryBlock() ──                   │ ← profile/knowledge 冻结注入
│   │       <global_memory> profile + knowledge    │
│   │       <project_memory> knowledge             │
│   ├── context-summary / task-list                │
│   └── tools 定义                                  │
├─────────────────────────────────────────────────┤
│ messages (对话历史)                               │
└─────────────────────────────────────────────────┘

变更后：
┌─────────────────────────────────────────────────┐
│ system prompt (完全稳定，跨 session cache 命中)   │
│   ├── agent 基础提示                              │
│   ├── 记忆闭环指令（BEFORE/DURING/AFTER）         │
│   └── tools 定义                                  │
├─────────────────────────────────────────────────┤
│ messages                                         │
│   ├── msg0: <memory_block> (SessionStart 注入)   │ ← hook 注入，session 级
│   │       profile + knowledge                    │
│   └── msg1+: 对话历史                            │
└─────────────────────────────────────────────────┘
```

---

## 四、详细设计

### 4.1 工具层补全（§2 已确认）

#### 4.1.1 KBFile.Snippet 字段修复

**问题**：`store.go:477` 的 `scanKBFiles` 将 SQL 查出的 snippet（`snippet(file_fts,...)` / `substr(content,1,200)`）扫入局部变量后丢弃。`memory_search` 工具只返回 title+tags，LLM 无法判断相关性。

**改动**：

1. `internal/memory/types.go` — `KBFile` 结构体新增字段：
   ```go
   Snippet string `json:"snippet,omitempty"`
   ```

2. `internal/memory/store.go:473-495` — `scanKBFiles` 将已读出的 snippet 赋值给字段：
   ```go
   f.Snippet = snippet
   ```
   （第 479 行 rows.Scan 已读出 snippet 到局部变量，只需加一行赋值）

3. `internal/tool/builtin/memory_search.go:66-76` — 输出格式增加 snippet 行：
   ```
   1. **CORS 配置修复** [project/lesson] confidence: high
      path: wiki/lessons/... | chars: 580
      tags: cors, wails, frontend
      snippet: ...Wails v3 dev 模式下前端运行在 localhost:5173，未在配置中...
   ```

**成本**：零额外查询——snippet 数据本就在 SQL 结果集里。

#### 4.1.2 memory_read（新增）

**定位**：读单条记忆全文。`memory_search` 只返回 snippet，要看全文用这个。

**文件**：`internal/tool/builtin/memory_read.go`（新建）

| 项 | 设计 |
|----|------|
| Name | `memory_read` |
| Parameters | `path`（必填，来自 search/index 结果）、`scope`（选填，默认 project） |
| 实现 | 直接复用 `KBStore.ReadFile(scope, path)`（store.go:303），已有路径校验（防 `..` 穿越） |
| 返回 | 原始 markdown 全文（含 frontmatter） |

#### 4.1.3 memory_update（新增）

**定位**：更新已存在记忆。选项 1 下，LLM 负责合并语义——LLM 自己先 `memory_read` 读全文 → 自己合并 → 传完整新内容写入。工具不做合并，只负责"按 path 覆盖写 + 刷新索引 + 更新时间戳"。

**文件**：`internal/tool/builtin/memory_update.go`（新建）

| 项 | 设计 |
|----|------|
| Name | `memory_update` |
| Parameters | `path`（必填，定位目标）、`content`（必填，**合并后的完整新内容**）、`scope`（选填，默认 project） |
| 实现 | 复用 `WriteFile`，但需先校验 path 存在（不存在返回错误，引导用 `memory_write`） |
| 返回 | 成功/失败 + updated_at |

**校验逻辑**：先 `ReadFile(path)` 探测存在性。不存在则返回错误信息：
```
Memory not found at path '%s'. Use memory_write to create a new memory.
```

**职责划分**：合并质量 LLM 语义理解 >> bm25 规则；工具保持简单，职责单一（只覆盖写）。

#### 4.1.4 memory_write 语义收窄

**改动**：`internal/tool/builtin/memory_write.go`

1. Description 改为：`"Create a NEW memory entry. If a similar memory may already exist, use memory_search first then memory_update."`
2. 写入成功返回追加提示：`"Memory '%s' written. If you intended to update an existing memory, use memory_update instead."`

不改核心逻辑，只改描述引导 LLM 选对工具。

#### 4.1.5 scope 默认值约定

| 工具 | scope 默认 | 原因 |
|------|-----------|------|
| memory_search | `auto`（project + global 合并） | 搜索是模糊的，应跨 scope 检索 |
| memory_read | `project` | 操作具体文件路径，必须明确定位到某个 KB |
| memory_update | `project` | 同上 |
| memory_write | `project`（现有） | 同上 |

read/update/write 操作的是具体文件路径，scope 决定根目录，不能用 auto（auto 是搜索合并策略，无单一根目录）。

#### 4.1.6 工具注册

`internal/tool/builtin/register.go:261` 附近，新增：
```go
r.Register(NewMemoryRead(store))
r.Register(NewMemoryUpdate(store))
```

---

### 4.2 Prompt 闭环改造（§3 已确认）

**改动**：`main.go:236-255`，替换整个 Knowledge Base 区块。

替换后内容（约 35 行）：

```
## Knowledge Base (Memory)

You have access to a self-evolving knowledge base that persists across sessions.
Relevant profile and core knowledge is loaded at the start of each session.

**MEMORY USAGE — mandatory task lifecycle:**

Every task MUST follow this closed loop. Skipping steps degrades quality over time.

1. **BEFORE acting** — memory_search(query) to check for relevant past experience
   (similar problems, conventions, user preferences). If results look relevant,
   memory_read(path) for full content. Apply what you find.
2. **DURING** — execute the task normally.
3. **AFTER completing** — if you learned something worth keeping (a bug root cause,
   a working pattern, a user preference, a project convention):
   - memory_search first to check if a similar memory already exists
   - exists → memory_read full content → merge new insight → memory_update(path, merged)
   - not exists → memory_write(title, content, category)

**Also:** Before any web search, MCP tool, or asking the user, you MUST first call
memory_search — only fall back to external sources if no relevant memory exists.

**Tools:**
- memory_search(query, scope?, category?, limit?) — search; returns title + snippet
- memory_read(path, scope?) — read a single memory's full content
- memory_write(title, content, category, scope?, tags?, confidence?) — create NEW memory
- memory_update(path, content, scope?) — overwrite existing memory with merged content
- memory_index(scope?) — list all memories by category

**Memory types:** lessons (bugs/causes/solutions), topics (architecture/patterns),
knowledge (preferences/constraints/persistent facts).
```

**设计要点**：

| 决策 | 理由 |
|------|------|
| 用编号步骤描述闭环 | 比 prose 更易被 LLM 遵守（结构化 > 散文） |
| "BEFORE / DURING / AFTER" 三段 | 对齐：先搜经验 → 执行 → 总结保存 |
| 显式写出"update vs write"判断分支 | 落实选项 1：LLM 自主查重，决策树写在 prompt 里降低偷懒概率 |
| 保留原 web search 触发规则 | 现状证明有效，不破坏 |
| 工具列表加 snippet 说明 | 让 LLM 知道 search 返回带片段，降低"搜了不够用就放弃"概率 |
| 不加"每轮必搜" | 任务生命周期驱动，不搞死板每轮 |

---

### 4.3 profile/knowledge 注入改造（§4 已确认）

**改动 1**：`internal/memory/inject.go` — `BuildMemoryBlock()` 方法保留，但不再在 system prompt 构建时调用。改为由 session 启动 hook 注入为 msg0。

**改动 2**：`internal/agent/agent_loop.go:1181-1185` — 移除 `BuildMemoryBlock` 注入到 system prompt 的逻辑：
```go
// 删除：
if a.kbStore != nil {
    if block := a.kbStore.BuildMemoryBlock(); block != "" {
        parts = append(parts, block)
    }
}
```

**改动 3**：每次 `buildMessages` 时，在 system prompt 消息之后、filteredMsgs 之前，插入一条独立的 system 消息作为 memory_block。

**设计决策（无状态注入）**：不使用 `memoryBlockInjected` 等状态标记。每次 `buildMessages` 都重新插入。理由：
1. 内容在同一 session 内不变（profile/knowledge 后台更新下次 session 才生效），重复读取无副作用
2. compaction 会截断 messages（CompactionFrom），状态标记会导致截断后 memory_block 永久丢失；无状态方案每次重建自动恢复
3. messages 序列对 LLM provider 一致

注入位置：`buildMessages`（agent_loop.go:1130），在现有 system prompt 消息（messages[0]）append 之后、`messages = append(messages, filteredMsgs...)` 之前：
```go
// 在现有 system prompt 消息 append 之后
if a.kbStore != nil {
    if block := a.kbStore.BuildMemoryBlock(); block != "" {
        messages = append(messages, engine.ChatMessage{
            Role:    "system",
            Content: block,
        })
    }
}
// 然后追加 filteredMsgs
```

**消息角色**：用 `system`。memory_block 是系统注入的上下文（非用户发言、非 assistant 回复）。现有代码已支持多 system 消息（agent_loop.go:1193-1200 在尾部追加 system reminder）。

**消息序列**：
```
messages[0] = system (agent 基础提示 + 闭环指令 + tools，完全稳定)
messages[1] = system (<memory_block>，session 级稳定)
messages[2+] = filteredMsgs (对话历史)
```

**缓存影响分析**：

| 部分 | 变更前 | 变更后 |
|------|--------|--------|
| system prompt 内容 | 每轮可能变（BuildMemoryBlock 拼入） | 完全稳定 |
| system prompt cache | profile/knowledge 变更即失效 | 跨 session 始终命中 ✅ |
| messages[1] | N/A | session 级稳定（同一 session 内不变） |
| messages[1] cache | N/A | 同 session 内命中；跨 session 可能变（可接受） |

---

### 4.4 profile/knowledge 后台更新（§4 已确认）

**核心**：复用现有懒归档机制，归档时触发 `OnArchive` 更新 profile/knowledge。

#### 4.4.1 SessionManager 回调注入

**文件**：`internal/api/session_manager.go`

当前懒归档（172-178 行）只改 status 并 Save。改为检测到新归档时调用回调，不修改 `List()` 签名（避免影响 4+ 个调用点：app.go x2、worktree.go、测试 x2）。

**SessionManager 新增字段**：
```go
type SessionManager struct {
    // ... 现有字段 ...
    OnLazyArchive func(sessionID string)  // 新增：懒归档回调
}
```

**懒归档逻辑改造**（172-178 行）：
```go
if s.Status == StatusPending && s.LastViewedAt != nil {
    if time.Since(*s.LastViewedAt) > time.Hour {
        s.Status = StatusArchived
        sm.Save(s)
        if sm.OnLazyArchive != nil {
            sm.OnLazyArchive(s.ID)  // 新增：通知回调
        }
    }
}
```

保持 `SessionManager` 单一职责——只检测并标记 + 通知，不决定"归档后做什么"。`List()` 签名不变。

#### 4.4.2 app.go 注册回调并触发 hook

**文件**：`internal/api/app.go`

`SetMemoryHook`（643 行）或 session 初始化处，注册回调：
```go
func (a *App) SetMemoryHook(hook *memory.ArchiveHook) {
    a.memoryHook = hook
    // 新增：为所有已创建的 SessionManager 注册懒归档回调
    a.sessionManagerFactory = func(home, projectDir string) *SessionManager {
        sm := NewSessionManager(home, projectDir)
        sm.OnLazyArchive = func(sessionID string) {
            if a.memoryHook == nil {
                return
            }
            sm.Lock()
            s, err := sm.Load(sessionID)
            sm.Unlock()
            if err != nil || s == nil || len(s.Messages) == 0 {
                return
            }
            summary := extractCompactionSummary(s)
            go func() {
                defer func() {
                    if r := recover(); r != nil {
                        fmt.Printf("[memory] OnArchive panic: %v\n", r)
                    }
                }()
                a.memoryHook.OnArchive(context.Background(), memory.ScopeProject, sessionID, summary)
            }()
        }
        return sm
    }
}
```

注意：异步执行 + `defer recover` 防止 hook 内 panic 影响主流程。

#### 4.4.3 显式归档也触发 hook

**文件**：`internal/api/app.go:591` `ArchiveSession`

```go
func (a *App) ArchiveSession(projectPath, sessionID string) error {
    // ... 现有 status 改为 archived 的逻辑 ...
    
    // 新增：触发记忆提取
    if a.memoryHook != nil {
        s, _ := sm.Load(sessionID)
        if s != nil && len(s.Messages) > 0 {
            summary := extractCompactionSummary(s)
            go a.memoryHook.OnArchive(context.Background(), memory.ScopeProject, sessionID, summary)
        }
    }
    return nil
}
```

同时删除 `app.go:604-609` 的注释死代码——逻辑已由上面的统一路径覆盖。

#### 4.4.4 数据流

```
用户离开 session → 1h 后前端刷新 List()
  → SessionManager.List() 检测 pending + idle>1h
  → status 改 archived，调用 OnLazyArchive 回调
  → app.go 回调内 go memoryHook.OnArchive(...)
  → ExtractMemories → 更新 profile/knowledge（后台 LLM，异步）
  → 下次 session 启动 → BuildMemoryBlock 读到新内容 → 注入 messages[1]
```

**崩溃恢复**：下次 List() 仍会检测到 idle session 并补归档，不丢数据。

---

## 五、涉及文件清单

| 文件 | 改动类型 | 说明 |
|------|----------|------|
| `internal/memory/types.go` | 修改 | KBFile 加 Snippet 字段 |
| `internal/memory/store.go` | 修改 | scanKBFiles 赋值 Snippet |
| `internal/tool/builtin/memory_search.go` | 修改 | 输出加 snippet 行 |
| `internal/tool/builtin/memory_read.go` | **新增** | memory_read 工具 |
| `internal/tool/builtin/memory_update.go` | **新增** | memory_update 工具 |
| `internal/tool/builtin/memory_write.go` | 修改 | Description + 返回提示 |
| `internal/tool/builtin/register.go` | 修改 | 注册新工具 |
| `main.go` | 修改 | system prompt 替换（§3） |
| `internal/agent/agent_loop.go` | 修改 | 移除 system prompt 注入，改为 messages[1] 无状态注入 |
| `internal/api/session_manager.go` | 修改 | 新增 OnLazyArchive 回调字段；懒归档时调用 |
| `internal/api/app.go` | 修改 | SetMemoryHook 注册回调；ArchiveSession 触发 hook；删注释死代码 |

---

## 六、错误处理

| 场景 | 处理 |
|------|------|
| memory_read path 不存在 | 返回错误：`Memory not found at path '%s'.` |
| memory_update path 不存在 | 返回错误并引导：`Use memory_write to create a new memory.` |
| memory_read/update path 含 `..` | 由现有 `ReadFile` 路径校验拦截（store.go:305） |
| OnArchive LLM 调用失败 | hook.go:29 已有处理——打印日志，状态设为"归纳失败"，不阻塞 |
| OnArchive panic | `go func()` 异步执行，recover 防崩溃（需在 app.go 触发处加 defer/recover） |
| List() 报错 | 回调不被调用，不影响列表展示 |

---

## 七、测试策略

| 测试项 | 类型 | 验证点 |
|--------|------|--------|
| KBFile.Snippet 赋值 | 单元 | scanKBFiles 后 Snippet 非空 |
| memory_search 输出含 snippet | 单元 | 结果字符串包含 "snippet:" 行 |
| memory_read 正常/不存在/路径穿越 | 单元 | 三种场景返回正确 |
| memory_update 正常/不存在 | 单元 | 不存在时返回引导错误 |
| memory_write 返回含 update 提示 | 单元 | 返回字符串含 "memory_update" |
| BuildMemoryBlock 不在 system prompt | 集成 | buildMessages 的 system message 不含 `<global_memory>` |
| messages[1] 注入 memory_block | 集成 | buildMessages 返回的第二条消息含 `<global_memory>` |
| OnLazyArchive 回调触发 | 单元 | 构造 idle session，List 后 OnLazyArchive 被调用 |
| ArchiveSession 触发 hook | 集成 | 调用后 memoryHook.OnArchive 被调用（mock 验证） |

---

## 八、不在本次范围内

- LLM query 改写（提升 memory_search 精度）——后续优化
- 记忆使用率指标采集——后续观测
- 前端 memory 管理界面——独立需求
- ArchiveHook 内部逻辑改造——保持现状
