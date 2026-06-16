# AI Agent 记忆系统调研报告

> 调研日期：2026-06-16
> 范围：主流记忆框架、自进化/经验沉淀机制、Loop Engine、Hermes Agent、最新学术论文

---

## 一、背景：为什么"记忆"成为 Agent 的核心瓶颈

LLM 本身是无状态的，每次推理只依赖有限的上下文窗口。当 Agent 需要跨会话、跨任务积累经验时，记忆系统成为决定其能否"持续进化"的关键基础设施。

目前业界对 Agent 记忆的核心诉求集中在四点：
1. **长期持久化** — 跨会话保留关键信息
2. **高效检索** — 在海量记忆中精准召回相关内容（低延迟）
3. **记忆演化** — 新信息能更新、合并、淘汰旧信息，而非只增不减
4. **经验沉淀 / 自进化** — 从成功与失败中提炼可复用策略，让 Agent 越用越聪明

---

## 二、记忆类型学（Taxonomy）— 行业共识

综合 LangMem、Mem0 及最新综述论文（arXiv:2512.13564 *Memory in the Age of AI Agents*），业界对记忆类型的划分已趋于一致：

| 记忆类型 | 对应人类认知 | 存储内容 | 典型实现 |
|---------|------------|---------|---------|
| **语义记忆 Semantic** | 事实/知识 | "用户用 Go 1.22"、"项目用 PostgreSQL" | 结构化事实、向量库 |
| **情景记忆 Episodic** | 经历/事件 | "上次部署失败了，原因是端口冲突" | 过往交互轨迹、Few-shot 示例 |
| **程序记忆 Procedural** | 技能/规则 | "遇到 X 应该用 Y 流程" | System Prompt 规则、Skill 文件 |
| **工作记忆 Working (短期)** | 当前思维 | 当前会话上下文 | Context Window |

> **生产级系统的关键洞察**（Mem0 State of Agent Memory 2026）：大多数系统只做了语义 + 情景记忆，但生产 Agent 还需要**程序记忆**——把"怎么做"沉淀成可复用的技能/规则。

---

## 三、主流记忆框架对比（工程产品层）

### 3.1 Mem0 — 生产就绪的事实抽取型记忆

- **论文**：arXiv:2504.19413 *Mem0: Building Production-Ready AI Agents with Scalable Long-Term Memory*
- **核心架构**：抽取 → 合并 → 检索 三阶段
  - **Extract**：从对话中提取显著事实（salient facts），而非存储原始对话块
  - **Consolidate**：去重、更新、合并冲突事实
  - **Retrieve**：向量检索 + GraphRAG（知识图谱）
- **优势**：LoCoMo 基准测试得分最高（92.5），延迟/成本/准确率平衡最佳
- **定位**：最接近"开箱即用"的生产级记忆层
- **GitHub**：github.com/mem0ai/mem0

### 3.2 Letta（原 MemGPT）— OS 启发的分层记忆

- **论文**：arXiv:2310.08560 *MemGPT: Towards LLMs as Operating Systems*
- **核心架构**：借鉴操作系统的虚拟内存分层
  - **Main Context（主存）**：System Prompt + Working Context，始终在窗口内
  - **External Context（外存）**：Recall Storage（对话历史）+ Archival Storage（归档事实）
  - Agent 通过**函数调用自主管理**自己的内存（自我分页/换入换出）
- **优势**：擅长超长对话、单 Agent 深度场景
- **GitHub**：github.com/letta-ai/letta

### 3.3 Zep / Graphiti — 时序知识图谱记忆

- **论文**：arXiv:2501.13956 *Zep: A Temporal Knowledge Graph Architecture for Agent Memory*
- **核心架构**：Graphiti 引擎构建**实时、带时间维度的知识图谱**（基于 Neo4j）
  - 实体抽取 → 关系建模 → 时序版本管理
  - 能回答"三个月前用户的项目结构是什么"这类时序问题
- **优势**：企业级（SOC 2 合规），检索 < 200ms，在 Deep Memory Retrieval 基准上超越 MemGPT
- **GitHub**：github.com/getzep/graphiti

### 3.4 LangMem / LangGraph — 三类记忆一体化 SDK

- **核心**：将语义、情景、程序记忆统一集成到 LangGraph 状态管理中
  - **Semantic**：结构化事实存入 Store
  - **Episodic**：历史成功交互作为 Few-shot 示例
  - **Procedural**：动态更新的 System Prompt 规则（Agent 自己改写自己的"操作系统"）
- **优势**：与 LangChain 生态深度集成，概念清晰，适合快速原型
- **文档**：langchain-ai.github.io/langmem

### 横向对比

| 维度 | Mem0 | Letta/MemGPT | Zep/Graphiti | LangMem |
|------|------|-------------|-------------|---------|
| 核心机制 | 事实抽取+合并 | OS分层自管理 | 时序知识图谱 | 三类记忆统一 |
| 检索方式 | 向量+GraphRAG | 向量+函数调用 | 图查询+向量 | 向量Store |
| 时序感知 | 弱 | 弱 | **强（核心优势）** | 弱 |
| 生产成熟度 | **高** | 中 | 高 | 中 |
| 最佳场景 | 通用事实记忆 | 超长对话 | 企业级关系推理 | LangChain生态 |

---

## 四、自进化与经验沉淀（前沿学术层）

这是当前最活跃的研究方向，核心理念：**记忆不只是存储，更是从经验中提炼可复用策略的闭环系统**。

### 4.1 ReasoningBank — 自进化推理记忆闭环 ⭐（Google Research, ICLR 2026）

- **论文**：arXiv:2509.25140
- **核心机制**：记忆工作流形成**持续闭环**
  ```
  检索(Retrieve) → 行动(Act) → 提取(Extract) → 合并(Consolidate) → 检索...
  ```
  1. **检索**：面对新任务，先从 ReasoningBank 召回相关记忆指导行动
  2. **提取**：任务结束后，LLM 对轨迹进行结果判断（成功/失败）
  3. **合并**：蒸馏出**可泛化的推理策略**（而非原始日志），存入 Bank
- **创新点**：Memory-aware Test-Time Scaling (MaTTS) — 推理时利用记忆做扩展思考
- **关键差异**：不仅记住"做了什么"，更提炼"应该怎么推理"，从失败中也能学习
- **意义**：这是"经验沉淀"理念最清晰的学术实现

### 4.2 A-MEM — Zettelkasten 式自组织记忆 ⭐（NeurIPS 2025）

- **论文**：arXiv:2502.12110
- **核心机制**：借鉴卡片盒笔记法（Zettelkasten）
  1. **笔记构建**：新记忆生成结构化笔记（上下文描述 + 关键词 + 标签）
  2. **链接生成**：分析历史记忆，建立有意义的关联链接
  3. **记忆演化**：新记忆整合时，**反向更新历史记忆**的上下文表示
- **创新点**：记忆网络自组织、自演化，形成动态知识网络，而非静态存储
- **GitHub**：github.com/WujiangXu/A-mem-sys

### 4.3 MIRIX — 多智能体记忆系统（arXiv:2507.07957）

- **核心**：6 种记忆类型的多 Agent 架构
- **性能**：比 RAG 基线提升 35%，存储需求降低 **99.9%**
- **特色**：能处理高分辨率屏幕截图（现有系统做不到）
- **适用**：AI 助理、客服、代码 Agent 等

### 4.4 其他自进化机制

| 方案 | 核心思路 |
|------|---------|
| **MemGen** | 隐式记忆（Latent Memory）：通过"记忆触发器"+"记忆编织器"在推理时实时生成记忆，不改模型参数 |
| **MemRL** | 基于运行时强化学习的情景记忆，将记忆建模为状态空间做策略优化 |
| **RMM** (Reflective Memory Management, ACL 2025) | 前向+后向反思的记忆合并机制，用于长期个性化对话 |
| **Memory OS** (arXiv:2506.06326) | 三级存储：短期/中期/长期个人记忆 |

---

## 五、Loop Engine — 治理型决策循环运行时

> 注意：Loop Engine **本身不是记忆系统**，而是与 Agent 记忆紧密相关的**决策治理运行时**。理解它有助于设计完整的记忆+决策架构。

- **开发者**：Better Data（loopengine.io）
- **许可证**：Apache-2.0，TypeScript（npm `@loop-engine/sdk`）
- **核心定位**：**治理运行时（Governance Runtime）**，而非通用工作流编排器

### 架构模型

```
Providers (LLM/检索工具) ← 智能输入
        ↓
Decision Loops + Guards ← 治理：策略在提交前校验
        ↓
Channels (Slack/审批)   ← 人类参与
        ↓
Integrations (CRM/API)  ← 系统执行
        ↓
Evidence + Learning      ← 审计 + 运营改进
```

### 核心原语

| 原语 | 作用 |
|------|------|
| **Loop** | 受治理的决策循环：状态 + 转换 + 信号 + 终态 |
| **Guard** | 确定性策略守卫（hard 阻断 / soft 警告） |
| **Signal** | 状态间转移的意图 |
| **Actor** | 触发转移的角色（human / automation / ai-agent） |
| **Evidence** | 转移时附加的证据（输入、守卫结果、模型元数据） |
| **Learning Signals** | 循环完成后，比较预测 vs 实际结果，用于改进闭环 |

### 与记忆系统的关系

Loop Engine 的 **Learning Signals** 机制本质是一种经验沉淀闭环：决策循环完成后，通过对比"预测结果"与"实际结果"生成学习信号。这些信号可以回灌到记忆系统中，实现 **决策 → 经验 → 记忆 → 改进** 的完整闭环。AI 作为 Actor 在**确定性治理边界内**运行，而非直接控制系统。

---

## 六、Hermes Agent — 自进化 Agent 的完整工程实践 ⭐

- **开发者**：Nous Research（github.com/NousResearch/hermes-agent）
- **定位**："The agent that grows with you" — 会自我成长的开源 AI Agent

### 6.1 记忆架构（多层）

| 层级 | 机制 | 特点 |
|------|------|------|
| **MEMORY.md** | Agent 个人笔记（环境事实、约定、经验教训） | 2,200字符上限(~800 token)，强制压缩聚焦 |
| **USER.md** | 用户画像（偏好、沟通风格） | 1,375字符上限(~500 token) |
| **Session Search** | SQLite + FTS5 全文检索所有历史会话 | 无限容量，~20ms 查询，零 LLM 成本 |
| **外部记忆 Provider** | 8 个插件（Mem0/Hindsight/Honcho/Supermemory 等） | 知识图谱、语义搜索、跨会话建模 |

### 6.2 自进化闭环（核心亮点）

Hermes 实现了完整的**经验沉淀闭环**：

```
用户交互 → 任务执行 → 后台自改进审查(Background Review)
                              ↓
                    提炼经验 → 写入 MEMORY.md / 生成 SKILL.md
                              ↓
                    下次会话自动加载 → 更聪明的 Agent
```

**关键设计决策**：
1. **有界记忆（Bounded Memory）** — 严格字符上限，逼迫 Agent 做信息密度最高的压缩，而非无限堆积
2. **冻结快照注入** — 会话开始时一次性注入记忆到 System Prompt（保护 prefix cache），会话中变更实时落盘但下次才生效
3. **程序记忆 = Skills** — Agent 自动创建 `SKILL.md` 文件，把"怎么做"沉淀为可复用技能
4. **后台自改进审查** — 每轮交互后自动运行，提取值得记住的经验
5. **Periodic Nudges** — 定期主动提醒，触发记忆整理
6. **写入审批门控**（`write_approval`）— 防止 Agent 写入错误假设，用户可逐条审批
7. **安全扫描** — 记忆写入前扫描注入/泄露模式，拦截恶意内容

### 6.3 与其他系统的关系

Hermes 证明了"自进化"不需要复杂的底层架构——通过**有界文件记忆 + 后台审查循环 + 技能沉淀**，在 Agent 应用层即可实现持续进化。它的外部 Provider 插件机制也表明：内置轻量记忆 + 外接专业记忆服务是务实的产品组合。

---

## 七、最新综述与研究趋势（2025-2026）

### 综述论文

**Memory in the Age of AI Agents**（arXiv:2512.13564，2025年12月）
- 最新的 Agent 记忆全景综述
- 配套论文列表：github.com/Shichun-Liu/Agent-Memory-Paper-List
- **核心发现**：记忆合并/固化（Consolidation）是最活跃的开放研究方向

**ICLR 2026 Workshop: Memory for LLM-Based Agentic Systems**
- 专门针对 Agent 记忆的学术研讨会，标志着该方向已成为独立研究热点

### 当前研究趋势总结

| 趋势 | 代表工作 |
|------|---------|
| 记忆自组织与演化 | A-MEM（Zettelkasten）、ReasoningBank |
| 时序感知记忆 | Zep/Graphiti |
| 经验→策略蒸馏 | ReasoningBank（从成功/失败中提炼推理策略） |
| 多 Agent 共享记忆 | MIRIX |
| 隐式/参数化记忆 | MemGen（不改参数的隐式记忆） |
| 记忆+强化学习 | MemRL（记忆作为状态空间） |

---

## 八、关键结论与选型建议

### 8.1 记忆系统的三个设计层次

```
┌─────────────────────────────────────────────┐
│  Layer 3: 经验沉淀 / 自进化闭环              │  ← ReasoningBank, Hermes
│  （从经验中提炼策略，越用越聪明）             │
├─────────────────────────────────────────────┤
│  Layer 2: 记忆管理服务                       │  ← Mem0, Zep, Letta, LangMem
│  （抽取/合并/检索/演化）                      │
├─────────────────────────────────────────────┤
│  Layer 1: 存储与检索基础设施                  │  ← 向量库, 知识图谱, SQLite
│  （向量/图/全文检索）                         │
└─────────────────────────────────────────────┘
```

### 8.2 选型决策树

| 需求场景 | 推荐方案 |
|---------|---------|
| 快速上线通用记忆 | **Mem0**（生产成熟度最高） |
| 超长对话/单 Agent 深度交互 | **Letta/MemGPT**（分层自管理） |
| 企业级关系推理/时序查询 | **Zep/Graphiti**（时序知识图谱） |
| LangChain 生态集成 | **LangMem**（三类记忆统一） |
| 需要"越用越聪明"的自进化 | **ReasoningBank 思路** + Hermes 模式 |
| 记忆自组织/动态演化 | **A-MEM**（Zettelkasten 自组织网络） |
| 需要 AI 决策治理 | **Loop Engine**（治理边界 + 学习信号） |

### 8.3 研发建议

如果要自研记忆系统，建议分层采用混合策略：

1. **存储层**：向量库（语义检索）+ 知识图谱（关系/时序）+ 全文索引（精确召回）
2. **管理层**：借鉴 Mem0 的事实抽取+合并机制，避免存储原始日志
3. **进化层**：借鉴 ReasoningBank 的闭环（检索→行动→提取→合并），从轨迹中蒸馏可复用策略
4. **程序记忆**：借鉴 Hermes 的 Skills 模式，把"怎么做"沉淀为可复用技能文件
5. **治理**：借鉴 Loop Engine 的 Guard + Evidence 模式，给 AI 的记忆写入加确定性边界

**最核心的设计原则**：记忆系统不能是"只增不减的便利贴狂魔"（Append-only Log）。必须有**合并、淘汰、演化**机制，否则会导致知识碎片化、冲突盲区和 Context 无限膨胀。

---

## 附录：核心资料索引

### 论文
| 论文 | arXiv | 会议 |
|------|-------|------|
| Mem0 | 2504.19413 | — |
| MemGPT | 2310.08560 | — |
| Zep | 2501.13956 | — |
| A-MEM | 2502.12110 | NeurIPS 2025 |
| ReasoningBank | 2509.25140 | ICLR 2026 |
| MIRIX | 2507.07957 | — |
| Memory OS | 2506.06326 | — |
| Memory in the Age of AI Agents (综述) | 2512.13564 | — |

### 开源项目
| 项目 | 地址 |
|------|------|
| Mem0 | github.com/mem0ai/mem0 |
| Letta/MemGPT | github.com/letta-ai/letta |
| Graphiti/Zep | github.com/getzep/graphiti |
| A-MEM | github.com/WujiangXu/A-mem-sys |
| Hermes Agent | github.com/NousResearch/hermes-agent |
| Loop Engine | loopengine.io |
| LangMem | langchain-ai.github.io/langmem |
| Agent Memory Paper List | github.com/Shichun-Liu/Agent-Memory-Paper-List |
