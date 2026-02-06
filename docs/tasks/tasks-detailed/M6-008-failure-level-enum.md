# M6-008: 定义失败等级枚举

**任务ID**: M6-008
**标题**: 定义失败等级枚举
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M0

---

## 任务描述

定义失败的等级系统，将失败分类为不同级别，为失败前进机制提供基础。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-008-01 | 设计失败等级体系 | 等级划分 | 30min |
| M6-008-02 | 实现枚举定义 | 类型定义 | 25min |
| M6-008-03 | 实现失败后果模型 | 后果定义 | 30min |
| M6-008-04 | 实现失败触发器 | 触发条件 | 25min |
| M6-008-05 | 编写文档和示例 | 使用说明 | 10min |

---

## 失败等级枚举

```python
# app/core/types/failure.py
from enum import Enum
from typing import Optional, List, Dict, Any
from dataclasses import dataclass

class FailureLevel(str, Enum):
    """失败等级"""
    CRITICAL = "critical"    # 严重失败: 无法继续，需付出代价才能前进
    MAJOR = "major"          # 重大失败: 受到惩罚，但可继续
    MINOR = "minor"          # 轻微失败: 小挫折，影响有限
    SOFT = "soft"            # 软失败: 失败但有意外收获
    NONE = "none"            # 非失败: 未成功但无负面后果

class FailureCategory(str, Enum):
    """失败类别"""
    SKILL_CHECK = "skill_check"     # 技能检定失败
    COMBAT = "combat"               # 战斗失败
    SOCIAL = "social"               # 社交失败
    INVESTIGATION = "investigation" # 调查失败
    SANITY = "sanity"               # 理智检定失败
    LUCK = "luck"                   # 运气检定失败
    TIME = "time"                   # 时间压力失败
    RESOURCE = "resource"           # 资源不足

@dataclass
class FailureConsequence:
    """失败后果"""
    level: FailureLevel
    category: FailureCategory

    # 后果描述
    description: str
    narrative: str          # 叙述性描述

    # 具体影响
    costs: Dict[str, Any]   # 代价
    penalties: List[str]    # 惩罚
    alternatives: List[str] # 替代选项

    # 恢复方式
    recovery: Optional[Dict[str, Any]] = None

    # 前进选项
    forward_options: List[Dict[str, Any]] = None
```

---

## 失败等级定义

```python
# app/core/failure/levels.py
from typing import Dict, Any

class FailureLevels:
    """失败等级定义"""

    CRITICAL = {
        'level': FailureLevel.CRITICAL,
        'name': '严重失败',
        'description': '行动完全失败，必须付出显著代价才能继续',
        'typical_costs': {
            'time_loss': '1d4 小时',
            'resource_loss': '1d3 耐力/魔力',
            'injury_chance': 0.5,
        },
        'narrative_impact': '需要重新规划，可能改变场景',
        'forward_options': [
            '等待并重试（需消耗资源）',
            '寻找替代方案（需调查）',
            '接受代价继续（需检定）',
        ],
        'examples': [
            '攀爬时严重坠落，受伤且无法到达目标',
            '审问失败，NPC 变得敌对',
            '潜行彻底失败，触发警报',
        ],
    }

    MAJOR = {
        'level': FailureLevel.MAJOR,
        'name': '重大失败',
        'description': '行动失败，受到惩罚但可继续',
        'typical_costs': {
            'time_loss': '30 分钟 - 1 小时',
            'resource_loss': '1 耐力/魔力',
            'injury_chance': 0.2,
        },
        'narrative_impact': '造成小障碍，需要调整计划',
        'forward_options': [
            '直接重试（需消耗资源）',
            '寻找临时解决方案',
            '接受惩罚继续',
        ],
        'examples': [
            '攀爬滑落，擦伤但可继续',
            '说服失败，NPC 态度变差',
            '潜行失误，敌人警觉',
        ],
    }

    MINOR = {
        'level': FailureLevel.MINOR,
        'name': '轻微失败',
        'description': '行动未成功，但影响有限',
        'typical_costs': {
            'time_loss': '10-30 分钟',
            'resource_loss': '轻微',
            'injury_chance': 0.0,
        },
        'narrative_impact': '小挫折，不影响大局',
        'forward_options': [
            '直接重试',
            '忽略此障碍继续',
            '寻找简单替代',
        ],
        'examples': [
            '攀爬费力，需要休息',
            '说服未果，但 NPC 仍友好',
            '潜行小失误，未被发现',
        ],
    }

    SOFT = {
        'level': FailureLevel.SOFT,
        'name': '软失败',
        'description': '表面失败但有意外收获',
        'typical_costs': {
            'time_loss': '无',
            'resource_loss': '无',
            'injury_chance': 0.0,
        },
        'narrative_impact': '虽失败但获得新信息',
        'forward_options': [
            '利用意外收获',
            '原计划调整后继续',
        ],
        'examples': [
            '攀爬失败但发现隐藏路径',
            '审问失败但得知其他线索',
            '潜行被逐但摸清守卫规律',
        ],
    }

    @classmethod
    def get_level_config(cls, level: FailureLevel) -> Dict[str, Any]:
        """获取等级配置"""
        return {
            FailureLevel.CRITICAL: cls.CRITICAL,
            FailureLevel.MAJOR: cls.MAJOR,
            FailureLevel.MINOR: cls.MINOR,
            FailureLevel.SOFT: cls.SOFT,
            FailureLevel.NONE: {},
        }.get(level, {})

    @classmethod
    def get_next_lower_level(cls, level: FailureLevel) -> FailureLevel:
        """获取下一等级"""
        order = [
            FailureLevel.CRITICAL,
            FailureLevel.MAJOR,
            FailureLevel.MINOR,
            FailureLevel.SOFT,
            FailureLevel.NONE,
        ]
        try:
            index = order.index(level)
            if index < len(order) - 1:
                return order[index + 1]
        except ValueError:
            pass
        return FailureLevel.MINOR
```

---

## 失败后果模型

```python
# app/core/failure/consequences.py
from typing import List, Dict, Any, Optional

class FailureConsequences:
    """失败后果生成器"""

    @staticmethod
    def generate_consequence(
        action: Dict[str, Any],
        failure_level: FailureLevel,
        context: Dict[str, Any]
    ) -> FailureConsequence:
        """生成失败后果"""
        config = FailureLevels.get_level_config(failure_level)

        # 基础描述
        description = config.get('description', '')
        narrative = FailureConsequences._generate_narrative(
            action,
            failure_level,
            context
        )

        # 计算代价
        costs = FailureConsequences._calculate_costs(
            action,
            failure_level,
            context
        )

        # 生成惩罚
        penalties = FailureConsequences._generate_penalties(
            action,
            failure_level
        )

        # 生成前进选项
        forward_options = config.get('forward_options', [])

        return FailureConsequence(
            level=failure_level,
            category=FailureCategory(action.get('category', 'skill_check')),
            description=description,
            narrative=narrative,
            costs=costs,
            penalties=penalties,
            alternatives=forward_options,
        )

    @staticmethod
    def _generate_narrative(
        action: Dict[str, Any],
        level: FailureLevel,
        context: Dict[str, Any]
    ) -> str:
        """生成叙述性描述"""
        action_type = action.get('type', '')
        actor = context.get('actor', '你')

        templates = {
            FailureLevel.CRITICAL: [
                f'{actor}试图{action_type}，但情况完全失控。',
                f'{actor}在{action_type}时犯了一个严重的错误。',
            ],
            FailureLevel.MAJOR: [
                f'{actor}的{action_type}没有成功，造成了麻烦。',
                f'{actor}在{action_type}时遇到了意外挫折。',
            ],
            FailureLevel.MINOR: [
                f'{actor}的{action_type}不太顺利。',
                f'{actor}在{action_type}时遇到了小困难。',
            ],
            FailureLevel.SOFT: [
                f'{actor}虽然{action_type}失败了，但意外发现了其他东西。',
                f'{actor}的{action_type}虽未成功，但获得了意外收获。',
            ],
        }

        import random
        return random.choice(templates.get(level, ['']))

    @staticmethod
    def _calculate_costs(
        action: Dict[str, Any],
        level: FailureLevel,
        context: Dict[str, Any]
    ) -> Dict[str, Any]:
        """计算代价"""
        config = FailureLevels.get_level_config(level)
        base_costs = config.get('typical_costs', {})

        costs = {
            'time': base_costs.get('time_loss', '无'),
            'resources': base_costs.get('resource_loss', '无'),
            'health': base_costs.get('injury_chance', 0),
        }

        # 根据行动类型调整
        if action.get('type') == 'combat':
            costs['health'] = min(1.0, costs['health'] + 0.3)

        return costs

    @staticmethod
    def _generate_penalties(
        action: Dict[str, Any],
        level: FailureLevel
    ) -> List[str]:
        """生成惩罚列表"""
        penalties = []

        if level == FailureLevel.CRITICAL:
            penalties.extend([
                '失去当前回合',
                '所有检定 -10',
                '可能受伤',
            ])
        elif level == FailureLevel.MAJOR:
            penalties.extend([
                '当前回合 -5',
                '相关检定 -5',
            ])
        elif level == FailureLevel.MINOR:
            penalties.append('轻微延误')

        return penalties
```

---

## 失败等级判定

```python
# app/core/failure/determination.py
from typing import Dict, Any

class FailureLevelDeterminer:
    """失败等级判定器"""

    @staticmethod
    def determine_level(
        action: Dict[str, Any],
        roll_result: int,
        difficulty: int,
        context: Dict[str, Any]
    ) -> FailureLevel:
        """判定失败等级"""
        # 计算失败程度
        failure_margin = difficulty - roll_result

        # 基础等级判定
        if failure_margin >= 30:
            base_level = FailureLevel.CRITICAL
        elif failure_margin >= 20:
            base_level = FailureLevel.MAJOR
        elif failure_margin >= 10:
            base_level = FailureLevel.MINOR
        else:
            base_level = FailureLevel.SOFT

        # 根据情况调整
        adjusted_level = FailureLevelDeterminer._adjust_for_context(
            base_level,
            action,
            context
        )

        return adjusted_level

    @staticmethod
    def _adjust_for_context(
        base_level: FailureLevel,
        action: Dict[str, Any],
        context: Dict[str, Any]
    ) -> FailureLevel:
        """根据上下文调整等级"""
        # 危险情况提升等级
        if context.get('is_dangerous'):
            if base_level in (FailureLevel.SOFT, FailureLevel.MINOR):
                return FailureLevel.MINOR
            elif base_level == FailureLevel.MAJOR:
                return FailureLevel.CRITICAL

        # 有准备降低等级
        if context.get('is_prepared'):
            if base_level == FailureLevel.CRITICAL:
                return FailureLevel.MAJOR
            elif base_level == FailureLevel.MAJOR:
                return FailureLevel.MINOR

        # 有协助降低等级
        if context.get('has_assistance'):
            if base_level in (FailureLevel.CRITICAL, FailureLevel.MAJOR):
                return FailureLevels.get_next_lower_level(base_level)

        return base_level
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/core/types/failure.py` | 创建 | 失败类型定义 |
| `app/core/failure/levels.py` | 创建 | 失败等级配置 |
| `app/core/failure/consequences.py` | 创建 | 失败后果 |
| `app/core/failure/determination.py` | 创建 | 等级判定 |
| `docs/failure/levels.md` | 创建 | 文档 |

---

## 验收标准

- [ ] 失败等级定义清晰
- [ ] 各等级区别明显
- [ ] 后果模型完整
- [ ] 判定逻辑合理
- [ ] 文档完整

---

## 参考文档

- CoC 7th Edition Rulebook
- M6-009: 实现失败前进算法

---

**最后更新**: 2026-02-07
**状态**: [ ] 待开始
