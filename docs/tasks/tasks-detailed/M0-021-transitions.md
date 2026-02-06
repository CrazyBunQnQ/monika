# M0-021 定义 transitions 跳转结构

## 概述
定义场景之间的跳转(transition)数据结构,支持条件跳转、随机跳转和脚本式跳转,实现剧情分支和多重结局系统。

## 验收标准
- [ ] 定义基础跳转结构(目标场景ID)
- [ ] 实现条件跳转(基于状态、物品、线索)
- [ ] 实现随机跳转(概率分配)
- [ ] 实现多目标跳转(玩家选择)
- [ ] 定义跳转触发器(对话选项、事件完成)
- [ ] 支持跳转参数传递

## 技术方案

### 数据结构设计

```typescript
interface Transition {
  id: string;
  type: 'immediate' | 'conditional' | 'random' | 'choice';
  trigger: {
    type: 'dialogue' | 'event' | 'manual' | 'auto';
    condition?: string; // 触发条件表达式
  };

  // 条件跳转
  condition?: {
    field: string; // 状态字段路径
    operator: 'eq' | 'ne' | 'gt' | 'lt' | 'gte' | 'lte' | 'in' | 'contains';
    value: any;
    logic?: 'AND' | 'OR';
  }[];

  // 随机跳转
  random?: {
    target: string;
    weight: number; // 权重,0-100
  }[];

  // 玩家选择
  choices?: {
    target: string;
    label: string;
    visible?: string; // 可见性条件
    enabled?: string; // 可用性条件
  }[];

  // 立即跳转
  target?: string;

  // 参数传递
  params?: Record<string, any>;

  // 附加效果
  effects?: {
    type: 'narrative' | 'state_change' | 'item_give' | 'san_check';
    data: any;
  }[];
}

// 场景中的跳转集合
interface SceneTransitions {
  on_enter?: Transition[]; // 进入时触发
  on_exit?: Transition[]; // 离开时触发
  on_action?: Transition[]; // 玩家行动后触发
  dialogue?: Transition[]; // 对话选项触发
}
```

### 条件表达式语法

```javascript
// 状态检查
state.HP < 10
state.has_clue('murder_weapon')
state.inventory.contains('key')

// 技能检定
skill_check('investigation', 30)

// 计数器
counter('visits') >= 3

// 组合条件
state.HP < 10 AND state.has_clue('murder_weapon')
```

### 跳转示例

```json
{
  "type": "conditional",
  "trigger": {
    "type": "dialogue",
    "condition": "state.investigation_complete"
  },
  "condition": [
    {"field": "state.SAN", "operator": "lt", "value": 50},
    {"field": "state.has_clue('truth')", "operator": "eq", "value": true, "logic": "OR"}
  ],
  "choices": [
    {
      "target": "scene_good_end",
      "label": "相信他",
      "visible": "state.SAN >= 50"
    },
    {
      "target": "scene_bad_end",
      "label": "揭穿他",
      "enabled": "state.has_clue('evidence')"
    }
  ]
}
```

## 依赖关系
- 前置任务: M0-016 定义 scenes 场景集合结构
- 被依赖: M0-022 编写场景包 JSON Schema

## 预估工时
2h
