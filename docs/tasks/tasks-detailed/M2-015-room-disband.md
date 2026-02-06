# M2-015: 实现房间解散

**任务类型**: Backend + Frontend
**预估工时**: 6h
**优先级**: P0
**依赖**: M2-012 (房间状态同步), M2-014 (转移 KP)

---

## 任务描述

实现房间（战役）解散功能。KP 可以永久关闭房间，所有玩家将被踢出，房间数据将被标记为已删除（软删除）。需要确保解散过程的安全性和所有客户端的通知。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 | 依赖 |
|----|--------|------|----------|------|
| M2-015-01 | 设计房间软删除机制 | 数据库 deleted_at 字段 | 1h | M2-012 |
| M2-015-02 | 实现后端解散 API | DELETE /campaigns/:id | 1.5h | M2-015-01 |
| M2-015-03 | 实现 WebSocket 解散事件 | 广播解散通知 | 1h | M2-015-02 |
| M2-015-04 | 实现玩家清理逻辑 | 移除所有成员连接 | 1h | M2-015-03 |
| M2-015-05 | 实现前端解散 UI | KP 管理面板解散按钮 | 1h | M2-015-04 |
| M2-015-06 | 实现解散确认对话框 | 二次确认防止误操作 | 0.5h | M2-015-05 |

---

## 后端代码示例

### 1. 房间解散服务

```python
# backend/services/disband_service.py
from typing import Optional
from datetime import datetime
from loguru import logger
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select, update, delete

from models.campaign import Campaign, CampaignMember
from models.room_state import RoomState
from services.room_state_manager import room_state_manager


class DisbandService:
    """房间解散服务"""

    def __init__(self, db: AsyncSession):
        self.db = db

    async def disband_campaign(
        self,
        campaign_id: str,
        operator_id: str
    ) -> dict:
        """
        解散战役

        Args:
            campaign_id: 战役 ID
            operator_id: 操作者 ID（必须是 KP）

        Returns:
            解散结果
        """
        # 1. 验证战役存在
        campaign = await self._get_campaign(campaign_id)
        if not campaign:
            raise ValueError("Campaign not found")

        if campaign.is_deleted:
            raise ValueError("Campaign already disbanded")

        # 2. 验证操作者是 KP
        operator_member = await self._get_member(campaign_id, operator_id)
        if not operator_member or operator_member.role != "kp":
            raise PermissionError("Only KP can disband the campaign")

        # 3. 获取所有成员（用于通知）
        members = await self._get_all_members(campaign_id)
        member_ids = [m.user_id for m in members]

        # 4. 软删除战役
        await self._soft_delete_campaign(campaign_id)

        # 5. 删除所有成员关系
        await self._remove_all_members(campaign_id)

        # 6. 清理房间状态
        await room_state_manager.remove_player(campaign_id, "")  # 特殊处理

        # 7. 记录解散操作
        await self._log_disband_action(campaign_id, operator_id)

        logger.info(
            f"Campaign {campaign_id} disbanded by {operator_id}. "
            f"Affected members: {len(member_ids)}"
        )

        return {
            "success": True,
            "campaign_id": campaign_id,
            "campaign_name": campaign.name,
            "disbanded_by": operator_member.user.username,
            "member_count": len(members),
            "disbanded_at": datetime.utcnow().isoformat()
        }

    async def _get_campaign(self, campaign_id: str) -> Optional[Campaign]:
        """获取战役"""
        result = await self.db.execute(
            select(Campaign).where(
                Campaign.id == campaign_id
            )
        )
        return result.scalar_one_or_none()

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

    async def _get_all_members(self, campaign_id: str) -> list[CampaignMember]:
        """获取所有成员"""
        result = await self.db.execute(
            select(CampaignMember).where(
                CampaignMember.campaign_id == campaign_id
            )
        )
        return list(result.scalars().all())

    async def _soft_delete_campaign(self, campaign_id: str):
        """软删除战役"""
        await self.db.execute(
            update(Campaign)
            .where(Campaign.id == campaign_id)
            .values(
                is_deleted=True,
                deleted_at=datetime.utcnow()
            )
        )
        await self.db.commit()

    async def _remove_all_members(self, campaign_id: str):
        """移除所有成员"""
        await self.db.execute(
            delete(CampaignMember).where(
                CampaignMember.campaign_id == campaign_id
            )
        )
        await self.db.commit()

    async def _log_disband_action(self, campaign_id: str, operator_id: str):
        """记录解散操作"""
        # 实现审计日志
        pass

    async def permanently_delete_campaign(
        self,
        campaign_id: str,
        operator_id: str,
        confirm_name: str
    ) -> dict:
        """
        永久删除战役（危险操作，需要确认战役名称）

        Args:
            campaign_id: 战役 ID
            operator_id: 操作者 ID
            confirm_name: 确认的战役名称

        Returns:
            删除结果
        """
        # 1. 验证战役存在
        campaign = await self._get_campaign(campaign_id)
        if not campaign:
            raise ValueError("Campaign not found")

        # 2. 验证名称确认
        if campaign.name != confirm_name:
            raise ValueError("Campaign name confirmation does not match")

        # 3. 验证权限
        operator_member = await self._get_member(campaign_id, operator_id)
        if not operator_member or operator_member.role != "kp":
            raise PermissionError("Only KP can permanently delete the campaign")

        # 4. 硬删除战役
        await self.db.execute(
            delete(Campaign).where(Campaign.id == campaign_id)
        )
        await self.db.commit()

        logger.info(f"Campaign {campaign_id} permanently deleted by {operator_id}")

        return {
            "success": True,
            "campaign_id": campaign_id,
            "deleted_at": datetime.utcnow().isoformat()
        }
```

### 2. API 路由

```python
# backend/api/routes/campaign_disband.py
from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel
from typing import Optional

from database import get_db
from services.disband_service import DisbandService
from middleware.auth import get_current_user

router = APIRouter(prefix="/api/campaigns", tags=["disband"])


class DisbandCampaignResponse(BaseModel):
    """解散战役响应"""
    success: bool
    campaign_id: str
    campaign_name: str
    disbanded_by: str
    member_count: int
    disbanded_at: str


class PermanentDeleteRequest(BaseModel):
    """永久删除请求"""
    confirm_name: str


@router.delete(
    "/{campaign_id}",
    response_model=DisbandCampaignResponse,
    status_code=status.HTTP_200_OK
)
async def disband_campaign(
    campaign_id: str,
    current_user = Depends(get_current_user),
    db = Depends(get_db)
):
    """
    解散战役（软删除）

    需要权限：KP
    """
    try:
        service = DisbandService(db)
        result = await service.disband_campaign(
            campaign_id=campaign_id,
            operator_id=current_user.id
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
            detail="Failed to disband campaign"
        )


@router.delete(
    "/{campaign_id}/permanent",
    status_code=status.HTTP_200_OK
)
async def permanently_delete_campaign(
    campaign_id: str,
    request: PermanentDeleteRequest,
    current_user = Depends(get_current_user),
    db = Depends(get_db)
):
    """
    永久删除战役（硬删除）

    需要权限：KP
    警告：此操作不可撤销！
    """
    try:
        service = DisbandService(db)
        result = await service.permanently_delete_campaign(
            campaign_id=campaign_id,
            operator_id=current_user.id,
            confirm_name=request.confirm_name
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
            detail="Failed to delete campaign"
        )
```

### 3. WebSocket 事件处理

```python
# backend/handlers/disband_events.py
from socketio import AsyncServer
from loguru import logger

from services.disband_service import DisbandService


class DisbandEventHandler:
    """解散相关 WebSocket 事件处理"""

    def __init__(self, sio: AsyncServer):
        self.sio = sio
        self._setup_handlers()

    def _setup_handlers(self):
        """设置事件处理器"""

        @self.sio.event
        async def disband_campaign(sid, data):
            """
            解散战役

            data: {
                room_id: str
            }
            """
            from database import get_db_context

            room_id = data.get("room_id")

            operator_info = self._get_socket_user(sid)
            if not operator_info:
                await self._emit_error(sid, "Unauthorized")
                return

            try:
                async with get_db_context() as db:
                    service = DisbandService(db)
                    result = await service.disband_campaign(
                        campaign_id=room_id,
                        operator_id=operator_info["user_id"]
                    )

                    # 通知房间内所有人
                    await self.sio.emit(
                        "campaign_disbanded",
                        {
                            "room_id": room_id,
                            "campaign_name": result["campaign_name"],
                            "disbanded_by": result["disbanded_by"],
                            "message": f"The campaign has been disbanded by the Game Master"
                        },
                        room=room_id
                    )

                    # 断开所有房间内连接
                    await self._disconnect_all_from_room(room_id)

                    # 向操作者确认
                    await self.sio.emit(
                        "disband_success",
                        result,
                        room=sid
                    )

                    logger.info(f"Campaign {room_id} disbanded via WebSocket")

            except PermissionError as e:
                await self._emit_error(sid, str(e), "PERMISSION_DENIED")
            except ValueError as e:
                await self._emit_error(sid, str(e), "INVALID_OPERATION")
            except Exception as e:
                logger.error(f"Error disbanding campaign: {e}")
                await self._emit_error(sid, "Failed to disband campaign")

    async def _disconnect_all_from_room(self, room_id: str):
        """断开房间内所有连接"""
        # 获取房间内所有 session
        rooms = self.sio.manager.get_rooms()
        if room_id in rooms:
            for sid in rooms[room_id]:
                await self.sio.emit(
                    "force_disconnect",
                    {
                        "reason": "campaign_disbanded",
                        "message": "The campaign has been disbanded"
                    },
                    room=sid
                )
                # 断开连接
                self.sio.disconnect(sid)

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

### 1. 解散 Hook

```typescript
// frontend/src/hooks/useDisbandCampaign.ts
import { useCallback } from "react";
import { useNavigate } from "react-router-dom";
import axios from "axios";

interface UseDisbandCampaignOptions {
  onDisbanded?: () => void;
}

interface DisbandResult {
  success: boolean;
  campaign_id: string;
  campaign_name: string;
  disbanded_by: string;
  member_count: number;
  disbanded_at: string;
}

export function useDisbandCampaign({ onDisbanded }: UseDisbandCampaignOptions = {}) {
  const navigate = useNavigate();

  const disbandCampaign = useCallback(async (campaignId: string): Promise<DisbandResult> => {
    try {
      const response = await axios.delete<DisbandResult>(
        `/api/campaigns/${campaignId}`
      );

      // 调用回调
      onDisbanded?.();

      // 导航到首页
      navigate("/", {
        state: {
          message: "Campaign disbanded successfully",
          type: "info"
        }
      });

      return response.data;
    } catch (error) {
      if (axios.isAxiosError(error)) {
        throw new Error(error.response?.data?.detail || "Failed to disband campaign");
      }
      throw error;
    }
  }, [navigate, onDisbanded]);

  const permanentlyDelete = useCallback(async (
    campaignId: string,
    confirmName: string
  ): Promise<void> => {
    try {
      await axios.delete(`/api/campaigns/${campaignId}/permanent`, {
        data: { confirm_name: confirmName }
      });

      navigate("/", {
        state: {
          message: "Campaign permanently deleted",
          type: "warning"
        }
      });
    } catch (error) {
      if (axios.isAxiosError(error)) {
        throw new Error(error.response?.data?.detail || "Failed to delete campaign");
      }
      throw error;
    }
  }, [navigate]);

  return {
    disbandCampaign,
    permanentlyDelete
  };
}
```

### 2. 解散检测 Hook

```typescript
// frontend/src/hooks/useDisbandDetection.ts
import { useEffect, useState, useCallback } from "react";
import { Socket } from "socket.io-client";
import { useNavigate } from "react-router-dom";

interface UseDisbandDetectionOptions {
  socket: Socket | null;
  roomId: string;
}

export function useDisbandDetection({ socket, roomId }: UseDisbandDetectionOptions) {
  const [disbandedData, setDisbandedData] = useState<{
    room_id: string;
    campaign_name: string;
    disbanded_by: string;
    message: string;
  } | null>(null);

  const navigate = useNavigate();

  const handleDisbanded = useCallback((data: {
    room_id: string;
    campaign_name: string;
    disbanded_by: string;
    message: string;
  }) => {
    setDisbandedData(data);

    // 断开 Socket
    socket?.disconnect();

    // 导航到解散通知页面
    navigate("/disbanded", { state: data });
  }, [socket, navigate]);

  useEffect(() => {
    if (!socket) return;

    const handleCampaignDisbanded = (data: {
      room_id: string;
      campaign_name: string;
      disbanded_by: string;
      message: string;
    }) => {
      if (data.room_id === roomId) {
        handleDisbanded(data);
      }
    };

    const handleForceDisconnect = (data: {
      reason: string;
      message: string;
    }) => {
      if (data.reason === "campaign_disbanded") {
        socket.disconnect();
        navigate("/", {
          state: {
            message: data.message,
            type: "warning"
          }
        });
      }
    };

    socket.on("campaign_disbanded", handleCampaignDisbanded);
    socket.on("force_disconnect", handleForceDisconnect);

    return () => {
      socket.off("campaign_disbanded", handleCampaignDisbanded);
      socket.off("force_disconnect", handleForceDisconnect);
    };
  }, [socket, roomId, handleDisbanded, navigate]);

  return {
    disbandedData,
    isDisbanded: disbandedData !== null
  };
}
```

### 3. 解散确认对话框

```typescript
// frontend/src/components/room/DisbandConfirmDialog.tsx
import React, { useState } from "react";
import { AlertTriangle, X, Trash2 } from "lucide-react";

interface DisbandConfirmDialogProps {
  campaignName: string;
  onConfirm: () => Promise<void>;
  onClose: () => void;
}

export function DisbandConfirmDialog({
  campaignName,
  onConfirm,
  onClose
}: DisbandConfirmDialogProps) {
  const [confirmText, setConfirmText] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const canConfirm = confirmText === campaignName;

  const handleConfirm = async () => {
    if (!canConfirm) return;

    setIsSubmitting(true);
    try {
      await onConfirm();
    } finally {
      setIsSubmitting(false);
    }
  };

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

        <div className="flex items-center gap-3 mb-4">
          <div className="w-12 h-12 bg-red-100 dark:bg-red-900/30 rounded-full flex items-center justify-center">
            <AlertTriangle className="w-6 h-6 text-red-600 dark:text-red-400" />
          </div>
          <div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
              Disband Campaign
            </h3>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              This action cannot be undone
            </p>
          </div>
        </div>

        <div className="mb-4">
          <p className="text-gray-700 dark:text-gray-300 mb-2">
            You are about to disband the campaign:
          </p>
          <p className="font-semibold text-gray-900 dark:text-white text-lg">
            {campaignName}
          </p>
        </div>

        <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-3 mb-4">
          <p className="text-sm text-yellow-800 dark:text-yellow-200">
            <strong>Warning:</strong> All players will be removed from the campaign.
            The campaign will be marked as deleted and can be recovered by administrators.
          </p>
        </div>

        <div className="mb-4">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Type <code>{campaignName}</code> to confirm
          </label>
          <input
            type="text"
            value={confirmText}
            onChange={(e) => setConfirmText(e.target.value)}
            placeholder="Enter campaign name..."
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-red-500 dark:bg-gray-700 dark:border-gray-600 dark:text-white"
          />
        </div>

        <div className="flex gap-3">
          <button
            onClick={onClose}
            disabled={isSubmitting}
            className="flex-1 px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200 disabled:opacity-50 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600"
          >
            Cancel
          </button>
          <button
            onClick={handleConfirm}
            disabled={!canConfirm || isSubmitting}
            className="flex-1 px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-md hover:bg-red-700 disabled:opacity-50 flex items-center justify-center gap-2"
          >
            {isSubmitting ? (
              <>
                <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                Disbanding...
              </>
            ) : (
              <>
                <Trash2 size={16} />
                Disband Campaign
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
```

### 4. 解散通知页面

```typescript
// frontend/src/pages/DisbandedPage.tsx
import React from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { AlertTriangle, Home } from "lucide-react";

export function DisbandedPage() {
  const location = useLocation();
  const navigate = useNavigate();
  const state = location.state as {
    campaign_name?: string;
    disbanded_by?: string;
    message?: string;
  } | null;

  const handleGoHome = () => {
    navigate("/");
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-4">
      <div className="max-w-md w-full bg-white dark:bg-gray-800 rounded-lg shadow-lg p-8">
        <div className="flex justify-center mb-6">
          <div className="w-16 h-16 bg-red-100 dark:bg-red-900/30 rounded-full flex items-center justify-center">
            <AlertTriangle className="w-8 h-8 text-red-600 dark:text-red-400" />
          </div>
        </div>

        <h1 className="text-2xl font-bold text-center text-gray-900 dark:text-white mb-4">
          Campaign Disbanded
        </h1>

        <div className="space-y-4 text-center">
          {state?.campaign_name && (
            <p className="text-gray-600 dark:text-gray-400">
              The campaign <strong>"{state.campaign_name}"</strong> has been disbanded.
            </p>
          )}

          {state?.disbanded_by && (
            <p className="text-gray-600 dark:text-gray-400">
              by {state.disbanded_by}
            </p>
          )}

          {state?.message && (
            <div className="bg-gray-100 dark:bg-gray-700 rounded-lg p-4">
              <p className="text-sm text-gray-700 dark:text-gray-300">
                {state.message}
              </p>
            </div>
          )}
        </div>

        <button
          onClick={handleGoHome}
          className="w-full mt-6 px-4 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors flex items-center justify-center gap-2"
        >
          <Home size={20} />
          Return to Home
        </button>
      </div>
    </div>
  );
}
```

### 5. KP 管理面板（解散按钮）

```typescript
// frontend/src/components/room/KPManagementPanel.tsx
import React, { useState } from "react";
import { Settings, Trash2, Shield } from "lucide-react";
import { useRoomState } from "@/contexts/RoomStateContext";
import { useDisbandCampaign } from "@/hooks/useDisbandCampaign";
import { DisbandConfirmDialog } from "./DisbandConfirmDialog";
import { toast } from "react-hot-toast";

export function KPManagementPanel() {
  const { roomState, isKP } = useRoomState();
  const { disbandCampaign } = useDisbandCampaign({
    onDisbanded: () => {
      toast.success("Campaign disbanded");
    }
  });

  const [showDisbandDialog, setShowDisbandDialog] = useState(false);

  if (!isKP || !roomState) return null;

  return (
    <div className="kp-management-panel">
      <div className="panel-header">
        <h3 className="panel-title">
          <Shield size={20} />
          Game Master Controls
        </h3>
      </div>

      <div className="panel-content">
        <div className="control-section">
          <h4>Campaign Management</h4>

          <button
            onClick={() => setShowDisbandDialog(true)}
            className="danger-button"
          >
            <Trash2 size={16} />
            Disband Campaign
          </button>

          <p className="text-sm text-gray-500 dark:text-gray-400">
            Permanently close this campaign. All players will be removed.
          </p>
        </div>
      </div>

      {showDisbandDialog && roomState.campaign && (
        <DisbandConfirmDialog
          campaignName={roomState.campaign.name}
          onConfirm={async () => {
            try {
              await disbandCampaign(roomState.room_id);
            } catch (error) {
              toast.error(error instanceof Error ? error.message : "Failed to disband");
              throw error;
            }
          }}
          onClose={() => setShowDisbandDialog(false)}
        />
      )}
    </div>
  );
}
```

---

## 涉及文件清单

### 后端文件

| 文件路径 | 操作 | 说明 |
|---------|------|------|
| `backend/services/disband_service.py` | 创建 | 解散战役业务逻辑 |
| `backend/api/routes/campaign_disband.py` | 创建 | 解散 API 路由 |
| `backend/handlers/disband_events.py` | 创建 | WebSocket 解散事件处理 |
| `backend/models/campaign.py` | 修改 | 添加 is_deleted 和 deleted_at 字段 |
| `backend/api/routes/campaigns.py` | 修改 | 添加软删除过滤 |

### 前端文件

| 文件路径 | 操作 | 说明 |
|---------|------|------|
| `frontend/src/hooks/useDisbandCampaign.ts` | 创建 | 解散战役 Hook |
| `frontend/src/hooks/useDisbandDetection.ts` | 创建 | 解散检测 Hook |
| `frontend/src/components/room/DisbandConfirmDialog.tsx` | 创建 | 解散确认对话框 |
| `frontend/src/components/room/KPManagementPanel.tsx` | 创建 | KP 管理面板 |
| `frontend/src/pages/DisbandedPage.tsx` | 创建 | 解散通知页面 |
| `frontend/src/router/index.tsx` | 修改 | 添加解散页面路由 |
| `frontend/src/pages/GamePage.tsx` | 修改 | 集成解散检测 |

---

## 验收标准

### 功能验收

- [ ] KP 可以看到解散按钮
- [ ] 点击解散显示确认对话框
- [ ] 需要输入战役名称确认
- [ ] 解散后所有玩家收到通知
- [ ] 所有玩家被强制断开连接
- [ ] 房间被标记为已删除（软删除）
- [ ] 玩家被重定向到解散通知页面
- [ ] 列表页不显示已解散的战役

### 安全验收

- [ ] 只有 KP 可以解散战役
- [ ] 后端验证所有权限
- [ ] 解散操作需要二次确认
- [ ] 操作记录到审计日志
- [ ] 防止 CSRF 攻击

### UX 验收

- [ ] 解散对话框突出显示警告
- [ ] 名称确认防止误操作
- [ ] 清晰说明操作后果
- [ ] 加载状态显示
- [ ] 错误信息友好

---

## 参考文档

- [软删除模式](https://martinfowler.com/articles/foundations-pattern2.html)
- [WebSocket 房间管理](https://socket.io/docs/v4/rooms/)
- [危险操作 UI 模式](https://www.nngroup.com/articles/confirmation-dialogs/)
- [RESTful DELETE 设计](https://restfulapi.net/http-methods/#delete)

---

**创建日期**: 2026-02-06
**负责人**: 待分配
**审核人**: 待分配
