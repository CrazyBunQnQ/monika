# M0-013: 编写战斗命令规范

**任务ID**: M0-013
**标题**: 编写战斗命令规范
**类型**: spec (规范定义)
**预估工时**: 1.5h
**依赖**: 无

---

## 任务描述

定义战斗相关命令的语法规范，包括攻击、伤害、战斗管理等功能。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M0-013-01 | 定义攻击命令 | Attack Command | 15min |
| M0-013-02 | 定义伤害命令 | Damage Command | 15min |
| M0-013-03 | 定义战斗管理 | Combat Management | 20min |
| M0-013-04 | 定义战斗状态 | Combat State | 15min |
| M0-013-05 | 编写战斗示例 | Examples | 15min |
| M0-013-06 | 编写输出格式 | Output Format | 15min |

---

## 命令列表

### `/attack` - 攻击检定

```
/attack [目标] [武器]
/attack [目标] [武器] [修正值]
```

**描述**: 对目标进行近战攻击检定

**参数**:
- `目标`: 目标名称（NPC 或玩家）
- `武器`: 使用的武器（可选，默认为主手武器）
- `修正值`: 技能修正值（可选）

**示例**:
```bash
# 使用默认武器攻击
/attack 僵尸A

# 指定武器攻击
/attack 僵尸A 左轮手枪

# 带修正值
/attack 僵尸A 猎枪 +20
```

**输出**:
```
⚔️ 攻击检定：张三 → 僵尸A
┌────────────────────────────────────┐
│ 武器: 左轮手枪                      │
│ 技能: 射击 (60%)                    │
│                                     │
│ 🎲 检定: 45 / 60 [成功]             │
│ 🎲 伤害: 1d10 = 7                  │
│                                     │
│ 结果: 僵尸A 受到 7 点伤害            │
│ 僵尸A HP: 15/22 [-7]                │
└────────────────────────────────────┘
```

---

### `/damage` - 造成伤害

```
/damage <目标> <伤害值> [伤害类型]
/damage <目标> <伤害骰> [伤害类型]
```

**描述**: 直接对目标造成伤害

**参数**:
- `目标`: 目标名称
- `伤害值`: 固定伤害值或骰子表达式
- `伤害类型`: 伤害类型（可选，如 钝器、穿刺、挥砍、火焰）

**示例**:
```bash
# 固定伤害
/damage 僵尸A 8

# 骰子伤害
/damage 僵尸A 1d6+2

# 指定伤害类型
/damage 僵尸A 2d8 火焰
```

**输出**:
```
💥 造成伤害
┌────────────────────────────────────┐
│ 目标: 僵尸A                         │
│ 伤害: 1d6+2 = 5                    │
│ 类型: 穿刺                          │
│                                     │
│ 结果: 僵尸A 受到 5 点伤害            │
│ 僵尸A HP: 17/22 [-5]                │
└────────────────────────────────────┘
```

---

### `/heal` - 治疗恢复

```
/heal [目标] <恢复值>
/heal [目标] <恢复骰>
```

**描述**: 为目标恢复生命值

**示例**:
```bash
# 治疗当前角色
/heal 5

# 治疗指定角色
/heal 张三 1d6+2

# 治疗所有队友
/heal all 3
```

---

### `/initiative` - 先攻检定

```
/initiative
/initiative [修正值]
```

**描述**: 进行先攻检定，确定战斗行动顺序

**示例**:
```bash
# 默认先攻检定
/initiative

# 带修正值
/initiative +5
```

**输出**:
```
🎯 先攻检定
┌────────────────────────────────────┐
│ 张三: 1d100+40 = 75                 │
│ 李四: 1d100+35 = 82                 │
│ 王五: 1d100+30 = 55                 │
│                                     │
│ 行动顺序:                           │
│ 1. 李四 (82)                        │
│ 2. 张三 (75)                        │
│ 3. 王五 (55)                        │
└────────────────────────────────────┘
```

---

### `/combat` - 战斗管理

```
/combat start [战斗名称]
/combat end
/combat turn
/combat status
```

**描述**: 管理战斗状态

**示例**:
```bash
# 开始战斗
/combat start 墓地遭遇战

# 结束战斗
/combat end

# 下一回合
/combat turn

# 查看状态
/combat status
```

---

## 战斗数据结构

```typescript
interface Combat {
  id: string
  name: string
  roomId: string
  status: 'active' | 'paused' | 'ended'
  round: number
  currentTurn: number
  participants: CombatParticipant[]
  startedAt: Date
  endedAt?: Date
}

interface CombatParticipant {
  id: string
  name: string
  type: 'player' | 'npc' | 'enemy'
  initiative: number
  hp: number
  maxHp: number
  status: 'active' | 'unconscious' | 'dead'
  conditions: string[]
}

interface Weapon {
  id: string
  name: string
  damage: string  // 如 "1d10"
  range: 'melee' | 'ranged'
  skill: string   // 如 "射击"
  hands: 1 | 2
}
```

---

## BNF 范式

```bnf
<combat_command> ::= <attack_cmd> | <damage_cmd> | <heal_cmd> | <initiative_cmd> | <combat_cmd>

<attack_cmd> ::= "/attack" [<target>] [<weapon>] [<modifier>]

<damage_cmd> ::= "/damage" <target> <damage_value> [<damage_type>]

<heal_cmd> ::= "/heal" [<target>] <heal_value>

<initiative_cmd> ::= "/initiative" [<modifier>]

<combat_cmd> ::= "/combat" <combat_action>
<combat_action> ::= "start" [<combat_name>] | "end" | "turn" | "status"

<target> ::= <string>
<weapon> ::= <string>
<modifier> ::= ("+" | "-") <number>
<damage_value> ::= <dice_expression> | <number>
<heal_value> ::= <dice_expression> | <number>
<damage_type> ::= <string>
<combat_name> ::= <string>
```

---

## 正则表达式

```regex
# /attack
^\/attack(?:\s+([\u4e00-\u9fa5\w]+))?(?:\s+([\u4e00-\u9fa5\w]+))?(?:\s+([+-]\d+))?$

# /damage
^\/damage\s+([\u4e00-\u9fa5\w]+)\s+(\d+d\d+(?:[+-]\d+)?|\d+)(?:\s+([\u4e00-\u9fa5\w]+))?$

# /heal
^\/heal(?:\s+(all|[\u4e00-\u9fa5\w]+))?\s+(\d+d\d+(?:[+-]\d+)?|\d+)$

# /initiative
^\/initiative(?:\s+([+-]\d+))?$

# /combat
^\/combat\s+(start|end|turn|status)(?:\s+([\u4e00-\u9fa5\w\s]+))?$
```

---

## 战斗状态输出

### `/combat status` 输出

```
⚔️ 战斗状态：墓地遭遇战 (第 3 回合)
┌────────────────────────────────────┐
│ 行动顺序:                           │
│ ─────────────────────────────────  │
│ ▶ 张三 [玩家]                      │
│   HP: 18/20  先攻: 75               │
│   状态: 正常                        │
│                                     │
│   李四 [玩家]                      │
│   HP: 15/15  先攻: 82               │
│   状态: 正常                        │
│                                     │
│   僵尸A [敌人]                     │
│   HP: 10/22  先攻: 40               │
│   状态: 受伤 (-12)                  │
│                                     │
│   僵尸B [敌人]                     │
│   HP: 22/22  先攻: 35               │
│   状态: 正常                        │
└────────────────────────────────────┘
当前回合: 张三
```

---

## 错误处理

### 目标不存在

```
❌ 错误：目标 "僵尸C" 不存在
请使用 /combat status 查看可用目标
```

### 伤害格式错误

```
❌ 错误：无效的伤害表达式 "abc"
伤害应为数字或骰子表达式 (如 1d6, 2d8+3)
```

### 战斗未开始

```
❌ 错误：当前没有进行中的战斗
请使用 /combat start 开始新战斗
```

### 武器未装备

```
❌ 错误：未装备武器
请使用 /equip 装备武器，或指定武器名称
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `docs/specs/commands/combat.md` | 创建 | 命令规范文档 |
| `frontend/src/components/commands/CombatCommand.tsx` | 创建 | 前端组件 |
| `app/services/combat.py` | 创建 | 战斗服务 |
| `app/api/combat.py` | 创建 | 战斗 API |

---

## 验收标准

- [ ] 所有命令定义完整
- [ ] 示例覆盖全面
- [ ] 正则表达式正确
- [ ] 输出格式友好
- [ ] 错误提示清晰
- [ ] BNF 范式完整

---

## 参考文档

- M0-010: 命令语法 BNF 范式
- M0-011: 命令参数正则表达式
- M1-020: 战斗系统

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
