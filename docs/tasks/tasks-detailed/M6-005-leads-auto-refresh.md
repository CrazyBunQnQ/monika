# M6-005: 实现 Leads 自动刷新

**任务ID**: M6-005
**标题**: 实现 Leads 自动刷新
**类型**: backend (后端开发)
**预估工时**: 4h
**依赖**: M6-004

---

## 任务描述

实现 Leads 自动刷新机制，确保玩家始终有可执行的行动选项。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-005-01 | 设计刷新触发器 | 刷新条件 | 30min |
| M6-005-02 | 实现定时刷新 | 定时任务 | 40min |
| M6-005-03 | 实现事件驱动刷新 | 事件监听 | 40min |
| M6-005-04 | 实现智能刷新算法 | 避免过度刷新 | 50min |
| M6-005-05 | 实现刷新通知 | WebSocket 推送 | 40min |
| M6-005-06 | 编写单元测试 | 测试覆盖 | 20min |

---

## 刷新管理器

```python
# app/services/leads/refresh_manager.py
from typing import Optional, List, Callable
from datetime import datetime, timedelta
from apscheduler.schedulers.asyncio import AsyncIOScheduler

from app.core.types.leads import LeadItem, GameContext
from app.services.leads.generator import LeadGenerator
from app.services.leads.state_manager import LeadsStateManager

class LeadsRefreshManager:
    """Leads 刷新管理器"""

    def __init__(
        self,
        generator: LeadGenerator,
        state_manager: LeadsStateManager,
        websocket_manager: 'WebSocketManager'
    ):
        self.generator = generator
        self.state_manager = state_manager
        self.websocket_manager = websocket_manager

        self.scheduler = AsyncIOScheduler()
        self.refresh_hooks: List[Callable] = []

        # 防抖：避免短时间内多次刷新
        self._refresh_cooldown: dict[str, datetime] = {}
        self._cooldown_period = timedelta(seconds=30)

    async def start(self):
        """启动刷新管理器"""
        self.scheduler.start()

    async def stop(self):
        """停止刷新管理器"""
        self.scheduler.shutdown()

    def setup_periodic_refresh(
        self,
        session_id: str,
        interval_seconds: int = 300
    ):
        """设置定时刷新"""
        self.scheduler.add_job(
            self._refresh_session,
            'interval',
            seconds=interval_seconds,
            args=[session_id],
            id=f'refresh_{session_id}',
            replace_existing=True
        )

    def cancel_periodic_refresh(self, session_id: str):
        """取消定时刷新"""
        self.scheduler.remove_job(f'refresh_{session_id}')

    async def trigger_refresh(
        self,
        session_id: str,
        reason: str = 'manual'
    ) -> bool:
        """触发刷新"""
        # 检查冷却时间
        if not self._can_refresh(session_id):
            return False

        state = await self.state_manager.get_state(session_id)
        if not state:
            return False

        # 执行刷新
        await self._refresh_session(session_id, reason)

        # 更新冷却时间
        self._refresh_cooldown[session_id] = datetime.now()

        return True

    def _can_refresh(self, session_id: str) -> bool:
        """检查是否可以刷新"""
        if session_id not in self._refresh_cooldown:
            return True

        last_refresh = self._refresh_cooldown[session_id]
        return datetime.now() - last_refresh > self._cooldown_period

    async def _refresh_session(
        self,
        session_id: str,
        reason: str = 'scheduled'
    ):
        """刷新会话的 Leads"""
        state = await self.state_manager.get_state(session_id)
        if not state:
            return

        # 清理过期
        await self.state_manager.cleanup_expired(session_id)

        # 检查是否需要刷新
        available_count = len(state.available)
        min_required = state.generation['min_active']

        if available_count >= min_required:
            return

        # 计算需要生成的数量
        target_count = state.generation['max_active']
        needed = target_count - available_count

        # 获取游戏上下文
        context = await self._get_game_context(session_id)

        # 生成新的 Leads
        new_leads = await self.generator.generate(context, count=needed)

        if new_leads:
            # 添加到状态
            await self.state_manager.add_leads(session_id, new_leads)

            # 执行钩子
            for hook in self.refresh_hooks:
                await hook(session_id, new_leads, reason)

            # 发送通知
            await self._notify_refresh(session_id, new_leads)

    async def on_game_event(
        self,
        session_id: str,
        event: dict
    ):
        """监听游戏事件并触发刷新"""
        event_type = event.get('type')

        # 定义需要刷新的事件
        refresh_triggers = {
            'clue_discovered': True,
            'scene_changed': True,
            'npc_met': True,
            'action_failed': True,
            'quest_updated': True,
        }

        if refresh_triggers.get(event_type):
            await self.trigger_refresh(session_id, reason=f'event:{event_type}')

    def register_hook(self, hook: Callable):
        """注册刷新钩子"""
        self.refresh_hooks.append(hook)

    async def _get_game_context(self, session_id: str) -> GameContext:
        """获取游戏上下文"""
        # 从会话中获取游戏状态
        # 实现略
        pass

    async def _notify_refresh(
        self,
        session_id: str,
        new_leads: List[LeadItem]
    ):
        """通知前端刷新"""
        await self.websocket_manager.broadcast(
            session_id,
            {
                'type': 'leads_refreshed',
                'leads': [lead.dict() for lead in new_leads],
                'timestamp': datetime.now().isoformat(),
            }
        )
```

---

## 智能刷新算法

```python
# app/services/leads/smart_refresh.py
from typing import List
from datetime import datetime

class SmartRefreshStrategy:
    """智能刷新策略"""

    def __init__(self):
        self.refresh_history: dict[str, List[datetime]] = {}
        self.max_history = 10

    def should_refresh(
        self,
        session_id: str,
        current_count: int,
        min_required: int,
        context: dict
    ) -> tuple[bool, str]:
        """
        判断是否应该刷新

        Returns:
            (should_refresh, reason)
        """
        # 1. 数量不足
        if current_count < min_required:
            return True, 'insufficient_count'

        # 2. 玩家困惑
        if context.get('player_confused'):
            return True, 'player_confused'

        # 3. 长时间无操作
        if self._is_stale(session_id):
            return True, 'stale_session'

        # 4. 高优先级 Lead 过少
        high_priority = self._count_high_priority(context.get('available_leads', []))
        if high_priority < 1:
            return True, 'low_priority'

        # 5. 刷新频率控制
        if self._is_refreshing_too_frequently(session_id):
            return False, 'rate_limit'

        return False, 'not_needed'

    def _is_stale(self, session_id: str) -> bool:
        """检查会话是否陈旧"""
        if session_id not in self.refresh_history:
            return False

        history = self.refresh_history[session_id]
        if not history:
            return True

        last_refresh = history[-1]
        return (datetime.now() - last_refresh) > timedelta(minutes=15)

    def _is_refreshing_too_frequently(self, session_id: str) -> bool:
        """检查刷新是否过于频繁"""
        if session_id not in self.refresh_history:
            return False

        history = self.refresh_history[session_id]
        if len(history) < 3:
            return False

        # 检查最近 3 次刷新
        recent = history[-3:]
        interval = (recent[-1] - recent[0]).total_seconds()

        # 如果 3 次刷新在 2 分钟内完成，认为过于频繁
        return interval < 120

    def _count_high_priority(self, leads: List) -> int:
        """统计高优先级 Lead 数量"""
        return sum(1 for lead in leads if lead.get('priority', 0) > 70)

    def record_refresh(self, session_id: str):
        """记录刷新"""
        if session_id not in self.refresh_history:
            self.refresh_history[session_id] = []

        self.refresh_history[session_id].append(datetime.now())

        # 保持历史记录大小
        if len(self.refresh_history[session_id]) > self.max_history:
            self.refresh_history[session_id] = (
                self.refresh_history[session_id][-self.max_history:]
            )
```

---

## 事件监听器

```python
# app/services/leads/event_listener.py
from typing import Callable, dict

class GameEventListener:
    """游戏事件监听器"""

    def __init__(self, refresh_manager: LeadsRefreshManager):
        self.refresh_manager = refresh_manager
        self.handlers: dict[str, List[Callable]] = {}

    def register_handler(
        self,
        event_type: str,
        handler: Callable
    ):
        """注册事件处理器"""
        if event_type not in self.handlers:
            self.handlers[event_type] = []
        self.handlers[event_type].append(handler)

    async def handle_event(
        self,
        session_id: str,
        event: dict
    ):
        """处理游戏事件"""
        event_type = event.get('type')

        # 调用注册的处理器
        if event_type in self.handlers:
            for handler in self.handlers[event_type]:
                await handler(session_id, event)

        # 触发刷新
        await self.refresh_manager.on_game_event(session_id, event)

# FastAPI 集成

# app/api/events.py
from fastapi import APIRouter, WebSocket
from app.services.leads.event_listener import GameEventListener

router = APIRouter()
event_listener = None  # 在应用启动时初始化

@router.post('/game/{session_id}/event')
async def emit_game_event(session_id: str, event: dict):
    """接收游戏事件"""
    if event_listener:
        await event_listener.handle_event(session_id, event)
    return {'status': 'ok'}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/leads/refresh_manager.py` | 创建 | 刷新管理器 |
| `app/services/leads/smart_refresh.py` | 创建 | 智能刷新策略 |
| `app/services/leads/event_listener.py` | 创建 | 事件监听器 |
| `app/api/events.py` | 修改 | 事件接口 |
| `tests/services/leads/test_refresh.py` | 创建 | 单元测试 |

---

## 验收标准

- [ ] 定时刷新正常工作
- [ ] 事件驱动刷新有效
- [ ] 智能刷新避免过度刷新
- [ ] 防抖机制有效
- [ ] WebSocket 通知正常
- [ ] 单元测试通过

---

## 参考文档

- M6-002: Leads 生成算法
- M6-004: Leads 状态管理

---

**最后更新**: 2026-02-07
**状态**: [ ] 待开始
