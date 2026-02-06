# M1-025 实现属性自动计算

## 概述
实现角色卡派生属性的自动计算逻辑,包括 HP/MP/Move/DB/Build 等。

## 验收标准
- [ ] 实现 HP 自动计算
- [ ] 实现 MP 自动计算
- [ ] 实现 Move 自动计算
- [ ] 实现 DB/Build 查表计算
- [ ] 实现 Luck 自动计算
- [ ] 提供重新计算接口

## 技术方案

### 计算器类

```python
from typing import Dict, Tuple
import math

class AttributeCalculator:
    """属性计算器"""

    @staticmethod
    def calculate_hp(con: int, siz: int) -> int:
        """
        计算 HP

        公式: (CON + SIZ) // 10
        """
        return (con + siz) // 10

    @staticmethod
    def calculate_mp(pow: int) -> int:
        """
        计算 MP

        公式: POW // 5
        """
        return pow // 5

    @staticmethod
    def calculate_luck(pow: int) -> int:
        """
        计算 Luck

        公式: POW * 5
        """
        return pow * 5

    @staticmethod
    def calculate_move(dex: int, siz: int, str: int) -> int:
        """
        计算 Move

        规则:
        - DEX >= SIZ 或 STR >= SIZ: 8
        - 否则: 7
        """
        if dex >= siz or str >= siz:
            return 8
        return 7

    @staticmethod
    def calculate_db_and_build(str: int, siz: int) -> Tuple[str, int]:
        """
        计算 DB 和 Build

        查表:
        | STR+SIZ | DB | Build |
        |---------|----|-------|
        | 2-64    | -1 | -2    |
        | 65-84   | -1 | -1    |
        | 85-124  | 0  | 0     |
        | 125-164 | +1d4 | 1   |
        | 165-204 | +1d6 | 2   |
        | 205-284 | +2d6 | 3   |
        | 285-364 | +3d6 | 4   |
        | 365-444 | +4d6 | 5   |
        | 445+    | +5d6 | 5   |
        """
        total = str + siz

        if total <= 64:
            return "-1", -2
        elif total <= 84:
            return "-1", -1
        elif total <= 124:
            return "0", 0
        elif total <= 164:
            return "+1d4", 1
        elif total <= 204:
            return "+1d6", 2
        elif total <= 284:
            return "+2d6", 3
        elif total <= 364:
            return "+3d6", 4
        elif total <= 444:
            return "+4d6", 5
        else:
            return "+5d6", 5

    @staticmethod
    def calculate_all(attributes: Dict[str, int]) -> Dict[str, any]:
        """
        计算所有派生属性
        """
        con = attributes.get("CON", 0)
        siz = attributes.get("SIZ", 0)
        dex = attributes.get("DEX", 0)
        str_val = attributes.get("STR", 0)
        pow_val = attributes.get("POW", 0)

        hp = AttributeCalculator.calculate_hp(con, siz)
        mp = AttributeCalculator.calculate_mp(pow_val)
        luck = AttributeCalculator.calculate_luck(pow_val)
        move = AttributeCalculator.calculate_move(dex, siz, str_val)
        db, build = AttributeCalculator.calculate_db_and_build(str_val, siz)

        return {
            "HP": hp,
            "HP_max": hp,
            "MP": mp,
            "MP_max": mp,
            "Luck": luck,
            "Luck_max": luck,
            "Move": move,
            "DB": db,
            "Build": build
        }
```

### 年龄修正

```python
class AgeModifier:
    """年龄修正"""

    MODIFIERS = {
        (15, 19): {"STR": -5, "SIZ": -5},
        (40, 49): {"STR": -5, "CON": -5, "DEX": -5},
        (50, 59): {"STR": -10, "CON": -10, "DEX": -10},
        (60, 69): {"STR": -20, "CON": -20, "DEX": -20},
        (70, 79): {"STR": -40, "CON": -40, "DEX": -40}
    }

    EDU_ROLLS = {
        (15, 19): 1,
        (20, 39): 1,
        (40, 49): 2,
        (50, 59): 3,
        (60, 69): 4,
        (70, 79): 4
    }

    @staticmethod
    def apply(age: int, attributes: Dict[str, int]) -> Dict[str, int]:
        """应用年龄修正"""
        modified = attributes.copy()

        for (min_age, max_age), mods in AgeModifier.MODIFIERS.items():
            if min_age <= age <= max_age:
                for attr, change in mods.items():
                    modified[attr] = max(0, modified[attr] + change)
                break

        return modified

    @staticmethod
    def get_edu_rolls(age: int) -> int:
        """获取 EDU 重掷次数"""
        for (min_age, max_age), rolls in AgeModifier.EDU_ROLLS.items():
            if min_age <= age <= max_age:
                return rolls
        return 0

    @staticmethod
    def roll_edu_improvement(
        current_edu: int,
        rolls: int
    ) -> int:
        """进行 EDU 改进掷骰"""
        best_edu = current_edu

        for _ in range(rolls):
            # 掷 d10 * 10
            roll = (random.randint(1, 10) * 10)
            if roll > best_edu:
                best_edu = min(99, roll)

        return best_edu
```

### 服务层

```python
from fastapi import Depends
from sqlalchemy.orm import Session

class CharacterCalculationService:
    """角色卡计算服务"""

    def __init__(self, db: Session):
        self.db = db
        self.calculator = AttributeCalculator()
        self.age_modifier = AgeModifier()

    def recalculate_character(
        self,
        character_id: str
    ) -> Dict[str, any]:
        """
        重新计算角色卡派生属性
        """
        character = self.db.query(Character).filter(
            Character.id == character_id
        ).first()

        if not character:
            raise ValueError("角色不存在")

        # 解析属性
        attributes = {
            "STR": character.str,
            "CON": character.con,
            "DEX": character.dex,
            "APP": character.app,
            "POW": character.pow,
            "INT": character.int,
            "SIZ": character.siz,
            "EDU": character.edu
        }

        # 计算派生属性
        derived = self.calculator.calculate_all(attributes)

        # 更新角色卡
        character.hp = derived["HP"]
        character.hp_max = derived["HP_max"]
        character.mp = derived["MP"]
        character.mp_max = derived["MP_max"]
        character.luck = derived["Luck"]
        character.luck_max = derived["Luck_max"]
        character.move = derived["Move"]
        character.db = derived["DB"]
        character.build = derived["Build"]

        self.db.commit()

        return derived

    def apply_age_modifiers(
        self,
        character_id: str,
        new_age: int
    ) -> Dict[str, any]:
        """
        应用年龄修正
        """
        character = self.db.query(Character).filter(
            Character.id == character_id
        ).first()

        if not character:
            raise ValueError("角色不存在")

        # 应用年龄修正
        current_attributes = {
            "STR": character.str,
            "CON": character.con,
            "DEX": character.dex,
            "APP": character.app,
            "POW": character.pow,
            "INT": character.int,
            "SIZ": character.siz,
            "EDU": character.edu
        }

        modified = self.age_modifier.apply(new_age, current_attributes)

        # 更新属性
        for attr, value in modified.items():
            setattr(character, attr.lower(), value)

        # 重新计算派生属性
        derived = self.calculator.calculate_all(modified)
        character.hp = derived["HP"]
        character.hp_max = derived["HP_max"]
        character.mp = derived["MP"]
        character.mp_max = derived["MP_max"]
        character.move = derived["Move"]

        # EDU 改进
        edu_rolls = self.age_modifier.get_edu_rolls(new_age)
        if edu_rolls > 0:
            character.edu = self.age_modifier.roll_edu_improvement(
                character.edu,
                edu_rolls
            )

        self.db.commit()

        return {
            "attributes": modified,
            "derived": derived
        }
```

### API 端点

```python
from fastapi import APIRouter, Depends

router = APIRouter(prefix="/characters", tags=["characters"])

@router.post("/{character_id}/recalculate")
async def recalculate_character(
    character_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """
    重新计算角色卡派生属性
    """
    service = CharacterCalculationService(db)

    # 权限检查
    character = db.query(Character).filter(
        Character.id == character_id
    ).first()

    if not character:
        raise HTTPException(status_code=404, detail="角色不存在")

    if character.user_id != current_user.id:
        raise HTTPException(status_code=403, detail="无权限")

    # 重新计算
    derived = service.recalculate_character(character_id)

    return {
        "message": "派生属性已重新计算",
        "character_id": character_id,
        "derived": derived
    }

@router.post("/{character_id}/age")
async def update_character_age(
    character_id: str,
    new_age: int,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """
    更新角色年龄并应用修正
    """
    service = CharacterCalculationService(db)

    # 权限检查
    character = db.query(Character).filter(
        Character.id == character_id
    ).first()

    if not character:
        raise HTTPException(status_code=404, detail="角色不存在")

    if character.user_id != current_user.id:
        raise HTTPException(status_code=403, detail="无权限")

    # 应用年龄修正
    result = service.apply_age_modifiers(character_id, new_age)

    return {
        "message": f"年龄已更新至 {new_age} 岁",
        "character_id": character_id,
        "changes": result
    }
```

### 测试用例

```python
import pytest

def test_calculate_hp():
    """测试 HP 计算"""
    assert AttributeCalculator.calculate_hp(50, 50) == 10
    assert AttributeCalculator.calculate_hp(30, 70) == 10
    assert AttributeCalculator.calculate_hp(80, 80) == 16

def test_calculate_mp():
    """测试 MP 计算"""
    assert AttributeCalculator.calculate_mp(50) == 10
    assert AttributeCalculator.calculate_mp(80) == 16

def test_calculate_move():
    """测试 Move 计算"""
    assert AttributeCalculator.calculate_move(60, 50, 50) == 8
    assert AttributeCalculator.calculate_move(40, 50, 40) == 7

def test_calculate_db_and_build():
    """测试 DB/Build 计算"""
    assert AttributeCalculator.calculate_db_and_build(40, 40) == ("-1", -1)
    assert AttributeCalculator.calculate_db_and_build(60, 60) == ("0", 0)
    assert AttributeCalculator.calculate_db_and_build(80, 80) == ("+1d4", 1)

def test_age_modifiers():
    """测试年龄修正"""
    attrs = {"STR": 50, "CON": 50, "DEX": 50}
    modified = AgeModifier.apply(45, attrs)
    assert modified["STR"] == 45
    assert modified["CON"] == 45
    assert modified["DEX"] == 45
```

## 依赖关系
- 前置任务: M1-024 实现角色卡字段验证
- 被依赖: M1-029 实现角色卡表单 CharacterForm

## 预估工时
2h
