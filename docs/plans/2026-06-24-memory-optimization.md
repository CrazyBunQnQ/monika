# 记忆系统全方位优化方案

> 版本：v3.2 | 日期：2026-06-24（v3.2 修复 composeUserMessage 逻辑缺陷、注入点混淆、模板替换等 review 问题）
> 前置文档：`docs/memory-system-design.md` (v2.0 架构设计) + `docs/memory-system-research.md` (调研报告)
> 本文档聚焦：现有实现的 27 个已识别问题 + 4 阶段解决方案

---

## 目录

- [一、问题全景](#一问题全景5-层-27-个问题)
- [二、Phase 1：缓存稳定 + Prompt 重构](#二phase-1缓存稳定--prompt-重构)
- [三、Phase 2：记忆索引 + 自动召回注入](#三phase-2记忆索引--自动召回注入)
- [四、Phase 3：检索质量提升](#四phase-3检索质量提升)
- [五、Phase 4：写入/管理质量](#五phase-4写入管理质量)
- [六、阶段依赖与实施顺序](#六阶段依赖与实施顺序)
- [七、改动文件清单](#七改动文件清单)
- [八、学术与实践依据](#八学术与实践依据)

---

## 零、Monika 生命周期与触发时机定义

> **本文档中所有"触发时机"都基于 Monika 的实际运行模型，而非抽象的"session"概念。**

Monika 没有持久的"session 进程"。实际生命周期：

```
App 启动 (main.go)
  → systemPrompt 构建一次（存在 App.loopOpts 中）      ← ① App 级别，全局唯一
  → 所有 session 共享同一个 systemPrompt

用户点击"新建会话"
  → App.NewSession()                                    ← 只写 JSON 到磁盘，无 Agent Loop

用户发送一条消息
  → App.SendMessage() → startAgentLoop()
    → loop := agent.NewLoop(..., a.loopOpts...)        ← ② 每条消息创建新的 AgentLoop
    → loop.Run(ctx, conv, text)                         ← 运行完即销毁
```

**三种精确触发点**（本文档统一使用这些术语）：

| 术语 | 实际触发点 | 代码位置 | 执行频率 | 用于 |
|------|-----------|---------|---------|------|
| **App 启动** | `main.go` 中构建 systemPrompt 时 | `main.go:285` 附近 | 整个 App 生命周期一次 | 记忆索引构建、自动遗忘/Review 检查 |
| **每条消息** | `App.startAgentLoop` 中，`loop.Run` 入口前 | `app.go:739` | 每次 SendMessage | 自动召回注入（`<recalled-memory>`） |
| **每轮 LLM 调用** | `buildMessages` 构建 messages 数组时 | `agent_loop.go:1124` | 每次工具调用后的 LLM 请求 | buildMessages 拼装 system message + 对话历史（动态内容已在入口注入，此处不再修改） |

**关键区分**：
- AgentLoop 是**每条消息**创建和销毁的，不是 per-session 持久存在的
- systemPrompt 在 `main.go` 里构建一次，存在 `App.loopOpts` 里，所有 session 共享
- "session 启动"在本文档中**不使用**——因为 Monika 中没有明确的 session 启动事件

---

## 一、问题全景（5 层 27 个问题）

以下问题来自三轮调研：学术论文分析、业界工具（Claude Code / Cursor / Devin / Mem0）对比、deepseek-reasonix 源码研究，以及对 Monika 现有代码的逐行审查。

### 第一层：检索质量（7 个问题）

> **核心矛盾**：检索是记忆系统效果的瓶颈（CMU 论文 arXiv:2603.02473：检索方法造成 14-23 个百分点准确率波动，写入策略仅 3-8 个百分点），而当前检索能力是最薄弱的环节。

| # | 问题 | 代码位置 | 影响 |
|---|------|---------|------|
| R1 | **无向量检索**：只有 FTS5(BM25 词频) + LIKE 子串匹配，完全没有 embedding 语义召回 | `store.go:187-191` `searchSingle` | 同义不同词 = 完全 miss |
| R2 | **词汇失配**：会话 A 存"并行写文件"，会话 B 查"同时保存多个文件"，词法检索无法匹配 | 同上 | 召回率低，用户感觉"不智能" |
| R3 | **CJK 二选一**：`containsCJK(query)` 为 true 走 LIKE，false 走 FTS5，永远不同时执行 | `store.go:188-191` | 中英混合查询丢失一半命中路径 |
| R4 | **无 rerank**：只有 BM25 原始排序，无混合重排 | `store.go:205` `ORDER BY bm25(...)` | 精度不足，噪声结果挤掉相关结果 |
| R5 | **searchAuto 无统一排序**：project + global 结果直接拼接，无跨源统一排名 | `store.go:161-184` | 相关性排序差 |
| R6 | **LIKE snippet 不相关**：返回 `substr(content, 1, 200)` 即正文前 200 字，不是匹配位置附近的上下文 | `store.go:238` | 摘要无信息量，LLM 无法判断是否相关 |
| R7 | **FTS 查询过于宽松**：`sanitizeFTSQuery` 将词间用 OR 连接，单字命中即可返回 | `store.go:129-136` | 噪声大，低质量结果挤掉高质量 |

### 第二层：召回/注入（6 个问题）

> **核心矛盾**：检索算法再好，LLM 根本不调用 `memory_search`，一切都是零。论文和业界实践都指向同一结论——不能依赖 LLM 自主召回。

| # | 问题 | 代码位置 | 影响 |
|---|------|---------|------|
| C1 | **无记忆索引**：LLM 被告知"去 memory_search"，但不知道有什么记忆存在。不知道就不搜 | system prompt 中无索引 | 认知盲区 → 不召回 |
| C2 | **100% 依赖 LLM 自主调用**：代码库无任何 prefetch / autoMemory / bootstrap 机制 | 整体架构 | 根本不召回 |
| C3 | **reminder 门槛 bug**：`len(msgs) >= 10` 条件导致前 9 轮（最需要召回的时刻）没有提醒 | `agent_loop.go:1182` | 短会话全程无提醒 |
| C4 | **Memory 段 40 行程序性指令**：详细的 BEFORE/DURING/AFTER 三阶段生命周期描述，LLM 对跨步骤程序性指令遵循不可靠 | `main.go:209-247` | 大量 token 无效果 |
| C5 | **无自动召回注入**：每条消息不自动预载相关记忆，LLM 每次从零开始 | 无此机制 | 冷启动体验差 |
| C6 | **无 context-trigger 检测**：用户提到文件路径、出现错误关键词时不自动触发记忆搜索 | 无此机制 | 错过高价值召回时机 |

### 第三层：System Prompt / 缓存（5 个问题）

> **核心矛盾**：DeepSeek 前缀缓存从第一个 token 开始匹配。system prompt 中的任何变动都会使整个缓存前缀失效。当前 prompt 结构几乎每轮都在变。

| # | 问题 | 代码位置 | 影响 |
|---|------|---------|------|
| P1 | **Current date 在第一行**：`time.Now().Format("2006-01-02")` 在 prompt 最开头，日期变了 → 整个缓存全失效 | `main.go:207` | 每天杀掉几千 token 的缓存 |
| P2 | **动态内容追加到 system message**：task-list、context-summary、reminder 每轮都追加到同一个 system message，使其每轮不同 | `agent_loop.go:1155-1179` | 每轮打断整个对话历史的缓存 |
| P3 | **reminder 第 10 轮突然出现**：前 9 轮无 reminder，第 10 轮追加 → system message 变化 → 缓存断裂 | `agent_loop.go:1182` | 缓存突变 |
| P4 | **Remember 段 12/18 条重复**：`defaultRemember` 的 18 条 NEVER/ALWAYS 中有约 12 条与前文 ToolUsage、Identity、CodeQuality 段重复 | `default.go:309-327` | ~200 token 浪费，稀释每条指令权重 |
| P5 | **Database Schema 异步加载后变**：schema 在后台加载完成后注入 system prompt，首次查询后内容变化 | `main.go:277-282` | 首次查询后缓存断裂 |

### 第四层：写入/管理（6 个问题）

> **核心矛盾**：Survey 论文（arXiv:2603.07670）指出"unbounded memory growth destabilizes long-running agents"。当前系统只增不减、不去重、不消解冲突，长期使用后噪声累积，反噬检索质量。

| # | 问题 | 代码位置 | 影响 |
|---|------|---------|------|
| W1 | **ComputeSimilarity 是假的**：`bm25Score = 0.9 - float64(idx)*0.2`，仅按检索结果排名线性衰减，非真实语义相似度 | `consolidate.go:34` | 合并/去重判断不可靠 |
| W2 | **写入无去重检查**：`WriteFile` 不检查是否已有高度相似的记忆存在 | `store.go:259` WriteFile | 相同知识不同标题重复写入 |
| W3 | **无冲突检测**：矛盾的记忆可共存，系统不会发现 | 无此机制 | 记忆自我矛盾，误导 LLM |
| W4 | **无遗忘/衰减**：记忆永不淘汰，`SoftDelete` 只在 Review 时手动调用 | 无此机制 | 噪声累积 → 检索质量下降 |
| W5 | **Review 不自动执行**：`Review()` 存在但从未被定时调用，且只看最近 7 天 | `review.go:40-66` | 审查机制形同虚设 |
| W6 | **tag 无归一化**：自由文本标签，同一个概念可能存为 `parallel`、`concurrent`、`batch-write`，`tagOverlap` 无法匹配 | `consolidate.go:46-60` | 相似度计算不准 |

### 第五层：架构（3 个问题）

| # | 问题 | 代码位置 | 影响 |
|---|------|---------|------|
| A1 | **无记忆队列机制**：消息处理中途写入的记忆，在同一次 AgentLoop 运行内无法即时生效（需等下次 App 启动重建索引） | 无此机制 | 同一对话中记忆写入"白写" |
| A2 | **ExtractMemories 一次性提取**：无自我提问循环（ProMem 模式），遗漏率高 | `extract.go:28` | 记忆写入质量低 |
| A3 | **双 DB 无统一索引**：global + project 两个独立 SQLite，跨源检索只是拼接，无统一排序 | `store.go:19-24` | 跨域检索质量差 |

---

## 二、Phase 1：缓存稳定 + Prompt 重构

> **解决**：P1-P5, C3, C4
> **目标**：system prompt 在整个 App 生命周期中绝不修改，DeepSeek 前缀缓存全量命中。
> **参考实现**：deepseek-reasonix `internal/control/input.go` Compose 模式

### 2.1 核心原则：静态/动态分离

学习 reasonix 的设计——**system prompt 是缓存稳定前缀，App 生命周期内绝不修改**：

```
当前结构（有问题）：
  message[0] = system: {静态prompt} + {context-summary} + {task-list} + {reminder}
                              ↑稳定              ↑动态            ↑动态        ↑条件动态
  → 每次 LLM 调用 system message 都不同 → 整个对话历史缓存失效

目标结构（缓存友好）：
  message[0] = system: {静态prompt + 记忆索引}          ← App 生命周期内绝不修改
  message[1..N-1] = 对话历史（含动态前插的用户消息）       ← 前缀缓存命中
```

**关键洞察**：动态内容（recalled-memory、task-list、memory-update）在**用户消息进入对话时一次性注入**（`startAgentLoop` 入口），成为用户消息的一部分永久存储在 `conv.Messages` 中。**不是**在 `buildMessages` 中每轮注入——因为 `buildMessages` 在多轮工具调用中被反复调用，最后一轮的 `filteredMsgs` 末尾是 `tool:result` 而非 `user` 消息，无法前插。

### 2.2 改动详情

#### 2.2.1 两个注入阶段的清晰定义

**阶段 A：用户消息入口注入（一次性，在 `startAgentLoop` 中）**

用户消息进入 `conv.Messages` 之前，系统侧一次性注入所有动态前缀。注入后该消息永久存储在对话历史中，后续 `buildMessages` 不再修改它。

**文件**：`internal/api/app.go`（`startAgentLoop` 方法）+ `internal/agent/agent_loop.go`（`runStreaming` / `RunBlocking` 入口）

```go
// runStreaming / RunBlocking 中，userMessage 进入 conv.Messages 之前
if userMessage != "" {
    prefix := ""

    // 1. recalled-memory（Phase 2 新增）
    if a.memSearchFn != nil {
        if recalled := a.memSearchFn(userMessage); recalled != "" {
            prefix += "<recalled-memory>\n" + recalled + "\n</recalled-memory>\n\n"
        }
    }

    // 2. memory-update（Phase 2 新增：drain 记忆队列）
    if a.memQueue != nil {
        if notes := a.memQueue.DrainPending(); len(notes) > 0 {
            prefix += "<memory-update>\n"
            for _, n := range notes {
                prefix += "- " + n + "\n"
            }
            prefix += "</memory-update>\n\n"
        }
    }

    // 3. task-list（从 system message 迁移到这里）
    if a.taskStore != nil && a.sessionID != "" {
        if tasks := a.taskStore.List(a.sessionID); len(tasks) > 0 {
            prefix += "<task-list>\n"
            for _, t := range tasks {
                fmt.Fprintf(&prefix, "- [%s] %s: %s\n", t.Status, t.ID, t.Subject)
            }
            prefix += "</task-list>\n\n"
        }
    }

    userMessage = prefix + userMessage
}

// 然后进入 conv.Messages（原有逻辑不变）
conv.Messages = append(conv.Messages, engine.ChatMessage{
    Role: "user", Content: userMessage,
})
```

**阶段 B：buildMessages 简化（不再注入动态内容）**

**文件**：`internal/agent/agent_loop.go`

`buildMessages` 不再追加 task-list、reminder 到 system message。也不再做 `composeUserMessage`。它只做两件事：

```go
func (a *AgentLoop) buildMessages(conv *Conversation) []engine.ChatMessage {
    var messages []engine.ChatMessage

    // 1. 静态 system message（含 compaction summary，见下方说明）
    if a.systemPrompt != "" || summaryContent != "" {
        var parts []string
        if a.systemPrompt != "" {
            parts = append(parts, a.systemPrompt)  // 已在启动时完成 {{WorkingDirectory}} 替换
        }
        if summaryContent != "" {
            // compaction summary 保留在 system message 中：
            // 它在整个会话中变化极少（仅在 compaction 发生时变一次），
            // 接受这一次性的缓存断裂，比 task-list 每轮变好得多
            parts = append(parts, "<context-summary>\n"+summaryContent+"\n</context-summary>")
        }
        messages = append(messages, engine.ChatMessage{
            Role: "system", Content: strings.Join(parts, "\n\n"),
        })
    }

    // 2. 对话历史原样追加（动态前插已在入口完成）
    messages = append(messages, filteredMsgs...)

    return messages
}
```

**compaction summary 保留在 system message 的原因**：它在整个会话中最多变化 1-2 次（仅在自动/手动 compaction 触发时），不像 task-list 每轮都变。把它移到用户消息入口会导致后续轮次看不到它，而它需要在整个会话后保持可见。

#### 2.2.2 移除条件 reminder

**文件**：`internal/agent/agent_loop.go`

删除 `len(msgs) >= 10` 条件 reminder（行 1182-1190）。该 reminder 的内容（"memory_search FIRST" 等）要么：
- **方案 A（推荐）**：删除——这些规则已在 system prompt 静态部分有完整描述，reminder 只是缩略重复
- **方案 B**：如果确实需要运行时提醒，固定写入 system prompt 静态部分（但内容要去重，见 2.2.4）

#### 2.2.3 Date 移出 system prompt、`{{WorkingDirectory}}` 替换提前、Schema 延迟加载

**文件**：`main.go` + `internal/agent/agent_loop.go`

三个改动：

**① 仅移除 date，保留 OS/Shell/WorkingDirectory**

当前 `main.go:207` 把 date、OS、WorkingDirectory、Shell 放在同一个字符串里。其中 OS Version、Shell、WorkingDirectory 是**静态的**（App 生命周期内不变），应保留在 system prompt 中。只移除 date：

```go
// 当前（main.go:207）
fmt.Sprintf("Current date: %s\nOS Version: %s\nWorking directory: {{WorkingDirectory}}\nShell: %s",
    time.Now().Format("2006-01-02"), runtime.GOOS, shellName)

// 改为（去掉 date 行）
fmt.Sprintf("OS Version: %s\nWorking directory: {{WorkingDirectory}}\nShell: %s",
    runtime.GOOS, shellName)
// date 不再注入 system prompt；LLM 需要时可自己 date 命令获取
```

**② `{{WorkingDirectory}}` 替换在 main.go 完成一次性**

当前 `agent_loop.go:1159` 每次 `buildMessages` 都执行 `strings.ReplaceAll(a.systemPrompt, "{{WorkingDirectory}}", normalized)`。改为在 `main.go` 构建 systemPrompt 时一次性替换，`buildMessages` 直接使用已替换的字符串：

```go
// main.go 中，构建 systemPrompt 后
normalized := strings.ReplaceAll(cwd, "\\", "/")
systemPrompt = strings.ReplaceAll(systemPrompt, "{{WorkingDirectory}}", normalized)
// 此后 systemPrompt 不再含 {{WorkingDirectory}}，buildMessages 无需替换
```

**③ Database Schema 改为延迟注入**

当前 `main.go:277-282` 在 system prompt 构建时注入 schema（或"loading"占位）。问题是 schema 异步加载完成后会变化。改为：不在 system prompt 中注入 schema，LLM 使用 `db_schema` 工具时自然获取（工具结果出现在对话中）。如果需要在首次消息时提示有数据库，在用户消息入口注入：

```go
// startAgentLoop 中，仅首次消息前插（通过 session 标记去重）
if !session.DBNotified && dbMgr != nil && dbMgr.HasDatabases() {
    prefix += "<database-schema-available>\nThis project has connected databases. Use db_schema to inspect them.\n</database-schema-available>\n\n"
    session.DBNotified = true
}
```

#### 2.2.4 Prompt 内容去重精简

**文件**：`internal/prompt/default.go`、`internal/prompt/deepseek.go`

**Remember 段**——删除与前文重复的条目：

```
保留（独有规则）：
- NEVER pass the entire file as new_string — edit only the lines that need to change
- NEVER hardcode secrets (API keys, passwords, tokens)
- ALWAYS read with file_read before editing with file_edit — never edit blind
- ALWAYS check if you already read a file before reading it again
- Before modifying shared code, grep for ALL references and verify no callers break
- When in doubt, do the smallest thing that works

删除（已在其他段出现）：
- NEVER read entire files blindly — grep first       [ToolUsage 已有]
- NEVER generate or guess URLs                        [Identity 已有]
- NEVER force push to main/master                     [SafetyBoundaries 已有]
- NEVER revert changes you did not make               [SafetyBoundaries 已有]
- NEVER ask "Should I proceed?"                       [ResponseStyle 已有]
- ALWAYS use absolute file paths                      [Identity 已有]
- ALWAYS prefer editing existing files                [ToolUsage 已有]
- ALWAYS check MCP tools before bash                  [ToolUsage 已有]
- ALWAYS run lint/typecheck after completing          [CodeQuality 已有]
- Prioritize technical accuracy over validating beliefs [Identity 已有]
- STRICTLY follow all rules in <project_rules>        [Identity 已有]
```

**Memory 段**——从 40 行压缩到 ~8 行：

```
当前（40 行，程序性指令为主）：
  详细描述 BEFORE/DURING/AFTER 三阶段生命周期
  "Your FIRST tool call for every task MUST be memory_search"
  → LLM 跨步骤程序性指令遵循不可靠，大量 token 无效果

目标（~8 行，事实描述为主）：
  ## Knowledge Base (Memory)

  You have a persistent knowledge base. The index below lists saved memories from
  previous sessions — use memory_read(path) when one looks relevant to the current
  task. After completing a task, save new insights with memory_write, or
  memory_update an existing entry.

  # Memory Index
  1. [lesson] Title (path) tags: ...
  2. [topic] Title (path) tags: ...
  ...
```

### 2.3 验证标准

- [ ] system message 在整个 App 生命周期（含多轮工具调用、跨 session）中文本完全不变
- [ ] 同一 App 进程中，第一个 session 第一轮和第十个 session 第一轮的 system message 内容完全一致
- [ ] task-list 更新后，system message 不变（变动在用户消息中）
- [ ] DeepSeek API 返回的 `cached_tokens` 在第二轮起显著大于 0
- [ ] system prompt 总 token 数减少 ~300（去重 + 精简）

### 2.4 前端显示过滤

动态内容注入到用户消息后，原始 `user` 消息内容会变成：
```
<recalled-memory>...</recalled-memory>

<memory-update>...</memory-update>

<task-list>...</task-list>

<database-schema-available>...</database-schema-available>

用户的实际消息
```

这些 XML 块是给 LLM 看的，**不能渲染到前端的用户消息气泡中**。

**当前问题**：`loadSessionMessages`（`store/index.ts:1848-1858`）直接把 `m.content` 原样推入 Message，`MessageBubble.tsx:671` 直接渲染 `{content}`。`queue_item_started`（`store/index.ts:2283`）同样直接用 `item.text`。流式发送时 `ChatInput` 的 `sendMessage` 也直接发送原始文本。

**需要过滤的 3 个位置**：

#### ① loadSessionMessages — 从磁盘/历史加载时过滤

**文件**：`frontend/src/store/index.ts:1848`

```typescript
// 当前
if (m.role === 'user') {
    result.push({
        id: crypto.randomUUID(),
        role: 'user',
        content: m.content || '',  // ← 原样使用，XML 块会显示
        ...
    })
}

// 改为
if (m.role === 'user') {
    result.push({
        id: crypto.randomUUID(),
        role: 'user',
        content: stripTransientBlocks(m.content || ''),  // ← 过滤 XML 块
        ...
    })
}
```

#### ② queue_item_started — 队列消息显示时过滤

**文件**：`frontend/src/store/index.ts:2283`

```typescript
// 当前
content: item.text,  // ← 可能含 XML 块

// 改为
content: stripTransientBlocks(item.text),
```

#### ③ 实时发送 — 流式发送时用户看到的内容

用户在输入框发消息时，前端先显示原始文本，然后后端注入 XML 块。这里有两种策略：

- **方案 A（推荐）**：前端发送时立即显示原始文本（不含 XML），后端注入的 XML 只存在于 `conv.Messages` 和磁盘持久化中。加载历史时通过 `stripTransientBlocks` 过滤。流式过程中的首轮显示本身就是干净的（用户输入框直接发的内容不含 XML）。
- **方案 B**：后端在 `startAgentLoop` 中注入 XML 后，通过 stream event 通知前端更新显示文本。复杂且不必要。

推荐方案 A：**前端发送时显示原始文本即可，无需额外处理。只有从历史加载时才需要过滤。**

#### stripTransientBlocks 实现

**文件**：`frontend/src/lib/stripTransientBlocks.ts`（新建）

```typescript
// stripTransientBlocks.ts
// 从用户消息内容中移除系统注入的临时 XML 块，只保留用户实际输入。

const TRANSIENT_BLOCK_PATTERNS = [
    /<recalled-memory>[\s\S]*?<\/recalled-memory>\s*/g,
    /<memory-update>[\s\S]*?<\/memory-update>\s*/g,
    /<task-list>[\s\S]*?<\/task-list>\s*/g,
    /<context-summary>[\s\S]*?<\/context-summary>\s*/g,
    /<database-schema-available>[\s\S]*?<\/database-schema-available>\s*/g,
]

export function stripTransientBlocks(content: string): string {
    let result = content
    for (const pattern of TRANSIENT_BLOCK_PATTERNS) {
        result = result.replace(pattern, '')
    }
    return result.trim()
}
```

**设计要点**：
- 用正则匹配 `<tag>...</tag>` 块及其后空行，全部移除
- 保留用户实际输入的文本（XML 块之后的部分）
- 学习 reasonix 的 `StripComposePrefixes`（`internal/control/input.go:44`）模式
- 如果未来新增其他注入块类型，只需在 `TRANSIENT_BLOCK_PATTERNS` 数组中添加

**注意**：`compaction_summary` 消息（`name === 'compaction_summary'`）已被前端单独处理为 `role: 'compaction'`，不走 `role: 'user'` 路径，因此无需在 `stripTransientBlocks` 中处理。

---

## 三、Phase 2：记忆索引 + 自动召回注入

> **解决**：C1, C2, C5, A1
> **目标**：LLM 不需要主动搜索就能看到相关记忆；消息处理中途写入的记忆即时生效。
> **参考实现**：reasonix `memory.Compose` + `memory.Queue` + Mem0 Proactive Memory

### 3.1 记忆索引折叠进 system prompt

**文件**：`main.go` + `internal/memory/store.go`

学习 reasonix 的 `memory.Compose(base, memSet)`，在 **App 启动**时一次性构建索引：

```go
// main.go，kbStore 初始化后
if kbStore != nil {
    memIndex := kbStore.BuildIndex("auto", 50)  // scope=auto, maxEntries=50
    if memIndex != "" {
        systemPrompt += "\n\n# Memory Index\n\n" + memIndex
    }
}
```

```go
// store.go 新增
func (s *KBStore) BuildIndex(scope string, limit int) (string, error) {
    files, err := s.ListFiles(scope, "")
    if err != nil || len(files) == 0 {
        return "", nil
    }
    // 按 updated_at DESC 排序，取最近的 limit 条
    sort.Slice(files, func(i, j int) bool {
        return files[i].UpdatedAt.After(files[j].UpdatedAt)
    })
    if len(files) > limit {
        files = files[:limit]
    }

    var b strings.Builder
    for i, f := range files {
        fmt.Fprintf(&b, "%d. [%s] %s (%s)", i+1, categoryLabel(f.Category), f.Title, f.Path)
        if len(f.Tags) > 0 {
            fmt.Fprintf(&b, " tags: %s", strings.Join(f.Tags, ", "))
        }
        b.WriteString("\n")
    }
    return b.String(), nil
}
```

**关键设计**：
- 索引在 **App 启动**时生成，整个 App 生命周期不变 → 不破坏缓存
- 只含标题 + tag + path，不含正文 → 控制在 ~2KB 以内
- 新写入的记忆通过记忆队列（3.3）在**下一条消息**即时生效（不依赖 App 重启）
- App 重启时索引自然更新（含所有已写入磁盘的记忆）

### 3.2 自动召回注入（每条消息）

> **注意**：此处的注入逻辑已在 Phase 1 的 2.2.1 阶段 A 中定义（`runStreaming`/`RunBlocking` 入口的 prefix 拼装）。本节只补充 `memSearchFn` 回调的注册方式。

**文件**：`internal/agent/agent_loop.go`（新增字段和 option）+ `main.go`（注册回调）

```go
// AgentLoop 新增字段
type AgentLoop struct {
    ...
    memSearchFn func(query string) string  // 记忆搜索回调
}

// 新增 option
func WithMemSearchFn(fn func(query string) string) LoopOption {
    return func(a *AgentLoop) { a.memSearchFn = fn }
}
```

```go
// main.go 中注册回调
memSearchFn := func(query string) string {
    results, err := kbStore.Search(query, "auto", 3)
    if err != nil || len(results) == 0 {
        return ""
    }
    var b strings.Builder
    for _, r := range results {
        fmt.Fprintf(&b, "- **%s** [%s] path: %s\n  snippet: %s\n",
            r.Title, r.Category, r.Path, r.Snippet)
    }
    return b.String()
}
loopOpts = append(loopOpts, agent.WithMemSearchFn(memSearchFn))
```

回调在 Phase 1 的 2.2.1 阶段 A 中被调用（`if a.memSearchFn != nil { ... prefix += recalled-memory ... }`）。

**关键设计**：
- 搜索结果挂到**用户消息前缀**（不是新 system message）→ 不破坏缓存
- 用回调函数避免 `agent` 包直接依赖 `memory` 包 → 无循环依赖
- 搜索结果在同一次 AgentLoop 运行内稳定（同一用户消息的多次 LLM 调用之间不重复搜索）
- store 为 nil 时优雅降级（不注入）

### 3.3 记忆队列（同一次 AgentLoop 运行内变更即时生效）

> **注意**：drain 逻辑已在 Phase 1 的 2.2.1 阶段 A 中定义（`runStreaming`/`RunBlocking` 入口的 prefix 拼装中的 memory-update 部分）。本节只补充队列的数据结构和工具触发方式。

**文件**：新建 `internal/agent/memory_queue.go` + 修改 `internal/tool/builtin/memory_write.go`

学习 reasonix 的 `memory.Queue` + `memoryManager.pending` + `Compose.drainPending`：

```go
// internal/agent/memory_queue.go
package agent

// MemoryQueue 接收 memory 工具写入后的通知，让变更在同一次 AgentLoop 运行内即时生效。
// 下一次 LLM 调用构建 messages 时，drain 出队列内容，前插到用户消息。
// 不碰 system prompt → 不破坏缓存。
type MemoryQueue interface {
    QueueMemory(note string)
}

type memoryQueueImpl struct {
    pending []string
    mu      sync.Mutex
}

func (q *memoryQueueImpl) QueueMemory(note string) {
    q.mu.Lock()
    defer q.mu.Unlock()
    q.pending = append(q.pending, note)
}

func (q *memoryQueueImpl) DrainPending() []string {
    q.mu.Lock()
    defer q.mu.Unlock()
    notes := q.pending
    q.pending = nil
    return notes
}
```

drain 在 Phase 1 的 2.2.1 阶段 A 入口注入中执行（`runStreaming`/`RunBlocking` 入口），不在 `buildMessages` 中。

在 `memory_write.go` / `memory_update.go` 工具中，执行成功后触发队列：

```go
// 通过 context value 传递 queue（同 reasonix 的 WithQueue 模式）
if q, ok := agent.MemoryQueueFromContext(ctx); ok {
    q.QueueMemory(fmt.Sprintf("Saved memory \"%s\" in %s", title, path))
}
```

### 3.4 Context-Trigger 检测（可选增强）

监控用户消息中的信号，自动触发额外记忆搜索：

```go
// 在 memSearchFn 中增加 trigger 逻辑
func(query string) string {
    results := basicSearch(query)

    // 文件路径引用触发
    if paths := extractFilePaths(query); len(paths) > 0 {
        for _, p := range paths {
            results = append(results, basicSearch(p)...)
        }
    }

    // 错误关键词触发
    if containsErrorKeywords(query) {
        results = append(results, basicSearch("error debug fix "+extractErrorContext(query))...)
    }

    return dedupAndFormat(results)
}
```

### 3.5 验证标准

- [ ] LLM 在 system prompt 中能看到记忆索引（标题 + tag 列表）
- [ ] 用户发送消息后，`<recalled-memory>` 块出现在用户消息前缀中
- [ ] 消息处理中途 `memory_write` 后，同一次 AgentLoop 运行内的下一次 LLM 调用出现 `<memory-update>` 块
- [ ] system message 在记忆写入后仍然不变
- [ ] store 为 nil 时系统正常工作（不注入，不报错）

---

## 四、Phase 3：检索质量提升

> **解决**：R1-R7
> **目标**：一旦 LLM 开始用记忆（Phase 2 保证），检索必须找到正确的东西。
> **学术依据**：CMU 论文——hybrid+rerank 比单一方法高 14-23 个百分点

### 4.1 混合检索（BM25 ∪ 向量 ∪ LIKE）

**文件**：`internal/memory/store.go` + 新建 `internal/memory/vector.go`

```go
// store.go 新增
func (s *KBStore) SearchHybrid(query string, scope string, limit int) ([]KBFile, error) {
    // 1. BM25 召回 (top 2*limit)
    lexicalHits := s.searchFTS(query, scope, limit*2)

    // 2. LIKE 召回（CJK 查询时同时执行，不再二选一）
    var likeHits []KBFile
    if containsCJK(query) {
        likeHits = s.searchLike(query, scope, limit)
    }

    // 3. 向量召回 (top 2*limit)
    var semanticHits []KBFile
    if s.hasEmbeddings(scope) {
        semanticHits = s.searchVector(query, scope, limit*2)
    }

    // 4. 合并去重
    candidates := mergeAndDedup(lexicalHits, likeHits, semanticHits)

    // 5. Rerank
    if len(candidates) > limit {
        candidates = s.rerank(query, candidates, limit)
    }
    return candidates, nil
}
```

**向量索引实现**：

```go
// internal/memory/vector.go

// Embeddings 表结构（SQLite）
// CREATE TABLE IF NOT EXISTS embeddings (
//     file_id    INTEGER NOT NULL,
//     scope      TEXT NOT NULL,
//     embedding  BLOB NOT NULL,         -- 1536维 float32 序列化
//     model      TEXT NOT NULL DEFAULT 'text-embedding-3-small',
//     created_at TEXT NOT NULL,
//     FOREIGN KEY (file_id) REFERENCES file_index(id),
//     UNIQUE(file_id)
// );

type EmbeddingProvider interface {
    Embed(ctx context.Context, text string) ([]float32, error)
}

// 写入时生成 embedding（WriteFile / UpdateFile 调用）
func (s *KBStore) ensureEmbedding(scope string, fileID int, content string, provider EmbeddingProvider) {
    vec, err := provider.Embed(ctx, content)
    if err != nil {
        return  // best-effort，失败不影响写入
    }
    blob := serializeFloat32(vec)
    s.dbFor(scope).Exec("INSERT OR REPLACE INTO embeddings (file_id, scope, embedding, created_at) VALUES (?, ?, ?, ?)",
        fileID, scope, blob, time.Now().UTC().Format(time.RFC3339))
}

// 向量检索
func (s *KBStore) searchVector(query string, scope string, limit int) ([]KBFile, error) {
    qVec, err := s.embedProvider.Embed(ctx, query)
    if err != nil {
        return nil, err
    }
    // 加载所有 embedding（文件数通常 <1000，内存计算可行）
    rows := s.dbFor(scope).Query("SELECT file_id, embedding FROM embeddings")
    // 计算余弦相似度，取 top-limit
    ...
}
```

**Embedding Provider 选项与接线方式**：

embedding provider 需要通过现有的 provider 配置体系接入。KBStore 当前不持有任何 LLM/embedding 能力，需要新增注入：

```go
// main.go 中，创建 kbStore 后注入 embedding provider
if kbStore != nil {
    // 从已配置的 provider 中选取支持 embedding 的（优先 OpenAI，其次其他）
    if embedProvider := createEmbeddingProvider(pr.Providers); embedProvider != nil {
        kbStore.SetEmbeddingProvider(embedProvider)
    }
}

// createEmbeddingProvider 遍历已配置的 providers，找到第一个支持 embedding 的
func createEmbeddingProvider(providers map[string]engine.ProviderEngine) memory.EmbeddingProvider {
    for id, p := range providers {
        if p.SupportsEmbedding() {  // ProviderEngine 需新增此方法
            return &providerEmbedAdapter{engine: p, model: p.DefaultEmbeddingModel()}
        }
    }
    return nil  // 无可用 provider → KBStore 降级为纯词法检索
}
```

Provider 选项：
- OpenAI text-embedding-3-small (1536d, 推荐，OpenAI provider 原生支持)
- DeepSeek embedding API（如果后续支持）
- 本地模型（如后续集成 ONNX runtime）

**降级策略**：无 embedding provider 时（`kbStore.embedProvider == nil`），`SearchHybrid` 自动退化为 BM25+LIKE 双路检索，不报错。

### 4.2 废弃 CJK 二选一

```go
// 当前（store.go:187-191）— CJK 和 FTS5 互斥
func (s *KBStore) searchSingle(query, scope string, limit int) ([]KBFile, error) {
    if containsCJK(query) {
        return s.searchLike(query, scope, limit)  // 只走 LIKE
    }
    return s.searchFTS(query, scope, limit)       // 只走 FTS5
}

// 改为 — 两者都执行，结果合并
func (s *KBStore) searchSingle(query, scope string, limit int) ([]KBFile, error) {
    var results []KBFile
    // FTS5 总是执行
    results = append(results, s.searchFTS(query, scope, limit)...)
    // CJK 查询额外走 LIKE（互补）
    if containsCJK(query) {
        results = append(results, s.searchLike(query, scope, limit)...)
    }
    return mergeAndDedup(results), nil
}
```

### 4.3 轻量 Rerank

**文件**：新建 `internal/memory/rerank.go`

```go
// rerank.go
// 对合并后的候选列表统一评分排序。
// 评分 = BM25 排名分(0-1) * 0.4 + 向量相似度(0-1) * 0.4 + tag重叠(0-1) * 0.2

type rerankCandidate struct {
    File           KBFile
    BM25Rank       float64  // 1 - rank/total，排名越靠前分越高
    VectorSim      float64  // 余弦相似度，0-1
    TagOverlap     float64  // Jaccard 相似度，0-1
}

func (s *KBStore) rerank(query string, candidates []KBFile, limit int) []KBFile {
    // 解析查询关键词用于 tag 匹配
    queryTags := extractKeywords(query)

    scored := make([]rerankCandidate, len(candidates))
    for i, c := range candidates {
        scored[i] = rerankCandidate{
            File:       c,
            BM25Rank:   1.0 - float64(i)/float64(len(candidates)),
            VectorSim:  c.vectorScore,  // 来自 searchVector 填充
            TagOverlap: tagJaccard(queryTags, c.Tags),
        }
    }

    // 加权评分
    for i := range scored {
        scored[i].score = scored[i].BM25Rank*0.4 + scored[i].VectorSim*0.4 + scored[i].TagOverlap*0.2
    }

    // 排序取 top-limit
    sort.Slice(scored, func(i, j int) bool {
        return scored[i].score > scored[j].score
    })

    result := make([]KBFile, 0, limit)
    for i := 0; i < limit && i < len(scored); i++ {
        result = append(result, scored[i].File)
    }
    return result
}
```

### 4.4 searchAuto 统一排序

```go
// store.go
func (s *KBStore) searchAuto(query string, limit int) ([]KBFile, error) {
    // 合并 project + global 后统一 rerank（而非简单拼接）
    proj := s.searchHybrid(query, ScopeProject, limit*2)
    glob := s.searchHybrid(query, ScopeGlobal, limit*2)
    merged := mergeAndDedup(proj, glob)
    return s.rerank(query, merged, limit), nil
}
```

### 4.5 Snippet 优化

LIKE 搜索的 snippet 改为匹配位置附近的上下文窗口：

```go
// 当前（store.go:238）：substr(f.content, 1, 200) — 正文前200字
// 改为：找到匹配位置，返回前后各 ~80 字的上下文
func buildContextualSnippet(content string, matchPos int, window int) string {
    start := max(0, matchPos-window)
    end := min(len(content), matchPos+window)
    snippet := content[start:end]
    if start > 0 { snippet = "..." + snippet }
    if end < len(content) { snippet += "..." }
    return snippet
}
```

### 4.6 FTS 查询优化

```go
// 当前（store.go:129-136）：词间用 OR，过于宽松
// 改为：默认 AND（提高精度），无结果时回退 OR（保证召回）
func (s *KBStore) searchFTS(query, scope string, limit int) ([]KBFile, error) {
    // 先尝试 AND
    andQuery := buildFTSQuery(query, "AND")
    results := s.execFTS(andQuery, scope, limit)
    if len(results) > 0 {
        return results, nil
    }
    // 无结果时回退 OR
    orQuery := buildFTSQuery(query, "OR")
    return s.execFTS(orQuery, scope, limit)
}
```

### 4.7 验证标准

- [ ] "并行写文件" 能检索到标题为 "同时保存多个文件" 的记忆（向量检索命中）
- [ ] 中英混合查询同时走 FTS5 和 LIKE
- [ ] 检索结果的 snippet 包含匹配关键词附近的上下文
- [ ] searchAuto 结果按统一相关性排序（非简单 project 优先拼接）
- [ ] 无 embedding provider 时降级为 BM25+LIKE 双路，不报错

---

## 五、Phase 4：写入/管理质量

> **解决**：W1-W6, A2
> **目标**：存进去的东西是高质量的、不重复的、不矛盾的、会过期的。
> **可与 Phase 3 并行开发**

### 5.1 写入去重 + 冲突检测

**文件**：`internal/memory/store.go`

```go
// WriteFile 改进 — 写入前检查相似记忆
func (s *KBStore) WriteFile(scope, category, title, content string, tags []string, confidence string) error {
    // 归一化 tags（5.5）
    tags = normalizeTags(tags)

    // 检查已有记忆
    queryText := title + " " + content
    existing, _ := s.SearchHybrid(queryText, scope, 3)
    for _, e := range existing {
        sim := s.computeRealSimilarity(content, e)
        if sim >= 0.85 {
            return fmt.Errorf("similar memory exists: %s (path: %s, similarity: %.2f). Use memory_update instead",
                e.Title, e.Path, sim)
        }
        if sim >= 0.5 && detectContradiction(content, e.Content) {
            // 标记冲突（不阻止写入，但记录冲突关系）
            s.markConflict(scope, e.Path, title)
        }
    }

    // 正常写入
    ...
}
```

### 5.2 相似度计算修正

**文件**：`internal/memory/consolidate.go`

废弃假的 `bm25Score = 0.9 - float64(idx)*0.2`：

```go
// 当前（consolidate.go:26-44）— 假相似度
func (s *KBStore) ComputeSimilarity(candidate ExtractCandidate) ([]SimilarityResult, error) {
    results, _ := s.Search(candidate.Title+" "+candidate.Content, "", 3)
    for idx, r := range results {
        bm25Score := 0.9 - float64(idx)*0.2  // ← 这不是相似度！
        sims = append(sims, SimilarityResult{File: r, Score: bm25Score + ...})
    }
}

// 改为 — 真实相似度
func (s *KBStore) ComputeSimilarity(candidate ExtractCandidate) ([]SimilarityResult, error) {
    queryText := candidate.Title + " " + candidate.Content
    results, _ := s.SearchHybrid(queryText, ScopeAuto, 5)

    var sims []SimilarityResult
    for _, r := range results {
        score := s.computeRealSimilarity(candidate.Content, r)
        // score = 向量余弦(0.5) + tag Jaccard(0.3) + 标题关键词重叠(0.2)
        sims = append(sims, SimilarityResult{File: r, Score: score})
    }
    return sims, nil
}
```

### 5.3 自动遗忘机制

**文件**：新建 `internal/memory/decay.go`

```go
// internal/memory/decay.go

type DecayPolicy struct {
    ArchiveAfterDays int     // 默认 90 天未更新 → archived
    DeleteAfterDays  int     // 默认 180 天 → trash
    ConfidenceMultiplier map[string]float64  // low=2.0(加速衰减), medium=1.0, high=0.5(减速)
}

func DefaultDecayPolicy() DecayPolicy {
    return DecayPolicy{
        ArchiveAfterDays: 90,
        DeleteAfterDays:  180,
        ConfidenceMultiplier: map[string]float64{
            "high":   0.5,
            "medium": 1.0,
            "low":    2.0,
        },
    }
}

// RunDecay 执行一轮衰减清理。
// 触发时机：**App 启动**时检查上次 decay 时间（只看时间戳，快），超过 24 小时则后台异步执行。
func (s *KBStore) RunDecay(scope string, policy DecayPolicy) (archived, deleted int, err error) {
    files, err := s.ListFiles(scope, "")
    if err != nil {
        return 0, 0, err
    }
    now := time.Now()

    for _, f := range files {
        if f.Status != "active" {
            continue
        }
        multiplier := policy.ConfidenceMultiplier[f.Confidence]
        if multiplier == 0 {
            multiplier = 1.0
        }

        ageDays := now.Sub(f.UpdatedAt).Hours() / 24
        effectiveAge := ageDays * multiplier

        if effectiveAge >= float64(policy.DeleteAfterDays) {
            s.SoftDelete(scope, f.Path)
            s.LogEntry(scope, "自动遗忘", fmt.Sprintf("%s 已过期删除 (age=%.0fd, conf=%s)", f.Path, ageDays, f.Confidence))
            deleted++
        } else if effectiveAge >= float64(policy.ArchiveAfterDays) {
            s.SetFileStatus(scope, f.Path, "archived")
            s.LogEntry(scope, "自动归档", fmt.Sprintf("%s 已归档 (age=%.0fd, conf=%s)", f.Path, ageDays, f.Confidence))
            archived++
        }
    }
    return archived, deleted, nil
}
```

**触发时机**：
- **App 启动**时检查上次 decay 时间（记录在 `.index/last_decay.txt`），超过 24 小时则后台异步执行
- 也可加一个 24 小时 ticker 在后台持续检查（`time.NewTicker(24 * time.Hour)`）

### 5.4 自动 Review 增强

**文件**：`internal/memory/review.go`

当前 `Review()` 的问题：
1. 从不自动调用
2. 只看最近 7 天
3. 无冲突消解

改进：

```go
// review.go 增强

// AutoReviewIfNeeded 检查是否需要执行 review，如需要则后台异步执行。
// 触发条件：距上次 review 超过阈值（默认 7 天），且有新的记忆写入。
func (s *KBStore) AutoReviewIfNeeded(ctx context.Context, llm ReviewLLM, scope string) {
    lastReview := s.getLastReviewTime(scope)
    if time.Since(lastReview) < 7*24*time.Hour {
        return  // 不到 7 天，跳过
    }
    // 后台异步执行
    go func() {
        s.ExecuteReview(ctx, llm, scope)
        s.setLastReviewTime(scope, time.Now())
    }()
}

// Review 改进 — 范围扩大到所有 active 记忆（不限 7 天）
func (s *KBStore) Review(ctx context.Context, llm ReviewLLM, scope string) (*ReviewResult, error) {
    files, err := s.ListFiles(scope, "")
    // 不再过滤 weekAgo，审查所有 active 记忆
    var active []KBFile
    for _, f := range files {
        if f.Status == "active" {
            active = append(active, f)
        }
    }
    // 如果记忆数量过大（>50），分批审查
    if len(active) > 50 {
        active = active[:50]  // 优先审查最近更新的
    }
    ...
}
```

### 5.5 Tag 归一化

**文件**：新建 `internal/memory/tags.go`

```go
// internal/memory/tags.go

// canonicalTags 将常见变体映射到规范形式。
// 写入时和检索时都调用 normalizeTags，确保一致性。
var canonicalTags = map[string]string{
    // I/O 类
    "parallel": "parallel-io", "concurrent": "parallel-io",
    "concurrent-write": "parallel-io", "batch-write": "batch-io",
    "batch-save": "batch-io",
    // 测试类
    "test": "testing", "tests": "testing", "unit-test": "testing",
    // 错误处理类
    "bug": "bugfix", "fix": "bugfix", "error": "error-handling",
    // 前端类
    "frontend": "ui", "react": "react-frontend",
    // ... 随使用积累扩展
}

func normalizeTags(tags []string) []string {
    seen := make(map[string]bool)
    var result []string
    for _, t := range tags {
        canonical := canonicalTags[strings.ToLower(strings.TrimSpace(t))]
        if canonical == "" {
            canonical = strings.ToLower(strings.TrimSpace(t))
        }
        if !seen[canonical] {
            seen[canonical] = true
            result = append(result, canonical)
        }
    }
    return result
}
```

在 `WriteFile` 和 `SearchHybrid`（查询关键词提取）中调用 `normalizeTags`。

### 5.6 提取质量提升（ProMem 自我提问）

**文件**：`internal/memory/extract.go`

当前 `ExtractMemories` 一次性提取，遗漏率高。改进为二阶段：

```go
// extract.go 增强

func ExtractMemories(ctx context.Context, llm ExtractionLLM, scope, sessionID, compactionSummary string) (*ExtractResult, error) {
    // Stage 1: 标准提取（当前逻辑）
    result, err := extractStage1(ctx, llm, scope, sessionID, compactionSummary)
    if err != nil {
        return nil, err
    }

    // Stage 2: 自我提问（ProMem 模式）
    gapFacts := extractStage2SelfQuestion(ctx, llm, compactionSummary, result)
    result.Candidates = append(result.Candidates, gapFacts...)

    return result, nil
}

// extractStage2SelfQuestion 基于 Stage 1 的提取结果，反问"还遗漏了什么"
func extractStage2SelfQuestion(ctx context.Context, llm ExtractionLLM, summary string, stage1 *ExtractResult) []ExtractCandidate {
    // 构建已提取知识的摘要
    extractedSummary := formatExtractedCandidates(stage1.Candidates)

    prompt := `基于以下对话总结和已提取的知识，思考：
1. 未来用户可能问什么问题，但当前提取的知识无法回答？
2. 有哪些重要的上下文细节被遗漏了？

对话总结：` + summary + `

已提取的知识：
` + extractedSummary + `

返回 JSON 格式的补充知识（与 Stage 1 相同格式）。如果没有遗漏，返回空数组。`

    resp, err := llm.Chat(ctx, prompt, "")
    if err != nil {
        return nil  // best-effort
    }
    // 解析返回的 candidates
    ...
}
```

### 5.7 验证标准

- [ ] 写入与已有记忆相似度 ≥ 0.85 时被拒绝，提示用 memory_update
- [ ] ComputeSimilarity 返回的 Score 在 0-1 之间，有语义含义
- [ ] 90 天未更新的 active 记忆自动标记为 archived
- [ ] tag "parallel" 和 "concurrent" 归一化为同一个 "parallel-io"
- [ ] ExtractMemories 的 Stage 2 能发现 Stage 1 遗漏的知识
- [ ] Review 在 **App 启动**时自动检查是否需要执行（距上次超过 7 天）

---

## 六、阶段依赖与实施顺序

```
Phase 1 (缓存 + Prompt)                    ← 必须先做
    │ system prompt 稳定后才能安全注入动态内容
    ↓
Phase 2 (索引 + 自动注入)                  ← 依赖 Phase 1
    │ LLM 能看到记忆后，检索质量才有意义
    ↓
Phase 3 (检索质量)          Phase 4 (写入/管理)
    │                        │              ← 可并行
    │ 向量检索独立于注入      │ 管理质量独立于检索
    ↓                        ↓
    └────────────────────────┘
              ↓
         全面验证
```

| 阶段 | 预估工作量 | 解决问题数 | 依赖 |
|------|-----------|-----------|------|
| Phase 1 | 2-3 天 | 7 (P1-P5, C3, C4) | 无 |
| Phase 2 | 2-3 天 | 4 (C1, C2, C5, A1) | Phase 1 |
| Phase 3 | 4-5 天 | 7 (R1-R7) | Phase 2（需要检索被使用后才有反馈） |
| Phase 4 | 3-4 天 | 7 (W1-W6, A2) | 无（可与 Phase 3 并行） |

---

## 七、改动文件清单

### Phase 1

| 文件 | 改动 |
|------|------|
| `internal/agent/agent_loop.go` | buildMessages 简化（只含静态 system prompt + compaction summary）；删除 reminder 条件追加；入口注入 recalled-memory/memory-update/task-list |
| `main.go` | date 移出 system prompt；`{{WorkingDirectory}}` 替换提前到启动时；schema 延迟注入 |
| `internal/prompt/default.go` | Remember 段去重（删 12 条）；Memory 段精简到 ~8 行 |
| `internal/prompt/deepseek.go` | 同步适配 |
| `frontend/src/store/index.ts` | `loadSessionMessages` 和 `queue_item_started` 增加 `stripTransientBlocks` 过滤 |
| `frontend/src/lib/stripTransientBlocks.ts` | **新建**：移除系统注入的 XML 块，保留用户实际输入 |

### Phase 2

| 文件 | 改动 |
|------|------|
| `main.go` | App 启动时构建记忆索引折叠进 system prompt；注册 memSearchFn 回调 |
| `internal/agent/agent_loop.go` | 新增 memSearchFn/memQueue 字段和 option；入口注入逻辑（与 Phase 1 阶段 A 合并实现） |
| `internal/agent/memory_queue.go` | **新建**：MemoryQueue + DrainPending |
| `internal/tool/builtin/memory_write.go` | 写入后触发队列 |
| `internal/tool/builtin/memory_update.go` | 更新后触发队列 |
| `internal/memory/store.go` | 新增 BuildIndex 方法 |

### Phase 3

| 文件 | 改动 |
|------|------|
| `internal/memory/store.go` | SearchHybrid；废弃 CJK 二选一；searchAuto 统一排序；FTS AND/OR 回退；snippet 优化；SetEmbeddingProvider 注入 |
| `internal/memory/vector.go` | **新建**：embedding 存储 + 向量检索 |
| `internal/memory/rerank.go` | **新建**：轻量重排 |
| `pkg/engine/provider.go` | ProviderEngine 新增 `SupportsEmbedding()` / `DefaultEmbeddingModel()` 接口方法 |

### Phase 4

| 文件 | 改动 |
|------|------|
| `internal/memory/consolidate.go` | 相似度计算修正（废弃假 bm25Score） |
| `internal/memory/store.go` | WriteFile 增加去重检查 |
| `internal/memory/decay.go` | **新建**：自动遗忘机制 |
| `internal/memory/review.go` | 自动触发 + 范围扩大 |
| `internal/memory/tags.go` | **新建**：tag 归一化 |
| `internal/memory/extract.go` | 二阶段自我提问提取 |

---

## 八、学术与实践依据

### 论文

| 论文 | 核心贡献 | 对应方案 |
|------|---------|---------|
| CMU: *Diagnosing Retrieval vs Utilization Bottlenecks* (arXiv:2603.02473) | 检索方法是主瓶颈（14-23pt）；写入策略影响小（3-8pt）；hybrid+rerank 最优；原始 chunk 存储 ≥ 精炼存储 | Phase 3 全部；Phase 4.1-4.2 |
| Mem0 (arXiv:2504.19413) | 事实抽取 + 冲突消解(add/update/noop) + 语义检索 + 结构化元数据 | Phase 3.1 向量检索；Phase 4.1 去重 |
| Survey: *Memory for Autonomous LLM Agents* (arXiv:2603.07670) | "有记忆 vs 无记忆"差距 > 换模型；write-manage-read 循环；管理环节最易被忽视；unbounded memory growth destabilizes | Phase 4 全部 |
| ProMem (arXiv:2601.04463) | 自我提问式二阶段记忆提取，捕获标准提取遗漏的事实 | Phase 4.6 |
| PASK (arXiv:2604.08000) | 意图感知的主动记忆触发；好的主动记忆必须同时擅长"何时触发"和"何时不触发" | Phase 2.2-2.4 |
| MemGPT (arXiv:2310.08560) | OS 式分层记忆（RAM/disk/cold）；实践落地少 | 架构参考 |
| A-MEM (arXiv:2502.12110) | 记忆间互链（inter-memory linking） | Phase 4.4 Review 的 LinkAdditions |
| Karpathy: LLM Wiki | 增量构建持久 wiki 而非每次 RAG 原始文档 | 整体架构（已有设计） |

### 业界工具参考

| 工具 | 做法借鉴 | 对应方案 |
|------|---------|---------|
| Claude Code | CLAUDE.md + MEMORY.md 全量注入 system prompt；topic 文件按需读 | Phase 2.1 记忆索引折叠 |
| Cursor | `.cursor/rules/*.mdc` 带 paths frontmatter 条件注入 | Phase 2.4 context-trigger |
| Devin | AGENTS.md 每次会话开头全量注入 | Phase 1 system prompt 稳定 |
| deepseek-reasonix | Compose 模式——动态内容挂用户消息；memory.Queue 会话内即时生效；memory.Compose 启动时折叠 | Phase 1 静态/动态分离；Phase 2.3 记忆队列 |
| Mem0 | embedding 语义检索 + 结构化元数据 + scoped 记忆 | Phase 3 向量检索 |

### 核心设计原则总结

1. **缓存稳定优于一切**：system prompt 绝不在 App 生命周期内修改（reasonix 铁律）
2. **自动注入优于自主召回**：不指望 LLM 主动搜索，系统在每条消息入口替它查好（Mem0 Proactive + reasonix Compose）
3. **检索质量优于写入精炼**：投入在 rerank 和 hybrid 上比优化提取 prompt 收益大（CMU 论文）
4. **有界记忆优于无限累积**：记忆必须能遗忘、合并、淘汰（Survey 论文）
