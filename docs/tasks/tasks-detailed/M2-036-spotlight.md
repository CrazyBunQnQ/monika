# M2-036: 设计 SpotlightState 数据结构

**任务ID**: M2-036
**标题**: 设计 SpotlightState 数据结构
**类型**: spec (规范设计)
**预估工时**: 2h
**依赖**: M0

---

## 任务描述

设计聚光灯 (Spotlight) 系统的数据结构，用于管理多人跑团时的发言权控制。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M2-036-01 | 分析聚光灯需求 | 理解 CoC 跑团发言流程 | 20min |
| M2-036-02 | 设计 SpotlightState | 核心状态结构 | 25min |
| M2-036-03 | 设计 QueueState | 队列状态结构 | 20min |
| M2-036-04 | 设计事件结构 | 聚光灯事件 | 15min |
| M2-036-05 | 编写 TypeScript 类型 | 类型定义 | 20min |
| M2-036-06 | 编写状态转换图 | 状态流转文档 | 15min |
| M2-036-07 | 编写使用示例 | 典型场景代码 | 10min |

---

## SpotlightState 结构

```typescript
interface SpotlightState {
  session_id: string;

  // 当前聚光灯持有者
  current: {
    user_id: string;           // 当前发言的用户
    character_id?: string;     // 控制的角色
    acquired_at: datetime;     // 获得时间
    timeout?: number;          // 超时时间(秒)
    type: 'kp' | 'player' | 'free';  // 聚光灯类型
  } | null;

  // 聚光灯历史
  history: {
    user_id: string;
    character_id?: string;
    from: datetime;
    to: datetime;
    duration_seconds: number;
  }[];

  // 聚光灯设置
  settings: {
    auto_timeout: boolean;     // 是否自动超时
    timeout_seconds: number;   // 默认超时时间
    kp_can_interrupt: boolean; // KP 是否可中断
    queue_enabled: boolean;    // 是否启用队列
  };

  // 更新时间
  updated_at: datetime;
}
```

---

## QueueState 结构

```typescript
interface QueueState {
  session_id: string;

  // 等待队列
  queue: {
    id: string;                // 队列项 ID
    user_id: string;
    character_id?: string;
    joined_at: datetime;
    priority: number;          // 优先级 (0-100)
    type: 'normal' | 'cut-in'; // 普通/插队
    notes?: string;            // 备注
  }[];

  // 队列设置
  settings: {
    max_size: number;          // 最大队列长度
    allow_cut_in: boolean;     // 是否允许插队
    auto_advance: boolean;     // 是否自动推进
  };

  // 统计
  stats: {
    total_served: number;      // 已服务总数
    average_wait_seconds: number;  // 平均等待时间
  };

  updated_at: datetime;
}
```

---

## 聚光灯事件

```typescript
interface SpotlightEvent {
  event_id: string;
  session_id: string;
  timestamp: datetime;
  type: SpotlightEventType;

  // 事件数据
  data: {
    // acquire - 获取聚光灯
    acquire?: {
      user_id: string;
      character_id?: string;
      from_queue?: boolean;     // 是否从队列获取
    };

    // release - 释放聚光灯
    release?: {
      user_id: string;
      reason: 'manual' | 'timeout' | 'kp_takeover';
    };

    // transfer - 转移聚光灯
    transfer?: {
      from_user: string;
      to_user: string;
      reason?: string;
    };

    // queue_join - 加入队列
    queue_join?: {
      user_id: string;
      position: number;
    };

    // queue_leave - 离开队列
    queue_leave?: {
      user_id: string;
    };

    // queue_advance - 队列推进
    queue_advance?: {
      user_id: string;
      from_position: number;
    };
  };

  // 操作者
  actor: {
    user_id: string;
    role: 'kp' | 'player';
  };
}

type SpotlightEventType =
  | 'acquire'
  | 'release'
  | 'transfer'
  | 'queue_join'
  | 'queue_leave'
  | 'queue_advance';
```

---

## 聚光灯规则

```typescript
// 聚光灯获取规则
interface SpotlightRules {
  // KP 获取规则
  kp: {
    can_take_anytime: true;      // KP 随时可获取
    can_interrupt: true;         // KP 可中断当前发言
    timeout_exempt: true;        // KP 不受超时限制
  };

  // 玩家获取规则
  player: {
    requires_queue: true;        // 需要在队列中
    wait_for_turn: true;         // 需要等待轮次
    can_request: true;           // 可以请求发言
    timeout_enabled: true;       // 启用超时
  };

  // 自由发言模式
  free: {
    enabled: false;              // 是否启用 (由 KP 控制)
    no_queue: true;              // 无需队列
    concurrent_limit: 0;         // 并发发言限制 (0=无限制)
  };
}
```

---

## 状态转换图

```
                    ┌─────────────┐
                    │   No Spot   │  无聚光灯
                    └──────┬──────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
    acquire(KP)      acquire(队列)      acquire(直接)
         │                 │                 │
         ▼                 ▼                 ▼
    ┌─────────┐      ┌─────────┐      ┌─────────┐
    │ KP Spot │      │Player Spot│    │Free Spot│
    └────┬────┘      └────┬────┘      └────┬────┘
         │                 │                 │
         │                 │                 │
         └─────────────────┴─────────────────┘
                           │
                    release/timeout
                           │
                           ▼
                    ┌─────────────┐
                    │   No Spot   │
                    └─────────────┘
```

---

## 队列操作示例

```typescript
// 加入队列
POST /game/spotlight/queue
{
  "user_id": "user123",
  "character_id": "char456",
  "priority": 50  // 可选，默认 50
}

// 离开队列
DELETE /game/spotlight/queue
{
  "user_id": "user123"
}

// 查看队列
GET /game/spotlight/queue
{
  "queue": [
    {
      "id": "q1",
      "user_id": "user123",
      "character_id": "char456",
      "position": 1,
      "joined_at": "2026-02-06T10:00:00Z"
    }
  ]
}

// KP 转移聚光灯
PUT /game/spotlight/transfer
{
  "from_user": "user123",
  "to_user": "user456",
  "reason": "Turn complete"
}

// 请求插队
POST /game/spotlight/queue/cut-in
{
  "user_id": "user789",
  "reason": "Important action"
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `docs/specs/spotlight.md` | 创建 | 聚光灯规范文档 |
| `app/core/types/spotlight.ts` | 创建 | 类型定义 |
| `app/db/models/spotlight.py` | 创建 | 数据模型 |

---

## 验收标准

- [ ] SpotlightState 结构完整
- [ ] QueueState 结构清晰
- [ ] 事件类型定义完整
- [ ] 状态转换图正确
- [ ] TypeScript 类型无错误

---

## 参考文档

- M0: 规范冻结
- M2-041: 队列状态管理

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
