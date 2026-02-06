# M2-009: 实现房间模板系统

**任务ID**: M2-009
**标题**: 实现房间模板系统
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M2-001

---

## 任务描述

实现房间模板系统，允许用户保存房间配置为模板，快速创建预设好的游戏房间。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M2-009-01 | 设计模板数据结构 | Data Model | 20min |
| M2-009-02 | 实现模板服务 | Template Service | 30min |
| M2-009-03 | 实现模板创建 | Create Template | 25min |
| M2-009-04 | 实现模板应用 | Apply Template | 25min |
| M2-009-05 | 实现模板管理 | Management | 25min |
| M2-009-06 | 编写模板测试 | 测试覆盖 | 15min |

---

## 模板数据模型

```python
# app/db/models/room_template.py
from sqlalchemy import Column, String, Text, ForeignKey, Boolean, JSON
from sqlalchemy.orm import relationship
from app.db.database import Base

class RoomTemplate(Base):
    """房间模板"""
    __tablename__ = 'room_templates'

    id = Column(String, primary_key=True, index=True)
    user_id = Column(String, ForeignKey('users.id'), nullable=False, index=True)

    # 基本信息
    name = Column(String, nullable=False)
    description = Column(Text)
    category = Column(String)  # public, private
    icon = Column(String)  # emoji 或图标 URL

    # 模板配置
    config = Column(JSON, nullable=False)
    rules = Column(JSON)  # 房间规则配置
    default_settings = Column(JSON)  # 默认设置

    # 场景关联（可选）
    scene_package_id = Column(String, ForeignKey('scene_packages.id'))

    # 使用统计
    usage_count = Column(Integer, default=0)

    # 是否公开
    is_public = Column(Boolean, default=False)

    # 创建者
    created_by = Column(String, ForeignKey('users.id'), nullable=False)

    # 关系
    user = relationship("User", back_populates="room_templates")
    scene_package = relationship("ScenePackage", back_populates="room_templates")

    def __repr__(self):
        return f"<RoomTemplate {self.name}>"
```

---

## 模板配置结构

```python
# 模板配置示例
TEMPLATE_CONFIG_EXAMPLE = {
    "name": "标准调查",
    "description": "包含基本调查工具的房间配置",
    "config": {
        "max_players": 6,
        "allow_spectators": True,
        "features": {
            "dice_roller": True,
            "character_sheet": True,
            "combat_tracker": True,
            "initiative_tracker": True,
            "whiteboard": False,
            "voice_chat": True,
            "video_share": False,
        },
        "permissions": {
            "players_can_post": True,
            "players_can_roll": True,
            "players_can_edit_characters": True,
        },
        "default_settings": {
            "difficulty": "normal",
            "auto_save_interval": 60,
            "show_dice_animations": True,
        },
    },
    "rules": {
        "allowed_commands": ["roll", "check", "attack", "damage", "heal"],
        "restricted_commands": [],
        "custom_commands": [],
    }
}
```

---

## 模板服务

```python
# app/services/room_template.py
from typing import List, Dict, Any, Optional
from sqlalchemy.orm import Session

from app.db.models.room_template import RoomTemplate
from app.core.security import generate_id

class RoomTemplateService:
    """房间模板服务"""

    def __init__(self, db: Session):
        self.db = db

    def create_template(
        self,
        user_id: str,
        name: str,
        config: Dict[str, Any],
        description: str = None,
        category: str = 'private',
        icon: str = '🎲',
        scene_package_id: str = None,
        rules: Dict = None,
        default_settings: Dict = None,
        is_public: bool = False,
    ) -> RoomTemplate:
        """创建模板"""
        template = RoomTemplate(
            id=generate_id('template'),
            user_id=user_id,
            name=name,
            description=description,
            category=category,
            icon=icon,
            config=config,
            rules=rules or {},
            default_settings=default_settings or {},
            scene_package_id=scene_package_id,
            is_public=is_public,
            created_by=user_id,
        )

        self.db.add(template)
        self.db.commit()
        self.db.refresh(template)

        return template

    def get_user_templates(
        self,
        user_id: str,
        category: str = None,
    ) -> List[RoomTemplate]:
        """获取用户模板"""
        query = self.db.query(RoomTemplate)\
            .filter(RoomTemplate.user_id == user_id)

        if category:
            query = query.filter(RoomTemplate.category == category)

        return query\
            .order_by(RoomTemplate.usage_count.desc(), RoomTemplate.created_at.desc())\
            .all()

    def get_public_templates(
        self,
        limit: int = 50,
    ) -> List[RoomTemplate]:
        """获取公开模板"""
        return self.db.query(RoomTemplate)\
            .filter(RoomTemplate.is_public == True)\
            .order_by(RoomTemplate.usage_count.desc())\
            .limit(limit)\
            .all()

    def get_template(self, template_id: str) -> Optional[RoomTemplate]:
        """获取模板详情"""
        return self.db.query(RoomTemplate)\
            .filter(RoomTemplate.id == template_id)\
            .first()

    def apply_template(
        self,
        template_id: str,
        room_id: str,
        user_id: str,
    ) -> Dict[str, Any]:
        """应用模板到房间"""
        template = self.get_template(template_id)

        if not template:
            raise ValueError("模板不存在")

        # 验证权限
        if template.user_id != user_id and not template.is_public:
            raise ValueError("无权使用此模板")

        # 应用模板配置
        from app.services.room import RoomService
        room_service = RoomService(self.db)

        # 更新房间配置
        room = room_service.get_room(room_id)
        if not room:
            raise ValueError("房间不存在")

        room.config = {**room.config, **template.config.get('config', {})}
        room.rules = template.rules
        room.settings = {**room.settings, **template.config.get('default_settings', {})}

        # 如果关联了场景包，加载场景
        if template.scene_package_id:
            # TODO: 加载场景包到房间
            pass

        # 更新使用计数
        template.usage_count += 1

        self.db.commit()

        return {
            "room_id": room_id,
            "template_id": template_id,
            "applied_settings": room.config,
        }

    def update_template(
        self,
        template_id: str,
        user_id: str,
        updates: Dict[str, Any],
    ) -> Optional[RoomTemplate]:
        """更新模板"""
        template = self.db.query(RoomTemplate)\
            .filter(
                RoomTemplate.id == template_id,
                RoomTemplate.user_id == user_id,
            )\
            .first()

        if not template:
            return None

        for key, value in updates.items():
            if hasattr(template, key):
                setattr(template, key, value)

        self.db.commit()
        self.db.refresh(template)

        return template

    def delete_template(
        self,
        template_id: str,
        user_id: str,
    ) -> bool:
        """删除模板"""
        template = self.db.query(RoomTemplate)\
            .filter(
                RoomTemplate.id == template_id,
                RoomTemplate.user_id == user_id,
            )\
            .first()

        if not template:
            return False

        self.db.delete(template)
        self.db.commit()

        return True

    def clone_template(
        self,
        template_id: str,
        user_id: str,
        new_name: str,
    ) -> RoomTemplate:
        """克隆模板"""
        original = self.get_template(template_id)

        if not original:
            raise ValueError("模板不存在")

        # 克隆模板
        return self.create_template(
            user_id=user_id,
            name=new_name,
            config=original.config,
            description=f"克隆自: {original.name}",
            icon=original.icon,
            rules=original.rules,
            default_settings=original.default_settings,
            scene_package_id=original.scene_package_id,
        )
```

---

## 模板 API

```python
# app/api/templates.py
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session
from pydantic import BaseModel
from typing import List, Optional

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.room_template import RoomTemplateService

router = APIRouter(prefix="/templates", tags=["templates"])

class CreateTemplateRequest(BaseModel):
    name: str
    description: str = None
    category: str = "private"
    icon: str = "🎲"
    config: dict
    rules: dict = None
    default_settings: dict = None
    scene_package_id: str = None

@router.post("")
async def create_template(
    request: CreateTemplateRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """创建模板"""
    service = RoomTemplateService(db)

    template = service.create_template(
        user_id=current_user.id,
        name=request.name,
        config=request.config,
        description=request.description,
        category=request.category,
        icon=request.icon,
        rules=request.rules,
        default_settings=request.default_settings,
        scene_package_id=request.scene_package_id,
    )

    return {"id": template.id, "name": template.name}

@router.get("")
async def list_templates(
    category: Optional[str] = None,
    include_public: bool = True,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取模板列表"""
    service = RoomTemplateService(db)

    templates = service.get_user_templates(current_user.id, category)

    if include_public:
        public_templates = service.get_public_templates()
        # 去重（用户自己的模板已经在列表中）
        public_templates = [
            t for t in public_templates
            if t.user_id != current_user.id
        ]
        templates = templates + public_templates

    return [
        {
            "id": t.id,
            "name": t.name,
            "description": t.description,
            "category": t.category,
            "icon": t.icon,
            "is_public": t.is_public,
            "usage_count": t.usage_count,
            "created_by": t.creator.username,
            "created_at": t.created_at.isoformat(),
        }
        for t in templates
    ]

@router.get("/{template_id}")
async def get_template(
    template_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取模板详情"""
    service = RoomTemplateService(db)
    template = service.get_template(template_id)

    if not template:
        raise HTTPException(status_code=404, detail="模板不存在")

    # 权限检查
    if template.user_id != current_user.id and not template.is_public:
        raise HTTPException(status_code=403, detail="无权访问此模板")

    return {
        "id": template.id,
        "name": template.name,
        "description": template.description,
        "category": template.category,
        "icon": template.icon,
        "config": template.config,
        "rules": template.rules,
        "default_settings": template.default_settings,
        "scene_package_id": template.scene_package_id,
        "is_public": template.is_public,
        "usage_count": template.usage_count,
    }

@router.post("/{template_id}/apply")
async def apply_template(
    template_id: str,
    room_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """应用模板到房间"""
    service = RoomTemplateService(db)

    try:
        result = service.apply_template(template_id, room_id, current_user.id)

        # 通知房间成员
        # TODO: WebSocket 事件

        return result
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))

@router.post("/{template_id}/clone")
async def clone_template(
    template_id: str,
    new_name: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """克隆模板"""
    service = RoomTemplateService(db)

    try:
        template = service.clone_template(template_id, current_user.id, new_name)
        return {"id": template.id, "name": template.name}
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))

@router.put("/{template_id}")
async def update_template(
    template_id: str,
    updates: dict,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """更新模板"""
    service = RoomTemplateService(db)
    template = service.update_template(template_id, current_user.id, updates)

    if not template:
        raise HTTPException(status_code=404, detail="模板不存在或无权修改")

    return {"message": "模板已更新"}

@router.delete("/{template_id}")
async def delete_template(
    template_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """删除模板"""
    service = RoomTemplateService(db)
    success = service.delete_template(template_id, current_user.id)

    if not success:
        raise HTTPException(status_code=404, detail="模板不存在或无权删除")

    return {"message": "模板已删除"}
```

---

## 预设模板

```python
# app/services/room_template/presets.py
PRESET_TEMPLATES = [
    {
        "id": "preset_investigation",
        "name": "经典调查",
        "description": "适合传统调查型跑团",
        "icon": "🔍",
        "config": {
            "max_players": 6,
            "features": {
                "dice_roller": True,
                "character_sheet": True,
                "combat_tracker": True,
                "initiative_tracker": True,
                "whiteboard": True,
                "voice_chat": True,
            },
        },
    },
    {
        "id": "preset_combat",
        "name": "战斗",
        "description": "注重战斗的房间",
        "icon": "⚔️",
        "config": {
            "max_players": 8,
            "features": {
                "dice_roller": True,
                "character_sheet": True,
                "combat_tracker": True,
                "initiative_tracker": True,
                "whiteboard": True,
                "voice_chat": True,
            },
        },
    },
    {
        "id": "preset_roleplay",
        "name": "角色扮演",
        "description": "注重剧情和角色扮演",
        "icon": "🎭",
        "config": {
            "max_players": 4,
            "features": {
                "dice_roller": True,
                "character_sheet": True,
                "combat_tracker": False,
                "whiteboard": True,
                "voice_chat": True,
            },
        },
    },
]
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/db/models/room_template.py` | 创建 | 模板数据模型 |
| `app/services/room_template.py` | 创建 | 模板服务 |
| `app/services/room_template/presets.py` | 创建 | 预设模板 |
| `app/api/templates.py` | 创建 | 模板 API |
| `frontend/src/components/game/TemplateManager.tsx` | 创建 | 模板管理组件 |

---

## 验收标准

- [ ] 模板创建成功
- [ ] 模板应用正确
- [ ] 配置导入完整
- [ ] 权限控制有效
- [ ] 公开模板可用
- [ ] 克隆功能正常

---

## 参考文档

- M2-001: 房间管理系统

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
