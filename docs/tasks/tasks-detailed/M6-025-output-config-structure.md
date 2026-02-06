# M6-025: 设计 OutputConfig 结构

**任务ID**: M6-025
**标题**: 设计 OutputConfig 结构
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M0

---

## 任务描述

设计输出配置的数据结构，控制 AI 输出的格式和内容。

---

## 数据结构

```python
# app/core/types/output.py
from pydantic import BaseModel
from typing import Optional

class OutputLengthConfig(BaseModel):
    """输出长度配置"""
    narrative: int = 500      # 叙述最大字数
    description: int = 300    # 描述最大字数
    dialogue: int = 200       # 对话最大字数
    total: int = 1000         # 总最大字数

class OutputIncludeConfig(BaseModel):
    """输出包含配置"""
    state_changes: bool = True    # 包含状态变化
    leads: bool = True            # 包含 Leads
    hints: bool = False           # 包含提示
    consequences: bool = True     # 包含后果

class OutputConfig(BaseModel):
    """输出配置"""
    format: str = "normal"        # brief, normal, detailed
    length: OutputLengthConfig = OutputLengthConfig()
    include: OutputIncludeConfig = OutputIncludeConfig()
    style: str = "narrative"      # narrative, mechanical, mixed
    tone: str = "neutral"         # neutral, dramatic, horror
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/core/types/output.py` | 创建 | 输出类型 |

---

## 验收标准

- [ ] 结构设计合理
- [ ] 配置项完整
- [ ] 易于使用

---

**最后更新**: 2026-02-07
**状态**: [ ] 待开始
