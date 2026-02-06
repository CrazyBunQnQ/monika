# M0-047: 梳理 shadcn/ui 组件需求

**任务ID**: M0-047
**标题**: 梳理 shadcn/ui 组件需求
**类型**: plan (规划)
**预估工时**: 2h
**依赖**: 无

---

## 任务描述

分析 CoC 跑团平台的 UI 需求，梳理需要从 shadcn/ui 使用的组件，以及需要自定义开发的游戏专用组件。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M0-047-01 | 审查各里程碑 UI 需求 | 梳理所有界面组件需求 | 30min |
| M0-047-02 | 映射到 shadcn/ui 组件 | 确定可直接使用的组件 | 30min |
| M0-047-03 | 识别自定义组件需求 | 需要自行开发的组件 | 30min |
| M0-047-04 | 评估 shadcn/ui 扩展性 | 确定哪些组件需要扩展 | 20min |
| M0-047-05 | 制定组件迁移计划 | 确定组件引入优先级 | 15min |
| M0-047-06 | 编写组件清单文档 | 输出完整组件列表 | 15min |

---

## shadcn/ui 组件需求映射

### 可直接使用的组件

| 组件 | 用途 | 使用场景 |
|------|------|----------|
| Button | 按钮 | 所有交互操作 |
| Input | 文本输入 | 聊天、表单 |
| Textarea | 多行输入 | 长文本输入 |
| Select | 下拉选择 | 角色选择、选项选择 |
| Dialog | 对话框 | 确认弹窗、设置 |
| Sheet | 侧边抽屉 | 详情面板 |
| Tabs | 标签页 | 多视图切换 |
| Card | 卡片 | 角色卡、脚本卡片 |
| Badge | 徽章 | 状态标签、数量提示 |
| Avatar | 头像 | 用户头像 |
| Progress | 进度条 | HP/SAN 进度条 |
| Slider | 滑块 | 数值调整 |
| Switch | 开关 | 设置项 |
| Tooltip | 工具提示 | 按钮说明 |
| Toast | 消息提示 | 操作反馈 |
| Table | 表格 | 角色卡列表、日志 |
| Separator | 分隔线 | 布局分隔 |
| ScrollArea | 滚动区域 | 消息列表 |

### 需要扩展的组件

| 组件 | 扩展需求 | 扩展方式 |
|------|----------|----------|
| Card | 游戏风格 | 添加边框样式、悬停效果 |
| Badge | 更多状态 | 添加 "kp-only"、"private" 等状态 |
| Progress | 游戏数值 | 添加颜色变化、阈值标记 |
| Tooltip | 引用显示 | 添加事件引用、规则引用 |

---

## 游戏专用自定义组件

### 消息相关
```typescript
- MessageBubble      // 消息气泡 (游戏风格化)
- MessageList        // 消息列表 (虚拟滚动)
- TypingIndicator    // 打字指示器
```

### 游戏相关
```typescript
- DiceRoll           // 骰子展示 (带动画)
- CombatTracker      // 战斗追踪器
- ChaseTracker       // 追逐追踪器
- SANMeter           // SAN 值条 (专用)
- LeadsPanel         // 可选行动面板
- SpotlightIndicator // 聚光灯指示器
```

### 角色相关
```typescript
- CharacterCard      // 角色卡预览 (完整)
- StatusBadge        // 状态徽章 (游戏化)
- InventoryGrid      // 物品栏网格
- AttributeDisplay   // 属性展示
- SkillList          // 技能列表
```

### UI 工具
```typescript
- ChatInput          // 聊天输入 (命令补全)
- CommandPalette     // 命令面板 (Ctrl+K)
- Timeline           // 时间线 (复盘)
- SceneNavigator     // 场景导航器
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `docs/specs/components.md` | 创建 | 组件设计文档 |
| `frontend/README.md` | 更新 | 添加组件清单 |

---

## 组件引入优先级

### P0 - 核心组件 (M0-M1)
- Button, Input, Textarea, Card, Badge
- MessageBubble, DiceRoll, StatePanel

### P1 - 扩展组件 (M2-M3)
- Dialog, Sheet, Tabs, Progress, Tooltip
- CombatTracker, ChaseTracker, Timeline

### P2 - 优化组件 (M4-M6)
- Avatar, Slider, Switch, Table
- InventoryGrid, CommandPalette

---

## 验收标准

- [ ] shadcn/ui 组件映射完整
- [ ] 自定义组件列表清晰
- [ ] 组件优先级明确
- [ ] 组件文档完整

---

## 参考文档

- shadcn/ui 组件文档
- M0-046: 响应式断点规范

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
