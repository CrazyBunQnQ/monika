# M2-019: 实现房间公告系统

**任务类型**: Backend + Frontend
**预估工时**: 8h
**优先级**: P0
**依赖**: M2-012 (房间状态同步)

---

## 任务描述

实现房间公告系统。KP 可以创建、编辑、删除房间公告，公告会以醒目的方式展示给所有房间成员。支持富文本、过期时间、重要级别标记。公告变更实时同步。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 | 依赖 |
|----|--------|------|----------|------|
| M2-019-01 | 设计公告数据结构 | Announcement 表和字段 | 1h | M2-012 |
| M2-019-02 | 实现后端公告 CRUD API | 创建、读取、更新、删除 | 2h | M2-019-01 |
| M2-019-03 | 实现公告优先级和过期 | 重要性标记、自动过期 | 1h | M2-019-02 |
| M2-019-04 | 实现 WebSocket 公告事件 | 实时广播公告变更 | 1.5h | M2-019-03 |
| M2-019-05 | 实现前端公告编辑器 | KP 公告管理界面 | 1.5h | M2-019-04 |
| M2-019-06 | 实现前端公告展示 | 公告横幅和列表 | 1h | M2-019-05 |

---

## 后端代码示例

### 1. 公告数据模型

```python
# backend/models/announcement.py
from sqlalchemy import Column, String, DateTime, Text, Integer, ForeignKey, Boolean
from sqlalchemy.orm import relationship
from datetime import datetime
from database import Base
from enum import Enum


class AnnouncementPriority(str, Enum):
    """公告优先级"""
    INFO = "info"           # 普通信息
    IMPORTANT = "important" # 重要
    URGENT = "urgent"       # 紧急


class Announcement(Base):
    """房间公告"""
    __tablename__ = "announcements"

    id = Column(String, primary_key=True)
    campaign_id = Column(String, ForeignKey("campaigns.id"), nullable=False)
    title = Column(String, nullable=False)
    content = Column(Text, nullable=False)
    priority = Column(String, default=AnnouncementPriority.INFO.value, nullable=False)

    # 创建者信息
    created_by = Column(String, ForeignKey("users.id"), nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow, nullable=False)

    # 更新信息
    updated_by = Column(String, ForeignKey("users.id"))
    updated_at = Column(DateTime)

    # 过期时间（可选）
    expires_at = Column(DateTime)

    # 是否置顶
    is_pinned = Column(Boolean, default=False)

    # 是否已删除
    is_deleted = Column(Boolean, default=False)
    deleted_at = Column(DateTime)

    # 关联
    campaign = relationship("Campaign", back_populates="announcements")
    creator = relationship("User", foreign_keys=[created_by])
    updater = relationship("User", foreign_keys=[updated_by])

    def to_dict(self):
        return {
            "id": self.id,
            "campaign_id": self.campaign_id,
            "title": self.title,
            "content": self.content,
            "priority": self.priority,
            "created_by": self.created_by,
            "created_at": self.created_at.isoformat(),
            "updated_by": self.updated_by,
            "updated_at": self.updated_at.isoformat() if self.updated_at else None,
            "expires_at": self.expires_at.isoformat() if self.expires_at else None,
            "is_pinned": self.is_pinned,
            "is_deleted": self.is_deleted
        }

    @property
    def is_expired(self):
        """是否已过期"""
        if not self.expires_at:
            return False
        return datetime.utcnow() > self.expires_at
```

```python
# backend/models/campaign.py (修改)
# 添加关联
from sqlalchemy.orm import relationship

class Campaign(Base):
    # ... 现有字段 ...

    # 添加关联
    announcements = relationship("Announcement", back_populates="campaign", cascade="all, delete-orphan")
```

### 2. 公告服务

```python
# backend/services/announcement_service.py
from typing import List, Optional
from datetime import datetime, timedelta
from loguru import logger
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select, and_, or_, desc
from uuid import uuid4

from models.announcement import Announcement, AnnouncementPriority
from models.campaign import CampaignMember


class AnnouncementService:
    """公告服务"""

    def __init__(self, db: AsyncSession):
        self.db = db

    async def create_announcement(
        self,
        campaign_id: str,
        title: str,
        content: str,
        created_by: str,
        priority: str = AnnouncementPriority.INFO.value,
        expires_in_hours: Optional[int] = None,
        is_pinned: bool = False
    ) -> dict:
        """
        创建公告

        Args:
            campaign_id: 战役 ID
            title: 公告标题
            content: 公告内容（支持 Markdown）
            created_by: 创建者 ID（必须是 KP）
            priority: 优先级（info/important/urgent）
            expires_in_hours: 过期时间（小时）
            is_pinned: 是否置顶

        Returns:
            创建的公告
        """
        # 1. 验证权限
        member = await self._get_member(campaign_id, created_by)
        if not member or member.role != "kp":
            raise PermissionError("Only KP can create announcements")

        # 2. 计算过期时间
        expires_at = None
        if expires_in_hours:
            expires_at = datetime.utcnow() + timedelta(hours=expires_in_hours)

        # 3. 创建公告
        announcement = Announcement(
            id=str(uuid4()),
            campaign_id=campaign_id,
            title=title,
            content=content,
            priority=priority,
            created_by=created_by,
            expires_at=expires_at,
            is_pinned=is_pinned
        )

        self.db.add(announcement)
        await self.db.commit()
        await self.db.refresh(announcement)

        logger.info(
            f"Announcement created: {announcement.id} in {campaign_id} "
            f"by {created_by}"
        )

        return announcement.to_dict()

    async def update_announcement(
        self,
        announcement_id: str,
        operator_id: str,
        title: Optional[str] = None,
        content: Optional[str] = None,
        priority: Optional[str] = None,
        expires_at: Optional[datetime] = None,
        is_pinned: Optional[bool] = None
    ) -> dict:
        """
        更新公告

        Args:
            announcement_id: 公告 ID
            operator_id: 操作者 ID（必须是 KP）
            title: 新标题
            content: 新内容
            priority: 新优先级
            expires_at: 新过期时间
            is_pinned: 是否置顶

        Returns:
            更新后的公告
        """
        # 1. 获取公告
        announcement = await self._get_announcement(announcement_id)
        if not announcement:
            raise ValueError("Announcement not found")

        # 2. 验证权限
        member = await self._get_member(announcement.campaign_id, operator_id)
        if not member or member.role != "kp":
            raise PermissionError("Only KP can update announcements")

        # 3. 更新字段
        if title is not None:
            announcement.title = title
        if content is not None:
            announcement.content = content
        if priority is not None:
            announcement.priority = priority
        if expires_at is not None:
            announcement.expires_at = expires_at
        if is_pinned is not None:
            announcement.is_pinned = is_pinned

        announcement.updated_by = operator_id
        announcement.updated_at = datetime.utcnow()

        await self.db.commit()
        await self.db.refresh(announcement)

        logger.info(f"Announcement updated: {announcement_id} by {operator_id}")

        return announcement.to_dict()

    async def delete_announcement(
        self,
        announcement_id: str,
        operator_id: str
    ) -> dict:
        """
        删除公告（软删除）

        Args:
            announcement_id: 公告 ID
            operator_id: 操作者 ID（必须是 KP）

        Returns:
            删除结果
        """
        # 1. 获取公告
        announcement = await self._get_announcement(announcement_id)
        if not announcement:
            raise ValueError("Announcement not found")

        # 2. 验证权限
        member = await self._get_member(announcement.campaign_id, operator_id)
        if not member or member.role != "kp":
            raise PermissionError("Only KP can delete announcements")

        # 3. 软删除
        announcement.is_deleted = True
        announcement.deleted_at = datetime.utcnow()

        await self.db.commit()

        logger.info(f"Announcement deleted: {announcement_id} by {operator_id}")

        return {
            "success": True,
            "announcement_id": announcement_id
        }

    async def get_announcements(
        self,
        campaign_id: str,
        include_deleted: bool = False
    ) -> List[dict]:
        """
        获取战役的所有公告

        Args:
            campaign_id: 战役 ID
            include_deleted: 是否包含已删除

        Returns:
            公告列表（按置顶、优先级、创建时间排序）
        """
        query = select(Announcement).where(
            Announcement.campaign_id == campaign_id
        )

        if not include_deleted:
            query = query.where(Announcement.is_deleted == False)

        # 排序：置顶 > 优先级 > 创建时间
        query = query.order_by(
            desc(Announcement.is_pinned),
            Announcement.priority.desc(),
            desc(Announcement.created_at)
        )

        result = await self.db.execute(query)
        announcements = result.scalars().all()

        # 过滤过期公告
        valid_announcements = [
            a for a in announcements
            if not a.is_expired
        ]

        return [a.to_dict() for a in valid_announcements]

    async def get_announcement(self, announcement_id: str) -> Optional[dict]:
        """获取单个公告"""
        announcement = await self._get_announcement(announcement_id)
        if announcement and not announcement.is_deleted and not announcement.is_expired:
            return announcement.to_dict()
        return None

    async def _get_announcement(self, announcement_id: str) -> Optional[Announcement]:
        """获取公告"""
        result = await self.db.execute(
            select(Announcement).where(Announcement.id == announcement_id)
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
```

### 3. API 路由

```python
# backend/api/routes/announcements.py
from fastapi import APIRouter, Depends, HTTPException, status, Query
from pydantic import BaseModel
from typing import List, Optional
from datetime import datetime

from database import get_db
from services.announcement_service import AnnouncementService
from middleware.auth import get_current_user

router = APIRouter(prefix="/api/campaigns", tags=["announcements"])


class CreateAnnouncementRequest(BaseModel):
    """创建公告请求"""
    title: str
    content: str
    priority: str = "info"  # info | important | urgent
    expires_in_hours: Optional[int] = None
    is_pinned: bool = False


class UpdateAnnouncementRequest(BaseModel):
    """更新公告请求"""
    title: Optional[str] = None
    content: Optional[str] = None
    priority: Optional[str] = None
    expires_at: Optional[datetime] = None
    is_pinned: Optional[bool] = None


@router.post(
    "/{campaign_id}/announcements",
    status_code=status.HTTP_201_CREATED
)
async def create_announcement(
    campaign_id: str,
    request: CreateAnnouncementRequest,
    current_user = Depends(get_current_user),
    db = Depends(get_db)
):
    """创建公告"""
    try:
        service = AnnouncementService(db)
        result = await service.create_announcement(
            campaign_id=campaign_id,
            title=request.title,
            content=request.content,
            created_by=current_user.id,
            priority=request.priority,
            expires_in_hours=request.expires_in_hours,
            is_pinned=request.is_pinned
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
            detail="Failed to create announcement"
        )


@router.get(
    "/{campaign_id}/announcements",
    response_model=List[dict],
    status_code=status.HTTP_200_OK
)
async def get_announcements(
    campaign_id: str,
    include_deleted: bool = Query(False),
    current_user = Depends(get_current_user),
    db = Depends(get_db)
):
    """获取公告列表"""
    try:
        service = AnnouncementService(db)
        result = await service.get_announcements(
            campaign_id=campaign_id,
            include_deleted=include_deleted
        )
        return result

    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to get announcements"
        )


@router.get(
    "/announcements/{announcement_id}",
    status_code=status.HTTP_200_OK
)
async def get_announcement(
    announcement_id: str,
    current_user = Depends(get_current_user),
    db = Depends(get_db)
):
    """获取单个公告"""
    try:
        service = AnnouncementService(db)
        result = await service.get_announcement(announcement_id)
        if not result:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail="Announcement not found"
            )
        return result

    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to get announcement"
        )


@router.put(
    "/announcements/{announcement_id}",
    status_code=status.HTTP_200_OK
)
async def update_announcement(
    announcement_id: str,
    request: UpdateAnnouncementRequest,
    current_user = Depends(get_current_user),
    db = Depends(get_db)
):
    """更新公告"""
    try:
        service = AnnouncementService(db)
        result = await service.update_announcement(
            announcement_id=announcement_id,
            operator_id=current_user.id,
            title=request.title,
            content=request.content,
            priority=request.priority,
            expires_at=request.expires_at,
            is_pinned=request.is_pinned
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
            detail="Failed to update announcement"
        )


@router.delete(
    "/announcements/{announcement_id}",
    status_code=status.HTTP_200_OK
)
async def delete_announcement(
    announcement_id: str,
    current_user = Depends(get_current_user),
    db = Depends(get_db)
):
    """删除公告"""
    try:
        service = AnnouncementService(db)
        result = await service.delete_announcement(
            announcement_id=announcement_id,
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
            detail="Failed to delete announcement"
        )
```

### 4. WebSocket 事件处理

```python
# backend/handlers/announcement_events.py
from socketio import AsyncServer
from loguru import logger


class AnnouncementEventHandler:
    """公告相关 WebSocket 事件处理"""

    def __init__(self, sio: AsyncServer):
        self.sio = sio
        self._setup_handlers()

    def _setup_handlers(self):
        """设置事件处理器"""

        @self.sio.event
        async def announcement_created(sid, data):
            """
            公告创建事件

            data: {
                announcement_id: str,
                campaign_id: str
            }
            """
            from database import get_db_context
            from services.announcement_service import AnnouncementService

            campaign_id = data.get("campaign_id")
            announcement_id = data.get("announcement_id")

            try:
                async with get_db_context() as db:
                    service = AnnouncementService(db)
                    announcement = await service.get_announcement(announcement_id)

                    if announcement:
                        # 广播到房间
                        await self.sio.emit(
                            "announcement_created",
                            announcement,
                            room=campaign_id
                        )

                        logger.info(f"Announcement {announcement_id} broadcasted")

            except Exception as e:
                logger.error(f"Error broadcasting announcement: {e}")

        @self.sio.event
        async def announcement_updated(sid, data):
            """公告更新事件"""
            from database import get_db_context
            from services.announcement_service import AnnouncementService

            campaign_id = data.get("campaign_id")
            announcement_id = data.get("announcement_id")

            try:
                async with get_db_context() as db:
                    service = AnnouncementService(db)
                    announcement = await service.get_announcement(announcement_id)

                    if announcement:
                        await self.sio.emit(
                            "announcement_updated",
                            announcement,
                            room=campaign_id
                        )

                        logger.info(f"Announcement {announcement_id} updated broadcasted")

            except Exception as e:
                logger.error(f"Error broadcasting announcement update: {e}")

        @self.sio.event
        async def announcement_deleted(sid, data):
            """公告删除事件"""
            campaign_id = data.get("campaign_id")
            announcement_id = data.get("announcement_id")

            try:
                await self.sio.emit(
                    "announcement_deleted",
                    {
                        "announcement_id": announcement_id,
                        "campaign_id": campaign_id
                    },
                    room=campaign_id
                )

                logger.info(f"Announcement {announcement_id} deleted broadcasted")

            except Exception as e:
                logger.error(f"Error broadcasting announcement deletion: {e}")
```

---

## 前端代码示例

### 1. 公告 Hook

```typescript
// frontend/src/hooks/useAnnouncements.ts
import { useCallback, useEffect, useState } from "react";
import { Socket } from "socket.io-client";
import axios from "axios";

export interface Announcement {
  id: string;
  campaign_id: string;
  title: string;
  content: string;
  priority: "info" | "important" | "urgent";
  created_by: string;
  created_at: string;
  updated_by?: string;
  updated_at?: string;
  expires_at?: string;
  is_pinned: boolean;
  is_deleted: boolean;
}

interface UseAnnouncementsOptions {
  socket?: Socket | null;
  campaignId: string;
}

export function useAnnouncements({
  socket,
  campaignId
}: UseAnnouncementsOptions) {
  const [announcements, setAnnouncements] = useState<Announcement[]>([]);
  const [loading, setLoading] = useState(true);

  // 获取公告列表
  const fetchAnnouncements = useCallback(async () => {
    setLoading(true);
    try {
      const response = await axios.get(
        `/api/campaigns/${campaignId}/announcements`
      );
      setAnnouncements(response.data);
    } catch (error) {
      console.error("Failed to fetch announcements:", error);
    } finally {
      setLoading(false);
    }
  }, [campaignId]);

  // 创建公告
  const createAnnouncement = useCallback(async (data: {
    title: string;
    content: string;
    priority?: string;
    expires_in_hours?: number;
    is_pinned?: boolean;
  }): Promise<Announcement> => {
    const response = await axios.post(
      `/api/campaigns/${campaignId}/announcements`,
      data
    );
    return response.data;
  }, [campaignId]);

  // 更新公告
  const updateAnnouncement = useCallback(async (
    announcementId: string,
    data: Partial<Announcement>
  ): Promise<Announcement> => {
    const response = await axios.put(
      `/api/announcements/${announcementId}`,
      data
    );
    return response.data;
  }, []);

  // 删除公告
  const deleteAnnouncement = useCallback(async (announcementId: string): Promise<void> => {
    await axios.delete(`/api/announcements/${announcementId}`);
  }, []);

  // 监听 WebSocket 事件
  useEffect(() => {
    if (!socket) return;

    const handleCreated = (announcement: Announcement) => {
      setAnnouncements(prev => [...prev, announcement]);
    };

    const handleUpdated = (announcement: Announcement) => {
      setAnnouncements(prev =>
        prev.map(a => a.id === announcement.id ? announcement : a)
      );
    };

    const handleDeleted = (data: { announcement_id: string }) => {
      setAnnouncements(prev =>
        prev.filter(a => a.id !== data.announcement_id)
      );
    };

    socket.on("announcement_created", handleCreated);
    socket.on("announcement_updated", handleUpdated);
    socket.on("announcement_deleted", handleDeleted);

    return () => {
      socket.off("announcement_created", handleCreated);
      socket.off("announcement_updated", handleUpdated);
      socket.off("announcement_deleted", handleDeleted);
    };
  }, [socket]);

  // 初始加载
  useEffect(() => {
    fetchAnnouncements();
  }, [fetchAnnouncements]);

  return {
    announcements,
    loading,
    refetch: fetchAnnouncements,
    createAnnouncement,
    updateAnnouncement,
    deleteAnnouncement
  };
}
```

### 2. 公告横幅组件

```typescript
// frontend/src/components/room/AnnouncementBanner.tsx
import React from "react";
import { Announcement } from "@/hooks/useAnnouncements";
import { AlertCircle, Info, AlertTriangle, X } from "lucide-react";

interface AnnouncementBannerProps {
  announcement: Announcement;
  onDismiss?: () => void;
}

export function AnnouncementBanner({ announcement, onDismiss }: AnnouncementBannerProps) {
  const priorityConfig = {
    info: {
      icon: Info,
      bgColor: "bg-blue-50 dark:bg-blue-900/20",
      borderColor: "border-blue-200 dark:border-blue-800",
      textColor: "text-blue-800 dark:text-blue-200"
    },
    important: {
      icon: AlertTriangle,
      bgColor: "bg-yellow-50 dark:bg-yellow-900/20",
      borderColor: "border-yellow-200 dark:border-yellow-800",
      textColor: "text-yellow-800 dark:text-yellow-200"
    },
    urgent: {
      icon: AlertCircle,
      bgColor: "bg-red-50 dark:bg-red-900/20",
      borderColor: "border-red-200 dark:border-red-800",
      textColor: "text-red-800 dark:text-red-200"
    }
  };

  const config = priorityConfig[announcement.priority as keyof typeof priorityConfig];
  const Icon = config.icon;

  return (
    <div className={`announcement-banner ${config.bgColor} ${config.borderColor} border`}>
      <div className="announcement-content">
        <Icon className={`announcement-icon ${config.textColor}`} size={20} />

        <div className="announcement-text">
          <h4 className={`announcement-title ${config.textColor}`}>
            {announcement.is_pinned && "📌 "}
            {announcement.title}
          </h4>
          <p className="announcement-message">{announcement.content}</p>
        </div>

        {onDismiss && (
          <button
            onClick={onDismiss}
            className="announcement-dismiss"
            title="Dismiss"
          >
            <X size={16} />
          </button>
        )}
      </div>

      {announcement.expires_at && (
        <div className="announcement-expiry">
          Expires: {new Date(announcement.expires_at).toLocaleString()}
        </div>
      )}
    </div>
  );
}
```

### 3. 公告管理面板（KP）

```typescript
// frontend/src/components/room/AnnouncementManager.tsx
import React, { useState } from "react";
import { useAnnouncements } from "@/hooks/useAnnouncements";
import { useRoomState } from "@/contexts/RoomStateContext";
import { AnnouncementBanner } from "./AnnouncementBanner";
import { AnnouncementEditor } from "./AnnouncementEditor";
import { Plus, Edit, Trash2 } from "lucide-react";
import { toast } from "react-hot-toast";

export function AnnouncementManager() {
  const { roomState, isKP } = useRoomState();
  const { announcements, createAnnouncement, updateAnnouncement, deleteAnnouncement } =
    useAnnouncements({
      campaignId: roomState?.room_id || ""
    });

  const [editingId, setEditingId] = useState<string | null>(null);
  const [showEditor, setShowEditor] = useState(false);

  if (!isKP) {
    // 普通玩家只显示公告
    return (
      <div className="announcement-list">
        {announcements.map(announcement => (
          <AnnouncementBanner key={announcement.id} announcement={announcement} />
        ))}
        {announcements.length === 0 && (
          <p className="no-announcements">No announcements</p>
        )}
      </div>
    );
  }

  // KP 管理界面
  const handleSave = async (data: {
    title: string;
    content: string;
    priority: string;
    expires_in_hours?: number;
    is_pinned: boolean;
  }) => {
    try {
      if (editingId) {
        await updateAnnouncement(editingId, data);
        toast.success("Announcement updated");
      } else {
        await createAnnouncement(data);
        toast.success("Announcement created");
      }
      setEditingId(null);
      setShowEditor(false);
    } catch (error) {
      toast.error("Failed to save announcement");
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Delete this announcement?")) return;

    try {
      await deleteAnnouncement(id);
      toast.success("Announcement deleted");
    } catch (error) {
      toast.error("Failed to delete announcement");
    }
  };

  return (
    <div className="announcement-manager">
      <div className="manager-header">
        <h3>Announcements</h3>
        <button
          onClick={() => {
            setEditingId(null);
            setShowEditor(true);
          }}
          className="create-btn"
        >
          <Plus size={16} />
          New Announcement
        </button>
      </div>

      {showEditor && (
        <AnnouncementEditor
          announcement={editingId ? announcements.find(a => a.id === editingId) : undefined}
          onSave={handleSave}
          onCancel={() => {
            setEditingId(null);
            setShowEditor(false);
          }}
        />
      )}

      <div className="announcement-list">
        {announcements.map(announcement => (
          <div key={announcement.id} className="announcement-item-admin">
            <AnnouncementBanner announcement={announcement} />

            <div className="announcement-actions">
              <button
                onClick={() => {
                  setEditingId(announcement.id);
                  setShowEditor(true);
                }}
                className="action-btn edit"
              >
                <Edit size={16} />
              </button>
              <button
                onClick={() => handleDelete(announcement.id)}
                className="action-btn delete"
              >
                <Trash2 size={16} />
              </button>
            </div>
          </div>
        ))}

        {announcements.length === 0 && (
          <p className="no-announcements">No announcements yet</p>
        )}
      </div>
    </div>
  );
}
```

### 4. 公告编辑器

```typescript
// frontend/src/components/room/AnnouncementEditor.tsx
import React, { useState } from "react";
import { X, Save } from "lucide-react";
import { Announcement } from "@/hooks/useAnnouncements";

interface AnnouncementEditorProps {
  announcement?: Announcement;
  onSave: (data: {
    title: string;
    content: string;
    priority: string;
    expires_in_hours?: number;
    is_pinned: boolean;
  }) => Promise<void>;
  onCancel: () => void;
}

export function AnnouncementEditor({ announcement, onSave, onCancel }: AnnouncementEditorProps) {
  const [title, setTitle] = useState(announcement?.title || "");
  const [content, setContent] = useState(announcement?.content || "");
  const [priority, setPriority] = useState(announcement?.priority || "info");
  const [expiresIn, setExpiresIn] = useState<number | undefined>(undefined);
  const [isPinned, setIsPinned] = useState(announcement?.is_pinned || false);
  const [isSaving, setIsSaving] = useState(false);

  const handleSave = async () => {
    if (!title.trim() || !content.trim()) {
      alert("Title and content are required");
      return;
    }

    setIsSaving(true);
    try {
      await onSave({
        title: title.trim(),
        content: content.trim(),
        priority,
        expires_in_hours: expiresIn || undefined,
        is_pinned: isPinned
      });
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="announcement-editor">
      <div className="editor-header">
        <h3>{announcement ? "Edit Announcement" : "New Announcement"}</h3>
        <button onClick={onCancel} className="close-btn">
          <X size={20} />
        </button>
      </div>

      <div className="editor-form">
        <div className="form-field">
          <label>Title</label>
          <input
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="Announcement title..."
            className="title-input"
          />
        </div>

        <div className="form-field">
          <label>Content</label>
          <textarea
            value={content}
            onChange={(e) => setContent(e.target.value)}
            placeholder="Announcement content (Markdown supported)..."
            rows={6}
            className="content-textarea"
          />
        </div>

        <div className="form-row">
          <div className="form-field">
            <label>Priority</label>
            <select
              value={priority}
              onChange={(e) => setPriority(e.target.value)}
              className="priority-select"
            >
              <option value="info">Info</option>
              <option value="important">Important</option>
              <option value="urgent">Urgent</option>
            </select>
          </div>

          <div className="form-field">
            <label>Expires In (Hours)</label>
            <input
              type="number"
              value={expiresIn || ""}
              onChange={(e) => setExpiresIn(e.target.value ? parseInt(e.target.value) : undefined)}
              placeholder="Optional"
              className="expires-input"
            />
          </div>
        </div>

        <div className="form-field checkbox">
          <label>
            <input
              type="checkbox"
              checked={isPinned}
              onChange={(e) => setIsPinned(e.target.checked)}
            />
            <span>Pin Announcement</span>
          </label>
        </div>

        <div className="editor-actions">
          <button
            onClick={onCancel}
            disabled={isSaving}
            className="cancel-btn"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={isSaving}
            className="save-btn"
          >
            {isSaving ? (
              <>Saving...</>
            ) : (
              <>
                <Save size={16} />
                Save
              </>
            )}
          </button>
        </div>
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
| `backend/models/announcement.py` | 创建 | 公告数据模型 |
| `backend/models/campaign.py` | 修改 | 添加 announcements 关联 |
| `backend/services/announcement_service.py` | 创建 | 公告业务逻辑 |
| `backend/api/routes/announcements.py` | 创建 | 公告 API 路由 |
| `backend/handlers/announcement_events.py` | 创建 | WebSocket 公告事件处理 |

### 前端文件

| 文件路径 | 操作 | 说明 |
|---------|------|------|
| `frontend/src/hooks/useAnnouncements.ts` | 创建 | 公告 Hook |
| `frontend/src/components/room/AnnouncementBanner.tsx` | 创建 | 公告横幅组件 |
| `frontend/src/components/room/AnnouncementManager.tsx` | 创建 | 公告管理组件 |
| `frontend/src/components/room/AnnouncementEditor.tsx` | 创建 | 公告编辑器组件 |

---

## 验收标准

### 功能验收

- [ ] KP 可以创建公告
- [ ] KP 可以编辑/删除公告
- [ ] 公告支持三种优先级（info/important/urgent）
- [ ] 公告支持置顶功能
- [ ] 公告支持过期时间
- [ ] 过期公告自动隐藏
- [ ] 公告变更实时同步
- [ ] 普通玩家可以查看公告

### UX 验收

- [ ] 不同优先级有明显的视觉区分
- [ ] 置顶公告始终显示在顶部
- [ ] 公告编辑器易用
- [ ] 支持 Markdown 格式
- [ ] 移动端友好

---

## 参考文档

- [Discord 公告功能](https://discord.com/blog/announcements-are-here)
- [Slack 公告频道](https://slack.com/help/articles/115004846788-Create-an-announcement-channel)
- [Markdown 渲染](https://marked.js.org/)
- [WebSocket 实时更新](https://socket.io/docs/v4/)

---

**创建日期**: 2026-02-06
**负责人**: 待分配
**审核人**: 待分配
