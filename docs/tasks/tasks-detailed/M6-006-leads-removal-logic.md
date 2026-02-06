# M6-006: 实现 Leads 移除逻辑

**任务ID**: M6-006
**标题**: 实现 Leads 移除逻辑
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M6-004

---

## 任务描述

实现 Leads 移除逻辑，包括完成移除、过期移除、手动移除和批量移除。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-006-01 | 实现移除策略 | 移除条件判断 | 20min |
| M6-006-02 | 实现单个移除 | 单个 Lead 移除 | 25min |
| M6-006-03 | 实现批量移除 | 批量操作 | 20min |
| M6-006-04 | 实现移除通知 | 通知前端 | 20min |
| M6-006-05 | 实现移除历史 | 记录移除历史 | 20min |
| M6-006-06 | 编写单元测试 | 测试覆盖 | 15min |

---

## 移除策略

```python
# app/services/leads/removal.py
from typing import List, Optional
from datetime import datetime
from enum import Enum

class RemovalReason(str, Enum):
    """移除原因"""
    COMPLETED = "completed"       # 已完成
    FAILED = "failed"            # 已失败
    EXPIRED = "expired"          # 已过期
    MANUAL = "manual"            # 手动移除
    BLOCKED = "blocked"          # 被阻塞
    REPLACED = "replaced"        # 被替换
    IRRELEVANT = "irrelevant"    # 不再相关

class RemovalStrategy:
    """移除策略"""

    def should_remove(
        self,
        lead: 'LeadItem',
        context: 'GameContext'
    ) -> tuple[bool, Optional[RemovalReason]]:
        """
        判断是否应该移除

        Returns:
            (should_remove, reason)
        """
        # 1. 检查过期
        if self._is_expired(lead):
            return True, RemovalReason.EXPIRED

        # 2. 检查是否不再相关
        if self._is_irrelevant(lead, context):
            return True, RemovalReason.IRRELEVANT

        # 3. 检查是否被阻塞
        if self._is_blocked(lead, context):
            return True, RemovalReason.BLOCKED

        return False, None

    def _is_expired(self, lead: 'LeadItem') -> bool:
        """检查是否过期"""
        if not lead.validity.expires_at:
            return False
        return datetime.now() > lead.validity.expires_at

    def _is_irrelevant(
        self,
        lead: 'LeadItem',
        context: 'GameContext'
    ) -> bool:
        """检查是否不再相关"""
        # 场景改变
        if lead.related.scene_id:
            if lead.related.scene_id != context.current_scene.scene_id:
                # 如果不是全局 Lead，则不再相关
                return not lead.is_global

        # NPC 不在场
        if lead.related.npcs:
            current_npcs = {
                npc.npc_id for npc in context.current_scene.npcs
            }
            if not any(npc_id in current_npcs for npc_id in lead.related.npcs):
                return True

        # 线索已解决
        if lead.related.clues:
            solved_clues = context.player.get_solved_clues()
            if all(clue in solved_clues for clue in lead.related.clues):
                return True

        return False

    def _is_blocked(
        self,
        lead: 'LeadItem',
        context: 'GameContext'
    ) -> bool:
        """检查是否被阻塞"""
        if not lead.action.requirements:
            return False

        reqs = lead.action.requirements

        # 检查线索要求
        if reqs.clues:
            player_clues = {c.clue_id for c in context.player.clues}
            if not any(clue in player_clues for clue in reqs.clues):
                return True

        # 检查技能要求
        if reqs.skills:
            player_skills = context.player.get_skills()
            for skill in reqs.skills:
                if player_skills.get(skill, 0) < reqs.skills.get(skill, 0):
                    return True

        # 检查物品要求
        if reqs.items:
            player_items = {i.item_id for i in context.player.inventory}
            if not any(item in player_items for item in reqs.items):
                return True

        # 检查状态要求
        if reqs.state:
            for key, value in reqs.state.items():
                if context.state.get(key) != value:
                    return True

        return False
```

---

## 移除管理器

```python
# app/services/leads/removal_manager.py
from typing import List, Optional
from datetime import datetime

from app.core.types.leads import LeadItem, GameContext
from app.services.leads.removal import RemovalReason, RemovalStrategy
from app.services.leads.state_manager import LeadsStateManager

class LeadsRemovalManager:
    """Leads 移除管理器"""

    def __init__(
        self,
        state_manager: LeadsStateManager,
        websocket_manager: 'WebSocketManager'
    ):
        self.state_manager = state_manager
        self.websocket_manager = websocket_manager
        self.strategy = RemovalStrategy()
        self.removal_history: dict[str, List[dict]] = {}

    async def remove_lead(
        self,
        session_id: str,
        lead_id: str,
        reason: RemovalReason,
        removed_by: str = 'system'
    ) -> bool:
        """移除单个 Lead"""
        state = await self.state_manager.get_state(session_id)
        if not state:
            return False

        # 查找 Lead
        lead = self._find_lead(state, lead_id)
        if not lead:
            return False

        # 从可用列表移除
        state.available.remove(lead)

        # 记录历史
        await self._record_removal(
            session_id,
            lead,
            reason,
            removed_by
        )

        # 更新状态
        state.updated_at = datetime.now()
        await self.state_manager.storage.save_state(state)

        # 发送通知
        await self._notify_removal(session_id, lead, reason)

        return True

    async def remove_leads_batch(
        self,
        session_id: str,
        lead_ids: List[str],
        reason: RemovalReason
    ) -> int:
        """批量移除 Leads"""
        removed_count = 0

        for lead_id in lead_ids:
            if await self.remove_lead(session_id, lead_id, reason):
                removed_count += 1

        return removed_count

    async def cleanup_leads(
        self,
        session_id: str,
        context: GameContext
    ) -> int:
        """清理不再有效的 Leads"""
        state = await self.state_manager.get_state(session_id)
        if not state:
            return 0

        to_remove = []

        for lead in state.available:
            should_remove, reason = self.strategy.should_remove(lead, context)
            if should_remove:
                to_remove.append((lead, reason))

        # 批量移除
        for lead, reason in to_remove:
            await self.remove_lead(
                session_id,
                lead.lead_id,
                reason
            )

        return len(to_remove)

    async def remove_completed_leads(
        self,
        session_id: str,
        lead_ids: List[str]
    ) -> dict:
        """移除已完成的 Leads"""
        results = {
            'removed': [],
            'failed': []
        }

        for lead_id in lead_ids:
            success = await self.remove_lead(
                session_id,
                lead_id,
                RemovalReason.COMPLETED
            )

            if success:
                results['removed'].append(lead_id)
            else:
                results['failed'].append(lead_id)

        return results

    async def _record_removal(
        self,
        session_id: str,
        lead: LeadItem,
        reason: RemovalReason,
        removed_by: str
    ):
        """记录移除历史"""
        if session_id not in self.removal_history:
            self.removal_history[session_id] = []

        record = {
            'lead_id': lead.lead_id,
            'lead_title': lead.title,
            'reason': reason.value,
            'removed_by': removed_by,
            'timestamp': datetime.now().isoformat(),
        }

        self.removal_history[session_id].append(record)

        # 保持历史记录大小
        if len(self.removal_history[session_id]) > 100:
            self.removal_history[session_id] = (
                self.removal_history[session_id][-100:]
            )

    async def _notify_removal(
        self,
        session_id: str,
        lead: LeadItem,
        reason: RemovalReason
    ):
        """通知移除"""
        await self.websocket_manager.broadcast(
            session_id,
            {
                'type': 'lead_removed',
                'lead_id': lead.lead_id,
                'reason': reason.value,
                'timestamp': datetime.now().isoformat(),
            }
        )

    def _find_lead(
        self,
        state: 'LeadsState',
        lead_id: str
    ) -> Optional[LeadItem]:
        """查找 Lead"""
        for lead in state.available:
            if lead.lead_id == lead_id:
                return lead
        return None

    async def get_removal_history(
        self,
        session_id: str,
        limit: int = 20
    ) -> List[dict]:
        """获取移除历史"""
        if session_id not in self.removal_history:
            return []

        history = self.removal_history[session_id]
        return history[-limit:]
```

---

## API 接口

```python
# app/api/leads.py
from fastapi import APIRouter, Depends, HTTPException
from app.services.leads.removal_manager import LeadsRemovalManager
from app.services.leads.removal import RemovalReason

router = APIRouter()

@router.delete('/game/{session_id}/leads/{lead_id}')
async def remove_lead(
    session_id: str,
    lead_id: str,
    reason: RemovalReason = RemovalReason.MANUAL,
    removal_manager: LeadsRemovalManager = Depends()
):
    """移除 Lead"""
    success = await removal_manager.remove_lead(
        session_id,
        lead_id,
        reason,
        removed_by='user'
    )

    if not success:
        raise HTTPException(status_code=404, detail='Lead not found')

    return {'status': 'removed'}

@router.post('/game/{session_id}/leads/remove-batch')
async def remove_leads_batch(
    session_id: str,
    lead_ids: List[str],
    reason: RemovalReason,
    removal_manager: LeadsRemovalManager = Depends()
):
    """批量移除 Leads"""
    count = await removal_manager.remove_leads_batch(
        session_id,
        lead_ids,
        reason
    )

    return {
        'status': 'batch_removed',
        'count': count
    }

@router.post('/game/{session_id}/leads/cleanup')
async def cleanup_leads(
    session_id: str,
    removal_manager: LeadsRemovalManager = Depends(),
    context: GameContext = Depends()
):
    """清理无效 Leads"""
    count = await removal_manager.cleanup_leads(session_id, context)

    return {
        'status': 'cleaned',
        'count': count
    }

@router.get('/game/{session_id}/leads/removal-history')
async def get_removal_history(
    session_id: str,
    limit: int = 20,
    removal_manager: LeadsRemovalManager = Depends()
):
    """获取移除历史"""
    history = await removal_manager.get_removal_history(
        session_id,
        limit
    )

    return {'history': history}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/leads/removal.py` | 创建 | 移除策略 |
| `app/services/leads/removal_manager.py` | 创建 | 移除管理器 |
| `app/api/leads.py` | 修改 | API 接口 |
| `tests/services/leads/test_removal.py` | 创建 | 单元测试 |

---

## 验收标准

- [ ] 移除策略正确
- [ ] 单个移除有效
- [ ] 批量移除有效
- [ ] 自动清理有效
- [ ] 历史记录完整
- [ ] 单元测试通过

---

## 参考文档

- M6-004: Leads 状态管理

---

**最后更新**: 2026-02-07
**状态**: [ ] 待开始
