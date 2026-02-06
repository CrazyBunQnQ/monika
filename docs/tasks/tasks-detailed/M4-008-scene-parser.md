# M4-008: 实现 JSON 解析器

**任务ID**: M4-008
**标题**: 实现 JSON 解析器
**类型**: backend (后端开发)
**预估工时**: 4h
**依赖**: M0

---

## 任务描述

实现场景包 JSON 文件的解析器，提取元数据、场景、NPC、线索等信息。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M4-008-01 | 设计解析器接口 | Parser interface | 20min |
| M4-008-02 | 实现基础 JSON 解析 | jsonschema 验证 | 30min |
| M4-008-03 | 实现元数据提取 | metadata 解析 | 30min |
| M4-008-04 | 实现场景解析 | scenes 解析 | 45min |
| M4-008-05 | 实现 shared 资源解析 | NPC/Location/Clue | 45min |
| M4-008-06 | 实现引用解析 | 转换引用为指针 | 30min |
| M4-008-07 | 错误处理和日志 | 解析错误处理 | 20min |
| M4-008-08 | 编写解析测试 | 单元测试 | 30min |
| M4-008-09 | 编写解析文档 | 使用说明 | 15min |

---

## 解析器接口

```python
# app/services/script_parser.py
from typing import Dict, Any, List
from pydantic import ValidationError

class ScriptParseError(Exception):
    """场景包解析错误"""
    def __init__(self, message: str, path: str = None, details: Any = None):
        self.message = message
        self.path = path
        self.details = details
        super().__init__(self.format_message())

    def format_message(self) -> str:
        msg = self.message
        if self.path:
            msg = f"[{self.path}] {msg}"
        if self.details:
            msg += f": {self.details}"
        return msg

class ScriptParser:
    """场景包解析器"""

    def __init__(self, schema_validator):
        self.validator = schema_validator

    def parse(self, file_path: str) -> Dict[str, Any]:
        """解析场景包文件"""
        import json
        from pathlib import Path

        # 1. 读取文件
        path = Path(file_path)
        if not path.exists():
            raise ScriptParseError(f"File not found: {file_path}")

        try:
            with open(path, 'r', encoding='utf-8') as f:
                data = json.load(f)
        except json.JSONDecodeError as e:
            raise ScriptParseError(
                "Invalid JSON format",
                path=file_path,
                details=str(e)
            )

        # 2. 验证 Schema
        try:
            self.validator.validate(data)
        except ValidationError as e:
            raise ScriptParseError(
                "Schema validation failed",
                path=file_path,
                details=e.errors()
            )

        # 3. 解析各部分
        result = {
            'metadata': self._parse_metadata(data.get('metadata', {})),
            'scenes': self._parse_scenes(data.get('scenes', {})),
            'shared': self._parse_shared(data.get('shared', {})),
        }

        # 4. 验证引用完整性
        self._validate_references(result)

        return result

    def _parse_metadata(self, metadata: Dict) -> Dict:
        """解析元数据"""
        required_fields = ['id', 'title', 'version', 'author']
        for field in required_fields:
            if field not in metadata:
                raise ScriptParseError(
                    f"Missing required metadata field: {field}",
                    path="metadata"
                )

        return {
            'id': metadata['id'],
            'title': metadata['title'],
            'version': metadata['version'],
            'author': metadata['author'],
            'description': metadata.get('description', ''),
            'duration': metadata.get('duration'),
            'player_count': metadata.get('player_count'),
            'tags': metadata.get('tags', []),
            'language': metadata.get('language', 'zh-CN'),
            'created_at': metadata.get('created_at'),
            'updated_at': metadata.get('updated_at'),
        }

    def _parse_scenes(self, scenes: Dict) -> Dict[str, Dict]:
        """解析场景集合"""
        parsed = {}

        for scene_id, scene_data in scenes.items():
            try:
                parsed[scene_id] = {
                    'id': scene_data.get('id', scene_id),
                    'title': scene_data.get('title', ''),
                    'order': scene_data.get('order', 0),
                    'narrative': scene_data.get('narrative', {}),
                    'npcs': scene_data.get('npcs', []),
                    'locations': scene_data.get('locations', []),
                    'clues': scene_data.get('clues', []),
                    'handouts': scene_data.get('handouts', []),
                    'transitions': scene_data.get('transitions', []),
                    'requirements': scene_data.get('requirements', {}),
                }
            except Exception as e:
                raise ScriptParseError(
                    f"Failed to parse scene: {scene_id}",
                    path=f"scenes.{scene_id}",
                    details=str(e)
                )

        return parsed

    def _parse_shared(self, shared: Dict) -> Dict:
        """解析共享资源"""
        return {
            'npcs': shared.get('npcs', {}),
            'locations': shared.get('locations', {}),
            'clues': shared.get('clues', {}),
            'handouts': shared.get('handouts', {}),
            'items': shared.get('items', {}),
        }

    def _validate_references(self, result: Dict):
        """验证引用完整性"""
        scenes = result['scenes']
        shared = result['shared']

        # 验证场景中的引用
        for scene_id, scene in scenes.items():
            # NPC 引用
            for npc_ref in scene.get('npcs', []):
                if npc_ref not in shared.get('npcs', {}):
                    raise ScriptParseError(
                        f"Invalid NPC reference: {npc_ref}",
                        path=f"scenes.{scene_id}.npcs"
                    )

            # 地点引用
            for loc_ref in scene.get('locations', []):
                if loc_ref not in shared.get('locations', {}):
                    raise ScriptParseError(
                        f"Invalid location reference: {loc_ref}",
                        path=f"scenes.{scene_id}.locations"
                    )

            # 跳转目标
            for trans in scene.get('transitions', []):
                target = trans.get('target')
                if target and target not in scenes:
                    raise ScriptParseError(
                        f"Invalid transition target: {target}",
                        path=f"scenes.{scene_id}.transitions"
                    )
```

---

## 使用示例

```python
# 使用解析器
from app.services.script_parser import ScriptParser
from jsonschema import validate

# 创建解析器
parser = ScriptParser(validator=validate)

# 解析场景包
try:
    result = parser.parse('scenarios/example.json')

    print(f"Script: {result['metadata']['title']}")
    print(f"Scenes: {len(result['scenes'])}")
    print(f"NPCs: {len(result['shared']['npcs'])}")

except ScriptParseError as e:
    print(f"Parse error: {e}")
    print(f"Path: {e.path}")
    print(f"Details: {e.details}")
```

---

## 解析结果结构

```typescript
interface ParsedScript {
  metadata: {
    id: string;
    title: string;
    version: string;
    author: string;
    description: string;
    duration?: string;
    player_count?: string;
    tags: string[];
    language: string;
    created_at?: string;
    updated_at?: string;
  };

  scenes: Record<string, ParsedScene>;

  shared: {
    npcs: Record<string, NPC>;
    locations: Record<string, Location>;
    clues: Record<string, Clue>;
    handouts: Record<string, Handout>;
    items: Record<string, Item>;
  };
}

interface ParsedScene {
  id: string;
  title: string;
  order: number;
  narrative: {
    opening: string;
    alternate?: string[];
  };
  npcs: string[];           // NPC 引用
  locations: string[];      // 地点引用
  clues: string[];          // 线索引用
  handouts: string[];       // 手递物引用
  transitions: Transition[];
  requirements?: {
    required_clues?: string[];
    required_state?: Record<string, any>;
    blocked_by?: string[];
  };
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/script_parser.py` | 创建 | 解析器实现 |
| `app/core/schema.py` | 创建 | Schema 验证 |
| `tests/test_parser.py` | 创建 | 解析器测试 |
| `docs/specs/parser.md` | 创建 | 解析器文档 |

---

## 验收标准

- [ ] 能正确解析有效的场景包
- [ ] 能检测 JSON 格式错误
- [ ] 能检测 Schema 验证错误
- [ ] 能检测引用完整性错误
- [ ] 错误信息清晰有用
- [ ] 解析性能良好 (< 1s for typical script)

---

## 参考文档

- M0-014: 场景包根结构
- M0-022: 场景包 JSON Schema
- Python jsonschema 文档

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
