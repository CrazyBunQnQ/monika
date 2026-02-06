# M0-032 定义 ChaseState 追逐状态

## 概述
定义 CoC 追逐系统(chase)的状态结构,包括距离等级、行动点数、障碍物、参与者和追逐结果。

## 验收标准
- [ ] 定义追逐基础信息(ID/类型/状态)
- [ ] 定义距离等级系统(0-5 级)
- [ ] 定义参与者列表(位置/速度)
- [ ] 定义障碍物系统
- [ ] 定义追逐回合信息
- [ ] 定义压力值和疲劳系统

## 技术方案

### 追逐状态结构

```typescript
interface ChaseState {
  // 追逐标识
  chase_id: string;
  session_id: string;

  // 追逐类型
  type: 'foot' | 'vehicle' | 'mixed';
  terrain: 'urban' | 'wilderness' | 'indoor' | 'road' | 'water';

  // 状态
  status: 'active' | 'paused' | 'completed' | 'failed';
  round: number;
  current_turn: string; // 当前行动的参与者 ID

  // 参与者
  participants: ChaseParticipant[];

  // 距离
  distance_level: 0 | 1 | 2 | 3 | 4 | 5;
  distance_modifier: number; // 距离修正值

  // 障碍
  obstacles: ChaseObstacle[];
  current_obstacle?: ChaseObstacle;

  // 压力
  stress_levels: Record<string, number>; // participant_id -> stress

  // 结果
  result?: {
    winner: 'prey' | 'predator';
    reason: string;
    timestamp: string;
  };

  // 元数据
  metadata: {
    started_at: string;
    updated_at: string;
    auto_end: boolean; // 达到极距自动结束
  };
}
```

### 参与者结构

```typescript
interface ChaseParticipant {
  // 标识
  id: string;
  name: string;
  type: 'pc' | 'npc' | 'vehicle';

  // 角色
  role: 'prey' | 'predator'; // 猎物或猎手

  // 位置
  position: {
    distance_level: 0 | 1 | 2 | 3 | 4 | 5;
    lane: number; // 车道(1-4)
  };

  // 行动
  action_points: number;
  actions_used: number;

  // 状态
  status: 'active' | 'stalled' | 'crashed' | 'captured' | 'escaped';

  // 能力
  movement_rate: number; // 移动速率
  current_speed: number; // 当前速度
  max_speed: number;

  // 修正
  modifiers: {
    terrain?: number;
    obstacle?: number;
    stress?: number;
  };

  // 载具(如果是车辆追逐)
  vehicle?: {
    id: string;
    name: string;
    hp: number;
    hp_max: number;
    handling: number; // 操控性
    acceleration: number; // 加速度
  };
}
```

### 障碍物结构

```typescript
interface ChaseObstacle {
  id: string;

  // 位置
  position: {
    distance_level: 0 | 1 | 2 | 3 | 4 | 5;
    lanes: number[]; // 影响哪些车道
  };

  // 类型
  type: 'static' | 'moving' | 'conditional';

  // 难度
  difficulty: {
    drive_check: number; // 驾驶检定难度
    dodge_check: number; // 闪避检定难度
  };

  // 效果
  effects: {
    on_fail: 'stall' | 'crash' | 'lose_ground' | 'damage';
    on_success: 'maintain' | 'gain_ground' | 'overtake';
    damage?: number;
    speed_penalty?: number;
  };

  // 描述
  description: {
    narrative: string; // 叙述文本
    visual?: string; // 视觉描述
  };

  // 条件(条件障碍)
  condition?: {
    trigger: string; // 触发条件
    chance: number; // 出现概率 0-100
  };
}
```

### 压力系统

```typescript
interface StressSystem {
  // 压力阈值
  thresholds: {
    warning: number; // 警告值
    critical: number; // 危险值
  };

  // 压力效果
  effects: {
    // 警告级: -10 技能
    warning_penalty: number;

    // 危险级: -20 技能,1/2 速度
    critical_penalty: number;
    speed_halved: boolean;

    // 极限: 恐慌,强制检定
    panic_check: boolean;
  };

  // 压力来源
  sources: {
    distance_change: number; // 距离变化
    obstacle_fail: number; // 障碍失败
    collision: number; // 碰撞
    pursuit: number; // 被追逐
  };
}

// 压力计算
function calculateStress(
  participant: ChaseParticipant,
  event: ChaseEvent
): number {
  let stress = 0;

  switch (event.type) {
    case 'distance_increase':
      stress += participant.role === 'prey' ? -5 : 5;
      break;
    case 'distance_decrease':
      stress += participant.role === 'prey' ? 5 : -5;
      break;
    case 'obstacle_failed':
      stress += 10;
      break;
    case 'collision':
      stress += 15;
      break;
  }

  return Math.max(0, Math.min(100, stress));
}
```

### 追逐行动

```typescript
type ChaseAction =
  | 'accelerate' // 加速
  | 'decelerate' // 减速
  | 'maintain' // 保持
  | 'attack' // 攻击
  | 'ram' // 冲撞
  | 'evasive' // 规避
  | 'shortcut' // 捷径
  | 'obstacle'; // 障碍

interface ChaseActionData {
  type: ChaseAction;
  actor_id: string;
  target_id?: string;

  // 检定
  check?: {
    skill: string;
    difficulty: number;
    roll: number;
    success_level: 'failure' | 'regular' | 'hard' | 'extreme';
  };

  // 结果
  result?: {
    distance_change?: -1 | 0 | 1;
    speed_change?: number;
    stress_change?: number;
    damage?: number;
    status_change?: ChaseParticipant['status'];
  };
}
```

### 追逐回合

```typescript
interface ChaseRound {
  round_number: number;
  actions: ChaseActionData[];
  distance_changes: {
    prev_level: number;
    new_level: number;
  }[];
  obstacles_triggered: string[];
  participants_exited: string[];
}
```

### 追逐结果

```typescript
interface ChaseResult {
  // 结束条件
  end_condition:
    | 'maximum_distance' // 达到极距
    | 'minimum_distance' // 距离归零
    | 'prey_captured' // 猎物被捕获
    | 'prey_escaped' // 猎物逃脱
    | 'predator_gave_up' // 猎手放弃
    | 'crash' // 撞毁
    | 'time_limit'; // 时间限制

  // 胜者
  winner: 'prey' | 'predator';

  // 叙述
  narrative: {
    ending: string;
    consequences: string[];
  };

  // 数据
  data: {
    rounds: number;
    total_time: number; // 分钟
    distance_covered: number; // 英里/公里
    casualties: string[];
    damage_report: Record<string, number>;
  };
}
```

### 追逐配置

```typescript
interface ChaseConfig {
  // 自动结束
  auto_end_on_max_distance: boolean; // 距离达到 5 自动结束
  auto_end_on_min_distance: boolean; // 距离达到 0 自动结束

  // 压力配置
  stress_enabled: boolean;
  stress_increment: number; // 每回合压力增长

  // 障碍配置
  obstacle_frequency: 'low' | 'medium' | 'high';
  obstacle_difficulty_curve: 'flat' | 'increasing' | 'random';

  // 回合限制
  max_rounds?: number;
  time_limit?: number; // 分钟

  // 速度限制
  max_speed?: number;
  min_speed?: number;
}
```

## 依赖关系
- 前置任务: M0-027 定义 SessionState 结构
- 被依赖: M1-085 实现 ChaseState 数据结构

## 预估工时
2h
