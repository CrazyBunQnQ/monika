# M3-005: 实现书签功能

**任务ID**: M3-005
**标题**: 实现书签功能
**类型**: fullstack (全栈开发)
**预估工时**: 1.5h
**依赖**: M3-001

---

## 任务描述

实现对战局关键位置的书签功能，允许玩家和 KP 标记重要事件、场景、线索等，方便快速回溯。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M3-005-01 | 设计书签数据结构 | Data Model | 15min |
| M3-005-02 | 实现书签服务 | Bookmark Service | 25min |
| M3-005-03 | 实现书签 API | API | 20min |
| M3-005-04 | 实现书签 UI 组件 | UI Component | 30min |
| M3-005-05 | 实现书签导航 | Navigation | 20min |
| M3-005-06 | 编写书签测试 | 测试覆盖 | 10min |

---

## 书签数据模型

```python
# app/db/models/bookmark.py
from sqlalchemy import Column, String, Text, ForeignKey, DateTime, Boolean
from sqlalchemy.orm import relationship
from sqlalchemy.sql import func
from app.db.database import Base

class Bookmark(Base):
    """书签"""
    __tablename__ = 'bookmarks'

    id = Column(String, primary_key=True, index=True)
    room_id = Column(String, ForeignKey('rooms.id'), nullable=False, index=True)
    campaign_id = Column(String, ForeignKey('campaigns.id'), nullable=False, index=True)

    # 基本信息
    title = Column(String, nullable=False)
    description = Column(Text)
    color = Column(String, default='#3b82f6')  # 十六进制颜色

    # 位置信息
    timestamp = Column(DateTime, nullable=False, index=True)  # 游戏内时间戳
    event_index = Column(Integer)  # 事件索引（用于跳转到特定事件）
    scene_id = Column(String, ForeignKey('scenes.id'))  # 关联场景

    # 标签
    tags = Column(JSON)  # 标签列表

    # 创建者
    created_by = Column(String, ForeignKey('users.id'), nullable=False)

    # 状态
    is_public = Column(Boolean, default=True, nullable=False)  # 是否对所有玩家可见

    # 时间
    created_at = Column(DateTime, default=func.now(), nullable=False)

    # 关系
    room = relationship("Room", back_populates="bookmarks")
    campaign = relationship("Campaign", back_populates="bookmarks")
    creator = relationship("User", back_populates="bookmarks")
    scene = relationship("Scene", back_populates="bookmarks")

    def __repr__(self):
        return f"<Bookmark {self.title}>"
```

---

## 书签服务

```python
# app/services/bookmark.py
from typing import List, Dict, Any, Optional
from sqlalchemy.orm import Session
from datetime import datetime

from app.db.models.bookmark import Bookmark
from app.core.security import generate_id

class BookmarkService:
    """书签服务"""

    def __init__(self, db: Session):
        self.db = db

    def create_bookmark(
        self,
        room_id: str,
        campaign_id: str,
        title: str,
        created_by: str,
        description: str = None,
        color: str = '#3b82f6',
        timestamp: datetime = None,
        event_index: int = None,
        scene_id: str = None,
        tags: List[str] = None,
        is_public: bool = True,
    ) -> Bookmark:
        """创建书签"""
        bookmark = Bookmark(
            id=generate_id('bookmark'),
            room_id=room_id,
            campaign_id=campaign_id,
            title=title,
            description=description,
            color=color,
            timestamp=timestamp or datetime.now(),
            event_index=event_index,
            scene_id=scene_id,
            tags=tags or [],
            created_by=created_by,
            is_public=is_public,
        )

        self.db.add(bookmark)
        self.db.commit()
        self.db.refresh(bookmark)

        return bookmark

    def get_bookmarks(
        self,
        campaign_id: str,
        user_id: str,
        is_public_only: bool = True,
    ) -> List[Bookmark]:
        """获取书签列表"""
        query = self.db.query(Bookmark)\
            .filter(Bookmark.campaign_id == campaign_id)

        if is_public_only:
            # 只显示公开书签和自己的书签
            query = query.filter(
                (Bookmark.is_public == True) | (Bookmark.created_by == user_id)
            )

        return query\
            .order_by(Bookmark.timestamp.desc())\
            .all()

    def get_bookmark(self, bookmark_id: str) -> Optional[Bookmark]:
        """获取单个书签"""
        return self.db.query(Bookmark)\
            .filter(Bookmark.id == bookmark_id)\
            .first()

    def update_bookmark(
        self,
        bookmark_id: str,
        user_id: str,
        **updates,
    ) -> Optional[Bookmark]:
        """更新书签"""
        bookmark = self.get_bookmark(bookmark_id)

        if not bookmark or bookmark.created_by != user_id:
            return None

        for key, value in updates.items():
            if hasattr(bookmark, key):
                setattr(bookmark, key, value)

        self.db.commit()
        self.db.refresh(bookmark)

        return bookmark

    def delete_bookmark(self, bookmark_id: str, user_id: str) -> bool:
        """删除书签"""
        bookmark = self.get_bookmark(bookmark_id)

        if not bookmark or bookmark.created_by != user_id:
            return False

        self.db.delete(bookmark)
        self.db.commit()

        return True

    def search_bookmarks(
        self,
        campaign_id: str,
        user_id: str,
        query: str,
        tags: List[str] = None,
    ) -> List[Bookmark]:
        """搜索书签"""
        bookmarks = self.get_bookmarks(campaign_id, user_id, is_public_only=False)

        # 过滤
        results = []
        for bookmark in bookmarks:
            # 文本搜索
            text_match = (
                query.lower() in bookmark.title.lower() or
                (bookmark.description and query.lower() in bookmark.description.lower())
            )

            # 标签过滤
            tag_match = not tags or any(tag in (bookmark.tags or []) for tag in tags)

            if text_match and tag_match:
                results.append(bookmark)

        return results

    def get_tags_summary(self, campaign_id: str) -> Dict[str, int]:
        """获取标签统计"""
        bookmarks = self.db.query(Bookmark)\
            .filter(Bookmark.campaign_id == campaign_id)\
            .all()

        tag_counts = {}
        for bookmark in bookmarks:
            for tag in (bookmark.tags or []):
                tag_counts[tag] = tag_counts.get(tag, 0) + 1

        return tag_counts
```

---

## 书签 API

```python
# app/api/bookmarks.py
from fastapi import APIRouter, Depends, Query
from sqlalchemy.orm import Session
from typing import List, Optional
from pydantic import BaseModel

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.bookmark import BookmarkService
from app.schemas.bookmark import BookmarkCreate, BookmarkResponse, BookmarkUpdate

router = APIRouter(prefix="/bookmarks", tags=["bookmarks"])

@router.post("")
async def create_bookmark(
    bookmark_data: BookmarkCreate,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """创建书签"""
    service = BookmarkService(db)

    bookmark = service.create_bookmark(
        room_id=bookmark_data.room_id,
        campaign_id=bookmark_data.campaign_id,
        title=bookmark_data.title,
        created_by=current_user.id,
        description=bookmark_data.description,
        color=bookmark_data.color,
        timestamp=bookmark_data.timestamp,
        event_index=bookmark_data.event_index,
        scene_id=bookmark_data.scene_id,
        tags=bookmark_data.tags,
        is_public=bookmark_data.is_public,
    )

    # 通知房间成员
    # TODO: WebSocket 事件

    return BookmarkResponse.from_orm(bookmark)

@router.get("")
async def list_bookmarks(
    campaign_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取书签列表"""
    service = BookmarkService(db)
    bookmarks = service.get_bookmarks(campaign_id, current_user.id)

    return [BookmarkResponse.from_orm(b) for b in bookmarks]

@router.get("/{bookmark_id}")
async def get_bookmark(
    bookmark_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取单个书签"""
    service = BookmarkService(db)
    bookmark = service.get_bookmark(bookmark_id)

    if not bookmark:
        raise HTTPException(status_code=404, detail="书签不存在")

    # 权限检查
    if not bookmark.is_public and bookmark.created_by != current_user.id:
        raise HTTPException(status_code=403, detail="无权访问此书签")

    return BookmarkResponse.from_orm(bookmark)

@router.put("/{bookmark_id}")
async def update_bookmark(
    bookmark_id: str,
    bookmark_data: BookmarkUpdate,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """更新书签"""
    service = BookmarkService(db)
    bookmark = service.update_bookmark(
        bookmark_id,
        current_user.id,
        **bookmark_data.dict(exclude_unset=True),
    )

    if not bookmark:
        raise HTTPException(status_code=404, detail="书签不存在或无权修改")

    return BookmarkResponse.from_orm(bookmark)

@router.delete("/{bookmark_id}")
async def delete_bookmark(
    bookmark_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """删除书签"""
    service = BookmarkService(db)
    success = service.delete_bookmark(bookmark_id, current_user.id)

    if not success:
        raise HTTPException(status_code=404, detail="书签不存在或无权删除")

    return {"message": "书签已删除"}

@router.get("/search")
async def search_bookmarks(
    campaign_id: str,
    query: str,
    tags: Optional[List[str]] = Query(None),
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """搜索书签"""
    service = BookmarkService(db)
    bookmarks = service.search_bookmarks(campaign_id, current_user.id, query, tags)

    return [BookmarkResponse.from_orm(b) for b in bookmarks]

@router.get("/tags/summary")
async def get_tags_summary(
    campaign_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取标签统计"""
    service = BookmarkService(db)
    return service.get_tags_summary(campaign_id)
```

---

## 前端书签组件

```tsx
// frontend/src/components/game/Bookmark.tsx
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Bookmark as BookmarkIcon, Plus, Edit, Trash2, Search, Tag } from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'
import { useToast } from '@/hooks/use-toast'

interface Bookmark {
  id: string
  title: string
  description?: string
  color: string
  timestamp: string
  scene_id?: string
  tags?: string[]
  created_by: string
  is_public: boolean
}

interface BookmarkProps {
  roomId: string
  campaignId: string
  onJumpToBookmark?: (bookmark: Bookmark) => void
}

export function Bookmark({ roomId, campaignId, onJumpToBookmark }: BookmarkProps) {
  const [bookmarks, setBookmarks] = useState<Bookmark[]>([])
  const [searchQuery, setSearchQuery] = useState('')
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [newBookmark, setNewBookmark] = useState({
    title: '',
    description: '',
    color: '#3b82f6',
    tags: [] as string[],
    is_public: true,
  })

  const { user } = useAuth()
  const { toast } = useToast()

  useEffect(() => {
    loadBookmarks()
  }, [campaignId])

  const loadBookmarks = async () => {
    try {
      const response = await fetch(`/api/bookmarks?campaign_id=${campaignId}`)
      if (!response.ok) throw new Error('加载失败')

      const data = await response.json()
      setBookmarks(data)
    } catch (error) {
      console.error('Failed to load bookmarks:', error)
    }
  }

  const handleCreate = async () => {
    try {
      const response = await fetch('/api/bookmarks', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          room_id: roomId,
          campaign_id: campaignId,
          ...newBookmark,
        }),
      })

      if (!response.ok) throw new Error('创建失败')

      await loadBookmarks()
      setShowCreateDialog(false)
      setNewBookmark({
        title: '',
        description: '',
        color: '#3b82f6',
        tags: [],
        is_public: true,
      })

      toast({
        title: '书签已创建',
      })
    } catch (error) {
      toast({
        title: '创建失败',
        variant: 'destructive',
      })
    }
  }

  const handleDelete = async (bookmarkId: string) => {
    try {
      await fetch(`/api/bookmarks/${bookmarkId}`, {
        method: 'DELETE',
      })

      setBookmarks(bookmarks.filter(b => b.id !== bookmarkId))
    } catch (error) {
      console.error('Failed to delete bookmark:', error)
    }
  }

  const handleJump = (bookmark: Bookmark) => {
    onJumpToBookmark?.(bookmark)
  }

  const filteredBookmarks = bookmarks.filter(b =>
    b.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
    b.description?.toLowerCase().includes(searchQuery.toLowerCase())
  )

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base flex items-center justify-between">
          <span className="flex items-center">
            <BookmarkIcon className="h-4 w-4 mr-2" />
            书签 ({bookmarks.length})
          </span>
          <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
            <DialogTrigger asChild>
              <Button size="sm">
                <Plus className="h-4 w-4" />
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>创建书签</DialogTitle>
              </DialogHeader>
              <div className="space-y-4">
                <div>
                  <label className="text-sm font-medium">标题</label>
                  <Input
                    value={newBookmark.title}
                    onChange={(e) => setNewBookmark({ ...newBookmark, title: e.target.value })}
                    placeholder="书签标题"
                  />
                </div>
                <div>
                  <label className="text-sm font-medium">描述</label>
                  <Input
                    value={newBookmark.description}
                    onChange={(e) => setNewBookmark({ ...newBookmark, description: e.target.value })}
                    placeholder="可选描述"
                  />
                </div>
                <div>
                  <label className="text-sm font-medium">颜色</label>
                  <Input
                    type="color"
                    value={newBookmark.color}
                    onChange={(e) => setNewBookmark({ ...newBookmark, color: e.target.value })}
                  />
                </div>
                <div className="flex justify-end space-x-2">
                  <Button variant="outline" onClick={() => setShowCreateDialog(false)}>
                    取消
                  </Button>
                  <Button onClick={handleCreate}>
                    创建
                  </Button>
                </div>
              </div>
            </DialogContent>
          </Dialog>
        </CardTitle>
      </CardHeader>

      <CardContent>
        {/* 搜索 */}
        <div className="relative mb-3">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="搜索书签..."
            className="pl-9"
          />
        </div>

        {/* 书签列表 */}
        <div className="space-y-2 max-h-64 overflow-y-auto">
          {filteredBookmarks.map((bookmark) => (
            <div
              key={bookmark.id}
              className="p-2 border rounded hover:bg-muted cursor-pointer"
              onClick={() => handleJump(bookmark)}
            >
              <div className="flex items-start justify-between">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center space-x-2">
                    <div
                      className="w-3 h-3 rounded-full"
                      style={{ backgroundColor: bookmark.color }}
                    />
                    <span className="font-medium text-sm truncate">
                      {bookmark.title}
                    </span>
                    {!bookmark.is_public && (
                      <Badge variant="secondary" className="text-xs">私有</Badge>
                    )}
                  </div>
                  {bookmark.description && (
                    <p className="text-xs text-muted-foreground mt-1 line-clamp-1">
                      {bookmark.description}
                    </p>
                  )}
                  {bookmark.tags && bookmark.tags.length > 0 && (
                    <div className="flex flex-wrap gap-1 mt-1">
                      {bookmark.tags.map(tag => (
                        <Badge key={tag} variant="outline" className="text-xs">
                          <Tag className="h-2 w-2 mr-1" />
                          {tag}
                        </Badge>
                      ))}
                    </div>
                  )}
                </div>
                {bookmark.created_by === user?.id && (
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={(e) => {
                      e.stopPropagation()
                      handleDelete(bookmark.id)
                    }}
                  >
                    <Trash2 className="h-3 w-3" />
                  </Button>
                )}
              </div>
            </div>
          ))}

          {filteredBookmarks.length === 0 && (
            <div className="text-center text-sm text-muted-foreground py-4">
              {searchQuery ? '没有找到匹配的书签' : '还没有书签'}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/db/models/bookmark.py` | 创建 | 书签数据模型 |
| `app/services/bookmark.py` | 创建 | 书签服务 |
| `app/api/bookmarks.py` | 创建 | 书签 API |
| `app/schemas/bookmark.py` | 创建 | 书签 Schema |
| `frontend/src/components/game/Bookmark.tsx` | 创建 | 书签组件 |

---

## 验收标准

- [ ] 书签创建成功
- [ ] 书签列表显示正确
- [ ] 搜索功能有效
- [ ] 标签系统可用
- [ ] 权限控制正确
- [ ] 导航跳转准确

---

## 参考文档

- M3-001: AI 总结服务
- M3-004: 时间轴功能

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
