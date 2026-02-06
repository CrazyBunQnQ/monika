# M0-002: 编写 /help 命令规范

**任务ID**: M0-002
**标题**: 编写 /help 命令规范
**类型**: spec (规范设计)
**预估工时**: 1h
**依赖**: M0-001

---

## 任务描述

定义 /help 命令的详细规范，包括帮助信息的格式、内容组织方式、参数支持等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M0-002-01 | 设计帮助信息结构 | 分层级帮助 | 15min |
| M0-002-02 | 定义帮助输出格式 | Markdown/纯文本 | 10min |
| M0-002-03 | 设计命令分类展示 | 按类别组织 | 10min |
| M0-002-04 | 设计参数帮助 | /help <command> | 10min |
| M0-002-05 | 编写命令列表文档 | 每个命令的说明 | 10min |
| M0-002-06 | 编写规范文档 | 完整帮助规范 | 5min |

---

## /help 命令规范

### 基础用法
```bash
/help                          # 显示所有命令列表
/help <command>                # 显示特定命令的详细帮助
/help --category <type>        # 按类别筛选命令
```

### 帮助信息格式

```typescript
interface HelpOutput {
  // 总览模式
  general?: {
    title: string;
    description: string;
    version: string;
    categories: CommandCategory[];
  };

  // 命令详情模式
  command?: {
    name: string;
    syntax: string;
    description: string;
    parameters: Parameter[];
    examples: Example[];
    related_commands?: string[];
    see_also?: string[];
  };
}

interface CommandCategory {
  name: string;
  description: string;
  commands: string[];
}

interface Parameter {
  name: string;
  type: string;
  required: boolean;
  description: string;
  default?: any;
}

interface Example {
  input: string;
  output: string;
  description: string;
}
```

### 帮助示例

```
=== CoC 跑团平台命令帮助 ===

基础命令:
  /help              - 显示此帮助信息
  /status            - 显示当前状态
  /leads             - 显示可选行动
  /rule <query>      - 规则问答
  /quit              - 结束会话

检定命令:
  /roll [skill]      - 技能检定
  /push              - 推骰 (失败后)
  /luck <n>          - 花幸运
  /diff <n>          - 设置难度 (KP)

战斗命令:
  /combat start      - 开始战斗
  /combat action     - 执行战斗动作
  /combat end        - 结束战斗

使用 /help <command> 查看详细说明
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `docs/specs/commands.md` | 更新 | 添加 /help 规范 |

---

## 验收标准

- [ ] /help 命令规范清晰
- [ ] 输出格式易读
- [ ] 分类合理
- [ ] 示例完整

---

## 参考文档

- M0-001: 核心命令清单

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
