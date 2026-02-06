# M2-002: 实现 WebSocket 事件系统

**任务ID**: M2-002
**标题**: 实现 WebSocket 事件系统
**类型**: backend (后端开发)
**预估工时**: 3h
**依赖**: M2-022

---

## 任务描述

实现基于 Socket.io 的实时事件系统，支持房间内的消息广播、状态同步等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M2-002-01 | 设计事件类型定义 | Event Types | 20min |
| M2-002-02 | 实现事件处理器 | Handler | 30min |
| M2-002-03 | 实现房间广播 | Room Broadcast | 30min |
| M2-002-04 | 实现私聊功能 | Private Message | 25min |
| M2-002-05 | 实现状态同步 | State Sync | 35min |
| M2-002-06 | 实现事件过滤 | Event Filter | 25min |
| M2-002-07 | 编写事件测试 | 测试覆盖 | 30min |
| M2-002-08 | 编写文档 | 使用说明 | 15min |

---

## 事件类型定义

```python
# app/websocket/events.py
from enum import Enum
from pydantic import BaseModel
from typing import Optional, Any, Dict, List

class EventType(str, Enum):
    """WebSocket 事件类型"""
    # 连接事件
    CONNECT = "connection"
    DISCONNECT = "disconnect"
    JOIN_ROOM = "join_room"
    LEAVE_ROOM = "leave_room"

    # 聊天事件
    CHAT_MESSAGE = "chat_message"
    PRIVATE_MESSAGE = "private_message"
    SYSTEM_MESSAGE = "system_message"
    KP_MESSAGE = "kp_message"

    # 游戏事件
    ROLL = "roll"
    CHECK = "check"
    DAMAGE = "damage"
    HEAL = "heal"

    # 状态事件
    STATE_UPDATE = "state_update"
    CHARACTER_UPDATE = "character_update"
    ROOM_UPDATE = "room_update"

    # 场景事件
    SCENE_CHANGE = "scene_change"
    CLUE_REVEAL = "clue_reveal"
    HANDOUT_DISTRIBUTE = "handout_distribute"

class WebSocketEvent(BaseModel):
    """WebSocket 事件基类"""
    type: EventType
    room_id: str
    sender_id: str
    data: Dict[str, Any]
    timestamp: float
    recipients: Optional[List[str]] = None  # None = all, [] = specific users
```

---

## 事件处理器

```python
# app/websocket/handler.py
from socketio import AsyncServer
from typing import Dict, List, Optional
import json
import time

from app.websocket.events import EventType, WebSocketEvent

class EventHandler:
    """WebSocket 事件处理器"""

    def __init__(self, sio: AsyncServer):
        self.sio = sio
        self._setup_handlers()

    def _setup_handlers(self):
        """设置事件处理器"""
        self.sio.on(EventType.CONNECT, self._handle_connect)
        self.sio.on(EventType.DISCONNECT, self._handle_disconnect)
        self.sio.on(EventType.JOIN_ROOM, self._handle_join_room)
        self.sio.on(EventType.LEAVE_ROOM, self._handle_leave_room)
        self.sio.on(EventType.CHAT_MESSAGE, self._handle_chat_message)
        self.sio.on(EventType.PRIVATE_MESSAGE, self._handle_private_message)
        self.sio.on(EventType.ROLL, self._handle_roll)
        self.sio.on(EventType.CHECK, self._handle_check)

    async def _handle_connect(self, sid, environ):
        """处理连接"""
        print(f"Client connected: {sid}")
        await self.sio.emit(EventType.SYSTEM_MESSAGE, {
            "message": "已连接到服务器",
            "type": "info"
        }, to=sid)

    async def _handle_disconnect(self, sid):
        """处理断开"""
        print(f"Client disconnected: {sid}")

    async def _handle_join_room(self, sid, data):
        """处理加入房间"""
        room_id = data.get("room_id")
        if not room_id:
            return

        # 加入 Socket.io 房间
        self.sio.enter_room(sid, room_id)

        # 通知房间其他人
        await self._emit_to_room(
            EventType.JOIN_ROOM,
            room_id,
            {
                "user_id": data.get("user_id"),
                "username": data.get("username"),
            },
            exclude_sid=sid
        )

    async def _handle_leave_room(self, sid, data):
        """处理离开房间"""
        room_id = data.get("room_id")
        if not room_id:
            return

        # 离开 Socket.io 房间
        self.sio.leave_room(sid, room_id)

        # 通知房间其他人
        await self._emit_to_room(
            EventType.LEAVE_ROOM,
            room_id,
            {
                "user_id": data.get("user_id"),
            },
            exclude_sid=sid
        )

    async def _handle_chat_message(self, sid, data):
        """处理聊天消息"""
        room_id = data.get("room_id")
        message = data.get("message")
        sender = data.get("sender")

        if not room_id or not message:
            return

        await self._emit_to_room(
            EventType.CHAT_MESSAGE,
            room_id,
            {
                "sender_id": sender.get("id"),
                "sender_name": sender.get("name"),
                "message": message,
                "timestamp": time.time(),
            }
        )

    async def _handle_private_message(self, sid, data):
        """处理私聊消息"""
        target_id = data.get("target_id")
        sender = data.get("sender")
        message = data.get("message")

        if not target_id or not message:
            return

        # 发送给目标
        await self.sio.emit(
            EventType.PRIVATE_MESSAGE,
            {
                "sender_id": sender.get("id"),
                "sender_name": sender.get("name"),
                "message": message,
                "timestamp": time.time(),
            },
            room=target_id
        )

        # 发送给发送者确认
        await self.sio.emit(
            EventType.PRIVATE_MESSAGE,
            {
                "is_sent": True,
                "target_id": target_id,
                "message": message,
                "timestamp": time.time(),
            },
            room=sid
        )

    async def _handle_roll(self, sid, data):
        """处理掷骰"""
        room_id = data.get("room_id")
        expression = data.get("expression")
        result = data.get("result")
        sender = data.get("sender")

        if not room_id or not result:
            return

        await self._emit_to_room(
            EventType.ROLL,
            room_id,
            {
                "sender_id": sender.get("id"),
                "sender_name": sender.get("name"),
                "expression": expression,
                "result": result,
                "timestamp": time.time(),
            }
        )

    async def _handle_check(self, sid, data):
        """处理检定"""
        room_id = data.get("room_id")
        check_data = data.get("check")
        sender = data.get("sender")

        if not room_id or not check_data:
            return

        await self._emit_to_room(
            EventType.CHECK,
            room_id,
            {
                "sender_id": sender.get("id"),
                "sender_name": sender.get("name"),
                "check": check_data,
                "timestamp": time.time(),
            }
        )

    async def _emit_to_room(
        self,
        event_type: EventType,
        room_id: str,
        data: Dict[str, Any],
        exclude_sid: Optional[str] = None
    ):
        """向房间发送事件"""
        event = WebSocketEvent(
            type=event_type,
            room_id=room_id,
            sender_id=data.get("sender_id", ""),
            data=data,
            timestamp=time.time()
        )

        if exclude_sid:
            await self.sio.emit(
                event_type,
                event.dict(),
                room=room_id,
                skip_sid=exclude_sid
            )
        else:
            await self.sio.emit(
                event_type,
                event.dict(),
                room=room_id
            )

    async def emit_to_user(
        self,
        event_type: EventType,
        user_id: str,
        data: Dict[str, Any]
    ):
        """向特定用户发送事件"""
        await self.sio.emit(
            event_type,
            data,
            room=user_id
        )
```

---

## Socket.io 服务配置

```python
# app/websocket/server.py
from socketio import AsyncServer
from app.websocket.handler import EventHandler

def create_socketio_app() -> AsyncServer:
    """创建 Socket.io 服务器"""
    sio = AsyncServer(
        async_mode='asgi',
        cors_allowed_origins='*',
        logger=True,
        engineio_logger=False,
    )

    # 设置事件处理器
    handler = EventHandler(sio)

    return sio

# 全局 Socket.io 实例
socketio_app = create_socketio_app()
```

---

## FastAPI 集成

```python
# app/main.py
from fastapi import FastAPI
from app.websocket.server import socketio_app

app = FastAPI()

# 挂载 Socket.io
app.mount("/socket.io", socketio_app)

@app.on_event("startup")
async def startup():
    """启动时初始化 WebSocket"""
    print("WebSocket server started")

@app.on_event("shutdown")
async def shutdown():
    """关闭时清理 WebSocket"""
    print("WebSocket server stopped")
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/websocket/events.py` | 创建 | 事件类型定义 |
| `app/websocket/handler.py` | 创建 | 事件处理器 |
| `app/websocket/server.py` | 创建 | Socket.io 服务器 |
| `app/main.py` | 更新 | 集成 WebSocket |
| `tests/test_websocket.py` | 创建 | WebSocket 测试 |

---

## 验收标准

- [ ] 事件类型完整
- [ ] 处理器正确响应
- [ ] 房间广播有效
- [ ] 私聊功能正常
- [ ] 状态同步准确
- [ ] 事件过滤正确
- [ ] 测试覆盖全面

---

## 参考文档

- M2-001: 房间管理系统
- M2-022: Socket.io 服务配置
- Socket.io 文档

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
