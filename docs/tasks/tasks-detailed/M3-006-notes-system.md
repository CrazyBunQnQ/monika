# M3-006: 实现笔记系统

**任务ID**: M3-006
**标题**: 实现笔记系统
**类型**: fullstack (全栈开发)
**预估工时**: 2h
**依赖**: M3-001

---

## 任务描述

实现玩家笔记系统，允许玩家记录游戏中的重要信息、线索、任务等，支持富文本编辑和标签分类。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M3-006-01 | 设计笔记数据结构 | Data Model | 15min |
| M3-006-02 | 实现笔记服务 | Note Service | 25min |
| M3-006-03 | 实现笔记 API | API | 20min |
| M3-006-04 | 实现笔记编辑器 | Note Editor | 30min |
| M3-006-05 | 实现笔记列表 | Note List | 20min |
| M3-006-06 | 实现搜索功能 | Search | 20min |
| M3-006-07 | 编写笔记测试 | 测试覆盖 | 15min |

---

## 笔记数据模型

```python
# app/db/models/note.py
from sqlalchemy import Column, String, Text, ForeignKey, DateTime, Boolean
from sqlalchemy.orm import relationship
from sqlalchemy.sql import func
from app.db.database import Base

class Note(Base):
    """笔记"""
    __tablename__ = 'notes'

    id = Column(String, primary_key=True, index=True)
    room_id = Column(String, ForeignKey('rooms.id'), nullable=False, index=True)
    campaign_id = Column(String, ForeignKey('campaigns.id'), nullable=False, index=True)

    # 基本信息
    title = Column(String, nullable=False)
    content = Column(Text, nullable=False)
    color = Column(String, default='#ffffff')  # 背景颜色

    # 分类
    category = Column(String, default='general')  # general, clue, task, npc, location
    tags = Column(JSON)  # 标签列表

    # 关联
    character_id = Column(String, ForeignKey('characters.id'))
    scene_id = Column(String, ForeignKey('scenes.id'))
    clue_id = Column(String, ForeignKey('clues.id'))

    # 创建者
    created_by = Column(String, ForeignKey('users.id'), nullable=False)

    # 状态
    is_pinned = Column(Boolean, default=False, nullable=False)
    is_archived = Column(Boolean, default=False, nullable=False)

    # 时间
    created_at = Column(DateTime, default=func.now(), nullable=False)
    updated_at = Column(DateTime, default=func.now(), onupdate=func.now(), nullable=False)

    # 关系
    room = relationship("Room", back_populates="notes")
    campaign = relationship("Campaign", back_populates="notes")
    creator = relationship("User", back_populates="notes")
    character = relationship("Character", back_populates="notes")
    scene = relationship("Scene", back_populates="notes")
    clue = relationship("Clue", back_populates="notes")

    def __repr__(self):
        return f"<Note {self.title}>"
```

---

## 笔记服务

```python
# app/services/note.py
from typing import List, Dict, Any, Optional
from sqlalchemy.orm import Session
from datetime import datetime

from app.db.models.note import Note
from app.core.security import generate_id

class NoteService:
    """笔记服务"""

    def __init__(self, db: Session):
        self.db = db

    def create_note(
        self,
        room_id: str,
        campaign_id: str,
        title: str,
        content: str,
        created_by: str,
        category: str = 'general',
        color: str = '#ffffff',
        tags: List[str] = None,
        character_id: str = None,
        scene_id: str = None,
        clue_id: str = None,
    ) -> Note:
        """创建笔记"""
        note = Note(
            id=generate_id('note'),
            room_id=room_id,
            campaign_id=campaign_id,
            title=title,
            content=content,
            category=category,
            color=color,
            tags=tags or [],
            character_id=character_id,
            scene_id=scene_id,
            clue_id=clue_id,
            created_by=created_by,
        )

        self.db.add(note)
        self.db.commit()
        self.db.refresh(note)

        return note

    def get_notes(
        self,
        campaign_id: str,
        user_id: str,
        category: str = None,
        is_archived: bool = False,
    ) -> List[Note]:
        """获取笔记列表"""
        query = self.db.query(Note)\
            .filter(
                Note.campaign_id == campaign_id,
                Note.created_by == user_id,
                Note.is_archived == is_archived,
            )

        if category:
            query = query.filter(Note.category == category)

        return query\
            .order_by(Note.is_pinned.desc(), Note.updated_at.desc())\
            .all()

    def get_note(self, note_id: str) -> Optional[Note]:
        """获取单个笔记"""
        return self.db.query(Note)\
            .filter(Note.id == note_id)\
            .first()

    def update_note(
        self,
        note_id: str,
        user_id: str,
        **updates,
    ) -> Optional[Note]:
        """更新笔记"""
        note = self.get_note(note_id)

        if not note or note.created_by != user_id:
            return None

        for key, value in updates.items():
            if hasattr(note, key):
                setattr(note, key, value)

        note.updated_at = datetime.now()

        self.db.commit()
        self.db.refresh(note)

        return note

    def delete_note(self, note_id: str, user_id: str) -> bool:
        """删除笔记"""
        note = self.get_note(note_id)

        if not note or note.created_by != user_id:
            return False

        self.db.delete(note)
        self.db.commit()

        return True

    def toggle_pin(self, note_id: str, user_id: str) -> Optional[Note]:
        """切换置顶状态"""
        note = self.get_note(note_id)

        if not note or note.created_by != user_id:
            return None

        note.is_pinned = not note.is_pinned
        self.db.commit()
        self.db.refresh(note)

        return note

    def archive_note(self, note_id: str, user_id: str) -> Optional[Note]:
        """归档笔记"""
        note = self.get_note(note_id)

        if not note or note.created_by != user_id:
            return None

        note.is_archived = True
        self.db.commit()
        self.db.refresh(note)

        return note

    def search_notes(
        self,
        campaign_id: str,
        user_id: str,
        query: str,
        tags: List[str] = None,
    ) -> List[Note]:
        """搜索笔记"""
        notes = self.get_notes(campaign_id, user_id)

        results = []
        for note in notes:
            # 文本搜索
            text_match = (
                query.lower() in note.title.lower() or
                query.lower() in note.content.lower()
            )

            # 标签过滤
            tag_match = not tags or any(tag in (note.tags or []) for tag in tags)

            if text_match and tag_match:
                results.append(note)

        return results

    def get_categories_summary(self, campaign_id: str, user_id: str) -> Dict[str, int]:
        """获取分类统计"""
        notes = self.get_notes(campaign_id, user_id)

        categories = {}
        for note in notes:
            cat = note.category or 'general'
            categories[cat] = categories.get(cat, 0) + 1

        return categories
```

---

## 笔记 API

```python
# app/api/notes.py
from fastapi import APIRouter, Depends, Query
from sqlalchemy.orm import Session
from typing import List, Optional
from pydantic import BaseModel

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.note import NoteService
from app.schemas.note import NoteCreate, NoteResponse, NoteUpdate

router = APIRouter(prefix="/notes", tags=["notes"])

@router.post("")
async def create_note(
    note_data: NoteCreate,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """创建笔记"""
    service = NoteService(db)

    note = service.create_note(
        room_id=note_data.room_id,
        campaign_id=note_data.campaign_id,
        title=note_data.title,
        content=note_data.content,
        created_by=current_user.id,
        category=note_data.category,
        color=note_data.color,
        tags=note_data.tags,
    )

    return NoteResponse.from_orm(note)

@router.get("")
async def list_notes(
    campaign_id: str,
    category: Optional[str] = None,
    is_archived: bool = False,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取笔记列表"""
    service = NoteService(db)
    notes = service.get_notes(campaign_id, current_user.id, category, is_archived)

    return [NoteResponse.from_orm(n) for n in notes]

@router.get("/{note_id}")
async def get_note(
    note_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取单个笔记"""
    service = NoteService(db)
    note = service.get_note(note_id)

    if not note:
        raise HTTPException(status_code=404, detail="笔记不存在")

    # 权限检查
    if note.created_by != current_user.id:
        raise HTTPException(status_code=403, detail="无权访问此笔记")

    return NoteResponse.from_orm(note)

@router.put("/{note_id}")
async def update_note(
    note_id: str,
    note_data: NoteUpdate,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """更新笔记"""
    service = NoteService(db)
    note = service.update_note(
        note_id,
        current_user.id,
        **note_data.dict(exclude_unset=True),
    )

    if not note:
        raise HTTPException(status_code=404, detail="笔记不存在或无权修改")

    return NoteResponse.from_orm(note)

@router.delete("/{note_id}")
async def delete_note(
    note_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """删除笔记"""
    service = NoteService(db)
    success = service.delete_note(note_id, current_user.id)

    if not success:
        raise HTTPException(status_code=404, detail="笔记不存在或无权删除")

    return {"message": "笔记已删除"}

@router.post("/{note_id}/pin")
async def toggle_pin_note(
    note_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """切换置顶状态"""
    service = NoteService(db)
    note = service.toggle_pin(note_id, current_user.id)

    if not note:
        raise HTTPException(status_code=404, detail="笔记不存在")

    return NoteResponse.from_orm(note)

@router.post("/{note_id}/archive")
async def archive_note(
    note_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """归档笔记"""
    service = NoteService(db)
    note = service.archive_note(note_id, current_user.id)

    if not note:
        raise HTTPException(status_code=404, detail="笔记不存在")

    return NoteResponse.from_orm(note)
```

---

## 前端笔记组件

```tsx
// frontend/src/components/game/NotesPanel.tsx
import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Plus, Search, Pin, Archive, FileText, Edit, Trash2 } from 'lucide-react'
import { RichTextEditor } from '@/components/ui/rich-text-editor'
import { useToast } from '@/hooks/use-toast'

interface Note {
  id: string
  title: string
  content: string
  category: string
  color: string
  tags?: string[]
  is_pinned: boolean
  updated_at: string
}

interface NotesPanelProps {
  campaignId: string
}

export function NotesPanel({ campaignId }: NotesPanelProps) {
  const [notes, setNotes] = useState<Note[]>([])
  const [activeNote, setActiveNote] = useState<Note | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [newNote, setNewNote] = useState({
    title: '',
    content: '',
    category: 'general',
    color: '#ffffff',
    tags: [] as string[],
  })

  const { toast } = useToast()

  useEffect(() => {
    loadNotes()
  }, [campaignId])

  const loadNotes = async () => {
    try {
      const response = await fetch(`/api/notes?campaign_id=${campaignId}`)
      if (!response.ok) throw new Error('加载失败')

      const data = await response.json()
      setNotes(data)
    } catch (error) {
      console.error('Failed to load notes:', error)
    }
  }

  const handleCreate = async () => {
    try {
      const response = await fetch('/api/notes', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          campaign_id: campaignId,
          room_id: '',
          ...newNote,
        }),
      })

      if (!response.ok) throw new Error('创建失败')

      await loadNotes()
      setShowCreateDialog(false)
      setNewNote({
        title: '',
        content: '',
        category: 'general',
        color: '#ffffff',
        tags: [],
      })

      toast({
        title: '笔记已创建',
      })
    } catch (error) {
      toast({
        title: '创建失败',
        variant: 'destructive',
      })
    }
  }

  const handleDelete = async (noteId: string) => {
    try {
      await fetch(`/api/notes/${noteId}`, {
        method: 'DELETE',
      })

      setNotes(notes.filter(n => n.id !== noteId))
      if (activeNote?.id === noteId) {
        setActiveNote(null)
      }
    } catch (error) {
      toast({
        title: '删除失败',
        variant: 'destructive',
      })
    }
  }

  const togglePin = async (noteId: string) => {
    try {
      const response = await fetch(`/api/notes/${noteId}/pin`, {
        method: 'POST',
      })

      if (!response.ok) throw new Error('操作失败')

      const data = await response.json()
      setNotes(notes.map(n => n.id === noteId ? data : n))
    } catch (error) {
      console.error('Failed to toggle pin:', error)
    }
  }

  const filteredNotes = notes.filter(n =>
    n.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
    n.content.toLowerCase().includes(searchQuery.toLowerCase())
  )

  const pinnedNotes = filteredNotes.filter(n => n.is_pinned)
  const otherNotes = filteredNotes.filter(n => !n.is_pinned)

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base flex items-center justify-between">
          <span className="flex items-center">
            <FileText className="h-4 w-4 mr-2" />
            笔记
          </span>
          <Button size="sm" onClick={() => setShowCreateDialog(true)}>
            <Plus className="h-4 w-4" />
          </Button>
        </CardTitle>
      </CardHeader>

      <CardContent>
        {/* 搜索 */}
        <div className="relative mb-4">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="搜索笔记..."
            className="pl-9"
          />
        </div>

        {/* 笔记列表 */}
        <div className="space-y-2 max-h-96 overflow-y-auto">
          {pinnedNotes.length > 0 && (
            <>
              <div className="text-xs text-muted-foreground font-medium">置顶</div>
              {pinnedNotes.map(note => (
                <NoteItem
                  key={note.id}
                  note={note}
                  isActive={activeNote?.id === note.id}
                  onClick={() => setActiveNote(note)}
                  onPin={() => togglePin(note.id)}
                  onDelete={() => handleDelete(note.id)}
                />
              ))}
            </>
          )}

          {otherNotes.length > 0 && (
            <>
              {pinnedNotes.length > 0 && (
                <div className="text-xs text-muted-foreground font-medium mt-4">其他</div>
              )}
              {otherNotes.map(note => (
                <NoteItem
                  key={note.id}
                  note={note}
                  isActive={activeNote?.id === note.id}
                  onClick={() => setActiveNote(note)}
                  onPin={() => togglePin(note.id)}
                  onDelete={() => handleDelete(note.id)}
                />
              ))}
            </>
          )}

          {filteredNotes.length === 0 && (
            <div className="text-center text-sm text-muted-foreground py-8">
              {searchQuery ? '没有找到匹配的笔记' : '还没有笔记'}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

function NoteItem({
  note,
  isActive,
  onClick,
  onPin,
  onDelete,
}: {
  note: Note
  isActive: boolean
  onClick: () => void
  onPin: () => void
  onDelete: () => void
}) {
  return (
    <div
      className={`p-3 border rounded cursor-pointer transition-colors ${
        isActive ? 'bg-primary/10 border-primary' : 'hover:bg-muted'
      }`}
      onClick={onClick}
    >
      <div className="flex items-start justify-between">
        <div className="flex-1 min-w-0">
          <div className="flex items-center space-x-2">
            {note.is_pinned && <Pin className="h-3 w-3 text-primary" />}
            <span className="font-medium text-sm truncate">{note.title}</span>
          </div>
          <p className="text-xs text-muted-foreground mt-1 line-clamp-2">
            {note.content.substring(0, 100)}
          </p>
          <div className="flex items-center space-x-2 mt-2">
            <Badge variant="outline" className="text-xs">
              {note.category}
            </Badge>
            {note.tags?.slice(0, 2).map(tag => (
              <Badge key={tag} variant="secondary" className="text-xs">
                {tag}
              </Badge>
            ))}
          </div>
        </div>
        <div className="flex space-x-1 ml-2">
          <Button
            size="sm"
            variant="ghost"
            onClick={(e) => {
              e.stopPropagation()
              onPin()
            }}
          >
            <Pin className="h-3 w-3" />
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onClick={(e) => {
              e.stopPropagation()
              onDelete()
            }}
          >
            <Trash2 className="h-3 w-3" />
          </Button>
        </div>
      </div>
    </div>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/db/models/note.py` | 创建 | 笔记数据模型 |
| `app/services/note.py` | 创建 | 笔记服务 |
| `app/api/notes.py` | 创建 | 笔记 API |
| `app/schemas/note.py` | 创建 | 笔记 Schema |
| `frontend/src/components/game/NotesPanel.tsx` | 创建 | 笔记面板组件 |

---

## 验收标准

- [ ] 笔记创建成功
- [ ] 富文本编辑有效
- [ ] 标签分类正常
- [ ] 搜索功能准确
- [ ] 置顶功能可用
- [ ] 归档功能正常

---

## 参考文档

- M3-001: AI 总结服务
- M5-006: 富文本编辑器

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
