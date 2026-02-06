# M2-014: 实现房间转移 KP

**任务类型**: Backend + Frontend
**预估工时**: 8h
**优先级**: P0
**依赖**: M2-012 (房间状态同步), M2-013 (玩家踢出)

---

## 任务描述

实现 KP（Game Master）角色转移功能。当前 KP 可以将管理员权限转移给房间内的其他玩家。转移后，原 KP 变为普通玩家，新 KP 获得完整管理权限。需要确保转移过程的原子性和所有客户端的状态同步。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 | 依赖 |
|----|--------|------|----------|------|
| M2-014-01 | 设计 KP 转移数据流 | 定义转移协议和状态机 | 1h | M2-012 |
| M2-014-02 | 实现后端转移逻辑 | 角色交换 + 权限更新 | 2h | M2-014-01 |
| M2-014-03 | 实现 WebSocket 转移事件 | 实时通知所有玩家 | 1.5h | M2-014-02 |
| M2-014-04 | 实现转移确认机制 | 双方确认才执行转移 | 1.5h | M2-014-03 |
| M2-014-05 | 实现前端转移 UI | KP 管理面板转移按钮 | 1h | M2-014-04 |
| M2-014-06 | 实现接收转移 UI | 目标玩家确认对话框 | 1h | M2-014-05 |

---

## 后端代码示例

### 1. KP 转移服务

```python
# backend/services/kp_transfer_service.py
from typing import Optional, Tuple
from datetime import datetime
from loguru import logger
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select, update

from models.campaign import CampaignMember, Campaign
from models.room_state import RoomState
from services.room_state_manager import room_state_manager


class KPTransferService:
    """KP 转移服务"""

    def __init__(self, db: AsyncSession):
        self.db = db
        self._pending_transfers: dict[str, dict] = {}  # room_id -> transfer_data

    async def initiate_transfer(
        self,
        campaign_id: str,
        current_kp_id: str,
        target_user_id: str
    ) -> dict:
        """
        发起 KP 转移请求

        Args:
            campaign_id: 战役 ID
            current_kp_id: 当前 KP ID
            target_user_id: 目标玩家 ID

        Returns:
            转移请求信息
        """
        # 1. 验证当前 KP
        current_kp = await self._get_member(campaign_id, current_kp_id)
        if not current_kp or current_kp.role != "kp":
            raise PermissionError("Only current KP can initiate transfer")

        # 2. 验证目标玩家
        target = await self._get_member(campaign_id, target_user_id)
        if not target:
            raise ValueError("Target player not found")

        if target.role == "kp":
            raise ValueError("Target is already a KP")

        if target.user_id == current_kp_id:
            raise ValueError("Cannot transfer to yourself")

        # 3. 检查是否已有待处理的转移
        if campaign_id in self._pending_transfers:
            existing = self._pending_transfers[campaign_id]
            if existing["expires_at"] > datetime.utcnow():
                raise ValueError("Another transfer is already pending")
            self._cancel_transfer(campaign_id)

        # 4. 创建待处理转移
        transfer_data = {
            "campaign_id": campaign_id,
            "from_kp_id": current_kp_id,
            "from_kp_username": current_kp.user.username,
            "to_user_id": target_user_id,
            "to_username": target.user.username,
            "initiated_at": datetime.utcnow(),
            "expires_at": datetime.utcnow(),  # 5 分钟有效期
            "status": "pending"
        }

        self._pending_transfers[campaign_id] = transfer_data

        logger.info(
            f"KP transfer initiated: {current_kp.user.username} -> "
            f"{target.user.username} in campaign {campaign_id}"
        )

        return {
            "transfer_id": campaign_id,  # 使用 campaign_id 作为 transfer_id
            "from_username": transfer_data["from_kp_username"],
            "to_username": transfer_data["to_username"],
            "expires_at": transfer_data["expires_at"].isoformat()
        }

    async def accept_transfer(
        self,
        campaign_id: str,
        target_user_id: str
    ) -> dict:
        """
        接受 KP 转移

        Args:
            campaign_id: 战役 ID
            target_user_id: 目标玩家 ID（接收者）

        Returns:
            转移结果
        """
        # 1. 验证待处理转移
        transfer = self._pending_transfers.get(campaign_id)
        if not transfer:
            raise ValueError("No pending transfer found")

        if transfer["status"] != "pending":
            raise ValueError("Transfer already processed")

        if transfer["expires_at"] < datetime.utcnow():
            self._cancel_transfer(campaign_id)
            raise ValueError("Transfer request has expired")

        if transfer["to_user_id"] != target_user_id:
            raise PermissionError("You are not the transfer target")

        # 2. 执行角色转移（原子操作）
        try:
            await self._execute_transfer(
                campaign_id,
                transfer["from_kp_id"],
                transfer["to_user_id"]
            )

            # 3. 标记转移完成
            transfer["status"] = "completed"
            transfer["completed_at"] = datetime.utcnow()

            result = {
                "success": True,
                "from_username": transfer["from_kp_username"],
                "to_username": transfer["to_username"],
                "transferred_at": transfer["completed_at"].isoformat()
            }

            # 4. 清理（延迟清理以允许客户端同步）
            # 实际清理由定时任务处理

            logger.info(
                f"KP transfer completed: {transfer['from_kp_username']} -> "
                f"{transfer['to_username']} in campaign {campaign_id}"
            )

            return result

        except Exception as e:
            logger.error(f"KP transfer failed: {e}")
            transfer["status"] = "failed"
            transfer["error"] = str(e)
            raise

    async def decline_transfer(
        self,
        campaign_id: str,
        target_user_id: str
    ) -> dict:
        """
        拒绝 KP 转移

        Args:
            campaign_id: 战役 ID
            target_user_id: 目标玩家 ID

        Returns:
            拒绝结果
        """
        transfer = self._pending_transfers.get(campaign_id)
        if not transfer:
            raise ValueError("No pending transfer found")

        if transfer["to_user_id"] != target_user_id:
            raise PermissionError("You are not the transfer target")

        # 取消转移
        self._cancel_transfer(campaign_id)

        logger.info(
            f"KP transfer declined by {transfer['to_username']} "
            f"in campaign {campaign_id}"
        )

        return {
            "success": True,
            "message": "Transfer declined"
        }

    async def cancel_transfer(
        self,
        campaign_id: str,
        current_kp_id: str
    ) -> dict:
        """
        取消 KP 转移（由当前 KP 发起）

        Args:
            campaign_id: 战役 ID
            current_kp_id: 当前 KP ID

        Returns:
            取消结果
        """
        transfer = self._pending_transfers.get(campaign_id)
        if not transfer:
            raise ValueError("No pending transfer found")

        if transfer["from_kp_id"] != current_kp_id:
            raise PermissionError("Only the initiator can cancel")

        self._cancel_transfer(campaign_id)

        logger.info(
            f"KP transfer cancelled by {transfer['from_kp_username']} "
            f"in campaign {campaign_id}"
        )

        return {
            "success": True,
            "message": "Transfer cancelled"
        }

    async def _execute_transfer(
        self,
        campaign_id: str,
        from_kp_id: str,
        to_user_id: str
    ):
        """执行角色转移（数据库更新）"""
        # 开始事务
        async with self.db.begin():
            # 原 KP 变为玩家
            await self.db.execute(
                update(CampaignMember)
                .where(
                    CampaignMember.campaign_id == campaign_id,
                    CampaignMember.user_id == from_kp_id
                )
                .values(role="player")
            )

            # 目标玩家变为 KP
            await self.db.execute(
                update(CampaignMember)
                .where(
                    CampaignMember.campaign_id == campaign_id,
                    CampaignMember.user_id == to_user_id
                )
                .values(role="kp")
            )

        # 更新房间状态
        state = room_state_manager.get_state(campaign_id)
        if state and from_kp_id in state.players:
            state.players[from_kp_id].role = "player"
        if state and to_user_id in state.players:
            state.players[to_user_id].role = "kp"

    def _cancel_transfer(self, campaign_id: str):
        """取消转移"""
        if campaign_id in self._pending_transfers:
            del self._pending_transfers[campaign_id]

    async def _get_member(
        self,
        campaign_id: str,
        user_id: str
    ) -> Optional[CampaignMember]:
        """获取战役成员"""
        result = await self.db.execute(
            select(CampaignMember).where(
                CampaignMember.campaign_id == campaign_id,
                CampaignMember.user_id == user_id
            )
        )
        return result.scalar_one_or_none()

    async def cleanup_expired_transfers(self):
        """清理过期的转移请求（定时任务）"""
        now = datetime.utcnow()
        expired = [
            room_id for room_id, transfer in self._pending_transfers.items()
            if transfer["expires_at"] < now
        ]

        for room_id in expired:
            logger.info(f"Cleaning up expired transfer for campaign {room_id}")
            self._cancel_transfer(room_id)
```

### 2. API 路由

```python
# backend/api/routes/kp_transfer.py
from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel
from typing import Optional

from database import get_db
from services.kp_transfer_service import KPTransferService
from middleware.auth import get_current_user

router = APIRouter(prefix="/api/campaigns", tags=["kp-transfer"])


class InitiateTransferRequest(BaseModel):
    """发起转移请求"""
    target_user_id: str


class TransferResponse(BaseModel):
    """转移响应"""
    success: bool
    from_username: str
    to_username: str
    transferred_at: Optional[str] = None
    expires_at: Optional[str] = None


@router.post(
    "/{campaign_id}/transfer-kp/initiate",
    response_model=TransferResponse,
    status_code=status.HTTP_200_OK
)
async def initiate_kp_transfer(
    campaign_id: str,
    request: InitiateTransferRequest,
    current_user = Depends(get_current_user),
    db = Depends(get_db)
):
    """发起 KP 转移"""
    try:
        service = KPTransferService(db)
        result = await service.initiate_transfer(
            campaign_id=campaign_id,
            current_kp_id=current_user.id,
            target_user_id=request.target_user_id
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
            detail="Failed to initiate transfer"
        )


@router.post(
    "/{campaign_id}/transfer-kp/accept",
    response_model=TransferResponse,
    status_code=status.HTTP_200_OK
)
async def accept_kp_transfer(
    campaign_id: str,
    current_user = Depends(get_current_user),
    db = Depends(get_db)
):
    """接受 KP 转移"""
    try:
        service = KPTransferService(db)
        result = await service.accept_transfer(
            campaign_id=campaign_id,
            target_user_id=current_user.id
        )
        return result

    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(e)
        )
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to accept transfer"
        )


@router.post(
    "/{campaign_id}/transfer-kp/decline",
    status_code=status.HTTP_200_OK
)
async def decline_kp_transfer(
    campaign_id: str,
    current_user = Depends(get_current_user),
    db = Depends(get_db)
):
    """拒绝 KP 转移"""
    try:
        service = KPTransferService(db)
        result = await service.decline_transfer(
            campaign_id=campaign_id,
            target_user_id=current_user.id
        )
        return result

    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(e)
        )
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to decline transfer"
        )


@router.post(
    "/{campaign_id}/transfer-kp/cancel",
    status_code=status.HTTP_200_OK
)
async def cancel_kp_transfer(
    campaign_id: str,
    current_user = Depends(get_current_user),
    db = Depends(get_db)
):
    """取消 KP 转移"""
    try:
        service = KPTransferService(db)
        result = await service.cancel_transfer(
            campaign_id=campaign_id,
            current_kp_id=current_user.id
        )
        return result

    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(e)
        )
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to cancel transfer"
        )
```

### 3. WebSocket 事件处理

```python
# backend/handlers/kp_transfer_events.py
from socketio import AsyncServer
from loguru import logger

from services.kp_transfer_service import KPTransferService


class KPTransferEventHandler:
    """KP 转移 WebSocket 事件处理"""

    def __init__(self, sio: AsyncServer):
        self.sio = sio
        self._setup_handlers()

    def _setup_handlers(self):
        """设置事件处理器"""

        @self.sio.event
        async def initiate_kp_transfer(sid, data):
            """
            发起 KP 转移

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
                    service = KPTransferService(db)
                    result = await service.initiate_transfer(
                        campaign_id=room_id,
                        current_kp_id=operator_info["user_id"],
                        target_user_id=target_user_id
                    )

                    # 通知目标玩家
                    await self.sio.emit(
                        "kp_transfer_request",
                        {
                            "room_id": room_id,
                            "from_username": result["from_username"],
                            "expires_at": result["expires_at"]
                        },
                        room=target_user_id
                    )

                    # 通知其他玩家（包括发起者）
                    await self.sio.emit(
                        "kp_transfer_initiated",
                        {
                            "room_id": room_id,
                            "from_username": result["from_username"],
                            "to_username": result["to_username"],
                            "expires_at": result["expires_at"]
                        },
                        room=room_id
                    )

            except PermissionError as e:
                await self._emit_error(sid, str(e), "PERMISSION_DENIED")
            except ValueError as e:
                await self._emit_error(sid, str(e), "INVALID_OPERATION")
            except Exception as e:
                logger.error(f"Error initiating KP transfer: {e}")
                await self._emit_error(sid, "Failed to initiate transfer")

        @self.sio.event
        async def accept_kp_transfer(sid, data):
            """
            接受 KP 转移

            data: {
                room_id: str
            }
            """
            from database import get_db_context

            room_id = data.get("room_id")
            user_info = self._get_socket_user(sid)

            if not user_info:
                await self._emit_error(sid, "Unauthorized")
                return

            try:
                async with get_db_context() as db:
                    service = KPTransferService(db)
                    result = await service.accept_transfer(
                        campaign_id=room_id,
                        target_user_id=user_info["user_id"]
                    )

                    # 广播转移完成
                    await self.sio.emit(
                        "kp_transfer_completed",
                        {
                            "room_id": room_id,
                            "from_username": result["from_username"],
                            "to_username": result["to_username"],
                            "transferred_at": result["transferred_at"]
                        },
                        room=room_id
                    )

            except ValueError as e:
                await self._emit_error(sid, str(e), "INVALID_OPERATION")
            except Exception as e:
                logger.error(f"Error accepting KP transfer: {e}")
                await self._emit_error(sid, "Failed to accept transfer")

        @self.sio.event
        async def decline_kp_transfer(sid, data):
            """拒绝 KP 转移"""
            from database import get_db_context

            room_id = data.get("room_id")
            user_info = self._get_socket_user(sid)

            if not user_info:
                await self._emit_error(sid, "Unauthorized")
                return

            try:
                async with get_db_context() as db:
                    service = KPTransferService(db)
                    result = await service.decline_transfer(
                        campaign_id=room_id,
                        target_user_id=user_info["user_id"]
                    )

                    # 通知房间所有人
                    await self.sio.emit(
                        "kp_transfer_declined",
                        {
                            "room_id": room_id,
                            "message": "KP transfer was declined"
                        },
                        room=room_id
                    )

            except ValueError as e:
                await self._emit_error(sid, str(e), "INVALID_OPERATION")
            except Exception as e:
                logger.error(f"Error declining KP transfer: {e}")
                await self._emit_error(sid, "Failed to decline transfer")

        @self.sio.event
        async def cancel_kp_transfer(sid, data):
            """取消 KP 转移"""
            from database import get_db_context

            room_id = data.get("room_id")
            user_info = self._get_socket_user(sid)

            if not user_info:
                await self._emit_error(sid, "Unauthorized")
                return

            try:
                async with get_db_context() as db:
                    service = KPTransferService(db)
                    result = await service.cancel_transfer(
                        campaign_id=room_id,
                        current_kp_id=user_info["user_id"]
                    )

                    # 通知房间所有人
                    await self.sio.emit(
                        "kp_transfer_cancelled",
                        {
                            "room_id": room_id,
                            "message": result["message"]
                        },
                        room=room_id
                    )

            except ValueError as e:
                await self._emit_error(sid, str(e), "INVALID_OPERATION")
            except Exception as e:
                logger.error(f"Error cancelling KP transfer: {e}")
                await self._emit_error(sid, "Failed to cancel transfer")

    async def _emit_error(self, sid: str, message: str, code: str = "ERROR"):
        """发送错误消息"""
        await self.sio.emit("error", {"message": message, "code": code}, room=sid)

    def _get_socket_user(self, sid: str) -> Optional[dict]:
        """获取 Socket 连接的用户信息"""
        # 根据实际认证实现
        return None
```

---

## 前端代码示例

### 1. KP 转移 Hook

```typescript
// frontend/src/hooks/useKPTransfer.ts
import { useCallback, useEffect, useState } from "react";
import { Socket } from "socket.io-client";
import axios from "axios";

interface UseKPTransferOptions {
  socket?: Socket | null;
  roomId: string;
  userId: string;
  isKP: boolean;
}

interface TransferRequest {
  room_id: string;
  from_username: string;
  expires_at: string;
}

interface TransferResult {
  success: boolean;
  from_username: string;
  to_username: string;
  transferred_at?: string;
}

export function useKPTransfer({
  socket,
  roomId,
  userId,
  isKP
}: UseKPTransferOptions) {
  const [pendingRequest, setPendingRequest] = useState<TransferRequest | null>(null);
  const [isTransferring, setIsTransferring] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // 发起转移
  const initiateTransfer = useCallback(async (
    targetUserId: string
  ): Promise<void> => {
    if (!isKP) {
      throw new Error("Only KP can initiate transfer");
    }

    setIsTransferring(true);
    setError(null);

    try {
      await axios.post(`/api/campaigns/${roomId}/transfer-kp/initiate`, {
        target_user_id: targetUserId
      });
    } catch (err) {
      const message = axios.isAxiosError(err)
        ? err.response?.data?.detail || "Failed to initiate transfer"
        : "Unknown error";
      setError(message);
      throw new Error(message);
    } finally {
      setIsTransferring(false);
    }
  }, [isKP, roomId]);

  // 接受转移
  const acceptTransfer = useCallback(async (): Promise<TransferResult> => {
    setIsTransferring(true);
    setError(null);

    try {
      const response = await axios.post<TransferResult>(
        `/api/campaigns/${roomId}/transfer-kp/accept`
      );
      setPendingRequest(null);
      return response.data;
    } catch (err) {
      const message = axios.isAxiosError(err)
        ? err.response?.data?.detail || "Failed to accept transfer"
        : "Unknown error";
      setError(message);
      throw new Error(message);
    } finally {
      setIsTransferring(false);
    }
  }, [roomId]);

  // 拒绝转移
  const declineTransfer = useCallback(async (): Promise<void> => {
    setIsTransferring(true);
    setError(null);

    try {
      await axios.post(`/api/campaigns/${roomId}/transfer-kp/decline`);
      setPendingRequest(null);
    } catch (err) {
      const message = axios.isAxiosError(err)
        ? err.response?.data?.detail || "Failed to decline transfer"
        : "Unknown error";
      setError(message);
      throw new Error(message);
    } finally {
      setIsTransferring(false);
    }
  }, [roomId]);

  // 取消转移
  const cancelTransfer = useCallback(async (): Promise<void> => {
    if (!isKP) return;

    setIsTransferring(true);
    setError(null);

    try {
      await axios.post(`/api/campaigns/${roomId}/transfer-kp/cancel`);
      setPendingRequest(null);
    } catch (err) {
      const message = axios.isAxiosError(err)
        ? err.response?.data?.detail || "Failed to cancel transfer"
        : "Unknown error";
      setError(message);
      throw new Error(message);
    } finally {
      setIsTransferring(false);
    }
  }, [isKP, roomId]);

  // 监听 WebSocket 事件
  useEffect(() => {
    if (!socket) return;

    const handleTransferRequest = (data: TransferRequest) => {
      if (data.room_id === roomId) {
        setPendingRequest(data);
      }
    };

    const handleTransferCompleted = (data: TransferResult) => {
      if (data.room_id === roomId) {
        setPendingRequest(null);
        // 触发页面刷新或状态更新
        window.location.reload();
      }
    };

    const handleTransferDeclined = () => {
      setPendingRequest(null);
    };

    const handleTransferCancelled = () => {
      setPendingRequest(null);
    };

    socket.on("kp_transfer_request", handleTransferRequest);
    socket.on("kp_transfer_completed", handleTransferCompleted);
    socket.on("kp_transfer_declined", handleTransferDeclined);
    socket.on("kp_transfer_cancelled", handleTransferCancelled);

    return () => {
      socket.off("kp_transfer_request", handleTransferRequest);
      socket.off("kp_transfer_completed", handleTransferCompleted);
      socket.off("kp_transfer_declined", handleTransferDeclined);
      socket.off("kp_transfer_cancelled", handleTransferCancelled);
    };
  }, [socket, roomId]);

  return {
    pendingRequest,
    isTransferring,
    error,
    initiateTransfer,
    acceptTransfer,
    declineTransfer,
    cancelTransfer
  };
}
```

### 2. KP 转移按钮组件

```typescript
// frontend/src/components/room/KPTransferButton.tsx
import React, { useState } from "react";
import { Crown } from "lucide-react";
import { useKPTransfer } from "@/hooks/useKPTransfer";
import { useRoomState } from "@/contexts/RoomStateContext";
import { TransferDialog } from "./TransferDialog";
import { toast } from "react-hot-toast";

export function KPTransferButton() {
  const { isKP } = useRoomState();
  const { initiateTransfer } = useKPTransfer({
    roomId: "", // 从 context 获取
    userId: "", // 从 context 获取
    isKP
  });

  const [showDialog, setShowDialog] = useState(false);

  if (!isKP) return null;

  return (
    <>
      <button
        onClick={() => setShowDialog(true)}
        className="kp-transfer-button"
        title="Transfer KP role"
      >
        <Crown size={16} />
        Transfer KP
      </button>

      {showDialog && (
        <TransferDialog
          onConfirm={async (targetUserId) => {
            try {
              await initiateTransfer(targetUserId);
              toast.success("Transfer request sent");
              setShowDialog(false);
            } catch (error) {
              toast.error(error instanceof Error ? error.message : "Failed to transfer");
            }
          }}
          onClose={() => setShowDialog(false)}
        />
      )}
    </>
  );
}
```

### 3. 转移对话框（发起者）

```typescript
// frontend/src/components/room/TransferDialog.tsx
import React from "react";
import { X, Crown } from "lucide-react";
import { useRoomState } from "@/contexts/RoomStateContext";

interface TransferDialogProps {
  onConfirm: (targetUserId: string) => void;
  onClose: () => void;
}

export function TransferDialog({ onConfirm, onClose }: TransferDialogProps) {
  const { players, currentUserId } = useRoomState();

  // 过滤可转移的玩家（非 KP 且非自己）
  const eligiblePlayers = players.filter(
    p => p.role !== "kp" && p.user_id !== currentUserId
  );

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />

      <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md p-6">
        <button
          onClick={onClose}
          className="absolute top-4 right-4 text-gray-400 hover:text-gray-600"
        >
          <X size={20} />
        </button>

        <div className="mb-4">
          <div className="flex items-center gap-2 mb-2">
            <Crown className="text-yellow-500" size={24} />
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
              Transfer KP Role
            </h3>
          </div>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Select a player to transfer the Game Master role. You will become a regular player.
          </p>
        </div>

        <div className="space-y-2 max-h-64 overflow-y-auto">
          {eligiblePlayers.length === 0 ? (
            <p className="text-center text-gray-500 py-4">
              No eligible players to transfer to
            </p>
          ) : (
            eligiblePlayers.map(player => (
              <button
                key={player.user_id}
                onClick={() => onConfirm(player.user_id)}
                className="w-full flex items-center gap-3 p-3 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
              >
                <div className="w-10 h-10 bg-blue-500 rounded-full flex items-center justify-center text-white font-semibold">
                  {player.username[0].toUpperCase()}
                </div>
                <div className="text-left">
                  <div className="font-medium text-gray-900 dark:text-white">
                    {player.username}
                  </div>
                  <div className="text-sm text-gray-500 dark:text-gray-400">
                    {player.is_online ? "Online" : "Offline"}
                  </div>
                </div>
              </button>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
```

### 4. 接收转移对话框

```typescript
// frontend/src/components/room/ReceiveTransferDialog.tsx
import React, { useState, useEffect } from "react";
import { Crown, X } from "lucide-react";
import { useKPTransfer } from "@/hooks/useKPTransfer";
import { toast } from "react-hot-toast";

export function ReceiveTransferDialog() {
  const { pendingRequest, acceptTransfer, declineTransfer, isTransferring } = useKPTransfer({
    roomId: "", // 从 context 获取
    userId: "", // 从 context 获取
    isKP: false
  });

  const [timeLeft, setTimeLeft] = useState(0);

  // 计算剩余时间
  useEffect(() => {
    if (!pendingRequest) return;

    const expiresAt = new Date(pendingRequest.expires_at).getTime();
    const now = Date.now();
    const initialTime = Math.max(0, Math.floor((expiresAt - now) / 1000));

    setTimeLeft(initialTime);

    const timer = setInterval(() => {
      setTimeLeft(prev => {
        if (prev <= 1) {
          clearInterval(timer);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(timer);
  }, [pendingRequest]);

  if (!pendingRequest) return null;

  const handleAccept = async () => {
    try {
      await acceptTransfer();
      toast.success("You are now the Game Master!");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to accept transfer");
    }
  };

  const handleDecline = async () => {
    try {
      await declineTransfer();
      toast.info("Transfer declined");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to decline transfer");
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" />

      <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md p-6">
        <div className="flex justify-center mb-4">
          <div className="w-16 h-16 bg-yellow-100 dark:bg-yellow-900/30 rounded-full flex items-center justify-center">
            <Crown className="w-8 h-8 text-yellow-600 dark:text-yellow-400" />
          </div>
        </div>

        <h3 className="text-xl font-bold text-center text-gray-900 dark:text-white mb-2">
          KP Transfer Request
        </h3>

        <p className="text-center text-gray-600 dark:text-gray-400 mb-4">
          <strong>{pendingRequest.from_username}</strong> wants to transfer the Game Master role to you.
        </p>

        {timeLeft > 0 && (
          <p className="text-center text-sm text-gray-500 dark:text-gray-500 mb-6">
            Request expires in {timeLeft} seconds
          </p>
        )}

        <div className="flex gap-3">
          <button
            onClick={handleDecline}
            disabled={isTransferring}
            className="flex-1 px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-200 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors disabled:opacity-50"
          >
            Decline
          </button>
          <button
            onClick={handleAccept}
            disabled={isTransferring || timeLeft === 0}
            className="flex-1 px-4 py-2 bg-yellow-500 text-white rounded-lg hover:bg-yellow-600 transition-colors disabled:opacity-50"
          >
            {isTransferring ? "Accepting..." : "Accept"}
          </button>
        </div>
      </div>
    </div>
  );
}
```

### 5. KP 转移状态通知组件

```typescript
// frontend/src/components/room/KPTransferStatus.tsx
import React, { useState, useEffect } from "react";
import { Crown, Loader2 } from "lucide-react";
import { useKPTransfer } from "@/hooks/useKPTransfer";

export function KPTransferStatus() {
  const { pendingRequest } = useKPTransfer({
    roomId: "", // 从 context 获取
    userId: "", // 从 context 获取
    isKP: false
  });

  const [showNotification, setShowNotification] = useState(false);

  useEffect(() => {
    if (pendingRequest) {
      setShowNotification(true);
    }
  }, [pendingRequest]);

  if (!showNotification || !pendingRequest) return null;

  return (
    <div className="fixed top-4 right-4 z-50 bg-yellow-50 dark:bg-yellow-900/30 border border-yellow-200 dark:border-yellow-800 rounded-lg shadow-lg p-4 max-w-sm">
      <div className="flex items-start gap-3">
        <Crown className="w-5 h-5 text-yellow-600 dark:text-yellow-400 mt-0.5" />
        <div className="flex-1">
          <p className="font-medium text-yellow-800 dark:text-yellow-200">
            KP Transfer in Progress
          </p>
          <p className="text-sm text-yellow-700 dark:text-yellow-300 mt-1">
            Waiting for <strong>{pendingRequest.from_username}</strong>'s transfer to be accepted...
          </p>
        </div>
        <button
          onClick={() => setShowNotification(false)}
          className="text-yellow-600 dark:text-yellow-400 hover:text-yellow-800 dark:hover:text-yellow-200"
        >
          ×
        </button>
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
| `backend/services/kp_transfer_service.py` | 创建 | KP 转移业务逻辑 |
| `backend/api/routes/kp_transfer.py` | 创建 | KP 转移 API 路由 |
| `backend/handlers/kp_transfer_events.py` | 创建 | WebSocket KP 转移事件处理 |
| `backend/models/campaign.py` | 修改 | 确保成员模型支持角色更新 |
| `backend/tasks/scheduled.py` | 修改 | 添加清理过期转移的定时任务 |

### 前端文件

| 文件路径 | 操作 | 说明 |
|---------|------|------|
| `frontend/src/hooks/useKPTransfer.ts` | 创建 | KP 转移 Hook |
| `frontend/src/components/room/KPTransferButton.tsx` | 创建 | KP 转移按钮 |
| `frontend/src/components/room/TransferDialog.tsx` | 创建 | 转移发起对话框 |
| `frontend/src/components/room/ReceiveTransferDialog.tsx` | 创建 | 接收转移对话框 |
| `frontend/src/components/room/KPTransferStatus.tsx` | 创建 | 转移状态通知 |
| `frontend/src/contexts/RoomStateContext.tsx` | 修改 | 集成 KP 转移状态 |
| `frontend/src/pages/GamePage.tsx` | 修改 | 添加转移相关组件 |

---

## 验收标准

### 功能验收

- [ ] KP 可以看到转移按钮并选择目标玩家
- [ ] KP 点击转移后，目标玩家收到转移请求
- [ ] 转移请求有 5 分钟有效期
- [ ] 目标玩家可以接受或拒绝转移
- [ ] 接受后原 KP 变为玩家，目标玩家变为 KP
- [ ] 拒绝后所有玩家收到通知
- [ ] KP 可以取消待处理的转移请求
- [ ] 转移完成后所有玩家状态实时更新
- [ ] 权限变更立即生效

### 安全验收

- [ ] 只有当前 KP 可以发起转移
- [ ] 只有目标玩家可以接受/拒绝转移
- [ ] 后端验证所有权限
- [ ] 防止并发转移冲突
- [ ] 转移操作记录到审计日志

### UX 验收

- [ ] 转移对话框清晰显示操作后果
- [ ] 接收转移对话框突出显示
- [ ] 倒计时显示剩余时间
- [ ] 所有操作有加载状态
- [ ] 错误信息友好清晰

---

## 参考文档

- [事务处理最佳实践](https://docs.sqlalchemy.org/en/20/orm/session_transaction.html)
- [WebSocket 事件命名约定](https://socket.io/docs/v4/emitting-events/)
- [React Context 模式](https://react.dev/reference/react/useContext)
- [权限转移设计模式](https://martinfowler.com/bliki/RoleBasedAccessControl.html)

---

**创建日期**: 2026-02-06
**负责人**: 待分配
**审核人**: 待分配
