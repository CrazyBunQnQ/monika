# M1-059: 实现奖励骰逻辑

**任务类型**: backend
**预估工时**: 2h
**依赖**: M1-057, M1-058
**状态**: [ ]

---

## 子任务拆解

### 1.1 定义奖励骰配置 (20min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-059-01 | [ ] 创建 `app/core/bonus.py` | [ ] |
| M1-059-02 | [ ] 定义 `BonusType` 枚举 | [ ] |
| M1-059-03 | [ ] 定义 `BonusConfig` 数据类 | [ ] |

```python
# app/core/bonus.py
from enum import Enum
from dataclasses import dataclass
from typing import Optional
from app.core.dice import DiceResult
from app.core.success import SuccessResult

class BonusType(Enum):
    """奖励骰类型"""
    LUCKY = "lucky"           # 幸运使用
    SKILL_BONUS = "skill_bonus"   # 技能奖励
    ITEM_BONUS = "item_bonus"     # 装备奖励
    CIRCUMSTANCE = "circumstance" # 情境奖励

@dataclass
class BonusConfig:
    """奖励骰配置"""
    bonus_count: int = 1      # 奖励骰数量
    dice_type: int = 100       # 骰子类型
    min_bonus_roll: int = 1    # 最小生效值
    reason: str = ""            # 触发原因
```

---

### 1.2 实现奖励骰判定算法 (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-059-04 | [ ] 实现 `apply_bonus()` 函数 | [ ] |
| M1-059-05 | [ ] 实现 `select_best_bonus()` 函数 | [ ] |
| M1-059-06 | [ ] 处理多个奖励骰 | [ ] |
| M1-059-07 | [ ] 添加边界值处理 | [ ] |

```python
def apply_bonus(
    base_result: DiceResult,
    bonus_config: BonusConfig,
    target_value: Optional[int] = None
) -> tuple[DiceResult, SuccessResult]:
    """
    应用奖励骰到基础掷骰结果

    规则:
    - 奖励骰取最高值
    - 如果奖励骰结果 < min_bonus_roll，则不使用奖励
    - 大成功/大失败规则仍然适用

    Args:
        base_result: 基础掷骰结果
        bonus_config: 奖励骰配置
        target_value: 目标值（用于成功判定）

    Returns:
        tuple: (最终DiceResult, SuccessResult)
    """
    # 掷出奖励骰
    bonus_rolls = [
        random.randint(1, bonus_config.dice_type)
        for _ in range(bonus_config.bonus_count)
    ]

    # 选择最高值作为奖励结果
    best_bonus = max(bonus_rolls)

    # 检查是否满足最小生效值
    if best_bonus < bonus_config.min_bonus_roll:
        # 奖励未生效，使用基础结果
        final_result = base_result
        bonus_used = False
    else:
        # 奖励生效，取高值
        final_result = DiceResult(
            dice_type=base_result.dice_type,
            raw_roll=base_result.raw_roll,
            modifier=base_result.modifier,
            final_value=max(base_result.final_value, best_bonus)
        )
        bonus_used = True

    # 计算成功等级
    success_result = None
    if target_value is not None and bonus_used:
        from app.core.success import calculate_success_level
        success_result = calculate_success_level(
            final_result.final_value,
            target_value
        )

    return final_result, success_result
```

---

### 1.3 实现幸运系统 (25min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-059-08 | [ ] 实现 `spend_luck()` 函数 | [ ] |
| M1-059-09 | [ ] 实现 `regain_luck()` 函数 | [ ] |
| M1-059-10 | [ ] 实现 `get_luck_balance()` 函数 | [ ] |

```python
@dataclass
class LuckTransaction:
    """幸运交易记录"""
    amount: int
    reason: str
    timestamp: datetime
    resulting_balance: int

class LuckSystem:
    """幸运值管理系统"""

    def __init__(self, initial_luck: int = 0):
        self._luck = initial_luck
        self._transactions: list[LuckTransaction] = []

    def spend_luck(self, amount: int, reason: str) -> bool:
        """
        花费幸运值

        Returns:
            bool: 是否成功花费
        """
        if self._luck >= amount:
            self._luck -= amount
            self._transactions.append(LuckTransaction(
                amount=-amount,
                reason=reason,
                timestamp=datetime.now(),
                resulting_balance=self._luck
            ))
            return True
        return False

    def regain_luck(self, amount: int, reason: str):
        """恢复幸运值"""
        self._luck += amount
        self._transactions.append(LuckTransaction(
            amount=amount,
            reason=reason,
            timestamp=datetime.now(),
            resulting_balance=self._luck
        ))

    def get_balance(self) -> int:
        """获取当前幸运值"""
        return self._luck
```

---

### 1.4 实现奖励骰组合 (20min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-059-11 | [ ] 实现 `combine_bonuses()` 函数 | [ ] |
| M1-059-12 | [ ] 实现 `calculate_total_bonus()` 函数 | [ ] |
| M1-059-13 | [ ] 添加优先级处理 | [ ] |

```python
def combine_bonuses(bonuses: list[BonusConfig]) -> BonusConfig:
    """
    合并多个奖励骰配置

    规则:
    - 相同类型奖励骰取最高数量
    - 不同类型奖励骰数量累加
    - 最小生效值取最严格（最大值）
    """
    if not bonuses:
        return BonusConfig()

    # 按类型分组
    type_groups: dict[BonusType, list[BonusConfig]] = {}
    for bonus in bonuses:
        if bonus.type not in type_groups:
            type_groups[bonus.type] = []
        type_groups[bonus.type].append(bonus)

    # 计算合并后的配置
    total_bonus_count = 0
    min_bonus_roll = 1

    for bonus_type, group in type_groups.items():
        # 相同类型取最高奖励骰数量
        max_count = max(b.bonus_count for b in group)
        total_bonus_count += max_count
        # 最严格的最小生效值
        min_bonus_roll = max(min_bonus_roll, *(b.min_bonus_roll for b in group))

    # 取第一个的原因作为主要理由
    primary_reason = bonuses[0].reason if bonuses else ""

    return BonusConfig(
        bonus_count=total_bonus_count,
        min_bonus_roll=min_bonus_roll,
        reason=primary_reason
    )
```

---

### 1.5 集成到掷骰模块 (20min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-059-14 | [ ] 在 `dice.py` 中导入 bonus 模块 | [ ] |
| M1-059-15 | [ ] 修改 `roll_d100()` 支持奖励骰参数 | [ ] |

```python
# app/core/dice.py
from app.core.bonus import BonusConfig, apply_bonus, BonusType

def roll_d100(
    modifier: int = 0,
    target_value: Optional[int] = None,
    bonus: Optional[BonusConfig] = None
) -> tuple[DiceResult, Optional[SuccessResult]]:
    """
    掷一个 d100，可选应用奖励骰

    Args:
        modifier: 修正值
        target_value: 目标值（用于成功判定）
        bonus: 奖励骰配置

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

    # 应用奖励骰
    if bonus is not None:
        final_result, success_result = apply_bonus(base_result, bonus, target_value)
        return final_result, success_result

    # 无奖励骰，只计算成功等级
    success_result = None
    if target_value is not None:
        from app.core.success import calculate_success_level
        success_result = calculate_success_level(base_result.final_value, target_value)

    return base_result, success_result
```

---

### 1.6 编写单元测试 (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-059-16 | [ ] 创建 `tests/test_bonus.py` | [ ] |
| M1-059-17 | [ ] 测试奖励骰取最高规则 | [ ] |
| M1-059-18 | [ ] 测试最小生效值 | [ ] |
| M1-059-19 | [ ] 测试幸运值消耗 | [ ] |
| M1-059-20 | [ ] 测试多奖励骰组合 | [ ] |

```python
# tests/test_bonus.py
import pytest
from app.core.bonus import (
    BonusConfig, BonusType, apply_bonus, combine_bonuses, LuckSystem
)

class TestBonusDice:
    def test_apply_bonus_takes_higher(self):
        """测试奖励骰取高值"""
        base = DiceResult(dice_type=100, raw_roll=30, modifier=0, final_value=30)
        bonus = BonusConfig(bonus_count=1, min_bonus_roll=1, reason="测试")

        # Mock 随机数以确保测试可预测
        with patch('random.randint', return_value=70):
            final, _ = apply_bonus(base, bonus)

        assert final.final_value == 70

    def test_bonus_below_min_not_used(self):
        """测试低于最小值的奖励不生效"""
        base = DiceResult(dice_type=100, raw_roll=90, modifier=0, final_value=90)
        bonus = BonusConfig(bonus_count=1, min_bonus_roll=50, reason="测试")

        with patch('random.randint', return_value=30):
            final, _ = apply_bonus(base, bonus)

        assert final.final_value == 90  # 使用基础值

class TestLuckSystem:
    def test_spend_luck_success(self):
        """测试成功花费幸运"""
        luck = LuckSystem(initial_luck=10)
        assert luck.spend_luck(5, "使用幸运")
        assert luck.get_balance() == 5

    def test_spend_luck_insufficient(self):
        """测试幸运不足"""
        luck = LuckSystem(initial_luck=3)
        assert not luck.spend_luck(5, "尝试使用幸运")
        assert luck.get_balance() == 3
```

---

## 验收标准

- [ ] 奖励骰取多个中的最高值
- [ ] 奖励骰可设置最小生效值
- [ ] 幸运值系统完整
- [ ] 多个奖励骰可合并
- [ ] 单元测试覆盖率 > 90%

---

## 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `app/core/bonus.py` | 创建 | 奖励骰模块 |
| `app/core/dice.py` | 修改 | 集成奖励骰 |
| `tests/test_bonus.py` | 创建 | 单元测试 |

---

## CoC 7e 规则参考

| 场景 | 奖励骰 | 说明 |
|------|--------|------|
| 幸运使用 | 1d6 | 消耗 1 点幸运 |
| 灵感 | 1d6 | 角色拥有高智力的奖励 |
| 奖励 | 1d10 | 特定情境奖励 |
| 惩罚 | -1d10 | 特定负面情境 |

---

## 奖励骰优先级

1. **情境奖励** ( circumstance ) - 可叠加
2. **技能奖励** ( skill ) - 可叠加
3. **装备奖励** ( item ) - 可叠加
4. **幸运使用** ( lucky ) - 独立判定
