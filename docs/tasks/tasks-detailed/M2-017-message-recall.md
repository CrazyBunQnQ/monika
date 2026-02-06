# M2-017: 实现消息撤回

**任务类型**: Backend + Frontend
**预估工时**: 8h
**优先级**: P1
**依赖**: M2-012 (房间状态同步)

---

## 任务描述

实现消息撤回功能。玩家可以撤回自己发送的消息（有 2 分钟时间限制），KP 可以撤回房间内任何消息。撤回后，消息内容被替换为"消息已撤回"标记，但保留撤回记录用于审计。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 | 依赖 |
|----|--------|------|----------|------|
| M2-017-01 | 设计消息撤回数据结构 | 添加 is_deleted、deleted_at 等字段 | 1h | M2-012 |
| M2-017-02 | 实现后端撤回 API | DELETE /messages/:id | 2h | M2-017-01 |
| M2-017-03 | 实现撤回权限验证 | 检查时间限制和发送者身份 | 1.5h | M2-017-02 |
| M2-017-04 | 实现 WebSocket 撤回事件 | 实时广播撤回通知 | 1.5h | M2-017-03 |
| M2-017-05 | 实现前端撤回 UI | 长按/右键显示撤回选项 | 1h | M2-017-04 |
| M2-017-06 | 实现撤回状态显示 | 已撤回消息样式 | 1h | M2-017-05 |

---

## 后端代码示例

### 1. 消息撤回服务

```python
# backend/services/message_recall_service.py
from typing import Optional
from datetime import datetime, timedelta
from loguru import logger
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select, update

from models.message import Message
from models.campaign import CampaignMember


class MessageRecallService:
    """消息撤回服务"""

    # 普通玩家撤回时间限制（分钟）
    RECALL_TIME_LIMIT_MINUTES = 2

    def __init__(self, db: AsyncSession):
        self.db = db

    async def recall_message(
        self,
        message_id: str,
        operator_id: str,
        campaign_id: str
    ) -> dict:
        """
        撤回消息

        Args:
            message_id: 消息 ID
            operator_id: 操作者 ID
            campaign_id: 战役 ID

        Returns:
            撤回结果
        """
        # 1. 获取消息
        message = await self._get_message(message_id)
        if not message:
            raise ValueError("Message not found")

        # 2. 检查是否已撤回
        if message.is_deleted:
            raise ValueError("Message already recalled")

        # 3. 验证权限
        operator_member = await self._get_member(campaign_id, operator_id)
        if not operator_member:
            raise PermissionError("You are not a member of this campaign")

        is_kp = operator_member.role == "kp"
        is_sender = message.sender_id == operator_id

        # KP 可以撤回任何消息，普通玩家只能撤回自己的
        if not is_kp and not is_sender:
            raise PermissionError("You can only recall your own messages")

        # 4. 检查时间限制（非 KP）
        if not is_kp:
            time_limit = timedelta(minutes=self.RECALL_TIME_LIMIT_MINUTES)
            if datetime.utcnow() - message.created_at > time_limit:
                raise ValueError(
                    f"Messages can only be recalled within "
                    f"{self.RECALL_TIME_LIMIT_MINUTES} minutes"
                )

        # 5. 执行撤回（软删除）
        await self._soft_delete_message(message_id)

        logger.info(
            f"Message {message_id} recalled by {operator_id} "
            f"(KP: {is_kp}, Sender: {is_sender})"
        )

        return {
            "success": True,
            "message_id": message_id,
            "recalled_by": operator_id,
            "recalled_by_username": operator_member.user.username,
            "is_kp": is_kp,
            "recalled_at": datetime.utcnow().isoformat()
        }

    async def _get_message(self, message_id: str) -> Optional[Message]:
        """获取消息"""
        result = await self.db.execute(
            select(Message).where(Message.id == message_id)
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

    async def _soft_delete_message(self, message_id: str):
        """软删除消息"""
        await self.db.execute(
            update(Message)
            .where(Message.id == message_id)
            .values(
                is_deleted=True,
                deleted_at=datetime.utcnow(),
                # 保留原始内容用于审计，但前端显示撤回标记
                # content = "[Message Recalled]"
            )
        )
        await self.db.commit()

    async def can_recall_message(
        self,
        message_id: str,
        user_id: str,
        campaign_id: str
    ) -> dict:
        """
        检查是否可以撤回消息

        Returns:
            {
                can_recall: bool,
                reason: str | None
            }
        """
        message = await self._get_message(message_id)
        if not message:
            return {"can_recall": False, "reason": "Message not found"}

        if message.is_deleted:
            return {"can_recall": False, "reason": "Message already recalled"}

        operator_member = await self._get_member(campaign_id, user_id)
        if not operator_member:
            return {"can_recall": False, "reason": "Not a campaign member"}

        is_kp = operator_member.role == "kp"
        is_sender = message.sender_id == user_id

        if not is_kp and not is_sender:
            return {"can_recall": False, "reason": "Can only recall own messages"}

        if not is_kp:
            time_limit = timedelta(minutes=self.RECALL_TIME_LIMIT_MINUTES)
            if datetime.utcnow() - message.created_at > time_limit:
                return {"can_recall": False, "reason": "Time limit exceeded"}

        return {"can_recall": True, "reason": None}
```

### 2. API 路由

```python
# backend/api/routes/message_recall.py
from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel

from database import get_db
from services.message_recall_service import MessageRecallService
from middleware.auth import get_current_user

router = APIRouter(prefix="/api/messages", tags=["recall"])


class RecallResponse(BaseModel):
    """撤回响应"""
    success: bool
    message_id: str
    recalled_by: str
    recalled_by_username: str
    is_kp: bool
    recalled_at: str


@router.delete(
    "/{message_id}/recall",
    response_model=RecallResponse,
    status_code=status.HTTP_200_OK
)
async def recall_message(
    message_id: str,
    campaign_id: str,
    current_user = Depends(get_current_user),
    db = Depends(get_db)
):
    """
    撤回消息

    - 普通玩家：只能撤回自己的消息，且在 2 分钟内
    - KP：可以撤回任何消息，无时间限制
    """
    try:
        service = MessageRecallService(db)
        result = await service.recall_message(
            message_id=message_id,
            operator_id=current_user.id,
            campaign_id=campaign_id
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
            detail="Failed to recall message"
        )


@router.get(
    "/{message_id}/can-recall",
    status_code=status.HTTP_200_OK
)
async def check_can_recall(
    message_id: str,
    campaign_id: str,
    current_user = Depends(get_current_user),
    db = Depends(get_db)
):
    """检查是否可以撤回消息"""
    try:
        service = MessageRecallService(db)
        result = await service.can_recall_message(
            message_id=message_id,
            user_id=current_user.id,
            campaign_id=campaign_id
        )
        return result

    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to check recall permission"
        )
```

### 3. WebSocket 事件处理

```python
# backend/handlers/recall_events.py
from socketio import AsyncServer
from loguru import logger

from services.message_recall_service import MessageRecallService


class RecallEventHandler:
    """撤回相关 WebSocket 事件处理"""

    def __init__(self, sio: AsyncServer):
        self.sio = sio
        self._setup_handlers()

    def _setup_handlers(self):
        """设置事件处理器"""

        @self.sio.event
        async def recall_message(sid, data):
            """
            撤回消息

            data: {
                message_id: str,
                campaign_id: str
            }
            """
            from database import get_db_context

            message_id = data.get("message_id")
            campaign_id = data.get("campaign_id")

            operator_info = self._get_socket_user(sid)
            if not operator_info:
                await self._emit_error(sid, "Unauthorized")
                return

            try:
                async with get_db_context() as db:
                    service = MessageRecallService(db)
                    result = await service.recall_message(
                        message_id=message_id,
                        operator_id=operator_info["user_id"],
                        campaign_id=campaign_id
                    )

                    # 广播撤回事件到房间
                    await self.sio.emit(
                        "message_recalled",
                        {
                            "message_id": message_id,
                            "recalled_by": result["recalled_by_username"],
                            "is_kp": result["is_kp"],
                            "recalled_at": result["recalled_at"]
                        },
                        room=campaign_id
                    )

                    # 向操作者确认
                    await self.sio.emit(
                        "recall_success",
                        result,
                        room=sid
                    )

                    logger.info(f"Message {message_id} recalled via WebSocket")

            except PermissionError as e:
                await self._emit_error(sid, str(e), "PERMISSION_DENIED")
            except ValueError as e:
                await self._emit_error(sid, str(e), "INVALID_OPERATION")
            except Exception as e:
                logger.error(f"Error recalling message: {e}")
                await self._emit_error(sid, "Failed to recall message")

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

### 1. 消息撤回 Hook

```typescript
// frontend/src/hooks/useMessageRecall.ts
import { useCallback } from "react";
import { Socket } from "socket.io-client";
import axios from "axios";

interface UseMessageRecallOptions {
  socket?: Socket | null;
  campaignId: string;
  userId: string;
  isKP: boolean;
}

interface RecallResult {
  success: boolean;
  message_id: string;
  recalled_by: string;
  recalled_by_username: string;
  is_kp: boolean;
  recalled_at: string;
}

export function useMessageRecall({
  socket,
  campaignId,
  userId,
  isKP
}: UseMessageRecallOptions) {
  const recallMessage = useCallback(async (
    messageId: string,
    senderId: string,
    createdAt: string
  ): Promise<RecallResult> => {
    try {
      const response = await axios.delete<RecallResult>(
        `/api/messages/${messageId}/recall`,
        { params: { campaign_id: campaignId } }
      );
      return response.data;
    } catch (error) {
      if (axios.isAxiosError(error)) {
        const detail = error.response?.data?.detail;
        throw new Error(detail || "Failed to recall message");
      }
      throw error;
    }
  }, [campaignId]);

  const recallViaSocket = useCallback((
    messageId: string,
    onSuccess?: (result: RecallResult) => void,
    onError?: (error: string) => void
  ) => {
    if (!socket?.connected) {
      onError?.("Socket not connected");
      return;
    }

    socket.emit("recall_message", {
      message_id: messageId,
      campaign_id: campaignId
    });

    const handleSuccess = (result: RecallResult) => {
      onSuccess?.(result);
      socket.off("recall_success", handleSuccess);
      socket.off("error", handleError);
    };

    const handleError = (error: { message: string }) => {
      onError?.(error.message);
      socket.off("recall_success", handleSuccess);
      socket.off("error", handleError);
    };

    socket.once("recall_success", handleSuccess);
    socket.once("error", handleError);
  }, [socket, campaignId]);

  const checkCanRecall = useCallback(async (
    messageId: string,
    senderId: string,
    createdAt: string
  ): Promise<{ can_recall: boolean; reason?: string }> => {
    // 客户端快速检查
    const isSender = senderId === userId;
    if (!isKP && !isSender) {
      return { can_recall: false, reason: "You can only recall your own messages" };
    }

    if (!isKP) {
      const timeLimit = 2 * 60 * 1000; // 2 分钟
      const messageAge = Date.now() - new Date(createdAt).getTime();
      if (messageAge > timeLimit) {
        return { can_recall: false, reason: "Messages can only be recalled within 2 minutes" };
      }
    }

    // 服务器验证（可选）
    try {
      const response = await axios.get(
        `/api/messages/${messageId}/can-recall`,
        { params: { campaign_id: campaignId } }
      );
      return response.data;
    } catch {
      return { can_recall: true }; // 客户端检查通过则默认允许
    }
  }, [campaignId, userId, isKP]);

  return {
    recallMessage,
    recallViaSocket,
    checkCanRecall
  };
}
```

### 2. 消息组件（带撤回功能）

```typescript
// frontend/src/components/room/MessageItem.tsx
import React, { useState } from "react";
import { Trash2, MoreVertical } from "lucide-react";
import { Message } from "@/types/message";
import { useMessageRecall } from "@/hooks/useMessageRecall";
import { useRoomState } from "@/contexts/RoomStateContext";
import { formatMessageTime } from "@/utils/format";
import { toast } from "react-hot-toast";

interface MessageItemProps {
  message: Message;
}

export function MessageItem({ message }: MessageItemProps) {
  const { currentUserId, isKP } = useRoomState();
  const { recallViaSocket, checkCanRecall } = useMessageRecall({
    campaignId: message.campaign_id,
    userId: currentUserId,
    isKP
  });

  const [showMenu, setShowMenu] = useState(false);
  const [isRecalling, setIsRecalling] = useState(false);

  const isSender = message.sender_id === currentUserId;
  const canRecall = isKP || isSender;

  const handleRecall = async () => {
    setIsRecalling(true);
    setShowMenu(false);

    try {
      // 检查是否可以撤回
      const check = await checkCanRecall(
        message.id,
        message.sender_id,
        message.created_at
      );

      if (!check.can_recall) {
        toast.error(check.reason || "Cannot recall this message");
        setIsRecalling(false);
        return;
      }

      // 执行撤回
      recallViaSocket(
        message.id,
        (result) => {
          toast.success("Message recalled");
          setIsRecalling(false);
        },
        (error) => {
          toast.error(`Failed to recall: ${error}`);
          setIsRecalling(false);
        }
      );
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to recall");
      setIsRecalling(false);
    }
  };

  // 已撤回的消息
  if (message.is_deleted) {
    return (
      <div className="message-item recalled">
        <div className="message-avatar">
          <div className="avatar-placeholder">
            {message.sender_username[0].toUpperCase()}
          </div>
        </div>
        <div className="message-content recalled-content">
          <div className="message-header">
            <span className="message-sender">{message.sender_username}</span>
            <span className="message-time">{formatMessageTime(message.created_at)}</span>
          </div>
          <div className="recalled-text">
            <Trash2 size={14} />
            <span>Message recalled</span>
            {message.deleted_at && (
              <span className="recall-time">
                {formatMessageTime(message.deleted_at)}
              </span>
            )}
          </div>
        </div>
      </div>
    );
  }

  // 正常消息
  return (
    <div className={`message-item ${isSender ? "sent" : "received"}`}>
      <div className="message-avatar">
        {message.sender_avatar ? (
          <img src={message.sender_avatar} alt={message.sender_username} />
        ) : (
          <div className="avatar-placeholder">
            {message.sender_username[0].toUpperCase()}
          </div>
        )}
      </div>

      <div className="message-content">
        <div className="message-header">
          <span className="message-sender">{message.sender_username}</span>
          <span className="message-time">{formatMessageTime(message.created_at)}</span>

          {canRecall && (
            <div className="message-menu-container">
              <button
                onClick={() => setShowMenu(!showMenu)}
                className="menu-trigger"
              >
                <MoreVertical size={16} />
              </button>

              {showMenu && (
                <div className="message-menu">
                  <button
                    onClick={handleRecall}
                    disabled={isRecalling}
                    className="menu-item recall"
                  >
                    <Trash2 size={14} />
                    {isRecalling ? "Recalling..." : "Recall Message"}
                  </button>
                </div>
              )}
            </div>
          )}
        </div>

        <div className="message-text">
          {message.content}
        </div>

        {message.attachments && message.attachments.length > 0 && (
          <div className="message-attachments">
            {message.attachments.map((attachment, index) => (
              <div key={index} className="attachment">
                {/* 渲染附件 */}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
```

### 3. 消息列表组件（处理撤回事件）

```typescript
// frontend/src/components/room/MessageList.tsx
import React, { useEffect } from "react";
import { Socket } from "socket.io-client";
import { Message } from "@/types/message";
import { MessageItem } from "./MessageItem";

interface MessageListProps {
  messages: Message[];
  socket: Socket | null;
  onMessageRecalled?: (messageId: string) => void;
}

export function MessageList({ messages, socket, onMessageRecalled }: MessageListProps) {
  useEffect(() => {
    if (!socket) return;

    const handleMessageRecalled = (data: {
      message_id: string;
      recalled_by: string;
      is_kp: boolean;
      recalled_at: string;
    }) => {
      // 通知父组件更新消息列表
      onMessageRecalled?.(data.message_id);
    };

    socket.on("message_recalled", handleMessageRecalled);

    return () => {
      socket.off("message_recalled", handleMessageRecalled);
    };
  }, [socket, onMessageRecalled]);

  return (
    <div className="message-list">
      {messages.map((message) => (
        <MessageItem key={message.id} message={message} />
      ))}
    </div>
  );
}
```

### 4. 撤回上下文菜单样式

```css
/* frontend/src/components/room/MessageItem.module.css */

.message-menu-container {
  position: relative;
}

.menu-trigger {
  padding: 4px;
  border: none;
  background: transparent;
  cursor: pointer;
  border-radius: 4px;
  color: #6b7280;
}

.menu-trigger:hover {
  background: #f3f4f6;
}

.message-menu {
  position: absolute;
  top: 100%;
  right: 0;
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
  min-width: 160px;
  z-index: 50;
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px 12px;
  border: none;
  background: transparent;
  cursor: pointer;
  font-size: 14px;
  text-align: left;
}

.menu-item:hover {
  background: #f3f4f6;
}

.menu-item.recall {
  color: #dc2626;
}

.menu-item.recall:hover {
  background: #fef2f2;
}

/* 撤回消息样式 */
.message-item.recalled {
  opacity: 0.6;
}

.recalled-content {
  font-style: italic;
}

.recalled-text {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #6b7280;
  font-size: 14px;
}

.recall-time {
  font-size: 12px;
  color: #9ca3af;
}
```

---

## 涉及文件清单

### 后端文件

| 文件路径 | 操作 | 说明 |
|---------|------|------|
| `backend/services/message_recall_service.py` | 创建 | 消息撤回业务逻辑 |
| `backend/api/routes/message_recall.py` | 创建 | 撤回 API 路由 |
| `backend/handlers/recall_events.py` | 创建 | WebSocket 撤回事件处理 |
| `backend/models/message.py` | 修改 | 添加 is_deleted、deleted_at 字段 |
| `backend/api/routes/messages.py` | 修改 | 集成撤回检查 |

### 前端文件

| 文件路径 | 操作 | 说明 |
|---------|------|------|
| `frontend/src/hooks/useMessageRecall.ts` | 创建 | 消息撤回 Hook |
| `frontend/src/components/room/MessageItem.tsx` | 修改 | 添加撤回菜单 |
| `frontend/src/components/room/MessageList.tsx` | 修改 | 处理撤回事件 |
| `frontend/src/components/room/MessageItem.module.css` | 创建 | 撤回样式 |
| `frontend/src/types/message.ts` | 修改 | 添加撤回相关类型 |

---

## 验收标准

### 功能验收

- [ ] 普通玩家可以撤回自己 2 分钟内的消息
- [ ] KP 可以撤回任何消息（无时间限制）
- [ ] 撤回后消息显示"消息已撤回"
- [ ] 撤回操作实时同步到所有玩家
- [ ] 超过 2 分钟的消息无法撤回
- [ ] 已撤回的消息无法再次撤回
- [ ] 撤回操作记录到审计日志

### UX 验收

- [ ] 消息菜单触发方式友好（点击/右键）
- [ ] 撤回确认对话框防止误操作
- [ ] 已撤回消息有明显视觉区分
- [ ] 时间限制提示清晰
- [ ] 撤回操作有加载状态

---

## 参考文档

- [软删除模式](https://martinfowler.com/articles/foundations-pattern2.html)
- [WebSocket 实时更新](https://socket.io/docs/v4/emitting-events/)
- [React Context 模式](https://react.dev/reference/react/useContext)
- [消息撤回最佳实践](https://slack.com/blog/productivity/mobile-message-editing-and-deletion)

---

**创建日期**: 2026-02-06
**负责人**: 待分配
**审核人**: 待分配
