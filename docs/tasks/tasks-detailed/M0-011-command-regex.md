# M0-011: 编写命令参数正则表达式

**任务ID**: M0-011
**标题**: 编写命令参数正则表达式
**类型**: spec (规范设计)
**预估工时**: 2h
**依赖**: M0-010

---

## 任务描述

编写命令参数的正则表达式，用于解析和验证用户输入的命令参数。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M0-011-01 | 设计参数格式规范 | 参数语法 | 20min |
| M0-011-02 | 编写基础参数正则 | 简单参数 | 25min |
| M0-011-03 | 编写复杂参数正则 | 带选项的参数 | 30min |
| M0-011-04 | 编写难度参数正则 | 难度表达式 | 20min |
| M0-011-05 | 编写骰子参数正则 | 骰子表达式 | 25min |
| M0-011-06 | 测试正则表达式 | 测试用例 | 25min |
| M0-011-07 | 编写正则文档 | 正则参考 | 10min |

---

## 参数正则表达式

### 基础参数正则
```regex
# 技能名称 (英文，小写，下划线)
SKILL_NAME = '[a-z_]+'

# 属性名称
ATTRIBUTE = 'STR|CON|DEX|APP|POW|INT|SIZ|EDU'

# 数字
NUMBER = '[0-9]+'

# 难度级别
DIFFICULTY = 'regular|hard|extreme'

# 选项
OPTION = '--[a-z]+'
```

### 复杂参数正则
```regex
# 难度参数
DIFFICULTY_PARAM = '--difficulty (regular|hard|extreme)|-d[1-3]'

# 修正值参数
MODIFIER_PARAM = '--modifier (-?[0-9]+)|[+-][0-9]+'

# 奖励骰参数
BONUS_PARAM = '--bonus ([1-3])|b([1-3])'

# 惩罚骰参数
PENALTY_PARAM = '--penalty ([1-3])|p([1-3])'
```

### 完整命令正则
```regex
# /roll 命令
ROLL_CMD = r'^/roll\s+(?:(?P<skill>{skill}|{attr})(?:\s+(?P<options>.*))?)?$'

# /combat 命令
COMBAT_CMD = r'^/combat\s+(?P<action>start|action|end)(?:\s+(?P<params>.*))?$'

# /san 命令
SAN_CMD = r'^/san\s+check\s+(?P<trigger>.+?)(?:\s+(?P<loss>\d+/\d+[dD]?\d*))?$'
```

---

## Python 实现

```python
# app/core/command_parser.py
import re
from typing import Optional, Dict, Any
from dataclasses import dataclass

@dataclass
class ParsedCommand:
    command: str
    params: Dict[str, Any]
    raw: str

class CommandParser:
    # 正则表达式模式
    PATTERNS = {
        'skill': re.compile(r'^[a-z_]+$'),
        'attribute': re.compile(r'^(STR|CON|DEX|APP|POW|INT|SIZ|EDU)$'),
        'difficulty': re.compile(r'^--difficulty\s+(regular|hard|extreme)$'),
        'modifier': re.compile(r'^--modifier\s+(-?\d+)$'),
        'bonus': re.compile(r'^--bonus\s+([1-3])$'),
        'penalty': re.compile(r'^--penalty\s+([1-3])$'),
    }

    def parse_roll(self, input: str) -> ParsedCommand:
        """解析 /roll 命令"""
        parts = input.strip().split()
        command = parts[0]

        if command != '/roll':
            raise ValueError(f"Not a roll command: {command}")

        params = {
            'target': None,
            'target_type': None,
            'difficulty': 'regular',
            'modifier': 0,
            'bonus': 0,
            'penalty': 0,
        }

        # 解析目标
        if len(parts) > 1:
            target = parts[1]
            if self.PATTERNS['attribute'].match(target):
                params['target'] = target
                params['target_type'] = 'attribute'
            elif self.PATTERNS['skill'].match(target):
                params['target'] = target
                params['target_type'] = 'skill'
            else:
                raise ValueError(f"Invalid target: {target}")

        # 解析选项
        for part in parts[2:]:
            if self.PATTERNS['difficulty'].match(part):
                params['difficulty'] = part.split()[1]
            elif self.PATTERNS['modifier'].match(part):
                params['modifier'] = int(part.split()[1])
            elif self.PATTERNS['bonus'].match(part):
                params['bonus'] = int(part.split()[1])
            elif self.PATTERNS['penalty'].match(part):
                params['penalty'] = int(part.split()[1])
            else:
                raise ValueError(f"Unknown option: {part}")

        return ParsedCommand(
            command='roll',
            params=params,
            raw=input
        )
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/core/command_parser.py` | 创建 | 命令解析器 |
| `docs/specs/command-regex.md` | 创建 | 正则表达式文档 |
| `tests/test_parser.py` | 创建 | 解析器测试 |

---

## 验收标准

- [ ] 正则表达式覆盖所有参数
- [ ] 解析器正确处理命令
- [ ] 错误处理完善
- [ ] 测试用例全面

---

## 参考文档

- M0-010: 命令语法 BNF

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
