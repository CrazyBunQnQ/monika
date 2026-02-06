# M5-001: 设计 SAN 检定数据结构

**任务ID**: M5-001
**标题**: 设计 SAN 检定数据结构
**类型**: spec (规范设计)
**预估工时**: 2h
**依赖**: M0

---

## 任务描述

设计 SAN (理智值) 检定系统的数据结构，定义 SAN 损失、检定结果、疯狂触发等相关数据模型。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M5-001-01 | 分析 CoC 7e SAN 规则 | 理解 SAN 检定机制 | 20min |
| M5-001-02 | 设计 SANCheck 结构 | 检定请求/响应 | 25min |
| M5-001-03 | 设计 SANLoss 结构 | SAN 损失计算 | 20min |
| M5-001-04 | 设计 SANThreshold 结构 | 场景 SAN 阈值 | 15min |
| M5-001-05 | 编写 TypeScript 类型 | 类型定义 | 20min |
| M5-001-06 | 编写检定流程图 | SAN 检定流程 | 15min |
| M5-001-07 | 编写示例数据 | 典型场景示例 | 10min |

---

## SANCheck 结构

```typescript
interface SANCheck {
  check_id: string;
  session_id: string;
  timestamp: datetime;

  // 触发信息
  trigger: {
    type: 'scene' | 'event' | 'encounter' | 'spell' | 'creature';
    source_id: string;          // 触发源 ID
    description: string;        // 触发描述
    kp_note?: string;           // KP 备注
  };

  // 检定参数
  check: {
    character_id: string;
    current_san: number;
    san_cap: number;            // 当前 SAN 上限 (99-C)
    difficulty: 'regular' | 'hard' | 'extreme';
  };

  // 检定结果
  result: {
    roll: number;               // d100 结果
    success_level: SuccessLevel;
    passed: boolean;
    san_loss: SANLoss;
    madness_triggered?: MadnessTrigger;
  };

  // 最终 SAN 值
  final_san: number;
}

type SuccessLevel =
  | 'critical'   // 大成功 (1)
  | 'extreme'    // 极难成功
  | 'hard'       // 困难成功
  | 'regular'    // 普通成功
  | 'failure'    // 失败
  | 'fumble';    // 大失败 (100)
```

---

## SANLoss 结构

```typescript
interface SANLoss {
  // 损失定义
  loss_definition: {
    success: {
      min: number;              // 成功最小损失
      max: number;              // 成功最大损失
      average?: number;         // 平均值 (通常用于快速计算)
    };
    failure: {
      min: number;
      max: number;
      average?: number;
    };
    special?: {
      critical?: number;        // 大成功损失 (通常为 0)
      fumble?: number;          // 大失败损失 (通常为 max)
    };
  };

  // 实际损失
  actual_loss: {
    success: number;            // 成功时的实际损失
    failure: number;            // 失败时的实际损失
  };

  // 损失原因
  reason: string;
  can_reduce?: boolean;         // 是否可减少 (如通过 RP)
}

// 预定义 SAN 阈值
interface SANThreshold {
  id: string;
  category: SANCategory;
  description: string;

  // 标准损失
  loss: {
    success: string;            // 如 "0/1d4"
    failure: string;            // 如 "1/1d6"
  };

  // 特殊处理
  special?: {
    once_only?: boolean;        // 只检定一次
    per_session?: boolean;      // 每次会话只检定一次
    cumulative?: boolean;       // 累积效果
  };
}

type SANCategory =
  | 'bodily_horror'       // 肉体恐怖
  | 'discovery'           // 发现真相
  | 'violence'            // 暴力场面
  | 'unnatural'           // 超自然现象
  | 'knowledge'           // 禁忌知识
  | 'helplessness'        // 无助感
  | 'personal'            // 个人创伤
  | 'creature_specific';  // 特定生物
```

---

## MadnessTrigger 结构

```typescript
interface MadnessTrigger {
  trigger_id: string;
  character_id: string;
  timestamp: datetime;

  // 触发条件
  condition: {
    current_san: number;
    threshold: number;          // 触发阈值
    type: 'temporary' | 'indefinite';
  };

  // 疯狂结果
  madness: {
    type: MadnessType;
    symptoms: MadnessSymptom[];
    duration?: number;          // 持续时间 (分钟/小时)
    real_life?: boolean;        // 是否是 Real Life 疯狂
  };

  // 恢复条件
  recovery: {
    conditions: string[];
    time_to_recover?: number;   // 恢复所需时间
  };
}

type MadnessType =
  // 临时疯狂 (1d10 分钟)
  | 'faint'              // 昏厥
  | 'panic'              // 恐慌逃跑
  | 'flee'               // 奔跑
  | 'stunned'            // 惊呆
  | 'raving'             // 谵妄

  // 不定疯狂 (1d10 小时)
  | 'amnesia'            // 记忆丧失
  | 'delusion'           // 妄想
  | 'hallucination'      // 幻觉
  | 'paranoia'           // 偏执
  | 'phobia'             // 恐惧症
  | 'mania'              // 躁狂
  | 'schizophrenia'      // 精神分裂;

interface MadnessSymptom {
  id: string;
  name: string;
  description: string;
  effects: {
    modifier?: number;   // 行为修正
    prohibited_actions?: string[];
    required_actions?: string[];
  };
}
```

---

## 预定义 SAN 阈值

```typescript
// 常见场景的 SAN 阈值
const SAN_THRESHOLDS: Record<string, SANThreshold> = {
  // 发现尸体
  'corpse_fresh': {
    id: 'corpse_fresh',
    category: 'bodily_horror',
    description: '发现新鲜的尸体',
    loss: {
      success: '0',
      failure: '1/1d4',
    },
  },

  'corpse_mutilated': {
    id: 'corpse_mutilated',
    category: 'bodily_horror',
    description: '发现被肢解的尸体',
    loss: {
      success: '0/1',
      failure: '1/1d6',
    },
  },

  'corpse_loved_one': {
    id: 'corpse_loved_one',
    category: 'personal',
    description: '发现亲友的尸体',
    loss: {
      success: '1/1d4',
      failure: '1/1d8',
    },
    special: {
      once_only: true,
    },
  },

  // 战斗
  'combat_violence': {
    id: 'combat_violence',
    category: 'violence',
    description: '极端暴力场面',
    loss: {
      success: '0',
      failure: '1/1d4',
    },
  },

  // 超自然
  'unnatural_creature': {
    id: 'unnatural_creature',
    category: 'unnatural',
    description: '首次遭遇神话生物',
    loss: {
      success: '0/1',
      failure: '1d6/1d20',
    },
    special: {
      once_only: true,
    },
  },

  // 禁忌知识
  'forbidden_knowledge': {
    id: 'forbidden_knowledge',
    category: 'knowledge',
    description: '阅读禁忌文本',
    loss: {
      success: '1d3',
      failure: '1d6',
    },
  },

  // 恐惧症触发
  'phobia_trigger': {
    id: 'phobia_trigger',
    category: 'personal',
    description: '遭遇恐惧源',
    loss: {
      success: '0/1',
      failure: '1d3/1d6',
    },
  },
};
```

---

## SAN 检定流程

```
                      ┌─────────────┐
                      │ 遭遇恐怖场景 │
                      └──────┬──────┘
                             │
                             ▼
                      ┌─────────────┐
                      │ 确定 SAN 阈值│
                      │ (查表/定义)  │
                      └──────┬──────┘
                             │
                             ▼
                      ┌─────────────┐
                      │ 进行 SAN 检定│
                      │ d100 <= SAN │
                      └──────┬──────┘
                             │
                   ┌─────────┴─────────┐
                   │                   │
               成功 ▼               失败 ▼
            ┌───────────┐       ┌───────────┐
            │ 扣除 0-N  │       │ 扣除 1-N  │
            │ 点 SAN    │       │ 点 SAN    │
            └─────┬─────┘       └─────┬─────┘
                  │                   │
                  ▼                   ▼
           ┌─────────────┐     ┌─────────────┐
           │ SAN >= 0?   │     │ SAN >= 0?   │
           └─────┬───────┘     └─────┬───────┘
                 │                   │
          是 ────┴──────      否 ────┴──────
          │                   │
          ▼                   ▼
   ┌─────────────┐     ┌─────────────┐
   │ 检查疯狂触发 │     │ 陷入不定疯狂 │
   │ (如适用)     │     │             │
   └─────────────┘     └─────────────┘
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `docs/specs/san-system.md` | 创建 | SAN 系统规范 |
| `app/core/types/san.ts` | 创建 | TypeScript 类型 |
| `app/db/models/san.py` | 创建 | 数据模型 |

---

## 验收标准

- [ ] SANCheck 结构完整
- [ ] SANLoss 定义清晰
- [ ] MadnessTrigger 正确
- [ ] 预定义阈值完整
- [ ] 检定流程图准确

---

## 参考文档

- CoC 7e 规则书 - SAN 章节
- M0-028: 角色状态定义

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
