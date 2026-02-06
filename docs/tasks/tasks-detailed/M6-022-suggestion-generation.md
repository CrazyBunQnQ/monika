# M6-022: 实现建议生成算法

**任务ID**: M6-022
**标题**: 实现建议生成算法
**类型**: backend (后端开发)
**预估工时**: 4h
**依赖**: M6-021

---

## 任务描述

实现基于游戏状态智能生成行动建议的算法。

---

## 实现方案

```python
# app/services/refusal/suggestion.py
class SuggestionGenerator:
    """建议生成器"""

    async def generate_suggestions(
        self,
        context: 'GameContext',
        count: int = 3
    ) -> List[str]:
        """生成行动建议"""
        suggestions = []

        # 基于场景生成
        suggestions.extend(self._scene_based_suggestions(context))

        # 基于线索生成
        suggestions.extend(self._clue_based_suggestions(context))

        # 基于可用性过滤
        return self._filter_available(suggestions, context)[:count]
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/refusal/suggestion.py` | 创建 | 建议生成 |

---

## 验收标准

- [ ] 建议相关且可执行
- [ ] 生成逻辑合理
- [ ] 单元测试通过

---

**最后更新**: 2026-02-07
**状态**: [ ] 待开始
