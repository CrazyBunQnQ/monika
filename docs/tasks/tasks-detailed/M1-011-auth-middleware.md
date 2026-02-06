# M1-011: 实现密码验证中间件

**任务ID**: M1-011
**标题**: 实现密码验证中间件
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M1-009

---

## 任务描述

实现密码验证中间件，用于保护需要认证的 API 端点。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-011-01 | 设计中间件接口 | FastAPI Dependency | 15min |
| M1-011-02 | 实现 HTTP Basic Auth | 基础认证 | 25min |
| M1-011-03 | 实现 JWT Token 验证 | Token 认证 | 30min |
| M1-011-04 | 实现 Session 验证 | Session 认证 | 25min |
| M1-011-05 | 编写认证错误处理 | 错误响应 | 20min |
| M1-011-06 | 编写中间件测试 | 测试各种场景 | 25min |
| M1-011-07 | 编写认证文档 | 使用说明 | 15min |

---

## 认证中间件实现

```python
# app/api/deps/auth.py
from fastapi import Depends, HTTPException, status, Header
from fastapi.security import HTTPBasic, HTTPBearer
from sqlalchemy.orm import Session
from typing import Optional

from app.db.models.user import User
from app.core.token import verify_token
from app.db.database import get_db

# HTTP Basic 认证
security = HTTPBearer()

# 可选认证
optional_security = HTTPBearer(auto_error=False)


async def get_current_user(
    token: str = Depends(security),
    db: Session = Depends(get_db)
) -> User:
    """获取当前认证用户"""
    credentials_exception = HTTPException(
        status_code=status.HTTP_401_UNAUTHORIZED,
        detail="Could not validate credentials",
        headers={"WWW-Authenticate": "Bearer"},
    )

    try:
        payload = verify_token(token)
        user_id: str = payload.get("sub")
        if user_id is None:
            raise credentials_exception
    except Exception:
        raise credentials_exception

    user = db.query(User).filter(User.id == user_id).first()
    if not user:
        raise credentials_exception

    return user


async def get_current_active_user(
    current_user: User = Depends(get_current_user)
) -> User:
    """获取当前活跃用户"""
    if not current_user.is_active:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Inactive user"
        )
    return current_user


async def get_optional_user(
    token: Optional[str] = Depends(optional_security),
    db: Session = Depends(get_db)
) -> Optional[User]:
    """可选认证，返回用户或 None"""
    if not token:
        return None

    try:
        payload = verify_token(token)
        user_id: str = payload.get("sub")
        if user_id is None:
            return None
    except Exception:
        return None

    user = db.query(User).filter(User.id == user_id).first()
    return user


async def get_current_kp(
    current_user: User = Depends(get_current_active_user)
) -> User:
    """获取当前 KP 用户"""
    if current_user.role != "kp":
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="KP privileges required"
        )
    return current_user


async def get_current_player(
    current_user: User = Depends(get_current_active_user)
) -> User:
    """获取当前玩家用户"""
    if current_user.role != "player":
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Player privileges required"
        )
    return current_user
```

---

## Token 验证

```python
# app/core/token.py
from datetime import datetime, timedelta
from typing import Optional
from jose import JWTError, jwt
from fastapi import HTTPException, status

SECRET_KEY = "your-secret-key-here"
ALGORITHM = "HS256"
ACCESS_TOKEN_EXPIRE_MINUTES = 30 * 24 * 60  # 30 天


def create_access_token(data: dict, expires_delta: Optional[timedelta] = None):
    """创建访问令牌"""
    to_encode = data.copy()

    if expires_delta:
        expire = datetime.utcnow() + expires_delta
    else:
        expire = datetime.utcnow() + timedelta(minutes=ACCESS_TOKEN_EXPIRE_MINUTES)

    to_encode.update({"exp": expire})
    encoded_jwt = jwt.encode(to_encode, SECRET_KEY, algorithm=ALGORITHM)

    return encoded_jwt


def verify_token(token: str) -> dict:
    """验证令牌"""
    credentials_exception = HTTPException(
        status_code=status.HTTP_401_UNAUTHORIZED,
        detail="Could not validate credentials",
        headers={"WWW-Authenticate": "Bearer"},
    )

    try:
        payload = jwt.decode(token, SECRET_KEY, algorithms=[ALGORITHM])
        return payload
    except JWTError:
        raise credentials_exception
```

---

## 使用示例

```python
# app/api/game.py
from fastapi import APIRouter, Depends
from app.api.deps.auth import get_current_user, get_current_kp
from app.db.models.user import User

router = APIRouter(prefix="/game", tags=["game"])

@router.post("/action")
async def perform_action(
    action_data: ActionRequest,
    current_user: User = Depends(get_current_user)
):
    """需要认证的端点"""
    # 处理行动
    return {"user": current_user.username, "action": action_data}

@router.post("/kp-only")
async def kp_only_endpoint(
    data: KPRequest,
    current_user: User = Depends(get_current_kp)
):
    """仅 KP 可访问"""
    # KP 专属功能
    return {"message": "KP action completed"}

@router.get("/public")
async def public_endpoint(
    maybe_user: Optional[User] = Depends(get_optional_user)
):
    """公开或认证都可访问"""
    if maybe_user:
        return {"message": f"Hello {maybe_user.username}"}
    else:
        return {"message": "Hello guest"}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/api/deps/auth.py` | 创建 | 认证依赖 |
| `app/core/token.py` | 创建 | Token 工具 |
| `tests/test_auth.py` | 创建 | 认证测试 |

---

## 验收标准

- [ ] 中间件正确验证 Token
- [ ] 错误处理完善
- [ ] KP/Player 区分正确
- [ ] 可选认证支持
- [ ] 单元测试覆盖

---

## 参考文档

- M1-009: 密码哈希
- FastAPI 安全文档

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
