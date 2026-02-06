# 数据字典

**版本**: v1.0
**最后更新**: 2026-02-07

---

## 概述

本文档定义 CoC 跑团平台中使用的所有数据字段和数据结构。

---

## 命名规范

### 基本规则

- 使用 **snake_case** 命名
- 布尔值以 **is_** 开头
- 时间戳以 **_at** 结尾
- 枚举值使用描述性名称

### 命名示例

```typescript
// ✅ 正确
character_id
is_alive
created_at
success_level

// ❌ 错误
characterID        // 驼峰命名
alive             // 不是布尔
createTime        // 驼峰命名
successlevel      // 可读性差
```

---

## 核心数据表

### users 表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| user_id | string | PK | 用户唯一标识 |
| username | string | UNIQUE | 用户名 |
| email | string | UNIQUE | 邮箱 |
| password_hash | string | NOT NULL | 密码哈希 |
| created_at | timestamptz | NOT NULL | 创建时间 |
| updated_at | timestamptzz | NOT NULL | 更新时间 |

### characters 表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| character_id | string | PK | 角色唯一标识 |
| player_id | string | FK | 玩家 ID（NPC 为 NULL） |
| name | string | NOT NULL | 角色名称 |
| type | enum | NOT NULL | 类型: player/npc |
| core_attributes | jsonb | NOT NULL | 核心属性 |
| derived_attributes | jsonb | NOT NULL | 派生属性 |
| skills | jsonb | NOT NULL | 技能列表 |
| inventory | jsonb | DEFAULT '{}' | 背包物品 |
| clues | jsonb | DEFAULT '{}' | 发现的线索 |
| status | jsonb | NOT NULL | 当前状态 |
| created_at | timestamptz | NOT NULL | 创建时间 |
| updated_at | timestamptz | NOT NULL | 更新时间 |

**core_attributes 结构**:
```json
{
  "STR": 50,
  "DEX": 55,
  "INT": 70,
  "EDU": 65,
  "APP": 40,
  "POW": 60,
  "SIZ": 50,
  "CON": 45
}
```

**derived_attributes 结构**:
```json
{
  "HP": 12,
  "HP_max": 12,
  "MP": 14,
  "MP_max": 14,
  "SAN": 60,
  "SAN_max": 99,
  "Luck": 50,
  "Luck_max": 50,
  "Move": 7,
  "Build": 0,
  "BonusDamage": 0
}
```

**skills 结构**:
```json
{
  "common": {
    "library_use": 60,
    "spot_hidden": 50,
    "listen": 40
  },
  "others": {
    "psychology": 30,
    "drive_auto": 25
  }
}
```

**status 结构**:
```json
{
  "alive": true,
  "conscious": true,
  "dying": false,
  "insane": false,
  "conditions": [
    {"type": "poisoned", "severity": "mild", "duration": 5}
  ]
}
```

### sessions 表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| session_id | string | PK | 会话唯一标识 |
| module_id | string | FK | 场景包 ID |
| kp_id | string | FK | KP 用户 ID |
| status | enum | NOT NULL | 状态: active/paused/ended |
| current_scene | string | NOT NULL | 当前场景 ID |
| started_at | timestamptz | NOT NULL | 开始时间 |
| updated_at | timestamptz | NOT NULL | 更新时间 |
| ended_at | timestamptz | | 结束时间 |

### events 表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| event_id | string | PK | 事件唯一标识 |
| session_id | string | FK | 会话 ID |
| sequence | integer | NOT NULL | 会话内序号 |
| timestamp | timestamptz | NOT NULL | 事件时间戳 |
| actor_player_id | string | FK | 行动玩家 ID（可能为 NULL） |
| actor_role | enum | NOT NULL | 角色: KP/Player/System |
| controlled_character_id | string | FK | 控制的角色 ID |
| event_type | jsonb | NOT NULL | 事件类型 {category, type, sub_type} |
| raw_message | text | NOT NULL | 原始消息 |
| parsed_action | jsonb | | 解析后的动作 |
| state_changes | jsonb | | 状态变化列表 |
| narration | text | | AI 生成的叙事 |
| visibility | enum | NOT NULL | 可见性: public/kp/private |
| metadata | jsonb | | 元数据 |

### scenarios 表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| scenario_id | string | PK | 场景包唯一标识 |
| title | string | NOT NULL | 场景包标题 |
| version | string | NOT NULL | 版本号 |
| author | string | NOT NULL | 作者 |
| description | text | | 描述 |
| file_path | string | NOT NULL | 文件存储路径 |
| schema_version | string | NOT NULL | Schema 版本 |
| tags | text[] | | 标签数组 |
| is_public | boolean | DEFAULT true | 是否公开 |
| created_at | timestamptz | NOT NULL | 创建时间 |
| updated_at | timestamptz | NOT NULL | 更新时间 |

---

## 枚举值定义

### 角色类型
```typescript
type CharacterType = "player" | "npc";
```

### 会话状态
```typescript
type SessionStatus = "active" | "paused" | "ended";
```

### 事件分类
```typescript
type EventCategory =
  | "interaction"   // 交互类
  | "check"         // 检定类
  | "combat"        // 战斗类
  | "chase"         // 追逐类
  | "sanity"        // 理智类
  | "state"         // 状态类
  | "system";       // 系统类
```

### 可见性级别
```typescript
type Visibility = "public" | "kp" | "private";
```

### 检定难度
```typescript
type Difficulty = "easy" | "regular" | "hard" | "extreme" | "critical";
```

### 成功等级
```typescript
type SuccessLevel =
  | "critical"      // 大成功
  | "extremeSuccess" // 极难成功
  | "hardSuccess"    // 困难成功
  | "regularSuccess" // 普通成功
  | "failure"        // 失败
  | "fumble";        // 大失败
```

---

## 数据类型

### JSON 字段类型

**core_attributes** (CharacterCore):
```typescript
interface CharacterCore {
  STR: number;  // 力量 0-100
  DEX: number;  // 敏捷 0-100
  INT: number;  // 智力 0-100
  EDU: number;  // 教育 0-100
  APP: number;  // 外貌 0-100
  POW: number;  // 意志 0-100
  SIZ: number;  // 体型 0-100
  CON: number;  // 体质 0-100
}
```

**state_changes** (StateChange):
```typescript
interface StateChange {
  path: string;           // JSON Path
  type: StateChangeType;
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

type StateChangeType =
  | "set"           // 设置值
  | "add"           // 添加到数组
  | "remove"        // 从数组移除
  | "increment"     // 增加
  | "decrement";    // 减少
```

---

## 数据验证规则

### 用户输入验证

```python
def validate_username(username: str) -> bool:
    """用户名验证"""
    return 3 <= len(username) <= 20
    return username.isalnum() or "_" in username
```

### 检定值验证

```python
def validate_roll(roll: int) -> bool:
    """检定值验证"""
    return 1 <= roll <= 100
```

### 属性值验证

```python
def validate_attribute(value: int, attr: str) -> bool:
    """属性值验证"""
    valid_ranges = {
        "STR": (0, 100),
        "DEX": (0, 100),
        # ... 其他属性
    }
    min_val, max_val = valid_ranges[attr]
    return min_val <= value <= max_val
```

---

## 索引设计

### users 表索引
```sql
CREATE UNIQUE INDEX idx_users_username ON users(username);
CREATE UNIQUE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_created_at ON users(created_at);
```

### characters 表索引
```sql
CREATE INDEX idx_characters_player ON characters(player_id);
CREATE INDEX idx_characters_name ON characters(name);
CREATE INDEX idx_characters_type ON characters(type);
```

### events 表索引
```sql
CREATE INDEX idx_events_session ON events(session_id);
CREATE INDEX idx_events_timestamp ON events(timestamp);
CREATE INDEX idx_events_sequence ON events(session_id, sequence);
CREATE INDEX idx_events_visibility ON events(visibility);
```

---

## 数据字典版本

| 版本 | 日期 | 变更 |
|------|------|------|
| v1.0 | 2026-02-07 | 初始版本 |

---

## 参考文档

- [API 参考](./api-reference.md)
- [系统架构](./architecture.md)
- [状态字段结构](../../specs/state-structure.md)
- [事件日志结构](../../specs/event-structure.md)
