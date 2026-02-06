# M0-024 编写类型校验规则

## 概述
定义场景包中所有字段的类型校验规则,确保数据类型正确性,防止类型错误导致的解析和运行时问题。

## 验收标准
- [ ] 定义基础类型校验(string, number, boolean, array, object)
- [ ] 定义枚举值校验
- [ ] 定义格式校验(datetime, uuid, url)
- [ ] 定义数值范围校验
- [ ] 定义字符串长度校验
- [ ] 定义数组元素类型校验
- [ ] 提供类型转换建议

## 技术方案

### 类型映射表

```typescript
interface TypeRule {
  type: 'string' | 'number' | 'boolean' | 'array' | 'object' | 'integer';
  format?: string;
  enum?: any[];
  min?: number;
  max?: number;
  minLength?: number;
  maxLength?: number;
  pattern?: RegExp;
  itemTypes?: TypeRule[];
}

// 字段类型定义
const FIELD_TYPES: Record<string, TypeRule> = {
  // metadata
  'metadata.id': { type: 'string', format: 'uuid', pattern: /^[a-z0-9_]+$/ },
  'metadata.version': { type: 'string', pattern: /^\d+\.\d+\.\d+$/ },
  'metadata.created_at': { type: 'string', format: 'datetime' },
  'metadata.player_count': { type: 'string', pattern: /^\d+-\d+$/ },

  // scene
  'scenes.*.order': { type: 'integer', min: 1 },
  'scenes.*.narrative.alternate': { type: 'array', itemTypes: [{ type: 'string' }] },

  // NPC stats
  'npcs.*.stats.STR': { type: 'integer', min: 0, max: 100 },
  'npcs.*.stats.CON': { type: 'integer', min: 0, max: 100 },
  'npcs.*.stats.DEX': { type: 'integer', min: 0, max: 100 },
  'npcs.*.stats.INT': { type: 'integer', min: 0, max: 100 },
  'npcs.*.stats.APP': { type: 'integer', min: 0, max: 100 },
  'npcs.*.stats.POW': { type: 'integer', min: 0, max: 100 },
  'npcs.*.stats.SIZ': { type: 'integer', min: 0, max: 100 },
  'npcs.*.stats.EDU': { type: 'integer', min: 0, max: 100 },
  'npcs.*.stats.HP': { type: 'integer', min: 1, max: 200 },
  'npcs.*.stats.MP': { type: 'integer', min: 0, max: 99 },
  'npcs.*.stats.SAN': { type: 'integer', min: 0, max: 99 },
  'npcs.*.db': { type: 'integer', min: -2, max: 2 },
  'npcs.*.build': { type: 'integer', min: -2, max: 5 },

  // Location
  'locations.*.type': {
    type: 'string',
    enum: ['indoor', 'outdoor', 'vehicle', 'abstract']
  }
};
```

### 校验函数

```typescript
function validateType(value: any, rule: TypeRule): ValidationResult {
  // 基础类型检查
  if (rule.type === 'string' && typeof value !== 'string') {
    return { valid: false, message: `期望字符串,实际: ${typeof value}` };
  }

  if (rule.type === 'number' && typeof value !== 'number') {
    return { valid: false, message: `期望数字,实际: ${typeof value}` };
  }

  if (rule.type === 'integer' && !Number.isInteger(value)) {
    return { valid: false, message: `期望整数,实际: ${typeof value}` };
  }

  if (rule.type === 'boolean' && typeof value !== 'boolean') {
    return { valid: false, message: `期望布尔值,实际: ${typeof value}` };
  }

  if (rule.type === 'array' && !Array.isArray(value)) {
    return { valid: false, message: `期望数组,实际: ${typeof value}` };
  }

  if (rule.type === 'object' && typeof value !== 'object') {
    return { valid: false, message: `期望对象,实际: ${typeof value}` };
  }

  // 格式检查
  if (rule.format === 'datetime' && !isValidDateTime(value)) {
    return { valid: false, message: '日期时间格式错误,应为 ISO 8601' };
  }

  if (rule.format === 'uuid' && !isValidUUID(value)) {
    return { valid: false, message: 'UUID 格式错误' };
  }

  // 正则检查
  if (rule.pattern && !rule.pattern.test(value)) {
    return { valid: false, message: `格式不匹配: ${rule.pattern}` };
  }

  // 枚举检查
  if (rule.enum && !rule.enum.includes(value)) {
    return { valid: false, message: `值必须是以下之一: ${rule.enum.join(', ')}` };
  }

  // 范围检查
  if (rule.min !== undefined && value < rule.min) {
    return { valid: false, message: `值不能小于 ${rule.min}` };
  }

  if (rule.max !== undefined && value > rule.max) {
    return { valid: false, message: `值不能大于 ${rule.max}` };
  }

  // 字符串长度
  if (rule.minLength && value.length < rule.minLength) {
    return { valid: false, message: `长度不能小于 ${rule.minLength}` };
  }

  if (rule.maxLength && value.length > rule.maxLength) {
    return { valid: false, message: `长度不能大于 ${rule.maxLength}` };
  }

  // 数组元素
  if (rule.type === 'array' && rule.itemTypes) {
    for (const item of value) {
      const result = validateType(item, rule.itemTypes[0]);
      if (!result.valid) {
        return { valid: false, message: `数组元素: ${result.message}` };
      }
    }
  }

  return { valid: true };
}
```

## 依赖关系
- 前置任务: M0-023 编写必填字段校验规则
- 被依赖: M0-022 编写场景包 JSON Schema

## 预估工时
2h
