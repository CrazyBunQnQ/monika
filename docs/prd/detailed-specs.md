# CoC 跑团平台 - 详细规格文档

**文档版本**: v1.0
**创建日期**: 2026-02-05
**最后更新**: 2026-02-05

> 本文档包含所有里程碑的详细规格说明，是开发实施的技术依据。

---

## 目录

- [M0：冻结规范](#m0冻结规范详细)
- [M1：单人单场景可玩循环](#m1单人单场景可玩循环详细)
- [M2：多人桌面流程](#m2多人桌面流程并发与可见性详细)
- [M3：长团记忆](#m3长团记忆可检索-可复盘-不泄露详细)
- [M4：模组/规则书知识库](#m4模组规则书知识库可读模组-可查规则详细)
- [M5：CoC 全功能补齐](#m5coc全功能补齐v1越多越好落地详细)
- [M6：体验打磨](#m6体验打磨防卡死与节奏控制详细)

---

## 重要说明

除 M0 中冻结的"最小用户命令清单"外，本章后续出现的 `/xxx` 多数为**内部动作/API/事件类型的占位符**，用于实现与审计；**不强制要求玩家输入**这些细粒度命令，玩家默认以自然语言推进。

---

## M0：冻结规范（详细）

### M0-A：最小用户命令 + 语义交互（TRPG-only 门禁）

#### 核心原则

- **A1 输入允许自然语言**：系统默认将其解释为"跑团内发言/行动/提问"，不强制命令前缀
- **A2 支持玩家身份与扮演对象切换**：同一 session 可区分"玩家名"和"角色名"
- **A3 区分 IC 与 OOC**：允许自然语言标记（例如 `OOC:`）或命令快捷键；避免把桌外沟通误判为角色行动
- **A4 任何机制变更必须"显式可见 + 可追溯"**：掷骰、伤害、SAN/Luck 变动、物品增减、角色卡字段修改都必须落为机制事件并在输出中汇总
- **A5 KP-only 指令与玩家指令分离**：玩家不能直接调用 KP-only（例如强制改场景、读取秘密线索）
- **A6 越界拒绝统一**：非跑团诉求统一拒绝模板 + 引导到可用跑团命令

#### 推荐的最小用户命令清单（M0 必须冻结）

| 类别 | 命令 | 说明 |
|------|------|------|
| **会话** | `/create_session` | 创建新会话 |
| | `/resume_session <id>` | 恢复已有会话 |
| | `/end_session` | 结束会话 |
| **入团** | `/join <player>` | 加入会话 |
| | `/as <player>` | 声明当前说话人/操控者 |
| **行动** | `/act "<行动/意图>"` | 可选；自然语言默认等价 |
| **规则** | `/rule "<query>"` | 可选；自然语言提问也可触发检索 |
| **沟通** | `/say "<text>"` | 可选公开发言 |
| | `/whisper <player> "<text>"` | 可选私聊 |
| **状态查询** | `/status` | HP/SAN/Luck/异常/当前位置/当前 leads |
| **复盘** | `/recap` | 最近摘要/当前场景摘要/当前线索账本 |
| **场景** | `/scene next` | 进入下一场景 |
| | `/scene set <scene_id>` | 设置当前场景（KP-only） |
| **安全** | `/safety` | 安全工具 |
| | `/pause` | 暂停 |
| | `/resume` | 恢复 |
| | `/retcon "<修正内容>"` | 回滚修正（KP确认后生效） |
| **求助/退出** | `/help` | 帮助 |
| | `/quit` | 退出 |

### M0-B：模组"场景包"格式（上传脚本的冻结字段）

#### 模组元信息（必填）

- **标题**：模组名称
- **年代/地点**：故事发生的年代和地点
- **推荐玩家数**：建议玩家数量（如 2-4 人）
- **风格**：调查/战斗/生存（可多选）
- **KP-only 标签**：是否包含 KP-only 秘密

#### 场景（一级单元）

每个场景必须包含：

| 字段 | 说明 | 必填 |
|------|------|------|
| `scene_id` | 场景唯一标识 | ✅ |
| 入口触发 | 进入场景的条件或事件 | ✅ |
| 退出条件 | 完成场景的条件 | ✅ |
| 可移动线索 | 2–4 条（玩家走偏也能拿到） | ✅ |
| NPC 列表 | 结构化 NPC 数据 | ✅ |
| 地点列表 | 结构化地点数据 | ✅ |
| 失败前进 | 失败时的代价与推进 | ✅ |
| 计时器/压力源 | 随时间推进的压力 | ✅ |

#### NPC 结构化字段

```typescript
interface NPC {
  npc_id: string
  name: string
  appearance: string        // 外貌描述
  motivation: string        // 动机
  secret: string            // 秘密（KP-only）
  clue_associations: string[] // 关联线索
  attitude_toward_pc: string // 对 PC 的态度
}
```

#### 地点结构化字段

```typescript
interface Location {
  location_id: string
  name: string
  description: string
  interaction_points: string[]  // 可互动点
  discoverable_clues: string[]  // 可发现的线索
  dangers: string[]             // 危险/代价
}
```

#### 线索结构化字段

```typescript
interface Clue {
  clue_id: string
  text: string
  source: string              // 来源（场景/NPC/手递物）
  visibility: 'public' | 'kp' | 'player:<id>'
  portable: boolean           // 是否可移动
  handout_ref: string | null  // 手递物引用
}
```

#### 失败前进机制

- 关键行动失败时也要能推进，只是代价不同
- 代价类型：
  - 时间损失
  - SAN 扣除
  - 资源消耗
  - 敌对度提升
  - 引入新威胁

#### 计时器/压力源

- 至少一个"压力源"随时间推进
  - 怪异加深
  - 警方介入
  - 证据消失
  - NPC 被杀害

### M0-C：状态字段清单（角色卡 + 世界状态 + 线索账本）

#### C1 角色卡（PC）最小字段

```typescript
interface Character {
  // 基础信息
  name: string
  occupation: string
  age: number
  residence: string
  background: string

  // 属性
  attributes: {
    STR: number  // 力量
    CON: number  // 体质
    DEX: number  // 敏捷
    APP: number  // 外貌
    POW: number  // 意志
    INT: number  // 智力
    SIZ: number  // 体型
    EDU: number  // 教育
  }

  // 派生
  derived: {
    HP: number
    HP_max: number
    MP: number
    MP_max: number
    SAN: number
    SAN_max: number
    Luck: number
    Luck_max: number
    Move: number
  }

  // 技能（支持自定义）
  skills: Record<string, number>

  // 状态
  status: {
    wounds: string[]              // 伤口描述
    major_wound: boolean          // 重大伤标记
    dying: boolean                // 濒死标记
    insanity: InsanityState[]     // 疯狂状态
    long_term_trauma: string[]    // 长期创伤
    temporary_modifiers: Modifier[] // 临时加减值
  }

  // 装备与现金
  inventory: string[]
  cash: number
  spending_level: number
}
```

#### C2 世界/场景状态最小字段

```typescript
interface WorldState {
  current_scene: string | null
  current_location: string
  time_period: string  // 粗粒度时间

  important_npcs: Array<{
    npc_id: string
    attitude: string  // 对 PC 的态度
  }>

  current_threats: Threat[]
  timer: Timer | null

  current_leads: Lead[]  // 2–6 条，必须对玩家可见
}
```

#### C3 线索账本最小字段

```typescript
interface ClueLedger {
  clue_id: string
  text: string
  source: string              // 场景/NPC/手递物
  credibility: number         // 可信度
  ownership: string           // 归属（谁知道）
  points_to: string[]         // 指向 leads
  verified: boolean           // 是否已验证
  visibility: 'public' | 'kp' | 'player:<id>'
}
```

#### C4 可见性规则冻结

三档可见性：
- `public`：所有人可见
- `kp`：仅 KP 可见
- `player:<id>`：仅指定玩家可见

默认规则：
- 线索默认公开
- 秘密默认 KP-only

### M0 验收标准

- ✅ 命令集与越界拒绝规则确定（玩家知道怎么说话、系统知道怎么拒绝）
- ✅ 场景包字段固定（可以开始写/转换模组）
- ✅ 状态字段固定（可以开始建卡、记录线索、做摘要）

---

## M1：单人单场景可玩循环（详细）

> 目标：让 1 名玩家在 1 个场景内"能玩、能结局、可审计"。交互默认自然语言；命令只是可选快捷键。

### M1-A：单人场景循环（核心叙事与门禁）

#### 用户命令（可选快捷键，越少越好）

| 命令 | 说明 |
|------|------|
| `/act "..."` | 可选；自然语言默认等价 |
| `/help` | 列出命令与示例 |
| `/status` | 输出当前场景摘要 |
| `/history [n]` | 输出最近 n 条可见对话 + 机制事件摘要 |
| `/recap` | 输出更长的阶段性摘要 |
| `/quit` | 结束会话并输出会话总结 |

#### 输出结构稳定要求

每轮至少包含三个固定块：

1. **[KP]**：叙事与裁定（必须）
2. **[State]**：本轮状态变化（若无变化也要显式说明 "no change"）
3. **[Next]**：下一步引导（必须给问题或 2–4 个可选行动建议）

#### 门禁与一致性要求

- 所有"会改变数值/条目/战斗态/追逐态"的变更不得静默发生
- 必须在输出的机制/状态变化块中显式呈现并写入事件日志
- 必要时要求自然语言确认

#### 场景内实体一致性

- NPC/道具/线索不能凭空出现
- 若需要引入，必须先在 `[State]` 中以明确新增条目落地

#### 结束条件（单场景内必须可达）

- 玩家主动 `/quit`
- 成功结局：至少一种输入序列能达成 goals 或 `scene_progress` 阈值
- 失败结局：至少一种输入序列能触发不可逆后果并结束

#### 验收标准

- 用仅 1 个场景的模组启动：能完成 ≥5 轮自然语言行动循环
- `/help`、`/status`、`/history 3`、`/quit` 均可用且不破坏状态
- 给出一条明确越界输入（跑团外请求）：系统必须拒绝，输出包含 `[Refusal]`
- `/quit` 输出会话总结至少包含：场景名、主要事件 3 条、最终 goals/flags、关键数值

### M1-B：检定闭环（语义触发，事件可审计）

#### 目标

在单场景内跑通：宣言 → 要求检定 → 掷骰 → 结果 →（可选）推骰/花幸运 → 后果 → 状态落地

#### 用户侧交互（不要求命令）

- 玩家用自然语言描述"做什么/用什么技能或属性/是否愿意冒险或推骰/是否花幸运"
- KP 在输出中展示检定与结算，并将结果写入机制事件日志与 `[State]`

#### 内部机制事件（用于审计）

| 事件 | 字段 | 说明 |
|------|------|------|
| `roll` | skill/stat, 难度, 奖惩骰, 目标值, 骰面/展开, 成功等级, 大成功/大失败 | 检定事件 |
| `push_roll` | 关联 roll, 风险承诺文本, 推骰结果, "更坏后果" | 推骰事件 |
| `luck_spend` | 花费点数, 用途, 前后剩余, 对结果的影响 | 花幸运事件 |
| `san_check` | reason, loss 表达式, 最终扣减 | SAN 检定 |
| `san_loss` | reason, loss 表达式, 最终扣减 | SAN 损失 |
| `damage` | 来源, 表达式, 最终数值, 目标 | 伤害事件 |
| `hp_change` | 来源, 表达式, 最终数值, 目标 | HP 变化 |

#### 需求细则

**检定请求必须"可选择"**：
- KP 要求检定时必须说明：检什么、成功会怎样、失败会怎样
- 若玩家未明确技能/属性，KP 必须给出 1–3 个可选项并解释差异

**`/roll` 输出必须包含稳定字段**：
- 目标值
- 掷骰明细（含奖/惩骰展开或简述）
- 成功等级
- 是否大成功/大失败（如适用）

**奖励/惩罚骰互斥**：
- 同一条 `/roll` 不允许同时 `bonus` 与 `penalty`
- 出现时拒绝并提示二选一

**推骰闭环**：
- 只有被标记为"可推"的检定允许 `/push`
- 不可推项可先以黑名单实现
- 推骰失败必须触发"更坏后果"，且写入 flags/leads/伤害/SAN 中至少一项

**幸运闭环**：
- 花幸运必须引用"最近一次检定或伤害事件"
- 记录：花了多少、剩余多少、用途
- 花幸运后必须在 `[State]` 中体现并解释"结果被改成什么"

**SAN 与伤害**：
- `/san`、`/damage` 必须携带可追溯 reason/source
- 落地到可审计的历史事件

#### 验收标准

- **成功路径**：至少覆盖 1 次奖励骰与 1 次花幸运，并产生 1 条新线索
- **失败变坏路径**：至少覆盖 1 次推骰失败，并触发 `/damage` 或 `/san`
- **门禁**：自然语言尝试"直接改幸运/直接扣 SAN/直接扣 HP"不得直接落地

### M1-C：战斗闭环（进入战斗 → 回合 → 对抗 → 结算 → 结束）

#### 目标

单场景内跑通战斗的最小可玩回合制流程，并保证每一步可审计。

#### 命令

无强制用户命令。玩家用自然语言声明行动；KP 自动管理战斗态/回合/对抗与结算。

#### 需求细则

**战斗态是显式状态**：
- KP 必须在输出中明确"已进入战斗/已结束战斗"
- 在状态中维护战斗态标记

**回合与当前行动者清晰可见**：
- 每轮输出必须标出"当前行动者/下一位"

**近战对抗最小规则**：
- 攻击与防御双方都需检定
- 按成功等级对比裁定
- 输出必须解释对抗结论

**伤害落地必须可审计**：
- 命中后必须显式给出伤害结算过程
- 在 `[State]` 中落地 HP 变化
- 写入机制事件日志

**重伤/濒死（可简化但要有状态位）**：
- 触发时必须在 `[State]` 写入标记
- `major_wound=true`、`dying=true`
- 在叙事中体现行动限制

**治疗同样可审计**：
- 必须引用受伤事件
- 完成一次治疗结算（含检定与后果）
- 成功/失败都要落地后果并写入机制事件

#### 验收标准

- **获胜路径**：至少 1 次对抗结算 + 1 次伤害落地 + 1 次战斗结束小结
- **濒死路径**：至少出现 1 名 PC 进入濒死/重伤阈值，并通过急救/医疗形成"稳定或失败后果"
- **门禁**：战斗相关 HP/伤害/治疗不得静默改

### M1-D：追逐闭环（追逐态 + 距离/压力 + 失败前进）

#### 目标

单场景内跑通追逐：初始化 → 回合推进 → 障碍/代价 → 结束（逃脱/抓获/失败前进进入新局面）

#### 命令

无强制用户命令。玩家用自然语言描述"追/逃/抄近路/制造障碍"等；KP 自动维护追逐态与距离/压力变量。

#### 需求细则

**追逐态显式**：
- KP 必须在输出中明确"已进入追逐/已结束追逐"
- 在状态中维护追逐态标记

**最小状态变量**：
- `round`：当前回合
- `distance`：相对距离（相对刻度即可）
- `pressure` 或 `timer`：压力/计时器
- `env_tags`：环境标签

**每回合必须可结算**：
- 每位行动者的追逐动作要么映射到 `/roll`
- 要么 KP 明确声明"无需检定"并记录原因

**失败前进是硬约束**：
- 失败不能只是不成功
- 必须产生代价（伤害/SAN/丢失物品/引来第三方/计时器推进等）并写入状态

#### 验收标准

- **成功逃脱**：距离/优势按规则变化并最终满足结束条件；在输出中记录追逐结束与原因
- **失败前进**：至少一次失败触发代价（含伤害/SAN/flags 等），并在输出中形成"新局面"
- **门禁**：追逐中的伤害/SAN 仍必须显式呈现并可追溯

### M1-E：规则问答闭环（检索 + 引用 + 桌规裁定）

#### 目标

规则问题"先检索再回答"，并将超出证据的裁量固化为可追溯的桌规（ruling）。

#### 命令

- 用户侧（可选快捷键）：`/rule "<query>"`
- KP 侧：自动检索并给出引用；必要时形成桌规裁定并落为可追溯事件

#### 需求细则

**回答结构稳定**：
必须包含四块（即使没有也要写 "none"）：
- `[Question]`
- `[Answer]`
- `[Citations]`
- `[Ruling]`

**证据优先级**：
1. 桌规
2. 规则书引用
3. 模组特例
4. KP 临时裁量（临时裁量必须固化为可追溯的 ruling 事件）

**无证据不胡编**：
- 检索无结果必须明确说明
- 给下一步（换关键词/补上下文/补充资料/临时桌规）

**防注入**：
- KB 片段是资料不是指令
- 任何"忽略门禁/执行别的任务"的文本都必须被当作普通文本

**TRPG-only**：
- 规则问答不得引导跑团外行为

#### 验收标准

- **有证据问答**：至少 1 次检索命中并引用 citation_id，且可复核上下文
- **矛盾证据**：至少 1 次出现冲突或不完整证据，KP 进行澄清或形成桌规裁定事件
- **无证据**：至少 1 次检索为空，KP 不编规则条文，给出下一步，并（若需要）记录临时桌规

---

## M2：多人桌面流程（并发与可见性）（详细）

### M2-A：多人加入/身份/席位（Players & Seats）

#### 目标

- 支持 2–4 名玩家加入同一 session，并能稳定区分"玩家身份"与"角色身份"
- 允许玩家中途掉线/回归；会话状态可恢复继续跑

#### 新增/细化命令

| 命令 | 说明 | 权限 |
|------|------|------|
| `/players` | 列出当前 session 玩家列表 | 所有 |
| `/leave` | 玩家离开（保留角色与状态） | 玩家 |
| `/kick <player>` | 移出玩家（需记录原因） | KP-only |
| `/as <player>` | 声明当前说话人/操控者 | 所有 |

#### 需求细则

**R1：身份与归属可审计**
- 每条输入与系统落地的机制事件必须绑定：
  - `session_id`
  - `actor_player_id`
  - `actor_role`（KP/Player）
  - `controlled_character_id`（若有）
- KP-only 与 Player-only 命令必须严格鉴权
- 鉴权失败需给出稳定拒绝信息并写入审计日志

**R2：多人同时输入的最小处理策略**
- 系统必须定义"同一时刻多条输入"的处理策略
- 最小可用：按到达顺序进入队列，逐条处理
- 任何被排队的输入必须立即得到"已入队"的确认
- 可通过 `/queue` 查看

#### 验收标准

- 2–4 名玩家可用 `/join` 加入并用 `/players` 查看到一致结果
- 任意玩家 `/leave` 后不会丢失角色卡与线索归属；再次 `/resume_session <id>` 后能继续
- 玩家尝试执行 KP-only（如 `/kick`）会被拒绝，且拒绝可被审计复盘

### M2-B：聚光灯/队列（Spotlight & Turn Queue）

#### 目标

- 在多人桌面中给出明确的"谁在行动"的规则，减少插话与并发造成的状态冲突
- 让 KP 能控制节奏：推进当前行动者、允许插话、或暂缓某人行动

#### 新增/细化命令

| 命令 | 说明 | 权限 |
|------|------|------|
| `/spotlight show` | 显示当前聚光灯 | 所有 |
| `/spotlight next` | 推进到下一行动者 | KP-only |
| `/queue` | 查看输入/行动队列 | 所有 |
| `/queue drop <id>` | 丢弃某条排队输入（需记录原因） | KP-only |
| `/interrupt request` | 申请插话（仅登记） | 玩家 |
| `/interrupt allow <player>` | 允许某玩家插话一次 | KP-only |

#### 需求细则

**R1：聚光灯是"状态变量"**
- 状态中至少包含：
  - `spotlight.current_player`
  - `spotlight.queue`（按行动者顺序，而不是按消息顺序）
  - `spotlight.interrupt_requests`（待处理插话请求）
- KP 每轮输出应包含一个稳定块（例如 `[Spotlight]`）提示当前行动者与下一步

**R2：并发写冲突的最小避免**
- 当非当前聚光灯玩家直接提交会改变状态的命令时：
  - 系统必须拒绝或入队（两者二选一，但行为需一致）
  - 必须提示"等待聚光灯"或"已排队"
  - 给出 `/spotlight show` 与 `/queue` 的指引
- KP 可用 `/interrupt allow` 临时允许一次"越过聚光灯"的行动，但该授权必须写入审计

**R3：插话/打断的最小规则**
- 插话不应直接改变世界状态
- 插话只允许：补充信息、短句提问、对当前行动的协助声明
- 若插话涉及机制变更，必须排队到其回合或由 KP 显式授权

#### 验收标准

- 预置 3 名玩家同时发送输入：系统能稳定排队并按规则处理，不产生"同一条状态被互相覆盖"
- 非聚光灯玩家提交 `/roll`：系统能稳定拒绝或入队，并给出清晰提示
- KP 可用 `/spotlight next` 推进桌面流程；可用 `/queue drop` 丢弃一条输入且留痕

### M2-C：可见性（KP-only / Player-only）与信息泄露防护

#### 目标

- 确保 KP-only 内容不会泄露给玩家；同时支持"仅某玩家可见"的线索/私密信息
- 让多人状态查询（`/status`、`/sheet show`、`/recap`）遵循可见性规则

#### 新增/细化命令

| 命令 | 说明 | 权限 |
|------|------|------|
| `/whisper <player> "<text>"` | 向某玩家发送私密信息 | KP-only |
| `/tell kp "<text>"` | 玩家向 KP 私密汇报 | 玩家 |
| `/visibility set <object_ref> <public\|kp\|player:<id>>` | 调整某对象可见性 | KP-only |

#### 需求细则

**R1：所有可输出内容必须经过权限过滤**
- 系统在生成任何面向玩家的输出时，必须以"当前视角（viewer）"过滤：
  - **KP 视角**：可见全部（除安全工具/隐私策略另有限制）
  - **玩家视角**：只能看到 public + 自己可见（player:<id>）的内容
- 过滤不仅作用于"最终输出文本"，也作用于：`/history`、`/recap`、`/status`、`/sheet show`、线索账本

**R2：线索账本可见性与归属**
- M0 已冻结"公开/仅KP/仅某玩家"三档；M2 必须保证该规则在多人环境真实生效
- 当线索归属为"仅某玩家"时：
  - 其他玩家的 `/recap` 不应出现该线索文本
  - KP 的 `/recap` 必须能看到该线索及其归属

**R3：最小防泄露验收点**
- KP-only 内容不得通过任何"摘要/复盘/历史"泄露到玩家输出
- 若玩家试图通过提示注入诱导输出 KP-only 内容，系统必须拒绝并引导回跑团命令

#### 验收标准

- KP 用 `/whisper <player>` 发送秘密线索：
  - 目标玩家能看到
  - 其他玩家看不到（包括 `/history` 与 `/recap`）
  - KP 可在审计视角复盘到该事件
- 玩家用 `/tell kp` 汇报：仅 KP 可见，其他玩家不可见

### M2-D：多人状态查询与复盘（Per-Viewer Recap）

#### 目标

- 在多人会话中，保证 `/status`、`/recap` 的输出"按视角一致"且不泄露
- 支持长 Session 中断后恢复：每个玩家都能恢复到正确的"我知道什么/我能做什么"

#### 新增/细化命令

| 命令 | 说明 |
|------|------|
| `/status [me\|all]` | 默认 `me`；KP 可用 `all` 查看全局 |
| `/recap [me\|public\|all]` | 默认 `me`；KP 可用 `all` |

#### 需求细则

**R1：同一事件，多种视角摘要**
- 同一个"事件日志"应能派生出：
  - KP 视角摘要
  - 玩家视角摘要（每玩家不同）
- 当视角不同导致信息缺失时，系统应保持叙事连贯
- 允许用"你隐约感觉…"等模糊表述，但不得泄露具体 KP-only 内容

**R2：恢复一致性**
- `/resume_session <id>` 后：
  - KP 能恢复全局状态（场景、计时器、全线索账本、所有 flags）
  - 玩家只能恢复到其可见子集（含自己私密线索）

#### 验收标准

- 预置：同一 session 中同时存在 public clue 与 player-only clue：
  - 不同玩家执行 `/recap` 得到不同内容（符合可见性）
  - KP 执行 `/recap all` 能看到完整内容
- 中断后恢复：任意玩家恢复后仍能看到"自己私密线索 + public 线索"，且不会突然看到别人的私密线索

### M2-E：共享状态写入与所有权（Sheets / Inventory / Clues / Flags）

#### 目标

- 定义多人环境下"谁能改什么"的最小规则，避免角色卡/物品/线索被他人误改
- 对可能冲突的写入（几乎同时修改同一字段）提供可解释的处理方式

#### 新增/细化命令

| 命令 | 说明 | 权限 |
|------|------|------|
| `/sheet show [char]` | 默认查看当前操控角色；KP 可指定 `char` | - |
| `/sheet set [char] <field>=<value>` | 默认修改当前操控角色；指定 `char` 仅允许 KP | - |
| `/inv add [char] "<item>"` | 同上 | - |
| `/inv remove [char] "<item>"` | 同上 | - |

#### 需求细则

**R1：写入权限（Owner / KP）**
- 角色卡与物品的"直接写入"遵循最小权限：
  - **玩家**：只能修改自己当前操控角色的角色卡/物品（且仍需显式可审计的变更记录；必要时要求确认）
  - **KP**：可以修改任意角色（需可审计：修改了谁、改了什么、为什么）
- 线索账本：
  - 玩家只能创建/补充"自己视角的记事"（如果允许），但不能把 KP-only 线索改成 public
  - 可见性与归属的修改（`/visibility set`）必须是 KP-only

**R2：冲突写入的最小处理**
- 对同一对象/字段的写入必须可检测冲突（最小方案二选一即可）：
  - **严格串行**：所有写入进入同一队列，按顺序落地（推荐 v1）
  - **乐观并发**：基于 `revision` 或时间戳检测冲突，冲突时拒绝并提示重试
- 冲突发生时必须提示：冲突对象、冲突字段、当前值、建议动作

**R3：机制事件落地必须绑定目标**
- `/roll`、`/damage`、`/san`、`/luck` 等机制事件在多人环境下必须能明确绑定到"哪个角色"
- 默认绑定到当前操控角色；若 KP 需要对 NPC/其他角色落地，应通过 KP-only 方式显式指定

#### 验收标准

- 玩家 A 尝试 `/sheet set` 修改玩家 B 的角色：被拒绝并给出原因
- KP 可指定 `char` 修改任意角色：修改被记录并能在复盘中追溯
- 两名玩家几乎同时修改同一字段：系统不会静默覆盖；要么串行落地可解释，要么检测冲突并拒绝其一

### M2-F：多人战斗/追逐的回合与桌面流程联动

#### 目标

- 让多人战斗/追逐时的"回合顺序"和"聚光灯"一致，避免同时多个人宣告攻击导致混乱
- 确保战斗/追逐中的机制变更（攻击、伤害、治疗、追逐行动）仍遵守门禁与可审计

#### 新增/细化命令

| 命令 | 说明 | 权限 |
|------|------|------|
| `/turn show` | 显示当前回合结构 | 所有 |
| `/turn next` | 在战斗/追逐态下推进行动者 | KP-only |

#### 需求细则

**R1：进入战斗/追逐时切换"回合驱动"**
- 当进入战斗/追逐后：
  - 聚光灯应切换为"按回合行动者"为准（可以是玩家，也可以是 NPC）
  - 非当前行动者提交会改变状态的命令必须拒绝或入队

**R2：宣告与结算的最小一致性**
- 当前行动者可以用 `/act` 宣告行动意图
- 实际机制落地仍通过对应命令链
- 若多个玩家同时宣告：
  - 只有当前行动者的"可改变状态"的命令会被立即执行
  - 其他人的命令必须排队或拒绝，并提示等待 `/turn next`

#### 验收标准

- 预置 2 名 PC + 1 名 NPC 的战斗：
  - 系统能稳定约束"只有当前行动者能攻击/治疗"
  - `/turn show` 与 `[Spotlight]`/回合提示一致
- 预置追逐中 2 名玩家同时提交追逐行动：系统不会并发写坏 distance/pressure 等状态变量

### M2-G：暂停/安全工具在多人桌面的行为

#### 目标

- 明确 `/pause`、`/resume`、`/retcon` 在多人环境的语义
- 避免有人暂停后他人仍能推进机制
- 让"安全中断"与"回滚修正"可审计且不造成权限泄露

#### 新增/细化命令

| 命令 | 说明 | 权限 |
|------|------|------|
| `/pause` | 任何玩家可发起全桌暂停 | 所有 |
| `/resume` | 仅 KP 可恢复 | KP-only |
| `/retcon "<text>"` | 任何人可提出，但必须 KP 确认后生效 | 所有 |

#### 需求细则

**R1：暂停态的输入处理**
- 当 `paused=true` 时：
  - 禁止任何会改变规则状态的命令落地
  - 允许最小集合：查看状态（`/status`、`/recap`）、求助（`/help`）、以及非机制的沟通

**R2：回滚修正（Retcon）可追溯**
- `/retcon` 的生效必须产生一条"修正事件"：
  - 提案人、提案内容、KP 采纳内容、影响范围
  - 若会影响既有机制结果，必须要求通过命令重新落地或明确标注"人工修正"

#### 验收标准

- 任意玩家 `/pause` 后，其他玩家尝试 `/roll`：会被拒绝并提示当前为暂停态
- KP `/resume` 后桌面可继续推进；暂停期间的队列输入不会被错误执行
- 发起一次 `/retcon` 并被 KP 采纳：复盘中能看到提案与最终修正内容

### M2-H：多人鲁棒性（掉线、重复输入、幂等性）

#### 目标

- 让多人长 Session 在掉线/重复发送/重连的情况下仍能保持状态一致
- 避免"同一条命令被重复执行两次"导致的数值漂移

#### 需求细则

**R1：输入去重（幂等性）**
- 系统应能识别并去重短时间内的重复输入
- 最小方案：对同一 player 的相同命令文本在时间窗内去重，并提示已处理
- 对高风险命令（`/damage`、`/san`、`/luck`、`/sheet set` 等）重复执行必须有保护：禁止静默重复扣数

**R2：掉线后的队列处理**
- 玩家离线时，其未执行的"会改变状态"的排队命令应被自动暂停或丢弃
- 玩家重连后，可通过 `/queue` 查看是否仍有待处理输入，并可选择重发

#### 验收标准

- 同一玩家因网络问题重复发送 `/luck 5`：只会被落地一次，另一条会被提示为重复
- 玩家离线后仍在队列中的攻击命令不会在其离线期间被执行

### M2 验收标准（做完即可进入后续里程碑）

- 2–4 名玩家同团可稳定跑 10+ 轮：
  - 聚光灯/队列规则明确且可操作
  - 并发输入不会导致状态冲突或泄露
- KP-only 与 player-only 权限边界稳定：
  - KP-only 内容不泄露（含 history/recap/status）
  - 仅某玩家可见信息在多人复盘中正确隔离
- 可审计与可复盘：
  - 任意一次敏感输出可追溯到产生原因与权限过滤结果
- 多人写入与机制幂等：
  - 角色卡/物品/关键数值不会被重复执行或被他人误改
  - 战斗/追逐状态在多人并发输入下仍能保持一致

---

## M3：长团记忆（可检索 + 可复盘 + 不泄露）（详细）

### M3-A：事件日志（Event Log）与索引（Append-only）

#### 目标

- 将 session 中的所有关键事件以"可追溯、可检索、可审计"的形式落盘
- 明确日志的最小结构，保证后续摘要/检索/复盘都有稳定数据源

#### 新增/细化命令

| 命令 | 说明 |
|------|------|
| `/history [n]` | 查看最近 n 条（按当前视角过滤） |
| `/log show <event_id>` | 查看单条事件详情（KP-only 或按权限过滤后的版本） |

#### 需求细则

**R1：事件最小字段（冻结）**

每条事件至少包含：

```typescript
interface Event {
  event_id: string
  timestamp: number
  session_id: string
  scene_id: string | null

  // 行为者信息
  actor_player_id: string | null
  actor_role: 'KP' | 'Player'
  controlled_character_id: string | null

  // 事件类型与内容
  type: EventType  // chat/command/roll/ruling/state_change/recap/checkpoint 等
  payload: object  // 结构化；原始文本也需保留

  // 可见性
  visibility: 'public' | 'kp' | 'player:<id>'
}
```

- 事件日志必须是 append-only
- 允许通过"修正事件（retcon）"覆盖语义，但不允许静默篡改历史

**R2：状态变更必须可追溯到事件**
- 任何持久化状态变更（HP/SAN/Luck/flags/线索/角色卡/物品/场景切换）必须能指回产生它的 event_id
- 若存在"人工修正"，必须以明确 type 记录原因与影响范围

#### 验收标准

- 任意一次 `/damage`、`/san`、`/luck` 的落地，都能通过 `/log show` 查到：
  - 原因、计算过程（如有）、前后数值、event_id
- 进行一次 `/retcon`：日志中能看到"修正事件"，且原事件不会消失

### M3-B：结构化摘要（Checkpoints / Scene / Session）

#### 目标

- 为长 Session 提供可重建的"低成本记忆"：检查点摘要、场景摘要、session 总结
- 摘要必须结构化，便于检索与"按视角过滤"

#### 新增/细化命令

| 命令 | 说明 | 权限 |
|------|------|------|
| `/checkpoint` | 生成一个检查点（写入结构化摘要） | KP-only |
| `/recap [me\|public\|all]` | 复用 M2-D；M3 要求输出稳定结构 | 所有 |

#### 需求细则

**R1：摘要最小字段**

每个检查点摘要至少包含：

```typescript
interface Checkpoint {
  checkpoint_id: string
  event_range: {
    from: string  // event_id
    to: string    // event_id
  }
  scene_id: string | null
  timeline: string  // 粗粒度时间线

  // 内容
  main_events: string[]        // 主要事件 3–7 条
  current_leads: string[]      // 当前 leads（2–6 条）
  promises: string[]           // 关键承诺/未竟事项
  key_stats_snapshot: {        // 关键数值快照
    hp: Record<string, number>
    san: Record<string, number>
    luck: Record<string, number>
    // 其他重要状态标记
  }
}
```

**R2：摘要频率与触发**
- 最小触发策略（二选一即可，但必须固定）：
  - KP 显式 `/checkpoint`
  - 系统在"场景结束/战斗结束/追逐结束"后建议 KP 创建检查点

#### 验收标准

- 跑 10+ 轮后生成 `/checkpoint`：检查点中能看到 leads、关键事件、关键数值快照
- 中断恢复后执行 `/recap`：能基于最近检查点与事件日志输出一致的摘要（不凭空编）

### M3-C：记忆检索（Memory Search）与引用

#### 目标

- 支持从"事件日志 + 摘要 + 线索账本"中检索信息
- 能将答案绑定到可引用的证据
- 玩家与 KP 的检索结果必须遵循可见性（不泄露）

#### 新增/细化命令

| 命令 | 说明 |
|------|------|
| `/memory "<query>" [topk=N]` | 在本 session 记忆中检索 |
| `/memory show <mem_id>` | 展开某条记忆引用（按视角过滤） |

#### 需求细则

**R1：检索结果必须可引用**

`/memory` 返回的每条结果必须包含：

```typescript
interface MemoryResult {
  mem_id: string
  source_type: 'event' | 'checkpoint' | 'clue'
  summary: string           // 简短摘要（1–2 句）
  event_id: string | null   // 至少一个
  checkpoint_id: string | null
  visibility: 'public' | 'kp' | 'player:<id>'
}
```

**R2：按视角过滤（不泄露）**
- 玩家调用 `/memory` 只能检索到其可见内容（public + player:<id>）
- KP 调用 `/memory` 可见全部
- KP 对玩家复述时，必须通过叙事而不是直接贴 KP-only 原文

**R3：最小"防幻觉"约束**
- 当模型使用记忆回答问题时，必须优先引用 `/memory` 的结果
- 没有证据时要明确不确定并建议用关键词再检索

#### 验收标准

- 玩家问"我们答应过谁什么？"：
  - KP 或玩家执行 `/memory "答应"` 能检索到承诺条目
  - 能用 `/memory show` 复核
- 存在 player-only 线索时：
  - 非该玩家执行 `/memory` 搜不到该线索
  - KP 能搜到且不会因摘要/检索而泄露给其他玩家

### M3-D：恢复与一致性（Resume Consistency）

#### 目标

- `/resume_session <id>` 后，KP 与玩家都能恢复到"正确的当前状态 + 正确的已知信息"
- 确保复盘/检索与当前状态一致：不会出现"回忆说 HP=10 但状态里 HP=6"这种漂移

#### 需求细则

**R1：状态快照与检查点一致**
- 检查点摘要中的关键数值快照必须能与当时状态对齐（通过 event_id 链可复核）
- 若后续发生 retcon/人工修正，必须：
  - 生成新的检查点或标记旧检查点已过期
  - 复盘输出必须优先使用最新有效检查点

#### 验收标准

- 在产生一次伤害与一次 SAN 损失后创建检查点；中断后恢复：
  - `/status` 的数值与 `/recap` 的快照一致
  - `/memory` 能检索到对应事件并可引用

### M3 验收标准（做完即可进入 M4）

- 记忆可用：
  - 10+ 轮的对话后，能用 `/recap` 与 `/memory` 找回关键人名/线索/承诺/当前 leads
- 不泄露：
  - KP-only 与 player-only 内容在 history/recap/memory 下都不会跨视角泄露
- 可审计：
  - 任意一个记忆结论都能追溯到 event_id/checkpoint_id/clue_id（至少一个）

---

## M4：模组/规则书知识库（可读模组、可查规则）（详细）

### M4-A：模组导入与场景包校验（Module Ingestion & Validation）

#### 目标

- 支持导入"模组/剧本/脚本"，并转换/落地为 M0 冻结的"场景包"结构
- 导入过程中必须做结构校验：缺字段可给出错误或降级策略，但不能静默吞掉关键字段

#### 新增/细化命令

| 命令 | 说明 | 权限 |
|------|------|------|
| `/module upload <path\|url>` | 导入模组来源 | KP-only |
| `/module list` | 列出已导入模组 | 所有 |
| `/module show <module_id>` | 查看模组元信息与场景列表 | 所有 |
| `/module validate <module_id>` | 运行结构校验并返回问题清单 | KP-only |
| `/scene list` | 列出当前模组的 scene_id 列表与简述 | 所有 |

#### 需求细则

**R1：导入结果必须是"可引用对象"**
- 导入后的场景包必须生成稳定引用：
  - module_id、scene_id、npc_id、clue_id、handout_id
- 所有可引用对象必须带可见性标记（public/kp/player:<id>）

**R2：结构校验最小集**

至少校验以下冻结字段是否存在且合法（来自 M0-B）：

| 校验项 | 字段 | 处理 |
|--------|------|------|
| 模组元信息 | 标题、年代/地点、推荐玩家数、风格、KP-only 标签 | 缺失则拒绝导入 |
| 场景 | scene_id、入口触发、退出条件 | 缺失则警告 |
| 可移动线索 | 每场景 2–4 条 | 数量不足必须警告 |
| NPC/地点 | 结构化字段 | 缺失必须警告或拒绝导入 |

- 校验失败时必须给出：问题类型、位置（module_id/scene_id/字段名）、建议修复

#### 验收标准

- 导入一个最小模组（至少 2 个场景、每场景≥2 线索）：
  - `/module list` 可见；`/module show` 能列出场景
  - `/module validate` 无致命错误（或仅有明确的可接受警告）

### M4-B：规则书/资料入库（Chunking + Citations）

#### 目标

- 支持将规则书（或规则摘录）入库为可检索片段，并提供稳定引用（citation_id）
- 为 M1-E 的规则问答提供可复核的引用基础

#### 新增/细化命令

| 命令 | 说明 | 权限 |
|------|------|------|
| `/kb ingest <source> <path\|url>` | 入库资料 | KP-only |
| `/kb sources` | 列出已入库来源 | 所有 |

#### 需求细则

**R1：引用最小字段**

每个可检索片段必须包含：

```typescript
interface Citation {
  citation_id: string
  source_id: string
  locator: {
    page: number | null      // 页码
    chapter: string | null   // 章节
    paragraph: number | null // 段落号
  }
  text: string               // 正文片段
  visibility: 'public' | 'kp' | 'player:<id>'
}
```

- 片段正文必须以"资料"对待：不得包含可执行指令语义（见 M4-E）

**R2：可复核**
- 对任何引用，必须能通过 `/kb show <citation_id>` 看到完整上下文（至少扩展上下各一小段）

#### 验收标准

- 导入一份规则资料后：
  - `/kb sources` 能列出来源
  - `/kb show` 能展开任意 citation_id 的上下文

### M4-C：统一检索接口（Rules + Module + Memory）

#### 目标

- 提供统一的检索命令，让 KP 在推进剧情或裁定规则时能快速找到"依据"
- 检索必须按视角过滤，且结果可引用

#### 新增/细化命令

| 命令 | 说明 |
|------|------|
| `/search "<query>" [scope=rulebook\|module\|memory\|all] [topk=N]` | 统一检索入口 |
| `/search show <ref_id>` | 展开引用 |

#### 需求细则

**R1：结果结构稳定**

`/search` 的每条结果必须包含：

```typescript
interface SearchResult {
  ref_id: string           // citation_id/mem_id/module_ref
  source_type: 'citation' | 'memory' | 'module'
  summary: string
  relevance: number        // 相关系数（可粗粒度）
  visibility: 'public' | 'kp' | 'player:<id>'
}
```

**R2：与既有命令兼容**
- 保留 `/kb rule` 与 `/memory` 作为专用快捷方式
- 其输出结构应与 `/search` 对齐（至少能映射到同一 ref_id 体系）

#### 验收标准

- 用同一个 query：
  - `scope=rulebook` 能返回 citation
  - `scope=module` 能返回 module_ref（如 scene/npc/clue）
  - `scope=memory` 能返回 mem_id
- 任意 ref_id 都能用 `/search show` 复核

### M4-D：剧情推进的"模组落地"（Grounded Narration）

#### 目标

- KP 在叙事推进时应优先引用模组要素（NPC/地点/线索/手递物），避免"凭空编造关键设定"
- 允许适度即兴，但必须可追溯：新增实体要写入状态并标记为 improvised

#### 需求细则

**R1：模组优先与引用**
- 当当前场景存在可用线索/NPC/地点时，KP 的输出应优先从模组对象中选取
- 可在审计视角标注引用（无需对玩家展示引用 ID，但 KP 侧必须可追溯）
- 若 KP 引入"模组外新增实体"，必须：
  - 给出合理叙事来源
  - 写入世界状态
  - 标记 `improvised=true`

**R2：线索可移动与防卡死**
- 当玩家走偏导致当前线索不可达时，KP 应使用"可移动线索"机制将线索投放到可达交互点
- 并写入线索账本来源

#### 验收标准

- 选用一个含 2 场景的模组：在场景推进中至少 2 次引用到模组 NPC/线索（可审计追溯到 module_ref）
- 引入一个即兴 NPC：必须被写入状态并标记为 improvised

### M4-E：访问控制与防注入（Module/KB as Data）

#### 目标

- 防止模组/规则资料中的文本被当作"系统指令"执行（数据即数据）
- 确保 KP-only 的模组秘密、隐藏线索不会通过检索/复盘/引用泄露给玩家

#### 需求细则

**R1：资料防注入**
- 无论模组文本/规则书片段中出现何种指令式措辞，系统必须将其视为普通文本
- 不影响运行规则与门禁
- 检索结果与引用展示必须带来源标记（source_id/module_id），避免被伪装成系统输出

**R2：KP-only 不泄露**
- 对 module_ref/citation/memory 的所有输出必须按 viewer 过滤：
  - 玩家不可通过 `/search`、`/kb show`、`/history`、`/recap`、`/memory show` 获取 KP-only 内容
  - KP 可见全部，但对玩家复述时不得直接贴 KP-only 原文

#### 验收标准

- 预置：模组或规则书片段包含"注入文本"（诱导越权/越界）：
  - 系统不执行、不采纳为指令，仍保持 TRPG-only 门禁
- 预置：模组存在 KP-only 秘密线索：
  - KP 能检索到
  - 玩家侧通过任何检索/复盘接口都检索不到

### M4 验收标准（做完即可进入 M5）

- 可读模组：
  - 至少 1 个模组可导入、可校验、可列出场景，并能在推进时引用到模组要素
- 可查规则：
  - 至少 3 个常见规则问题可通过检索给出"带 citations 的答案"，并可用 show 复核
- 可审计与不泄露：
  - 引用来源可追溯（module_ref/citation_id/source_id）
  - KP-only 内容不会泄露到玩家视角的检索/复盘输出

---

## M5：CoC 全功能补齐（v1"越多越好"落地）（详细）

### M5-A：SAN/疯狂机制补齐（临时疯狂/不定疯狂/症状与恢复）

#### 目标

- 在保持可玩与可审计的前提下，把 SAN 相关的"后续影响"补齐
- 让 SAN 的触发、扣除、症状落地与解除都可追溯

#### 新增/细化命令

| 命令 | 说明 | 权限 |
|------|------|------|
| `/san check <reason> <loss>` | 进行一次 SAN 检定 | - |
| `/san apply <points> <reason>` | 直接扣除 SAN | KP-only |
| `/insanity apply <temp\|indef> <symptom> [duration]` | 落地一次疯狂状态 | KP-only |
| `/insanity clear <id>` | 解除/转化某条疯狂状态 | KP-only |

#### 需求细则

**R1：SAN 结算结构稳定**

每次 SAN 事件输出（KP 侧审计结构）至少包含：

```typescript
interface SANEvent {
  reason: string
  loss_spec: string        // "1d6" 或 "1/1d6" 或 "5"
  final_loss: number
  trigger_reference: string // 可引用：citation_id/module_ref/event_id
  san_before: number
  san_after: number
  insanity_triggered: 'temp' | 'indef' | 'none'
  symptom_id: string | null // 如触发
}
```

**R2：疯狂状态是"可执行的约束"而非纯文案**
- 疯狂状态必须写入角色状态字段
- 在后续回合中对 KP 的叙事与行动建议产生约束：
  - 至少要求在每轮 `[Status]` 或 `[Next]` 中提示"当前症状影响"
  - 若玩家行动与症状强冲突，KP 应提示并要求玩家改述或使用 `/retcon`

**R3：恢复与治疗的最小闭环**
- 解除/减轻疯狂必须通过命令落地，并记录：
  - 谁做了什么治疗/安抚
  - 时间跨度

#### 验收标准

- 跑一次 SAN 检定（`/san check`）并触发临时疯狂：
  - 结果可审计（reason/loss_spec/最终 loss/前后 SAN）
  - 疯狂状态写入角色卡，并在后续 3 轮输出中持续提示其影响
- 通过治疗/时间推进解除疯狂：`/insanity clear` 可追溯并更新状态

### M5-B：战斗完整流程补齐（远程/护甲/弹药/故障/更完整对抗）

#### 目标

- 将 M1-C 的战斗从"近战为主 + 远程简化"升级到可跑短模组的完整度：
  - 远程射击与距离/掩体 penalty
  - 护甲与伤害减免
  - 弹药/装填
  - 重大成功/大失败导致的额外效果

#### 新增/细化命令

| 命令 | 说明 | 权限 |
|------|------|------|
| `/rules mode <basic\|standard\|extended>` | 规则复杂度开关 | KP-only |
| `/range set <target> <band>` | 设置距离带 | KP-only |
| `/ammo set <weapon> <current>/<max>` | 设置弹药 | KP/玩家对自己 |
| `/reload <weapon>` | 装填/换弹 | - |
| `/armor set <target> <value>` | 设置护甲值 | KP-only |

#### 需求细则

**R1：远程命中与修正**
- 在 extended 模式下，远程攻击结算必须明确给出 penalty 来源
- penalty/bonus 的来源必须可审计（写入事件 payload）

**R2：弹药与装填的最小一致性**
- 若武器被标记为需要弹药，则每次射击必须减少 ammo
- ammo 为 0 时必须拒绝射击并提示 `/reload`
- 装填必须消耗一次行动（与回合/聚光灯联动）

**R3：护甲与伤害减免**
- 若目标存在 armor，则 `/damage` 落地时必须计算减免
- 输出：原始伤害、护甲、最终伤害
- 护甲来源必须可追溯

**R4：故障与武器损坏（最小可用）**
- 若启用大失败：必须给出清晰后果
- 写入状态或 flags

#### 验收标准

- 远程战斗片段（至少 3 轮）：
  - distance/cover 导致的 penalty 可解释
  - ammo 会减少，ammo=0 时不能射击
  - 护甲减免被正确计算且可审计
- 触发一次故障：能看到后果写入状态，并影响后续行动

### M5-C：追逐补齐（多参与者/载具要素/更丰富障碍与代价）

#### 目标

- 将 M1-D 的追逐从"单场景可跑"扩展为可覆盖短模组常见情况：
  - 多追逐者/多逃跑者
  - 载具/地形标签对 Move/距离的影响
  - 更稳定的障碍生成/选择与代价落地

#### 新增/细化命令

| 命令 | 说明 | 权限 |
|------|------|------|
| `/chase vehicle <actor> <type>` | 声明载具类型 | KP-only |
| `/chase tag add <tag>` | 环境标签调整 | KP-only |
| `/chase tag remove <tag>` | 移除环境标签 | KP-only |
| `/chase obstacle roll` | 生成一个障碍 | KP-only |

#### 需求细则

**R1：多参与者一致性**
- distance/advantage 至少要支持以下二选一（但必须固定一种模型）：
  - "追逐阵营间的相对距离"
  - "每对关键对手的距离"
- 新加入参与者必须有初始化距离，并写入状态

**R2：失败前进必须落到可检索的新局面**
- 追逐失败/中断必须形成新局面，并写入：
  - 进入新场景入口
  - 丢失物品
  - 受伤
  - 引来第三方
  - 计时器推进等

#### 验收标准

- 1 名 PC + 2 名 NPC 的追逐：能稳定跑 5+ 回合，且每回合距离变化可审计
- 触发一次"失败前进"：结局不是原地踏步，且新局面可通过 `/recap` 或 `/history` 找回

### M5-D：成长与战后流程（技能成长/医疗恢复/战利品与资源）

#### 目标

- 支持短模组结束后的最小成长闭环：技能成长（经验检定）、战后治疗与状态清算
- 保持门禁：所有角色卡修改必须通过命令，且可审计

#### 新增/细化命令

| 命令 | 说明 | 权限 |
|------|------|------|
| `/xp mark <skill>` | 标记一次技能成长机会 | KP/玩家对自己 |
| `/xp list` | 查看当前角色的成长标记 | 所有 |
| `/advance roll <skill>` | 进行一次成长检定 | 所有 |
| `/downtime start` | 进入战后流程 | KP-only |
| `/downtime end` | 结束战后流程 | KP-only |

#### 需求细则

**R1：成长检定可追溯**
- 每次成长必须记录：
  - skill、原始值、掷骰、是否提升、提升后值、依据 event_id

**R2：治疗/恢复的窗口化**
- 恢复（HP/SAN 的缓慢恢复、伤口治疗等）必须限定在 downtime 窗口内
- 或 KP 显式命令允许
- 避免战斗中滥用

#### 验收标准

- 标记并完成一次技能成长：技能值变化可审计且可在 `/sheet show` 看到
- 进入 downtime 后执行一次恢复/治疗相关落地：过程有记录且不会越权修改其他玩家角色

### M5-E：端到端短模组验收（调查 + 战斗 + SAN 冲击 +（可选）追逐）

#### 目标

- 用一个 1–3 场景短模组跑通"调查推进 + 规则运转 + 可审计复盘"的完整体验
- 验证 M0–M4 的能力在 M5 的复杂度下仍不崩：门禁/可见性/记忆/检索/引用都稳定

#### 验收标准（M5 总验收）

- 跑一个含调查 + 战斗 + SAN 冲击的短模组（可选追逐）：
  - 至少发生 1 次规则检索并引用 citation
  - 至少发生 1 次模组要素引用（module_ref：NPC/线索/手递物）
  - 至少发生 1 次 SAN 检定并触发后续影响
  - 至少发生 1 次远程战斗事件（含 ammo 或护甲其中一项）
- 可复盘：
  - `/recap` 能总结关键事件与 leads
  - `/memory` 能检索回关键承诺/线索
  - 任意一次数值变化（HP/SAN/Luck/技能）都能指回对应 event_id

---

## M6：体验打磨（防卡死与节奏控制）（详细）

### M6-A：Leads 面板与线索账本体验（玩家不迷路）

#### 目标

- 将 FR-07 的"线索与 leads"从数据字段升级为实际可玩的桌面机制
- 在不强推铁路的前提下，稳定维持 2–4 个可行动的推进方向

#### 新增/细化命令

| 命令 | 说明 | 权限 |
|------|------|------|
| `/leads` | 查看当前 leads | 所有 |
| `/leads add "<text>" [link=...]` | 新增一条 lead | KP-only |
| `/leads close <lead_id> [reason]` | 关闭/完成某条 lead | KP-only |
| `/clues list [scope=me\|public\|all]` | 列出线索账本 | 所有 |
| `/clues show <clue_id>` | 查看线索详情 | 所有 |

#### 需求细则

**R1：leads 的结构与来源可追溯**

每条 lead 最小字段：

```typescript
interface Lead {
  lead_id: string
  text: string              // 一句话目标
  status: 'open' | 'closed'
  created_from: string      // event_id/module_ref/clue_id 任一
  links: string[]           // 0..n
  recommended_action: string // 推荐下一步动作
}
```

- leads 的变更（add/close）必须写入事件日志，可在 `/history` 与 `/recap` 中复盘

**R2：Leads 数量与质量阈值**
- KP 在每轮输出的 `[Next]` 或等价块中，必须给出：
  - 至少 1 个"立即可做"的动作建议（命令级）
  - 若当前 open leads < 2，则必须补齐到 2–4 条

#### 验收标准

- 在玩家发散探索 10 轮过程中：
  - `/leads` 始终保持 2–4 条 open leads（允许短暂波动，但必须在 1 轮内恢复）
  - 每条 lead 都能追溯到来源（至少一个 event_id/clue_id/module_ref）

### M6-B：防卡死与失败前进（Stuck Recovery）

#### 目标

- 当玩家走偏、信息不足或反复失败时，系统能主动提供"失败前进"的新局面
- 失败前进必须可审计（发生了什么代价/计时器推进/线索重投放）

#### 新增/细化命令

| 命令 | 说明 | 权限 |
|------|------|------|
| `/hint [level=1\|2\|3]` | 请求提示 | 玩家 |
| `/nudge` | 推进一小步 | KP-only |
| `/clock advance <step> [reason]` | 推进计时器/压力源 | KP-only |

#### 需求细则

**R1：卡死检测的最小启发式**

至少支持一种卡死检测（实现任选其一，但必须固定）：
- 连续 N 轮无新线索/无 leads 变化/无 flags 变化
- 连续 N 次关键检定失败且无状态推进

检测到卡死时，KP 必须触发：
- 生成提示（/hint 风格输出），或
- 通过 `/nudge` 落地一次失败前进

**R2：失败前进的落地要求**

失败前进必须产生"新局面"，并至少写入一种可检索变化：
- 新线索（clue）
- 计时器推进（clock）
- NPC 态度变化/介入
- 资源损失/受伤/SAN 冲击（必须显式可见、可追溯；必要时要求确认）

#### 验收标准

- 人为构造"卡死输入序列"（例如连续 5 轮无效搜索）：
  - KP 能给出 `/hint` 输出（不同 level 更具体，但不剧透 KP-only）
  - 或通过 `/nudge` 落地失败前进，并在 `/recap` 中可追溯

### M6-C：越界拒绝与命令引导（Better Guardrails UX）

#### 目标

- 保持 NFR-02 的门禁强度，同时让拒绝更友好
- 降低新玩家学习成本：把"命令怎么写/下一步怎么做"做成可交互引导

#### 新增/细化命令

| 命令 | 说明 |
|------|------|
| `/help [topic]` | 按主题输出（如 combat/chase/sanity/kb/memory） |
| `/examples [topic]` | 给出 3–5 个常见输入示例 |

#### 需求细则

**R1：拒绝模板必须结构稳定**

对越界/格式错误/权限不足的输入，系统输出至少包含：

```
[Refusal]
拒绝原因（简短、明确）

[Allowed]
允许的替代表达（至少 1 条命令示例）

[Next]
下一步建议（指向 `/help` 或相关命令）
```

**R2：拒绝不改变状态**
- 拒绝本身不得隐式推进剧情或修改数值
- 任何推进必须通过 KP-only 或明确命令链完成

#### 验收标准

- 玩家输入跑团外任务：系统稳定输出拒绝模板，并给出可用命令替代
- 玩家用自然语言试图改数值（例如"我花 5 点幸运"）：系统拒绝直接改动，并引导到 `/luck 5`

### M6-D：输出稳定性与信息密度（可读性/节奏）

#### 目标

- KP 输出更易读：固定块、可扫描、默认不冗长；需要时可展开详情
- 降低"长消息墙"对桌面节奏的伤害

#### 新增/细化命令

| 命令 | 说明 |
|------|------|
| `/format <compact\|standard\|verbose>` | 设置输出信息密度 |
| `/detail on\|off` | 切换是否输出审计细节 |

#### 需求细则

**R1：每轮输出最小块**

每轮至少包含三个块（标题可固定）：

```
[KP]
叙事与当前反馈

[State]
关键状态简报（HP/SAN/Luck/异常/当前场景/计时器）按视角过滤

[Next]
1–3 条下一步动作建议（命令级）
```

**R2：可展开而非默认全展开**
- 在 compact/standard 下，审计细节默认折叠为"可查看引用/详情"的提示
- 通过 `/detail on` 或 show 类命令展开

#### 验收标准

- 在 compact 模式跑 10 轮：每轮输出不超过约定上限（例如 1200–1800 字符）
- 但仍包含 `[KP]/[State]/[Next]`
- 在 verbose + detail on：同样输入能看到完整审计细节

### M6 验收标准（做完即可进入 v1 结束或后续迭代）

- **防卡死**：玩家发散探索不会长期无事可做；当卡死时能通过 hint/nudge/失败前进回到可玩态
- **节奏控制**：leads 始终维持 2–4 个可执行方向，且能关闭/完成并形成新的推进
- **门禁体验**：拒绝稳定、可理解、可操作，且拒绝不改变状态
- **可读性**：输出块稳定，信息密度可配置，玩家默认不被审计细节淹没

---

## 附录

### 数据结构总览

本文档中涉及的所有核心数据结构汇总：

```typescript
// 角色卡
interface Character {
  character_id: string
  name: string
  occupation: string
  age: number
  residence: string
  background: string
  attributes: CharacterAttributes
  derived: DerivedStats
  skills: Record<string, number>
  status: CharacterStatus
  inventory: string[]
  cash: number
  spending_level: number
}

// NPC
interface NPC {
  npc_id: string
  name: string
  appearance: string
  motivation: string
  secret: string
  clue_associations: string[]
  attitude_toward_pc: string
}

// 地点
interface Location {
  location_id: string
  name: string
  description: string
  interaction_points: string[]
  discoverable_clues: string[]
  dangers: string[]
}

// 线索
interface Clue {
  clue_id: string
  text: string
  source: string
  visibility: 'public' | 'kp' | 'player:<id>'
  portable: boolean
  handout_ref: string | null
}

// 线索账本
interface ClueLedger {
  clue_id: string
  text: string
  source: string
  credibility: number
  ownership: string
  points_to: string[]
  verified: boolean
  visibility: 'public' | 'kp' | 'player:<id>'
}

// 世界状态
interface WorldState {
  current_scene: string | null
  current_location: string
  time_period: string
  important_npcs: Array<{
    npc_id: string
    attitude: string
  }>
  current_threats: Threat[]
  timer: Timer | null
  current_leads: Lead[]
}

// Leads
interface Lead {
  lead_id: string
  text: string
  status: 'open' | 'closed'
  created_from: string
  links: string[]
  recommended_action: string
}

// 事件
interface Event {
  event_id: string
  timestamp: number
  session_id: string
  scene_id: string | null
  actor_player_id: string | null
  actor_role: 'KP' | 'Player'
  controlled_character_id: string | null
  type: EventType
  payload: object
  visibility: 'public' | 'kp' | 'player:<id>'
}

// 检查点
interface Checkpoint {
  checkpoint_id: string
  event_range: {
    from: string
    to: string
  }
  scene_id: string | null
  timeline: string
  main_events: string[]
  current_leads: string[]
  promises: string[]
  key_stats_snapshot: {
    hp: Record<string, number>
    san: Record<string, number>
    luck: Record<string, number>
  }
}

// 检索结果
interface MemoryResult {
  mem_id: string
  source_type: 'event' | 'checkpoint' | 'clue'
  summary: string
  event_id: string | null
  checkpoint_id: string | null
  visibility: 'public' | 'kp' | 'player:<id>'
}

// 引用
interface Citation {
  citation_id: string
  source_id: string
  locator: {
    page: number | null
    chapter: string | null
    paragraph: number | null
  }
  text: string
  visibility: 'public' | 'kp' | 'player:<id>'
}

// SAN 事件
interface SANEvent {
  reason: string
  loss_spec: string
  final_loss: number
  trigger_reference: string
  san_before: number
  san_after: number
  insanity_triggered: 'temp' | 'indef' | 'none'
  symptom_id: string | null
}
```

---

**文档状态**: ✅ 已完成，整合自原始需求文档
