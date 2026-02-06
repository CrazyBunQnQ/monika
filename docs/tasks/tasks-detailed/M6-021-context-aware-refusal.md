# M6-021: 实现上下文感知拒绝

**任务ID**: M6-021
**标题**: 实现上下文感知拒绝
**类型**: backend (后端开发)
**预估工时**: 4h
**依赖**: M6-020

---

## 任务描述

实现根据游戏上下文生成更智能的拒绝响应。

---

## 实现方案

```python
# app/services/refusal/context_aware.py
class ContextAwareRefusal:
    """上下文感知拒绝"""

    async def generate_refusal(
        self,
        user_input: str,
        context: 'GameContext',
        reason: str
    ) -> str:
        """生成上下文感知的拒绝"""
        template = self._select_template(context, reason)

        # 填充上下文信息
        return template.format(
            location=context.current_scene.name,
            available_actions=self._get_available_actions(context),
            nearby_npcs=self._get_nearby_npcs(context),
        )
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/refusal/context_aware.py` | 创建 | 上下文感知 |

---

## 验收标准

- [ ] 拒绝响应与上下文相关
- [ ] 提供可执行的建议
- [ ] 单元测试通过

---

**最后更新**: 2026-02-07
**状态**: [ ] 待开始
