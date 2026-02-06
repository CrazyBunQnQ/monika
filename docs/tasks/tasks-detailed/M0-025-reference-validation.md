# M0-025 编写引用完整性校验

## 概述
定义场景包中引用字段的完整性校验规则,确保所有引用(如 NPC ref、Location ref)都指向实际存在的对象,防止悬空引用。

## 验收标准
- [ ] 定义场景内引用校验(场景引用 NPC/Location/Clue)
- [ ] 定义共享资源引用校验(引用 shared.npcs/shared.locations)
- [ ] 定义跳转目标校验(transitions.target 必须是有效场景 ID)
- [ ] 定义物品引用校验(inventory 中的物品)
- [ ] 定义技能引用校验(技能名称必须在技能列表中)
- [ ] 提供引用路径追踪
- [ ] 生成引用关系图

## 技术方案

### 引用类型定义

```typescript
interface ReferenceRule {
  field: string; // 字段路径
  refType: 'scene' | 'shared' | 'enum';
  refPath: string; // 引用路径,如 'shared.npcs', 'scenes'
  allowMultiple?: boolean; // 是否允许数组引用
  optional?: boolean; // 是否可选
}

const REFERENCE_RULES: ReferenceRule[] = [
  // NPC 引用
  {
    field: 'scenes.*.npcs.*.ref',
    refType: 'shared',
    refPath: 'shared.npcs',
    allowMultiple: false
  },
  {
    field: 'scenes.*.npcs.*.id',
    refType: 'scene',
    refPath: 'scenes.*.npcs',
    allowMultiple: false
  },

  // Location 引用
  {
    field: 'scenes.*.locations.*.ref',
    refType: 'shared',
    refPath: 'shared.locations',
    allowMultiple: false
  },

  // Clue 引用
  {
    field: 'scenes.*.clues.*.ref',
    refType: 'shared',
    refPath: 'shared.clues',
    allowMultiple: false
  },

  // Handout 引用
  {
    field: 'scenes.*.handouts.*.ref',
    refType: 'shared',
    refPath: 'shared.handouts',
    allowMultiple: false
  },

  // 跳转目标
  {
    field: 'scenes.*.transitions.*.target',
    refType: 'scene',
    refPath: 'scenes',
    allowMultiple: false
  },

  // 物品引用
  {
    field: 'characters.*.inventory.*',
    refType: 'enum',
    refPath: 'defined_items',
    allowMultiple: true
  },

  // 技能引用
  {
    field: 'characters.*.skills',
    refType: 'enum',
    refPath: 'defined_skills',
    allowMultiple: true
  }
];
```

### 校验实现

```typescript
interface ReferenceError {
  field: string;
  reference: string;
  target: string;
  message: string;
  suggestions?: string[];
}

function validateReferences(data: any): ValidationResult {
  const errors: ReferenceError[] = [];

  // 构建索引
  const indexes = buildReferenceIndexes(data);

  // 检查每个引用规则
  for (const rule of REFERENCE_RULES) {
    const references = extractReferences(data, rule.field);

    for (const ref of references) {
      const value = ref.value;
      const path = ref.path;

      if (rule.allowMultiple && Array.isArray(value)) {
        for (const item of value) {
          const error = checkReference(item, rule, indexes, path);
          if (error) errors.push(error);
        }
      } else if (!rule.optional || value !== undefined) {
        const error = checkReference(value, rule, indexes, path);
        if (error) errors.push(error);
      }
    }
  }

  return {
    valid: errors.length === 0,
    errors
  };
}

function checkReference(
  value: string,
  rule: ReferenceRule,
  indexes: Record<string, Set<string>>,
  path: string
): ReferenceError | null {
  const targetSet = indexes[rule.refPath];

  if (!targetSet || !targetSet.has(value)) {
    // 生成建议
    const suggestions = targetSet
      ? findSimilarStrings(value, Array.from(targetSet))
      : [];

    return {
      field: path,
      reference: value,
      target: rule.refPath,
      message: `引用 "${value}" 在 ${rule.refPath} 中不存在`,
      suggestions: suggestions.length > 0 ? suggestions : undefined
    };
  }

  return null;
}

function buildReferenceIndexes(data: any): Record<string, Set<string>> {
  return {
    'scenes': new Set(Object.keys(data.scenes || {})),
    'shared.npcs': new Set(Object.keys(data.shared?.npcs || {})),
    'shared.locations': new Set(Object.keys(data.shared?.locations || {})),
    'shared.clues': new Set(Object.keys(data.shared?.clues || {})),
    'shared.handouts': new Set(Object.keys(data.shared?.handouts || {}))
  };
}

// 提取引用(支持通配符)
function extractReferences(data: any, fieldPattern: string): Array<{path: string, value: string}> {
  const results: Array<{path: string, value: string}> = [];

  // 简化实现,实际需要完整的路径匹配
  const parts = fieldPattern.split('.');
  const hasWildcard = parts.some(p => p === '*');

  if (hasWildcard) {
    // 递归遍历匹配通配符
    traverseWildcard(data, parts, '', results);
  } else {
    // 直接路径访问
    const value = getNestedValue(data, fieldPattern);
    if (value) {
      results.push({ path: fieldPattern, value });
    }
  }

  return results;
}
```

### 错误报告

```json
{
  "field": "scenes.scene_001.npcs[0].ref",
  "reference": "npc_missing",
  "target": "shared.npcs",
  "message": "引用 \"npc_missing\" 在 shared.npcs 中不存在",
  "suggestions": ["npc_butler", "npc_maid", "npc_cook"]
}
```

### 引用关系图

```typescript
interface ReferenceGraph {
  nodes: Array<{
    id: string;
    type: 'scene' | 'npc' | 'location' | 'clue';
  }>;
  edges: Array<{
    from: string;
    to: string;
    type: string;
  }>;
}

function buildReferenceGraph(data: any): ReferenceGraph {
  const nodes: any[] = [];
  const edges: any[] = [];

  // 提取所有节点
  Object.keys(data.scenes || {}).forEach(sceneId => {
    nodes.push({ id: sceneId, type: 'scene' });
  });

  Object.keys(data.shared?.npcs || {}).forEach(npcId => {
    nodes.push({ id: npcId, type: 'npc' });
  });

  // 提取所有边
  // ... (遍历所有引用)

  return { nodes, edges };
}
```

## 依赖关系
- 前置任务: M0-023 编写必填字段校验规则
- 被依赖: M0-026 编写循环引用检测规则

## 预估工时
2h
