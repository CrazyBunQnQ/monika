# M0-023 编写必填字段校验规则

## 概述
定义场景包中所有必填字段的校验规则,确保基础数据完整性,防止因缺少关键字段导致的运行时错误。

## 验收标准
- [ ] 列出所有顶层必填字段
- [ ] 列出 metadata 必填字段
- [ ] 列出 scene 必填字段
- [ ] 列出 NPC 必填字段
- [ ] 列出 Location 必填字段
- [ ] 定义嵌套对象必填规则
- [ ] 提供清晰的错误提示

## 技术方案

### 必填字段列表

```typescript
// 顶层必填
const REQUIRED_ROOT = [
  'metadata',
  'metadata.id',
  'metadata.title',
  'metadata.version',
  'metadata.author',
  'scenes'
];

// metadata 必填
const REQUIRED_METADATA = [
  'id',
  'title',
  'version',
  'author',
  'created_at'
];

// scene 必填
const REQUIRED_SCENE = [
  'id',
  'title',
  'order',
  'narrative.opening'
];

// NPC 必填
const REQUIRED_NPC = [
  'id',
  'name',
  'stats.STR',
  'stats.CON',
  'stats.DEX',
  'stats.INT',
  'stats.APP',
  'stats.POW',
  'stats.SIZ',
  'stats.EDU'
];

// Location 必填
const REQUIRED_LOCATION = [
  'id',
  'name'
];

// Clue 必填
const REQUIRED_CLUE = [
  'id',
  'description'
];
```

### 校验实现

```typescript
interface RequiredFieldRule {
  path: string;
  message: string;
  level: 'error' | 'warning';
}

function validateRequired(data: any, rules: RequiredFieldRule[]): ValidationResult {
  const errors: ValidationError[] = [];

  for (const rule of rules) {
    const value = getNestedValue(data, rule.path);
    if (value === undefined || value === null || value === '') {
      errors.push({
        field: rule.path,
        message: rule.message || `必填字段缺失: ${rule.path}`,
        level: rule.level
      });
    }
  }

  return {
    valid: errors.length === 0,
    errors
  };
}

// 嵌套访问
function getNestedValue(obj: any, path: string): any {
  return path.split('.').reduce((current, key) => {
    return current?.[key];
  }, obj);
}
```

### 错误提示

```json
{
  "field": "metadata.id",
  "message": "脚本 ID 必填,格式: script_unique_id",
  "level": "error",
  "suggestion": "添加 metadata.id 字段,使用 UUID 或自定义唯一标识"
}
```

## 依赖关系
- 前置任务: M0-022 编写场景包 JSON Schema
- 被依赖: M0-024 编写类型校验规则, M0-025 编写引用完整性校验

## 预估工时
2h
