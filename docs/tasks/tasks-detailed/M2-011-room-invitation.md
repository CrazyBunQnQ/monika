# M2-011: 实现房间邀请系统

**任务ID**: M2-011
**标题**: 实现房间邀请系统
**类型**: fullstack (全栈开发)
**预估工时**: 2h
**依赖**: M2-001

---

## 任务描述

实现房间邀请系统，支持通过链接、二维码、邀请码等方式邀请玩家加入游戏房间。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M2-011-01 | 设计邀请数据模型 | Invitation Model | 20min |
| M2-011-02 | 实现邀请码生成 | Invite Code Gen | 25min |
| M2-011-03 | 实现链接邀请 | Link Invitation | 25min |
| M2-011-04 | 实现二维码邀请 | QR Code Invitation | 25min |
| M2-011-05 | 实现邀请管理 | Invitation Management | 20min |
| M2-011-06 | 实现前端邀请UI | Invitation UI | 25min |

---

## 邀请数据模型

```python
# app/db/models/invitation.py
from sqlalchemy import Column, String, Integer, DateTime, ForeignKey, Boolean, JSON
from sqlalchemy.orm import relationship
from app.db.database import Base
from datetime import datetime, timedelta

class RoomInvitation(Base):
    """房间邀请"""
    __tablename__ = 'room_invitations'

    id = Column(String, primary_key=True, index=True)
    room_id = Column(String, ForeignKey('rooms.id'), nullable=False, index=True)

    # 邀请者
    created_by = Column(String, ForeignKey('users.id'), nullable=False)

    # 邀请码
    code = Column(String, unique=True, nullable=False, index=True)

    # 类型
    type = Column(String, default='link')  # link, qr_code, one_time

    # 权限
    role = Column(String, default='player')  # player, co_kp

    # 使用限制
    max_uses = Column(Integer)  # 最大使用次数
    used_count = Column(Integer, default=0)

    # 有效期
    expires_at = Column(DateTime)
    is_active = Column(Boolean, default=True)

    # 使用记录
    used_by = Column(JSON, default=list)  # [{user_id, used_at}]

    # 元数据
    description = Column(String)
    metadata = Column(JSON)

    # 时间戳
    created_at = Column(DateTime, default=datetime.utcnow)

    # 关系
    room = relationship("Room", back_populates="invitations")
    creator = relationship("User", foreign_keys=[created_by])

    def __repr__(self):
        return f"<RoomInvitation {self.code}>"

    def is_valid(self):
        """检查邀请是否有效"""
        if not self.is_active:
            return False

        if self.expires_at and datetime.utcnow() > self.expires_at:
            return False

        if self.max_uses and self.used_count >= self.max_uses:
            return False

        return True

    def can_use(self, user_id: str):
        """检查用户是否可以使用邀请"""
        if not self.is_valid():
            return False

        # 检查是否已使用过
        if self.type == 'one_time':
            for record in self.used_by or []:
                if record['user_id'] == user_id:
                    return False

        return True

    def use(self, user_id: str):
        """使用邀请"""
        self.used_count += 1
        if not self.used_by:
            self.used_by = []
        self.used_by.append({
            'user_id': user_id,
            'used_at': datetime.utcnow().isoformat(),
        })
```

---

## 邀请服务

```python
# app/services/invitation.py
from typing import Dict, Any, Optional, List
from sqlalchemy.orm import Session
from datetime import datetime, timedelta
import string
import random

from app.db.models.invitation import RoomInvitation
from app.core.security import generate_id

class InvitationService:
    """邀请服务"""

    def __init__(self, db: Session):
        self.db = db

    def generate_code(self, length: int = 8) -> str:
        """生成邀请码"""
        chars = string.ascii_uppercase + string.digits
        # 排除易混淆的字符
        chars = chars.replace('0', '').replace('O', '').replace('I', '').replace('1', '')
        return ''.join(random.choice(chars) for _ in range(length))

    def create_invitation(
        self,
        room_id: str,
        created_by: str,
        type: str = 'link',
        role: str = 'player',
        max_uses: int = None,
        expires_in_hours: int = None,
        description: str = None,
    ) -> RoomInvitation:
        """创建邀请"""
        # 生成唯一邀请码
        code = self._generate_unique_code()

        # 计算过期时间
        expires_at = None
        if expires_in_hours:
            expires_at = datetime.utcnow() + timedelta(hours=expires_in_hours)

        invitation = RoomInvitation(
            id=generate_id('invitation'),
            room_id=room_id,
            created_by=created_by,
            code=code,
            type=type,
            role=role,
            max_uses=max_uses,
            expires_at=expires_at,
            description=description,
        )

        self.db.add(invitation)
        self.db.commit()
        self.db.refresh(invitation)

        return invitation

    def _generate_unique_code(self) -> str:
        """生成唯一邀请码"""
        while True:
            code = self.generate_code()
            existing = self.db.query(RoomInvitation)\
                .filter(RoomInvitation.code == code)\
                .first()
            if not existing:
                return code

    def get_invitation(
        self,
        code: str,
    ) -> Optional[RoomInvitation]:
        """获取邀请"""
        return self.db.query(RoomInvitation)\
            .filter(RoomInvitation.code == code)\
            .first()

    def validate_invitation(
        self,
        code: str,
        user_id: str,
    ) -> Dict[str, Any]:
        """验证邀请"""
        invitation = self.get_invitation(code)

        if not invitation:
            return {
                "valid": False,
                "error": "邀请不存在",
            }

        if not invitation.is_valid():
            if invitation.expires_at and datetime.utcnow() > invitation.expires_at:
                return {
                    "valid": False,
                    "error": "邀请已过期",
                }
            if invitation.max_uses and invitation.used_count >= invitation.max_uses:
                return {
                    "valid": False,
                    "error": "邀请已达到使用次数上限",
                }
            return {
                "valid": False,
                "error": "邀请已失效",
            }

        if not invitation.can_use(user_id):
            return {
                "valid": False,
                "error": "您已使用过此邀请",
            }

        return {
            "valid": True,
            "room_id": invitation.room_id,
            "role": invitation.role,
        }

    def use_invitation(
        self,
        code: str,
        user_id: str,
    ) -> Dict[str, Any]:
        """使用邀请"""
        invitation = self.get_invitation(code)

        if not invitation:
            return {"success": False, "error": "邀请不存在"}

        if not invitation.can_use(user_id):
            return {"success": False, "error": "邀请无效或已使用"}

        # 使用邀请
        invitation.use(user_id)

        self.db.commit()

        return {
            "success": True,
            "room_id": invitation.room_id,
            "role": invitation.role,
        }

    def get_room_invitations(
        self,
        room_id: str,
        include_expired: bool = False,
    ) -> List[RoomInvitation]:
        """获取房间邀请列表"""
        query = self.db.query(RoomInvitation)\
            .filter(RoomInvitation.room_id == room_id)

        if not include_expired:
            query = query.filter(RoomInvitation.is_active == True)

        return query.order_by(RoomInvitation.created_at.desc()).all()

    def revoke_invitation(
        self,
        invitation_id: str,
        user_id: str,
    ) -> bool:
        """撤销邀请"""
        invitation = self.db.query(RoomInvitation)\
            .filter(
                RoomInvitation.id == invitation_id,
                RoomInvitation.created_by == user_id,
            )\
            .first()

        if not invitation:
            return False

        invitation.is_active = False
        self.db.commit()

        return True

    def get_invitation_url(
        self,
        code: str,
    ) -> str:
        """获取邀请链接"""
        # 从配置获取基础 URL
        base_url = "https://monika.example.com"
        return f"{base_url}/invite/{code}"

    def generate_qr_code(
        self,
        invitation_url: str,
    ) -> bytes:
        """生成二维码"""
        try:
            import qrcode
            from io import BytesIO

            qr = qrcode.QRCode(
                version=1,
                error_correction=qrcode.constants.ERROR_CORRECT_L,
                box_size=10,
                border=4,
            )
            qr.add_data(invitation_url)
            qr.make(fit=True)

            img = qr.make_image(fill_color="black", back_color="white")

            # 转换为 bytes
            buffer = BytesIO()
            img.save(buffer, format='PNG')
            return buffer.getvalue()

        except ImportError:
            # 如果没有 qrcode 库，返回空
            return b""
```

---

## 邀请 API

```python
# app/api/invitation.py
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session
from pydantic import BaseModel
from typing import List, Optional
from fastapi.responses import Response

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.invitation import InvitationService

router = APIRouter(prefix="/invitations", tags=["invitations"])

class CreateInvitationRequest(BaseModel):
    room_id: str
    type: str = 'link'
    role: str = 'player'
    max_uses: int = None
    expires_in_hours: int = None
    description: str = None

@router.post("")
async def create_invitation(
    request: CreateInvitationRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """创建邀请"""
    service = InvitationService(db)

    invitation = service.create_invitation(
        room_id=request.room_id,
        created_by=current_user.id,
        type=request.type,
        role=request.role,
        max_uses=request.max_uses,
        expires_in_hours=request.expires_in_hours,
        description=request.description,
    )

    invite_url = service.get_invitation_url(invitation.code)

    return {
        "id": invitation.id,
        "code": invitation.code,
        "url": invite_url,
        "expires_at": invitation.expires_at.isoformat() if invitation.expires_at else None,
    }

@router.get("/room/{room_id}")
async def get_room_invitations(
    room_id: str,
    include_expired: bool = False,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取房间邀请列表"""
    service = InvitationService(db)
    invitations = service.get_room_invitations(room_id, include_expired)

    return {
        "invitations": [
            {
                "id": inv.id,
                "code": inv.code,
                "type": inv.type,
                "role": inv.role,
                "max_uses": inv.max_uses,
                "used_count": inv.used_count,
                "expires_at": inv.expires_at.isoformat() if inv.expires_at else None,
                "is_active": inv.is_active,
                "created_at": inv.created_at.isoformat(),
            }
            for inv in invitations
        ]
    }

@router.get("/{code}")
async def get_invitation(
    code: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取邀请详情"""
    service = InvitationService(db)
    invitation = service.get_invitation(code)

    if not invitation:
        raise HTTPException(status_code=404, detail="邀请不存在")

    return {
        "id": invitation.id,
        "code": invitation.code,
        "type": invitation.type,
        "role": invitation.role,
        "is_active": invitation.is_active,
        "expires_at": invitation.expires_at.isoformat() if invitation.expires_at else None,
    }

@router.post("/{code}/validate")
async def validate_invitation(
    code: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """验证邀请"""
    service = InvitationService(db)
    result = service.validate_invitation(code, current_user.id)

    return result

@router.post("/{code}/use")
async def use_invitation(
    code: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """使用邀请（加入房间）"""
    service = InvitationService(db)
    result = service.use_invitation(code, current_user.id)

    if not result["success"]:
        raise HTTPException(status_code=400, detail=result["error"])

    # 添加用户到房间
    # TODO: 调用房间服务添加成员

    return result

@router.post("/{invitation_id}/revoke")
async def revoke_invitation(
    invitation_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """撤销邀请"""
    service = InvitationService(db)
    success = service.revoke_invitation(invitation_id, current_user.id)

    if not success:
        raise HTTPException(status_code=404, detail="邀请不存在或无权撤销")

    return {"message": "邀请已撤销"}

@router.get("/{code}/qr")
async def get_qr_code(
    code: str,
    db: Session = Depends(get_db),
):
    """获取邀请二维码"""
    service = InvitationService(db)

    invitation = service.get_invitation(code)
    if not invitation:
        raise HTTPException(status_code=404, detail="邀请不存在")

    invite_url = service.get_invitation_url(code)
    qr_data = service.generate_qr_code(invite_url)

    if not qr_data:
        raise HTTPException(status_code=500, detail="二维码生成失败")

    return Response(content=qr_data, media_type="image/png")
```

---

## 前端邀请组件

```tsx
// frontend/src/components/room/RoomInvitation.tsx
import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Copy, Link, QrCode, Trash2, Plus } from 'lucide-react'
import { useToast } from '@/hooks/use-toast'

interface Invitation {
  id: string
  code: string
  type: string
  role: string
  max_uses: number
  used_count: number
  expires_at: string | null
  is_active: boolean
  created_at: string
}

interface RoomInvitationProps {
  roomId: string
}

export function RoomInvitation({ roomId }: RoomInvitationProps) {
  const [invitations, setInvitations] = useState<Invitation[]>([])
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [showQrDialog, setShowQrDialog] = useState(false)
  const [selectedInvitation, setSelectedInvitation] = useState<Invitation | null>(null)
  const [qrCodeUrl, setQrCodeUrl] = useState('')
  const [createForm, setCreateForm] = useState({
    type: 'link',
    role: 'player',
    max_uses: '',
    expires_in_hours: '',
  })
  const { toast } = useToast()

  useEffect(() => {
    fetchInvitations()
  }, [roomId])

  const fetchInvitations = async () => {
    try {
      const response = await fetch(`/api/invitations/room/${roomId}`)
      if (response.ok) {
        const data = await response.json()
        setInvitations(data.invitations)
      }
    } catch (error) {
      console.error('Failed to fetch invitations:', error)
    }
  }

  const handleCreate = async () => {
    try {
      const response = await fetch('/api/invitations', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          room_id: roomId,
          type: createForm.type,
          role: createForm.role,
          max_uses: createForm.max_uses ? parseInt(createForm.max_uses) : null,
          expires_in_hours: createForm.expires_in_hours ? parseInt(createForm.expires_in_hours) : null,
        }),
      })

      if (response.ok) {
        const data = await response.json()
        toast({
          title: '邀请创建成功',
          description: '邀请链接已复制到剪贴板',
        })
        await navigator.clipboard.writeText(data.url)
        setShowCreateDialog(false)
        fetchInvitations()
      }
    } catch (error) {
      console.error('Failed to create invitation:', error)
    }
  }

  const handleCopyLink = async (code: string) => {
    const url = `${window.location.origin}/invite/${code}`
    await navigator.clipboard.writeText(url)
    toast({
      title: '链接已复制',
      description: url,
    })
  }

  const handleShowQr = async (invitation: Invitation) => {
    setSelectedInvitation(invitation)
    const url = `/api/invitations/${invitation.code}/qr`
    setQrCodeUrl(url)
    setShowQrDialog(true)
  }

  const handleRevoke = async (invitationId: string) => {
    try {
      const response = await fetch(`/api/invitations/${invitationId}/revoke`, {
        method: 'POST',
      })

      if (response.ok) {
        toast({
          title: '邀请已撤销',
        })
        fetchInvitations()
      }
    } catch (error) {
      console.error('Failed to revoke invitation:', error)
    }
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle>房间邀请</CardTitle>
          <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
            <DialogTrigger asChild>
              <Button size="sm">
                <Plus className="h-4 w-4 mr-2" />
                创建邀请
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>创建邀请链接</DialogTitle>
                <DialogDescription>
                  创建一个新的邀请链接来邀请玩家加入房间
                </DialogDescription>
              </DialogHeader>

              <div className="space-y-4">
                <div>
                  <Label>邀请类型</Label>
                  <Select
                    value={createForm.type}
                    onValueChange={(value) => setCreateForm({ ...createForm, type: value })}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="link">链接</SelectItem>
                      <SelectItem value="one_time">一次性</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div>
                  <Label>角色</Label>
                  <Select
                    value={createForm.role}
                    onValueChange={(value) => setCreateForm({ ...createForm, role: value })}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="player">玩家</SelectItem>
                      <SelectItem value="co_kp">协作者(KP)</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div>
                  <Label>最大使用次数（可选）</Label>
                  <Input
                    type="number"
                    value={createForm.max_uses}
                    onChange={(e) => setCreateForm({ ...createForm, max_uses: e.target.value })}
                    placeholder="不限制"
                  />
                </div>

                <div>
                  <Label>有效期（小时，可选）</Label>
                  <Input
                    type="number"
                    value={createForm.expires_in_hours}
                    onChange={(e) => setCreateForm({ ...createForm, expires_in_hours: e.target.value })}
                    placeholder="永不过期"
                  />
                </div>

                <Button onClick={handleCreate} className="w-full">
                  创建并复制链接
                </Button>
              </div>
            </DialogContent>
          </Dialog>
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {invitations.map((invitation) => (
            <div
              key={invitation.id}
              className="flex items-center justify-between p-3 rounded-lg border"
            >
              <div className="flex-1">
                <div className="font-medium">{invitation.code}</div>
                <div className="text-sm text-muted-foreground">
                  {invitation.used_count} / {invitation.max_uses || '∞'} 次使用
                  {invitation.expires_at && ` · 过期: ${new Date(invitation.expires_at).toLocaleString()}`}
                </div>
              </div>

              <div className="flex gap-2">
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => handleCopyLink(invitation.code)}
                >
                  <Copy className="h-4 w-4" />
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => handleShowQr(invitation)}
                >
                  <QrCode className="h-4 w-4" />
                </Button>
                {invitation.is_active && (
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => handleRevoke(invitation.id)}
                  >
                    <Trash2 className="h-4 w-4 text-destructive" />
                  </Button>
                )}
              </div>
            </div>
          ))}

          {invitations.length === 0 && (
            <div className="text-center py-8 text-muted-foreground">
              暂无邀请链接
            </div>
          )}
        </div>
      </CardContent>

      <Dialog open={showQrDialog} onOpenChange={setShowQrDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>邀请二维码</DialogTitle>
            <DialogDescription>
              扫描二维码快速加入房间
            </DialogDescription>
          </DialogHeader>

          {selectedInvitation && (
            <div className="flex flex-col items-center space-y-4">
              <img src={qrCodeUrl} alt="QR Code" className="w-64 h-64" />
              <div className="text-center">
                <div className="font-medium">{selectedInvitation.code}</div>
                <div className="text-sm text-muted-foreground">
                  {selectedInvitation.used_count} / {selectedInvitation.max_uses || '∞'} 次使用
                </div>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </Card>
  )
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/db/models/invitation.py` | 创建 | 邀请数据模型 |
| `app/services/invitation.py` | 创建 | 邀请服务 |
| `app/api/invitation.py` | 创建 | 邀请 API |
| `frontend/src/components/room/RoomInvitation.tsx` | 创建 | 房间邀请组件 |
| `frontend/src/components/room/InvitationAccept.tsx` | 创建 | 邀请接受页面 |

---

## 验收标准

- [ ] 邀请码生成唯一
- [ ] 链接邀请正常
- [ ] 二维码显示正确
- [ ] 使用限制有效
- [ ] 过期检查准确
- [ ] 撤销功能正常

---

## 参考文档

- M2-001: 房间管理系统
- Discord 邀请系统设计

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
