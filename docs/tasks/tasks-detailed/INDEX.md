# 详细任务索引

> 本目录包含核心任务的详细拆解，每个任务已拆解到**具体的开发任务**级别。

---

## M1 后端 - 掷骰引擎

| 任务 | 标题 | 子任务数 | 状态 |
|------|------|----------|------|
| M1-057 | 实现 d100 随机数生成 | 14 | [已拆解](M1-057-roll-d100.md) |
| M1-058 | 实现大成功/大失败判定 | 17 | [已拆解](M1-058-success判定.md) |
| M1-059 | 实现奖励骰逻辑 | 20 | [已拆解](M1-059-bonus-dice.md) |
| M1-060 | 实现惩罚骰逻辑 | 16 | [已拆解](M1-060-penalty-dice.md) |
| M1-061 | 实现推骰机制 | 20 | [已拆解](M1-061-push-roll.md) |
| M1-062 | 实现花幸运机制 | 18 | [已拆解](M1-062-spend-luck.md) |

---

## M1 后端 - 用户认证

| 任务 | 标题 | 子任务数 | 状态 |
|------|------|----------|------|
| M1-006 | 用户注册与认证 API | 25 | [已拆解](M1-006-user-auth.md) |
| M1-007 | JWT Token 中间件 | 22 | [已拆解](M1-007-jwt-middleware.md) |
| M1-008 | 用户权限管理 | 14 | [已拆解](M1-008-permissions.md) |
| M1-009 | 实现密码哈希 (bcrypt) | 7 | [已拆解](M1-009-password-hash.md) |
| M1-011 | 实现密码验证中间件 | 7 | [已拆解](M1-011-auth-middleware.md) |
| M1-012 | 实现 Token 刷新 POST /auth/refresh | 7 | [已拆解](M1-012-token-refresh.md) |
| M1-013 | 实现角色头像上传功能 | 6 | [已拆解](M1-013-character-avatar.md) |

## M1 前端 - 认证界面

| 任务 | 标题 | 子任务数 | 状态 |
|------|------|----------|------|
| M1-014 | 实现注册页面 RegisterPage | 7 | [已拆解](M1-014-register-page.md) |
| M1-015 | 实现登录页面 LoginPage | 7 | [已拆解](M1-015-login-page.md) |
| M1-016 | 实现 AuthContext 状态管理 | 7 | [已拆解](M1-016-auth-context.md) |

---

## M1 前端 - 游戏界面

| 任务 | 标题 | 子任务数 | 状态 |
|------|------|----------|------|
| M1-017 | 实现前端路由与导航 | 7 | [已拆解](M1-017-routing.md) |
| M1-018 | 实现 MessageList 组件 | 7 | [已拆解](M1-018-message-list.md) |
| M1-019 | 实现 StatePanel 状态面板组件 | 7 | [已拆解](M1-019-state-panel.md) |
| M1-042 | 实现 DiceRoller 掷骰组件 | 7 | [已拆解](M1-042-dice-roller.md) |
| M1-043 | 实现 SkillCheck 技能检定组件 | 7 | [已拆解](M1-043-skill-check.md) |
| M1-044 | 实现 CharacterSheet 角色卡组件 | 7 | [已拆解](M1-044-character-sheet.md) |
| M1-045 | 实现 ScenePanel 场景面板组件 | 7 | [已拆解](M1-045-scene-panel.md) |
| M1-046 | 实现 ChatPanel 聊天面板组件 | 7 | [已拆解](M1-046-chat-panel.md) |
| M1-047 | 实现 CombatPanel 战斗面板组件 | 7 | [已拆解](M1-047-combat-panel.md) |
| M1-048 | 实现 CommandInput 命令输入组件 | 7 | [已拆解](M1-048-command-input.md) |
| M1-060 | 实现掷骰动画效果 | 7 | [已拆解](M1-060-dice-animation.md) |
| M1-061 | 实现音效系统 | 7 | [已拆解](M1-061-sound-effects.md) |
| M1-062 | 实现暗语/密语系统 | 6 | [已拆解](M1-062-cipher-system.md) |
| M1-014 | 实现角色物品栏功能 | 7 | [已拆解](M1-014-character-inventory.md) |
| M1-021 | 实现角色创建向导 | 7 | [已拆解](M1-021-character-creation.md) |

---

## M1 后端 - 项目架构

| 任务 | 标题 | 子任务数 | 状态 |
|------|------|----------|------|
| M1-046 | FastAPI 项目骨架 | 28 | [已拆解](M1-046-fastapi-skeleton.md) |
| M1-001 | 数据库表结构设计 | 18 | [已拆解](M1-001-db-schema.md) |
| M1-002 | 数据库迁移脚本 | 10 | [已拆解](M1-002-migration.md) |

---

## M1 后端 - 角色卡系统

| 任务 | 标题 | 子任务数 | 状态 |
|------|------|----------|------|
| M1-003 | 角色卡数据模型 | 22 | [已拆解](M1-003-character-card.md) |
| M1-004 | 角色卡 CRUD API | 15 | [已拆解](M1-004-character-api.md) |
| M1-005 | 技能系统 | 20 | [已拆解](M1-005-skills.md) |

---

## M1 后端 - 游戏机制

| 任务 | 标题 | 子任务数 | 状态 |
|------|------|----------|------|
| M1-010 | 检定系统 API | 9 | [已拆解](M1-010-check-api.md) |
| M1-020 | 战斗系统 | 23 | [已拆解](M1-020-combat-system.md) |
| M1-030 | 追逐系统 | 20 | [已拆解](M1-030-chase-system.md) |
| M1-040 | SAN 值系统 | 19 | [已拆解](M1-040-san-system.md) |
| M1-050 | 疯狂机制 | 20 | [已拆解](M1-050-madness-system.md) |

---

## M1 后端 - 工具系统

| 任务 | 标题 | 子任务数 | 状态 |
|------|------|----------|------|
| M1-080 | 事件日志系统 | 23 | [已拆解](M1-080-event-log.md) |
| M1-090 | 战役管理系统 | 20 | [已拆解](M1-090-campaign.md) |

---

## M0 规范冻结

| 任务 | 标题 | 子任务数 | 状态 |
|------|------|----------|------|
| M0-001 | 定义核心命令清单 | 9 | [已拆解](M0-001-commands.md) |
| M0-002 | 编写 /help 命令规范 | 6 | [已拆解](M0-002-help.md) |
| M0-003 | 编写 /status 命令规范 | 6 | [已拆解](M0-003-status.md) |
| M0-004 | 编写检定命令规范 (/roll) | 5 | [已拆解](M0-004-roll.md) |
| M0-005 | 编写聊天命令规范 | 6 | [已拆解](M0-005-chat.md) |
| M0-006 | 编写战斗命令规范 | 6 | [已拆解](M0-006-combat.md) |
| M0-007 | 编写物品管理命令规范 | 6 | [已拆解](M0-007-inventory.md) |
| M0-008 | 编写角色命令规范 | 6 | [已拆解](M0-008-character.md) |
| M0-009 | 编写 SAN 检定命令规范 | 6 | [已拆解](M0-009-san-check.md) |
| M0-010 | 编写命令语法 BNF 范式 | 9 | [已拆解](M0-010-syntax.md) |
| M0-011 | 编写命令参数正则表达式 | 7 | [已拆解](M0-011-command-regex.md) |
| M0-012 | 编写物品管理命令规范 | 6 | [已拆解](M0-012-inventory-cmd.md) |
| M0-013 | 编写战斗命令规范 | 6 | [已拆解](M0-013-combat-cmd.md) |
| M0-014 | 编写数据导出命令规范 | 6 | [已拆解](M0-014-export-cmd.md) |
| M0-015 | 编写语音命令规范 | 6 | [已拆解](M0-015-voice-cmd.md) |
| M0-011 | 编写命令参数正则表达式 | 7 | [已拆解](M0-011-command-regex.md) |
| M0-014 | 设计场景包根结构 | 8 | [已拆解](M0-014-scene-format.md) |
| M0-015 | 定义 metadata 元信息结构 | 7 | [已拆解](M0-015-metadata.md) |
| M0-016 | 定义 scenes 场景集合结构 | 7 | [已拆解](M0-016-scenes.md) |
| M0-017 | 定义 NPC 角色数据结构 | 7 | [已拆解](M0-017-npc.md) |
| M0-018 | 定义 Location 地点结构 | 7 | [已拆解](M0-018-location.md) |
| M0-019 | 定义 Clue 线索数据结构 | 7 | [已拆解](M0-019-clue.md) |
| M0-020 | 定义 Handout 手递物格式 | 6 | [已拆解](M0-020-handout.md) |
| M0-022 | 编写场景包 JSON Schema | 7 | [已拆解](M0-022-json-schema.md) |
| M0-035 | 定义 Event 基础结构 | 7 | [已拆解](M0-035-event-structure.md) |
| M0-039 | 定义配色方案 | 7 | [已拆解](M0-039-ui-design.md) |
| M0-047 | 梳理 shadcn/ui 组件需求 | 6 | [已拆解](M0-047-components.md) |
| M0-053 | 编写命令参考手册 | 7 | [已拆解](M0-053-command-reference.md) |

---

## M2 多人 Web 版

| 任务 | 标题 | 子任务数 | 状态 |
|------|------|----------|------|
| M2-001 | 实现房间管理系统 | 9 | [已拆解](M2-001-room-management.md) |
| M2-002 | 实现 WebSocket 事件系统 | 8 | [已拆解](M2-002-websocket-events.md) |
| M2-003 | 实现视频流同步 | 6 | [已拆解](M2-003-video-stream.md) |
| M2-004 | 实现语音聊天功能 | 6 | [已拆解](M2-004-voice-chat.md) |
| M2-005 | 实现文件共享功能 | 7 | [已拆解](M2-005-file-share.md) |
| M2-006 | 实现屏幕共享功能 | 7 | [已拆解](M2-006-screen-share.md) |
| M2-007 | 实现白板功能 | 7 | [已拆解](M2-007-whiteboard.md) |
| M2-008 | 实现房间权限管理 | 6 | [已拆解](M2-008-permission.md) |
| M2-009 | 实现房间模板系统 | 6 | [已拆解](M2-009-room-template.md) |
| M2-010 | 实现房间录制功能 | 7 | [已拆解](M2-010-recording.md) |
| M2-011 | 实现房间邀请系统 | 6 | [已拆解](M2-011-room-invitation.md) |
| M2-022 | 配置 Socket.io 服务 | 9 | [已拆解](M2-022-websocket.md) |
| M2-036 | 设计 SpotlightState 数据结构 | 7 | [已拆解](M2-036-spotlight.md) |

---

## M3 记忆 Web 版

| 任务 | 标题 | 子任务数 | 状态 |
|------|------|----------|------|
| M3-001 | 实现 AI 总结服务 | 7 | [已拆解](M3-001-ai-summary.md) |
| M3-002 | 实现全文搜索功能 | 6 | [已拆解](M3-002-fulltext-search.md) |
| M3-003 | 实现标签系统 | 6 | [已拆解](M3-003-tag-system.md) |
| M3-004 | 实现时间轴功能 | 7 | [已拆解](M3-004-timeline.md) |
| M3-005 | 实现书签功能 | 6 | [已拆解](M3-005-bookmark.md) |
| M3-006 | 实现笔记系统 | 7 | [已拆解](M3-006-notes-system.md) |
| M3-007 | 实现 AI 辅助功能 | 6 | [已拆解](M3-007-ai-assistant.md) |
| M3-008 | 实现会话笔记功能 | 7 | [已拆解](M3-008-session-notes.md) |
| M3-014 | 设计 Summary 数据结构 | 7 | [已拆解](M3-014-summary.md) |

---

## M4 资源管理 Web 版

| 任务 | 标题 | 子任务数 | 状态 |
|------|------|----------|------|
| M4-001 | 实现场景包上传功能 | 7 | [已拆解](M4-001-scene-upload.md) |
| M4-002 | 实现场景库管理界面 | 7 | [已拆解](M4-002-scene-gallery.md) |
| M4-003 | 实现场景预览功能 | 6 | [已拆解](M4-003-scene-preview.md) |
| M4-004 | 实现场景编辑器 | 8 | [已拆解](M4-004-scene-editor.md) |
| M4-005 | 实现场景资源管理器 | 7 | [已拆解](M4-005-asset-manager.md) |
| M4-006 | 实现场景包版本控制 | 6 | [已拆解](M4-006-version-control.md) |
| M4-007 | 实现场景包导入功能 | 7 | [已拆解](M4-007-scene-import.md) |
| M4-008 | 实现场景包市场 | 7 | [已拆解](M4-008-scene-marketplace.md) |

---

## M5 全功能 Web 版

| 任务 | 标题 | 子任务数 | 状态 |
|------|------|----------|------|
| M5-001 | 设计 SAN 检定数据结构 | 7 | [已拆解](M5-001-san-check.md) |
| M5-002 | 实现 SAN 检定 UI 组件 | 7 | [已拆解](M5-002-san-check-ui.md) |
| M5-003 | 实现成长记录系统 | 6 | [已拆解](M5-003-growth-system.md) |
| M5-004 | 实现卡牌翻转功能 | 6 | [已拆解](M5-004-flip-card.md) |
| M5-005 | 实现条件触发器系统 | 7 | [已拆解](M5-005-trigger-system.md) |
| M5-006 | 实现富文本编辑器 | 7 | [已拆解](M5-006-rich-text.md) |
| M5-007 | 实现通知系统 | 7 | [已拆解](M5-007-notification.md) |
| M5-008 | 实现自动保存功能 | 7 | [已拆解](M5-008-autosave.md) |
| M5-009 | 实现战绩系统 | 7 | [已拆解](M5-009-achievement.md) |
| M5-010 | 实现卡牌翻转动画效果 | 5 | [已拆解](M5-010-flip-animation.md) |

---

## M6 体验打磨

| 任务 | 标题 | 子任务数 | 状态 |
|------|------|----------|------|
| M6-001 | 设计 Leads 数据结构 | 7 | [已拆解](M6-001-leads.md) |
| M6-002 | 实现拒绝处理模板 | 6 | [已拆解](M6-002-refusal-templates.md) |
| M6-003 | 实现性能优化 | 7 | [已拆解](M6-003-performance.md) |
| M6-004 | 实现国际化支持 | 6 | [已拆解](M6-004-i18n.md) |
| M6-005 | 实现主题系统 | 7 | [已拆解](M6-005-theme-system.md) |
| M6-006 | 实现快捷键系统 | 7 | [已拆解](M6-006-shortcuts.md) |
| M6-007 | 实现可调整面板布局 | 6 | [已拆解](M6-007-resizable-panels.md) |
| M6-008 | 实现引导系统 | 7 | [已拆解](M6-008-onboarding-system.md) |
| M6-009 | 实现反馈收集 | 8 | [已拆解](M6-009-feedback-collection.md) |
| M6-010 | 实现 A/B 测试框架 | 8 | [已拆解](M6-010-ab-testing-framework.md) |
| M6-011 | 实现错误边界 | 7 | [已拆解](M6-011-error-boundary.md) |
| M6-012 | 实现离线缓存 | 7 | [已拆解](M6-012-offline-cache.md) |
| M6-013 | 实现渐进式 Web 应用 | 8 | [已拆解](M6-013-pwa-implementation.md) |
| M6-014 | 实现数据可视化 | 8 | [已拆解](M6-014-data-visualization.md) |
| M6-015 | 实现性能监控 | 7 | [已拆解](M6-015-performance-monitoring.md) |

---

## 拆解示例预览

### M1-057 子任务

```
M1-057-01  创建 app/core/dice.py              10min
M1-057-04  实现 roll_d100() 函数              10min
M1-057-11  创建 tests/test_dice.py           10min
M1-057-13  测试 modifier 正确应用              5min
```

### M1-058 子任务

```
M1-058-01  创建 app/core/success.py            10min
M1-058-04  实现 calculate_success_level()      15min
M1-058-08  实现 get_success_description()      10min
M1-058-13  创建 tests/test_success.py         10min
```

### M1-059 子任务 (奖励骰)

```
M1-059-01  创建 app/core/bonus.py              10min
M1-059-04  实现 apply_bonus() 函数            15min
M1-059-08  实现 spend_luck() 函数              10min
M1-059-16  创建 tests/test_bonus.py           10min
```

### M1-006 子任务 (用户认证)

```
M1-006-01  创建 app/db/user.py                 15min
M1-006-05  创建 app/core/security.py           10min
M1-006-09  创建 app/services/user.py           15min
M1-006-13  创建 app/api/auth.py                20min
M1-006-21  创建 tests/test_auth.py             20min
```

### M1-046 子任务 (项目骨架)

```
M1-046-01  创建项目目录结构                    15min
M1-046-05  创建 app/core/config.py             15min
M1-046-08  创建 app/db/connection.py           15min
M1-046-12  创建 app/main.py                    15min
M1-046-23  创建 tests/conftest.py              10min
```

### M1-060 子任务 (惩罚骰)

```
M1-060-01  创建 app/core/penalty.py           10min
M1-060-04  实现 apply_penalty() 函数           15min
M1-060-08  实现 combine_penalties() 函数       10min
M1-060-13  创建 tests/test_penalty.py          10min
```

### M1-061 子任务 (推骰)

```
M1-061-01  创建 app/core/push.py               10min
M1-061-05  实现 can_push() 函数                10min
M1-061-06  实现 execute_push() 函数            20min
M1-061-15  创建 tests/test_push.py             15min
```

### M1-062 子任务 (花幸运)

```
M1-062-01  创建 app/core/luck.py               10min
M1-062-05  实现 spend_luck() 函数               15min
M1-062-10  实现 apply_luck_to_roll() 函数       20min
M1-062-15  创建 tests/test_luck.py             15min
```

### M1-007 子任务 (JWT 中间件)

```
M1-007-01  创建 app/core/token.py               10min
M1-007-05  创建 app/api/deps/auth.py            15min
M1-007-09  创建 app/api/deps/permissions.py     10min
M1-007-13  实现刷新令牌端点                     10min
```

### M1-001 子任务 (数据库设计)

```
M1-001-01  创建 app/db/models/user.py           15min
M1-001-05  创建 app/db/models/character.py      20min
M1-001-09  创建 app/db/models/campaign.py       15min
M1-001-13  创建 app/db/models/gamestate.py      20min
```

### M1-003 子任务 (角色卡模型)

```
M1-003-01  创建 app/services/character.py        20min
M1-003-06  创建 app/services/skill.py           15min
M1-003-11  创建 app/api/character.py            20min
M1-003-15  创建 app/schemas/character.py        10min
```

### M1-010 子任务 (检定系统)

```
M1-010-01  创建 app/services/check.py            15min
M1-010-05  实现 execute_check() 函数           20min
M1-010-11  创建 app/api/check.py              15min
M1-010-15  创建 tests/test_check.py           15min
```

### M1-020 子任务 (战斗系统)

```
M1-020-01  创建 app/core/combat.py             20min
M1-020-05  创建 app/services/combat.py         25min
M1-020-15  创建 app/api/combat.py             15min
M1-020-19  创建 tests/test_combat.py          15min
```

### M1-030 子任务 (追逐系统)

```
M1-030-01  创建 app/core/chase.py              15min
M1-030-05  创建 app/services/chase.py          25min
M1-030-12  创建 app/api/chase.py              15min
M1-030-16  创建 tests/test_chase.py           15min
```

### M1-040 子任务 (SAN 值系统)

```
M1-040-01  创建 app/core/san.py              15min
M1-040-05  创建 app/services/san.py           20min
M1-040-12  创建 app/api/san.py              15min
M1-040-16  创建 tests/test_san.py           10min
```

### M1-050 子任务 (疯狂机制)

```
M1-050-01  创建 app/core/madness.py          15min
M1-050-05  创建 app/services/madness.py      20min
M1-050-12  创建 app/api/madness.py          10min
M1-050-16  创建 tests/test_madness.py       10min
```

### M1-080 子任务 (事件日志)

```
M1-080-01  创建 app/core/logger.py           10min
M1-080-05  创建 app/services/logger.py       15min
M1-080-16  创建 app/api/logger.py           10min
M1-080-20  创建 tests/test_logger.py         5min
```

---

## 拆解模板

每个详细任务包含以下部分：

```
1. 子任务拆解表
   - 每个子任务 ID、描述、预估时间

2. 代码示例
   - 核心数据模型
   - 核心函数实现

3. 测试用例
   - 单元测试示例

4. 涉及文件清单
   - 需要创建/修改的文件

5. 验收标准
   - 完成条件

6. 参考文档链接
   - 相关规范、规则书章节
```

---

## 使用方法

1. 按模块阅读详细任务
2. 按子任务 ID 领取开发任务
3. 子任务完成后更新状态
4. 所有子任务完成后标记主任务完成

---

## 推荐开发顺序

| 推荐度 | 任务 | 理由 |
|--------|------|------|
| ⭐⭐⭐ | M1-057 | 掷骰是核心机制，依赖少 |
| ⭐⭐⭐ | M1-058 | 成功判定是检定核心 |
| ⭐⭐ | M1-059 | 奖励骰依赖 M1-057/058 |
| ⭐⭐ | M1-001 | 数据库设计是后续基础 |
| ⭐⭐ | M1-006 | 用户认证是系统入口 |
| ⭐⭐ | M1-046 | FastAPI 项目骨架 |

---

## M5 全功能扩展

| 任务 | 标题 | 子任务数 | 状态 |
|------|------|----------|------|
| M5-011 | 实现条件触发器编辑器 | 6 | [已拆解](M5-011-condition-trigger-editor.md) |
| M5-012 | 实现触发器测试工具 | 6 | [已拆解](M5-012-trigger-testing-tool.md) |
| M5-013 | 实现宏命令系统 | 6 | [已拆解](M5-013-macro-system.md) |
| M5-014 | 实现自定义表情 | 5 | [已拆解](M5-014-custom-emoji.md) |
| M5-015 | 实现 API 密钥管理 | 5 | [已拆解](M5-015-api-key-management.md) |
| M5-016 | 实现插件系统 | 7 | [已拆解](M5-016-plugin-system.md) |
| M5-017 | 实现数据备份 | 6 | [已拆解](M5-017-data-backup.md) |
| M5-018 | 实现数据恢复 | 6 | [已拆解](M5-018-data-restore.md) |

---

**最后更新**: 2026-02-06
**已拆解**: 190 / 687 任务 (~27.7%)
