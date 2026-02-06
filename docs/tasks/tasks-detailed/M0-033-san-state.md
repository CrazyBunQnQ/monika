# M0-033 定义 SAN/疯狂状态

## 概述
定义 CoC 理智值(SAN)和疯狂(madness)系统的状态结构,包括 SAN 检定、SAN 丢失、疯狂类型和持续时间。

## 验收标准
- [ ] 定义 SAN 值结构(当前值/最大值)
- [ ] 定义 SAN 检定结果(成功等级/丢失值)
- [ ] 定义临时疯狂状态
- [ ] 定义不定疯狂状态
- [ ] 定义疯狂症状表
- [ ] 定义恢复机制

## 技术方案

### SAN 状态结构

```typescript
interface SanityState {
  // SAN 值
  current: number; // 当前 SAN 值 (0-99)
  max: number; // 最大 SAN 值 (通常 99)

  // 检定历史
  check_history: SanityCheck[];

  // 疯狂状态
  madness: {
    active: boolean;
    current_madness?: MadnessState;
    past_madness: MadnessState[];
  };

  // 恐慌标志
  panic_flags: string[]; // 触发过的恐慌场景

  // 特殊规则
  special: {
    catatonic: boolean; // 木僵状态
    berserk: boolean; // 狂暴状态
    phobias: string[]; // 恐惧症
    manias: string[]; // 狂躁症
  };

  // 恢复追踪
  recovery: {
    last_check?: string; // 上次检定时间
    recovery_points: number; // 已恢复点数
    therapy_sessions: number; // 心理治疗次数
  };
}
```

### SAN 检定结构

```typescript
interface SanityCheck {
  id: string;
  timestamp: string;

  // 触发
  trigger: {
    type: 'encounter' | 'event' | 'violence' | 'knowledge' | 'spell';
    description: string;
  };

  // 难度
  difficulty: {
    base: number; // 基础 SAN 值
    modifier?: number; // 修正值
    final: number; // 最终检定值
  };

  // 检定结果
  check_result: {
    roll: number; // 掷骰结果
    success: boolean;
    level: 'failure' | 'regular' | 'hard' | 'extreme' | 'critical';
  };

  // SAN 丢失
  loss: {
    amount: number; // 实际丢失值
    max_possible: number; // 最大可能丢失
    rolled: boolean; // 是否掷骰确定
  };

  // 后果
  consequences: {
    madness_triggered?: MadnessType;
    panic?: boolean;
    catatonia?: boolean;
    berserk?: boolean;
  };

  // 叙述
  narrative?: string;
}
```

### 疯狂状态结构

```typescript
type MadnessType = 'temporary' | 'indefinite' | 'phobia' | 'mania';

interface MadnessState {
  id: string;
  type: MadnessType;

  // 触发
  triggered_by: string; // SAN 检定 ID
  trigger_event: string;

  // 开始时间
  start_time: string;

  // 持续时间
  duration?: {
    type: 'rounds' | 'hours' | 'days' | 'permanent';
    value: number;
    end_time?: string;
  };

  // 症状
  symptoms: MadnessSymptom[];

  // 效果
  effects: {
    incapacitated: boolean; // 是否丧失能力
    skill_penalties: Record<string, number>; // 技能惩罚
    behavior_modifiers: string[]; // 行为修正
  };

  // 恢复
  recovery: {
    method: 'time' | 'therapy' | 'medicine' | 'rest';
    progress: number; // 恢复进度 0-100
    recovered: boolean;
    end_time?: string;
  };
}
```

### 疯狂症状

```typescript
interface MadnessSymptom {
  type: string;
  description: string;

  // 行为表现
  behaviors: string[];

  // 游戏效果
  effects: {
    roleplay_required: boolean;
    skill_penalties?: Record<string, number>;
    restricted_actions?: string[];
    compelled_actions?: string[];
  };

  // 持续
  duration?: {
    rounds?: number;
    scenes?: number;
  };
}

// 疯狂症状表
const MADNESS_SYMPTOMS = {
  temporary: [
    {
      type: 'flee',
      description: '恐慌逃跑',
      behaviors: ['尽可能远离刺激源', '不顾危险'],
      effects: {
        roleplay_required: true,
        restricted_actions: ['attack', 'investigate'],
        compelled_actions: ['run_away']
      },
      duration: { rounds: 3 } // 1d3 回合
    },
    {
      type: 'faint',
      description: '昏厥',
      behaviors: ['失去意识', '无法行动'],
      effects: {
        roleplay_required: true,
        incapacitated: true
      },
      duration: { rounds: 6 } // 1d6 回合
    },
    {
      type: 'hysteria',
      description: '歇斯底里',
      behaviors: ['尖叫', '哭喊', '无法控制的情绪'],
      effects: {
        roleplay_required: true,
        skill_penalties: { 'all': -20 }
      },
      duration: { rounds: 10 } // 1d10 回合
    },
    {
      type: 'violence',
      description: '暴力冲动',
      behaviors: ['攻击最近的目标'],
      effects: {
        roleplay_required: true,
        compelled_actions: ['attack_nearest']
      },
      duration: { rounds: 3 }
    }
  ],

  indefinite: [
    {
      type: 'amnesia',
      description: '失忆',
      behaviors: ['忘记最近事件'],
      effects: {
        roleplay_required: true,
        skill_penalties: { 'psychology': -20 }
      }
    },
    {
      type: 'hallucination',
      description: '幻觉',
      behaviors: ['看到不存在的事物', '听到声音'],
      effects: {
        roleplay_required: true,
        skill_penalties: { 'spot_hidden': -30, 'listen': -30 }
      }
    },
    {
      type: 'paranoia',
      description: '偏执',
      behaviors: ['不信任他人', '怀疑被追踪'],
      effects: {
        roleplay_required: true,
        skill_penalties: { 'psychology': -10 }
      }
    }
  ],

  phobia: [
    {
      type: 'claustrophobia',
      description: '幽闭恐惧症',
      behaviors: ['害怕封闭空间', '恐慌发作'],
      effects: {
        roleplay_required: true,
        skill_penalties: { 'all': -20 }
      }
    },
    {
      type: 'arachnophobia',
      description: '蜘蛛恐惧症',
      behaviors: ['害怕蜘蛛', '无法靠近'],
      effects: {
        roleplay_required: true,
        skill_penalties: { 'all': -10 }
      }
    }
  ]
};
```

### SAN 检定逻辑

```typescript
interface SanityCheckOptions {
  difficulty: number; // 检定难度
  loss_on_fail: [number, number]; // 失败丢失值 [最小, 最大]
  loss_on_success?: [number, number]; // 成功丢失值
  modifier?: number; // 修正值
}

function performSanityCheck(
  state: SanityState,
  options: SanityCheckOptions,
  roll: number
): SanityCheckResult {
  // 计算修正后难度
  const finalDifficulty = options.difficulty + (options.modifier || 0);

  // 判断成功
  const success = roll <= finalDifficulty;
  const level = getSuccessLevel(roll, finalDifficulty);

  // 计算丢失
  let loss = 0;
  if (success) {
    if (options.loss_on_success) {
      const [min, max] = options.loss_on_success;
      loss = rollDice(max - min + 1, 1) + min;
    }
  } else {
    const [min, max] = options.loss_on_fail;
    loss = rollDice(max - min + 1, 1) + min;
  }

  // 限制丢失值
  loss = Math.min(loss, state.current);

  // 更新 SAN 值
  const newSan = state.current - loss;

  // 检查疯狂触发
  let madnessTriggered: MadnessType | undefined;
  if (loss > 5) {
    madnessTriggered = 'temporary';
  }
  if (newSan <= 0) {
    madnessTriggered = 'indefinite';
  }

  // 检查恐慌
  let panic = false;
  if (loss >= state.current * 0.2) {
    panic = true;
  }

  return {
    roll,
    success,
    level,
    loss,
    newSan,
    madnessTriggered,
    panic,
    narrative: generateSanityNarrative(success, loss, level)
  };
}

function getSuccessLevel(roll: number, difficulty: number): SanityCheckResult['level'] {
  if (roll === 1) return 'critical';
  if (roll <= difficulty / 5) return 'extreme';
  if (roll <= difficulty / 2) return 'hard';
  if (roll <= difficulty) return 'regular';
  return 'failure';
}
```

### 恢复机制

```typescript
interface SanityRecovery {
  // 时间恢复
  time_recovery: {
    rate: number; // 每小时恢复点数
    max_per_day: number; // 每日最大恢复
    requires_rest: boolean; // 是否需要休息
  };

  // 治疗恢复
  therapy_recovery: {
    per_session: [number, number]; // 每次治疗恢复 [1d6]
    success_required: boolean; // 是否需要成功检定
    skill: 'psychology' | 'psychoanalysis';
    difficulty: number;
  };

  // 药物恢复
  medicine_recovery: {
    per_dose: number;
    max_daily: number;
    side_effects?: string[];
  };
}

function recoverSanity(
  state: SanityState,
  method: 'time' | 'therapy' | 'medicine',
  points: number
): number {
  const recovery = Math.min(points, state.max - state.current);

  state.current += recovery;
  state.recovery.recovery_points += recovery;

  return recovery;
}
```

### 特殊状态

```typescript
interface SpecialSanityState {
  // 木僵
  catatonia: {
    active: boolean;
    duration: number; // 回合
    can_be_roused: boolean; // 可否唤醒
    wake_check: {
      skill: 'CON';
      difficulty: number;
      interval: number; // 每几回合检定一次
    };
  };

  // 狂暴
  berserk: {
    active: boolean;
    duration: number; // 回合
    target: 'nearest' | 'specific';
    attack_bonus: number; // 攻击加值
    defense_penalty: number; // 防御减值
  };

  // 恐惧症
  phobia: {
    trigger: string; // 触发物
    reaction: 'fear' | 'panic' | 'freeze';
    san_check: number; // 面对触发物的 SAN 检定
  };

  // 狂躁症
  mania: {
    compulsion: string; // 强迫行为
    frequency: 'always' | 'situational';
    resist_check: {
      skill: 'POW';
      difficulty: number;
    };
  };
}
```

## 依赖关系
- 前置任务: M0-027 定义 SessionState 结构
- 被依赖: M1-093 实现 ChaseTracker 组件

## 预估工时
2h
