# M0-014: 设计场景包根结构

**任务ID**: M0-014
**标题**: 设计场景包根结构
**类型**: spec (规范设计)
**预估工时**: 2h
**依赖**: 无

---

## 任务描述

定义 CoC 跑团场景包 (Scenario Package) 的根数据结构，这是整个场景包格式规范的基础。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M0-014-01 | 分析场景包需求 | 确定场景包需要包含的内容 | 20min |
| M0-014-02 | 设计根结构 schema | 定义顶层字段 | 20min |
| M0-014-03 | 设计 metadata 结构 | 元信息字段设计 | 20min |
| M0-014-04 | 设计 scenes 结构 | 场景集合结构 | 20min |
| M0-014-05 | 设计 shared 结构 | 共享资源结构 | 15min |
| M0-014-06 | 编写 TypeScript 类型定义 | 类型安全的数据结构 | 15min |
| M0-014-07 | 创建示例场景包 | 供参考的完整示例 | 15min |
| M0-014-08 | 编写结构说明文档 | 解释各字段用途 | 10min |

---

## 场景包根结构

```typescript
interface ScenarioPackage {
  // === 元信息 (必填) ===
  metadata: {
    id: string;                    // 唯一标识符
    title: string;                 // 脚本标题
    version: string;               // 版本号 (语义化版本)
    author: string;                // 作者名
    description: string;           // 简短描述 (1-2句话)
    duration: string;              // 预计时长 (如 "2-4h")
    player_count: string;          // 推荐人数 (如 "3-5")
    tags: string[];                // 标签 (如 ["入门", "现代", "恐怖"])
    language: string;              // 语言代码 (如 "zh-CN")
    created_at: string;            // ISO 8601 时间戳
    updated_at: string;            // ISO 8601 时间戳
    min_players?: number;          // 最少玩家数
    max_players?: number;          // 最多玩家数
    difficulty?: 'easy' | 'normal' | 'hard';  // 难度
    age_rating?: string;           // 年龄分级
  };

  // === 场景集合 ===
  scenes: Record<string, Scene>;

  // === 共享资源 ===
  shared: {
    npcs?: Record<string, NPC>;           // 共享 NPC
    locations?: Record<string, Location>;  // 共享地点
    clues?: Record<string, Clue>;          // 共享线索
    handouts?: Record<string, Handout>;    // 共享手递物
    items?: Record<string, Item>;          // 共享物品
  };

  // === 扩展字段 ===
  extensions?: Record<string, any>;  // 扩展数据
}
```

---

## 场景结构

```typescript
interface Scene {
  id: string;                  // 场景唯一 ID
  title: string;               // 场景标题
  order: number;               // 播放顺序

  // 叙事内容
  narrative: {
    opening: string;           // 开场叙事文本
    alternate?: string[];      // 变体叙事 (KP 可选)
  };

  // 场景元素引用
  npcs: string[];              // NPC 引用 (指向 shared.npcs)
  locations: string[];         // 地点引用
  clues: string[];             // 线索引用
  handouts: string[];          // 手递物引用

  // 状态转换
  transitions: Transition[];   // 跳转规则

  // 场景配置
  requirements?: {
    required_clues?: string[]; // 必需线索
    required_state?: Record<string, any>;
    blocked_by?: string[];     // 阻塞条件
  };

  // 元数据
  tags?: string[];
  notes?: string;              // KP 备注
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `docs/specs/script-schema.md` | 创建 | 场景包格式规范 |
| `docs/specs/types/scenario.ts` | 创建 | TypeScript 类型定义 |
| `examples/scenarios/minimal.json` | 创建 | 最小示例场景包 |

---

## 验收标准

- [ ] 根结构定义完整
- [ ] 字段命名清晰一致
- [ ] 必填/可选字段明确
- [ ] 有完整的类型定义
- [ ] 提供可运行的示例

---

## 参考文档

- JSON Schema 规范
- TypeScript 类型系统

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
