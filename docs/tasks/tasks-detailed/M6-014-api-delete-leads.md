# M6-014: 实现 DELETE /game/leads/:id

**任务ID**: M6-014
**标题**: 实现 DELETE /game/leads/:id
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M6-006

---

## 任务描述

实现删除 Lead 的 API 端点。

---

## API 规范

```python
@router.delete("/{lead_id}")
async def delete_lead(
    session_id: str,
    lead_id: str,
    reason: str = Query("manual", description="删除原因"),
    current_user: dict = Depends(get_current_user),
    removal_manager: 'LeadsRemovalManager' = Depends(get_removal_manager)
):
    """删除 Lead"""
    if not await _can_access_session(current_user, session_id):
        raise HTTPException(status_code=403)

    success = await removal_manager.remove_lead(
        session_id,
        lead_id,
        RemovalReason(reason)
    )

    if not success:
        raise HTTPException(status_code=404)

    return {"status": "deleted", "lead_id": lead_id}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/api/leads.py` | 修改 | 添加删除端点 |
| `tests/api/leads/test_delete.py` | 创建 | 单元测试 |

---

## 验收标准

- [ ] API 端点正常工作
- [ ] 删除逻辑正确
- [ ] 权限验证有效
- [ ] 单元测试通过

---

**最后更新**: 2026-02-07
**状态**: [ ] 待开始
