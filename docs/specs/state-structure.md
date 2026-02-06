# 状态字段结构规范

**版本**: v1.0
**最后更新**: 2026-02-07
**状态**: ✅ 设计完成

---

## 概述

本文档定义 CoC 跑团平台的游戏状态数据结构规范。状态字段是系统运行时的核心数据，采用**分层架构**按更新频率组织，优化性能和同步效率。

**设计原则**:
- **分层存储**: 按更新频率分离高频/中频/低频数据
- **混合模式**: 核心数据直接存储，派生数据按需计算
- **事件驱动**: SAN/状态变化保留历史记录
- **可见性控制**: 支持公开/KP-only/私有的多层次可见性

---

## 整体架构

### 三层状态结构

```
┌─────────────────────────────────────────────┐
│          高频状态 (DynamicState)             │
│  • 每次操作同步                              │
│  • 当前场景、最近检定、Leads                 │
└─────────────────────────────────────────────┘
                    ↕ (每回合同步)
┌─────────────────────────────────────────────┐
│          中频状态 (SessionState)              │
│  • 角色数据、战斗、追逐                       │
│  • 标记和变量                                │
└─────────────────────────────────────────────┘
                    ↕ (会话级同步)
┌─────────────────────────────────────────────┐
│          低频状态 (StaticState)               │
│  • 模组配置、全局设置                         │
└─────────────────────────────────────────────┘
```

### 高频状态 (DynamicState)

每次玩家操作后立即同步的数据：

```typescript
interface DynamicState {
  // 当前场景
  current_scene: string;
  current_speaker?: string;  // 聚光灯系统

  // Leads (可选行动)
  leads: {
    available: Lead[];
    completed: string[];
    refresh_count: number;
  };

  // 最近检定结果 (用于推骰/花幸运)
  last_roll: {
    skill: string;
    result: number;
    difficulty: string;
    can_push: boolean;
    timestamp: number;
  } | null;

  // 临时标志
  temp_flags: Record<string, any>;
}
```

**同步时机**: 每次命令执行后
**同步范围**: 所有参与者

---

## 角色状态结构

### 核心属性 (固定存储)

```typescript
interface CharacterCore {
  character_id: string;
  name: string;
  player_id?: string;  // NPC 为空
  type: "player" | "npc";

  // 核心属性 (固定存储)
  core: {
    STR: number;  // 力量
    DEX: number;  // 敏捷
    INT: number;  // 智力
    EDU: number;  // 教育
    APP: number;  // 外貌
    POW: number;  // 意志
    SIZ: number;  // 体型
    CON: number;  // 体质
  };

  // 派生属性
  derived: {
    HP: number;      // 当前生命值
    HP_max: number;  // 最大生命值 = (CON + SIZ) / 10
    MP: number;      // 当前魔法值
    MP_max: number;  // 最大魔法值 = POW / 5
    SAN: number;     // 当前理智值
    SAN_max: number; // 最大理智值
    Luck: number;    // 当前幸运值
    Luck_max: number; // 最大幸运值 = POW * 5
    Move: number;    // 移动速率
    Build: number;   // 体型/体质
    BonusDamage: number;  // 额外伤害
  };

  // 当前状态
  status: {
    alive: boolean;
    conscious: boolean;
    dying: boolean;
    insane: boolean;
    conditions: Condition[];
  };
}
```

### 玩家角色完整状态

```typescript
interface PlayerCharacterState extends CharacterCore {
  player_id: string;

  // 技能
  skills: {
    // 常用技能 (直接存储)
    common: {
      library_use: number;
      spot_hidden: number;
      listen: number;
      psychology: number;
      persuasion: number;
      intimidate: number;
      dodge: number;
      brawling: number;
      firearms_handgun: number;
    };

    // 其他技能 (按需加载)
    others?: {
      [skill_name: string]: number;
    };
  };

  // 背包
  inventory: {
    items: string[];
    encumbrance: number;
    max_encumbrance: number;
  };

  // 线索
  clues: {
    discovered: string[];
    private: string[];
  };

  // 个人变量
  variables: {
    visited_scenes: string[];
    talked_to_npcs: string[];
    custom: Record<string, any>;
  };

  // 疯狂症状
  madness: {
    phobias: string[];
    manias: string[];
  };
}
```

### NPC 状态 (简化)

```typescript
interface NPCState {
  npc_id: string;
  ref: string;  // 引用场景包定义

  name: string;
  description?: string;

  // 战斗数据 (仅在战斗时)
  combat: {
    hp: number;
    hp_max: number;
    damage_bonus?: number;
    build?: number;
    dodge: number;
  } | null;

  // 关系
  attitude: "hostile" | "neutral" | "friendly";

  // 状态
  alive: boolean;
  visible_to: "all" | "kp" | "player:*";
}
```

### 状态条件

```typescript
interface Condition {
  type:
    | "poisoned"     // 中毒
    | "bleeding"     // 出血
    | "stunned"      // 眩晕
    | "restrained"   // 受限
    | "blinded"      // 目盲
    | "deafened"     // 失聪
    | "prone";       // 倒地

  severity: "mild" | "moderate" | "severe";
  duration?: number;  // 剩余回合数
  effects?: string[];
}
```

---

## 战斗状态结构

```typescript
interface CombatState {
  active: boolean;
  round: number;
  current_actor: string;

  // 玩家队伍
  players: {
    [character_id: string]: CombatCharacter
  };

  // 敌对 NPC
  enemies: {
    [npc_id: string]: CombatCharacter
  };

  // 先攻顺序
  initiative_order: string[];
  current_index: number;

  // 战斗配置
  config: {
    surprise_round: boolean;
    surprise_participants?: string[];
  };
}

interface CombatCharacter {
  id: string;
  name: string;
  team: "players" | "enemies";

  initiative: number;
  hp: number;
  hp_max: number;

  status: CombatStatus;
  conditions: CombatCondition[];

  // 本回合状态
  current_round: {
    acted: boolean;
    dodged: boolean;
    actions_used: number;
  };
}

type CombatStatus =
  | "active"       // 正常行动
  | "dying"        // 濒死
  | "unconscious"  // 昏迷
  | "dead"         // 死亡
  | "fled";        // 逃跑

interface CombatCondition {
  type:
    | "stunned"     // 眩晕
    | "restrained"  // 受限
    | "grappled"    // 被擒抱
    | "prone"       // 倒地
    | "blinded"     // 目盲
    | "deafened";   // 失聪

  duration?: number;
  effects?: string[];
}
```

---

## 追逐状态结构

```typescript
interface ChaseState {
  active: boolean;
  round: number;

  // 位置网格
  grid: {
    positions: ChasePosition[];
    obstacles: ChaseObstacle[];
  };

  // 参与者
  runners: string[];    // 逃跑者
  pursuers: string[];   // 追逐者

  // 相对位置
  relative: {
    distance: number;       // 距离等级 (0-4)
    can_catch_up: boolean;
    leader: string;
  };

  // 压力值
  pressure: {
    current: number;
    threshold: number;
    effects: string[];
  };

  // 配置
  config: {
    type: "foot" | "vehicle" | "mixed";
    base_speed: number;
    escape_distance: number;
  };
}

interface ChasePosition {
  cell: number;
  character: string;
  type: "runner" | "pursuer";
  movement_mod: number;
}

interface ChaseObstacle {
  id: string;
  cell: number;
  type:
    | "crowd"
    | "traffic"
    | "fence"
    | "locked_door"
    | "rough_terrain";

  active: boolean;
  difficulty: "easy" | "regular" | "hard" | "extreme";

  overcome_by: {
    skill?: string;
    skill_difficulty?: string;
    alternative?: string;
  };

  effects: {
    movement_penalty?: number;
    speed_modifier?: number;
    requires_action?: boolean;
  };
}
```

---

## SAN 和疯狂状态结构

```typescript
interface SanityState {
  current: number;
  max: number;

  // 损失历史
  history: SanityLossEvent[];

  // 总损失
  total_lost: number;

  // 疯狂状态
  madness: MadnessState;
}

interface SanityLossEvent {
  event_id: string;
  timestamp: string;

  amount: number;
  trigger: string;
  source?: string;

  // 检定结果
  check_result: {
    roll: number;
    difficulty: string;
    passed: boolean;
  } | null;

  // 后果
  consequences: {
    temporary_madness?: boolean;
    indefinite_madness?: boolean;
    phobia_acquired?: string;
    mania_acquired?: string;
  };
}

interface MadnessState {
  current_state: "sane" | "temporary" | "indefinite";

  // 临时疯狂
  temporary: {
    active: boolean;
    type: TemporaryMadnessType | null;
    rounds_left: number;
    symptoms: MadnessSymptom[];
    triggered_by?: string;
    started_at?: string;
  };

  // 不明疯狂
  indefinite: {
    active: boolean;
    type: IndefiniteMadnessType | null;
    symptoms: MadnessSymptom[];
    cure_condition?: string;
    treated: boolean;
  };

  // 个人特质
  traits: {
    phobias: Phobia[];
    manias: Mania[];
  };
}

type TemporaryMadnessType =
  | "fear"          // 恐惧
  | "panic"         // 恐慌
  | "rage"          // 狂怒
  | "berserk"       // 暴走
  | "faint";        // 昏厥

type IndefiniteMadnessType =
  | "schizophrenia"    // 精神分裂
  | "paranoia"         // 偏执狂
  | "hallucinations"   // 幻觉
  | "amnesia"          // 失忆
  | "multiple_personality";  // 人格分裂

interface MadnessSymptom {
  type: string;
  description: string;
  effects: MadnessEffect[];
}

interface MadnessEffect {
  category: "penalty" | "restriction" | "behavior" | "trigger";

  penalty?: {
    target: string;
    value: number;
    when?: string;
  };

  restriction?: {
    action: string;
    override?: string;
  };

  behavior?: {
    pattern: string;
    frequency: "always" | "sometimes" | "rarely";
  };

  trigger?: {
    stimulus: string;
    response: string;
    check?: string;
  };
}

interface Phobia {
  id: string;
  name: string;
  trigger: string;
  severity: "mild" | "moderate" | "severe";
  effects: MadnessEffect[];
  overcome?: {
    method: string;
    difficulty: string;
  };
}

interface Mania {
  id: string;
  name: string;
  trigger?: string;
  compulsion: string;
  severity: "mild" | "moderate" | "severe";
  effects: MadnessEffect[];
}
```

---

## Leads 机制结构

```typescript
interface LeadsState {
  // 可用 Leads
  available: Lead[];

  // 已完成
  completed: string[];

  // 刷新计数
  refresh_count: number;

  // 自动刷新
  auto_refresh: {
    enabled: boolean;
    threshold: number;
    trigger: "on_scene_change" | "on_clue_found" | "on_action_completed";
  };
}

interface Lead {
  id: string;
  title: string;
  description: string;

  // 优先级
  priority: "critical" | "high" | "medium" | "low";

  // 类型
  type:
    | "investigate"
    | "interact"
    | "travel"
    | "combat"
    | "rest"
    | "custom";

  // 执行方式
  execution: {
    method: "command" | "choice" | "automatic";

    command?: string;
    choices?: LeadChoice[];
    automatic?: {
      target_scene: string;
      condition?: string;
    };
  };

  // 可见性
  visibility: {
    show_to: "all" | "kp" | "specific_player";
    player_ids?: string[];
  };

  // 状态
  status: "available" | "completed" | "failed" | "expired";

  // 过期条件
  expires_on?: {
    event: string;
    target_id?: string;
  };

  // 效果
  on_complete?: {
    rewards?: LeadReward[];
    consequences?: string[];
    narrative?: string;
  };

  on_fail?: {
    consequences?: string[];
    narrative?: string;
  };
}

interface LeadChoice {
  id: string;
  label: string;
  description?: string;
  target?: string;
  condition?: string;
  consequences?: string[];
  requires_check?: {
    skill: string;
    difficulty: string;
  };
}

interface LeadReward {
  type: "clue" | "item" | "healing" | "information" | "access";
  id?: string;
  description: string;
}
```

---

## 可见性控制

```typescript
interface VisibilityControl {
  // 公开 (所有人可见)
  public: {
    visible_to: "all";
  };

  // 仅 KP 可见
  kp_only: {
    visible_to: "kp";
  };

  // 特定玩家
  private: {
    visible_to: "player:*";
    player_ids: string[];
  };

  // 条件可见
  conditional: {
    visible_to: "conditional";
    condition: string;  // 表达式
  };
}
```

---

## 状态持久化

### 保存点

```typescript
interface SavePoint {
  save_id: string;
  session_id: string;
  timestamp: string;

  // 快照数据
  snapshot: {
    dynamic_state: DynamicState;
    session_state: SessionState;
  };

  // 元数据
  metadata: {
    scene: string;
    round?: number;
    players: string[];
    notes?: string;
  };
}
```

### 状态版本控制

```typescript
interface StateVersion {
  version: number;
  timestamp: string;
  changes: StateChange[];
  rollback_data: any;
}

interface StateChange {
  path: string;  // "characters.players.char_001.hp"
  old_value: any;
  new_value: any;
  actor: string;  // 谁触发的变化
  reason: string;
}
```

---

## 相关文档

- [M0-027 定义 SessionState 结构](../tasks/tasks-detailed/M0-027-session-state.md)
- [M0-028 定义 CharacterState 角色状态](../tasks/tasks-detailed/M0-028-character-state.md)
- [M0-031 定义 CombatState 战斗状态](../tasks/tasks/tasks-detailed/M0-031-combat-state.md)
- [M0-032 定义 ChaseState 追逐状态](../tasks/tasks/tasks-detailed/M0-032-chase-state.md)
- [M0-033 定义 SAN/疯狂状态](../tasks/tasks/tasks-detailed/M0-033-san-state.md)
- [M0-034 定义 Leads 机制结构](../tasks/tasks/tasks-detailed/M0-034-leads-mechanism.md)
- [命令集规范](./commands.md)
- [场景包格式规范](./scenario-schema.md)
