# M1-058: 实现大成功/大失败判定

**任务类型**: backend
**预估工时**: 2h
**依赖**: M1-057
**状态**: [ ]

---

## 子任务拆解

### 1.1 定义成功等级枚举 (20min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-058-01 | [ ] 创建 `app/core/success.py` | [ ] |
| M1-058-02 | [ ] 定义 `SuccessLevel` 枚举 | [ ] |
| M1-058-03 | [ ] 添加 Docstring | [ ] |

```python
# app/core/success.py
from enum import Enum
from dataclasses import dataclass
from typing import Optional

class SuccessLevel(Enum):
    """CoC 7e 成功等级"""
    CRITICAL_SUCCESS = "critical"   # 大成功 (1/5 成功率)
    EXTREME_SUCCESS = "extreme"    # 极难成功
    HARD_SUCCESS = "hard"         # 困难成功
    REGULAR_SUCCESS = "regular"   # 普通成功
    FAILURE = "failure"           # 失败
    FUMBLE = "fumble"             # 大失败 (96-100)

@dataclass
class SuccessResult:
    level: SuccessLevel
    roll_value: int
    target_value: int
    is_critical: bool = False
    is_fumble: bool = False
```

---

### 1.2 实现判定算法 (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-058-04 | [ ] 实现 `calculate_success_level()` 函数 | [ ] |
| M1-058-05 | [ ] 处理大成功判定 (≤1/5) | [ ] |
| M1-058-06 | [ ] 处理大失败判定 (96-100) | [ ] |
| M1-058-07 | [ ] 添加边界值处理 | [ ] |

```python
def calculate_success_level(
    roll_value: int,
    target_value: int,
    skill_value: Optional[int] = None
) -> SuccessResult:
    """
    根据 CoC 7e 规则计算成功等级

    Args:
        roll_value: 掷骰结果 (1-100)
        target_value: 目标值 (技能值)
        skill_value: 技能值 (与 target 相同)

    Returns:
        SuccessResult: 包含成功等级和详情
    """
    # 大失败判定
    if roll_value >= 96:
        return SuccessResult(
            level=SuccessLevel.FUMBLE,
            roll_value=roll_value,
            target_value=target_value,
            is_fumble=True
        )

    # 大成功判定 (1/5 目标值，向下取整，至少为1)
    critical_threshold = max(1, target_value // 5)
    if roll_value <= critical_threshold:
        return SuccessResult(
            level=SuccessLevel.CRITICAL_SUCCESS,
            roll_value=roll_value,
            target_value=target_value,
            is_critical=True
        )

    # 极难成功判定 (≤1/5)
    extreme_threshold = max(1, target_value // 5)
    if roll_value <= extreme_threshold:
        return SuccessResult(
            level=SuccessLevel.EXTREME_SUCCESS,
            roll_value=roll_value,
            target_value=target_value
        )

    # 困难成功判定 (≤1/2)
    hard_threshold = target_value // 2
    if roll_value <= hard_threshold:
        return SuccessResult(
            level=SuccessLevel.HARD_SUCCESS,
            roll_value=roll_value,
            target_value=target_value
        )

    # 普通成功判定 (≤目标值)
    if roll_value <= target_value:
        return SuccessResult(
            level=SuccessLevel.REGULAR_SUCCESS,
            roll_value=roll_value,
            target_value=target_value
        )

    # 失败
    return SuccessResult(
        level=SuccessLevel.FAILURE,
        roll_value=roll_value,
        target_value=target_value
    )
```

---

### 1.3 实现辅助函数 (20min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-058-08 | [ ] 实现 `get_success_description()` 函数 | [ ] |
| M1-058-09 | [ ] 实现 `get_success_emoji()` 函数 | [ ] |
| M1-058-10 | [ ] 实现 `is_pushable()` 函数 | [ ] |

```python
def get_success_description(level: SuccessLevel) -> str:
    """获取成功等级的中文描述"""
    descriptions = {
        SuccessLevel.CRITICAL_SUCCESS: "大成功！",
        SuccessLevel.EXTREME_SUCCESS: "极难成功",
        SuccessLevel.HARD_SUCCESS: "困难成功",
        SuccessLevel.REGULAR_SUCCESS: "成功",
        SuccessLevel.FAILURE: "失败",
        SuccessLevel.FUMBLE: "大失败！",
    }
    return descriptions.get(level, "未知")

def is_pushable(result: SuccessResult) -> bool:
    """判断是否可以推骰"""
    return (
        result.level == SuccessLevel.FAILURE and
        not result.is_fumble
    )
```

---

### 1.4 集成到掷骰模块 (20min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-058-11 | [ ] 在 `dice.py` 中导入 success 模块 | [ ] |
| M1-058-12 | [ ] 修改 `roll_d100()` 返回成功等级 | [ ] |

```python
# app/core/dice.py
from app.core.success import calculate_success_level, SuccessLevel

def roll_d100(
    modifier: int = 0,
    target_value: Optional[int] = None
) -> tuple[DiceResult, Optional[SuccessResult]]:
    """掷一个 d100，可选计算成功等级"""
    dice_result = DiceResult(
        dice_type=100,
        raw_roll=random.randint(1, 100),
        modifier=modifier,
        final_value=raw_roll + modifier
    )

    if target_value is not None:
        success_result = calculate_success_level(
            dice_result.final_value,
            target_value
        )
        return dice_result, success_result

    return dice_result, None
```

---

### 1.5 编写单元测试 (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-058-13 | [ ] 创建 `tests/test_success.py` | [ ] |
| M1-058-14 | [ ] 测试大成功判定 (1/5) | [ ] |
| M1-058-15 | [ ] 测试大失败判定 (96-100) | [ ] |
| M1-058-16 | [ ] 测试各种成功等级 | [ ] |
| M1-058-17 | [ ] 测试边界值 | [ ] |

```python
# tests/test_success.py
import pytest
from app.core.success import (
    calculate_success_level,
    SuccessLevel
)

class TestSuccessLevel:
    def test_critical_success_1_of_5(self):
        """测试 1/5 阈值为大成功"""
        result = calculate_success_level(5, 50)  # 5 <= 10 (50/5)
        assert result.level == SuccessLevel.CRITICAL_SUCCESS

    def test_extreme_success_1_of_5(self):
        """测试极难成功边界"""
        result = calculate_success_level(10, 50)  # 10 <= 10
        assert result.level == SuccessLevel.EXTREME_SUCCESS

    def test_hard_success_half(self):
        """测试困难成功"""
        result = calculate_success_level(20, 50)  # 20 <= 25
        assert result.level == SuccessLevel.HARD_SUCCESS

    def test_regular_success(self):
        """测试普通成功"""
        result = calculate_success_level(40, 50)
        assert result.level == SuccessLevel.REGULAR_SUCCESS

    def test_failure(self):
        """测试失败"""
        result = calculate_success_level(60, 50)
        assert result.level == SuccessLevel.FAILURE

    def test_fumble_96(self):
        """测试大失败 96"""
        result = calculate_success_level(96, 50)
        assert result.level == SuccessLevel.FUMBLE
        assert result.is_fumble

    def test_fumble_100(self):
        """测试大失败 100"""
        result = calculate_success_level(100, 1)  # 即使目标1，大失败也是失败
        assert result.level == SuccessLevel.FUMBLE
```

---

## 验收标准

- [ ] `SuccessLevel` 枚举包含所有等级
- [ ] 大成功: 掷骰 ≤ 目标值/5
- [ ] 大失败: 掷骰 ≥ 96
- [ ] 单元测试覆盖率 > 90%
- [ ] 成功等级可正确翻译为中文

---

## 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `app/core/success.py` | 创建 | 成功等级模块 |
| `app/core/dice.py` | 修改 | 集成成功判定 |
| `tests/test_success.py` | 创建 | 单元测试 |

---

## CoC 7e 规则参考

| 成功等级 | 阈值 | 描述 |
|----------|------|------|
| 大成功 | ≤ 1/5 目标值 | 极其出色的表现 |
| 极难成功 | ≤ 1/5 目标值 | 非常困难的成功 |
| 困难成功 | ≤ 1/2 目标值 | 有难度的成功 |
| 普通成功 | ≤ 目标值 | 基础成功 |
| 失败 | > 目标值 | 未成功 |
| 大失败 | 96-100 | 糟糕的失误 |

---

## 成功等级阈值示例

| 技能值 | 大成功 | 极难 | 困难 | 成功 | 失败 |
|--------|--------|------|------|------|------|
| 25 | 1-5 | 1-5 | 1-13 | 1-25 | 26+ |
| 50 | 1-10 | 1-10 | 1-25 | 1-50 | 51+ |
| 75 | 1-15 | 1-15 | 1-38 | 1-75 | 76+ |
| 99 | 1-20 | 1-20 | 1-50 | 1-99 | 100 |
