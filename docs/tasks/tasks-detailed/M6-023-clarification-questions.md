# M6-023: 实现澄清问题生成

**任务ID**: M6-023
**标题**: 实现澄清问题生成
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M6-022

---

## 任务描述

实现当输入不明确时生成澄清问题。

---

## 实现方案

```python
# app/services/refusal/clarification.py
class ClarificationGenerator:
    """澄清问题生成器"""

    async def generate_clarification(
        self,
        user_input: str,
        context: 'GameContext'
    ) -> List[str]:
        """生成澄清问题"""
        questions = []

        # 分析意图
        intent = await self._analyze_intent(user_input)

        if intent.is_ambiguous:
            questions.extend(self._generate_questions(intent, context))

        return questions
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/refusal/clarification.py` | 创建 | 澄清生成 |

---

## 验收标准

- [ ] 问题指向明确
- [ ] 帮助用户澄清意图
- [ ] 单元测试通过

---

**最后更新**: 2026-02-07
**状态**: [ ] 待开始
