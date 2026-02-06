# M1-061: 实现推骰机制

**任务类型**: backend
**预估工时**: 3h
**依赖**: M1-057, M1-058
**状态**: [ ]

---

## 子任务拆解

### 1.1 定义推骰状态 (25min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-061-01 | [ ] 创建 `app/core/push.py` | [ ] |
| M1-061-02 | [ ] 定义 `PushState` 枚举 | [ ] |
| M1-061-03 | [ ] 定义 `PushRoll` 数据类 | [ ] |
| M1-061-04 | [ ] 定义 `PushResult` 数据类 | [ ] |

```python
# app/core/push.py
from enum import Enum
from dataclasses import dataclass, field
from typing import Optional
from datetime import datetime
from app.core.dice import DiceResult
from app.core.success import SuccessResult, SuccessLevel

class PushState(Enum):
    """推骰状态"""
    NOT_PUSHED = "not_pushed"     # 未推骰
    PUSHED = "pushed"            # 已推骰
    EXHAUSTED = "exhausted"      # 已耗尽（不能再推）

@dataclass
class PushRoll:
    """推骰记录"""
    original_roll: DiceResult    # 原始掷骰
    pushed_roll: Optional[DiceResult]  # 推骰结果
    original_result: SuccessResult
    pushed_result: Optional[SuccessResult]
    push_state: PushState
    timestamp: datetime = field(default_factory=datetime.utcnow)

@dataclass
class PushConfig:
    """推骰配置"""
    max_pushes: int = 1         # 最大推骰次数
    allow_critical_improvement: bool = True  # 是否允许大成功改进
    allow_fumble_worsen: bool = True         # 是否允许大失败恶化
```

---

### 1.2 实现推骰算法 (40min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-061-05 | [ ] 实现 `can_push()` 函数 | [ ] |
| M1-061-06 | [ ] 实现 `execute_push()` 函数 | [ ] |
| M1-061-07 | [ ] 实现 `evaluate_push_result()` 函数 | [ ] |
| M1-061-08 | [ ] 实现 `is_push_better()` 函数 | [ ] |

```python
def can_push(result: SuccessResult, config: PushConfig) -> bool:
    """
    判断是否可以推骰

    规则:
    - 必须是失败（但不是大失败）
    - 推骰次数未达上限

    Args:
        result: 原始检定结果
        config: 推骰配置

    Returns:
        bool: 是否可以推骰
    """
    # 大失败不能推骰
    if result.is_fumble:
        return False

    # 只有失败可以推骰
    if result.level != SuccessLevel.FAILURE:
        return False

    return True


def execute_push(
    original_roll: DiceResult,
    original_result: SuccessResult,
    config: PushConfig
) -> PushRoll:
    """
    执行推骰

    规则:
    - 推骰后使用两个结果中较好的一个
    - 如果两次都失败，第二次失败后果更严重
    - 大成功改进为普通成功/困难成功等
    - 大失败恶化

    Args:
        original_roll: 原始掷骰
        original_result: 原始成功判定
        config: 推骰配置

    Returns:
        PushRoll: 推骰结果
    """
    # 执行第二次掷骰
    pushed_roll = DiceResult(
        dice_type=100,
        raw_roll=random.randint(1, 100),
        modifier=original_roll.modifier,
        final_value=random.randint(1, 100) + original_roll.modifier
    )

    # 计算推骰后的成功等级
    # 注意：使用原始目标值
    pushed_result = calculate_success_level(
        pushed_roll.final_value,
        original_result.target_value
    )

    return PushRoll(
        original_roll=original_roll,
        pushed_roll=pushed_roll,
        original_result=original_result,
        pushed_result=pushed_result,
        push_state=PushState.PUSHED
    )


def evaluate_push_result(
    original: SuccessResult,
    pushed: SuccessResult
) -> SuccessResult:
    """
    评估推骰后的最终结果

    规则:
    - 成功 > 失败：保留成功
    - 失败 > 成功：保留成功（推骰成功）
    - 失败 = 失败：第二次更严重
    """
    # 成功等级数值（越低越好）
    success_order = [
        SuccessLevel.CRITICAL_SUCCESS,  # 0 - 最好
        SuccessLevel.EXTREME_SUCCESS,  # 1
        SuccessLevel.HARD_SUCCESS,      # 2
        SuccessLevel.REGULAR_SUCCESS,  # 3
        SuccessLevel.FAILURE,          # 4
        SuccessLevel.FUMBLE,           # 5 - 最差
    ]

    def get_rank(level: SuccessLevel) -> int:
        try:
            return success_order.index(level)
        except ValueError:
            return 4  # 默认为 Failure

    original_rank = get_rank(original.level)
    pushed_rank = get_rank(pushed.level)

    # 如果推骰结果更好，使用推骰结果
    if pushed_rank < original_rank:
        # 检查是否允许改进
        if original.level in [SuccessLevel.CRITICAL_SUCCESS, SuccessLevel.EXTREME_SUCCESS]:
            # 已经是极难/大成功，不能再改进
            return original
        return pushed

    # 如果一样差或更差，保持原样
    return original
```

---

### 1.3 实现推骰历史 (25min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-061-09 | [ ] 定义 `PushHistory` 数据类 | [ ] |
| M1-061-10 | [ ] 实现 `record_push()` 函数 | [ ] |
| M1-061-11 | [ ] 实现 `get_push_summary()` 函数 | [ ] |

```python
@dataclass
class PushHistory:
    """推骰历史记录"""
    push_rolls: list[PushRoll] = field(default_factory=list)
    push_count: int = 0
    successful_pushes: int = 0

    def record_push(self, push_roll: PushRoll):
        """记录一次推骰"""
        self.push_rolls.append(push_roll)
        self.push_count += 1

        # 检查推骰是否成功（结果改善）
        original_rank = get_success_rank(push_roll.original_result.level)
        pushed_rank = get_success_rank(push_roll.pushed_result.level)

        if pushed_rank < original_rank:
            self.successful_pushes += 1

    def get_summary(self) -> dict:
        """获取推骰摘要"""
        return {
            "total_pushes": self.push_count,
            "successful_pushes": self.successful_pushes,
            "success_rate": (
                self.successful_pushes / self.push_count
                if self.push_count > 0 else 0.0
            ),
            "push_history": [
                {
                    "original_value": pr.original_roll.final_value,
                    "original_level": pr.original_result.level.value,
                    "pushed_value": pr.pushed_roll.final_value if pr.pushed_roll else None,
                    "pushed_level": pr.pushed_result.level.value if pr.pushed_result else None,
                }
                for pr in self.push_rolls
            ]
        }


def get_success_rank(level: SuccessLevel) -> int:
    """获取成功等级排名（越低越好）"""
    order = [
        SuccessLevel.CRITICAL_SUCCESS,
        SuccessLevel.EXTREME_SUCCESS,
        SuccessLevel.HARD_SUCCESS,
        SuccessLevel.REGULAR_SUCCESS,
        SuccessLevel.FAILURE,
        SuccessLevel.FUMBLE,
    ]
    return order.index(level)
```

---

### 1.4 集成到游戏状态 (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-061-12 | [ ] 在 `game_state.py` 中添加推骰状态 | [ ] |
| M1-061-13 | [ ] 实现 `push_current_check()` 函数 | [ ] |
| M1-061-14 | [ ] 实现 `can_push_check()` 函数 | [ ] |

```python
# app/core/game_state.py
from app.core.push import PushState, PushConfig, PushHistory

class GameState:
    """游戏状态（简化版）"""

    def __init__(self):
        self.push_config = PushConfig()
        self.push_history: dict[str, PushHistory] = {}  # check_id -> history

    def can_push_check(self, check_id: str) -> bool:
        """检查是否可以对指定检定推骰"""
        if check_id not in self.push_history:
            return True  # 还未推过

        history = self.push_history[check_id]
        return history.push_count < self.push_config.max_pushes

    def push_check(
        self,
        check_id: str,
        roll_result: DiceResult,
        success_result: SuccessResult
    ) -> PushRoll:
        """对检定执行推骰"""
        if check_id not in self.push_history:
            self.push_history[check_id] = PushHistory()

        history = self.push_history[check_id]

        # 执行推骰
        push_roll = execute_push(roll_result, success_result, self.push_config)

        # 评估结果
        if push_roll.pushed_result:
            push_roll.pushed_result = evaluate_push_result(
                success_result,
                push_roll.pushed_result
            )

        history.record_push(push_roll)
        return push_roll

    def get_push_info(self, check_id: str) -> dict:
        """获取推骰信息"""
        if check_id not in self.push_history:
            return {"can_push": True, "history": None}

        history = self.push_history[check_id]
        return {
            "can_push": self.can_push_check(check_id),
            "history": history.get_summary()
        }
```

---

### 1.5 编写单元测试 (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-061-15 | [ ] 创建 `tests/test_push.py` | [ ] |
| M1-061-16 | [ ] 测试不能推骰的情况 | [ ] |
| M1-061-17 | [ ] 测试推骰改进结果 | [ ] |
| M1-061-18 | [ ] 测试推骰历史记录 | [ ] |
| M1-061-19 | [ ] 测试最大推骰次数 | [ ] |
| M1-061-20 | [ ] 测试边界情况 | [ ] |

```python
# tests/test_push.py
import pytest
from app.core.push import (
    PushConfig, PushState, can_push, execute_push, evaluate_push_result
)
from app.core.success import SuccessLevel, SuccessResult

class TestPushMechanic:
    def test_can_push_failure(self):
        """测试失败时可以推骰"""
        result = SuccessResult(
            level=SuccessLevel.FAILURE,
            roll_value=75,
            target_value=50,
            is_critical=False,
            is_fumble=False
        )
        assert can_push(result, PushConfig()) is True

    def test_cannot_push_critical(self):
        """测试大成功时不能推骰"""
        result = SuccessResult(
            level=SuccessLevel.CRITICAL_SUCCESS,
            roll_value=5,
            target_value=50,
            is_critical=True,
            is_fumble=False
        )
        assert can_push(result, PushConfig()) is False

    def test_cannot_push_fumble(self):
        """测试大失败时不能推骰"""
        result = SuccessResult(
            level=SuccessLevel.FUMBLE,
            roll_value=97,
            target_value=50,
            is_critical=False,
            is_fumble=True
        )
        assert can_push(result, PushConfig()) is False

    def test_push_improves_failure_to_success(self):
        """测试推骰将失败改进为成功"""
        original = SuccessResult(
            level=SuccessLevel.FAILURE,
            roll_value=75,
            target_value=50,
            is_critical=False,
            is_fumble=False
        )
        pushed = SuccessResult(
            level=SuccessLevel.REGULAR_SUCCESS,
            roll_value=30,
            target_value=50,
            is_critical=False,
            is_fumble=False
        )

        result = evaluate_push_result(original, pushed)
        assert result.level == SuccessLevel.REGULAR_SUCCESS

    def test_push_keeps_original_on_failure(self):
        """测试推骰仍失败时保持原结果"""
        original = SuccessResult(
            level=SuccessLevel.FAILURE,
            roll_value=60,
            target_value=50,
            is_critical=False,
            is_fumble=False
        )
        pushed = SuccessResult(
            level=SuccessLevel.FAILURE,
            roll_value=80,
            target_value=50,
            is_critical=False,
            is_fumble=False
        )

        result = evaluate_push_result(original, pushed)
        assert result.level == SuccessLevel.FAILURE
```

---

## 验收标准

- [ ] 失败时可以推骰，大成功/大失败不能推骰
- [ ] 推骰后取两个结果中较好的一个
- [ ] 推骰次数可配置（默认1次）
- [ ] 推骰历史可追溯
- [ ] 单元测试覆盖率 > 90%

---

## 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `app/core/push.py` | 创建 | 推骰模块 |
| `app/core/game_state.py` | 修改 | 集成推骰状态 |
| `tests/test_push.py` | 创建 | 单元测试 |

---

## CoC 7e 规则参考

| 场景 | 能否推骰 | 说明 |
|------|----------|------|
| 大成功 | 否 | 已经是最好结果 |
| 极难/困难/普通成功 | 否 | 已成功 |
| 失败 | 是 | 推骰争取成功 |
| 大失败 | 否 | 不能再糟了 |
