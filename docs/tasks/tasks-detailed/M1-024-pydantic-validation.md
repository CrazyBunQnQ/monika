# M1-024 实现角色卡字段验证 (Pydantic)

## 概述
使用 Pydantic 实现角色卡字段的验证逻辑,确保数据完整性和正确性。

## 验收标准
- [ ] 定义角色卡 Pydantic 模型
- [ ] 实现属性值范围验证
- [ ] 实现技能值验证
- [ ] 实现派生值计算验证
- [ ] 实现自定义验证器
- [ ] 提供清晰的错误信息

## 技术方案

### 基础模型

```python
from pydantic import BaseModel, Field, validator, field_validator
from typing import Optional, Dict, List
from datetime import datetime
from enum import Enum

class CharacterStatus(str, Enum):
    ALIVE = "alive"
    UNCONSCIOUS = "unconscious"
    DYING = "dying"
    DEAD = "dead"
    INSANE = "insane"

class Attributes(BaseModel):
    """属性模型"""
    STR: int = Field(..., ge=0, le=100, description="力量")
    CON: int = Field(..., ge=0, le=100, description="体质")
    DEX: int = Field(..., ge=0, le=100, description="敏捷")
    APP: int = Field(..., ge=0, le=100, description="外貌")
    POW: int = Field(..., ge=0, le=100, description="意志")
    INT: int = Field(..., ge=0, le=100, description="智力")
    SIZ: int = Field(..., ge=0, le=100, description="体型")
    EDU: int = Field(..., ge=0, le=100, description="教育")

    @field_validator('*')
    @classmethod
    def validate_ranges(cls, v: int, info) -> int:
        """验证属性值在合理范围内"""
        if not (0 <= v <= 100):
            raise ValueError(f"{info.field_name} 必须在 0-100 之间")
        return v

class DerivedStats(BaseModel):
    """派生属性"""
    HP: int = Field(..., ge=1, le=200, description="耐久")
    HP_max: int = Field(..., ge=1, le=200, description="最大耐久")
    MP: int = Field(..., ge=0, le=99, description="魔法值")
    MP_max: int = Field(..., ge=0, le=99, description="最大魔法值")
    SAN: int = Field(..., ge=0, le=99, description="理智")
    SAN_max: int = Field(..., ge=0, le=99, description="最大理智")
    Luck: int = Field(..., ge=0, le=99, description="幸运")
    Luck_max: int = Field(..., ge=0, le=99, description="最大幸运")
    Move: int = Field(..., ge=1, le=10, description="移动速率")
    DB: str = Field(..., description="伤害加值")
    Build: int = Field(..., ge=-2, le=5, description="体格")

    @field_validator('DB')
    @classmethod
    def validate_db(cls, v: str) -> str:
        """验证伤害加值格式"""
        valid_formats = [
            "-2", "-1", "0", "+1d4", "+1d6",
            "+2d6", "+3d6", "+4d6", "+5d6"
        ]
        if v not in valid_formats:
            raise ValueError(f"无效的 DB 格式: {v}")
        return v

    @field_validator('HP')
    @classmethod
    def validate_hp_not_exceed_max(cls, v: int, info) -> int:
        """HP 不能超过最大值"""
        if 'HP_max' in info.data and v > info.data['HP_max']:
            raise ValueError("HP 不能超过 HP_max")
        return v

class Skill(BaseModel):
    """技能"""
    name: str = Field(..., min_length=1, max_length=50)
    value: int = Field(..., ge=0, le=100, description="技能值")
    category: Optional[str] = Field(None, description="技能分类")
    uses: Optional[int] = Field(None, ge=0, description="使用次数")

class CharacterCreate(BaseModel):
    """角色卡创建模型"""
    name: str = Field(..., min_length=1, max_length=100, description="角色名称")
    age: int = Field(..., ge=15, le=90, description="年龄")
    occupation: str = Field(..., min_length=1, max_length=100, description="职业")
    player: str = Field(..., min_length=1, max_length=100, description="玩家名")

    attributes: Attributes
    derived: DerivedStats
    skills: Dict[str, int] = Field(default_factory=dict)

    status: CharacterStatus = CharacterStatus.ALIVE
    inventory: List[str] = Field(default_factory=list)
    notes: Optional[str] = Field(None, max_length=5000)

    # 计算字段
    @field_validator('derived')
    @classmethod
    def validate_derived_calculation(cls, v: DerivedStats, info) -> DerivedStats:
        """验证派生属性计算正确"""
        if 'attributes' in info.data:
            attrs = info.data['attributes']

            # HP 计算
            expected_hp = (attrs.CON + attrs.SIZ) // 10
            if v.HP != expected_hp:
                raise ValueError(
                    f"HP 计算错误: 期望 {expected_hp}, 实际 {v.HP}"
                )

            # MP 计算
            expected_mp = attrs.POW // 5
            if v.MP != expected_mp:
                raise ValueError(
                    f"MP 计算错误: 期望 {expected_mp}, 实际 {v.MP}"
                )

            # Luck 计算
            expected_luck = attrs.POW * 5
            if v.Luck != expected_luck:
                raise ValueError(
                    f"Luck 计算错误: 期望 {expected_luck}, 实际 {v.Luck}"
                )

            # Move 计算
            expected_move = 8 if attrs.DEX >= attrs.SIZ or attrs.STR >= attrs.SIZ else 7
            if v.Move != expected_move:
                raise ValueError(
                    f"Move 计算错误: 期望 {expected_move}, 实际 {v.Move}"
                )

        return v

    @field_validator('skills')
    @classmethod
    def validate_skills(cls, v: Dict[str, int]) -> Dict[str, int]:
        """验证技能值"""
        # 常用技能列表
        common_skills = [
            "library_use", "hide", "psychology",
            "spot_hidden", "listen", "first_aid"
        ]

        for skill_name, skill_value in v.items():
            if skill_value < 0 or skill_value > 100:
                raise ValueError(
                    f"技能 {skill_name} 的值必须在 0-100 之间"
                )

            # 检查技能名称
            if not skill_name.replace('_', '').isalnum():
                raise ValueError(
                    f"技能名称 {skill_name} 格式无效"
                )

        return v

    @field_validator('inventory')
    @classmethod
    def validate_inventory(cls, v: List[str]) -> List[str]:
        """验证物品列表"""
        if len(v) > 100:
            raise ValueError("物品数量不能超过 100")
        return v
```

### 更新模型

```python
class CharacterUpdate(BaseModel):
    """角色卡更新模型"""
    name: Optional[str] = Field(None, min_length=1, max_length=100)
    age: Optional[int] = Field(None, ge=15, le=90)
    occupation: Optional[str] = Field(None, min_length=1, max_length=100)
    player: Optional[str] = Field(None, min_length=1, max_length=100)

    attributes: Optional[Attributes] = None
    derived: Optional[DerivedStats] = None
    skills: Optional[Dict[str, int]] = None

    status: Optional[CharacterStatus] = None
    inventory: Optional[List[str]] = None
    notes: Optional[str] = Field(None, max_length=5000)

    class Config:
        # 允许部分更新
        extra = 'forbid'
```

### 响应模型

```python
class CharacterResponse(BaseModel):
    """角色卡响应模型"""
    id: str
    name: str
    age: int
    occupation: str
    player: str

    attributes: Attributes
    derived: DerivedStats
    skills: Dict[str, int]

    status: CharacterStatus
    inventory: List[str]
    notes: Optional[str]

    created_at: datetime
    updated_at: datetime

    class Config:
        from_attributes = True

class CharacterListResponse(BaseModel):
    """角色卡列表响应"""
    characters: List[CharacterResponse]
    total: int
    page: int
    limit: int
```

### 自定义验证器

```python
from pydantic import field_validator

class CustomValidators:
    """自定义验证器集合"""

    @staticmethod
    def validate_age_modifiers(
        age: int,
        attributes: Attributes
    ) -> Attributes:
        """应用年龄修正"""
        modified = attributes.model_copy()

        if age >= 40:
            modified.STR = max(0, modified.STR - 5)
            modified.CON = max(0, modified.CON - 5)
            modified.DEX = max(0, modified.DEX - 5)

        if age >= 50:
            modified.STR = max(0, modified.STR - 10)
            modified.CON = max(0, modified.CON - 10)
            modified.DEX = max(0, modified.DEX - 10)

        if age >= 70:
            modified.STR = max(0, modified.STR - 20)
            modified.CON = max(0, modified.CON - 20)
            modified.DEX = max(0, modified.DEX - 20)

        return modified

    @staticmethod
    def validate_skill_check(
        skill_name: str,
        skill_value: int,
        roll: int
    ) -> dict:
        """验证技能检定"""
        if skill_value < 0 or skill_value > 100:
            raise ValueError("技能值必须在 0-100 之间")

        if roll < 1 or roll > 100:
            raise ValueError("掷骰值必须在 1-100 之间")

        success = roll <= skill_value

        # 成功等级
        if roll == 1:
            level = "critical"
        elif roll <= skill_value // 5:
            level = "extreme"
        elif roll <= skill_value // 2:
            level = "hard"
        elif roll <= skill_value:
            level = "regular"
        else:
            level = "failure"

        return {
            "success": success,
            "level": level,
            "roll": roll,
            "skill_value": skill_value,
            "diff": skill_value - roll
        }
```

### 错误处理

```python
from pydantic import ValidationError

class CharacterValidationError(Exception):
    """角色卡验证错误"""

    def __init__(self, errors: List[dict]):
        self.errors = errors
        super().__init__(f"角色卡验证失败: {len(errors)} 个错误")

    def to_response(self) -> dict:
        """转换为响应格式"""
        return {
            "detail": "角色卡验证失败",
            "errors": self.errors
        }

# 使用示例
try:
    character = CharacterCreate(**data)
except ValidationError as e:
    # 格式化错误
    errors = []
    for error in e.errors():
        errors.append({
            "field": ".".join(str(loc) for loc in error["loc"]),
            "message": error["msg"],
            "type": error["type"]
        })

    raise CharacterValidationError(errors)
```

## 依赖关系
- 前置任务: M0-030 定义 Skill 技能结构
- 被依赖: M1-025 实现属性自动计算

## 预估工时
2h
