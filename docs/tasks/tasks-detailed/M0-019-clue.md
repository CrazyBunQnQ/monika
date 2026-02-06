# M0-019: 定义 Clue 线索数据结构

**任务ID**: M0-019
**标题**: 定义 Clue 线索数据结构
**类型**: spec (规范设计)
**预估工时**: 2h
**依赖**: M0-016

---

## 任务描述

定义 Clue (线索) 的数据结构，线索是玩家在调查过程中发现的重要信息。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M0-019-01 | 设计 Clue 基础结构 | 基本信息 | 25min |
| M0-019-02 | 设计 Clue 分类系统 | 线索类型 | 20min |
| M0-019-03 | 设计 Clue 重要性系统 | 优先级 | 15min |
| M0-019-04 | 设计 Clue 关联系统 | 线索关联 | 25min |
| M0-019-05 | 设计 Clue 揭示机制 | 分层揭示 | 25min |
| M0-019-06 | 编写 TypeScript 类型 | 类型定义 | 20min |
| M0-019-07 | 编写示例线索 | 示例数据 | 15min |

---

## Clue 结构

```typescript
interface Clue {
  // === 标识 ===
  id: string;
  /** 线索唯一标识符 */

  title: string;
  /** 线索标题/名称 */

  type: ClueType;
  /** 线索类型 */

  category: ClueCategory;
  /** 线索分类 */

  // === 内容 ===
  content: ClueContent;
  /** 线索内容 */

  // === 重要性 ===
  importance: ClueImportance;
  /** 重要程度 */

  priority: number;
  /** 优先级 (0-100) */

  // === 可见性 ===
  visibility: ClueVisibility;
  /** 可见性控制 */

  // === 来源 ===
  source: ClueSource;
  /** 线索来源 */

  // === 关联 ===
  related_to?: ClueRelations;
  /** 关联信息 */

  // === 揭示 ===
  revelation?: ClueRevelation;
  /** 揭示机制 */

  // === 元数据 ===
  tags?: string[];
  /** 标签 */

  discovered_by?: string[];
  /** 已发现的玩家 ID */

  discovery_count?: number;
  /** 发现次数 */

  // === KP 信息 ===
  notes?: string;
  /** KP 备注 (仅 KP 可见) */

  full_truth?: string;
  /** 完整真相 (仅 KP 可见) */

  // === 扩展 ===
  custom_data?: Record<string, any>;
}

type ClueType =
  | 'information'       // 信息类线索
  | 'physical'          // 物理线索 (物品)
  | 'testimony'         // 证词
  | 'document'          // 文档
  | 'observation'       // 观察结果
  | 'deduction'         // 推理结果
  | 'pattern';          // 模式/规律

type ClueCategory =
  | 'background'        // 背景信息
  | 'plot'              // 剧情相关
  | 'character'         // 角色相关
  | 'location'          // 地点相关
  | 'event'             // 事件相关
  | 'mechanism'         // 机制/方法
  | 'mystery'           // 谜团核心
  | 'red_herring';      // 红鲱 (误导)

type ClueImportance =
  | 'trivial'           // 琐碎 (1)
  | 'minor'             // 次要 (2)
  | 'moderate'          // 中等 (3)
  | 'major'             // 重要 (4)
  | 'critical';         // 关键 (5)
```

---

## ClueContent 结构

```typescript
interface ClueContent {
  brief: string;
  /** 简短描述 (1句话) */

  detailed?: string;
  /** 详细描述 (1-2段) */

  exact_words?: string;
  /** 精确措辞 (如文档内容) */

  visual?: string;
  /** 视觉描述 */

  sensory?: {
    /** 感官信息 */
    sound?: string;
    smell?: string;
    touch?: string;
  };

  context?: string;
  /** 上下文信息 */

  implications?: string[];
  /** 可能的含义/推论 */

  questions?: string[];
  /** 引发的问题 */
}
```

---

## ClueVisibility 结构

```typescript
interface ClueVisibility {
  default: VisibilityLevel;
  /** 默认可见性 */

  conditions?: ClueVisibilityCondition[];
  /** 可见性条件 */

  reveal_triggers?: ClueRevealTrigger[];
  /** 揭示触发器 */

  auto_reveal?: {
    /** 自动揭示 */
    after_clues?: string[];
    /** 发现这些线索后自动揭示 */

    after_event?: string;
    /** 特定事件后揭示 */

    time_delay?: string;
    /** 时间延迟 */
  };
}

type VisibilityLevel =
  | 'public'            // 所有人可见
  | 'party'             // 队伍可见
  | 'finder_only'       // 仅发现者可见
  | 'kp_reveal'         // KP 选择揭示
  | 'conditional'       // 条件可见
  | 'hidden';           // 隐藏 (需要特殊条件)
```

---

## ClueSource 结构

```typescript
interface ClueSource {
  type: ClueSourceType;
  /** 来源类型 */

  location?: string;
  /** 地点 ID */

  npc?: string;
  /** NPC ID */

  item?: string;
  /** 物品 ID */

  feature?: string;
  /** 特征 ID */

  event?: string;
  /** 事件 ID */

  method?: string;
  /** 发现方法 */

  difficulty?: 'auto' | 'easy' | 'normal' | 'hard' | 'extreme';
  /** 发现难度 */

  description?: string;
  /** 来源描述 */
}

type ClueSourceType =
  | 'location'           // 来自地点
  | 'npc'               // 来自 NPC
  | 'item'              // 来自物品
  | 'document'          // 来自文档
  | 'observation'       // 来自观察
  | 'investigation'     // 来自调查
  | 'interaction'       // 来自交互
  | 'dream'             // 来自梦境
  | 'flashback';        // 来自闪回
```

---

## ClueRelations 结构

```typescript
interface ClueRelations {
  leads_to?: string[];
  /** 指向的其他线索 ID */

  contradicts?: string[];
  /** 矛盾的线索 ID */

  supports?: string[];
  /** 支持的线索 ID */

  requires?: string[];
  /** 需要先发现的线索 */

  related_npcs?: string[];
  /** 相关 NPC */

  related_locations?: string[];
  /** 相关地点 */

  related_events?: string[];
  /** 相关事件 */

  timeline?: ClueTimelineEntry[];
  /** 时间线索引 */
}

interface ClueTimelineEntry {
  date: string;
  /** 日期 */

  event: string;
  /** 事件描述 */

  importance?: 'major' | 'minor';
  /** 重要程度 */
}
```

---

## ClueRevelation 结构

```typescript
interface ClueRevelation {
  layers: ClueLayer[];
  /** 分层信息 */

  current_layer?: number;
  /** 当前揭示层级 */
}

interface ClueLayer {
  layer: number;
  /** 层级 (1, 2, 3...) */

  content: string;
  /** 该层内容 */

  trigger?: ClueLayerTrigger;
  /** 揭示触发条件 */

  is_full_truth?: boolean;
  /** 是否是完整真相 */
}

interface ClueLayerTrigger {
  type: 'auto' | 'conditional' | 'manual';
  /** 触发类型 */

  conditions?: {
    /** 条件触发 */
    clues_found?: string[];
    /** 需要发现的线索 */

    skill_check?: {
      /** 技能检定 */
      skill: string;
      difficulty: string;
      threshold: number;
    };

    time_passed?: string;
    /** 时间流逝 */

    event?: string;
    /** 事件触发 */
  };

  description?: string;
  /** 触发描述 */
}
```

---

## 示例 Clue

```json
{
  "id": "clue_old_newspaper",
  "title": "1923年的旧报纸",
  "type": "document",
  "category": "background",

  "content": {
    "brief": "一张1923年的旧报纸，报道了本地失踪案",
    "detailed": "这是一张保存完好的《波士顿环球报》，日期是1923年6月15日。头版报道了一则新闻：《第三名市民神秘失踪》。文章提到了警方对连环失踪案的无力，以及市民的恐慌情绪。",
    "exact_words": "警方发言人表示：'我们正在尽一切努力调查此案，但目前还没有实质性进展。'",
    "context": "夹在图书馆档案室的档案中"
  },

  "importance": "moderate",
  "priority": 50,

  "visibility": {
    "default": "public",
    "reveal_triggers": [
      {
        "layer": 2,
        "trigger": {
          "type": "conditional",
          "conditions": {
            "skill_check": {
              "skill": "图书馆使用",
              "difficulty": "hard",
              "threshold": 65
            }
          }
        }
      }
    ]
  },

  "source": {
    "type": "location",
    "location": "loc_library_archives",
    "feature": "feature_archive_cabinet",
    "method": "search",
    "difficulty": "normal",
    "description": "在档案柜的旧报纸中发现"
  },

  "related_to": {
    "leads_to": ["clue_victim_pattern", "clue_library_connection"],
    "related_npcs": ["npc_librarian_ms_higgins"],
    "related_events": ["event_1923_disappearance"],
    "timeline": [
      {
        "date": "1923-06-15",
        "event": "报纸报道失踪案",
        "importance": "major"
      }
    ]
  },

  "revelation": {
    "layers": [
      {
        "layer": 1,
        "content": "报道了失踪案的基本信息",
        "is_full_truth": false
      },
      {
        "layer": 2,
        "content": "文章暗示警方在掩盖某些信息，报纸上的报道可能被审查过",
        "trigger": {
          "type": "conditional",
          "conditions": {
            "skill_check": {
              "skill": "图书馆使用",
              "difficulty": "hard",
              "threshold": 65
            }
          }
        },
        "is_full_truth": false
      },
      {
        "layer": 3,
        "content": "完整真相：希金斯女士的哥哥是受害者之一，她保留了关键证据",
        "is_full_truth": true
      }
    ]
  },

  "tags": ["document", "背景", "案件相关"],
  "notes": "这是一个关键线索，连接了多个角色和事件",

  "full_truth": "这张报纸是希金斯女士故意留下的，她知道哥哥失踪的真相，但一直不敢公开。报纸上有一个她用铅笔做的标记，指向一个关键日期。"
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `docs/specs/clue.md` | 创建 | 线索结构规范 |
| `app/core/types/clue.ts` | 创建 | TypeScript 类型 |

---

## 验收标准

- [ ] Clue 结构完整
- [ ] 分类系统清晰
- [ ] 关联系统合理
- [ ] 揭示机制可用
- [ ] 示例线索有效

---

## 参考文档

- M0-016: scenes 场景集合结构
- M0-018: Location 地点结构

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
