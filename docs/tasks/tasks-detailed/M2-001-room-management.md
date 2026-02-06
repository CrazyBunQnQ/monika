# M2-001: 实现房间管理系统

**任务ID**: M2-001
**标题**: 实现房间管理系统
**类型**: backend (后端开发)
**预估工时**: 3h
**依赖**: M1-090, M2-022

---

## 任务描述

实现多人游戏房间管理系统，包括房间创建、加入、离开、状态同步等功能。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M2-001-01 | 设计房间数据模型 | Room Schema | 25min |
| M2-001-02 | 实现房间创建 API | POST /rooms | 30min |
| M2-001-03 | 实现房间加入 API | POST /rooms/:id/join | 25min |
| M2-001-04 | 实现房间离开 API | POST /rooms/:id/leave | 20min |
| M2-001-05 | 实现房间列表 API | GET /rooms | 20min |
| M2-001-06 | 实现 WebSocket 连接 | Socket.io 集成 | 35min |
| M2-001-07 | 实现房间状态同步 | State Sync | 30min |
| M2-001-08 | 实现房间事件广播 | Event Broadcast | 25min |
| M2-001-09 | 编写房间管理测试 | 测试覆盖 | 30min |

---

## 房间数据模型

```python
# app/db/models/room.py
from sqlalchemy import Column, String, Integer, Boolean, DateTime, JSON, ForeignKey
from sqlalchemy.orm import relationship
from sqlalchemy.sql import func
from app.db.database import Base

class Room(Base):
    """游戏房间"""
    __tablename__ = "rooms"

    id = Column(String, primary_key=True, index=True)
    name = Column(String, nullable=False)
    description = Column(String, nullable=True)

    # 房间设置
    max_players = Column(Integer, default=6)
    is_private = Column(Boolean, default=False)
    password = Column(String, nullable=True)

    # 房间状态
    status = Column(String, default="waiting")  # waiting, playing, paused, ended

    # 房主
    owner_id = Column(String, ForeignKey("users.id"), nullable=False)
    campaign_id = Column(String, ForeignKey("campaigns.id"), nullable=True)

    # 房间状态数据
    state_data = Column(JSON, nullable=True)

    # 时间戳
    created_at = Column(DateTime, default=func.now(), nullable=False)
    updated_at = Column(DateTime, default=func.now(), onupdate=func.now(), nullable=False)

    # 关系
    owner = relationship("User", back_populates="owned_rooms")
    campaign = relationship("Campaign", back_populates="rooms")
    participants = relationship("RoomParticipant", back_populates="room", cascade="all, delete-orphan")

    def add_participant(self, user_id: str, role: str = "player"):
        """添加参与者"""
        participant = RoomParticipant(
            room_id=self.id,
            user_id=user_id,
            role=role
        )
        self.participants.append(participant)
        return participant

    def remove_participant(self, user_id: str):
        """移除参与者"""
        self.participants = [p for p in self.participants if p.user_id != user_id]

    @property
    def current_players(self) -> int:
        """当前玩家数量"""
        return len(self.participants)

    @property
    def is_full(self) -> bool:
        """房间是否已满"""
        return self.current_players >= self.max_players

class RoomParticipant(Base):
    """房间参与者"""
    __tablename__ = "room_participants"

    id = Column(String, primary_key=True, index=True)
    room_id = Column(String, ForeignKey("rooms.id"), nullable=False)
    user_id = Column(String, ForeignKey("users.id"), nullable=False)

    # 参与者角色: kp, player
    role = Column(String, default="player")

    # 连接状态
    is_online = Column(Boolean, default=False)
    last_seen = Column(DateTime, default=func.now())

    # 角色卡关联
    character_id = Column(String, ForeignKey("characters.id"), nullable=True)

    # 时间戳
    joined_at = Column(DateTime, default=func.now(), nullable=False)

    # 关系
    room = relationship("Room", back_populates="participants")
    user = relationship("User", back_populates="room_participations")
    character = relationship("Character", back_populates="room_participations")
```

---

## 房间服务

```python
# app/services/room.py
from typing import Optional, List
from sqlalchemy.orm import Session
from app.db.models.room import Room, RoomParticipant
from app.db.models.user import User
from app.core.security import generate_id
import json

class RoomService:
    def __init__(self, db: Session):
        self.db = db

    def create_room(
        self,
        name: str,
        owner_id: str,
        description: Optional[str] = None,
        max_players: int = 6,
        is_private: bool = False,
        password: Optional[str] = None,
        campaign_id: Optional[str] = None,
    ) -> Room:
        """创建房间"""
        room = Room(
            id=generate_id("room"),
            name=name,
            description=description,
            owner_id=owner_id,
            max_players=max_players,
            is_private=is_private,
            password=password,
            campaign_id=campaign_id,
            state_data={
                "phase": "waiting",  # waiting, setup, playing
                "turn_order": [],
                "current_scene": None,
            }
        )

        self.db.add(room)
        self.db.commit()
        self.db.refresh(room)

        # 添加房主为参与者
        room.add_participant(owner_id, role="kp")
        self.db.commit()

        return room

    def get_room(self, room_id: str) -> Optional[Room]:
        """获取房间"""
        return self.db.query(Room).filter(Room.id == room_id).first()

    def list_rooms(
        self,
        skip: int = 0,
        limit: int = 20,
        status: Optional[str] = None,
        include_full: bool = False,
    ) -> List[Room]:
        """列出房间"""
        query = self.db.query(Room)

        if status:
            query = query.filter(Room.status == status)

        if not include_full:
            # 只显示未满的房间
            query = query.filter(
                Room.id.notin_(
                    self.db.query(RoomParticipant.room_id)
                    .group_by(RoomParticipant.room_id)
                    .having(
                        func.count(RoomParticipant.user_id) >= Room.max_players
                    )
                )
            )

        return query.offset(skip).limit(limit).all()

    def join_room(
        self,
        room_id: str,
        user_id: str,
        password: Optional[str] = None,
        role: str = "player",
    ) -> Optional[Room]:
        """加入房间"""
        room = self.get_room(room_id)
        if not room:
            return None

        # 检查密码
        if room.is_private and room.password != password:
            return None

        # 检查房间是否已满
        if room.is_full:
            return None

        # 检查是否已参与
        existing = self.db.query(RoomParticipant).filter(
            RoomParticipant.room_id == room_id,
            RoomParticipant.user_id == user_id
        ).first()

        if existing:
            # 更新在线状态
            existing.is_online = True
            existing.role = role
        else:
            # 添加新参与者
            room.add_participant(user_id, role)

        self.db.commit()
        self.db.refresh(room)

        return room

    def leave_room(self, room_id: str, user_id: str) -> bool:
        """离开房间"""
        room = self.get_room(room_id)
        if not room:
            return False

        room.remove_participant(user_id)

        # 如果房主离开，转移房主或删除房间
        if room.owner_id == user_id:
            if room.participants:
                # 转移房主给第一个参与者
                room.owner_id = room.participants[0].user_id
            else:
                # 删除空房间
                self.db.delete(room)

        self.db.commit()
        return True

    def update_room_state(
        self,
        room_id: str,
        state_data: dict,
    ) -> Optional[Room]:
        """更新房间状态"""
        room = self.get_room(room_id)
        if not room:
            return None

        room.state_data = {**(room.state_data or {}), **state_data}
        self.db.commit()
        self.db.refresh(room)

        return room
```

---

## 房间 API

```python
# app/api/room.py
from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.orm import Session
from pydantic import BaseModel
from typing import Optional, List

from app.db.database import get_db
from app.services.room import RoomService
from app.api.deps.auth import get_current_user
from app.db.models.user import User

router = APIRouter(prefix="/rooms", tags=["rooms"])

# Request/Response Models
class RoomCreate(BaseModel):
    name: str
    description: Optional[str] = None
    max_players: int = 6
    is_private: bool = False
    password: Optional[str] = None
    campaign_id: Optional[str] = None

class RoomResponse(BaseModel):
    id: str
    name: str
    description: Optional[str]
    max_players: int
    current_players: int
    is_private: bool
    status: str
    owner_id: str
    campaign_id: Optional[str]
    created_at: str

class RoomJoin(BaseModel):
    password: Optional[str] = None
    role: str = "player"

@router.post("", response_model=RoomResponse)
async def create_room(
    room_data: RoomCreate,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """创建房间"""
    service = RoomService(db)

    room = service.create_room(
        name=room_data.name,
        owner_id=current_user.id,
        description=room_data.description,
        max_players=room_data.max_players,
        is_private=room_data.is_private,
        password=room_data.password,
        campaign_id=room_data.campaign_id,
    )

    return RoomResponse(
        id=room.id,
        name=room.name,
        description=room.description,
        max_players=room.max_players,
        current_players=room.current_players,
        is_private=room.is_private,
        status=room.status,
        owner_id=room.owner_id,
        campaign_id=room.campaign_id,
        created_at=room.created_at.isoformat(),
    )

@router.get("", response_model=List[RoomResponse])
async def list_rooms(
    skip: int = 0,
    limit: int = 20,
    status: Optional[str] = None,
    include_full: bool = False,
    db: Session = Depends(get_db),
):
    """列出房间"""
    service = RoomService(db)
    rooms = service.list_rooms(skip=skip, limit=limit, status=status, include_full=include_full)

    return [
        RoomResponse(
            id=room.id,
            name=room.name,
            description=room.description,
            max_players=room.max_players,
            current_players=room.current_players,
            is_private=room.is_private,
            status=room.status,
            owner_id=room.owner_id,
            campaign_id=room.campaign_id,
            created_at=room.created_at.isoformat(),
        )
        for room in rooms
    ]

@router.post("/{room_id}/join", response_model=RoomResponse)
async def join_room(
    room_id: str,
    join_data: RoomJoin,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """加入房间"""
    service = RoomService(db)
    room = service.join_room(
        room_id=room_id,
        user_id=current_user.id,
        password=join_data.password,
        role=join_data.role,
    )

    if not room:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="无法加入房间"
        )

    return RoomResponse(
        id=room.id,
        name=room.name,
        description=room.description,
        max_players=room.max_players,
        current_players=room.current_players,
        is_private=room.is_private,
        status=room.status,
        owner_id=room.owner_id,
        campaign_id=room.campaign_id,
        created_at=room.created_at.isoformat(),
    )

@router.post("/{room_id}/leave")
async def leave_room(
    room_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """离开房间"""
    service = RoomService(db)
    success = service.leave_room(room_id=room_id, user_id=current_user.id)

    if not success:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="房间不存在"
        )

    return {"message": "已离开房间"}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/db/models/room.py` | 创建 | 房间数据模型 |
| `app/services/room.py` | 创建 | 房间服务 |
| `app/api/room.py` | 创建 | 房间 API |
| `tests/test_room.py` | 创建 | 房间测试 |

---

## 验收标准

- [ ] 房间创建功能正常
- [ ] 房间加入功能正常
- [ ] 房间离开功能正常
- [ ] 房间列表正确显示
- [ ] 密码保护有效
- [ ] 人数限制正确
- [ ] 测试覆盖全面

---

## 参考文档

- M1-090: 战役管理系统
- M2-022: Socket.io 服务配置
- Socket.io 文档

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
