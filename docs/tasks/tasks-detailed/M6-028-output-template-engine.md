# M6-028: 实现输出模板引擎

**任务ID**: M6-028
**标题**: 实现输出模板引擎
**类型**: backend (后端开发)
**预估工时**: 4h
**依赖**: M6-027

---

## 任务描述

实现输出模板引擎，根据配置格式化 AI 输出。

---

## 实现方案

```python
# app/services/output/template_engine.py
class OutputTemplateEngine:
    """输出模板引擎"""

    async def render(
        self,
        content: dict,
        config: OutputConfig
    ) -> str:
        """渲染输出"""
        template = self._select_template(config.format)

        return template.render(
            narrative=content.get("narrative", ""),
            description=content.get("description", ""),
            dialogue=content.get("dialogue", ""),
            state_changes=content.get("state_changes", []),
            leads=content.get("leads", []),
        )
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/output/template_engine.py` | 创建 | 模板引擎 |
| `app/templates/output/` | 创建 | 模板文件 |

---

## 验收标准

- [ ] 模板渲染正确
- [ ] 支持多种格式
- [ ] 单元测试通过

---

**最后更新**: 2026-02-07
**状态**: [ ] 待开始
