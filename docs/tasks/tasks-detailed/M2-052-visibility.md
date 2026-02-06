# M2-052: 定义 Visibility 枚举

**任务ID**: M2-052
**标题**: 定义 Visibility 枚举
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M0

---

## 任务描述

定义可见性 (Visibility) 枚举和规则，用于控制游戏中的信息可见性，确保 KP-only 信息不泄露给玩家。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M2-052-01 | 分析可见性需求 | 确定可见性级别 | 20min |
| M2-052-02 | 设计 Visibility 枚举 | 可见性类型 | 25min |
| M2-052-03 | 设计可见性规则 | 权限判断逻辑 | 25min |
| M2-052-04 | 实现可见性检查 | Python 检查函数 | 30min |
| M2-052-05 | 编写单元测试 | 测试各种可见性场景 | 25min |
| M2-052-06 | 编写可见性文档 | 使用说明 | 10min |

---

## Visibility 枚举

```typescript
enum Visibility {
  // === 基础级别 ===
  PUBLIC = 'public',           // 所有人可见
  KP_ONLY = 'kp',              // 仅 KP 可见
  PARTY = 'party',             // 所有玩家可见 (KP 也可见)
  SELF = 'self',               // 仅自己可见

  // === 玩家特定 ===
  PLAYER_PREFIX = 'player:',   // player:<user_id> 特定玩家
  PARTY_EXCEPT = 'party-except', // 除某玩家外的所有人

  // === 动态规则 ===
  CUSTOM = 'custom',           // 自定义规则
  CONTEXTUAL = 'contextual',   // 根据上下文判断

  // === 特殊场景 ===
  INITIALLY_SECRET = 'initially-secret', // 初始私密，后续公开
  ON_SUCCESS = 'on-success',   // 成功后可见
  ON_FAILURE = 'on-failure',   // 失败后可见
}

// 可见性配置
interface VisibilityConfig {
  type: Visibility;

  // 特定用户
  users?: string[];           // 可见用户列表
  exclude?: string[];         // 排除用户列表

  // 自定义规则
  condition?: string;         // 可见条件表达式
  rule?: (context: GameContext) => boolean;

  // 时间限制
  expires_after?: number;     // 多少秒后公开
  reveal_on?: string;         // 触发公开的事件

  // 元数据
  metadata?: Record<string, any>;
}
```

---

## 可见性规则

```python
# app/core/visibility.py
from enum import Enum
from typing import List, Optional, Callable
from dataclasses import dataclass

class Visibility(Enum):
    PUBLIC = "public"
    KP_ONLY = "kp"
    PARTY = "party"
    SELF = "self"
    PLAYER = "player"
    CUSTOM = "custom"

@dataclass
class VisibilityConfig:
    type: Visibility
    users: Optional[List[str]] = None
    exclude: Optional[List[str]] = None
    condition: Optional[str] = None
    expires_after: Optional[int] = None
    reveal_on: Optional[str] = None

class VisibilityChecker:
    def __init__(self):
        pass

    def can_view(
        self,
        visibility: VisibilityConfig,
        viewer_user_id: str,
        viewer_role: str,
        context: dict
    ) -> bool:
        """检查用户是否可以查看内容"""

        # PUBLIC - 所有人可见
        if visibility.type == Visibility.PUBLIC:
            return True

        # KP_ONLY - 仅 KP
        if visibility.type == Visibility.KP_ONLY:
            return viewer_role == 'kp'

        # PARTY - 所有玩家和 KP
        if visibility.type == Visibility.PARTY:
            return viewer_role in ['kp', 'player']

        # SELF - 仅自己
        if visibility.type == Visibility.SELF:
            owner_id = context.get('owner_user_id')
            return viewer_user_id == owner_id

        # PLAYER - 特定玩家
        if visibility.type == Visibility.PLAYER:
            if not visibility.users:
                return False
            return viewer_user_id in visibility.users

        # CUSTOM - 自定义规则
        if visibility.type == Visibility.CUSTOM:
            return self._check_custom(visibility, viewer_user_id, context)

        return False

    def _check_custom(
        self,
        visibility: VisibilityConfig,
        viewer_user_id: str,
        context: dict
    ) -> bool:
        """检查自定义可见性规则"""

        # 检查排除列表
        if visibility.exclude and viewer_user_id in visibility.exclude:
            return False

        # 检查用户列表
        if visibility.users and viewer_user_id not in visibility.users:
            return False

        # 检查条件表达式
        if visibility.condition:
            return self._evaluate_condition(visibility.condition, context)

        return True

    def _evaluate_condition(self, condition: str, context: dict) -> bool:
        """评估条件表达式"""
        # 简单实现，可以使用更复杂的表达式解析器
        try:
            return eval(condition, {}, context)
        except:
            return False

    def filter_visible(
        self,
        items: List[dict],
        viewer_user_id: str,
        viewer_role: str,
        context: dict
    ) -> List[dict]:
        """过滤出可见的项目"""
        return [
            item for item in items
            if self.can_view(
                item.get('visibility'),
                viewer_user_id,
                viewer_role,
                {**context, **item}
            )
        ]
```

---

## 消息可见性示例

```typescript
// KP 私密消息
{
  event_id: "evt_001",
  type: "message",
  content: "NPC 实际上是凶手",
  visibility: {
    type: Visibility.KP_ONLY
  }
}

// 给特定玩家的私密线索
{
  event_id: "evt_002",
  type: "clue_discovered",
  content: "你发现了一张照片",
  visibility: {
    type: Visibility.PLAYER,
    users: ["user_123"]
  }
}

// 公开消息
{
  event_id: "evt_003",
  type: "message",
  content: "大家进入了图书馆",
  visibility: {
    type: Visibility.PUBLIC
  }
}

// 初始私密，成功后公开
{
  event_id: "evt_004",
  type: "check_result",
  content: "检定成功，发现了隐藏的暗门",
  visibility: {
    type: Visibility.ON_SUCCESS,
    reveal_on: "evt_003"  // 当事件 evt_003 发生时公开
  }
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/core/visibility.py` | 创建 | 可见性检查 |
| `app/core/types/visibility.ts` | 创建 | TypeScript 类型 |
| `tests/test_visibility.py` | 创建 | 可见性测试 |

---

## API 使用示例

```python
# 在 API 路由中使用
from app.core.visibility import VisibilityChecker, Visibility, VisibilityConfig

@router.get("/events")
async def get_events(
    current_user = Depends(get_current_user),
    events_service: EventsService = Depends()
):
    checker = VisibilityChecker()

    # 获取所有事件
    all_events = await events_service.get_all()

    # 过滤可见事件
    visible_events = checker.filter_visible(
        all_events,
        viewer_user_id=current_user.id,
        viewer_role=current_user.role,
        context={'session_id': current_user.session_id}
    )

    return visible_events
```

---

## 验收标准

- [ ] Visibility 枚举定义完整
- [ ] 可见性检查正确
- [ ] KP-only 信息不泄露
- [ ] 私密信息正确隔离
- [ ] 单元测试覆盖全面

---

## 参考文档

- M0-035: Event 基础结构
- M2-053: 消息可见性设置

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
