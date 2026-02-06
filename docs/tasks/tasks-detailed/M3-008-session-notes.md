# M3-008: 实现会话笔记功能

**任务ID**: M3-008
**标题**: 实现会话笔记功能
**类型**: fullstack (全栈开发)
**预估工时**: 2h
**依赖**: M3-001

---

## 任务描述

实现会话笔记功能，让 KP 和玩家可以记录游戏过程中的重要信息、决策和剧情发展，支持富文本编辑和多人协作。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M3-008-01 | 设计笔记数据模型 | Note Model | 20min |
| M3-008-02 | 实现笔记服务 | Notes Service | 30min |
| M3-008-03 | 实现富文本编辑 | Rich Text Editor | 35min |
| M3-008-04 | 实现笔记共享 | Note Sharing | 25min |
| M3-008-05 | 实现笔记搜索 | Note Search | 20min |
| M3-008-06 | 实现笔记导出 | Note Export | 15min |
| M3-008-07 | 编写测试 | 测试覆盖 | 15min |

---

## 笔记数据模型

```python
# app/db/models/note.py
from sqlalchemy import Column, String, Text, ForeignKey, Boolean, JSON, DateTime
from sqlalchemy.orm import relationship
from app.db.database import Base
from datetime import datetime

class SessionNote(Base):
    """会话笔记"""
    __tablename__ = 'session_notes'

    id = Column(String, primary_key=True, index=True)
    room_id = Column(String, ForeignKey('rooms.id'), nullable=False, index=True)
    session_id = Column(String, ForeignKey('game_sessions.id'), index=True)  # 关联游戏会话

    # 基本信息
    title = Column(String, nullable=False)
    content = Column(Text)  # Markdown 或 HTML
    content_json = Column(JSON)  # 富文本编辑器的 JSON 格式

    # 元数据
    author_id = Column(String, ForeignKey('users.id'), nullable=False)
    author_name = Column(String)  # 冗余存储，方便展示

    # 分类
    category = Column(String)  # general, plot, character, location, clue, rules
    tags = Column(JSON, default=list)

    # 权限
    is_public = Column(Boolean, default=False)  # 是否对房间公开
    is_pinned = Column(Boolean, default=False)  # 是否置顶

    # 时间戳
    created_at = Column(DateTime, default=datetime.utcnow, index=True)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # 关联
    author = relationship("User", back_populates="session_notes")
    room = relationship("Room", back_populates="session_notes")
    session = relationship("GameSession", back_populates="notes")

    def __repr__(self):
        return f"<SessionNote {self.title}>"

class NoteComment(Base):
    """笔记评论"""
    __tablename__ = 'note_comments'

    id = Column(String, primary_key=True, index=True)
    note_id = Column(String, ForeignKey('session_notes.id'), nullable=False, index=True)

    # 评论内容
    content = Column(Text, nullable=False)

    # 作者
    author_id = Column(String, ForeignKey('users.id'), nullable=False)
    author_name = Column(String)

    # 时间戳
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # 关联
    note = relationship("SessionNote", back_populates="comments")
    author = relationship("User")

SessionNote.comments = relationship("NoteComment", cascade="all, delete-orphan")
```

---

## 笔记服务

```python
# app/services/notes.py
from typing import Dict, Any, List, Optional
from sqlalchemy.orm import Session
from sqlalchemy import and_, or_

from app.db.models.note import SessionNote, NoteComment
from app.core.security import generate_id

class NoteService:
    """笔记服务"""

    def __init__(self, db: Session):
        self.db = db

    def create_note(
        self,
        room_id: str,
        author_id: str,
        title: str,
        content: str = None,
        content_json: dict = None,
        category: str = 'general',
        tags: list = None,
        is_public: bool = False,
        session_id: str = None,
    ) -> SessionNote:
        """创建笔记"""
        note = SessionNote(
            id=generate_id('note'),
            room_id=room_id,
            session_id=session_id,
            title=title,
            content=content,
            content_json=content_json,
            author_id=author_id,
            category=category,
            tags=tags or [],
            is_public=is_public,
        )

        self.db.add(note)
        self.db.commit()
        self.db.refresh(note)

        return note

    def update_note(
        self,
        note_id: str,
        user_id: str,
        updates: Dict[str, Any],
    ) -> Optional[SessionNote]:
        """更新笔记"""
        note = self.db.query(SessionNote)\
            .filter(SessionNote.id == note_id)\
            .first()

        if not note:
            return None

        # 检查权限
        if note.author_id != user_id:
            return None

        for key, value in updates.items():
            if hasattr(note, key):
                setattr(note, key, value)

        self.db.commit()
        self.db.refresh(note)

        return note

    def delete_note(
        self,
        note_id: str,
        user_id: str,
    ) -> bool:
        """删除笔记"""
        note = self.db.query(SessionNote)\
            .filter(
                SessionNote.id == note_id,
                SessionNote.author_id == user_id,
            )\
            .first()

        if not note:
            return False

        self.db.delete(note)
        self.db.commit()

        return True

    def get_note(
        self,
        note_id: str,
        user_id: str,
    ) -> Optional[SessionNote]:
        """获取笔记详情"""
        note = self.db.query(SessionNote)\
            .filter(SessionNote.id == note_id)\
            .first()

        if not note:
            return None

        # 检查权限
        if note.author_id != user_id and not note.is_public:
            return None

        return note

    def get_room_notes(
        self,
        room_id: str,
        user_id: str,
        category: str = None,
        tags: list = None,
        session_id: str = None,
        limit: int = 100,
    ) -> List[SessionNote]:
        """获取房间笔记列表"""
        query = self.db.query(SessionNote)\
            .filter(SessionNote.room_id == room_id)

        # 只显示公开的或自己的笔记
        query = query.filter(
            or_(
                SessionNote.is_public == True,
                SessionNote.author_id == user_id,
            )
        )

        if category:
            query = query.filter(SessionNote.category == category)

        if session_id:
            query = query.filter(SessionNote.session_id == session_id)

        if tags:
            # 查找包含任一标签的笔记
            query = query.filter(
                SessionNote.tags.overlap(tags)
            )

        return query\
            .order_by(SessionNote.is_pinned.desc(), SessionNote.updated_at.desc())\
            .limit(limit)\
            .all()

    def search_notes(
        self,
        room_id: str,
        user_id: str,
        query: str,
    ) -> List[SessionNote]:
        """搜索笔记"""
        search_pattern = f"%{query}%"

        return self.db.query(SessionNote)\
            .filter(
                SessionNote.room_id == room_id,
                or_(
                    SessionNote.is_public == True,
                    SessionNote.author_id == user_id,
                ),
                or_(
                    SessionNote.title.ilike(search_pattern),
                    SessionNote.content.ilike(search_pattern),
                ),
            )\
            .order_by(SessionNote.updated_at.desc())\
            .all()

    def toggle_pin(
        self,
        note_id: str,
        user_id: str,
    ) -> Optional[SessionNote]:
        """切换置顶状态"""
        note = self.get_note(note_id, user_id)
        if not note:
            return None

        note.is_pinned = not note.is_pinned
        self.db.commit()
        self.db.refresh(note)

        return note

    def add_comment(
        self,
        note_id: str,
        author_id: str,
        author_name: str,
        content: str,
    ) -> NoteComment:
        """添加评论"""
        comment = NoteComment(
            id=generate_id('note_comment'),
            note_id=note_id,
            author_id=author_id,
            author_name=author_name,
            content=content,
        )

        self.db.add(comment)
        self.db.commit()
        self.db.refresh(comment)

        return comment

    def get_comments(
        self,
        note_id: str,
    ) -> List[NoteComment]:
        """获取笔记评论"""
        return self.db.query(NoteComment)\
            .filter(NoteComment.note_id == note_id)\
            .order_by(NoteComment.created_at.asc())\
            .all()

    def export_notes(
        self,
        room_id: str,
        user_id: str,
        format: str = 'markdown',
    ) -> str:
        """导出笔记"""
        notes = self.get_room_notes(room_id, user_id)

        if format == 'markdown':
            return self._export_markdown(notes)
        elif format == 'json':
            return self._export_json(notes)
        else:
            raise ValueError(f"不支持的导出格式: {format}")

    def _export_markdown(self, notes: List[SessionNote]) -> str:
        """导出为 Markdown"""
        lines = []
        lines.append(f"# 游戏笔记\n")
        lines.append(f"导出时间: {datetime.utcnow().strftime('%Y-%m-%d %H:%M:%S')}\n\n")
        lines.append("---\n\n")

        for note in notes:
            lines.append(f"## {note.title}\n\n")
            lines.append(f"**分类**: {note.category}\n")
            lines.append(f"**作者**: {note.author_name}\n")
            lines.append(f"**时间**: {note.created_at.strftime('%Y-%m-%d %H:%M')}\n")
            if note.tags:
                lines.append(f"**标签**: {', '.join(note.tags)}\n")
            lines.append("\n")
            lines.append(note.content or "")
            lines.append("\n\n---\n\n")

        return "".join(lines)

    def _export_json(self, notes: List[SessionNote]) -> str:
        """导出为 JSON"""
        import json

        data = [
            {
                "id": note.id,
                "title": note.title,
                "content": note.content,
                "category": note.category,
                "tags": note.tags,
                "author": note.author_name,
                "created_at": note.created_at.isoformat(),
                "updated_at": note.updated_at.isoformat(),
            }
            for note in notes
        ]

        return json.dumps(data, ensure_ascii=False, indent=2)
```

---

## 笔记 API

```python
# app/api/notes.py
from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy.orm import Session
from pydantic import BaseModel
from typing import List, Optional

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.notes import NoteService

router = APIRouter(prefix="/notes", tags=["notes"])

class CreateNoteRequest(BaseModel):
    room_id: str
    title: str
    content: str = None
    content_json: dict = None
    category: str = 'general'
    tags: list = None
    is_public: bool = False
    session_id: str = None

class UpdateNoteRequest(BaseModel):
    title: str = None
    content: str = None
    content_json: dict = None
    category: str = None
    tags: list = None
    is_public: bool = None

@router.post("")
async def create_note(
    request: CreateNoteRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """创建笔记"""
    service = NoteService(db)

    note = service.create_note(
        room_id=request.room_id,
        author_id=current_user.id,
        author_name=current_user.username,
        title=request.title,
        content=request.content,
        content_json=request.content_json,
        category=request.category,
        tags=request.tags,
        is_public=request.is_public,
        session_id=request.session_id,
    )

    return {
        "id": note.id,
        "title": note.title,
        "created_at": note.created_at.isoformat(),
    }

@router.put("/{note_id}")
async def update_note(
    note_id: str,
    request: UpdateNoteRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """更新笔记"""
    service = NoteService(db)

    updates = {k: v for k, v in request.dict().items() if v is not None}
    note = service.update_note(note_id, current_user.id, updates)

    if not note:
        raise HTTPException(status_code=404, detail="笔记不存在或无权修改")

    return {"message": "笔记已更新"}

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

@router.get("/{note_id}")
async def get_note(
    note_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取笔记详情"""
    service = NoteService(db)
    note = service.get_note(note_id, current_user.id)

    if not note:
        raise HTTPException(status_code=404, detail="笔记不存在或无权访问")

    return {
        "id": note.id,
        "title": note.title,
        "content": note.content,
        "content_json": note.content_json,
        "category": note.category,
        "tags": note.tags,
        "is_public": note.is_public,
        "is_pinned": note.is_pinned,
        "author_name": note.author_name,
        "created_at": note.created_at.isoformat(),
        "updated_at": note.updated_at.isoformat(),
    }

@router.get("/room/{room_id}")
async def get_room_notes(
    room_id: str,
    category: Optional[str] = None,
    tags: Optional[str] = None,
    session_id: Optional[str] = None,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取房间笔记列表"""
    service = NoteService(db)

    tag_list = tags.split(',') if tags else None
    notes = service.get_room_notes(
        room_id=room_id,
        user_id=current_user.id,
        category=category,
        tags=tag_list,
        session_id=session_id,
    )

    return {
        "notes": [
            {
                "id": note.id,
                "title": note.title,
                "category": note.category,
                "tags": note.tags,
                "is_public": note.is_public,
                "is_pinned": note.is_pinned,
                "author_name": note.author_name,
                "created_at": note.created_at.isoformat(),
                "updated_at": note.updated_at.isoformat(),
                "preview": (note.content or "")[:200],
            }
            for note in notes
        ]
    }

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
        raise HTTPException(status_code=404, detail="笔记不存在或无权操作")

    return {"is_pinned": note.is_pinned}

@router.post("/{note_id}/comments")
async def add_comment(
    note_id: str,
    content: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """添加评论"""
    service = NoteService(db)

    comment = service.add_comment(
        note_id=note_id,
        author_id=current_user.id,
        author_name=current_user.username,
        content=content,
    )

    return {
        "id": comment.id,
        "content": comment.content,
        "created_at": comment.created_at.isoformat(),
    }

@router.get("/{note_id}/comments")
async def get_comments(
    note_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取笔记评论"""
    service = NoteService(db)
    comments = service.get_comments(note_id)

    return {
        "comments": [
            {
                "id": c.id,
                "content": c.content,
                "author_name": c.author_name,
                "created_at": c.created_at.isoformat(),
            }
            for c in comments
        ]
    }

@router.get("/room/{room_id}/search")
async def search_notes(
    room_id: str,
    q: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """搜索笔记"""
    service = NoteService(db)
    notes = service.search_notes(room_id, current_user.id, q)

    return {
        "notes": [
            {
                "id": note.id,
                "title": note.title,
                "category": note.category,
                "preview": (note.content or "")[:200],
            }
            for note in notes
        ]
    }

@router.get("/room/{room_id}/export")
async def export_notes(
    room_id: str,
    format: str = 'markdown',
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """导出笔记"""
    service = NoteService(db)

    try:
        content = service.export_notes(room_id, current_user.id, format)
        filename = f"notes_{room_id}_{datetime.utcnow().strftime('%Y%m%d_%H%M%S')}"
        media_type = "text/markdown" if format == "markdown" else "application/json"

        from fastapi.responses import Response
        return Response(
            content=content,
            media_type=media_type,
            headers={
                "Content-Disposition": f'attachment; filename="{filename}.{format}"'
            }
        )
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))
```

---

## 前端笔记组件

```tsx
// frontend/src/components/notes/NoteEditor.tsx
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { Save, Trash2, Pin, PinOff, Search, Download } from 'lucide-react'
import { TiptapEditor } from '@/components/editor/TiptapEditor'

interface Note {
  id: string
  title: string
  content: string
  content_json: any
  category: string
  tags: string[]
  is_public: boolean
  is_pinned: boolean
  author_name: string
  created_at: string
  updated_at: string
}

interface NoteEditorProps {
  roomId: string
  noteId?: string
  onSave?: () => void
}

export function NoteEditor({ roomId, noteId, onSave }: NoteEditorProps) {
  const [note, setNote] = useState<Note | null>(null)
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [category, setCategory] = useState('general')
  const [tags, setTags] = useState<string[]>([])
  const [tagInput, setTagInput] = useState('')
  const [isPublic, setIsPublic] = useState(false)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (noteId) {
      fetchNote()
    }
  }, [noteId])

  const fetchNote = async () => {
    try {
      const response = await fetch(`/api/notes/${noteId}`)
      if (response.ok) {
        const data = await response.json()
        setNote(data)
        setTitle(data.title)
        setContent(data.content || '')
        setCategory(data.category)
        setTags(data.tags || [])
        setIsPublic(data.is_public)
      }
    } catch (error) {
      console.error('Failed to fetch note:', error)
    }
  }

  const handleSave = async () => {
    setSaving(true)
    try {
      const method = noteId ? 'PUT' : 'POST'
      const url = noteId ? `/api/notes/${noteId}` : '/api/notes'

      const response = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          room_id: roomId,
          title,
          content,
          category,
          tags,
          is_public,
        }),
      })

      if (response.ok) {
        onSave?.()
      }
    } catch (error) {
      console.error('Failed to save note:', error)
    } finally {
      setSaving(false)
    }
  }

  const handleAddTag = () => {
    if (tagInput && !tags.includes(tagInput)) {
      setTags([...tags, tagInput])
      setTagInput('')
    }
  }

  const handleRemoveTag = (tag: string) => {
    setTags(tags.filter((t) => t !== tag))
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Input
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="笔记标题"
          className="flex-1"
        />
        <Select value={category} onValueChange={setCategory}>
          <SelectTrigger className="w-[150px]">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="general">通用</SelectItem>
            <SelectItem value="plot">剧情</SelectItem>
            <SelectItem value="character">角色</SelectItem>
            <SelectItem value="location">地点</SelectItem>
            <SelectItem value="clue">线索</SelectItem>
            <SelectItem value="rules">规则</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="flex items-center gap-2 flex-wrap">
        {tags.map((tag) => (
          <Badge key={tag} variant="secondary">
            {tag}
            <button
              onClick={() => handleRemoveTag(tag)}
              className="ml-1 hover:text-destructive"
            >
              ×
            </button>
          </Badge>
        ))}
        <Input
          value={tagInput}
          onChange={(e) => setTagInput(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && handleAddTag()}
          placeholder="添加标签..."
          className="w-32"
        />
      </div>

      <TiptapEditor
        content={content}
        onChange={setContent}
        placeholder="开始记录你的游戏笔记..."
      />

      <div className="flex justify-between">
        <div className="flex gap-2">
          <Button onClick={handleSave} disabled={saving}>
            <Save className="h-4 w-4 mr-2" />
            {saving ? '保存中...' : '保存'}
          </Button>
          <Button variant="outline" size="sm">
            <Download className="h-4 w-4 mr-2" />
            导出
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
| `app/services/notes.py` | 创建 | 笔记服务 |
| `app/api/notes.py` | 创建 | 笔记 API |
| `frontend/src/components/notes/NoteEditor.tsx` | 创建 | 笔记编辑器 |
| `frontend/src/components/notes/NoteList.tsx` | 创建 | 笔记列表 |
| `frontend/src/components/notes/NoteViewer.tsx` | 创建 | 笔记查看器 |

---

## 验收标准

- [ ] 笔记创建/编辑正常
- [ ] 富文本编辑功能完整
- [ ] 分类和标签有效
- [ ] 权限控制正确
- [ ] 搜索功能准确
- [ ] 导出格式正确

---

## 参考文档

- M3-001: 会话管理系统
- M5-006: 富文本编辑器
- Tiptap 文档

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
