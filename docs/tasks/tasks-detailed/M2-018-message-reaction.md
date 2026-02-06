# M2-018: 实现消息表情反应

**任务类型**: Backend + Frontend
**预估工时**: 10h
**优先级**: P1
**依赖**: M2-012 (房间状态同步)

---

## 任务描述

实现消息表情反应功能。玩家可以对消息添加表情反应（如 👍、❤️、😂 等），支持多个玩家对同一消息添加多个反应。实时同步反应状态，显示反应数量和用户列表。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 | 依赖 |
|----|--------|------|----------|------|
| M2-018-01 | 设计反应数据结构 | MessageReaction 表和关联 | 1.5h | M2-012 |
| M2-018-02 | 实现后端反应 API | POST/DELETE /messages/:id/reactions | 2h | M2-018-01 |
| M2-018-03 | 实现反应聚合查询 | 获取消息的所有反应 | 1.5h | M2-018-02 |
| M2-018-04 | 实现 WebSocket 反应事件 | 实时广播反应变化 | 2h | M2-018-03 |
| M2-018-05 | 实现前端表情选择器 | Emoji picker 组件 | 2h | M2-018-04 |
| M2-018-06 | 实现反应显示 UI | 反应气泡和用户列表 | 1h | M2-018-05 |

---

## 后端代码示例

### 1. 反应数据模型

```python
# backend/models/message_reaction.py
from sqlalchemy import Column, String, DateTime, ForeignKey, UniqueConstraint
from sqlalchemy.orm import relationship
from datetime import datetime
from database import Base


class MessageReaction(Base):
    """消息反应"""
    __tablename__ = "message_reactions"

    id = Column(String, primary_key=True)
    message_id = Column(String, ForeignKey("messages.id"), nullable=False)
    user_id = Column(String, ForeignKey("users.id"), nullable=False)
    emoji = Column(String, nullable=False)  # 存储表情字符
    created_at = Column(DateTime, default=datetime.utcnow, nullable=False)

    # 关联
    message = relationship("Message", back_populates="reactions")
    user = relationship("User")

    # 唯一约束：每个用户对每条消息的每个表情只能有一个
    __table_args__ = (
        UniqueConstraint("message_id", "user_id", "emoji", name="unique_reaction"),
    )

    def to_dict(self):
        return {
            "id": self.id,
            "message_id": self.message_id,
            "user_id": self.user_id,
            "emoji": self.emoji,
            "created_at": self.created_at.isoformat()
        }
```

```python
# backend/models/message.py (修改)
# 添加关联
from sqlalchemy.orm import relationship

class Message(Base):
    # ... 现有字段 ...

    # 添加关联
    reactions = relationship("MessageReaction", back_populates="message", cascade="all, delete-orphan")
```

### 2. 反应服务

```python
# backend/services/reaction_service.py
from typing import List, Dict
from datetime import datetime
from loguru import logger
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select, delete, func
from uuid import uuid4

from models.message_reaction import MessageReaction
from models.message import Message


class ReactionService:
    """消息反应服务"""

    # 支持的表情列表（可扩展）
    AVAILABLE_EMOJIS = [
        "👍", "👎", "❤️", "🔥", "😂", "😮", "😢", "🎉",
        "🚀", "💯", "✨", "🤔", "👀", "👏", "🙏", "💪"
    ]

    def __init__(self, db: AsyncSession):
        self.db = db

    async def add_reaction(
        self,
        message_id: str,
        user_id: str,
        emoji: str
    ) -> dict:
        """
        添加表情反应

        Args:
            message_id: 消息 ID
            user_id: 用户 ID
            emoji: 表情字符

        Returns:
            反应结果
        """
        # 1. 验证表情
        if emoji not in self.AVAILABLE_EMOJIS:
            raise ValueError(f"Emoji not supported: {emoji}")

        # 2. 验证消息存在
        message = await self._get_message(message_id)
        if not message:
            raise ValueError("Message not found")

        # 3. 检查是否已存在相同反应
        existing = await self._get_reaction(message_id, user_id, emoji)
        if existing:
            # 如果已存在，则删除（切换效果）
            await self._remove_reaction(message_id, user_id, emoji)
            logger.info(f"Reaction removed: {message_id} by {user_id}")
            return {
                "action": "removed",
                "message_id": message_id,
                "emoji": emoji,
                "user_id": user_id
            }

        # 4. 添加新反应
        reaction = MessageReaction(
            id=str(uuid4()),
            message_id=message_id,
            user_id=user_id,
            emoji=emoji
        )

        self.db.add(reaction)
        await self.db.commit()

        logger.info(f"Reaction added: {message_id} by {user_id} - {emoji}")

        return {
            "action": "added",
            "message_id": message_id,
            "emoji": emoji,
            "user_id": user_id,
            "reaction_id": reaction.id
        }

    async def remove_reaction(
        self,
        message_id: str,
        user_id: str,
        emoji: str
    ) -> dict:
        """
        删除表情反应

        Args:
            message_id: 消息 ID
            user_id: 用户 ID
            emoji: 表情字符

        Returns:
            删除结果
        """
        # 检查反应是否存在
        existing = await self._get_reaction(message_id, user_id, emoji)
        if not existing:
            raise ValueError("Reaction not found")

        await self._remove_reaction(message_id, user_id, emoji)

        logger.info(f"Reaction removed: {message_id} by {user_id} - {emoji}")

        return {
            "action": "removed",
            "message_id": message_id,
            "emoji": emoji,
            "user_id": user_id
        }

    async def get_message_reactions(
        self,
        message_id: str
    ) -> List[dict]:
        """
        获取消息的所有反应（聚合）

        Returns:
            [
                {
                    "emoji": "👍",
                    "count": 3,
                    "users": ["user_id1", "user_id2", "user_id3"]
                },
                ...
            ]
        """
        from sqlalchemy import and_

        # 查询所有反应
        result = await self.db.execute(
            select(MessageReaction).where(
                MessageReaction.message_id == message_id
            )
        )
        reactions = result.scalars().all()

        # 按表情聚合
        emoji_map: Dict[str, List[str]] = {}
        for reaction in reactions:
            if reaction.emoji not in emoji_map:
                emoji_map[reaction.emoji] = []
            emoji_map[reaction.emoji].append(reaction.user_id)

        # 构建返回数据
        return [
            {
                "emoji": emoji,
                "count": len(users),
                "users": users
            }
            for emoji, users in emoji_map.items()
        ]

    async def get_user_reaction(
        self,
        message_id: str,
        user_id: str
    ) -> List[str]:
        """
        获取用户对消息的反应列表

        Returns:
            ["👍", "❤️"]  # 用户添加的表情列表
        """
        result = await self.db.execute(
            select(MessageReaction.emoji).where(
                and_(
                    MessageReaction.message_id == message_id,
                    MessageReaction.user_id == user_id
                )
            )
        )
        return list(result.scalars().all())

    async def toggle_reaction(
        self,
        message_id: str,
        user_id: str,
        emoji: str
    ) -> dict:
        """
        切换反应（添加或删除）

        Args:
            message_id: 消息 ID
            user_id: 用户 ID
            emoji: 表情字符

        Returns:
            操作结果
        """
        return await self.add_reaction(message_id, user_id, emoji)

    async def _get_message(self, message_id: str) -> Message:
        """获取消息"""
        result = await self.db.execute(
            select(Message).where(Message.id == message_id)
        )
        return result.scalar_one_or_none()

    async def _get_reaction(
        self,
        message_id: str,
        user_id: str,
        emoji: str
    ) -> MessageReaction:
        """获取特定反应"""
        from sqlalchemy import and_

        result = await self.db.execute(
            select(MessageReaction).where(
                and_(
                    MessageReaction.message_id == message_id,
                    MessageReaction.user_id == user_id,
                    MessageReaction.emoji == emoji
                )
            )
        )
        return result.scalar_one_or_none()

    async def _remove_reaction(
        self,
        message_id: str,
        user_id: str,
        emoji: str
    ):
        """删除反应"""
        from sqlalchemy import and_

        await self.db.execute(
            delete(MessageReaction).where(
                and_(
                    MessageReaction.message_id == message_id,
                    MessageReaction.user_id == user_id,
                    MessageReaction.emoji == emoji
                )
            )
        )
        await self.db.commit()
```

### 3. API 路由

```python
# backend/api/routes/reactions.py
from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel
from typing import List

from database import get_db
from services.reaction_service import ReactionService
from middleware.auth import get_current_user

router = APIRouter(prefix="/api/messages", tags=["reactions"])


class ToggleReactionRequest(BaseModel):
    """切换反应请求"""
    emoji: str


class ReactionResponse(BaseModel):
    """反应响应"""
    action: str  # "added" | "removed"
    message_id: str
    emoji: str
    user_id: str


@router.post(
    "/{message_id}/reactions/toggle",
    response_model=ReactionResponse,
    status_code=status.HTTP_200_OK
)
async def toggle_reaction(
    message_id: str,
    request: ToggleReactionRequest,
    current_user = Depends(get_current_user),
    db = Depends(get_db)
):
    """切换消息表情反应"""
    try:
        service = ReactionService(db)
        result = await service.toggle_reaction(
            message_id=message_id,
            user_id=current_user.id,
            emoji=request.emoji
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
            detail="Failed to toggle reaction"
        )


@router.delete(
    "/{message_id}/reactions",
    status_code=status.HTTP_200_OK
)
async def remove_reaction(
    message_id: str,
    emoji: str,
    current_user = Depends(get_current_user),
    db = Depends(get_db)
):
    """删除消息表情反应"""
    try:
        service = ReactionService(db)
        result = await service.remove_reaction(
            message_id=message_id,
            user_id=current_user.id,
            emoji=emoji
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
            detail="Failed to remove reaction"
        )


@router.get(
    "/{message_id}/reactions",
    status_code=status.HTTP_200_OK
)
async def get_reactions(
    message_id: str,
    current_user = Depends(get_current_user),
    db = Depends(get_db)
):
    """获取消息的所有反应"""
    try:
        service = ReactionService(db)
        reactions = await service.get_message_reactions(message_id)
        return {"reactions": reactions}
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to get reactions"
        )


@router.get(
    "/available-emojis",
    status_code=status.HTTP_200_OK
)
async def get_available_emojis():
    """获取可用的表情列表"""
    return {
        "emojis": ReactionService.AVAILABLE_EMOJIS
    }
```

### 4. WebSocket 事件处理

```python
# backend/handlers/reaction_events.py
from socketio import AsyncServer
from loguru import logger

from services.reaction_service import ReactionService


class ReactionEventHandler:
    """反应相关 WebSocket 事件处理"""

    def __init__(self, sio: AsyncServer):
        self.sio = sio
        self._setup_handlers()

    def _setup_handlers(self):
        """设置事件处理器"""

        @self.sio.event
        async def toggle_reaction(sid, data):
            """
            切换表情反应

            data: {
                message_id: str,
                emoji: str,
                campaign_id: str
            }
            """
            from database import get_db_context

            message_id = data.get("message_id")
            emoji = data.get("emoji")
            campaign_id = data.get("campaign_id")

            user_info = self._get_socket_user(sid)
            if not user_info:
                await self._emit_error(sid, "Unauthorized")
                return

            try:
                async with get_db_context() as db:
                    service = ReactionService(db)
                    result = await service.toggle_reaction(
                        message_id=message_id,
                        user_id=user_info["user_id"],
                        emoji=emoji
                    )

                    # 获取更新后的反应列表
                    reactions = await service.get_message_reactions(message_id)

                    # 广播到房间
                    await self.sio.emit(
                        "reaction_updated",
                        {
                            "message_id": message_id,
                            "action": result["action"],
                            "emoji": emoji,
                            "user_id": result["user_id"],
                            "username": user_info["username"],
                            "reactions": reactions
                        },
                        room=campaign_id
                    )

                    # 向操作者确认
                    await self.sio.emit(
                        "reaction_toggle_success",
                        result,
                        room=sid
                    )

                    logger.info(f"Reaction toggled: {message_id} - {emoji}")

            except ValueError as e:
                await self._emit_error(sid, str(e), "INVALID_OPERATION")
            except Exception as e:
                logger.error(f"Error toggling reaction: {e}")
                await self._emit_error(sid, "Failed to toggle reaction")

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

### 1. 反应 Hook

```typescript
// frontend/src/hooks/useMessageReactions.ts
import { useCallback, useEffect, useState } from "react";
import { Socket } from "socket.io-client";
import axios from "axios";

interface Reaction {
  emoji: string;
  count: number;
  users: string[];
}

interface UseMessageReactionsOptions {
  socket?: Socket | null;
  campaignId: string;
  userId: string;
}

export function useMessageReactions({
  socket,
  campaignId,
  userId
}: UseMessageReactionsOptions) {
  const [reactions, setReactions] = useState<Record<string, Reaction[]>>({});

  // 切换反应
  const toggleReaction = useCallback(async (
    messageId: string,
    emoji: string
  ): Promise<void> => {
    try {
      const response = await axios.post(
        `/api/messages/${messageId}/reactions/toggle`,
        { emoji }
      );

      // 本地更新状态（乐观更新）
      setReactions(prev => {
        const messageReactions = prev[messageId] || [];
        const existingIndex = messageReactions.findIndex(r => r.emoji === emoji);

        if (existingIndex >= 0) {
          // 反应已存在，根据 action 决定
          if (response.data.action === "removed") {
            const updated = [...messageReactions];
            const reaction = updated[existingIndex];
            if (reaction.count <= 1) {
              updated.splice(existingIndex, 1);
            } else {
              updated[existingIndex] = {
                ...reaction,
                count: reaction.count - 1,
                users: reaction.users.filter(id => id !== userId)
              };
            }
            return { ...prev, [messageId]: updated };
          }
        }

        // 添加新反应
        const existingReaction = messageReactions.find(r => r.emoji === emoji);
        if (existingReaction) {
          return {
            ...prev,
            [messageId]: messageReactions.map(r =>
              r.emoji === emoji
                ? { ...r, count: r.count + 1, users: [...r.users, userId] }
                : r
            )
          };
        }

        return {
          ...prev,
          [messageId]: [...messageReactions, { emoji, count: 1, users: [userId] }]
        };
      });
    } catch (error) {
      if (axios.isAxiosError(error)) {
        throw new Error(error.response?.data?.detail || "Failed to toggle reaction");
      }
      throw error;
    }
  }, [userId]);

  // 获取消息反应
  const fetchReactions = useCallback(async (messageId: string) => {
    try {
      const response = await axios.get(`/api/messages/${messageId}/reactions`);
      setReactions(prev => ({
        ...prev,
        [messageId]: response.data.reactions
      }));
    } catch (error) {
      console.error("Failed to fetch reactions:", error);
    }
  }, []);

  // 监听 WebSocket 反应更新
  useEffect(() => {
    if (!socket) return;

    const handleReactionUpdated = (data: {
      message_id: string;
      action: string;
      emoji: string;
      user_id: string;
      username: string;
      reactions: Reaction[];
    }) => {
      setReactions(prev => ({
        ...prev,
        [data.message_id]: data.reactions
      }));
    };

    socket.on("reaction_updated", handleReactionUpdated);

    return () => {
      socket.off("reaction_updated", handleReactionUpdated);
    };
  }, [socket]);

  // 获取用户对消息的反应
  const getUserReactions = useCallback((messageId: string): string[] => {
    const messageReactions = reactions[messageId] || [];
    return messageReactions
      .filter(r => r.users.includes(userId))
      .map(r => r.emoji);
  }, [reactions, userId]);

  return {
    reactions,
    toggleReaction,
    fetchReactions,
    getUserReactions
  };
}
```

### 2. 表情选择器组件

```typescript
// frontend/src/components/room/EmojiPicker.tsx
import React, { useState } from "react";
import { Smile } from "lucide-react";

interface EmojiPickerProps {
  onSelect: (emoji: string) => void;
  onClose: () => void;
}

const AVAILABLE_EMOJIS = [
  "👍", "👎", "❤️", "🔥", "😂", "😮", "😢", "🎉",
  "🚀", "💯", "✨", "🤔", "👀", "👏", "🙏", "💪"
];

export function EmojiPicker({ onSelect, onClose }: EmojiPickerProps) {
  return (
    <div className="emoji-picker">
      <div className="emoji-picker-header">
        <span>React with an emoji</span>
        <button onClick={onClose} className="close-btn">×</button>
      </div>

      <div className="emoji-picker-grid">
        {AVAILABLE_EMOJIS.map(emoji => (
          <button
            key={emoji}
            onClick={() => {
              onSelect(emoji);
              onClose();
            }}
            className="emoji-button"
          >
            {emoji}
          </button>
        ))}
      </div>
    </div>
  );
}
```

### 3. 反应气泡组件

```typescript
// frontend/src/components/room/ReactionBubble.tsx
import React, { useState } from "react";
import { useRoomState } from "@/contexts/RoomStateContext";
import { useMessageReactions } from "@/hooks/useMessageReactions";
import { EmojiPicker } from "./EmojiPicker";

interface ReactionBubbleProps {
  messageId: string;
}

export function ReactionBubble({ messageId }: ReactionBubbleProps) {
  const { currentUserId } = useRoomState();
  const { reactions, toggleReaction, getUserReactions } = useMessageReactions({
    campaignId: "", // 从 context 获取
    userId: currentUserId
  });

  const [showPicker, setShowPicker] = useState(false);

  const messageReactions = reactions[messageId] || [];
  const userReactions = getUserReactions(messageId);

  const handleReactionClick = async (emoji: string) => {
    try {
      await toggleReaction(messageId, emoji);
    } catch (error) {
      console.error("Failed to toggle reaction:", error);
    }
  };

  if (messageReactions.length === 0 && !showPicker) {
    return (
      <div className="reaction-bubble">
        <button
          onClick={() => setShowPicker(!showPicker)}
          className="add-reaction-btn"
          title="Add reaction"
        >
          <Smile size={16} />
        </button>

        {showPicker && (
          <EmojiPicker
            onSelect={handleReactionClick}
            onClose={() => setShowPicker(false)}
          />
        )}
      </div>
    );
  }

  return (
    <div className="reaction-bubble">
      <div className="reactions-list">
        {messageReactions.map(reaction => (
          <button
            key={reaction.emoji}
            onClick={() => handleReactionClick(reaction.emoji)}
            className={`reaction-item ${userReactions.includes(reaction.emoji) ? "active" : ""}`}
            title={`Reacted by: ${reaction.users.length} users`}
          >
            <span className="reaction-emoji">{reaction.emoji}</span>
            <span className="reaction-count">{reaction.count}</span>
          </button>
        ))}
      </div>

      <button
        onClick={() => setShowPicker(!showPicker)}
        className="add-reaction-btn"
        title="Add reaction"
      >
        <Smile size={16} />
      </button>

      {showPicker && (
        <EmojiPicker
          onSelect={handleReactionClick}
          onClose={() => setShowPicker(false)}
        />
      )}
    </div>
  );
}
```

### 4. 消息组件（集成反应）

```typescript
// frontend/src/components/room/MessageItem.tsx
import React from "react";
import { Message } from "@/types/message";
import { ReactionBubble } from "./ReactionBubble";

export function MessageItem({ message }: { message: Message }) {
  return (
    <div className="message-item">
      {/* 消息内容 */}

      <div className="message-footer">
        <ReactionBubble messageId={message.id} />
      </div>
    </div>
  );
}
```

### 5. 样式

```css
/* frontend/src/components/room/ReactionBubble.module.css */

.reaction-bubble {
  position: relative;
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 4px;
}

.reactions-list {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.reaction-item {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  background: #f3f4f6;
  border: 1px solid #e5e7eb;
  border-radius: 16px;
  cursor: pointer;
  transition: all 0.2s;
  font-size: 14px;
}

.reaction-item:hover {
  background: #e5e7eb;
}

.reaction-item.active {
  background: #dbeafe;
  border-color: #3b82f6;
}

.reaction-emoji {
  font-size: 16px;
}

.reaction-count {
  font-size: 12px;
  color: #6b7280;
}

.add-reaction-btn {
  padding: 4px 8px;
  background: transparent;
  border: 1px solid #e5e7eb;
  border-radius: 50%;
  cursor: pointer;
  color: #6b7280;
  transition: all 0.2s;
}

.add-reaction-btn:hover {
  background: #f3f4f6;
  color: #374151;
}

/* 表情选择器 */
.emoji-picker {
  position: absolute;
  bottom: 100%;
  left: 0;
  margin-bottom: 8px;
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
  padding: 12px;
  z-index: 50;
  min-width: 280px;
}

.emoji-picker-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 500;
  color: #374151;
}

.close-btn {
  background: none;
  border: none;
  font-size: 20px;
  cursor: pointer;
  color: #6b7280;
  padding: 0;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.emoji-picker-grid {
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  gap: 4px;
}

.emoji-button {
  font-size: 24px;
  padding: 8px;
  background: transparent;
  border: none;
  cursor: pointer;
  border-radius: 4px;
  transition: background 0.2s;
}

.emoji-button:hover {
  background: #f3f4f6;
}
```

---

## 涉及文件清单

### 后端文件

| 文件路径 | 操作 | 说明 |
|---------|------|------|
| `backend/models/message_reaction.py` | 创建 | 消息反应数据模型 |
| `backend/models/message.py` | 修改 | 添加 reactions 关联 |
| `backend/services/reaction_service.py` | 创建 | 反应业务逻辑 |
| `backend/api/routes/reactions.py` | 创建 | 反应 API 路由 |
| `backend/handlers/reaction_events.py` | 创建 | WebSocket 反应事件处理 |

### 前端文件

| 文件路径 | 操作 | 说明 |
|---------|------|------|
| `frontend/src/hooks/useMessageReactions.ts` | 创建 | 消息反应 Hook |
| `frontend/src/components/room/EmojiPicker.tsx` | 创建 | 表情选择器组件 |
| `frontend/src/components/room/ReactionBubble.tsx` | 创建 | 反应气泡组件 |
| `frontend/src/components/room/MessageItem.tsx` | 修改 | 集成反应功能 |
| `frontend/src/components/room/ReactionBubble.module.css` | 创建 | 反应样式 |

---

## 验收标准

### 功能验收

- [ ] 玩家可以添加表情反应到消息
- [ ] 再次点击相同表情可以取消反应
- [ ] 每个表情显示反应数量
- [ ] 反应状态实时同步到所有玩家
- [ ] 支持多种表情选择
- [ ] 用户可以看到自己添加的反应（高亮显示）
- [ ] 悬停显示反应用户列表

### UX 验收

- [ ] 表情选择器易于使用
- [ ] 反应气泡布局合理
- [ ] 交互反馈及时
- [ ] 支持键盘快捷键（可选）
- [ ] 移动端友好

---

## 参考文档

- [Slack 反应功能](https://slack.com/help/articles/201357156-Add-reactions-to-messages)
- [Discord 表情反应](https://discord.com/blog/react-to-express-yourself-even-more)
- [React 组件设计模式](https://react.dev/reference/react)
- [WebSocket 实时更新](https://socket.io/docs/v4/emitting-events/)

---

**创建日期**: 2026-02-06
**负责人**: 待分配
**审核人**: 待分配
