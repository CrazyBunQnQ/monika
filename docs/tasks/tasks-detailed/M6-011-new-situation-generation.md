# M6-011: 实现新局面生成

**任务ID**: M6-011
**标题**: 实现新局面生成
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M6-010

---

## 任务描述

实现失败后的新局面生成，创建失败后的新场景和状态。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-011-01 | 设计局面转换规则 | 失败到新局面的映射 | 20min |
| M6-011-02 | 实现场景生成器 | 新场景创建 | 35min |
| M6-011-03 | 实现状态转移 | 玩家状态更新 | 25min |
| M6-011-04 | 实现叙述生成 | 过渡叙述 | 25min |
| M6-011-05 | 编写单元测试 | 测试覆盖 | 15min |

---

## 局面转换规则

```python
# app/services/fail_forward/situation_rules.py
from typing import Dict, Any, Optional, List
from enum import Enum

class SituationType(str, Enum):
    """局面类型"""
    CONTINUE = "continue"         # 继续当前场景
    ALTERED = "altered"           # 当前场景变化
    NEW_SCENE = "new_scene"       # 新场景
    COMBAT = "combat"             # 进入战斗
    CHASE = "chase"               # 进入追逐
    CAPTURED = "captured"         # 被俘
    RECOVERY = "recovery"         # 恢复场景

class SituationTransition:
    """局面转换规则"""

    TRANSITIONS = {
        # 战斗失败
        ('combat', 'critical'): {
            'situation': SituationType.CAPTURED,
            'probability': 0.7,
            'alternatives': [SituationType.RECOVERY],
        },
        ('combat', 'major'): {
            'situation': SituationType.RECOVERY,
            'probability': 0.8,
            'alternatives': [SituationType.CONTINUE],
        },

        # 潜行失败
        ('stealth', 'critical'): {
            'situation': SituationType.CHASE,
            'probability': 0.9,
            'alternatives': [SituationType.COMBAT],
        },
        ('stealth', 'major'): {
            'situation': SituationType.ALTERED,
            'probability': 0.7,
            'alternatives': [SituationType.COMBAT, SituationType.CONTINUE],
        },

        # 社交失败
        ('social', 'critical'): {
            'situation': SituationType.ALTERED,
            'probability': 0.8,
            'alternatives': [SituationType.NEW_SCENE],
        },

        # 调查失败
        ('investigation', 'critical'): {
            'situation': SituationType.ALTERED,
            'probability': 0.6,
            'alternatives': [SituationType.CONTINUE, SituationType.NEW_SCENE],
        },
    }

    @classmethod
    def determine_situation(
        cls,
        action_type: str,
        failure_level: str,
        context: 'GameContext'
    ) -> tuple[SituationType, Dict[str, Any]]:
        """确定新局面"""
        key = (action_type, failure_level)

        if key not in cls.TRANSITIONS:
            return SituationType.CONTINUE, {}

        import random
        rule = cls.TRANSITIONS[key]

        # 概率判定
        if random.random() < rule['probability']:
            situation = rule['situation']
        else:
            # 选择替代方案
            situation = random.choice(rule['alternatives'])

        # 生成局面参数
        params = cls._generate_situation_params(situation, context)

        return situation, params

    @classmethod
    def _generate_situation_params(
        cls,
        situation: SituationType,
        context: 'GameContext'
    ) -> Dict[str, Any]:
        """生成局面参数"""
        params = {'situation_type': situation}

        if situation == SituationType.CAPTURED:
            params.update({
                'location': 'prison',
                'captors': cls._determine_captors(context),
                'escape_difficulty': 'hard',
            })
        elif situation == SituationType.CHASE:
            params.update({
                'chase_type': cls._determine_chase_type(context),
                'duration': '1d3 rounds',
                'consequences': 'exhausted',
            })
        elif situation == SituationType.RECOVERY:
            params.update({
                'recovery_type': 'rest',
                'duration': '1d2 hours',
                'location': 'safe_house',
            })

        return params

    @classmethod
    def _determine_captors(cls, context: 'GameContext') -> str:
        """确定俘虏者"""
        # 从当前场景的敌对 NPC 中选择
        enemies = context.current_scene.get_hostile_npcs()
        if enemies:
            return enemies[0].faction
        return 'unknown'

    @classmethod
    def _determine_chase_type(cls, context: 'GameContext') -> str:
        """确定追逐类型"""
        if context.player.is_mounted():
            return 'mounted'
        elif context.current_scene.is_urban():
            return 'urban'
        else:
            return 'foot'
```

---

## 场景生成器

```python
# app/services/fail_forward/scene_generator.py
from typing import Dict, Any, Optional

class NewSceneGenerator:
    """新场景生成器"""

    def __init__(self):
        self.scene_templates = self._load_scene_templates()

    async def generate_scene(
        self,
        situation: SituationType,
        params: Dict[str, Any],
        context: 'GameContext'
    ) -> 'Scene':
        """生成新场景"""
        if situation == SituationType.CONTINUE:
            return await self._generate_continuation(context, params)
        elif situation == SituationType.ALTERED:
            return await self._generate_altered_scene(context, params)
        elif situation == SituationType.NEW_SCENE:
            return await self._generate_fresh_scene(context, params)
        elif situation == SituationType.COMBAT:
            return await self._generate_combat_scene(context, params)
        elif situation == SituationType.CHASE:
            return await self._generate_chase_scene(context, params)
        elif situation == SituationType.CAPTURED:
            return await self._generate_captured_scene(context, params)
        elif situation == SituationType.RECOVERY:
            return await self._generate_recovery_scene(context, params)

    async def _generate_continuation(
        self,
        context: 'GameContext',
        params: Dict[str, Any]
    ) -> 'Scene':
        """生成延续场景"""
        current = context.current_scene

        # 基于当前场景创建轻微变化
        new_scene = Scene(
            scene_id=f"{current.scene_id}_continued",
            name=current.name,
            location_type=current.location_type,
            description=current.description,
            atmosphere=self._alter_atmosphere(current.atmosphere, 'tense'),
            npcs=current.npcs,
            exits=current.exits,
            interactables=current.interactables,
        )

        # 添加失败后果
        new_scene.add_status('aftermath')

        return new_scene

    async def _generate_altered_scene(
        self,
        context: 'GameContext',
        params: Dict[str, Any]
    ) -> 'Scene':
        """生成变化场景"""
        current = context.current_scene

        # 显著改变场景
        new_scene = Scene(
            scene_id=f"{current.scene_id}_altered",
            name=current.name,
            location_type=current.location_type,
            description=self._alter_description(current.description),
            atmosphere=self._alter_atmosphere(current.atmosphere, 'dramatic'),
            npcs=self._alter_npcs(current.npcs, 'hostile'),
            exits=current.exits,
            interactables=current.interactables,
        )

        # 添加新状态
        new_scene.add_status('changed')
        new_scene.add_tag('failure_consequence')

        return new_scene

    async def _generate_combat_scene(
        self,
        context: 'GameContext',
        params: Dict[str, Any]
    ) -> 'Scene':
        """生成战斗场景"""
        return Scene(
            scene_id=f"combat_{context.session_id}_{context.round}",
            name='突发战斗',
            location_type='combat',
            description='敌人突然发起攻击！',
            atmosphere='dangerous',
            npcs=self._generate_combat_enemies(context),
            exits=[],
            interactables=[],
        )

    async def _generate_chase_scene(
        self,
        context: 'GameContext',
        params: Dict[str, Any]
    ) -> 'Scene':
        """生成追逐场景"""
        chase_type = params.get('chase_type', 'foot')

        return Scene(
            scene_id=f"chase_{context.session_id}_{context.round}",
            name=f'追逐 ({chase_type})',
            location_type='chase',
            description='你被迫开始逃亡！',
            atmosphere='urgent',
            npcs=self._generate_chasers(context),
            exits=self._generate_chase_exits(chase_type),
            interactables=self._generate_chase_obstacles(),
        )

    async def _generate_captured_scene(
        self,
        context: 'GameContext',
        params: Dict[str, Any]
    ) -> 'Scene':
        """生成被俘场景"""
        captors = params.get('captors', 'unknown')

        return Scene(
            scene_id=f"captured_{context.session_id}",
            name='囚禁',
            location_type='prison',
            description=f'你被{captors}俘虏了，现在被困在一个未知的地方。',
            atmosphere='bleak',
            npcs=[],
            exits=[],
            interactables=[
                {
                    'name': '牢门',
                    'type': 'exit',
                    'locked': True,
                    'difficulty': params.get('escape_difficulty', 'hard'),
                },
                {
                    'name': '牢房环境',
                    'type': 'investigate',
                    'description': '仔细检查牢房可能找到逃生工具',
                },
            ],
        )

    async def _generate_recovery_scene(
        self,
        context: 'GameContext',
        params: Dict[str, Any]
    ) -> 'Scene':
        """生成恢复场景"""
        return Scene(
            scene_id=f"recovery_{context.session_id}",
            name='安全屋',
            location_type='safe_house',
            description='一个可以暂时休息和恢复的地方。',
            atmosphere='calm',
            npcs=[],
            exits=[],
            interactables=[
                {
                    'name': '床铺',
                    'type': 'rest',
                    'effect': '恢复生命值和耐力',
                },
                {
                    'name': '补给',
                    'type': 'resource',
                    'effect': '获得基础资源',
                },
            ],
        )

    def _alter_atmosphere(self, current: str, direction: str) -> str:
        """改变氛围"""
        atmosphere_map = {
            'calm': ['tense', 'normal'],
            'normal': ['tense', 'dramatic', 'calm'],
            'tense': ['dramatic', 'dangerous', 'normal'],
            'dramatic': ['dangerous', 'tense'],
            'dangerous': ['urgent', 'dramatic'],
        }

        options = atmosphere_map.get(current, ['normal'])

        if direction == 'tense':
            return options[0]
        elif direction == 'dramatic':
            return 'dramatic' if 'dramatic' in options else options[-1]

        return current

    def _alter_description(self, current: str) -> str:
        """改变描述"""
        # 添加失败后果的描述
        additions = [
            '现场一片混乱。',
            '事情变得更复杂了。',
            '情况急转直下。',
        ]

        import random
        addition = random.choice(additions)
        return f'{current}\n\n{addition}'

    def _alter_npcs(self, npcs: List, attitude: str) -> List:
        """改变 NPC 态度"""
        altered = []

        for npc in npcs:
            new_npc = npc.copy()
            new_npc.attitude = attitude
            altered.append(new_npc)

        return altered

    def _load_scene_templates(self) -> Dict[str, Any]:
        """加载场景模板"""
        return {}
```

---

## 状态转移管理器

```python
# app/services/fail_forward/state_transition.py
from typing import Dict, Any

class StateTransitionManager:
    """状态转移管理器"""

    async def transition_to_new_situation(
        self,
        situation: SituationType,
        params: Dict[str, Any],
        context: 'GameContext'
    ) -> Dict[str, Any]:
        """转移到新局面"""
        # 1. 生成新场景
        new_scene = await self.scene_generator.generate_scene(
            situation,
            params,
            context
        )

        # 2. 转移玩家状态
        await self._transition_player_state(situation, context)

        # 3. 更新游戏状态
        context.change_scene(new_scene)

        # 4. 生成过渡叙述
        transition = self._generate_transition_narrative(
            situation,
            context.current_scene,
            new_scene
        )

        # 5. 生成新 Leads
        new_leads = await self._generate_transition_leads(
            situation,
            new_scene,
            context
        )

        return {
            'new_scene': new_scene,
            'transition': transition,
            'new_leads': new_leads,
        }

    async def _transition_player_state(
        self,
        situation: SituationType,
        context: 'GameContext'
    ):
        """转移玩家状态"""
        player = context.player

        if situation == SituationType.CAPTURED:
            # 移除装备
            player.unequip_all()
            # 添加状态
            player.add_status('captured')
            player.add_status('restrained')

        elif situation == SituationType.CHASE:
            # 添加状态
            player.add_status('fleeing')
            player.add_status('exhausted')

        elif situation == SituationType.RECOVERY:
            # 移除负面状态
            player.remove_status('exhausted')
            player.remove_status('injured')

    def _generate_transition_narrative(
        self,
        situation: SituationType,
        old_scene: 'Scene',
        new_scene: 'Scene'
    ) -> str:
        """生成过渡叙述"""
        transitions = {
            SituationType.CONTINUE: (
                f'尽管失败了，但你仍在{old_scene.name}。'
            ),
            SituationType.ALTERED: (
                f'失败改变了{old_scene.name}的情况。'
            ),
            SituationType.COMBAT: (
                f'失败引发了战斗！'
            ),
            SituationType.CHASE: (
                f'你被迫从{old_scene.name}逃离！'
            ),
            SituationType.CAPTURED: (
                f'你被俘虏并带到了{new_scene.name}。'
            ),
            SituationType.RECOVERY: (
                f'你找到了{new_scene.name}来恢复。'
            ),
        }

        return transitions.get(situation, '情况发生了变化。')

    async def _generate_transition_leads(
        self,
        situation: SituationType,
        new_scene: 'Scene',
        context: 'GameContext'
    ) -> List['LeadItem']:
        """生成过渡 Leads"""
        leads = []

        if situation == SituationType.CAPTURED:
            leads.append(LeadItem(
                title='寻找逃脱方法',
                description='仔细观察牢房，寻找可能的逃脱途径',
                category='action',
                type='escape',
                priority=90,
                action={'type': 'investigate'},
                source={'type': 'system', 'auto_generated': True},
            ))

        elif situation == SituationType.CHASE:
            leads.append(LeadItem(
                title='摆脱追兵',
                description='寻找机会甩掉追兵',
                category='action',
                type='flee',
                priority=95,
                action={'type': 'flee'},
                source={'type': 'system', 'auto_generated': True},
            ))

        return leads
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/fail_forward/situation_rules.py` | 创建 | 局面转换规则 |
| `app/services/fail_forward/scene_generator.py` | 创建 | 场景生成器 |
| `app/services/fail_forward/state_transition.py` | 创建 | 状态转移管理 |
| `tests/services/fail_forward/test_situation.py` | 创建 | 单元测试 |

---

## 验收标准

- [ ] 转换规则合理
- [ ] 场景生成有效
- [ ] 状态转移正确
- [ ] 叙述自然
- [ ] 新 Leads 生成有效
- [ ] 单元测试通过

---

## 参考文档

- M6-009: 失败前进算法
- M6-010: 代价计算模型

---

**最后更新**: 2026-02-07
**状态**: [ ] 待开始
