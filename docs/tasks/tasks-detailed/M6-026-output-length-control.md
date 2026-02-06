# M6-026: 实现输出长度控制

**任务ID**: M6-026
**标题**: 实现输出长度控制
**类型**: backend (后端开发)
**预估工时**: 4h
**依赖**: M6-025

---

## 任务描述

实现根据配置控制 AI 输出长度的功能。

---

## 实现方案

```python
# app/services/output/length_control.py
class OutputLengthController:
    """输出长度控制器"""

    async def control_length(
        self,
        content: str,
        config: OutputLengthConfig
    ) -> str:
        """控制输出长度"""
        # 分段处理
        sections = self._split_sections(content)

        # 按优先级截断
        return self._truncate_by_priority(sections, config)

    def _split_sections(self, content: str) -> dict:
        """分段"""
        return {
            "narrative": self._extract_narrative(content),
            "description": self._extract_description(content),
            "dialogue": self._extract_dialogue(content),
        }
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/output/length_control.py` | 创建 | 长度控制 |

---

## 验收标准

- [ ] 长度限制生效
- [ ] 截断合理
- [ ] 保持内容完整性

---

**最后更新**: 2026-02-07
**状态**: [ ] 待开始
