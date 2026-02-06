# M5-003: 实现成长记录系统

**任务ID**: M5-003
**标题**: 实现成长记录系统
**类型**: backend (后端开发)
**预估工时**: 2.5h
**依赖**: M1-003

---

## 任务描述

实现角色成长记录系统，追踪技能成长、属性变化、关键经历等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M5-003-01 | 设计成长记录模型 | Schema | 25min |
| M5-003-02 | 实现记录服务 | Record Service | 30min |
| M5-003-03 | 实现成长追踪 | Tracking | 35min |
| M5-003-04 | 实现里程碑系统 | Milestones | 25min |
| M5-003-05 | 实现成长 API | Growth API | 30min |
| M5-003-06 | 编写成长测试 | 测试覆盖 | 25min |

---

## 成长记录模型

```python
# app/db/models/growth.py
from sqlalchemy import Column, String, Integer, DateTime, Text, ForeignKey, JSON
from sqlalchemy.sql import func
from app.db.database import Base

class GrowthRecord(Base):
    """成长记录"""
    __tablename__ = "growth_records"

    id = Column(String, primary_key=True, index=True)
    character_id = Column(String, ForeignKey("characters.id"), nullable=False)
    campaign_id = Column(String, ForeignKey("campaigns.id"), nullable=False)

    # 记录类型
    record_type = Column(String, nullable=False)  # skill_up, attribute_change, milestone, story
    category = Column(String, nullable=False)

    # 记录内容
    title = Column(String, nullable=False)
    description = Column(Text)
    before_value = Column(JSON)  # 变化前的值
    after_value = Column(JSON)   # 变化后的值

    # 上下文
    scene_id = Column(String, ForeignKey("scenes.id"))
    session_id = Column(String)  # 游戏会话 ID
    related_event_id = Column(String)  # 相关事件

    # 元数据
    metadata = Column(JSON)
    created_at = Column(DateTime, default=func.now(), nullable=False)

class Milestone(Base):
    """里程碑"""
    __tablename__ = "milestones"

    id = Column(String, primary_key=True, index=True)
    character_id = Column(String, ForeignKey("characters.id"), nullable=False)
    campaign_id = Column(String, ForeignKey("campaigns.id"), nullable=False)

    # 里程碑信息
    title = Column(String, nullable=False)
    description = Column(Text)
    icon = Column(String)  # 图标

    # 类型和等级
    category = Column(String, nullable=False)  # combat, investigation, social, horror
    level = Column(String, default="minor")  # minor, major, epic

    # 时间戳
    achieved_at = Column(DateTime, default=func.now(), nullable=False)
```

---

## 成长记录服务

```python
# app/services/growth.py
from typing import List, Dict, Any, Optional
from sqlalchemy.orm import Session
from datetime import datetime

from app.db.models.growth import GrowthRecord, Milestone
from app.db.models.character import Character
from app.core.security import generate_id

class GrowthService:
    """成长记录服务"""

    def __init__(self, db: Session):
        self.db = db

    def record_skill_growth(
        self,
        character_id: str,
        skill_name: str,
        old_value: int,
        new_value: int,
        scene_id: Optional[str] = None,
        session_id: Optional[str] = None,
    ) -> GrowthRecord:
        """记录技能成长"""
        record = GrowthRecord(
            id=generate_id("growth"),
            character_id=character_id,
            campaign_id=self._get_campaign_id(character_id),
            record_type="skill_up",
            category="skill",
            title=f"{skill_name} 提升",
            description=f"{skill_name} 从 {old_value} 提升到 {new_value}",
            before_value={"skill": skill_name, "value": old_value},
            after_value={"skill": skill_name, "value": new_value},
            scene_id=scene_id,
            session_id=session_id,
        )

        self.db.add(record)
        self.db.commit()

        return record

    def record_attribute_change(
        self,
        character_id: str,
        attribute: str,
        old_value: int,
        new_value: int,
        reason: str,
        scene_id: Optional[str] = None,
    ) -> GrowthRecord:
        """记录属性变化"""
        # 属性通常是永久降低，如 SAN
        record = GrowthRecord(
            id=generate_id("growth"),
            character_id=character_id,
            campaign_id=self._get_campaign_id(character_id),
            record_type="attribute_change",
            category="attribute",
            title=f"{attribute} 变化",
            description=f"{attribute} 从 {old_value} 变为 {new_value}",
            before_value={"attribute": attribute, "value": old_value},
            after_value={"attribute": attribute, "value": new_value},
            scene_id=scene_id,
            metadata={"reason": reason},
        )

        self.db.add(record)
        self.db.commit()

        return record

    def record_milestone(
        self,
        character_id: str,
        title: str,
        description: str,
        category: str,
        level: str = "minor",
        icon: Optional[str] = None,
    ) -> Milestone:
        """记录里程碑"""
        milestone = Milestone(
            id=generate_id("milestone"),
            character_id=character_id,
            campaign_id=self._get_campaign_id(character_id),
            title=title,
            description=description,
            category=category,
            level=level,
            icon=icon,
        )

        self.db.add(milestone)
        self.db.commit()

        return milestone

    def get_growth_timeline(
        self,
        character_id: str,
        limit: int = 50,
    ) -> List[Dict[str, Any]]:
        """获取成长时间线"""
        records = self.db.query(GrowthRecord)\
            .filter(GrowthRecord.character_id == character_id)\
            .order_by(GrowthRecord.created_at.desc())\
            .limit(limit)\
            .all()

        return [
            {
                "id": r.id,
                "type": r.record_type,
                "title": r.title,
                "description": r.description,
                "before": r.before_value,
                "after": r.after_value,
                "timestamp": r.created_at.isoformat(),
            }
            for r in records
        ]

    def get_milestones(
        self,
        character_id: str,
    ) -> List[Dict[str, Any]]:
        """获取里程碑"""
        milestones = self.db.query(Milestone)\
            .filter(Milestone.character_id == character_id)\
            .order_by(Milestone.achieved_at.desc())\
            .all()

        return [
            {
                "id": m.id,
                "title": m.title,
                "description": m.description,
                "category": m.category,
                "level": m.level,
                "icon": m.icon,
                "achieved_at": m.achieved_at.isoformat(),
            }
            for m in milestones
        ]

    def get_growth_summary(
        self,
        character_id: str,
    ) -> Dict[str, Any]:
        """获取成长摘要"""
        # 技能成长统计
        skill_records = self.db.query(GrowthRecord)\
            .filter(
                GrowthRecord.character_id == character_id,
                GrowthRecord.record_type == "skill_up"
            )\
            .all()

        skills_improved = {}
        for record in skill_records:
            skill = record.after_value.get("skill")
            if skill:
                if skill not in skills_improved:
                    skills_improved[skill] = 0
                skills_improved[skill] += 1

        # 里程碑统计
        milestones = self.db.query(Milestone)\
            .filter(Milestone.character_id == character_id)\
            .all()

        milestone_counts = {}
        for m in milestones:
            if m.category not in milestone_counts:
                milestone_counts[m.category] = 0
            milestone_counts[m.category] += 1

        return {
            "total_records": self.db.query(GrowthRecord)
                .filter(GrowthRecord.character_id == character_id)
                .count(),
            "skills_improved": skills_improved,
            "milestone_counts": milestone_counts,
        }

    def _get_campaign_id(self, character_id: str) -> str:
        """获取角色所在的战役 ID"""
        character = self.db.query(Character)\
            .filter(Character.id == character_id)\
            .first()
        return character.campaign_id if character else ""
```

---

## 成长 API

```python
# app/api/growth.py
from fastapi import APIRouter, Depends, Query
from sqlalchemy.orm import Session
from pydantic import BaseModel

from app.db.database import get_db
from app.services.growth import GrowthService
from app.api.deps.auth import get_current_user
from app.db.models.user import User

router = APIRouter(prefix="/growth", tags=["growth"])

class GrowthTimelineResponse(BaseModel):
    records: list
    total: int

@router.get("/timeline/{character_id}")
async def get_growth_timeline(
    character_id: str,
    limit: int = Query(50, ge=1, le=100),
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取角色成长时间线"""
    service = GrowthService(db)

    records = service.get_growth_timeline(character_id, limit)

    return GrowthTimelineResponse(
        records=records,
        total=len(records),
    )

@router.get("/milestones/{character_id}")
async def get_milestones(
    character_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取角色里程碑"""
    service = GrowthService(db)

    milestones = service.get_milestones(character_id)

    return {"milestones": milestones}

@router.get("/summary/{character_id}")
async def get_growth_summary(
    character_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取角色成长摘要"""
    service = GrowthService(db)

    summary = service.get_growth_summary(character_id)

    return summary
```

---

## 自动追踪触发器

```python
# app/services/auto_growth.py
from typing import Dict, Any

class AutoGrowthTracker:
    """自动成长追踪器"""

    def __init__(self, growth_service: GrowthService):
        self.growth = growth_service

    async def on_check_success(
        self,
        character_id: str,
        skill_name: str,
        difficulty: int,
        rolled: int,
        scene_id: str = None,
    ):
        """检定成功后检查是否应该提升技能"""
        # 成功检定可能提升技能
        if rolled <= difficulty / 5:  # 极难成功
            character = self.growth.db.query(Character)\
                .filter(Character.id == character_id)\
                .first()

            if character:
                skill = next(
                    (s for s in character.skills if s.name == skill_name),
                    None
                )

                if skill:
                    old_value = skill.value
                    new_value = min(old_value + 1, 99)  # 最大 99

                    if new_value > old_value:
                        self.growth.record_skill_growth(
                            character_id=character_id,
                            skill_name=skill_name,
                            old_value=old_value,
                            new_value=new_value,
                            scene_id=scene_id,
                        )

    async def on_sanity_loss(
        self,
        character_id: str,
        loss: int,
        reason: str,
        scene_id: str = None,
    ):
        """SAN 损失时记录"""
        character = self.growth.db.query(Character)\
            .filter(Character.id == character_id)\
            .first()

        if character:
            old_san = character.status.get("san", 0)
            new_san = max(0, old_san - loss)

            if new_san != old_san:
                self.growth.record_attribute_change(
                    character_id=character_id,
                    attribute="san",
                    old_value=old_san,
                    new_value=new_san,
                    reason=reason,
                    scene_id=scene_id,
                )

                # 检查是否达到疯狂里程碑
                if new_san <= 0:
                    self.growth.record_milestone(
                        character_id=character_id,
                        title="陷入疯狂",
                        description=f"SAN 值降至 {new_san}",
                        category="horror",
                        level="major",
                    )
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/db/models/growth.py` | 创建 | 成长记录模型 |
| `app/services/growth.py` | 创建 | 成长记录服务 |
| `app/services/auto_growth.py` | 创建 | 自动追踪 |
| `app/api/growth.py` | 创建 | 成长 API |
| `tests/test_growth.py` | 创建 | 成长测试 |

---

## 验收标准

- [ ] 技能成长自动记录
- [ ] 属性变化正确追踪
- [ ] 里程碑触发准确
- [ ] 时间线显示完整
- [ ] 摘要统计正确
- [ ] API 响应及时

---

## 参考文档

- M1-003: 角色卡数据模型
- M1-040: SAN 值系统

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
