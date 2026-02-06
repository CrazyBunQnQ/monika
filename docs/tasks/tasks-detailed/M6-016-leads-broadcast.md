# M6-016: 实现 Leads 广播

**任务ID**: M6-016
**标题**: 实现 Leads 广播
**类型**: backend (后端开发)
**预估工时**: 4h
**依赖**: M6-014

---

## 任务描述

实现 Leads 变化的 WebSocket 广播，实时通知前端。

---

## 广播管理器

```python
# app/services/leads/broadcast.py
class LeadsBroadcastManager:
    """Leads 广播管理器"""

    def __init__(self, websocket_manager: 'WebSocketManager'):
        self.ws_manager = websocket_manager

    async def broadcast_lead_added(
        self,
        session_id: str,
        lead: 'LeadItem'
    ):
        """广播 Lead 添加"""
        await self.ws_manager.broadcast(session_id, {
            "type": "lead_added",
            "lead": lead.dict()
        })

    async def broadcast_lead_updated(
        self,
        session_id: str,
        lead_id: str,
        changes: dict
    ):
        """广播 Lead 更新"""
        await self.ws_manager.broadcast(session_id, {
            "type": "lead_updated",
            "lead_id": lead_id,
            "changes": changes
        })

    async def broadcast_lead_removed(
        self,
        session_id: str,
        lead_id: str,
        reason: str
    ):
        """广播 Lead 移除"""
        await self.ws_manager.broadcast(session_id, {
            "type": "lead_removed",
            "lead_id": lead_id,
            "reason": reason
        })
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/leads/broadcast.py` | 创建 | 广播管理器 |
| `tests/services/leads/test_broadcast.py` | 创建 | 单元测试 |

---

## 验收标准

- [ ] 广播机制正常工作
- [ ] 所有事件正确广播
- [ ] WebSocket 连接稳定
- [ ] 单元测试通过

---

**最后更新**: 2026-02-07
**状态**: [ ] 待开始
