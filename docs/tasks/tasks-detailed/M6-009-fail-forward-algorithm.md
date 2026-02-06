# M6-009: 实现失败前进算法

**任务ID**: M6-009
**标题**: 实现失败前进算法
**类型**: backend (后端开发)
**预估工时**: 4h
**依赖**: M6-008

---

## 任务描述

实现失败前进（Fail Forward）算法，确保失败不阻断游戏进程，而是推动故事发展。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-009-01 | 设计前进策略 | 失败如何推进故事 | 40min |
| M6-009-02 | 实现前进引擎 | 核心算法 | 70min |
| M6-009-03 | 实现新场景生成 | 失败后的新局面 | 50min |
| M6-009-04 | 实现时间线调整 | 时间流逝处理 | 30min |
| M6-009-05 | 实现叙事衔接 | 失败到前进的过渡 | 30min |
| M6-009-06 | 编写单元测试 | 测试覆盖 | 20min |

---

## 前进策略

```python
# app/services/fail_forward/strategy.py
from typing import List, Dict, Any, Optional
from enum import Enum

class ForwardStrategy(str, Enum):
    """前进策略"""
    CONTINUE_WITH_COST = "continue_with_cost"  # 付出代价继续
    ALTER_PATH = "alter_path"                  # 改变路径
    DISCOVER_ALTERNATIVE = "discover_alternative"  # 发现替代方案
    TIME_SKIP = "time_skip"                    # 时间跳跃
    CONSEQUENCE_DRIVEN = "consequence_driven"  # 后果驱动

@dataclass
class ForwardOption:
    """前进选项"""
    option_id: str
    strategy: ForwardStrategy
    description: str
    narrative: str

    # 执行要求
    requirements: Optional[Dict[str, Any]] = None

    # 后果
    consequences: List[Dict[str, Any]] = None

    # 新状态
    new_scene: Optional[str] = None
    new_leads: Optional[List[Dict]] = None
    time_advance: Optional[str] = None

class FailForwardStrategy:
    """失败前进策略"""

    def __init__(self):
        self.strategies = {
            ForwardStrategy.CONTINUE_WITH_COST: self._continue_with_cost,
            ForwardStrategy.ALTER_PATH: self._alter_path,
            ForwardStrategy.DISCOVER_ALTERNATIVE: self._discover_alternative,
            ForwardStrategy.TIME_SKIP: self._time_skip,
            ForwardStrategy.CONSEQUENCE_DRIVEN: self._consequence_driven,
        }

    async def generate_forward_options(
        self,
        failed_action: Dict[str, Any],
        failure_level: 'FailureLevel',
        context: 'GameContext'
    ) -> List[ForwardOption]:
        """生成前进选项"""
        options = []

        # 根据失败等级选择策略
        if failure_level in ('critical', 'major'):
            # 严重失败：提供多种前进选项
            options.extend([
                await self._continue_with_cost(failed_action, context),
                await self._discover_alternative(failed_action, context),
                await self._consequence_driven(failed_action, context),
            ])
        elif failure_level == 'minor':
            # 轻微失败：简单调整
            options.append(await self._alter_path(failed_action, context))
        else:
            # 软失败：直接继续
            options.append(await self._continue_with_cost(failed_action, context))

        return options

    async def _continue_with_cost(
        self,
        failed_action: Dict[str, Any],
        context: 'GameContext'
    ) -> ForwardOption:
        """付出代价继续"""
        return ForwardOption(
            option_id='continue_cost',
            strategy=ForwardStrategy.CONTINUE_WITH_COST,
            description='接受失败并付出代价继续前进',
            narrative=self._generate_narrative(failed_action, 'cost'),
            requirements={
                'pay_cost': True,
            },
            consequences=[
                {'type': 'resource_loss', 'value': '1d3'},
                {'type': 'time_loss', 'value': '30min'},
            ],
        )

    async def _alter_path(
        self,
        failed_action: Dict[str, Any],
        context: 'GameContext'
    ) -> ForwardOption:
        """改变路径"""
        return ForwardOption(
            option_id='alter_path',
            strategy=ForwardStrategy.ALTER_PATH,
            description='尝试替代方法',
            narrative=self._generate_narrative(failed_action, 'alter'),
            new_leads=self._generate_alternative_leads(failed_action, context),
        )

    async def _discover_alternative(
        self,
        failed_action: Dict[str, Any],
        context: 'GameContext'
    ) -> ForwardOption:
        """发现替代方案"""
        return ForwardOption(
            option_id='discover_alt',
            strategy=ForwardStrategy.DISCOVER_ALTERNATIVE,
            description='在失败中发现新的可能性',
            narrative=self._generate_narrative(failed_action, 'discover'),
            new_leads=self._generate_discovery_leads(failed_action, context),
            consequences=[
                {'type': 'clue_gain', 'value': 'unexpected_clue'},
            ],
        )

    async def _time_skip(
        self,
        failed_action: Dict[str, Any],
        context: 'GameContext'
    ) -> ForwardOption:
        """时间跳跃"""
        return ForwardOption(
            option_id='time_skip',
            strategy=ForwardStrategy.TIME_SKIP,
            description='经过一段时间后重新尝试',
            narrative=self._generate_narrative(failed_action, 'time'),
            time_advance='1d4 hours',
            new_leads=[{
                'title': '再次尝试',
                'description': '情况可能已经改变',
            }],
        )

    async def _consequence_driven(
        self,
        failed_action: Dict[str, Any],
        context: 'GameContext'
    ) -> ForwardOption:
        """后果驱动"""
        return ForwardOption(
            option_id='consequence',
            strategy=ForwardStrategy.CONSEQUENCE_DRIVEN,
            description='失败引发新的情况',
            narrative=self._generate_narrative(failed_action, 'consequence'),
            new_scene=self._determine_new_scene(failed_action, context),
            consequences=self._generate_consequences(failed_action),
        )

    def _generate_narrative(
        self,
        action: Dict[str, Any],
        strategy_type: str
    ) -> str:
        """生成叙述"""
        action_type = action.get('type', '行动')

        narratives = {
            'cost': f'{action_type}失败了，但这不会阻止你。你决定接受代价继续前进。',
            'alter': f'{action_type}没成功，但你想到另一个办法。',
            'discover': f'{action_type}虽然失败了，但你在过程中发现了意想不到的线索。',
            'time': f'{action_type}失败后，你决定等待更好的时机。',
            'consequence': f'{action_type}的失败引发了新的情况，你必须应对。',
        }

        return narratives.get(strategy_type, '你决定继续前进。')

    def _generate_alternative_leads(
        self,
        failed_action: Dict[str, Any],
        context: 'GameContext'
    ) -> List[Dict]:
        """生成替代方案 Leads"""
        action_type = failed_action.get('type')

        return [
            {
                'title': f'寻找{action_type}的替代方法',
                'description': '尝试不同的方式达成目标',
                'category': 'investigate',
                'priority': 70,
            },
            {
                'title': '寻求帮助',
                'description': '寻找能协助的人或工具',
                'category': 'social',
                'priority': 65,
            },
        ]

    def _generate_discovery_leads(
        self,
        failed_action: Dict[str, Any],
        context: 'GameContext'
    ) -> List[Dict]:
        """生成发现线索 Leads"""
        return [
            {
                'title': '调查意外的发现',
                'description': '失败过程中发现了有趣的事情',
                'category': 'investigate',
                'priority': 75,
            }
        ]

    def _determine_new_scene(
        self,
        failed_action: Dict[str, Any],
        context: 'GameContext'
    ) -> Optional[str]:
        """确定新场景"""
        action_type = failed_action.get('type')

        if action_type == 'combat' and context.player.is_defeated:
            return 'prison'  # 被俘

        if action_type == 'stealth' and context.player.detected:
            return 'chase'  # 追逐

        return None

    def _generate_consequences(
        self,
        failed_action: Dict[str, Any]
    ) -> List[Dict]:
        """生成后果列表"""
        return [
            {'type': 'status_change', 'effect': 'alerted'},
            {'type': 'npc_reaction', 'effect': 'hostile'},
        ]
```

---

## 前进引擎

```python
# app/services/fail_forward/engine.py
from typing import Dict, Any, Optional

class FailForwardEngine:
    """失败前进引擎"""

    def __init__(
        self,
        strategy: FailForwardStrategy,
        consequence_generator: 'FailureConsequences',
        lead_generator: 'LeadGenerator'
    ):
        self.strategy = strategy
        self.consequence_generator = consequence_generator
        self.lead_generator = lead_generator

    async def process_failure(
        self,
        failed_action: Dict[str, Any],
        roll_result: int,
        difficulty: int,
        context: 'GameContext'
    ) -> Dict[str, Any]:
        """处理失败"""
        # 1. 判定失败等级
        failure_level = FailureLevelDeterminer.determine_level(
            failed_action,
            roll_result,
            difficulty,
            context
        )

        # 2. 生成后果
        consequence = await self.consequence_generator.generate_consequence(
            failed_action,
            failure_level,
            context
        )

        # 3. 生成前进选项
        forward_options = await self.strategy.generate_forward_options(
            failed_action,
            failure_level,
            context
        )

        # 4. 应用后果
        await self._apply_consequences(consequence, context)

        # 5. 生成新 Leads
        new_leads = await self._generate_failure_leads(
            failed_action,
            failure_level,
            forward_options,
            context
        )

        return {
            'failure_level': failure_level,
            'consequence': consequence,
            'forward_options': forward_options,
            'new_leads': new_leads,
        }

    async def _apply_consequences(
        self,
        consequence: 'FailureConsequence',
        context: 'GameContext'
    ):
        """应用后果"""
        # 应用代价
        for cost_type, cost_value in consequence.costs.items():
            if cost_type == 'time':
                context.advance_time(cost_value)
            elif cost_type == 'resources':
                context.consume_resource(cost_value)
            elif cost_type == 'health':
                context.damage(cost_value)

        # 应用惩罚
        for penalty in consequence.penalties:
            context.apply_penalty(penalty)

    async def _generate_failure_leads(
        self,
        failed_action: Dict[str, Any],
        failure_level: 'FailureLevel',
        forward_options: List['ForwardOption'],
        context: 'GameContext'
    ) -> List['LeadItem']:
        """生成失败后的 Leads"""
        leads = []

        # 从前进选项生成 Leads
        for option in forward_options:
            if option.new_leads:
                for lead_data in option.new_leads:
                    lead = LeadItem(
                        lead_id=f'failure_{option.option_id}',
                        title=lead_data['title'],
                        description=lead_data['description'],
                        category=lead_data.get('category', 'action'),
                        type='failure_recovery',
                        priority=lead_data.get('priority', 60),
                        action={
                            'type': 'recover',
                            'context': option.description,
                        },
                        source={
                            'type': 'system',
                            'source_id': 'fail_forward',
                            'auto_generated': True,
                        }
                    )
                    leads.append(lead)

        # 根据失败类型生成特定 Leads
        specific_leads = await self._generate_specific_failure_leads(
            failed_action,
            failure_level,
            context
        )
        leads.extend(specific_leads)

        return leads

    async def _generate_specific_failure_leads(
        self,
        failed_action: Dict[str, Any],
        failure_level: 'FailureLevel',
        context: 'GameContext'
    ) -> List['LeadItem']:
        """生成特定类型的失败 Leads"""
        action_type = failed_action.get('type')
        leads = []

        if action_type == 'combat':
            leads.append(LeadItem(
                title='重新评估战斗策略',
                description='考虑撤退、寻求帮助或改变战术',
                category='action',
                type='strategic',
                priority=80 if failure_level == 'critical' else 60,
                action={'type': 'plan'},
                source={'type': 'system', 'auto_generated': True},
            ))

        elif action_type == 'investigation':
            leads.append(LeadItem(
                title='换个角度调查',
                description='尝试不同的调查方法或信息源',
                category='investigate',
                type='alternative',
                priority=70,
                action={'type': 'investigate'},
                source={'type': 'system', 'auto_generated': True},
            ))

        elif action_type == 'social':
            leads.append(LeadItem(
                title='修复关系或寻找其他途径',
                description='社交失败后的补救措施',
                category='social',
                type='recovery',
                priority=65,
                action={'type': 'social'},
                source={'type': 'system', 'auto_generated': True},
            ))

        return leads

    async def execute_forward_option(
        self,
        option: 'ForwardOption',
        context: 'GameContext'
    ) -> bool:
        """执行前进选项"""
        # 检查要求
        if option.requirements:
            if not self._check_requirements(option.requirements, context):
                return False

        # 应用后果
        if option.consequences:
            for consequence in option.consequences:
                await self._apply_single_consequence(consequence, context)

        # 切换场景
        if option.new_scene:
            await context.change_scene(option.new_scene)

        # 推进时间
        if option.time_advance:
            context.advance_time(option.time_advance)

        # 添加新 Leads
        if option.new_leads:
            await self.lead_generator.add_leads(
                context.session_id,
                option.new_leads
            )

        return True

    def _check_requirements(
        self,
        requirements: Dict[str, Any],
        context: 'GameContext'
    ) -> bool:
        """检查要求"""
        if requirements.get('pay_cost'):
            return context.can_pay_cost()

        return True

    async def _apply_single_consequence(
        self,
        consequence: Dict[str, Any],
        context: 'GameContext'
    ):
        """应用单个后果"""
        consequence_type = consequence['type']

        if consequence_type == 'resource_loss':
            context.consume_resource(consequence['value'])
        elif consequence_type == 'time_loss':
            context.advance_time(consequence['value'])
        elif consequence_type == 'clue_gain':
            context.add_clue(consequence['value'])
        elif consequence_type == 'status_change':
            context.set_status(consequence['effect'])
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/fail_forward/strategy.py` | 创建 | 前进策略 |
| `app/services/fail_forward/engine.py` | 创建 | 前进引擎 |
| `tests/services/fail_forward/test_engine.py` | 创建 | 单元测试 |
| `docs/fail_forward/strategy.md` | 创建 | 策略文档 |

---

## 验收标准

- [ ] 前进策略多样且合理
- [ ] 引擎正确处理失败
- [ ] 后果应用正确
- [ ] 新 Leads 生成有效
- [ ] 叙述衔接自然
- [ ] 单元测试通过

---

## 参考文档

- M6-008: 失败等级枚举
- M6-010: 代价计算模型

---

**最后更新**: 2026-02-07
**状态**: [ ] 待开始
