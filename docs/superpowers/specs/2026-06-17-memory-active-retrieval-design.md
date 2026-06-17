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
Use memory_search/memory_read to look up relevant knowledge on demand.

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
   
   For profile (wiki/profile.md) and core knowledge (wiki/knowledge.md), use
   memory_update with the specific path. These files have character limits
   (profile: 1500, knowledge: 3000) — the tool warns on overflow, then you
   must read back and trim.
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

### 4.3 profile/knowledge 不再注入，改由 LLM 自主检索

**决策**：删除 `BuildMemoryBlock` 在 system prompt 中的注入。profile/knowledge 与 lessons/topics 一样，通过 `memory_search` / `memory_read` 由 LLM 主动检索。

**可行性依据**（已验证）：
- `store.go:147` `Search(query, scope, limit)` 无 category 过滤参数
- 底层 `searchFTS` / `searchLike` 的 SQL 仅过滤 `status != 'trash'`，不过滤 category
- `memory_search` 工具的 `category` 参数实际未传给 store（dead 参数），默认搜全部
- 因此 `memory_search("用户偏好")` / `memory_search("技术栈")` 等查询天然能命中 profile.md / knowledge.md

**改动 1**：`internal/agent/agent_loop.go:1181-1185` — 移除 system prompt 注入：
```go
// 删除整段：
if a.kbStore != nil {
    if block := a.kbStore.BuildMemoryBlock(); block != "" {
        parts = append(parts, block)
    }
}
```

**改动 2**：`internal/memory/inject.go` — `BuildMemoryBlock` 函数保留（`store_test.go:57` 仍有单元测试覆盖），但不再被生产代码调用。后续如确认无其他引用可删除。

**为什么不引入"多 system 消息"方案**（已否决）：
- `sanitizeMessageSequence`（agent_loop.go:1286-1294）硬编码 `append(messages[:1], ...)` 只保留 messages[0]，第二条 system 会被 trim 掉
- Claude 原生 API 明令禁止连续 system 消息
- OpenAI 兼容实现（textgen 等）会后者覆盖前者
- sanitizer 改造属于共享代码风险，且需实测多 provider 兼容性

**§4.2 Prompt 闭环已覆盖此场景**：BEFORE 阶段的 `memory_search(query)` 自然包含 profile/knowledge 内容。LLM 搜到相关 profile/knowledge 后调 `memory_read(path)` 看全文。

**消息序列**（最终，无 memory_block 注入）：
```
messages[0]   = system (基础提示 + 闭环指令 + reminder，完全稳定)
messages[1+]  = filteredMsgs (对话历史，含工具调用)
```

**缓存影响**：

| 部分 | 变更前 | 变更后 |
|------|--------|--------|
| system prompt 内容 | 含 profile/knowledge（变更即失效） | 完全稳定 |
| system prompt cache | profile/knowledge 变更即失效 | 跨 session 始终命中 ✅ |
| profile/knowledge 可见性 | 被动注入（会话内可见） | 主动检索（LLM 调 search 时可见） |

**取舍**：放弃"无脑可见"，换取 cache 稳定 + 实时性 + 零结构改动。profile/knowledge 是否被用到，取决于 §4.2 prompt 闭环的执行——但这正是本次改造的核心目标（让 LLM 主动用记忆）。
---

### 4.4 删除归档提取机制，完全工具化

**决策**：删除整个 ArchiveHook 后台提取链路。profile/knowledge 与其他记忆一样，由 LLM 在 AFTER 阶段主动 memory_write/memory_update。不再有后台/异步/归档触发的提取。

**理由**：
1. §4.3 已把 profile/knowledge 移出 system prompt，会话中写入不影响 cache
2. 后台异步写入有风险（LLM 提取质量不可控、panic 难定位、时序不确定）
3. 完全工具化 = LLM 自主决定 = 实时 + 可控 + 可追溯（每条写入都是 LLM 显式工具调用，对话历史可见）
4. 减少代码量：删除整套 OnArchive → ExtractMemories → CompactKnowledge 链路及相关胶水代码

#### 4.4.1 删除清单

**`internal/memory/hook.go`** — 整文件删除：
- `ArchiveHook` 结构体
- `OnArchive` 方法
- 其内部的 ExtractMemories → Consolidate → WriteFile → CompactKnowledge 全链路

**`internal/api/app.go`**：
- 删除 `memoryHook` 字段（第 77 行）
- 删除 `SetMemoryHook` 方法（第 643-645 行）
- 删除 `TriggerMemorySummarize` 方法（第 647-668 行）
- 删除 `extractCompactionSummary` 辅助函数（第 615-640 行）
- 删除 ArchiveSession 内的注释死代码（第 604-609 行）
- 移除 `import "monika/internal/memory"`（如果该文件不再使用 memory 包）

**`main.go`**：
- 删除 ArchiveHook 构造（第 392-397 行）
- 删除 `appService.SetMemoryHook(hook)`（第 397 行）

**`frontend/src/components/Chat/ChatInput.tsx`**：
- 删除 `/memory` 命令处理（第 688-701 行）
- 删除 `App.TriggerMemorySummarize` 调用

**`internal/memory/compact_knowledge.go`** — 保留：
- `CompactKnowledge` / `CompactProfile` 函数保留，供 memory_update 工具内部使用（见 4.4.2）
- 删除文件中仅供 hook.go 使用的 `CompactionLLM` interface（如确认无其他引用）

**`internal/memory/extract.go`** / **`consolidate.go`** / **`review.go`** — 保留：
- 这些是知识提取核心逻辑，当前无引用但后续可能复用（如未来恢复部分自动化）。本次不删除，避免过度清理。

#### 4.4.2 memory_update 内置字符上限保护

由于不再有后台 CompactKnowledge，LLM 通过 memory_update 写入 profile/knowledge 时可能超出字符上限（knowledge 3000、profile 1500）。

**方案**：memory_update 工具在写入前检查字符数。超限时：

1. **优先**：写入并返回警告信息，提示 LLM 内容已超限，建议调用 memory_read 后手动精简：
   ```
   ⚠️ Content exceeds %s limit (%d/%d chars). Written successfully, but please
   read it back and trim to fit the limit using memory_update.
   ```
   理由：工具不调用 LLM（保持工具无 LLM 依赖原则），让 LLM 自己决定怎么压缩（它的语义理解优于规则压缩）。

2. 字符上限常量复用 `compact_knowledge.go:10-12` 的 `maxKnowledgeChars` / `maxProfileChars`（改为导出：`MaxKnowledgeChars` / `MaxProfileChars`）。

**不自动压缩**：不在工具内调 CompactKnowledge（那需要 LLM 调用，增加复杂度和延迟）。写入放行 + 警告是最简方案，LLM 会自然遵守"看到警告就读回来精简"。

#### 4.4.3 数据流（最终）

```
LLM AFTER 阶段：
  学到用户偏好/画像 → memory_search("profile") 查重 → memory_update("wiki/profile.md", ...)
  学到项目事实/约定 → memory_search("knowledge") 查重 → memory_update("wiki/knowledge.md", ...)
  学到教训/模式 → memory_search 关键词 → memory_write(category) 或 memory_update(path)
  
  ↓ 实时写入，立即生效
  
下一轮 LLM BEFORE 阶段：
  memory_search → 命中刚写入的记忆（实时性保证）
```

无后台任务、无异步、无归档触发。所有写入都是 LLM 显式工具调用，对话历史可见、可追溯。
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
| `internal/agent/agent_loop.go` | 修改 | 移除 system prompt 中 BuildMemoryBlock 注入（§4.3） |
| `internal/memory/hook.go` | **删除** | 整文件删除：ArchiveHook + OnArchive（§4.4） |
| `internal/api/app.go` | 修改 | 删 memoryHook/SetMemoryHook/TriggerMemorySummarize/extractCompactionSummary；删注释死代码 |
| `main.go` | 修改 | 删 ArchiveHook 构造 + SetMemoryHook 调用 |
| `frontend/src/components/Chat/ChatInput.tsx` | 修改 | 删 `/memory` 命令处理 |
| `internal/memory/compact_knowledge.go` | 修改 | 导出 MaxKnowledgeChars/MaxProfileChars 常量供 memory_update 使用 |

---

## 六、错误处理

| 场景 | 处理 |
|------|------|
| memory_read path 不存在 | 返回错误：`Memory not found at path '%s'.` |
| memory_update path 不存在 | 返回错误并引导：`Use memory_write to create a new memory.` |
| memory_read/update path 含 `..` | 由现有 `ReadFile` 路径校验拦截（store.go:305） |
| profile/knowledge 写入超字符上限 | 照常写入 + 返回警告，LLM 自行精简（§4.4.2） |

---

## 七、测试策略

| 测试项 | 类型 | 验证点 |
|--------|------|--------|
| KBFile.Snippet 赋值 | 单元 | scanKBFiles 后 Snippet 非空 |
| memory_search 输出含 snippet | 单元 | 结果字符串包含 "snippet:" 行 |
| memory_read 正常/不存在/路径穿越 | 单元 | 三种场景返回正确 |
| memory_update 正常/不存在 | 单元 | 不存在时返回引导错误 |
| memory_write 返回含 update 提示 | 单元 | 返回字符串含 "memory_update" |
| buildMessages 不含 memory_block | 集成 | buildMessages 的 system message 不含 `<global_memory>` |
| memory_search 可命中 profile/knowledge | 单元 | 写入 profile.md 后，search("用户偏好") 能命中 |
| memory_update 超 profile/knowledge 上限 | 单元 | 写入 profile 超 1500 字符，返回含警告 |
| ArchiveHook 已删除 | 集成 | app.go 编译通过，无 memoryHook/SetMemoryHook 残留引用 |

---

## 八、不在本次范围内

- LLM query 改写（提升 memory_search 精度）——后续优化
- 记忆使用率指标采集——后续观测
- 前端 memory 管理界面——独立需求
- ArchiveHook 内部逻辑——已整体删除（§4.4），不再保留
