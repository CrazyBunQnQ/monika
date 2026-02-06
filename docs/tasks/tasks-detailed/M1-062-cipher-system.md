# M1-062: 实现暗语/密语系统

**任务ID**: M1-062
**标题**: 实现暗语/密语系统
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M2-002

---

## 任务描述

实现暗语/密语系统，允许 KP 创建只有特定角色能看到的信息，用于秘密通信和隐藏剧情。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-062-01 | 设计暗语数据模型 | Data Model | 20min |
| M1-062-02 | 实现暗语服务 | Code Service | 30min |
| M1-062-03 | 实现暗语解析 | Parser | 25min |
| M1-062-04 | 实现权限过滤 | Permission Filter | 25min |
| M1-062-05 | 实现 WebSocket 过滤 | WS Filter | 20min |
| M1-062-06 | 编写暗语测试 | 测试覆盖 | 15min |

---

## 暗语数据模型

```python
# app/db/models/cipher.py
from sqlalchemy import Column, String, Text, ForeignKey, Boolean, JSON
from sqlalchemy.orm import relationship
from app.db.database import Base

class Cipher(Base):
    """暗语/密语"""
    __tablename__ = 'ciphers'

    id = Column(String, primary_key=True, index=True)
    room_id = Column(String, ForeignKey('rooms.id'), nullable=False, index=True)

    # 基本信息
    name = Column(String, nullable=False)
    description = Column(Text)

    # 暗语规则
    type = Column(String, nullable=False)  # tag, keyword, character
    pattern = Column(String, nullable=False)  # 匹配模式
    replacement = Column(String, nullable=False)  # 替换内容

    # 可见性设置
    visible_to = Column(JSON)  # ['player1', 'player2'] 或 ['role:kp']

    # 状态
    is_enabled = Column(Boolean, default=True, nullable=False)

    # 创建者
    created_by = Column(String, ForeignKey('users.id'), nullable=False)

    # 关系
    room = relationship("Room", back_populates="ciphers")
    creator = relationship("User", back_populates="ciphers")

    def __repr__(self):
        return f"<Cipher {self.name}>"
```

---

## 暗语服务

```python
# app/services/cipher.py
from typing import List, Dict, Any, Optional, Set
from sqlalchemy.orm import Session
import re

from app.db.models.cipher import Cipher
from app.core.security import generate_id

class CipherService:
    """暗语服务"""

    def __init__(self, db: Session):
        self.db = db

    def create_cipher(
        self,
        room_id: str,
        name: str,
        type: str,
        pattern: str,
        replacement: str,
        visible_to: List[str],
        created_by: str,
        description: str = None,
    ) -> Cipher:
        """创建暗语"""
        cipher = Cipher(
            id=generate_id('cipher'),
            room_id=room_id,
            name=name,
            type=type,
            pattern=pattern,
            replacement=replacement,
            visible_to=visible_to,
            description=description,
            created_by=created_by,
        )

        self.db.add(cipher)
        self.db.commit()
        self.db.refresh(cipher)

        return cipher

    def get_room_ciphers(
        self,
        room_id: str,
        enabled_only: bool = True,
    ) -> List[Cipher]:
        """获取房间暗语列表"""
        query = self.db.query(Cipher)\
            .filter(Cipher.room_id == room_id)

        if enabled_only:
            query = query.filter(Cipher.is_enabled == True)

        return query.all()

    def process_message(
        self,
        room_id: str,
        message: str,
        user_id: str,
        user_role: str,
    ) -> tuple[str, List[str]]:
        """处理消息，应用暗语"""
        ciphers = self.get_room_ciphers(room_id)

        original_message = message
        processed_message = message
        revealed_ciphers = []

        for cipher in ciphers:
            # 检查用户是否有权限看到此暗语
            if not self._can_view_cipher(cipher, user_id, user_role):
                # 隐藏暗语内容
                processed_message = self._apply_cipher_hide(
                    processed_message,
                    cipher,
                )
            else:
                # 显示暗语内容
                processed_message, revealed = self._apply_cipher_reveal(
                    processed_message,
                    cipher,
                )
                if revealed:
                    revealed_ciphers.append(cipher.name)

        return processed_message, revealed_ciphers

    def _can_view_cipher(
        self,
        cipher: Cipher,
        user_id: str,
        user_role: str,
    ) -> bool:
        """检查用户是否可以查看暗语"""
        if not cipher.visible_to:
            return True

        for item in cipher.visible_to:
            if item.startswith('role:'):
                # 角色检查
                required_role = item.split(':', 1)[1]
                if user_role == required_role:
                    return True
            else:
                # 用户 ID 检查
                if user_id == item:
                    return True

        return False

    def _apply_cipher_hide(
        self,
        message: str,
        cipher: Cipher,
    ) -> str:
        """隐藏暗语内容"""
        if cipher.type == 'tag':
            # 标签格式: [暗语名:内容] -> [???]
            pattern = rf'\[{cipher.name}:.+?\]'
            return re.sub(pattern, f'[???]', message)

        elif cipher.type == 'keyword':
            # 关键词格式: 暗语模式 -> ***
            pattern = cipher.pattern
            return re.sub(pattern, '***', message, flags=re.IGNORECASE)

        elif cipher.type == 'character':
            # 角色专属
            return message

        return message

    def _apply_cipher_reveal(
        self,
        message: str,
        cipher: Cipher,
    ) -> tuple[str, bool]:
        """揭示暗语内容"""
        revealed = False

        if cipher.type == 'tag':
            # 标签格式: [暗语名:内容] -> 内容
            pattern = rf'\[{cipher.name}:(.+?)\]'
            replacement = cipher.replacement or r'\1'

            def replacer(match):
                nonlocal revealed
                revealed = True
                content = match.group(1)
                if cipher.replacement == '{content}':
                    return content
                return replacement

            new_message = re.sub(pattern, replacer, message)
            return new_message, revealed

        elif cipher.type == 'keyword':
            # 关键词替换
            if cipher.replacement:
                new_message = re.sub(
                    cipher.pattern,
                    cipher.replacement,
                    message,
                    flags=re.IGNORECASE,
                )
                if new_message != message:
                    revealed = True
                return new_message, revealed

        return message, revealed

    def get_user_visible_message(
        self,
        room_id: str,
        message: str,
        user_id: str,
        user_role: str,
    ) -> str:
        """获取用户可见的消息内容"""
        processed, _ = self.process_message(
            room_id,
            message,
            user_id,
            user_role,
        )
        return processed

    def update_cipher(
        self,
        cipher_id: str,
        updates: Dict[str, Any],
    ) -> Optional[Cipher]:
        """更新暗语"""
        cipher = self.db.query(Cipher)\
            .filter(Cipher.id == cipher_id)\
            .first()

        if not cipher:
            return None

        for key, value in updates.items():
            if hasattr(cipher, key):
                setattr(cipher, key, value)

        self.db.commit()
        self.db.refresh(cipher)

        return cipher

    def delete_cipher(
        self,
        cipher_id: str,
    ) -> bool:
        """删除暗语"""
        cipher = self.db.query(Cipher)\
            .filter(Cipher.id == cipher_id)\
            .first()

        if not cipher:
            return False

        self.db.delete(cipher)
        self.db.commit()

        return True
```

---

## 暗语 API

```python
# app/api/ciphers.py
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session
from pydantic import BaseModel
from typing import List, Optional

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.api.deps.permissions import require_room_role
from app.db.models.user import User
from app.services.cipher import CipherService

router = APIRouter(prefix="/ciphers", tags=["ciphers"])

class CreateCipherRequest(BaseModel):
    room_id: str
    name: str
    type: str
    pattern: str
    replacement: str
    visible_to: List[str]
    description: Optional[str] = None

@router.post("")
async def create_cipher(
    request: CreateCipherRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """创建暗语"""
    perm_service = PermissionService(db)

    # 只有 KP 可以创建暗语
    if perm_service.get_user_role(request.room_id, current_user.id) != 'kp':
        raise HTTPException(status_code=403, detail="只有 KP 可以创建暗语")

    service = CipherService(db)
    cipher = service.create_cipher(
        room_id=request.room_id,
        name=request.name,
        type=request.type,
        pattern=request.pattern,
        replacement=request.replacement,
        visible_to=request.visible_to,
        created_by=current_user.id,
        description=request.description,
    )

    return {
        "id": cipher.id,
        "name": cipher.name,
        "type": cipher.type,
    }

@router.get("/room/{room_id}")
async def list_room_ciphers(
    room_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取房间暗语列表"""
    perm_service = PermissionService(db)

    # 只有 KP 可以查看所有暗语
    user_role = perm_service.get_user_role(room_id, current_user.id)
    if user_role != 'kp':
        raise HTTPException(status_code=403, detail="只有 KP 可以查看暗语")

    service = CipherService(db)
    ciphers = service.get_room_ciphers(room_id)

    return [
        {
            "id": c.id,
            "name": c.name,
            "type": c.type,
            "pattern": c.pattern,
            "replacement": c.replacement,
            "visible_to": c.visible_to,
            "description": c.description,
            "is_enabled": c.is_enabled,
        }
        for c in ciphers
    ]

@router.put("/{cipher_id}")
async def update_cipher(
    cipher_id: str,
    updates: dict,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """更新暗语"""
    service = CipherService(db)
    cipher = service.update_cipher(cipher_id, updates)

    if not cipher:
        raise HTTPException(status_code=404, detail="暗语不存在")

    return {"message": "暗语已更新"}

@router.delete("/{cipher_id}")
async def delete_cipher(
    cipher_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """删除暗语"""
    service = CipherService(db)
    success = service.delete_cipher(cipher_id)

    if not success:
        raise HTTPException(status_code=404, detail="暗语不存在")

    return {"message": "暗语已删除"}

@router.post("/preview")
async def preview_message(
    room_id: str,
    message: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """预览消息在暗语处理后的效果"""
    perm_service = PermissionService(db)
    user_role = perm_service.get_user_role(room_id, current_user.id)

    service = CipherService(db)
    processed, revealed = service.process_message(
        room_id,
        message,
        current_user.id,
        user_role,
    )

    return {
        "original": message,
        "processed": processed,
        "revealed_ciphers": revealed,
    }
```

---

## 使用示例

### 标签式暗语

```
KP 输入: "你发现了一个 [secret:刻有符文的古老钥匙]"
玩家A 看到: "你发现了一个 [secret:刻有符文的古老钥匙]"
玩家B 看到: "你发现了一个 [???]"
```

### 关键词暗语

```
暗语: { name: "真凶", type: "keyword", pattern: "管家", replacement: "凶手" }

原始消息: "管家在案发时不在场"
无权限玩家看到: "管家在案发时不在场"
有权限玩家看到: "凶手在案发时不在场"
```

### WebSocket 消息过滤

```python
# app/api/websocket/handlers/chat.py
async def send_chat_message(
    room_id: str,
    message: str,
    from_user_id: str,
    ws_manager,
    db: Session,
):
    cipher_service = CipherService(db)
    perm_service = PermissionService(db)

    # 获取房间所有成员
    participants = get_room_participants(room_id, db)

    # 为每个用户生成个性化的消息
    for participant in participants:
        user_role = perm_service.get_user_role(room_id, participant.user_id)

        # 应用暗语过滤
        processed_message = cipher_service.get_user_visible_message(
            room_id,
            message,
            participant.user_id,
            user_role,
        )

        # 发送个性化消息
        await ws_manager.send_to_user(
            participant.user_id,
            {
                "type": "chat_message",
                "message": processed_message,
                "from": from_user_id,
            },
        )
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/db/models/cipher.py` | 创建 | 暗语数据模型 |
| `app/services/cipher.py` | 创建 | 暗语服务 |
| `app/api/ciphers.py` | 创建 | 暗语 API |
| `frontend/src/components/game/CipherManager.tsx` | 创建 | 暗语管理组件 |

---

## 验收标准

- [ ] 暗语创建成功
- [ ] 权限过滤正确
- [ ] 消息处理准确
- [ ] WebSocket 过滤有效
- [ ] 暗语禁用正常
- [ ] 预览功能正确

---

## 参考文档

- M2-001: 房间管理系统
- M2-002: WebSocket 事件系统

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
