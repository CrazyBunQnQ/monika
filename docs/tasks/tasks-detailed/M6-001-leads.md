# M6-001: 设计 Leads 数据结构

**任务ID**: M6-001
**标题**: 设计 Leads 数据结构
**类型**: spec (规范设计)
**预估工时**: 2h
**依赖**: M0

---

## 任务描述

设计 Leads (可选行动) 系统的数据结构，用于在游戏过程中为玩家提供可执行的行动建议，防止卡死。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-001-01 | 分析 Leads 需求 | 确定系统功能需求 | 20min |
| M6-001-02 | 设计 LeadItem 结构 | 单个 Lead 结构 | 25min |
| M6-001-03 | 设计 LeadsState 结构 | 集合状态 | 20min |
| M6-001-04 | 设计 Lead 生成规则 | 生成逻辑 | 25min |
| M6-001-05 | 设计优先级系统 | 优先级排序 | 15min |
| M6-001-06 | 编写 TypeScript 类型 | 类型定义 | 20min |
| M6-001-07 | 编写示例数据 | 典型场景示例 | 10min |

---

## LeadItem 结构

```typescript
interface LeadItem {
  lead_id: string;
  session_id: string;
  timestamp: datetime;
  updated_at: datetime;

  // 基本信息
  title: string;               // 行动标题 (简短)
  description: string;         // 详细描述

  // 分类
  category: LeadCategory;
  type: LeadType;

  // 优先级
  priority: number;            // 0-100, 越高越优先
  urgency: 'low' | 'medium' | 'high' | 'urgent';

  // 状态
  status: LeadStatus;

  // 行动详情
  action: {
    type: ActionType;          // 行动类型
    target?: string;           // 目标 (NPC/地点/物品)
    context?: string;          // 上下文信息
    requirements?: {
      clues?: string[];        // 需要的线索
      skills?: string[];       // 需要的技能
      items?: string[];        // 需要的物品
      state?: Record<string, any>;  // 状态条件
    };
    expected_outcome?: string; // 预期结果
  };

  // 关联信息
  related: {
    scene_id?: string;
    clues?: string[];          // 相关线索
    npcs?: string[];           // 相关 NPC
    locations?: string[];      // 相关地点
  };

  // 来源
  source: {
    type: 'system' | 'kp' | 'player' | 'clue';
    source_id?: string;
    auto_generated: boolean;
  };

  // 有效期
  validity: {
    expires_at?: datetime;     // 过期时间
    one_time_only: boolean;    // 是否一次性
  };

  // 失败后果
  failure_consequence?: {
    description: string;
    cost?: string;             // 时间/资源消耗
    alternatives?: LeadItem[]; // 失败后的替代选项
  };
}

type LeadCategory =
  | 'investigate'        // 调查类
  | 'action'            // 行动类
  | 'social'            // 社交类
  | 'prep'              // 准备类
  | 'explore'           // 探索类
  | 'combat'            // 战斗类
  | 'escape';           // 逃跑类

type LeadType =
  | 'clue_follow'       // 跟随线索
  | 'npc_talk'         // 与 NPC 对话
  | 'location_search'  // 搜索地点
  | 'item_use'         // 使用物品
  | 'skill_check'      // 技能检定
  | 'information_gather' // 收集信息
  | 'rest_recover'     // 休息恢复
  | 'plan_prepare'     // 计划准备
  | 'custom';          // 自定义

type LeadStatus =
  | 'available'         // 可执行
  | 'in_progress'       // 进行中
  | 'completed'         // 已完成
  | 'failed'            // 已失败
  | 'expired'           // 已过期
  | 'blocked';          // 被阻塞

type ActionType =
  | 'talk'              // 对话
  | 'investigate'       // 调查
  | 'search'            // 搜索
  | 'use_item'          // 使用物品
  | 'skill_check'       // 技能检定
  | 'move'              // 移动
  | 'wait'              // 等待
  | 'attack'            // 攻击
  | 'defend'            // 防御
  | 'flee';             // 逃跑
```

---

## LeadsState 结构

```typescript
interface LeadsState {
  session_id: string;
  updated_at: datetime;

  // 当前可用的 Leads
  available: LeadItem[];

  // 已完成/失效的 Leads (历史)
  history: {
    lead: LeadItem;
    completed_at: datetime;
    result?: {
      success: boolean;
      outcome: string;
      rewards?: string[];
    };
  }[];

  // 队列管理
  queue: {
    waiting: string[];      // 等待中的 Lead ID
    active: string;         // 当前活跃的 Lead
    max_concurrent: number; // 最大并发数
  };

  // 生成规则
  generation: {
    min_active: number;     // 最小活跃数
    max_active: number;     // 最大活跃数
    refresh_interval: number; // 刷新间隔 (秒)
    auto_refresh: boolean;  // 自动刷新
  };

  // 设置
  settings: {
    show_priority: boolean;  // 显示优先级
    show_urgency: boolean;   // 显示紧急度
    group_by_category: boolean; // 按类别分组
    sort_by: 'priority' | 'urgency' | 'category' | 'custom';
  };

  // 统计
  stats: {
    total_generated: number;
    total_completed: number;
    completion_rate: number;
    average_time_to_complete: number; // 秒
  };
}
```

---

## Lead 生成规则

```typescript
interface LeadGenerator {
  // 生成触发器
  triggers: LeadTrigger[];

  // 生成模板
  templates: LeadTemplate[];

  // 优先级规则
  priority_rules: PriorityRule[];

  // 场景特定规则
  scene_rules: Record<string, SceneLeadRules>;
}

interface LeadTrigger {
  type: 'event' | 'state' | 'time' | 'clue' | 'failure';
  condition: any;  // 触发条件

  // 触发后生成的 Leads
  generate: {
    template_id?: string;
    custom_lead?: Partial<LeadItem>;
    count?: number;
  };
}

interface LeadTemplate {
  template_id: string;
  name: string;
  category: LeadCategory;
  type: LeadType;

  // 模板变量
  variables: {
    name: string;
    default?: any;
    required: boolean;
  }[];

  // 生成逻辑
  generate: (context: GameContext) => Partial<LeadItem>;

  // 适用条件
  applicable_if: (context: GameContext) => boolean;
}

interface PriorityRule {
  rule_id: string;
  description: string;

  // 优先级计算
  calculate: (lead: LeadItem, context: GameContext) => number;

  // 条件
  applies_if: (lead: LeadItem, context: GameContext) => boolean;
}

interface SceneLeadRules {
  scene_id: string;

  // 场景基础 Leads
  base_leads: Partial<LeadItem>[];

  // 场景特定生成规则
  generation: {
    min_leads: number;
    max_leads: number;
    refresh_triggers: string[];
  };

  // 场景特定优先级
  priority_modifiers: {
    category?: Record<LeadCategory, number>;
    type?: Record<LeadType, number>;
  };
}
```

---

## 优先级系统

```typescript
// 优先级计算
function calculateLeadPriority(
  lead: LeadItem,
  context: GameContext
): number {
  let priority = 50; // 基础优先级

  // 1. 根据类别调整
  const categoryModifier = CATEGORY_PRIORITY[lead.category] || 0;
  priority += categoryModifier;

  // 2. 根据紧急度调整
  const urgencyModifier = URGENCY_PRIORITY[lead.urgency] || 0;
  priority += urgencyModifier;

  // 3. 根据时间调整 (新 Lead 优先)
  const ageMinutes = (Date.now() - lead.timestamp.getTime()) / 60000;
  if (ageMinutes < 5) {
    priority += 20; // 新 Lead 提升
  }

  // 4. 根据玩家状态调整
  if (context.player_status === 'confused') {
    // 提供明确指引的 Lead 优先
    if (lead.action.type === 'talk' || lead.action.type === 'investigate') {
      priority += 15;
    }
  }

  // 5. 根据线索关联调整
  if (lead.related.clues && lead.related.clues.length > 0) {
    // 与新发现线索相关的 Lead 提升
    const hasNewClue = lead.related.clues.some(clue_id =>
      context.new_clues.includes(clue_id)
    );
    if (hasNewClue) {
      priority += 10;
    }
  }

  // 限制在 0-100 范围内
  return Math.max(0, Math.min(100, priority));
}

const CATEGORY_PRIORITY: Record<LeadCategory, number> = {
  investigate: 10,
  action: 5,
  social: 0,
  prep: -5,
  explore: 5,
  combat: 20,    // 战斗优先
  escape: 25,    // 逃生最优先
};

const URGENCY_PRIORITY: Record<string, number> = {
  low: -10,
  medium: 0,
  high: 15,
  urgent: 30,
};
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `docs/specs/leads.md` | 创建 | Leads 系统规范 |
| `app/core/types/leads.ts` | 创建 | TypeScript 类型 |
| `app/services/leads.py` | 创建 | Leads 服务 |

---

## 验收标准

- [ ] LeadItem 结构完整
- [ ] LeadsState 定义清晰
- [ ] 优先级计算合理
- [ ] 生成规则可配置
- [ ] TypeScript 类型正确

---

## 参考文档

- M0: 规范冻结
- M6-002: Leads 生成算法

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
