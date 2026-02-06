# M1-090: 战役管理系统

**任务ID**: M1-090
**标题**: 战役管理系统
**类型**: backend (后端开发)
**预估工时**: 6h
**依赖**: M1-001

---

## 任务描述

实现战役 (Session) 的生命周期管理，包括创建、加入、开始、暂停、结束等操作。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-090-01 | 设计 Session 数据模型 | 会话状态结构 | 20min |
| M1-090-02 | 实现 Session 创建服务 | 创建新会话 | 30min |
| M1-090-03 | 实现 Session 状态管理 | 状态转换 | 30min |
| M1-090-04 | 实现 Session API | CRUD 端点 | 45min |
| M1-090-05 | 实现角色加入会话 | 角色绑定 | 30min |
| M1-090-06 | 实现会话暂停/恢复 | 断点续跑基础 | 30min |
| M1-090-07 | 实现会话历史记录 | Session 历史查询 | 20min |
| M1-090-08 | 编写会话测试 | 单元测试 | 30min |
| M1-090-09 | 编写会话文档 | API 说明 | 15min |

---

## Session 数据模型

```typescript
enum SessionStatus {
  CREATED = 'created',     // 已创建
  WAITING = 'waiting',     // 等待玩家
  ACTIVE = 'active',       // 进行中
  PAUSED = 'paused',       // 已暂停
  ENDED = 'ended',         // 已结束
}

interface Session {
  id: string;

  // 基本信息
  name: string;
  description?: string;
  status: SessionStatus;

  // 关联
  kp_id: string;           // KP 用户 ID
  script_id?: string;      // 关联的脚本 ID
  campaign_id?: string;    // 战役 ID (用于系列 Session)

  // 玩家
  players: {
    user_id: string;
    character_id: string;
    joined_at: datetime;
  }[];

  // 时间
  created_at: datetime;
  started_at?: datetime;
  ended_at?: datetime;
  last_activity: datetime;

  // 状态快照
  state_snapshot?: {
    scene_id?: string;
    characters: Record<string, any>;
    game_state: Record<string, any>;
  };

  // 统计
  stats: {
    message_count: number;
    roll_count: number;
    duration_seconds?: number;
  };
}
```

---

## API 端点设计

### POST /sessions
创建新会话

```yaml
requestBody:
  content:
    application/json:
      schema:
        type: object
        properties:
          name: { type: string }
          description: { type: string }
          script_id: { type: string }

responses:
  201:
    description: 会话创建成功
```

### POST /sessions/:id/start
开始会话

```yaml
responses:
  200:
    description: 会话已开始
  400:
    description: 会话状态不允许开始
```

### POST /sessions/:id/pause
暂停会话

```yaml
responses:
  200:
    description: 会话已暂停
```

### POST /sessions/:id/resume
恢复会话

```yaml
responses:
  200:
    description: 会话已恢复
```

### POST /sessions/:id/end
结束会话

```yaml
responses:
  200:
    description: 会话已结束
```

### POST /sessions/:id/join
加入会话 (玩家)

```yaml
requestBody:
  content:
    application/json:
      schema:
        type: object
        required: [character_id]
        properties:
          character_id: { type: string }

responses:
  200:
    description: 成功加入
```

### GET /sessions
列出会话

```yaml
parameters:
  - name: status
    in: query
    schema:
      type: string
      enum: [created, waiting, active, paused, ended]
  - name: kp_id
    in: query
    schema:
      type: string

responses:
  200:
    description: 会话列表
```

---

## Session 状态转换

```
                    ┌──────────┐
                    │ CREATED  │
                    └────┬─────┘
                         │ start()
                         ▼
                   ┌──────────┐
                   │ WAITING  │◄─────┐
                   └────┬─────┘      │
                        │ add_player()
                        ▼             │ resume()
                   ┌──────────┐       │
                   │  ACTIVE  ├───────┘
                   └────┬─────┘
                        │ pause()
                        ▼
                   ┌──────────┐
                   │  PAUSED  │
                   └────┬─────┘
                        │ resume() / end()
                        ▼
                   ┌──────────┐
                   │  ENDED   │
                   └──────────┘
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/db/models/session.py` | 创建 | Session 数据模型 |
| `app/services/session.py` | 创建 | Session 服务 |
| `app/api/sessions.py` | 创建 | Session API |
| `tests/test_sessions.py` | 创建 | Session 测试 |

---

## 验收标准

- [ ] Session 可以创建
- [ ] 玩家可以加入
- [ ] 状态转换正确
- [ ] 暂停/恢复正常
- [ ] 历史记录完整
- [ ] 权限控制正确

---

## 参考文档

- M1-001: 数据库表结构设计
- M1-103: 事件记录服务

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
