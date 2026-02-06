# M6-004: 实现 Leads 状态管理

**任务ID**: M6-004
**标题**: 实现 Leads 状态管理
**类型**: backend (后端开发)
**预估工时**: 4h
**依赖**: M6-003

---

## 任务描述

实现 Leads 状态管理系统，跟踪 Lead 的生命周期，管理可用和历史状态。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-004-01 | 设计状态机 | Lead 状态转换 | 40min |
| M6-004-02 | 实现状态管理器 | 核心管理逻辑 | 60min |
| M6-004-03 | 实现状态持久化 | 数据库存储 | 50min |
| M6-004-04 | 实现状态查询接口 | 查询 API | 30min |
| M6-004-05 | 实现状态转换钩子 | 状态变化回调 | 40min |
| M6-004-06 | 编写单元测试 | 测试覆盖 | 20min |

---

## 状态机设计

```python
# app/services/leads/state_machine.py
from enum import Enum
from typing import Optional, List, Callable
from dataclasses import dataclass

class LeadStatus(str, Enum):
    """Lead 状态"""
    AVAILABLE = "available"       # 可执行
    IN_PROGRESS = "in_progress"   # 进行中
    COMPLETED = "completed"       # 已完成
    FAILED = "failed"            # 已失败
    EXPIRED = "expired"          # 已过期
    BLOCKED = "blocked"          # 被阻塞

@dataclass
class StateTransition:
    """状态转换"""
    from_status: LeadStatus
    to_status: LeadStatus
    condition: Optional[Callable] = None
    action: Optional[Callable] = None

class LeadStateMachine:
    """Lead 状态机"""

    # 定义合法的状态转换
    TRANSITIONS = {
        LeadStatus.AVAILABLE: [
            LeadStatus.IN_PROGRESS,
            LeadStatus.EXPIRED,
            LeadStatus.BLOCKED,
        ],
        LeadStatus.IN_PROGRESS: [
            LeadStatus.COMPLETED,
            LeadStatus.FAILED,
            LeadStatus.AVAILABLE,  # 可撤销
        ],
        LeadStatus.BLOCKED: [
            LeadStatus.AVAILABLE,  # 解除阻塞
            LeadStatus.EXPIRED,
        ],
        # COMPLETED, FAILED, EXPIRED 是终态
    }

    def __init__(self):
        self.hooks = {
            'before_transition': [],
            'after_transition': [],
        }

    def can_transition(
        self,
        from_status: LeadStatus,
        to_status: LeadStatus
    ) -> bool:
        """检查是否可以转换"""
        return to_status in self.TRANSITIONS.get(from_status, [])

    def transition(
        self,
        lead: 'LeadItem',
        to_status: LeadStatus,
        context: Optional[dict] = None
    ) -> bool:
        """执行状态转换"""
        from_status = lead.status

        # 检查是否可以转换
        if not self.can_transition(from_status, to_status):
            return False

        # 执行前置钩子
        for hook in self.hooks['before_transition']:
            hook(lead, from_status, to_status, context)

        # 更新状态
        lead.status = to_status

        # 执行后置钩子
        for hook in self.hooks['after_transition']:
            hook(lead, from_status, to_status, context)

        return True

    def register_hook(
        self,
        event: str,
        hook: Callable
    ):
        """注册钩子"""
        if event in self.hooks:
            self.hooks[event].append(hook)
```

---

## 状态管理器

```python
# app/services/leads/state_manager.py
from typing import List, Optional, Dict, Any
from datetime import datetime

from app.core.types.leads import LeadItem, LeadsState, LeadStatus
from app.services.leads.state_machine import LeadStateMachine

class LeadsStateManager:
    """Leads 状态管理器"""

    def __init__(self, storage: 'LeadStorage'):
        self.storage = storage
        self.state_machine = LeadStateMachine()
        self._setup_hooks()

    def _setup_hooks(self):
        """设置钩子"""
        self.state_machine.register_hook(
            'after_transition',
            self._on_status_changed
        )

    async def get_state(
        self,
        session_id: str
    ) -> Optional[LeadsState]:
        """获取会话状态"""
        return await self.storage.load_state(session_id)

    async def initialize_state(
        self,
        session_id: str,
        config: Optional[dict] = None
    ) -> LeadsState:
        """初始化状态"""
        state = LeadsState(
            session_id=session_id,
            updated_at=datetime.now(),
            available=[],
            history=[],
            queue={
                'waiting': [],
                'active': None,
                'max_concurrent': 3,
            },
            generation={
                'min_active': 2,
                'max_active': 4,
                'refresh_interval': 300,
                'auto_refresh': True,
            },
            settings=config or self._default_settings(),
            stats={
                'total_generated': 0,
                'total_completed': 0,
                'completion_rate': 0.0,
                'average_time_to_complete': 0.0,
            },
        )

        await self.storage.save_state(state)
        return state

    async def add_leads(
        self,
        session_id: str,
        leads: List[LeadItem]
    ) -> int:
        """添加 Leads"""
        state = await self.get_state(session_id)
        if not state:
            state = await self.initialize_state(session_id)

        # 过滤已存在的
        new_leads = [
            lead for lead in leads
            if not self._exists(state, lead)
        ]

        # 添加到可用列表
        state.available.extend(new_leads)

        # 更新队列
        for lead in new_leads:
            state.queue['waiting'].append(lead.lead_id)

        # 更新统计
        state.stats['total_generated'] += len(new_leads)
        state.updated_at = datetime.now()

        await self.storage.save_state(state)
        return len(new_leads)

    async def update_lead_status(
        self,
        session_id: str,
        lead_id: str,
        new_status: LeadStatus,
        result: Optional[dict] = None
    ) -> bool:
        """更新 Lead 状态"""
        state = await self.get_state(session_id)
        if not state:
            return False

        # 查找 Lead
        lead = self._find_lead(state, lead_id)
        if not lead:
            return False

        # 执行状态转换
        success = self.state_machine.transition(
            lead,
            new_status,
            {'session_id': session_id}
        )

        if not success:
            return False

        # 如果是完成或失败，移到历史
        if new_status in (LeadStatus.COMPLETED, LeadStatus.FAILED):
            state.available.remove(lead)
            state.history.append({
                'lead': lead,
                'completed_at': datetime.now(),
                'result': result,
            })

            # 更新统计
            state.stats['total_completed'] += 1
            state.stats['completion_rate'] = (
                state.stats['total_completed'] /
                state.stats['total_generated']
            )

        # 更新队列
        if lead_id in state.queue['waiting']:
            state.queue['waiting'].remove(lead_id)
        if state.queue['active'] == lead_id:
            state.queue['active'] = None

        state.updated_at = datetime.now()
        await self.storage.save_state(state)
        return True

    async def get_available_leads(
        self,
        session_id: str,
        limit: int = 10
    ) -> List[LeadItem]:
        """获取可用 Leads"""
        state = await self.get_state(session_id)
        if not state:
            return []

        # 按优先级排序
        sorted_leads = sorted(
            state.available,
            key=lambda x: x.priority,
            reverse=True
        )

        return sorted_leads[:limit]

    async def cleanup_expired(
        self,
        session_id: str
    ) -> int:
        """清理过期的 Leads"""
        state = await self.get_state(session_id)
        if not state:
            return 0

        now = datetime.now()
        expired = []

        for lead in state.available:
            if lead.validity.expires_at and lead.validity.expires_at < now:
                expired.append(lead)

        for lead in expired:
            await self.update_lead_status(
                session_id,
                lead.lead_id,
                LeadStatus.EXPIRED
            )

        return len(expired)

    def _exists(self, state: LeadsState, lead: LeadItem) -> bool:
        """检查是否已存在"""
        return any(
            existing.lead_id == lead.lead_id
            for existing in state.available
        )

    def _find_lead(
        self,
        state: LeadsState,
        lead_id: str
    ) -> Optional[LeadItem]:
        """查找 Lead"""
        for lead in state.available:
            if lead.lead_id == lead_id:
                return lead
        return None

    def _default_settings(self) -> dict:
        """默认设置"""
        return {
            'show_priority': True,
            'show_urgency': True,
            'group_by_category': True,
            'sort_by': 'priority',
        }

    def _on_status_changed(
        self,
        lead: LeadItem,
        from_status: LeadStatus,
        to_status: LeadStatus,
        context: dict
    ):
        """状态变化回调"""
        # 可以在这里发送通知、记录日志等
        pass
```

---

## 持久化层

```python
# app/services/leads/storage.py
from typing import Optional
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.types.leads import LeadsState
from app.db.models.lead import LeadModel

class LeadStorage:
    """Lead 持久化存储"""

    def __init__(self, db: AsyncSession):
        self.db = db

    async def save_state(self, state: LeadsState) -> bool:
        """保存状态"""
        try:
            # 删除旧状态
            await self.db.execute(
                delete(LeadModel).where(
                    LeadModel.session_id == state.session_id
                )
            )

            # 插入新状态
            for lead in state.available:
                model = LeadModel(
                    lead_id=lead.lead_id,
                    session_id=state.session_id,
                    data=lead.dict(),
                )
                self.db.add(model)

            await self.db.commit()
            return True
        except Exception as e:
            await self.db.rollback()
            raise e

    async def load_state(
        self,
        session_id: str
    ) -> Optional[LeadsState]:
        """加载状态"""
        result = await self.db.execute(
            select(LeadModel).where(
                LeadModel.session_id == session_id
            )
        )
        rows = result.scalars().all()

        if not rows:
            return None

        # 重建 LeadsState
        state = LeadsState(
            session_id=session_id,
            available=[row.data for row in rows],
            # ... 其他字段
        )

        return state
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/leads/state_machine.py` | 创建 | 状态机 |
| `app/services/leads/state_manager.py` | 创建 | 状态管理器 |
| `app/services/leads/storage.py` | 创建 | 持久化层 |
| `app/db/models/lead.py` | 创建 | 数据模型 |
| `tests/services/leads/test_state_manager.py` | 创建 | 单元测试 |

---

## 验收标准

- [ ] 状态转换正确
- [ ] 钩子机制可用
- [ ] 持久化正常工作
- [ ] 查询接口完整
- [ ] 过期清理有效
- [ ] 单元测试通过

---

## 参考文档

- M6-001: Leads 数据结构
- M6-003: Leads 优先级排序

---

**最后更新**: 2026-02-07
**状态**: [ ] 待开始
