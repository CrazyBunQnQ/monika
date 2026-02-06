# M0-012: 编写物品管理命令规范

**任务ID**: M0-012
**标题**: 编写物品管理命令规范
**类型**: spec (规范定义)
**预估工时**: 1h
**依赖**: 无

---

## 任务描述

定义物品管理相关命令的语法规范，包括物品查看、使用、丢弃、转移等功能。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M0-012-01 | 定义物品查看命令 | View Command | 10min |
| M0-012-02 | 定义物品使用命令 | Use Command | 10min |
| M0-012-03 | 定义物品转移命令 | Transfer Command | 15min |
| M0-012-04 | 定义物品丢弃命令 | Drop Command | 10min |
| M0-012-05 | 编写命令示例 | Examples | 10min |
| M0-012-06 | 编写输出格式 | Output Format | 10min |

---

## 命令列表

### `/items` - 查看物品列表

```
/items
/items [角色名]
```

**描述**: 查看当前角色或指定角色的物品列表

**示例**:
```bash
# 查看当前角色物品
/items

# 查看指定角色物品
/items 张三
```

**输出**:
```
📦 物品列表：张三
┌────────────────────────────────────┐
│ 1. 古老的怀表          [珍品]      │
│    描述: 一个刻有神秘符文的怀表      │
│    重量: 0.1kg                       │
│                                     │
│ 2. 左轮手枪           [武器]       │
│    弹药: 6/6          伤害: 1d10    │
│    重量: 0.8kg                       │
│                                     │
│ 3. 医疗包             [消耗品]     │
│    恢复: 1d6 HP                       │
│    重量: 0.3kg                       │
└────────────────────────────────────┘
总重量: 1.2kg / 10kg
```

---

### `/item` - 查看物品详情

```
/item <物品名称或ID>
/item <角色名> <物品名称或ID>
```

**描述**: 查看物品的详细信息

**示例**:
```bash
# 查看当前角色物品
/item 古老的怀表

# 查看指定角色物品
/item 李四 左轮手枪
```

---

### `/use` - 使用物品

```
/use <物品名称或ID>
/use <角色名> <物品名称或ID>
```

**描述**: 使用消耗品或装备物品

**示例**:
```bash
# 使用医疗包
/use 医疗包

# 使用指定角色的物品
/use 张三 急救包
```

**输出**:
```
🎲 使用物品：医疗包
┌────────────────────────────────────┐
│ 恢复: 1d6 = 4 HP                    │
│ 当前 HP: 18/20 (+4)                 │
└────────────────────────────────────┘
```

---

### `/equip` - 装备物品

```
/equip <武器名称>
/equip <角色名> <武器名称>
```

**描述**: 装备武器或护甲

**示例**:
```bash
# 装备武器
/equip 左轮手枪

# 装备指定角色武器
/equip 李四 猎枪
```

---

### `/give` - 转移物品

```
/give <物品> <数量> <目标角色>
/give <角色名> <物品> <数量> <目标角色>
```

**描述**: 将物品转移给其他角色

**示例**:
```bash
# 转移物品
/give 古老的怀表 1 李四

# 指定来源角色转移
/give 张三 医疗包 2 李四
```

**输出**:
```
📦 物品转移
┌────────────────────────────────────┐
│ 张三 → 李四                         │
│ 物品: 古老的怀表 ×1                 │
└────────────────────────────────────┘
```

---

### `/drop` - 丢弃物品

```
/drop <物品名称或ID> [数量]
/drop <角色名> <物品名称或ID> [数量]
```

**描述**: 丢弃物品

**示例**:
```bash
# 丢弃物品
/drop 旧报纸

# 丢弃多个
/drop 子弹 10

# 指定角色丢弃
/drop 张三 破损的工具
```

---

### `/loot` - 拾取物品

```
/loot
/loot <物品名称>
```

**描述**: 从场景中拾取物品

**示例**:
```bash
# 拾取场景中所有物品
/loot

# 拾取特定物品
/loot 神秘钥匙
```

---

## 物品属性

### 基础属性

| 属性 | 类型 | 说明 |
|------|------|------|
| id | string | 物品唯一标识 |
| name | string | 物品名称 |
| description | string | 物品描述 |
| type | string | 类型: weapon, armor, consumable, tool, treasure |
| rarity | string | 稀有度: common, uncommon, rare, epic, legendary |
| weight | number | 重量 (kg) |
| value | number | 价值 (美元) |
| quantity | number | 数量 |

### 武器属性

| 属性 | 类型 | 说明 |
|------|------|------|
| damage | string | 伤害表达式 (如 1d10) |
| range | string | 射程 (如 近战, 远程) |
| ammo_type | string | 弹药类型 |
| ammo_capacity | number | 弹药容量 |
| ammo_current | number | 当前弹药 |

### 护甲属性

| 属性 | 类型 | 说明 |
|------|------|------|
| defense | number | 防御值 |
| damage_reduction | number | 伤害减免 |

### 消耗品属性

| 属性 | 类型 | 说明 |
|------|------|------|
| effect | string | 效果描述 |
| uses | number | 使用次数 |
| uses_remaining | number | 剩余次数 |

---

## BNF 范式

```bnf
<item_command> ::= <items_cmd> | <item_cmd> | <use_cmd> | <equip_cmd> | <give_cmd> | <drop_cmd> | <loot_cmd>

<items_cmd> ::= "/items" [<character_name>]
<item_cmd> ::= "/item" <item_name> | "/item" <character_name> <item_name>

<use_cmd> ::= "/use" <item_name> | "/use" <character_name> <item_name>
<equip_cmd> ::= "/equip" <item_name> | "/equip" <character_name> <item_name>

<give_cmd> ::= "/give" <item_name> <quantity> <target_character>
             | "/give" <character_name> <item_name> <quantity> <target_character>

<drop_cmd> ::= "/drop" <item_name> [<quantity>]
              | "/drop" <character_name> <item_name> [<quantity>]

<loot_cmd> ::= "/loot" [<item_name>]

<quantity> ::= <number>
<item_name> ::= <string>
<character_name> ::= <string>
```

---

## 正则表达式

```regex
# /items
^\/items(?:\s+([\u4e00-\u9fa5\w]+))?$

# /item
^\/item(?:\s+([\u4e00-\u9fa5\w]+))?(?:\s+([\u4e00-\u9fa5\w\s]+))?$

# /use
^\/use(?:\s+([\u4e00-\u9fa5\w]+))?(?:\s+([\u4e00-\u9fa5\w\s]+))?$

# /equip
^\/equip(?:\s+([\u4e00-\u9fa5\w]+))?(?:\s+([\u4e00-\u9fa5\w\s]+))?$

# /give
^\/give(?:\s+([\u4e00-\u9fa5\w]+))?\s+([\u4e00-\u9fa5\w\s]+)\s+(\d+)\s+([\u4e00-\u9fa5\w]+)$

# /drop
^\/drop(?:\s+([\u4e00-\u9fa5\w]+))?\s+([\u4e00-\u9fa5\w\s]+)(?:\s+(\d+))?$

# /loot
^\/loot(?:\s+([\u4e00-\u9fa5\w\s]+))?$
```

---

## 错误处理

### 物品不存在

```
❌ 错误：物品 "神秘的钥匙" 不存在
```

### 数量不足

```
❌ 错误：数量不足
当前只有 5 个 "子弹"，但试图转移 10 个
```

### 目标角色无效

```
❌ 错误：目标角色 "王五" 不在房间中
```

### 重量限制

```
❌ 错误：超重
"李四" 的背包已满 (9.5/10kg)
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `docs/specs/commands/items.md` | 创建 | 命令规范文档 |
| `frontend/src/components/commands/ItemsCommand.tsx` | 创建 | 前端组件 |
| `app/services/item.py` | 创建 | 物品服务 |
| `app/api/items.py` | 创建 | 物品 API |

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
