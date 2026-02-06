# M0-009: 编写 SAN 检定命令规范

**任务ID**: M0-009
**标题**: 编写 SAN 检定命令规范
**类型**: spec (规范定义)
**预估工时**: 1h
**依赖**: 无

---

## 任务描述

定义 `/san` 和 `/sancheck` 命令的语法规范，用于执行 SAN 值检定，这是 CoC 7e 中核心的理智机制。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M0-009-01 | 定义基本语法 | Basic Syntax | 10min |
| M0-009-02 | 定义伤害参数 | Damage Parameter | 10min |
| M0-009-03 | 定义描述参数 | Description Parameter | 10min |
| M0-009-04 | 编写示例 | Examples | 10min |
| M0-009-05 | 编写输出格式 | Output Format | 10min |
| M0-009-06 | 编写错误处理 | Error Handling | 5min |

---

## 命令语法

### 基本语法

```
/san <伤害骰> [/ <最大伤害>] [描述]
/sancheck <伤害骰> [/ <最大伤害>] [描述]
```

### 参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| 伤害骰 | string | 是 | d100 表达式，如 `1d6`、`2d8` |
| 最大伤害 | string | 否 | 可选，用 `/` 分隔，如 `/1d6` |
| 描述 | string | 否 | 触发 SAN 检定的事件描述 |

---

## 示例

### 基本 SAN 检定

```bash
# 看到尸体，1d6 SAN 伤害
/san 1d6 看到尸体

# 遭遇不可名状之物，1d6/1d20 SAN 伤害
/san 1d6/1d20 看到克苏鲁

# 简写形式
/sc 1d6 恐怖场景
```

### 完整命令

```bash
# 完整命令格式
/san 1d6 看到血腥场景

# 带最大伤害
/san 1d6/1d20 目击神话生物

# 阅读禁书
/san 1d4 阅读死灵之书
```

---

## 输出格式

### 成功输出

```
🎲 SAN 检定：张三
┌─────────────────────────────┐
│ SAN 值: 45/50 (-5)          │
│ 检定结果: 32 (成功)         │
│ 伤害骰: 1d6 = 4             │
│ 当前 SAN: 41/50             │
└─────────────────────────────┘
```

### 失败输出

```
🎲 SAN 检定：张三 [失败]
┌─────────────────────────────┐
│ SAN 值: 45/50               │
│ 检定结果: 78 (失败)         │
│ 伤害骰: 1d6 = 5             │
│ 最大伤害: 1d20 = 18         │
│ 实际伤害: 18                │
│ 当前 SAN: 27/50 [-18]       │
└─────────────────────────────┘
⚠️ 临时疯狂！请进行疯狂检定
```

---

## 临时疯狂触发

当 SAN 值降到 0 时：

```
💀 角色陷入永久疯狂！
┌─────────────────────────────┐
│ 当前 SAN: 0/50              │
│ 状态: 永久疯狂              │
│ 建议: 参考疯狂表            │
└─────────────────────────────┘
```

---

## 错误处理

### 参数错误

```
❌ 错误：缺少伤害骰参数
用法: /san <伤害骰> [/ <最大伤害>] [描述]
示例: /san 1d6 看到尸体
```

### 骰子格式错误

```
❌ 错误：无效的骰子格式 "abc"
骰子格式应为: NdM (如 1d6, 2d8)
```

---

## BNF 范式

```bnf
<san_check> ::= "/" ("san" | "sancheck" | "sc") <damage_dice> ["/" <max_damage>] [<description>]

<damage_dice> ::= <dice_expression>
<max_damage> ::= <dice_expression>
<description> ::= <string>

<dice_expression> ::= <number> "d" <number> ["+" <number>] | "-" <number> | <number>
```

---

## 正则表达式

```regex
^\/(?:san|sancheck|sc)\s+(\d+d\d+(?:[+-]\d+)?|\d+)(?:\s*\/\s*(\d+d\d+(?:[+-]\d+)?|\d+))?(?:\s+(.+))?
```

### 捕获组

| 组 | 内容 | 示例 |
|----|------|------|
| 1 | 伤害骰 | `1d6` |
| 2 | 最大伤害 | `1d20` |
| 3 | 描述 | `看到尸体` |

---

## 前端组件结构

```tsx
// frontend/src/components/commands/SanCheckCommand.tsx
interface SanCheckResult {
  characterId: string
  characterName: string
  currentSan: number
  maxSan: number
  checkRoll: number
  checkSuccess: boolean
  damageRoll: number
  maxDamageRoll?: number
  actualDamage: number
  newSan: number
  madnessType?: 'temporary' | 'indefinite' | 'permanent'
  description?: string
}

interface SanCheckProps {
  command: string
  onResult?: (result: SanCheckResult) => void
}

export function SanCheckCommand({ command, onResult }: SanCheckProps) {
  // 解析命令
  // 执行 SAN 检定
  // 显示结果
}
```

---

## 后端 API 结构

```python
# app/services/san_check.py
from pydantic import BaseModel
from typing import Optional

class SanCheckRequest(BaseModel):
    character_id: str
    damage_dice: str  # "1d6"
    max_damage_dice: Optional[str] = None  # "1d20"
    description: Optional[str] = None
    is_secret: bool = False

class SanCheckResult(BaseModel):
    character_id: str
    character_name: str
    current_san: int
    max_san: int
    check_roll: int
    check_success: bool
    damage_roll: int
    max_damage_roll: Optional[int] = None
    actual_damage: int
    new_san: int
    madness_type: Optional[str] = None
    description: Optional[str] = None

async def execute_san_check(
    request: SanCheckRequest,
    db: Session,
) -> SanCheckResult:
    """执行 SAN 检定"""
    # 1. 获取角色数据
    # 2. 执行 SAN 检定 (1d100 <= current_san)
    # 3. 成功：投伤害骰
    # 4. 失败：投 max(damage, max_damage)
    # 5. 扣除 SAN 值
    # 6. 检查疯狂状态
    # 7. 返回结果
    pass
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `docs/specs/commands/san-check.md` | 创建 | 命令规范文档 |
| `frontend/src/components/commands/SanCheckCommand.tsx` | 创建 | 前端组件 |
| `app/services/san_check.py` | 创建 | 后端服务 |
| `app/api/san.py` | 创建 | API 端点 |

---

## 验收标准

- [ ] 语法定义完整
- [ ] 示例覆盖全面
- [ ] 正则表达式正确
- [ ] 输出格式友好
- [ ] 错误提示清晰
- [ ] BNF 范式完整

---

## 参考文档

- CoC 7e 规则书第 8 章：理智
- M0-010: 命令语法 BNF 范式
- M0-011: 命令参数正则表达式
- M1-040: SAN 值系统

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
