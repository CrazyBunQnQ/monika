# M0-001: 定义核心命令清单

**任务ID**: M0-001
**标题**: 定义核心命令清单 (15个)
**类型**: spec (规范设计)
**预估工时**: 2h
**依赖**: 无

---

## 任务描述

定义 CoC 跑团平台的核心命令清单，包括基础命令、检定命令、战斗命令、状态命令等15个核心命令。这是整个命令系统的基础，后续所有命令规范都基于此清单展开。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M0-001-01 | 分析 CoC 7e 规则书 | 提取需要系统支持的命令类型 | 15min |
| M0-001-02 | 设计基础命令列表 | /help, /status, /leads, /rule, /quit | 15min |
| M0-001-03 | 设计检定命令列表 | /roll, /push, /luck, /diff | 15min |
| M0-001-04 | 设计战斗命令列表 | /combat start, /combat action, /combat end | 10min |
| M0-001-05 | 设计状态命令列表 | /san check, /heal, /rest | 10min |
| M0-001-06 | 编写命令清单文档 | Markdown 格式输出 | 20min |
| M0-001-07 | 命令命名规范审查 | 确保命名一致性 | 10min |
| M0-001-08 | 添加命令示例 | 为每个命令添加使用示例 | 15min |
| M0-001-09 | 团队评审与修订 | 组织评审会议收集反馈 | 10min |

---

## 核心命令清单

### 基础命令 (5个)
```bash
/help          - 显示帮助信息
/status        - 显示当前状态
/leads         - 显示可选行动
/rule [query]  - 规则问答
/quit          - 结束会话
```

### 检定命令 (4个)
```bash
/roll [skill]  - 技能检定 (支持属性)
/push          - 推骰 (失败后可推)
/luck [n]      - 花幸运 (需引用事件)
/diff [n]      - 设置难度 (KP)
```

### 战斗命令 (3个)
```bash
/combat start  - 开始战斗
/combat action - 执行战斗动作
/combat end    - 结束战斗
```

### 状态命令 (3个)
```bash
/san check     - SAN 检定
/heal [n]      - 治疗
/rest [type]   - 休息
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `docs/specs/commands.md` | 创建 | 命令规范主文档 |
| `docs/specs/command-list.md` | 创建 | 命令清单文档 |

---

## 数据结构示例

```typescript
interface CommandDefinition {
  name: string;           // 命令名称
  category: string;       // 命令分类
  syntax: string;         // 语法格式
  description: string;    // 命令描述
  examples: string[];     // 使用示例
  requires_auth: boolean; // 是否需要认证
  role: 'all' | 'kp' | 'player'; // 可用角色
}
```

---

## 验收标准

- [✅] 定义15个核心命令
- [✅] 命令命名符合直觉
- [✅] 每个命令有明确描述和示例
- [✅] 命令分类清晰
- [✅] 文档输出完整

**完成状态**: ✅ 已完成 (2026-02-07)
**交付物**: `docs/specs/commands.md`

---

## 参考文档

- CoC 7e 规则书
- 游戏命令设计最佳实践

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
