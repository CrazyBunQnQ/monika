# M3-004: 实现时间轴功能

**任务ID**: M3-004
**标题**: 实现时间轴功能
**类型**: fullstack (全栈开发)
**预估工时**: 2.5h
**依赖**: M1-080, M3-001

---

## 任务描述

实现游戏事件时间轴功能，按时间顺序展示游戏中的重要事件、检定结果、角色状态变化等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M3-004-01 | 设计时间轴数据结构 | Data Structure | 20min |
| M3-004-02 | 实现事件聚合服务 | Event Aggregation | 30min |
| M3-004-03 | 实现时间轴 API | Timeline API | 25min |
| M3-004-04 | 实现时间轴组件 | Timeline Component | 35min |
| M3-004-05 | 实现过滤功能 | Filter | 20min |
| M3-004-06 | 实现导出功能 | Export | 20min |
| M3-004-07 | 编写时间轴测试 | 测试覆盖 | 15min |

---

## 时间轴数据结构

```python
# app/db/models/timeline.py
from sqlalchemy import Column, String, DateTime, ForeignKey, JSON
from sqlalchemy.orm import relationship
from sqlalchemy.sql import func
from app.db.database import Base

class TimelineEntry(Base):
    """时间轴条目"""
    __tablename__ = 'timeline_entries'

    id = Column(String, primary_key=True, index=True)
    room_id = Column(String, ForeignKey('rooms.id'), nullable=False, index=True)
    campaign_id = Column(String, ForeignKey('campaigns.id'), nullable=False, index=True)

    # 事件类型
    event_type = Column(String, nullable=False, index=True)  # roll, check, combat, san, dialogue, scene_change, etc.
    category = Column(String, nullable=False, index=True)  # dice, roleplay, combat, story

    # 时间
    timestamp = Column(DateTime, default=func.now(), nullable=False, index=True)
    game_time = Column(DateTime)  # 游戏内时间（如 1920-01-15 14:30）

    # 内容
    title = Column(String, nullable=False)
    description = Column(String)
    data = Column(JSON)  # 额外数据，如检定结果、骰子点数等

    # 关联
    character_id = Column(String, ForeignKey('characters.id'))
    scene_id = Column(String, ForeignKey('scenes.id'))

    # 重要性
    importance = Column(String, default='normal')  # low, normal, high, critical
    is_highlighted = Column(String, default=False)  # 是否被 KP 标记为重点

    created_at = Column(DateTime, default=func.now(), nullable=False)

    # 关系
    room = relationship("Room", back_populates="timeline_entries")
    campaign = relationship("Campaign", back_populates="timeline_entries")
    character = relationship("Character", back_populates="timeline_entries")
    scene = relationship("Scene", back_populates="timeline_entries")

    def __repr__(self):
        return f"<TimelineEntry {self.title} at {self.timestamp}>"
```

---

## 时间轴服务

```python
# app/services/timeline.py
from typing import List, Dict, Any, Optional
from sqlalchemy.orm import Session
from datetime import datetime, timedelta

from app.db.models.timeline import TimelineEntry
from app.db.models.event import EventLog

class TimelineService:
    """时间轴服务"""

    def __init__(self, db: Session):
        self.db = db

    def add_entry(
        self,
        room_id: str,
        campaign_id: str,
        event_type: str,
        title: str,
        description: str = None,
        data: Dict = None,
        character_id: str = None,
        scene_id: str = None,
        importance: str = 'normal',
        game_time: datetime = None,
    ) -> TimelineEntry:
        """添加时间轴条目"""
        category = self._get_category(event_type)

        entry = TimelineEntry(
            id=generate_id('timeline'),
            room_id=room_id,
            campaign_id=campaign_id,
            event_type=event_type,
            category=category,
            title=title,
            description=description,
            data=data or {},
            character_id=character_id,
            scene_id=scene_id,
            importance=importance,
            game_time=game_time,
        )

        self.db.add(entry)
        self.db.commit()

        return entry

    def get_timeline(
        self,
        campaign_id: str,
        start_time: datetime = None,
        end_time: datetime = None,
        event_types: List[str] = None,
        categories: List[str] = None,
        character_ids: List[str] = None,
        importance: List[str] = None,
        limit: int = 100,
        offset: int = 0,
    ) -> List[TimelineEntry]:
        """获取时间轴"""
        query = self.db.query(TimelineEntry)\
            .filter(TimelineEntry.campaign_id == campaign_id)

        # 时间范围
        if start_time:
            query = query.filter(TimelineEntry.timestamp >= start_time)
        if end_time:
            query = query.filter(TimelineEntry.timestamp <= end_time)

        # 事件类型
        if event_types:
            query = query.filter(TimelineEntry.event_type.in_(event_types))

        # 分类
        if categories:
            query = query.filter(TimelineEntry.category.in_(categories))

        # 角色
        if character_ids:
            query = query.filter(TimelineEntry.character_id.in_(character_ids))

        # 重要性
        if importance:
            query = query.filter(TimelineEntry.importance.in_(importance))

        # 排序和分页
        return query\
            .order_by(TimelineEntry.timestamp.desc())\
            .limit(limit)\
            .offset(offset)\
            .all()

    def get_summary(
        self,
        campaign_id: str,
        start_time: datetime = None,
        end_time: datetime = None,
    ) -> Dict[str, Any]:
        """获取时间轴摘要"""
        query = self.db.query(TimelineEntry)\
            .filter(TimelineEntry.campaign_id == campaign_id)

        if start_time:
            query = query.filter(TimelineEntry.timestamp >= start_time)
        if end_time:
            query = query.filter(TimelineEntry.timestamp <= end_time)

        entries = query.all()

        # 统计
        return {
            'total_events': len(entries),
            'by_category': self._count_by_field(entries, 'category'),
            'by_type': self._count_by_field(entries, 'event_type'),
            'by_character': self._count_by_character(entries),
            'highlighted': len([e for e in entries if e.is_highlighted]),
            'date_range': self._get_date_range(entries),
        }

    def highlight_entry(self, entry_id: str) -> TimelineEntry:
        """标记为重点"""
        entry = self.db.query(TimelineEntry)\
            .filter(TimelineEntry.id == entry_id)\
            .first()

        if entry:
            entry.is_highlighted = not entry.is_highlighted
            self.db.commit()

        return entry

    def _get_category(self, event_type: str) -> str:
        """获取事件分类"""
        category_map = {
            'roll': 'dice',
            'check': 'dice',
            'damage': 'combat',
            'attack': 'combat',
            'heal': 'combat',
            'san': 'dice',
            'dialogue': 'roleplay',
            'scene_change': 'story',
            'clue_found': 'story',
            'handout': 'story',
        }
        return category_map.get(event_type, 'other')

    def _count_by_field(self, entries: List, field: str) -> Dict:
        """按字段统计"""
        counts = {}
        for entry in entries:
            value = getattr(entry, field, 'unknown')
            counts[value] = counts.get(value, 0) + 1
        return counts

    def _count_by_character(self, entries: List) -> Dict:
        """按角色统计"""
        counts = {}
        for entry in entries:
            if entry.character:
                name = entry.character.name
                counts[name] = counts.get(name, 0) + 1
        return counts

    def _get_date_range(self, entries: List) -> Dict:
        """获取日期范围"""
        if not entries:
            return {}

        timestamps = [e.timestamp for e in entries]
        return {
            'start': min(timestamps),
            'end': max(timestamps),
        }
```

---

## 时间轴 API

```python
# app/api/timeline.py
from fastapi import APIRouter, Depends, Query
from sqlalchemy.orm import Session
from typing import List, Optional
from datetime import datetime

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.timeline import TimelineService
from app.schemas.timeline import TimelineEntryCreate, TimelineEntryResponse

router = APIRouter(prefix="/timeline", tags=["timeline"])

@router.get("/{campaign_id}", response_model=List[TimelineEntryResponse])
async def get_timeline(
    campaign_id: str,
    start: Optional[datetime] = None,
    end: Optional[datetime] = None,
    event_types: Optional[List[str]] = Query(None),
    categories: Optional[List[str]] = Query(None),
    character_ids: Optional[List[str]] = Query(None),
    importance: Optional[List[str]] = Query(None),
    limit: int = Query(100, ge=1, le=500),
    offset: int = Query(0, ge=0),
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取时间轴"""
    service = TimelineService(db)

    entries = service.get_timeline(
        campaign_id=campaign_id,
        start_time=start,
        end_time=end,
        event_types=event_types,
        categories=categories,
        character_ids=character_ids,
        importance=importance,
        limit=limit,
        offset=offset,
    )

    return [TimelineEntryResponse.from_orm(e) for e in entries]

@router.get("/{campaign_id}/summary")
async def get_timeline_summary(
    campaign_id: str,
    start: Optional[datetime] = None,
    end: Optional[datetime] = None,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取时间轴摘要"""
    service = TimelineService(db)
    return service.get_summary(campaign_id, start, end)

@router.post("/{campaign_id}/entries")
async def add_timeline_entry(
    campaign_id: str,
    entry_data: TimelineEntryCreate,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """添加时间轴条目"""
    service = TimelineService(db)

    entry = service.add_entry(
        room_id=entry_data.room_id,
        campaign_id=campaign_id,
        event_type=entry_data.event_type,
        title=entry_data.title,
        description=entry_data.description,
        data=entry_data.data,
        character_id=entry_data.character_id,
        scene_id=entry_data.scene_id,
        importance=entry_data.importance,
        game_time=entry_data.game_time,
    )

    return TimelineEntryResponse.from_orm(entry)

@router.put("/{campaign_id}/entries/{entry_id}/highlight")
async def highlight_entry(
    campaign_id: str,
    entry_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """标记/取消标记重点"""
    service = TimelineService(db)
    entry = service.highlight_entry(entry_id)
    return TimelineEntryResponse.from_orm(entry)
```

---

## 前端时间轴组件

```tsx
// frontend/src/components/game/Timeline.tsx
import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Filter, Download, Star, Clock } from 'lucide-react'

interface TimelineEntry {
  id: string
  event_type: string
  category: string
  title: string
  description?: string
  timestamp: string
  game_time?: string
  data: any
  character_name?: string
  is_highlighted: boolean
  importance: string
}

interface TimelineProps {
  campaignId: string
}

export function Timeline({ campaignId }: TimelineProps) {
  const [entries, setEntries] = useState<TimelineEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [filters, setFilters] = useState({
    categories: [] as string[],
    event_types: [] as string[],
    importance: [] as string[],
  })

  useEffect(() => {
    loadTimeline()
  }, [campaignId, filters])

  const loadTimeline = async () => {
    setLoading(true)

    try {
      const params = new URLSearchParams()
      filters.categories.forEach(c => params.append('categories', c))
      filters.event_types.forEach(t => params.append('event_types', t))
      filters.importance.forEach(i => params.append('importance', i))

      const response = await fetch(`/api/timeline/${campaignId}?${params}`)
      if (!response.ok) throw new Error('加载失败')

      const data = await response.json()
      setEntries(data)
    } catch (error) {
      console.error('Failed to load timeline:', error)
    } finally {
      setLoading(false)
    }
  }

  const toggleHighlight = async (entryId: string) => {
    try {
      const response = await fetch(
        `/api/timeline/${campaignId}/entries/${entryId}/highlight`,
        { method: 'PUT' }
      )

      if (!response.ok) throw new Error('操作失败')

      setEntries(entries.map(e =>
        e.id === entryId ? { ...e, is_highlighted: !e.is_highlighted } : e
      ))
    } catch (error) {
      console.error('Failed to toggle highlight:', error)
    }
  }

  const getCategoryColor = (category: string) => {
    const colors = {
      dice: 'bg-blue-100 text-blue-800',
      combat: 'bg-red-100 text-red-800',
      roleplay: 'bg-green-100 text-green-800',
      story: 'bg-purple-100 text-purple-800',
      other: 'bg-gray-100 text-gray-800',
    }
    return colors[category] || colors.other
  }

  const getImportanceIcon = (importance: string) => {
    if (importance === 'critical' || importance === 'high') {
      return <Star className="h-4 w-4 text-yellow-500 fill-yellow-500" />
    }
    return null
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <span className="flex items-center">
            <Clock className="h-5 w-5 mr-2" />
            时间轴
          </span>
          <div className="flex space-x-2">
            <Button size="sm" variant="outline">
              <Filter className="h-4 w-4 mr-1" />
              过滤
            </Button>
            <Button size="sm" variant="outline">
              <Download className="h-4 w-4 mr-1" />
              导出
            </Button>
          </div>
        </CardTitle>
      </CardHeader>

      <CardContent>
        {loading ? (
          <div className="text-center text-muted-foreground py-8">
            加载中...
          </div>
        ) : entries.length === 0 ? (
          <div className="text-center text-muted-foreground py-8">
            暂无事件
          </div>
        ) : (
          <div className="relative">
            {/* 时间线 */}
            <div className="absolute left-4 top-0 bottom-0 w-0.5 bg-border" />

            {/* 条目 */}
            <div className="space-y-4 ml-8">
              {entries.map((entry, index) => (
                <div
                  key={entry.id}
                  className="relative"
                >
                  {/* 时间点 */}
                  <div className="absolute -left-8 top-2 w-4 h-4 rounded-full bg-primary border-2 border-background" />

                  {/* 内容 */}
                  <div className={`p-3 rounded-lg border ${entry.is_highlighted ? 'bg-yellow-50 border-yellow-200' : ''}`}>
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <div className="flex items-center space-x-2 mb-1">
                          <span className="text-sm font-medium">
                            {entry.title}
                          </span>
                          <Badge className={getCategoryColor(entry.category)}>
                            {entry.category}
                          </Badge>
                          {getImportanceIcon(entry.importance)}
                        </div>

                        {entry.description && (
                          <p className="text-sm text-muted-foreground mb-2">
                            {entry.description}
                          </p>
                        )}

                        <div className="flex items-center space-x-4 text-xs text-muted-foreground">
                          <span>
                            {new Date(entry.timestamp).toLocaleString('zh-CN')}
                          </span>
                          {entry.character_name && (
                            <span>{entry.character_name}</span>
                          )}
                        </div>
                      </div>

                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => toggleHighlight(entry.id)}
                      >
                        <Star className={`h-4 w-4 ${entry.is_highlighted ? 'fill-yellow-500 text-yellow-500' : ''}`} />
                      </Button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/db/models/timeline.py` | 创建 | 时间轴数据模型 |
| `app/services/timeline.py` | 创建 | 时间轴服务 |
| `app/api/timeline.py` | 创建 | 时间轴 API |
| `app/schemas/timeline.py` | 创建 | 时间轴 Schema |
| `frontend/src/components/game/Timeline.tsx` | 创建 | 时间轴组件 |

---

## 验收标准

- [ ] 事件记录完整
- [ ] 过滤功能有效
- [ ] 重点标记可用
- [ ] 导出功能正常
- [ ] 时间顺序正确
- [ ] 性能良好

---

## 参考文档

- M1-080: 事件日志系统
- M3-001: AI 总结服务

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
