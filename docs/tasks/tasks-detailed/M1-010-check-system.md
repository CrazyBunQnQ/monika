# M1-010: 检定系统 API

**任务类型**: backend
**预估工时**: 4h
**依赖**: M1-057, M1-058, M1-003
**状态**: [ ]

---

## 子任务拆解

### 1.1 检定服务 (40min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-010-01 | [ ] 创建 `app/services/check.py` | [ ] |
| M1-010-02 | [ ] 定义 `CheckType` 枚举 | [ ] |
| M1-010-03 | [ ] 定义 `CheckContext` 数据类 | [ ] |
| M1-010-04 | [ ] 定义 `CheckResult` 数据类 | [ ] |

```python
# app/services/check.py
from enum import Enum
from dataclasses import dataclass, field
from typing import Optional, List, Dict, Any
from datetime import datetime
from app.core.dice import DiceResult
from app.core.success import SuccessLevel, SuccessResult
from app.core.bonus import BonusConfig, apply_bonus
from app.core.penalty import PenaltyConfig, apply_penalty

class CheckType(str, Enum):
    """检定类型"""
    SKILL = "skill"             # 技能检定
    ATTRIBUTE = "attribute"     # 属性检定
    COMBAT = "combat"           # 战斗检定
    SANITY = "sanity"           # SAN 检定
    IDEA = "idea"               # 灵感
    LUCK = "luck"              # 幸运
    OPPOSED = "opposed"         # 对抗

class Difficulty(str, Enum):
    """难度等级"""
    ROUTINE = "routine"     # 常规 (1/2)
    EASY = "easy"          # 简单 (1/3)
    HARD = "hard"          # 困难 (1/5)
    EXTREME = "extreme"    # 极难 (1/10)
    IMPOSSIBLE = "impossible"  # 不可能 (1/100)

@dataclass
class CheckContext:
    """检定上下文"""
    character_id: int
    check_type: CheckType
    skill_key: Optional[str] = None
    target_value: int
    difficulty: Difficulty = Difficulty.ROUTINE

    # 修正值
    modifier: int = 0
    bonus: Optional[BonusConfig] = None
    penalty: Optional[PenaltyConfig] = None

    # 推骰
    allow_push: bool = True

    # 描述
    description: str = ""

@dataclass
class CheckResult:
    """检定结果"""
    success_level: SuccessLevel
    is_critical: bool = False
    is_fumble: bool = False

    # 数值
    raw_roll: int = 0
    final_value: int = 0
    target_value: int = 0

    # 奖励/惩罚
    bonus_applied: bool = False
    penalty_applied: bool = False

    # 推骰
    can_push: bool = False
    pushed: bool = False
    push_result: Optional["CheckResult"] = None

    # 描述
    description: str = ""
    narrative: str = ""

    # 元数据
    timestamp: datetime = field(default_factory=datetime.utcnow)

    def to_dict(self) -> Dict[str, Any]:
        """转换为字典"""
        result = {
            "success_level": self.success_level.value,
            "is_critical": self.is_critical,
            "is_fumble": self.is_fumble,
            "raw_roll": self.raw_roll,
            "final_value": self.final_value,
            "target_value": self.target_value,
            "can_push": self.can_push,
            "pushed": self.pushed,
            "description": self.description,
            "narrative": self.narrative,
            "timestamp": self.timestamp.isoformat()
        }
        if self.push_result:
            result["push_result"] = self.push_result.to_dict()
        return result
```

---

### 1.2 检定算法实现 (50min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-010-05 | [ ] 实现 `execute_check()` 函数 | [ ] |
| M1-010-06 | [ ] 实现 `calculate_target_from_difficulty()` | [ ] |
| M1-010-07 | [ ] 实现 `generate_narrative()` | [ ] |
| M1-010-08 | [ ] 实现 `execute_opposed_check()` | [ ] |

```python
class CheckService:
    """检定服务"""

    DIFFICULTY_MULTIPLIERS = {
        Difficulty.ROUTINE: 2,
        Difficulty.EASY: 3,
        Difficulty.HARD: 5,
        Difficulty.EXTREME: 10,
        Difficulty.IMPOSSIBLE: 100,
    }

    def calculate_target(
        self,
        skill_value: int,
        difficulty: Difficulty,
        modifier: int = 0
    ) -> int:
        """根据难度计算目标值"""
        multiplier = self.DIFFICULTY_MULTIPLIERS[difficulty]
        base_target = max(1, skill_value // multiplier)
        return base_target + modifier

    def execute_check(
        self,
        context: CheckContext
    ) -> CheckResult:
        """执行检定"""
        # 掷骰
        from app.core.dice import roll_d100
        dice_result = roll_d100(modifier=context.modifier)

        # 记录原始值
        raw_roll = dice_result.final_value

        # 应用惩罚（如果有）
        if context.penalty is not None:
            dice_result, _ = apply_penalty(dice_result, context.penalty)

        # 应用奖励（如果有）
        if context.bonus is not None:
            dice_result, success_result = apply_bonus(
                dice_result, context.bonus, context.target_value
            )

        # 计算成功等级
        from app.core.success import calculate_success_level
        success_result = calculate_success_level(
            dice_result.final_value,
            context.target_value
        )

        # 构建结果
        result = CheckResult(
            success_level=success_result.level,
            is_critical=success_result.is_critical,
            is_fumble=success_result.is_fumble,
            raw_roll=raw_roll,
            final_value=dice_result.final_value,
            target_value=context.target_value,
            bonus_applied=context.bonus is not None,
            penalty_applied=context.penalty is not None,
            can_push=(
                success_result.level == SuccessLevel.FAILURE and
                not success_result.is_fumble and
                context.allow_push
            ),
            description=context.description,
            narrative=self.generate_narrative(context, success_result)
        )

        return result

    def execute_opposed_check(
        self,
        context_a: CheckContext,
        context_b: CheckContext
    ) -> Dict[str, CheckResult]:
        """执行对抗检定"""
        result_a = self.execute_check(context_a)
        result_b = self.execute_check(context_b)

        # 判定胜负
        if result_a.success_level.value < result_b.success_level.value:
            winner = "a"
        elif result_b.success_level.value < result_a.success_level.value:
            winner = "b"
        else:
            # 平局时，比较原始掷骰值
            winner = "tie"
            if result_a.raw_roll < result_b.raw_roll:
                winner = "a"
            elif result_b.raw_roll < result_a.raw_roll:
                winner = "b"

        return {
            "winner": winner,
            "player_a": result_a,
            "player_b": result_b
        }

    def generate_narrative(
        self,
        context: CheckContext,
        result: SuccessResult
    ) -> str:
        """生成叙述文本"""
        level_descriptions = {
            SuccessLevel.CRITICAL_SUCCESS: "完美成功！",
            SuccessLevel.EXTREME_SUCCESS: "极其出色的成功",
            SuccessLevel.HARD_SUCCESS: "成功",
            SuccessLevel.REGULAR_SUCCESS: "勉强成功",
            SuccessLevel.FAILURE: "失败了...",
            SuccessLevel.FUMBLE: "大失败！",
        }

        base = level_descriptions.get(result.level, "")

        if context.description:
            return f"{context.description}: {result.roll_value}/{context.target_value} - {base}"

        return f"检定结果: {result.roll_value}/{context.target_value} - {base}"
```

---

### 1.3 灵感检定 (20min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-010-09 | [ ] 实现 `execute_idea_check()` 函数 | [ ] |
| M1-010-10 | [ ] 实现 `reveal_idea()` 函数 | [ ] |

```python
class CheckService:
    # ...

    def execute_idea_check(
        self,
        character_id: int,
        idea_type: str
    ) -> CheckResult:
        """执行灵感检定

        灵感规则:
        - 自动成功
        - 揭示信息
        - 消耗灵感次数（可选）
        """
        # CoC 7e 灵感: 技能值的 1/5，向下取整
        idea_threshold = 1  # 灵感自动成功

        result = CheckResult(
            success_level=SuccessLevel.CRITICAL_SUCCESS,
            is_critical=True,
            raw_roll=1,
            final_value=1,
            target_value=idea_threshold,
            description=f"灵感 - {idea_type}",
            narrative=f"你灵光一闪！{idea_type}"
        )

        return result
```

---

### 1.4 检定 API 路由 (40min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-010-11 | [ ] 创建 `app/api/check.py` | [ ] |
| M1-010-12 | [ ] 实现 POST /check/skill | [ ] |
| M1-010-13 | [ ] 实现 POST /check/opposed | [ ] |
| M1-010-14 | [ ] 实现 POST /check/push | [ ] |

```python
# app/api/check.py
from typing import Optional
from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel, Field
from sqlmodel import Session
from app.db.user import User
from app.api.deps.auth import get_current_active_user
from app.db.connection import get_session
from app.services.check import CheckService, CheckContext, CheckType, Difficulty
from app.services.character import CharacterService

router = APIRouter(prefix="/check", tags=["检定"])

class SkillCheckRequest(BaseModel):
    """技能检定请求"""
    character_id: int
    skill_key: str = Field(..., min_length=1)
    modifier: int = 0
    difficulty: Difficulty = Difficulty.ROUTINE
    use_bonus: bool = False
    use_penalty: bool = False
    description: Optional[str] = None

class OpposedCheckRequest(BaseModel):
    """对抗检定请求"""
    character_a_id: int
    character_b_id: int
    skill_a_key: str
    skill_b_key: str
    difficulty_a: Difficulty = Difficulty.ROUTINE
    difficulty_b: Difficulty = Difficulty.ROUTINE
    description_a: Optional[str] = None
    description_b: Optional[str] = None

class PushCheckRequest(BaseModel):
    """推骰请求"""
    original_check_id: str
    character_id: int

@router.post("/skill")
async def skill_check(
    request: SkillCheckRequest,
    current_user: User = Depends(get_current_active_user),
    session: Session = Depends(get_session)
):
    """执行技能检定"""
    char_service = CharacterService(session)
    character = char_service.get_character(request.character_id)

    if not character:
        raise HTTPException(status_code=404, detail="角色不存在")

    # 验证所有权
    if character.owner_id != current_user.id:
        raise HTTPException(status_code=403, detail="无权使用此角色")

    # 获取技能值
    skill_service = SkillService()
    skill_check = skill_service.get_skill_check(character, request.skill_key)

    if not skill_check:
        raise HTTPException(status_code=404, detail="技能不存在")

    # 计算目标值
    char_service._recalculate_derived_values(character)

    check_service = CheckService()
    target_value = check_service.calculate_target(
        skill_check["total"],
        request.difficulty,
        request.modifier
    )

    # 构建上下文
    context = CheckContext(
        character_id=request.character_id,
        check_type=CheckType.SKILL,
        skill_key=request.skill_key,
        target_value=target_value,
        difficulty=request.difficulty,
        modifier=request.modifier,
        description=request.description or f"技能检定 - {skill_check['skill_name']}"
    )

    result = check_service.execute_check(context)

    return {
        "result": result.to_dict(),
        "skill": skill_check
    }

@router.post("/opposed")
async def opposed_check(
    request: OpposedCheckRequest,
    current_user: User = Depends(get_current_active_user),
    session: Session = Depends(get_session)
):
    """执行对抗检定"""
    char_service = CharacterService(session)

    # 验证两个角色
    char_a = char_service.get_character(request.character_a_id)
    char_b = char_service.get_character(request.character_b_id)

    if not char_a or not char_b:
        raise HTTPException(status_code=404, detail="角色不存在")

    # 验证所有权
    if char_a.owner_id != current_user.id or char_b.owner_id != current_user.id:
        raise HTTPException(status_code=403, detail="无权使用此角色")

    # 执行对抗
    skill_service = SkillService()

    context_a = CheckContext(
        character_id=request.character_a_id,
        check_type=CheckType.OPPOSED,
        skill_key=request.skill_a_key,
        target_value=50,  # 简化处理
        description=request.description_a
    )

    context_b = CheckContext(
        character_id=request.character_b_id,
        check_type=CheckType.OPPOSED,
        skill_key=request.skill_b_key,
        target_value=50,
        description=request.description_b
    )

    check_service = CheckService()
    results = check_service.execute_opposed_check(context_a, context_b)

    return {
        "results": {
            "player_a": results["player_a"].to_dict(),
            "player_b": results["player_b"].to_dict()
        },
        "winner": results["winner"]
    }
```

---

### 1.5 单元测试 (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-010-15 | [ ] 创建 `tests/test_check.py` | [ ] |
| M1-010-16 | [ ] 测试技能检定 | [ ] |
| M1-010-17 | [ ] 测试难度计算 | [ ] |
| M1-010-18 | [ ] 测试对抗检定 | [ ] |
| M1-010-19 | [ ] 测试推骰逻辑 | [ ] |
| M1-010-20 | [ ] 测试边界情况 | [ ] |

```python
# tests/test_check.py
import pytest
from app.services.check import CheckService, CheckContext, CheckType, Difficulty
from app.core.success import SuccessLevel

class TestCheckService:
    def test_calculate_target_routine(self):
        """测试常规难度目标值"""
        service = CheckService()
        target = service.calculate_target(50, Difficulty.ROUTINE, 0)
        assert target == 25  # 50 / 2

    def test_calculate_target_hard(self):
        """测试困难难度目标值"""
        service = CheckService()
        target = service.calculate_target(50, Difficulty.HARD, 0)
        assert target == 10  # 50 / 5

    def test_calculate_target_extreme(self):
        """测试极难难度目标值"""
        service = CheckService()
        target = service.calculate_target(50, Difficulty.EXTREME, 0)
        assert target == 5  # 50 / 10

    def test_skill_check_success(self):
        """测试技能检定成功"""
        context = CheckContext(
            character_id=1,
            check_type=CheckType.SKILL,
            target_value=50,
            description="图书馆使用"
        )

        service = CheckService()
        # Mock 掷骰为 30
        with patch('app.core.dice.random.randint', return_value=30):
            result = service.execute_check(context)

        assert result.final_value == 30
        assert result.success_level == SuccessLevel.REGULAR_SUCCESS

    def test_skill_check_failure(self):
        """测试技能检定失败"""
        context = CheckContext(
            character_id=1,
            check_type=CheckType.SKILL,
            target_value=50,
            description="侦查"
        )

        service = CheckService()
        with patch('app.core.dice.random.randint', return_value=75):
            result = service.execute_check(context)

        assert result.success_level == SuccessLevel.FAILURE
        assert result.can_push is True  # 失败可以推骰

    def test_critical_success(self):
        """测试大成功"""
        context = CheckContext(
            character_id=1,
            check_type=CheckType.SKILL,
            target_value=50,
            description="侦查"
        )

        service = CheckService()
        with patch('app.core.dice.random.randint', return_value=5):
            result = service.execute_check(context)

        assert result.is_critical is True
        assert result.success_level == SuccessLevel.CRITICAL_SUCCESS

    def test_fumble(self):
        """测试大失败"""
        context = CheckContext(
            character_id=1,
            check_type=CheckType.SKILL,
            target_value=50,
            description="战斗"
        )

        service = CheckService()
        with patch('app.core.dice.random.randint', return_value=97):
            result = service.execute_check(context)

        assert result.is_fumble is True
        assert result.success_level == SuccessLevel.FUMBLE
```

---

## 验收标准

- [ ] 技能检定 API 完整
- [ ] 支持多种难度
- [ ] 支持奖励/惩罚骰
- [ ] 失败后可推骰
- [ ] 支持对抗检定
- [ ] 单元测试覆盖率 > 90%

---

## 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `app/services/check.py` | 创建 | 检定服务 |
| `app/api/check.py` | 创建 | 检定 API |
| `tests/test_check.py` | 创建 | 单元测试 |

---

## API 文档

### POST /check/skill

```json
{
  "character_id": 1,
  "skill_key": "library_use",
  "modifier": 0,
  "difficulty": "routine",
  "description": "在图书馆查找线索"
}
```

**响应:**
```json
{
  "result": {
    "success_level": "regular_success",
    "is_critical": false,
    "is_fumble": false,
    "raw_roll": 35,
    "final_value": 35,
    "target_value": 40,
    "can_push": false,
    "narrative": "在图书馆查找线索: 35/40 - 成功"
  },
  "skill": {
    "skill_key": "library_use",
    "skill_name": "图书馆使用",
    "base_value": 25,
    "occupation_bonus": 10,
    "total": 50
  }
}
```
