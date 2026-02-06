# M2-016: 实现玩家静音

**任务类型**: Backend + Frontend
**预估工时**: 5h
**优先级**: P1
**依赖**: M2-012 (房间状态同步)

---

## 任务描述

实现玩家静音功能。KP 可以静音/取消静音房间内的任何玩家。被静音的玩家无法发送消息，但可以接收消息。支持实时状态同步和持久化。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 | 依赖 |
|----|--------|------|----------|------|
| M2-016-01 | 设计静音状态数据结构 | RoomState 添加 is_muted 字段 | 0.5h | M2-012 |
| M2-016-02 | 实现后端静音 API | PUT /campaigns/:id/members/:uid/mute | 1h | M2-016-01 |
| M2-016-03 | 实现消息发送验证 | 检查发送者是否被静音 | 1h | M2-016-02 |
| M2-016-04 | 实现 WebSocket 静音事件 | 实时广播静音状态 | 1h | M2-016-03 |
| M2-016-05 | 实现前端静音 UI | 玩家列表静音按钮 | 1h | M2-016-04 |
| M2-016-06 | 实现被静音提示 | 用户发送消息时提示 | 0.5h | M2-016-05 |

---

## 后端代码示例

### 1. 静音服务

```python
# backend/services/mute_service.py
from typing import List
from datetime import datetime
from loguru import logger
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select, update

from models.campaign import CampaignMember
from models.room_state import RoomState, PlayerState
from services.room_state_manager import room_state_manager


class MuteService:
    """玩家静音服务"""

    def __init__(self, db: AsyncSession):
        self.db = db

    async def mute_player(
        self,
        campaign_id: str,
        operator_id: str,
        target_user_id: str
    ) -> dict:
        """
        静音玩家

        Args:
            campaign_id: 战役 ID
            operator_id: 操作者 ID（必须是 KP）
            target_user_id: 目标玩家 ID

        Returns:
            操作结果
        """
        # 1. 验证操作者权限
        operator = await self._get_member(campaign_id, operator_id)
        if not operator or operator.role != "kp":
            raise PermissionError("Only KP can mute players")

        # 2. 验证目标玩家
        target = await self._get_member(campaign_id, target_user_id)
        if not target:
            raise ValueError("Target player not found")

        # 3. 不能静音 KP
        if target.role == "kp":
            raise ValueError("Cannot mute KP")

        # 4. 不能静音自己
        if operator_id == target_user_id:
            raise ValueError("Cannot mute yourself")

        # 5. 更新数据库（如果需要持久化）
        await self._update_mute_status(campaign_id, target_user_id, True)

        # 6. 更新房间状态
        await room_state_manager.update_player_status(
            campaign_id,
            target_user_id,
            is_muted=True
        )

        logger.info(
            f"Player {target_user_id} muted by {operator_id} "
            f"in campaign {campaign_id}"
        )

        return {
            "success": True,
            "target_user_id": target_user_id,
            "target_username": target.user.username,
            "is_muted": True,
            "muted_at": datetime.utcnow().isoformat()
        }

    async def unmute_player(
        self,
        campaign_id: str,
        operator_id: str,
        target_user_id: str
    ) -> dict:
        """
        取消静音玩家

        Args:
            campaign_id: 战役 ID
            operator_id: 操作者 ID（必须是 KP）
            target_user_id: 目标玩家 ID

        Returns:
            操作结果
        """
        # 1. 验证操作者权限
        operator = await self._get_member(campaign_id, operator_id)
        if not operator or operator.role != "kp":
            raise PermissionError("Only KP can unmute players")

        # 2. 验证目标玩家
        target = await self._get_member(campaign_id, target_user_id)
        if not target:
            raise ValueError("Target player not found")

        # 3. 更新数据库
        await self._update_mute_status(campaign_id, target_user_id, False)

        # 4. 更新房间状态
        await room_state_manager.update_player_status(
            campaign_id,
            target_user_id,
            is_muted=False
        )

        logger.info(
            f"Player {target_user_id} unmuted by {operator_id} "
            f"in campaign {campaign_id}"
        )

        return {
            "success": True,
            "target_user_id": target_user_id,
            "target_username": target.user.username,
            "is_muted": False,
            "unmuted_at": datetime.utcnow().isoformat()
        }

    async def toggle_mute(
        self,
        campaign_id: str,
        operator_id: str,
        target_user_id: str
    ) -> dict:
        """
        切换静音状态

        Args:
            campaign_id: 战役 ID
            operator_id: 操作者 ID
            target_user_id: 目标玩家 ID

        Returns:
            操作结果
        """
        target = await self._get_member(campaign_id, target_user_id)
        if not target:
            raise ValueError("Target player not found")

        # 获取当前静音状态
        state = room_state_manager.get_state(campaign_id)
        if not state or target_user_id not in state.players:
            raise ValueError("Player not in room state")

        current_muted = state.players[target_user_id].is_muted

        if current_muted:
            return await self.unmute_player(campaign_id, operator_id, target_user_id)
        else:
            return await self.mute_player(campaign_id, operator_id, target_user_id)

    async def is_player_muted(
        self,
        campaign_id: str,
        user_id: str
    ) -> bool:
        """
        检查玩家是否被静音

        Args:
            campaign_id: 战役 ID
            user_id: 用户 ID

        Returns:
            是否被静音
        """
        state = room_state_manager.get_state(campaign_id)
        if not state or user_id not in state.players:
            return False

        return state.players[user_id].is_muted

    async def get_muted_players(self, campaign_id: str) -> List[str]:
        """
        获取所有被静音的玩家 ID

        Args:
            campaign_id: 战役 ID

        Returns:
            被静音的玩家 ID 列表
        """
        state = room_state_manager.get_state(campaign_id)
        if not state:
            return []

        return [
            user_id for user_id, player in state.players.items()
            if player.is_muted
        ]

    async def _get_member(
        self,
        campaign_id: str,
        user_id: str
    ):
        """获取战役成员"""
        result = await self.db.execute(
            select(CampaignMember).where(
                CampaignMember.campaign_id == campaign_id,
                CampaignMember.user_id == user_id
            )
        )
        return result.scalar_one_or_none()

    async def _update_mute_status(
        self,
        campaign_id: str,
        user_id: str,
        is_muted: bool
    ):
        """更新数据库中的静音状态（可选）"""
        # 如果需要持久化静音状态，可以实现数据库更新
        # 目前使用内存状态即可
        pass
```

### 2. 消息发送验证中间件

```python
# backend/middleware/message_validator.py
from fastapi import HTTPException, status
from loguru import logger

from services.mute_service import MuteService


async def validate_message_not_muted(
    campaign_id: str,
    user_id: str,
    mute_service: MuteService
):
    """
    验证用户是否被静音

    Raises:
        HTTPException: 如果用户被静音
    """
    is_muted = await mute_service.is_player_muted(campaign_id, user_id)

    if is_muted:
        logger.warning(f"Muted user {user_id} attempted to send message in {campaign_id}")
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail={
                "error": "user_muted",
                "message": "You are muted and cannot send messages"
            }
        )
```

### 3. API 路由

```python
# backend/api/routes/player_mute.py
from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel

from database import get_db
from services.mute_service import MuteService
from middleware.auth import get_current_user

router = APIRouter(prefix="/api/campaigns", tags=["mute"])


class MuteResponse(BaseModel):
    """静音响应"""
    success: bool
    target_user_id: str
    target_username: str
    is_muted: bool
    muted_at: str | None = None
    unmuted_at: str | None = None


@router.put(
    "/{campaign_id}/members/{user_id}/mute",
    response_model=MuteResponse,
    status_code=status.HTTP_200_OK
)
async def mute_player(
    campaign_id: str,
    user_id: str,
    current_user = Depends(get_current_user),
    db = Depends(get_db)
):
    """静音玩家"""
    try:
        service = MuteService(db)
        result = await service.mute_player(
            campaign_id=campaign_id,
            operator_id=current_user.id,
            target_user_id=user_id
        )
        return result

    except PermissionError as e:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail=str(e)
        )
    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(e)
        )
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to mute player"
        )


@router.put(
    "/{campaign_id}/members/{user_id}/unmute",
    response_model=MuteResponse,
    status_code=status.HTTP_200_OK
)
async def unmute_player(
    campaign_id: str,
    user_id: str,
    current_user = Depends(get_current_user),
    db = Depends(get_db)
):
    """取消静音玩家"""
    try:
        service = MuteService(db)
        result = await service.unmute_player(
            campaign_id=campaign_id,
            operator_id=current_user.id,
            target_user_id=user_id
        )
        return result

    except PermissionError as e:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail=str(e)
        )
    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(e)
        )
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to unmute player"
        )


@router.put(
    "/{campaign_id}/members/{user_id}/mute/toggle",
    response_model=MuteResponse,
    status_code=status.HTTP_200_OK
)
async def toggle_mute(
    campaign_id: str,
    user_id: str,
    current_user = Depends(get_current_user),
    db = Depends(get_db)
):
    """切换静音状态"""
    try:
        service = MuteService(db)
        result = await service.toggle_mute(
            campaign_id=campaign_id,
            operator_id=current_user.id,
            target_user_id=user_id
        )
        return result

    except PermissionError as e:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail=str(e)
        )
    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(e)
        )
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to toggle mute"
        )
```

### 4. WebSocket 事件处理

```python
# backend/handlers/mute_events.py
from socketio import AsyncServer
from loguru import logger

from services.mute_service import MuteService


class MuteEventHandler:
    """静音相关 WebSocket 事件处理"""

    def __init__(self, sio: AsyncServer):
        self.sio = sio
        self._setup_handlers()

    def _setup_handlers(self):
        """设置事件处理器"""

        @self.sio.event
        async def toggle_mute(sid, data):
            """
            切换静音状态

            data: {
                room_id: str,
                target_user_id: str
            }
            """
            from database import get_db_context

            room_id = data.get("room_id")
            target_user_id = data.get("target_user_id")

            operator_info = self._get_socket_user(sid)
            if not operator_info:
                await self._emit_error(sid, "Unauthorized")
                return

            try:
                async with get_db_context() as db:
                    service = MuteService(db)
                    result = await service.toggle_mute(
                        campaign_id=room_id,
                        operator_id=operator_info["user_id"],
                        target_user_id=target_user_id
                    )

                    # 通知目标玩家
                    await self.sio.emit(
                        "mute_status_changed",
                        {
                            "is_muted": result["is_muted"],
                            "message": "You have been muted" if result["is_muted"] else "You have been unmuted"
                        },
                        room=target_user_id
                    )

                    # 通知房间所有人
                    await self.sio.emit(
                        "player_mute_updated",
                        {
                            "user_id": target_user_id,
                            "username": result["target_username"],
                            "is_muted": result["is_muted"]
                        },
                        room=room_id
                    )

                    # 向操作者确认
                    await self.sio.emit(
                        "mute_toggle_success",
                        result,
                        room=sid
                    )

                    logger.info(
                        f"Player {target_user_id} mute toggled to {result['is_muted']} "
                        f"by {operator_info['username']}"
                    )

            except PermissionError as e:
                await self._emit_error(sid, str(e), "PERMISSION_DENIED")
            except ValueError as e:
                await self._emit_error(sid, str(e), "INVALID_OPERATION")
            except Exception as e:
                logger.error(f"Error toggling mute: {e}")
                await self._emit_error(sid, "Failed to toggle mute")

    async def _emit_error(self, sid: str, message: str, code: str = "ERROR"):
        """发送错误消息"""
        await self.sio.emit("error", {"message": message, "code": code}, room=sid)

    def _get_socket_user(self, sid: str):
        """获取 Socket 连接的用户信息"""
        # 根据实际认证实现
        return None
```

---

## 前端代码示例

### 1. 静音 Hook

```typescript
// frontend/src/hooks/usePlayerMute.ts
import { useCallback } from "react";
import { Socket } from "socket.io-client";
import axios from "axios";

interface UsePlayerMuteOptions {
  socket?: Socket | null;
  roomId: string;
}

interface MuteResult {
  success: boolean;
  target_user_id: string;
  target_username: string;
  is_muted: boolean;
  muted_at?: string;
  unmuted_at?: string;
}

export function usePlayerMute({ socket, roomId }: UsePlayerMuteOptions) {
  const mutePlayer = useCallback(async (userId: string): Promise<MuteResult> => {
    try {
      const response = await axios.put<MuteResult>(
        `/api/campaigns/${roomId}/members/${userId}/mute`
      );
      return response.data;
    } catch (error) {
      if (axios.isAxiosError(error)) {
        throw new Error(error.response?.data?.detail || "Failed to mute player");
      }
      throw error;
    }
  }, [roomId]);

  const unmutePlayer = useCallback(async (userId: string): Promise<MuteResult> => {
    try {
      const response = await axios.put<MuteResult>(
        `/api/campaigns/${roomId}/members/${userId}/unmute`
      );
      return response.data;
    } catch (error) {
      if (axios.isAxiosError(error)) {
        throw new Error(error.response?.data?.detail || "Failed to unmute player");
      }
      throw error;
    }
  }, [roomId]);

  const toggleMute = useCallback((
    userId: string,
    onSuccess?: (result: MuteResult) => void,
    onError?: (error: string) => void
  ) => {
    if (!socket?.connected) {
      onError?.("Socket not connected");
      return;
    }

    socket.emit("toggle_mute", {
      room_id: roomId,
      target_user_id: userId
    });

    const handleSuccess = (result: MuteResult) => {
      onSuccess?.(result);
      socket.off("mute_toggle_success", handleSuccess);
      socket.off("error", handleError);
    };

    const handleError = (error: { message: string }) => {
      onError?.(error.message);
      socket.off("mute_toggle_success", handleSuccess);
      socket.off("error", handleError);
    };

    socket.once("mute_toggle_success", handleSuccess);
    socket.once("error", handleError);
  }, [socket, roomId]);

  return {
    mutePlayer,
    unmutePlayer,
    toggleMute
  };
}
```

### 2. 静音状态检测 Hook

```typescript
// frontend/src/hooks/useMuteDetection.ts
import { useEffect, useState, useCallback } from "react";
import { Socket } from "socket.io-client";
import { toast } from "react-hot-toast";

interface UseMuteDetectionOptions {
  socket: Socket | null;
  userId: string;
}

export function useMuteDetection({ socket, userId }: UseMuteDetectionOptions) {
  const [isMuted, setIsMuted] = useState(false);
  const [muteReason, setMuteReason] = useState<string | null>(null);

  const handleMuteStatusChanged = useCallback((data: {
    is_muted: boolean;
    message: string;
  }) => {
    setIsMuted(data.is_muted);
    setMuteReason(data.message);

    if (data.is_muted) {
      toast.error(data.message, {
        duration: 5000,
        icon: "🔇"
      });
    } else {
      toast.success(data.message, {
        duration: 3000,
        icon: "🔊"
      });
    }
  }, []);

  useEffect(() => {
    if (!socket) return;

    socket.on("mute_status_changed", handleMuteStatusChanged);

    return () => {
      socket.off("mute_status_changed", handleMuteStatusChanged);
    };
  }, [socket, handleMuteStatusChanged]);

  return {
    isMuted,
    muteReason
  };
}
```

### 3. 玩家列表项（带静音按钮）

```typescript
// frontend/src/components/room/PlayerItem.tsx
import React from "react";
import { Volume2, VolumeX } from "lucide-react";
import { PlayerState } from "@/types/room";
import { usePlayerMute } from "@/hooks/usePlayerMute";
import { useRoomState } from "@/contexts/RoomStateContext";
import { toast } from "react-hot-toast";

interface PlayerItemProps {
  player: PlayerState;
  isCurrentUser: boolean;
}

export function PlayerItem({ player, isCurrentUser }: PlayerItemProps) {
  const { isKP } = useRoomState();
  const { toggleMute } = usePlayerMute({
    roomId: "" // 从 context 获取
  });

  const handleToggleMute = () => {
    if (!isKP) return;

    toggleMute(
      player.user_id,
      (result) => {
        toast.success(
          `${result.target_username} is now ${result.is_muted ? 'muted' : 'unmuted'}`
        );
      },
      (error) => {
        toast.error(`Failed to toggle mute: ${error}`);
      }
    );
  };

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
          {player.is_muted && <span className="muted-badge">(Muted)</span>}
        </div>
        <div className="player-role">
          {player.role === "kp" ? "Game Master" : "Player"}
        </div>
      </div>

      <div className="player-actions">
        {!player.is_online && (
          <span className="offline-badge">Offline</span>
        )}

        {isKP && player.role !== "kp" && !isCurrentUser && (
          <button
            onClick={handleToggleMute}
            className={`mute-button ${player.is_muted ? "muted" : ""}`}
            title={player.is_muted ? "Unmute player" : "Mute player"}
          >
            {player.is_muted ? <VolumeX size={16} /> : <Volume2 size={16} />}
          </button>
        )}
      </div>
    </div>
  );
}
```

### 4. 消息输入组件（静音检查）

```typescript
// frontend/src/components/room/MessageInput.tsx
import React, { useState } from "react";
import { Send } from "lucide-react";
import { useMuteDetection } from "@/hooks/useMuteDetection";
import { useRoomState } from "@/contexts/RoomStateContext";
import { toast } from "react-hot-toast";

export function MessageInput() {
  const { currentUserId, socket } = useRoomState();
  const { isMuted } = useMuteDetection({
    socket,
    userId: currentUserId
  });

  const [message, setMessage] = useState("");

  const handleSend = () => {
    if (!message.trim()) return;

    if (isMuted) {
      toast.error("You are muted and cannot send messages", {
        icon: "🔇",
        id: "muted-error"
      });
      return;
    }

    // 发送消息
    socket?.emit("send_message", {
      room_id: "", // 从 context 获取
      content: message.trim()
    });

    setMessage("");
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="message-input-container">
      <textarea
        value={message}
        onChange={(e) => setMessage(e.target.value)}
        onKeyDown={handleKeyPress}
        placeholder={isMuted ? "You are muted..." : "Type a message..."}
        disabled={isMuted}
        className={`message-input ${isMuted ? "muted" : ""}`}
        rows={1}
      />

      <button
        onClick={handleSend}
        disabled={!message.trim() || isMuted}
        className="send-button"
        title={isMuted ? "You are muted" : "Send message"}
      >
        <Send size={20} />
      </button>
    </div>
  );
}
```

### 5. 静音状态指示器

```typescript
// frontend/src/components/room/MuteIndicator.tsx
import React from "react";
import { VolumeX } from "lucide-react";
import { useMuteDetection } from "@/hooks/useMuteDetection";

export function MuteIndicator() {
  const { isMuted, muteReason } = useMuteDetection({
    socket: null, // 从 context 获取
    userId: "" // 从 context 获取
  });

  if (!isMuted) return null;

  return (
    <div className="mute-indicator">
      <VolumeX size={16} />
      <span>{muteReason || "You are muted"}</span>
    </div>
  );
}
```

---

## 涉及文件清单

### 后端文件

| 文件路径 | 操作 | 说明 |
|---------|------|------|
| `backend/services/mute_service.py` | 创建 | 静音业务逻辑 |
| `backend/middleware/message_validator.py` | 创建 | 消息发送验证 |
| `backend/api/routes/player_mute.py` | 创建 | 静音 API 路由 |
| `backend/handlers/mute_events.py` | 创建 | WebSocket 静音事件处理 |
| `backend/api/routes/messages.py` | 修改 | 集成静音验证 |

### 前端文件

| 文件路径 | 操作 | 说明 |
|---------|------|------|
| `frontend/src/hooks/usePlayerMute.ts` | 创建 | 玩家静音 Hook |
| `frontend/src/hooks/useMuteDetection.ts` | 创建 | 静音状态检测 Hook |
| `frontend/src/components/room/PlayerItem.tsx` | 修改 | 添加静音按钮 |
| `frontend/src/components/room/MessageInput.tsx` | 修改 | 添加静音检查 |
| `frontend/src/components/room/MuteIndicator.tsx` | 创建 | 静音状态指示器 |

---

## 验收标准

### 功能验收

- [ ] KP 可以看到每个玩家的静音按钮
- [ ] KP 点击静音按钮切换玩家静音状态
- [ ] 被静音的玩家收到通知
- [ ] 被静音的玩家无法发送消息
- [ ] 被静音的玩家尝试发送消息时显示错误提示
- [ ] KP 可以取消静音玩家
- [ ] 静音状态实时同步到所有玩家
- [ ] KP 无法静音自己或其他 KP

### UX 验收

- [ ] 静音按钮有清晰的视觉反馈
- [ ] 被静音玩家的用户名显示 "Muted" 标签
- [ ] 消息输入框在静音时显示禁用状态
- [ ] 静音通知友好且清晰
- [ ] 音量图标直观表示静音状态

---

## 参考文档

- [WebSocket 事件广播](https://socket.io/docs/v4/broadcasting-events/)
- [React 状态管理最佳实践](https://react.dev/learn/managing-state)
- [权限控制模式](https://en.wikipedia.org/wiki/Role-based_access_control)
- [Toast 通知库](https://react-hot-toast.com/)

---

**创建日期**: 2026-02-06
**负责人**: 待分配
**审核人**: 待分配
