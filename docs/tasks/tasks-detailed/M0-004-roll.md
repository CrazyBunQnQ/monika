# M0-004: 编写检定命令规范 (/roll)

**任务ID**: M0-004
**标题**: 编写检定命令规范 (/roll)
**类型**: spec (规范设计)
**预估工时**: 1h
**依赖**: M0-001

---

## 任务描述

定义 /roll 检定命令的详细规范，包括技能检定、属性检定、难度设置等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M0-004-01 | 设计检定命令语法 | 参数和选项 | 15min |
| M0-004-02 | 定义技能/属性映射 | 支持的检定类型 | 15min |
| M0-004-03 | 定义难度系统 | 难度级别和修正 | 15min |
| M0-004-04 | 设计响应格式 | 检定结果输出 | 10min |
| M0-004-05 | 编写命令示例 | 各种使用场景 | 10min |

---

## /roll 命令规范

### 语法
```bash
/roll [skill|attribute] [options]

# 技能检定
/roll 侦查
/roll library_use

# 属性检定
/roll STR
/roll DEX

# 带难度
/roll 侦查 --difficulty hard
/roll STR -20

# 带奖励骰
/roll 侦查 --bonus 1

# 带惩罚骰
/roll 侦查 --penalty 2
```

### 参数说明
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| skill/attribute | string | 否 | 技能名或属性名 |
| --difficulty | string | 否 | regular/hard/extreme |
| --modifier | number | 否 | 修正值 (-99 到 +99) |
| --bonus | number | 否 | 奖励骰数量 |
| --penalty | number | 否 | 惩罚骰数量 |

### 响应格式
```json
{
  "type": "roll",
  "skill": "侦查",
  "skill_value": 55,
  "roll": 38,
  "difficulty": "regular",
  "modifier": 0,
  "success_level": "regular",
  "description": "成功",
  "raw": "1d100 <= 55 = 38 → 成功"
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `docs/specs/commands.md` | 更新 | 添加 /roll 规范 |

---

## 验收标准

- [ ] 命令语法清晰
- [ ] 参数说明完整
- [ ] 响应格式明确
- [ ] 示例覆盖主要场景

---

## 参考文档

- M0-001: 核心命令清单
- CoC 7e 规则书

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
