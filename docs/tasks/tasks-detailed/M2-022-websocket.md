# M2-022: 配置 Socket.io 服务

**任务ID**: M2-022
**标题**: 配置 Socket.io 服务
**类型**: backend (后端开发)
**预估工时**: 4h
**依赖**: M1-046

---

## 任务描述

配置 Socket.io 实时通信服务，实现多人在线的 WebSocket 连接管理。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M2-022-01 | 安装 Socket.io 依赖 | python-socketio | 15min |
| M2-022-02 | 配置 Socket.io 服务器 | 初始化配置 | 30min |
| M2-022-03 | 集成到 FastAPI | 与现有 API 共存 | 30min |
| M2-022-04 | 定义事件命名空间 | 区分不同类型事件 | 20min |
| M2-022-05 | 实现连接管理 | 连接/断开处理 | 30min |
| M2-022-06 | 实现房间管理 | 房间加入/离开 | 30min |
| M2-022-07 | 实现消息广播 | 单播/多播/广播 | 30min |
| M2-022-08 | 编写 WebSocket 测试 | 连接和消息测试 | 30min |
| M2-022-09 | 编写配置文档 | 部署和配置说明 | 15min |

---

## Socket.io 配置

```python
# app/websocket/server.py
import socketio
from fastapi import FastAPI
from typing import Dict

# 创建 Socket.io 服务器
sio = socketio.AsyncServer(
    async_mode='asgi',
    cors_allowed_origins='*',
    ping_timeout=60,
    ping_interval=25,
    engineio_logger=False,
    socketio_logger=False,
)

# 存储连接信息
connected_users: Dict[str, Dict] = {}  # sid -> user_info
rooms: Dict[str, set] = {}  # room_id -> set of sids

app = FastAPI()
socket_app = socketio.ASGIApp(sio, app)

# 连接事件
@sio.event
async def connect(sid, environ):
    """客户端连接"""
    print(f"Client connected: {sid}")
    await sio.emit('connected', {'sid': sid}, to=sid)

# 断开事件
@sio.event
async def disconnect(sid):
    """客户端断开"""
    print(f"Client disconnected: {sid}")

    # 从所有房间移除
    for room_id, members in rooms.items():
        if sid in members:
            members.remove(sid)
            await sio.emit('user_left', {
                'sid': sid,
                'room': room_id
            }, room=room_id)

    # 清理用户信息
    if sid in connected_users:
        del connected_users[sid]

# 加入房间
@sio.event
async def join(sid, data):
    """加入房间"""
    room_id = data.get('room')
    user_id = data.get('user_id')

    if not room_id:
        return {'error': 'Room ID required'}

    # 加入房间
    sio.enter_room(sid, room_id)

    # 记录房间成员
    if room_id not in rooms:
        rooms[room_id] = set()
    rooms[room_id].add(sid)

    # 记录用户信息
    connected_users[sid] = {
        'user_id': user_id,
        'room': room_id,
    }

    # 通知房间其他成员
    await sio.emit('user_joined', {
        'user_id': user_id,
        'sid': sid
    }, room=room_id, skip_sid=sid)

    # 返回房间当前成员
    members = [
        connected_users[m]['user_id']
        for m in rooms[room_id]
        if m in connected_users
    ]
    return {
        'room': room_id,
        'members': members
    }

# 离开房间
@sio.event
async def leave(sid, data):
    """离开房间"""
    room_id = data.get('room')

    if room_id in rooms and sid in rooms[room_id]:
        rooms[room_id].remove(sid)
        sio.leave_room(sid, room_id)

        await sio.emit('user_left', {
            'sid': sid
        }, room=room_id)

    return {'success': True}

# 发送消息
@sio.event
async def send_message(sid, data):
    """发送消息到房间"""
    room_id = data.get('room')
    message = data.get('message')

    if not room_id or not message:
        return {'error': 'Room and message required'}

    # 广播到房间所有人 (除发送者)
    await sio.emit('message', {
        'sid': sid,
        'message': message,
        'timestamp': datetime.now().isoformat()
    }, room=room_id, skip_sid=sid)

    return {'success': True}
```

---

## 事件命名空间

```typescript
// 系统事件
namespace: /system
events:
  - connect       // 连接成功
  - disconnect    // 断开连接
  - error         // 错误信息

// 会话事件
namespace: /session
events:
  - join          // 加入会话
  - leave         // 离开会话
  - user_joined   // 用户加入通知
  - user_left     // 用户离开通知
  - state_update  // 状态更新

// 聊天事件
namespace: /chat
events:
  - message       // 聊天消息
  - typing        // 正在输入
  - read          // 消息已读

// 游戏事件
namespace: /game
events:
  - roll          // 掷骰
  - combat        // 战斗动作
  - chase         // 追逐动作
  - spotlight     // 聚光灯变更
```

---

## FastAPI 集成

```python
# app/main.py
from fastapi import FastAPI
from app.websocket.server import sio, socket_app

# 创建主应用
app = FastAPI()

# 挂载 Socket.io
app.mount("/socket.io", socket_app)

# 注册路由
from app.api import auth, characters, game
app.include_router(auth.router, prefix="/api")
app.include_router(characters.router, prefix="/api")
app.include_router(game.router, prefix="/api")

# 普通路由和 WebSocket 并存
@app.get("/")
async def root():
    return {"message": "CoC TRPG Platform"}
```

---

## 客户端连接示例

```typescript
// frontend/src/lib/socket.ts
import { io, Socket } from 'socket.io-client';

class SocketService {
  private socket: Socket | null = null;

  connect(token: string) {
    this.socket = io({
      path: '/socket.io',
      transports: ['websocket'],
      auth: { token },
    });

    this.socket.on('connect', () => {
      console.log('Connected:', this.socket?.id);
    });

    this.socket.on('disconnect', () => {
      console.log('Disconnected');
    });

    this.socket.on('error', (error) => {
      console.error('Socket error:', error);
    });

    return this.socket;
  }

  joinRoom(roomId: string, userId: string) {
    this.socket?.emit('join', {
      room: roomId,
      user_id: userId,
    });
  }

  sendMessage(roomId: string, message: string) {
    this.socket?.emit('send_message', {
      room: roomId,
      message: message,
    });
  }

  onMessage(callback: (data: any) => void) {
    this.socket?.on('message', callback);
  }

  disconnect() {
    this.socket?.disconnect();
  }
}

export const socketService = new SocketService();
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/websocket/server.py` | 创建 | Socket.io 服务器 |
| `app/websocket/events/` | 创建 | 事件处理器 |
| `frontend/src/lib/socket.ts` | 创建 | 客户端连接服务 |
| `requirements.txt` | 更新 | 添加 python-socketio |
| `package.json` | 更新 | 添加 socket.io-client |

---

## 验收标准

- [ ] Socket.io 服务正常启动
- [ ] 客户端可以连接
- [ ] 房间加入/离开正常
- [ ] 消息广播正确
- [ ] 断线重连工作
- [ ] 与 FastAPI 共存无冲突

---

## 参考文档

- Socket.io Python 文档
- M1-046: FastAPI 项目骨架
- M2-023: 连接认证中间件

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
