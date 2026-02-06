# M0-016: 定义 scenes 场景集合结构

**任务ID**: M0-016
**标题**: 定义 scenes 场景集合结构
**类型**: spec (规范设计)
**预估工时**: 2h
**依赖**: M0-014

---

## 任务描述

定义场景包中 scenes 场景集合的数据结构，包括单个场景的定义和场景集合的组织方式。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M0-016-01 | 设计 Scene 结构 | 单个场景定义 | 30min |
| M0-016-02 | 设计 Scenes 集合结构 | 场景集合 | 20min |
| M0-016-03 | 设计 Narrative 结构 | 叙事内容 | 25min |
| M0-016-04 | 设计 Transitions 结构 | 场景跳转 | 25min |
| M0-016-05 | 设计 Requirements 结构 | 场景前置条件 | 15min |
| M0-016-06 | 编写 TypeScript 类型 | 类型定义 | 20min |
| M0-016-07 | 编写示例场景 | 完整示例 | 10min |

---

## Scene 结构

```typescript
interface Scene {
  // === 标识 ===
  id: string;
  /** 场景唯一标识符 */

  title: string;
  /** 场景标题 */

  order: number;
  /** 播放顺序 (从小到大) */

  // === 叙事内容 ===
  narrative: NarrativeContent;

  // === 场景元素引用 ===
  npcs: string[];
  /** NPC 引用列表，指向 shared.npcs */

  locations: string[];
  /** 地点引用列表，指向 shared.locations */

  clues: string[];
  /** 线索引用列表，指向 shared.clues */

  handouts: string[];
  /** 手递物引用列表，指向 shared.handouts */

  // === 状态转换 ===
  transitions: Transition[];
  /** 跳转规则列表 */

  // === 场景配置 ===
  requirements?: SceneRequirements;
  /** 场景进入条件 */

  // === 元数据 ===
  tags?: string[];
  /** 场景标签 */

  notes?: string;
  /** KP 备注 (仅 KP 可见) */

  estimated_duration?: string;
  /** 预计持续时间，如 "15-30min" */

  difficulty?: 'easy' | 'normal' | 'hard';
  /** 场景难度 */
}

interface NarrativeContent {
  opening: string;
  /** 开场叙事文本 (必填) */

  alternate?: string[];
  /** 变体叙事 (可选，KP 可选) */

  atmosphere?: string;
  /** 氛围描述 */

  sensory?: {
    /** 感官描述 */
    sight?: string;
    sound?: string;
    smell?: string;
    touch?: string;
  };
}
```

---

## Transitions 结构

```typescript
interface Transition {
  id: string;
  /** 跳转规则 ID */

  target: string;
  /** 目标场景 ID */

  condition: TransitionCondition;
  /** 触发条件 */

  description?: string;
  /** 跳转描述 (给玩家看) */

  auto_trigger?: boolean;
  /** 是否自动触发 */

  kp_only?: boolean;
  /** 是否仅 KP 可触发 */
}

interface TransitionCondition {
  type: 'manual' | 'automatic' | 'clue' | 'state' | 'choice';
  /** 条件类型 */

  requirements?: {
    /** 条件要求 */
    clues_found?: string[];
    /** 需要发现的线索 */
    state?: Record<string, any>;
    /** 状态要求 */
    choice?: string;
    /** 玩家选择 */
  };

  time_limit?: string;
  /** 时间限制 */
}
```

---

## SceneRequirements 结构

```typescript
interface SceneRequirements {
  // 线索要求
  required_clues?: string[];
  /** 必须发现的线索 */

  any_of_clues?: string[][];
  /** 满足任一组的线索 (OR 逻辑) */

  // 状态要求
  required_state?: Record<string, any>;
  /** 必须的游戏状态 */

  // 阻塞条件
  blocked_by?: {
    clues_not_found?: string[];
    /** 未发现这些线索时阻塞 */
    state?: Record<string, any>;
    /** 特定状态时阻塞 */
  };

  // 角色要求
  min_characters?: number;
  /** 最少角色数 */

  required_attributes?: {
    /** 属性要求 */
    attribute: string;
    min_value: number;
  }[];

  // 时间要求
  time_of_day?: 'morning' | 'afternoon' | 'evening' | 'night';
  /** 特定时间 */

  max_duration?: string;
  /** 最大停留时间 */
}
```

---

## Scenes 集合结构

```typescript
interface ScenesCollection {
  [sceneId: string]: Scene;
}

// 使用示例
const scenes: ScenesCollection = {
  "scene_001": {
    id: "scene_001",
    title: "图书馆初探",
    order: 1,
    narrative: {
      opening: "你们站在波士顿公共图书馆的门前...",
      alternate: [
        "如果是白天：阳光透过彩色玻璃窗洒入...",
        "如果是夜晚：图书馆已经关门，但你们注意到..."
      ],
      atmosphere: "知识的殿堂，也是秘密的藏身处",
      sensory: {
        sight: "高耸的书架，陈旧的木桌",
        sound: "翻书声，远处低语",
        smell: "旧纸张和灰尘的味道"
      }
    },
    npcs: ["npc_librarian"],
    locations: ["loc_library_main", "loc_library_archives"],
    clues: ["clue_old_newspaper"],
    handouts: [],
    transitions: [
      {
        id: "trans_001_002",
        target: "scene_002",
        condition: {
          type: "manual",
          requirements: {
            clues_found: ["clue_old_newspaper"]
          }
        },
        description: "前往档案室"
      }
    ],
    requirements: {
      min_characters: 1
    },
    tags: ["investigation", "intro"],
    estimated_duration: "15-20min"
  },

  "scene_002": {
    id: "scene_002",
    title: "档案馆的秘密",
    order: 2,
    narrative: {
      opening: "档案室里堆满了发黄的文件...",
    },
    // ... 更多场景
  }
}
```

---

## 场景跳转规则

```typescript
// 条件类型示例

// 1. 手动跳转 (玩家选择)
{
  type: "manual",
  description: "玩家主动选择下一步行动"
}

// 2. 线索触发
{
  type: "clue",
  requirements: {
    clues_found: ["clue_key"]
  },
  description: "找到钥匙后可以开锁"
}

// 3. 状态触发
{
  type: "state",
  requirements: {
    state: {
      "door_unlocked": true
    }
  },
  description: "门已解锁，可以进入"
}

// 4. 玩家选择
{
  type: "choice",
  requirements: {
    choice: "investigate_sound"
  },
  description: "选择调查声音的来源"
}

// 5. 自动触发
{
  type: "automatic",
  auto_trigger: true,
  description: "满足条件后自动跳转"
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `docs/specs/scenes.md` | 创建 | 场景结构规范 |
| `app/core/types/scene.ts` | 创建 | TypeScript 类型 |
| `app/services/scene_validator.py` | 创建 | 场景验证器 |

---

## 验收标准

- [ ] Scene 结构完整
- [ ] Narrative 定义清晰
- [ ] Transitions 规则明确
- [ ] Requirements 条件可验证
- [ ] 示例场景有效

---

## 参考文档

- M0-014: 场景包根结构
- M0-015: metadata 元信息

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
