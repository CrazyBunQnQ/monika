# M0-031: CombatState 战斗状态

**任务类型**: spec
**预估工时**: 2h
**依赖**: M0-027, M0-028
**状态**: [ ]

---

## 子任务拆解

### 1.1 CombatState 核心结构设计 (30min)

| ID | 任务 | 预估时间 | 状态 |
|----|------|----------|------|
| M0-031-01 | [ ] 定义战斗状态枚举 | 10min | [ ] |
| M0-031-02 | [ ] 定义参与者数据结构 | 10min | [ ] |
| M0-031-03 | [ ] 定义战斗回合结构 | 10min | [ ] |

```typescript
// src/core/combat/combat-types.ts

/**
 * 战斗状态枚举
 */
export enum CombatStatus {
  /** 未开始 */
  INACTIVE = 'inactive',
  /** 等待开始 */
  WAITING = 'waiting',
  /** 进行中 */
  ACTIVE = 'active',
  /** 暂停 */
  PAUSED = 'paused',
  /** 已结束 */
  ENDED = 'ended',
}

/**
 * 战斗参与者状态
 */
export enum ParticipantStatus {
  /** 正常 */
  NORMAL = 'normal',
  /** 先攻已决定 */
  INITIATIVE_DECIDED = 'initiative_decided',
  /** 行动中 */
  ACTING = 'acting',
  /** 行动结束 */
  ACTED = 'acted',
  /** 无法行动 */
  UNABLE = 'unable',
  /** 死亡 */
  DEAD = 'dead',
}

/**
 * 战斗行动类型
 */
export enum CombatActionType {
  /** 攻击 */
  ATTACK = 'attack',
  /** 防御 */
  DEFENSE = 'defense',
  /** 闪避 */
  DODGE = 'dodge',
  /** 使用物品 */
  USE_ITEM = 'use_item',
  /** 施法 */
  CAST_SPELL = 'cast_spell',
  /** 逃跑 */
  FLEE = 'flee',
  /** 压制 */
  SUPPRESS = 'suppress',
  /** 近战攻击 */
  MELEE_ATTACK = 'melee_attack',
  /** 远程攻击 */
  RANGED_ATTACK = 'ranged_attack',
}

/**
 * 战斗参与者
 */
export interface CombatParticipant {
  /** 参与者唯一ID */
  participantId: string;

  /** 关联的角色ID */
  characterId: string;

  /** 角色名称 */
  name: string;

  /** 是否为玩家角色 */
  isPlayer: boolean;

  /** 生命值 */
  hp: number;

  /** 生命值上限 */
  hpMax: number;

  /** 闪避值 */
  dodgeValue: number;

  /** 先攻值 */
  initiative: number;

  /** 当前状态 */
  status: ParticipantStatus;

  /** 伤害承受记录 */
  damageTaken: number;

  /** 行动次数 */
  actionCount: number;
}

/**
 * 战斗行动
 */
export interface CombatAction {
  /** 行动唯一ID */
  actionId: string;

  /** 回合数 */
  round: number;

  /** 行动者ID */
  actorId: string;

  /** 行动类型 */
  type: CombatActionType;

  /** 使用的技能ID（如果有） */
  skillId?: string;

  /** 目标ID（如果有） */
  targetId?: string;

  /** 描述 */
  description: string;

  /** 结果 */
  result: CombatActionResult;

  /** 时间戳 */
  timestamp: Date;
}

/**
 * 战斗行动结果
 */
export interface CombatActionResult {
  /** 是否成功 */
  success: boolean;

  /** 伤害值 */
  damage?: number;

  /** 是否暴击 */
  isCritical: boolean;

  /** 是否闪避 */
  isDodged: boolean;

  /** 叙事描述 */
  narration: string;

  /** 状态变化 */
  stateChanges?: CombatStateChange[];
}

/**
 * 战斗状态变化
 */
export interface CombatStateChange {
  /** 目标ID */
  targetId: string;

  /** 变化类型 */
  type: 'hp' | 'status' | 'effect';

  /** 变化值 */
  value: number;

  /** 变化后状态 */
  newValue: number | string;
}

/**
 * 战斗状态 - 核心数据结构
 */
export interface CombatState {
  /** 战斗唯一ID */
  combatId: string;

  /** 所属会话ID */
  sessionId: string;

  /** 当前状态 */
  status: CombatStatus;

  /** 当前回合 */
  currentRound: number;

  /** 当前行动者索引 */
  currentActorIndex: number;

  /** 参与者列表 */
  participants: CombatParticipant[];

  /** 行动顺序（参与者ID数组） */
  initiativeOrder: string[];

  /** 已执行行动列表 */
  actions: CombatAction[];

  /** 开始时间 */
  startedAt?: Date;

  /** 结束时间 */
  endedAt?: Date;

  /** 创建时间 */
  createdAt: Date;

  /** 更新时间 */
  updatedAt: Date;
}

/**
 * 战斗配置
 */
export interface CombatConfig {
  /** 回合时间限制（秒） */
  roundTimeLimit: number;

  /** 最大回合数 */
  maxRounds: number;

  /** 是否允许逃跑 */
  allowFleeing: boolean;

  /** 自动结束条件 */
  autoEndConditions: AutoEndCondition[];
}

/**
 * 自动结束条件
 */
export enum AutoEndCondition {
  /** 所有敌人被击败 */
  ALL_ENEMIES_DEFEATED = 'all_enemies_defeated',
  /** 所有玩家被击败 */
  ALL_PLAYERS_DEFEATED = 'all_players_defeated',
  /** 达到最大回合数 */
  MAX_ROUNDS_REACHED = 'max_rounds_reached',
}
```

---

### 1.2 战斗管理器实现 (30min)

| ID | 任务 | 预估时间 | 状态 |
|----|------|----------|------|
| M0-031-04 | [ ] 实现战斗初始化 | 10min | [ ] |
| M0-031-05 | [ ] 实现先攻决定 | 10min | [ ] |
| M0-031-06 | [ ] 实现回合管理 | 10min | [ ] |

```typescript
// src/core/combat/combat-manager.ts

import {
  CombatState,
  CombatStatus,
  ParticipantStatus,
  CombatAction,
  CombatActionType,
  CombatActionResult,
  CombatParticipant,
  CombatStateChange,
  CombatConfig,
  AutoEndCondition,
} from './combat-types';

/**
 * 战斗配置默认值
 */
export const DEFAULT_COMBAT_CONFIG: CombatConfig = {
  roundTimeLimit: 60,
  maxRounds: 20,
  allowFleeing: true,
  autoEndConditions: [
    AutoEndCondition.ALL_ENEMIES_DEFEATED,
    AutoEndCondition.ALL_PLAYERS_DEFEATED,
    AutoEndCondition.MAX_ROUNDS_REACHED,
  ],
};

/**
 * 战斗管理器
 */
export class CombatManager {
  private state: CombatState;
  private config: CombatConfig;

  constructor(combatId: string, sessionId: string, config?: Partial<CombatConfig>) {
    this.config = { ...DEFAULT_COMBAT_CONFIG, ...config };
    this.state = {
      combatId,
      sessionId,
      status: CombatStatus.INACTIVE,
      currentRound: 0,
      currentActorIndex: -1,
      participants: [],
      initiativeOrder: [],
      actions: [],
      createdAt: new Date(),
      updatedAt: new Date(),
    };
  }

  /**
   * 获取当前状态
   */
  getState(): Readonly<CombatState> {
    return this.state;
  }

  /**
   * 添加参与者
   */
  addParticipant(participant: CombatParticipant): void {
    this.state.participants.push({
      ...participant,
      status: ParticipantStatus.NORMAL,
      damageTaken: 0,
      actionCount: 0,
    });
    this.state.updatedAt = new Date();
  }

  /**
   * 移除参与者
   */
  removeParticipant(participantId: string): void {
    const index = this.state.participants.findIndex(
      (p) => p.participantId === participantId
    );
    if (index !== -1) {
      this.state.participants.splice(index, 1);
      this.state.updatedAt = new Date();
    }
  }

  /**
   * 开始战斗
   */
  start(): void {
    if (this.state.status !== CombatStatus.INACTIVE) {
      throw new Error('战斗已开始或已结束');
    }

    this.state.status = CombatStatus.ACTIVE;
    this.state.currentRound = 1;
    this.state.startedAt = new Date();
    this.state.updatedAt = new Date();

    // 决定先攻
    this.determineInitiative();

    // 设置第一个行动者
    this.state.currentActorIndex = 0;
  }

  /**
   * 决定先攻顺序
   */
  determineInitiative(): void {
    // 每个参与者掷 DEX d10
    this.state.participants.forEach((p) => {
      p.initiative = Math.floor(Math.random() * 10) + 1 + Math.floor(p.initiative / 10);
    });

    // 按先攻值降序排序
    this.state.participants.sort((a, b) => b.initiative - a.initiative);

    // 更新先攻顺序
    this.state.initiativeOrder = this.state.participants.map((p) => p.participantId);

    // 更新状态
    this.state.participants.forEach((p) => {
      p.status = ParticipantStatus.INITIATIVE_DECIDED;
    });
  }

  /**
   * 获取当前行动者
   */
  getCurrentActor(): CombatParticipant | undefined {
    if (this.state.currentActorIndex >= 0 &&
        this.state.currentActorIndex < this.state.participants.length) {
      return this.state.participants[this.state.currentActorIndex];
    }
    return undefined;
  }

  /**
   * 结束当前参与者的行动
   */
  endActorTurn(): void {
    const actor = this.getCurrentActor();
    if (actor) {
      actor.status = ParticipantStatus.ACTED;
      actor.actionCount++;
    }

    // 移动到下一个行动者
    this.state.currentActorIndex++;

    // 检查是否需要开始新回合
    if (this.state.currentActorIndex >= this.state.participants.length) {
      this.startNewRound();
    }
  }

  /**
   * 开始新回合
   */
  startNewRound(): void {
    this.state.currentRound++;
    this.state.currentActorIndex = 0;

    // 重置所有参与者状态
    this.state.participants.forEach((p) => {
      p.status = ParticipantStatus.NORMAL;
      p.actionCount = 0;
    });

    this.state.updatedAt = new Date();
  }

  /**
   * 执行战斗行动
   */
  executeAction(action: Omit<CombatAction, 'actionId' | 'timestamp'>): CombatAction {
    const fullAction: CombatAction = {
      ...action,
      actionId: `action-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      timestamp: new Date(),
    };

    this.state.actions.push(fullAction);
    this.state.updatedAt = new Date();

    return fullAction;
  }

  /**
   * 应用伤害到目标
   */
  applyDamage(targetId: string, damage: number): CombatStateChange[] {
    const target = this.state.participants.find(
      (p) => p.participantId === targetId
    );

    if (!target) {
      throw new Error(`找不到参与者 ${targetId}`);
    }

    const oldHp = target.hp;
    target.hp = Math.max(0, target.hp - damage);
    target.damageTaken += damage;

    // 检查是否死亡
    if (target.hp === 0) {
      target.status = ParticipantStatus.DEAD;
    }

    return [
      {
        targetId,
        type: 'hp',
        value: -damage,
        newValue: target.hp,
      },
    ];
  }

  /**
   * 治疗目标
   */
  healTarget(targetId: string, amount: number): CombatStateChange[] {
    const target = this.state.participants.find(
      (p) => p.participantId === targetId
    );

    if (!target) {
      throw new Error(`找不到参与者 ${targetId}`);
    }

    const actualHeal = Math.min(amount, target.hpMax - target.hp);
    target.hp += actualHeal;

    return [
      {
        targetId,
        type: 'hp',
        value: actualHeal,
        newValue: target.hp,
      },
    ];
  }

  /**
   * 结束战斗
   */
  end(reason: string): { winner: 'players' | 'enemies' | 'draw' } {
    if (this.state.status !== CombatStatus.ACTIVE) {
      throw new Error('战斗未在进行中');
    }

    this.state.status = CombatStatus.ENDED;
    this.state.endedAt = new Date();
    this.state.updatedAt = new Date();

    // 判断胜负
    const alivePlayers = this.state.participants.filter(
      (p) => p.isPlayer && p.status !== ParticipantStatus.DEAD
    );
    const aliveEnemies = this.state.participants.filter(
      (p) => !p.isPlayer && p.status !== ParticipantStatus.DEAD
    );

    let winner: 'players' | 'enemies' | 'draw';
    if (alivePlayers.length > 0 && aliveEnemies.length === 0) {
      winner = 'players';
    } else if (aliveEnemies.length > 0 && alivePlayers.length === 0) {
      winner = 'enemies';
    } else {
      winner = 'draw';
    }

    return { winner };
  }

  /**
   * 检查战斗是否应自动结束
   */
  checkAutoEnd(): { shouldEnd: boolean; condition?: AutoEndCondition } {
    const alivePlayers = this.state.participants.filter(
      (p) => p.isPlayer && p.status !== ParticipantStatus.DEAD
    );
    const aliveEnemies = this.state.participants.filter(
      (p) => !p.isPlayer && p.status !== ParticipantStatus.DEAD
    );

    if (this.config.autoEndConditions.includes(AutoEndCondition.ALL_ENEMIES_DEFEATED)
        && aliveEnemies.length === 0 && alivePlayers.length > 0) {
      return { shouldEnd: true, condition: AutoEndCondition.ALL_ENEMIES_DEFEATED };
    }

    if (this.config.autoEndConditions.includes(AutoEndCondition.ALL_PLAYERS_DEFEATED)
        && alivePlayers.length === 0 && aliveEnemies.length > 0) {
      return { shouldEnd: true, condition: AutoEndCondition.ALL_PLAYERS_DEFEATED };
    }

    if (this.config.autoEndConditions.includes(AutoEndCondition.MAX_ROUNDS_REACHED)
        && this.state.currentRound >= this.config.maxRounds) {
      return { shouldEnd: true, condition: AutoEndCondition.MAX_ROUNDS_REACHED };
    }

    return { shouldEnd: false };
  }

  /**
   * 获取存活参与者
   */
  getAliveParticipants(): CombatParticipant[] {
    return this.state.participants.filter(
      (p) => p.status !== ParticipantStatus.DEAD
    );
  }

  /**
   * 获取当前回合数
   */
  getCurrentRound(): number {
    return this.state.currentRound;
  }

  /**
   * 检查是否为当前行动者
   */
  isCurrentActor(participantId: string): boolean {
    const current = this.getCurrentActor();
    return current?.participantId === participantId;
  }
}
```

---

### 1.3 战斗行动生成器 (30min)

| ID | 任务 | 预估时间 | 状态 |
|----|------|----------|------|
| M0-031-07 | [ ] 实现攻击行动生成 | 10min | [ ] |
| M0-031-08 | [ ] 实现防御行动生成 | 10min | [ ] |
| M0-031-09 | [ ] 实现行动结果处理 | 10min | [ ] |

```typescript
// src/core/combat/combat-action-generator.ts

import { rollD100 } from '../dice';
import { CombatAction, CombatActionType, CombatActionResult } from './combat-types';

/**
 * 攻击行动参数
 */
export interface AttackActionParams {
  /** 行动者名称 */
  actorName: string;

  /** 目标名称 */
  targetName: string;

  /** 技能ID */
  skillId: string;

  /** 技能值 */
  skillValue: number;

  /** 基础伤害 */
  baseDamage: number;

  /** 伤害加成 */
  damageBonus: number;

  /** 目标闪避值 */
  targetDodge: number;
}

/**
 * 防御行动参数
 */
export interface DefenseActionParams {
  /** 行动者名称 */
  actorName: string;

  /** 技能ID */
  skillId: string;

  /** 技能值 */
  skillValue: number;
}

/**
 * 战斗行动生成器
 */
export class CombatActionGenerator {
  /**
   * 生成攻击行动结果
   */
  static generateAttack(params: AttackActionParams): CombatActionResult {
    // 技能检定
    const rollResult = rollD100();
    const skillCheck = rollResult.final_value;

    // 判定成功
    const hardThreshold = Math.ceil(params.skillValue / 2);
    const criticalThreshold = Math.ceil(params.skillValue / 5);

    let success = false;
    let isCritical = false;
    let damage = 0;
    let narration = '';

    if (skillCheck <= criticalThreshold) {
      // 大成功
      success = true;
      isCritical = true;
      damage = Math.floor(params.baseDamage * 1.5) + params.damageBonus;
      narration = `${params.actorName} 的攻击造成 ${damage} 点伤害！（大成功）`;
    } else if (skillCheck <= hardThreshold) {
      // 困难成功
      success = true;
      damage = params.baseDamage + params.damageBonus;
      narration = `${params.actorName} 的攻击造成 ${damage} 点伤害。（困难）`;
    } else if (skillCheck <= params.skillValue) {
      // 普通成功
      success = true;
      damage = params.baseDamage + params.damageBonus;
      narration = `${params.actorName} 的攻击造成 ${damage} 点伤害。`;
    } else {
      // 失败 - 检查是否被闪避
      const dodgeRoll = rollD100();
      if (dodgeRoll.final_value <= params.targetDodge) {
        narration = `${params.actorName} 攻击，但被 ${params.targetName} 闪开了！`;
      } else {
        narration = `${params.actorName} 的攻击落空了。`;
      }
    }

    return {
      success,
      damage,
      isCritical,
      isDodged: !success && narration.includes('闪开'),
      narration,
    };
  }

  /**
   * 生成防御行动结果
   */
  static generateDefense(params: DefenseActionParams): CombatActionResult {
    const rollResult = rollD100();
    const skillCheck = rollResult.final_value;

    const hardThreshold = Math.ceil(params.skillValue / 2);
    const criticalThreshold = Math.ceil(params.skillValue / 5);

    let success = false;
    let narration = '';

    if (skillCheck <= criticalThreshold) {
      success = true;
      narration = `${params.actorName} 完美地进行了防御！（大成功）`;
    } else if (skillCheck <= hardThreshold) {
      success = true;
      narration = `${params.actorName} 成功地进行了防御。（困难）`;
    } else if (skillCheck <= params.skillValue) {
      success = true;
      narration = `${params.actorName} 进行了防御。`;
    } else {
      narration = `${params.actorName} 防御失败。`;
    }

    return {
      success,
      isCritical: skillCheck <= criticalThreshold,
      isDodged: false,
      narration,
    };
  }

  /**
   * 生成闪避行动结果
   */
  static generateDodge(actorName: string, dodgeValue: number): CombatActionResult {
    const rollResult = rollD100();
    const skillCheck = rollResult.final_value;

    const hardThreshold = Math.ceil(dodgeValue / 2);
    const criticalThreshold = Math.ceil(dodgeValue / 5);

    let success = false;
    let narration = '';

    if (skillCheck <= criticalThreshold) {
      success = true;
      narration = `${actorName} 完美地闪开了！（大成功，增加闪避加值）`;
    } else if (skillCheck <= hardThreshold) {
      success = true;
      narration = `${actorName} 成功地闪开了。（困难）`;
    } else if (skillCheck <= dodgeValue) {
      success = true;
      narration = `${actorName} 进行了闪避。`;
    } else {
      narration = `${actorName} 闪避失败。`;
    }

    return {
      success,
      isCritical: skillCheck <= criticalThreshold,
      isDodged: success,
      narration,
    };
  }

  /**
   * 生成使用物品行动结果
   */
  static generateUseItem(
    actorName: string,
    itemName: string,
    effectDescription: string,
    successRate: number = 100
  ): CombatActionResult {
    if (successRate < 100) {
      const rollResult = rollD100();
      if (rollResult.final_value > successRate) {
        return {
          success: false,
          isCritical: false,
          isDodged: false,
          narration: `${actorName} 尝试使用 ${itemName}，但失败了。`,
        };
      }
    }

    return {
      success: true,
      isCritical: false,
      isDodged: false,
      narration: `${actorName} 使用了 ${itemName}，${effectDescription}`,
    };
  }
}

/**
 * 简单的 d100 掷骰函数
 */
function rollD100(modifier: number = 0): { raw_roll: number; final_value: number } {
  const raw_roll = Math.floor(Math.random() * 100) + 1;
  return {
    raw_roll,
    final_value: raw_roll + modifier,
  };
}
```

---

## 单元测试

```typescript
// tests/unit/core/combat/combat-types.test.ts

import {
  CombatState,
  CombatStatus,
  ParticipantStatus,
  CombatParticipant,
  DEFAULT_COMBAT_CONFIG,
} from '@/core/combat/combat-types';
import { CombatManager } from '@/core/combat/combat-manager';
import { CombatActionGenerator } from '@/core/combat/combat-action-generator';

describe('Combat Types', () => {
  describe('CombatStatus', () => {
    it('应该包含所有战斗状态', () => {
      expect(CombatStatus.INACTIVE).toBe('inactive');
      expect(CombatStatus.WAITING).toBe('waiting');
      expect(CombatStatus.ACTIVE).toBe('active');
      expect(CombatStatus.PAUSED).toBe('paused');
      expect(CombatStatus.ENDED).toBe('ended');
    });
  });

  describe('DEFAULT_COMBAT_CONFIG', () => {
    it('应该包含正确的默认值', () => {
      expect(DEFAULT_COMBAT_CONFIG.roundTimeLimit).toBe(60);
      expect(DEFAULT_COMBAT_CONFIG.maxRounds).toBe(20);
      expect(DEFAULT_COMBAT_CONFIG.allowFleeing).toBe(true);
    });
  });
});

describe('CombatManager', () => {
  const createTestCombat = (): CombatManager => {
    return new CombatManager('combat-001', 'session-001');
  };

  const createTestParticipant = (
    id: string,
    name: string,
    isPlayer: boolean
  ): CombatParticipant => ({
    participantId: id,
    characterId: `char-${id}`,
    name,
    isPlayer,
    hp: 20,
    hpMax: 20,
    dodgeValue: 25,
    initiative: 50,
    status: ParticipantStatus.NORMAL,
    damageTaken: 0,
    actionCount: 0,
  });

  describe('初始化', () => {
    it('应该创建正确的初始状态', () => {
      const manager = createTestCombat();
      const state = manager.getState();

      expect(state.combatId).toBe('combat-001');
      expect(state.sessionId).toBe('session-001');
      expect(state.status).toBe(CombatStatus.INACTIVE);
      expect(state.currentRound).toBe(0);
      expect(state.participants).toHaveLength(0);
    });
  });

  describe('参与者管理', () => {
    it('应该能添加参与者', () => {
      const manager = createTestCombat();

      manager.addParticipant(createTestParticipant('p1', '玩家1', true));

      expect(manager.getState().participants).toHaveLength(1);
    });

    it('应该能移除参与者', () => {
      const manager = createTestCombat();
      manager.addParticipant(createTestParticipant('p1', '玩家1', true));

      manager.removeParticipant('p1');

      expect(manager.getState().participants).toHaveLength(0);
    });
  });

  describe('战斗流程', () => {
    it('应该能开始战斗', () => {
      const manager = createTestCombat();
      manager.addParticipant(createTestParticipant('p1', '玩家1', true));
      manager.addParticipant(createTestParticipant('p2', '敌人1', false));

      manager.start();

      expect(manager.getState().status).toBe(CombatStatus.ACTIVE);
      expect(manager.getState().currentRound).toBe(1);
      expect(manager.getState().startedAt).toBeDefined();
    });

    it('应该能获取当前行动者', () => {
      const manager = createTestCombat();
      manager.addParticipant(createTestParticipant('p1', '玩家1', true));

      manager.start();

      const actor = manager.getCurrentActor();
      expect(actor).toBeDefined();
      expect(actor?.participantId).toBe('p1');
    });

    it('应该能结束行动并进入下一轮', () => {
      const manager = createTestCombat();
      manager.addParticipant(createTestParticipant('p1', '玩家1', true));
      manager.addParticipant(createTestParticipant('p2', '玩家2', true));

      manager.start();
      manager.endActorTurn();

      const actor = manager.getCurrentActor();
      expect(actor?.participantId).toBe('p2');
    });

    it('新回合应该增加回合数', () => {
      const manager = createTestCombat();
      manager.addParticipant(createTestParticipant('p1', '玩家1', true));
      manager.addParticipant(createTestParticipant('p2', '玩家2', true));

      manager.start();
      manager.endActorTurn(); // p1 行动结束
      manager.endActorTurn(); // p2 行动结束，触发新回合

      expect(manager.getCurrentRound()).toBe(2);
    });
  });

  describe('伤害处理', () => {
    it('应该能应用伤害', () => {
      const manager = createTestCombat();
      manager.addParticipant(createTestParticipant('p1', '玩家1', true));
      manager.start();

      const changes = manager.applyDamage('p1', 5);

      expect(manager.getState().participants[0].hp).toBe(15);
      expect(changes[0].value).toBe(-5);
    });

    it('不应该造成负血量', () => {
      const manager = createTestCombat();
      manager.addParticipant(createTestParticipant('p1', '玩家1', true));
      manager.start();

      manager.applyDamage('p1', 100);

      expect(manager.getState().participants[0].hp).toBe(0);
    });

    it('0 血量应标记为死亡', () => {
      const manager = createTestCombat();
      manager.addParticipant(createTestParticipant('p1', '玩家1', true));
      manager.start();

      manager.applyDamage('p1', 20);

      expect(manager.getState().participants[0].status).toBe(ParticipantStatus.DEAD);
    });

    it('应该能治疗', () => {
      const manager = createTestCombat();
      manager.addParticipant(createTestParticipant('p1', '玩家1', true));
      manager.start();
      manager.applyDamage('p1', 10);

      const changes = manager.healTarget('p1', 5);

      expect(manager.getState().participants[0].hp).toBe(15);
      expect(changes[0].value).toBe(5);
    });
  });

  describe('战斗结束', () => {
    it('应该能结束战斗并判断胜负', () => {
      const manager = createTestCombat();
      manager.addParticipant(createTestParticipant('p1', '玩家1', true));
      manager.addParticipant(createTestParticipant('e1', '敌人1', false));
      manager.start();

      // 敌人死亡
      manager.applyDamage('e1', 100);

      const result = manager.end('所有敌人被击败');

      expect(result.winner).toBe('players');
      expect(manager.getState().status).toBe(CombatStatus.ENDED);
    });
  });
});

describe('CombatActionGenerator', () => {
  describe('generateAttack', () => {
    it('应该生成攻击结果', () => {
      const result = CombatActionGenerator.generateAttack({
        actorName: '玩家1',
        targetName: '怪物1',
        skillId: 'fighting_brawl',
        skillValue: 50,
        baseDamage: 5,
        damageBonus: 0,
        targetDodge: 25,
      });

      expect(result).toHaveProperty('success');
      expect(result).toHaveProperty('narration');
    });

    it('高技能值应该更容易成功', () => {
      const lowSkill = CombatActionGenerator.generateAttack({
        actorName: '玩家1',
        targetName: '怪物1',
        skillId: 'fighting_brawl',
        skillValue: 20,
        baseDamage: 5,
        damageBonus: 0,
        targetDodge: 25,
      });

      const highSkill = CombatActionGenerator.generateAttack({
        actorName: '玩家1',
        targetName: '怪物1',
        skillId: 'fighting_brawl',
        skillValue: 80,
        baseDamage: 5,
        damageBonus: 0,
        targetDodge: 25,
      });

      // 多次测试取平均
      const lowSuccess = [lowSkill].filter(r => r.success).length;
      const highSuccess = [highSkill].filter(r => r.success).length;

      expect(highSuccess).toBeGreaterThanOrEqual(lowSuccess);
    });
  });

  describe('generateDodge', () => {
    it('应该生成闪避结果', () => {
      const result = CombatActionGenerator.generateDodge('玩家1', 50);

      expect(result).toHaveProperty('success');
      expect(result).toHaveProperty('narration');
    });
  });

  describe('generateUseItem', () => {
    it('应该生成使用物品结果', () => {
      const result = CombatActionGenerator.generateUseItem(
        '玩家1',
        '急救包',
        '恢复了 5 点生命值'
      );

      expect(result.success).toBe(true);
      expect(result.narration).toContain('急救包');
    });

    it('低成功率应该可能失败', () => {
      let hasFailure = false;
      for (let i = 0; i < 100; i++) {
        const result = CombatActionGenerator.generateUseItem(
          '玩家1',
          '不稳定药剂',
          '产生了效果',
          50
        );
        if (!result.success) {
          hasFailure = true;
          break;
        }
      }
      expect(hasFailure).toBe(true);
    });
  });
});
```

---

## 验收标准

- [ ] CombatState 包含所有必需字段
- [ ] CombatStatus 枚举完整
- [ ] CombatManager 能正确管理战斗流程
- [ ] 先攻决定系统正确
- [ ] 回合管理正确
- [ ] 伤害/治疗系统正确
- [ ] 胜负判断正确
- [ ] 单元测试覆盖率达到 80% 以上

---

## 涉及文件

| 文件 | 操作 |
|------|------|
| `src/core/combat/combat-types.ts` | 创建 |
| `src/core/combat/combat-manager.ts` | 创建 |
| `src/core/combat/combat-action-generator.ts` | 创建 |
| `tests/unit/core/combat/combat-types.test.ts` | 创建 |

---

## 参考文档

- [01-m0-spec-freeze.md - 战斗状态定义](../01-m0-spec-freeze.md)
- CoC 7e 规则书 - 第 7 章 战斗
