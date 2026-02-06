# M0-014: 编写数据导出命令规范

**任务ID**: M0-014
**标题**: 编写数据导出命令规范
**类型**: spec (规范定义)
**预估工时**: 1h
**依赖**: 无

---

## 任务描述

定义数据导出相关命令的语法规范，包括导出战局记录、角色卡、场景数据等功能。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M0-014-01 | 定义导出命令 | Export Command | 15min |
| M0-014-02 | 定义导出格式 | Export Formats | 15min |
| M0-014-03 | 定义导出范围 | Export Scope | 15min |
| M0-014-04 | 编写命令示例 | Examples | 15min |
| M0-014-05 | 编写输出格式 | Output Format | 10min |
| M0-014-06 | 编写错误处理 | Error Handling | 10min |

---

## 命令列表

### `/export` - 导出数据

```
/export <类型> [格式] [选项]
```

**描述**: 导出各种游戏数据

**参数**:
- `类型`: 导出的数据类型
- `格式`: 导出格式 (可选，默认为 json)
- `选项`: 额外选项 (可选)

**支持的类型**:

| 类型 | 说明 | 支持的格式 |
|------|------|------------|
| `log` | 战局记录 | json, txt, md, pdf |
| `character` | 角色卡 | json, pdf, txt |
| `scene` | 场景数据 | json, md |
| `handouts` | 手递物 | json, zip |
| `timeline` | 时间轴 | json, md |
| `all` | 全部数据 | zip |

---

## 命令示例

### 导出战局记录

```bash
# 导出为 JSON
/export log json

# 导出为 Markdown
/export log md

# 导出为 PDF
/export log pdf

# 指定时间范围
/export log json --from "2024-01-01" --to "2024-12-31"

# 只导出特定类型的事件
/export log json --type "roll,check,combat"
```

**输出**:
```
📤 导出战局记录
┌────────────────────────────────────┐
│ 导出中...                           │
│                                     │
│ ✓ 收集事件: 1,234 条                │
│ ✓ 生成文件                          │
│                                     │
│ 下载链接:                           │
│ /api/exports/log_20240105.json      │
│                                     │
│ 文件大小: 2.3 MB                    │
└────────────────────────────────────┘
```

---

### 导出角色卡

```bash
# 导出当前角色卡
/export character pdf

# 导出指定角色卡
/export character pdf --character "张三"

# 导出所有角色卡
/export characters json

# 导出为文本格式
/export character txt --character "李四"
```

---

### 导出场景数据

```bash
# 导出当前场景
/export scene md

# 导出所有场景
/export scenes json

# 包含 NPC 和线索
/export scene json --include "npcs,clues"
```

---

### 导出全部数据

```bash
# 导出全部数据为 ZIP
/export all zip

# 导出全部数据为 JSON
/export all json

# 只导出特定类型
/export all zip --include "characters,scenes,handouts"
```

---

## 导出格式

### JSON 格式

```json
{
  "export_type": "log",
  "exported_at": "2024-01-05T12:00:00Z",
  "campaign": {
    "id": "camp_123",
    "name": "鬼屋惊魂"
  },
  "events": [
    {
      "id": "evt_456",
      "type": "roll",
      "timestamp": "2024-01-05T10:00:00Z",
      "data": {
        "character": "张三",
        "roll": 45,
        "skill": "侦查"
      }
    }
  ]
}
```

### Markdown 格式

```markdown
# 战局记录：鬼屋惊魂

**导出时间**: 2024-01-05 12:00:00
**战役**: 鬼屋惊魂

## 事件记录

### 2024-01-05 10:00:00
**类型**: 掷骰
**角色**: 张三
**技能**: 侦查
**结果**: 45 / 60 (成功)

---

*共 1,234 条事件*
```

---

## BNF 范式

```bnf
<export_command> ::= "/export" <export_type> [<format>] [<options>]

<export_type> ::= "log" | "character" | "characters" | "scene" | "scenes" | "handouts" | "timeline" | "all"

<format> ::= "json" | "txt" | "md" | "pdf" | "zip"

<options> ::= <option> | <option> <options>
<option> ::= "--" <option_name> [<value>]

<option_name> ::= "from" | "to" | "character" | "type" | "include"
```

---

## 正则表达式

```regex
^\/export\s+(log|character|characters|scene|scenes|handouts|timeline|all)(?:\s+(json|txt|md|pdf|zip))?(?:\s+(.+))?$
```

---

## 错误处理

### 不支持的导出类型

```
❌ 错误：不支持的导出类型 "invalid"
支持的类型: log, character, scene, handouts, timeline, all
```

### 不支持的格式

```
❌ 错误：类型 "log" 不支持格式 "docx"
支持的格式: json, txt, md, pdf
```

### 权限错误

```
❌ 错误：没有导出权限
只有 KP 可以导出完整数据
```

### 数据为空

```
⚠️ 警告：没有可导出的数据
当前战局没有任何记录
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `docs/specs/commands/export.md` | 创建 | 命令规范文档 |
| `frontend/src/components/commands/ExportCommand.tsx` | 创建 | 前端组件 |
| `app/services/export.py` | 创建 | 导出服务 |
| `app/api/export.py` | 创建 | 导出 API |

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

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
