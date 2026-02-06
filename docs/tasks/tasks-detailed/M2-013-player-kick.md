# M2-013: 实现玩家踢出功能

**任务类型**: Backend + Frontend
**预估工时**: 6h
**优先级**: P0
**依赖**: M2-012 (房间状态同步), M2-014 (角色绑定)

---

## 任务描述

实现房间管理员（KP）踢出玩家功能。KP 可以将任何玩家踢出房间，被踢出的玩家将收到通知并被强制断开连接。需要确保权限验证和通知机制。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 | 依赖 |
|----|--------|------|----------|------|
| M2-013-01 | 设计踢出权限验证逻辑 | 验证操作者是否为 KP | 1h | M2-014 |
| M2-013-02 | 实现后端踢出 API | DELETE /campaigns/:id/members/:uid | 1.5h | M2-013-01 |
| M2-013-03 | 实现 WebSocket 踢出事件 | real-time kick 通知 | 1h | M2-013-02 |
| M2-013-04 | 实现被踢玩家处理 | 断开连接并显示原因 | 1h | M2-013-03 |
| M2-013-05 | 实现 KP 管理界面 | 玩家列表踢出按钮 | 1h | M2-013-04 |
| M2-013-06 | 添加踢出确认对话框 | 防止误操作 | 0.5h | M2-013-05 |

---

## 后端代码示例

### 1. 踢出服务

```python
# backend/services/kick_service.py
from typing import Optional
from datetime import datetime
from loguru import logger
from sqlalchemy.ext.asyncio import AsyncSession

from models.campaign import CampaignMember
from models.room_state import RoomState
from services.room_state_manager import room_state_manager


class KickService:
    """踢出玩家服务"""

    def __init__(self, db: AsyncSession):
        self.db = db

    async def kick_player(
        self,
        campaign_id: str,
        operator_id: str,
        target_user_id: str,
        reason: Optional[str] = None
    ) -> dict:
        """
        踢出玩家

        Args:
            campaign_id: 战役 ID
            operator_id: 操作者 ID（必须是 KP）
            target_user_id: 被踢出的玩家 ID
            reason: 踢出原因

        Returns:
            操作结果
        """
        # 1. 验证操作者权限
        operator = await self._get_member(campaign_id, operator_id)
        if not operator:
            raise ValueError("Operator not in campaign")

        if operator.role != "kp":
            raise PermissionError("Only KP can kick players")

        # 2. 不能踢出自己
        if operator_id == target_user_id:
            raise ValueError("Cannot kick yourself")

        # 3. 验证目标玩家存在
        target = await self._get_member(campaign_id, target_user_id)
        if not target:
            raise ValueError("Target player not found")

        # 4. 不能踢出其他 KP
        if target.role == "kp":
            raise ValueError("Cannot kick another KP")

        # 5. 从数据库移除成员
        await self._remove_member(campaign_id, target_user_id)

        # 6. 从房间状态移除
        await room_state_manager.remove_player(campaign_id, target_user_id)

        # 7. 记录操作日志
        await self._log_kick_action(
            campaign_id,
            operator_id,
            target_user_id,
            reason
        )

        logger.info(
            f"User {operator_id} kicked {target_user_id} "
            f"from campaign {campaign_id}. Reason: {reason}"
        )

        return {
            "success": True,
            "target_user_id": target_user_id,
            "target_username": target.user.username,
            "reason": reason,
            "kicked_at": datetime.utcnow().isoformat()
        }

    async def _get_member(
        self,
        campaign_id: str,
        user_id: str
    ) -> Optional[CampaignMember]:
        """获取战役成员"""
        from sqlalchemy import select

        result = await self.db.execute(
            select(CampaignMember).where(
                CampaignMember.campaign_id == campaign_id,
                CampaignMember.user_id == user_id
            )
        )
        return result.scalar_one_or_none()

    async def _remove_member(self, campaign_id: str, user_id: str):
        """从数据库移除成员"""
        from sqlalchemy import delete

        await self.db.execute(
            delete(CampaignMember).where(
                CampaignMember.campaign_id == campaign_id,
                CampaignMember.user_id == user_id
            )
        )
        await self.db.commit()

    async def _log_kick_action(
        self,
        campaign_id: str,
        operator_id: str,
        target_user_id: str,
        reason: Optional[str]
    ):
        """记录踢出操作日志"""
        # 实现日志记录，可用于审计
        pass
```

### 2. API 路由

```python
# backend/api/routes/campaign_members.py
from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel
from typing import Optional

from database import get_db
from services.kick_service import KickService
from middleware.auth import get_current_user

router = APIRouter(prefix="/api/campaigns", tags=["members"])


class KickPlayerRequest(BaseModel):
    """踢出玩家请求"""
    target_user_id: str
    reason: Optional[str] = None


class KickPlayerResponse(BaseModel):
    """踢出玩家响应"""
    success: bool
    target_user_id: str
    target_username: str
    reason: Optional[str]
    kicked_at: str


@router.delete(
    "/{campaign_id}/members/{user_id}",
    response_model=KickPlayerResponse,
    status_code=status.HTTP_200_OK
)
async def kick_player(
    campaign_id: str,
    user_id: str,
    request: KickPlayerRequest,
    current_user = Depends(get_current_user),
    db = Depends(get_db)
):
    """
    踢出玩家

    需要权限：KP
    """
    try:
        service = KickService(db)
        result = await service.kick_player(
            campaign_id=campaign_id,
            operator_id=current_user.id,
            target_user_id=user_id,
            reason=request.reason
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
            detail="Failed to kick player"
        )


@router.get("/{campaign_id}/members")
async def list_members(
    campaign_id: str,
    current_user = Depends(get_current_user),
    db = Depends(get_db)
):
    """获取成员列表"""
    from sqlalchemy import select
    from models.campaign import CampaignMember

    # 验证用户在战役中
    member = await db.execute(
        select(CampaignMember).where(
            CampaignMember.campaign_id == campaign_id,
            CampaignMember.user_id == current_user.id
        )
    )
    member = member.scalar_one_or_none()

    if not member:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Campaign not found or user not a member"
        )

    # 获取所有成员
    result = await db.execute(
        select(CampaignMember).where(
            CampaignMember.campaign_id == campaign_id
        )
    )
    members = result.scalars().all()

    return {
        "members": [
            {
                "user_id": m.user_id,
                "username": m.user.username,
                "role": m.role,
                "joined_at": m.joined_at.isoformat()
            }
            for m in members
        ]
    }
```

### 3. WebSocket 事件处理

```python
# backend/handlers/kick_events.py
from socketio import AsyncServer
from loguru import logger

from services.kick_service import KickService


class KickEventHandler:
    """踢出相关 WebSocket 事件处理"""

    def __init__(self, sio: AsyncServer):
        self.sio = sio
        self._setup_handlers()

    def _setup_handlers(self):
        """设置事件处理器"""

        @self.sio.event
        async def kick_player(sid, data):
            """
            踢出玩家 WebSocket 事件

            data: {
                room_id: str,
                target_user_id: str,
                reason?: str
            }
            """
            from database import get_db_context

            room_id = data.get("room_id")
            target_user_id = data.get("target_user_id")
            reason = data.get("reason")

            # 获取操作者信息
            operator_info = self._get_socket_user(sid)
            if not operator_info:
                await self.sio.emit(
                    "error",
                    {"message": "Unauthorized"},
                    room=sid
                )
                return

            try:
                async with get_db_context() as db:
                    service = KickService(db)
                    result = await service.kick_player(
                        campaign_id=room_id,
                        operator_id=operator_info["user_id"],
                        target_user_id=target_user_id,
                        reason=reason
                    )

                    # 通知被踢出的玩家
                    await self.sio.emit(
                        "kicked_from_room",
                        {
                            "room_id": room_id,
                            "reason": reason,
                            "kicked_by": operator_info["username"]
                        },
                        room=target_user_id  # 假设用户有自己的 room
                    )

                    # 断开被踢玩家的连接
                    await self._disconnect_user(target_user_id, room_id)

                    # 通知其他玩家
                    await self.sio.emit(
                        "player_kicked",
                        {
                            "user_id": target_user_id,
                            "username": result["target_username"],
                            "reason": reason,
                            "kicked_by": operator_info["username"]
                        },
                        room=room_id,
                        skip_sid=sid
                    )

                    # 向操作者确认
                    await self.sio.emit(
                        "kick_success",
                        result,
                        room=sid
                    )

                    logger.info(
                        f"Player {target_user_id} kicked from {room_id} "
                        f"by {operator_info['username']}"
                    )

            except PermissionError as e:
                await self.sio.emit(
                    "error",
                    {"message": str(e), "code": "PERMISSION_DENIED"},
                    room=sid
                )
            except ValueError as e:
                await self.sio.emit(
                    "error",
                    {"message": str(e), "code": "INVALID_OPERATION"},
                    room=sid
                )
            except Exception as e:
                logger.error(f"Error kicking player: {e}")
                await self.sio.emit(
                    "error",
                    {"message": "Failed to kick player", "code": "INTERNAL_ERROR"},
                    room=sid
                )

    def _get_socket_user(self, sid: str) -> Optional[dict]:
        """获取 Socket 连接的用户信息"""
        # 从 session 或其他存储中获取
        # 这里需要根据实际认证实现
        return None

    async def _disconnect_user(self, user_id: str, room_id: str):
        """断开用户连接"""
        # 查找用户的所有连接
        # 这里需要根据实际连接管理实现
        pass
```

---

## 前端代码示例

### 1. 踢出 Hook

```typescript
// frontend/src/hooks/useKickPlayer.ts
import { useCallback } from "react";
import { Socket } from "socket.io-client";
import axios from "axios";

interface UseKickPlayerOptions {
  socket?: Socket | null;
  roomId: string;
}

interface KickPlayerResult {
  success: boolean;
  target_user_id: string;
  target_username: string;
  reason?: string;
  kicked_at: string;
}

export function useKickPlayer({ socket, roomId }: UseKickPlayerOptions) {
  const kickPlayer = useCallback(async (
    targetUserId: string,
    reason?: string
  ): Promise<KickPlayerResult> => {
    try {
      // 使用 HTTP API
      const response = await axios.delete<KickPlayerResult>(
        `/api/campaigns/${roomId}/members/${targetUserId}`,
        { data: { reason } }
      );

      return response.data;
    } catch (error) {
      if (axios.isAxiosError(error)) {
        throw new Error(error.response?.data?.detail || "Failed to kick player");
      }
      throw error;
    }
  }, [roomId]);

  const kickPlayerViaSocket = useCallback((
    targetUserId: string,
    reason?: string,
    callback?: (result: KickPlayerResult) => void,
    errorCallback?: (error: string) => void
  ) => {
    if (!socket?.connected) {
      errorCallback?.("Socket not connected");
      return;
    }

    // 发送踢出事件
    socket.emit("kick_player", {
      room_id: roomId,
      target_user_id: targetUserId,
      reason
    });

    // 监听结果
    const handleSuccess = (result: KickPlayerResult) => {
      callback?.(result);
      socket.off("kick_success", handleSuccess);
      socket.off("error", handleError);
    };

    const handleError = (error: { message: string; code: string }) => {
      errorCallback?.(error.message);
      socket.off("kick_success", handleSuccess);
      socket.off("error", handleError);
    };

    socket.once("kick_success", handleSuccess);
    socket.once("error", handleError);
  }, [socket, roomId]);

  return {
    kickPlayer,
    kickPlayerViaSocket
  };
}
```

### 2. 被踢出监听 Hook

```typescript
// frontend/src/hooks/useKickDetection.ts
import { useEffect, useState, useCallback } from "react";
import { Socket } from "socket.io-client";
import { useNavigate } from "react-router-dom";

interface UseKickDetectionOptions {
  socket: Socket | null;
  onKicked?: (data: { room_id: string; reason?: string; kicked_by?: string }) => void;
}

export function useKickDetection({ socket, onKicked }: UseKickDetectionOptions) {
  const [kickedData, setKickedData] = useState<{
    room_id: string;
    reason?: string;
    kicked_by?: string;
  } | null>(null);
  const navigate = useNavigate();

  const handleKicked = useCallback((data: {
    room_id: string;
    reason?: string;
    kicked_by?: string;
  }) => {
    setKickedData(data);

    // 断开 Socket 连接
    socket?.disconnect();

    // 调用回调
    onKicked?.(data);

    // 导航到踢出页面
    navigate("/kicked", { state: data });
  }, [socket, onKicked, navigate]);

  useEffect(() => {
    if (!socket) return;

    socket.on("kicked_from_room", handleKicked);

    return () => {
      socket.off("kicked_from_room", handleKicked);
    };
  }, [socket, handleKicked]);

  return {
    kickedData,
    isKicked: kickedData !== null
  };
}
```

### 3. 玩家管理组件

```typescript
// frontend/src/components/room/PlayerManagement.tsx
import React, { useState } from "react";
import { useRoomState } from "@/contexts/RoomStateContext";
import { useKickPlayer } from "@/hooks/useKickPlayer";
import { KickConfirmDialog } from "./KickConfirmDialog";
import { PlayerItem } from "./PlayerItem";
import { toast } from "react-hot-toast";

export function PlayerManagement() {
  const { players, currentUserId, isKP } = useRoomState();
  const { kickPlayerViaSocket } = useKickPlayer({
    roomId: "" // 从 context 获取
  });

  const [selectedPlayer, setSelectedPlayer] = useState<{
    userId: string;
    username: string;
  } | null>(null);
  const [showKickDialog, setShowKickDialog] = useState(false);

  const handleKickClick = (userId: string, username: string) => {
    if (!isKP) {
      toast.error("Only Game Master can kick players");
      return;
    }

    setSelectedPlayer({ userId, username });
    setShowKickDialog(true);
  };

  const handleKickConfirm = async (reason?: string) => {
    if (!selectedPlayer) return;

    kickPlayerViaSocket(
      selectedPlayer.userId,
      reason,
      (result) => {
        toast.success(`Kicked ${result.target_username}`);
        setShowKickDialog(false);
        setSelectedPlayer(null);
      },
      (error) => {
        toast.error(`Failed to kick player: ${error}`);
      }
    );
  };

  return (
    <>
      <div className="player-management">
        <div className="player-list">
          {players.map(player => (
            <PlayerItem
              key={player.user_id}
              player={player}
              isCurrentUser={player.user_id === currentUserId}
              showKickButton={isKP && player.role !== "kp"}
              onKick={() => handleKickClick(player.user_id, player.username)}
            />
          ))}
        </div>
      </div>

      {showKickDialog && selectedPlayer && (
        <KickConfirmDialog
          username={selectedPlayer.username}
          onConfirm={handleKickConfirm}
          onCancel={() => {
            setShowKickDialog(false);
            setSelectedPlayer(null);
          }}
        />
      )}
    </>
  );
}
```

### 4. 踢出确认对话框

```typescript
// frontend/src/components/room/KickConfirmDialog.tsx
import React, { useState } from "react";
import { X } from "lucide-react";

interface KickConfirmDialogProps {
  username: string;
  onConfirm: (reason?: string) => void;
  onCancel: () => void;
}

export function KickConfirmDialog({ username, onConfirm, onCancel }: KickConfirmDialogProps) {
  const [reason, setReason] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleConfirm = async () => {
    setIsSubmitting(true);
    try {
      await onConfirm(reason || undefined);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onCancel} />

      <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md p-6">
        <button
          onClick={onCancel}
          className="absolute top-4 right-4 text-gray-400 hover:text-gray-600"
        >
          <X size={20} />
        </button>

        <div className="mb-4">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
            Kick Player
          </h3>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
            Are you sure you want to kick <strong>{username}</strong> from the room?
          </p>
        </div>

        <div className="mb-4">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Reason (optional)
          </label>
          <textarea
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder="Enter a reason for kicking this player..."
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
            rows={3}
          />
        </div>

        <div className="flex justify-end gap-3">
          <button
            onClick={onCancel}
            disabled={isSubmitting}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200 disabled:opacity-50 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600"
          >
            Cancel
          </button>
          <button
            onClick={handleConfirm}
            disabled={isSubmitting}
            className="px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-md hover:bg-red-700 disabled:opacity-50"
          >
            {isSubmitting ? "Kicking..." : "Kick Player"}
          </button>
        </div>
      </div>
    </div>
  );
}
```

### 5. 被踢出通知页面

```typescript
// frontend/src/pages/KickedPage.tsx
import React from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { AlertCircle } from "lucide-react";

export function KickedPage() {
  const location = useLocation();
  const navigate = useNavigate();
  const state = location.state as {
    room_id?: string;
    reason?: string;
    kicked_by?: string;
  } | null;

  const handleGoHome = () => {
    navigate("/");
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
      <div className="max-w-md w-full bg-white dark:bg-gray-800 rounded-lg shadow-lg p-8">
        <div className="flex justify-center mb-6">
          <div className="w-16 h-16 bg-red-100 dark:bg-red-900/30 rounded-full flex items-center justify-center">
            <AlertCircle className="w-8 h-8 text-red-600 dark:text-red-400" />
          </div>
        </div>

        <h1 className="text-2xl font-bold text-center text-gray-900 dark:text-white mb-4">
          You've Been Kicked
        </h1>

        <div className="space-y-4 text-center">
          {state?.kicked_by && (
            <p className="text-gray-600 dark:text-gray-400">
              Kicked by <strong>{state.kicked_by}</strong>
            </p>
          )}

          {state?.reason && (
            <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4">
              <p className="text-sm text-yellow-800 dark:text-yellow-200">
                <strong>Reason:</strong> {state.reason}
              </p>
            </div>
          )}

          {!state?.reason && (
            <p className="text-gray-600 dark:text-gray-400">
              You have been removed from the room.
            </p>
          )}
        </div>

        <button
          onClick={handleGoHome}
          className="w-full mt-6 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
        >
          Return to Home
        </button>
      </div>
    </div>
  );
}
```

### 6. 玩家列表项（带踢出按钮）

```typescript
// frontend/src/components/room/PlayerItem.tsx
import React from "react";
import { PlayerState } from "@/types/room";
import { MoreVertical, UserMinus } from "lucide-react";

interface PlayerItemProps {
  player: PlayerState;
  isCurrentUser: boolean;
  showKickButton?: boolean;
  onKick?: () => void;
}

export function PlayerItem({
  player,
  isCurrentUser,
  showKickButton = false,
  onKick
}: PlayerItemProps) {
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

      <div className="player-actions">
        {player.is_muted && (
          <span className="muted-badge" title="Muted">🔇</span>
        )}

        {!player.is_online && (
          <span className="offline-badge">Offline</span>
        )}

        {showKickButton && (
          <button
            onClick={onKick}
            className="kick-button"
            title="Kick player"
          >
            <UserMinus size={16} />
          </button>
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
| `backend/services/kick_service.py` | 创建 | 踢出玩家业务逻辑 |
| `backend/api/routes/campaign_members.py` | 创建/修改 | 踢出 API 路由 |
| `backend/handlers/kick_events.py` | 创建 | WebSocket 踢出事件处理 |
| `backend/models/campaign.py` | 修改 | 确保成员模型支持踢出 |
| `backend/middleware/auth.py` | 修改 | 添加权限验证 |

### 前端文件

| 文件路径 | 操作 | 说明 |
|---------|------|------|
| `frontend/src/hooks/useKickPlayer.ts` | 创建 | 踢出玩家 Hook |
| `frontend/src/hooks/useKickDetection.ts` | 创建 | 被踢出检测 Hook |
| `frontend/src/components/room/PlayerManagement.tsx` | 创建 | 玩家管理组件 |
| `frontend/src/components/room/KickConfirmDialog.tsx` | 创建 | 踢出确认对话框 |
| `frontend/src/components/room/PlayerItem.tsx` | 修改 | 添加踢出按钮 |
| `frontend/src/pages/KickedPage.tsx` | 创建 | 被踢出通知页面 |
| `frontend/src/router/index.tsx` | 修改 | 添加踢出页面路由 |

---

## 验收标准

### 功能验收

- [ ] KP 可以看到所有玩家的踢出按钮（除自己和其他 KP）
- [ ] KP 点击踢出按钮显示确认对话框
- [ ] KP 可以输入踢出原因（可选）
- [ ] 被踢出的玩家立即收到通知
- [ ] 被踢出的玩家被强制断开连接
- [ ] 其他玩家收到玩家被踢出的通知
- [ ] 普通玩家无法看到踢出按钮
- [ ] KP 无法踢出自己或其他 KP
- [ ] 被踢出的玩家跳转到通知页面

### 安全验收

- [ ] 只有 KP 可以执行踢出操作
- [ ] 后端验证权限（前端限制不可绕过）
- [ ] 踢出操作记录到审计日志
- [ ] 防止 CSRF 攻击

### UX 验收

- [ ] 踢出确认对话框清晰明了
- [ ] 被踢出页面显示原因和操作者
- [ ] 所有操作有加载状态反馈
- [ ] 错误信息友好清晰

---

## 参考文档

- [FastAPI 依赖注入](https://fastapi.tiangolo.com/tutorial/dependencies/)
- [Socket.IO 事件处理](https://socket.io/docs/v4/server-api/)
- [React Dialog 模式](https://headlessui.com/react/dialog)
- [权限检查最佳实践](https://owasp.org/www-project-top-ten/)
- [WebSocket 安全](https://socket.io/docs/v4/security/)

---

**创建日期**: 2026-02-06
**负责人**: 待分配
**审核人**: 待分配
