# 场景包制作指南

**版本**: v1.0
**最后更新**: 2026-02-07

---

## 概述

本指南介绍如何创建和编辑 CoC 跑团场景包（模组）。

---

## 快速开始

### 第一步：创建场景包目录

```bash
mkdir my_first_module
cd my_first_module
```

### 第二步：创建目录结构

```bash
mkdir -p scenes shared assets/images
```

最终目录结构：
```
my_first_module/
├── module.json
├── scenes/
│   └── scene_001.json
├── shared/
│   ├── npcs.json
│   └── locations.json
└── assets/
    └── images/
```

### 第三步：编写 module.json

```json
{
  "id": "my_first_module",
  "title": "我的第一个模组",
  "version": "1.0.0",
  "author": "你的名字",
  "description": "一个简短的调查模组",
  "duration": "1-2h",
  "player_count": "3-5",
  "tags": ["入门", "现代", "调查"],
  "entry_scene": "scene_001",
  "created_at": "2026-02-07T00:00:00Z",
  "updated_at": "2026-02-07T00:00:00Z"
}
```

### 第四步：创建第一个场景

在 `scenes/scene_001.json` 中：

```json
{
  "id": "scene_001",
  "title": "开场",
  "order": 1,
  "type": "introduction",
  "narrative": {
    "opening": "你们在咖啡馆里，偶然听到关于一本神秘古籍的讨论..."
  },
  "npcs": [
    {"ref": "npc_informant", "position": "corner"}
  ],
  "transitions": [
    {
      "id": "trans_001",
      "label": "继续打听",
      "target": "scene_002"
    },
    {
      "id": "trans_002",
      "label": "离开咖啡馆",
      "target": "scene_end"
    }
  ]
}
```

### 第五步：测试场景包

将场景包上传到平台，创建新会话进行测试。

---

## 进阶功能

### 条件跳转

```json
{
  "transitions": [
    {
      "id": "trans_001",
      "label": "进入密室",
      "target": "scene_003",
      "condition": "items.key == true"
    }
  ]
}
```

### 动态解析器

```json
{
  "transitions": [
    {
      "id": "trans_001",
      "label": "根据选择决定",
      "target": "dynamic",
      "resolver": {
        "type": "conditional",
        "cases": [
          {"condition": "items.key", "target": "scene_003"},
          {"condition": "clues.secret", "target": "scene_002"},
          {"default": true, "target": "scene_001"}
        ]
      }
    }
  ]
}
```

### 触发器

```json
{
  "on_enter": {
    "trigger": "always",
    "effects": [
      {
        "action": "set_variable",
        "name": "visited_library",
        "value": "true"
      },
      {
        "action": "check",
        "type": "sanity",
        "difficulty": "easy"
      }
    ]
  }
}
```

---

## 最佳实践

### 1. 命名规范

**场景 ID**: 使用 `scene_XXX` 格式
```json
{"id": "scene_001"}
```

**NPC 引用**: 使用描述性引用名
```json
{"ref": "npc_librarian"}
```

**线索 ID**: 使用 `clue_XXX` 格式
```json
{"ref": "clue_old_book"}
```

### 2. 场景包大小

- 短篇模组（1-2小时）：5-10 个场景
- 中篇模组（2-4小时）：10-20 个场景
- 长篇模组（4+小时）：20+ 个场景

### 3. 线索设计

**线索分级**:
- `major`: 主要线索，推动剧情
- `minor`: 次要线索，丰富背景

**线索链**:
```
clue_a → clue_b → clue_c → clue_final
```

### 4. 难度平衡

- 检定难度多样化（简单、普通、困难）
- 提供多条解决路径
- 避免卡死的"必检"

---

## 调试技巧

### 启用调试模式

```json
{
  "debug": true,
  "debug_info": {
    "show_conditions": true,
    "show_variables": true,
    "allow_cheats": true
  }
}
```

### 查看变量

```
/leads                    # 显示当前可用行动
/vars                     # 显示当前变量（调试模式）
/checkpoint                # 创建检查点
```

---

## 示例场景包

### 完整示例：午夜图书馆

参考：`examples/scenarios/midnight-library/`

包含：
- 6 个场景
- 3 个 NPC
- 5 个线索
- 2 个结局

---

## 参考文档

- [场景包格式规范](../../specs/scenario-schema.md)
- [KP 命令参考](./command-reference.md)
- [数据字典](../developer/data-dictionary.md)
