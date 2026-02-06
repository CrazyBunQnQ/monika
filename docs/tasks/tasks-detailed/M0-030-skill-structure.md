# M0-030: Skill 技能结构

**任务类型**: spec
**预估工时**: 2h
**依赖**: M0-028
**状态**: [ ]

---

## 子任务拆解

### 1.1 Skill 核心结构设计 (30min)

| ID | 任务 | 预估时间 | 状态 |
|----|------|----------|------|
| M0-030-01 | [ ] 定义技能分类枚举 | 10min | [ ] |
| M0-030-02 | [ ] 定义技能数据结构 | 10min | [ ] |
| M0-030-03 | [ ] 定义技能列表常量 | 10min | [ ] |

```typescript
// src/core/skills/skill-types.ts

/**
 * 技能分类枚举
 */
export enum SkillCategory {
  /** 战斗技能 */
  COMBAT = 'combat',
  /** 运动技能 */
  ATHLETICS = 'athletics',
  /** 学术技能 */
  ACADEMIC = 'academic',
  /** 社交技能 */
  SOCIAL = 'social',
  /** 潜行技能 */
  STEALTH = 'stealth',
  /** 调查技能 */
  INVESTIGATION = 'investigation',
  /** 驾驶技能 */
  DRIVING = 'driving',
  /** 艺术/手艺技能 */
  ART_CRAFT = 'art_craft',
  /** 科学技术技能 */
  SCIENCE_TECH = 'science_tech',
  /** 灵异技能 */
  OCCULT = 'occult',
  /** 医疗技能 */
  MEDICAL = 'medical',
  /** 行为技能 */
  BEHAVIORAL = 'behavioral',
}

/**
 * 技能类型
 */
export interface Skill {
  /** 技能 ID (英文名，下划线分隔) */
  id: string;

  /** 技能名称 */
  name: string;

  /** 所属分类 */
  category: SkillCategory;

  /** 基础值公式 */
  baseValue: number;

  /** 基础值说明 */
  baseDescription: string;

  /** 技能说明 */
  description: string;

  /** 是否可作为属性使用 */
  canBeAttribute: boolean;

  /** 关联的属性 */
  linkedAttribute?: keyof typeof AttributeName;
}

/**
 * 属性名称类型
 */
export const AttributeName = {
  STR: '力量',
  CON: '体质',
  DEX: '敏捷',
  APP: '外貌',
  POW: '意志',
  INT: '智力',
  SIZ: '体型',
  EDU: '教育',
} as const;

/**
 * CoC 7e 标准技能定义
 */
export const COC7_SKILLS: Skill[] = [
  // === 战斗技能 ===
  {
    id: 'fighting_brawl',
    name: '斗殴',
    category: SkillCategory.COMBAT,
    baseValue: 25,
    baseDescription: '25% 或 (STR+DEX)/2',
    description: '使用拳头、脚、棍棒等近战武器进行攻击',
    canBeAttribute: false,
    linkedAttribute: 'DEX',
  },
  {
    id: 'fighting_handgun',
    name: '手枪',
    category: SkillCategory.COMBAT,
    baseValue: 20,
    baseDescription: '20% 或 (DEX*2)',
    description: '使用手枪进行远程攻击',
    canBeAttribute: false,
    linkedAttribute: 'DEX',
  },
  {
    id: 'fighting_rifle',
    name: '步枪/霰弹枪',
    category: SkillCategory.COMBAT,
    baseValue: 20,
    baseDescription: '20% 或 (DEX*2)',
    description: '使用步枪或霰弹枪进行远程攻击',
    canBeAttribute: false,
    linkedAttribute: 'DEX',
  },

  // === 运动技能 ===
  {
    id: 'athletics',
    name: '运动',
    category: SkillCategory.ATHLETICS,
    baseValue: 25,
    baseDescription: '25% 或 (STR+DEX)',
    description: '跑步、游泳、攀爬、体操等运动活动',
    canBeAttribute: false,
    linkedAttribute: 'STR',
  },
  {
    id: 'dodge',
    name: '闪避',
    category: SkillCategory.ATHLETICS,
    baseValue: 25,
    baseDescription: '25% 或 DEX/2',
    description: '躲避攻击',
    canBeAttribute: false,
    linkedAttribute: 'DEX',
  },

  // === 学术技能 ===
  {
    id: 'accounting',
    name: '会计',
    category: SkillCategory.ACADEMIC,
    baseValue: 5,
    baseDescription: '5% 或 (EDU*5)',
    description: '处理账目、税务等财务工作',
    canBeAttribute: false,
    linkedAttribute: 'INT',
  },
  {
    id: 'law',
    name: '法律',
    category: SkillCategory.ACADEMIC,
    baseValue: 5,
    baseDescription: '5% 或 (EDU*5)',
    description: '了解法律条文和程序',
    canBeAttribute: false,
    linkedAttribute: 'INT',
  },
  {
    id: 'library_use',
    name: '图书馆使用',
    category: SkillCategory.ACADEMIC,
    baseValue: 20,
    baseDescription: '20% 或 (INT*5)',
    description: '在图书馆或档案室中查找信息',
    canBeAttribute: false,
    linkedAttribute: 'INT',
  },
  {
    id: 'natural_world',
    name: '自然界',
    category: SkillCategory.ACADEMIC,
    baseValue: 10,
    baseDescription: '10% 或 (INT*5)',
    description: '动植物学、气象学等自然科学知识',
    canBeAttribute: false,
    linkedAttribute: 'INT',
  },

  // === 社交技能 ===
  {
    id: 'credit_rating',
    name: '信用评级',
    category: SkillCategory.SOCIAL,
    baseValue: 0,
    baseDescription: '财富和社会地位的体现',
    description: '社交网络、信誉和可用资金',
    canBeAttribute: false,
    linkedAttribute: 'EDU',
  },
  {
    id: 'charm',
    name: '魅惑',
    category: SkillCategory.SOCIAL,
    baseValue: 15,
    baseDescription: '15% 或 (APP*5)',
    description: '通过个人魅力影响他人',
    canBeAttribute: false,
    linkedAttribute: 'APP',
  },
  {
    id: 'fast_talk',
    name: '话术',
    category: SkillCategory.SOCIAL,
    baseValue: 5,
    baseDescription: '5% 或 (APP*5)',
    description: '通过花言巧语说服他人',
    canBeAttribute: false,
    linkedAttribute: 'APP',
  },
  {
    id: 'intimidate',
    name: '恐吓',
    category: SkillCategory.SOCIAL,
    baseValue: 15,
    baseDescription: '15% 或 (STR*5)',
    description: '通过威胁迫使他人服从',
    canBeAttribute: false,
    linkedAttribute: 'STR',
  },
  {
    id: 'persuade',
    name: '说服',
    category: SkillCategory.SOCIAL,
    baseValue: 10,
    baseDescription: '10% 或 (APP*5)',
    description: '通过逻辑和论证说服他人',
    canBeAttribute: false,
    linkedAttribute: 'APP',
  },

  // === 潜行技能 ===
  {
    id: 'hide',
    name: '隐藏',
    category: SkillCategory.STEALTH,
    baseValue: 10,
    baseDescription: '10% 或 (DEX*5)',
    description: '在暗处隐藏自己',
    canBeAttribute: false,
    linkedAttribute: 'DEX',
  },
  {
    id: 'listen',
    name: '聆听',
    category: SkillCategory.STEALTH,
    baseValue: 20,
    baseDescription: '20% 或 (DEX*5)',
    description: '通过听觉察觉动静',
    canBeAttribute: false,
    linkedAttribute: 'DEX',
  },
  {
    id: 'sneak',
    name: '潜行',
    category: SkillCategory.STEALTH,
    baseValue: 10,
    baseDescription: '10% 或 (DEX*5)',
    description: '悄悄移动而不被发现',
    canBeAttribute: false,
    linkedAttribute: 'DEX',
  },

  // === 调查技能 ===
  {
    id: 'appraise',
    name: '估价',
    category: SkillCategory.INVESTIGATION,
    baseValue: 5,
    baseDescription: '5% 或 (INT*5)',
    description: '评估物品的价值',
    canBeAttribute: false,
    linkedAttribute: 'INT',
  },
  {
    id: 'forensics',
    name: '法医学',
    category: SkillCategory.INVESTIGATION,
    baseValue: 10,
    baseDescription: '10% 或 (INT*5)',
    description: '分析犯罪现场证据',
    canBeAttribute: false,
    linkedAttribute: 'INT',
  },
  {
    id: 'investigate',
    name: '调查',
    category: SkillCategory.INVESTIGATION,
    baseValue: 10,
    baseDescription: '10% 或 (INT*5)',
    description: '系统性地搜集线索和信息',
    canBeAttribute: false,
    linkedAttribute: 'INT',
  },
  {
    id: 'spot_hidden',
    name: '侦查',
    category: SkillCategory.INVESTIGATION,
    baseValue: 10,
    baseDescription: '10% 或 (INT*5)',
    description: '发现隐藏的物品或线索',
    canBeAttribute: false,
    linkedAttribute: 'INT',
  },

  // === 驾驶技能 ===
  {
    id: 'drive_auto',
    name: '汽车驾驶',
    category: SkillCategory.DRIVING,
    baseValue: 20,
    baseDescription: '20% 或 (DEX*5)',
    description: '驾驶汽车和其他机动车',
    canBeAttribute: false,
    linkedAttribute: 'DEX',
  },
  {
    id: 'pilot',
    name: '驾驶',
    category: SkillCategory.DRIVING,
    baseValue: 1,
    baseDescription: '1% 或 (INT*5)',
    description: '驾驶飞机、船只等交通工具',
    canBeAttribute: false,
    linkedAttribute: 'INT',
  },

  // === 艺术/手艺技能 ===
  {
    id: 'art_craft',
    name: '艺术/手艺',
    category: SkillCategory.ART_CRAFT,
    baseValue: 5,
    baseDescription: '5% 或 (DEX*5)',
    description: '各种艺术创作或手工艺技能',
    canBeAttribute: false,
    linkedAttribute: 'DEX',
  },

  // === 科学技术技能 ===
  {
    id: 'computer_use',
    name: '计算机使用',
    category: SkillCategory.SCIENCE_TECH,
    baseValue: 5,
    baseDescription: '5% 或 (INT*5)',
    description: '使用计算机和软件',
    canBeAttribute: false,
    linkedAttribute: 'INT',
  },
  {
    id: 'electronics',
    name: '电子学',
    category: SkillCategory.SCIENCE_TECH,
    baseValue: 1,
    baseDescription: '1% 或 (INT*5)',
    description: '理解和维修电子设备',
    canBeAttribute: false,
    linkedAttribute: 'INT',
  },
  {
    id: 'mechanical_repair',
    name: '机械维修',
    category: SkillCategory.SCIENCE_TECH,
    baseValue: 10,
    baseDescription: '10% 或 (DEX*5)',
    description: '维修机械设备和车辆',
    canBeAttribute: false,
    linkedAttribute: 'DEX',
  },

  // === 灵异技能 ===
  {
    id: 'occult',
    name: '神秘学',
    category: SkillCategory.OCCULT,
    baseValue: 5,
    baseDescription: '5% 或 (INT*5)',
    description: '关于神秘学、神话和古代禁忌的知识',
    canBeAttribute: false,
    linkedAttribute: 'INT',
  },

  // === 医疗技能 ===
  {
    id: 'first_aid',
    name: '急救',
    category: SkillCategory.MEDICAL,
    baseValue: 10,
    baseDescription: '10% 或 (EDU*5)',
    description: '紧急医疗处理',
    canBeAttribute: false,
    linkedAttribute: 'EDU',
  },
  {
    id: 'medicine',
    name: '医学',
    category: SkillCategory.MEDICAL,
    baseValue: 1,
    baseDescription: '1% 或 (EDU*5)',
    description: '诊断和治疗疾病',
    canBeAttribute: false,
    linkedAttribute: 'EDU',
  },

  // === 行为技能 ===
  {
    id: 'psychology',
    name: '心理学',
    category: SkillCategory.BEHAVIORAL,
    baseValue: 5,
    baseDescription: '5% 或 (APP*5)',
    description: '理解他人心理和动机',
    canBeAttribute: false,
    linkedAttribute: 'INT',
  },
  {
    id: 'psychoanalysis',
    name: '精神分析',
    category: SkillCategory.BEHAVIORAL,
    baseValue: 1,
    baseDescription: '1% 或 (EDU*5)',
    description: '治疗精神疾病',
    canBeAttribute: false,
    linkedAttribute: 'EDU',
  },
];

/**
 * 角色技能记录
 */
export interface CharacterSkills {
  /** 角色ID */
  characterId: string;

  /** 技能值映射 */
  skills: Record<string, number>;

  /** 最后更新时间 */
  updatedAt: Date;
}
```

---

### 1.2 技能服务实现 (30min)

| ID | 任务 | 预估时间 | 状态 |
|----|------|----------|------|
| M0-030-04 | [ ] 实现技能查找服务 | 10min | [ ] |
| M0-030-05 | [ ] 实现技能值计算 | 10min | [ ] |
| M0-030-06 | [ ] 实现技能分类查询 | 10min | [ ] |

```typescript
// src/core/skills/skill-service.ts

import {
  Skill,
  Skills,
  COC7_SKILLS,
  COC7_SKILLS_BY_ID,
  SkillCategory,
  CharacterSkills,
} from './skill-types';

/**
 * 按 ID 查找技能
 */
export function findSkillById(id: string): Skill | undefined {
  return COC7_SKILLS_BY_ID.get(id);
}

/**
 * 按名称查找技能
 */
export function findSkillByName(name: string): Skill | undefined {
  return COC7_SKILLS.find((s) => s.name === name);
}

/**
 * 获取所有技能
 */
export function getAllSkills(): Skill[] {
  return [...COC7_SKILLS];
}

/**
 * 获取指定分类的技能
 */
export function getSkillsByCategory(category: SkillCategory): Skill[] {
  return COC7_SKILLS.filter((s) => s.category === category);
}

/**
 * 获取所有技能分类
 */
export function getAllCategories(): SkillCategory[] {
  return Object.values(SkillCategory);
}

/**
 * 技能服务类
 */
export class SkillService {
  private characterSkills: CharacterSkills;

  constructor(characterSkills: CharacterSkills) {
    this.characterSkills = characterSkills;
  }

  /**
   * 获取指定技能的值
   */
  getSkillValue(skillId: string): number {
    return this.characterSkills.skills[skillId] ?? 0;
  }

  /**
   * 设置技能值
   */
  setSkillValue(skillId: string, value: number): void {
    this.characterSkills.skills[skillId] = Math.max(0, Math.min(100, value));
    this.characterSkills.updatedAt = new Date();
  }

  /**
   * 增加技能值
   */
  increaseSkill(skillId: string, amount: number): void {
    const current = this.getSkillValue(skillId);
    this.setSkillValue(skillId, current + amount);
  }

  /**
   * 减少技能值
   */
  decreaseSkill(skillId: string, amount: number): void {
    const current = this.getSkillValue(skillId);
    this.setSkillValue(skillId, current - amount);
  }

  /**
   * 获取所有技能的数组
   */
  getAllSkillValues(): { skill: Skill; value: number }[] {
    return COC7_SKILLS.map((skill) => ({
      skill,
      value: this.getSkillValue(skill.id),
    }));
  }

  /**
   * 获取指定分类的技能值
   */
  getSkillValuesByCategory(category: SkillCategory): { skill: Skill; value: number }[] {
    return this.getAllSkillValues().filter(
      ({ skill }) => skill.category === category
    );
  }

  /**
   * 获取最高技能值
   */
  getTopSkills(limit: number = 10): { skill: Skill; value: number }[] {
    return this.getAllSkillValues()
      .sort((a, b) => b.value - a.value)
      .slice(0, limit);
  }

  /**
   * 获取专业技能（值 >= 50）
   */
  getProfessionalSkills(): { skill: Skill; value: number }[] {
    return this.getAllSkillValues().filter(({ value }) => value >= 50);
  }

  /**
   * 检查技能是否达标 (值 >= 50)
   */
  isProfessional(skillId: string): boolean {
    return this.getSkillValue(skillId) >= 50;
  }

  /**
   * 获取技能检定结果
   * 返回 { success: boolean, degree: 'critical' | 'hard' | 'regular' | 'failure' }
   */
  getCheckResult(skillId: string, roll: number): {
    success: boolean;
    degree: 'critical' | 'hard' | 'regular' | 'failure';
  } {
    const skillValue = this.getSkillValue(skillId);
    const hardThreshold = Math.ceil(skillValue / 2);
    const criticalThreshold = Math.ceil(skillValue / 5);

    if (roll <= criticalThreshold) {
      return { success: true, degree: 'critical' };
    }
    if (roll <= hardThreshold) {
      return { success: true, degree: 'hard' };
    }
    if (roll <= skillValue) {
      return { success: true, degree: 'regular' };
    }
    return { success: false, degree: 'failure' };
  }
}

// 创建技能映射表
const COC7_SKILLS_BY_ID = new Map<string, Skill>();
COC7_SKILLS.forEach((skill) => {
  COC7_SKILLS_BY_ID.set(skill.id, skill);
});
```

---

### 1.3 技能初始化工具 (30min)

| ID | 任务 | 预估时间 | 状态 |
|----|------|----------|------|
| M0-030-07 | [ ] 实现职业技能推荐 | 10min | [ ] |
| M0-030-08 | [ ] 实现初始技能分配 | 10min | [ ] |
| M0-030-09 | [ ] 实现技能点计算 | 10min | [ ] |

```typescript
// src/core/skills/skill-initialization.ts

import {
  Skill,
  COC7_SKILLS,
  SkillCategory,
  CharacterSkills,
} from './skill-types';
import { Attributes } from '../states/character-state';

/**
 * 职业技能推荐配置
 */
export interface OccupationRecommendation {
  /** 职业名称 */
  occupation: string;

  /** 推荐技能及其最小值 */
  recommendedSkills: { skillId: string; minValue: number }[];

  /** 信用评级范围 */
  creditRating: { min: number; max: number };
}

/**
 * CoC 7e 职业技能推荐表
 */
export const OCCUPATION_RECOMMENDATIONS: OccupationRecommendation[] = [
  {
    occupation: '医生',
    recommendedSkills: [
      { skillId: 'medicine', minValue: 60 },
      { skillId: 'first_aid', minValue: 50 },
      { skillId: 'psychology', minValue: 40 },
      { skillId: 'science_tech', minValue: 30 },
      { skillId: 'library_use', minValue: 30 },
      { skillId: 'bargain', minValue: 20 },
    ],
    creditRating: { min: 30, max: 70 },
  },
  {
    occupation: '私家侦探',
    recommendedSkills: [
      { skillId: 'spot_hidden', minValue: 60 },
      { skillId: 'listen', minValue: 50 },
      { skillId: 'psychology', minValue: 50 },
      { skillId: 'library_use', minValue: 40 },
      { skillId: 'photography', minValue: 30 },
      { skillId: 'disguise', minValue: 20 },
    ],
    creditRating: { min: 15, max: 40 },
  },
  {
    occupation: '记者',
    recommendedSkills: [
      { skillId: 'library_use', minValue: 60 },
      { skillId: 'persuade', minValue: 50 },
      { skillId: 'psychology', minValue: 40 },
      { skillId: 'photography', minValue: 30 },
      { skillId: 'history', minValue: 30 },
      { skillId: 'charm', minValue: 30 },
    ],
    creditRating: { min: 10, max: 30 },
  },
  {
    occupation: '教授',
    recommendedSkills: [
      { skillId: 'library_use', minValue: 70 },
      { skillId: 'natural_world', minValue: 50 },
      { skillId: 'history', minValue: 50 },
      { skillId: 'teach', minValue: 40 },
      { skillId: 'anthropology', minValue: 30 },
      { skillId: 'archeology', minValue: 30 },
    ],
    creditRating: { min: 20, max: 50 },
  },
  {
    occupation: '警察',
    recommendedSkills: [
      { skillId: 'firearms', minValue: 50 },
      { skillId: 'fighting_brawl', minValue: 50 },
      { skillId: 'drive_auto', minValue: 50 },
      { skillId: 'psychology', minValue: 40 },
      { skillId: 'law', minValue: 30 },
      { skillId: 'spot_hidden', minValue: 30 },
    ],
    creditRating: { min: 20, max: 40 },
  },
];

/**
 * 创建空技能记录
 */
export function createEmptySkills(characterId: string): CharacterSkills {
  return {
    characterId,
    skills: {},
    updatedAt: new Date(),
  };
}

/**
 * 从职业初始化技能
 */
export function initializeSkillsFromOccupation(
  characterId: string,
  occupation: string,
  attributes: Attributes
): CharacterSkills {
  const skills = createEmptySkills(characterId);

  // 查找职业推荐
  const recommendation = OCCUPATION_RECOMMENDATIONS.find(
    (r) => r.occupation === occupation
  );

  if (recommendation) {
    // 设置推荐技能值
    recommendation.recommendedSkills.forEach(({ skillId, minValue }) => {
      skills.skills[skillId] = minValue;
    });
  }

  // 计算基于属性的技能值
  COC7_SKILLS.forEach((skill) => {
    if (!skills.skills[skill.id]) {
      // 如果技能未被设置，使用基础值
      if (typeof skill.baseValue === 'number') {
        skills.skills[skill.id] = skill.baseValue;
      }
    }
  });

  return skills;
}

/**
 * 计算角色可用技能点数
 * 基础 480 点，INT 每高 10 点增加 10 点
 */
export function calculateSkillPoints(attributes: Attributes): number {
  const basePoints = 480;
  const intBonus = Math.floor((attributes.INT - 50) / 10) * 10;
  return Math.max(0, basePoints + intBonus);
}

/**
 * 计算已分配的技能点数
 */
export function calculateAllocatedPoints(skills: CharacterSkills): number {
  let allocated = 0;
  COC7_SKILLS.forEach((skill) => {
    allocated += skills.skills[skill.id] ?? 0;
  });
  return allocated;
}

/**
 * 获取职业完成度
 */
export function getOccupationCompletion(
  skills: CharacterSkills,
  occupation: string
): number {
  const recommendation = OCCUPATION_RECOMMENDATIONS.find(
    (r) => r.occupation === occupation
  );

  if (!recommendation) {
    return 0;
  }

  let completed = 0;
  recommendation.recommendedSkills.forEach(({ skillId, minValue }) => {
    if (skills.skills[skillId] >= minValue) {
      completed++;
    }
  });

  return Math.round((completed / recommendation.recommendedSkills.length) * 100);
}
```

---

## 单元测试

```typescript
// tests/unit/core/skills/skill-types.test.ts

import {
  Skill,
  COC7_SKILLS,
  COC7_SKILLS_BY_ID,
  SkillCategory,
  CharacterSkills,
} from '@/core/skills/skill-types';
import {
  SkillService,
  findSkillById,
  findSkillByName,
  getSkillsByCategory,
} from '@/core/skills/skill-service';
import {
  createEmptySkills,
  initializeSkillsFromOccupation,
  calculateSkillPoints,
  calculateAllocatedPoints,
  getOccupationCompletion,
} from '@/core/skills/skill-initialization';

describe('Skill Types', () => {
  describe('COC7_SKILLS', () => {
    it('应该包含所有 CoC 7e 标准技能', () => {
      expect(COC7_SKILLS.length).toBeGreaterThan(50);
    });

    it('所有技能应该有唯一的 ID', () => {
      const ids = COC7_SKILLS.map((s) => s.id);
      const uniqueIds = new Set(ids);
      expect(uniqueIds.size).toBe(ids.length);
    });

    it('所有技能应该有有效的分类', () => {
      const validCategories = Object.values(SkillCategory);
      COC7_SKILLS.forEach((skill) => {
        expect(validCategories).toContain(skill.category);
      });
    });

    it('应该包含战斗技能', () => {
      const combatSkills = COC7_SKILLS.filter(
        (s) => s.category === SkillCategory.COMBAT
      );
      expect(combatSkills.length).toBeGreaterThan(0);
    });

    it('应该包含社交技能', () => {
      const socialSkills = COC7_SKILLS.filter(
        (s) => s.category === SkillCategory.SOCIAL
      );
      expect(socialSkills.length).toBeGreaterThan(0);
    });
  });

  describe('COC7_SKILLS_BY_ID', () => {
    it('应该能通过 ID 查找斗殴技能', () => {
      const skill = COC7_SKILLS_BY_ID.get('fighting_brawl');
      expect(skill).toBeDefined();
      expect(skill?.name).toBe('斗殴');
    });

    it('应该能通过 ID 查找图书馆使用技能', () => {
      const skill = COC7_SKILLS_BY_ID.get('library_use');
      expect(skill).toBeDefined();
      expect(skill?.name).toBe('图书馆使用');
    });
  });
});

describe('Skill Service', () => {
  const createTestSkills = (): SkillService => {
    const skills: CharacterSkills = {
      characterId: 'char-001',
      skills: {
        library_use: 60,
        psychology: 45,
        fighting_brawl: 50,
        spot_hidden: 30,
      },
      updatedAt: new Date(),
    };
    return new SkillService(skills);
  };

  describe('getSkillValue', () => {
    it('应该返回已设置的技能值', () => {
      const service = createTestSkills();
      expect(service.getSkillValue('library_use')).toBe(60);
    });

    it('未设置的技能应返回 0', () => {
      const service = createTestSkills();
      expect(service.getSkillValue('medicine')).toBe(0);
    });
  });

  describe('setSkillValue', () => {
    it('应该正确设置技能值', () => {
      const service = createTestSkills();
      service.setSkillValue('medicine', 75);
      expect(service.getSkillValue('medicine')).toBe(75);
    });

    it('技能值不应超过 100', () => {
      const service = createTestSkills();
      service.setSkillValue('medicine', 150);
      expect(service.getSkillValue('medicine')).toBe(100);
    });

    it('技能值不应小于 0', () => {
      const service = createTestSkills();
      service.setSkillValue('medicine', -50);
      expect(service.getSkillValue('medicine')).toBe(0);
    });
  });

  describe('increaseSkill / decreaseSkill', () => {
    it('应该能增加技能值', () => {
      const service = createTestSkills();
      service.increaseSkill('library_use', 10);
      expect('library_use')).(service.getSkillValuetoBe(70);
    });

    it('应该能减少技能值', () => {
      const service = createTestSkills();
      service.decreaseSkill('library_use', 10);
      expect(service.getSkillValue('library_use')).toBe(50);
    });
  });

  describe('getCheckResult', () => {
    it('大成功 (roll <= 1/5 值)', () => {
      const service = createTestSkills();
      // library_use = 60, 1/5 = 12
      const result = service.getCheckResult('library_use', 10);
      expect(result.success).toBe(true);
      expect(result.degree).toBe('critical');
    });

    it('困难成功 (roll <= 1/2 值)', () => {
      const service = createTestSkills();
      // library_use = 60, 1/2 = 30
      const result = service.getCheckResult('library_use', 25);
      expect(result.success).toBe(true);
      expect(result.degree).toBe('hard');
    });

    it('普通成功 (roll <= 值)', () => {
      const service = createTestSkills();
      // library_use = 60
      const result = service.getCheckResult('library_use', 55);
      expect(result.success).toBe(true);
      expect(result.degree).toBe('regular');
    });

    it('失败 (roll > 值)', () => {
      const service = createTestSkills();
      const result = service.getCheckResult('library_use', 70);
      expect(result.success).toBe(false);
      expect(result.degree).toBe('failure');
    });
  });

  describe('getTopSkills', () => {
    it('应该返回按值排序的技能', () => {
      const service = createTestSkills();
      const topSkills = service.getTopSkills(3);

      expect(topSkills).toHaveLength(3);
      expect(topSkills[0].value).toBeGreaterThanOrEqual(topSkills[1].value);
      expect(topSkills[1].value).toBeGreaterThanOrEqual(topSkills[2].value);
    });
  });

  describe('isProfessional', () => {
    it('专业技能值 >= 50', () => {
      const service = createTestSkills();
      expect(service.isProfessional('library_use')).toBe(true);
      expect(service.isProfessional('spot_hidden')).toBe(false);
    });
  });
});

describe('Skill Initialization', () => {
  describe('createEmptySkills', () => {
    it('应该创建空的技能记录', () => {
      const skills = createEmptySkills('char-001');
      expect(skills.characterId).toBe('char-001');
      expect(Object.keys(skills.skills)).toHaveLength(0);
    });
  });

  describe('initializeSkillsFromOccupation', () => {
    it('应该从职业初始化技能', () => {
      const attributes = {
        STR: 50, CON: 50, DEX: 50, APP: 50,
        POW: 50, INT: 50, SIZ: 50, EDU: 50,
      };
      const skills = initializeSkillsFromOccupation(
        'char-001',
        '医生',
        attributes
      );

      expect(skills.characterId).toBe('char-001');
      expect(skills.skills['medicine']).toBeGreaterThanOrEqual(60);
      expect(skills.skills['first_aid']).toBeGreaterThanOrEqual(50);
    });
  });

  describe('calculateSkillPoints', () => {
    it('基础智力应返回 480 点', () => {
      const attributes = {
        STR: 50, CON: 50, DEX: 50, APP: 50,
        POW: 50, INT: 50, SIZ: 50, EDU: 50,
      };
      expect(calculateSkillPoints(attributes)).toBe(480);
    });

    it('高智力应增加技能点', () => {
      const attributes = {
        STR: 50, CON: 50, DEX: 50, APP: 50,
        POW: 50, INT: 70, SIZ: 50, EDU: 50,
      };
      expect(calculateSkillPoints(attributes)).toBe(500); // INT=70, +20
    });
  });

  describe('getOccupationCompletion', () => {
    it('应该计算职业完成度', () => {
      const skills: CharacterSkills = {
        characterId: 'char-001',
        skills: {
          medicine: 70,      // >= 60 ✓
          first_aid: 50,     // >= 50 ✓
          psychology: 30,    // < 40 ✗
          library_use: 30,   // < 30 ✗
        },
        updatedAt: new Date(),
      };

      const completion = getOccupationCompletion(skills, '医生');
      expect(completion).toBe(50); // 2/4
    });
  });
});
```

---

## 验收标准

- [ ] Skill 包含所有 CoC 7e 标准技能（50+）
- [ ] 技能按分类组织完整
- [ ] SkillService 能正确获取/设置技能值
- [ ] 技能检定结果计算正确（大成功/困难成功/普通成功/失败）
- [ ] 职业推荐技能系统完整
- [ ] 技能点数计算正确
- [ ] 单元测试覆盖率达到 80% 以上

---

## 涉及文件

| 文件 | 操作 |
|------|------|
| `src/core/skills/skill-types.ts` | 创建 |
| `src/core/skills/skill-service.ts` | 创建 |
| `src/core/skills/skill-initialization.ts` | 创建 |
| `tests/unit/core/skills/skill-types.test.ts` | 创建 |

---

## 参考文档

- [01-m0-spec-freeze.md - 技能结构定义](../01-m0-spec-freeze.md)
- CoC 7e 规则书 - 第 3 章 技能
- CoC 7e 规则书 - 角色卡示例
