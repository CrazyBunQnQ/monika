# M6-003: 实现 Leads 优先级排序

**任务ID**: M6-003
**标题**: 实现 Leads 优先级排序
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M6-002

---

## 任务描述

实现 Leads 优先级排序系统，根据多个维度计算优先级，确保最相关的行动排在前面。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-003-01 | 设计优先级计算模型 | 多因素评分模型 | 25min |
| M6-003-02 | 实现优先级计算器 | 核心计算逻辑 | 35min |
| M6-003-03 | 实现优先级规则引擎 | 可配置规则 | 25min |
| M6-003-04 | 实现动态调整 | 根据玩家状态调整 | 20min |
| M6-003-05 | 编写单元测试 | 测试各种场景 | 15min |

---

## 优先级计算模型

```python
# app/services/leads/priority.py
from typing import List, Dict, Any
from app.core.types.leads import LeadItem, GameContext

class PriorityCalculator:
    """优先级计算器"""

    def __init__(self, rules: List[PriorityRule]):
        self.rules = rules
        self.weights = {
            'category': 0.25,
            'urgency': 0.30,
            'freshness': 0.15,
            'relevance': 0.20,
            'player_state': 0.10,
        }

    def calculate(
        self,
        lead: LeadItem,
        context: GameContext
    ) -> int:
        """
        计算 Lead 优先级

        Args:
            lead: 待计算的 Lead
            context: 游戏上下文

        Returns:
            优先级分数 (0-100)
        """
        scores = {
            'category': self._calculate_category_score(lead, context),
            'urgency': self._calculate_urgency_score(lead, context),
            'freshness': self._calculate_freshness_score(lead, context),
            'relevance': self._calculate_relevance_score(lead, context),
            'player_state': self._calculate_player_state_score(lead, context),
        }

        # 加权计算总分
        total = sum(
            scores[key] * self.weights[key]
            for key in scores
        )

        # 应用规则调整
        adjusted = self._apply_rules(lead, context, total)

        # 限制在 0-100
        return max(0, min(100, int(adjusted)))

    def _calculate_category_score(
        self,
        lead: LeadItem,
        context: GameContext
    ) -> float:
        """类别分数"""
        base_scores = {
            'combat': 90,      # 战斗最高
            'escape': 95,      # 逃生最高
            'investigate': 70, # 调查中等偏上
            'action': 60,
            'social': 50,
            'explore': 55,
            'prep': 40,        # 准备较低
        }

        # 根据场景调整
        scene = context.current_scene
        modifiers = {
            'combat': lambda: 1.2 if scene.in_combat else 0.5,
            'escape': lambda: 1.3 if scene.danger_level > 3 else 0.3,
            'social': lambda: 1.2 if scene.crowd_level == 'high' else 0.8,
        }

        base = base_scores.get(lead.category, 50)
        modifier = modifiers.get(lead.category, lambda: 1.0)()

        return base * modifier

    def _calculate_urgency_score(
        self,
        lead: LeadItem,
        context: GameContext
    ) -> float:
        """紧急度分数"""
        urgency_scores = {
            'urgent': 100,
            'high': 80,
            'medium': 50,
            'low': 20,
        }
        return urgency_scores.get(lead.urgency, 50)

    def _calculate_freshness_score(
        self,
        lead: LeadItem,
        context: GameContext
    ) -> float:
        """新鲜度分数"""
        from datetime import datetime, timedelta

        age = datetime.now() - lead.timestamp

        # 新 Lead 5 分钟内加分
        if age < timedelta(minutes=5):
            return 90
        # 5-15 分钟
        elif age < timedelta(minutes=15):
            return 70
        # 15-30 分钟
        elif age < timedelta(minutes=30):
            return 50
        # 30-60 分钟
        elif age < timedelta(hours=1):
            return 30
        # 超过 1 小时
        else:
            return 10

    def _calculate_relevance_score(
        self,
        lead: LeadItem,
        context: GameContext
    ) -> float:
        """关联度分数"""
        score = 50  # 基础分

        # 与新线索相关
        if lead.related.clues:
            new_clues = context.player.get_new_clues()
            if any(c in new_clues for c in lead.related.clues):
                score += 30

        # 与当前位置相关
        if lead.related.scene_id == context.current_scene.scene_id:
            score += 15

        # 与当前任务相关
        if context.active_quest:
            quest = context.active_quest
            if lead.related.scene_id == quest.target_scene:
                score += 20

        return min(100, score)

    def _calculate_player_state_score(
        self,
        lead: LeadItem,
        context: GameContext
    ) -> float:
        """玩家状态分数"""
        player = context.player

        # 困惑状态：提供明确指引
        if player.status == 'confused':
            if lead.action.type in ('talk', 'investigate'):
                return 80
            else:
                return 30

        # 受伤状态：优先治疗
        if player.is_injured:
            if lead.action.type == 'rest_recover':
                return 90
            elif lead.category == 'combat':
                return 30

        # 正常状态
        return 50

    def _apply_rules(
        self,
        lead: LeadItem,
        context: GameContext,
        base_score: float
    ) -> float:
        """应用优先级规则"""
        score = base_score

        for rule in self.rules:
            if rule.applies_if(lead, context):
                score = rule.adjust(score, lead, context)

        return score

    def sort_leads(
        self,
        leads: List[LeadItem],
        context: GameContext
    ) -> List[LeadItem]:
        """排序 Leads"""
        return sorted(
            leads,
            key=lambda lead: self.calculate(lead, context),
            reverse=True
        )
```

---

## 优先级规则

```python
# app/services/leads/rules.py
from typing import Callable
from app.core.types.leads import LeadItem, GameContext

class PriorityRule:
    """优先级规则"""

    def __init__(
        self,
        rule_id: str,
        name: str,
        condition: Callable[[LeadItem, GameContext], bool],
        adjustment: Callable[[float, LeadItem, GameContext], float]
    ):
        self.rule_id = rule_id
        self.name = name
        self.condition = condition
        self.adjustment = adjustment

    def applies_if(self, lead: LeadItem, context: GameContext) -> bool:
        """判断规则是否适用"""
        return self.condition(lead, context)

# 预定义规则

class PriorityRules:
    """优先级规则集合"""

    @staticmethod
    def new_player_bonus() -> PriorityRule:
        """新玩家引导"""
        return PriorityRule(
            rule_id='new_player_bonus',
            name='新玩家引导',
            condition=lambda l, c: c.player.is_new,
            adjustment=lambda score, l, c: score + 20
        )

    @staticmethod
    def deadline_urgent() -> PriorityRule:
        """截止日期紧迫"""
        return PriorityRule(
            rule_id='deadline_urgent',
            name='截止日期紧迫',
            condition=lambda l, c: (
                c.active_quest and
                c.active_quest.deadline and
                c.active_quest.time_remaining < timedelta(hours=1)
            ),
            adjustment=lambda score, l, c: (
                score + 30 if l.related.quest_id == c.active_quest.quest_id
                else score - 10
            )
        )

    @staticmethod
    def repeated_action_penalty() -> PriorityRule:
        """重复行动降权"""
        return PriorityRule(
            rule_id='repeated_action_penalty',
            name='重复行动降权',
            condition=lambda l, c: (
                c.player.get_action_count(l.action.type) > 3
            ),
            adjustment=lambda score, l, c: score * 0.7
        )

    @staticmethod
    def failed_action_recovery() -> PriorityRule:
        """失败后替代方案提升"""
        return PriorityRule(
            rule_id='failed_action_recovery',
            name='失败后替代方案',
            condition=lambda l, c: (
                c.player.last_failed_action and
                l.is_alternative_to(c.player.last_failed_action)
            ),
            adjustment=lambda score, l, c: score + 25
        )

    @staticmethod
    def low_sanity_warning() -> PriorityRule:
        """低理智值警告"""
        return PriorityRule(
            rule_id='low_sanilty_warning',
            name='低理智值警告',
            condition=lambda l, c: c.player.sanity < 20,
            adjustment=lambda score, l, c: (
                score + 30 if l.action.type == 'rest_recover'
                else score - 10
            )
        )
```

---

## 优先级配置

```python
# app/services/leads/config.py
from typing import Dict, Any

class PriorityConfig:
    """优先级配置"""

    # 类别基础分
    CATEGORY_BASE_SCORES: Dict[str, int] = {
        'combat': 90,
        'escape': 95,
        'investigate': 70,
        'action': 60,
        'social': 50,
        'explore': 55,
        'prep': 40,
    }

    # 紧急度分数
    URGENCY_SCORES: Dict[str, int] = {
        'urgent': 100,
        'high': 80,
        'medium': 50,
        'low': 20,
    }

    # 权重配置
    WEIGHTS: Dict[str, float] = {
        'category': 0.25,
        'urgency': 0.30,
        'freshness': 0.15,
        'relevance': 0.20,
        'player_state': 0.10,
    }

    # 新鲜度阈值
    FRESHNESS_THRESHOLDS: Dict[str, int] = {
        'very_fresh': 5,      # 分钟
        'fresh': 15,
        'normal': 30,
        'stale': 60,
    }

    @classmethod
    def get_category_score(cls, category: str) -> int:
        """获取类别基础分"""
        return cls.CATEGORY_BASE_SCORES.get(category, 50)

    @classmethod
    def get_urgency_score(cls, urgency: str) -> int:
        """获取紧急度分数"""
        return cls.URGENCY_SCORES.get(urgency, 50)

    @classmethod
    def validate_weights(cls) -> bool:
        """验证权重配置"""
        total = sum(cls.WEIGHTS.values())
        return abs(total - 1.0) < 0.01
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/leads/priority.py` | 创建 | 优先级计算器 |
| `app/services/leads/rules.py` | 创建 | 优先级规则 |
| `app/services/leads/config.py` | 创建 | 配置文件 |
| `tests/services/leads/test_priority.py` | 创建 | 单元测试 |

---

## 验收标准

- [ ] 优先级计算正确
- [ ] 多因素加权合理
- [ ] 规则引擎可配置
- [ ] 动态调整有效
- [ ] 排序结果符合预期
- [ ] 单元测试通过

---

## 参考文档

- M6-001: Leads 数据结构
- M6-002: Leads 生成算法

---

**最后更新**: 2026-02-07
**状态**: [ ] 待开始
