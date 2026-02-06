# M3-014: 设计 Summary 数据结构

**任务ID**: M3-014
**标题**: 设计 Summary 数据结构
**类型**: spec (规范设计)
**预估工时**: 2h
**依赖**: M0

---

## 任务描述

设计 Session 总结摘要的数据结构，用于生成结构化的游戏会话总结，包括叙事摘要、关键事件、状态变化等信息。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M3-014-01 | 分析总结需求 | 确定需要总结的内容 | 20min |
| M3-014-02 | 设计 SessionSummary | 主结构 | 25min |
| M3-014-03 | 设计 NarrativeSummary | 叙事摘要 | 20min |
| M3-014-04 | 设计 KeyEvents | 关键事件 | 15min |
| M3-014-05 | 设计 StateChanges | 状态变化 | 20min |
| M3-014-06 | 编写 TypeScript 类型 | 类型定义 | 20min |
| M3-014-07 | 编写示例总结 | 供参考的完整示例 | 15min |

---

## SessionSummary 结构

```typescript
interface SessionSummary {
  // === 基础信息 ===
  summary_id: string;
  session_id: string;
  created_at: datetime;
  updated_at: datetime;

  // === 会话信息 ===
  session_info: {
    started_at: datetime;
    ended_at?: datetime;
    duration_seconds?: number;
    scene_id: string;
    scene_title: string;
  };

  // === 叙事摘要 ===
  narrative_summary: {
    brief: string;              // 1-2 句话简述
    detailed: string;           // 2-3 段详细叙述
    mood: 'calm' | 'tense' | 'horror' | 'mystery' | 'action';
    tone: string;               // 描述性文字
  };

  // === 关键事件 ===
  key_events: KeyEvent[];

  // === 状态变化 ===
  state_changes: {
    characters: CharacterStateChange[];
    discoveries: Discovery[];
    consequences: Consequence[];
  };

  // === 线索和承诺 ===
  leads: {
    discovered: string[];       // 新发现的线索 ID
    resolved: string[];         // 已解决的线索 ID
    pending: string[];          // 待处理的线索 ID
  };

  promises: {
    description: string;
    source_event_id: string;
    status: 'pending' | 'fulfilled' | 'broken';
  }[];

  // === 统计 ===
  statistics: {
    message_count: number;
    roll_count: number;
    combat_count: number;
    san_check_count: number;
    injury_count: number;
    clue_discovery_count: number;
  };

  // === 可见性 ===
  visibility: {
    public: string[];           // 所有人可见的字段
    kp_only: string[];          // 仅 KP 可见的字段
    players: Record<string, string[]>;  // per-player visibility
  };
}
```

---

## KeyEvent 结构

```typescript
interface KeyEvent {
  event_id: string;
  timestamp: datetime;
  type: EventType;
  title: string;                // 事件标题
  description: string;          // 事件描述

  // 参与者
  participants: {
    user_id: string;
    character_id?: string;
    role: 'active' | 'passive' | 'witness';
  }[];

  // 结果
  outcome?: {
    success: boolean;
    description: string;
    consequences?: string[];
  };

  // 相关线索
  related_clues: string[];

  // 可见性
  visibility: 'public' | 'kp' | 'player:*';
}

type EventType =
  | 'clue_discovered'     // 发现线索
  | 'combat_occurred'     // 发生战斗
  | 'san_check_failed'    // SAN 检定失败
  | 'madness_triggered'   // 触发疯狂
  | 'character_injured'   // 角色受伤
  | 'character_died'      // 角色死亡
  | 'scene_transition'    // 场景转换
  | 'puzzle_solved'       // 解开谜题
  | 'mystery_revealed'    // 揭示真相
  | 'critical_failure';   // 关键失败
```

---

## CharacterStateChange 结构

```typescript
interface CharacterStateChange {
  character_id: string;
  character_name: string;

  // 数值变化
  changes: {
    hp: {
      old: number;
      new: number;
      delta: number;
    };
    san: {
      old: number;
      new: number;
      delta: number;
      events: string[];      // 导致变化的事件
    };
    luck: {
      old: number;
      new: number;
      delta: number;
    };
    mp?: {
      old: number;
      new: number;
      delta: number;
    };
  };

  // 状态变化
  status_changes: {
    old: CharacterStatus;
    new: CharacterStatus;
    reason: string;
  }[];

  // 技能变化
  skill_changes: {
    skill_id: string;
    old_value: number;
    new_value: number;
    reason: 'growth' | 'injury' | 'other';
  }[];

  // 物品变化
  inventory_changes: {
    added: string[];
    removed: string[];
    used: string[];
  };
}

type CharacterStatus =
  | 'healthy'
  | 'injured'
  | 'wounded'
  | 'critical'
  | 'unconscious'
  | 'dying'
  | 'dead'
  | 'insane'
  | 'temporary_madness'
  | 'indefinite_madness';
```

---

## Discovery 和 Consequence

```typescript
interface Discovery {
  discovery_id: string;
  timestamp: datetime;
  type: 'clue' | 'information' | 'item' | 'location' | 'npc_secret';

  content: {
    title: string;
    description: string;
    evidence?: string[];      // 支持证据的事件 ID
  };

  discoverer: {
    user_id: string;
    character_id?: string;
  };

  // 可见性
  visibility: 'public' | 'party' | 'private' | 'kp';
}

interface Consequence {
  consequence_id: string;
  timestamp: datetime;
  type: 'injury' | 'san_loss' | 'madness' | 'resource_loss' | 'story_branch';

  description: string;
  severity: 'minor' | 'moderate' | 'major' | 'critical';

  // 原因
  cause: {
    event_id: string;
    description: string;
  };

  // 影响范围
  affected: {
    characters: string[];     // 受影响的角色 ID
    party: boolean;           // 是否影响全队
  };

  // 当前状态
  status: 'active' | 'resolved' | 'ongoing';
}
```

---

## 摘要生成模板

```python
# app/services/summary.py
from typing import List, Dict

class SummaryGenerator:
    def generate_narrative_summary(
        self,
        events: List[GameEvent],
        context: Dict
    ) -> str:
        """生成叙事摘要"""
        # 1. 提取关键事件
        key_events = self._extract_key_events(events)

        # 2. 分析事件流程
        flow = self._analyze_event_flow(key_events)

        # 3. 生成摘要
        summary = {
            'brief': self._generate_brief(flow),
            'detailed': self._generate_detailed(flow),
            'mood': self._determine_mood(events),
            'tone': self._determine_tone(flow),
        }

        return summary

    def extract_state_changes(
        self,
        events: List[GameEvent]
    ) -> Dict[str, CharacterStateChange]:
        """提取状态变化"""
        changes = {}

        for event in events:
            if event.state_changes:
                for change in event.state_changes:
                    char_id = change.character_id
                    if char_id not in changes:
                        changes[char_id] = CharacterStateChange(
                            character_id=char_id,
                            changes={},
                            status_changes=[],
                            skill_changes=[],
                            inventory_changes={}
                        )
                    # 应用变化
                    changes[char_id].apply(change)

        return changes
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `docs/specs/summary.md` | 创建 | 摘要格式规范 |
| `app/core/types/summary.ts` | 创建 | TypeScript 类型 |
| `app/services/summary.py` | 创建 | 摘要生成服务 |
| `app/db/models/summary.py` | 创建 | 数据模型 |

---

## 验收标准

- [ ] SessionSummary 结构完整
- [ ] KeyEvent 类型定义清晰
- [ ] StateChange 结构可追溯
- [ ] 可见性控制正确
- [ ] 示例数据有效

---

## 参考文档

- M0-038: 事件日志结构
- M3-015: 检查点摘要生成器

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
