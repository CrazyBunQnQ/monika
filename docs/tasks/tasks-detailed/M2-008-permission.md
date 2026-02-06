# M2-008: 实现房间权限管理

**任务ID**: M2-008
**标题**: 实现房间权限管理
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M2-001

---

## 任务描述

实现房间权限管理系统，支持角色权限（KP、玩家）、房间设置、访问控制等功能。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M2-008-01 | 设计权限模型 | Permission Model | 25min |
| M2-008-02 | 实现角色权限 | Role Permissions | 30min |
| M2-008-03 | 实现房间设置 | Room Settings | 25min |
| M2-008-04 | 实现访问控制 | Access Control | 25min |
| M2-008-05 | 实现权限 API | API | 25min |
| M2-008-06 | 编写权限测试 | 测试覆盖 | 15min |

---

## 权限模型

```python
# app/db/models/permission.py
from sqlalchemy import Column, String, Boolean, ForeignKey, JSON
from sqlalchemy.orm import relationship
from app.db.database import Base

class RoomPermission(Base):
    """房间权限"""
    __tablename__ = 'room_permissions'

    id = Column(String, primary_key=True, index=True)
    room_id = Column(String, ForeignKey('rooms.id'), nullable=False, index=True)
    user_id = Column(String, ForeignKey('users.id'), nullable=False, index=True)

    # 角色
    role = Column(String, nullable=False)  # kp, player, observer

    # 权限标志
    can_edit_settings = Column(Boolean, default=False)
    can_kick_players = Column(Boolean, default=False)
    can_manage_npcs = Column(Boolean, default=False)
    can_control_combat = Column(Boolean, default=True)
    can_post_messages = Column(Boolean, default=True)
    can_roll_dice = Column(Boolean, default=True)
    can_view_hidden = Column(Boolean, default=False)

    # 额外权限
    custom_permissions = Column(JSON)

    # 关系
    room = relationship("Room", back_populates="permissions")
    user = relationship("User", back_populates="room_permissions")

    def __repr__(self):
        return f"<RoomPermission {self.user_id}@{self.room_id} ({self.role})>"
```

---

## 权限服务

```python
# app/services/permission.py
from typing import List, Dict, Any, Optional, Set
from sqlalchemy.orm import Session

from app.db.models.permission import RoomPermission
from app.db.models.room import Room, RoomParticipant
from app.core.security import generate_id

class PermissionService:
    """权限服务"""

    def __init__(self, db: Session):
        self.db = db

    def get_user_role(self, room_id: str, user_id: str) -> str:
        """获取用户在房间中的角色"""
        participant = self.db.query(RoomParticipant)\
            .filter(
                RoomParticipant.room_id == room_id,
                RoomParticipant.user_id == user_id,
            )\
            .first()

        if not participant:
            return None

        return participant.role

    def get_user_permissions(
        self,
        room_id: str,
        user_id: str,
    ) -> Optional[RoomPermission]:
        """获取用户权限"""
        return self.db.query(RoomPermission)\
            .filter(
                RoomPermission.room_id == room_id,
                RoomPermission.user_id == user_id,
            )\
            .first()

    def create_permission(
        self,
        room_id: str,
        user_id: str,
        role: str,
        created_by: str,
    ) -> RoomPermission:
        """创建权限记录"""
        permission = RoomPermission(
            id=generate_id('permission'),
            room_id=room_id,
            user_id=user_id,
            role=role,
            # 根据角色设置默认权限
            **self._get_default_permissions(role),
        )

        self.db.add(permission)
        self.db.commit()
        self.db.refresh(permission)

        return permission

    def _get_default_permissions(self, role: str) -> Dict[str, Any]:
        """根据角色获取默认权限"""
        defaults = {
            'kp': {
                'can_edit_settings': True,
                'can_kick_players': True,
                'can_manage_npcs': True,
                'can_control_combat': True,
                'can_post_messages': True,
                'can_roll_dice': True,
                'can_view_hidden': True,
            },
            'player': {
                'can_edit_settings': False,
                'can_kick_players': False,
                'can_manage_npcs': False,
                'can_control_combat': True,
                'can_post_messages': True,
                'can_roll_dice': True,
                'can_view_hidden': False,
            },
            'observer': {
                'can_edit_settings': False,
                'can_kick_players': False,
                'can_manage_npcs': False,
                'can_control_combat': False,
                'can_post_messages': False,
                'can_roll_dice': False,
                'can_view_hidden': False,
            },
        }

        return defaults.get(role, defaults['player'])

    def check_permission(
        self,
        room_id: str,
        user_id: str,
        permission: str,
    ) -> bool:
        """检查用户是否有特定权限"""
        perm = self.get_user_permissions(room_id, user_id)

        if not perm:
            return False

        return getattr(perm, permission, False)

    def update_permission(
        self,
        room_id: str,
        user_id: str,
        updates: Dict[str, Any],
    ) -> Optional[RoomPermission]:
        """更新权限"""
        perm = self.get_user_permissions(room_id, user_id)

        if not perm:
            return None

        for key, value in updates.items():
            if hasattr(perm, key):
                setattr(perm, key, value)

        self.db.commit()
        self.db.refresh(perm)

        return perm

    def grant_permission(
        self,
        room_id: str,
        user_id: str,
        permission: str,
    ) -> bool:
        """授予权限"""
        perm = self.get_user_permissions(room_id, user_id)

        if not perm:
            return False

        setattr(perm, permission, True)
        self.db.commit()

        return True

    def revoke_permission(
        self,
        room_id: str,
        user_id: str,
        permission: str,
    ) -> bool:
        """撤销权限"""
        perm = self.get_user_permissions(room_id, user_id)

        if not perm:
            return False

        setattr(perm, permission, False)
        self.db.commit()

        return True

    def get_room_permissions(
        self,
        room_id: str,
    ) -> List[Dict[str, Any]]:
        """获取房间所有用户的权限"""
        permissions = self.db.query(RoomPermission)\
            .filter(RoomPermission.room_id == room_id)\
            .all()

        return [
            {
                'user_id': p.user_id,
                'username': p.user.username,
                'role': p.role,
                'can_edit_settings': p.can_edit_settings,
                'can_kick_players': p.can_kick_players,
                'can_manage_npcs': p.can_manage_npcs,
                'can_control_combat': p.can_control_combat,
                'can_post_messages': p.can_post_messages,
                'can_roll_dice': p.can_roll_dice,
                'can_view_hidden': p.can_view_hidden,
            }
            for p in permissions
        ]
```

---

## 权限中间件

```python
# app/api/deps/permissions.py
from fastapi import Depends, HTTPException, status
from sqlalchemy.orm import Session

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.permission import PermissionService

def get_permission_service(db: Session = Depends(get_db)):
    return PermissionService(db)

def require_room_permission(permission: str):
    """权限检查装饰器工厂"""
    def decorator(
        room_id: str,
        current_user: User = Depends(get_current_user),
        perm_service: PermissionService = Depends(get_permission_service),
    ):
        if not perm_service.check_permission(room_id, current_user.id, permission):
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail=f"缺少权限: {permission}",
            )
        return True

    return decorator

def require_room_role(allowed_roles: list[str]):
    """角色检查装饰器工厂"""
    def decorator(
        room_id: str,
        current_user: User = Depends(get_current_user),
        perm_service: PermissionService = Depends(get_permission_service),
    ):
        user_role = perm_service.get_user_role(room_id, current_user.id)

        if user_role not in allowed_roles:
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail=f"需要以下角色之一: {', '.join(allowed_roles)}",
            )
        return True

    return decorator
```

---

## 权限 API

```python
# app/api/permissions.py
from fastapi import APIRouter, Depends
from sqlalchemy.orm import Session
from pydantic import BaseModel

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.api.deps.permissions import require_room_permission, require_room_role
from app.db.models.user import User
from app.services.permission import PermissionService
from app.api.deps.permissions import get_permission_service

router = APIRouter(prefix="/permissions", tags=["permissions"])

class UpdatePermissionRequest(BaseModel):
    permissions: dict

@router.get("/room/{room_id}")
async def get_room_permissions(
    room_id: str,
    current_user: User = Depends(get_current_user),
    perm_service: PermissionService = Depends(get_permission_service),
):
    """获取房间权限列表"""
    # 只有 KP 可以查看权限列表
    if perm_service.get_user_role(room_id, current_user.id) != 'kp':
        raise HTTPException(status_code=403, detail="只有 KP 可以查看权限")

    return perm_service.get_room_permissions(room_id)

@router.put("/room/{room_id}/user/{user_id}")
async def update_user_permission(
    room_id: str,
    user_id: str,
    request: UpdatePermissionRequest,
    current_user: User = Depends(get_current_user),
    perm_service: PermissionService = Depends(get_permission_service),
):
    """更新用户权限"""
    # 只有 KP 可以修改权限
    if perm_service.get_user_role(room_id, current_user.id) != 'kp':
        raise HTTPException(status_code=403, detail="只有 KP 可以修改权限")

    perm = perm_service.update_permission(
        room_id=room_id,
        user_id=user_id,
        updates=request.permissions,
    )

    if not perm:
        raise HTTPException(status_code=404, detail="用户不存在")

    return {"message": "权限已更新"}

@router.post("/room/{room_id}/role/{user_id}")
async def change_user_role(
    room_id: str,
    user_id: str,
    role: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """更改用户角色"""
    perm_service = PermissionService(db)

    # 只有 KP 可以更改角色
    if perm_service.get_user_role(room_id, current_user.id) != 'kp':
        raise HTTPException(status_code=403, detail="只有 KP 可以更改角色")

    # 删除旧权限，创建新权限
    db.query(RoomPermission)\
        .filter(
            RoomPermission.room_id == room_id,
            RoomPermission.user_id == user_id,
        )\
        .delete()

    perm_service.create_permission(room_id, user_id, role, current_user.id)

    # 更新参与者的角色
    participant = db.query(RoomParticipant)\
        .filter(
            RoomParticipant.room_id == room_id,
            RoomParticipant.user_id == user_id,
        )\
        .first()

    if participant:
        participant.role = role
        db.commit()

    return {"message": f"角色已更改为 {role}"}
```

---

## 前端权限管理组件

```tsx
// frontend/src/components/game/PermissionManager.tsx
import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { Shield, UserPlus, UserMinus } from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'
import { useToast } from '@/hooks/use-toast'

interface Permission {
  user_id: string
  username: string
  role: string
  can_edit_settings: boolean
  can_kick_players: boolean
  can_manage_npcs: boolean
  can_control_combat: boolean
  can_post_messages: boolean
  can_roll_dice: boolean
  can_view_hidden: boolean
}

interface PermissionManagerProps {
  roomId: string
}

export function PermissionManager({ roomId }: PermissionManagerProps) {
  const [permissions, setPermissions] = useState<Permission[]>([])
  const { user } = useAuth()
  const { toast } = useToast()

  useEffect(() => {
    loadPermissions()
  }, [roomId])

  const loadPermissions = async () => {
    try {
      const response = await fetch(`/api/permissions/room/${roomId}`)
      if (!response.ok) {
        // 可能不是 KP
        return
      }

      const data = await response.json()
      setPermissions(data)
    } catch (error) {
      console.error('Failed to load permissions:', error)
    }
  }

  const updatePermission = async (userId: string, updates: Record<string, boolean>) => {
    try {
      await fetch(`/api/permissions/room/${roomId}/user/${userId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ permissions: updates }),
      })

      setPermissions(prev =>
        prev.map(p =>
          p.user_id === userId ? { ...p, ...updates } : p
        )
      )
    } catch (error) {
      toast({
        title: '更新失败',
        variant: 'destructive',
      })
    }
  }

  const changeRole = async (userId: string, newRole: string) => {
    try {
      await fetch(`/api/permissions/room/${roomId}/role/${userId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ role: newRole }),
      })

      setPermissions(prev =>
        prev.map(p =>
          p.user_id === userId ? { ...p, role: newRole } : p
        )
      )

      toast({
        title: '角色已更新',
      })
    } catch (error) {
      toast({
        title: '更新失败',
        variant: 'destructive',
      })
    }
  }

  const getRoleBadgeColor = (role: string) => {
    switch (role) {
      case 'kp':
        return 'default'
      case 'player':
        return 'secondary'
      case 'observer':
        return 'outline'
      default:
        return 'secondary'
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base flex items-center">
          <Shield className="h-4 w-4 mr-2" />
          权限管理
        </CardTitle>
      </CardHeader>

      <CardContent>
        <div className="space-y-4">
          {permissions.map((perm) => (
            <div key={perm.user_id} className="border rounded p-3">
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center space-x-2">
                  <span className="font-medium">{perm.username}</span>
                  <Badge variant={getRoleBadgeColor(perm.role)}>
                    {perm.role.toUpperCase()}
                  </Badge>
                </div>

                {user?.id !== perm.user_id && (
                  <div className="flex space-x-2">
                    <select
                      value={perm.role}
                      onChange={(e) => changeRole(perm.user_id, e.target.value)}
                      className="h-8 rounded px-2 border text-sm"
                    >
                      <option value="kp">KP</option>
                      <option value="player">玩家</option>
                      <option value="observer">观察者</option>
                    </select>
                  </div>
                )}
              </div>

              <div className="grid grid-cols-2 gap-2 text-sm">
                <div className="flex items-center justify-between">
                  <span>编辑设置</span>
                  <Switch
                    checked={perm.can_edit_settings}
                    onCheckedChange={(checked) =>
                      updatePermission(perm.user_id, { can_edit_settings: checked })
                    }
                  />
                </div>
                <div className="flex items-center justify-between">
                  <span>踢出玩家</span>
                  <Switch
                    checked={perm.can_kick_players}
                    onCheckedChange={(checked) =>
                      updatePermission(perm.user_id, { can_kick_players: checked })
                    }
                  />
                </div>
                <div className="flex items-center justify-between">
                  <span>管理 NPC</span>
                  <Switch
                    checked={perm.can_manage_npcs}
                    onCheckedChange={(checked) =>
                      updatePermission(perm.user_id, { can_manage_npcs: checked })
                    }
                  />
                </div>
                <div className="flex items-center justify-between">
                  <span>控制战斗</span>
                  <Switch
                    checked={perm.can_control_combat}
                    onCheckedChange={(checked) =>
                      updatePermission(perm.user_id, { can_control_combat: checked })
                    }
                  />
                </div>
                <div className="flex items-center justify-between">
                  <span>发送消息</span>
                  <Switch
                    checked={perm.can_post_messages}
                    onCheckedChange={(checked) =>
                      updatePermission(perm.user_id, { can_post_messages: checked })
                    }
                  />
                </div>
                <div className="flex items-center justify-between">
                  <span>掷骰</span>
                  <Switch
                    checked={perm.can_roll_dice}
                    onCheckedChange={(checked) =>
                      updatePermission(perm.user_id, { can_roll_dice: checked })
                    }
                  />
                </div>
                <div className="flex items-center justify-between col-span-2">
                  <span>查看隐藏内容</span>
                  <Switch
                    checked={perm.can_view_hidden}
                    onCheckedChange={(checked) =>
                      updatePermission(perm.user_id, { can_view_hidden: checked })
                    }
                  />
                </div>
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/db/models/permission.py` | 创建 | 权限数据模型 |
| `app/services/permission.py` | 创建 | 权限服务 |
| `app/api/deps/permissions.py` | 创建 | 权限依赖 |
| `app/api/permissions.py` | 创建 | 权限 API |
| `frontend/src/components/game/PermissionManager.tsx` | 创建 | 权限管理组件 |

---

## 验收标准

- [ ] 角色权限正确
- [ ] 权限检查有效
- [ ] 权限更新成功
- [ ] 中间件拦截正确
- [ ] UI 控制精确
- [ ] 角色切换流畅

---

## 参考文档

- M2-001: 房间管理系统

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
