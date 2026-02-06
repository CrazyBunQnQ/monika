# M0-015: 定义 metadata 元信息结构

**任务ID**: M0-015
**标题**: 定义 metadata 元信息结构
**类型**: spec (规范设计)
**预估工时**: 2h
**依赖**: M0-014

---

## 任务描述

定义场景包元信息 (metadata) 的详细结构，包括脚本ID、标题、版本、作者等字段。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M0-015-01 | 设计 metadata 结构 | 元信息字段 | 25min |
| M0-015-02 | 定义必填字段 | 最小元信息 | 15min |
| M0-015-03 | 定义可选字段 | 扩展元信息 | 20min |
| M0-015-04 | 定义验证规则 | 格式验证 | 25min |
| M0-015-05 | 编写 TypeScript 类型 | 类型定义 | 20min |
| M0-015-06 | 编写示例元信息 | 示例数据 | 10min |

---

## Metadata 结构

```typescript
interface ScenarioMetadata {
  // === 必填字段 ===

  // 唯一标识
  id: string;
  /** 脚本唯一标识符，格式: [a-z0-9_-]+ */

  // 基本信息
  title: string;
  /** 脚本标题，1-200 字符 */

  version: string;
  /** 语义化版本号，格式: X.Y.Z */

  author: string;
  /** 作者名称或 ID */

  // === 可选字段 ===

  description?: string;
  /** 简短描述，1-500 字符 */

  duration?: string;
  /** 预计游戏时长，格式: "X-Yh" (如 "2-4h") */

  player_count?: string;
  /** 推荐玩家数，格式: "X-Y" (如 "3-5") */

  tags?: string[];
  /** 标签列表，如 ["入门", "现代", "恐怖"] */

  language?: string;
  /** 语言代码，默认 "zh-CN" */

  // === 时间戳 ===

  created_at?: string;
  /** ISO 8601 格式创建时间 */

  updated_at?: string;
  /** ISO 8601 格式更新时间 */

  published_at?: string;
  /** ISO 8601 格式发布时间 */

  // === 扩展信息 ===

  min_players?: number;
  /** 最少玩家数 */

  max_players?: number;
  /** 最多玩家数 */

  difficulty?: 'easy' | 'normal' | 'hard' | 'extreme';
  /** 难度级别 */

  age_rating?: string;
  /** 年龄分级，如 "12+", "16+", "18+" */

  genre?: string[];
  /** 类型标签，如 ["恐怖", "悬疑", "调查"] */

  setting?: string;
  /** 背景设定，如 "1920s 美国", "现代日本" */

  // === 联系方式 ===

  author_contact?: string;
  /** 作者联系方式 */

  license?: string;
  /** 许可证信息 */

  // === 统计 ===

  play_count?: number;
  /** 累计游戏次数 */

  rating?: number;
  /** 平均评分 (0-5) */

  // === 扩展 ===

  extensions?: Record<string, any>;
  /** 自定义扩展字段 */
}
```

---

## 验证规则

```python
# app/services/metadata_validator.py
from typing import Dict, Any, List
from dataclasses import dataclass

@dataclass
class ValidationError:
    field: str
    message: str
    value: Any

class MetadataValidator:
    REQUIRED_FIELDS = ['id', 'title', 'version', 'author']

    VERSION_PATTERN = r'^\d+\.\d+\.\d+$'
    ID_PATTERN = r'^[a-z0-9_-]+$'
    DURATION_PATTERN = r'^\d+-\d+h$'
    PLAYER_COUNT_PATTERN = r'^\d+-\d+$'

    def validate(self, metadata: Dict) -> List[ValidationError]:
        """验证元信息"""
        errors = []

        # 检查必填字段
        for field in self.REQUIRED_FIELDS:
            if field not in metadata:
                errors.append(ValidationError(
                    field=field,
                    message=f"Missing required field: {field}",
                    value=None
                ))

        # 验证 ID 格式
        if 'id' in metadata:
            import re
            if not re.match(self.ID_PATTERN, metadata['id']):
                errors.append(ValidationError(
                    field='id',
                    message=f"Invalid ID format",
                    value=metadata['id']
                ))

        # 验证版本号
        if 'version' in metadata:
            if not re.match(self.VERSION_PATTERN, metadata['version']):
                errors.append(ValidationError(
                    field='version',
                    message="Version must be in X.Y.Z format",
                    value=metadata['version']
                ))

        # 验证时长格式
        if 'duration' in metadata:
            if not re.match(self.DURATION_PATTERN, metadata['duration']):
                errors.append(ValidationError(
                    field='duration',
                    message="Duration must be in X-Yh format",
                    value=metadata['duration']
                ))

        # 验证玩家数格式
        if 'player_count' in metadata:
            if not re.match(self.PLAYER_COUNT_PATTERN, metadata['player_count']):
                errors.append(ValidationError(
                    field='player_count',
                    message="Player count must be in X-Y format",
                    value=metadata['player_count']
                ))

        # 验证字符串长度
        if 'title' in metadata:
            title = metadata['title']
            if not title or len(title) > 200:
                errors.append(ValidationError(
                    field='title',
                    message="Title must be 1-200 characters",
                    value=title
                ))

        if 'description' in metadata:
            desc = metadata['description']
            if len(desc) > 500:
                errors.append(ValidationError(
                    field='description',
                    message="Description must be <= 500 characters",
                    value=desc
                ))

        return errors
```

---

## 示例元信息

```json
{
  "id": "haunted_mansion_1920",
  "title": "1920年的凶宅",
  "version": "1.0.0",
  "author": "KP 张三",
  "description": "玩家们受委托调查一栋据说闹鬼的维多利亚式豪宅。",
  "duration": "2-4h",
  "player_count": "3-5",
  "tags": ["入门", "现代", "恐怖", "经典"],
  "language": "zh-CN",
  "difficulty": "normal",
  "age_rating": "16+",
  "genre": ["恐怖", "调查", "超自然"],
  "setting": "1920s 马萨诸塞州",
  "created_at": "2026-01-15T10:00:00Z",
  "updated_at": "2026-02-01T15:30:00Z",
  "license": "CC-BY-NC-SA 4.0"
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `docs/specs/metadata.md` | 创建 | 元信息规范 |
| `app/core/types/metadata.ts` | 创建 | TypeScript 类型 |
| `app/services/metadata_validator.py` | 创建 | 验证器 |

---

## 验收标准

- [ ] metadata 结构定义完整
- [ ] 必填字段明确
- [ ] 验证规则正确
- [ ] 示例数据有效
- [ ] TypeScript 类型正确

---

## 参考文档

- M0-014: 场景包根结构

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
