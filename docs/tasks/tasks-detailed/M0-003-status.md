# M0-003: 编写 /status 命令规范

**任务ID**: M0-003
**标题**: 编写 /status 命令规范
**类型**: spec (规范设计)
**预估工时**: 1h
**依赖**: M0-001

---

## 任务描述

定义 /status 命令的详细规范，包括状态信息的格式、内容、显示方式等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M0-003-01 | 设计状态信息结构 | 状态字段组织 | 15min |
| M0-003-02 | 定义状态输出格式 | 文本/JSON | 10min |
| M0-003-03 | 设计角色状态展示 | 角色卡信息 | 15min |
| M0-003-04 | 设计会话状态展示 | 会话信息 | 10min |
| M0-003-05 | 设计参数支持 | --detail, --json | 5min |
| M0-003-06 | 编写规范文档 | 完整规范 | 5min |

---

## /status 命令规范

### 基础用法
```bash
/status                        # 显示基本状态
/status --detail               # 显示详细状态
/status --json                 # JSON 格式输出
/status <character_id>         # 显示特定角色状态 (KP)
```

### 状态输出格式

```typescript
interface StatusOutput {
  // 会话信息
  session: {
    session_id: string;
    status: 'waiting' | 'active' | 'paused';
    scene_id?: string;
    kp: string;
    players: string[];
  };

  // 角色信息
  character?: {
    name: string;
    attributes: Record<string, number>;
    derived: {
      hp: { current: number; max: number };
      san: { current: number; max: number };
      luck: { current: number; max: number };
    };
    status: string;
    skills?: Record<string, number>;
  };

  // 游戏状态
  game?: {
    in_combat: boolean;
    in_chase: boolean;
    current_round?: number;
  };

  // 时间戳
  updated_at: datetime;
}
```

### 状态示例

```
=== 当前状态 ===

会话: #12345
状态: 进行中
场景: 图书馆 investigating

角色: 陈安娜 (HP: 12/12, SAN: 45/99, Luck: 50/50)
状态: 健康

属性: STR 50 | CON 55 | DEX 60 | INT 70
      APP 40 | POW 45 | SIZ 50 | EDU 75

技能: 侦查 55, 聆听 50, 图书馆使用 65
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `docs/specs/commands.md` | 更新 | 添加 /status 规范 |

---

## 验收标准

- [ ] /status 命令规范清晰
- [ ] 信息完整有用
- [ ] 格式易读
- [ ] 参数合理

---

## 参考文档

- M0-001: 核心命令清单
- M0-028: 角色状态定义

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
