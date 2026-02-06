# M1-060: 实现惩罚骰逻辑

**任务类型**: backend
**预估工时**: 2h
**依赖**: M1-057, M1-058
**状态**: [ ]

---

## 子任务拆解

### 1.1 定义惩罚骰配置 (20min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-060-01 | [ ] 创建 `app/core/penalty.py` | [ ] |
| M1-060-02 | [ ] 定义 `PenaltyType` 枚举 | [ ] |
| M1-060-03 | [ ] 定义 `PenaltyConfig` 数据类 | [ ] |

```python
# app/core/penalty.py
from enum import Enum
from dataclasses import dataclass
from typing import Optional
from app.core.dice import DiceResult
from app.core.success import SuccessResult

class PenaltyType(Enum):
    """惩罚骰类型"""
    UNLUCKY = "unlucky"         # 不幸
    INJURY = "injury"           # 伤害惩罚
    ENVIRONMENT = "environment"  # 环境惩罚
    ITEM_BROKEN = "item_broken" # 装备损坏

@dataclass
class PenaltyConfig:
    """惩罚骰配置"""
    penalty_count: int = 1      # 惩罚骰数量
    dice_type: int = 100        # 骰子类型
    max_penalty_roll: int = 100 # 最大生效值
    reason: str = ""             # 触发原因
```

---

### 1.2 实现惩罚骰判定 (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-060-04 | [ ] 实现 `apply_penalty()` 函数 | [ ] |
| M1-060-05 | [ ] 实现 `select_worst_penalty()` 函数 | [ ] |
| M1-060-06 | [ ] 处理多个惩罚骰 | [ ] |
| M1-060-07 | [ ] 添加边界值处理 | [ ] |

```python
def apply_penalty(
    base_result: DiceResult,
    penalty_config: PenaltyConfig,
    target_value: Optional[int] = None
) -> tuple[DiceResult, SuccessResult]:
    """
    应用惩罚骰到基础掷骰结果

    规则:
    - 惩罚骰取最低值
    - 如果惩罚骰结果 > max_penalty_roll，则不使用惩罚
    - 大成功/大失败规则仍然适用

    Args:
        base_result: 基础掷骰结果
        penalty_config: 惩罚骰配置
        target_value: 目标值（用于成功判定）

    Returns:
        tuple: (最终DiceResult, SuccessResult)
    """
    # 掷出惩罚骰
    penalty_rolls = [
        random.randint(1, penalty_config.dice_type)
        for _ in range(penalty_config.penalty_count)
    ]

    # 选择最低值作为惩罚结果
    worst_penalty = min(penalty_rolls)

    # 检查是否触发惩罚
    if worst_penalty > penalty_config.max_penalty_roll:
        # 惩罚未触发，使用基础结果
        final_result = base_result
        penalty_triggered = False
    else:
        # 惩罚生效，取低值
        final_result = DiceResult(
            dice_type=base_result.dice_type,
            raw_roll=base_result.raw_roll,
            modifier=base_result.modifier,
            final_value=min(base_result.final_value, worst_penalty)
        )
        penalty_triggered = True

    # 计算成功等级
    success_result = None
    if target_value is not None and penalty_triggered:
        from app.core.success import calculate_success_level
        success_result = calculate_success_level(
            final_result.final_value,
            target_value
        )

    return final_result, success_result
```

---

### 1.3 实现惩罚骰组合 (20min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-060-08 | [ ] 实现 `combine_penalties()` 函数 | [ ] |
| M1-060-09 | [ ] 实现 `calculate_total_penalty()` 函数 | [ ] |
| M1-060-10 | [ ] 添加优先级处理 | [ ] |

```python
def combine_penalties(penalties: list[PenaltyConfig]) -> PenaltyConfig:
    """
    合并多个惩罚骰配置

    规则:
    - 相同类型惩罚骰取最高数量
    - 不同类型惩罚骰数量累加
    - 最大生效值取最宽松（最小值）
    """
    if not penalties:
        return PenaltyConfig()

    # 按类型分组
    type_groups: dict[PenaltyType, list[PenaltyConfig]] = {}
    for penalty in penalties:
        if penalty.type not in type_groups:
            type_groups[penalty.type] = []
        type_groups[penalty.type].append(penalty)

    # 计算合并后的配置
    total_penalty_count = 0
    max_penalty_roll = 100

    for penalty_type, group in type_groups.items():
        # 相同类型取最高惩罚骰数量
        max_count = max(p.penalty_count for p in group)
        total_penalty_count += max_count
        # 最宽松的最大生效值
        max_penalty_roll = min(max_penalty_roll, *(p.max_penalty_roll for p in group))

    # 取第一个的原因作为主要理由
    primary_reason = penalties[0].reason if penalties else ""

    return PenaltyConfig(
        penalty_count=total_penalty_count,
        max_penalty_roll=max_penalty_roll,
        reason=primary_reason
    )
```

---

### 1.4 集成到掷骰模块 (20min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-060-11 | [ ] 在 `dice.py` 中导入 penalty 模块 | [ ] |
| M1-060-12 | [ ] 修改 `roll_d100()` 支持惩罚骰参数 | [ ] |

```python
# app/core/dice.py
from app.core.penalty import PenaltyConfig, apply_penalty, PenaltyType

def roll_d100(
    modifier: int = 0,
    target_value: Optional[int] = None,
    bonus: Optional[BonusConfig] = None,
    penalty: Optional[PenaltyConfig] = None
) -> tuple[DiceResult, Optional[SuccessResult]]:
    """
    掷一个 d100，可选应用奖励骰和惩罚骰

    Args:
        modifier: 修正值
        target_value: 目标值（用于成功判定）
        bonus: 奖励骰配置
        penalty: 惩罚骰配置

    Returns:
        tuple: (DiceResult, SuccessResult 或 None)
    """
    # 基础掷骰
    base_result = DiceResult(
        dice_type=100,
        raw_roll=random.randint(1, 100),
        modifier=modifier,
        final_value=random.randint(1, 100) + modifier
    )

    final_result = base_result
    success_result = None

    # 应用惩罚骰（先处理惩罚）
    if penalty is not None:
        final_result, _ = apply_penalty(final_result, penalty, None)

    # 应用奖励骰
    if bonus is not None:
        final_result, success_result = apply_bonus(final_result, bonus, target_value)
    elif target_value is not None:
        from app.core.success import calculate_success_level
        success_result = calculate_success_level(final_result.final_value, target_value)

    return final_result, success_result
```

---

### 1.5 编写单元测试 (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-060-13 | [ ] 创建 `tests/test_penalty.py` | [ ] |
| M1-060-14 | [ ] 测试惩罚骰取最低规则 | [ ] |
| M1-060-15 | [ ] 测试最大生效值 | [ ] |
| M1-060-16 | [ ] 测试奖惩同时存在 | [ ] |

```python
# tests/test_penalty.py
import pytest
from app.core.penalty import PenaltyConfig, PenaltyType, apply_penalty, combine_penalties
from app.core.dice import DiceResult

class TestPenaltyDice:
    def test_apply_penalty_takes_lower(self):
        """测试惩罚骰取低值"""
        base = DiceResult(dice_type=100, raw_roll=80, modifier=0, final_value=80)
        penalty = PenaltyConfig(penalty_count=1, max_penalty_roll=100, reason="测试")

        with patch('random.randint', return_value=30):
            final, _ = apply_penalty(base, penalty)

        assert final.final_value == 30

    def test_penalty_above_max_not_used(self):
        """测试高于最大值的惩罚不生效"""
        base = DiceResult(dice_type=100, raw_roll=80, modifier=0, final_value=80)
        penalty = PenaltyConfig(penalty_count=1, max_penalty_roll=50, reason="测试")

        with patch('random.randint', return_value=70):
            final, _ = apply_penalty(base, penalty)

        assert final.final_value == 80  # 使用基础值

    def test_bonus_and_penalty_combined(self):
        """测试奖励和惩罚同时存在"""
        base = DiceResult(dice_type=100, raw_roll=50, modifier=0, final_value=50)

        bonus = BonusConfig(bonus_count=1, min_bonus_roll=1, reason="奖励")
        penalty = PenaltyConfig(penalty_count=1, max_penalty_roll=100, reason="惩罚")

        # 先惩罚后奖励
        with patch('random.randint', side_effect=[20, 90]):  # 惩罚=20, 奖励=90
            after_penalty, _ = apply_penalty(base, penalty, None)
            final, _ = apply_bonus(after_penalty, bonus, 75)

        assert final.final_value == 90  # 奖励生效
```

---

## 验收标准

- [ ] 惩罚骰取多个中的最低值
- [ ] 惩罚骰可设置最大生效值
- [ ] 奖励骰优先于惩罚骰
- [ ] 多个惩罚骰可合并
- [ ] 单元测试覆盖率 > 90%

---

## 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `app/core/penalty.py` | 创建 | 惩罚骰模块 |
| `app/core/dice.py` | 修改 | 集成惩罚骰 |
| `tests/test_penalty.py` | 创建 | 单元测试 |
