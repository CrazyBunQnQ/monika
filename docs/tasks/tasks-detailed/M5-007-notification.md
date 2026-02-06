# M5-007: 实现通知系统

**任务ID**: M5-007
**标题**: 实现通知系统
**类型**: fullstack (全栈开发)
**预估工时**: 2h
**依赖**: M2-002

---

## 任务描述

实现实时通知系统，支持各种类型的通知（消息、系统通知、事件提醒等）的发送和接收。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M5-007-01 | 设计通知数据结构 | Data Model | 15min |
| M5-007-02 | 实现通知服务 | Notification Service | 25min |
| M5-007-03 | 实现 WebSocket 通知 | WS Notification | 25min |
| M5-007-04 | 实现通知组件 | UI Component | 30min |
| M5-007-05 | 实现通知设置 | Settings | 25min |
| M5-007-06 | 实现通知历史 | History | 15min |
| M5-007-07 | 编写通知测试 | 测试覆盖 | 15min |

---

## 通知数据模型

```python
# app/db/models/notification.py
from sqlalchemy import Column, String, Text, ForeignKey, Boolean, JSON, DateTime
from sqlalchemy.orm import relationship
from sqlalchemy.sql import func
from app.db.database import Base

class Notification(Base):
    """通知"""
    __tablename__ = 'notifications'

    id = Column(String, primary_key=True, index=True)
    user_id = Column(String, ForeignKey('users.id'), nullable=False, index=True)
    room_id = Column(String, ForeignKey('rooms.id'), nullable=False, index=True)

    # 基本信息
    type = Column(String, nullable=False, index=True)  # message, system, event, alert
    title = Column(String, nullable=False)
    content = Column(Text)

    # 优先级
    priority = Column(String, default='normal')  # low, normal, high, urgent

    # 数据
    data = Column(JSON)  # 额外数据

    # 状态
    is_read = Column(Boolean, default=False, nullable=False, index=True)
    read_at = Column(DateTime)

    # 过期时间
    expires_at = Column(DateTime)

    # 时间
    created_at = Column(DateTime, default=func.now(), nullable=False, index=True)

    # 关系
    user = relationship("User", back_populates="notifications")
    room = relationship("Room", back_populates="notifications")

    def __repr__(self):
        return f"<Notification {self.title}>"
```

---

## 通知服务

```python
# app/services/notification.py
from typing import List, Dict, Any, Optional
from sqlalchemy.orm import Session
from datetime import datetime, timedelta

from app.db.models.notification import Notification
from app.core.security import generate_id

class NotificationService:
    """通知服务"""

    def __init__(self, db: Session):
        self.db = db

    def create_notification(
        self,
        user_id: str,
        room_id: str,
        type: str,
        title: str,
        content: str = None,
        priority: str = 'normal',
        data: Dict = None,
        expires_in: int = None,  # 秒
    ) -> Notification:
        """创建通知"""
        notification = Notification(
            id=generate_id('notification'),
            user_id=user_id,
            room_id=room_id,
            type=type,
            title=title,
            content=content,
            priority=priority,
            data=data or {},
            expires_at=datetime.now() + timedelta(seconds=expires_in) if expires_in else None,
        )

        self.db.add(notification)
        self.db.commit()
        self.db.refresh(notification)

        return notification

    def get_user_notifications(
        self,
        user_id: str,
        unread_only: bool = False,
        limit: int = 50,
    ) -> List[Notification]:
        """获取用户通知"""
        query = self.db.query(Notification)\
            .filter(Notification.user_id == user_id)

        if unread_only:
            query = query.filter(Notification.is_read == False)

        # 过滤未过期的通知
        query = query.filter(
            (Notification.expires_at == None) |
            (Notification.expires_at > datetime.now())
        )

        return query\
            .order_by(Notification.created_at.desc())\
            .limit(limit)\
            .all()

    def mark_as_read(
        self,
        notification_id: str,
        user_id: str,
    ) -> Optional[Notification]:
        """标记为已读"""
        notification = self.db.query(Notification)\
            .filter(
                Notification.id == notification_id,
                Notification.user_id == user_id,
            )\
            .first()

        if notification and not notification.is_read:
            notification.is_read = True
            notification.read_at = datetime.now()
            self.db.commit()
            self.db.refresh(notification)

        return notification

    def mark_all_as_read(self, user_id: str) -> int:
        """标记所有通知为已读"""
        count = self.db.query(Notification)\
            .filter(
                Notification.user_id == user_id,
                Notification.is_read == False,
            )\
            .update({
                'is_read': True,
                'read_at': datetime.now(),
            })

        self.db.commit()
        return count

    def delete_notification(
        self,
        notification_id: str,
        user_id: str,
    ) -> bool:
        """删除通知"""
        notification = self.db.query(Notification)\
            .filter(
                Notification.id == notification_id,
                Notification.user_id == user_id,
            )\
            .first()

        if not notification:
            return False

        self.db.delete(notification)
        self.db.commit()

        return True

    def get_unread_count(self, user_id: str) -> int:
        """获取未读通知数量"""
        return self.db.query(Notification)\
            .filter(
                Notification.user_id == user_id,
                Notification.is_read == False,
                (Notification.expires_at == None) |
                (Notification.expires_at > datetime.now()),
            )\
            .count()

    def cleanup_expired_notifications(self) -> int:
        """清理过期通知"""
        count = self.db.query(Notification)\
            .filter(
                Notification.expires_at < datetime.now(),
                Notification.is_read == True,
            )\
            .delete()

        self.db.commit()
        return count

    def broadcast_to_room(
        self,
        room_id: str,
        type: str,
        title: str,
        content: str = None,
        priority: str = 'normal',
        data: Dict = None,
        exclude_user_id: str = None,
    ) -> List[Notification]:
        """向房间内所有用户广播通知"""
        # 获取房间成员
        from app.db.models.room import RoomParticipant
        participants = self.db.query(RoomParticipant)\
            .filter(RoomParticipant.room_id == room_id)\
            .all()

        notifications = []
        for participant in participants:
            if exclude_user_id and participant.user_id == exclude_user_id:
                continue

            notification = self.create_notification(
                user_id=participant.user_id,
                room_id=room_id,
                type=type,
                title=title,
                content=content,
                priority=priority,
                data=data,
            )
            notifications.append(notification)

        return notifications
```

---

## 通知 API

```python
# app/api/notifications.py
from fastapi import APIRouter, Depends, Query, WebSocket, WebSocketDisconnect
from sqlalchemy.orm import Session
from typing import List
from pydantic import BaseModel

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.notification import NotificationService

router = APIRouter(prefix="/notifications", tags=["notifications"])

class CreateNotificationRequest(BaseModel):
    user_id: str
    room_id: str
    type: str
    title: str
    content: str = None
    priority: str = 'normal'
    data: dict = None

@router.get("")
async def list_notifications(
    unread_only: bool = False,
    limit: int = Query(50, ge=1, le=100),
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取通知列表"""
    service = NotificationService(db)
    notifications = service.get_user_notifications(
        current_user.id,
        unread_only=unread_only,
        limit=limit,
    )

    return [
        {
            "id": n.id,
            "type": n.type,
            "title": n.title,
            "content": n.content,
            "priority": n.priority,
            "is_read": n.is_read,
            "data": n.data,
            "created_at": n.created_at.isoformat(),
        }
        for n in notifications
    ]

@router.get("/unread-count")
async def get_unread_count(
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取未读数量"""
    service = NotificationService(db)
    count = service.get_unread_count(current_user.id)

    return {"count": count}

@router.post("/{notification_id}/read")
async def mark_as_read(
    notification_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """标记为已读"""
    service = NotificationService(db)
    notification = service.mark_as_read(notification_id, current_user.id)

    if not notification:
        raise HTTPException(status_code=404, detail="通知不存在")

    return {"message": "已标记为已读"}

@router.post("/read-all")
async def mark_all_as_read(
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """标记所有为已读"""
    service = NotificationService(db)
    count = service.mark_all_as_read(current_user.id)

    return {"count": count, "message": f"已标记 {count} 条通知为已读"}

@router.delete("/{notification_id}")
async def delete_notification(
    notification_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """删除通知"""
    service = NotificationService(db)
    success = service.delete_notification(notification_id, current_user.id)

    if not success:
        raise HTTPException(status_code=404, detail="通知不存在")

    return {"message": "通知已删除"}

# WebSocket 端点用于实时通知
@router.websocket("/ws")
async def notification_websocket(
    websocket: WebSocket,
    token: str,
    db: Session = Depends(get_db),
):
    """通知 WebSocket"""
    # 验证 token
    user = await get_current_user(token, db)
    await websocket.accept()

    try:
        while True:
            # 保持连接，接收心跳
            data = await websocket.receive_text()

            if data == "ping":
                await websocket.send_json({"type": "pong"})

    except WebSocketDisconnect:
        pass
```

---

## 前端通知组件

```tsx
// frontend/src/components/game/NotificationCenter.tsx
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Bell, Check, Trash2, X, AlertCircle, Info, CheckCircle, AlertTriangle } from 'lucide-react'
import { useToast } from '@/hooks/use-toast'
import { useWebSocket } from '@/hooks/useWebSocket'

interface Notification {
  id: string
  type: string
  title: string
  content?: string
  priority: string
  is_read: boolean
  data?: any
  created_at: string
}

interface NotificationCenterProps {
  userId: string
  roomId: string
}

export function NotificationCenter({ userId, roomId }: NotificationCenterProps) {
  const [notifications, setNotifications] = useState<Notification[]>([])
  const [unreadCount, setUnreadCount] = useState(0)
  const [showPanel, setShowPanel] = useState(false)

  const { toast } = useToast()

  // WebSocket 连接接收实时通知
  const { lastMessage } = useWebSocket(`ws://localhost:8000/api/notifications/ws?token=${localStorage.getItem('token')}`)

  useEffect(() => {
    if (lastMessage) {
      const data = JSON.parse(lastMessage.data)
      if (data.type === 'notification') {
        setNotifications(prev => [data.notification, ...prev])
        setUnreadCount(prev => prev + 1)

        // 显示 toast
        toast({
          title: data.notification.title,
          description: data.notification.content,
        })
      }
    }
  }, [lastMessage])

  useEffect(() => {
    loadNotifications()
    loadUnreadCount()

    // 定期刷新未读数
    const interval = setInterval(loadUnreadCount, 30000)
    return () => clearInterval(interval)
  }, [])

  const loadNotifications = async () => {
    try {
      const response = await fetch('/api/notifications?limit=20')
      if (!response.ok) throw new Error('加载失败')

      const data = await response.json()
      setNotifications(data)
    } catch (error) {
      console.error('Failed to load notifications:', error)
    }
  }

  const loadUnreadCount = async () => {
    try {
      const response = await fetch('/api/notifications/unread-count')
      if (!response.ok) throw new Error('加载失败')

      const data = await response.json()
      setUnreadCount(data.count)
    } catch (error) {
      console.error('Failed to load unread count:', error)
    }
  }

  const markAsRead = async (notificationId: string) => {
    try {
      await fetch(`/api/notifications/${notificationId}/read`, {
        method: 'POST',
      })

      setNotifications(prev =>
        prev.map(n => n.id === notificationId ? { ...n, is_read: true } : n)
      )
      setUnreadCount(Math.max(0, unreadCount - 1))
    } catch (error) {
      console.error('Failed to mark as read:', error)
    }
  }

  const markAllAsRead = async () => {
    try {
      await fetch('/api/notifications/read-all', {
        method: 'POST',
      })

      setNotifications(prev => prev.map(n => ({ ...n, is_read: true })))
      setUnreadCount(0)
    } catch (error) {
      console.error('Failed to mark all as read:', error)
    }
  }

  const deleteNotification = async (notificationId: string) => {
    try {
      await fetch(`/api/notifications/${notificationId}`, {
        method: 'DELETE',
      })

      setNotifications(prev => prev.filter(n => n.id !== notificationId))
      if (notifications.find(n => n.id === notificationId)?.is_read === false) {
        setUnreadCount(Math.max(0, unreadCount - 1))
      }
    } catch (error) {
      console.error('Failed to delete notification:', error)
    }
  }

  const getPriorityIcon = (priority: string) => {
    switch (priority) {
      case 'urgent':
        return <AlertCircle className="h-4 w-4 text-red-500" />
      case 'high':
        return <AlertTriangle className="h-4 w-4 text-orange-500" />
      case 'low':
        return <Info className="h-4 w-4 text-blue-500" />
      default:
        return <Bell className="h-4 w-4 text-muted-foreground" />
    }
  }

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'urgent':
        return 'border-red-500 bg-red-50 dark:bg-red-900/20'
      case 'high':
        return 'border-orange-500 bg-orange-50 dark:bg-orange-900/20'
      case 'low':
        return 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
      default:
        return 'border-muted'
    }
  }

  return (
    <>
      {/* 通知按钮 */}
      <Button
        variant="ghost"
        size="sm"
        className="relative"
        onClick={() => {
          setShowPanel(!showPanel)
          if (showPanel) loadNotifications()
        }}
      >
        <Bell className="h-4 w-4" />
        {unreadCount > 0 && (
          <Badge
            variant="destructive"
            className="absolute -top-1 -right-1 h-5 w-5 flex items-center justify-center p-0 text-xs"
          >
            {unreadCount > 9 ? '9+' : unreadCount}
          </Badge>
        )}
      </Button>

      {/* 通知面板 */}
      {showPanel && (
        <Card className="absolute right-0 top-12 w-80 max-h-[500px] overflow-hidden shadow-lg z-50">
          <CardHeader className="pb-3">
            <CardTitle className="text-base flex items-center justify-between">
              <span>通知</span>
              <div className="flex items-center space-x-2">
                {unreadCount > 0 && (
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={markAllAsRead}
                    className="text-xs"
                  >
                    <Check className="h-3 w-3 mr-1" />
                    全部已读
                  </Button>
                )}
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => setShowPanel(false)}
                >
                  <X className="h-4 w-4" />
                </Button>
              </div>
            </CardTitle>
          </CardHeader>

          <CardContent className="p-0">
            <div className="max-h-[400px] overflow-y-auto">
              {notifications.length === 0 ? (
                <div className="text-center py-8 text-sm text-muted-foreground">
                  暂无通知
                </div>
              ) : (
                <div className="divide-y">
                  {notifications.map((notification) => (
                    <div
                      key={notification.id}
                      className={`p-3 hover:bg-muted/50 transition-colors ${
                        !notification.is_read ? 'bg-muted/30' : ''
                      }`}
                    >
                      <div className="flex items-start space-x-3">
                        {getPriorityIcon(notification.priority)}

                        <div className="flex-1 min-w-0">
                          <div className="flex items-center justify-between">
                            <p className="text-sm font-medium truncate">
                              {notification.title}
                            </p>
                            <span className="text-xs text-muted-foreground whitespace-nowrap ml-2">
                              {new Date(notification.created_at).toLocaleTimeString('zh-CN', {
                                hour: '2-digit',
                                minute: '2-digit',
                              })}
                            </span>
                          </div>
                          {notification.content && (
                            <p className="text-xs text-muted-foreground mt-1 line-clamp-2">
                              {notification.content}
                            </p>
                          )}
                        </div>

                        <div className="flex space-x-1">
                          {!notification.is_read && (
                            <Button
                              size="sm"
                              variant="ghost"
                              onClick={() => markAsRead(notification.id)}
                            >
                              <Check className="h-3 w-3" />
                            </Button>
                          )}
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => deleteNotification(notification.id)}
                          >
                            <Trash2 className="h-3 w-3" />
                          </Button>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}
    </>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/db/models/notification.py` | 创建 | 通知数据模型 |
| `app/services/notification.py` | 创建 | 通知服务 |
| `app/api/notifications.py` | 创建 | 通知 API |
| `frontend/src/components/game/NotificationCenter.tsx` | 创建 | 通知中心组件 |
| `frontend/src/hooks/useWebSocket.ts` | 创建 | WebSocket Hook |

---

## 验收标准

- [ ] 通知发送成功
- [ ] 实时接收有效
- [ ] 已读状态正确
- [ ] 未读计数准确
- [ ] 优先级显示清晰
- [ ] 过期通知清理

---

## 参考文档

- M2-002: WebSocket 事件系统

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
