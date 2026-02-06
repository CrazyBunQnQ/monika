# M1-009: 实现密码哈希 (bcrypt)

**任务ID**: M1-009
**标题**: 实现密码哈希 (bcrypt)
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M1-006

---

## 任务描述

实现使用 bcrypt 对用户密码进行安全哈希，确保密码存储的安全。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-009-01 | 选择密码哈希方案 | bcrypt vs 其他 | 15min |
| M1-009-02 | 安装 bcrypt 依赖 | python-passlib/bcrypt | 10min |
| M1-009-03 | 实现密码哈希函数 | hash_password() | 20min |
| M1-009-04 | 实现密码验证函数 | verify_password() | 20min |
| M1-009-05 | 集成到认证流程 | 注册/登录 | 20min |
| M1-009-06 | 编写单元测试 | 测试哈希和验证 | 25min |
| M1-009-07 | 编写密码策略文档 | 密码要求说明 | 15min |

---

## 密码哈希实现

```python
# app/core/security.py
from passlib.context import CryptContext
from typing import Optional

# 配置 bcrypt
pwd_context = CryptContext(
    schemes=["bcrypt"],
    deprecated="auto",
    bcrypt__rounds=12,  # 工作因子，数值越高越安全但越慢
    bcrypt__ident="2b"  # bcrypt 版本
)

# 密码策略
PASSWORD_MIN_LENGTH = 8
PASSWORD_MAX_LENGTH = 128
PASSWORD_REQUIRE_UPPERCASE = True
PASSWORD_REQUIRE_LOWERCASE = True
PASSWORD_REQUIRE_DIGIT = True
PASSWORD_REQUIRE_SPECIAL = False


def hash_password(password: str) -> str:
    """对密码进行哈希

    Args:
        password: 明文密码

    Returns:
        哈希后的密码

    Raises:
        ValueError: 密码不符合要求
    """
    if not password:
        raise ValueError("Password cannot be empty")

    if len(password) < PASSWORD_MIN_LENGTH:
        raise ValueError(
            f"Password must be at least {PASSWORD_MIN_LENGTH} characters"
        )

    if len(password) > PASSWORD_MAX_LENGTH:
        raise ValueError(
            f"Password must not exceed {PASSWORD_MAX_LENGTH} characters"
        )

    # 验证密码复杂度
    if PASSWORD_REQUIRE_UPPERCASE and not any(c.isupper() for c in password):
        raise ValueError("Password must contain at least one uppercase letter")

    if PASSWORD_REQUIRE_LOWERCASE and not any(c.islower() for c in password):
        raise ValueError("Password must contain at least one lowercase letter")

    if PASSWORD_REQUIRE_DIGIT and not any(c.isdigit() for c in password):
        raise ValueError("Password must contain at least one digit")

    # 哈希密码
    hashed = pwd_context.hash(password)

    return hashed


def verify_password(plain_password: str, hashed_password: str) -> bool:
    """验证密码

    Args:
        plain_password: 明文密码
        hashed_password: 哈希后的密码

    Returns:
        密码是否匹配
    """
    try:
        return pwd_context.verify(plain_password, hashed_password)
    except Exception:
        return False


def needs_rehash(hashed_password: str) -> bool:
    """检查密码是否需要重新哈希

    当工作因子更新时，可能需要重新哈希

    Args:
        hashed_password: 哈希后的密码

    Returns:
        是否需要重新哈希
    """
    return pwd_context.needs_update(hashed_password)
```

---

## 集成到认证流程

```python
# app/services/user.py
from app.core.security import hash_password, verify_password

class UserService:
    def __init__(self, db: Session):
        self.db = db

    def create_user(self, username: str, password: str, email: str) -> User:
        """创建新用户"""
        # 检查用户名是否已存在
        existing = self.db.query(User).filter(
            User.username == username
        ).first()
        if existing:
            raise ValueError("Username already exists")

        # 哈希密码
        hashed_password = hash_password(password)

        # 创建用户
        user = User(
            username=username,
            email=email,
            hashed_password=hashed_password,
        )

        self.db.add(user)
        self.db.commit()
        self.db.refresh(user)

        return user

    def authenticate(self, username: str, password: str) -> Optional[User]:
        """验证用户凭据"""
        user = self.db.query(User).filter(
            User.username == username
        ).first()

        if not user:
            return None

        # 验证密码
        if not verify_password(password, user.hashed_password):
            return None

        # 检查密码是否需要重新哈希
        from app.core.security import needs_rehash
        if needs_rehash(user.hashed_password):
            user.hashed_password = hash_password(password)
            self.db.commit()

        return user
```

---

## 单元测试

```python
# tests/test_security.py
import pytest
from app.core.security import (
    hash_password,
    verify_password,
    needs_rehash
)
from fastapi import HTTPException

class TestPasswordHashing:
    def test_hash_password(self):
        """测试密码哈希"""
        password = "TestPassword123"
        hashed = hash_password(password)

        assert hashed != password
        assert hashed.startswith("$2b$")  # bcrypt 哈希

    def test_verify_password_correct(self):
        """测试正确的密码验证"""
        password = "TestPassword123"
        hashed = hash_password(password)

        assert verify_password(password, hashed) is True

    def test_verify_password_incorrect(self):
        """测试错误的密码验证"""
        password = "TestPassword123"
        wrong_password = "WrongPassword123"
        hashed = hash_password(password)

        assert verify_password(wrong_password, hashed) is False

    def test_password_too_short(self):
        """测试密码过短"""
        with pytest.raises(ValueError):
            hash_password("short")

    def test_password_too_long(self):
        """测试密码过长"""
        with pytest.raises(ValueError):
            hash_password("a" * 129)

    def test_password_missing_uppercase(self):
        """测试缺少大写字母"""
        with pytest.raises(ValueError):
            hash_password("lowercase123")

    def test_password_missing_lowercase(self):
        """测试缺少小写字母"""
        with pytest.raises(ValueError):
            hash_password("UPPERCASE123")

    def test_password_missing_digit(self):
        """测试缺少数字"""
        with pytest.raises(ValueError):
            hash_password("NoDigits")

    def test_hash_is_consistent(self):
        """测试哈希一致性"""
        password = "TestPassword123"
        hashed1 = hash_password(password)
        hashed2 = hash_password(password)

        # 每次哈希结果应该不同 (salt 不同)
        assert hashed1 != hashed2

        # 但都应该能验证成功
        assert verify_password(password, hashed1) is True
        assert verify_password(password, hashed2) is True
```

---

## 依赖配置

```python
# requirements.txt
passlib[bcrypt]==1.7.4
bcrypt==4.0.1
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/core/security.py` | 创建 | 安全模块 |
| `app/services/user.py` | 更新 | 用户服务 |
| `tests/test_security.py` | 创建 | 安全测试 |

---

## 验收标准

- [ ] 密码使用 bcrypt 哈希
- [ ] 密码复杂度验证正确
- [ ] 密码验证准确
- [ ] 单元测试覆盖全面
- [ ] 文档说明清晰

---

## 参考文档

- OWASP 密码存储指南
- bcrypt 文档
- M1-006: 用户注册与认证 API

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
