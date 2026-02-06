# M0-036 定义 EventType 枚举

## 概述
定义 CoC 游戏中所有可能的事件类型枚举,包括玩家行动、检定结果、状态变化、战斗事件等,用于事件日志和状态追踪。

## 验收标准
- [ ] 定义所有事件类型枚举
- [ ] 定义事件类型分组
- [ ] 定义事件优先级
- [ ] 定义事件可见性默认值
- [ ] 定义事件数据结构

## 技术方案

### 事件类型枚举

```typescript
enum EventType {
  // 基础事件
  MESSAGE = 'message',              // 普通消息
  ACTION = 'action',                // 行动声明
  NARRATIVE = 'narrative',          // 叙述文本

  // 检定事件
  ROLL = 'roll',                    // 普通检定
  ROLL_PUSHED = 'roll_pushed',      // 推骰
  LUCK_SPENT = 'luck_spent',        // 花幸运

  // 战斗事件
  COMBAT_START = 'combat_start',    // 战斗开始
  COMBAT_ACTION = 'combat_action',  // 战斗行动
  COMBAT_END = 'combat_end',        // 战斗结束
  DAMAGE = 'damage',                // 伤害结算
  HEAL = 'heal',                    // 治疗

  // 追逐事件
  CHASE_START = 'chase_start',      // 追逐开始
  CHASE_ACTION = 'chase_action',    // 追逐行动
  CHASE_END = 'chase_end',          // 追逐结束

  // SAN 事件
  SAN_CHECK = 'san_check',          // SAN 检定
  SAN_LOSS = 'san_loss',            // SAN 丢失
  MADNESS_START = 'madness_start',  // 疯狂开始
  MADNESS_END = 'madness_end',      // 疯狂结束

  // 状态事件
  CHECKPOINT = 'checkpoint',        // 检查点
  SCENE_CHANGE = 'scene_change',    // 场景切换

  // 角色事件
  CHARACTER_CREATE = 'character_create',  // 角色创建
  CHARACTER_UPDATE = 'character_update',  // 角色更新
  CHARACTER_DELETE = 'character_delete',  // 角色删除

  // 物品事件
  ITEM_ACQUIRE = 'item_acquire',    // 获得物品
  ITEM_LOSE = 'item_lose',          // 失去物品
  ITEM_USE = 'item_use',            // 使用物品

  // 系统事件
  SESSION_START = 'session_start',  // 会话开始
  SESSION_END = 'session_end',      // 会话结束
  SAVE = 'save',                    // 保存游戏
  LOAD = 'load',                    // 加载游戏

  // KP 事件
  KP_ANNOUNCE = 'kp_announce',      // KP 公告
  KP_GIVE_CLUE = 'kp_give_clue',    // KP 给予线索
  KP_MODIFY = 'kp_modify',          // KP 修改状态

  // 玩家事件
  PLAYER_JOIN = 'player_join',      // 玩家加入
  PLAYER_LEAVE = 'player_leave',    // 玩家离开
  PLAYER_READY = 'player_ready',    // 玩家准备
}
```

### 事件类型分组

```typescript
type EventGroup = {
  name: string;
  types: EventType[];
  color: string; // UI 显示颜色
  icon: string;  // UI 图标
};

const EVENT_GROUPS: EventGroup[] = [
  {
    name: 'basic',
    types: [EventType.MESSAGE, EventType.ACTION, EventType.NARRATIVE],
    color: '#6c757d',
    icon: 'comment'
  },
  {
    name: 'roll',
    types: [EventType.ROLL, EventType.ROLL_PUSHED, EventType.LUCK_SPENT],
    color: '#5c6bc0',
    icon: 'dice'
  },
  {
    name: 'combat',
    types: [
      EventType.COMBAT_START,
      EventType.COMBAT_ACTION,
      EventType.COMBAT_END,
      EventType.DAMAGE,
      EventType.HEAL
    ],
    color: '#ef5350',
    icon: 'sword'
  },
  {
    name: 'chase',
    types: [
      EventType.CHASE_START,
      EventType.CHASE_ACTION,
      EventType.CHASE_END
    ],
    color: '#ffa726',
    icon: 'run'
  },
  {
    name: 'sanity',
    types: [
      EventType.SAN_CHECK,
      EventType.SAN_LOSS,
      EventType.MADNESS_START,
      EventType.MADNESS_END
    ],
    color: '#ab47bc',
    icon: 'brain'
  },
  {
    name: 'character',
    types: [
      EventType.CHARACTER_CREATE,
      EventType.CHARACTER_UPDATE,
      EventType.CHARACTER_DELETE
    ],
    color: '#26a69a',
    icon: 'user'
  },
  {
    name: 'item',
    types: [
      EventType.ITEM_ACQUIRE,
      EventType.ITEM_LOSE,
      EventType.ITEM_USE
    ],
    color: '#78909c',
    icon: 'box'
  },
  {
    name: 'system',
    types: [
      EventType.SESSION_START,
      EventType.SESSION_END,
      EventType.SAVE,
      EventType.LOAD,
      EventType.CHECKPOINT,
      EventType.SCENE_CHANGE
    ],
    color: '#42a5f5',
    icon: 'cog'
  },
  {
    name: 'kp',
    types: [
      EventType.KP_ANNOUNCE,
      EventType.KP_GIVE_CLUE,
      EventType.KP_MODIFY
    ],
    color: '#7e57c2',
    icon: 'megaphone'
  },
  {
    name: 'player',
    types: [
      EventType.PLAYER_JOIN,
      EventType.PLAYER_LEAVE,
      EventType.PLAYER_READY
    ],
    color: '#66bb6a',
    icon: 'users'
  }
];
```

### 事件优先级

```typescript
enum EventPriority {
  LOW = 0,       // 低优先级: 普通消息
  NORMAL = 1,    // 普通优先级: 大多数事件
  HIGH = 2,      // 高优先级: 检定、战斗
  CRITICAL = 3   // 关键优先级: 死亡、疯狂
}

const EVENT_PRIORITIES: Record<EventType, EventPriority> = {
  // 基础事件
  [EventType.MESSAGE]: EventPriority.LOW,
  [EventType.ACTION]: EventPriority.NORMAL,
  [EventType.NARRATIVE]: EventPriority.NORMAL,

  // 检定事件
  [EventType.ROLL]: EventPriority.HIGH,
  [EventType.ROLL_PUSHED]: EventPriority.HIGH,
  [EventType.LUCK_SPENT]: EventPriority.HIGH,

  // 战斗事件
  [EventType.COMBAT_START]: EventPriority.HIGH,
  [EventType.COMBAT_ACTION]: EventPriority.HIGH,
  [EventType.COMBAT_END]: EventPriority.HIGH,
  [EventType.DAMAGE]: EventPriority.HIGH,
  [EventType.HEAL]: EventPriority.NORMAL,

  // 追逐事件
  [EventType.CHASE_START]: EventPriority.HIGH,
  [EventType.CHASE_ACTION]: EventPriority.HIGH,
  [EventType.CHASE_END]: EventPriority.HIGH,

  // SAN 事件
  [EventType.SAN_CHECK]: EventPriority.HIGH,
  [EventType.SAN_LOSS]: EventPriority.HIGH,
  [EventType.MADNESS_START]: EventPriority.CRITICAL,
  [EventType.MADNESS_END]: EventPriority.HIGH,

  // 状态事件
  [EventType.CHECKPOINT]: EventPriority.NORMAL,
  [EventType.SCENE_CHANGE]: EventPriority.NORMAL,

  // 角色事件
  [EventType.CHARACTER_CREATE]: EventPriority.NORMAL,
  [EventType.CHARACTER_UPDATE]: EventPriority.NORMAL,
  [EventType.CHARACTER_DELETE]: EventPriority.HIGH,

  // 物品事件
  [EventType.ITEM_ACQUIRE]: EventPriority.NORMAL,
  [EventType.ITEM_LOSE]: EventPriority.NORMAL,
  [EventType.ITEM_USE]: EventPriority.NORMAL,

  // 系统事件
  [EventType.SESSION_START]: EventPriority.HIGH,
  [EventType.SESSION_END]: EventPriority.HIGH,
  [EventType.SAVE]: EventPriority.NORMAL,
  [EventType.LOAD]: EventPriority.NORMAL,

  // KP 事件
  [EventType.KP_ANNOUNCE]: EventPriority.HIGH,
  [EventType.KP_GIVE_CLUE]: EventPriority.NORMAL,
  [EventType.KP_MODIFY]: EventPriority.HIGH,

  // 玩家事件
  [EventType.PLAYER_JOIN]: EventPriority.HIGH,
  [EventType.PLAYER_LEAVE]: EventPriority.HIGH,
  [EventType.PLAYER_READY]: EventPriority.NORMAL
};
```

### 事件可见性默认值

```typescript
const EVENT_VISIBILITY_DEFAULTS: Record<EventType, GameEvent['visibility']> = {
  // 基础事件
  [EventType.MESSAGE]: 'public',
  [EventType.ACTION]: 'public',
  [EventType.NARRATIVE]: 'public',

  // 检定事件
  [EventType.ROLL]: 'public',
  [EventType.ROLL_PUSHED]: 'public',
  [EventType.LUCK_SPENT]: 'public',

  // 战斗事件
  [EventType.COMBAT_START]: 'public',
  [EventType.COMBAT_ACTION]: 'public',
  [EventType.COMBAT_END]: 'public',
  [EventType.DAMAGE]: 'public',
  [EventType.HEAL]: 'public',

  // 追逐事件
  [EventType.CHASE_START]: 'public',
  [EventType.CHASE_ACTION]: 'public',
  [EventType.CHASE_END]: 'public',

  // SAN 事件
  [EventType.SAN_CHECK]: 'public',
  [EventType.SAN_LOSS]: 'public',
  [EventType.MADNESS_START]: 'public',
  [EventType.MADNESS_END]: 'public',

  // 状态事件
  [EventType.CHECKPOINT]: 'kp',
  [EventType.SCENE_CHANGE]: 'public',

  // 角色事件
  [EventType.CHARACTER_CREATE]: 'kp',
  [EventType.CHARACTER_UPDATE]: 'kp',
  [EventType.CHARACTER_DELETE]: 'kp',

  // 物品事件
  [EventType.ITEM_ACQUIRE]: 'public',
  [EventType.ITEM_LOSE]: 'public',
  [EventType.ITEM_USE]: 'public',

  // 系统事件
  [EventType.SESSION_START]: 'public',
  [EventType.SESSION_END]: 'public',
  [EventType.SAVE]: 'kp',
  [EventType.LOAD]: 'kp',

  // KP 事件
  [EventType.KP_ANNOUNCE]: 'public',
  [EventType.KP_GIVE_CLUE]: 'player:*',
  [EventType.KP_MODIFY]: 'kp',

  // 玩家事件
  [EventType.PLAYER_JOIN]: 'public',
  [EventType.PLAYER_LEAVE]: 'public',
  [EventType.PLAYER_READY]: 'kp'
};
```

### 事件数据结构

```typescript
interface TypedEventData {
  // 检定事件
  roll: {
    skill: string;
    difficulty: number;
    roll: number;
    success_level: 'failure' | 'regular' | 'hard' | 'extreme' | 'critical';
    bonus_dice?: number[];
    penalty_dice?: number[];
    pushed?: boolean;
  };

  // 战斗事件
  combat_action: {
    actor: string;
    target: string;
    action_type: 'attack' | 'maneuver' | 'defend';
    roll?: number;
    success?: boolean;
    damage?: number;
  };

  // SAN 事件
  san_check: {
    trigger: string;
    difficulty: number;
    roll: number;
    loss: number;
    madness?: boolean;
  };

  // 场景切换
  scene_change: {
    from_scene: string;
    to_scene: string;
    method: 'transition' | 'teleport' | 'narrative';
  };
}
```

### 事件类型工具函数

```typescript
// 获取事件组
function getEventGroup(type: EventType): EventGroup | undefined {
  return EVENT_GROUPS.find(group => group.types.includes(type));
}

// 获取事件优先级
function getEventPriority(type: EventType): EventPriority {
  return EVENT_PRIORITIES[type];
}

// 获取默认可见性
function getDefaultVisibility(type: EventType): GameEvent['visibility'] {
  return EVENT_VISIBILITY_DEFAULTS[type];
}

// 判断是否为关键事件
function isCriticalEvent(type: EventType): boolean {
  return getEventPriority(type) === EventPriority.CRITICAL;
}

// 判断是否为可撤销事件
function isReversibleEvent(type: EventType): boolean {
  return [
    EventType.ROLL,
    EventType.DAMAGE,
    EventType.HEAL,
    EventType.ITEM_ACQUIRE,
    EventType.ITEM_LOSE
  ].includes(type);
}

// 获取事件关联的角色
function getEventCharacters(event: GameEvent): string[] {
  const characters: string[] = [];

  if (event.controlled_character_id) {
    characters.push(event.controlled_character_id);
  }

  if (event.state_changes) {
    event.state_changes.forEach(change => {
      if (change.character_id) {
        characters.push(change.character_id);
      }
    });
  }

  return [...new Set(characters)];
}
```

## 依赖关系
- 前置任务: M0-035 定义 Event 基础结构
- 被依赖: M0-038 定义 StateChange 变更结构

## 预估工时
2h
