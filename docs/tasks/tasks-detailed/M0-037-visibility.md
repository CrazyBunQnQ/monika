# M0-037 定义 Visibility 可见性

## 概述
定义游戏事件的可见性(visibility)规则,控制哪些信息对 KP、玩家和公众可见,实现信息分级和权限管理。

## 验收标准
- [ ] 定义可见性类型(公开/KP/玩家)
- [ ] 定义可见性表达式语法
- [ ] 定义条件可见性
- [ ] 定义可见性继承规则
- [ ] 定义可见性覆盖机制

## 技术方案

### 可见性类型

```typescript
type Visibility =
  | 'public'      // 公开: 所有人可见
  | 'kp'          // 仅 KP: 只有 KP 可见
  | 'player:*'    // 所有玩家: 所有玩家可见,KP 也可见
  | 'player:X'    // 特定玩家: 只有玩家 X 可见
  | 'conditional'; // 条件可见: 根据条件动态决定

interface VisibilityRule {
  // 基础可见性
  base: Visibility;

  // 条件表达式
  condition?: VisibilityCondition;

  // 过期时间
  expires_at?: string;

  // 可升级为(满足条件后)
  upgradable_to?: Visibility;
  upgrade_condition?: string;
}
```

### 可见性条件

```typescript
interface VisibilityCondition {
  // 角色条件
  character?: {
    has_id: string; // 特定角色
    has_item: string; // 拥有物品
    has_clue: string; // 发现线索
  };

  // 状态条件
  state?: {
    field: string;
    operator: 'eq' | 'ne' | 'gt' | 'lt';
    value: any;
  };

  // 场景条件
  scene?: {
    in_scene: string; // 在特定场景
    left_scene: string; // 离开特定场景
  };

  // 事件条件
  event?: {
    occurred: string; // 事件已发生
    not_occurred: string; // 事件未发生
  };

  // 时间条件
  time?: {
    after: string; // 时间之后
    before: string; // 时间之前
  };

  // 组合条件
  logic?: 'AND' | 'OR';
  conditions?: VisibilityCondition[];
}
```

### 可见性判断

```typescript
interface VisibilityContext {
  viewer_id: string;
  viewer_role: 'kp' | 'player';
  character_id?: string;
  session_state: GameState;
}

function checkVisibility(
  visibility: Visibility | VisibilityRule,
  context: VisibilityContext
): boolean {
  // 基础可见性
  const baseVisibility = typeof visibility === 'string'
    ? visibility
    : visibility.base;

  // 公开: 所有人可见
  if (baseVisibility === 'public') {
    return true;
  }

  // 仅 KP: 只有 KP 可见
  if (baseVisibility === 'kp') {
    return context.viewer_role === 'kp';
  }

  // 所有玩家: 玩家可见,KP 也可见
  if (baseVisibility === 'player:*') {
    return context.viewer_role === 'player' || context.viewer_role === 'kp';
  }

  // 特定玩家
  if (baseVisibility.startsWith('player:')) {
    const targetPlayerId = baseVisibility.split(':')[1];
    return context.viewer_id === targetPlayerId || context.viewer_role === 'kp';
  }

  // 条件可见
  if (baseVisibility === 'conditional' && typeof visibility !== 'string') {
    return evaluateVisibilityCondition(visibility.condition, context);
  }

  return false;
}
```

### 条件求值

```typescript
function evaluateVisibilityCondition(
  condition: VisibilityCondition | undefined,
  context: VisibilityContext
): boolean {
  if (!condition) return false;

  // 角色条件
  if (condition.character) {
    const { character_id } = context;
    if (!character_id) return false;

    if (condition.character.has_id && character_id !== condition.character.has_id) {
      return false;
    }

    if (condition.character.has_item) {
      const character = context.session_state.characters[character_id];
      if (!character?.inventory.includes(condition.character.has_item)) {
        return false;
      }
    }

    if (condition.character.has_clue) {
      const character = context.session_state.characters[character_id];
      if (!character?.clues.includes(condition.character.has_clue)) {
        return false;
      }
    }
  }

  // 状态条件
  if (condition.state) {
    const value = getNestedValue(
      context.session_state,
      condition.state.field
    );
    if (!compare(value, condition.state.operator, condition.state.value)) {
      return false;
    }
  }

  // 场景条件
  if (condition.scene) {
    const currentScene = context.session_state.current_scene;

    if (condition.scene.in_scene && currentScene !== condition.scene.in_scene) {
      return false;
    }

    if (condition.scene.left_scene) {
      const visited = context.session_state.scenes_visited || [];
      if (!visited.includes(condition.scene.left_scene)) {
        return false;
      }
    }
  }

  // 事件条件
  if (condition.event) {
    const events = context.session_state.events || [];

    if (condition.event.occurred && !events.includes(condition.event.occurred)) {
      return false;
    }

    if (condition.event.not_occurred && events.includes(condition.event.not_occurred)) {
      return false;
    }
  }

  // 时间条件
  if (condition.time) {
    const now = Date.now();

    if (condition.time.after) {
      const after = new Date(condition.time.after).getTime();
      if (now < after) return false;
    }

    if (condition.time.before) {
      const before = new Date(condition.time.before).getTime();
      if (now >= before) return false;
    }
  }

  // 组合条件
  if (condition.conditions) {
    const results = condition.conditions.map(c =>
      evaluateVisibilityCondition(c, context)
    );

    return condition.logic === 'AND'
      ? results.every(r => r)
      : results.some(r => r);
  }

  return true;
}

function compare(value: any, operator: string, target: any): boolean {
  switch (operator) {
    case 'eq': return value === target;
    case 'ne': return value !== target;
    case 'gt': return value > target;
    case 'lt': return value < target;
    case 'gte': return value >= target;
    case 'lte': return value <= target;
    default: return false;
  }
}
```

### 可见性继承

```typescript
interface VisibilityInheritanceRule {
  // 父对象的可见性
  parent_visibility: Visibility;

  // 子对象默认可见性
  child_default: Visibility;

  // 可覆盖的字段
  overridable_fields: string[];

  // 强制继承的字段
  inherited_fields: string[];
}

// 可见性继承规则
const VISIBILITY_INHERITANCE: Record<string, VisibilityInheritanceRule> = {
  scene: {
    parent_visibility: 'public',
    child_default: 'public',
    overridable_fields: ['narrative', 'clues', 'npcs'],
    inherited_fields: ['id', 'title', 'order']
  },

  npc: {
    parent_visibility: 'public',
    child_default: 'public',
    overridable_fields: ['stats', 'inventory', 'notes'],
    inherited_fields: ['id', 'name']
  },

  clue: {
    parent_visibility: 'kp', // 线索默认只有 KP 可见
    child_default: 'kp',
    overridable_fields: ['description', 'source'],
    inherited_fields: ['id', 'title']
  }
};

function applyInheritance(
  parent: { visibility?: Visibility },
  child: any,
  rule: VisibilityInheritanceRule
): Visibility {
  // 子对象已指定可见性
  if (child.visibility) {
    return child.visibility;
  }

  // 使用父对象的可见性
  if (parent.visibility) {
    return parent.visibility;
  }

  // 使用默认值
  return rule.child_default;
}
```

### 可见性覆盖

```typescript
interface VisibilityOverride {
  // 覆盖者
  overrider: 'kp' | 'system';

  // 覆盖原因
  reason: string;

  // 临时覆盖
  temporary?: boolean;
  duration?: number; // 毫秒
}

class VisibilityManager {
  private overrides: Map<string, VisibilityOverride> = new Map();

  // 设置覆盖
  setOverride(
    eventId: string,
    override: VisibilityOverride
  ): void {
    this.overrides.set(eventId, override);

    if (override.temporary && override.duration) {
      setTimeout(() => {
        this.overrides.delete(eventId);
      }, override.duration);
    }
  }

  // 获取最终可见性
  getFinalVisibility(
    eventId: string,
    baseVisibility: Visibility,
    context: VisibilityContext
  ): Visibility {
    // 检查覆盖
    const override = this.overrides.get(eventId);
    if (override) {
      // KP 可以覆盖任何可见性
      if (override.overrider === 'kp' && context.viewer_role === 'kp') {
        return 'public';
      }
    }

    return baseVisibility;
  }

  // 清除覆盖
  clearOverride(eventId: string): void {
    this.overrides.delete(eventId);
  }
}
```

### 可见性表达式

```typescript
// 支持的表达式语法
type VisibilityExpression =
  | string  // 简单表达式: "public", "kp", "player:123"
  | {
      // 条件表达式
      if: string;
      then: VisibilityExpression;
      else?: VisibilityExpression;
    }
  | {
      // 组合表达式
      any: VisibilityExpression[];
    }
  | {
      all: VisibilityExpression[];
    };

// 解析表达式
function parseVisibilityExpression(
  expr: VisibilityExpression,
  context: VisibilityContext
): boolean {
  if (typeof expr === 'string') {
    return checkVisibility(expr as Visibility, context);
  }

  if ('if' in expr) {
    const condition = evaluateCondition(expr.if, context);
    return condition
      ? parseVisibilityExpression(expr.then, context)
      : expr_else
        ? parseVisibilityExpression(expr.else, context)
        : false;
  }

  if ('any' in expr) {
    return expr.any.some(e => parseVisibilityExpression(e, context));
  }

  if ('all' in expr) {
    return expr.all.every(e => parseVisibilityExpression(e, context));
  }

  return false;
}

// 表达示例
const visibilityExamples = {
  // 简单公开
  simplePublic: 'public',

  // 有特定物品才可见
  hasItem: {
    if: 'has_item("secret_key")',
    then: 'public',
    else: 'kp'
  },

  // KP 或特定玩家
  kpOrPlayer: {
    any: ['kp', 'player:123']
  },

  // 满足所有条件
  allConditions: {
    all: [
      { if: 'has_clue("murder_weapon")', then: 'public' },
      { if: 'state.HP < 10', then: 'public' }
    ]
  }
};
```

## 依赖关系
- 前置任务: M0-035 定义 Event 基础结构
- 被依赖: M0-038 定义 StateChange 变更结构

## 预估工时
2h
