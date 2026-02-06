# M2-012: 实现房间状态同步

**任务类型**: Backend + Frontend
**预估工时**: 8h
**优先级**: P0
**依赖**: M2-022 (Socket.io 配置)

---

## 任务描述

实现多人房间状态的实时同步机制，确保所有客户端在加入房间、离开房间、角色变化等场景下能够实时获得一致的状态视图。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 | 依赖 |
|----|--------|------|----------|------|
| M2-012-01 | 设计 RoomState 数据结构 | 定义房间状态数据模型 | 1h | - |
| M2-012-02 | 实现状态序列化/反序列化 | 确保状态可安全传输 | 1h | M2-012-01 |
| M2-012-03 | 实现状态变化监听器 | 监听状态变更事件 | 1.5h | M2-012-01 |
| M2-012-04 | 实现增量更新机制 | 只传输变化的部分 | 1.5h | M2-012-03 |
| M2-012-05 | 实现全量同步机制 | 新用户加入时同步完整状态 | 1h | M2-012-04 |
| M2-012-06 | 实现前端状态同步 Hook | React 状态管理集成 | 1h | M2-012-05 |
| M2-012-07 | 编写单元测试 | 测试同步逻辑 | 1h | M2-012-06 |

---

## 后端代码示例

### 1. RoomState 数据模型

```python
# backend/models/room_state.py
from dataclasses import dataclass, field, asdict
from datetime import datetime
from typing import Dict, List, Optional, Any
from enum import Enum
import json

class RoomStatus(Enum):
    """房间状态枚举"""
    WAITING = "waiting"      # 等待中
    ACTIVE = "active"        # 游戏中
    PAUSED = "paused"        # 暂停
    ENDED = "ended"          # 已结束

@dataclass
class PlayerState:
    """玩家状态"""
    user_id: str
    username: str
    role: str  # "kp" | "player"
    is_online: bool
    joined_at: datetime
    last_active: datetime
    avatar_url: Optional[str] = None
    is_muted: bool = False
    connection_id: Optional[str] = None

@dataclass
class CampaignState:
    """战役状态"""
    campaign_id: str
    name: str
    current_scene: Optional[str] = None
    turn_count: int = 0

@dataclass
class SpotlightState:
    """聚光灯状态"""
    current_user_id: Optional[str] = None
    queue: List[str] = field(default_factory=list)

@dataclass
class RoomState:
    """房间完整状态"""
    room_id: str
    status: RoomStatus
    campaign: CampaignState
    players: Dict[str, PlayerState]
    spotlight: SpotlightState
    created_at: datetime
    updated_at: datetime
    metadata: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> dict:
        """序列化为字典"""
        return {
            "room_id": self.room_id,
            "status": self.status.value,
            "campaign": asdict(self.campaign),
            "players": {
                uid: {
                    **asdict(player),
                    "joined_at": player.joined_at.isoformat(),
                    "last_active": player.last_active.isoformat()
                }
                for uid, player in self.players.items()
            },
            "spotlight": asdict(self.spotlight),
            "created_at": self.created_at.isoformat(),
            "updated_at": self.updated_at.isoformat(),
            "metadata": self.metadata
        }

    def to_json(self) -> str:
        """序列化为 JSON"""
        return json.dumps(self.to_dict())

    @classmethod
    def from_dict(cls, data: dict) -> "RoomState":
        """从字典反序列化"""
        players = {
            uid: PlayerState(
                **{k: v for k, v in player.items()
                  if k not in ["joined_at", "last_active"]},
                joined_at=datetime.fromisoformat(player["joined_at"]),
                last_active=datetime.fromisoformat(player["last_active"])
            )
            for uid, player in data["players"].items()
        }

        return cls(
            room_id=data["room_id"],
            status=RoomStatus(data["status"]),
            campaign=CampaignState(**data["campaign"]),
            players=players,
            spotlight=SpotlightState(**data["spotlight"]),
            created_at=datetime.fromisoformat(data["created_at"]),
            updated_at=datetime.fromisoformat(data["updated_at"]),
            metadata=data.get("metadata", {})
        )
```

### 2. 状态管理器

```python
# backend/services/room_state_manager.py
from typing import Dict, Optional, Callable, List
from datetime import datetime
import asyncio
from loguru import logger

from models.room_state import RoomState, PlayerState, RoomStatus

class RoomStateManager:
    """房间状态管理器"""

    def __init__(self):
        self._states: Dict[str, RoomState] = {}
        self._listeners: Dict[str, List[Callable]] = {}
        self._lock = asyncio.Lock()

    def get_state(self, room_id: str) -> Optional[RoomState]:
        """获取房间状态"""
        return self._states.get(room_id)

    async def create_room(self, room_id: str, campaign_data: dict) -> RoomState:
        """创建新房间"""
        async with self._lock:
            state = RoomState(
                room_id=room_id,
                status=RoomStatus.WAITING,
                campaign=CampaignState(
                    campaign_id=campaign_data["id"],
                    name=campaign_data["name"]
                ),
                players={},
                spotlight=SpotlightState(),
                created_at=datetime.utcnow(),
                updated_at=datetime.utcnow()
            )
            self._states[room_id] = state
            await self._notify_change(room_id, "room_created", state.to_dict())
            logger.info(f"Room {room_id} created")
            return state

    async def add_player(
        self,
        room_id: str,
        user_id: str,
        username: str,
        role: str,
        connection_id: str
    ) -> PlayerState:
        """添加玩家到房间"""
        async with self._lock:
            state = self._states.get(room_id)
            if not state:
                raise ValueError(f"Room {room_id} not found")

            player = PlayerState(
                user_id=user_id,
                username=username,
                role=role,
                is_online=True,
                joined_at=datetime.utcnow(),
                last_active=datetime.utcnow(),
                connection_id=connection_id
            )

            state.players[user_id] = player
            state.updated_at = datetime.utcnow()

            await self._notify_change(
                room_id,
                "player_joined",
                {
                    "player": player.to_dict() if hasattr(player, 'to_dict') else self._player_to_dict(player),
                    "room_state": state.to_dict()
                }
            )
            logger.info(f"Player {username} joined room {room_id}")
            return player

    async def remove_player(self, room_id: str, user_id: str):
        """从房间移除玩家"""
        async with self._lock:
            state = self._states.get(room_id)
            if not state or user_id not in state.players:
                return

            player = state.players.pop(user_id)
            state.updated_at = datetime.utcnow()

            await self._notify_change(
                room_id,
                "player_left",
                {
                    "user_id": user_id,
                    "username": player.username,
                    "room_state": state.to_dict()
                }
            )
            logger.info(f"Player {user_id} left room {room_id}")

    async def update_player_status(
        self,
        room_id: str,
        user_id: str,
        is_online: Optional[bool] = None,
        is_muted: Optional[bool] = None
    ):
        """更新玩家状态"""
        async with self._lock:
            state = self._states.get(room_id)
            if not state or user_id not in state.players:
                return

            player = state.players[user_id]

            if is_online is not None:
                player.is_online = is_online
            if is_muted is not None:
                player.is_muted = is_muted

            player.last_active = datetime.utcnow()
            state.updated_at = datetime.utcnow()

            await self._notify_change(
                room_id,
                "player_status_updated",
                {
                    "user_id": user_id,
                    "is_online": player.is_online,
                    "is_muted": player.is_muted,
                    "last_active": player.last_active.isoformat()
                }
            )

    async def update_room_status(
        self,
        room_id: str,
        status: RoomStatus
    ):
        """更新房间状态"""
        async with self._lock:
            state = self._states.get(room_id)
            if not state:
                return

            state.status = status
            state.updated_at = datetime.utcnow()

            await self._notify_change(
                room_id,
                "room_status_updated",
                {
                    "status": status.value,
                    "updated_at": state.updated_at.isoformat()
                }
            )

    async def get_incremental_update(
        self,
        room_id: str,
        since: datetime
    ) -> dict:
        """获取增量更新（自指定时间以来的变化）"""
        state = self._states.get(room_id)
        if not state:
            return {}

        if state.updated_at <= since:
            return {"type": "no_update"}

        return {
            "type": "incremental_update",
            "room_id": room_id,
            "state": state.to_dict(),
            "updated_at": state.updated_at.isoformat()
        }

    async def _notify_change(
        self,
        room_id: str,
        event_type: str,
        data: dict
    ):
        """通知状态变化"""
        if room_id in self._listeners:
            for callback in self._listeners[room_id]:
                try:
                    await callback(event_type, data)
                except Exception as e:
                    logger.error(f"Error in state change listener: {e}")

    def subscribe(
        self,
        room_id: str,
        callback: Callable
    ):
        """订阅状态变化"""
        if room_id not in self._listeners:
            self._listeners[room_id] = []
        self._listeners[room_id].append(callback)

    def unsubscribe(
        self,
        room_id: str,
        callback: Callable
    ):
        """取消订阅"""
        if room_id in self._listeners and callback in self._listeners[room_id]:
            self._listeners[room_id].remove(callback)

    def _player_to_dict(self, player: PlayerState) -> dict:
        """PlayerState 转字典辅助方法"""
        return {
            "user_id": player.user_id,
            "username": player.username,
            "role": player.role,
            "is_online": player.is_online,
            "joined_at": player.joined_at.isoformat(),
            "last_active": player.last_active.isoformat(),
            "avatar_url": player.avatar_url,
            "is_muted": player.is_muted,
            "connection_id": player.connection_id
        }

# 全局实例
room_state_manager = RoomStateManager()
```

### 3. WebSocket 事件处理器

```python
# backend/handlers/room_events.py
from socketio import AsyncServer
from loguru import logger

from services.room_state_manager import room_state_manager

class RoomEventHandler:
    """房间事件处理器"""

    def __init__(self, sio: AsyncServer):
        self.sio = sio
        self._setup_handlers()

    def _setup_handlers(self):
        """设置事件处理器"""

        @self.sio.event
        async def join_room(sid, data):
            """加入房间"""
            room_id = data.get("room_id")
            user_info = data.get("user", {})

            # 加入 Socket.IO 房间
            self.sio.enter_room(sid, room_id)

            # 获取或创建房间状态
            state = room_state_manager.get_state(room_id)
            if not state:
                # 新房间
                state = await room_state_manager.create_room(
                    room_id,
                    {"id": room_id, "name": data.get("campaign_name", "New Campaign")}
                )

            # 添加玩家
            await room_state_manager.add_player(
                room_id,
                user_info.get("user_id"),
                user_info.get("username"),
                user_info.get("role", "player"),
                sid
            )

            # 发送完整状态给新加入的玩家
            await self.sio.emit(
                "room_state_full",
                state.to_dict(),
                room=sid
            )

            # 通知其他玩家
            await self.sio.emit(
                "player_joined",
                {
                    "username": user_info.get("username"),
                    "player_count": len(state.players)
                },
                room=room_id,
                skip_sid=sid
            )

            logger.info(f"User {user_info.get('username')} joined room {room_id}")

        @self.sio.event
        async def leave_room(sid, data):
            """离开房间"""
            room_id = data.get("room_id")
            user_id = data.get("user_id")

            self.sio.leave_room(sid, room_id)
            await room_state_manager.remove_player(room_id, user_id)

            # 通知其他玩家
            await self.sio.emit(
                "player_left",
                {"user_id": user_id},
                room=room_id,
                skip_sid=sid
            )

            logger.info(f"User {user_id} left room {room_id}")

        @self.sio.event
        async def request_state_sync(sid, data):
            """请求状态同步"""
            room_id = data.get("room_id")
            sync_type = data.get("type", "full")  # "full" or "incremental"

            state = room_state_manager.get_state(room_id)
            if not state:
                await self.sio.emit(
                    "error",
                    {"message": "Room not found"},
                    room=sid
                )
                return

            if sync_type == "full":
                await self.sio.emit(
                    "room_state_full",
                    state.to_dict(),
                    room=sid
                )
            else:
                since = data.get("since")
                if since:
                    update = await room_state_manager.get_incremental_update(
                        room_id,
                        datetime.fromisoformat(since)
                    )
                    await self.sio.emit(
                        "room_state_incremental",
                        update,
                        room=sid
                    )

        @self.sio.event
        async def update_presence(sid, data):
            """更新在线状态"""
            room_id = data.get("room_id")
            user_id = data.get("user_id")

            await room_state_manager.update_player_status(
                room_id,
                user_id,
                is_online=True
            )

        # 订阅状态变化事件
        async def on_state_change(event_type: str, data: dict):
            """状态变化时广播"""
            room_id = data.get("room_id") or data.get("room_state", {}).get("room_id")
            if room_id:
                await self.sio.emit(
                    f"state_{event_type}",
                    data,
                    room=room_id
                )

        room_state_manager.subscribe("*", on_state_change)
```

---

## 前端代码示例

### 1. 类型定义

```typescript
// frontend/src/types/room.ts

export enum RoomStatus {
  WAITING = "waiting",
  ACTIVE = "active",
  PAUSED = "paused",
  ENDED = "ended"
}

export interface PlayerState {
  user_id: string;
  username: string;
  role: "kp" | "player";
  is_online: boolean;
  joined_at: string;
  last_active: string;
  avatar_url?: string;
  is_muted: boolean;
  connection_id?: string;
}

export interface CampaignState {
  campaign_id: string;
  name: string;
  current_scene?: string;
  turn_count: number;
}

export interface SpotlightState {
  current_user_id?: string;
  queue: string[];
}

export interface RoomState {
  room_id: string;
  status: RoomStatus;
  campaign: CampaignState;
  players: Record<string, PlayerState>;
  spotlight: SpotlightState;
  created_at: string;
  updated_at: string;
  metadata: Record<string, unknown>;
}

export interface StateUpdateEvent {
  type: "player_joined" | "player_left" | "player_status_updated" | "room_status_updated";
  data: unknown;
}
```

### 2. 房间状态同步 Hook

```typescript
// frontend/src/hooks/useRoomStateSync.ts
import { useEffect, useState, useCallback, useRef } from "react";
import { Socket } from "socket.io-client";
import { RoomState, PlayerState, StateUpdateEvent } from "@/types/room";

interface UseRoomStateSyncOptions {
  socket: Socket | null;
  roomId: string;
  userId: string;
  onStateChange?: (state: RoomState) => void;
  onPlayerJoined?: (player: PlayerState) => void;
  onPlayerLeft?: (userId: string) => void;
}

interface UseRoomStateSyncReturn {
  roomState: RoomState | null;
  isConnected: boolean;
  isSyncing: boolean;
  error: string | null;
  requestSync: (type?: "full" | "incremental") => Promise<void>;
  updatePresence: () => void;
}

export function useRoomStateSync({
  socket,
  roomId,
  userId,
  onStateChange,
  onPlayerJoined,
  onPlayerLeft
}: UseRoomStateSyncOptions): UseRoomStateSyncReturn {
  const [roomState, setRoomState] = useState<RoomState | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [isSyncing, setIsSyncing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const lastUpdateRef = useRef<string | null>(null);

  // 请求完整同步
  const requestSync = useCallback(async (type: "full" | "incremental" = "full") => {
    if (!socket || !socket.connected) {
      setError("Socket not connected");
      return;
    }

    setIsSyncing(true);
    setError(null);

    try {
      socket.emit("request_state_sync", {
        room_id: roomId,
        type,
        since: lastUpdateRef.current
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Sync request failed");
      setIsSyncing(false);
    }
  }, [socket, roomId]);

  // 更新在线状态
  const updatePresence = useCallback(() => {
    if (!socket || !socket.connected) return;

    socket.emit("update_presence", {
      room_id: roomId,
      user_id: userId
    });
  }, [socket, roomId, userId]);

  // 处理完整状态更新
  useEffect(() => {
    if (!socket) return;

    const handleFullState = (state: RoomState) => {
      setRoomState(state);
      lastUpdateRef.current = state.updated_at;
      setIsSyncing(false);
      onStateChange?.(state);
    };

    const handleIncrementalUpdate = (update: {
      type: string;
      room_id: string;
      state: RoomState;
      updated_at: string;
    }) => {
      if (update.type === "no_update") {
        setIsSyncing(false);
        return;
      }

      setRoomState(update.state);
      lastUpdateRef.current = update.updated_at;
      setIsSyncing(false);
      onStateChange?.(update.state);
    };

    const handlePlayerJoined = (data: {
      username: string;
      player_count: number;
    }) => {
      onPlayerJoined?.({
        user_id: "",
        username: data.username,
        role: "player",
        is_online: true,
        joined_at: new Date().toISOString(),
        last_active: new Date().toISOString(),
        is_muted: false
      });
    };

    const handlePlayerLeft = (data: { user_id: string }) => {
      onPlayerLeft?.(data.user_id);
    };

    const handleStatePlayerJoined = (data: { player: PlayerState; room_state: RoomState }) => {
      setRoomState(data.room_state);
      onPlayerJoined?.(data.player);
    };

    const handleStatePlayerLeft = (data: { user_id: string; username: string; room_state: RoomState }) => {
      setRoomState(data.room_state);
      onPlayerLeft?.(data.user_id);
    };

    const handleStatePlayerStatusUpdated = (data: {
      user_id: string;
      is_online: boolean;
      is_muted: boolean;
      last_active: string;
    }) => {
      setRoomState(prev => {
        if (!prev) return prev;
        return {
          ...prev,
          players: {
            ...prev.players,
            [data.user_id]: {
              ...prev.players[data.user_id],
              is_online: data.is_online,
              is_muted: data.is_muted,
              last_active: data.last_active
            }
          }
        };
      });
    };

    const handleStateRoomStatusUpdated = (data: { status: RoomStatus; updated_at: string }) => {
      setRoomState(prev => prev ? { ...prev, status: data.status } : null);
    };

    const handleError = (err: { message: string }) => {
      setError(err.message);
      setIsSyncing(false);
    };

    // 注册事件监听
    socket.on("room_state_full", handleFullState);
    socket.on("room_state_incremental", handleIncrementalUpdate);
    socket.on("player_joined", handlePlayerJoined);
    socket.on("player_left", handlePlayerLeft);
    socket.on("state_player_joined", handleStatePlayerJoined);
    socket.on("state_player_left", handleStatePlayerLeft);
    socket.on("state_player_status_updated", handleStatePlayerStatusUpdated);
    socket.on("state_room_status_updated", handleStateRoomStatusUpdated);
    socket.on("error", handleError);

    return () => {
      socket.off("room_state_full", handleFullState);
      socket.off("room_state_incremental", handleIncrementalUpdate);
      socket.off("player_joined", handlePlayerJoined);
      socket.off("player_left", handlePlayerLeft);
      socket.off("state_player_joined", handleStatePlayerJoined);
      socket.off("state_player_left", handleStatePlayerLeft);
      socket.off("state_player_status_updated", handleStatePlayerStatusUpdated);
      socket.off("state_room_status_updated", handleStateRoomStatusUpdated);
      socket.off("error", handleError);
    };
  }, [socket, onStateChange, onPlayerJoined, onPlayerLeft]);

  // 监听连接状态
  useEffect(() => {
    if (!socket) return;

    const handleConnect = () => setIsConnected(true);
    const handleDisconnect = () => setIsConnected(false);

    socket.on("connect", handleConnect);
    socket.on("disconnect", handleDisconnect);

    setIsConnected(socket.connected);

    return () => {
      socket.off("connect", handleConnect);
      socket.off("disconnect", handleDisconnect);
    };
  }, [socket]);

  // 定期更新在线状态
  useEffect(() => {
    if (!isConnected) return;

    const interval = setInterval(() => {
      updatePresence();
    }, 30000); // 每30秒

    return () => clearInterval(interval);
  }, [isConnected, updatePresence]);

  // 加入房间时自动同步
  useEffect(() => {
    if (isConnected && roomId) {
      requestSync("full");
    }
  }, [isConnected, roomId]);

  return {
    roomState,
    isConnected,
    isSyncing,
    error,
    requestSync,
    updatePresence
  };
}
```

### 3. 房间状态上下文

```typescript
// frontend/src/contexts/RoomStateContext.tsx
import React, { createContext, useContext, ReactNode } from "react";
import { Socket } from "socket.io-client";
import { useRoomStateSync } from "@/hooks/useRoomStateSync";
import { RoomState, PlayerState } from "@/types/room";

interface RoomStateContextValue {
  roomState: RoomState | null;
  isConnected: boolean;
  isSyncing: boolean;
  error: string | null;
  players: PlayerState[];
  onlinePlayers: PlayerState[];
  currentUserId: string;
  isKP: boolean;
  requestSync: () => Promise<void>;
  updatePresence: () => void;
}

const RoomStateContext = createContext<RoomStateContextValue | undefined>(undefined);

interface RoomStateProviderProps {
  children: ReactNode;
  socket: Socket | null;
  roomId: string;
  userId: string;
}

export function RoomStateProvider({
  children,
  socket,
  roomId,
  userId
}: RoomStateProviderProps) {
  const sync = useRoomStateSync({
    socket,
    roomId,
    userId,
    onPlayerJoined: (player) => {
      console.log("Player joined:", player);
    },
    onPlayerLeft: (userId) => {
      console.log("Player left:", userId);
    }
  });

  const players = React.useMemo(() => {
    if (!sync.roomState) return [];
    return Object.values(sync.roomState.players);
  }, [sync.roomState]);

  const onlinePlayers = React.useMemo(() => {
    return players.filter(p => p.is_online);
  }, [players]);

  const isKP = React.useMemo(() => {
    if (!sync.roomState) return false;
    const currentUser = sync.roomState.players[userId];
    return currentUser?.role === "kp";
  }, [sync.roomState, userId]);

  const value: RoomStateContextValue = {
    roomState: sync.roomState,
    isConnected: sync.isConnected,
    isSyncing: sync.isSyncing,
    error: sync.error,
    players,
    onlinePlayers,
    currentUserId: userId,
    isKP,
    requestSync: sync.requestSync,
    updatePresence: sync.updatePresence
  };

  return (
    <RoomStateContext.Provider value={value}>
      {children}
    </RoomStateContext.Provider>
  );
}

export function useRoomState() {
  const context = useContext(RoomStateContext);
  if (!context) {
    throw new Error("useRoomState must be used within RoomStateProvider");
  }
  return context;
}
```

### 4. 玩家列表组件

```typescript
// frontend/src/components/room/PlayerList.tsx
import React from "react";
import { useRoomState } from "@/contexts/RoomStateContext";
import { PlayerItem } from "./PlayerItem";

export function PlayerList() {
  const { players, onlinePlayers, currentUserId } = useRoomState();

  return (
    <div className="player-list">
      <div className="player-list-header">
        <h3>Players ({onlinePlayers.length}/{players.length})</h3>
      </div>

      <div className="player-list-content">
        {players.map(player => (
          <PlayerItem
            key={player.user_id}
            player={player}
            isCurrentUser={player.user_id === currentUserId}
          />
        ))}
      </div>

      {onlinePlayers.length === 0 && (
        <div className="player-list-empty">
          No players online
        </div>
      )}
    </div>
  );
}
```

```typescript
// frontend/src/components/room/PlayerItem.tsx
import React from "react";
import { PlayerState } from "@/types/room";

interface PlayerItemProps {
  player: PlayerState;
  isCurrentUser: boolean;
}

export function PlayerItem({ player, isCurrentUser }: PlayerItemProps) {
  return (
    <div className={`player-item ${!player.is_online ? "offline" : ""} ${isCurrentUser ? "current" : ""}`}>
      <div className="player-avatar">
        {player.avatar_url ? (
          <img src={player.avatar_url} alt={player.username} />
        ) : (
          <div className="avatar-placeholder">
            {player.username[0].toUpperCase()}
          </div>
        )}
        <div className={`online-indicator ${player.is_online ? "online" : "offline"}`} />
      </div>

      <div className="player-info">
        <div className="player-name">
          {player.username}
          {isCurrentUser && <span className="you-badge">(You)</span>}
        </div>
        <div className="player-role">
          {player.role === "kp" ? "Game Master" : "Player"}
        </div>
      </div>

      <div className="player-status">
        {player.is_muted && (
          <span className="muted-badge" title="Muted">🔇</span>
        )}
        {!player.is_online && (
          <span className="offline-badge">Offline</span>
        )}
      </div>
    </div>
  );
}
```

---

## 涉及文件清单

### 后端文件

| 文件路径 | 操作 | 说明 |
|---------|------|------|
| `backend/models/room_state.py` | 创建 | 房间状态数据模型 |
| `backend/services/room_state_manager.py` | 创建 | 房间状态管理服务 |
| `backend/handlers/room_events.py` | 创建 | WebSocket 房间事件处理 |
| `backend/api/routes/rooms.py` | 修改 | 添加房间状态查询 API |
| `backend/config/socketio.py` | 修改 | 集成房间事件处理器 |

### 前端文件

| 文件路径 | 操作 | 说明 |
|---------|------|------|
| `frontend/src/types/room.ts` | 创建 | 房间相关类型定义 |
| `frontend/src/hooks/useRoomStateSync.ts` | 创建 | 房间状态同步 Hook |
| `frontend/src/contexts/RoomStateContext.tsx` | 创建 | 房间状态上下文 |
| `frontend/src/components/room/PlayerList.tsx` | 创建 | 玩家列表组件 |
| `frontend/src/components/room/PlayerItem.tsx` | 创建 | 玩家列表项组件 |
| `frontend/src/pages/GamePage.tsx` | 修改 | 集成房间状态同步 |
| `frontend/src/utils/socket.ts` | 修改 | 添加房间相关事件类型 |

---

## 验收标准

### 功能验收

- [ ] 玩家加入房间时，其他玩家能实时收到通知
- [ ] 玩家离开房间时，其他玩家能实时收到通知
- [ ] 新加入的玩家能获取完整的房间状态
- [ ] 在线玩家状态每 30 秒自动更新
- [ ] 玩家离线后状态能正确更新
- [ ] 支持全量和增量两种同步模式
- [ ] 房间状态变化能实时广播给所有玩家

### 性能验收

- [ ] 状态更新延迟 < 500ms
- [ ] 增量更新只传输变化的数据
- [ ] 支持 10+ 玩家同时在线
- [ ] 内存占用稳定（无内存泄漏）

### 兼容性验收

- [ ] 支持桌面浏览器（Chrome、Firefox、Safari、Edge）
- [ ] 支持移动浏览器（iOS Safari、Android Chrome）
- [ ] 断线重连后状态能正确恢复

---

## 参考文档

- [Socket.IO 官方文档 - Rooms](https://socket.io/docs/v4/rooms/)
- [Socket.IO 官方文档 - Emitting cheatsheet](https://socket.io/docs/v4/emitting-events/)
- [React Hooks 最佳实践](https://react.dev/reference/react)
- [WebSocket 状态同步模式](https://en.wikipedia.org/wiki/State_synchronization)
- [乐观更新 UI 模式](https://react.dev/learn/keeping-components-pure)

---

**创建日期**: 2026-02-06
**负责人**: 待分配
**审核人**: 待分配
