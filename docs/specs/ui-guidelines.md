# UI 设计规范

**版本**: v1.0
**最后更新**: 2026-02-07
**状态**: ✅ 设计完成

---

## 概述

本文档定义 CoC 跑团平台的 UI 设计规范，采用**混合风格**设计理念——游戏氛围的外壳结合现代化的交互内核。

**设计原则**:
- **混合风格**: 游戏氛围装饰 + 现代化组件
- **双色调**: KP 和玩家使用不同配色方案
- **三层字体**: 正文可读 + 标题优雅 + 装饰哥特
- **响应式**: 支持移动端、平板、桌面三种布局

---

## 配色方案

### 双色调系统

**KP 侧配色（紫色系）**:

```typescript
const kpTheme = {
  // 主色
  primary: "#7c3aed",      // 深紫
  primaryHover: "#6d28d9",
  primaryLight: "#a78bfa",
  secondary: "#a855f7",    // 浅紫
  accent: "#f59e0b",       // 金色（强调）

  // 背景色
  background: "#0f0a1a",   // 深色背景，带紫调
  surface: "#1a1425",      // 卡片背景
  surfaceHover: "#251d32",

  // 文字色
  text: "#e2e8f0",
  textSecondary: "#94a3b8",
  textMuted: "#64748b",

  // 状态色
  success: "#10b981",
  warning: "#f59e0b",
  danger: "#ef4444",
  info: "#3b82f6",
};
```

**玩家侧配色（青绿色系）**:

```typescript
const playerTheme = {
  // 主色
  primary: "#14b8a6",      // 青绿
  primaryHover: "#0d9488",
  primaryLight: "#5eead4",
  secondary: "#06b6d4",    // 青
  accent: "#f59e0b",       // 金色（统一）

  // 背景色
  background: "#0a192f",   // 深色背景，带青调
  surface: "#112240",      // 卡片背景
  surfaceHover: "#1a365d",

  // 文字色
  text: "#e2e8f0",
  textSecondary: "#94a3b8",
  textMuted: "#64748b",

  // 状态色（统一）
  success: "#10b981",
  warning: "#f59e0b",
  danger: "#ef4444",
  info: "#3b82f6",
};
```

### 检定结果配色

```typescript
const resultColors = {
  critical: "#f59e0b",     // 大成功/大失败 🌟
  extremeSuccess: "#22c55e", // 极难成功 ✨
  hardSuccess: "#84cc16",   // 困难成功 👍
  regularSuccess: "#3b82f6", // 普通成功 ✅
  failure: "#ef4444",        // 失败 ❌
  fumble: "#dc2626",         // 大失败 💀
};
```

### 角色状态配色

```typescript
const statusColors = {
  healthy: "#10b981",   // 健康
  injured: "#f59e0b",   // 受伤
  critical: "#ef4444",  // 危急
  dying: "#dc2626",     // 濒死
  dead: "#71717a",      // 死亡
  insane: "#a855f7",    // 疯狂
};
```

---

## 字体系统

### 字体栈

```typescript
const fonts = {
  // 正文：可读性优先
  sans: [
    "Inter",
    "-apple-system",
    "BlinkMacSystemFont",
    "sans-serif"
  ],

  // 标题：优雅风格
  serif: [
    "Cormorant Garamond",
    "Playfair Display",
    "serif"
  ],

  // 装饰：哥特风格（仅大标题/LOGO）
  display: [
    "UnifrakturMaguntia",
    "MedievalSharp",
    "cursive"
  ],

  // 代码/数值：等宽字体
  mono: [
    "JetBrains Mono",
    "Fira Code",
    "monospace"
  ]
};
```

### 字体大小

```typescript
const fontSizes = {
  xs: "0.75rem",      // 12px
  sm: "0.875rem",     // 14px
  base: "1rem",       // 16px - 正文基础
  lg: "1.125rem",     // 18px
  xl: "1.25rem",      // 20px
  "2xl": "1.5rem",    // 24px - 小标题
  "3xl": "1.875rem",  // 30px - 大标题
  "4xl": "2.25rem",   // 36px - 装饰标题
  "5xl": "3rem",      // 48px - 页面标题
};
```

### 字重和行高

```typescript
const fontWeights = {
  normal: 400,
  medium: 500,
  semibold: 600,
  bold: 700,
};

const lineHeights = {
  tight: 1.25,
  normal: 1.5,
  relaxed: 1.75,
};
```

### 字体使用规则

| 字体 | 用途 | 示例 |
|------|------|------|
| Inter | 正文、UI 文本 | 消息内容、按钮、表单 |
| Cormorant Garamond | 标题、章节名 | 场景标题、NPC 名、面板标题 |
| UnifrakturMaguntia | 装饰、LOGO | 页面大标题、特殊场合 |
| JetBrains Mono | 数值、代码 | 骰子结果、检定值、状态值 |

---

## 间距系统

### 4px 基准间距

```typescript
const spacing = {
  0: "0",
  0.5: "0.125rem",  // 2px
  1: "0.25rem",    // 4px
  2: "0.5rem",     // 8px
  3: "0.75rem",    // 12px
  4: "1rem",       // 16px
  5: "1.25rem",    // 20px
  6: "1.5rem",     // 24px
  8: "2rem",       // 32px
  10: "2.5rem",    // 40px
  12: "3rem",      // 48px
  16: "4rem",      // 64px
  20: "5rem",      // 80px
  24: "6rem",      // 96px
};
```

### 应用规则

```typescript
const spacingRules = {
  // 组件内边距
  padding: {
    xs: spacing[1],   // 4px
    sm: spacing[2],   // 8px
    md: spacing[3],   // 12px
    lg: spacing[4],   // 16px
    xl: spacing[6],   // 24px
  },

  // 组件间边距
  gap: {
    xs: spacing[2],   // 8px
    sm: spacing[3],   // 12px
    md: spacing[4],   // 16px
    lg: spacing[6],   // 24px
    xl: spacing[8],   // 32px
  },

  // 区块间距
  section: {
    sm: spacing[8],   // 32px
    md: spacing[12],  // 48px
    lg: spacing[16],  // 64px
    xl: spacing[24],  // 96px
  },
};
```

---

## 组件样式

### 消息气泡（角色卡片风格）

```typescript
// 结构
<MessageBubble>
  <Avatar />
  <Content>
    <Name>角色名</Name>
    <Text>消息内容</Text>
    <Timestamp>10:30</Timestamp>
  </Content>
</MessageBubble>

// 样式特征
{
  display: "flex",
  gap: "12px",
  marginBottom: "16px",

  // 头像 40x40 圆形
  avatar: {
    width: "40px",
    height: "40px",
    borderRadius: "50%",
    border: "2px solid",
    borderColor: "roleColor",
  },

  // 内容区 左侧彩色边框
  content: {
    maxWidth: "70%",
    padding: "12px",
    borderRadius: "12px",
    borderLeft: "4px solid",
    borderLeftColor: "roleColor",
  },
}
```

### 骰子结果展示

```typescript
// 结构
<DiceRoll>
  <Header>
    <SkillName>图书馆使用</SkillName>
    <SkillValue>60%</SkillValue>
  </Header>

  <RollDisplay>78</RollDisplay>

  <SuccessLevel>
    <Icon>✅</Icon>
    <Text>普通成功</Text>
  </SuccessLevel>

  <Actions>
    <PushButton>推骰</PushButton>
  </Actions>
</DiceRoll>

// 样式特征
{
  // 骰子大号等宽字体
  rollDisplay: {
    fontFamily: "JetBrains Mono",
    fontSize: "36px",
    fontWeight: "bold",
    padding: "24px",
    border: "3px solid",
    borderColor: "primary",
    boxShadow: "0 0 20px rgba(124, 58, 237, 0.2)",
  },

  // 成功等级彩色标识
  successLevel: {
    display: "inline-flex",
    padding: "8px 16px",
    borderRadius: "20px",
    border: "2px solid",

    variants: {
      critical: { color: "#f59e0b", icon: "🌟" },
      regularSuccess: { color: "#3b82f6", icon: "✅" },
      failure: { color: "#ef4444", icon: "❌" },
    },
  },
}
```

### 状态面板

```typescript
// 结构
<StatusPanel>
  <StatCard>
    <Label>HP</Label>
    <ProgressBar value={10} max={12} />
    <Value>10/12</Value>
  </StatCard>

  <StatCard>
    <Label>SAN</Label>
    <ProgressBar value={45} max={60} status="injured" />
    <Value>45/60</Value>
  </StatCard>
</StatusPanel>

// 样式特征
{
  // 进度条
  progressBar: {
    height: "8px",
    borderRadius: "4px",
    overflow: "hidden",

    fill: {
      transition: "width 0.3s ease",
      variants: {
        healthy: { backgroundColor: "#10b981" },
        injured: { backgroundColor: "#f59e0b" },
        critical: {
          backgroundColor: "#ef4444",
          animation: "pulse 1s infinite",
        },
      },
    },
  },
}
```

---

## 响应式断点

### 断点定义

```typescript
const breakpoints = {
  xs: "0px",       // 移动设备（小）
  sm: "640px",     // 移动设备（大）
  md: "768px",     // 平板设备（小）
  lg: "1024px",    // 平板设备（大）/ 桌面（小）
  xl: "1280px",    // 桌面设备
  "2xl": "1536px", // 大屏幕
};
```

### 响应式布局

| 屏幕尺寸 | 消息列表 | 消息气泡 | 状态面板 |
|----------|----------|----------|----------|
| 移动端 (<768px) | 100% 宽度 | 最大 85% | 底部抽屉 |
| 平板端 (768-1024px) | 90% 宽度 | 最大 70% | 右侧抽屉 |
| 桌面端 (>1024px) | 70% 宽度 | 最大 60% | 固定侧边栏 |

### 触控优化

```typescript
const touchOptimizations = {
  // 最小触控目标
  minTouchSize: "44px × 44px",

  // 增大点击区域
  buttonPadding: {
    mobile: "12px 24px",
    desktop: "8px 16px",
  },

  // 手势支持
  gestures: {
    swipeToClose: true,
    pullToRefresh: false,
    pinchToZoom: false,
    longPress: true,
  },
};
```

---

## 动画和过渡

### 缓动函数

```typescript
const easings = {
  ease: "cubic-bezier(0.4, 0, 0.2, 1)",
  easeOut: "cubic-bezier(0, 0, 0.2, 1)",
  smooth: "cubic-bezier(0.25, 0.1, 0.25, 1)",
  bounce: "cubic-bezier(0.68, -0.55, 0.265, 1.55)",
};
```

### 动画时长

```typescript
const durations = {
  instant: "100ms",
  fast: "200ms",
  normal: "300ms",
  slow: "500ms",
  slower: "1000ms",
};
```

### 常用动画

| 动画 | 用途 | 时长 |
|------|------|------|
| diceRoll | 骰子滚动 | 1000ms |
| messageAppear | 消息出现 | 300ms |
| statDecrease | 状态减少 | 1000ms |
| combatStart | 战斗开始 | 300ms |
| pulse | 重要提醒 | 无限循环 |
| slideIn | 面板滑入 | 300ms |

---

## 暗色模式

### 主题切换

```typescript
// 支持的主题
const themes = {
  kpDark: "KP 深色主题（默认）",
  kpLight: "KP 浅色主题",
  playerDark: "玩家深色主题（默认）",
  playerLight: "玩家浅色主题",
};

// 切换方式
const themeSwitching = {
  auto: "基于系统偏好自动切换",
  manual: "用户手动切换",
  scheduled: "按时间切换（18:00-06:00）",
};
```

### CSS Variables

```css
:root {
  /* 基础颜色 */
  --background: #0f0a1a;
  --surface: #1a1425;
  --text: #e2e8f0;

  /* 主题色 */
  --primary: #7c3aed;
  --secondary: #a855f7;
  --accent: #f59e0b;

  /* 状态色 */
  --success: #10b981;
  --warning: #f59e0b;
  --danger: #ef4444;

  /* 间距 */
  --spacing-sm: 8px;
  --spacing-md: 16px;
  --spacing-lg: 24px;

  /* 圆角 */
  --radius-md: 8px;
  --radius-lg: 12px;

  /* 阴影 */
  --shadow-md: 0 4px 6px rgba(0, 0, 0, 0.3);
  --glow-primary: 0 0 20px rgba(124, 58, 237, 0.4);
}
```

---

## 装饰元素

### 氛围装饰

```typescript
const decorations = {
  // 边框装饰
  borders: {
    corner: "corner-border",      // 角落装饰
    double: "double-border",      // 双线边框
    ornate: "ornate-border",      // 华丽边框
  },

  // 分隔符
  dividers: {
    fancy: "✦ • ✦",              // 装饰分隔
    simple: "───",                // 简单分隔
    chapter: "❧",                 // 章节分隔
  },

  // 图标
  icons: {
    dice: ["⚀", "⚁", "⚂", "⚃", "⚄", "⚅"],
    suit: ["♠️", "♥️", "♣️", "♦️"],
    status: ["✓", "✗", "⚠", "★"],
    mood: ["🌙", "🕯️", "⚰️", "👁️"],
  },
};
```

### KP 专属装饰

```typescript
const kpDecorations = {
  // 私密标识
  private: {
    icon: "🔒",
    badge: "KP ONLY",
    color: "warning",
  },

  // 可见性标识
  visibility: {
    public: "👁️ 公开",
    party: "👥 玩家",
    private: "🔒 私密",
  },
};
```

---

## 可访问性

### WCAG 标准

```typescript
const accessibility = {
  // 对比度要求
  contrastRatio: {
    aa: "4.5:1",      // 正文文本
    aaLarge: "3:1",   // 大号文本
    aaa: "7:1",       // 高对比度
  },

  // 焦点管理
  focus: {
    indicator: "visible",
    offset: "2px",
    borderRadius: "2px",
  },

  // 键盘导航
  keyboard: {
    skipLinks: true,
    tabOrder: "logical",
    shortcuts: {
      focusChat: "Alt+C",
      focusStatus: "Alt+S",
      toggleTheme: "Alt+T",
    },
  },
};
```

---

## 相关文档

- [M0-039 定义配色方案](../tasks/tasks-detailed/M0-039-color-scheme.md)
- [M0-040 定义字体层级规范](../tasks/tasks-detailed/M0-040-font-hierarchy.md)
- [M0-041 定义间距系统](../tasks/tasks-detailed/M0-041-spacing-system.md)
- [M0-042 定义消息气泡样式](../tasks/tasks-detailed/M0-042-message-bubble.md)
- [M0-043 定义状态指示器样式](../tasks/tasks-detailed/M0-043-status-indicator.md)
- [M0-044 定义骰子结果展示规范](../tasks/tasks-detailed/M0-044-dice-display.md)
- [M0-045 定义动画/过渡效果规范](../tasks/tasks-detailed/M0-045-animations.md)
- [M0-046 定义响应式断点规范](../tasks/tasks-detailed/M0-046-responsive-breakpoints.md)
- [组件设计文档](./components.md) (待创建)
