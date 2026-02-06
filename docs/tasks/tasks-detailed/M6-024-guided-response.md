# M6-024: 实现引导式回复

**任务ID**: M6-024
**标题**: 实现引导式回复
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M6-023

---

## 任务描述

实现引导式的拒绝回复，帮助玩家回到正确的游戏轨道。

---

## 实现方案

```python
# app/services/refusal/guided.py
class GuidedResponseGenerator:
    """引导式回复生成器"""

    async def generate_guided_response(
        self,
        user_input: str,
        context: 'GameContext',
        refusal_reason: str
    ) -> dict:
        """生成引导式回复"""
        return {
            "refusal": self._generate_refusal(refusal_reason),
            "clarification": await self._generate_clarification(user_input, context),
            "suggestions": await self._generate_suggestions(context),
            "examples": self._generate_examples(context),
        }
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/refusal/guided.py` | 创建 | 引导回复 |

---

## 验收标准

- [ ] 回复结构清晰
- [ ] 提供具体引导
- [ ] 单元测试通过

---

**最后更新**: 2026-02-07
**状态**: [ ] 待开始
