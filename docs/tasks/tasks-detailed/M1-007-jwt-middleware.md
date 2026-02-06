# M1-007: JWT Token 中间件

**任务类型**: backend
**预估工时**: 2h
**依赖**: M1-006
**状态**: [ ]

---

## 子任务拆解

### 1.1 Token 服务 (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-007-01 | [ ] 创建 `app/core/token.py` | [ ] |
| M1-007-02 | [ ] 实现 `create_access_token()` | [ ] |
| M1-007-03 | [ ] 实现 `create_refresh_token()` | [ ] |
| M1-007-04 | [ ] 实现 `decode_token()` | [ ] |

```python
# app/core/token.py
from datetime import datetime, timedelta
from typing import Optional
from jose import JWTError, jwt
from pydantic import BaseModel
from app.core.config import settings

class TokenType(str, str):
    """Token 类型"""
    ACCESS = "access"
    REFRESH = "refresh"

class TokenPayload(BaseModel):
    """Token 载荷"""
    sub: str              # 用户 ID
    username: str         # 用户名
    token_type: TokenType = TokenType.ACCESS
    exp: datetime         # 过期时间

class TokenService:
    """Token 服务"""

    ACCESS_EXPIRE = timedelta(minutes=30)      # 访问令牌 30 分钟
    REFRESH_EXPIRE = timedelta(days=7)         # 刷新令牌 7 天

    def create_access_token(
        self,
        user_id: str,
        username: str,
        expires_delta: Optional[timedelta] = None
    ) -> str:
        """创建访问令牌"""
        to_encode = {
            "sub": str(user_id),
            "username": username,
            "token_type": TokenType.ACCESS,
            "exp": datetime.utcnow() + (expires_delta or self.ACCESS_EXPIRE),
        }
        return jwt.encode(
            to_encode,
            settings.SECRET_KEY,
            algorithm=settings.ALGORITHM
        )

    def create_refresh_token(
        self,
        user_id: str,
        expires_delta: Optional[timedelta] = None
    ) -> str:
        """创建刷新令牌"""
        to_encode = {
            "sub": str(user_id),
            "token_type": TokenType.REFRESH,
            "exp": datetime.utcnow() + (expires_delta or self.REFRESH_EXPIRE),
        }
        return jwt.encode(
            to_encode,
            settings.SECRET_KEY,
            algorithm=settings.ALGORITHM
        )

    def decode_token(self, token: str) -> Optional[TokenPayload]:
        """解码 Token"""
        try:
            payload = jwt.decode(
                token,
                settings.SECRET_KEY,
                algorithms=[settings.ALGORITHM]
            )
            return TokenPayload(**payload)
        except JWTError:
            return None

    def verify_token_type(
        self,
        payload: TokenPayload,
        expected_type: TokenType
    ) -> bool:
        """验证 Token 类型"""
        return payload.token_type == expected_type
```

---

### 1.2 认证依赖 (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-007-05 | [ ] 创建 `app/api/deps/auth.py` | [ ] |
| M1-007-06 | [ ] 实现 `get_current_user()` | [ ] |
| M1-007-07 | [ ] 实现 `get_current_active_user()` | [ ] |
| M1-007-08 | [ ] 实现 `get_optional_user()` | [ ] |

```python
# app/api/deps/auth.py
from typing import Optional, Annotated
from fastapi import Depends, HTTPException, status
from sqlmodel import Session

from app.db.user import User
from app.db.connection import get_session
from app.core.token import TokenService, TokenPayload, TokenType

token_service = TokenService()

async def get_current_user(
    authorization: str,
    session: Annotated[Session, Depends(get_session)]
) -> User:
    """获取当前登录用户"""
    if not authorization:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="未提供认证凭据",
            headers={"WWW-Authenticate": "Bearer"},
        )

    token = authorization.replace("Bearer ", "").strip()
    payload = token_service.decode_token(token)

    if payload is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="无效的 Token",
            headers={"WWW-Authenticate": "Bearer"},
        )

    if not token_service.verify_token_type(payload, TokenType.ACCESS):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Token 类型不正确",
            headers={"WWW-Authenticate": "Bearer"},
        )

    user = session.get(User, int(payload.sub))
    if user is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="用户不存在",
            headers={"WWW-Authenticate": "Bearer"},
        )

    return user

async def get_current_active_user(
    current_user: Annotated[User, Depends(get_current_user)]
) -> User:
    """获取当前活跃用户"""
    if not current_user.is_active:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="用户已被禁用"
        )
    return current_user

async def get_optional_user(
    authorization: Optional[str] = None,
    session: Annotated[Session, Depends(get_session)] = None
) -> Optional[User]:
    """获取可选用户（未登录返回 None）"""
    if not authorization:
        return None

    token = authorization.replace("Bearer ", "").strip()
    payload = token_service.decode_token(token)

    if payload is None:
        return None

    return session.get(User, int(payload.sub))
```

---

### 1.3 角色权限依赖 (25min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-007-09 | [ ] 创建 `app/api/deps/permissions.py` | [ ] |
| M1-007-10 | [ ] 实现 `require_role()` | [ ] |
| M1-007-11 | [ ] 实现 `require_keeper()` | [ ] |
| M1-007-12 | [ ] 实现 `require_admin()` | [ ] |

```python
# app/api/deps/permissions.py
from typing import Callable, Any
from fastapi import HTTPException, status
from app.db.user import User, UserRole

def require_role(*allowed_roles: UserRole) -> Callable:
    """角色权限依赖工厂"""
    def role_checker(get_user: Callable) -> Callable:
        async def wrapper(*args, **kwargs) -> Any:
            user = await get_user(*args, **kwargs)

            if user.role not in allowed_roles:
                raise HTTPException(
                    status_code=status.HTTP_403_FORBIDDEN,
                    detail=f"需要以下角色之一: {[r.value for r in allowed_roles]}"
                )
            return user
        return wrapper
    return role_checker

def require_keeper(get_user: Callable) -> Callable:
    """守密人权限"""
    async def wrapper(*args, **kwargs) -> User:
        user = await get_user(*args, **kwargs)

        if user.role not in [UserRole.KEEPER, UserRole.ADMIN]:
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="只有守密人或管理员可以执行此操作"
            )
        return user
    return wrapper

def require_admin(get_user: Callable) -> Callable:
    """管理员权限"""
    async def wrapper(*args, **kwargs) -> User:
        user = await get_user(*args, **kwargs)

        if user.role != UserRole.ADMIN:
            raise HTTPException(
                status_code=status.HTTP_403_FORBIDDEN,
                detail="只有管理员可以执行此操作"
            )
        return user
    return wrapper
```

---

### 1.4 刷新令牌 API (20min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-007-13 | [ ] 在 auth.py 中添加刷新端点 | [ ] |
| M1-007-14 | [ ] 实现 POST /auth/refresh | [ ] |
| M1-007-15 | [ ] 实现 POST /auth/revoke | [ ] |

```python
# app/api/auth.py 新增

class RefreshTokenRequest(BaseModel):
    """刷新令牌请求"""
    refresh_token: str

class TokenResponse(BaseModel):
    """Token 响应"""
    access_token: str
    refresh_token: str
    token_type: str = "bearer"

@router.post("/refresh", response_model=TokenResponse)
async def refresh_token(
    request: RefreshTokenRequest,
    session: Annotated[Session, Depends(get_session)]
):
    """使用刷新令牌获取新的访问令牌"""
    payload = token_service.decode_token(request.refresh_token)

    if payload is None or not token_service.verify_token_type(
        payload, TokenType.REFRESH
    ):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="无效的刷新令牌"
        )

    user = session.get(User, int(payload.sub))
    if user is None or not user.is_active:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="用户不存在或已被禁用"
        )

    # 创建新令牌
    access_token = token_service.create_access_token(
        user_id=str(user.id),
        username=user.username
    )
    refresh_token = token_service.create_refresh_token(
        user_id=str(user.id)
    )

    return TokenResponse(
        access_token=access_token,
        refresh_token=refresh_token
    )

@router.post("/revoke")
async def revoke_token(
    current_user: Annotated[User, Depends(get_current_active_user)]
):
    """撤销当前用户的刷新令牌（退出所有设备）"""
    # 在实际应用中，这里应该将令牌加入黑名单
    # 可以使用 Redis 存储被撤销的令牌 ID
    return {"message": "已成功退出所有设备"}
```

---

### 1.5 单元测试 (25min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-007-16 | [ ] 创建 `tests/test_token.py` | [ ] |
| M1-007-17 | [ ] 测试 Token 创建/验证 | [ ] |
| M1-007-18 | [ ] 测试 Token 过期 | [ ] |
| M1-007-19 | [ ] 测试角色权限依赖 | [ ] |

```python
# tests/test_token.py
import pytest
from datetime import timedelta
from app.core.token import TokenService, TokenType

class TestTokenService:
    def setup_method(self):
        self.token_service = TokenService()

    def test_create_access_token(self):
        """测试创建访问令牌"""
        token = self.token_service.create_access_token(
            user_id="123",
            username="testuser"
        )
        assert isinstance(token, str)
        assert len(token) > 50

    def test_decode_valid_token(self):
        """测试解码有效令牌"""
        token = self.token_service.create_access_token(
            user_id="123",
            username="testuser"
        )
        payload = self.token_service.decode_token(token)

        assert payload is not None
        assert payload.sub == "123"
        assert payload.username == "testuser"
        assert payload.token_type == TokenType.ACCESS

    def test_verify_token_type(self):
        """测试验证 Token 类型"""
        token = self.token_service.create_access_token(user_id="123", username="test")
        payload = self.token_service.decode_token(token)

        assert self.token_service.verify_token_type(payload, TokenType.ACCESS) is True
        assert self.token_service.verify_token_type(payload, TokenType.REFRESH) is False

    def test_expired_token(self):
        """测试过期令牌"""
        token = self.token_service.create_access_token(
            user_id="123",
            username="test",
            expires_delta=timedelta(seconds=-1)  # 已过期
        )
        payload = self.token_service.decode_token(token)
        assert payload is None
```

---

## 验收标准

- [ ] 访问令牌 30 分钟有效
- [ ] 刷新令牌 7 天有效
- [ ] Token 可正确解码验证
- [ ] 角色权限依赖正常工作
- [ ] 单元测试覆盖率 > 90%

---

## 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `app/core/token.py` | 创建 | Token 服务 |
| `app/api/deps/auth.py` | 创建 | 认证依赖 |
| `app/api/deps/permissions.py` | 创建 | 权限依赖 |
| `app/api/auth.py` | 修改 | 添加刷新端点 |
| `tests/test_token.py` | 创建 | 单元测试 |
