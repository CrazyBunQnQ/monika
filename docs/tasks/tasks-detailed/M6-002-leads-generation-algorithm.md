# M6-002: 实现 Leads 生成算法

**任务ID**: M6-002
**标题**: 实现 Leads 生成算法
**类型**: backend (后端开发)
**预估工时**: 4h
**依赖**: M6-001

---

## 任务描述

实现 Leads (可选行动) 的自动生成算法，根据游戏当前状态动态生成合理的行动建议。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-002-01 | 设计生成策略框架 | 基础架构 | 40min |
| M6-002-02 | 实现场景分析器 | 分析当前场景 | 50min |
| M6-002-03 | 实现线索分析器 | 分析可用线索 | 50min |
| M6-002-04 | 实现 NPC 分析器 | 分析可用 NPC | 40min |
| M6-002-05 | 实现模板匹配器 | 匹配生成模板 | 40min |
| M6-002-06 | 实现去重和过滤 | 避免重复 Lead | 30min |
| M6-002-07 | 编写单元测试 | 测试覆盖率 > 80% | 30min |

---

## 生成策略框架

```python
# app/services/leads/generator.py
from typing import List, Optional
from app.core.types.leads import LeadItem, GameContext

class LeadGenerator:
    """Leads 生成器"""

    def __init__(self, templates: List[LeadTemplate]):
        self.templates = templates
        self.analyzers = {
            'scene': SceneAnalyzer(),
            'clue': ClueAnalyzer(),
            'npc': NPCAnalyzer(),
            'location': LocationAnalyzer(),
        }

    async def generate(
        self,
        context: GameContext,
        count: int = 3
    ) -> List[LeadItem]:
        """
        根据当前游戏状态生成 Leads

        Args:
            context: 游戏上下文
            count: 目标生成数量

        Returns:
            生成的 Lead 列表
        """
        candidates = []

        # 1. 分析当前状态
        analysis = await self._analyze_context(context)

        # 2. 应用生成模板
        for template in self.templates:
            if template.applicable_if(context):
                leads = await template.generate(analysis)
                candidates.extend(leads)

        # 3. 去重和过滤
        unique_leads = self._deduplicate(candidates)

        # 4. 优先级排序
        sorted_leads = self._sort_by_priority(unique_leads)

        # 5. 返回前 N 个
        return sorted_leads[:count]

    async def _analyze_context(self, context: GameContext) -> dict:
        """分析游戏上下文"""
        analysis = {}

        # 并行分析
        for name, analyzer in self.analyzers.items():
            analysis[name] = await analyzer.analyze(context)

        return analysis

    def _deduplicate(self, leads: List[LeadItem]) -> List[LeadItem]:
        """去重"""
        seen = set()
        unique = []

        for lead in leads:
            # 使用标题+类型作为唯一键
            key = (lead.title, lead.type)
            if key not in seen:
                seen.add(key)
                unique.append(lead)

        return unique

    def _sort_by_priority(self, leads: List[LeadItem]) -> List[LeadItem]:
        """按优先级排序"""
        return sorted(leads, key=lambda x: x.priority, reverse=True)
```

---

## 场景分析器

```python
# app/services/leads/analyzers/scene.py
from typing import Dict, Any

class SceneAnalyzer:
    """场景分析器"""

    async def analyze(self, context: GameContext) -> Dict[str, Any]:
        """分析当前场景"""
        scene = context.current_scene

        return {
            'scene_id': scene.scene_id,
            'scene_type': scene.type,
            'location_type': scene.location_type,

            # 可互动元素
            'interactables': scene.interactables,
            'exits': scene.exits,
            'items': scene.items,

            # 氛围和状态
            'atmosphere': scene.atmosphere,
            'lighting': scene.lighting,
            'crowd_level': scene.crowd_level,

            # 特殊事件
            'active_events': scene.active_events,

            # 时间信息
            'time_of_day': scene.time_of_day,
            'weather': scene.weather,
        }
```

---

## 线索分析器

```python
# app/services/leads/analyzers/clue.py
from typing import Dict, Any, List

class ClueAnalyzer:
    """线索分析器"""

    async def analyze(self, context: GameContext) -> Dict[str, Any]:
        """分析可用线索"""
        player_clues = context.player.clues

        # 分类线索
        new_clues = [c for c in player_clues if c.is_new]
        active_clues = [c for c in player_clues if c.is_active]
        solved_clues = [c for c in player_clues if c.is_solved]
        expired_clues = [c for c in player_clues if c.is_expired]

        return {
            'total_count': len(player_clues),
            'new_clues': new_clues,
            'active_clues': active_clues,
            'solved_clues': solved_clues,
            'expired_clues': expired_clues,

            # 线索关联
            'related_clues': self._find_related_clues(new_clues, active_clues),
            'clue_chains': self._find_clue_chains(player_clues),

            # 未追踪的线索
            'untracked': self._find_untracked(player_clues),
        }

    def _find_related_clues(self, new_clues, active_clues) -> List[dict]:
        """查找相关线索"""
        related = []
        for new in new_clues:
            for active in active_clues:
                if self._are_related(new, active):
                    related.append({
                        'new': new,
                        'related_to': active,
                        'connection': new.connection_to(active)
                    })
        return related

    def _find_clue_chains(self, clues) -> List[List]:
        """查找线索链"""
        chains = []
        # 实现线索链查找逻辑
        return chains

    def _are_related(self, clue1, clue2) -> bool:
        """判断两个线索是否相关"""
        return (
            clue1.location == clue2.location or
            clue1.npc == clue2.npc or
            set(clue1.tags) & set(clue2.tags)
        )
```

---

## NPC 分析器

```python
# app/services/leads/analyzers/npc.py
from typing import Dict, Any, List

class NPCAnalyzer:
    """NPC 分析器"""

    async def analyze(self, context: GameContext) -> Dict[str, Any]:
        """分析可用 NPC"""
        current_scene = context.current_scene
        available_npcs = current_scene.npcs

        return {
            'present_npcs': available_npcs,

            # NPC 分类
            'friendly': [n for n in available_npcs if n.attitude == 'friendly'],
            'neutral': [n for n in available_npcs if n.attitude == 'neutral'],
            'hostile': [n for n in available_npcs if n.attitude == 'hostile'],

            # 可对话 NPC
            'conversable': [n for n in available_npcs if n.can_converse],

            # 拥有信息的 NPC
            'informative': [n for n in available_npcs if n.has_information],

            # 可交易 NPC
            'merchants': [n for n in available_npcs if n.is_merchant],

            # 任务 NPC
            'quest_givers': [n for n in available_npcs if n.has_quest],
        }
```

---

## 生成模板

```python
# app/services/leads/templates.py
from typing import List
from app.core.types.leads import LeadItem, GameContext, LeadCategory

class LeadTemplate:
    """Lead 生成模板基类"""

    def __init__(
        self,
        template_id: str,
        name: str,
        category: LeadCategory
    ):
        self.template_id = template_id
        self.name = name
        self.category = category

    async def generate(self, analysis: dict) -> List[LeadItem]:
        """生成 Leads"""
        raise NotImplementedError

    def applicable_if(self, context: GameContext) -> bool:
        """判断模板是否适用"""
        return True

# 具体模板示例

class TalkToNPCTemplate(LeadTemplate):
    """与 NPC 对话模板"""

    def __init__(self):
        super().__init__(
            template_id='talk_to_npc',
            name='与 NPC 对话',
            category='social'
        )

    async def generate(self, analysis: dict) -> List[LeadItem]:
        leads = []
        npcs = analysis['npc']['conversable']

        for npc in npcs:
            lead = LeadItem(
                title=f'与 {npc.name} 对话',
                description=f'{npc.name} 可能知道一些信息',
                category=self.category,
                type='npc_talk',
                priority=self._calculate_priority(npc),
                action={
                    'type': 'talk',
                    'target': npc.npc_id,
                    'context': npc.current_location,
                },
                related={
                    'npcs': [npc.npc_id],
                },
                source={
                    'type': 'system',
                    'auto_generated': True,
                }
            )
            leads.append(lead)

        return leads

    def _calculate_priority(self, npc) -> int:
        """计算优先级"""
        base = 50
        if npc.has_quest:
            base += 20
        if npc.has_information:
            base += 10
        return base

class InvestigateLocationTemplate(LeadTemplate):
    """调查地点模板"""

    def __init__(self):
        super().__init__(
            template_id='investigate_location',
            name='调查地点',
            category='investigate'
        )

    async def generate(self, analysis: dict) -> List[LeadItem]:
        leads = []
        interactables = analysis['scene']['interactables']

        for item in interactables:
            if item.investigable:
                lead = LeadItem(
                    title=f'调查 {item.name}',
                    description=item.description,
                    category=self.category,
                    type='location_search',
                    priority=50,
                    action={
                        'type': 'investigate',
                        'target': item.item_id,
                    },
                    related={
                        'scene_id': analysis['scene']['scene_id'],
                    },
                    source={
                        'type': 'system',
                        'auto_generated': True,
                    }
                )
                leads.append(lead)

        return leads
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/leads/generator.py` | 创建 | Leads 生成器核心 |
| `app/services/leads/analyzers/__init__.py` | 创建 | 分析器包 |
| `app/services/leads/analyzers/scene.py` | 创建 | 场景分析器 |
| `app/services/leads/analyzers/clue.py` | 创建 | 线索分析器 |
| `app/services/leads/analyzers/npc.py` | 创建 | NPC 分析器 |
| `app/services/leads/templates.py` | 创建 | 生成模板 |
| `tests/services/leads/test_generator.py` | 创建 | 单元测试 |

---

## 验收标准

- [ ] 能根据场景生成合理的 Leads
- [ ] 能根据线索生成相关行动
- [ ] 能根据 NPC 生成对话选项
- [ ] 去重机制正常工作
- [ ] 优先级排序正确
- [ ] 单元测试覆盖率 > 80%

---

## 参考文档

- M6-001: Leads 数据结构
- M6-003: Leads 优先级排序

---

**最后更新**: 2026-02-07
**状态**: [ ] 待开始
