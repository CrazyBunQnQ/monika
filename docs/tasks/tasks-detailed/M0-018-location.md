# M0-018: 定义 Location 地点结构

**任务ID**: M0-018
**标题**: 定义 Location 地点结构
**类型**: spec (规范设计)
**预估工时**: 2h
**依赖**: M0-016

---

## 任务描述

定义 Location (地点) 的数据结构，用于描述场景中的各种场所。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M0-018-01 | 设计 Location 基础结构 | 基本信息 | 25min |
| M0-018-02 | 设计 Location 描述系统 | 多层描述 | 25min |
| M0-018-03 | 设计 Location 连接系统 | 地点连接 | 20min |
| M0-018-04 | 设计 Location 状态系统 | 地点状态 | 20min |
| M0-018-05 | 设计 Location 交互系统 | 可交互元素 | 20min |
| M0-018-06 | 编写 TypeScript 类型 | 类型定义 | 20min |
| M0-018-07 | 编写示例地点 | 示例数据 | 15min |

---

## Location 结构

```typescript
interface Location {
  // === 标识 ===
  id: string;
  /** 地点唯一标识符 */

  name: string;
  /** 地点名称 */

  type: LocationType;
  /** 地点类型 */

  // === 描述 ===
  description: LocationDescription;

  // === 连接 ===
  connections?: LocationConnection[];
  /** 与其他地点的连接 */

  // === 特征 ===
  features?: LocationFeature[];
  /** 地点特征 */

  // === 交互元素 ===
  interactables?: LocationInteractable[];
  /** 可交互元素 */

  // === 状态 ===
  state?: LocationState;

  // === 条件 ===
  access_requirements?: LocationAccessRequirements;
  /** 进入条件 */

  // === 元数据 ===
  is_private?: boolean;
  /** 是否私密场所 */

  is_dangerous?: boolean;
  /** 是否危险场所 */

  ambient_effects?: string[];
  /** 环境效果，如 "嘈杂", "昏暗", "寒冷" */

  // === 扩展 ===
  custom_data?: Record<string, any>;
}

type LocationType =
  | 'building'          // 建筑
  | 'room'              // 房间
  | 'outdoor'           // 户外
  | 'street'            // 街道
  | 'vehicle'           // 载具
  | 'natural'           // 自然环境
  | 'underground'       // 地下
  | 'water'             // 水域
  | 'other';            // 其他
```

---

## LocationDescription 结构

```typescript
interface LocationDescription {
  brief: string;
  /** 简短描述 (1-2句话) */

  detailed: string;
  /** 详细描述 (1-2段) */

  sensory?: {
    /** 感官描述 */
    sight?: string;
    sound?: string;
    smell?: string;
    touch?: string;
    taste?: string;
  };

  atmosphere?: string;
  /** 氛围描述 */

  condition?: string;
  /** 当前状态，如 "破败", "维护良好", "杂乱" */

  lighting?: lighting;
  /** 光照条件 */

  size?: string;
  /** 尺寸，如 "小型 (10x10英尺)", "大型大厅" */

  capacity?: number;
  /** 容纳人数 */
}

type lighting =
  | 'bright'            // 明亮
  | 'normal'            // 正常
  | 'dim'               // 昏暗
  | 'dark'              // 黑暗
  | 'pitch_black';      // 漆黑
```

---

## LocationConnection 结构

```typescript
interface LocationConnection {
  target: string;
  /** 目标地点 ID */

  type: ConnectionType;
  /** 连接类型 */

  description: string;
  /** 连接描述 */

  distance?: string;
  /** 距离，如 "50英尺", "5分钟路程" */

  direction?: string;
  /** 方向，如 "北", "楼上" */

  accessibility?: Accessibility;
  /** 可达性 */

  is_two_way?: boolean;
  /** 是否双向通行 */

  locked?: boolean;
  /** 是否锁定/阻塞 */

  key_required?: string;
  /** 需要的钥匙/物品 */

  conditions?: string[];
  /** 通过条件 */
}

type ConnectionType =
  | 'door'              // 门
  | 'passage'           // 通道
  | 'stairs'           // 楼梯
  | 'elevator'         // 电梯
  | 'window'           // 窗户
  | 'gate'             // 大门
  | 'path'             // 小路
  | 'open'             // 开放空间;

type Accessibility =
  | 'easy'              // 容易通过
  | 'normal'            // 正常
  | 'difficult'         // 困难 (需要检定)
  | 'blocked'           // 阻塞
  | 'hidden';           // 隐藏通道
```

---

## LocationFeature 结构

```typescript
interface LocationFeature {
  id: string;
  /** 特征 ID */

  name: string;
  /** 特征名称 */

  type: FeatureType;
  /** 特征类型 */

  description: string;
  /** 描述 */

  is_interactive: boolean;
  /** 是否可交互 */

  is_hidden?: boolean;
  /** 是否隐藏 */

  search_difficulty?: 'easy' | 'normal' | 'hard';
  /** 搜索难度 */

  contains?: string[];
  /** 包含的物品/线索 ID */
}

type FeatureType =
  | 'furniture'         // 家具
  | 'decoration'        // 装饰
  | 'fixture'           // 固定设施
  | 'object'           // 可移动物品
  | 'hiding_spot'      // 藏匿处
  | 'clue_source'      // 线索来源
  | 'hazard'           // 危险源
  | 'resource';        // 资源
```

---

## LocationInteractable 结构

```typescript
interface LocationInteractable {
  id: string;
  /** 交互元素 ID */

  name: string;
  /** 名称 */

  type: InteractableType;
  /** 类型 */

  description: string;
  /** 描述 */

  interactions?: Interaction[];
  /** 可用交互 */

  state?: Record<string, any>;
  /** 当前状态 */
}

type InteractableType =
  | 'container'         // 容器 (可打开)
  | 'door'             // 门
  | 'switch'           // 开关
  | 'device'           // 设备
  | 'furniture'        // 家具
  | 'bookshelf'        // 书架
  | 'safe'             // 保险箱
  | 'body'             // 尸体;

interface Interaction {
  action: string;
  /** 交互动作，如 "打开", "搜索", "检查" */

  requirements?: {
    /** 要求 */
    items?: string[];
    skills?: { skill: string; value: number };
    flags?: string[];
  };

  results?: {
    /** 结果 */
    success?: string;
    failure?: string;
    items?: string[];
    clues?: string[];
    damage?: string;
  };
}
```

---

## LocationState 结构

```typescript
interface LocationState {
  // === 时间相关 ===
  time_of_day?: 'morning' | 'afternoon' | 'evening' | 'night';
  /** 时间段 */

  weather?: string;
  /** 天气 (如果是户外) */

  // === 环境状态 ===
  lighting?: lighting;
  /** 当前光照 */

  noise_level?: 'quiet' | 'normal' | 'loud';
  /** 噪音等级 */

  crowded?: boolean;
  /** 是否拥挤 */

  // === 特殊状态 ===
  conditions?: string[];
  /** 特殊条件，如 ["停电", "维修中", "警戒中"] */

  // === 事件 ===
  ongoing_events?: string[];
  /** 正在发生的事件 */

  // === 访问记录 ===
  visited_by?: string[];
  /** 已访问的角色 ID */

  visit_count?: number;
  /** 访问次数 */
}
```

---

## LocationAccessRequirements 结构

```typescript
interface LocationAccessRequirements {
  // === 时间限制 ===
  time_restrictions?: {
    /** 时间限制 */
    allowed_times?: string[];
    /** 允许的时间段 */
    forbidden_times?: string[];
    /** 禁止的时间段 */
  };

  // === 条件限制 ===
  requires_clues?: string[];
  /** 需要发现的线索 */

  requires_flags?: string[];
  /** 需要的状态标志 */

  requires_items?: string[];
  /** 需要的物品 */

  requires_skills?: {
    skill: string;
    value: number;
  }[];
  /** 需要的技能 */

  // === 权限限制 ===
  requires_permission?: {
    /** 权限要求 */
    from_character?: string;
    /** 需要特定角色许可 */

    from_role?: 'kp' | 'player';
    /** 需要特定角色权限 */
  };

  // === 检定限制 ===
  requires_check?: {
    /** 需要通过检定 */
    skill: string;
    difficulty: 'regular' | 'hard' | 'extreme';
    description: string;
  };

  // === 特殊条件 ===
  special_conditions?: string[];
  /** 特殊条件描述 */
}
```

---

## 示例 Location

```json
{
  "id": "loc_library_main_hall",
  "name": "图书馆主大厅",
  "type": "room",

  "description": {
    "brief": "图书馆的主大厅，高挑的天花板和古老的阅读桌",
    "detailed": "踏入图书馆，首先看到的是高挑的天花板和悬挂的吊灯。阳光透过彩色玻璃窗洒入，在地板上投下斑驳的光影。多张古老的橡木阅读桌整齐排列，空气中弥漫着旧纸张和灰尘的味道。",
    "sensory": {
      "sight": "彩色玻璃窗，高天花板，橡木家具",
      "sound": "翻书声，远处的低语",
      "smell": "旧纸张，灰尘，淡淡的书香",
      "touch": "光滑的木质桌面，冰冷的金属扶手"
    },
    "atmosphere": "知识的殿堂，也是秘密的藏身处",
    "lighting": "normal",
    "size": "大型 (50x100英尺)",
    "capacity": 50
  },

  "connections": [
    {
      "target": "loc_library_archives",
      "type": "door",
      "description": "一扇标有'档案室'的木门",
      "direction": "东",
      "is_two_way": true,
      "locked": true,
      "key_required": "key_archives",
      "accessibility": "normal"
    },
    {
      "target": "loc_library_basement",
      "type": "stairs",
      "description": "通往地下室的楼梯",
      "direction": "下",
      "is_two_way": true,
      "accessibility": "difficult"
    }
  ],

  "features": [
    {
      "id": "feature_reception_desk",
      "name": "接待桌",
      "type": "furniture",
      "description": "一张古老的橡木接待桌，后面坐着一个管理员",
      "is_interactive": true
    },
    {
      "id": "feature_notice_board",
      "name": "公告板",
      "type": "decoration",
      "description": "墙上贴着各种通知和海报",
      "is_interactive": true,
      "search_difficulty": "easy",
      "contains": ["clue_old_notice"]
    }
  ],

  "interactables": [
    {
      "id": "interactable_reception_desk",
      "name": "接待桌",
      "type": "furniture",
      "description": "可以询问管理员",
      "interactions": [
        {
          "action": "talk",
          "results": {
            "success": "管理员可以回答一些问题",
            "clues": ["clue_librarian_info"]
          }
        },
        {
          "action": "search",
          "requirements": {
            "skills": { "skill": "侦查", "value": 50 }
          },
          "results": {
            "success": "你发现桌上的访客记录",
            "clues": ["clue_visitor_log"],
            "failure": "你什么也没发现"
          }
        }
      ]
    }
  ],

  "state": {
    "lighting": "normal",
    "noise_level": "quiet",
    "crowded": false,
    "conditions": ["开放时间"],
    "visited_by": [],
    "visit_count": 0
  },

  "access_requirements": {
    "time_restrictions": {
      "allowed_times": ["09:00-17:00"],
      "forbidden_times": ["22:00-06:00"]
    }
  },

  "is_private": false,
  "is_dangerous": false,
  "ambient_effects": ["安静", "学术氛围"]
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `docs/specs/location.md` | 创建 | 地点结构规范 |
| `app/core/types/location.ts` | 创建 | TypeScript 类型 |

---

## 验收标准

- [ ] Location 结构完整
- [ ] 连接系统清晰
- [ ] 交互元素可用
- [ ] 状态系统合理
- [ ] 示例地点有效

---

## 参考文档

- M0-016: scenes 场景集合结构

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
