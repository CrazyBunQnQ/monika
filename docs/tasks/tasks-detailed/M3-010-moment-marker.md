# M3-010: 实现重要时刻标记

**任务ID**: M3-010
**标题**: 实现重要时刻标记
**类型**: backend + frontend (全栈开发)
**预估工时**: 4h
**依赖**: M1-080, M3-006

---

## 任务描述

实现游戏过程中的重要时刻标记功能，允许 KP 和玩家标记关键时刻（如发现重要线索、剧情转折、角色死亡等），方便后续回顾和检索。支持标记分类、注释和可见性控制。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M3-010-01 | 设计标记数据模型 | MomentMarker 表结构 | 30min |
| M3-010-02 | 实现标记创建服务 | 创建和更新标记 | 45min |
| M3-010-03 | 实现标记查询 API | 查询和过滤标记 | 30min |
| M3-010-04 | 实现标记分类系统 | 预设分类和自定义分类 | 30min |
| M3-010-05 | 实现可见性控制 | 公开/私密/KP-only | 30min |
| M3-010-06 | 实现标记 UI 组件 | 标记按钮和弹窗 | 1h |
| M3-010-07 | 实现标记时间线展示 | 在时间线上显示标记 | 30min |
| M3-010-08 | 编写标记测试 | 测试覆盖 | 15min |

---

## 后端代码示例

### 标记数据模型

```python
# app/db/models/moment.py
from sqlalchemy import Column, String, DateTime, ForeignKey, Text, Boolean, JSON
from sqlalchemy.orm import relationship
from datetime import datetime

from app.db.database import Base

class MomentMarker(Base):
    """重要时刻标记"""
    __tablename__ = "moment_markers"

    id = Column(String, primary_key=True, index=True)
    campaign_id = Column(String, ForeignKey("campaigns.id"), nullable=False)
    session_id = Column(String, ForeignKey("game_sessions.id"), nullable=False)

    # 标记信息
    title = Column(String(200), nullable=False)
    description = Column(Text)
    category = Column(String(50))  # 标记分类

    # 关联
    event_id = Column(String, ForeignKey("game_events.id"))  # 关联的事件
    scene_id = Column(String, ForeignKey("scenes.id"))  # 所在场景

    # 可见性
    visibility = Column(String(20), default="public")  # public/private/kp_only
    visible_to = Column(JSON)  # 特定用户可见

    # 创建者
    created_by = Column(String, ForeignKey("users.id"), nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)

    # 元数据
    color = Column(String(7), default="#FF6B6B")  # 标记颜色
    icon = Column(String(50))  # 标记图标
    is_starred = Column(Boolean, default=False)  # 是否星标

class MarkerCategory(Base):
    """标记分类"""
    __tablename__ = "marker_categories"

    id = Column(String, primary_key=True, index=True)
    campaign_id = Column(String, ForeignKey("campaigns.id"), nullable=False)

    # 分类信息
    name = Column(String(50), nullable=False)
    description = Column(String(200))
    color = Column(String(7), default="#6B6BFF")
    icon = Column(String(50))

    # 预设分类标识
    is_preset = Column(Boolean, default=False)
    preset_key = Column(String(50))  # clue/plot_turn/combat/death/etc.

    created_at = Column(DateTime, default=datetime.utcnow)

# 预设分类定义
PRESET_CATEGORIES = [
    {
        "preset_key": "clue",
        "name": "线索发现",
        "description": "发现重要线索的时刻",
        "color": "#FFD93D",
        "icon": "search",
    },
    {
        "preset_key": "plot_turn",
        "name": "剧情转折",
        "description": "故事发生重大转折",
        "color": "#FF6B6B",
        "icon": "arrow-right",
    },
    {
        "preset_key": "combat",
        "name": "战斗时刻",
        "description": "重要战斗事件",
        "color": "#FF4444",
        "icon": "sword",
    },
    {
        "preset_key": "death",
        "name": "角色死亡",
        "description": "角色死亡时刻",
        "color": "#000000",
        "icon": "skull",
    },
    {
        "preset_key": "madness",
        "name": "疯狂时刻",
        "description": "触发疯狂",
        "color": "#9B59B6",
        "icon": "brain",
    },
    {
        "preset_key": "revelation",
        "name": "真相揭露",
        "description": "重要真相揭露",
        "color": "#3498DB",
        "icon": "lightbulb",
    },
    {
        "preset_key": "humor",
        "name": "趣味时刻",
        "description": "有趣的对话或事件",
        "color": "#2ECC71",
        "icon": "laugh",
    },
]
```

### 标记服务

```python
# app/services/moment_marker.py
from typing import List, Dict, Any, Optional
from sqlalchemy.orm import Session
from datetime import datetime

from app.db.models.moment import MomentMarker, MarkerCategory, PRESET_CATEGORIES
from app.core.logger import EventLogger

class MomentMarkerService:
    """重要时刻标记服务"""

    def __init__(self, db: Session):
        self.db = db
        self.logger = EventLogger()

    async def create_marker(
        self,
        campaign_id: str,
        session_id: str,
        title: str,
        created_by: str,
        description: Optional[str] = None,
        category: Optional[str] = None,
        event_id: Optional[str] = None,
        scene_id: Optional[str] = None,
        visibility: str = "public",
        visible_to: Optional[List[str]] = None,
        color: Optional[str] = None,
        icon: Optional[str] = None,
    ) -> MomentMarker:
        """创建标记"""
        # 获取或创建分类
        if category:
            category_obj = await self._get_or_create_category(campaign_id, category)
            if not color:
                color = category_obj.color
            if not icon:
                icon = category_obj.icon

        marker = MomentMarker(
            id=self._generate_marker_id(),
            campaign_id=campaign_id,
            session_id=session_id,
            title=title,
            description=description,
            category=category,
            event_id=event_id,
            scene_id=scene_id,
            visibility=visibility,
            visible_to=visible_to or [],
            created_by=created_by,
            color=color or "#FF6B6B",
            icon=icon,
        )

        self.db.add(marker)
        self.db.commit()
        self.db.refresh(marker)

        # 记录事件
        await self.logger.log_event(
            campaign_id=campaign_id,
            event_type="marker_created",
            description=f"创建标记: {title}",
            data={
                "marker_id": marker.id,
                "category": category,
            },
            user_id=created_by,
        )

        return marker

    async def get_markers(
        self,
        campaign_id: str,
        category: Optional[str] = None,
        visibility: Optional[str] = None,
        user_id: Optional[str] = None,
        session_id: Optional[str] = None,
        starred_only: bool = False,
    ) -> List[MomentMarker]:
        """获取标记列表"""
        query = self.db.query(MomentMarker).filter(
            MomentMarker.campaign_id == campaign_id
        )

        # 过滤条件
        if category:
            query = query.filter(MomentMarker.category == category)
        if session_id:
            query = query.filter(MomentMarker.session_id == session_id)
        if starred_only:
            query = query.filter(MomentMarker.is_starred == True)

        markers = query.order_by(MomentMarker.created_at.desc()).all()

        # 可见性过滤
        if user_id:
            markers = [m for m in markers if self._check_visibility(m, user_id)]

        return markers

    async def update_marker(
        self,
        marker_id: str,
        user_id: str,
        title: Optional[str] = None,
        description: Optional[str] = None,
        category: Optional[str] = None,
        visibility: Optional[str] = None,
        color: Optional[str] = None,
        is_starred: Optional[bool] = None,
    ) -> Optional[MomentMarker]:
        """更新标记"""
        marker = self.db.query(MomentMarker).filter(
            MomentMarker.id == marker_id
        ).first()

        if not marker:
            return None

        # 权限检查
        if marker.created_by != user_id:
            raise PermissionError("只有创建者可以修改标记")

        # 更新字段
        if title is not None:
            marker.title = title
        if description is not None:
            marker.description = description
        if category is not None:
            marker.category = category
        if visibility is not None:
            marker.visibility = visibility
        if color is not None:
            marker.color = color
        if is_starred is not None:
            marker.is_starred = is_starred

        self.db.commit()
        self.db.refresh(marker)
        return marker

    async def delete_marker(self, marker_id: str, user_id: str) -> bool:
        """删除标记"""
        marker = self.db.query(MomentMarker).filter(
            MomentMarker.id == marker_id
        ).first()

        if not marker:
            return False

        # 权限检查
        if marker.created_by != user_id:
            raise PermissionError("只有创建者可以删除标记")

        self.db.delete(marker)
        self.db.commit()
        return True

    async def get_categories(
        self,
        campaign_id: str,
    ) -> List[MarkerCategory]:
        """获取标记分类"""
        # 确保预设分类存在
        await self._ensure_preset_categories(campaign_id)

        categories = self.db.query(MarkerCategory).filter(
            MarkerCategory.campaign_id == campaign_id
        ).all()

        return categories

    async def _get_or_create_category(
        self,
        campaign_id: str,
        category_name: str,
    ) -> MarkerCategory:
        """获取或创建分类"""
        category = self.db.query(MarkerCategory).filter(
            MarkerCategory.campaign_id == campaign_id,
            MarkerCategory.name == category_name,
        ).first()

        if not category:
            # 检查是否是预设分类
            preset = next((c for c in PRESET_CATEGORIES if c["name"] == category_name), None)
            if preset:
                category = MarkerCategory(
                    id=self._generate_category_id(),
                    campaign_id=campaign_id,
                    name=preset["name"],
                    description=preset["description"],
                    color=preset["color"],
                    icon=preset["icon"],
                    is_preset=True,
                    preset_key=preset["preset_key"],
                )
            else:
                category = MarkerCategory(
                    id=self._generate_category_id(),
                    campaign_id=campaign_id,
                    name=category_name,
                    is_preset=False,
                )

            self.db.add(category)
            self.db.commit()
            self.db.refresh(category)

        return category

    async def _ensure_preset_categories(self, campaign_id: str):
        """确保预设分类存在"""
        for preset in PRESET_CATEGORIES:
            existing = self.db.query(MarkerCategory).filter(
                MarkerCategory.campaign_id == campaign_id,
                MarkerCategory.preset_key == preset["preset_key"],
            ).first()

            if not existing:
                category = MarkerCategory(
                    id=self._generate_category_id(),
                    campaign_id=campaign_id,
                    name=preset["name"],
                    description=preset["description"],
                    color=preset["color"],
                    icon=preset["icon"],
                    is_preset=True,
                    preset_key=preset["preset_key"],
                )
                self.db.add(category)

        self.db.commit()

    def _check_visibility(self, marker: MomentMarker, user_id: str) -> bool:
        """检查可见性"""
        if marker.visibility == "public":
            return True
        if marker.visibility == "kp_only":
            # 需要 KP 权限检查（简化处理）
            return True
        if marker.visibility == "private":
            return marker.created_by == user_id
        if marker.visible_to:
            return user_id in marker.visible_to
        return False

    def _generate_marker_id(self) -> str:
        """生成标记 ID"""
        import uuid
        return f"marker_{uuid.uuid4().hex[:12]}"

    def _generate_category_id(self) -> str:
        """生成分类 ID"""
        import uuid
        return f"mcat_{uuid.uuid4().hex[:12]}"
```

### 标记 API

```python
# app/api/marker.py
from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy.orm import Session
from pydantic import BaseModel
from typing import List, Optional

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.moment_marker import MomentMarkerService

router = APIRouter(prefix="/markers", tags=["markers"])

class CreateMarkerRequest(BaseModel):
    campaign_id: str
    session_id: str
    title: str
    description: Optional[str] = None
    category: Optional[str] = None
    event_id: Optional[str] = None
    scene_id: Optional[str] = None
    visibility: str = "public"
    visible_to: Optional[List[str]] = None
    color: Optional[str] = None
    icon: Optional[str] = None

class UpdateMarkerRequest(BaseModel):
    title: Optional[str] = None
    description: Optional[str] = None
    category: Optional[str] = None
    visibility: Optional[str] = None
    color: Optional[str] = None
    is_starred: Optional[bool] = None

@router.post("")
async def create_marker(
    request: CreateMarkerRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """创建标记"""
    service = MomentMarkerService(db)

    marker = await service.create_marker(
        campaign_id=request.campaign_id,
        session_id=request.session_id,
        title=request.title,
        created_by=current_user.id,
        description=request.description,
        category=request.category,
        event_id=request.event_id,
        scene_id=request.scene_id,
        visibility=request.visibility,
        visible_to=request.visible_to,
        color=request.color,
        icon=request.icon,
    )

    return {
        "id": marker.id,
        "title": marker.title,
        "created_at": marker.created_at.isoformat(),
    }

@router.get("")
async def get_markers(
    campaign_id: str,
    category: Optional[str] = None,
    session_id: Optional[str] = None,
    starred_only: bool = False,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取标记列表"""
    service = MomentMarkerService(db)

    markers = await service.get_markers(
        campaign_id=campaign_id,
        category=category,
        user_id=current_user.id,
        session_id=session_id,
        starred_only=starred_only,
    )

    return {
        "markers": [
            {
                "id": m.id,
                "title": m.title,
                "description": m.description,
                "category": m.category,
                "color": m.color,
                "icon": m.icon,
                "created_at": m.created_at.isoformat(),
                "created_by": m.created_by,
                "is_starred": m.is_starred,
            }
            for m in markers
        ]
    }

@router.put("/{marker_id}")
async def update_marker(
    marker_id: str,
    request: UpdateMarkerRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """更新标记"""
    service = MomentMarkerService(db)

    marker = await service.update_marker(
        marker_id=marker_id,
        user_id=current_user.id,
        title=request.title,
        description=request.description,
        category=request.category,
        visibility=request.visibility,
        color=request.color,
        is_starred=request.is_starred,
    )

    if not marker:
        raise HTTPException(status_code=404, detail="标记不存在")

    return {"message": "标记已更新"}

@router.delete("/{marker_id}")
async def delete_marker(
    marker_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """删除标记"""
    service = MomentMarkerService(db)

    success = await service.delete_marker(
        marker_id=marker_id,
        user_id=current_user.id,
    )

    if not success:
        raise HTTPException(status_code=404, detail="标记不存在")

    return {"message": "标记已删除"}

@router.get("/categories")
async def get_categories(
    campaign_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取标记分类"""
    service = MomentMarkerService(db)

    categories = await service.get_categories(campaign_id=campaign_id)

    return {
        "categories": [
            {
                "id": c.id,
                "name": c.name,
                "description": c.description,
                "color": c.color,
                "icon": c.icon,
                "is_preset": c.is_preset,
            }
            for c in categories
        ]
    }
```

---

## 前端代码示例

### 标记按钮组件

```typescript
// frontend/src/components/marker/MarkerButton.tsx
import React, { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Bookmark } from 'lucide-react';

interface MarkerButtonProps {
  eventId?: string;
  sceneId?: string;
  onMarkerCreated?: (marker: any) => void;
}

export function MarkerButton({ eventId, sceneId, onMarkerCreated }: MarkerButtonProps) {
  const [open, setOpen] = useState(false);
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [category, setCategory] = useState<string>('');
  const [loading, setLoading] = useState(false);

  const categories = [
    { value: 'clue', label: '线索发现', color: '#FFD93D' },
    { value: 'plot_turn', label: '剧情转折', color: '#FF6B6B' },
    { value: 'combat', label: '战斗时刻', color: '#FF4444' },
    { value: 'death', label: '角色死亡', color: '#000000' },
    { value: 'madness', label: '疯狂时刻', color: '#9B59B6' },
    { value: 'revelation', label: '真相揭露', color: '#3498DB' },
    { value: 'humor', label: '趣味时刻', color: '#2ECC71' },
  ];

  const handleCreate = async () => {
    if (!title.trim()) return;

    setLoading(true);
    try {
      // 获取当前 campaign 和 session
      const campaignId = localStorage.getItem('currentCampaignId') || '';
      const sessionId = localStorage.getItem('currentSessionId') || '';

      const response = await fetch('/api/markers', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          campaign_id: campaignId,
          session_id: sessionId,
          title,
          description,
          category,
          event_id: eventId,
          scene_id: sceneId,
        }),
      });

      if (response.ok) {
        const data = await response.json();
        onMarkerCreated?.(data);

        // 重置表单
        setTitle('');
        setDescription('');
        setCategory('');
        setOpen(false);
      }
    } catch (error) {
      console.error('创建标记失败:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm">
          <Bookmark className="h-4 w-4 mr-2" />
          标记时刻
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-80">
        <div className="space-y-4">
          <div>
            <Label htmlFor="title">标题</Label>
            <Input
              id="title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="输入标记标题..."
            />
          </div>

          <div>
            <Label htmlFor="category">分类</Label>
            <Select value={category} onValueChange={setCategory}>
              <SelectTrigger>
                <SelectValue placeholder="选择分类" />
              </SelectTrigger>
              <SelectContent>
                {categories.map((cat) => (
                  <SelectItem key={cat.value} value={cat.value}>
                    <div className="flex items-center gap-2">
                      <div
                        className="w-3 h-3 rounded"
                        style={{ backgroundColor: cat.color }}
                      />
                      {cat.label}
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div>
            <Label htmlFor="description">描述（可选）</Label>
            <Textarea
              id="description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="添加描述..."
              rows={3}
            />
          </div>

          <Button
            onClick={handleCreate}
            disabled={!title.trim() || loading}
            className="w-full"
          >
            {loading ? '创建中...' : '创建标记'}
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
}
```

### 标记列表组件

```typescript
// frontend/src/components/marker/MarkerList.tsx
import React, { useEffect, useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Bookmark, Star, Filter } from 'lucide-react';

interface Marker {
  id: string;
  title: string;
  description: string | null;
  category: string | null;
  color: string;
  icon: string;
  created_at: string;
  created_by: string;
  is_starred: boolean;
}

export function MarkerList({ campaignId }: { campaignId: string }) {
  const [markers, setMarkers] = useState<Marker[]>([]);
  const [filter, setFilter] = useState<string>('all');
  const [starredOnly, setStarredOnly] = useState(false);

  useEffect(() => {
    loadMarkers();
  }, [campaignId, filter, starredOnly]);

  const loadMarkers = async () => {
    const params = new URLSearchParams({
      campaign_id: campaignId,
      starred_only: starredOnly.toString(),
    });
    if (filter !== 'all') {
      params.append('category', filter);
    }

    const response = await fetch(`/api/markers?${params}`);
    const data = await response.json();
    setMarkers(data.markers);
  };

  const toggleStar = async (markerId: string) => {
    await fetch(`/api/markers/${markerId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ is_starred: true }),
    });
    loadMarkers();
  };

  const categories = [
    { value: 'all', label: '全部' },
    { value: 'clue', label: '线索发现' },
    { value: 'plot_turn', label: '剧情转折' },
    { value: 'combat', label: '战斗时刻' },
    { value: 'death', label: '角色死亡' },
    { value: 'madness', label: '疯狂时刻' },
    { value: 'revelation', label: '真相揭露' },
    { value: 'humor', label: '趣味时刻' },
  ];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold">重要时刻</h3>
        <div className="flex items-center gap-2">
          <Filter className="h-4 w-4" />
          <Select value={filter} onValueChange={setFilter}>
            <SelectTrigger className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {categories.map((cat) => (
                <SelectItem key={cat.value} value={cat.value}>
                  {cat.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            variant={starredOnly ? 'default' : 'outline'}
            size="sm"
            onClick={() => setStarredOnly(!starredOnly)}
          >
            <Star className="h-4 w-4" />
          </Button>
        </div>
      </div>

      <div className="space-y-3">
        {markers.map((marker) => (
          <Card key={marker.id}>
            <CardHeader className="pb-3">
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-2">
                  <div
                    className="w-3 h-3 rounded-full"
                    style={{ backgroundColor: marker.color }}
                  />
                  <CardTitle className="text-base">{marker.title}</CardTitle>
                  {marker.category && (
                    <Badge variant="secondary">{marker.category}</Badge>
                  )}
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => toggleStar(marker.id)}
                >
                  <Star
                    className={`h-4 w-4 ${
                      marker.is_starred ? 'fill-yellow-400' : ''
                    }`}
                  />
                </Button>
              </div>
            </CardHeader>
            {marker.description && (
              <CardContent>
                <p className="text-sm text-muted-foreground">
                  {marker.description}
                </p>
              </CardContent>
            )}
          </Card>
        ))}
      </div>

      {markers.length === 0 && (
        <div className="text-center py-8 text-muted-foreground">
          <Bookmark className="h-12 w-12 mx-auto mb-4 opacity-50" />
          <p>还没有标记</p>
          <p className="text-sm">标记重要时刻以便后续回顾</p>
        </div>
      )}
    </div>
  );
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/db/models/moment.py` | 创建 | 标记数据模型 |
| `app/services/moment_marker.py` | 创建 | 标记服务 |
| `app/api/marker.py` | 创建 | 标记 API |
| `frontend/src/components/marker/MarkerButton.tsx` | 创建 | 标记按钮组件 |
| `frontend/src/components/marker/MarkerList.tsx` | 创建 | 标记列表组件 |
| `tests/test_marker.py` | 创建 | 标记测试 |

---

## 验收标准

- [ ] 能创建带分类的标记
- [ ] 标记可见性控制有效
- [ ] 星标功能正常
- [ ] 分类过滤有效
- [ ] 标记与事件正确关联
- [ ] 颜色和图标正确显示
- [ ] 标记在时间线上正确展示

---

## 参考文档

- M1-080: 事件日志系统
- M3-006: 事件写入服务
- shadcn/ui Popover 文档

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
