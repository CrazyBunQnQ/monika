# M0-029 定义 Attribute 属性结构

## 概述
定义 CoC 角色的 8 个核心属性(STR/CON/DEX/APP/POW/INT/SIZ/EDU)的数据结构,包括属性值计算规则和派生属性。

## 验收标准
- [ ] 定义 8 个核心属性的数据结构
- [ ] 定义属性取值范围(0-100)
- [ ] 定义属性生成规则(掷骰规则)
- [ ] 定义派生属性计算公式(HP/MP/Move/DB/Build)
- [ ] 定义属性修正规则(年龄、伤害)
- [ ] 提供属性验证函数

## 技术方案

### 核心属性结构

```typescript
interface Attributes {
  // 力量 (Strength)
  STR: number;

  // 体质 (Constitution)
  CON: number;

  // 敏捷 (Dexterity)
  DEX: number;

  // 外貌 (Appearance)
  APP: number;

  // 意志 (Power)
  POW: number;

  // 智力 (Intelligence)
  INT: number;

  // 体型 (Size)
  SIZ: number;

  // 教育 (Education)
  EDU: number;
}

// 派生属性
interface DerivedAttributes {
  // 耐久值 = (CON + SIZ) / 10
  HP: number;
  HP_max: number;

  // 魔法值 = POW / 5
  MP: number;
  MP_max: number;

  // 幸运 = POW * 5
  Luck: number;
  Luck_max: number;

  // 移动速率 = DEX < SIZ 且 STR < SIZ ? 7 : 8
  Move: number;

  // 伤害加值 = STR + SIZ 查表
  DB: string; // "-1", "0", "+1d4", "+1d6"

  // 体格 = STR + SIZ 查表
  Build: number; // -2, -1, 0, 1, 2, 3, 4, 5
}

// 完整属性结构
interface AttributeData {
  primary: Attributes;
  derived: DerivedAttributes;
  bonks: {
    // 属性修正
    age_modifiers?: Partial<Attributes>;
    damage_modifiers?: {
      STR?: number;
      CON?: number;
      DEX?: number;
    };
  };
}
```

### 属性生成规则

```typescript
// CoC 7th Edition 属性生成
interface AttributeRollRule {
  attribute: keyof Attributes;
  formula: string;
  example: string;
}

const ATTRIBUTE_ROLL_RULES: AttributeRollRule[] = [
  {
    attribute: 'STR',
    formula: '(3d6 * 5)',
    example: '掷 3d6,结果乘以 5'
  },
  {
    attribute: 'CON',
    formula: '(3d6 * 5)',
    example: '掷 3d6,结果乘以 5'
  },
  {
    attribute: 'DEX',
    formula: '(3d6 * 5)',
    example: '掷 3d6,结果乘以 5'
  },
  {
    attribute: 'APP',
    formula: '(3d6 * 5)',
    example: '掷 3d6,结果乘以 5'
  },
  {
    attribute: 'POW',
    formula: '(3d6 * 5)',
    example: '掷 3d6,结果乘以 5'
  },
  {
    attribute: 'INT',
    formula: '(2d6+6 * 5)',
    example: '掷 2d6+6,结果乘以 5'
  },
  {
    attribute: 'SIZ',
    formula: '(2d6+6 * 5)',
    example: '掷 2d6+6,结果乘以 5'
  },
  {
    attribute: 'EDU',
    formula: '(2d6+6 * 5)',
    example: '掷 2d6+6,结果乘以 5,可重掷以提升'
  }
];

function rollAttribute(attribute: keyof Attributes): number {
  const rule = ATTRIBUTE_ROLL_RULES.find(r => r.attribute === attribute);
  if (!rule) return 0;

  // 解析公式并计算
  return parseAndRoll(rule.formula);
}
```

### 派生属性计算

```typescript
function calculateDerivedAttributes(primary: Attributes): DerivedAttributes {
  // HP 计算
  const hpSum = primary.CON + primary.SIZ;
  const HP = Math.floor(hpSum / 10);
  const HP_max = HP;

  // MP 计算
  const MP = Math.floor(primary.POW / 5);
  const MP_max = MP;

  // Luck 计算
  const Luck = primary.POW * 5;
  const Luck_max = Luck;

  // Move 计算
  const moveBase = primary.DEX < primary.SIZ && primary.STR < primary.SIZ ? 7 : 8;
  const Move = moveBase;

  // DB 和 Build 查表
  const strSizSum = primary.STR + primary.SIZ;
  const { DB, Build } = lookupDBAndBuild(strSizSum);

  return {
    HP,
    HP_max,
    MP,
    MP_max,
    Luck,
    Luck_max,
    Move,
    DB,
    Build
  };
}

// DB/Build 查表
function lookupDBAndBuild(sum: number): { DB: string; Build: number } {
  const table = [
    { max: 64, DB: '-1', Build: -2 },
    { max: 84, DB: '-1', Build: -1 },
    { max: 124, DB: '0', Build: 0 },
    { max: 164, DB: '+1d4', Build: 1 },
    { max: 204, DB: '+1d6', Build: 2 },
    { max: 284, DB: '+2d6', Build: 3 },
    { max: 364, DB: '+3d6', Build: 4 },
    { max: 444, DB: '+4d6', Build: 5 }
  ];

  for (const entry of table) {
    if (sum <= entry.max) {
      return { DB: entry.DB, Build: entry.Build };
    }
  }

  return { DB: '+5d6', Build: 5 };
}
```

### 年龄修正

```typescript
interface AgeModifierRule {
  ageRange: [number, number];
  modifiers: Partial<Attributes>;
  rolls: {
    STR: number;
    CON: number;
    DEX: number;
    EDU: number;
  };
}

const AGE_MODIFIERS: AgeModifierRule[] = [
  {
    ageRange: [15, 19],
    modifiers: { STR: -5, SIZ: -5 },
    rolls: { EDU: 1 }
  },
  {
    ageRange: [20, 39],
    modifiers: {},
    rolls: { EDU: 1 }
  },
  {
    ageRange: [40, 49],
    modifiers: { STR: -5, CON: -5, DEX: -5 },
    rolls: { EDU: 2 }
  },
  {
    ageRange: [50, 59],
    modifiers: { STR: -10, CON: -10, DEX: -10 },
    rolls: { EDU: 3 }
  },
  {
    ageRange: [60, 69],
    modifiers: { STR: -20, CON: -20, DEX: -20 },
    rolls: { EDU: 4 }
  },
  {
    ageRange: [70, 79],
    modifiers: { STR: -40, CON: -40, DEX: -40 },
    rolls: { EDU: 4 }
  }
];

function applyAgeModifiers(attributes: Attributes, age: number): Attributes {
  const rule = AGE_MODIFIERS.find(r => age >= r.ageRange[0] && age <= r.ageRange[1]);
  if (!rule) return attributes;

  const modified = { ...attributes };

  // 应用修正
  Object.entries(rule.modifiers).forEach(([attr, mod]) => {
    modified[attr as keyof Attributes] = Math.max(0, modified[attr as keyof Attributes] + mod);
  });

  // EDU 可重掷
  if (rule.rolls.EDU > 0) {
    for (let i = 0; i < rule.rolls.EDU; i++) {
      const roll = (Math.floor(Math.random() * 6) + 1) * 10;
      if (roll > modified.EDU) {
        modified.EDU = roll;
      }
    }
  }

  return modified;
}
```

### 属性验证

```typescript
function validateAttributes(attributes: Attributes): ValidationResult {
  const errors: string[] = [];

  // 取值范围
  Object.entries(attributes).forEach(([key, value]) => {
    if (value < 0 || value > 100) {
      errors.push(`${key} 必须在 0-100 之间,当前: ${value}`);
    }
  });

  // 派生属性合理性
  const derived = calculateDerivedAttributes(attributes);
  if (derived.HP < 1) {
    errors.push('HP 不能小于 1');
  }
  if (derived.Move < 1 || derived.Move > 10) {
    errors.push(`Move 异常: ${derived.Move}`);
  }

  return {
    valid: errors.length === 0,
    errors
  };
}
```

## 依赖关系
- 前置任务: M0-028 定义 CharacterState 角色状态
- 被依赖: M0-030 定义 Skill 技能结构

## 预估工时
2h
