# CoC 跑团平台任务清单

**项目**: CoC 跑团平台 (AI-Driven CoC 7e TRPG Platform)
**技术栈**: React + shadcn/ui / Python + Agno
**总周期**: 约 19 周 (M0-M6)

---

## 目录

| 文件 | 内容 | 周期 |
|------|------|------|
| [01-m0-spec-freeze.md](./01-m0-spec-freeze.md) | M0: 规范冻结 | 1 周 |
| [02-m1-single-player-web.md](./02-m1-single-player-web.md) | M1: 单人 Web 版 | 4 周 |
| [03-m2-multiplayer-web.md](./03-m2-multiplayer-web.md) | M2: 多人 Web 版 | 3 周 |
| [04-m3-memory-web.md](./04-m3-memory-web.md) | M3: 记忆 Web 版 | 3 周 |
| [05-m4-resource-web.md](./05-m4-resource-web.md) | M4: 资源管理 Web 版 | 2 周 |
| [06-m5-full-feature-web.md](./06-m5-full-feature-web.md) | M5: 全功能 Web 版 | 4 周 |
| [07-m6-polishing.md](./07-m6-polishing.md) | M6: 体验打磨 | 2 周 |

---

## 里程碑依赖关系

```
M0 (规范冻结)
  │
  ▼
M1 (单人可玩 + 单人Web界面)
  │
  ▼
M2 (多人支持 + 多人Web界面)
  │
  ▼
M3 (长记忆 + 复盘Web界面)
  │
  ▼
M4 (知识库 + 资源管理Web界面)
  │
  ▼
M5 (全功能 + 完整游戏UI)
  │
  ▼
M6 (体验打磨 + 交互优化)
```

---

## 任务状态标记

| 标记 | 含义 |
|------|------|
| [ ] | 待开始 |
| [P] | 进行中 |
| [D] | 待审核 |
| [✅] | 已完成 |

---

## 快速开始

建议按以下顺序开始任务：

1. 先阅读 M0 任务，理解规范冻结阶段的工作
2. 创建 GitHub Projects 看板，导入各阶段任务
3. 每个里程碑开始前，召开技术评审会议
4. 完成后更新本文件的进度追踪

---

## 验收标准速查

| 里程碑 | 核心验收标准 |
|--------|-------------|
| M0 | 命令集/场景包/状态字段定义完成 |
| M1 | 单人检定/战斗/追逐闭环 + Web 界面可用 |
| M2 | 2-4 人同团 10+ 轮稳定运行 |
| M3 | `/recap` 输出稳定结构 + 断点恢复 |
| M4 | 模组导入校验 + 规则检索 |
| M5 | SAN/疯狂/完整战斗 + 成长系统 |
| M6 | 响应式适配 + 性能优化 |

---

**最后更新**: 2026-02-05
**文档版本**: v1.0
