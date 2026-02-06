# M0-035: 定义 Event 基础结构

**任务ID**: M0-035
**标题**: 定义 Event 基础结构
**类型**: spec (规范设计)
**预估工时**: 2h
**依赖**: 无

---

## 任务描述

定义游戏事件 (Event) 的基础数据结构，这是事件日志系统的核心，记录游戏中发生的所有事件。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M0-035-01 | 分析事件需求 | 确定需要记录的事件类型 | 20min |
| M0-035-02 | 设计 GameEvent 结构 | 主事件结构 | 30min |
| M0-035-03 | 设计 EventType 枚举 | 事件类型定义 | 20min |
| M0-035-04 | 设计 StateChange 结构 | 状态变更记录 | 25min |
| M0-035-05 | 设计 Visibility 结构 | 可见性控制 | 15min |
| M0-035-06 | 编写 TypeScript 类型 | 类型定义 | 20min |
| M0-035-07 | 编写事件示例 | 各类型事件示例 | 15min |

---

## GameEvent 结构

```typescript
interface GameEvent {
  // === 标识 ===
  event_id: string;
  session_id: string;
  timestamp: datetime;
  sequence: number;           // 事件序号 (递增)

  // === 参与者 ===
  actor: {
    user_id: string | null;   // 操作用户 ID
    role: 'kp' | 'player' | 'system';
    character_id?: string;    // 控制的角色
  };

  // === 内容 ===
  event_type: EventType;
  raw_message: string;        // 原始消息/命令

  // 解析后的动作
  parsed_action?: {
    command?: string;         // 命令名称
    target?: string;          // 目标
    params?: Record<string, any>;  // 参数
  };

  // === 结果 ===
  result?: {
    success: boolean;
    data?: any;
    error?: string;
  };

  // === 状态变更 ===
  state_changes?: StateChange[];

  // === 叙事内容 ===
  narration?: {
    text: string;             // 叙事文本
    type: 'narration' | 'dialogue' | 'description';
    speaker?: string;         // 说话者 (如果是对话)
  };

  // === 可见性 ===
  visibility: Visibility;

  // === 元数据 ===
  metadata: {
    client_timestamp?: number; // 客户端时间戳
    source: 'web' | 'api' | 'system';
    related_events?: string[]; // 关联事件 ID
  };
}
```

---

## EventType 枚举

```typescript
type EventType =
  // === 消息类 ===
  | 'message'                // 普通消息
  | 'dialogue'               // 对话
  | 'narration'              // 叙事

  // === 动作类 ===
  | 'action'                 // 一般行动
  | 'roll'                   // 掷骰
  | 'roll_pushed'            // 推骰
  | 'luck_spent'             // 花幸运
  | 'check'                  // 检定

  // === 战斗类 ===
  | 'combat_start'           // 战斗开始
  | 'combat_action'          // 战斗动作
  | 'combat_end'             // 战斗结束
  | 'damage'                 // 伤害
  | 'heal'                  // 治疗

  // === 追逐类 ===
  | 'chase_start'            // 追逐开始
  | 'chase_action'           // 追逐动作
  | 'chase_end'              // 追逐结束

  // === SAN 类 ===
  | 'san_check'              // SAN 检定
  | 'san_loss'               // SAN 损失
  | 'madness_start'          // 疯狂开始
  | 'madness_end'            // 疯狂结束

  // === 会话类 ===
  | 'scene_change'           // 场景转换
  | 'checkpoint'             // 检查点
  | 'pause'                 // 暂停
  | 'resume'                // 恢复

  // === 系统类 ===
  | 'system'                 // 系统事件
  | 'error'                 // 错误事件;
```

---

## StateChange 结构

```typescript
interface StateChange {
  // 变更目标
  target: {
    type: 'character' | 'session' | 'scene';
    id: string;
  };

  // 变更内容
  changes: {
    path: string;            // 状态路径，如 "hp", "san", "skills.library_use"
    old_value: any;
    new_value: any;
    delta?: number;          // 数值变化量
  }[];

  // 变更原因
  reason: {
    event_id: string;        // 触发事件
    description: string;
  };

  // 时间戳
  timestamp: datetime;
}
```

---

## Visibility 结构

```typescript
type Visibility =
  | 'public'                 // 公开 - 所有人可见
  | 'kp'                     // 仅 KP 可见
  | 'party'                  // 所有玩家可见
  | 'self'                   // 仅自己可见
  | 'player:<user_id>'       // 特定玩家可见
  | 'custom:<rule>';         // 自定义规则

interface VisibilityConfig {
  type: Visibility;
  users?: string[];         // 可见用户列表
  exclude?: string[];       // 排除用户列表
  condition?: string;       // 可见条件 (表达式)
}
```

---

## 事件示例

```typescript
// 掷骰事件示例
{
  event_id: "evt_123",
  session_id: "sess_456",
  timestamp: "2026-02-06T10:00:00Z",
  sequence: 42,
  actor: {
    user_id: "user_789",
    role: "player",
    character_id: "char_101"
  },
  event_type: "roll",
  raw_message: "/roll 侦查",
  parsed_action: {
    command: "roll",
    target: "侦查",
    params: { difficulty: "regular" }
  },
  result: {
    success: true,
    data: {
      skill: "侦查",
      skill_value: 55,
      roll: 38,
      success_level: "regular",
      description: "成功"
    }
  },
  state_changes: [
    {
      target: { type: "character", id: "char_101" },
      changes: [
        {
          path: "last_roll",
          old_value: null,
          new_value: { skill: "侦查", result: 38 }
        }
      ],
      reason: {
        event_id: "evt_123",
        description: "技能检定"
      },
      timestamp: "2026-02-06T10:00:00Z"
    }
  ],
  visibility: "public",
  metadata: {
    client_timestamp: 1738780800000,
    source: "web"
  }
}

// SAN 检定事件示例
{
  event_id: "evt_124",
  session_id: "sess_456",
  timestamp: "2026-02-06T10:05:00Z",
  sequence: 43,
  actor: {
    user_id: "user_kp",
    role: "kp"
  },
  event_type: "san_check",
  raw_message: "/san check 发现尸体 0/1d6",
  parsed_action: {
    command: "san_check",
    params: {
      trigger: "发现尸体",
      loss: "0/1d6"
    }
  },
  result: {
    success: true,
    data: {
      roll: 65,
      san_value: 45,
      passed: true,
      loss: 0,
      new_san: 45
    }
  },
  state_changes: [],
  visibility: "kp",
  metadata: {
    source: "web"
  }
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `docs/specs/event-structure.md` | 创建 | 事件结构规范 |
| `app/core/types/event.ts` | 创建 | TypeScript 类型 |
| `app/db/models/event.py` | 创建 | 数据模型 |

---

## 验收标准

- [ ] GameEvent 结构完整
- [ ] EventType 定义清晰
- [ ] StateChange 可追溯
- [ ] Visibility 控制正确
- [ ] 示例数据有效

---

## 参考文档

- CoC 7e 规则书
- 事件驱动架构模式

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
