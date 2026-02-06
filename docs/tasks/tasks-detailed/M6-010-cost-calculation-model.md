# M6-010: 实现代价计算模型

**任务ID**: M6-010
**标题**: 实现代价计算模型
**类型**: backend (后端开发)
**预估工时**: 4h
**依赖**: M6-009

---

## 任务描述

实现失败代价计算模型，量化失败的资源、时间和状态损失。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-010-01 | 设计代价类型体系 | 代价分类 | 30min |
| M6-010-02 | 实现代价计算器 | 核心计算逻辑 | 60min |
| M6-010-03 | 实现动态调整模型 | 根据情况调整 | 50min |
| M6-010-04 | 实现代价上限保护 | 防止过度惩罚 | 30min |
| M6-010-05 | 实现代价豁免机制 | 特殊情况 | 30min |
| M6-010-06 | 编写单元测试 | 测试覆盖 | 20min |

---

## 代价类型体系

```python
# app/core/cost/types.py
from enum import Enum
from typing import Dict, Any, Optional
from dataclasses import dataclass

class CostType(str, Enum):
    """代价类型"""
    TIME = "time"                    # 时间损失
    RESOURCE = "resource"            # 资源消耗
    HEALTH = "health"                # 生命值/耐力
    SANITY = "sanity"                # 理智值
    REPUTATION = "reputation"        # 声望
    INFORMATION = "information"      # 信息损失
    OPPORTUNITY = "opportunity"      # 机会损失
    POSITION = "position"            # 位置劣势

@dataclass
class Cost:
    """代价"""
    cost_type: CostType
    value: Any
    description: str
    recoverable: bool = True
    recovery_method: Optional[str] = None

    # 随机性
    is_random: bool = False
    random_range: Optional[tuple] = None

@dataclass
class CostPackage:
    """代价包"""
    costs: list[Cost]
    total_weight: float  # 总权重 (0-1)
    narrative: str

    # 条件
    applies_if: Optional[callable] = None
```

---

## 代价计算器

```python
# app/services/cost/calculator.py
from typing import Dict, Any, List
from app.core.cost.types import Cost, CostPackage, CostType
from app.core.types.failure import FailureLevel

class CostCalculator:
    """代价计算器"""

    def __init__(self):
        self.base_costs = self._initialize_base_costs()
        self.modifiers = self._initialize_modifiers()

    async def calculate_cost(
        self,
        action: Dict[str, Any],
        failure_level: FailureLevel,
        context: 'GameContext'
    ) -> CostPackage:
        """计算失败代价"""
        # 1. 获取基础代价
        base_costs = self._get_base_costs(failure_level)

        # 2. 应用修正
        modified_costs = self._apply_modifiers(
            base_costs,
            action,
            context
        )

        # 3. 计算总权重
        total_weight = self._calculate_total_weight(modified_costs)

        # 4. 生成叙述
        narrative = self._generate_narrative(modified_costs, action)

        return CostPackage(
            costs=modified_costs,
            total_weight=total_weight,
            narrative=narrative
        )

    def _initialize_base_costs(self) -> Dict[FailureLevel, List[Cost]]:
        """初始化基础代价"""
        return {
            FailureLevel.CRITICAL: [
                Cost(
                    cost_type=CostType.TIME,
                    value='1d4 hours',
                    description='大量时间损失',
                    recoverable=False,
                ),
                Cost(
                    cost_type=CostType.HEALTH,
                    value='2d6',
                    description='可能受伤',
                    recoverable=True,
                    recovery_method='医疗检定或休息',
                ),
                Cost(
                    cost_type=CostType.POSITION,
                    value='disadvantaged',
                    description='位置劣势',
                    recoverable=True,
                    recovery_method='重新定位',
                ),
            ],
            FailureLevel.MAJOR: [
                Cost(
                    cost_type=CostType.TIME,
                    value='30min - 1 hour',
                    description='时间损失',
                ),
                Cost(
                    cost_type=CostType.RESOURCE,
                    value='1d3',
                    description='资源消耗',
                ),
            ],
            FailureLevel.MINOR: [
                Cost(
                    cost_type=CostType.TIME,
                    value='10-30min',
                    description='轻微延误',
                ),
            ],
            FailureLevel.SOFT: [
                Cost(
                    cost_type=CostType.OPPORTUNITY,
                    value='minor',
                    description='小机会损失',
                ),
            ],
        }

    def _initialize_modifiers(self) -> Dict[str, Any]:
        """初始化修正因子"""
        return {
            'prepared': 0.5,       # 有准备: 代价减半
            'assisted': 0.7,       # 有协助: 代价70%
            'dangerous': 1.5,      # 危险环境: 代价增加50%
            'rushed': 1.3,         # 匆忙: 代价增加30%
            'desperate': 2.0,      # 绝境: 代价翻倍
        }

    def _get_base_costs(self, failure_level: FailureLevel) -> List[Cost]:
        """获取基础代价"""
        return self.base_costs.get(failure_level, [])

    def _apply_modifiers(
        self,
        base_costs: List[Cost],
        action: Dict[str, Any],
        context: 'GameContext'
    ) -> List[Cost]:
        """应用修正因子"""
        modified = []

        for cost in base_costs:
            # 计算修正系数
            modifier = self._calculate_modifier(action, context)

            # 应用修正
            if cost.cost_type == CostType.TIME:
                modified_cost = self._modify_time_cost(cost, modifier)
            elif cost.cost_type == CostType.HEALTH:
                modified_cost = self._modify_health_cost(cost, modifier)
            else:
                modified_cost = self._modify_generic_cost(cost, modifier)

            modified.append(modified_cost)

        return modified

    def _calculate_modifier(
        self,
        action: Dict[str, Any],
        context: 'GameContext'
    ) -> float:
        """计算总修正系数"""
        modifier = 1.0

        # 检查修正条件
        if context.player.is_prepared():
            modifier *= self.modifiers['prepared']

        if context.player.has_assistance():
            modifier *= self.modifiers['assisted']

        if context.is_dangerous():
            modifier *= self.modifiers['dangerous']

        if action.get('is_rushed'):
            modifier *= self.modifiers['rushed']

        if context.player.is_desperate():
            modifier *= self.modifiers['desperate']

        # 限制修正范围
        return max(0.25, min(3.0, modifier))

    def _modify_time_cost(self, cost: Cost, modifier: float) -> Cost:
        """修改时间代价"""
        # 解析时间值
        base_value = self._parse_time_value(cost.value)

        # 应用修正
        modified_value = base_value * modifier

        return Cost(
            cost_type=cost.cost_type,
            value=self._format_time_value(modified_value),
            description=cost.description,
            recoverable=cost.recoverable,
        )

    def _modify_health_cost(self, cost: Cost, modifier: float) -> Cost:
        """修改健康代价"""
        # 解析骰子表达式
        base_value = self._parse_dice_value(cost.value)

        # 应用修正
        modified_value = max(1, int(base_value * modifier))

        return Cost(
            cost_type=cost.cost_type,
            value=f'{modified_value}d6',
            description=cost.description,
            recoverable=cost.recoverable,
            recovery_method=cost.recovery_method,
        )

    def _modify_generic_cost(self, cost: Cost, modifier: float) -> Cost:
        """修改通用代价"""
        return Cost(
            cost_type=cost.cost_type,
            value=cost.value,
            description=cost.description,
            recoverable=cost.recoverable,
        )

    def _parse_time_value(self, value: str) -> float:
        """解析时间值（分钟）"""
        value = value.lower()

        if 'hour' in value:
            # 提取数字
            import re
            match = re.search(r'(\d+(?:d\d+)?)\s*hours?', value)
            if match:
                dice_part = match.group(1)
                if 'd' in dice_part:
                    # 骰子表达式，返回期望值
                    parts = dice_part.split('d')
                    return float(parts[0]) * float(parts[1]) / 2 * 60
                else:
                    return float(dice_part) * 60

        if 'min' in value:
            match = re.search(r'(\d+(?:-\d+)?)\s*min', value)
            if match:
                range_part = match.group(1)
                if '-' in range_part:
                    low, high = map(int, range_part.split('-'))
                    return (low + high) / 2
                return float(range_part)

        return 30.0  # 默认30分钟

    def _format_time_value(self, minutes: float) -> str:
        """格式化时间值"""
        if minutes >= 60:
            hours = minutes / 60
            if hours == int(hours):
                return f'{int(hours)} hour{"s" if hours > 1 else ""}'
            return f'{hours:.1f} hours'
        return f'{int(minutes)} minutes'

    def _parse_dice_value(self, value: str) -> float:
        """解析骰子表达式的期望值"""
        import re
        match = re.match(r'(\d+)d(\d+)', value)
        if match:
            count, sides = map(int, match.groups())
            return count * (sides + 1) / 2
        return float(value) if value.replace('.', '').isdigit() else 1.0

    def _calculate_total_weight(self, costs: List[Cost]) -> float:
        """计算总权重"""
        # 简化版本：根据代价类型和数量计算
        weight = 0.0

        for cost in costs:
            if cost.cost_type == CostType.HEALTH:
                weight += 0.4
            elif cost.cost_type == CostType.TIME:
                weight += 0.3
            elif cost.cost_type == CostType.SANITY:
                weight += 0.3
            elif cost.cost_type == CostType.RESOURCE:
                weight += 0.2
            else:
                weight += 0.1

        return min(1.0, weight)

    def _generate_narrative(
        self,
        costs: List[Cost],
        action: Dict[str, Any]
    ) -> str:
        """生成代价叙述"""
        if not costs:
            return '没有显著代价。'

        parts = []
        for cost in costs:
            if cost.cost_type == CostType.TIME:
                parts.append(f'损失了{cost.value}的时间')
            elif cost.cost_type == CostType.HEALTH:
                parts.append(f'受到{cost.value}点伤害')
            elif cost.cost_type == CostType.RESOURCE:
                parts.append(f'消耗了{cost.value}点资源')

        if len(parts) == 1:
            return f'{action.get("type", "行动")}失败，{parts[0]}。'
        elif len(parts) == 2:
            return f'{action.get("type", "行动")}失败，{parts[0]}，{parts[1]}。'
        else:
            return f'{action.get("type", "行动")}失败，{", ".join(parts[:-1])}，以及{parts[-1]}。'
```

---

## 代价上限保护

```python
# app/services/cost/limiter.py
from typing import List
from app.core.cost.types import Cost, CostPackage

class CostLimiter:
    """代价限制器"""

    def __init__(self):
        self.limits = {
            CostType.HEALTH: {
                'max_per_failure': 0.5,  # 最多50%生命值
                'max_per_session': 1.0,   # 每次会话最多100%
            },
            CostType.SANITY: {
                'max_per_failure': 0.3,  # 最多30%理智值
                'max_per_session': 0.5,   # 每次会话最多50%
            },
            CostType.TIME: {
                'max_per_failure': 240,   # 最多4小时
                'max_per_session': 480,   # 每次会话最多8小时
            },
        }

        self.session_totals: dict[str, dict[CostType, float]] = {}

    def limit_cost(
        self,
        cost_package: CostPackage,
        session_id: str,
        context: 'GameContext'
    ) -> CostPackage:
        """限制代价"""
        limited_costs = []

        for cost in cost_package.costs:
            limited = self._limit_single_cost(cost, session_id, context)
            limited_costs.append(limited)

            # 更新会话总计
            self._update_session_total(session_id, limited)

        return CostPackage(
            costs=limited_costs,
            total_weight=cost_package.total_weight,
            narrative=cost_package.narrative
        )

    def _limit_single_cost(
        self,
        cost: Cost,
        session_id: str,
        context: 'GameContext'
    ) -> Cost:
        """限制单个代价"""
        if cost.cost_type not in self.limits:
            return cost

        limits = self.limits[cost.cost_type]
        session_total = self.session_totals.setdefault(
            session_id,
            {}
        ).setdefault(cost.cost_type, 0.0)

        # 解析当前值
        current_value = self._parse_cost_value(cost, cost.cost_type, context)

        # 检查单次失败上限
        max_single = self._get_max_single(cost.cost_type, context)
        if current_value > max_single:
            return self._create_limited_cost(
                cost,
                max_single,
                '达到单次失败上限'
            )

        # 检查会话累计上限
        if session_total + current_value > limits['max_per_session']:
            remaining = limits['max_per_session'] - session_total
            if remaining > 0:
                return self._create_limited_cost(
                    cost,
                    remaining,
                    '达到会话累计上限'
                )
            else:
                return self._create_zero_cost(cost)

        return cost

    def _get_max_single(self, cost_type: CostType, context: 'GameContext') -> float:
        """获取单次失败最大值"""
        limit = self.limits[cost_type]['max_per_failure']

        if cost_type == CostType.HEALTH:
            return limit * context.player.max_health
        elif cost_type == CostType.SANITY:
            return limit * context.player.max_sanity
        else:
            return limit

    def _parse_cost_value(
        self,
        cost: Cost,
        cost_type: CostType,
        context: 'GameContext'
    ) -> float:
        """解析代价值"""
        value = cost.value

        if cost_type in (CostType.HEALTH, CostType.SANITY, CostType.RESOURCE):
            # 骰子表达式，返回期望值
            return self._parse_dice_expectation(value)
        elif cost_type == CostType.TIME:
            # 时间值（分钟）
            return self._parse_time_value(value)

        return 0.0

    def _parse_dice_expectation(self, value: str) -> float:
        """解析骰子期望值"""
        import re
        match = re.match(r'(\d+)d(\d+)', value)
        if match:
            count, sides = map(int, match.groups())
            return count * (sides + 1) / 2
        return float(value) if str(value).replace('.', '').isdigit() else 1.0

    def _parse_time_value(self, value: str) -> float:
        """解析时间值"""
        import re
        value = value.lower()

        if 'hour' in value:
            match = re.search(r'(\d+(?:d\d+)?)\s*hours?', value)
            if match:
                dice_part = match.group(1)
                if 'd' in dice_part:
                    parts = dice_part.split('d')
                    return float(parts[0]) * float(parts[1]) / 2 * 60
                return float(dice_part) * 60

        if 'min' in value:
            match = re.search(r'(\d+(?:-\d+)?)', value)
            if match:
                range_part = match.group(1)
                if '-' in range_part:
                    low, high = map(int, range_part.split('-'))
                    return (low + high) / 2
                return float(range_part)

        return 30.0

    def _create_limited_cost(
        self,
        original: Cost,
        limited_value: float,
        reason: str
    ) -> Cost:
        """创建受限代价"""
        return Cost(
            cost_type=original.cost_type,
            value=str(limited_value),
            description=f'{original.description} ({reason})',
            recoverable=original.recoverable,
            recovery_method=original.recovery_method,
        )

    def _create_zero_cost(self, original: Cost) -> Cost:
        """创建零代价"""
        return Cost(
            cost_type=original.cost_type,
            value='0',
            description=f'{original.description} (已豁免)',
            recoverable=True,
        )

    def _update_session_total(self, session_id: str, cost: Cost):
        """更新会话总计"""
        if cost.cost_type in self.limits:
            value = self._parse_cost_value(
                cost,
                cost.cost_type,
                None  # context 不需要
            )
            self.session_totals[session_id][cost.cost_type] = (
                self.session_totals[session_id].get(cost.cost_type, 0.0) + value
            )

    def reset_session(self, session_id: str):
        """重置会话统计"""
        if session_id in self.session_totals:
            del self.session_totals[session_id]
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/core/cost/types.py` | 创建 | 代价类型 |
| `app/services/cost/calculator.py` | 创建 | 代价计算器 |
| `app/services/cost/limiter.py` | 创建 | 代价限制器 |
| `tests/services/cost/test_calculator.py` | 创建 | 单元测试 |

---

## 验收标准

- [ ] 代价类型完整
- [ ] 计算逻辑正确
- [ ] 修正机制有效
- [ ] 上限保护正常
- [ ] 叙述清晰
- [ ] 单元测试通过

---

## 参考文档

- M6-008: 失败等级枚举
- M6-009: 失败前进算法

---

**最后更新**: 2026-02-07
**状态**: [ ] 待开始
