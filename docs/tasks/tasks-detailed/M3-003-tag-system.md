# M3-003: 实现标签系统

**任务ID**: M3-003
**标题**: 实现标签系统
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M1-080

---

## 任务描述

实现内容标签系统，支持对游戏事件、场景、NPC 等进行标签化管理。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M3-003-01 | 设计标签数据模型 | Tag Schema | 20min |
| M3-003-02 | 实现标签服务 | Tag Service | 30min |
| M3-003-03 | 实现标签关联 | Tag Relations | 25min |
| M3-003-04 | 实现标签搜索 | Tag Search | 25min |
| M3-003-05 | 实现标签推荐 | Tag Suggestion | 30min |
| M3-003-06 | 编写标签测试 | 测试覆盖 | 20min |

---

## 标签数据模型

```python
# app/db/models/tag.py
from sqlalchemy import Column, String, Integer, ForeignKey, Table
from sqlalchemy.orm import relationship
from sqlalchemy.sql import func
from app.db.database import Base

# 标签关联表 (多对多)
entity_tags = Table(
    'entity_tags',
    Base.metadata,
    Column('entity_id', String, ForeignKey('tags.id')),
    Column('tag_id', String, ForeignKey('tags.id')),
)

class Tag(Base):
    """标签"""
    __tablename__ = 'tags'

    id = Column(String, primary_key=True, index=True)
    name = Column(String, unique=True, nullable=False, index=True)
    category = Column(String, nullable=False, index=True)  # scene, npc, clue, event
    color = Column(String)  # 十六进制颜色

    # 统计
    usage_count = Column(Integer, default=0)
    created_at = Column(DateTime, default=func.now(), nullable=False)

    def __repr__(self):
        return f"<Tag {self.name} ({self.category})>"
```

---

## 标签服务

```python
# app/services/tag.py
from typing import List, Dict, Any, Optional
from sqlalchemy.orm import Session

from app.db.models.tag import Tag, entity_tags
from app.core.security import generate_id

class TagService:
    """标签服务"""

    def __init__(self, db: Session):
        self.db = db

    def create_tag(
        self,
        name: str,
        category: str,
        color: str = '#3b82f6',
    ) -> Tag:
        """创建标签"""
        tag = Tag(
            id=generate_id('tag'),
            name=name,
            category=category,
            color=color,
        )

        self.db.add(tag)
        self.db.commit()

        return tag

    def get_or_create(self, name: str, category: str) -> Tag:
        """获取或创建标签"""
        tag = self.db.query(Tag)\
            .filter(
                Tag.name == name,
                Tag.category == category
            )\
            .first()

        if not tag:
            tag = self.create_tag(name, category)

        return tag

    def tag_entity(
        self,
        entity_type: str,
        entity_id: str,
        tag_names: List[str],
        category: str,
    ) -> List[Tag]:
        """给实体打标签"""
        tags = []

        for tag_name in tag_names:
            tag = self.get_or_create(tag_name, category)
            tags.append(tag)

            # 增加使用计数
            tag.usage_count += 1

        self.db.commit()

        return tags

    def get_entity_tags(
        self,
        entity_type: str,
        entity_id: str,
    ) -> List[Tag]:
        """获取实体的标签"""
        tags = self.db.query(Tag)\
            .join(entity_tags, entity_tags.c.tag_id == Tag.id)\
            .filter(
                entity_tags.c.entity_id == entity_id,
                Tag.category == entity_type,
            )\
            .order_by(Tag.usage_count.desc())\
            .all()

        return tags

    def get_popular_tags(
        self,
        category: str,
        limit: int = 20,
    ) -> List[Tag]:
        """获取热门标签"""
        return self.db.query(Tag)\
            .filter(Tag.category == category)\
            .order_by(Tag.usage_count.desc())\
            .limit(limit)\
            .all()

    def search_tags(
        self,
        query: str,
        category: Optional[str] = None,
        limit: int = 20,
    ) -> List[Tag]:
        """搜索标签"""
        tags_query = self.db.query(Tag)\
            .filter(Tag.name.ilike(f'%{query}%'))

        if category:
            tags_query = tags_query.filter(Tag.category == category)

        return tags_query.order_by(Tag.usage_count.desc())\
            .limit(limit)\
            .all()

    def get_related_entities(
        self,
        tag_name: str,
        category: str,
        entity_type: str,
        limit: int = 20,
    ) -> List[Dict[str, Any]]:
        """获取带有指定标签的实体"""
        tag = self.db.query(Tag)\
            .filter(
                Tag.name == tag_name,
                Tag.category == category,
            )\
            .first()

        if not tag:
            return []

        # 根据实体类型获取关联的实体
        # 这里简化处理，实际应根据 entity_type 查询对应表
        return []

    def suggest_tags(
        self,
        entity_type: str,
        entity_id: str,
        limit: int = 5,
    ) -> List[Tag]:
        """推荐标签"""
        # 基于相似实体的标签推荐
        current_tags = self.get_entity_tags(entity_type, entity_id)

        # 获取使用这些标签的其他实体
        # 统计这些实体还用了什么标签
        # 返回热门标签

        return self.get_popular_tags(entity_type, limit)
```

---

## 标签 API

```python
# app/api/tag.py
from fastapi import APIRouter, Depends, Query
from sqlalchemy.orm import Session
from pydantic import BaseModel
from typing import List, Optional

from app.db.database import get_db
from app.services.tag import TagService
from app.api.deps.auth import get_current_user
from app.db.models.user import User

router = APIRouter(prefix="/tags", tags=["tags"])

class TagCreate(BaseModel):
    name: str
    category: str
    color: str = '#3b82f6'

class TagResponse(BaseModel):
    id: str
    name: str
    category: str
    color: str
    usage_count: int

@router.get("", response_model=List[TagResponse])
async def list_tags(
    category: Optional[str] = None,
    limit: int = Query(20, ge=1, le=100),
    search: Optional[str] = None,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """列出标签"""
    service = TagService(db)

    if search:
        tags = service.search_tags(search, category, limit)
    else:
        tags = service.get_popular_tags(category or 'event', limit)

    return [
        TagResponse(
            id=tag.id,
            name=tag.name,
            category=tag.category,
            color=tag.color,
            usage_count=tag.usage_count,
        )
        for tag in tags
    ]

@router.post("", response_model=TagResponse)
async def create_tag(
    tag_data: TagCreate,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """创建标签"""
    service = TagService(db)

    tag = service.create_tag(
        name=tag_data.name,
        category=tag_data.category,
        color=tag_data.color,
    )

    return TagResponse(
        id=tag.id,
        name=tag.name,
        category=tag.category,
        color=tag.color,
        usage_count=tag.usage_count,
    )

@router.get("/entity/{entity_type}/{entity_id}")
async def get_entity_tags(
    entity_type: str,
    entity_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取实体标签"""
    service = TagService(db)

    tags = service.get_entity_tags(entity_type, entity_id)

    return {
        'tags': [
            TagResponse(
                id=tag.id,
                name=tag.name,
                category=tag.category,
                color=tag.color,
                usage_count=tag.usage_count,
            )
            for tag in tags
        ]
    }

@router.post("/entity/{entity_type}/{entity_id}")
async def tag_entity(
    entity_type: str,
    entity_id: str,
    tag_names: List[str],
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """给实体打标签"""
    service = TagService(db)

    tags = service.tag_entity(entity_type, entity_id, tag_names, entity_type)

    return {
        'tags': [
            TagResponse(
                id=tag.id,
                name=tag.name,
                category=tag.category,
                color=tag.color,
                usage_count=tag.usage_count,
            )
            for tag in tags
        ]
    }
```

---

## 标签管理组件

```tsx
// frontend/src/components/game/TagManager.tsx
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { X, Plus } from 'lucide-react'
import { Tag } from '@/components/ui/tag'

interface Tag {
  id: string
  name: string
  category: string
  color: string
}

interface TagManagerProps {
  entityType: string
  entityId: string
}

export function TagManager({ entityType, entityId }: TagManagerProps) {
  const [tags, setTags] = useState<Tag[]>([])
  const [input, setInput] = useState('')

  useEffect(() => {
    loadTags()
  }, [entityType, entityId])

  const loadTags = async () => {
    try {
      const response = await fetch(`/api/tags/entity/${entityType}/${entityId}`)
      if (!response.ok) throw new Error('Failed to load tags')

      const data = await response.json()
      setTags(data.tags || [])
    } catch (error) {
      console.error('Failed to load tags:', error)
    }
  }

  const addTag = async (tagName: string) => {
    if (!tagName.trim()) return

    try {
      const response = await fetch(`/api/tags/entity/${entityType}/${entityId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tag_names: [tagName] }),
      })

      if (!response.ok) throw new Error('Failed to add tag')

      const data = await response.json()
      setTags(data.tags || [])
      setInput('')
    } catch (error) {
      console.error('Failed to add tag:', error)
    }
  }

  const removeTag = async (tagId: string) => () => {
    try {
      await fetch(`/api/tags/entity/${entityType}/${entityId}/${tagId}`, {
        method: 'DELETE',
      })

      setTags(tags.filter(t => t.id !== tagId))
    } catch (error) {
      console.error('Failed to remove tag:', error)
    }
  }

  return (
    <div className="space-y-4">
      {/* 已有标签 */}
      {tags.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {tags.map((tag) => (
            <Tag
              key={tag.id}
              style={{ backgroundColor: tag.color}}
              className="px-3 py-1"
            >
              {tag.name}
              <Button
                variant="ghost"
                size="sm"
                className="ml-1 h-4 w-4 p-0"
                onClick={removeTag(tag.id)}
              >
                <X className="h-3 w-3" />
              </Button>
            </Tag>
          ))}
        </div>
      )}

      {/* 添加标签 */}
      <div className="flex space-x-2">
        <Input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="添加标签..."
          className="flex-1"
        />
        <Button
          size="sm"
          onClick={() => addTag(input)}
          disabled={!input.trim()}
        >
          <Plus className="h-4 w-4 mr-1" />
          添加
        </Button>
      </div>
    </div>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/db/models/tag.py` | 创建 | 标签模型 |
| `app/services/tag.py` | 创建 | 标签服务 |
| `app/api/tag.py` | 创建 | 标签 API |
| `frontend/src/components/game/TagManager.tsx` | 创建 | 标签管理组件 |

---

## 验收标准

- [ ] 标签创建成功
- [ ] 标签关联正确
- [ ] 标签搜索有效
- [ ] 标签推荐准确
- [ ] 使用统计正确
- [ ] 组件交互流畅

---

## 参考文档

- M1-080: 事件日志系统

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
