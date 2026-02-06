# M1-006: 用户注册与认证 API

**任务类型**: backend
**预估工时**: 4h
**依赖**: M0
**状态**: [ ]

---

## 子任务拆解

### 1.1 数据库模型 (40min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-006-01 | [ ] 创建 `app/db/user.py` | [ ] |
| M1-006-02 | [ ] 定义 `User` SQLModel 模型 | [ ] |
| M1-006-03 | [ ] 定义 `UserCreate` 数据类 | [ ] |
| M1-006-04 | [ ] 定义 `UserResponse` 数据类 | [ ] |

```python
# app/db/user.py
from datetime import datetime
from typing import Optional
from sqlmodel import SQLModel, Field, Relationship
from enum import Enum

class UserRole(str, Enum):
    """用户角色"""
    PLAYER = "player"       # 玩家
    KEEPER = "keeper"       # 守密人
    ADMIN = "admin"        # 管理员

class User(SQLModel, table=True):
    """用户模型"""
    id: Optional[int] = Field(default=None, primary_key=True)
    username: str = Field(unique=True, index=True, max_length=50)
    email: str = Field(unique=True, index=True, max_length=255)
    hashed_password: str = Field(max_length=255)
    role: UserRole = Field(default=UserRole.PLAYER)
    is_active: bool = Field(default=True)
    created_at: datetime = Field(default_factory=datetime.utcnow)
    updated_at: datetime = Field(default_factory=datetime.utcnow)

    # 关系
    characters: list["Character"] = Relationship(back_populates="owner")
    campaigns: list["Campaign"] = Relationship(back_populates="keeper")

class UserCreate(SQLModel):
    """用户注册请求"""
    username: str = Field(..., min_length=3, max_length=50)
    email: str = Field(..., max_length=255)
    password: str = Field(..., min_length=8, max_length=128)
    confirm_password: str

    def validate_passwords_match(self):
        if self.password != self.confirm_password:
            raise ValueError("两次输入的密码不一致")

class UserLogin(SQLModel):
    """用户登录请求"""
    username: str
    password: str

class UserResponse(SQLModel):
    """用户响应"""
    id: int
    username: str
    email: str
    role: UserRole
    is_active: bool
    created_at: datetime

class UserUpdate(SQLModel):
    """用户更新请求"""
    email: Optional[str] = None
    password: Optional[str] = None
    role: Optional[UserRole] = None
    is_active: Optional[bool] = None
```

---

### 1.2 密码处理 (20min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-006-05 | [ ] 创建 `app/core/security.py` | [ ] |
| M1-006-06 | [ ] 实现 `hash_password()` 函数 | [ ] |
| M1-006-07 | [ ] 实现 `verify_password()` 函数 | [ ] |
| M1-006-08 | [ ] 实现 `generate_token()` 函数 | [ ] |

```python
# app/core/security.py
from datetime import datetime, timedelta
from typing import Optional
from passlib.context import CryptContext
from jose import JWTError, jwt
from pydantic import BaseModel

# 密码加密配置
pwd_context = CryptContext(schemes=["bcrypt"], deprecated="auto")

# JWT 配置
SECRET_KEY = "your-secret-key-change-in-production"
ALGORITHM = "HS256"
ACCESS_TOKEN_EXPIRE_MINUTES = 60 * 24  # 24小时

class Token(BaseModel):
    """JWT Token 响应"""
    access_token: str
    token_type: str = "bearer"
    expires_in: int

class TokenData(BaseModel):
    """Token 数据"""
    user_id: Optional[int] = None
    username: Optional[str] = None

def hash_password(password: str) -> str:
    """哈希密码"""
    return pwd_context.hash(password)

def verify_password(plain_password: str, hashed_password: str) -> bool:
    """验证密码"""
    return pwd_context.verify(plain_password, hashed_password)

def create_access_token(data: dict, expires_delta: Optional[timedelta] = None) -> str:
    """创建 JWT Token"""
    to_encode = data.copy()
    if expires_delta:
        expire = datetime.utcnow() + expires_delta
    else:
        expire = datetime.utcnow() + timedelta(minutes=ACCESS_TOKEN_EXPIRE_MINUTES)
    to_encode.update({"exp": expire})
    encoded_jwt = jwt.encode(to_encode, SECRET_KEY, algorithm=ALGORITHM)
    return encoded_jwt

def decode_token(token: str) -> Optional[TokenData]:
    """解码 JWT Token"""
    try:
        payload = jwt.decode(token, SECRET_KEY, algorithms=[ALGORITHM])
        user_id: int = payload.get("sub")
        username: str = payload.get("username")
        if user_id is None:
            return None
        return TokenData(user_id=user_id, username=username)
    except JWTError:
        return None
```

---

### 1.3 用户服务层 (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-006-09 | [ ] 创建 `app/services/user.py` | [ ] |
| M1-006-10 | [ ] 实现 `create_user()` 函数 | [ ] |
| M1-006-11 | [ ] 实现 `authenticate_user()` 函数 | [ ] |
| M1-006-12 | [ ] 实现 `get_user_by_id()` 函数 | [ ] |

```python
# app/services/user.py
from typing import Optional
from app.db.user import User, UserCreate
from app.core.security import hash_password, verify_password
from sqlmodel import Session, select

class UserService:
    """用户服务"""

    def __init__(self, session: Session):
        self.session = session

    def create_user(self, user_data: UserCreate) -> User:
        """创建新用户"""
        # 检查用户名是否存在
        existing = self.session.exec(
            select(User).where(User.username == user_data.username)
        ).first()
        if existing:
            raise ValueError("用户名已存在")

        # 检查邮箱是否存在
        existing = self.session.exec(
            select(User).where(User.email == user_data.email)
        ).first()
        if existing:
            raise ValueError("邮箱已被注册")

        # 创建用户
        user = User(
            username=user_data.username,
            email=user_data.email,
            hashed_password=hash_password(user_data.password)
        )
        self.session.add(user)
        self.session.commit()
        self.session.refresh(user)
        return user

    def authenticate_user(self, username: str, password: str) -> Optional[User]:
        """验证用户登录"""
        user = self.session.exec(
            select(User).where(User.username == username)
        ).first()

        if not user:
            return None
        if not verify_password(password, user.hashed_password):
            return None
        return user

    def get_user_by_id(self, user_id: int) -> Optional[User]:
        """根据 ID 获取用户"""
        return self.session.get(User, user_id)

    def get_user_by_username(self, username: str) -> Optional[User]:
        """根据用户名获取用户"""
        return self.session.exec(
            select(User).where(User.username == username)
        ).first()
```

---

### 1.4 API 路由 (40min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-006-13 | [ ] 创建 `app/api/auth.py` | [ ] |
| M1-006-14 | [ ] 实现 POST /auth/register | [ ] |
| M1-006-15 | [ ] 实现 POST /auth/login | [ ] |
| M1-006-16 | [ ] 实现 POST /auth/logout | [ ] |
| M1-006-17 | [ ] 实现 GET /auth/me | [ ] |
| M1-006-18 | [ ] 添加依赖注入 | [ ] |

```python
# app/api/auth.py
from fastapi import APIRouter, Depends, HTTPException, status
from typing import Annotated
from app.db.user import UserCreate, UserLogin, UserResponse, User
from app.services.user import UserService
from app.core.security import create_access_token, decode_token, Token
from app.db import get_session

router = APIRouter(prefix="/auth", tags=["认证"])

async def get_current_user(
    token: str,
    session: Annotated[Session, Depends(get_session)]
) -> User:
    """获取当前登录用户"""
    token_data = decode_token(token)
    if token_data is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="无效的认证凭据",
            headers={"WWW-Authenticate": "Bearer"},
        )

    user = session.get(User, token_data.user_id)
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="用户不存在",
            headers={"WWW-Authenticate": "Bearer"},
        )
    return user

@router.post("/register", response_model=UserResponse, status_code=status.HTTP_201_CREATED)
async def register(
    user_data: UserCreate,
    session: Annotated[Session, Depends(get_session)]
):
    """用户注册"""
    user_data.validate_passwords_match()
    service = UserService(session)
    try:
        user = service.create_user(user_data)
        return UserResponse(
            id=user.id,
            username=user.username,
            email=user.email,
            role=user.role,
            is_active=user.is_active,
            created_at=user.created_at
        )
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))

@router.post("/login", response_model=Token)
async def login(
    login_data: UserLogin,
    session: Annotated[Session, Depends(get_session)]
):
    """用户登录"""
    service = UserService(session)
    user = service.authenticate_user(login_data.username, login_data.password)

    if not user:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="用户名或密码错误",
            headers={"WWW-Authenticate": "Bearer"},
        )

    access_token = create_access_token(data={"sub": str(user.id), "username": user.username})

    return Token(
        access_token=access_token,
        token_type="bearer",
        expires_in=60 * 24 * 7  # 7天
    )

@router.get("/me", response_model=UserResponse)
async def get_me(current_user: Annotated[User, Depends(get_current_user)]):
    """获取当前用户信息"""
    return UserResponse(
        id=current_user.id,
        username=current_user.username,
        email=current_user.email,
        role=current_user.role,
        is_active=current_user.is_active,
        created_at=current_user.created_at
    )

@router.post("/logout")
async def logout():
    """用户登出（客户端清除 Token 即可）"""
    return {"message": "已成功登出"}
```

---

### 1.5 数据库迁移 (20min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-006-19 | [ ] 创建初始迁移脚本 | [ ] |
| M1-006-20 | [ ] 创建数据库初始化脚本 | [ ] |

```python
# scripts/init_db.py
import sys
sys.path.insert(0, ".")

from sqlmodel import create_engine, SQLModel
from app.db.user import User
from app.db import Base

DATABASE_URL = "postgresql://user:password@localhost/monika"

def init_db():
    engine = create_engine(DATABASE_URL)
    SQLModel.metadata.create_all(engine)
    print("数据库初始化完成")

if __name__ == "__main__":
    init_db()
```

---

### 1.6 单元测试 (40min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-006-21 | [ ] 创建 `tests/test_auth.py` | [ ] |
| M1-006-22 | [ ] 测试用户注册 | [ ] |
| M1-006-23 | [ ] 测试用户登录 | [ ] |
| M1-006-24 | [ ] 测试 Token 生成/验证 | [ ] |
| M1-006-25 | [ ] 测试密码哈希 | [ ] |

```python
# tests/test_auth.py
import pytest
from fastapi.testclient import TestClient
from sqlmodel import Session, create_engine, SQLModel
from app.db.user import User
from app.core.security import hash_password, verify_password, create_access_token
from app.db import get_session

# 测试数据库
TEST_DATABASE_URL = "sqlite:///./test.db"
engine = create_engine(TEST_DATABASE_URL)

@pytest.fixture(scope="function")
def session():
    SQLModel.metadata.create_all(engine)
    with Session(engine) as s:
        yield s
    SQLModel.metadata.drop_all(engine)

class TestPasswordHashing:
    def test_hash_password(self):
        """测试密码哈希"""
        hashed = hash_password("test_password")
        assert hashed != "test_password"
        assert len(hashed) > 50

    def test_verify_password_correct(self):
        """测试正确密码验证"""
        hashed = hash_password("test_password")
        assert verify_password("test_password", hashed) is True

    def test_verify_password_incorrect(self):
        """测试错误密码验证"""
        hashed = hash_password("test_password")
        assert verify_password("wrong_password", hashed) is False

class TestTokenGeneration:
    def test_create_access_token(self):
        """测试 Token 创建"""
        token = create_access_token(data={"sub": "123", "username": "test"})
        assert isinstance(token, str)
        assert len(token) > 50

    def test_token_contains_claims(self):
        """测试 Token 包含声明"""
        from app.core.security import decode_token
        token = create_access_token(data={"sub": "123", "username": "test"})
        decoded = decode_token(token)
        assert decoded is not None
        assert decoded.user_id == "123"
        assert decoded.username == "test"
```

---

## 验收标准

- [ ] 用户可以注册（用户名、邮箱、密码）
- [ ] 用户可以登录并获取 JWT Token
- [ ] Token 有效期为 7 天
- [ ] 密码安全哈希存储
- [ ] 用户名/邮箱唯一性约束
- [ ] API 文档完整
- [ ] 单元测试覆盖率 > 90%

---

## 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `app/db/user.py` | 创建 | 用户数据库模型 |
| `app/core/security.py` | 创建 | 安全工具（密码、Token） |
| `app/services/user.py` | 创建 | 用户服务层 |
| `app/api/auth.py` | 创建 | 认证 API 路由 |
| `tests/test_auth.py` | 创建 | 单元测试 |
| `scripts/init_db.py` | 创建 | 数据库初始化 |

---

## API 文档

### POST /auth/register

**请求体:**
```json
{
  "username": "player1",
  "email": "player1@example.com",
  "password": "securepassword123",
  "confirm_password": "securepassword123"
}
```

**响应 (201):**
```json
{
  "id": 1,
  "username": "player1",
  "email": "player1@example.com",
  "role": "player",
  "is_active": true,
  "created_at": "2026-02-05T10:00:00Z"
}
```

### POST /auth/login

**请求体:**
```json
{
  "username": "player1",
  "password": "securepassword123"
}
```

**响应 (200):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "bearer",
  "expires_in": 604800
}
```
