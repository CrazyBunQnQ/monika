# M3-016: 实现跨会话引用

**任务ID**: M3-016
**标题**: 实现跨会话引用
**类型**: backend + frontend (全栈开发)
**预估工时**: 4h
**依赖**: M3-001, M3-027

---

## 任务描述

实现跨会话引用功能，允许用户在不同游戏会话之间引用和关联内容，包括：
- 引用其他会话的事件
- 关联跨会话的线索
- 建立跨会话的承诺和任务
- 追踪跨会话的角色状态变化
- 显示引用的上下文信息

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M3-016-01 | 设计引用数据模型 | CrossReference 表 | 30min |
| M3-016-02 | 实现引用创建服务 | 创建和管理引用 | 45min |
| M3-016-03 | 实现引用查询服务 | 查询引用关系 | 30min |
| M3-016-04 | 实现引用上下文加载 | 加载被引用内容 | 30min |
| M3-016-05 | 实现引用检测 API | 自动检测潜在引用 | 45min |
| M3-016-06 | 实现引用 UI 组件 | 引用选择器和展示器 | 1h |
| M3-016-07 | 实现引用通知系统 | 引用更新通知 | 15min |
| M3-016-08 | 编写引用测试 | 测试覆盖 | 15min |

---

## 后端代码示例

### 引用数据模型

```python
# app/db/models/cross_reference.py
from sqlalchemy import Column, String, DateTime, ForeignKey, Text, JSON
from sqlalchemy.orm import relationship
from datetime import datetime

from app.db.database import Base

class CrossReference(Base):
    """跨会话引用"""
    __tablename__ = "cross_references"

    id = Column(String, primary_key=True, index=True)

    # 引用源
    source_campaign_id = Column(String, ForeignKey("campaigns.id"), nullable=False)
    source_session_id = Column(String, ForeignKey("game_sessions.id"), nullable=False)
    source_event_id = Column(String, ForeignKey("game_events.id"))
    source_type = Column(String(50))  # event/clue/promise/note/etc.

    # 引用目标
    target_campaign_id = Column(String, ForeignKey("campaigns.id"), nullable=False)
    target_session_id = Column(String, ForeignKey("game_sessions.id"))
    target_event_id = Column(String, ForeignKey("game_events.id"))
    target_type = Column(String(50))  # event/clue/promise/note/etc.

    # 引用信息
    reference_type = Column(String(50))  # direct/indirect/context/continuation
    context = Column(Text)  # 引用上下文说明
    metadata = Column(JSON)  # 额外元数据

    # 创建者
    created_by = Column(String, ForeignKey("users.id"), nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)

    # 状态
    is_active = Column(Column(Boolean), default=True)

class ReferenceChain(Base):
    """引用链"""
    __tablename__ = "reference_chains"

    id = Column(String, primary_key=True, index=True)
    campaign_id = Column(String, ForeignKey("campaigns.id"), nullable=False)

    # 链信息
    chain_type = Column(String(50))  # storyline/clue/character/promise
    title = Column(String(200))

    # 引用序列
    reference_ids = Column(JSON)  # [ref_id1, ref_id2, ...]

    # 元数据
    metadata = Column(JSON)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
```

### 引用服务

```python
# app/services/cross_reference.py
from typing import List, Dict, Any, Optional
from datetime import datetime
from sqlalchemy.orm import Session

from app.db.models.cross_reference import CrossReference, ReferenceChain
from app.core.logger import EventLogger

class CrossReferenceService:
    """跨会话引用服务"""

    def __init__(self, db: Session):
        self.db = db
        self.logger = EventLogger()

    async def create_reference(
        self,
        source_campaign_id: str,
        source_session_id: str,
        source_type: str,
        target_campaign_id: str,
        target_type: str,
        source_event_id: Optional[str] = None,
        target_session_id: Optional[str] = None,
        target_event_id: Optional[str] = None,
        reference_type: str = "direct",
        context: Optional[str] = None,
        metadata: Optional[Dict] = None,
        created_by: str = None,
    ) -> CrossReference:
        """创建引用

        Args:
            source_campaign_id: 源战役 ID
            source_session_id: 源会话 ID
            source_type: 源类型
            target_campaign_id: 目标战役 ID
            target_type: 目标类型
            source_event_id: 源事件 ID
            target_session_id: 目标会话 ID
            target_event_id: 目标事件 ID
            reference_type: 引用类型
            context: 引用上下文
            metadata: 元数据
            created_by: 创建者 ID

        Returns:
            创建的引用
        """
        reference = CrossReference(
            id=self._generate_reference_id(),
            source_campaign_id=source_campaign_id,
            source_session_id=source_session_id,
            source_type=source_type,
            target_campaign_id=target_campaign_id,
            target_type=target_type,
            source_event_id=source_event_id,
            target_session_id=target_session_id,
            target_event_id=target_event_id,
            reference_type=reference_type,
            context=context,
            metadata=metadata or {},
            created_by=created_by,
        )

        self.db.add(reference)
        self.db.commit()
        self.db.refresh(reference)

        # 尝试加入或创建引用链
        await self._update_reference_chain(reference)

        return reference

    async def get_references(
        self,
        campaign_id: str,
        session_id: Optional[str] = None,
        reference_type: Optional[str] = None,
    ) -> List[CrossReference]:
        """获取引用列表

        Args:
            campaign_id: 战役 ID
            session_id: 会话 ID
            reference_type: 引用类型

        Returns:
            引用列表
        """
        query = self.db.query(CrossReference).filter(
            CrossReference.source_campaign_id == campaign_id,
            CrossReference.is_active == True,
        )

        if session_id:
            query = query.filter(
                (CrossReference.source_session_id == session_id) |
                (CrossReference.target_session_id == session_id)
            )

        if reference_type:
            query = query.filter(CrossReference.reference_type == reference_type)

        return query.order_by(CrossReference.created_at.desc()).all()

    async def get_reference_context(
        self,
        reference_id: str,
    ) -> Dict[str, Any]:
        """获取引用上下文

        Args:
            reference_id: 引用 ID

        Returns:
            引用上下文信息
        """
        reference = self.db.query(CrossReference).filter(
            CrossReference.id == reference_id
        ).first()

        if not reference:
            raise ValueError("引用不存在")

        # 加载源内容
        source_context = await self._load_content_context(
            campaign_id=reference.source_campaign_id,
            session_id=reference.source_session_id,
            event_id=reference.source_event_id,
            content_type=reference.source_type,
        )

        # 加载目标内容
        target_context = await self._load_content_context(
            campaign_id=reference.target_campaign_id,
            session_id=reference.target_session_id,
            event_id=reference.target_event_id,
            content_type=reference.target_type,
        )

        return {
            "reference": {
                "id": reference.id,
                "type": reference.reference_type,
                "context": reference.context,
                "created_at": reference.created_at.isoformat(),
            },
            "source": source_context,
            "target": target_context,
        }

    async def detect_potential_references(
        self,
        campaign_id: str,
        session_id: str,
        event_description: str,
    ) -> List[Dict[str, Any]]:
        """检测潜在的引用

        Args:
            campaign_id: 战役 ID
            session_id: 会话 ID
            event_description: 事件描述

        Returns:
            潜在引用列表
        """
        # 获取同战役的历史事件
        past_events = await self.logger.get_events(
            campaign_id=campaign_id,
        )

        # 过滤出当前会话之前的事件
        past_events = [
            e for e in past_events
            if e.session_id != session_id
        ]

        # 简单的关键词匹配（实际可以使用更复杂的 NLP）
        potential_refs = []

        for past_event in past_events:
            # 检查是否有共同的关键词
            similarity = self._calculate_similarity(
                event_description,
                past_event.description,
            )

            if similarity > 0.3:  # 相似度阈值
                potential_refs.append({
                    "event_id": past_event.id,
                    "session_id": past_event.session_id,
                    "description": past_event.description,
                    "similarity": similarity,
                    "suggested_type": "context",
                })

        # 按相似度排序
        potential_refs.sort(key=lambda x: x["similarity"], reverse=True)

        return potential_refs[:10]  # 返回前10个

    async def get_reference_chain(
        self,
        reference_id: str,
    ) -> List[Dict[str, Any]]:
        """获取引用链

        Args:
            reference_id: 引用 ID

        Returns:
            引用链
        """
        reference = self.db.query(CrossReference).filter(
            CrossReference.id == reference_id
        ).first()

        if not reference:
            raise ValueError("引用不存在")

        # 查找相关链
        chain = self.db.query(ReferenceChain).filter(
            ReferenceChain.campaign_id == reference.source_campaign_id,
        ).all()

        # 找到包含此引用的链
        for ref_chain in chain:
            if reference_id in ref_chain.reference_ids:
                # 加载链中的所有引用
                references = self.db.query(CrossReference).filter(
                    CrossReference.id.in_(ref_chain.reference_ids)
                ).all()

                return [
                    {
                        "id": ref.id,
                        "source": {
                            "session_id": ref.source_session_id,
                            "type": ref.source_type,
                        },
                        "target": {
                            "session_id": ref.target_session_id,
                            "type": ref.target_type,
                        },
                        "type": ref.reference_type,
                    }
                    for ref in references
                ]

        return []

    async def _load_content_context(
        self,
        campaign_id: str,
        session_id: Optional[str],
        event_id: Optional[str],
        content_type: str,
    ) -> Dict[str, Any]:
        """加载内容上下文"""
        if event_id:
            # 加载事件
            event = await self.logger.get_event(event_id)
            if event:
                return {
                    "type": content_type,
                    "session_id": session_id,
                    "event_id": event_id,
                    "description": event.description,
                    "timestamp": event.timestamp.isoformat(),
                    "data": event.data,
                }

        # 加载会话摘要
        if session_id:
            # 从缓存或数据库加载会话摘要
            summary = await self._get_session_summary(session_id)
            if summary:
                return {
                    "type": "session",
                    "session_id": session_id,
                    "title": summary.get("title", ""),
                    "summary": summary.get("brief", ""),
                }

        return {
            "type": content_type,
            "session_id": session_id,
        }

    async def _update_reference_chain(self, reference: CrossReference):
        """更新引用链"""
        # 查找是否有相关的链
        chain = self.db.query(ReferenceChain).filter(
            ReferenceChain.campaign_id == reference.source_campaign_id,
            ReferenceChain.chain_type == reference.target_type,
        ).first()

        if chain:
            # 添加到现有链
            if reference.id not in chain.reference_ids:
                chain.reference_ids.append(reference.id)
                chain.updated_at = datetime.utcnow()
        else:
            # 创建新链
            chain = ReferenceChain(
                id=self._generate_chain_id(),
                campaign_id=reference.source_campaign_id,
                chain_type=reference.target_type,
                title=f"{reference.target_type} 引用链",
                reference_ids=[reference.id],
            )
            self.db.add(chain)

        self.db.commit()

    async def _get_session_summary(self, session_id: str) -> Optional[Dict]:
        """获取会话摘要"""
        # 简化实现，实际应该从 summary 表加载
        return None

    def _calculate_similarity(self, text1: str, text2: str) -> float:
        """计算文本相似度"""
        # 简单的 Jaccard 相似度
        words1 = set(text1.lower().split())
        words2 = set(text2.lower().split())

        if not words1 or not words2:
            return 0.0

        intersection = len(words1 & words2)
        union = len(words1 | words2)

        return intersection / union if union > 0 else 0.0

    def _generate_reference_id(self) -> str:
        """生成引用 ID"""
        import uuid
        return f"xref_{uuid.uuid4().hex[:12]}"

    def _generate_chain_id(self) -> str:
        """生成链 ID"""
        import uuid
        return f"chain_{uuid.uuid4().hex[:12]}"
```

### 引用 API

```python
# app/api/cross_reference.py
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session
from pydantic import BaseModel
from typing import List, Optional

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.cross_reference import CrossReferenceService

router = APIRouter(prefix="/references", tags=["cross-references"])

class CreateReferenceRequest(BaseModel):
    source_campaign_id: str
    source_session_id: str
    source_type: str
    target_campaign_id: str
    target_type: str
    source_event_id: Optional[str] = None
    target_session_id: Optional[str] = None
    target_event_id: Optional[str] = None
    reference_type: str = "direct"
    context: Optional[str] = None
    metadata: Optional[dict] = None

@router.post("")
async def create_reference(
    request: CreateReferenceRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """创建引用"""
    service = CrossReferenceService(db)

    reference = await service.create_reference(
        source_campaign_id=request.source_campaign_id,
        source_session_id=request.source_session_id,
        source_type=request.source_type,
        target_campaign_id=request.target_campaign_id,
        target_type=request.target_type,
        source_event_id=request.source_event_id,
        target_session_id=request.target_session_id,
        target_event_id=request.target_event_id,
        reference_type=request.reference_type,
        context=request.context,
        metadata=request.metadata,
        created_by=current_user.id,
    )

    return {"reference_id": reference.id}

@router.get("")
async def get_references(
    campaign_id: str,
    session_id: Optional[str] = None,
    reference_type: Optional[str] = None,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取引用列表"""
    service = CrossReferenceService(db)

    references = await service.get_references(
        campaign_id=campaign_id,
        session_id=session_id,
        reference_type=reference_type,
    )

    return {
        "references": [
            {
                "id": ref.id,
                "source": {
                    "campaign_id": ref.source_campaign_id,
                    "session_id": ref.source_session_id,
                    "type": ref.source_type,
                    "event_id": ref.source_event_id,
                },
                "target": {
                    "campaign_id": ref.target_campaign_id,
                    "session_id": ref.target_session_id,
                    "type": ref.target_type,
                    "event_id": ref.target_event_id,
                },
                "type": ref.reference_type,
                "context": ref.context,
                "created_at": ref.created_at.isoformat(),
            }
            for ref in references
        ]
    }

@router.get("/{reference_id}/context")
async def get_reference_context(
    reference_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取引用上下文"""
    service = CrossReferenceService(db)

    context = await service.get_reference_context(reference_id=reference_id)

    return context

@router.get("/{reference_id}/chain")
async def get_reference_chain(
    reference_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取引用链"""
    service = CrossReferenceService(db)

    chain = await service.get_reference_chain(reference_id=reference_id)

    return {"chain": chain}

@router.post("/detect")
async def detect_potential_references(
    campaign_id: str,
    session_id: str,
    description: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """检测潜在引用"""
    service = CrossReferenceService(db)

    potential = await service.detect_potential_references(
        campaign_id=campaign_id,
        session_id=session_id,
        event_description=description,
    )

    return {"potential_references": potential}
```

---

## 前端代码示例

### 引用选择器组件

```typescript
// frontend/src/components/reference/ReferencePicker.tsx
import React, { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Search, Link } from 'lucide-react';

interface ReferencePickerProps {
  campaignId: string;
  sessionId: string;
  onSelect: (reference: {
    target_session_id: string;
    target_event_id?: string;
    context?: string;
  }) => void;
}

export function ReferencePicker({
  campaignId,
  sessionId,
  onSelect,
}: ReferencePickerProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState('');
  const [sessions, setSessions] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (open) {
      loadSessions();
    }
  }, [open, campaignId]);

  const loadSessions = async () => {
    setLoading(true);
    try {
      const response = await fetch(
        `/api/sessions?campaign_id=${campaignId}&exclude=${sessionId}`
      );
      const data = await response.json();
      setSessions(data.sessions || []);
    } catch (error) {
      console.error('加载会话失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleDetect = async () => {
    if (!search.trim()) return;

    setLoading(true);
    try {
      const response = await fetch('/api/references/detect', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          campaign_id: campaignId,
          session_id: sessionId,
          description: search,
        }),
      });
      const data = await response.json();

      // 显示检测结果
      console.log('潜在引用:', data.potential_references);
    } catch (error) {
      console.error('检测引用失败:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          <Link className="h-4 w-4 mr-2" />
          引用
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>引用其他会话内容</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          {/* 搜索框 */}
          <div className="flex gap-2">
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="搜索关键词或输入内容自动检测..."
            />
            <Button onClick={handleDetect} disabled={loading}>
              <Search className="h-4 w-4 mr-2" />
              检测
            </Button>
          </div>

          {/* 会话列表 */}
          <div className="space-y-2 max-h-96 overflow-y-auto">
            {sessions.map((session) => (
              <div
                key={session.id}
                className="border rounded-lg p-3 cursor-pointer hover:bg-accent"
                onClick={() => {
                  onSelect({
                    target_session_id: session.id,
                    context: search,
                  });
                  setOpen(false);
                }}
              >
                <div className="flex items-center justify-between">
                  <div>
                    <h4 className="font-medium">{session.title || '未命名会话'}</h4>
                    <p className="text-sm text-muted-foreground">
                      {new Date(session.started_at).toLocaleDateString()}
                    </p>
                  </div>
                  <Badge variant="outline">{session.events_count} 事件</Badge>
                </div>
              </div>
            ))}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
```

### 引用展示组件

```typescript
// frontend/src/components/reference/ReferenceDisplay.tsx
import React, { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { ChevronDown, ChevronUp, ExternalLink } from 'lucide-react';

interface ReferenceDisplayProps {
  reference: {
    id: string;
    source: {
      session_id: string;
      type: string;
      event_id?: string;
    };
    target: {
      session_id: string;
      type: string;
      event_id?: string;
    };
    type: string;
    context?: string;
    created_at: string;
  };
  onViewContext?: (referenceId: string) => void;
}

export function ReferenceDisplay({ reference, onViewContext }: ReferenceDisplayProps) {
  const [expanded, setExpanded] = useState(false);

  const typeLabels: Record<string, string> = {
    direct: '直接引用',
    indirect: '间接引用',
    context: '上下文关联',
    continuation: '续作',
  };

  return (
    <Card className="border-l-4 border-l-primary">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Badge variant="outline">{typeLabels[reference.type]}</Badge>
            <span className="text-sm text-muted-foreground">
              {reference.source.type} → {reference.target.type}
            </span>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => onViewContext?.(reference.id)}
            >
              <ExternalLink className="h-3 w-3 mr-1" />
              查看上下文
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setExpanded(!expanded)}
            >
              {expanded ? (
                <ChevronUp className="h-4 w-4" />
              ) : (
                <ChevronDown className="h-4 w-4" />
              )}
            </Button>
          </div>
        </div>
      </CardHeader>

      <CardContent>
        {reference.context && (
          <p className="text-sm text-muted-foreground mb-2">
            {reference.context}
          </p>
        )}

        {expanded && (
          <div className="space-y-2 pt-2 border-t">
            <div className="flex items-center gap-2 text-sm">
              <span className="font-medium">来源:</span>
              <Badge variant="secondary">{reference.source.session_id.slice(0, 8)}</Badge>
            </div>
            <div className="flex items-center gap-2 text-sm">
              <span className="font-medium">目标:</span>
              <Badge variant="secondary">{reference.target.session_id.slice(0, 8)}</Badge>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/db/models/cross_reference.py` | 创建 | 引用数据模型 |
| `app/services/cross_reference.py` | 创建 | 引用服务 |
| `app/api/cross_reference.py` | 创建 | 引用 API |
| `frontend/src/components/reference/ReferencePicker.tsx` | 创建 | 引用选择器组件 |
| `frontend/src/components/reference/ReferenceDisplay.tsx` | 创建 | 引用展示组件 |
| `tests/test_cross_reference.py` | 创建 | 引用测试 |

---

## 验收标准

- [ ] 能创建跨会话引用
- [ ] 引用上下文正确加载
- [ ] 潜在引用检测准确
- [ ] 引用链完整追踪
- [ ] 引用展示清晰
- [ ] 引用通知及时
- [ ] 引用与被引用内容正确关联
- [ ] 引用类型分类合理

---

## 参考文档

- M3-001: AI 总结服务
- M3-027: 全文检索功能
- 引用系统最佳实践

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
