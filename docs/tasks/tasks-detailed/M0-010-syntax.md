# M0-010: 编写命令语法 BNF 范式

**任务ID**: M0-010
**标题**: 编写命令语法 BNF 范式
**类型**: spec (规范设计)
**预估工时**: 2h
**依赖**: M0-001

---

## 任务描述

使用 BNF (巴克斯-诺尔范式) 定义所有命令的语法规则，确保命令解析器有明确的语法规范可循。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M0-010-01 | 定义 BNF 基础符号 | 定义终结束和非终结符 | 10min |
| M0-010-02 | 编写 roll 命令 BNF | /roll 命令的完整语法 | 20min |
| M0-010-03 | 编写 combat 命令 BNF | /combat 相关命令语法 | 15min |
| M0-010-04 | 编写状态命令 BNF | /san, /heal, /rest 语法 | 15min |
| M0-010-05 | 编写基础命令 BNF | /help, /status, /leads 等语法 | 15min |
| M0-010-06 | 编写通用参数定义 | 数字/字符串/枚举类型 | 15min |
| M0-010-07 | 整合完整 BNF 文档 | 合并所有命令语法 | 15min |
| M0-010-08 | 添加语法注释 | 为复杂规则添加说明 | 10min |
| M0-010-09 | 验证 BNF 无二义性 | 检查语法规则完整性 | 10min |

---

## BNF 范式示例

### 基础定义
```bnf
<command> ::= "/" <command_name> [ <arguments> ]
<command_name> ::= "help" | "status" | "leads" | "roll" | "push" | "luck" | "combat" | "san" | "heal" | "rest" | "rule" | "quit"
<arguments> ::= <argument> | <argument> <whitespace> <arguments>

<skill_name> ::= [a-z_]+
<attribute_name> ::= "STR" | "CON" | "DEX" | "APP" | "POW" | "INT" | "SIZ" | "EDU"
<number> ::= [0-9]+
<difficulty> ::= "regular" | "hard" | "extreme"
```

### roll 命令
```bnf
<roll_command> ::= "/roll" [ <target> ] [ <difficulty_modifier> ]
<target> ::= <skill_name> | <attribute_name>
<difficulty_modifier> ::= "difficulty=" <difficulty>
```

### combat 命令
```bnf
<combat_command> ::= "/combat" <combat_action>
<combat_action> ::= "start" | "end" | "action"
```

### san 命令
```bnf
<san_command> ::= "/san" "check" [ <san_value> ]
<san_value> ::= <number> "/" <number>
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `docs/specs/command-bnf.md` | 创建 | BNF 语法规范文档 |

---

## 验收标准

- [ ] 所有15个命令都有 BNF 定义
- [ ] BNF 语法无二义性
- [ ] 可选参数正确标识
- [ ] 复杂语法有注释说明

---

## 参考文档

- M0-001: 核心命令清单
- BNF 范式标准

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
