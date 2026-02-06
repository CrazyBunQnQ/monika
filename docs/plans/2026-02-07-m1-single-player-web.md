# M1: 单人 Web 版实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**目标**: 完成单人可玩的 Web 界面，实现检定、战斗、追逐闭环，支持用户认证、角色卡管理、自然语言交互和规则问答

**架构**: 前后端分离 - React + shadcn/ui 前端，FastAPI + Socket.io 后端，PostgreSQL + Redis 数据层，OpenAI API 提供 LLM 能力

**Tech Stack**: React 18, Vite, TypeScript, shadcn/ui, TailwindCSS, Zustand, Socket.io-client | FastAPI, SQLAlchemy, Alembic, Pydantic, OpenAI SDK, Agno

---

## Phase 1: 基础设施层 (Backend)

### Task 1: PostgreSQL 数据库初始化

**Files:**
- Create: `backend/core/database.py`
- Create: `backend/alembic/versions/001_initial.py`
- Create: `backend/core/config.py`
- Create: `docker-compose.yml`

**Step 1: 创建数据库配置文件**

```python
# backend/core/config.py
from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    # Database
    DATABASE_URL: str = "postgresql://user:pass@localhost:5432/monika"

    # Redis
    REDIS_URL: str = "redis://localhost:6379/0"

    # JWT
    JWT_SECRET_KEY: str
    JWT_ALGORITHM: str = "HS256"
    ACCESS_TOKEN_EXPIRE_MINUTES: int = 30

    # OpenAI
    OPENAI_API_KEY: str

    class Config:
        env_file = ".env"

settings = Settings()
```

**Step 2: 创建数据库连接**

```python
# backend/core/database.py
from sqlalchemy import create_engine
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import sessionmaker
from .config import settings

engine = create_engine(settings.DATABASE_URL)
SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)
Base = declarative_base()

def get_db():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()
```

**Step 3: 创建 Docker Compose 配置**

```yaml
# docker-compose.yml
version: '3.8'
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_USER: monika
      POSTGRES_PASSWORD: monika_pass
      POSTGRES_DB: monika
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

volumes:
  postgres_data:
  redis_data:
```

**Step 4: 启动数据库**

Run: `docker-compose up -d`
Expected: PostgreSQL and Redis containers running

**Step 5: 初始化 Alembic**

Run: `cd backend && alembic init alembic`
Expected: Alembic directory created

**Step 6: Commit**

```bash
git add backend/core/config.py backend/core/database.py docker-compose.yml
git commit -m "feat: add database configuration and docker setup"
```

---

### Task 2: 用户表结构设计

**Files:**
- Create: `backend/models/user.py`
- Create: `backend/schemas/user.py`

**Step 1: 创建用户模型**

```python
# backend/models/user.py
from sqlalchemy import Column, String, Boolean, DateTime
from sqlalchemy.sql import func
from ..core.database import Base

class User(Base):
    __tablename__ = "users"

    user_id = Column(String, primary_key=True, index=True)
    username = Column(String, unique=True, nullable=False, index=True)
    email = Column(String, unique=True, nullable=False, index=True)
    password_hash = Column(String, nullable=False)
    is_active = Column(Boolean, default=True)
    created_at = Column(DateTime(timezone=True), server_default=func.now())
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())
```

**Step 2: 创建用户 Schema**

```python
# backend/schemas/user.py
from pydantic import BaseModel, EmailStr
from datetime import datetime

class UserBase(BaseModel):
    username: str
    email: EmailStr

class UserCreate(UserBase):
    password: str

class UserResponse(UserBase):
    user_id: str
    is_active: bool
    created_at: datetime

    class Config:
        from_attributes = True

class Token(BaseModel):
    access_token: str
    refresh_token: str
    token_type: str = "bearer"
```

**Step 3: 创建 Alembic 迁移**

```python
# backend/alembic/versions/001_create_users.py
from alembic import op
import sqlalchemy as sa

def upgrade():
    op.create_table(
        'users',
        sa.Column('user_id', sa.String(), nullable=False),
        sa.Column('username', sa.String(), nullable=False),
        sa.Column('email', sa.String(), nullable=False),
        sa.Column('password_hash', sa.String(), nullable=False),
        sa.Column('is_active', sa.Boolean(), default=True),
        sa.Column('created_at', sa.DateTime(timezone=True), server_default=sa.text('NOW()')),
        sa.Column('updated_at', sa.DateTime(timezone=True), server_default=sa.text('NOW()')),
        sa.PrimaryKeyConstraint('user_id')
    )
    op.create_index('ix_users_username', 'users', ['username'], unique=True)
    op.create_index('ix_users_email', 'users', ['email'], unique=True)

def downgrade():
    op.drop_index('ix_users_email', table_name='users')
    op.drop_index('ix_users_username', table_name='users')
    op.drop_table('users')
```

**Step 4: 运行迁移**

Run: `alembic upgrade head`
Expected: users table created in PostgreSQL

**Step 5: Commit**

```bash
git add backend/models/user.py backend/schemas/user.py backend/alembic/versions/001_create_users.py
git commit -m "feat: add users table model and migration"
```

---

### Task 3: 密码哈希和验证

**Files:**
- Create: `backend/core/security.py`

**Step 1: 编写密码哈希函数**

```python
# backend/core/security.py
from passlib.context import CryptContext
from datetime import datetime, timedelta
from jose import JWTError, jwt
from .config import settings

pwd_context = CryptContext(schemes=["bcrypt"], deprecated="auto")

def verify_password(plain_password: str, hashed_password: str) -> bool:
    return pwd_context.verify(plain_password, hashed_password)

def get_password_hash(password: str) -> str:
    return pwd_context.hash(password)

def create_access_token(data: dict) -> str:
    to_encode = data.copy()
    expire = datetime.utcnow() + timedelta(minutes=settings.ACCESS_TOKEN_EXPIRE_MINUTES)
    to_encode.update({"exp": expire})
    encoded_jwt = jwt.encode(to_encode, settings.JWT_SECRET_KEY, algorithm=settings.JWT_ALGORITHM)
    return encoded_jwt

def create_refresh_token(data: dict) -> str:
    to_encode = data.copy()
    expire = datetime.utcnow() + timedelta(days=7)
    to_encode.update({"exp": expire})
    encoded_jwt = jwt.encode(to_encode, settings.JWT_SECRET_KEY, algorithm=settings.JWT_ALGORITHM)
    return encoded_jwt

def decode_token(token: str) -> dict | None:
    try:
        payload = jwt.decode(token, settings.JWT_SECRET_KEY, algorithms=[settings.JWT_ALGORITHM])
        return payload
    except JWTError:
        return None
```

**Step 2: 编写测试**

```python
# tests/test_security.py
import pytest
from backend.core.security import verify_password, get_password_hash, create_access_token, decode_token

def test_password_hash():
    password = "test_password"
    hashed = get_password_hash(password)
    assert verify_password(password, hashed) is True
    assert verify_password("wrong", hashed) is False

def test_jwt_token():
    data = {"sub": "user_123"}
    token = create_access_token(data)
    payload = decode_token(token)
    assert payload["sub"] == "user_123"
```

**Step 3: 运行测试**

Run: `pytest tests/test_security.py -v`
Expected: All tests pass

**Step 4: 安装依赖**

Run: `pip install passlib[bcrypt] python-jose[cryptography]`
Expected: Packages installed

**Step 5: Commit**

```bash
git add backend/core/security.py tests/test_security.py
git commit -m "feat: add password hashing and JWT utilities"
```

---

### Task 4: 认证依赖

**Files:**
- Create: `backend/api/dependencies.py`

**Step 1: 创建认证依赖**

```python
# backend/api/dependencies.py
from fastapi import Depends, HTTPException, status
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
from sqlalchemy.orm import Session
from ..core.database import get_db
from ..core.security import decode_token
from ..models.user import User

security = HTTPBearer()

def get_current_user(
    credentials: HTTPAuthorizationCredentials = Depends(security),
    db: Session = Depends(get_db)
) -> User:
    token = credentials.credentials
    payload = decode_token(token)
    if payload is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid authentication credentials"
        )

    user_id = payload.get("sub")
    if user_id is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid token payload"
        )

    user = db.query(User).filter(User.user_id == user_id).first()
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="User not found"
        )

    return user

def get_optional_user(
    credentials: HTTPAuthorizationCredentials = Depends(HTTPBearer(auto_error=False)),
    db: Session = Depends(get_db)
) -> User | None:
    if credentials is None:
        return None
    try:
        return get_current_user(credentials, db)
    except HTTPException:
        return None
```

**Step 2: Commit**

```bash
git add backend/api/dependencies.py
git commit -m "feat: add authentication dependencies"
```

---

### Task 5: 用户注册 API

**Files:**
- Create: `backend/api/auth.py`

**Step 1: 创建注册端点**

```python
# backend/api/auth.py
from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.orm import Session
from ..core.database import get_db
from ..core.security import get_password_hash, create_access_token, create_refresh_token
from ..models.user import User
from ..schemas.user import UserCreate, UserResponse, Token
import uuid

router = APIRouter(prefix="/auth", tags=["auth"])

@router.post("/register", response_model=UserResponse, status_code=status.HTTP_201_CREATED)
def register(user: UserCreate, db: Session = Depends(get_db)):
    # Check existing user
    existing = db.query(User).filter(
        (User.username == user.username) | (User.email == user.email)
    ).first()
    if existing:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Username or email already registered"
        )

    # Create user
    user_id = str(uuid.uuid4())
    hashed_password = get_password_hash(user.password)
    db_user = User(
        user_id=user_id,
        username=user.username,
        email=user.email,
        password_hash=hashed_password
    )
    db.add(db_user)
    db.commit()
    db.refresh(db_user)

    return db_user
```

**Step 2: 测试注册 API**

Run: `curl -X POST http://localhost:8000/auth/register -H "Content-Type: application/json" -d '{"username":"testuser","email":"test@example.com","password":"password123"}'`
Expected: User object returned with user_id

**Step 3: Commit**

```bash
git add backend/api/auth.py
git commit -m "feat: add user registration endpoint"
```

---

### Task 6: 用户登录 API

**Files:**
- Modify: `backend/api/auth.py`

**Step 1: 添加登录端点**

```python
# backend/api/auth.py (add to existing file)
@router.post("/login", response_model=Token)
def login(username: str, password: str, db: Session = Depends(get_db)):
    # Find user
    user = db.query(User).filter(User.username == username).first()
    if not user:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid username or password"
        )

    # Verify password
    from ..core.security import verify_password
    if not verify_password(password, user.password_hash):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid username or password"
        )

    # Create tokens
    access_token = create_access_token({"sub": user.user_id})
    refresh_token = create_refresh_token({"sub": user.user_id})

    return Token(access_token=access_token, refresh_token=refresh_token)
```

**Step 2: 测试登录**

Run: `curl -X POST "http://localhost:8000/auth/login?username=testuser&password=password123"`
Expected: Token response with access_token and refresh_token

**Step 3: Commit**

```bash
git add backend/api/auth.py
git commit -m "feat: add user login endpoint"
```

---

### Task 7: Token 刷新 API

**Files:**
- Modify: `backend/api/auth.py`

**Step 1: 添加刷新端点**

```python
# backend/api/auth.py (add to existing file)
class RefreshTokenRequest(BaseModel):
    refresh_token: str

@router.post("/refresh", response_model=Token)
def refresh_token(request: RefreshTokenRequest):
    payload = decode_token(request.refresh_token)
    if payload is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid refresh token"
        )

    user_id = payload.get("sub")
    if user_id is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid token payload"
        )

    # Create new tokens
    access_token = create_access_token({"sub": user_id})
    new_refresh_token = create_refresh_token({"sub": user_id})

    return Token(access_token=access_token, refresh_token=new_refresh_token)
```

**Step 2: Commit**

```bash
git add backend/api/auth.py
git commit -m "feat: add token refresh endpoint"
```

---

### Task 8: FastAPI 项目骨架

**Files:**
- Create: `backend/main.py`
- Create: `backend/api/__init__.py`

**Step 1: 创建主应用**

```python
# backend/main.py
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from .api import auth
from .core.database import Base, engine

# Create tables
Base.metadata.create_all(bind=engine)

app = FastAPI(title="Monika CoC TRPG Platform", version="1.0.0")

# CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:5173"],  # Vite dev server
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Routers
app.include_router(auth.router)

@app.get("/")
def root():
    return {"message": "Monika CoC TRPG Platform API"}

@app.get("/health")
def health():
    return {"status": "healthy"}
```

**Step 2: 测试运行**

Run: `uvicorn backend.main:app --reload --port 8000`
Expected: Server starts at http://localhost:8000

**Step 3: 测试健康检查**

Run: `curl http://localhost:8000/health`
Expected: {"status": "healthy"}

**Step 4: Commit**

```bash
git add backend/main.py backend/api/__init__.py
git commit -m "feat: initialize FastAPI application"
```

---

## Phase 2: 角色卡管理 (Backend)

### Task 9: 角色卡表结构

**Files:**
- Create: `backend/models/character.py`
- Create: `backend/schemas/character.py`

**Step 1: 创建角色卡模型**

```python
# backend/models/character.py
from sqlalchemy import Column, String, Integer, JSON, DateTime, ForeignKey, Enum as SQLEnum
from sqlalchemy.orm import relationship
from sqlalchemy.sql import func
from ..core.database import Base
import enum

class CharacterType(str, enum.Enum):
    PLAYER = "player"
    NPC = "npc"

class Character(Base):
    __tablename__ = "characters"

    character_id = Column(String, primary_key=True, index=True)
    player_id = Column(String, ForeignKey("users.user_id"), nullable=True)
    name = Column(String, nullable=False)
    type = Column(SQLEnum(CharacterType), nullable=False)

    # Core attributes
    core_attributes = Column(JSON, nullable=False)  # STR, DEX, INT, EDU, APP, POW, SIZ, CON
    derived_attributes = Column(JSON, nullable=False)  # HP, MP, SAN, Luck, Move, etc.
    skills = Column(JSON, nullable=False)  # {"common": {...}, "others": {...}}

    # Additional data
    inventory = Column(JSON, default=list)
    clues = Column(JSON, default=list)
    status = Column(JSON, nullable=False)

    # Timestamps
    created_at = Column(DateTime(timezone=True), server_default=func.now())
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())

    # Relationship
    player = relationship("User", backref="characters")
```

**Step 2: 创建角色卡 Schema**

```python
# backend/schemas/character.py
from pydantic import BaseModel, Field
from typing import Dict, Optional
from datetime import datetime

class CoreAttributes(BaseModel):
    STR: int = Field(ge=0, le=100)
    DEX: int = Field(ge=0, le=100)
    INT: int = Field(ge=0, le=100)
    EDU: int = Field(ge=0, le=100)
    APP: int = Field(ge=0, le=100)
    POW: int = Field(ge=0, le=100)
    SIZ: int = Field(ge=0, le=100)
    CON: int = Field(ge=0, le=100)

class DerivedAttributes(BaseModel):
    HP: int
    HP_max: int
    MP: int
    MP_max: int
    SAN: int
    SAN_max: int
    Luck: int
    Luck_max: int
    Move: int
    Build: int = 0
    BonusDamage: int = 0

class CharacterStatus(BaseModel):
    alive: bool = True
    conscious: bool = True
    dying: bool = False
    insane: bool = False
    conditions: list = []

class CharacterBase(BaseModel):
    name: str
    type: str = "player"
    core_attributes: CoreAttributes
    derived_attributes: DerivedAttributes
    skills: Dict[str, Dict[str, int]]
    inventory: list = []
    clues: list = []
    status: CharacterStatus

class CharacterCreate(CharacterBase):
    pass

class CharacterUpdate(CharacterBase):
    pass

class CharacterResponse(CharacterBase):
    character_id: str
    player_id: Optional[str]
    created_at: datetime
    updated_at: datetime

    class Config:
        from_attributes = True
```

**Step 3: 创建迁移**

```python
# backend/alembic/versions/002_create_characters.py
def upgrade():
    op.create_table(
        'characters',
        sa.Column('character_id', sa.String(), nullable=False),
        sa.Column('player_id', sa.String(), nullable=True),
        sa.Column('name', sa.String(), nullable=False),
        sa.Column('type', sa.Enum('player', 'npc', name='charactertype'), nullable=False),
        sa.Column('core_attributes', sa.JSON(), nullable=False),
        sa.Column('derived_attributes', sa.JSON(), nullable=False),
        sa.Column('skills', sa.JSON(), nullable=False),
        sa.Column('inventory', sa.JSON(), default=[]),
        sa.Column('clues', sa.JSON(), default=[]),
        sa.Column('status', sa.JSON(), nullable=False),
        sa.Column('created_at', sa.DateTime(timezone=True), server_default=sa.text('NOW()')),
        sa.Column('updated_at', sa.DateTime(timezone=True), server_default=sa.text('NOW()')),
        sa.ForeignKeyConstraint(['player_id'], ['users.user_id']),
        sa.PrimaryKeyConstraint('character_id')
    )
    op.create_index('ix_characters_player_id', 'characters', ['player_id'])

def downgrade():
    op.drop_index('ix_characters_player_id', table_name='characters')
    op.drop_table('characters')
```

**Step 4: 运行迁移**

Run: `alembic upgrade head`
Expected: characters table created

**Step 5: Commit**

```bash
git add backend/models/character.py backend/schemas/character.py
git commit -m "feat: add characters table and schemas"
```

---

### Task 10: 角色卡 CRUD API

**Files:**
- Create: `backend/api/characters.py`

**Step 1: 创建角色卡路由**

```python
# backend/api/characters.py
from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.orm import Session
from typing import List
import uuid
from ..core.database import get_db
from ..models.character import Character
from ..schemas.character import CharacterCreate, CharacterUpdate, CharacterResponse
from ..api.dependencies import get_current_user
from ..models.user import User

router = APIRouter(prefix="/characters", tags=["characters"])

@router.post("", response_model=CharacterResponse, status_code=status.HTTP_201_CREATED)
def create_character(
    character: CharacterCreate,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    character_id = str(uuid.uuid4())
    db_character = Character(
        character_id=character_id,
        player_id=current_user.user_id,
        **character.model_dump()
    )
    db.add(db_character)
    db.commit()
    db.refresh(db_character)
    return db_character

@router.get("", response_model=List[CharacterResponse])
def list_characters(
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    return db.query(Character).filter(Character.player_id == current_user.user_id).all()

@router.get("/{character_id}", response_model=CharacterResponse)
def get_character(
    character_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    character = db.query(Character).filter(
        Character.character_id == character_id,
        Character.player_id == current_user.user_id
    ).first()
    if not character:
        raise HTTPException(status_code=404, detail="Character not found")
    return character

@router.put("/{character_id}", response_model=CharacterResponse)
def update_character(
    character_id: str,
    character: CharacterUpdate,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    db_character = db.query(Character).filter(
        Character.character_id == character_id,
        Character.player_id == current_user.user_id
    ).first()
    if not db_character:
        raise HTTPException(status_code=404, detail="Character not found")

    for key, value in character.model_dump().items():
        setattr(db_character, key, value)

    db.commit()
    db.refresh(db_character)
    return db_character

@router.delete("/{character_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_character(
    character_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    db_character = db.query(Character).filter(
        Character.character_id == character_id,
        Character.player_id == current_user.user_id
    ).first()
    if not db_character:
        raise HTTPException(status_code=404, detail="Character not found")

    db.delete(db_character)
    db.commit()
```

**Step 2: 注册路由**

```python
# backend/main.py (add import and include)
from .api import characters

app.include_router(characters.router)
```

**Step 3: 测试创建角色卡**

Run: `curl -X POST http://localhost:8000/characters -H "Authorization: Bearer YOUR_TOKEN" -H "Content-Type: application/json" -d '{"name":"调查员A","type":"player","core_attributes":{"STR":50,"DEX":55,"INT":70,"EDU":65,"APP":40,"POW":60,"SIZ":50,"CON":45},"derived_attributes":{"HP":12,"HP_max":12,"MP":14,"MP_max":14,"SAN":60,"SAN_max":99,"Luck":50,"Luck_max":50,"Move":7,"Build":0,"BonusDamage":0},"skills":{"common":{"library_use":60,"spot_hidden":50}},"status":{"alive":true,"conscious":true,"dying":false,"insane":false,"conditions":[]}}'`

**Step 4: Commit**

```bash
git add backend/api/characters.py backend/main.py
git commit -m "feat: add character CRUD endpoints"
```

---

## Phase 3: 掷骰引擎 (Backend)

### Task 11: d100 随机数生成

**Files:**
- Create: `backend/services/dice.py`

**Step 1: 实现随机数生成**

```python
# backend/services/dice.py
from dataclasses import dataclass
from enum import Enum
from typing import Optional
import random

class SuccessLevel(str, Enum):
    CRITICAL = "critical"          # 大成功: 1
    EXTREME_SUCCESS = "extremeSuccess"  # 极难成功
    HARD_SUCCESS = "hardSuccess"    # 困难成功
    REGULAR_SUCCESS = "regularSuccess"  # 普通成功
    FAILURE = "failure"             # 失败
    FUMBLE = "fumble"              # 大失败: 100

@dataclass
class RollResult:
    roll: int
    skill_value: int
    success_level: SuccessLevel
    is_critical: bool = False
    is_fumble: bool = False

    @property
    def is_success(self) -> bool:
        return self.success_level in [
            SuccessLevel.CRITICAL,
            SuccessLevel.EXTREME_SUCCESS,
            SuccessLevel.HARD_SUCCESS,
            SuccessLevel.REGULAR_SUCCESS
        ]

def calculate_success_level(roll: int, skill_value: int) -> SuccessLevel:
    # 大成功
    if roll == 1:
        return SuccessLevel.CRITICAL

    # 大失败
    if roll == 100:
        return SuccessLevel.FUMBLE

    # 大失败 (技能 < 50 时掷出 96+)
    if skill_value < 50 and roll >= 96:
        return SuccessLevel.FUMBLE

    # 计算难度阈值
    extreme_threshold = skill_value // 5
    hard_threshold = skill_value // 2

    if roll <= extreme_threshold:
        return SuccessLevel.EXTREME_SUCCESS
    elif roll <= hard_threshold:
        return SuccessLevel.HARD_SUCCESS
    elif roll <= skill_value:
        return SuccessLevel.REGULAR_SUCCESS
    else:
        return SuccessLevel.FAILURE

def roll_d100() -> int:
    return random.randint(1, 100)

def roll_check(skill_value: int, bonus_dice: int = 0, penalty_dice: int = 0) -> RollResult:
    """
    进行检定
    skill_value: 技能值
    bonus_dice: 奖励骰数量 (取最小值)
    penalty_dice: 惩罚骰数量 (取最大值)
    """
    if bonus_dice > 0 and penalty_dice > 0:
        raise ValueError("Cannot have both bonus and penalty dice")

    # 基础掷骰
    base_roll = roll_d100()

    # 应用奖励骰 (取十位最小的)
    if bonus_dice > 0:
        rolls = [base_roll] + [roll_d100() for _ in range(bonus_dice)]
        # 找出十位最小的
        tens = [r // 10 for r in rolls]
        min_tens_index = tens.index(min(tens))
        final_roll = rolls[min_tens_index]
    # 应用惩罚骰 (取十位最大的)
    elif penalty_dice > 0:
        rolls = [base_roll] + [roll_d100() for _ in range(penalty_dice)]
        tens = [r // 10 for r in rolls]
        max_tens_index = tens.index(max(tens))
        final_roll = rolls[max_tens_index]
    else:
        final_roll = base_roll

    success_level = calculate_success_level(final_roll, skill_value)

    return RollResult(
        roll=final_roll,
        skill_value=skill_value,
        success_level=success_level,
        is_critical=success_level == SuccessLevel.CRITICAL,
        is_fumble=success_level == SuccessLevel.FUMBLE
    )
```

**Step 2: 编写测试**

```python
# tests/test_dice.py
import pytest
from backend.services.dice import roll_check, calculate_success_level, SuccessLevel

def test_critical_success():
    result = roll_check(skill_value=60, bonus_dice=0, penalty_dice=0)
    # Mock the random to return 1
    # For now just check the logic
    assert calculate_success_level(1, 60) == SuccessLevel.CRITICAL

def test_fumble_on_100():
    assert calculate_success_level(100, 60) == SuccessLevel.FUMBLE

def test_fumble_on_high_roll_low_skill():
    assert calculate_success_level(96, 40) == SuccessLevel.FUMBLE
    assert calculate_success_level(96, 60) == SuccessLevel.FAILURE

def test_extreme_success():
    assert calculate_success_level(10, 60) == SuccessLevel.EXTREME_SUCCESS

def test_hard_success():
    assert calculate_success_level(28, 60) == SuccessLevel.HARD_SUCCESS

def test_regular_success():
    assert calculate_success_level(55, 60) == SuccessLevel.REGULAR_SUCCESS

def test_failure():
    assert calculate_success_level(70, 60) == SuccessLevel.FAILURE
```

**Step 3: 运行测试**

Run: `pytest tests/test_dice.py -v`
Expected: All tests pass

**Step 4: Commit**

```bash
git add backend/services/dice.py tests/test_dice.py
git commit -m "feat: implement d100 dice engine"
```

---

### Task 12: 推骰机制

**Files:**
- Modify: `backend/services/dice.py`

**Step 1: 添加推骰函数**

```python
# backend/services/dice.py (add to existing file)
@dataclass
class PushResult:
    original_roll: RollResult
    pushed_roll: RollResult
    luck_spent: int = 0

def push_roll(original_result: RollResult, luck_available: int) -> PushResult:
    """
    推骰：失败后可以尝试再次检定，消耗幸运值
    """
    if original_result.is_success:
        raise ValueError("Cannot push a successful roll")

    if luck_available < 1:
        raise ValueError("Not enough luck to push")

    # 进行新的检定
    pushed_result = roll_check(original_result.skill_value)

    return PushResult(
        original_roll=original_result,
        pushed_roll=pushed_result,
        luck_spent=1
    )
```

**Step 2: Commit**

```bash
git add backend/services/dice.py
git commit -m "feat: add push roll mechanic"
```

---

### Task 13: 花幸运机制

**Files:**
- Modify: `backend/services/dice.py`

**Step 1: 添加花幸运函数**

```python
# backend/services/dice.py (add to existing file)
@dataclass
class SpendLuckResult:
    original_roll: RollResult
    new_roll: RollResult
    luck_spent: int

def spend_luck(original_result: RollResult, luck_amount: int, luck_available: int) -> SpendLuckResult:
    """
    花幸运：消耗幸运值修改检定结果
    每点幸运可以改变 1% 的检定结果
    """
    if luck_amount > luck_available:
        raise ValueError(f"Not enough luck (available: {luck_available}, requested: {luck_amount})")

    # 计算新的掷骰值
    if original_result.roll < original_result.skill_value:
        # 原本成功，可以降低骰子值
        new_roll = max(1, original_result.roll - luck_amount)
    else:
        # 原本失败，提高成功率
        new_roll = min(100, original_result.roll - luck_amount)

    new_result = RollResult(
        roll=new_roll,
        skill_value=original_result.skill_value,
        success_level=calculate_success_level(new_roll, original_result.skill_value)
    )

    return SpendLuckResult(
        original_roll=original_result,
        new_roll=new_result,
        luck_spent=luck_amount
    )
```

**Step 2: Commit**

```bash
git add backend/services/dice.py
git commit -m "feat: add spend luck mechanic"
```

---

### Task 14: 检定 API 路由

**Files:**
- Create: `backend/api/game.py`

**Step 1: 创建游戏路由**

```python
# backend/api/game.py
from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel
from typing import Optional
from ..services.dice import roll_check, push_roll, spend_luck, RollResult
from ..api.dependencies import get_current_user
from ..models.user import User

router = APIRouter(prefix="/game", tags=["game"])

class RollRequest(BaseModel):
    skill_value: int
    bonus_dice: int = 0
    penalty_dice: int = 0

class PushRequest(BaseModel):
    event_id: str  # 引用之前的事件
    luck_available: int

class SpendLuckRequest(BaseModel):
    event_id: str
    luck_amount: int
    luck_available: int

@router.post("/roll")
def make_roll(request: RollRequest, current_user: User = Depends(get_current_user)):
    """执行检定"""
    result = roll_check(
        skill_value=request.skill_value,
        bonus_dice=request.bonus_dice,
        penalty_dice=request.penalty_dice
    )
    return {
        "roll": result.roll,
        "skill_value": result.skill_value,
        "success_level": result.success_level,
        "is_critical": result.is_critical,
        "is_fumble": result.is_fumble,
        "is_success": result.is_success
    }

@router.post("/push")
def push_roll_endpoint(request: PushRequest, current_user: User = Depends(get_current_user)):
    """推骰（暂未实现事件存储，先返回基础响应）"""
    # TODO: 实现事件存储后，从 event_id 获取原始检定
    return {"message": "Push roll - event storage not yet implemented"}

@router.post("/luck")
def spend_luck_endpoint(request: SpendLuckRequest, current_user: User = Depends(get_current_user)):
    """花幸运（暂未实现事件存储）"""
    return {"message": "Spend luck - event storage not yet implemented"}
```

**Step 2: 注册路由**

```python
# backend/main.py (add import and include)
from .api import game

app.include_router(game.router)
```

**Step 3: 测试检定**

Run: `curl -X POST http://localhost:8000/game/roll -H "Authorization: Bearer YOUR_TOKEN" -H "Content-Type: application/json" -d '{"skill_value":60,"bonus_dice":0,"penalty_dice":0}'`

**Step 4: Commit**

```bash
git add backend/api/game.py backend/main.py
git commit -m "feat: add roll check endpoints"
```

---

## Phase 4: 消息处理与 LLM 集成 (Backend)

### Task 15: 意图识别服务

**Files:**
- Create: `backend/services/intent.py`

**Step 1: 创建意图分类**

```python
# backend/services/intent.py
from dataclasses import dataclass
from enum import Enum
from typing import Optional, Dict, Any
import re

class IntentType(str, Enum):
    ROLL = "roll"
    PUSH = "push"
    LUCK = "luck"
    HELP = "help"
    STATUS = "status"
    COMBAT = "combat"
    CHASE = "chase"
    SANITY = "sanity"
    RULE = "rule"
    CHAT = "chat"

@dataclass
class ParsedIntent:
    intent_type: IntentType
    parameters: Dict[str, Any]
    raw_message: str

class IntentRecognizer:
    # 命令模式
    PATTERNS = {
        IntentType.ROLL: [
            r'^/roll\s+(\w+)(?:\s*([+-]\d+))?$',
            r'^/roll\s+(\w+)(?:\s*bonus\s*=\s*(\d+))?$',
            r'^检定\s*(\w+)?$',
        ],
        IntentType.PUSH: [
            r'^/push$',
            r'^推骰$',
        ],
        IntentType.LUCK: [
            r'^/luck\s+(\d+)$',
            r'^花幸运\s*(\d+)?$',
        ],
        IntentType.HELP: [
            r'^/help$',
            r'^帮助$',
        ],
        IntentType.STATUS: [
            r'^/status$',
            r'^状态$',
        ],
        IntentType.RULE: [
            r'^/rule\s+(.+)$',
            r'^规则\s*(.+)?$',
        ],
    }

    @classmethod
    def recognize(cls, message: str) -> ParsedIntent:
        message = message.strip()

        # 尝试匹配命令模式
        for intent_type, patterns in cls.PATTERNS.items():
            for pattern in patterns:
                match = re.match(pattern, message, re.IGNORECASE)
                if match:
                    return cls._parse_intent(intent_type, match, message)

        # 默认为聊天
        return ParsedIntent(
            intent_type=IntentType.CHAT,
            parameters={},
            raw_message=message
        )

    @classmethod
    def _parse_intent(cls, intent_type: IntentType, match: re.Match, raw_message: str) -> ParsedIntent:
        if intent_type == IntentType.ROLL:
            groups = match.groups()
            skill = groups[0] if groups[0] else None
            modifier = groups[1] if len(groups) > 1 and groups[1] else None

            params = {"skill": skill}
            if modifier:
                params["modifier"] = modifier

            return ParsedIntent(intent_type, params, raw_message)

        elif intent_type == IntentType.LUCK:
            amount = int(match.group(1)) if match.group(1) else 1
            return ParsedIntent(intent_type, {"amount": amount}, raw_message)

        elif intent_type == IntentType.RULE:
            query = match.group(1) if match.group(1) else ""
            return ParsedIntent(intent_type, {"query": query}, raw_message)

        else:
            return ParsedIntent(intent_type, {}, raw_message)

def recognize_intent(message: str) -> ParsedIntent:
    return IntentRecognizer.recognize(message)
```

**Step 2: 编写测试**

```python
# tests/test_intent.py
import pytest
from backend.services.intent import recognize_intent, IntentType

def test_recognize_roll_command():
    result = recognize_intent("/roll library_use")
    assert result.intent_type == IntentType.ROLL
    assert result.parameters["skill"] == "library_use"

def test_recognize_roll_with_bonus():
    result = recognize_intent("/roll spot_hidden +1")
    assert result.intent_type == IntentType.ROLL
    assert result.parameters["modifier"] == "+1"

def test_recognize_push():
    result = recognize_intent("/push")
    assert result.intent_type == IntentType.PUSH

def test_recognize_luck():
    result = recognize_intent("/luck 5")
    assert result.intent_type == IntentType.LUCK
    assert result.parameters["amount"] == 5

def test_recognize_help():
    result = recognize_intent("/help")
    assert result.intent_type == IntentType.HELP

def test_recognize_chat():
    result = recognize_intent("我想检查书架上的书")
    assert result.intent_type == IntentType.CHAT
```

**Step 3: 运行测试**

Run: `pytest tests/test_intent.py -v`
Expected: All tests pass

**Step 4: Commit**

```bash
git add backend/services/intent.py tests/test_intent.py
git commit -m "feat: add intent recognition service"
```

---

### Task 16: 消息处理服务

**Files:**
- Create: `backend/services/message.py`

**Step 1: 创建消息处理器**

```python
# backend/services/message.py
from dataclasses import dataclass
from typing import Optional, Dict, Any
from .intent import recognize_intent, ParsedIntent, IntentType
from .dice import roll_check, RollResult

@dataclass
class MessageResponse:
    content: str
    event_type: str
    data: Optional[Dict[str, Any]] = None

class MessageProcessor:
    def __init__(self, character_state: Optional[Dict] = None):
        self.character_state = character_state or {}

    def process(self, message: str) -> MessageResponse:
        intent = recognize_intent(message)

        if intent.intent_type == IntentType.ROLL:
            return self._handle_roll(intent)
        elif intent.intent_type == IntentType.PUSH:
            return self._handle_push(intent)
        elif intent.intent_type == IntentType.LUCK:
            return self._handle_luck(intent)
        elif intent.intent_type == IntentType.HELP:
            return self._handle_help()
        elif intent.intent_type == IntentType.STATUS:
            return self._handle_status()
        elif intent.intent_type == IntentType.RULE:
            return self._handle_rule(intent)
        elif intent.intent_type == IntentType.CHAT:
            return self._handle_chat(intent)
        else:
            return MessageResponse(
                content="Unknown command",
                event_type="error"
            )

    def _handle_roll(self, intent: ParsedIntent) -> MessageResponse:
        skill = intent.parameters.get("skill")
        if not skill:
            return MessageResponse(
                content="请指定要检定的技能，例如：/roll library_use",
                event_type="error"
            )

        # 从角色状态获取技能值
        skill_value = self._get_skill_value(skill)

        result = roll_check(skill_value)

        success_text = self._format_success(result)

        return MessageResponse(
            content=f"【检定】{skill} ({skill_value}%): 掷出 {result.roll} → {success_text}",
            event_type="roll",
            data={
                "skill": skill,
                "skill_value": skill_value,
                "roll": result.roll,
                "success_level": result.success_level,
                "is_critical": result.is_critical,
                "is_fumble": result.is_fumble
            }
        )

    def _handle_push(self, intent: ParsedIntent) -> MessageResponse:
        return MessageResponse(
            content="推骰功能 - 需要引用之前的检定事件",
            event_type="push"
        )

    def _handle_luck(self, intent: ParsedIntent) -> MessageResponse:
        amount = intent.parameters.get("amount", 1)
        return MessageResponse(
            content=f"花幸运 {amount} 点 - 需要引用之前的检定事件",
            event_type="luck",
            data={"amount": amount}
        )

    def _handle_help(self) -> MessageResponse:
        help_text = """
# 可用命令

## 检定
- `/roll <技能名>` - 技能检定
- `/roll <技能名> +1` - 带奖励骰
- `/roll <技能名> -1` - 带惩罚骰

## 特殊
- `/push` - 推骰（失败后）
- `/luck [数量]` - 花幸运
- `/status` - 查看状态
- `/rule <关键词>` - 查询规则
- `/help` - 显示帮助
        """
        return MessageResponse(
            content=help_text.strip(),
            event_type="help"
        )

    def _handle_status(self) -> MessageResponse:
        if not self.character_state:
            return MessageResponse(
                content="暂无角色状态",
                event_type="status"
            )

        return MessageResponse(
            content=self._format_status(),
            event_type="status",
            data=self.character_state
        )

    def _handle_rule(self, intent: ParsedIntent) -> MessageResponse:
        query = intent.parameters.get("query", "")
        # TODO: 实现规则库查询
        return MessageResponse(
            content=f"规则查询「{query}」- 规则库尚未实现",
            event_type="rule",
            data={"query": query}
        )

    def _handle_chat(self, intent: ParsedIntent) -> MessageResponse:
        return MessageResponse(
            content=f"你说：{intent.raw_message}",
            event_type="chat"
        )

    def _get_skill_value(self, skill: str) -> int:
        # 从 character_state 获取技能值
        skills = self.character_state.get("skills", {}).get("common", {})
        return skills.get(skill, 50)  # 默认 50

    def _format_success(self, result: RollResult) -> str:
        labels = {
            "critical": "🌟 大成功！",
            "extremeSuccess": "✅ 极难成功",
            "hardSuccess": "👍 困难成功",
            "regularSuccess": "✅ 成功",
            "failure": "❌ 失败",
            "fumble": "💀 大失败！"
        }
        return labels.get(result.success_level, "未知")

    def _format_status(self) -> str:
        state = self.character_state
        derived = state.get("derived_attributes", {})

        return f"""
### {state.get('name', '未知')}

**状态**: HP {derived.get('HP', 0)}/{derived.get('HP_max', 0)} |
SAN {derived.get('SAN', 0)}/{derived.get('SAN_max', 0)} |
幸运 {derived.get('Luck', 0)}/{derived.get('Luck_max', 0)}
        """.strip()

def process_message(message: str, character_state: Optional[Dict] = None) -> MessageResponse:
    processor = MessageProcessor(character_state)
    return processor.process(message)
```

**Step 2: 测试消息处理**

```python
# backend/api/game.py (add endpoint)
class MessageRequest(BaseModel):
    message: str
    session_id: Optional[str] = None

@router.post("/message")
def process_game_message(request: MessageRequest, current_user: User = Depends(get_current_user)):
    from ..services.message import process_message

    # TODO: 从 session 获取角色状态
    response = process_message(request.message)

    return {
        "event_id": "evt_001",  # 临时
        "timestamp": "...",      # 临时
        "content": response.content,
        "event_type": response.event_type,
        "data": response.data
    }
```

**Step 3: Commit**

```bash
git add backend/services/message.py backend/api/game.py
git commit -m "feat: add message processing service"
```

---

### Task 17: LLM 集成

**Files:**
- Create: `backend/services/llm.py`

**Step 1: 创建 LLM 服务**

```python
# backend/services/llm.py
from openai import OpenAI
from ..core.config import settings

client = OpenAI(api_key=settings.OPENAI_API_KEY)

SYSTEM_PROMPT = """你是 Monika，一个克苏鲁的呼唤（CoC 7th Edition）TRPG 的 AI 游戏助手。

你的职责：
1. 作为 KP（守密人）推进游戏剧情
2. 解析玩家的自然语言输入
3. 进行检定和计算
4. 生成沉浸式的叙事

重要规则：
- 检定结果用指定格式返回
- 保持神秘和恐怖氛围
- 不要替玩家做决定
- 失败也是故事的一部分
"""

def generate_response(user_message: str, game_context: str = "") -> str:
    """生成 LLM 响应"""

    messages = [
        {"role": "system", "content": SYSTEM_PROMPT},
    ]

    if game_context:
        messages.append({
            "role": "system",
            "content": f"当前游戏状态：\n{game_context}"
        })

    messages.append({
        "role": "user",
        "content": user_message
    })

    response = client.chat.completions.create(
        model="gpt-4o-mini",
        messages=messages,
        temperature=0.8,
        max_tokens=500
    )

    return response.choices[0].message.content

def generate_response_stream(user_message: str, game_context: str = ""):
    """生成流式响应"""

    messages = [
        {"role": "system", "content": SYSTEM_PROMPT},
    ]

    if game_context:
        messages.append({
            "role": "system",
            "content": f"当前游戏状态：\n{game_context}"
        })

    messages.append({
        "role": "user",
        "content": user_message
    })

    stream = client.chat.completions.create(
        model="gpt-4o-mini",
        messages=messages,
        temperature=0.8,
        max_tokens=500,
        stream=True
    )

    return stream
```

**Step 2: 添加流式响应端点**

```python
# backend/api/game.py (add streaming endpoint)
from fastapi.responses import StreamingResponse
from ..services.llm import generate_response_stream

@router.post("/message/stream")
def process_message_stream(request: MessageRequest, current_user: User = Depends(get_current_user)):
    def generate():
        stream = generate_response_stream(request.message)
        for chunk in stream:
            if chunk.choices[0].delta.content:
                yield chunk.choices[0].delta.content

    return StreamingResponse(generate(), media_type="text/plain")
```

**Step 3: Commit**

```bash
git add backend/services/llm.py backend/api/game.py
git commit -m "feat: add OpenAI LLM integration"
```

---

## Phase 5: WebSocket 实现 (Backend)

### Task 18: WebSocket 连接

**Files:**
- Create: `backend/websocket/manager.py`
- Create: `backend/api/websocket.py`

**Step 1: 创建 WebSocket 管理器**

```python
# backend/websocket/manager.py
from fastapi import WebSocket
from typing import Dict, Set
import json

class ConnectionManager:
    def __init__(self):
        self.active_connections: Dict[str, WebSocket] = {}

    async def connect(self, user_id: str, websocket: WebSocket):
        await websocket.accept()
        self.active_connections[user_id] = websocket

    def disconnect(self, user_id: str):
        if user_id in self.active_connections:
            del self.active_connections[user_id]

    async def send_personal(self, user_id: str, message: dict):
        if user_id in self.active_connections:
            await self.active_connections[user_id].send_json(message)

    async def broadcast(self, message: dict, exclude: Set[str] = None):
        exclude = exclude or set()
        for user_id, connection in self.active_connections.items():
            if user_id not in exclude:
                await connection.send_json(message)

manager = ConnectionManager()
```

**Step 2: 创建 WebSocket 端点**

```python
# backend/api/websocket.py
from fastapi import APIRouter, WebSocket, WebSocketDisconnect, Query
from ..websocket.manager import manager
from ..core.security import decode_token

router = APIRouter(prefix="/ws", tags=["websocket"])

@router.websocket("")
async def websocket_endpoint(
    websocket: WebSocket,
    token: str = Query(...)
):
    # 验证 token
    payload = decode_token(token)
    if not payload:
        await websocket.close(code=1008, reason="Invalid token")
        return

    user_id = payload.get("sub")
    if not user_id:
        await websocket.close(code=1008, reason="Invalid token payload")
        return

    # 连接
    await manager.connect(user_id, websocket)

    try:
        while True:
            # 接收消息
            data = await websocket.receive_json()

            # 处理消息
            message_type = data.get("type")
            message_data = data.get("data", {})

            if message_type == "client:message":
                # 处理游戏消息
                from ..services.message import process_message
                response = process_message(
                    message_data.get("message", ""),
                    # character_state from session
                )

                # 发送响应
                await manager.send_personal(user_id, {
                    "type": "server:message",
                    "data": {
                        "content": response.content,
                        "event_type": response.event_type,
                        "data": response.data
                    }
                })

    except WebSocketDisconnect:
        manager.disconnect(user_id)
```

**Step 3: 注册 WebSocket 路由**

```python
# backend/main.py (add)
from .api import websocket

app.include_router(websocket.router)
```

**Step 4: Commit**

```bash
git add backend/websocket/manager.py backend/api/websocket.py backend/main.py
git commit -m "feat: add WebSocket support"
```

---

## Phase 6: 前端项目初始化

### Task 19: Vite + React 项目

**Files:**
- Create: `frontend/vite.config.ts`
- Create: `frontend/tsconfig.json`
- Create: `frontend/package.json`
- Create: `frontend/index.html`
- Create: `frontend/src/main.tsx`

**Step 1: 初始化项目**

Run: `cd frontend && npm create vite@latest . -- --template react-ts`

**Step 2: 安装依赖**

```json
// frontend/package.json (add dependencies)
{
  "dependencies": {
    "react": "^18.3.1",
    "react-dom": "^18.3.1",
    "react-router-dom": "^6.22.0",
    "zustand": "^4.5.0",
    "socket.io-client": "^4.6.0",
    "@radix-ui/react-slot": "^1.0.2",
    "class-variance-authority": "^0.7.0",
    "clsx": "^2.1.0",
    "tailwind-merge": "^2.2.0",
    "lucide-react": "^0.344.0"
  },
  "devDependencies": {
    "@types/react": "^18.3.0",
    "@types/react-dom": "^18.3.0",
    "@vitejs/plugin-react": "^4.2.1",
    "autoprefixer": "^10.4.17",
    "postcss": "^8.4.35",
    "tailwindcss": "^3.4.1",
    "typescript": "^5.3.3",
    "vite": "^5.1.0"
  }
}
```

Run: `npm install`

**Step 3: 配置 TailwindCSS**

```css
/* frontend/src/index.css */
@tailwind base;
@tailwind components;
@tailwind utilities;

@layer base {
  :root {
    /* KP Theme (Purple) */
    --kp-primary: 5c6bc0;
    --kp-primary-hover: 3949ab;

    /* Player Theme (Teal) */
    --player-primary: 26a69a;
    --player-primary-hover: 00897b;

    /* Common */
    --background: 0 0% 100%;
    --foreground: 222.2 84% 4.9%;
  }
}
```

**Step 4: 配置 Vite**

```typescript
// frontend/vite.config.ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8000',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://localhost:8000',
        ws: true,
      },
    },
  },
})
```

**Step 5: Commit**

```bash
git add frontend/
git commit -m "feat: initialize React + Vite frontend"
```

---

### Task 20: shadcn/ui 组件库

**Files:**
- Create: `frontend/src/components/ui/button.tsx`
- Create: `frontend/src/components/ui/input.tsx`
- Create: `frontend/src/components/ui/card.tsx`
- Create: `frontend/src/lib/utils.ts`

**Step 1: 创建工具函数**

```typescript
// frontend/src/lib/utils.ts
import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
```

**Step 2: 创建 Button 组件**

```typescript
// frontend/src/components/ui/button.tsx
import * as React from "react"
import { Slot } from "@radix-ui/react-slot"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "@/lib/utils"

const buttonVariants = cva(
  "inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50",
  {
    variants: {
      variant: {
        default: "bg-[#5c6bc0] text-white hover:bg-[#3949ab]",
        player: "bg-[#26a69a] text-white hover:bg-[#00897b]",
        outline: "border border-gray-300 bg-transparent hover:bg-gray-100",
        ghost: "hover:bg-gray-100",
      },
      size: {
        default: "h-10 px-4 py-2",
        sm: "h-9 rounded-md px-3",
        lg: "h-11 rounded-md px-8",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button"
    return (
      <Comp
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        {...props}
      />
    )
  }
)
Button.displayName = "Button"

export { Button, buttonVariants }
```

**Step 3: 创建 Input 组件**

```typescript
// frontend/src/components/ui/input.tsx
import * as React from "react"
import { cn } from "@/lib/utils"

export interface InputProps
  extends React.InputHTMLAttributes<HTMLInputElement> {}

const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, type, ...props }, ref) => {
    return (
      <input
        type={type}
        className={cn(
          "flex h-10 w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm ring-offset-white file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-gray-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#5c6bc0] focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
          className
        )}
        ref={ref}
        {...props}
      />
    )
  }
)
Input.displayName = "Input"

export { Input }
```

**Step 4: 创建 Card 组件**

```typescript
// frontend/src/components/ui/card.tsx
import * as React from "react"
import { cn } from "@/lib/utils"

const Card = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={cn(
      "rounded-lg border border-gray-200 bg-white shadow-sm",
      className
    )}
    {...props}
  />
))
Card.displayName = "Card"

const CardHeader = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div
    ref={ref}
    className={cn("flex flex-col space-y-1.5 p-6", className)}
    {...props}
  />
))
CardHeader.displayName = "CardHeader"

const CardTitle = React.forwardRef<
  HTMLParagraphElement,
  React.HTMLAttributes<HTMLHeadingElement>
>(({ className, ...props }, ref) => (
  <h3
    ref={ref}
    className={cn("text-2xl font-semibold leading-none tracking-tight", className)}
    {...props}
  />
))
CardTitle.displayName = "CardTitle"

const CardContent = React.forwardRef<
  HTMLDivElement,
  React.HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
  <div ref={ref} className={cn("p-6 pt-0", className)} {...props} />
))
CardContent.displayName = "CardContent"

export { Card, CardHeader, CardTitle, CardContent }
```

**Step 5: Commit**

```bash
git add frontend/src/components/ui frontend/src/lib
git commit -m "feat: add shadcn/ui base components"
```

---

### Task 21: 认证状态管理

**Files:**
- Create: `frontend/src/stores/authStore.ts`

**Step 1: 创建认证 Store**

```typescript
// frontend/src/stores/authStore.ts
import { create } from 'zustand'

interface User {
  user_id: string
  username: string
  email: string
}

interface AuthState {
  user: User | null
  token: string | null
  isAuthenticated: boolean

  setAuth: (user: User, token: string) => void
  logout: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  token: localStorage.getItem('token'),
  isAuthenticated: !!localStorage.getItem('token'),

  setAuth: (user, token) => {
    localStorage.setItem('token', token)
    set({ user, token, isAuthenticated: true })
  },

  logout: () => {
    localStorage.removeItem('token')
    set({ user: null, token: null, isAuthenticated: false })
  },
}))
```

**Step 2: Commit**

```bash
git add frontend/src/stores/authStore.ts
git commit -m "feat: add auth state management"
```

---

### Task 22: API 客户端

**Files:**
- Create: `frontend/src/lib/api.ts`

**Step 1: 创建 API 客户端**

```typescript
// frontend/src/lib/api.ts
import { useAuthStore } from '@/stores/authStore'

const API_BASE = '/api'

async function request<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const token = useAuthStore.getState().token

  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token && { Authorization: `Bearer ${token}` }),
      ...options.headers,
    },
  })

  if (!response.ok) {
    throw new Error(`API Error: ${response.status}`)
  }

  return response.json()
}

export const api = {
  // Auth
  register: (data: { username: string; email: string; password: string }) =>
    request<{ user_id: string; token: string }>('/auth/register', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  login: (data: { username: string; password: string }) =>
    request<{ access_token: string; refresh_token: string }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  // Characters
  createCharacter: (data: any) =>
    request('/characters', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  getCharacters: () =>
    request('/characters'),

  getCharacter: (id: string) =>
    request(`/characters/${id}`),

  updateCharacter: (id: string, data: any) =>
    request(`/characters/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  deleteCharacter: (id: string) =>
    request(`/characters/${id}`, {
      method: 'DELETE',
    }),

  // Game
  sendMessage: (data: { message: string; session_id?: string }) =>
    request('/game/message', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  roll: (data: { skill_value: number; bonus_dice?: number; penalty_dice?: number }) =>
    request('/game/roll', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
}
```

**Step 2: Commit**

```bash
git add frontend/src/lib/api.ts
git commit -m "feat: add API client"
```

---

### Task 23: 登录页面

**Files:**
- Create: `frontend/src/pages/LoginPage.tsx`

**Step 1: 创建登录组件**

```typescript
// frontend/src/pages/LoginPage.tsx
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '@/stores/authStore'
import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'

export function LoginPage() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()
  const setAuth = useAuthStore((s) => s.setAuth)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      const result = await api.login({ username, password })
      // Get user info
      // For now just set the token
      setAuth(
        { user_id: 'temp', username, email: '' },
        result.access_token
      )
      navigate('/game')
    } catch (err) {
      setError('登录失败，请检查用户名和密码')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 px-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle className="text-2xl text-center">Monika - CoC 跑团平台</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm font-medium mb-1">用户名</label>
              <Input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">密码</label>
              <Input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>
            {error && (
              <div className="text-red-500 text-sm">{error}</div>
            )}
            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? '登录中...' : '登录'}
            </Button>
          </form>
          <div className="mt-4 text-center text-sm text-gray-600">
            还没有账号？{' '}
            <button
              type="button"
              onClick={() => navigate('/register')}
              className="text-[#5c6bc0] hover:underline"
            >
              注册
            </button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
```

**Step 2: 创建注册页面**

```typescript
// frontend/src/pages/RegisterPage.tsx
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'

export function RegisterPage() {
  const [formData, setFormData] = useState({
    username: '',
    email: '',
    password: '',
    confirmPassword: '',
  })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    if (formData.password !== formData.confirmPassword) {
      setError('密码不匹配')
      return
    }

    setLoading(true)

    try {
      await api.register({
        username: formData.username,
        email: formData.email,
        password: formData.password,
      })
      navigate('/login')
    } catch (err) {
      setError('注册失败，用户名或邮箱可能已被使用')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 px-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle className="text-2xl text-center">创建账号</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm font-medium mb-1">用户名</label>
              <Input
                type="text"
                value={formData.username}
                onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">邮箱</label>
              <Input
                type="email"
                value={formData.email}
                onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">密码</label>
              <Input
                type="password"
                value={formData.password}
                onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">确认密码</label>
              <Input
                type="password"
                value={formData.confirmPassword}
                onChange={(e) => setFormData({ ...formData, confirmPassword: e.target.value })}
                required
              />
            </div>
            {error && (
              <div className="text-red-500 text-sm">{error}</div>
            )}
            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? '注册中...' : '注册'}
            </Button>
          </form>
          <div className="mt-4 text-center text-sm text-gray-600">
            已有账号？{' '}
            <button
              type="button"
              onClick={() => navigate('/login')}
              className="text-[#5c6bc0] hover:underline"
            >
              登录
            </button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
```

**Step 3: 设置路由**

```typescript
// frontend/src/App.tsx
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { LoginPage } from './pages/LoginPage'
import { RegisterPage } from './pages/RegisterPage'
import { useAuthStore } from './stores/authStore'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" />
}

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route
          path="/game"
          element={
            <ProtectedRoute>
              <div>游戏台（待实现）</div>
            </ProtectedRoute>
          }
        />
        <Route path="/" element={<Navigate to="/game" />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
```

**Step 4: Commit**

```bash
git add frontend/src/pages/LoginPage.tsx frontend/src/pages/RegisterPage.tsx frontend/src/App.tsx
git commit -m "feat: add login and register pages"
```

---

### Task 24: 游戏台布局

**Files:**
- Create: `frontend/src/pages/GamePage.tsx`
- Create: `frontend/src/components/layout/Header.tsx`
- Create: `frontend/src/components/layout/GameStatePanel.tsx`

**Step 1: 创建 Header 组件**

```typescript
// frontend/src/components/layout/Header.tsx
import { useAuthStore } from '@/stores/authStore'
import { Button } from '@/components/ui/button'

export function Header() {
  const user = useAuthStore((s) => s.user)
  const logout = useAuthStore((s) => s.logout)

  return (
    <header className="h-14 border-b bg-white flex items-center justify-between px-4">
      <h1 className="text-xl font-bold text-[#5c6bc0]">Monika</h1>
      <div className="flex items-center gap-4">
        <span className="text-sm text-gray-600">{user?.username}</span>
        <Button variant="ghost" size="sm" onClick={logout}>
          退出
        </Button>
      </div>
    </header>
  )
}
```

**Step 2: 创建状态面板**

```typescript
// frontend/src/components/layout/GameStatePanel.tsx
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'

interface CharacterState {
  name: string
  derived_attributes: {
    HP: number
    HP_max: number
    SAN: number
    SAN_max: number
    Luck: number
    Luck_max: number
  }
}

export function GameStatePanel({ character }: { character?: CharacterState }) {
  if (!character) {
    return (
      <Card className="w-64">
        <CardHeader>
          <CardTitle className="text-lg">角色状态</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-gray-500 text-sm">未选择角色</p>
        </CardContent>
      </Card>
    )
  }

  const { derived_attributes } = character

  return (
    <Card className="w-64">
      <CardHeader>
        <CardTitle className="text-lg">{character.name}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* HP */}
        <div>
          <div className="flex justify-between text-sm mb-1">
            <span>HP</span>
            <span>{derived_attributes.HP}/{derived_attributes.HP_max}</span>
          </div>
          <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
            <div
              className="h-full bg-red-500 transition-all"
              style={{
                width: `${(derived_attributes.HP / derived_attributes.HP_max) * 100}%`
              }}
            />
          </div>
        </div>

        {/* SAN */}
        <div>
          <div className="flex justify-between text-sm mb-1">
            <span>SAN</span>
            <span>{derived_attributes.SAN}/{derived_attributes.SAN_max}</span>
          </div>
          <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
            <div
              className="h-full bg-purple-500 transition-all"
              style={{
                width: `${(derived_attributes.SAN / derived_attributes.SAN_max) * 100}%`
              }}
            />
          </div>
        </div>

        {/* Luck */}
        <div>
          <div className="flex justify-between text-sm mb-1">
            <span>幸运</span>
            <span>{derived_attributes.Luck}/{derived_attributes.Luck_max}</span>
          </div>
          <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
            <div
              className="h-full bg-yellow-500 transition-all"
              style={{
                width: `${(derived_attributes.Luck / derived_attributes.Luck_max) * 100}%`
              }}
            />
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
```

**Step 3: 创建游戏台页面**

```typescript
// frontend/src/pages/GamePage.tsx
import { useState } from 'react'
import { Header } from '@/components/layout/Header'
import { GameStatePanel } from '@/components/layout/GameStatePanel'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'

export function GamePage() {
  const [messages, setMessages] = useState<Array<{
    id: string
    role: 'user' | 'assistant'
    content: string
  }>>([
    {
      id: '1',
      role: 'assistant',
      content: '欢迎来到 Monika！输入 /help 查看可用命令。'
    }
  ])
  const [input, setInput] = useState('')

  const handleSend = async () => {
    if (!input.trim()) return

    const userMessage = {
      id: Date.now().toString(),
      role: 'user' as const,
      content: input
    }

    setMessages(prev => [...prev, userMessage])
    setInput('')

    try {
      const response = await fetch('/api/game/message', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${localStorage.getItem('token')}`
        },
        body: JSON.stringify({ message: input })
      })

      const data = await response.json()

      setMessages(prev => [...prev, {
        id: (Date.now() + 1).toString(),
        role: 'assistant',
        content: data.content
      }])
    } catch (err) {
      setMessages(prev => [...prev, {
        id: (Date.now() + 1).toString(),
        role: 'assistant',
        content: '抱歉，发生了错误。'
      }])
    }
  }

  return (
    <div className="h-screen flex flex-col">
      <Header />

      <div className="flex-1 flex overflow-hidden">
        {/* 消息区 */}
        <div className="flex-1 flex flex-col">
          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            {messages.map((msg) => (
              <div
                key={msg.id}
                className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
              >
                <div
                  className={`max-w-[70%] rounded-lg px-4 py-2 ${
                    msg.role === 'user'
                      ? 'bg-[#26a69a] text-white'
                      : 'bg-gray-100 text-gray-900'
                  }`}
                >
                  {msg.content}
                </div>
              </div>
            ))}
          </div>

          {/* 输入区 */}
          <div className="border-t p-4">
            <div className="flex gap-2">
              <Input
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && handleSend()}
                placeholder="输入命令或消息..."
              />
              <Button onClick={handleSend}>发送</Button>
            </div>
          </div>
        </div>

        {/* 状态面板 */}
        <div className="w-64 border-l p-4">
          <GameStatePanel />
        </div>
      </div>
    </div>
  )
}
```

**Step 4: 更新路由**

```typescript
// frontend/src/App.tsx (update game route)
import { GamePage } from './pages/GamePage'

// In Routes:
<Route
  path="/game"
  element={
    <ProtectedRoute>
      <GamePage />
    </ProtectedRoute>
  }
/>
```

**Step 5: Commit**

```bash
git add frontend/src/pages/GamePage.tsx frontend/src/components/layout
git commit -m "feat: add game console layout"
```

---

### Task 25: WebSocket 客户端

**Files:**
- Create: `frontend/src/lib/websocket.ts`

**Step 1: 创建 WebSocket Hook**

```typescript
// frontend/src/lib/websocket.ts
import { useEffect, useRef } from 'react'
import { io, Socket } from 'socket.io-client'

const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8000'

export function useWebSocket(token: string) {
  const socketRef = useRef<Socket | null>(null)

  useEffect(() => {
    // Connect with token
    socketRef.current = io(WS_URL, {
      auth: { token },
      transports: ['websocket'],
    })

    const socket = socketRef.current

    socket.on('connect', () => {
      console.log('WebSocket connected')
    })

    socket.on('disconnect', () => {
      console.log('WebSocket disconnected')
    })

    socket.on('error', (error) => {
      console.error('WebSocket error:', error)
    })

    return () => {
      socket.disconnect()
    }
  }, [token])

  const sendMessage = (type: string, data: any) => {
    socketRef.current?.emit(type, data)
  }

  const onMessage = (event: string, callback: (...args: any[]) => void) => {
    socketRef.current?.on(event, callback)
  }

  const offMessage = (event: string, callback: (...args: any[]) => void) => {
    socketRef.current?.off(event, callback)
  }

  return { sendMessage, onMessage, offMessage }
}
```

**Step 2: 更新游戏台使用 WebSocket**

```typescript
// frontend/src/pages/GamePage.tsx (update to use WebSocket)
import { useWebSocket } from '@/lib/websocket'
import { useAuthStore } from '@/stores/authStore'
import { useEffect, useRef } from 'react'

export function GamePage() {
  const token = useAuthStore((s) => s.token)
  const [messages, setMessages] = useState<Array<...>>(...)
  const { sendMessage, onMessage, offMessage } = useWebSocket(token || '')

  // ... existing handleSend code, but use WebSocket instead

  useEffect(() => {
    const handleMessage = (data: any) => {
      setMessages(prev => [...prev, {
        id: Date.now().toString(),
        role: 'assistant',
        content: data.data.content
      }])
    }

    onMessage('server:message', handleMessage)

    return () => {
      offMessage('server:message', handleMessage)
    }
  }, [onMessage, offMessage])

  // Update handleSend to use WebSocket
  const handleSend = () => {
    if (!input.trim()) return

    const userMessage = { id: Date.now().toString(), role: 'user' as const, content: input }
    setMessages(prev => [...prev, userMessage])

    sendMessage('client:message', { message: input })
    setInput('')
  }

  // ... rest of component
}
```

**Step 3: Commit**

```bash
git add frontend/src/lib/websocket.ts
git commit -m "feat: add WebSocket client"
```

---

### Task 26: 骰子结果显示组件

**Files:**
- Create: `frontend/src/components/game/DiceRollResult.tsx`

**Step 1: 创建骰子结果组件**

```typescript
// frontend/src/components/game/DiceRollResult.tsx
import { Card } from '@/components/ui/card'

interface RollData {
  skill: string
  skill_value: number
  roll: number
  success_level: string
  is_critical: boolean
  is_fumble: boolean
}

const SUCCESS_LABELS: Record<string, { text: string; emoji: string; color: string }> = {
  critical: { text: '大成功！', emoji: '🌟', color: 'text-yellow-500' },
  extremeSuccess: { text: '极难成功', emoji: '✅', color: 'text-green-500' },
  hardSuccess: { text: '困难成功', emoji: '👍', color: 'text-green-600' },
  regularSuccess: { text: '成功', emoji: '✅', color: 'text-green-700' },
  failure: { text: '失败', emoji: '❌', color: 'text-red-500' },
  fumble: { text: '大失败！', emoji: '💀', color: 'text-red-700' },
}

export function DiceRollResult({ data }: { data: RollData }) {
  const result = SUCCESS_LABELS[data.success_level] || SUCCESS_LABELS.failure

  return (
    <Card className="border-l-4 border-l-[#5c6bc0] p-4">
      <div className="flex items-center gap-4">
        {/* 骰子数字 */}
        <div className="text-4xl font-bold text-[#5c6bc0]">
          {data.roll}
        </div>

        {/* 结果信息 */}
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <span className="text-2xl">{result.emoji}</span>
            <span className={`text-lg font-semibold ${result.color}`}>
              {result.text}
            </span>
          </div>
          <div className="text-sm text-gray-600 mt-1">
            {data.skill} ({data.skill_value}%)
          </div>
        </div>

        {/* 特殊标记 */}
        {(data.is_critical || data.is_fumble) && (
          <div className="animate-pulse">
            {data.is_critical ? '✨' : '💥'}
          </div>
        )}
      </div>
    </Card>
  )
}
```

**Step 2: Commit**

```bash
git add frontend/src/components/game/DiceRollResult.tsx
git commit -m "feat: add dice roll result component"
```

---

## Phase 7: 测试和验收

### Task 27: E2E 测试

**Files:**
- Create: `frontend/tests/e2e/game.spec.ts`

**Step 1: 设置 Playwright**

Run: `npm install -D @playwright/test`

**Step 2: 创建 E2E 测试**

```typescript
// frontend/tests/e2e/game.spec.ts
import { test, expect } from '@playwright/test'

test.describe('Game Flow', () => {
  test.beforeEach(async ({ page }) => {
    // Login
    await page.goto('http://localhost:5173/login')
    await page.fill('input[type="text"]', 'testuser')
    await page.fill('input[type="password"]', 'password123')
    await page.click('button[type="submit"]')
    await page.waitForURL('**/game')
  })

  test('can send message', async ({ page }) => {
    await page.fill('input[placeholder*="输入"]', '你好')
    await page.click('button:has-text("发送")')

    // Check message appears
    await expect(page.locator('text=你好')).toBeVisible()
  })

  test('can roll dice', async ({ page }) => {
    await page.fill('input[placeholder*="输入"]', '/roll library_use')
    await page.click('button:has-text("发送")')

    // Check roll result appears
    await expect(page.locator('text=检定')).toBeVisible()
    await expect(page.locator('text=library_use')).toBeVisible()
  })

  test('can show help', async ({ page }) => {
    await page.fill('input[placeholder*="输入"]', '/help')
    await page.click('button:has-text("发送")')

    await expect(page.locator('text=可用命令')).toBeVisible()
  })
})
```

**Step 3: 运行测试**

Run: `npx playwright test`

**Step 4: Commit**

```bash
git add frontend/tests/e2e/
git commit -m "test: add E2E tests"
```

---

## 验收检查清单

完成以上所有任务后，运行以下检查：

### Backend
- [ ] PostgreSQL 数据库已创建并初始化
- [ ] 用户注册/登录 API 可用
- [ ] JWT Token 生成和验证正常
- [ ] 角色卡 CRUD API 可用
- [ ] 掷骰引擎正确计算成功等级
- [ ] 意图识别正确解析命令
- [ ] WebSocket 连接稳定
- [ ] LLM 集成返回响应

### Frontend
- [ ] 用户可以注册和登录
- [ ] 游戏台界面正常显示
- [ ] 消息发送和接收正常
- [ ] 骰子检定显示正确结果
- [ ] 状态面板显示角色信息
- [ ] WebSocket 实时通信正常
- [ ] 响应式布局在不同屏幕可用

### Integration
- [ ] 前后端 API 通信正常
- [ ] WebSocket 双向通信正常
- [ ] 认证中间件正确保护路由
- [ ] 错误处理和降级策略有效

---

## 风险和注意事项

1. **LLM API 稳定性**: 如果 OpenAI API 不稳定，实现降级到模板响应
2. **WebSocket 断线**: 实现自动重连机制
3. **状态同步**: 确保前后端状态一致，考虑实现状态快照
4. **输入验证**: 加强后端输入验证，防止注入攻击
5. **性能监控**: 添加日志和监控，追踪 API 响应时间

---

## 下一步

完成 M1 后，可以开始 M2（多人实时联机）的开发。
