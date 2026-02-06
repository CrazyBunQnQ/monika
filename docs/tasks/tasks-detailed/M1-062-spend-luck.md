# M1-062: 实现花幸运机制

**任务类型**: backend
**预估工时**: 2.5h
**依赖**: M1-057, M1-058
**状态**: [ ]

---

## 子任务拆解

### 1.1 定义幸运系统 (25min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-062-01 | [ ] 创建 `app/core/luck.py` | [ ] |
| M1-062-02 | [ ] 定义 `LuckSpendType` 枚举 | [ ] |
| M1-062-03 | [ ] 定义 `LuckTransaction` 数据类 | [ ] |
| M1-062-04 | [ ] 定义 `LuckRecord` 数据类 | [ ] |

```python
# app/core/luck.py
from enum import Enum
from dataclasses import dataclass, field
from typing import Optional
from datetime import datetime

class LuckSpendType(Enum):
    """幸运消耗类型"""
    ROLL_IMPROVEMENT = "roll_improvement"  # 掷骰改进
    AVOID_DAMAGE = "avoid_damage"         # 避免伤害
    RECALL_KNOWLEDGE = "recall_knowledge" # 回忆知识
    INFLUENCE_NPC = "influence_npc"        # 影响 NPC
    OTHER = "other"                        # 其他

@dataclass
class LuckTransaction:
    """幸运交易记录"""
    amount: int                   # 负数为消耗，正数为恢复
    transaction_type: LuckSpendType
    reason: str
    timestamp: datetime = field(default_factory=datetime.utcnow)
    character_id: Optional[str] = None

@dataclass
class LuckRecord:
    """幸运使用记录"""
    character_id: str
    current_luck: int
    max_luck: int                # 最大幸运值
    luck_bonus: int = 0          # 幸运加成
    transactions: list[LuckTransaction] = field(default_factory=list)

    @property
    def luck_percentage(self) -> float:
        """幸运百分比"""
        if self.max_luck <= 0:
            return 0.0
        return (self.current_luck / self.max_luck) * 100
```

---

### 1.2 实现幸运管理 (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-062-05 | [ ] 实现 `spend_luck()` 函数 | [ ] |
| M1-062-06 | [ ] 实现 `regain_luck()` 函数 | [ ] |
| M1-062-07 | [ ] 实现 `get_luck_balance()` 函数 | [ ] |
| M1-062-08 | [ ] 实现 `can_spend()` 函数 | [ ] |

```python
class LuckManager:
    """幸运值管理器"""

    def __init__(self, record: LuckRecord):
        self.record = record

    def spend_luck(
        self,
        amount: int,
        spend_type: LuckSpendType,
        reason: str
    ) -> tuple[bool, str]:
        """
        消耗幸运值

        Args:
            amount: 消耗数量
            spend_type: 消耗类型
            reason: 消耗原因

        Returns:
            tuple: (是否成功, 消息)
        """
        if amount <= 0:
            return False, "消耗数量必须大于0"

        if self.record.current_luck < amount:
            remaining = self.record.current_luck
            return False, f"幸运值不足，需要 {amount} 点，当前剩余 {remaining} 点"

        # 执行消耗
        self.record.current_luck -= amount

        # 记录交易
        transaction = LuckTransaction(
            amount=-amount,
            transaction_type=spend_type,
            reason=reason,
            character_id=self.record.character_id
        )
        self.record.transactions.append(transaction)

        return True, f"成功消耗 {amount} 点幸运，剩余 {self.record.current_luck} 点"

    def regain_luck(
        self,
        amount: int,
        reason: str,
        max_override: Optional[int] = None
    ) -> int:
        """
        恢复幸运值

        Args:
            amount: 恢复数量
            reason: 恢复原因
            max_override: 最大值覆盖（可选）

        Returns:
            int: 实际恢复的数量
        """
        max_luck = max_override or self.record.max_luck
        available_space = max_luck - self.record.current_luck

        actual_regain = min(amount, available_space)
        if actual_regain <= 0:
            return 0

        self.record.current_luck += actual_regain

        # 记录交易
        transaction = LuckTransaction(
            amount=actual_regain,
            transaction_type=LuckSpendType.OTHER,
            reason=reason,
            character_id=self.record.character_id
        )
        self.record.transactions.append(transaction)

        return actual_regain

    def can_spend(self, amount: int) -> bool:
        """检查是否可以消耗指定数量的幸运"""
        return self.record.current_luck >= amount

    def get_balance(self) -> int:
        """获取当前幸运值"""
        return self.record.current_luck

    def get_usage_summary(self) -> dict:
        """获取幸运使用摘要"""
        summary = {
            "current": self.record.current_luck,
            "max": self.record.max_luck,
            "percentage": self.record.luck_percentage,
            "total_spent": 0,
            "total_regained": 0,
            "by_type": {t.value: 0 for t in LuckSpendType}
        }

        for t in self.record.transactions:
            if t.amount < 0:
                summary["total_spent"] += abs(t.amount)
                summary["by_type"][t.transaction_type.value] += abs(t.amount)
            else:
                summary["total_regained"] += t.amount

        return summary
```

---

### 1.3 实现掷骰幸运改进 (35min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-062-09 | [ ] 定义 `LuckImprovementConfig` 数据类 | [ ] |
| M1-062-10 | [ ] 实现 `apply_luck_to_roll()` 函数 | [ ] |
| M1-062-11 | [ ] 实现 `calculate_luck_cost()` 函数 | [ ] |

```python
@dataclass
class LuckImprovementConfig:
    """幸运改进配置"""
    improvement_type: str = "d10"      # d10 改进
    luck_cost_per_d10: int = 1        # 每d10消耗1点幸运
    max_improvement: int = 0          # 最大改进次数 (0=无限制)

@dataclass
class LuckImprovedResult:
    """幸运改进结果"""
    original_roll: int
    improved_roll: Optional[int]
    luck_spent: int
    improvement_used: bool
    success_before: Optional[str] = None
    success_after: Optional[str] = None
    message: str = ""


def calculate_luck_cost(
    improvement_count: int,
    config: LuckImprovementConfig
) -> int:
    """计算幸运消耗"""
    return improvement_count * config.luck_cost_per_d10


def apply_luck_to_roll(
    original_roll: int,
    target_value: int,
    luck_spent: int,
    config: LuckImprovementConfig
) -> LuckImprovedResult:
    """
    将幸运应用于改进掷骰结果

    规则:
    - 每消耗 1 点幸运，可以掷 1d10
    - 取 d10 和原始结果中的较低值
    - 可以多次改进

    Args:
        original_roll: 原始掷骰结果
        target_value: 目标值（用于成功判定）
        luck_spent: 消耗的幸运点数
        config: 改进配置

    Returns:
        LuckImprovedResult: 改进结果
    """
    if luck_spent <= 0:
        return LuckImprovedResult(
            original_roll=original_roll,
            improved_roll=None,
            luck_spent=0,
            improvement_used=False,
            message="未消耗幸运"
        )

    # 计算可以改进的次数
    max_possible = luck_spent
    if config.max_improvement > 0:
        max_possible = min(max_possible, config.max_improvement)

    # 掷出改进骰
    improvement_rolls = [
        random.randint(1, 10)
        for _ in range(max_possible)
    ]

    # 取所有改进骰和原始值中的最小值
    improved_value = original_roll
    for roll in improvement_rolls:
        improved_value = min(improved_value, roll)

    improvement_used = improved_value < original_roll

    # 计算成功等级变化
    from app.core.success import calculate_success_level, SuccessLevel

    success_before = calculate_success_level(original_roll, target_value)
    success_after = calculate_success_level(improved_value, target_value)

    # 生成消息
    if improvement_used:
        message = (
            f"消耗 {luck_spent} 点幸运进行改进！\n"
            f"原始掷骰: {original_roll} ({success_before.level.value})\n"
            f"改进结果: {improved_value} ({success_after.level.value})\n"
            f"改进骰: {improvement_rolls}"
        )
    else:
        message = (
            f"消耗 {luck_spent} 点幸运，但改进骰未产生更好的结果\n"
            f"原始掷骰: {original_roll}\n"
            f"改进骰: {improvement_rolls}"
        )

    return LuckImprovedResult(
        original_roll=original_roll,
        improved_roll=improved_value if improvement_used else None,
        luck_spent=luck_spent if improvement_used else 0,
        improvement_used=improvement_used,
        success_before=success_before.level.value,
        success_after=success_after.level.value if improvement_used else None,
        message=message
    )
```

---

### 1.4 实现幸运恢复规则 (25min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-062-12 | [ ] 实现 `rest_recover_luck()` 函数 | [ ] |
| M1-062-13 | [ ] 实现 `event_recover_luck()` 函数 | [ ] |
| M1-062-14 | [ ] 定义 `LuckRecoveryRule` | [ ] |

```python
@dataclass
class LuckRecoveryRule:
    """幸运恢复规则"""
    short_rest_amount: int = 1          # 短休息恢复
    long_rest_percentage: float = 0.2   # 长休息恢复百分比
    event_bonus: int = 0                # 事件奖励
    max_recovery_per_day: int = 5       # 每日最大恢复


class LuckRecoveryManager:
    """幸运恢复管理器"""

    def __init__(self, record: LuckRecord, rule: LuckRecoveryRule):
        self.record = record
        self.rule = rule

    def short_rest(self) -> int:
        """短休息恢复"""
        if self.rule.short_rest_amount <= 0:
            return 0

        regained = self.regain_luck(
            self.rule.short_rest_amount,
            "短休息恢复"
        )

        return regained

    def long_rest(self) -> int:
        """长休息恢复（恢复百分比）"""
        if self.rule.long_rest_percentage <= 0:
            return 0

        amount_to_recover = int(self.record.max_luck * self.rule.long_rest_percentage)

        # 检查是否超过每日限制
        today_recovery = self._get_today_recovery()
        remaining = self.rule.max_recovery_per_day - today_recovery

        if remaining <= 0:
            return 0

        actual_recovery = min(amount_to_recover, remaining)

        regained = self.regain_luck(
            actual_recovery,
            "长休息恢复"
        )

        return regained

    def event_bonus(self, amount: int, reason: str) -> int:
        """事件奖励恢复"""
        if amount <= 0:
            return 0

        return self.regain_luck(amount, f"事件奖励: {reason}")

    def _get_today_recovery(self) -> int:
        """获取今日已恢复的幸运值"""
        today = datetime.now().date()
        today_transactions = [
            t for t in self.record.transactions
            if t.timestamp.date() == today and t.amount > 0
        ]
        return sum(t.amount for t in today_transactions)

    def get_recovery_info(self) -> dict:
        """获取恢复信息"""
        today_recovery = self._get_today_recovery()
        remaining_daily = self.rule.max_recovery_per_day - today_recovery

        return {
            "current_luck": self.record.current_luck,
            "max_luck": self.record.max_luck,
            "today_recovered": today_recovery,
            "daily_limit": self.rule.max_recovery_per_day,
            "remaining_daily": max(0, remaining_daily),
            "short_rest_recovery": self.rule.short_rest_amount,
            "long_rest_percentage": f"{self.rule.long_rest_percentage * 100}%"
        }
```

---

### 1.5 编写单元测试 (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-062-15 | [ ] 创建 `tests/test_luck.py` | [ ] |
| M1-062-16 | [ ] 测试幸运消耗 | [ ] |
| M1-062-17 | [ ] 测试幸运恢复 | [ ] |
| M1-062-18 | [ ] 测试掷骰幸运改进 | [ ] |

```python
# tests/test_luck.py
import pytest
from app.core.luck import (
    LuckRecord, LuckManager, LuckSpendType,
    LuckImprovementConfig, apply_luck_to_roll
)

class TestLuckManagement:
    def setup_method(self):
        """设置测试数据"""
        self.record = LuckRecord(
            character_id="test_char",
            current_luck=20,
            max_luck=50
        )
        self.manager = LuckManager(self.record)

    def test_spend_luck_success(self):
        """测试成功消耗幸运"""
        success, msg = self.manager.spend_luck(
            5,
            LuckSpendType.ROLL_IMPROVEMENT,
            "改进掷骰"
        )
        assert success is True
        assert self.record.current_luck == 15

    def test_spend_luck_insufficient(self):
        """测试幸运不足"""
        success, msg = self.manager.spend_luck(
            30,
            LuckSpendType.ROLL_IMPROVEMENT,
            "尝试改进"
        )
        assert success is False
        assert "不足" in msg
        assert self.record.current_luck == 20  # 未变化

    def test_regain_luck(self):
        """测试幸运恢复"""
        self.manager.spend_luck(10, LuckSpendType.ROLL_IMPROVEMENT, "测试")
        regained = self.manager.regain_luck(5, "短休息")
        assert regained == 5
        assert self.record.current_luck == 15

    def test_regain_respects_max(self):
        """测试恢复不超过最大值"""
        regained = self.manager.regain_luck(100, "大量恢复")
        assert regained == 30  # 50 - 20 = 30
        assert self.record.current_luck == 50


class TestLuckImprovement:
    def test_improve_roll_with_luck(self):
        """测试幸运改进掷骰"""
        config = LuckImprovementConfig()

        with patch('random.randint', return_value=3):
            result = apply_luck_to_roll(
                original_roll=70,
                target_value=50,
                luck_spent=1,
                config=config
            )

        assert result.improvement_used is True
        assert result.luck_spent == 1
        assert result.improved_roll == 3

    def test_no_luck_spent(self):
        """测试未消耗幸运"""
        config = LuckImprovementConfig()

        result = apply_luck_to_roll(
            original_roll=70,
            target_value=50,
            luck_spent=0,
            config=config
        )

        assert result.improvement_used is False
        assert result.luck_spent == 0
```

---

## 验收标准

- [ ] 幸运可以消耗和恢复
- [ ] 幸运不足时正确拒绝
- [ ] 幸运改进掷骰取低值
- [ ] 恢复规则符合 CoC 7e
- [ ] 单元测试覆盖率 > 90%

---

## 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `app/core/luck.py` | 创建 | 幸运系统模块 |
| `tests/test_luck.py` | 创建 | 单元测试 |

---

## CoC 7e 幸运规则

| 场景 | 消耗 | 说明 |
|------|------|------|
| 掷骰改进 (1d10) | 1 点 | 改进成功判定 |
| 回忆知识 | 1-5 点 | 取决于难度 |
| 影响 NPC | 1-5 点 | 取决于难度 |
| 避免伤害 | 1-2 点 | 取决于伤害 |

| 恢复方式 | 数量 | 频率 |
|----------|------|------|
| 短休息 | 1d3 | 每场景 |
| 长休息 | 1/5 最大值 | 每天 |
| 成功故事 | 1-2 | 守密人决定 |
