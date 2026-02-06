# M1-008: 用户权限管理

**任务ID**: M1-008
**标题**: 用户权限管理
**类型**: backend (后端开发)
**预估工时**: 4h
**依赖**: M1-007

---

## 任务描述

实现基于角色的用户权限管理系统，区分 KP (守密人) 和 Player (玩家) 权限，支持不同角色的操作权限控制。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-008-01 | 定义权限枚举 | 列出所有系统权限 | 20min |
| M1-008-02 | 定义角色类型 | KP / Player / Guest | 15min |
| M1-008-03 | 设计权限数据结构 | Role-Permission 映射 | 20min |
| M1-008-04 | 实现权限检查中间件 | FastAPI dependency | 30min |
| M1-008-05 | 实现 Campaign 角色绑定 | Campaign 中的角色 | 30min |
| M1-008-06 | 实现权限检查装饰器 | @require_permission | 20min |
| M1-008-07 | 编写权限单元测试 | 测试各种权限场景 | 30min |
| M1-008-08 | 编写权限文档 | API 权限说明 | 20min |
| M1-008-09 | 更新 OpenAPI 文档 | 添加权限标注 | 15min |

---

## 权限定义

```typescript
// 权限枚举
enum Permission {
  // 基础权限
  READ_SESSION = 'read:session',
  WRITE_MESSAGE = 'write:message',

  // KP 权限
  MANAGE_CAMPAIGN = 'manage:campaign',
  MANAGE_NPCS = 'manage:npcs',
  SET_DIFFICULTY = 'set:difficulty',
  VIEW_ALL_STATES = 'view:all_states',
  MODIFY_PLAYER_STATE = 'modify:player_state',

  // Player 权限
  VIEW_OWN_STATE = 'view:own_state',
  CONTROL_OWN_CHARACTER = 'control:own_character',

  // 通用权限
  ROLL_DICE = 'roll:dice',
  VIEW_RULES = 'view:rules',
}

// 角色定义
interface Role {
  name: string;
  permissions: Permission[];
}

const ROLES: Record<string, Role> = {
  kp: {
    name: 'KP',
    permissions: [
      Permission.READ_SESSION,
      Permission.WRITE_MESSAGE,
      Permission.MANAGE_CAMPAIGN,
      Permission.MANAGE_NPCS,
      Permission.SET_DIFFICULTY,
      Permission.VIEW_ALL_STATES,
      Permission.MODIFY_PLAYER_STATE,
      Permission.VIEW_RULES,
    ],
  },
  player: {
    name: 'Player',
    permissions: [
      Permission.READ_SESSION,
      Permission.WRITE_MESSAGE,
      Permission.VIEW_OWN_STATE,
      Permission.CONTROL_OWN_CHARACTER,
      Permission.ROLL_DICE,
      Permission.VIEW_RULES,
    ],
  },
  guest: {
    name: 'Guest',
    permissions: [
      Permission.READ_SESSION,
      Permission.VIEW_RULES,
    ],
  },
};
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/core/permissions.py` | 创建 | 权限定义和检查 |
| `app/api/deps/permissions.py` | 创建 | 权限依赖 |
| `app/db/models/role.py` | 创建 | 角色数据模型 |
| `tests/test_permissions.py` | 创建 | 权限测试 |

---

## 权限检查中间件

```python
# app/api/deps/permissions.py
from fastapi import Depends, HTTPException, status
from app.core.permissions import Permission, ROLES

async def require_permission(permission: Permission):
    """权限检查依赖"""
    async def check(current_user = Depends(get_current_user)):
        user_role = ROLES.get(current_user.role)
        if permission not in user_role.permissions:
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail=f"Permission '{permission}' required"
            )
        return current_user
    return check

# 使用示例
@router.post("/game/difficulty")
async def set_difficulty(
    difficulty: int,
    current_user = Depends(require_permission(Permission.SET_DIFFICULTY))
):
    ...
```

---

## 验收标准

- [ ] 权限枚举定义完整
- [ ] KP/Player 角色区分清晰
- [ ] 权限检查中间件工作正常
- [ ] 越权请求被正确拒绝
- [ ] API 文档标注权限要求

---

## 参考文档

- M1-007: JWT Token 中间件
- FastAPI 权限管理最佳实践

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
