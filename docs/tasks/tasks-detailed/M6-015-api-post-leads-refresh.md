# M6-015: 实现 POST /game/leads/refresh

**任务ID**: M6-015
**标题**: 实现 POST /game/leads/refresh
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M6-005

---

## 任务描述

实现手动刷新 Leads 的 API 端点。

---

## API 规范

```python
@router.post("/refresh")
async def refresh_leads(
    session_id: str,
    force: bool = Query(False, description="强制刷新"),
    current_user: dict = Depends(get_current_user),
    refresh_manager: 'LeadsRefreshManager' = Depends(get_refresh_manager)
):
    """刷新 Leads"""
    if not await _can_access_session(current_user, session_id):
        raise HTTPException(status_code=403)

    success = await refresh_manager.trigger_refresh(
        session_id,
        reason="manual_force" if force else "manual"
    )

    return {"status": "refreshed"}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/api/leads.py` | 修改 | 添加刷新端点 |
| `tests/api/leads/test_refresh.py` | 创建 | 单元测试 |

---

## 验收标准

- [ ] API 端点正常工作
- [ ] 刷新逻辑正确
- [ ] 强制刷新有效
- [ ] 单元测试通过

---

**最后更新**: 2026-02-07
**状态**: [ ] 待开始
