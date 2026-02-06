# M0: 规范冻结

**目标**: 完成所有规范的定义和冻结，为后续开发奠定基础
**周期**: 1 周 (5 工作日)
**优先级**: P0
**状态**: [ ]

---

## 阶段目标

- [ ] 明确命令集格式和语义
- [ ] 定义场景包数据格式
- [ ] 确立状态字段和数据结构
- [ ] 完成 UI 设计规范
- [ ] 规划组件库

---

## 任务清单

### 1.1 命令集定义 (10h)

| ID | 任务 | 类型 | 预估工时 | 依赖 | 状态 |
|----|------|------|----------|------|------|
| M0-001 | [ ] 定义核心命令清单 (15个) | spec | 2h | - | [ ] |
| M0-002 | [ ] 编写 /help 命令规范 | spec | 1h | M0-001 | [ ] |
| M0-003 | [ ] 编写 /status 命令规范 | spec | 1h | M0-001 | [ ] |
| M0-004 | [ ] 编写检定命令规范 (/roll) | spec | 1h | M0-001 | [ ] |
| M0-005 | [ ] 编写 /push /luck 命令规范 | spec | 1h | M0-001 | [ ] |
| M0-006 | [ ] 编写战斗命令规范 (/combat) | spec | 1h | M0-001 | [ ] |
| M0-007 | [ ] 编写追逐命令规范 (/chase) | spec | 1h | M0-001 | [ ] |
| M0-008 | [ ] 编写 SAN 命令规范 (/san) | spec | 1h | M0-001 | [ ] |
| M0-009 | [ ] 编写规则命令规范 (/rule) | spec | 1h | M0-001 | [ ] |

#### 命令清单详情

```
基础命令 (5):
  /help          - 显示帮助信息
  /status        - 显示当前状态
  /leads         - 显示可选行动
  /rule [query]  - 规则问答
  /quit          - 结束会话

检定命令 (4):
  /roll [skill]  - 技能检定 (支持属性)
  /push          - 推骰 (失败后可推)
  /luck [n]      - 花幸运 (需引用事件)
  /diff [n]      - 设置难度 (KP)

战斗命令 (3):
  /combat start  - 开始战斗
  /combat action - 执行战斗动作
  /combat end    - 结束战斗

状态命令 (3):
  /san check     - SAN 检定
  /heal [n]      - 治疗
  /rest [type]   - 休息
```

---

### 1.2 命令语法规范 (8h)

| ID | 任务 | 类型 | 预估工时 | 依赖 | 状态 |
|----|------|------|----------|------|------|
| M0-010 | [ ] 编写命令语法 BNF 范式 | spec | 2h | M0-001 | [ ] |
| M0-011 | [ ] 编写命令参数正则表达式 | spec | 2h | M0-010 | [ ] |
| M0-012 | [ ] 编写命令响应格式规范 | spec | 2h | M0-010 | [ ] |
| M0-013 | [ ] 定义命令别名和快捷方式 | spec | 2h | M0-010 | [ ] |

#### 语法规范示例

```bnf
<roll_command> ::= "/roll" [ <skill_name> | <attribute_name> ] [ "difficulty" "=" <number> ]
<skill_name> ::= [a-z_]+   # 例如: "library_use", "hide"
<attribute_name> ::= "STR" | "CON" | "DEX" | "APP" | "POW" | "INT" | "SIZ" | "EDU"
<number> ::= [0-9]+

<response_format> ::= "[" <role> "]" <content> "\n" [ "[" "State" "]" <changes> ]
```

---

### 1.3 场景包格式定义 (18h)

| ID | 任务 | 类型 | 预估工时 | 依赖 | 状态 |
|----|------|------|----------|------|------|
| M0-014 | [ ] 设计场景包根结构 | spec | 2h | - | [ ] |
| M0-015 | [ ] 定义 metadata 元信息结构 | spec | 2h | M0-014 | [ ] |
| M0-016 | [ ] 定义 scenes 场景集合结构 | spec | 2h | M0-014 | [ ] |
| M0-017 | [ ] 定义 NPC 角色数据结构 | spec | 2h | M0-016 | [ ] |
| M0-018 | [ ] 定义 Location 地点结构 | spec | 2h | M0-016 | [ ] |
| M0-019 | [ ] 定义 Clue 线索数据结构 | spec | 2h | M0-016 | [ ] |
| M0-020 | [ ] 定义 Handout 手递物格式 | spec | 2h | M0-016 | [ ] |
| M0-021 | [ ] 定义 transitions 跳转结构 | spec | 2h | M0-016 | [ ] |
| M0-022 | [ ] 编写场景包 JSON Schema | spec | 4h | M0-014 | [ ] |

#### 场景包结构

```json
{
  "metadata": {
    "id": "script_unique_id",
    "title": "脚本标题",
    "version": "1.0.0",
    "author": "作者名",
    "description": "简短描述",
    "duration": "2-4h",
    "player_count": "3-5",
    "tags": ["入门", "现代"],
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-01-01T00:00:00Z"
  },
  "scenes": {
    "scene_001": {
      "id": "scene_001",
      "title": "场景标题",
      "order": 1,
      "narrative": {
        "opening": "开场叙事文本...",
        "alternate": ["变体1...", "变体2..."]
      },
      "npcs": [],
      "locations": [],
      "clues": [],
      "handouts": [],
      "transitions": [],
      "requirements": {}
    }
  },
  "shared": {
    "npcs": {},
    "locations": {},
    "clues": {},
    "handouts": {}
  }
}
```

---

### 1.4 场景包校验规则 (8h)

| ID | 任务 | 类型 | 预估工时 | 依赖 | 状态 |
|----|------|------|----------|------|------|
| M0-023 | [ ] 编写必填字段校验规则 | spec | 2h | M0-022 | [ ] |
| M0-024 | [ ] 编写类型校验规则 | spec | 2h | M0-023 | [ ] |
| M0-025 | [ ] 编写引用完整性校验 | spec | 2h | M0-023 | [ ] |
| M0-026 | [ ] 编写循环引用检测规则 | spec | 2h | M0-023 | [ ] |

#### 校验规则示例

```typescript
interface ValidationRule {
  field: string;
  type: 'required' | 'type' | 'enum' | 'unique' | 'reference' | 'circular';
  params?: any;
  message: string;
}

const RULES: ValidationRule[] = [
  { field: 'metadata.id', type: 'required', message: '脚本 ID 必填' },
  { field: 'metadata.version', type: 'required', message: '版本号必填' },
  { field: 'scenes.*.id', type: 'unique', message: '场景 ID 必须唯一' },
  { field: 'scenes.*.npcs.*.ref', type: 'reference', ref: 'shared.npcs',
    message: 'NPC 引用必须存在于 shared.npcs' },
  { field: 'scenes.*.transitions.*.target', type: 'reference',
    ref: 'scenes', message: '跳转目标必须存在' },
]
```

---

### 1.5 状态字段定义 (16h)

| ID | 任务 | 类型 | 预估工时 | 依赖 | 状态 |
|----|------|------|----------|------|------|
| M0-027 | [ ] 定义 SessionState 结构 | spec | 2h | - | [ ] |
| M0-028 | [ ] 定义 CharacterState 角色状态 | spec | 2h | M0-027 | [ ] |
| M0-029 | [ ] 定义 Attribute 属性结构 | spec | 2h | M0-028 | [ ] |
| M0-030 | [ ] 定义 Skill 技能结构 | spec | 2h | M0-028 | [ ] |
| M0-031 | [ ] 定义 CombatState 战斗状态 | spec | 2h | M0-027 | [ ] |
| M0-032 | [ ] 定义 ChaseState 追逐状态 | spec | 2h | M0-027 | [ ] |
| M0-033 | [ ] 定义 SAN/疯狂状态 | spec | 2h | M0-027 | [ ] |
| M0-034 | [ ] 定义 Leads 机制结构 | spec | 2h | M0-027 | [ ] |

#### 状态结构定义

```typescript
// 角色状态
interface CharacterState {
  // 标识
  character_id: string;
  name: string;

  // 属性 (STR/CON/DEX/APP/POW/INT/SIZ/EDU)
  attributes: {
    STR: number; CON: number; DEX: number; APP: number;
    POW: number; INT: number; SIZ: number; EDU: number;
  };

  // 派生数值
  derived: {
    HP: number; HP_max: number;
    MP: number; MP_max: number;
    SAN: number; SAN_max: number;
    Luck: number; Luck_max: number;
    Move: number;
  };

  // 技能
  skills: Record<string, number>;

  // 状态
  status: 'alive' | 'unconscious' | 'dying' | 'dead' | 'insane';
  inventory: string[];
  notes: string;
}

// 战斗状态
interface CombatState {
  in_combat: boolean;
  round: number;
  current_actor: string;
  participants: {
    id: string;
    name: string;
    initiative: number;
    hp: number;
    status: string;
  }[];
  initiative_order: string[];
}
```

---

### 1.6 事件日志结构 (8h)

| ID | 任务 | 类型 | 预估工时 | 依赖 | 状态 |
|----|------|------|----------|------|------|
| M0-035 | [ ] 定义 Event 基础结构 | spec | 2h | - | [ ] |
| M0-036 | [ ] 定义 EventType 枚举 | spec | 2h | M0-035 | [ ] |
| M0-037 | [ ] 定义 Visibility 可见性 | spec | 2h | M0-035 | [ ] |
| M0-038 | [ ] 定义 StateChange 变更结构 | spec | 2h | M0-035 | [ ] |

#### 事件结构

```typescript
interface GameEvent {
  event_id: string;
  session_id: string;
  timestamp: string;

  // 参与者
  actor_player_id: string | null;
  actor_role: 'KP' | 'Player';
  controlled_character_id: string | null;

  // 内容
  event_type: EventType;
  raw_message: string;
  parsed_action?: {
    type: string;
    target?: string;
    params?: Record<string, any>;
  };

  // 结果
  state_changes?: StateChange[];
  narration?: string;

  // 可见性
  visibility: 'public' | 'kp' | 'player:*';

  // 元数据
  metadata: {
    client_timestamp: number;
    sequence: number;
  };
}

type EventType =
  | 'message' | 'action' | 'roll' | 'roll_pushed'
  | 'luck_spent' | 'combat_start' | 'combat_action'
  | 'combat_end' | 'chase_start' | 'chase_action'
  | 'chase_end' | 'san_check' | 'san_loss'
  | 'madness_start' | 'madness_end' | 'damage'
  | 'heal' | 'checkpoint' | 'scene_change';
```

---

### 1.7 UI 设计规范 (16h)

| ID | 任务 | 类型 | 预估工时 | 依赖 | 状态 |
|----|------|------|----------|------|------|
| M0-039 | [ ] 定义配色方案 (暗色/亮色) | design | 2h | - | [ ] |
| M0-040 | [ ] 定义字体层级规范 | design | 2h | M0-039 | [ ] |
| M0-041 | [ ] 定义间距系统 (4px grid) | design | 2h | M0-039 | [ ] |
| M0-042 | [ ] 定义消息气泡样式 | design | 2h | M0-039 | [ ] |
| M0-043 | [ ] 定义状态指示器样式 | design | 2h | M0-039 | [ ] |
| M0-044 | [ ] 定义骰子结果展示规范 | design | 2h | M0-039 | [ ] |
| M0-045 | [ ] 定义动画/过渡效果规范 | design | 2h | M0-039 | [ ] |
| M0-046 | [ ] 定义响应式断点规范 | design | 2h | M0-039 | [ ] |

#### 配色方案

```typescript
const THEME = {
  light: {
    // 背景
    background: '#ffffff',
    surface: '#f8f9fa',
    surfaceHover: '#f1f3f5',

    // 文字
    text: '#212529',
    textSecondary: '#6c757d',
    textMuted: '#adb5bd',

    // 主题色
    primary: '#5c6bc0',
    primaryHover: '#3949ab',
    secondary: '#78909c',

    // 状态
    success: '#66bb6a',
    warning: '#ffa726',
    danger: '#ef5350',
    info: '#42a5f5',

    // KP / Player 区分
    kp: '#7e57c2',
    player: '#26a69a',
  },
  dark: {
    // 暗色变体...
  }
}
```

---

### 1.8 组件库规划 (8h)

| ID | 任务 | 类型 | 预估工时 | 依赖 | 状态 |
|----|------|------|----------|------|------|
| M0-047 | [ ] 梳理 shadcn/ui 组件需求 | plan | 2h | - | [ ] |
| M0-048 | [ ] 定义游戏专用组件列表 | plan | 2h | M0-047 | [ ] |
| M0-049 | [ ] 编写 MessageBubble API | spec | 1h | M0-048 | [ ] |
| M0-050 | [ ] 编写 DiceRoll API | spec | 1h | M0-048 | [ ] |
| M0-051 | [ ] 编写 StatePanel API | spec | 1h | M0-048 | [ ] |
| M0-052 | [ ] 编写 CombatTracker API | spec | 1h | M0-048 | [ ] |

#### 游戏专用组件

```
消息相关:
  - MessageBubble     - 消息气泡
  - MessageList       - 消息列表
  - TypingIndicator   - 打字指示器

游戏相关:
  - DiceRoll          - 骰子展示
  - CombatTracker     - 战斗追踪
  - ChaseTracker      - 追逐追踪
  - SANMeter          - SAN 值条
  - LeadsPanel        - 可选行动
  - SpotlightIndicator - 聚光灯

角色相关:
  - CharacterCard     - 角色卡预览
  - StatusBadge       - 状态徽章
  - InventoryGrid    - 物品栏

UI 工具:
  - ChatInput         - 聊天输入
  - CommandPalette    - 命令面板
  - Timeline          - 时间线
```

---

### 1.9 文档输出 (16h)

| ID | 任务 | 类型 | 预估工时 | 依赖 | 状态 |
|----|------|------|----------|------|------|
| M0-053 | [ ] 编写命令参考手册 | doc | 4h | M0-013 | [ ] |
| M0-054 | [ ] 编写场景包开发指南 | doc | 4h | M0-026 | [ ] |
| M0-055 | [ ] 编写数据字典 | doc | 4h | M0-038 | [ ] |
| M0-056 | [ ] 编写组件设计文档 | doc | 4h | M0-052 | [ ] |

---

## 验收标准

- [ ] 命令集文档完整，格式统一
- [ ] 场景包 Schema 可用 JSON Schema 验证
- [ ] 状态字段定义完整，无遗漏
- [ ] UI 规范可交付设计评审
- [ ] 组件清单和技术选型一致
- [ ] M0 交付物冻结，进入 M1

---

## 交付物清单

| 交付物 | 文件路径 | 说明 |
|--------|----------|------|
| 命令规范 | `docs/specs/commands.md` | 命令语法、响应格式 |
| 场景包格式 | `docs/specs/script-schema.md` | JSON Schema + 示例 |
| 数据字典 | `docs/specs/data-dictionary.md` | 所有状态字段定义 |
| UI 规范 | `docs/specs/ui-guidelines.md` | 设计规范 |
| 组件设计 | `docs/specs/components.md` | 组件 API 设计 |
| API 文档 | `docs/api/openapi.yaml` | OpenAPI 3.0 规范 |

---

## 风险与应对

| 风险 | 可能性 | 影响 | 应对措施 |
|------|--------|------|----------|
| 规范反复修改 | 中 | 高 | 每日评审，快速迭代 |
| 场景包格式过于复杂 | 中 | 中 | MVP 阶段只支持基础字段 |
| UI 规范与 shadcn 冲突 | 低 | 低 | 以 shadcn 为准，扩展为辅 |

---

**负责人**: -
**开始日期**: -
**结束日期**: -
**实际工时**: -
