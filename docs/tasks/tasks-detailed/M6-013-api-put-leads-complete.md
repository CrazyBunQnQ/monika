# M6-013: 实现 PUT /game/leads/:id/complete

**任务ID**: M6-013
**标题**: 实现 PUT /game/leads/:id/complete
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M6-006

---

## 任务描述

实现标记 Lead 为完成的 API 端点。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-013-01 | 设计 API 规范 | 请求/响应格式 | 20min |
| M6-013-02 | 实现完成逻辑 | 状态更新 | 30min |
| M6-013-03 | 实现结果记录 | 完成结果存储 | 30min |
| M6-013-04 | 实现触发器 | 完成后的触发 | 25min |
| M6-013-05 | 编写 API 文档 | OpenAPI 规范 | 10min |
| M6-013-06 | 编写单元测试 | 测试覆盖 | 15min |

---

## API 规范

```python
# app/api/leads.py
from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel

class CompleteLeadRequest(BaseModel):
    """完成 Lead 请求"""
    success: bool = True
    outcome: Optional[str] = None
    notes: Optional[str] = None
    rewards: Optional[dict] = None

class CompleteLeadResponse(BaseModel):
    """完成 Lead 响应"""
    lead_id: str
    status: str
    completed_at: str
    result: dict

@router.put("/{lead_id}/complete", response_model=CompleteLeadResponse)
async def complete_lead(
    session_id: str,
    lead_id: str,
    request: CompleteLeadRequest,
    current_user: dict = Depends(get_current_user),
    leads_manager: 'LeadsStateManager' = Depends(get_leads_manager),
    event_bus: 'EventBus' = Depends(get_event_bus)
):
    """
    标记 Lead 为完成

    参数:
    - session_id: 游戏会话 ID
    - lead_id: Lead ID
    - success: 是否成功 (默认 true)
    - outcome: 完成结果描述
    - notes: 备注
    - rewards: 获得的奖励

    返回:
    - lead_id: Lead ID
    - status: 新状态
    - completed_at: 完成时间
    - result: 完成结果详情
    """
    # 验证权限
    if not await _can_access_session(current_user, session_id):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="无权访问此会话"
        )

    # 检查 Lead 是否存在
    state = await leads_manager.get_state(session_id)
    if not state:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="会话不存在"
        )

    lead = _find_lead(state, lead_id)
    if not lead:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Lead 不存在"
        )

    # 检查状态
    if lead.status != LeadStatus.AVAILABLE:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Lead 状态为 {lead.status}，无法完成"
        )

    # 确定新状态
    new_status = LeadStatus.COMPLETED if request.success else LeadStatus.FAILED

    # 更新状态
    result_data = {
        'success': request.success,
        'outcome': request.outcome or (
            "行动成功完成" if request.success else "行动未能完成"
        ),
        'notes': request.notes,
        'rewards': request.rewards or {},
    }

    success = await leads_manager.update_lead_status(
        session_id,
        lead_id,
        new_status,
        result=result_data
    )

    if not success:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="更新失败"
        )

    # 发布事件
    await event_bus.publish(
        session_id,
        'lead_completed',
        {
            'lead_id': lead_id,
            'status': new_status,
            'result': result_data,
        }
    )

    return CompleteLeadResponse(
        lead_id=lead_id,
        status=new_status.value,
        completed_at=datetime.now().isoformat(),
        result=result_data
    )
```

---

## 完成逻辑

```python
# app/services/leads/completion.py
from typing import Dict, Any, Optional
from app.core.types.leads import LeadItem, LeadStatus

class LeadCompletionHandler:
    """Lead 完成处理器"""

    def __init__(
        self,
        state_manager: 'LeadsStateManager',
        event_bus: 'EventBus',
        reward_manager: 'RewardManager'
    ):
        self.state_manager = state_manager
        self.event_bus = event_bus
        self.reward_manager = reward_manager

    async def handle_completion(
        self,
        session_id: str,
        lead: LeadItem,
        success: bool,
        outcome: Optional[str] = None,
        notes: Optional[str] = None,
        rewards: Optional[Dict[str, Any]] = None
    ) -> Dict[str, Any]:
        """处理 Lead 完成"""
        # 1. 确定结果
        result = self._determine_result(lead, success, outcome)

        # 2. 应用奖励/惩罚
        if success:
            await self._apply_rewards(session_id, lead, rewards)
        else:
            await self._apply_penalties(session_id, lead)

        # 3. 触发后续事件
        await self._trigger_followup_events(session_id, lead, success)

        # 4. 检查并生成新 Leads
        new_leads = await self._generate_followup_leads(
            session_id,
            lead,
            success
        )

        return {
            'result': result,
            'new_leads': new_leads,
        }

    def _determine_result(
        self,
        lead: LeadItem,
        success: bool,
        outcome: Optional[str]
    ) -> Dict[str, Any]:
        """确定完成结果"""
        if success:
            return {
                'success': True,
                'outcome': outcome or lead.action.expected_outcome or "行动成功",
                'lead_title': lead.title,
                'lead_type': lead.type,
            }
        else:
            return {
                'success': False,
                'outcome': outcome or "行动未能完成",
                'lead_title': lead.title,
                'lead_type': lead.type,
                'failure_consequence': lead.failure_consequence,
            }

    async def _apply_rewards(
        self,
        session_id: str,
        lead: LeadItem,
        additional_rewards: Optional[Dict[str, Any]]
    ):
        """应用奖励"""
        rewards = {}

        # 基础奖励
        if lead.category == 'investigate':
            rewards['xp'] = 10
        elif lead.category == 'action':
            rewards['xp'] = 15
        elif lead.category == 'social':
            rewards['xp'] = 12
        elif lead.category == 'combat':
            rewards['xp'] = 25

        # 额外奖励
        if additional_rewards:
            rewards.update(additional_rewards)

        # 应用奖励
        if rewards:
            await self.reward_manager.grant_rewards(session_id, rewards)

    async def _apply_penalties(
        self,
        session_id: str,
        lead: LeadItem
    ):
        """应用惩罚"""
        penalties = {}

        # 基础惩罚
        if lead.failure_consequence:
            if lead.failure_consequence.cost:
                penalties['cost'] = lead.failure_consequence.cost

        # 应用惩罚
        if penalties:
            await self.reward_manager.apply_penalties(session_id, penalties)

    async def _trigger_followup_events(
        self,
        session_id: str,
        lead: LeadItem,
        success: bool
    ):
        """触发后续事件"""
        event_type = 'lead_success' if success else 'lead_failure'

        await self.event_bus.publish(
            session_id,
            event_type,
            {
                'lead_id': lead.lead_id,
                'lead_type': lead.type,
                'category': lead.category,
                'related_clues': lead.related.clues,
                'related_npcs': lead.related.npcs,
            }
        )

    async def _generate_followup_leads(
        self,
        session_id: str,
        lead: LeadItem,
        success: bool
    ) -> List[LeadItem]:
        """生成后续 Leads"""
        # 获取游戏上下文
        context = await self._get_context(session_id)

        # 根据完成的 Lead 生成新的 Leads
        new_leads = []

        # 如果成功，基于结果生成新选项
        if success:
            if lead.type == 'clue_follow':
                # 线索跟进成功，可能发现更多线索
                new_leads.extend(await self._generate_clue_followups(lead, context))

            elif lead.type == 'npc_talk':
                # 对话成功，可能解锁新选项
                new_leads.extend(await self._generate_social_followups(lead, context))

        # 如果失败，提供替代方案
        else:
            new_leads.extend(await self._generate_alternatives(lead, context))

        return new_leads

    async def _generate_clue_followups(
        self,
        lead: LeadItem,
        context: 'GameContext'
    ) -> List[LeadItem]:
        """生成线索跟进后的 Leads"""
        leads = []

        # 检查是否发现新线索
        if lead.related.clues:
            for clue_id in lead.related.clues:
                clue = context.get_clue(clue_id)
                if clue and clue.leads_to:
                    leads.append(LeadItem(
                        title=f'调查新发现的线索',
                        description=f'从{clue.title}中发现了更多信息',
                        category='investigate',
                        type='clue_follow',
                        priority=70,
                        action={
                            'type': 'investigate',
                            'target': clue.leads_to,
                        },
                        related={
                            'clues': [clue.clue_id],
                        },
                        source={
                            'type': 'system',
                            'source_id': lead.lead_id,
                            'auto_generated': True,
                        }
                    ))

        return leads

    async def _generate_social_followups(
        self,
        lead: LeadItem,
        context: 'GameContext'
    ) -> List[LeadItem]:
        """生成社交跟进后的 Leads"""
        leads = []

        if lead.related.npcs:
            for npc_id in lead.related.npcs:
                npc = context.get_npc(npc_id)
                if npc and npc.has_quest:
                    leads.append(LeadItem(
                        title=f'接受{npc.name}的任务',
                        description=f'{npc.name}似乎有任务需要帮助',
                        category='social',
                        type='quest_accept',
                        priority=75,
                        action={
                            'type': 'talk',
                            'target': npc_id,
                        },
                        related={
                            'npcs': [npc_id],
                        },
                        source={
                            'type': 'system',
                            'source_id': lead.lead_id,
                            'auto_generated': True,
                        }
                    ))

        return leads

    async def _generate_alternatives(
        self,
        lead: LeadItem,
        context: 'GameContext'
    ) -> List[LeadItem]:
        """生成替代方案"""
        return [
            LeadItem(
                title=f'重试: {lead.title}',
                description='用不同的方法再次尝试',
                category=lead.category,
                type=lead.type,
                priority=lead.priority - 10,
                action=lead.action,
                related=lead.related,
                source={
                    'type': 'system',
                    'source_id': lead.lead_id,
                    'auto_generated': True,
                }
            )
        ]

    async def _get_context(self, session_id: str) -> 'GameContext':
        """获取游戏上下文"""
        # 实现略
        pass
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/api/leads.py` | 修改 | 添加完成端点 |
| `app/services/leads/completion.py` | 创建 | 完成处理器 |
| `docs/api/leads-complete.yaml` | 创建 | API 文档 |
| `tests/api/leads/test_complete.py` | 创建 | 单元测试 |

---

## 验收标准

- [ ] API 端点正常工作
- [ ] 状态更新正确
- [ ] 结果记录完整
- [ ] 奖励/惩罚应用正确
- [ ] 事件触发有效
- [ ] 后续 Leads 生成有效
- [ ] 单元测试通过

---

## 参考文档

- M6-006: Leads 移除逻辑
- M6-004: Leads 状态管理

---

**最后更新**: 2026-02-07
**状态**: [ ] 待开始
