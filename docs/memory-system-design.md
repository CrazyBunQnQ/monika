# Monika 自进化记忆系统 — 详细设计文档

> 基于调研报告 (`memory-system-research.md`) 和设计讨论。
> 本设计中引用的第三方项目名称（Hermes、Mem0、MemGPT、Zep/Graphiti、LangMem、A-MEM、ReasoningBank 等）
> 均为其各自所有者的商标，仅用于描述性引用以说明设计思路来源，不暗示 endorsement。
> 本系统为独立实现，不包含任何第三方项目的源代码。
>
> 版本：v2.0 | 日期：2026-06-16

---

## 一、设计哲学

### 1.1 核心理念

```
原始资料 (Raw)  ──LLM 编译──▶  结构化 Wiki  ──注入──▶  Agent 决策
    ▲ 不可变                          ▲ LLM 维护           ▲ frozen snapshot
```

借鉴 Andrej Karpathy 的编译器类比：原始资料是"源代码"（不可变），LLM 将其编译为结构化 wiki（可执行文件），Agent 每次启动时加载编译产物，而非每次都翻原始资料。

### 1.2 设计原则

| 原则 | 来源 | 说明 |
|------|------|------|
| **编译优于检索** | Karpathy | LLM 主动合成知识，而非被动 RAG 原始片段 |
| **有界记忆** | Hermes | knowledge.md 有字符上限，强制信息压缩 |
| **冻结快照注入** | Hermes | 会话开始一次性注入，保护 prefix cache |
| **文件系统优先** | Karpathy | Canonical store 是 markdown 文件，SQLite 是索引层 |
| **自进化闭环** | ReasoningBank + Hermes | 检索→行动→提取→合并→检索... |
| **人类可读** | Monika 原生 | 所有知识库文件人类可直接阅读、编辑、diff |

### 1.3 为什么不用纯向量库/纯 SQLite

- **纯向量库**：黑盒检索，不可 diff，LLM 无法直接"打开看看我记住了什么"
- **纯 SQLite**：结构化好但 LLM 不天然理解数据库，需要额外工具层
- **文件系统 + FTS5 索引**：LLM 原生理解 markdown，人类可读，FTS5 提供毫秒级检索

---

## 二、三层架构

```
┌─────────────────────────────────────────────────────────┐
│  Layer 3: 进化层 (Evolution)                             │
│  后台审查 → 经验蒸馏 → 更新 Wiki → 生成 Skill           │
│  借鉴：ReasoningBank 闭环 + Hermes Background Review     │
├─────────────────────────────────────────────────────────┤
│  Layer 2: 管理层 (Management)                            │
│  Compaction → 事实提取 → 合并/去重 → 淘汰 → 检索        │
│  借鉴：Mem0 Extract-Consolidate + A-MEM 自组织链接       │
├─────────────────────────────────────────────────────────┤
│  Layer 1: 存储层 (Storage)                               │
│  Markdown 文件系统 + SQLite FTS5 全文索引                │
│  借鉴：Karpathy wiki 结构 + Hermes Session Search        │
└─────────────────────────────────────────────────────────┘
```

---

## 三、三层知识库

模拟人脑的知识储备方式，分为三类：

### 3.1 文档库 (Document Library) — `raw/docs/`

**对应认知**：显性知识 — 通过阅读获取的外部信息

| 属性 | 说明 |
|------|------|
| 内容 | 用户上传的文档（PDF/Markdown/纯文本/网页摘要） |
| 写入方 | 用户手动上传（拖拽/粘贴/URL） |
| LLM 角色 | 只读 — 可读取作为证据，但**不修改原始文档** |
| 存储 | `raw/docs/<slug>.md`（上传时自动转为 markdown） |
| 索引 | FTS5 全文索引（内容 + 标题） |
| 示例 | `go-best-practices.md`、`company-style-guide.md` |

### 3.2 代码库 (Code Library) — `raw/code/`

**对应认知**：程序知识 — 对其他代码仓库的理解

| 属性 | 说明 |
|------|------|
| 内容 | LLM 分析外部仓库后生成的架构摘要、API 说明、使用模式 |
| 写入方 | LLM（用户关联仓库后，Agent 自动分析并生成摘要） |
| LLM 角色 | 读写 — 分析仓库并维护摘要 |
| 存储 | `raw/code/<repo-slug>.md` |
| 更新 | 仓库有更新时，用户可触发重新分析 |
| 示例 | `kubernetes-client-go.repo.md`、`react-core.repo.md` |

### 3.3 经验库 (Experience Library) — `wiki/`

**对应认知**：经验知识 — 从实践中学到的教训、模式、认知

| 属性 | 说明 |
|------|------|
| 内容 | Session 归纳的经验教训、主题知识、用户画像 |
| 写入方 | LLM（自动提取 + 后台审查） |
| LLM 角色 | 读写 — 维护整个 wiki |
| 存储 | `wiki/knowledge.md`、`wiki/lessons/`、`wiki/topics/`、`wiki/profile.md` |
| 演化 | 新记忆合并旧记忆、淘汰过时信息 |

---

## 四、目录结构设计

### 4.1 全局 vs 项目级归属判断

```
这条知识换一个项目还有效吗？
  ├── 是 → 全局 kb (~/.monika/kb/)
  └── 否 → 项目 kb (<project>/.monika/kb/)
```

### 4.2 完整目录树

```
~/.monika/kb/                                   # 全局知识库（跨项目通用）
├── SCHEMA.md                                   # wiki 维护规则（程序记忆）
│
├── raw/                                        # 不可变原始资料
│   ├── docs/                                   # 📄 文档库：用户上传
│   │   ├── company-style-guide.md
│   │   └── go-concurrency-patterns.md
│   └── code/                                   # 📦 代码库：外部仓库
│       └── go-standard-library.repo.md
│
├── wiki/                                       # LLM 编译维护的知识
│   ├── index.md                                # 导航目录（快速定位）
│   ├── log.md                                  # 操作日志（情景记忆）
│   ├── knowledge.md                            # 核心语义事实（有界 ~3000 字符）
│   ├── profile.md                              # 用户画像（偏好、风格）
│   ├── topics/                                 # 跨项目通用主题
│   │   ├── sqlite-fts5.md
│   │   └── llm-prompt-engineering.md
│   ├── lessons/                                # 跨项目通用教训
│   │   ├── always-check-nil-before-call.md
│   │   └── prefer-context-over-cancel.md
│   └── .trash/                                 # 软删除暂存
│
├── .index/
│   └── kb.db                                   # SQLite FTS5 索引
│
└── .trash/                                     # 软删除暂存（全局级）


<project>/.monika/kb/                           # 项目知识库（项目专属）
├── SCHEMA.md                                   # 可继承/覆盖全局 SCHEMA
│
├── raw/
│   ├── docs/                                   # 项目专属文档
│   │   └── product-spec.md
│   └── code/                                   # 项目关联仓库
│       └── kubernetes-client.repo.md
│
├── wiki/
│   ├── index.md
│   ├── log.md
│   ├── knowledge.md                            # 项目事实/约定（~2000 字符）
│   ├── topics/                                 # 项目架构/模块/约定
│   │   ├── architecture.md
│   │   ├── conventions.md
│   │   └── dependencies.md
│   ├── lessons/                                # 本项目教训
│   │   ├── 2026-06-16-cors-config-fix.md
│   │   └── deployment-gotchas.md
│   └── .trash/
│
├── .index/
│   └── kb.db
│
└── .trash/
```

### 4.3 路径解析约定

沿用现有 config 模式（`internal/config/config.go`），新增：

```go
// 全局 KB 路径
func GlobalKBPath() string {
    return filepath.Join(config.HomeDir(), ".monika", "kb")
}

// 项目 KB 路径
func ProjectKBPath(projectDir string) string {
    return filepath.Join(projectDir, ".monika", "kb")
}
```

与现有 `.monika/skills/` 和 `.monika/config.json` 同级，保持一致性。

---

## 五、数据模型

### 5.1 Wiki Markdown 文件格式

每个 wiki 文件遵循统一模板：

```markdown
# <标题>

> 类型：semantic | episodic | procedural
> 作用域：global | project:<slug>
> 创建：2026-06-16T10:30:00Z
> 更新：2026-06-16T15:45:00Z
> 置信度：high | medium | low
> 标签：tag1, tag2, tag3
> 关联：[[another-page]] | [text](another-page.md)
> 状态：active | archived | deprecated

<正文内容 — 由 LLM 维护>
```

### 5.2 knowledge.md（有界注入文件）

```markdown
# Core Knowledge

> 更新：2026-06-16T15:45:00Z
> 字符上限：3000（严格强制）
> 压缩策略：新事实到达时，合并/覆盖旧事实；必要时淘汰置信度最低的信息

## 用户偏好
- 语言：中文
- 代码风格：Go idiomatic，偏向简洁
- 沟通：直接、不啰嗦

## 常驻事实
- 项目使用 Go 1.25 + Wails v3
- 数据库优先使用 modernc.org/sqlite（纯 Go，无 CGO）
- 已有的基础设施：compaction、skills、session JSON 存储

## 已知约束
- prefix cache 保护：会话内记忆变更下次会话才生效
- 不生成 URL，除非确认真实存在
```

### 5.3 profile.md

```markdown
# User Profile

> 更新：2026-06-16T10:00:00Z
> 字符上限：1500

## 身份与角色
- 独立开发者，Monika 项目的创建者

## 沟通偏好
- 语言：中文
- 风格：直接、技术导向
- 不喜欢：过度解释、啰嗦

## 技术栈
- 主力语言：Go、TypeScript/React
- 编辑器：VS Code / Monika

## 长期目标
- 构建 Monika 为生产级 AI 代码编辑器
```

### 5.4 lessons 文件

```markdown
# Cors 配置修复

> 类型：episodic
> 作用域：project:monika
> 创建：2026-06-16
> 置信度：high
> 标签：cors, wails, frontend, bug
> 关联：[[topics/wails-v3-patterns]]

## 问题
前端请求被 CORS 策略阻止，表现为 API 调用返回网络错误。

## 根因
Wails v3 dev 模式下前端运行在 localhost:5173，而后端在 localhost:34115。
未在 Wails 配置中设置允许的源。

## 解决方案
在 `wails.json` 的 `frontend:devServerURL` 配置中显式添加 CORS 头。

## 泛化教训
- Wails v3 的 dev/prod 模式网络拓扑不同，需要分别验证 CORS
- 任何跨源请求的 bug 先查 CORS 配置，而非网络层
```

### 5.5 topics 文件

```markdown
# Wails v3 开发模式

> 类型：semantic
> 作用域：global
> 创建：2026-06-16
> 置信度：high
> 标签：wails, go, frontend, desktop
> 关联：[[topics/go-build-constraints]]

## 核心概念
Wails v3 使用 Go 后端 + 前端 webview 架构。开发模式下前端通过 Vite dev server 运行。

## 关键路径
- 后端入口：`main.go`
- 前端入口：`frontend/src/main.tsx`
- 配置：`wails.json`

## 常见陷阱
1. CORS：dev 模式下前后端不同源
2. 构建标签：需要正确的 CGO 配置
3. 资源嵌入：生产模式使用 embed.FS

## 相关教训
- [[lessons/2026-06-16-cors-config-fix]]
```

### 5.6 index.md

```markdown
# Knowledge Base Index

> 最后更新：2026-06-16T15:45:00Z
> 总条目：12 | 活跃：10 | 已归档：2

## 按类型
### 语义 (Semantic)
- [[topics/sqlite-fts5]]
- [[topics/wails-v3-patterns]]
- [[topics/go-concurrency]]

### 情景 (Episodic)
- [[lessons/2026-06-16-cors-config-fix]]
- [[lessons/deployment-gotchas]]

## 按标签
- `wails`: [[topics/wails-v3-patterns]], [[lessons/2026-06-16-cors-config-fix]]
- `go`: [[topics/go-concurrency]], [[topics/sqlite-fts5]]
- `bug`: [[lessons/2026-06-16-cors-config-fix]]

## 最近更新
1. 2026-06-16 — [[lessons/2026-06-16-cors-config-fix]] (新建)
2. 2026-06-15 — [[topics/wails-v3-patterns]] (更新 — 添加 CORS 陷阱)
```

### 5.7 log.md

```markdown
# Operation Log

> 类型：episodic

## 2026-06-16

### 15:45 — 合并记忆
- 将 "CORS bug fix" 合并到 [[topics/wails-v3-patterns]] 常见陷阱
- 新建 [[lessons/2026-06-16-cors-config-fix]]

### 14:30 — Session 归档
- Session `abc123` 归档完成
- 提取 2 条教训，1 条主题更新
- 经验库 size：10 -> 12

### 10:00 — 文档上传
- 上传 `go-concurrency-patterns.md`
- 入 raw/docs/，已生成摘要
```

---

## 六、SQLite FTS5 索引

不存知识正文，只存元数据 + 全文索引，加速检索。

### 6.1 表结构

```sql
-- 文件元数据索引
CREATE TABLE file_index (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    path        TEXT NOT NULL UNIQUE,       -- 相对 kb 根目录的路径
    scope       TEXT NOT NULL,              -- 'global' | 'project:<slug>'
    category    TEXT NOT NULL,              -- 'raw/doc' | 'raw/code' | 'wiki/knowledge' | 'wiki/profile' | 'wiki/lesson' | 'wiki/topic'
    title       TEXT NOT NULL,              -- 文件 H1 标题
    tags        TEXT,                       -- JSON array
    confidence  TEXT,                       -- 'high' | 'medium' | 'low'
    status      TEXT DEFAULT 'active',      -- 'active' | 'archived' | 'deprecated' | 'trash'
    char_count  INTEGER,                    -- 正文字符数
    created_at  TEXT,
    updated_at  TEXT,
    linked_to   TEXT                        -- JSON array of linked paths
);

-- FTS5 全文索引
CREATE VIRTUAL TABLE file_fts USING fts5(
    path,                                   -- 文件路径
    title,                                  -- 标题
    content,                                -- 正文全文（用于搜索）
    content=file_index,
    content_rowid=id
);

-- 触发器：自动同步 FTS
CREATE TRIGGER file_index_ai AFTER INSERT ON file_index BEGIN
    INSERT INTO file_fts(rowid, path, title, content)
    VALUES (new.id, new.path, new.title, ...);
END;
```

### 6.2 索引更新时机

| 事件 | 触发 |
|------|------|
| 文件写入 | `file_index` upsert + FTS 重建该行 |
| 文件删除 | `file_index` 标记 `trash`，FTS 删除该行 |
| 文件重命名 | `file_index` 更新 path |
| 批量重建 | `memory_reindex` 工具触发全量重建 |

---

## 七、核心流程

### 7.1 记忆注入流程（每次会话启动）

```
会话启动
    │
    ├── 读取 ~/.monika/kb/wiki/profile.md
    ├── 读取 ~/.monika/kb/wiki/knowledge.md
    ├── 读取 <project>/.monika/kb/wiki/knowledge.md
    │
    ▼
buildMessages() 注入 <memory> 块到 system prompt
    │
    └── Frozen Snapshot：会话内变更不反映，下次会话才生效
```

**注入格式**：

```xml
<global_memory>
<user_profile>
<profile.md 内容>
</user_profile>

<core_knowledge>
<knowledge.md 内容>
</core_knowledge>
</global_memory>

<project_memory>
<project_knowledge>
<project/knowledge.md 内容>
</project_knowledge>
</project_memory>
```

**在 buildMessages 中的位置**（紧跟 `<context-summary>` 之后）：

```go
// agent_loop.go:buildMessages() — 伪代码
if summaryContent != "" {
    systemPrompt += "\n\n<context-summary>\n" + summaryContent + "\n</context-summary>"
}
// 新增：
memoryBlock := buildMemoryBlock()  // 组装 global + project 两层 memory
if memoryBlock != "" {
    systemPrompt += "\n\n" + memoryBlock
}
```

### 7.2 记忆提取流程（Session 结束时）

#### 触发时机

| 触发方式 | 说明 |
|----------|------|
| **手动命令** | 用户输入 `/memory-summarize`，Agent 即时归纳 |
| **归档触发** | 用户归档 session 时，后台 goroutine 异步提取 |
| **定期提醒** | 每 10 个 session 后，状态栏提示"要整理记忆吗？" |

#### 归档触发流程

```
用户点击"归档 Session"
    │
    ├── 前端：状态栏显示 "归纳中..."
    ├── 后端：go func() 异步执行
    │       │
    │       ├── 1. 收集本 session 的 CompactionSummary（如果存在）
    │       ├── 2. 如果无 summary，用最后 N 轮对话作为输入
    │       ├── 3. 调用 LLM 提取：
    │       │      - 值得记住的事实/决定 → 候选写入 knowledge.md
    │       │      - 值得记住的教训 → 候选写入 lessons/
    │       │      - 新的主题认知 → 候选写入 topics/
    │       ├── 4. 对每条候选记忆执行 FTS 相似度匹配：
    │       │      - 高相似度 → 合并到现有文件
    │       │      - 中相似度 → 新建文件 + 建立链接
    │       │      - 低相似度 → 新建文件
    │       ├── 5. 如果有用户画像相关 → 更新 profile.md
    │       ├── 6. 写入 log.md
    │       └── 7. 更新 index.md
    │
    └── 前端：状态栏更新为 "记忆已更新 ✓"
```

#### 合并策略

借鉴 Mem0 consolidate + A-MEM 链接机制：

```
新记忆 & FTS 检索 Top-3
    │
    ├── score ≥ 0.8："同一件事"
    │   → LLM 判断是 update（更新旧文件）还是 supersede（新文件 + 标记旧文件 deprecated）
    │
    ├── 0.4 ≤ score < 0.8："相关但不同"
    │   → 新建文件，在 Linked 字段添加双向链接
    │   → 更新关联文件的 "关联" 元数据
    │
    └── score < 0.4："全新知识"
        → 直接新建文件
```

**相似度算法**：SQLite FTS5 的 `bm25()` 排序 + 对标题做简单的 Jaccard 标签重叠。不需要向量嵌入，降低依赖。

### 7.3 记忆检索流程

Agent 使用 `memory_search` 工具时：

```
memory_search(query="cors bug fix", scope="auto")
    │
    ├── 1. 确定搜索范围：auto → 先项目，无结果回退全局
    ├── 2. SQLite FTS5 全文检索
    │      SELECT path, title, snippet(file_fts, 2, '<b>', '</b>', '...', 40)
    │      FROM file_fts
    │      WHERE file_fts MATCH 'cors bug fix'
    │      ORDER BY bm25(file_fts, 0, 10, 5)
    │      LIMIT 5
    ├── 3. 返回结果：文件路径 + 标题 + 高亮摘要
    └── 4. Agent 根据需要 read_file 获取完整内容
```

### 7.4 记忆演进流程（后台审查）

借鉴 Hermes Background Review + ReasoningBank 闭环：

```
后台定时任务（每 24h 或每 N 个 session）
    │
    ├── 1. 审查最近入库的记忆（过去 7 天）
    ├── 2. 检查矛盾：同一主题有冲突结论？
    │      → LLM 裁决，标记较旧的为 deprecated
    ├── 3. 检查过时：事实已不再适用？
    │      → 标记为 archived
    ├── 4. 知识升级：多条 lessons 揭示同一模式？
    │      → 提炼为 topic（语义记忆）
    ├── 5. 链接补全：新文件未链接到已有相关文件？
    │      → LLM 分析并建立链接
    ├── 6. knowledge.md 压缩：超出字符上限？
    │      → LLM 压缩：保留高频/高置信度，淘汰低频/过时
    └── 7. 更新 index.md
```

### 7.5 经验 → 技能沉淀

当同一模式的 lesson 出现 ≥ 3 次时，触发技能生成：

```
3 条相关 lessons（如：Go nil pointer 相关）
    │
    ▼
LLM 分析模式
    │
    ├── 可否抽象为可复用的 Skill？
    │   ├── 是 → 生成 SKILL.md，写入 .monika/skills/<slug>/
    │   └── 否 → 仅保留为 topic
    │
    └── 通知用户："检测到重复模式，已生成 Skill: go-nil-safety"
```

---

## 八、工具设计

### 8.1 `memory_search` — 记忆检索

```
参数：
  query      string  必填 — 搜索关键词
  scope      string  可选 — "global" | "project" | "auto"（默认 auto）
  category   string  可选 — "lesson" | "topic" | "knowledge" | "raw" | "all"（默认 all）
  limit      int     可选 — 默认 5，最大 10

返回：
  results[]  {path, title, snippet, category, confidence, updated_at}
```

### 8.2 `memory_write` — 写入记忆

```
参数：
  title      string  必填 — 记忆标题
  content    string  必填 — markdown 内容
  category   string  必填 — "lesson" | "topic" | "knowledge_update"
  scope      string  可选 — "global" | "project"（默认 project）
  tags       string[] 可选
  confidence string  可选 — "high" | "medium" | "low"（默认 medium）

流程（写入审批门控 — 可选启用）：
  write_approval = true 时 → 前端弹出审批框 → 用户确认后生效
  write_approval = false 时 → 直接写入
```

### 8.3 `memory_index` — 查看索引

```
返回 index.md 内容，或按 category 筛选后的目录列表。
```

### 8.4 `memory_reindex` — 重建索引

```
重建 SQLite FTS5 索引（文件批量变更后手动触发）。
```

---

## 九、设置页 UI

在设置页面新增"知识库" Tab，功能矩阵：

### 9.1 功能列表

| 功能 | 说明 | 优先级 |
|------|------|--------|
| **作用域切换** | 全局 / 当前项目 tab 切换 | P4 |
| **目录树** | 树形展示 wiki 目录结构（index.md 驱动） | P4 |
| **文件预览** | 点击文件显示 markdown 渲染内容 | P4 |
| **全文搜索** | 搜索全局+项目 kb | P4 |
| **直接编辑** | 编辑 markdown 文件（覆盖 LLM 内容） | P4 |
| **启用/禁用** | 切换单条记忆是否注入 system prompt | P4 |
| **软删除** | 移到 `.trash/`，可恢复 | P4 |
| **统计面板** | 记忆总数、token 估算、最后更新时间 | P4 |
| **文档上传** | 拖拽上传文档到 `raw/docs/` | P4 |
| **仓库关联** | 输入 GitHub URL，触发 LLM 分析 | P4 |

### 9.2 状态栏集成

| 状态 | UI 表现 |
|------|---------|
| 归纳中 | 旋转图标 + "归纳记忆中..." |
| 完成 | 绿色勾 + "记忆已更新"（3s 后消失） |
| 失败 | 红色叉 + "归纳失败" + 点击重试 |
| 定期提醒 | 铃铛图标 + "要整理记忆吗？" + 点击触发后台审查 |

---

## 十、实施计划

### P1 — 存储基础设施 + 注入 + 检索（MVP）

**目标**：能存、能检索、能注入到 system prompt

**新增文件**：
- `internal/memory/types.go` — 路径常量、MemoryFile 结构体、KBScope 枚举
- `internal/memory/store.go` — 文件读写、FTS 索引管理、`search()` / `getIndex()`
- `internal/memory/inject.go` — `BuildMemoryBlock()` 组装注入文本
- `internal/tool/builtin/memory_search.go` — `memory_search` 工具
- `internal/tool/builtin/memory_write.go` — `memory_write` 工具
- `internal/tool/builtin/memory_index.go` — `memory_index` 工具

**修改文件**：
- `internal/agent/agent_loop.go` — `buildMessages()` 中注入 `<memory>` 块
- `internal/tool/builtin/register.go` — 注册 3 个新工具

**不包含**：
- 自动提取（P2）
- 合并/去重逻辑（P2）
- 后台审查（P3）

### P2 — 自动提取 + 合并 + knowledge.md 重建

**目标**：Session 结束时自动提取记忆，合并去重，维持 knowledge.md 在字符上限内

**新增文件**：
- `internal/memory/extract.go` — 从 CompactionSummary 提取记忆
- `internal/memory/consolidate.go` — 合并策略、相似度计算、去重
- `internal/memory/compact_knowledge.go` — knowledge.md 压缩
- `internal/memory/log.go` — log.md 写入

**修改文件**：
- `internal/api/session_manager.go` — 归档 hook 触发后台提取
- `internal/agent/compaction.go` — 可能需要调整 CompactionSummary 输出格式

**新增命令**：
- `/memory-summarize` — 手动触发归纳

### P3 — 后台审查 + 经验→技能沉淀

**目标**：定期后台审查，发现模式，生成 Skill

**新增文件**：
- `internal/memory/review.go` — 后台审查逻辑
- `internal/memory/skill_gen.go` — 经验→技能生成

**修改文件**：
- `main.go` 或 `internal/api/app.go` — 启动后台 goroutine
- `internal/engines/skill/skill.go` — 可能需适配自动生成的 skill 路径

### P4 — 前端 UI

**目标**：设置页 Knowledge Base Tab + 状态栏集成

**前端新增**：
- `frontend/src/pages/Settings/KnowledgeBaseTab.tsx` — 知识库管理面板
- `frontend/src/components/StatusBar.tsx` — 归纳状态提示

**后端新增**：
- `internal/api/kb_api.go` — 设置页的后端 API（CRUD + 搜索 + 统计）

---

## 十一、集成点与风险

### 11.1 与现有系统的关系

| 现有系统 | 与新记忆系统的关系 |
|----------|-------------------|
| **Compaction** | 记忆提取的输入源。Compaction 产出的结构化摘要 → LLM 二次蒸馏 → 写入 wiki |
| **Skills** | 程序记忆的载体。经验→技能沉淀复用现有 Skill 格式 |
| **Session Manager** | 归档 hook 触发记忆提取 |
| **Prompt System** | `buildMessages()` 新增 `<memory>` 注入块 |
| **Config System** | 复用全局/项目两级路径解析 |
| **Runner** | `memory_search` 工具可供子 Agent 使用 |

### 11.2 风险与缓解

| 风险 | 缓解 |
|------|------|
| LLM 写垃圾记忆 | 写入审批门控（write_approval）、置信度标记、软删除可恢复 |
| knowledge.md 膨胀 | 严格字符上限 + P2 压缩逻辑 |
| 文件系统性能 | FTS5 索引解耦检索，1 万文件以内无压力 |
| 记忆分裂（同一事多个文件） | P2 合并策略 + P3 后台审查去重 |
| 项目 kb 进入 git 的隐私风险 | `.monika/kb/` 默认加入 `.gitignore`（文档提示） |
| 前端开发量 | P4 放最后，P1-P3 后端完全独立可用 |

---

## 十二、依赖

```
modernc.org/sqlite  — 纯 Go SQLite（已依赖或计划依赖）
github.com/mattn/go-sqlite3  — 备选（需 CGO）
```

现有 `go.mod` 中如无 SQLite 依赖，P1 需添加 `modernc.org/sqlite`。

---

## 附录 A：与调研报告的对应关系

| 调研报告章节 | 设计文档对应 |
|-------------|------------|
| 记忆类型学 | §5 Wiki 文件类型映射（semantic/episodic/procedural） |
| Mem0 抽取+合并 | §7.2 记忆提取 + 合并策略 |
| MemGPT 分层自管理 | §7.4 LLM 自主维护 wiki |
| A-MEM Zettelkasten | §7.2 链接机制 + §5.6 index.md |
| ReasoningBank 闭环 | §7.4 后台审查 + §7.5 经验→技能 |
| Hermes 自进化 | §7.2 归档触发 + §7.4 后台审查 + 有界 knowledge.md |
| Loop Engine | 未来考虑：Guard 用于写入审批 |

## 附录 B：与 Karpathy LLM Wiki 的对应关系

| Karpathy 概念 | Monika 实现 |
|--------------|------------|
| raw/ 不可变源 | `raw/docs/` + `raw/code/` |
| wiki/ 编译产物 | `wiki/`（knowledge.md, lessons/, topics/, profile.md） |
| AGENTS.md 维护规则 | `SCHEMA.md`（全局 + 项目） |
| index.md 导航 | `wiki/index.md` |
| log.md 操作日志 | `wiki/log.md` |
| 编译器 | LLM（通过 memory 提取 + 合并 + 审查流程） |
| lint 维护 | P3 后台审查 |
