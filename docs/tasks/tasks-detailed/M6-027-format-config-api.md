# M6-027: 实现格式配置 API

**任务ID**: M6-027
**标题**: 实现格式配置 API
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M6-026

---

## 任务描述

实现配置输出格式的 API 端点。

---

## API 实现

```python
# app/api/output.py
@router.put("/game/{session_id}/output/config")
async def set_output_config(
    session_id: str,
    config: OutputConfig,
    current_user: dict = Depends(get_current_user)
):
    """设置输出配置"""
    await output_service.set_config(session_id, config)
    return {"status": "configured"}

@router.get("/game/{session_id}/output/config")
async def get_output_config(
    session_id: str,
    current_user: dict = Depends(get_current_user)
):
    """获取输出配置"""
    return await output_service.get_config(session_id)
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/api/output.py` | 创建 | 输出 API |

---

## 验收标准

- [ ] API 正常工作
- [ ] 配置保存生效
- [ ] 单元测试通过

---

**最后更新**: 2026-02-07
**状态**: [ ] 待开始
