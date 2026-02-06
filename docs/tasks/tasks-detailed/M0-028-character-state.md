# M0-028: CharacterState 角色状态

**任务类型**: spec
**预估工时**: 2h
**依赖**: M0-027
**状态**: [ ]

---

## 子任务拆解

### 1.1 CharacterState 核心结构设计 (30min)

| ID | 任务 | 预估时间 | 状态 |
|----|------|----------|------|
| M0-028-01 | [ ] 定义角色标识字段 | 10min | [ ] |
| M0-028-02 | [ ] 定义属性结构 | 10min | [ ] |
| M0-028-03 | [ ] 定义派生数值结构 | 10min | [ ] |

```typescript
// src/core/states/character-state.ts

/**
 * 角色状态枚举
 */
export enum CharacterStatus {
  /** 存活 */
  ALIVE = 'alive',
  /** 失去意识 */
  UNCONSCIOUS = 'unconscious',
  /** 濒死 */
  DYING = 'dying',
  /** 死亡 */
  DEAD = 'dead',
  /** 疯狂 */
  INSANE = 'insane',
}

/**
 * CoC 7e 属性名称枚举
 */
export enum AttributeName {
  /** 力量 */
  STR = 'STR',
  /** 体质 */
  CON = 'CON',
  /** 敏捷 */
  DEX = 'DEX',
  /** 外貌 */
  APP = 'APP',
  /** 意志 */
  POW = 'POW',
  /** 智力 */
  INT = 'POW',
  /** 体型 */
  SIZ = 'SIZ',
  /** 教育 */
  EDU = 'EDU',
}

/**
 * 角色属性结构
 *
 * 包含 CoC 7e 的 8 个基础属性
 */
export interface Attributes {
  /** 力量 - 力量值，影响近战伤害等 */
  STR: number;
  /** 体质 - 生命值上限，抗打击能力 */
  CON: number;
  /** 敏捷 - 行动顺序，闪避等 */
  DEX: number;
  /** 外貌 - 社交能力，第一印象 */
  APP: number;
  /** 意志 - 意志力，SAN 值相关 */
  POW: number;
  /** 智力 - 知识，灵感检定 */
  INT: number;
  /** 体型 - 体型，与 SIZ 合并计算体格 */
  SIZ: number;
  /** 教育 - 知识，教育相关检定 */
  EDU: number;
}

/**
 * 派生数值结构
 *
 * 由基础属性计算得出的游戏数值
 */
export interface DerivedValues {
  /** 生命值 */
  HP: number;
  /** 生命值上限 */
  HP_max: number;
  /** 魔法值 */
  MP: number;
  /** 魔法值上限 */
  MP_max: number;
  /** 理智值 */
  SAN: number;
  /** 理智值上限 */
  SAN_max: number;
  /** 幸运值 */
  Luck: number;
  /** 幸运值上限 */
  Luck_max: number;
  /** 移动力 */
  Move: number;
  /** 体格 (DB) */
  DamageBonus: number;
  /** 闪避值 */
  Dodge: number;
}

/**
 * 角色状态
 *
 * 完整的角色数据模型，包含所有游戏所需的状态信息
 */
export interface CharacterState {
  /** 角色唯一ID */
  characterId: string;

  /** 角色名称 */
  name: string;

  /** 玩家名称（真实世界玩家） */
  playerName?: string;

  /** 角色职业 */
  occupation?: string;

  /** 性别 */
  gender?: string;

  /** 年龄 */
  age?: number;

  /** 头像URL */
  avatarUrl?: string;

  /** 基础属性 */
  attributes: Attributes;

  /** 派生数值 */
  derived: DerivedValues;

  /** 当前状态 */
  status: CharacterStatus;

  /** 背包物品列表 */
  inventory: string[];

  /** 角色笔记 */
  notes: string;

  /** 创建时间 */
  createdAt: Date;

  /** 更新时间 */
  updatedAt: Date;
}

/**
 * 默认空属性
 */
export const DEFAULT_ATTRIBUTES: Attributes = {
  STR: 0,
  CON: 0,
  DEX: 0,
  APP: 0,
  POW: 0,
  INT: 0,
  SIZ: 0,
  EDU: 0,
};

/**
 * 默认派生数值
 */
export const DEFAULT_DERIVED: DerivedValues = {
  HP: 0,
  HP_max: 0,
  MP: 0,
  MP_max: 0,
  SAN: 0,
  SAN_max: 0,
  Luck: 0,
  Luck_max: 0,
  Move: 0,
  DamageBonus: 0,
  Dodge: 0,
};
```

---

### 1.2 属性计算方法实现 (30min)

| ID | 任务 | 预估时间 | 状态 |
|----|------|----------|------|
| M0-028-04 | [ ] 实现 HP 计算 | 10min | [ ] |
| M0-028-05 | [ ] 实现 MP 计算 | 10min | [ ] |
| M0-028-06 | [ ] 实现 Move 和闪避计算 | 10min | [ ] |

```typescript
// src/core/states/character-calculator.ts

import {
  Attributes,
  DerivedValues,
  DEFAULT_DERIVED,
} from './character-state';

/**
 * CoC 7e 属性值范围
 */
export const ATTRIBUTE_RANGES = {
  /** 普通人类属性范围 */
  human: { min: 0, max: 99 },
  /** 超人类属性范围 */
  supernatural: { min: 100, max: 999 },
} as const;

/**
 * 属性值类型
 */
export type AttributeValue = typeof ATTRIBUTE_RANGES.human.min |
                             typeof ATTRIBUTE_RANGES.human.max |
                             typeof ATTRIBUTE_RANGES.supernatural.max;

/**
 * CoC 7e 角色数值计算器
 */
export class CharacterCalculator {
  private attributes: Attributes;

  constructor(attributes: Attributes) {
    this.attributes = attributes;
  }

  /**
   * 计算生命值 (HP)
   * HP = (CON + SIZ) / 10，向上取整
   */
  calculateHP(): number {
    return Math.ceil((this.attributes.CON + this.attributes.SIZ) / 10);
  }

  /**
   * 计算魔法值 (MP)
   * MP = POW / 10，向上取整
   */
  calculateMP(): number {
    return Math.ceil(this.attributes.POW / 10);
  }

  /**
   * 计算理智值 (SAN)
   * SAN = POW * 5
   */
  calculateSAN(): number {
    return this.attributes.POW * 5;
  }

  /**
   * 计算幸运值 (Luck)
   * Luck = POW * 5
   */
  calculateLuck(): number {
    return this.attributes.POW * 5;
  }

  /**
   * 计算移动力 (Move)
   * Move = min(DEX, STR) + SIZ 等级对应的移动力
   */
  calculateMove(): number {
    const age = 20; // 默认 20 岁，可根据年龄调整
    const minDexStr = Math.min(this.attributes.DEX, this.attributes.STR);

    let move = 8;
    if (this.attributes.SIZ + this.attributes.STR >= 100) {
      move = 7;
    }
    if (this.attributes.SIZ + this.attributes.STR >= 125) {
      move = 6;
    }
    if (this.attributes.SIZ + this.attributes.STR >= 150) {
      move = 5;
    }
    if (this.attributes.SIZ + this.attributes.STR >= 175) {
      move = 4;
    }
    if (this.attributes.SIZ + this.attributes.STR >= 200) {
      move = 3;
    }

    // 年龄调整
    if (age >= 40) move -= 1;
    if (age >= 50) move -= 1;
    if (age >= 60) move -= 1;
    if (age >= 70) move -= 1;
    if (age >= 80) move -= 1;

    return Math.max(move, 0);
  }

  /**
   * 计算体格伤害加成 (DB)
   * DB = STR + SIZ - 64，分段计算
   */
  calculateDamageBonus(): number {
    const total = this.attributes.STR + this.attributes.SIZ;

    if (total < 64) return -2;
    if (total < 84) return -1;
    if (total < 124) return 0;
    if (total < 164) return '+1';
    if (total < 204) return '+2';
    if (total < 284) return '+3';
    if (total < 364) return '+4';
    if (total < 444) return '+5';
    return '+6';
  }

  /**
   * 计算闪避值 (Dodge)
   * Dodge = DEX / 2，向上取整
   */
  calculateDodge(): number {
    return Math.ceil(this.attributes.DEX / 2);
  }

  /**
   * 计算所有派生数值
   */
  calculateAll(): DerivedValues {
    return {
      HP: this.calculateHP(),
      HP_max: this.calculateHP(),
      MP: this.calculateMP(),
      MP_max: this.calculateMP(),
      SAN: this.calculateSAN(),
      SAN_max: this.calculateSAN(),
      Luck: this.calculateLuck(),
      Luck_max: this.calculateLuck(),
      Move: this.calculateMove(),
      DamageBonus: this.calculateDamageBonus() as unknown as number,
      Dodge: this.calculateDodge(),
    };
  }
}

/**
 * 从骰点生成随机属性
 */
export function rollRandomAttributes(): Attributes {
  // 每个属性掷 3d6 (0-21 范围)
  const roll3d6 = (): number => {
    const dice = [rollD6(), rollD6(), rollD6()];
    return dice.reduce((a, b) => a + b, 0);
  };

  return {
    STR: roll3d6(),
    CON: roll3d6(),
    DEX: roll3d6(),
    APP: roll3d6(),
    POW: roll3d6(),
    INT: roll3d6() + 6, // 智力加 6 使范围更大
    SIZ: roll3d6() + 6, // 体型加 6
    EDU: roll3d6() * 2 + 6, // 教育掷 2d6+6
  };
}

function rollD6(): number {
  return Math.floor(Math.random() * 6) + 1;
}
```

---

### 1.3 角色状态管理 (30min)

| ID | 任务 | 预估时间 | 状态 |
|----|------|----------|------|
| M0-028-07 | [ ] 实现状态变更方法 | 10min | [ ] |
| M0-028-08 | [ ] 实现伤害/治疗逻辑 | 10min | [ ] |
| M0-028-09 | [ ] 实现背包管理 | 10min | [ ] |

```typescript
// src/core/states/character-manager.ts

import {
  CharacterState,
  CharacterStatus,
  Attributes,
  DerivedValues,
  DEFAULT_ATTRIBUTES,
  DEFAULT_DERIVED,
} from './character-state';
import { CharacterCalculator } from './character-calculator';

/**
 * 角色管理器
 */
export class CharacterManager {
  private state: CharacterState;

  constructor(state: CharacterState) {
    this.state = state;
  }

  /**
   * 获取当前状态
   */
  getState(): Readonly<CharacterState> {
    return this.state;
  }

  /**
   * 应用伤害
   * 返回实际扣除的生命值
   */
  applyDamage(amount: number): number {
    const actualDamage = Math.min(amount, this.state.derived.HP);
    this.state.derived.HP -= actualDamage;

    // 检查状态变化
    if (this.state.derived.HP <= 0) {
      this.state.status = CharacterStatus.DEAD;
    } else if (this.state.derived.HP <= this.state.derived.HP_max * 0.2) {
      this.state.status = CharacterStatus.DYING;
    } else if (this.state.derived.HP <= 0) {
      this.state.status = CharacterStatus.UNCONSCIOUS;
    }

    this.state.updatedAt = new Date();
    return actualDamage;
  }

  /**
   * 应用治疗
   * 返回实际恢复的生命值
   */
  applyHeal(amount: number): number {
    const maxHeal = this.state.derived.HP_max - this.state.derived.HP;
    const actualHeal = Math.min(amount, maxHeal);

    this.state.derived.HP += actualHeal;

    // 恢复后检查状态
    if (this.state.derived.HP > 0 &&
        this.state.status !== CharacterStatus.INSANE) {
      this.state.status = CharacterStatus.ALIVE;
    }

    this.state.updatedAt = new Date();
    return actualHeal;
  }

  /**
   * 消耗幸运
   */
  spendLuck(amount: number): boolean {
    if (amount > this.state.derived.Luck) {
      return false;
    }
    this.state.derived.Luck -= amount;
    this.state.updatedAt = new Date();
    return true;
  }

  /**
   * 恢复幸运
   */
  restoreLuck(amount: number): void {
    this.state.derived.Luck = Math.min(
      this.state.derived.Luck + amount,
      this.state.derived.Luck_max
    );
    this.state.updatedAt = new Date();
  }

  /**
   * 减少 SAN 值
   * 返回减少后的 SAN 值
   */
  reduceSAN(amount: number): number {
    const oldSAN = this.state.derived.SAN;
    this.state.derived.SAN = Math.max(0, this.state.derived.SAN - amount);

    // 检查是否进入疯狂状态
    if (this.state.derived.SAN <= this.state.derived.SAN_max * 0.2) {
      this.state.status = CharacterStatus.INSANE;
    }

    this.state.updatedAt = new Date();
    return oldSAN - this.state.derived.SAN;
  }

  /**
   * 添加物品到背包
   */
  addItem(item: string): void {
    this.state.inventory.push(item);
    this.state.updatedAt = new Date();
  }

  /**
   * 从背包移除物品
   */
  removeItem(item: string): boolean {
    const index = this.state.inventory.indexOf(item);
    if (index === -1) {
      return false;
    }
    this.state.inventory.splice(index, 1);
    this.state.updatedAt = new Date();
    return true;
  }

  /**
   * 检查是否有物品
   */
  hasItem(item: string): boolean {
    return this.state.inventory.includes(item);
  }

  /**
   * 更新笔记
   */
  updateNotes(notes: string): void {
    this.state.notes = notes;
    this.state.updatedAt = new Date();
  }

  /**
   * 添加笔记
   */
  appendNotes(content: string): void {
    this.state.notes += (this.state.notes ? '\n' : '') + content;
    this.state.updatedAt = new Date();
  }
}

/**
 * 创建默认角色
 */
export function createDefaultCharacter(
  characterId: string,
  name: string
): CharacterState {
  const attributes = DEFAULT_ATTRIBUTES;
  const calculator = new CharacterCalculator(attributes);

  return {
    characterId,
    name,
    attributes,
    derived: calculator.calculateAll(),
    status: CharacterStatus.ALIVE,
    inventory: [],
    notes: '',
    createdAt: new Date(),
    updatedAt: new Date(),
  };
}
```

---

## 单元测试

```typescript
// tests/unit/core/states/character-state.test.ts

import {
  CharacterState,
  CharacterStatus,
  DEFAULT_ATTRIBUTES,
  DEFAULT_DERIVED,
  createDefaultCharacter,
} from '@/core/states/character-state';
import { CharacterCalculator } from '@/core/states/character-calculator';
import { CharacterManager } from '@/core/states/character-manager';

describe('CharacterCalculator', () => {
  const createCalculator = (attrs: Partial<typeof DEFAULT_ATTRIBUTES> = {}): CharacterCalculator => {
    const attributes = { ...DEFAULT_ATTRIBUTES, ...attrs };
    return new CharacterCalculator(attributes);
  };

  describe('calculateHP', () => {
    it('应该正确计算 HP (CON=50, SIZ=50)', () => {
      const calculator = createCalculator({ CON: 50, SIZ: 50 });
      expect(calculator.calculateHP()).toBe(10);
    });

    it('应该向上取整 (CON=65, SIZ=50)', () => {
      const calculator = createCalculator({ CON: 65, SIZ: 50 });
      expect(calculator.calculateHP()).toBe(12); // 115/10 = 11.5 -> 12
    });
  });

  describe('calculateMP', () => {
    it('应该正确计算 MP (POW=60)', () => {
      const calculator = createCalculator({ POW: 60 });
      expect(calculator.calculateMP()).toBe(6);
    });

    it('应该向上取整 (POW=65)', () => {
      const calculator = createCalculator({ POW: 65 });
      expect(calculator.calculateMP()).toBe(7);
    });
  });

  describe('calculateSAN', () => {
    it('应该正确计算 SAN (POW=60)', () => {
      const calculator = createCalculator({ POW: 60 });
      expect(calculator.calculateSAN()).toBe(300);
    });
  });

  describe('calculateLuck', () => {
    it('应该等于 SAN 值 (POW=60)', () => {
      const calculator = createCalculator({ POW: 60 });
      expect(calculator.calculateLuck()).toBe(calculator.calculateSAN());
    });
  });

  describe('calculateDodge', () => {
    it('应该正确计算闪避 (DEX=60)', () => {
      const calculator = createCalculator({ DEX: 60 });
      expect(calculator.calculateDodge()).toBe(30);
    });

    it('应该向上取整 (DEX=65)', () => {
      const calculator = createCalculator({ DEX: 65 });
      expect(calculator.calculateDodge()).toBe(33); // 65/2 = 32.5 -> 33
    });
  });

  describe('calculateDamageBonus', () => {
    it('STR+SIZ < 64 应该返回 -2', () => {
      const calculator = createCalculator({ STR: 30, SIZ: 30 });
      expect(calculator.calculateDamageBonus()).toBe(-2);
    });

    it('STR+SIZ >= 164 应该返回 +1', () => {
      const calculator = createCalculator({ STR: 80, SIZ: 90 });
      expect(calculator.calculateDamageBonus()).toBe('+1');
    });
  });

  describe('calculateMove', () => {
    it('普通体型角色应该有 8 点移动力', () => {
      const calculator = createCalculator({
        STR: 50,
        DEX: 50,
        SIZ: 50,
      });
      expect(calculator.calculateMove()).toBe(8);
    });
  });

  describe('calculateAll', () => {
    it('应该返回完整的派生数值', () => {
      const calculator = createCalculator({
        STR: 50,
        CON: 50,
        DEX: 50,
        APP: 50,
        POW: 50,
        INT: 50,
        SIZ: 50,
        EDU: 50,
      });

      const derived = calculator.calculateAll();

      expect(derived).toHaveProperty('HP');
      expect(derived).toHaveProperty('MP');
      expect(derived).toHaveProperty('SAN');
      expect(derived).toHaveProperty('Luck');
      expect(derived).toHaveProperty('Move');
      expect(derived).toHaveProperty('Dodge');
    });
  });
});

describe('CharacterManager', () => {
  const createTestCharacter = (): CharacterManager => {
    const state = createDefaultCharacter('char-001', '测试角色');
    return new CharacterManager(state);
  };

  describe('applyDamage', () => {
    it('应该正确扣除 HP', () => {
      const manager = createTestCharacter();

      const actualDamage = manager.applyDamage(5);

      expect(actualDamage).toBe(5);
      expect(manager.getState().derived.HP).toBe(
        manager.getState().derived.HP_max - 5
      );
    });

    it('不应该扣除超过当前 HP', () => {
      const manager = createTestCharacter();
      const maxHP = manager.getState().derived.HP_max;

      const actualDamage = manager.applyDamage(maxHP + 100);

      expect(actualDamage).toBe(maxHP);
      expect(manager.getState().derived.HP).toBe(0);
    });
  });

  describe('applyHeal', () => {
    it('应该正确恢复 HP', () => {
      const manager = createTestCharacter();
      manager.applyDamage(10);

      const actualHeal = manager.applyHeal(5);

      expect(actualHeal).toBe(5);
      expect(manager.getState().derived.HP).toBe(
        manager.getState().derived.HP_max - 5
      );
    });

    it('不应该超过最大 HP', () => {
      const manager = createTestCharacter();
      manager.applyDamage(5);

      const actualHeal = manager.applyHeal(100);

      expect(actualHeal).toBe(5);
      expect(manager.getState().derived.HP).toBe(manager.getState().derived.HP_max);
    });
  });

  describe('spendLuck', () => {
    it('应该能消耗幸运', () => {
      const manager = createTestCharacter();
      const initialLuck = manager.getState().derived.Luck;

      const result = manager.spendLuck(10);

      expect(result).toBe(true);
      expect(manager.getState().derived.Luck).toBe(initialLuck - 10);
    });

    it('幸运不足时应返回 false', () => {
      const manager = createTestCharacter();

      const result = manager.spendLuck(10000);

      expect(result).toBe(false);
      expect(manager.getState().derived.Luck).toBe(manager.getState().derived.Luck_max);
    });
  });

  describe('reduceSAN', () => {
    it('应该正确减少 SAN', () => {
      const manager = createTestCharacter();
      const initialSAN = manager.getState().derived.SAN;

      const reduced = manager.reduceSAN(50);

      expect(reduced).toBe(50);
      expect(manager.getState().derived.SAN).toBe(initialSAN - 50);
    });

    it('不应该减少到 0 以下', () => {
      const manager = createTestCharacter();

      manager.reduceSAN(10000);

      expect(manager.getState().derived.SAN).toBe(0);
    });
  });

  describe('inventory', () => {
    it('应该能添加物品', () => {
      const manager = createTestCharacter();

      manager.addItem('手枪');

      expect(manager.hasItem('手枪')).toBe(true);
    });

    it('应该能移除物品', () => {
      const manager = createTestCharacter();
      manager.addItem('手枪');

      const removed = manager.removeItem('手枪');

      expect(removed).toBe(true);
      expect(manager.hasItem('手枪')).toBe(false);
    });

    it('移除不存在的物品应返回 false', () => {
      const manager = createTestCharacter();

      const removed = manager.removeItem('不存在的物品');

      expect(removed).toBe(false);
    });
  });
});
```

---

## 验收标准

- [ ] CharacterState 包含所有必需字段
- [ ] Attributes 包含所有 8 个 CoC 7e 属性
- [ ] DerivedValues 包含所有派生数值
- [ ] CharacterCalculator 正确计算所有派生值
- [ ] CharacterManager 正确处理伤害/治疗/SAN 变化
- [ ] 背包管理功能完整
- [ ] 单元测试覆盖率达到 80% 以上

---

## 涉及文件

| 文件 | 操作 |
|------|------|
| `src/core/states/character-state.ts` | 创建 |
| `src/core/states/character-calculator.ts` | 创建 |
| `src/core/states/character-manager.ts` | 创建 |
| `tests/unit/core/states/character-state.test.ts` | 创建 |

---

## 参考文档

- [01-m0-spec-freeze.md - 状态字段定义](../01-m0-spec-freeze.md)
- CoC 7e 规则书 - 第 2 章 角色创建
- CoC 7e 规则书 - 第 6 章 游戏进行
