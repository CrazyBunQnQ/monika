# M5-005: 实现条件触发器系统

**任务ID**: M5-005
**标题**: 实现条件触发器系统
**类型**: backend (后端开发)
**预估工时**: 2.5h
**依赖**: M1-080, M2-002

---

## 任务描述

实现事件触发器系统，允许在特定条件下自动触发事件、通知或动作，如 SAN 值过低时警告、特定物品获取时触发剧情等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M5-005-01 | 设计触发器数据模型 | Trigger Schema | 25min |
| M5-005-02 | 实现条件解析器 | Condition Parser | 35min |
| M5-005-03 | 实现动作执行器 | Action Executor | 30min |
| M5-005-04 | 实现触发器引擎 | Trigger Engine | 30min |
| M5-005-05 | 实现 WebSocket 通知 | WS Notification | 20min |
| M5-005-06 | 实现触发器 API | API | 20min |
| M5-005-07 | 编写触发器测试 | 测试覆盖 | 15min |

---

## 触发器数据模型

```python
# app/db/models/trigger.py
from sqlalchemy import Column, String, Boolean, JSON, ForeignKey, DateTime
from sqlalchemy.orm import relationship
from sqlalchemy.sql import func
from app.db.database import Base

class Trigger(Base):
    """触发器"""
    __tablename__ = 'triggers'

    id = Column(String, primary_key=True, index=True)
    room_id = Column(String, ForeignKey('rooms.id'), nullable=False, index=True)
    campaign_id = Column(String, ForeignKey('campaigns.id'), nullable=False, index=True)

    # 基本信息
    name = Column(String, nullable=False)
    description = Column(String)
    is_enabled = Column(Boolean, default=True, nullable=False)

    # 触发条件
    condition = Column(JSON, nullable=False)  # 条件表达式
    condition_type = Column(String, nullable=False)  # event, state, time, custom

    # 触发动作
    actions = Column(JSON, nullable=False)  # 动作列表

    # 触发限制
    max_triggers = Column(Integer)  # 最大触发次数，null = 无限
    trigger_count = Column(Integer, default=0, nullable=False)
    cooldown_seconds = Column(Integer)  # 冷却时间
    last_triggered = Column(DateTime)

    # 创建者
    created_by = Column(String, ForeignKey('users.id'), nullable=False)

    # 时间
    created_at = Column(DateTime, default=func.now(), nullable=False)
    updated_at = Column(DateTime, default=func.now(), onupdate=func.now(), nullable=False)

    # 关系
    room = relationship("Room", back_populates="triggers")
    campaign = relationship("Campaign", back_populates="triggers")
    creator = relationship("User", back_populates="created_triggers")

    def __repr__(self):
        return f"<Trigger {self.name}>"
```

---

## 条件表达式结构

```python
# 条件表达式示例
condition_examples = {
    # SAN 值低于 20
    "san_low": {
        "type": "state",
        "field": "san",
        "operator": "lt",
        "value": 20,
    },

    # 获得特定物品
    "item_acquired": {
        "type": "event",
        "event_type": "item_acquired",
        "filters": {
            "item_id": "ancient_book",
        },
    },

    # HP 低于 10 且处于战斗状态
    "critical_combat": {
        "type": "and",
        "conditions": [
            {
                "type": "state",
                "field": "hp",
                "operator": "lt",
                "value": 10,
            },
            {
                "type": "state",
                "field": "in_combat",
                "value": True,
            },
        ],
    },

    # 时间到达特定点
    "time_trigger": {
        "type": "time",
        "game_time": "1920-01-15 20:00",
    },

    # 组合条件：OR
    "danger_state": {
        "type": "or",
        "conditions": [
            {"type": "state", "field": "san", "operator": "lt", "value": 10},
            {"type": "state", "field": "hp", "operator": "lt", "value": 5},
        ],
    },
}
```

---

## 动作定义

```python
# 动作类型定义
action_types = {
    # 发送通知
    "notification": {
        "type": "notification",
        "message": "SAN 值过低！",
        "recipients": ["all"],  # all, kp, player, specific_user_id
        "priority": "high",  # low, normal, high, urgent
    },

    # WebSocket 广播
    "websocket": {
        "type": "websocket",
        "event": "custom_event",
        "data": {...},
        "recipients": ["all"],
    },

    # 修改角色状态
    "modify_character": {
        "type": "modify_character",
        "character_id": "...",
        "modifications": {
            "san": -5,
            "hp": -10,
        },
    },

    # 添加日志
    "log": {
        "type": "log",
        "level": "warning",
        "message": "触发器执行：xxx",
    },

    # 执行命令
    "command": {
        "type": "command",
        "command": "/roll 1d100",
        "execute_as": "system",  # system, kp, player
    },

    # 触发其他触发器
    "cascade": {
        "type": "cascade",
        "trigger_ids": ["trigger_1", "trigger_2"],
    },
}
```

---

## 条件解析器

```python
# app/services/trigger/condition_parser.py
from typing import Dict, Any, List
from sqlalchemy.orm import Session

class ConditionParser:
    """条件解析器"""

    def __init__(self, db: Session):
        self.db = db

    async def evaluate(
        self,
        condition: Dict[str, Any],
        context: Dict[str, Any],
    ) -> bool:
        """评估条件是否满足"""
        condition_type = condition.get("type")

        if condition_type == "state":
            return await self._evaluate_state(condition, context)
        elif condition_type == "event":
            return await self._evaluate_event(condition, context)
        elif condition_type == "time":
            return await self._evaluate_time(condition, context)
        elif condition_type == "and":
            return await self._evaluate_and(condition, context)
        elif condition_type == "or":
            return await self._evaluate_or(condition, context)
        elif condition_type == "not":
            return await self._evaluate_not(condition, context)
        else:
            return False

    async def _evaluate_state(
        self,
        condition: Dict[str, Any],
        context: Dict[str, Any],
    ) -> bool:
        """评估状态条件"""
        field = condition["field"]
        operator = condition["operator"]
        value = condition["value"]

        # 从上下文获取字段值
        actual_value = context.get("state", {}).get(field)

        if actual_value is None:
            return False

        # 执行比较
        if operator == "eq":
            return actual_value == value
        elif operator == "ne":
            return actual_value != value
        elif operator == "gt":
            return actual_value > value
        elif operator == "gte":
            return actual_value >= value
        elif operator == "lt":
            return actual_value < value
        elif operator == "lte":
            return actual_value <= value
        elif operator == "in":
            return actual_value in value
        elif operator == "contains":
            return value in actual_value
        else:
            return False

    async def _evaluate_event(
        self,
        condition: Dict[str, Any],
        context: Dict[str, Any],
    ) -> bool:
        """评估事件条件"""
        event = context.get("event", {})
        event_type = condition.get("event_type")
        filters = condition.get("filters", {})

        # 检查事件类型
        if event_type and event.get("type") != event_type:
            return False

        # 检查过滤条件
        for key, value in filters.items():
            if event.get(key) != value:
                return False

        return True

    async def _evaluate_time(
        self,
        condition: Dict[str, Any],
        context: Dict[str, Any],
    ) -> bool:
        """评估时间条件"""
        from datetime import datetime

        trigger_time = condition.get("game_time")
        current_time = context.get("game_time")

        if not trigger_time or not current_time:
            return False

        try:
            trigger_dt = datetime.fromisoformat(trigger_time)
            current_dt = datetime.fromisoformat(current_time)
            return current_dt >= trigger_dt
        except:
            return False

    async def _evaluate_and(
        self,
        condition: Dict[str, Any],
        context: Dict[str, Any],
    ) -> bool:
        """评估 AND 条件"""
        conditions = condition.get("conditions", [])
        for cond in conditions:
            if not await self.evaluate(cond, context):
                return False
        return True

    async def _evaluate_or(
        self,
        condition: Dict[str, Any],
        context: Dict[str, Any],
    ) -> bool:
        """评估 OR 条件"""
        conditions = condition.get("conditions", [])
        for cond in conditions:
            if await self.evaluate(cond, context):
                return True
        return False

    async def _evaluate_not(
        self,
        condition: Dict[str, Any],
        context: Dict[str, Any],
    ) -> bool:
        """评估 NOT 条件"""
        inner = condition.get("condition")
        return not await self.evaluate(inner, context)
```

---

## 触发器引擎

```python
# app/services/trigger/trigger_engine.py
from typing import List, Dict, Any
from sqlalchemy.orm import Session
from datetime import datetime, timedelta

from app.db.models.trigger import Trigger
from app.services.trigger.condition_parser import ConditionParser
from app.services.trigger/action_executor import ActionExecutor

class TriggerEngine:
    """触发器引擎"""

    def __init__(self, db: Session):
        self.db = db
        self.condition_parser = ConditionParser(db)
        self.action_executor = ActionExecutor(db)

    async def process_event(
        self,
        room_id: str,
        event: Dict[str, Any],
        context: Dict[str, Any],
    ) -> List[Dict[str, Any]]:
        """处理事件，检查并执行触发器"""
        # 获取房间的所有启用触发器
        triggers = self.db.query(Trigger)\
            .filter(
                Trigger.room_id == room_id,
                Trigger.is_enabled == True,
            )\
            .all()

        results = []

        for trigger in triggers:
            # 检查触发限制
            if not self._can_trigger(trigger):
                continue

            # 构建上下文
            trigger_context = {
                **context,
                "event": event,
            }

            # 评估条件
            if await self.condition_parser.evaluate(trigger.condition, trigger_context):
                # 执行动作
                result = await self._execute_trigger(trigger, trigger_context)
                results.append(result)

        return results

    async def process_state_change(
        self,
        room_id: str,
        state_changes: Dict[str, Any],
        context: Dict[str, Any],
    ) -> List[Dict[str, Any]]:
        """处理状态变化"""
        # 类似 process_event，但针对状态变化
        return await self.process_event(room_id, {"type": "state_change", **state_changes}, context)

    def _can_trigger(self, trigger: Trigger) -> bool:
        """检查触发器是否可以触发"""
        # 检查最大触发次数
        if trigger.max_triggers and trigger.trigger_count >= trigger.max_triggers:
            return False

        # 检查冷却时间
        if trigger.cooldown_seconds and trigger.last_triggered:
            cooldown_end = trigger.last_triggered + timedelta(seconds=trigger.cooldown_seconds)
            if datetime.now() < cooldown_end:
                return False

        return True

    async def _execute_trigger(
        self,
        trigger: Trigger,
        context: Dict[str, Any],
    ) -> Dict[str, Any]:
        """执行触发器"""
        results = []

        # 执行所有动作
        for action in trigger.actions:
            result = await self.action_executor.execute(action, context)
            results.append(result)

        # 更新触发器状态
        trigger.trigger_count += 1
        trigger.last_triggered = datetime.now()
        self.db.commit()

        return {
            "trigger_id": trigger.id,
            "trigger_name": trigger.name,
            "action_results": results,
            "timestamp": datetime.now().isoformat(),
        }
```

---

## 动作执行器

```python
# app/services/trigger/action_executor.py
from typing import Dict, Any
from sqlalchemy.orm import Session

class ActionExecutor:
    """动作执行器"""

    def __init__(self, db: Session):
        self.db = db

    async def execute(
        self,
        action: Dict[str, Any],
        context: Dict[str, Any],
    ) -> Dict[str, Any]:
        """执行动作"""
        action_type = action.get("type")

        if action_type == "notification":
            return await self._execute_notification(action, context)
        elif action_type == "websocket":
            return await self._execute_websocket(action, context)
        elif action_type == "modify_character":
            return await self._execute_modify_character(action, context)
        elif action_type == "log":
            return await self._execute_log(action, context)
        elif action_type == "command":
            return await self._execute_command(action, context)
        elif action_type == "cascade":
            return await self._execute_cascade(action, context)
        else:
            return {"success": False, "error": f"Unknown action type: {action_type}"}

    async def _execute_notification(
        self,
        action: Dict[str, Any],
        context: Dict[str, Any],
    ) -> Dict[str, Any]:
        """执行通知动作"""
        message = action["message"]
        recipients = action.get("recipients", ["all"])
        priority = action.get("priority", "normal")

        # TODO: 发送通知给用户
        # 可能通过 WebSocket 或其他机制

        return {
            "success": True,
            "type": "notification",
            "message": message,
            "recipients": recipients,
        }

    async def _execute_websocket(
        self,
        action: Dict[str, Any],
        context: Dict[str, Any],
    ) -> Dict[str, Any]:
        """执行 WebSocket 广播动作"""
        event = action["event"]
        data = action.get("data", {})
        recipients = action.get("recipients", ["all"])

        # TODO: 通过 WebSocket 广播事件

        return {
            "success": True,
            "type": "websocket",
            "event": event,
            "data": data,
        }

    async def _execute_modify_character(
        self,
        action: Dict[str, Any],
        context: Dict[str, Any],
    ) -> Dict[str, Any]:
        """执行修改角色动作"""
        character_id = action["character_id"]
        modifications = action["modifications"]

        # TODO: 修改角色数据
        # 可能需要调用角色服务

        return {
            "success": True,
            "type": "modify_character",
            "character_id": character_id,
            "modifications": modifications,
        }

    async def _execute_log(
        self,
        action: Dict[str, Any],
        context: Dict[str, Any],
    ) -> Dict[str, Any]:
        """执行日志动作"""
        level = action.get("level", "info")
        message = action["message"]

        # TODO: 记录日志

        return {
            "success": True,
            "type": "log",
            "level": level,
            "message": message,
        }

    async def _execute_command(
        self,
        action: Dict[str, Any],
        context: Dict[str, Any],
    ) -> Dict[str, Any]:
        """执行命令动作"""
        command = action["command"]
        execute_as = action.get("execute_as", "system")

        # TODO: 执行命令

        return {
            "success": True,
            "type": "command",
            "command": command,
        }

    async def _execute_cascade(
        self,
        action: Dict[str, Any],
        context: Dict[str, Any],
    ) -> Dict[str, Any]:
        """执行级联触发动作"""
        trigger_ids = action.get("trigger_ids", [])

        # TODO: 触发其他触发器
        # 需要小心避免无限循环

        return {
            "success": True,
            "type": "cascade",
            "trigger_ids": trigger_ids,
        }
```

---

## 触发器 API

```python
# app/api/triggers.py
from fastapi import APIRouter, Depends, Body
from sqlalchemy.orm import Session
from typing import List

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.schemas.trigger import TriggerCreate, TriggerResponse, TriggerUpdate
from app.services.trigger.trigger_engine import TriggerEngine

router = APIRouter(prefix="/triggers", tags=["triggers"])

@router.post("/{room_id}")
async def create_trigger(
    room_id: str,
    trigger_data: TriggerCreate,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """创建触发器"""
    trigger = Trigger(
        id=generate_id('trigger'),
        room_id=room_id,
        campaign_id=trigger_data.campaign_id,
        name=trigger_data.name,
        description=trigger_data.description,
        condition=trigger_data.condition,
        condition_type=trigger_data.condition_type,
        actions=trigger_data.actions,
        max_triggers=trigger_data.max_triggers,
        cooldown_seconds=trigger_data.cooldown_seconds,
        created_by=current_user.id,
    )

    db.add(trigger)
    db.commit()
    db.refresh(trigger)

    return TriggerResponse.from_orm(trigger)

@router.get("/{room_id}")
async def list_triggers(
    room_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取房间触发器列表"""
    triggers = db.query(Trigger)\
        .filter(Trigger.room_id == room_id)\
        .all()

    return [TriggerResponse.from_orm(t) for t in triggers]

@router.put("/{trigger_id}")
async def update_trigger(
    trigger_id: str,
    trigger_data: TriggerUpdate,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """更新触发器"""
    trigger = db.query(Trigger)\
        .filter(Trigger.id == trigger_id)\
        .first()

    if not trigger:
        raise HTTPException(status_code=404, detail="触发器不存在")

    for key, value in trigger_data.dict(exclude_unset=True).items():
        setattr(trigger, key, value)

    db.commit()
    db.refresh(trigger)

    return TriggerResponse.from_orm(trigger)

@router.delete("/{trigger_id}")
async def delete_trigger(
    trigger_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """删除触发器"""
    trigger = db.query(Trigger)\
        .filter(Trigger.id == trigger_id)\
        .first()

    if not trigger:
        raise HTTPException(status_code=404, detail="触发器不存在")

    db.delete(trigger)
    db.commit()

    return {"message": "触发器已删除"}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/db/models/trigger.py` | 创建 | 触发器数据模型 |
| `app/services/trigger/condition_parser.py` | 创建 | 条件解析器 |
| `app/services/trigger/action_executor.py` | 创建 | 动作执行器 |
| `app/services/trigger/trigger_engine.py` | 创建 | 触发器引擎 |
| `app/api/triggers.py` | 创建 | 触发器 API |
| `app/schemas/trigger.py` | 创建 | 触发器 Schema |

---

## 验收标准

- [ ] 条件解析正确
- [ ] 动作执行成功
- [ ] 触发器引擎稳定
- [ ] WebSocket 通知及时
- [ ] 冷却时间有效
- [ ] 级联触发安全

---

## 参考文档

- M1-080: 事件日志系统
- M2-002: WebSocket 事件系统

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
