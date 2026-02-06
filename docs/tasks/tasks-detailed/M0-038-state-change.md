# M0-038 定义 StateChange 变更结构

## 概述
定义游戏状态变更(StateChange)的数据结构,用于记录事件对游戏状态的影响,支持状态追溯和撤销。

## 验收标准
- [ ] 定义状态变更类型
- [ ] 定义变更值结构
- [ ] 定义变更目标(角色/场景/全局)
- [ ] 定义变更元数据
- [ ] 定义变更操作(增/删/改)
- [ ] 支持批量变更

## 技术方案

### 状态变更结构

```typescript
interface StateChange {
  // 变更 ID
  id: string;

  // 变更类型
  type: StateChangeType;

  // 目标
  target: StateChangeTarget;

  // 变更内容
  change: StateChangeValue;

  // 原值
  previous_value?: any;

  // 操作类型
  operation: 'add' | 'subtract' | 'set' | 'push' | 'delete';

  // 元数据
  metadata: {
    timestamp: string;
    event_id: string;
    actor_id: string;
    reason?: string;
    reversible: boolean; // 是否可撤销
  };
}
```

### 变更类型

```typescript
enum StateChangeType {
  // 属性变更
  ATTRIBUTE = 'attribute',

  // 技能变更
  SKILL = 'skill',

  // 派生值变更
  DERIVED = 'derived',

  // 状态标志变更
  STATUS = 'status',

  // 物品变更
  INVENTORY = 'inventory',

  // 线索变更
  CLUE = 'clue',

  // 场景状态变更
  SCENE = 'scene',

  // 战斗状态变更
  COMBAT = 'combat',

  // 追逐状态变更
  CHASE = 'chase',

  // SAN 变更
  SANITY = 'sanity',

  // 自定义变更
  CUSTOM = 'custom'
}
```

### 变更目标

```typescript
interface StateChangeTarget {
  // 目标类型
  scope: 'character' | 'scene' | 'global' | 'combat' | 'chase';

  // 目标 ID
  target_id?: string; // character_id, scene_id 等

  // 字段路径
  field: string; // 例如: "attributes.STR", "skills.library_use"

  // 数组索引(如果是数组操作)
  index?: number;
}
```

### 变更值

```typescript
type StateChangeValue =
  | number    // 数值: HP 变更,技能经验增加
  | string    // 字符串: 状态变更
  | boolean   // 布尔: 标志位变更
  | any[]     // 数组: 物品列表,线索列表
  | object    // 对象: 复杂结构
  | null;     // 删除

// 类型化的变更值
interface TypedStateChange {
  // 属性变更
  attribute: {
    attribute: keyof Attributes;
    delta: number;
    new_value: number;
    reason?: string;
  };

  // 技能变更
  skill: {
    skill_name: string;
    delta: number; // 经验增加
    new_value: number;
    check_success?: boolean; // 是否通过成功检定
  };

  // 派生值变更
  derived: {
    field: 'HP' | 'MP' | 'SAN' | 'Luck' | 'Move';
    delta: number;
    new_value: number;
    max?: number;
  };

  // 状态变更
  status: {
    status: 'alive' | 'unconscious' | 'dying' | 'dead' | 'insane';
    previous_status?: string;
  };

  // 物品变更
  inventory: {
    item_id: string;
    operation: 'add' | 'remove' | 'update';
    quantity?: number;
    item_data?: any;
  };

  // 线索变更
  clue: {
    clue_id: string;
    operation: 'discover' | 'forget';
    clue_data?: ClueData;
  };

  // 场景变更
  scene: {
    property: string;
    value: any;
  };
}
```

### 批量变更

```typescript
interface StateChangeBatch {
  // 批次 ID
  batch_id: string;

  // 变更列表
  changes: StateChange[];

  // 原子性: 全部成功或全部失败
  atomic: boolean;

  // 元数据
  metadata: {
    timestamp: string;
    event_id: string;
    description?: string;
  };

  // 执行结果
  result?: {
    success: boolean;
    failed_index?: number;
    error?: string;
  };
}

// 执行批量变更
function applyStateChangeBatch(
  state: GameState,
  batch: StateChangeBatch
): { success: boolean; new_state: GameState; error?: string } {
  let newState = { ...state };
  const failedIndex = -1;

  for (let i = 0; i < batch.changes.length; i++) {
    const change = batch.changes[i];

    try {
      const result = applyStateChange(newState, change);
      if (!result.success) {
        if (batch.atomic) {
          // 原子性: 失败则回滚
          return {
            success: false,
            new_state: state,
            error: `Change at index ${i} failed: ${result.error}`
          };
        }
        // 非原子: 继续执行
      }
      newState = result.new_state;
    } catch (error) {
      if (batch.atomic) {
        return {
          success: false,
          new_state: state,
          error: `Change at index ${i} threw error: ${error.message}`
        };
      }
    }
  }

  return {
    success: true,
    new_state: newState
  };
}
```

### 状态变更应用

```typescript
function applyStateChange(
  state: GameState,
  change: StateChange
): { success: boolean; new_state: GameState; error?: string } {
  const newState = JSON.parse(JSON.stringify(state)); // 深拷贝
  const target = change.target;

  // 获取目标对象
  let targetObj = newState;
  if (target.scope === 'character' && target.target_id) {
    targetObj = newState.characters[target.target_id];
    if (!targetObj) {
      return { success: false, new_state: state, error: 'Character not found' };
    }
  }

  // 获取当前值
  const currentValue = getNestedValue(targetObj, target.field);
  change.previous_value = currentValue;

  // 应用变更
  let newValue = change.change;

  switch (change.operation) {
    case 'add':
      newValue = currentValue + change.change;
      break;

    case 'subtract':
      newValue = currentValue - change.change;
      break;

    case 'set':
      newValue = change.change;
      break;

    case 'push':
      if (!Array.isArray(currentValue)) {
        return { success: false, new_state: state, error: 'Target is not an array' };
      }
      newValue = [...currentValue, change.change];
      break;

    case 'delete':
      if (Array.isArray(currentValue)) {
        newValue = currentValue.filter((_, i) => i !== target.index);
      } else {
        newValue = undefined;
      }
      break;
  }

  // 设置新值
  setNestedValue(targetObj, target.field, newValue);

  // 验证
  const validation = validateStateChange(change, targetObj);
  if (!validation.valid) {
    return {
      success: false,
      new_state: state,
      error: validation.error
    };
  }

  return {
    success: true,
    new_state: newState
  };
}

// 嵌套值访问
function getNestedValue(obj: any, path: string): any {
  return path.split('.').reduce((current, key) => {
    return current?.[key];
  }, obj);
}

// 嵌套值设置
function setNestedValue(obj: any, path: string, value: any): void {
  const keys = path.split('.');
  const lastKey = keys.pop()!;
  const target = keys.reduce((current, key) => {
    if (!current[key]) {
      current[key] = {};
    }
    return current[key];
  }, obj);
  target[lastKey] = value;
}
```

### 状态验证

```typescript
function validateStateChange(
  change: StateChange,
  target: any
): { valid: boolean; error?: string } {
  // 数值范围验证
  if (typeof change.change === 'number') {
    // HP 不能超过最大值
    if (change.target.field === 'derived.HP') {
      const maxHP = target.derived?.HP_max || 100;
      if (change.change > maxHP) {
        return {
          valid: false,
          error: `HP cannot exceed maximum ${maxHP}`
        };
      }
    }

    // SAN 值范围 0-99
    if (change.target.field === 'derived.SAN') {
      if (change.change < 0 || change.change > 99) {
        return {
          valid: false,
          error: 'SAN must be between 0 and 99'
        };
      }
    }
  }

  // 数组验证
  if (change.operation === 'push' && !Array.isArray(getNestedValue(target, change.target.field))) {
    return {
      valid: false,
      error: 'Cannot push to non-array field'
    };
  }

  // 存在性验证
  if (change.operation === 'delete') {
    const current = getNestedValue(target, change.target.field);
    if (current === undefined) {
      return {
        valid: false,
        error: 'Cannot delete non-existent field'
      };
    }
  }

  return { valid: true };
}
```

### 状态撤销

```typescript
interface StateChangeHistory {
  changes: StateChange[];
  undo_stack: StateChange[][];
  redo_stack: StateChange[][];

  // 当前位置
  position: number;
}

// 撤销变更
function undoStateChanges(
  state: GameState,
  history: StateChangeHistory,
  count: number = 1
): { success: boolean; new_state: GameState; new_history: StateChangeHistory } {
  if (history.position < count) {
    return { success: false, new_state: state, new_history: history };
  }

  let newState = state;
  const changesToUndo: StateChange[][] = [];

  for (let i = 0; i < count; i++) {
    const changes = history.undo_stack[history.position - 1 - i];
    changesToUndo.push(changes);

    for (const change of changes) {
      if (change.metadata.reversible && change.previous_value !== undefined) {
        // 恢复原值
        const reverseChange: StateChange = {
          ...change,
          change: change.previous_value,
          operation: 'set'
        };
        const result = applyStateChange(newState, reverseChange);
        if (result.success) {
          newState = result.new_state;
        }
      }
    }
  }

  const newHistory: StateChangeHistory = {
    ...history,
    redo_stack: [...history.redo_stack, ...changesToUndo.reverse()],
    undo_stack: history.undo_stack.slice(0, -count),
    position: history.position - count
  };

  return {
    success: true,
    new_state: newState,
    new_history
  };
}

// 重做变更
function redoStateChanges(
  state: GameState,
  history: StateChangeHistory,
  count: number = 1
): { success: boolean; new_state: GameState; new_history: StateChangeHistory } {
  if (history.redo_stack.length < count) {
    return { success: false, new_state: state, new_history: history };
  }

  let newState = state;
  const changesToRedo = history.redo_stack.slice(-count);

  for (const changes of changesToRedo) {
    for (const change of changes) {
      const result = applyStateChange(newState, change);
      if (result.success) {
        newState = result.new_state;
      }
    }
  }

  const newHistory: StateChangeHistory = {
    ...history,
    redo_stack: history.redo_stack.slice(0, -count),
    undo_stack: [...history.undo_stack, ...changesToRedo],
    position: history.position + count
  };

  return {
    success: true,
    new_state: newState,
    new_history
  };
}
```

### 变更序列化

```typescript
// 序列化变更
function serializeStateChange(change: StateChange): string {
  return JSON.stringify({
    id: change.id,
    type: change.type,
    target: change.target,
    change: change.change,
    operation: change.operation,
    metadata: change.metadata
  });
}

// 反序列化变更
function deserializeStateChange(data: string): StateChange {
  const parsed = JSON.parse(data);
  return {
    id: parsed.id,
    type: parsed.type,
    target: parsed.target,
    change: parsed.change,
    operation: parsed.operation,
    metadata: parsed.metadata
  };
}

// 导出变更日志
function exportChangeLog(history: StateChangeHistory): string {
  const log = history.changes.map(change => ({
    timestamp: change.metadata.timestamp,
    event_id: change.metadata.event_id,
    actor_id: change.metadata.actor_id,
    type: change.type,
    target: change.target,
    previous: change.previous_value,
    new: change.change,
    reason: change.metadata.reason
  }));

  return JSON.stringify(log, null, 2);
}
```

## 依赖关系
- 前置任务: M0-036 定义 EventType 枚举, M0-037 定义 Visibility 可见性
- 被依赖: M1-103 设计 Events 表结构

## 预估工时
2h
