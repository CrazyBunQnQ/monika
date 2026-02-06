# M1-003: 角色卡数据模型

**任务类型**: backend
**预估工时**: 3h
**依赖**: M1-001
**状态**: [ ]

---

## 子任务拆解

### 1.1 角色卡服务层 (40min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-003-01 | [ ] 创建 `app/services/character.py` | [ ] |
| M1-003-02 | [ ] 实现 `create_character()` | [ ] |
| M1-003-03 | [ ] 实现 `get_character()` | [ ] |
| M1-003-04 | [ ] 实现 `update_character()` | [ ] |
| M1-003-05 | [ ] 实现 `delete_character()` | [ ] |

```python
# app/services/character.py
from typing import Optional, List
from sqlmodel import Session, select
from app.db.models.character import (
    Character, CharacterSkill, CharacterAttribute,
    CharacterStatus
)
from app.services.base import BaseService

class CharacterService(BaseService):
    """角色卡服务"""

    def create_character(
        self,
        owner_id: int,
        name: str,
        **kwargs
    ) -> Character:
        """创建角色卡"""
        character = Character(
            owner_id=owner_id,
            name=name,
            **kwargs
        )
        self.session.add(character)
        self.session.commit()
        self.session.refresh(character)
        return character

    def get_character(
        self,
        character_id: int,
        include_skills: bool = True,
        include_attributes: bool = True
    ) -> Optional[Character]:
        """获取角色卡"""
        statement = select(Character).where(Character.id == character_id)

        if include_skills:
            statement = statement.where(
                Character.skills  # 加载关系
            )

        result = self.session.exec(statement).first()
        return result

    def get_user_characters(
        self,
        user_id: int,
        campaign_id: Optional[int] = None,
        status: Optional[CharacterStatus] = None
    ) -> List[Character]:
        """获取用户的角色卡列表"""
        statement = select(Character).where(Character.owner_id == user_id)

        if campaign_id is not None:
            statement = statement.where(Character.campaign_id == campaign_id)

        if status is not None:
            statement = statement.where(Character.status == status)

        return self.session.exec(statement).all()

    def update_character(
        self,
        character_id: int,
        **updates
    ) -> Optional[Character]:
        """更新角色卡"""
        character = self.session.get(Character, character_id)
        if not character:
            return None

        for key, value in updates.items():
            if hasattr(character, key):
                setattr(character, key, value)

        self.session.commit()
        self.session.refresh(character)
        return character

    def update_attribute(
        self,
        character_id: int,
        attr_name: str,
        value: int
    ) -> Optional[Character]:
        """更新角色属性"""
        character = self.session.get(Character, character_id)
        if not character:
            return None

        # 验证属性名
        valid_attrs = ['str', 'con', 'pow', 'dex', 'app', 'san', 'edu', 'luck', 'hp', 'mp']
        if attr_name not in valid_attrs:
            raise ValueError(f"无效的属性名: {attr_name}")

        # 检查范围
        if attr_name in ['str', 'con', 'pow', 'dex', 'app', 'san', 'edu', 'luck']:
            if not 0 <= value <= 100:
                raise ValueError(f"{attr_name} 必须在 0-100 之间")

        setattr(character, attr_name, value)
        self.session.commit()

        # 更新派生值
        self._recalculate_derived_values(character)

        self.session.refresh(character)
        return character

    def _recalculate_derived_values(self, character: Character):
        """重新计算派生值"""
        # HP = (STR + CON) / 2
        character.hp = (character.str + character.con) // 2

        # MP = POW / 2
        character.mp = character.pow // 2

    def take_damage(
        self,
        character_id: int,
        damage: int,
        damage_type: str = "hp"
    ) -> tuple[bool, int, int]:
        """造成伤害"""
        character = self.session.get(Character, character_id)
        if not character:
            return False, 0, 0

        if damage_type == "hp":
            character.hp_current = max(0, character.hp_current - damage)
            remaining = character.hp_current

            # 检查死亡
            if character.hp_current <= 0:
                character.status = CharacterStatus.DEAD

        elif damage_type == "san":
            character.san_current = max(0, character.san_current - damage)
            remaining = character.san_current

            # 检查疯狂
            if character.san_current <= 0:
                character.status = CharacterStatus.INSANE

        self.session.commit()
        return True, damage, remaining

    def heal(
        self,
        character_id: int,
        amount: int,
        heal_type: str = "hp"
    ) -> tuple[int, int]:
        """治疗"""
        character = self.session.get(Character, character_id)
        if not character:
            return 0, 0

        if heal_type == "hp":
            actual = min(amount, character.hp - character.hp_current)
            character.hp_current += actual
        elif heal_type == "san":
            actual = min(amount, character.san - character.san_current)
            character.san_current += actual

        self.session.commit()
        return actual, character.hp_current if heal_type == "hp" else character.san_current
```

---

### 1.2 技能服务 (35min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-003-06 | [ ] 创建 `app/services/skill.py` | [ ] |
| M1-003-07 | [ ] 实现 `add_skill()` | [ ] |
| M1-003-08 | [ ] 实现 `update_skill()` | [ ] |
| M1-003-09 | [ ] 实现 `calculate_skill_total()` | [ ] |
| M1-003-10 | [ ] 实现 `get_skill_check()` | [ ] |

```python
# app/services/skill.py
from typing import Optional, List, Dict
from sqlmodel import Session, select
from app.db.models.character import Character, CharacterSkill

class SkillService:
    """技能服务"""

    # CoC 7e 标准技能
    DEFAULT_SKILLS = [
        ("accounting", "会计", 10),
        ("anthropology", "人类学", 1),
        ("archeology", "考古学", 1),
        ("art_craft", "艺术/手工艺", 5),
        ("astronomy", "天文学", 1),
        ("bargain", "议价", 5),
        ("biology", "生物学", 1),
        ("chemistry", "化学", 1),
        ("climb", "攀爬", 20),
        ("computer_use", "计算机使用", 5),
        ("cthulu_mythos", "克苏鲁神话", 0),
        ("disguise", "伪装", 5),
        ("dodge", "闪避", 1),
        ("drive_auto", "驾驶", 20),
        ("electric_repair", "电气维修", 10),
        ("fast_talk", "快速交谈", 5),
        ("first_aid", "急救", 30),
        ("geology", "地质学", 1),
        ("history", "历史", 5),
        ("jump", "跳跃", 20),
        ("law", "法律", 5),
        ("library_use", "图书馆使用", 25),
        ("listen", "聆听", 20),
        ("locksmith", "开锁", 1),
        ("mechanical_repair", "机械维修", 10),
        ("medicine", "医学", 1),
        ("natural_history", "博物学", 10),
        ("navigate", "导航", 10),
        ("occult", "神秘学", 5),
        ("operate_heavy_machine", "操作重型机械", 1),
        ("persuade", "说服", 15),
        ("pilot", "驾驶 (船舶/飞机)", 1),
        ("psychology", "心理学", 10),
        ("psychoanalysis", "心理分析", 1),
        ("read_lips", "读唇语", 1),
        ("ride", "骑术", 5),
        ("science", "科学", 30),
        ("sleight_of_hand", "手法", 10),
        ("spot_hidden", "侦查", 25),
        ("stealth", "潜行", 20),
        ("survival", "生存", 15),
        ("swim", "游泳", 20),
        ("throw", "投掷", 20),
        ("track", "追踪", 10),
    ]

    def add_default_skills(self, character_id: int, session: Session):
        """为角色添加默认技能"""
        for skill_key, skill_name, base_value in self.DEFAULT_SKILLS:
            skill = CharacterSkill(
                character_id=character_id,
                skill_key=skill_key,
                skill_name=skill_name,
                base_value=base_value
            )
            session.add(skill)
        session.commit()

    def add_skill(
        self,
        character_id: int,
        skill_key: str,
        skill_name: str,
        base_value: int = 0
    ) -> CharacterSkill:
        """添加技能"""
        skill = CharacterSkill(
            character_id=character_id,
            skill_key=skill_key,
            skill_name=skill_name,
            base_value=base_value
        )
        self.session.add(skill)
        self.session.commit()
        return skill

    def update_skill(
        self,
        skill_id: int,
        **updates
    ) -> Optional[CharacterSkill]:
        """更新技能"""
        skill = self.session.get(CharacterSkill, skill_id)
        if not skill:
            return None

        for key, value in updates.items():
            if hasattr(skill, key):
                setattr(skill, key, value)

        self.session.commit()
        self.session.refresh(skill)
        return skill

    def calculate_total(
        self,
        skill: CharacterSkill
    ) -> int:
        """计算技能总值"""
        return (
            skill.base_value +
            skill.occupation_bonus +
            skill.personal_interest_bonus +
            skill.growth_bonus
        )

    def get_skill_check(
        self,
        character: Character,
        skill_key: str
    ) -> Optional[Dict]:
        """获取技能检定信息"""
        statement = select(CharacterSkill).where(
            CharacterSkill.character_id == character.id,
            CharacterSkill.skill_key == skill_key
        )
        skill = self.session.exec(statement).first()

        if not skill:
            return None

        total = self.calculate_total(skill)

        return {
            "skill_key": skill_key,
            "skill_name": skill.skill_name,
            "base_value": skill.base_value,
            "occupation_bonus": skill.occupation_bonus,
            "personal_interest_bonus": skill.personal_interest_bonus,
            "growth_bonus": skill.growth_bonus,
            "total": total,
            "half": total // 2,
            "fifth": max(1, total // 5),
        }
```

---

### 1.3 角色卡 API (40min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-003-11 | [ ] 创建 `app/api/character.py` | [ ] |
| M1-003-12 | [ ] 实现 CRUD 端点 | [ ] |
| M1-003-13 | [ ] 实现属性修改端点 | [ ] |
| M1-003-14 | [ ] 实现伤害/治疗端点 | [ ] |

```python
# app/api/character.py
from typing import Optional, List
from fastapi import APIRouter, Depends, HTTPException, status
from sqlmodel import Session
from app.db.user import User
from app.api.deps.auth import get_current_active_user
from app.db.connection import get_session
from app.services.character import CharacterService
from app.services.skill import SkillService
from app.db.models.character import Character, CharacterStatus

router = APIRouter(prefix="/characters", tags=["角色卡"])

@router.post("/", response_model=Character, status_code=status.HTTP_201_CREATED)
async def create_character(
    character_data: dict,
    current_user: User = Depends(get_current_active_user),
    session: Session = Depends(get_session)
):
    """创建角色卡"""
    service = CharacterService(session)
    character = service.create_character(
        owner_id=current_user.id,
        **character_data
    )
    return character

@router.get("/", response_model=List[Character])
async def list_characters(
    campaign_id: Optional[int] = None,
    status: Optional[CharacterStatus] = None,
    current_user: User = Depends(get_current_active_user),
    session: Session = Depends(get_session)
):
    """获取角色卡列表"""
    service = CharacterService(session)
    return service.get_user_characters(
        user_id=current_user.id,
        campaign_id=campaign_id,
        status=status
    )

@router.get("/{character_id}", response_model=dict)
async def get_character(
    character_id: int,
    include_skills: bool = True,
    current_user: User = Depends(get_current_active_user),
    session: Session = Depends(get_session)
):
    """获取角色卡详情"""
    service = CharacterService(session)
    character = service.get_character(
        character_id=character_id,
        include_skills=include_skills
    )

    if not character:
        raise HTTPException(status_code=404, detail="角色卡不存在")

    # 验证所有权
    if character.owner_id != current_user.id:
        raise HTTPException(status_code=403, detail="无权访问此角色卡")

    return character

@router.patch("/{character_id}/attribute")
async def update_attribute(
    character_id: int,
    attr_name: str,
    value: int,
    current_user: User = Depends(get_current_active_user),
    session: Session = Depends(get_session)
):
    """更新角色属性"""
    service = CharacterService(session)

    character = service.get_character(character_id)
    if not character:
        raise HTTPException(status_code=404, detail="角色卡不存在")

    if character.owner_id != current_user.id:
        raise HTTPException(status_code=403, detail="无权修改此角色卡")

    try:
        updated = service.update_attribute(character_id, attr_name, value)
        return {"success": True, "character": updated}
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))

@router.post("/{character_id}/damage")
async def take_damage(
    character_id: int,
    damage: int,
    damage_type: str = "hp",
    current_user: User = Depends(get_current_active_user),
    session: Session = Depends(get_session)
):
    """造成伤害"""
    service = CharacterService(session)

    character = service.get_character(character_id)
    if not character:
        raise HTTPException(status_code=404, detail="角色卡不存在")

    success, dmg, remaining = service.take_damage(
        character_id, damage, damage_type
    )

    return {
        "success": success,
        "damage": dmg,
        "remaining": remaining,
        "status": character.status.value
    }
```

---

### 1.4 Pydantic Schemas (25min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-003-15 | [ ] 创建 `app/schemas/character.py` | [ ] |
| M1-003-16 | [ ] 定义 `CharacterCreate` | [ ] |
| M1-003-17 | [ ] 定义 `CharacterUpdate` | [ ] |
| M1-003-18 | [ ] 定义 `CharacterResponse` | [ ] |

```python
# app/schemas/character.py
from datetime import datetime
from typing import Optional, List
from pydantic import BaseModel, Field
from app.db.models.character import CharacterStatus, Gender

class CharacterCreate(BaseModel):
    """角色卡创建"""
    name: str = Field(..., max_length=100)
    age: Optional[int] = None
    gender: Gender = Gender.UNKNOWN
    occupation: Optional[str] = Field(max_length=100)
    residence: Optional[str] = Field(max_length=200)
    backstory: Optional[str] = Field(max_length=5000)

    # 初始属性
    str: int = Field(default=50, ge=0, le=100)
    con: int = Field(default=50, ge=0, le=100)
    pow: int = Field(default=50, ge=0, le=100)
    dex: int = Field(default=50, ge=0, le=100)
    app: int = Field(default=50, ge=0, le=100)
    edu: int = Field(default=50, ge=0, le=100)

    # 幸运初始值
    luck: int = Field(default=50, ge=0, le=100)

class CharacterUpdate(BaseModel):
    """角色卡更新"""
    name: Optional[str] = Field(max_length=100)
    age: Optional[int] = None
    gender: Optional[Gender] = None
    occupation: Optional[str] = Field(max_length=100)
    residence: Optional[str] = Field(max_length=200)
    backstory: Optional[str] = Field(max_length=5000)
    status: Optional[CharacterStatus] = None

class SkillResponse(BaseModel):
    """技能响应"""
    skill_key: str
    skill_name: str
    base_value: int
    occupation_bonus: int
    personal_interest_bonus: int
    growth_bonus: int
    total: int

class CharacterResponse(BaseModel):
    """角色卡响应"""
    id: int
    name: str
    age: Optional[int]
    gender: Gender
    occupation: Optional[str]
    residence: Optional[str]

    # 属性
    str: int
    con: int
    pow: int
    dex: int
    app: int
    san: int
    edu: int
    luck: int
    hp: int
    mp: int

    # 状态
    status: CharacterStatus
    hp_current: int
    mp_current: int
    san_current: int

    # 技能
    skills: List[SkillResponse] = []

    # 时间
    created_at: datetime
    updated_at: datetime

    class Config:
        from_attributes = True
```

---

### 1.5 单元测试 (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-003-19 | [ ] 创建 `tests/test_character.py` | [ ] |
| M1-003-20 | [ ] 测试角色卡 CRUD | [ ] |
| M1-003-21 | [ ] 测试属性计算 | [ ] |
| M1-003-22 | [ ] 测试伤害/治疗 | [ ] |

```python
# tests/test_character.py
import pytest
from app.services.character import CharacterService
from app.db.models.character import Character, CharacterStatus

class TestCharacterService:
    def test_create_character(self, session):
        """测试创建角色"""
        service = CharacterService(session)

        character = service.create_character(
            owner_id=1,
            name="测试角色",
            str=60,
            con=50,
            pow=70
        )

        assert character.name == "测试角色"
        assert character.str == 60
        assert character.status == CharacterStatus.ACTIVE

    def test_update_attribute(self, session):
        """测试更新属性"""
        service = CharacterService(session)

        character = service.create_character(
            owner_id=1,
            name="测试角色"
        )

        updated = service.update_attribute(character.id, "str", 80)
        assert updated.str == 80

    def test_take_damage(self, session):
        """测试造成伤害"""
        service = CharacterService(session)

        character = service.create_character(
            owner_id=1,
            name="测试角色",
            hp=20,
            hp_current=20,
            san=50,
            san_current=50
        )

        success, dmg, remaining = service.take_damage(character.id, 5, "hp")
        assert success is True
        assert dmg == 5
        assert remaining == 15

    def test_san_zero_triggers_insanity(self, session):
        """测试 SAN 归零触发疯狂"""
        service = CharacterService(session)

        character = service.create_character(
            owner_id=1,
            name="测试角色",
            san_current=3,
            status=CharacterStatus.ACTIVE
        )

        service.take_damage(character.id, 5, "san")
        assert character.status == CharacterStatus.INSANE
```

---

## 验收标准

- [ ] 角色卡 CRUD API 完整
- [ ] 属性更新包含范围验证
- [ ] HP/SAN 伤害逻辑正确
- [ ] 技能系统支持 CoC 7e 标准
- [ ] 单元测试覆盖率 > 90%

---

## 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `app/services/character.py` | 创建 | 角色卡服务 |
| `app/services/skill.py` | 创建 | 技能服务 |
| `app/api/character.py` | 创建 | 角色卡 API |
| `app/schemas/character.py` | 创建 | Pydantic 模式 |
| `tests/test_character.py` | 创建 | 单元测试 |
