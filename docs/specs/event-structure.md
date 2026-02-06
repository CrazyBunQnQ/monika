# 事件日志结构规范

**版本**: v1.0
**最后更新**: 2026-02-07
**状态**: ✅ 设计完成

---

## 概述

本文档定义 CoC 跑团平台的事件日志结构规范。事件日志是系统复盘、追溯和断点恢复的核心数据，采用**完整事件级别**记录所有信息及上下文。

**设计原则**:
- **完整记录**: 记录用户输入、状态变化、上下文快照
- **层级分类**: 按功能模块分组，便于筛选和查询
- **可见性控制**: 细粒度控制事件可见范围
- **高效存储**: 混合模式存储核心变化和大对象引用

---

## 事件基础结构

```typescript
interface GameEvent {
  // === 基础信息 ===
  event_id: string;
  session_id: string;
  timestamp: string;
  sequence: number;

  // === 参与者信息 ===
  actor: {
    player_id: string | null;
    character_id: string | null;
    role: "KP" | "Player" | "System";
  };

  // === 事件类型 ===
  type: {
    category: EventCategory;
    type: string;
    sub_type?: string;
  };

  // === 输入内容 ===
  input: {
    raw_message: string;
    parsed_command?: ParsedCommand;
  };

  // === 执行结果 ===
  result: {
    success: boolean;
    error?: string;
    data?: any;
  };

  // === 叙事内容 ===
  narration: {
    text: string;
    style: "narrative" | "compact" | "detailed";
  };

  // === 状态变化 ===
  state_changes: StateChange[];

  // === 大对象引用 ===
  large_objects?: {
    before_ref?: string;
    after_ref?: string;
  };

  // === 可见性 ===
  visibility: EventVisibility;

  // === 元数据 ===
  metadata: {
    client_timestamp?: number;
    source: "web" | "api" | "system";
    tags?: string[];
  };
}
```

---

## 事件类型分类

### 分类层级

```
interaction  - 交互类
check        - 检定类
combat       - 战斗类
chase        - 追逐类
sanity       - 理智类
state        - 状态类
system       - 系统类
```

### 交互类事件

| 类型 | 子类型 | 说明 |
|------|--------|------|
| message | chat | 聊天消息 |
| message | description | 场景描述 |
| message | system | 系统消息 |
| scene_change | - | 场景变化 |

### 检定类事件

| 类型 | 说明 |
|------|------|
| roll | 技能检定 |
| roll_pushed | 推骰 |
| luck_spent | 花幸运 |

### 战斗类事件

| 类型 | 说明 |
|------|------|
| combat_start | 战斗开始 |
| combat_action | 战斗动作 |
| combat_end | 战斗结束 |
| damage | 伤害 |
| death | 死亡 |

### 理智类事件

| 类型 | 说明 |
|------|------|
| san_check | SAN 检定 |
| san_loss | SAN 损失 |
| madness_start | 疯狂开始 |
| madness_end | 疯狂结束 |

### 状态类事件

| 类型 | 说明 |
|------|------|
| condition_added | 添加状态 |
| condition_removed | 移除状态 |
| heal | 治疗 |

### 系统类事件

| 类型 | 说明 |
|------|------|
| checkpoint | 检查点 |
| session_start | 会话开始 |
| session_end | 会话结束 |

---

## 状态变化结构

```typescript
interface StateChange {
  path: string;
  type: "set" | "add" | "remove" | "increment" | "decrement";

  old_value?: any;
  new_value?: any;
  added?: any[];
  removed?: any[];
  delta?: number;

  metadata?: {
    reason?: string;
    source?: string;
  };
}
```

### 状态变化示例

```typescript
// HP 减少
{
  path: "characters.player_001.derived.HP",
  type: "decrement",
  old_value: 10,
  new_value: 8,
  delta: -2,
  metadata: { reason: "combat_damage", source: "npc_goblin" }
}

// 添加条件
{
  path: "characters.player_001.status.conditions",
  type: "add",
  added: [{ type: "poisoned", severity: "mild", duration: 5 }]
}

// 发现线索
{
  path: "characters.player_001.clues.discovered",
  type: "add",
  added: ["clue_old_book"]
}
```

---

## 可见性控制

```typescript
interface EventVisibility {
  base: VisibilityBase;
  overrides?: VisibilityOverride[];
  conditional?: VisibilityCondition[];
}

type VisibilityBase =
  | "public"      // 所有人可见
  | "party"       // 所有玩家可见
  | "kp"          // 仅KP可见
  | "private";    // 私密事件

interface VisibilityOverride {
  type: "exclude" | "include";
  target: string;
}

interface VisibilityCondition {
  expression: string;
  show_if_true: boolean;
}
```

### 可见性示例

```typescript
// 公开事件
{ base: "public" }

// KP-only
{ base: "kp" }

// 排除特定玩家
{
  base: "party",
  overrides: [
    { type: "exclude", target: "player:002" }
  ]
}

// 私密线索
{
  base: "private",
  overrides: [
    { type: "include", target: "player:001" },
    { type: "include", target: "kp" }
  ]
}

// 条件可见
{
  base: "private",
  conditional: [
    { expression: "clues.contains('clue_secret')", show_if_true: true }
  ]
}
```

---

## 事件存储结构

```typescript
interface EventStore {
  memory: {
    current_session: string;
    events: Map<string, GameEvent>;
    by_sequence: Map<number, string>;
    by_actor: Map<string, string[]>;
  };

  persistence: {
    sessions: Map<string, SessionEvents>;
  };
}

interface SessionEvents {
  session_id: string;
  start_time: string;
  end_time?: string;

  events: GameEvent[];

  indexes: {
    by_category: Map<string, string[]>;
    by_type: Map<string, string[]>;
    by_visibility: Map<string, string[]>;
    by_timestamp: Map<string, string[]>;
  };

  snapshots: {
    [checkpoint_id: string]: string;
  };
}
```

---

## 事件查询

```typescript
interface EventQuery {
  filters: {
    session_id?: string;
    event_ids?: string[];
    categories?: EventCategory[];
    types?: string[];
    actors?: string[];
    time_range?: { start: string; end: string; };
  };

  visibility: {
    requester: "kp" | "player";
    player_id?: string;
  };

  sort: {
    field: "timestamp" | "sequence";
    order: "asc" | "desc";
  };

  pagination: {
    offset: number;
    limit: number;
  };
}
```

### 查询示例

```typescript
// 查询某玩家所有检定
{
  filters: {
    actors: ["player_001"],
    categories: ["check"]
  },
  visibility: { requester: "player", player_id: "player_001" },
  sort: { field: "timestamp", order: "desc" },
  pagination: { offset: 0, limit: 20 }
}

// 查询战斗相关事件
{
  filters: {
    categories: ["combat", "state"],
    time_range: {
      start: "2026-02-07T10:00:00Z",
      end: "2026-02-07T11:00:00Z"
    }
  },
  visibility: { requester: "kp" }
}
```

---

## 检查点和恢复

```typescript
interface CheckpointManager {
  create(state: any, metadata: CheckpointMetadata): string;
  restore(checkpoint_id: string): RestoreResult;
  list(session_id: string): CheckpointInfo[];
}

interface CheckpointMetadata {
  session_id: string;
  scene: string;
  round?: number;
  description?: string;
  created_at: string;
}

interface RestoreResult {
  success: boolean;
  state: any;
  events_after: GameEvent[];
  can_resume: boolean;
}
```

### 检查点触发策略

```typescript
type CheckpointTrigger =
  | { type: "scene_change" }
  | { type: "combat_end" }
  | { type: "interval"; rounds: number }
  | { type: "manual" };
```

---

## 事件回放

```typescript
interface EventReplay {
  session_id: string;
  from_sequence: number;
  to_sequence?: number;

  config: {
    speed?: number;
    pause_at?: number[];
    show_hidden?: boolean;
  };

  replay(): AsyncGenerator<ReplayStep>;
}

interface ReplayStep {
  sequence: number;
  event: GameEvent;
  state_after: any;
  narration: string;
}
```

---

## 完整事件示例

```typescript
// 检定事件示例
{
  event_id: "evt_001",
  session_id: "session_123",
  timestamp: "2026-02-07T10:30:00Z",
  sequence: 42,

  actor: {
    player_id: "player_001",
    character_id: "char_001",
    role: "Player"
  },

  type: {
    category: "check",
    type: "roll",
    sub_type: "skill"
  },

  input: {
    raw_message: "/roll library_use",
    parsed_command: {
      command: "roll",
      skill: "library_use",
      modifiers: []
    }
  },

  result: {
    success: true,
    data: {
      skill: "library_use",
      value: 60,
      roll: 78,
      difficulty: "regular",
      success_level: "success"
    }
  },

  narration: {
    text: "你仔细查阅书架上的古籍...",
    style: "narrative"
  },

  state_changes: [
    {
      path: "characters.char_001.clues.discovered",
      type: "add",
      added: ["clue_old_book"]
    }
  ],

  visibility: {
    base: "public"
  },

  metadata: {
    source: "web",
    tags: ["check", "investigation"]
  }
}

// SAN 损失事件示例（KP-only）
{
  event_id: "evt_002",
  session_id: "session_123",
  timestamp: "2026-02-07T10:35:00Z",
  sequence: 45,

  actor: {
    player_id: null,
    character_id: null,
    role: "System"
  },

  type: {
    category: "sanity",
    type: "san_loss"
  },

  input: {
    raw_message: "[系统] 触发 SAN 检定"
  },

  result: {
    success: true,
    data: {
      amount: 10,
      current_san: 50,
      total_lost: 10
    }
  },

  narration: {
    text: "你感到一阵寒意穿过全身...",
    style: "narrative"
  },

  state_changes: [
    {
      path: "characters.char_001.derived.SAN",
      type: "decrement",
      old_value: 60,
      new_value: 50,
      delta: -10
    }
  ],

  visibility: {
    base: "kp"
  },

  metadata: {
    source: "system",
    tags: ["sanity", "loss"]
  }
}
```

---

## 相关文档

- [M0-035 定义 Event 基础结构](../tasks/tasks-detailed/M0-035-event-structure.md)
- [M0-036 定义 EventType 枚举](../tasks/tasks-detailed/M0-036-event-type.md)
- [M0-037 定义 Visibility 可见性](../tasks/tasks/tasks-detailed/M0-037-visibility.md)
- [M0-038 定义 StateChange 变更结构](../tasks/tasks/tasks-detailed/M0-038-state-change.md)
- [状态字段结构规范](./state-structure.md)
- [命令集规范](./commands.md)
