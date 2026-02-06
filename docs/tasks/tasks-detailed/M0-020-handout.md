# M0-020: 定义 Handout 手递物格式

**任务ID**: M0-020
**标题**: 定义 Handout 手递物格式
**类型**: spec (规范设计)
**预估工时**: 2h
**依赖**: M0-016

---

## 任务描述

定义 Handout (手递物) 的数据结构，手递物是 KP 分发给玩家的信息卡片。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M0-020-01 | 设计 Handout 基础结构 | 基本信息 | 25min |
| M0-020-02 | 设计 Handout 内容系统 | 多种内容类型 | 30min |
| M0-020-03 | 设计 Handout 分发规则 | 分发时机 | 20min |
| M0-020-04 | 设计 Handout 可见性 | 谁可以看到 | 20min |
| M0-020-05 | 编写 TypeScript 类型 | 类型定义 | 20min |
| M0-020-06 | 编写示例手递物 | 示例数据 | 20min |

---

## Handout 结构

```typescript
interface Handout {
  // === 标识 ===
  id: string;
  /** 手递物唯一标识符 */

  title: string;
  /** 标题 */

  type: HandoutType;
  /** 手递物类型 */

  // === 内容 ===
  content: HandoutContent;
  /** 内容 */

  // === 分发 ===
  distribution: HandoutDistribution;
  /** 分发配置 */

  // === 可见性 ===
  visibility: HandoutVisibility;
  /** 可见性 */

  // === 元数据 ===
  tags?: string[];
  /** 标签 */

  is_private?: boolean;
  /** 是否私密 (仅接收者可见) */

  // === KP 信息 ===
  notes?: string;
  /** KP 备注 */

  // === 扩展 ===
  custom_data?: Record<string, any>;
}

type HandoutType =
  | 'image'             // 图片
  | 'text'              // 文本
  | 'document'          // 文档
  | 'map'               // 地图
  | 'evidence'          // 证据
  | 'note'              // 笔记
  | 'photo'             // 照片
  | 'letter'            // 信件
  | 'diary'             // 日记
  | 'newspaper'         // 报纸
  | 'blueprint';        // 蓝图

type HandoutContentType =
  | 'url'               // URL 引用
  | 'base64'            // Base64 编码
  | 'text'              // 纯文本
  | 'markdown'          // Markdown
  | 'html';             // HTML
```

---

## HandoutContent 结构

```typescript
interface HandoutContent {
  format: HandoutContentType;

  // === URL 引用 ===
  url?: string;
  /** 外部 URL */

  // === 图片/文件 ===
  data?: string;
  /** Base64 或 URL */

  filename?: string;
  /** 文件名 */

  mime_type?: string;
  /** MIME 类型 */

  // === 文本内容 ===
  text?: string;
  /** 文本内容 */

  markdown?: string;
  /** Markdown 内容 */

  // === 元数据 ===
  metadata?: {
    width?: number;
    height?: number;
    pages?: number;
    size?: number;
    author?: string;
    date?: string;
  };

  // === 描述 ===
  description?: string;
  /** 内容描述 */

  caption?: string;
  /** 说明文字 */
}
```

---

## HandoutDistribution 结构

```typescript
interface HandoutDistribution {
  // === 分发方式 ===
  method: DistributionMethod;
  /** 分发方法 */

  // === 分发条件 ===
  trigger?: DistributionTrigger;
  /** 分发触发器 */

  // === 接收者 ===
  recipients?: string[];
  /** 特定接收者 (user_id) */

  recipient_role?: 'all' | 'players' | 'specific';
  /** 接收者角色 */

  // === 分发时机 ===
  timing?: DistributionTiming;
  /** 分发时机 */
}

type DistributionMethod =
  | 'manual'            // KP 手动分发
  | 'automatic'         // 自动分发
  | 'on_request'        // 玩家请求
  | 'on_event'          // 事件触发
  | 'on_clue_found';    // 发现线索

interface DistributionTrigger {
  type: 'event' | 'clue' | 'state' | 'time' | 'location';

  event_id?: string;
  /** 触发事件 ID */

  clue_id?: string;
  /** 触发线索 ID */

  state?: Record<string, any>;
  /** 状态条件 */

  location_id?: string;
  /** 地点条件 */

  delay?: string;
  /** 延迟时间 */
}

interface DistributionTiming {
  at_event?: string;
  /** 特定事件时 */

  after_clue?: string;
  /** 发现线索后 */

  at_location?: string;
  /** 到达地点时 */

  time_passed?: string;
  /** 时间流逝后 */

  manual?: boolean;
  /** 仅手动 */
}
```

---

## HandoutVisibility 结构

```typescript
interface HandoutVisibility {
  // === 基础可见性 ===
  base: 'public' | 'private' | 'selective';
  /** 基础可见性 */

  // === 可见者 ===
  visible_to?: {
    /** 可见者配置 */
    users?: string[];
    /** 特定用户 */

    role?: 'all' | 'players' | 'kp';
    /** 角色过滤 */
  };

  // === 例外 ===
  exceptions?: {
    /** 可见性例外 */
    hide_from?: string[];
    /** 对这些人隐藏 */

    show_to?: string[];
    /** 对这些人显示 */
  };

  // === 条件可见 ===
  conditional?: {
    /** 条件可见 */
    condition: string;
    /** 条件表达式 */

    true_result: 'show' | 'hide';
    /** 满足条件时 */

    false_result: 'show' | 'hide';
    /** 不满足时 */
  };

  // === 时间限制 ===
  time_limit?: {
    /** 时间限制 */
    expires_after?: string;
    /** 多久后过期 */

    expires_at?: string;
    /** 过期时间点 */
  };

  // === 可操作 ===
  can_share?: boolean;
  /** 接收者是否可以分享给他人 */

  can_copy?: boolean;
  /** 是否可以复制 */
}
```

---

## 示例 Handout

```json
{
  "id": "handout_victim_photo",
  "title": "受害者的照片",
  "type": "image",

  "content": {
    "format": "url",
    "url": "https://example.com/images/victim_photo.jpg",
    "filename": "victim_photo.jpg",
    "mime_type": "image/jpeg",
    "metadata": {
      "width": 800,
      "height": 600,
      "date": "1923-06-10",
      "author": "未知摄影师"
    },
    "description": "一张黑白照片，显示了一名年轻男子",
    "caption": "这是最后一张失踪者约翰·史密斯的照片"
  },

  "distribution": {
    "method": "on_clue_found",
    "trigger": {
      "type": "clue",
      "clue_id": "clue_victim_identity"
    },
    "recipient_role": "all",
    "timing": {
      "after_clue": "clue_victim_identity",
      "delay": "immediate"
    }
  },

  "visibility": {
    "base": "public",
    "can_share": true,
    "can_copy": true
  },

  "tags": ["证据", "受害者", "案件相关"],

  "notes": "KP 可以在玩家确认受害者身份后分发此照片"
}

// 文档类型手递物示例
{
  "id": "handout_diary_page",
  "title": "撕下的日记页",
  "type": "document",

  "content": {
    "format": "markdown",
    "text": "# 日记\n\n**1923年6月10日**\n\n我发现了地下室里隐藏的东西。我不敢相信这是真的。如果其他人知道...我必须把这些记录下来，以防万一。\n\n他们正在监视我。我知道他们是谁。\n\n---\n\n(日记在这里中断，后面有几页被撕掉了)",
    "metadata": {
      "pages": 1,
      "author": "未知",
      "date": "1923-06-10"
    },
    "description": "一张发黄的日记页，字迹潦草"
  },

  "distribution": {
    "method": "on_event",
    "trigger": {
      "type": "event",
      "event_id": "event_found_diary"
    },
    "recipient_role": "players"
  },

  "visibility": {
    "base": "private",
    "visible_to": {
      "users": ["player_detective"],
      "role": "kp"
    },
    "can_share": true,
    "can_copy": false
  }
}
```

---

## Handout 管理操作

```typescript
// 手递物服务接口
interface HandoutService {
  // 分发手递物
  distribute(handoutId: string, options: DistributionOptions): Promise<void>;

  // 撤回手递物
  revoke(handoutId: string, reason: string): Promise<void>;

  // 查询可见的手递物
  listVisible(userId: string): Promise<Handout[]>;

  // 获取手递物内容
  get(handoutId: string, userId: string): Promise<HandoutContent>;

  // 分享手递物
  share(handoutId: string, fromUserId: string, toUserIds: string[]): Promise<void>;
}

interface DistributionOptions {
  recipients?: string[];
  timing?: 'immediate' | 'delayed';
  delay?: string;
  message?: string;
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `docs/specs/handout.md` | 创建 | 手递物规范 |
| `app/core/types/handout.ts` | 创建 | TypeScript 类型 |
| `app/services/handout.py` | 创建 | 手递物服务 |

---

## 验收标准

- [ ] Handout 结构完整
- [ ] 支持多种内容类型
- [ ] 分发规则清晰
- [ ] 可见性控制正确
- [ ] 示例手递物有效

---

## 参考文档

- M0-016: scenes 场景集合结构
- M0-019: Clue 线索结构

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
