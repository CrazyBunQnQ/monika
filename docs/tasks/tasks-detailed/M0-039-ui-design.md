# M0-039: 定义配色方案 (暗色/亮色)

**任务ID**: M0-039
**标题**: 定义配色方案 (暗色/亮色)
**类型**: design (UI设计)
**预估工时**: 2h
**依赖**: 无

---

## 任务描述

定义 CoC 跑团平台的配色方案，包括亮色模式和暗色模式。配色需符合恐怖/悬疑类游戏的氛围，同时保证良好的可读性。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M0-039-01 | 研究游戏配色趋势 | 参考 TRPG/恐怖类游戏配色 | 15min |
| M0-039-02 | 定义亮色主题配色 | 背景色、文字色、主题色等 | 30min |
| M0-039-03 | 定义暗色主题配色 | 对应的暗色变体 | 30min |
| M0-039-04 | 定义状态颜色 | success/warning/danger/info | 15min |
| M0-039-05 | 定义 KP/Player 区分色 | 角色身份区分 | 10min |
| M0-039-06 | 创建色板文档 | CSS 变量格式 | 20min |
| M0-039-07 | 可访问性检查 | 对比度符合 WCAG AA 标准 | 10min |

---

## 配色方案

### 亮色主题
```typescript
const lightTheme = {
  // 背景
  background: '#ffffff',
  surface: '#f8f9fa',
  surfaceHover: '#f1f3f5',
  border: '#dee2e6',

  // 文字
  text: '#212529',
  textSecondary: '#6c757d',
  textMuted: '#adb5bd',

  // 主题色 (偏紫色调，符合神秘氛围)
  primary: '#5c6bc0',
  primaryHover: '#3949ab',
  primaryLight: '#9fa8da',
  secondary: '#78909c',

  // 状态色
  success: '#66bb6a',   // 绿色 - 成功/恢复
  warning: '#ffa726',   // 橙色 - 警告
  danger: '#ef5350',    // 红色 - 危险/伤害
  info: '#42a5f5',      // 蓝色 - 信息

  // KP / Player 区分
  kp: '#7e57c2',        // 紫色 - KP
  player: '#26a69a',    // 青色 - Player

  // 游戏特定
  sanity: '#7e57c2',    // SAN 值
  health: '#ef5350',    // HP
  luck: '#ffa726',      // Luck

  // 语义
  rollSuccess: '#66bb6a',
  rollFailure: '#ef5350',
  rollCritical: '#ab47bc',
  rollFumble: '#ff7043',
};
```

### 暗色主题
```typescript
const darkTheme = {
  // 背景
  background: '#1a1a2e',
  surface: '#16213e',
  surfaceHover: '#1f2b4d',
  border: '#3f3f5f',

  // 文字
  text: '#e4e6eb',
  textSecondary: '#a0a3b1',
  textMuted: '#6e7181',

  // 主题色
  primary: '#7c8fff',
  primaryHover: '#5c6eff',
  primaryLight: '#b3b8ff',
  secondary: '#8fa3bf',

  // 状态色 (保持高对比度)
  success: '#81c784',
  warning: '#ffb74d',
  danger: '#e57373',
  info: '#64b5f6',

  // KP / Player 区分
  kp: '#b388ff',
  player: '#4db6ac',

  // 游戏特定
  sanity: '#b388ff',
  health: '#e57373',
  luck: '#ffb74d',

  // 语义
  rollSuccess: '#81c784',
  rollFailure: '#e57373',
  rollCritical: '#ce93d8',
  rollFumble: '#ff8a65',
};
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `docs/specs/ui-guidelines.md` | 创建 | UI 设计规范 |
| `frontend/src/styles/theme.css` | 创建 | 主题 CSS 变量 |
| `frontend/src/styles/theme-light.css` | 创建 | 亮色主题 |
| `frontend/src/styles/theme-dark.css` | 创建 | 暗色主题 |

---

## CSS 变量示例
```css
:root {
  /* 基础色 */
  --color-background: #ffffff;
  --color-surface: #f8f9fa;
  --color-text: #212529;

  /* 主题色 */
  --color-primary: #5c6bc0;
  --color-primary-hover: #3949ab;

  /* 状态色 */
  --color-success: #66bb6a;
  --color-warning: #ffa726;
  --color-danger: #ef5350;

  /* 角色色 */
  --color-kp: #7e57c2;
  --color-player: #26a69a;
}

[data-theme="dark"] {
  --color-background: #1a1a2e;
  --color-surface: #16213e;
  --color-text: #e4e6eb;
  /* ... */
}
```

---

## 验收标准

- [ ] 亮色/暗色主题完整定义
- [ ] 对比度符合 WCAG AA 标准 (4.5:1)
- [ ] 语义色清晰
- [ ] KP/Player 身份可区分
- [ ] 游戏状态值有专用色

---

## 参考文档

- Material Design 配色指南
- WCAG 对比度标准
- shadcn/ui 默认配色

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
