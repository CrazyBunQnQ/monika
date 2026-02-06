# M0-022: 编写场景包 JSON Schema

**任务ID**: M0-022
**标题**: 编写场景包 JSON Schema
**类型**: spec (规范设计)
**预估工时**: 4h
**依赖**: M0-014

---

## 任务描述

编写场景包的 JSON Schema 定义，用于验证场景包文件的格式正确性。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M0-022-01 | 设计根 Schema | 场景包根结构 | 30min |
| M0-022-02 | 编写 metadata Schema | 元信息定义 | 30min |
| M0-022-03 | 编写 scene Schema | 场景定义 | 45min |
| M0-022-04 | 编写 shared Schema | 共享资源定义 | 30min |
| M0-022-05 | 添加自定义约束 | 业务规则验证 | 30min |
| M0-022-06 | 编写示例文件 | 符合 Schema 的示例 | 30min |
| M0-022-07 | 编写验证文档 | 如何使用 Schema | 15min |

---

## JSON Schema 结构

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://monika.app/schemas/scenario-package.json",
  "title": "CoC Scenario Package",
  "description": "CoC 跑团场景包格式定义",
  "type": "object",
  "required": ["metadata", "scenes", "shared"],
  "properties": {
    "metadata": {
      "$ref": "#/definitions/metadata"
    },
    "scenes": {
      "type": "object",
      "patternProperties": {
        "^[a-z0-9_]+$": {
          "$ref": "#/definitions/scene"
        }
      }
    },
    "shared": {
      "$ref": "#/definitions/shared"
    }
  },
  "definitions": {
    "metadata": {
      "type": "object",
      "required": ["id", "title", "version", "author"],
      "properties": {
        "id": {
          "type": "string",
          "pattern": "^[a-z0-9_-]+$",
          "description": "唯一标识符"
        },
        "title": {
          "type": "string",
          "minLength": 1,
          "maxLength": 200
        },
        "version": {
          "type": "string",
          "pattern": "^\\d+\\.\\d+\\.\\d+$"
        },
        "author": {
          "type": "string",
          "minLength": 1
        },
        "description": {
          "type": "string",
          "maxLength": 500
        },
        "duration": {
          "type": "string",
          "pattern": "^\\d+-\\d+h$"
        },
        "player_count": {
          "type": "string",
          "pattern": "^\\d+-\\d+$"
        },
        "tags": {
          "type": "array",
          "items": { "type": "string" }
        },
        "language": {
          "type": "string",
          "default": "zh-CN"
        },
        "created_at": {
          "type": "string",
          "format": "date-time"
        },
        "updated_at": {
          "type": "string",
          "format": "date-time"
        }
      }
    },
    "scene": {
      "type": "object",
      "required": ["id", "title", "order", "narrative"],
      "properties": {
        "id": {
          "type": "string"
        },
        "title": {
          "type": "string"
        },
        "order": {
          "type": "integer",
          "minimum": 1
        },
        "narrative": {
          "type": "object",
          "required": ["opening"],
          "properties": {
            "opening": {
              "type": "string",
              "minLength": 1
            },
            "alternate": {
              "type": "array",
              "items": { "type": "string" }
            }
          }
        },
        "npcs": {
          "type": "array",
          "items": { "type": "string" }
        },
        "locations": {
          "type": "array",
          "items": { "type": "string" }
        },
        "clues": {
          "type": "array",
          "items": { "type": "string" }
        },
        "handouts": {
          "type": "array",
          "items": { "type": "string" }
        },
        "transitions": {
          "type": "array",
          "items": {
            "$ref": "#/definitions/transition"
          }
        }
      }
    },
    "transition": {
      "type": "object",
      "required": ["target", "condition"],
      "properties": {
        "target": {
          "type": "string"
        },
        "condition": {
          "type": "string"
        },
        "description": {
          "type": "string"
        }
      }
    },
    "shared": {
      "type": "object",
      "properties": {
        "npcs": {
          "type": "object",
          "patternProperties": {
            ".*": { "$ref": "#/definitions/npc" }
          }
        },
        "locations": {
          "type": "object",
          "patternProperties": {
            ".*": { "$ref": "#/definitions/location" }
          }
        },
        "clues": {
          "type": "object",
          "patternProperties": {
            ".*": { "$ref": "#/definitions/clue" }
          }
        }
      }
    },
    "npc": {
      "type": "object",
      "required": ["id", "name"],
      "properties": {
        "id": { "type": "string" },
        "name": { "type": "string" },
        "description": { "type": "string" },
        "stats": {
          "type": "object",
          "properties": {
            "str": { "type": "integer" },
            "con": { "type": "integer" },
            "dex": { "type": "integer" },
            "int": { "type": "integer" },
            "pow": { "type": "integer" }
          }
        }
      }
    },
    "location": {
      "type": "object",
      "required": ["id", "name"],
      "properties": {
        "id": { "type": "string" },
        "name": { "type": "string" },
        "description": { "type": "string" },
        "connections": {
          "type": "array",
          "items": { "type": "string" }
        }
      }
    },
    "clue": {
      "type": "object",
      "required": ["id", "description"],
      "properties": {
        "id": { "type": "string" },
        "title": { "type": "string" },
        "description": { "type": "string" },
        "visibility": {
          "type": "string",
          "enum": ["public", "kp", "player:*"]
        }
      }
    }
  }
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `docs/specs/scenario-schema.json` | 创建 | JSON Schema 文件 |
| `docs/specs/script-schema.md` | 更新 | Schema 说明文档 |

---

## 验证工具

```bash
# 使用 jsonschema 验证
pip install jsonschema

jsonschema -i scenario.json docs/specs/scenario-schema.json

# 使用 ajv 验证 (Node.js)
npm install -g ajv-cli
ajv validate -s docs/specs/scenario-schema.json -d scenario.json
```

---

## 验收标准

- [ ] Schema 定义完整
- [ ] 必填字段正确
- [ ] 类型约束准确
- [ ] 格式验证有效
- [ ] 示例文件通过验证

---

## 参考文档

- M0-014: 场景包根结构
- JSON Schema 规范

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
