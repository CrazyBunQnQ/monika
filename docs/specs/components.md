# 组件库规划

**版本**: v1.0
**最后更新**: 2026-02-07
**状态**: ✅ 设计完成

---

## 概述

本文档定义 CoC 跑团平台的组件库规划，采用 **shadcn/ui 作为基础组件库**，游戏专用组件基于 shadcn/ui 扩展封装。

**设计原则**:
- **全面采用 shadcn/ui**: 作为 UI 基础，保证一致性和开发效率
- **扩展封装游戏组件**: 基于 shadcn/ui 的 Card、Button 等组件封装
- **按里程碑分批开发**: 配合开发进度，优先实现核心组件
- **统一命名规范**: 遵循 React 和 TypeScript 最佳实践

---

## shadcn/ui 组件选择

### 全部引入的组件

```typescript
// 基础组件
Button, Input, Textarea, Select, Checkbox, Switch, Slider, Label, Badge

// 布局组件
Card, CardHeader, CardContent, CardFooter, Separator, Divider, Spacer

// 导航组件
Tabs, TabList, Tab, Breadcrumb, Menu, DropdownMenu, ContextMenu

// 反馈组件
Alert, Toast, Progress, Skeleton, Spinner

// 覆盖层组件
Dialog, Sheet, Popover, Tooltip, HoverCard

// 数据展示组件
Table, List, ListItem, Accordion, Collapsible

// 表单组件
Form, FormField, FormMessage
```

### 主题定制

```typescript
// tailwind.config.js
export default {
  darkMode: ["class"],
  theme: {
    extend: {
      colors: {
        // KP 主题色（紫色系）
        primary: {
          DEFAULT: "#7c3aed",
          hover: "#6d28d9",
          light: "#a78bfa",
        },

        // 玩家主题色（青绿色系）
        "player-primary": {
          DEFAULT: "var(--player-primary)",
        },

        // 强调色（统一）
        accent: "#f59e0b",

        // 背景色
        background: "var(--background)",
        surface: "var(--surface)",
        "surface-hover": "var(--surface-hover)",

        // 文字色
        text: "var(--text)",
        "text-secondary": "var(--text-secondary)",
        textMuted: "var(--text-muted)",
      },

      fontFamily: {
        sans: ["Inter", "sans-serif"],
        serif: ["Cormorant Garamond", "serif"],
        display: ["UnifrakturMaguntia", "cursive"],
        mono: ["JetBrains Mono", "monospace"],
      },

      borderRadius: {
        lg: "12px",
        xl: "16px",
      },

      boxShadow: {
        glow: "0 0 20px rgba(124, 58, 237, 0.3)",
      },

      animation: {
        "dice-roll": "diceRoll 1s cubic-bezier(0.68, -0.55, 0.265, 1.55)",
        "pulse-slow": "pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite",
      },
    },
  },
};
```

---

## 游戏专用组件

### 组件开发顺序

| 里程碑 | 组件 | 基于组件 | 说明 |
|--------|------|----------|------|
| M1 | MessageBubble | Card | 消息气泡 |
| M1 | DiceRoll | Card + Badge | 骰子结果 |
| M1 | StatePanel | Card + Progress | 状态面板 |
| M1 | ChatInput | Input + Button | 聊天输入 |
| M2 | SpotlightIndicator | Badge | 聚光灯指示器 |
| M2 | QueueList | List + Badge | 发言队列 |
| M2 | CombatTracker | Card + Table | 战斗追踪 |
| M2 | ChaseTracker | Card + Progress | 追逐追踪 |
| M3 | ClueLedger | Card + Collapsible | 线索账本 |
| M3 | RecapPanel | Card + Timeline | 复盘面板 |
| M4 | SceneGallery | Card + Grid | 场景画廊 |
| M5 | SANMeter | Progress + Badge | 理智值条 |
| M5 | MadnessPanel | Card + Alert | 疯狂面板 |

### M1 核心组件

#### MessageBubble

```typescript
interface MessageBubbleProps {
  role: "KP" | "Player" | "NPC" | "System";
  characterName: string;
  avatar?: string;
  content: string;
  timestamp: string;
  isVisibleTo: Visibility;
}

// 基于 Card 组件扩展
<MessageBubble
  role="Player"
  characterName="玩家A"
  content="你想做什么？"
  timestamp="10:30"
  isVisibleTo="public"
/>
```

#### DiceRoll

```typescript
interface DiceRollProps {
  skill: string;
  skillValue: number;
  roll: number;
  difficulty: string;
  successLevel: SuccessLevel;
  canPush: boolean;
  onPush?: () => void;
  onLuck?: () => void;
}

// 基于 Card + Badge 组件扩展
<DiceRoll
  skill="图书馆使用"
  skillValue={60}
  roll={78}
  difficulty="regular"
  successLevel="regularSuccess"
  canPush={true}
  onPush={() => pushRoll()}
/>
```

#### StatePanel

```typescript
interface StatePanelProps {
  character: CharacterState;
  isVisible: boolean;
}

// 基于 Card + Progress 组件扩展
<StatePanel
  character={currentCharacter}
  isVisible={true}
/>
```

### M2 多人组件

#### CombatTracker

```typescript
interface CombatTrackerProps {
  combat: CombatState;
  currentPlayer: string;
}

// 基于 Card + Table 组件扩展
<CombatTracker
  combat={currentCombat}
  currentPlayer="player_001"
/>
```

### M3 记忆组件

#### ClueLedger

```typescript
interface ClueLedgerProps {
  clues: Clue[];
  filter?: "all" | "major" | "minor";
}

// 基于 Card + Collapsible 组件扩展
<ClueLedger
  clues={discoveredClues}
  filter="major"
/>
```

---

## 组件目录结构

```
src/
├── components/
│   ├── ui/                    # shadcn/ui 基础组件
│   │   ├── button.tsx
│   │   ├── input.tsx
│   │   ├── card.tsx
│   │   └── index.ts
│   │
│   ├── game/                  # 游戏专用组件
│   │   ├── message/
│   │   │   ├── message-bubble.tsx
│   │   │   ├── message-list.tsx
│   │   │   └── index.ts
│   │   ├── dice/
│   │   │   ├── dice-roll.tsx
│   │   │   └── index.ts
│   │   ├── status/
│   │   │   ├── state-panel.tsx
│   │   │   └── index.ts
│   │   ├── combat/
│   │   │   ├── combat-tracker.tsx
│   │   │   └── index.ts
│   │   └── index.ts
│   │
│   ├── layout/                # 布局组件
│   │   ├── header.tsx
│   │   ├── sidebar.tsx
│   │   └── index.ts
│   │
│   └── shared/               # 共享组件
│       ├── avatar.tsx
│       └── index.ts
```

---

## 命名规范

### 文件命名

```typescript
// kebab-case
message-bubble.tsx
dice-roll.tsx
state-panel.tsx
```

### 组件命名

```typescript
// PascalCase
const MessageBubble: React.FC = () => {...};

// 复合组件：父组件名 + 子组件名
const MessageBubbleHeader = () => {...};
const MessageBubbleContent = () => {...};
```

### 属性命名

```typescript
// camelCase
interface MessageBubbleProps {
  characterName: string;
  isVisibleTo: Visibility;
  onMessageClick?: () => void;
}
```

### 样式类命名

```typescript
// kebab-case 或 Tailwind 工具类
className="message-bubble-container"
className="flex gap-3 mb-4"
```

---

## 组件属性标准

```typescript
// 基础属性
interface BaseComponentProps {
  isVisible?: boolean;
  className?: string;
  style?: React.CSSProperties;
  "data-testid"?: string;
}

// 游戏组件属性
interface GameComponentProps extends BaseComponentProps {
  role?: "KP" | "Player" | "NPC" | "System";
  visibility?: "public" | "kp" | "private";
  onAction?: (data: any) => void;
}
```

---

## 组件开发规范

### 组件模板

```typescript
/**
 * ComponentName
 *
 * @description 组件描述
 *
 * @example
 * ```tsx
 * <ComponentName prop="value" />
 * ```
 */
import React from "react";
import { cn } from "@/lib/utils";
import { Card } from "@/components/ui/card";

interface ComponentNameProps {
  // 属性定义
}

export const ComponentName: React.FC<ComponentNameProps> = ({
  prop1,
  prop2,
  className,
  ...props
}) => {
  return (
    <Card className={cn("base-class", className)}>
      {/* 组件内容 */}
    </Card>
  );
};
```

### 组件文档

每个游戏组件应包含：

1. **JSDoc 注释**: 组件描述、使用示例
2. **Props 接口**: 详细的属性说明
3. **使用示例**: 实际代码示例
4. **样式说明**: 样式定制方式

---

## 相关文档

- [M0-047 梳理 shadcn/ui 组件需求](../tasks/tasks-detailed/M0-047-shadcn-analysis.md)
- [M0-048 定义游戏专用组件列表](../tasks/tasks-detailed/M0-048-game-components.md)
- [M0-049 编写 MessageBubble API](../tasks/tasks-detailed/M0-049-message-bubble-api.md)
- [M0-050 编写 DiceRoll API](../tasks/tasks-detailed/M0-050-dice-roll-api.md)
- [M0-051 编写 StatePanel API](../tasks/tasks/detailed/M0-051-state-panel-api.md)
- [M0-052 编写 CombatTracker API](../tasks/tasks/detailed/M0-052-combat-tracker-api.md)
- [UI 设计规范](./ui-guidelines.md)
