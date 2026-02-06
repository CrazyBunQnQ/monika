# M6-007: 实现 Leads 关联线索

**任务ID**: M6-007
**标题**: 实现 Leads 关联线索
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M6-006

---

## 任务描述

实现 Leads 与线索的关联机制，使 Leads 能够根据玩家发现的线索动态生成和更新。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-007-01 | 设计关联模型 | 线索关联数据结构 | 20min |
| M6-007-02 | 实现线索跟踪器 | 跟踪玩家线索 | 30min |
| M6-007-03 | 实现关联生成器 | 基于线索生成 Lead | 40min |
| M6-007-04 | 实现线索链分析 | 分析线索关联链 | 30min |
| M6-007-05 | 实现关联更新 | 线索变化时更新 | 30min |
| M6-007-06 | 编写单元测试 | 测试覆盖 | 10min |

---

## 关联模型

```python
# app/services/leads/association.py
from typing import List, Optional, Set, Dict, Any
from dataclasses import dataclass, field
from datetime import datetime

@dataclass
class ClueAssociation:
    """线索关联"""
    clue_id: str
    association_type: str  # 'direct', 'indirect', 'chain'
    strength: float  # 0-1, 关联强度
    discovered_at: datetime = field(default_factory=datetime.now)

@dataclass
class LeadClueRelation:
    """Lead 与线索的关系"""
    lead_id: str
    related_clues: Set[str]
    associations: Dict[str, ClueAssociation] = field(default_factory=dict)

    # 关联规则
    require_all: bool = False      # 是否需要所有线索
    require_any: bool = True       # 是否需要任意线索
    unlock_on_discovery: bool = True  # 发现线索时解锁

    def is_accessible(self, player_clues: Set[str]) -> bool:
        """检查是否可访问"""
        if self.require_all:
            return self.related_clues.issubset(player_clues)
        elif self.require_any:
            return bool(self.related_clues & player_clues)
        return True

    def get_completion_rate(self, player_clues: Set[str]) -> float:
        """获取完成率"""
        if not self.related_clues:
            return 1.0
        discovered = len(self.related_clues & player_clues)
        return discovered / len(self.related_clues)
```

---

## 线索跟踪器

```python
# app/services/leads/clue_tracker.py
from typing import List, Set, Dict, Any
from collections import defaultdict

class ClueTracker:
    """线索跟踪器"""

    def __init__(self):
        self.player_clues: Dict[str, Set[str]] = {}  # session_id -> clue_ids
        self.clue_chains: Dict[str, List[str]] = {}  # clue_id -> related_clues
        self.clue_locations: Dict[str, str] = {}     # clue_id -> location
        self.clue_npcs: Dict[str, List[str]] = {}    # clue_id -> npc_ids

    async def on_clue_discovered(
        self,
        session_id: str,
        clue_id: str,
        clue_data: dict
    ):
        """处理线索发现"""
        # 添加到玩家线索
        if session_id not in self.player_clues:
            self.player_clues[session_id] = set()
        self.player_clues[session_id].add(clue_id)

        # 记录线索元数据
        self.clue_locations[clue_id] = clue_data.get('location')
        self.clue_npcs[clue_id] = clue_data.get('npcs', [])

        # 构建线索链
        self._build_clue_chain(clue_id, clue_data)

    def _build_clue_chain(self, clue_id: str, clue_data: dict):
        """构建线索链"""
        # 从线索数据中提取关联
        related = clue_data.get('related_clues', [])

        if clue_id not in self.clue_chains:
            self.clue_chains[clue_id] = []

        for related_id in related:
            self.clue_chains[clue_id].append(related_id)
            # 反向关联
            if related_id not in self.clue_chains:
                self.clue_chains[related_id] = []
            if clue_id not in self.clue_chains[related_id]:
                self.clue_chains[related_id].append(clue_id)

    def get_player_clues(self, session_id: str) -> Set[str]:
        """获取玩家的线索"""
        return self.player_clues.get(session_id, set())

    def get_related_clues(
        self,
        clue_id: str,
        max_depth: int = 2
    ) -> Set[str]:
        """获取相关线索"""
        related = set()
        queue = [(clue_id, 0)]

        while queue:
            current, depth = queue.pop(0)
            if depth >= max_depth or current in related:
                continue

            related.add(current)

            # 获取直接关联
            for next_id in self.clue_chains.get(current, []):
                if next_id not in related:
                    queue.append((next_id, depth + 1))

        return related

    def get_clues_by_location(
        self,
        location: str,
        session_id: str
    ) -> List[str]:
        """获取位置相关的线索"""
        player_clues = self.get_player_clues(session_id)
        return [
            clue_id for clue_id, loc in self.clue_locations.items()
            if loc == location and clue_id in player_clues
        ]

    def get_clues_by_npc(
        self,
        npc_id: str,
        session_id: str
    ) -> List[str]:
        """获取 NPC 相关的线索"""
        player_clues = self.get_player_clues(session_id)
        return [
            clue_id for clue_id, npcs in self.clue_npcs.items()
            if npc_id in npcs and clue_id in player_clues
        ]

    def find_clue_chains(
        self,
        session_id: str
    ) -> List[List[str]]:
        """查找线索链"""
        player_clues = self.get_player_clues(session_id)
        chains = []
        visited = set()

        for clue_id in player_clues:
            if clue_id in visited:
                continue

            # BFS 查找连通的线索
            chain = []
            queue = [clue_id]

            while queue:
                current = queue.pop(0)
                if current in visited:
                    continue

                visited.add(current)
                chain.append(current)

                for related_id in self.clue_chains.get(current, []):
                    if related_id in player_clues and related_id not in visited:
                        queue.append(related_id)

            if len(chain) > 1:
                chains.append(chain)

        return chains
```

---

## 关联生成器

```python
# app/services/leads/association_generator.py
from typing import List, Optional
from app.core.types.leads import LeadItem, GameContext
from app.services.leads.clue_tracker import ClueTracker

class ClueLeadGenerator:
    """基于线索的 Lead 生成器"""

    def __init__(self, clue_tracker: ClueTracker):
        self.clue_tracker = clue_tracker

    async def generate_from_clues(
        self,
        context: GameContext,
        count: int = 3
    ) -> List[LeadItem]:
        """根据线索生成 Leads"""
        session_id = context.session_id
        leads = []

        # 1. 获取玩家的线索
        player_clues = self.clue_tracker.get_player_clues(session_id)

        # 2. 按位置生成
        location_leads = await self._generate_by_location(context, player_clues)
        leads.extend(location_leads)

        # 3. 按 NPC 生成
        npc_leads = await self._generate_by_npc(context, player_clues)
        leads.extend(npc_leads)

        # 4. 按线索链生成
        chain_leads = await self._generate_by_chains(context, player_clues)
        leads.extend(chain_leads)

        # 5. 按未解决线索生成
        unsolved_leads = await self._generate_for_unsolved(context, player_clues)
        leads.extend(unsolved_leads)

        # 去重并限制数量
        unique_leads = self._deduplicate(leads)
        return unique_leads[:count]

    async def _generate_by_location(
        self,
        context: GameContext,
        player_clues: set
    ) -> List[LeadItem]:
        """根据位置生成"""
        leads = []
        current_location = context.current_scene.scene_id

        # 获取位置相关的线索
        location_clues = self.clue_tracker.get_clues_by_location(
            current_location,
            context.session_id
        )

        for clue_id in location_clues:
            lead = LeadItem(
                title=f'调查线索',
                description=f'在此地发现的线索值得进一步调查',
                category='investigate',
                type='clue_follow',
                priority=60,
                action={
                    'type': 'investigate',
                    'target': clue_id,
                },
                related={
                    'scene_id': current_location,
                    'clues': [clue_id],
                },
                source={
                    'type': 'clue',
                    'source_id': clue_id,
                    'auto_generated': True,
                }
            )
            leads.append(lead)

        return leads

    async def _generate_by_npc(
        self,
        context: GameContext,
        player_clues: set
    ) -> List[LeadItem]:
        """根据 NPC 生成"""
        leads = []
        current_scene = context.current_scene

        for npc in current_scene.npcs:
            # 获取 NPC 相关的线索
            npc_clues = self.clue_tracker.get_clues_by_npc(
                npc.npc_id,
                context.session_id
            )

            if npc_clues:
                lead = LeadItem(
                    title=f'向 {npc.name} 询问线索',
                    description=f'{npc.name} 可能知道关于这些线索的信息',
                    category='social',
                    type='npc_talk',
                    priority=70,
                    action={
                        'type': 'talk',
                        'target': npc.npc_id,
                    },
                    related={
                        'npcs': [npc.npc_id],
                        'clues': npc_clues,
                    },
                    source={
                        'type': 'clue',
                        'auto_generated': True,
                    }
                )
                leads.append(lead)

        return leads

    async def _generate_by_chains(
        self,
        context: GameContext,
        player_clues: set
    ) -> List[LeadItem]:
        """根据线索链生成"""
        leads = []
        chains = self.clue_tracker.find_clue_chains(context.session_id)

        for chain in chains:
            if len(chain) < 3:
                continue

            # 查找链条中的缺失环节
            missing = self._find_missing_links(chain, player_clues)

            if missing:
                lead = LeadItem(
                    title='追寻线索链',
                    description=f'发现了一条线索链，继续追踪可能揭示更多真相',
                    category='investigate',
                    type='clue_follow',
                    priority=75,
                    action={
                        'type': 'investigate',
                        'context': f'线索链: {", ".join(chain[:3])}',
                    },
                    related={
                        'clues': chain,
                    },
                    source={
                        'type': 'system',
                        'auto_generated': True,
                    }
                )
                leads.append(lead)

        return leads

    async def _generate_for_unsolved(
        self,
        context: GameContext,
        player_clues: set
    ) -> List[LeadItem]:
        """为未解决线索生成 Lead"""
        leads = []

        # 查找未解决的重要线索
        unsolved = context.player.get_unsolved_clues()
        important = [c for c in unsolved if c.importance >= 3]

        for clue in important:
            lead = LeadItem(
                title=f'调查: {clue.title}',
                description=clue.description,
                category='investigate',
                type='clue_follow',
                priority=80,
                action={
                    'type': 'investigate',
                    'target': clue.clue_id,
                },
                related={
                    'clues': [clue.clue_id],
                },
                source={
                    'type': 'clue',
                    'source_id': clue.clue_id,
                    'auto_generated': True,
                }
            )
            leads.append(lead)

        return leads

    def _find_missing_links(
        self,
        chain: List[str],
        player_clues: set
    ) -> List[str]:
        """查找链条中的缺失环节"""
        missing = []
        for clue_id in chain:
            if clue_id not in player_clues:
                missing.append(clue_id)
        return missing

    def _deduplicate(self, leads: List[LeadItem]) -> List[LeadItem]:
        """去重"""
        seen = set()
        unique = []

        for lead in leads:
            # 使用标题+目标作为唯一键
            key = (lead.title, lead.action.target)
            if key not in seen:
                seen.add(key)
                unique.append(lead)

        return unique
```

---

## 关联更新器

```python
# app/services/leads/association_updater.py
from typing import List

class AssociationUpdater:
    """关联更新器"""

    def __init__(
        self,
        clue_tracker: ClueTracker,
        lead_generator: ClueLeadGenerator,
        state_manager: LeadsStateManager
    ):
        self.clue_tracker = clue_tracker
        self.lead_generator = lead_generator
        self.state_manager = state_manager

    async def on_clue_discovered(
        self,
        session_id: str,
        clue_id: str,
        clue_data: dict
    ):
        """线索发现时更新 Leads"""
        # 1. 更新线索跟踪
        await self.clue_tracker.on_clue_discovered(
            session_id,
            clue_id,
            clue_data
        )

        # 2. 移除不再相关的 Leads
        await self._update_existing_leads(session_id, clue_id)

        # 3. 生成新的相关 Leads
        context = await self._get_context(session_id)
        new_leads = await self.lead_generator.generate_from_clues(
            context,
            count=2
        )

        if new_leads:
            await self.state_manager.add_leads(session_id, new_leads)

    async def _update_existing_leads(
        self,
        session_id: str,
        new_clue_id: str
    ):
        """更新现有的 Leads"""
        state = await self.state_manager.get_state(session_id)
        if not state:
            return

        # 查找与新线索相关的 Lead
        related_leads = [
            lead for lead in state.available
            if new_clue_id in (lead.related.clues or [])
        ]

        # 更新这些 Lead 的优先级
        for lead in related_leads:
            # 新线索发现，提升优先级
            lead.priority = min(100, lead.priority + 15)
            lead.updated_at = datetime.now()

        await self.state_manager.storage.save_state(state)

    async def _get_context(self, session_id: str) -> GameContext:
        """获取游戏上下文"""
        # 实现略
        pass
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/leads/association.py` | 创建 | 关联模型 |
| `app/services/leads/clue_tracker.py` | 创建 | 线索跟踪器 |
| `app/services/leads/association_generator.py` | 创建 | 关联生成器 |
| `app/services/leads/association_updater.py` | 创建 | 关联更新器 |
| `tests/services/leads/test_association.py` | 创建 | 单元测试 |

---

## 验收标准

- [ ] 关联模型正确
- [ ] 线索跟踪有效
- [ ] 关联生成合理
- [ ] 线索链分析正确
- [ ] 动态更新有效
- [ ] 单元测试通过

---

## 参考文档

- M6-002: Leads 生成算法
- M6-006: Leads 移除逻辑

---

**最后更新**: 2026-02-07
**状态**: [ ] 待开始
