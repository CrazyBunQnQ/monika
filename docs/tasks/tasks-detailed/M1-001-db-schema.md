# M1-001: 数据库表结构设计

**任务类型**: db
**预估工时**: 4h
**依赖**: M0
**状态**: [ ]

---

## 子任务拆解

### 1.1 用户表设计 (40min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-001-01 | [ ] 创建 `app/db/models/user.py` | [ ] |
| M1-001-02 | [ ] 定义 `User` 模型 | [ ] |
| M1-001-03 | [ ] 定义 `UserProfile` 模型 | [ ] |
| M1-001-04 | [ ] 添加索引优化 | [ ] |

```python
# app/db/models/user.py
from datetime import datetime
from typing import Optional
from sqlmodel import SQLModel, Field, Relationship
from enum import Enum

class UserRole(str, Enum):
    PLAYER = "player"
    KEEPER = "keeper"
    ADMIN = "admin"

class UserBase(SQLModel):
    """用户基础模型"""
    username: str = Field(..., min_length=3, max_length=50, unique=True)
    email: str = Field(..., max_length=255, unique=True)
    role: UserRole = Field(default=UserRole.PLAYER)
    is_active: bool = Field(default=True)

class User(UserBase, table=True):
    """用户表"""
    id: Optional[int] = Field(default=None, primary_key=True)
    hashed_password: str = Field(..., max_length=255)
    created_at: datetime = Field(default_factory=datetime.utcnow)
    updated_at: datetime = Field(default_factory=datetime.utcnow)

    # 关系
    profile: Optional["UserProfile"] = Relationship(back_populates="user")
    characters: list["Character"] = Relationship(back_populates="owner")
    campaigns: list["Campaign"] = Relationship(back_populates="keeper")
    sessions: list["GameSession"] = Relationship(back_populates="user")

class UserProfile(SQLModel, table=True):
    """用户资料表"""
    id: Optional[int] = Field(default=None, primary_key=True)
    user_id: int = Field(foreign_key="user.id", unique=True)
    display_name: Optional[str] = Field(max_length=100)
    avatar_url: Optional[str] = Field(max_length=500)
    bio: Optional[str] = Field(max_length=1000)
    timezone: str = Field(default="UTC", max_length=50)
    language: str = Field(default="zh-CN", max_length=10)
    created_at: datetime = Field(default_factory=datetime.utcnow)
    updated_at: datetime = Field(default_factory=datetime.utcnow)

    # 关系
    user: User = Relationship(back_populates="profile")
```

---

### 1.2 角色卡表设计 (60min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-001-05 | [ ] 创建 `app/db/models/character.py` | [ ] |
| M1-001-06 | [ ] 定义 `Character` 模型 | [ ] |
| M1-001-07 | [ ] 定义 `CharacterSkill` 模型 | [ ] |
| M1-001-08 | [ ] 定义 `CharacterAttribute` 模型 | [ ] |

```python
# app/db/models/character.py
from datetime import datetime
from typing import Optional
from sqlmodel import SQLModel, Field, Relationship
from enum import Enum

class Gender(str, Enum):
    MALE = "male"
    FEMALE = "female"
    OTHER = "other"
    UNKNOWN = "unknown"

class CharacterStatus(str, Enum):
    ACTIVE = "active"         # 活跃
    DEAD = "dead"             # 死亡
    MISSING = "missing"       # 失踪
    INSANE = "insane"         # 疯狂
    RETIRED = "retired"       # 退休

class CharacterBase(SQLModel):
    """角色基础"""
    name: str = Field(..., max_length=100)
    age: Optional[int] = None
    gender: Gender = Field(default=Gender.UNKNOWN)
    occupation: Optional[str] = Field(max_length=100)
    residence: Optional[str] = Field(max_length=200)
    backstory: Optional[str] = Field(max_length=5000)

class Character(CharacterBase, table=True):
    """角色卡表"""
    id: Optional[int] = Field(default=None, primary_key=True)
    owner_id: int = Field(foreign_key="user.id", index=True)
    campaign_id: Optional[int] = Field(foreign_key="campaign.id", index=True, default=None)

    # 核心属性
    str: int = Field(default=50, ge=0, le=100)
    con: int = Field(default=50, ge=0, le=100)
    pow: int = Field(default=50, ge=0, le=100)
    dex: int = Field(default=50, ge=0, le=100)
    app: int = Field(default=50, ge=0, le=100)
    san: int = Field(default=99, ge=0, le=100)
    edu: int = Field(default=50, ge=0, le=100)
    luck: int = Field(default=50, ge=0, le=100)
    hp: int = Field(default=10, ge=0)
    mp: int = Field(default=10, ge=0)

    # 状态
    status: CharacterStatus = Field(default=CharacterStatus.ACTIVE)
    hp_current: int = Field(default=10)
    mp_current: int = Field(default=10)
    san_current: int = Field(default=99)

    # 时间戳
    created_at: datetime = Field(default_factory=datetime.utcnow)
    updated_at: datetime = Field(default_factory=datetime.utcnow)

    # 关系
    owner: "User" = Relationship(back_populates="characters")
    campaign: Optional["Campaign"] = Relationship(back_populates="characters")
    skills: list["CharacterSkill"] = Relationship(back_populates="character")
    attributes: list["CharacterAttribute"] = Relationship(back_populates="character")

class CharacterSkill(SQLModel, table=True):
    """角色技能表"""
    id: Optional[int] = Field(default=None, primary_key=True)
    character_id: int = Field(foreign_key="character.id", index=True)
    skill_key: str = Field(..., max_length=50)
    skill_name: str = Field(..., max_length=100)
    base_value: int = Field(default=0, ge=0)
    occupation_bonus: int = Field(default=0)
    personal_interest_bonus: int = Field(default=0)
    growth_bonus: int = Field(default=0)

    # 关系
    character: Character = Relationship(back_populates="skills")

class CharacterAttribute(SQLModel, table=True):
    """角色属性扩展表"""
    id: Optional[int] = Field(default=None, primary_key=True)
    character_id: int = Field(foreign_key="character.id", index=True)
    attribute_key: str = Field(..., max_length=50)
    attribute_value: str = Field(max_length=500)

    # 关系
    character: Character = Relationship(back_populates="attributes")
```

---

### 1.3 战役(Campaign)表设计 (40min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-001-09 | [ ] 创建 `app/db/models/campaign.py` | [ ] |
| M1-001-10 | [ ] 定义 `Campaign` 模型 | [ ] |
| M1-001-11 | [ ] 定义 `CampaignMember` 模型 | [ ] |
| M1-001-12 | [ ] 定义 `GameSession` 模型 | [ ] |

```python
# app/db/models/campaign.py
from datetime import datetime
from typing import Optional
from sqlmodel import SQLModel, Field, Relationship
from enum import Enum

class CampaignVisibility(str, Enum):
    PUBLIC = "public"
    PRIVATE = "private"
    INVITE_ONLY = "invite_only"

class CampaignStatus(str, Enum):
    DRAFT = "draft"
    ACTIVE = "active"
    COMPLETED = "completed"
    ARCHIVED = "archived"

class CampaignRole(str, Enum):
    KEEPER = "keeper"        # 守密人
    PLAYER = "player"        # 玩家
    OBSERVER = "observer"    # 观察者

class CampaignBase(SQLModel):
    """战役基础"""
    name: str = Field(..., max_length=200)
    description: Optional[str] = Field(max_length=2000)
    scenario_name: Optional[str] = Field(max_length=200)  # 模组名称
    visibility: CampaignVisibility = Field(default=CampaignVisibility.PRIVATE)

class Campaign(CampaignBase, table=True):
    """战役表"""
    id: Optional[int] = Field(default=None, primary_key=True)
    keeper_id: int = Field(foreign_key="user.id", index=True)

    status: CampaignStatus = Field(default=CampaignStatus.DRAFT)
    current_session: int = Field(default=0)

    # 时间
    created_at: datetime = Field(default_factory=datetime.utcnow)
    updated_at: Field(default_factory=datetime.utcnow)
    started_at: Optional[datetime] = None
    ended_at: Optional[datetime] = None

    # 关系
    keeper: "User" = Relationship(back_populates="campaigns")
    characters: list["Character"] = Relationship(back_populates="campaign")
    members: list["CampaignMember"] = Relationship(back_populates="campaign")
    sessions: list["GameSession"] = Relationship(back_populates="campaign")
    game_state: Optional["GameState"] = Relationship(back_populates="campaign")

class CampaignMember(SQLModel, table=True):
    """战役成员表"""
    id: Optional[int] = Field(default=None, primary_key=True)
    campaign_id: int = Field(foreign_key="campaign.id", index=True)
    user_id: int = Field(foreign_key="user.id", index=True)
    role: CampaignRole = Field(default=CampaignRole.PLAYER)
    joined_at: datetime = Field(default_factory=datetime.utcnow)

    # 关系
    campaign: Campaign = Relationship(back_populates="members")
    user: "User" = Relationship(back_populates="campaigns")

class GameSession(SQLModel, table=True):
    """游戏回合表"""
    id: Optional[int] = Field(default=None, primary_key=True)
    campaign_id: int = Field(foreign_key="campaign.id", index=True)
    session_number: int = Field(default=1)
    title: Optional[str] = Field(max_length=200)
    summary: Optional[str] = Field(max_length=5000)

    # 时间
    started_at: datetime = Field(default_factory=datetime.utcnow)
    ended_at: Optional[datetime] = None
    duration_minutes: Optional[int] = None

    # 关系
    campaign: Campaign = Relationship(back_populates="sessions")
    user: "User" = Relationship(back_populates="sessions")
    logs: list["GameLog"] = Relationship(back_populates="session")
```

---

### 1.4 游戏状态表设计 (50min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-001-13 | [ ] 创建 `app/db/models/gamestate.py` | [ ] |
| M1-001-14 | [ ] 定义 `GameState` 模型 | [ ] |
| M1-001-15 | [ ] 定义 `GameLog` 模型 | [ ] |
| M1-001-16 | [ ] 定义 `DiceRoll` 模型 | [ ] |

```python
# app/db/models/gamestate.py
from datetime import datetime
from typing import Optional
from sqlmodel import SQLModel, Field, Relationship
from enum import Enum
import json

class GamePhase(str, Enum):
    EXPLORATION = "exploration"   # 探索
    COMBAT = "combat"             # 战斗
    CHASE = "chase"               # 追逐
    SOCIAL = "social"             # 社交
    EVENT = "event"               # 事件
    REST = "rest"                 # 休息

class GameLogType(str, Enum):
    NARRATIVE = "narrative"       # 叙述
    ACTION = "action"             # 动作
    CHECK = "check"                # 检定
    COMBAT = "combat"             # 战斗
    CHAT = "chat"                 # 聊天
    SYSTEM = "system"             # 系统

class GameState(SQLModel, table=True):
    """游戏状态表"""
    id: Optional[int] = Field(default=None, primary_key=True)
    campaign_id: int = Field(foreign_key="campaign.id", unique=True)

    # 当前阶段
    current_phase: GamePhase = Field(default=GamePhase.EXPLORATION)
    round_number: int = Field(default=0)
    turn_order: Optional[str] = Field(default=None)  # JSON 排序

    # 场景信息
    location_name: Optional[str] = Field(max_length=200)
    location_description: Optional[str] = Field(max_length=2000)

    # 状态数据 (JSON)
    state_data: str = Field(default="{}")  # JSON

    # 时间
    created_at: datetime = Field(default_factory=datetime.utcnow)
    updated_at: datetime = Field(default_factory=datetime.utcnow)

    # 关系
    campaign: "Campaign" = Relationship(back_populates="game_state")

class GameLog(SQLModel, table=True):
    """游戏日志表"""
    id: Optional[int] = Field(default=None, primary_key=True)
    session_id: int = Field(foreign_key="gamesession.id", index=True)
    character_id: Optional[int] = Field(foreign_key="character.id", index=True, default=None)
    log_type: GameLogType = Field(default=GameLogType.NARRATIVE)

    # 内容
    content: str = Field(max_length=5000)
    metadata: Optional[str] = Field(default=None)  # JSON

    # 时间
    created_at: datetime = Field(default_factory=datetime.utcnow)
    order_index: int = Field(default=0)

    # 关系
    session: GameSession = Relationship(back_populates="logs")
    character: Optional[Character] = None

class DiceRoll(SQLModel, table=True):
    """掷骰记录表"""
    id: Optional[int] = Field(default=None, primary_key=True)
    character_id: Optional[int] = Field(foreign_key="character.id", index=True, default=None)
    log_id: Optional[int] = Field(foreign_key="gamelog.id", index=True, default=None)

    # 掷骰信息
    dice_type: str = Field(max_length=20)  # "d100", "2d6" 等
    raw_rolls: str = Field(max_length=100)  # JSON: [70, 30]
    modifier: int = Field(default=0)
    final_value: int = Field(default=0)

    # 成功判定
    target_value: Optional[int] = None
    success_level: Optional[str] = Field(max_length=20)  # "success", "failure"
    is_critical: bool = Field(default=False)
    is_fumble: bool = Field(default=False)

    # 时间
    created_at: datetime = Field(default_factory=datetime.utcnow)
```

---

### 1.5 索引与约束 (20min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-001-17 | [ ] 添加复合索引 | [ ] |
| M1-001-18 | [ ] 添加外键约束 | [ ] |

```python
# 复合索引示例（在模型中定义）
class Character(SQLModel, table=True):
    # ...
    __table_args__ = (
        # 用户 + 战役 复合索引
        Index("idx_character_owner_campaign", "owner_id", "campaign_id"),
        # 状态索引
        Index("idx_character_status", "status"),
    )

class GameLog(SQLModel, table=True):
    # ...
    __table_args__ = (
        # 回合 + 排序 复合索引
        Index("idx_gamelog_session_order", "session_id", "order_index"),
        # 角色 + 时间 复合索引
        Index("idx_gamelog_character_time", "character_id", "created_at"),
    )
```

---

## 验收标准

- [ ] 所有表都有主键
- [ ] 外键关系正确
- [ ] 常用查询字段有索引
- [ ] 符合 PostgreSQL 最佳实践
- [ ] SQLModel 注解完整

---

## 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `app/db/models/user.py` | 创建 | 用户模型 |
| `app/db/models/character.py` | 创建 | 角色卡模型 |
| `app/db/models/campaign.py` | 创建 | 战役模型 |
| `app/db/models/gamestate.py` | 创建 | 游戏状态模型 |
| `app/db/__init__.py` | 修改 | 导出所有模型 |

---

## ER 关系图

```
User (1) ──── (n) Character      # 用户拥有多个角色
    │                           │
    └─── (n) Campaign ─── (n) Character  # 战役包含多个角色
         │
         └─── (n) GameSession
              │
              └─── (n) GameLog
```

```
User (1) ──── (1) UserProfile   # 用户有一份资料
```
