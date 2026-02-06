# 系统架构

**版本**: v1.0
**最后更新**: 2026-02-07

---

## 概述

CoC 跑团平台采用**前后端分离架构**，支持单人/多人实时跑团，具备完整的检定、战斗、追逐、理智系统。

---

## 整体架构

```
┌─────────────────────────────────────────────────────────┐
│                         客户端                          │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐      │
│  │ React App  │──│ shadcn/ui  │──│ 游戏组件   │      │
│  └────────────┘  └────────────┘  └────────────┘      │
│        │                                            │
│        │ WebSocket                                 │
└────────┼────────────────────────────────────────────┘
         │ HTTP + WebSocket
┌────────▼─────────────────────────────────────────────┐
│                      API 层                           │
│              FastAPI + Socket.io                       │
└────────┬────────────────────────────────────────────┘
         │
┌────────▼─────────────────────────────────────────────┐
│                      服务层                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐          │
│  │ 游戏服务  │──│ LLM 服务  │──│ 规则服务  │          │
│  └──────────┘  └──────────┘  └──────────┘          │
│        │                                            │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐          │
│  │ 状态管理  │  │ 事件服务  │  │ 通知服务  │          │
│  └──────────┘  └──────────┘  └──────────┘          │
└────────┬────────────────────────────────────────────┘
         │
┌────────▼─────────────────────────────────────────────┐
│                      数据层                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐          │
│  │PostgreSQL │──│   Redis   │  │  文件存储  │          │
│  └──────────┘  └──────────┘  └──────────┘          │
└─────────────────────────────────────────────────────┘
```

---

## 前端架构

### 技术栈

| 层级 | 技术 | 说明 |
|------|------|------|
| 框架 | React 18+ | UI 框架 |
| 构建工具 | Vite | 快速构建 |
| UI 库 | shadcn/ui | 组件库 |
| 样式 | TailwindCSS | 原子化 CSS |
| 类型检查 | TypeScript | 类型安全 |
| 状态管理 | Zustand | 轻量级状态管理 |
| 路由 | React Router v6 | 客户端路由 |
| WebSocket | Socket.io-client | 实时通信 |

### 目录结构

```
src/
├── components/           # 组件
│   ├── ui/                # shadcn/ui 基础组件
│   ├── game/              # 游戏专用组件
│   │   ├── message/      # 消息相关
│   │   ├── dice/         # 骰子相关
│   │   ├── status/       # 状态相关
│   │   └── combat/       # 战斗相关
│   ├── layout/            # 布局组件
│   └── shared/           # 共享组件
├── pages/                # 页面
│   ├── home/             # 首页
│   ├── game/             # 游戏页面
│   ├── character/        # 角色管理
│   └── campaign/         # 模组管理
├── hooks/                # 自定义 Hooks
│   ├── useGameState.ts   # 游戏状态
│   ├── useWebSocket.ts   # WebSocket 连接
│   └── useAuth.ts        # 认证
├── lib/                  # 工具函数
│   ├── utils.ts          # 通用工具
│   ├── api.ts            # API 客户端
│   └── ws.ts             # WebSocket 客户端
├── stores/               # 状态管理
│   ├── gameStore.ts      # 游戏状态
│   ├── authStore.ts      # 认证状态
│   └── uiStore.ts        # UI 状态
├── services/             # API 服务
│   ├── game.ts           # 游戏 API
│   ├── character.ts      # 角色 API
│   └── scenario.ts       # 场景包 API
├── types/                # TypeScript 类型
│   ├── models.ts         # 数据模型
│   ├── api.ts            # API 类型
│   └── game.ts           # 游戏类型
└── App.tsx               # 应用入口
```

### 状态管理

```typescript
// 游戏状态（Zustand）
interface GameStore {
  // 会话信息
  session: SessionState | null;

  // 角色状态
  characters: {
    players: Record<string, PlayerCharacter>;
    npcs: Record<string, NPC>;
  };

  // 战斗状态
  combat: CombatState | null;

  // 追逐状态
  chase: ChaseState | null;

  // Leads
  leads: Lead[];

  // 操作
  connectSession: (sessionId: string) => void;
  sendMessage: (message: string) => Promise<void>;
  rollDice: (skill: string) => Promise<void>;
  // ...
}
```

---

## 后端架构

### 技术栈

| 层级 | 技术 | 说明 |
|------|------|------|
| 框架 | FastAPI | Python Web 框架 |
| WebSocket | Socket.io | 实时通信 |
| ORM | SQLAlchemy | 数据库 ORM |
| 数据库 | PostgreSQL | 主数据库 |
| 缓存 | Redis | 缓存和会话 |
| LLM | Agno SDK | AI 集成 |

### 目录结构

```
app/
├── api/                  # REST API 路由
│   ├── dependencies.py
│   ├── auth.py           # 认证 API
│   ├── characters.py     # 角色 API
│   ├── game.py           # 游戏 API
│   ├── scenarios.py      # 场景包 API
│   └── websocket.py      # WebSocket 处理
├── core/                # 核心配置
│   ├── config.py         # 配置
│   ├── security.py       # 安全
│   └── database.py       # 数据库连接
├── models/              # 数据模型
│   ├── user.py
│   ├── character.py
│   ├── session.py
│   └── event.py
├── services/            # 业务逻辑
│   ├── game/             # 游戏服务
│   │   ├── dice.py        # 掷骰服务
│   │   ├── combat.py      # 战斗服务
│   │   ├── sanity.py      # 理智服务
│   │   └── state.py       # 状态管理
│   ├── character/        # 角色服务
│   ├── llm/              # LLM 服务
│   │   ├── openai.py      # OpenAI 集成
│   │   ├── prompts.py     # Prompt 模板
│   │   └── response.py   # 响应解析
│   └── event/            # 事件服务
├── websocket/           # WebSocket
│   ├── connection.py     # 连接管理
│   ├── rooms.py          # 房间管理
│   └── events.py         # 事件处理
└── main.py              # 应用入口
```

---

## 数据流

### 消息处理流程

```
用户输入 "/roll library_use"
   ↓
前端解析命令
   ↓
WebSocket 发送到后端
   ↓
后端接收消息
   ↓
意图识别 → 识别为 "检定" 命令
   ↓
执行服务层 → DiceService.roll()
   ├─ 计算 d100 随机数
   ├─ 应用奖励/惩罚骰
   ├─ 计算成功等级
   └─ 更新状态
   ↓
事件服务 → 记录事件
   ├─ 保存到 events 表
   └─ 更新状态快照
   ↓
LLM 服务 → 生成叙事
   ├─ 构建 Prompt
   ├─ 调用 LLM API
   └─ 解析响应
   ↓
WebSocket 推送结果
   ↓
前端接收并渲染
```

### 状态同步流程

```
状态变更事件
   ↓
后端检测变更
   ↓
更新状态快照
   ↓
计算差异（delta）
   ↓
广播给所有连接的客户端
   ├─ public 状态 → 所有人
   ├─ kp 状态 → 仅 KP
   └─ private 状态 → 特定玩家
   ↓
前端接收更新
   ├─ 应用到本地状态
   └─ UI 刷新
```

---

## 部署架构

### 生产环境

```
┌─────────────────────────────────────────────────────────┐
│                      负载均衡                          │
│                    (Nginx / AWS ALB)                   │
└────────────────┬────────────────────────────────────────┘
                 │
     ┌───────────┴──────────┬───────────┐
     │                      │           │
┌────▼─────┐      ┌───────────────┐    ┌──────────────┐
│ API 节点 │      │ API 节点      │    │ API 节点    │
└──────────┘      └───────────────┘    └──────────────┘
     │                      │           │
┌────▼────────┬─────┴────────┬───────┴─────┐
│  PostgreSQL    │    Redis      │   文件存储   │
└───────────────┘───────────────┴───────────────┘
```

### 扩展性设计

**水平扩展**:
- 无状态 API 服务，可任意扩展
- WebSocket 连接通过 Redis 共享状态
- 数据库连接池管理

**垂直扩展**:
- LLM 服务独立部署
- 事件服务异步处理
- 静态资源 CDN 加速

---

## 安全架构

### 认证流程

```
用户登录
   ↓
POST /auth/login
   ↓
验证凭据
   ↓
生成 JWT Token (有效期 24h)
   ↓
返回 Token + RefreshToken
   ↓
后续请求携带 Token
   ↓
验证 Token 有效性
   ↓
允许/拒绝访问
```

### 授权

| 角色 | 权限 |
|------|------|
| KP | 所有操作 |
| Player | 自己的角色操作、公开信息查看 |
| 匿名 | 只读模式（如果启用） |

### 数据加密

- 传输加密: HTTPS / WSS
- 密码加密: bcrypt
- 敏感数据: 数据库加密存储

---

## 监控和日志

### 日志级别

| 级别 | 用途 | 示例 |
|------|------|------|
| DEBUG | 开发调试 | 函数调用细节 |
| INFO | 一般信息 | 用户登录、游戏事件 |
| WARNING | 警告信息 | 检定异常、API 错误 |
| ERROR | 错误信息 | 系统异常、失败 |

### 监控指标

- **性能指标**
  - API 响应时间
  - WebSocket 延迟
  - 数据库查询时间
  - LLM API 调用时间

- **业务指标**
  - 活跃用户数
  - 游戏会话数
  - 命令执行次数
  - 错误率

---

## 参考文档

- [API 参考](./api-reference.md)
- [数据字典](./data-dictionary.md)
- [命令集规范](../../specs/commands.md)
- [状态字段结构](../../specs/state-structure.md)
