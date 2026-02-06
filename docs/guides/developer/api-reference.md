# API 参考文档

**版本**: v1.0
**Base URL**: `https://api.example.com/v1`
**最后更新**: 2026-02-07

---

## 认证

### 获取 Token

所有 API 请求需要在 Header 中携带 JWT Token：

```
Authorization: Bearer {token}
```

### Token 获取

#### POST /auth/register
**描述**: 用户注册

**请求体**:
```json
{
  "username": "player1",
  "email": "player1@example.com",
  "password": "password123"
}
```

**响应** (200 OK):
```json
{
  "user_id": "user_001",
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

#### POST /auth/login
**描述**: 用户登录

**请求体**:
```json
{
  "email": "player1@example.com",
  "password": "password123"
}
```

**响应** (200 OK):
```json
{
  "user_id": "user_001",
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

#### POST /auth/refresh
**描述**: 刷新 Token

**请求体**:
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**响应** (200 OK):
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

---

## 游戏 API

### POST /game/message
**描述**: 发送游戏消息/命令

**请求头**:
```
Authorization: Bearer {token}
Content-Type: application/json
```

**请求体**:
```json
{
  "message": "/roll library_use",
  "session_id": "session_001"
}
```

**响应** (200 OK):
```json
{
  "event_id": "evt_001",
  "timestamp": "2026-02-07T10:30:00Z",
  "sequence": 42,
  "narration": "你仔细查阅书架上的古籍...",
  "result": {
    "skill": "library_use",
    "skill_value": 60,
    "roll": 78,
    "difficulty": "regular",
    "success_level": "regularSuccess"
  },
  "state_changes": [
    {
      "path": "characters.player_001.clues.discovered",
      "type": "add",
      "added": ["clue_old_book"]
    }
  ],
  "visibility": "public"
}
```

### GET /game/state/{session_id}
**描述**: 获取游戏状态

**请求头**:
```
Authorization: Bearer {token}
```

**响应** (200 OK):
```json
{
  "session_id": "session_001",
  "current_scene": "scene_001",
  "characters": {
    "players": {...},
    "npcs": {...}
  },
  "combat": null,
  "leads": [...]
}
```

---

## 角色卡 API

### POST /characters
**描述**: 创建角色卡

**请求体**:
```json
{
  "name": "调查员 A",
  "age": 25,
  "occupation": "私家侦探",
  "attributes": {
    "STR": 50,
    "DEX": 55,
    "INT": 70,
    "EDU": 65,
    "APP": 40,
    "POW": 60,
    "SIZ": 50,
    "CON": 45
  },
  "skills": {
    "library_use": 60,
    "spot_hidden": 50
  }
}
```

**响应** (201 Created):
```json
{
  "character_id": "char_001",
  "player_id": "user_001",
  "created_at": "2026-02-07T10:30:00Z"
}
```

### GET /characters/{character_id}
**描述**: 获取角色卡详情

**响应** (200 OK):
```json
{
  "character_id": "char_001",
  "name": "调查员 A",
  "derived": {
    "HP": 12,
    "HP_max": 12,
    "SAN": 60,
    "SAN_max": 99
  },
  "skills": {...}
}
```

### PUT /characters/{character_id}
**描述**: 更新角色卡

**请求体**: 同 POST /characters

**响应** (200 OK): 更新后的角色数据

---

## 场景包 API

### POST /scenarios
**描述**: 上传场景包

**请求**:
- Content-Type: `multipart/form-data`
- 字段: `file` (场景包 ZIP 文件)

**响应** (201 Created):
```json
{
  "scenario_id": "scenario_001",
  "title": "午夜图书馆",
  "version": "1.0.0",
  "uploaded_at": "2026-02-07T10:30:00Z"
}
```

### GET /scenarios
**描述**: 获取场景包列表

**查询参数**:
- `page`: 页码
- `limit`: 每页数量
- `tags`: 标签筛选

**响应** (200 OK):
```json
{
  "scenarios": [...],
  "total": 10,
  "page": 1,
  "pages": 2
}
```

---

## 错误处理

### 错误响应格式

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request data",
    "details": {...}
  }
}
```

### 错误代码

| 代码 | HTTP 状态 | 说明 |
|------|-----------|------|
| VALIDATION_ERROR | 400 | 请求参数验证失败 |
| UNAUTHORIZED | 401 | 未认证或 Token 无效 |
| FORBIDDEN | 403 | 权限不足 |
| NOT_FOUND | 404 | 资源不存在 |
| INTERNAL_ERROR | 500 | 服务器内部错误 |

---

## WebSocket API

### 连接

**URL**: `wss://api.example.com/ws`

**认证**: 通过 URL 参数传递 token
```
wss://api.example.com/ws?token={jwt_token}
```

### 客户端发送事件

#### client:message
```json
{
  "type": "client:message",
  "data": {
    "message": "/roll library_use",
    "session_id": "session_001"
  }
}
```

### 服务器推送事件

#### server:message
```json
{
  "type": "server:message",
  "data": {
    "event_id": "evt_001",
    "narration": "...",
    "result": {...}
  }
}
```

#### server:state_update
```json
{
  "type": "server:state_update",
  "data": {
    "character_id": "char_001",
    "hp": 8,
    "san": 50
  }
}
```

#### server:combat_start
```json
{
  "type": "server:combat_start",
  "data": {
    "combat_id": "combat_001",
    "participants": [...]
  }
}
```

---

## 速率限制

| 端点 | 限制 |
|------|------|
| POST /game/message | 10 次/分钟 |
| WebSocket 消息 | 60 次/分钟 |
| 其他 API | 100 次/分钟 |

---

## 参考文档

- [数据字典](./data-dictionary.md)
- [系统架构](./architecture.md)
- [命令集规范](../../specs/commands.md)
- [状态字段结构](../../specs/state-structure.md)
