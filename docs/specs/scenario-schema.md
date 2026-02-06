# 场景包格式规范

**版本**: v1.0
**最后更新**: 2026-02-07
**状态**: ✅ 设计完成

---

## 概述

本文档定义 CoC 跑团平台的场景包（Scenario Package）格式规范。场景包是 KP 创建和分享跑团模组的核心工具，采用多文件结构，支持线性、分支、沙盒等多种剧情模式。

**设计原则**:
- **模块化**: 文件按类型组织，便于协作编辑
- **灵活性**: 支持混合剧情模式（线性/分支/沙盒）
- **可扩展**: 支持触发器、变量、表达式等高级功能
- **可验证**: 完整的 JSON Schema 和引用完整性检查

---

## 文件结构

```
module_name/                    # 模组根目录
├── module.json                 # 模组元信息（必需）
├── scenes/                     # 场景目录（必需）
│   ├── scene_001.json
│   ├── scene_002.json
│   └── ...
├── shared/                     # 共享内容目录（可选）
│   ├── npcs.json              # NPC 定义
│   ├── locations.json         # 地点定义
│   ├── clues.json             # 线索定义
│   ├── items.json             # 物品定义
│   └── handouts.json          # 手递物定义
├── locales/                    # 本地化目录（可选）
│   ├── en.json
│   ├── zh-CN.json
│   └── ja.json
└── assets/                     # 资源目录（可选）
    ├── images/
    ├── audio/
    └── documents/
```

---

## 模组元信息 (module.json)

```json
{
  "id": "module_unique_id",
  "title": "午夜图书馆",
  "version": "1.0.0",
  "schema_version": "1.0",
  "min_engine_version": "1.2.0",
  "author": "KP 名字",
  "description": "玩家调查一座神秘的图书馆...",
  "duration": "2-4h",
  "player_count": "3-5",
  "tags": ["入门", "现代", "调查"],
  "difficulty": "中等",
  "min_level": 1,
  "max_level": 5,
  "entry_scene": "scene_001",
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z",
  "debug": false
}
```

**字段说明**:

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| id | string | ✅ | 唯一标识符，只能包含小写字母、数字和下划线 |
| title | string | ✅ | 模组标题 |
| version | string | ✅ | 版本号，格式 x.y.z |
| schema_version | string | ✅ | 使用的 schema 版本 |
| author | string | ✅ | 作者名字 |
| entry_scene | string | ✅ | 入口场景 ID |
| description | string | ✅ | 模组描述 |
| duration | string | ❌ | 预计游戏时长 |
| player_count | string | ❌ | 推荐玩家数量 |
| tags | array | ❌ | 标签数组 |

---

## 场景结构 (scenes/scene_xxx.json)

### 完整示例

```json
{
  "id": "scene_001",
  "title": "古老图书馆",
  "order": 1,
  "type": "exploration",

  "narrative": {
    "opening": "你推开沉重的木门，尘埃在透过彩色玻璃窗的光线中飞舞。书架高耸入云，空气中弥漫着旧纸张的味道...",
    "alternate": ["你再次回到图书馆...", "夜晚的图书馆显得格外宁静..."],
    "on_revisit": "你熟悉地穿过书架..."
  },

  "npcs": [
    {"ref": "npc_librarian", "position": "前台"},
    {"ref": "npc_mysterious_stranger", "position": "角落", "condition": "visited_scene_002 == true"}
  ],

  "locations": [
    {"ref": "loc_library_main", "accessible": true},
    {"ref": "loc_library_basement", "accessible": "items.key == true"}
  ],

  "clues": [
    {"ref": "clue_old_book", "obtainable": "roll.library_use > 50"},
    {"ref": "clue_secret_note", "obtainable": "locations.basement == true"}
  ],

  "handouts": [
    {"ref": "handout_map", "trigger": "clues.old_book == true"}
  ],

  "transitions": [
    {
      "id": "trans_001",
      "label": "深入调查",
      "target": "dynamic",
      "resolver": {
        "type": "conditional",
        "cases": [
          {"condition": "items.key == true", "target": "scene_003"},
          {"condition": "clues.old_book == true", "target": "scene_002"},
          {"default": true, "target": "scene_002"}
        ]
      }
    },
    {
      "id": "trans_002",
      "label": "离开图书馆",
      "target": "scene_999",
      "condition": null
    }
  ],

  "on_enter": {
    "trigger": "always",
    "effects": [
      {"action": "set_variable", "name": "visited_library", "value": "variables.visited_library + 1"},
      {"action": "check", "type": "sanity", "difficulty": "easy", "on_fail": "add_temporary_insanity"}
    ]
  },

  "on_exit": {
    "trigger": "once",
    "effects": [
      {"action": "narrate", "text": "你离开时，管理员意味深长地看着你..."}
    ]
  },

  "requirements": {
    "items": [],
    "clues": [],
    "variables": {}
  }
}
```

### 字段说明

**narrative（叙述）**:

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| opening | string | ✅ | 首次进入的叙述 |
| alternate | array | ❌ | 后续访问的替代叙述（随机选择） |
| on_revisit | string | ❌ | 重访时的固定叙述 |

**transitions（跳转）**:

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| id | string | ✅ | 跳转 ID |
| label | string | ✅ | 显示给玩家的标签 |
| target | string | ✅ | 目标场景 ID 或 "dynamic" |
| condition | string/null | ❌ | 显示条件表达式 |
| resolver | object | ❌ | 动态目标解析器（target=dynamic 时必需） |

**动态跳转解析器**:

```json
{
  "resolver": {
    "type": "conditional",
    "cases": [
      {"condition": "表达式", "target": "scene_id"},
      {"default": true, "target": "scene_id"}
    ]
  }
}
```

---

## 共享内容定义

### NPC 定义 (shared/npcs.json)

```json
{
  "npc_librarian": {
    "id": "npc_librarian",
    "name": "艾琳·韦伯",
    "age": 45,
    "occupation": "图书管理员",
    "description": "一位戴着厚眼镜的中年女性，对图书馆的每一本书都了如指掌。",
    "personality": ["严谨", "健谈", "神秘"],
    "portrait": "assets/images/npc_librarian.png",
    "stats": {
      "STR": 40, "DEX": 50, "INT": 75, "EDU": 80,
      "APP": 40, "POW": 60, "SIZ": 50, "CON": 45
    },
    "skills": {
      "library_use": 80,
      "listen": 60,
      "psychology": 50
    },
    "dialogue": {
      "greeting": "欢迎来到午夜图书馆...有什么我可以帮你的吗？",
      "topics": {
        "old_book": "你是说那本古书？很少有人注意到它...",
        "basement": "地下室？那里已经封存多年了..."
      },
      "on_clue": "clue.old_book",
      "unlock": "secret_dialogue"
    },
    "inventory": ["item_key"],
    "attitude": "neutral",
    "fighting": false
  }
}
```

### 地点定义 (shared/locations.json)

```json
{
  "loc_library_main": {
    "id": "loc_library_main",
    "name": "图书馆大厅",
    "description": "高耸的书架排列整齐，阳光透过彩色玻璃窗洒下...",
    "image": "assets/images/library_main.jpg",
    "atmosphere": "宁静、古老",
    "lighting": "dim",
    "sounds": ["翻书声", "脚步声回响"],
    "smells": ["旧纸张", "灰尘"],
    "features": [
      {
        "name": "前台",
        "description": "一位管理员正在整理书籍...",
        "interactable": true,
        "interaction": "talk.npc_librarian"
      },
      {
        "name": "旧书区",
        "description": "角落里堆满了古老的书卷...",
        "interactable": true,
        "interaction": "roll.library_use",
        "on_success": "obtain.clue_old_book"
      }
    ],
    "connections": ["loc_library_basement"],
    "hidden_content": ["secret_door"]
  }
}
```

### 线索定义 (shared/clues.json)

```json
{
  "clue_old_book": {
    "id": "clue_old_book",
    "name": "古书的发现",
    "description": "你在旧书区发现了一本关于神秘教派的记载...",
    "content": "这个教团崇拜一位沉睡的古神，他们相信通过特定的仪式可以唤醒它...",
    "importance": "major",
    "sanity_cost": 0,
    "tags": ["教团", "仪式", "古神"],
    "references": ["scene_002", "scene_003"],
    "private": false
  }
}
```

**importance 可选值**: `major`, `minor`, `plot_critical`

### 物品定义 (shared/items.json)

```json
{
  "item_key": {
    "id": "item_key",
    "name": "生锈的钥匙",
    "description": "一把古老的铜钥匙，上面刻着奇怪的符号...",
    "type": "key",
    "usable": true,
    "use_target": "loc_library_basement",
    "use_effect": "locations.basement.accessible = true",
    "weight": 0.1,
    "value": 0,
    "sanity_cost": 0,
    "damage": null,
    "armor": null
  }
}
```

**type 可选值**: `weapon`, `tool`, `key`, `consumable`, `armor`

### 手递物定义 (shared/handouts.json)

```json
{
  "handout_map": {
    "id": "handout_map",
    "name": "图书馆地图",
    "type": "image",
    "content": "assets/images/library_map.png",
    "description": "一张详细的图书馆平面图，标注了各个区域...",
    "shown_to": ["all"],
    "editable": false,
    "notes": []
  }
}
```

**type 可选值**: `image`, `text`, `document`

---

## 表达式条件系统

### 运算符

| 类别 | 运算符 |
|------|--------|
| 比较 | `==`, `!=`, `>`, `<`, `>=`, `<=` |
| 逻辑 | `&&`, `\|\|`, `!` |
| 算术 | `+`, `-`, `*`, `/`, `%` |
| 成员 | `in`, `contains` |
| 存在 | `exists`, `defined` |

### 可访问的变量命名空间

```javascript
// 物品
items.key                    // 是否拥有物品
items.count("sword")         // 物品数量

// 线索
clues.old_book               // 是否发现线索
clues.count("major")         // 某类线索数量

// NPC
npcs.librarian.met           // 是否见过
npcs.librarian.attitude      // 关系状态

// 地点
locations.basement.visited   // 是否访问过
locations.basement.accessible // 是否可进入

// 变量
variables.visited_count      // 自定义变量
variables.player_choice      // 玩家选择

// 技能/属性
skills.library_use           // 技能值
stats.STR                    // 属性值

// 检定结果
roll.library_use             // 最近一次检定的结果
```

### 表达式示例

```
// 基础条件
items.key == true
clues.old_book == true
variables.visited_count >= 2

// 复合条件
(items.key || skills.lockpick >= 50) && locations.basement.visited == false
clues.count("major") >= 3 && npcs.librarian.attitude == 'friendly'

// 检定相关
roll.library_use > 50
roll.library_use >= skills.library_use

// 场景相关
visited_scene_002 == true
scenes_visited.contains("scene_003")
```

---

## 触发器与效果

### 触发器类型

```json
"on_enter": {
  "trigger": "always",  // always, once, conditional
  "condition": "变量表达式",
  "effects": [...]
}
```

### 效果类型

```json
{
  "effects": [
    {
      "action": "set_variable",
      "name": "visited_count",
      "value": "variables.visited_count + 1"
    },
    {
      "action": "give_item",
      "item": "item_key",
      "condition": "roll.investigate >= 50"
    },
    {
      "action": "give_clue",
      "clue": "clue_secret"
    },
    {
      "action": "narrate",
      "text": "你感到一阵寒意..."
    },
    {
      "action": "check",
      "type": "sanity",
      "difficulty": "regular",
      "on_fail": [
        {"action": "set_variable", "name": "temporary_insanity", "value": true}
      ]
    },
    {
      "action": "branch",
      "cases": [
        {"if": "items.key", "then": "scene_003"},
        {"if": "clues.secret", "then": "scene_002"},
        {"else": "scene_001"}
      ]
    },
    {
      "action": "end_module",
      "ending": "bad_ending_01"
    }
  ]
}
```

---

## 本地化支持

```
module/
├── locales/
│   ├── en.json
│   ├── zh-CN.json
│   └── ja.json
```

```json
// locales/zh-CN.json
{
  "module": {
    "title": "午夜图书馆",
    "description": "..."
  },
  "scene_001": {
    "title": "古老图书馆",
    "narrative": {
      "opening": "你推开沉重的木门..."
    }
  },
  "npc_librarian": {
    "name": "艾琳·韦伯",
    "dialogue": {
      "greeting": "欢迎来到午夜图书馆..."
    }
  }
}
```

---

## JSON Schema

完整的 JSON Schema 定义请参考: `docs/specs/schemas/scenario-schema.json`

### 必填字段校验

| 文件 | 必填字段 |
|------|----------|
| module.json | id, title, version, author, entry_scene |
| scene_xxx.json | id, title, narrative.opening |
| npcs.json | id, name, description |
| locations.json | id, name, description |
| clues.json | id, name, description |
| items.json | id, name, type |
| handouts.json | id, name, type, content |

### 引用完整性校验

1. **场景引用**: entry_scene 必须存在于 scenes 中
2. **跳转目标**: transition.target 必须是有效的场景 ID
3. **共享内容**: scenes 中引用的 npc/location/clue 必须在 shared 中定义
4. **资源引用**: image/audio/document 路径必须存在于 assets 中

### 循环引用检测

检测场景跳转中可能存在的无限循环，警告创作者。

---

## 扩展能力

### 场景继承

```json
{
  "id": "scene_002",
  "extends": "template_combat_scene",
  "overrides": {
    "title": "图书馆战斗",
    "npcs": [{"ref": "npc_creature", "count": 2}]
  }
}
```

### 事件钩子

```json
{
  "on_roll_success": {
    "threshold": "critical",
    "effect": "trigger_secret_event"
  },
  "on_all_clues_found": {
    "required": ["clue_a", "clue_b", "clue_c"],
    "effect": "unlock_finale"
  }
}
```

### 调试模式

```json
{
  "debug": true,
  "debug_info": {
    "show_conditions": true,
    "show_variables": true,
    "allow_cheats": true,
    "skip_checks": false
  }
}
```

---

## 版本兼容性

| Schema 版本 | 引擎版本 | 说明 |
|-------------|----------|------|
| 1.0 | 1.2.0+ | 初始版本 |

---

## 最佳实践

1. **命名规范**: 使用小写字母和下划线，如 `scene_001`, `npc_librarian`
2. **文件组织**: 大型模组考虑按章节分组场景文件
3. **条件简化**: 避免过于复杂的嵌套条件
4. **引用清晰**: 使用语义化的 ID，便于理解和维护
5. **测试充分**: 使用调试模式测试所有分支路径

---

## 相关文档

- [M0-014 设计场景包根结构](../tasks/tasks-detailed/M0-014-scene-format.md)
- [M0-022 编写场景包 JSON Schema](../tasks/tasks-detailed/M0-022-json-schema.md)
- [命令集规范](./commands.md)
