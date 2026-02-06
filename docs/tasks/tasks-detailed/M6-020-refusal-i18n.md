# M6-020: 实现多语言模板支持

**任务ID**: M6-020
**标题**: 实现多语言模板支持
**类型**: backend (后端开发)
**预估工时**: 4h
**依赖**: M6-019

---

## 任务描述

实现拒绝模板的多语言支持。

---

## 实现方案

```python
# app/services/refusal/i18n.py
from typing import Dict

class RefusalTemplateI18n:
    """拒绝模板国际化"""

    TEMPLATES: Dict[str, Dict[str, str]] = {
        "out_of_bounds": {
            "zh": "...",
            "en": "...",
        },
        "not_understood": {
            "zh": "...",
            "en": "...",
        },
    }

    @classmethod
    def get_template(cls, template_type: str, lang: str = "zh") -> str:
        """获取模板"""
        if template_type not in cls.TEMPLATES:
            template_type = "not_understood"

        templates = cls.TEMPLATES[template_type]
        return templates.get(lang, templates.get("zh", ""))
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/refusal/i18n.py` | 创建 | 国际化支持 |

---

## 验收标准

- [ ] 支持中英文
- [ ] 易于扩展新语言
- [ ] 模板格式正确

---

**最后更新**: 2026-02-07
**状态**: [ ] 待开始
