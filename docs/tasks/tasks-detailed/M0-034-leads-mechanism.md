# M0-034 定义 Leads 机制结构

## 概述
定义 CoC 游戏中的 Leads(可选行动/线索提示)机制,帮助玩家了解当前可用的行动和选择,提升游戏体验。

## 验收标准
- [ ] 定义 Lead 数据结构
- [ ] 定义 Lead 类型(行动/线索/提示)
- [ ] 定义 Lead 触发条件
- [ ] 定义 Lead 可见性规则(KP/玩家)
- [ ] 定义 Lead 过期机制
- [ ] 定义 Lead 优先级

## 技术方案

### Lead 结构

```typescript
interface Lead {
  id: string;

  // 类型
  type: 'action' | 'clue' | 'hint' | 'dialogue' | 'investigation';

  // 内容
  title: string; // 简短标题
  description: string; // 详细描述

  // 可见性
  visibility: 'public' | 'kp' | 'player' | 'conditional';

  // 触发条件
  trigger?: {
    type: 'always' | 'location' | 'item' | 'clue' | 'state' | 'time';
    condition: string; // 条件表达式
  };

  // 行动
  action?: {
    type: 'check' | 'move' | 'dialogue' | 'use_item';
    data: any;
  };

  // 优先级
  priority: 0 | 1 | 2 | 3; // 0=低, 3=高

  // 过期
  expire_on?: {
    event: string; // 事件 ID
    state: string; // 状态条件
  };

  // 元数据
  metadata: {
    scene_id: string;
    category?: string; // 分类
    tags?: string[]; // 标签
    suggested_by?: 'system' | 'kp';
  };
}
```

### Leads 集合

```typescript
interface LeadsCollection {
  scene_id: string;

  // 当前可用线索
  available: Lead[];

  // 已使用线索
  used: string[]; // Lead IDs

  // 隐藏线索(KP 可见)
  hidden: Lead[];

  // 过期线索
  expired: Lead[];

  // 分类索引
  by_category: Record<string, Lead[]>;

  // 优先级排序
  by_priority: Lead[];
}
```

### Lead 类型详解

```typescript
// 行动类 Lead
interface ActionLead extends Lead {
  type: 'action';

  // 行动类型
  action: {
    type: 'check' | 'combat' | 'chase' | 'rest';

    // 检定行动
    check?: {
      skill: string;
      difficulty: number;
      target?: string;
    };

    // 战斗行动
    combat?: {
      target: string;
      action: 'attack' | 'defend' | 'maneuver';
    };

    // 移动行动
    move?: {
      destination: string;
      method?: 'walk' | 'run' | 'drive';
    };
  };
}

// 线索类 Lead
interface ClueLead extends Lead {
  type: 'clue';

  clue: {
    clue_id: string;
    description: string;
    difficulty?: number; // 发现难度
    skill?: string; // 所需技能
  };
}

// 对话类 Lead
interface DialogueLead extends Lead {
  type: 'dialogue';

  dialogue: {
    target: string; // NPC ID
    topic: string; // 话题
    options: string[]; // 对话选项
  };
}

// 提示类 Lead
interface HintLead extends Lead {
  type: 'hint';

  hint: {
    level: 'subtle' | 'obvious' | 'direct';
    content: string;
    spoiler: boolean; // 是否剧透
  };
}

// 调查类 Lead
interface InvestigationLead extends Lead {
  type: 'investigation';

  investigation: {
    location: string; // 地点 ID
    actions: string[]; // 可用行动
    requirements?: {
      items?: string[];
      skills?: string[];
    };
  };
}
```

### 触发条件

```typescript
interface TriggerCondition {
  // 位置触发
  location?: {
    in_location: string; // 在特定位置
    near_object: string; // 靠近物品
  };

  // 状态触发
  state?: {
    field: string; // 状态字段
    operator: 'eq' | 'ne' | 'gt' | 'lt' | 'gte' | 'lte';
    value: any;
  };

  // 物品触发
  item?: {
    has_item: string; // 拥有物品
    quantity?: number;
  };

  // 线索触发
  clue?: {
    has_clue: string; // 发现线索
  };

  // 时间触发
  time?: {
    round?: number; // 特定回合
    time_of_day?: string; // 一天中的时间
  };

  // 事件触发
  event?: {
    after_event: string; // 事件发生后
    before_event: string; // 事件发生前
  };

  // 组合条件
  logic?: 'AND' | 'OR';
  conditions?: TriggerCondition[];
}
```

### 条件表达式

```typescript
// 支持的表达式
type ConditionExpression =
  | string // 简单字段: "state.HP < 10"
  | {
      and: ConditionExpression[];
    }
  | {
      or: ConditionExpression[];
    }
  | {
      not: ConditionExpression;
    };

// 示例
const conditionExamples = [
  // 简单条件
  "state.HP < 10",

  // 拥有物品
  "has_item('key')",

  // 发现线索
  "has_clue('murder_weapon')",

  // 技能检定
  "skill_check('investigation', 30)",

  // 组合条件
  {
    and: [
      "state.HP < 10",
      "has_item('medkit')"
    ]
  },

  // 复杂条件
  {
    or: [
      "has_clue('evidence_a')",
      {
        and: [
          "has_clue('evidence_b')",
          "skill_check('psychology', 50)"
        ]
      }
    ]
  }
];

// 条件求值
function evaluateCondition(
  expression: ConditionExpression,
  state: GameState
): boolean {
  if (typeof expression === 'string') {
    return parseAndEvaluate(expression, state);
  }

  if ('and' in expression) {
    return expression.and.every(expr => evaluateCondition(expr, state));
  }

  if ('or' in expression) {
    return expression.or.some(expr => evaluateCondition(expr, state));
  }

  if ('not' in expression) {
    return !evaluateCondition(expression.not, state);
  }

  return false;
}
```

### 可见性规则

```typescript
interface VisibilityRule {
  // 基础可见性
  base: 'public' | 'kp' | 'player';

  // 条件可见性
  condition?: ConditionExpression;

  // KP 覆盖(可强制显示)
  kp_override: boolean;

  // 玩家发现条件
  player_discovery?: {
    check: {
      skill: string;
      difficulty: number;
    };
    automatic?: boolean; // 自动显示
  };
}

function getVisibleLeads(
  leads: Lead[],
  viewer: 'kp' | 'player',
  state: GameState
): Lead[] {
  return leads.filter(lead => {
    // KP 可见所有(除非明确隐藏)
    if (viewer === 'kp' && lead.visibility !== 'hidden') {
      return true;
    }

    // 玩家只能看公开的
    if (viewer === 'player') {
      if (lead.visibility === 'kp') return false;

      // 条件可见性
      if (lead.visibility === 'conditional' && lead.trigger?.condition) {
        return evaluateCondition(lead.trigger.condition, state);
      }
    }

    return true;
  });
}
```

### 过期机制

```typescript
interface ExpiryRule {
  // 事件过期
  on_event: string; // 事件发生后过期

  // 状态过期
  on_state: {
    field: string;
    operator: 'eq' | 'ne' | 'gt' | 'lt';
    value: any;
  };

  // 时间过期
  on_time: {
    type: 'absolute' | 'relative';
    timestamp?: string; // 绝对时间
    duration?: number; // 相对时长(秒)
  };

  // 使用过期
  on_use: boolean; // 使用后过期
}

function checkExpiry(lead: Lead, state: GameState): boolean {
  if (!lead.expire_on) return false;

  // 检查事件过期
  if (lead.expire_on.event) {
    if (state.events.includes(lead.expire_on.event)) {
      return true;
    }
  }

  // 检查状态过期
  if (lead.expire_on.state) {
    const value = getNestedValue(state, lead.expire_on.state.field);
    if (compare(value, lead.expire_on.state.operator, lead.expire_on.state.value)) {
      return true;
    }
  }

  // 检查时间过期
  if (lead.expire_on.on_time) {
    const expiryRule = lead.expire_on.on_time;
    const now = Date.now();

    if (expiryRule.type === 'absolute' && expiryRule.timestamp) {
      return now >= new Date(expiryRule.timestamp).getTime();
    }

    if (expiryRule.type === 'relative' && expiryRule.duration) {
      const created = new Date(lead.metadata.created_at || now).getTime();
      return now >= created + expiryRule.duration * 1000;
    }
  }

  return false;
}
```

### 优先级系统

```typescript
interface LeadPriority {
  // 优先级等级
  level: 0 | 1 | 2 | 3;

  // 排序权重
  weight: number;

  // 显示顺序
  order: 'first' | 'last' | 'normal';
}

function sortLeadsByPriority(leads: Lead[]): Lead[] {
  return leads.sort((a, b) => {
    // 先按优先级排序
    if (a.priority !== b.priority) {
      return b.priority - a.priority; // 高优先级在前
    }

    // 同优先级按类型排序
    const typeOrder = ['action', 'clue', 'dialogue', 'investigation', 'hint'];
    const aTypeIndex = typeOrder.indexOf(a.type);
    const bTypeIndex = typeOrder.indexOf(b.type);

    if (aTypeIndex !== bTypeIndex) {
      return aTypeIndex - bTypeIndex;
    }

    // 同类型按 ID 排序
    return a.id.localeCompare(b.id);
  });
}
```

### Lead 生成示例

```typescript
// 自动生成行动 Lead
function generateActionLeads(scene: Scene, state: GameState): ActionLead[] {
  const leads: ActionLead[] = [];

  // NPC 对话
  scene.npcs.forEach(npc => {
    leads.push({
      id: `lead_dialogue_${npc.id}`,
      type: 'dialogue',
      title: `与 ${npc.name} 对话`,
      description: '询问信息或建立关系',
      visibility: 'public',
      trigger: {
        type: 'location',
        condition: `in_location('${scene.id}')`
      },
      action: {
        type: 'dialogue',
        data: { target: npc.id }
      },
      priority: 2,
      metadata: {
        scene_id: scene.id,
        category: 'social',
        suggested_by: 'system'
      }
    });
  });

  // 调查行动
  scene.locations.forEach(location => {
    leads.push({
      id: `lead_investigate_${location.id}`,
      type: 'investigation',
      title: `调查 ${location.name}`,
      description: location.description,
      visibility: 'public',
      trigger: {
        type: 'location',
        condition: `in_location('${scene.id}')`
      },
      action: {
        type: 'check',
        data: { skill: 'spot_hidden', difficulty: location.difficulty || 0 }
      },
      priority: 1,
      metadata: {
        scene_id: scene.id,
        category: 'investigation',
        suggested_by: 'system'
      }
    });
  });

  return leads;
}
```

## 依赖关系
- 前置任务: M0-027 定义 SessionState 结构
- 被依赖: 无

## 预估工时
2h
