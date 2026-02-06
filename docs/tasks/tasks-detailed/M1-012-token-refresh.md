# M1-012: 实现 Token 刷新 POST /auth/refresh

**任务ID**: M1-012
**标题**: 实现 Token 刷新 POST /auth/refresh
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M1-008

---

## 任务描述

实现 Token 刷新端点，允许用户使用过期但有效的刷新令牌获取新的访问令牌。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-012-01 | 设计刷新令牌机制 | Refresh Token | 20min |
| M1-012-02 | 扩展 Token 模型 | 添加 refresh_token | 20min |
| M1-012-03 | 实现 POST /auth/refresh | 刷新端点 | 30min |
| M1-012-04 | 实现刷新令牌验证 | 验证逻辑 | 25min |
| M1-012-05 | 实现令牌撤销 | 黑名单 | 25min |
| M1-012-06 | 编写刷新测试 | 测试刷新流程 | 25min |
| M1-012-07 | 编写刷新文档 | 使用说明 | 10min |

---

## 刷新令牌机制

```python
# app/core/token.py
from datetime import datetime, timedelta
from typing import Optional
import secrets

SECRET_KEY = "your-secret-key-here"
ALGORITHM = "HS256"
ACCESS_TOKEN_EXPIRE_MINUTES = 30
REFRESH_TOKEN_EXPIRE_DAYS = 7


def create_access_token(data: dict) -> str:
    """创建访问令牌"""
    to_encode = data.copy()
    expire = datetime.utcnow() + timedelta(minutes=ACCESS_TOKEN_EXPIRE_MINUTES)
    to_encode.update({"exp": expire, "type": "access"})
    return jwt.encode(to_encode, SECRET_KEY, algorithm=ALGORITHM)


def create_refresh_token(data: dict) -> str:
    """创建刷新令牌"""
    to_encode = data.copy()
    expire = datetime.utcnow() + timedelta(days=REFRESH_TOKEN_EXPIRE_DAYS)
    to_encode.update({
        "exp": expire,
        "type": "refresh",
        "jti": secrets.token_urlsafe(32)  # JWT ID
    })
    return jwt.encode(to_encode, SECRET_KEY, algorithm=ALGORITHM)


def verify_token(token: str, token_type: str = "access") -> dict:
    """验证令牌"""
    try:
        payload = jwt.decode(token, SECRET_KEY, algorithms=[ALGORITHM])

        # 检查令牌类型
        if payload.get("type") != token_type:
            raise ValueError("Invalid token type")

        return payload
    except jwt.ExpiredSignatureError:
        raise ValueError("Token expired")
    except jwt.InvalidTokenError:
        raise ValueError("Invalid token")


def decode_refresh_token(token: str) -> dict:
    """解码刷新令牌（即使过期）"""
    try:
        payload = jwt.decode(token, SECRET_KEY, algorithms=[ALGORITHM], options={"verify_exp": False})

        if payload.get("type") != "refresh":
            raise ValueError("Not a refresh token")

        return payload
    except jwt.InvalidTokenError:
        raise ValueError("Invalid refresh token")
```

---

## 数据库模型扩展

```python
# app/db/models/token.py
from sqlalchemy import Column, String, DateTime, Boolean
from sqlalchemy.sql import func
from app.db.database import Base

class RefreshToken(Base):
    __tablename__ = "refresh_tokens"

    id = Column(String, primary_key=True, index=True)
    jti = Column(String, unique=True, nullable=False, index=True)
    user_id = Column(String, nullable=False, index=True)
    token = Column(String, nullable=False)

    expires_at = Column(DateTime, nullable=False)
    created_at = Column(DateTime, default=func.now(), nullable=False)
    revoked = Column(Boolean, default=False, nullable=False)
    revoked_at = Column(DateTime, nullable=True)

    def revoke(self):
        """撤销令牌"""
        self.revoked = True
        self.revoked_at = func.now()
```

---

## 刷新端点实现

```python
# app/api/auth.py
from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.orm import Session
from pydantic import BaseModel

from app.core.token import (
    create_access_token,
    create_refresh_token,
    verify_token,
    decode_refresh_token
)
from app.db.models.user import User
from app.db.models.token import RefreshToken
from app.db.database import get_db

router = APIRouter(prefix="/auth", tags=["auth"])


class RefreshRequest(BaseModel):
    refresh_token: str


class RefreshResponse(BaseModel):
    access_token: str
    refresh_token: Optional[str] = None
    token_type: str = "bearer"


@router.post("/refresh", response_model=RefreshResponse)
async def refresh_token(
    request: RefreshRequest,
    db: Session = Depends(get_db)
):
    """刷新访问令牌"""
    credentials_exception = HTTPException(
        status_code=status.HTTP_401_UNAUTHORIZED,
        detail="Invalid refresh token",
        headers={"WWW-Authenticate": "Bearer"},
    )

    try:
        # 1. 解码刷新令牌（允许过期）
        payload = decode_refresh_token(request.refresh_token)

        # 2. 获取 JWT ID
        jti = payload.get("jti")
        user_id = payload.get("sub")

        if not jti or not user_id:
            raise credentials_exception

        # 3. 检查令牌是否存在且未被撤销
        token_record = db.query(RefreshToken).filter(
            RefreshToken.jti == jti
        ).first()

        if not token_record or token_record.revoked:
            raise credentials_exception

        # 4. 检查令牌是否过期
        if token_record.expires_at < datetime.utcnow():
            # 删除过期的刷新令牌
            db.delete(token_record)
            db.commit()
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Refresh token expired"
            )

        # 5. 获取用户信息
        user = db.query(User).filter(User.id == user_id).first()
        if not user or not user.is_active:
            raise credentials_exception

        # 6. 创建新的访问令牌
        access_token = create_access_token({"sub": user.id})

        # 7. 可选：轮换刷新令牌
        new_refresh_token = None
        if REFRESH_TOKEN_ROTATION:
            # 创建新的刷新令牌
            new_refresh_token = create_refresh_token({"sub": user.id})

            # 保存到数据库
            new_token_record = RefreshToken(
                id=secrets.token_urlsafe(32),
                jti=payload.get("jti"),  # 从新令牌获取
                user_id=user.id,
                token=new_refresh_token,
                expires_at=datetime.utcnow() + timedelta(days=REFRESH_TOKEN_EXPIRE_DAYS)
            )
            db.add(new_token_record)

            # 撤销旧的刷新令牌
            token_record.revoke()
            db.commit()

        return RefreshResponse(
            access_token=access_token,
            refresh_token=new_refresh_token
        )

    except ValueError as e:
        raise credentials_exception
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Could not refresh token"
        )


@router.post("/logout")
async def logout(
    refresh_token: str,
    db: Session = Depends(get_db)
):
    """撤销刷新令牌"""
    try:
        payload = decode_refresh_token(refresh_token)
        jti = payload.get("jti")

        if jti:
            # 撤销刷新令牌
            token_record = db.query(RefreshToken).filter(
                RefreshToken.jti == jti
            ).first()

            if token_record:
                token_record.revoke()
                db.commit()

        return {"message": "Successfully logged out"}

    except Exception:
        # 即使令牌无效也返回成功（幂等性）
        return {"message": "Successfully logged out"}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/core/token.py` | 更新 | Token 工具 |
| `app/db/models/token.py` | 创建 | Token 模型 |
| `app/api/auth.py` | 更新 | 认证 API |
| `alembic/versions/xxx_add_refresh_tokens.py` | 创建 | 迁移脚本 |

---

## 验收标准

- [ ] 刷新令牌正确生成
- [ ] 刷新端点正常工作
- [ ] 过期令牌被正确处理
- [ ] 撤销令牌不能使用
- [ ] 测试覆盖刷新流程

---

## 参考文档

- M1-008: JWT Token 中间件
- OAuth 2.0 刷新令牌规范

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
