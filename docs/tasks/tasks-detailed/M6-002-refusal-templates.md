# M6-002: 实现拒绝处理模板

**任务ID**: M6-002
**标题**: 实现拒绝处理模板
**类型**: backend (后端开发)
**预估工时**: 2h
**依赖**: M0

---

## 任务描述

实现友好的错误和拒绝处理模板，为各种错误情况提供清晰的用户反馈。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-002-01 | 设计错误分类体系 | Error Types | 20min |
| M6-002-02 | 实现错误模板系统 | Templates | 25min |
| M6-002-03 | 实现错误响应格式 | Response Format | 25min |
| M6-002-04 | 实现本地化支持 | i18n | 30min |
| M6-002-05 | 编写常用错误模板 | Common Errors | 25min |
| M6-002-06 | 编写错误文档 | Error Docs | 15min |

---

## 错误分类体系

```python
# app/core/errors.py
from enum import Enum
from typing import Optional, Dict, Any

class ErrorType(str, Enum):
    """错误类型"""
    # 认证错误
    UNAUTHORIZED = "unauthorized"
    FORBIDDEN = "forbidden"
    TOKEN_EXPIRED = "token_expired"
    INVALID_CREDENTIALS = "invalid_credentials"

    # 验证错误
    VALIDATION_ERROR = "validation_error"
    INVALID_INPUT = "invalid_input"
    MISSING_REQUIRED = "missing_required"

    # 资源错误
    NOT_FOUND = "not_found"
    ALREADY_EXISTS = "already_exists"
    CONFLICT = "conflict"

    # 业务错误
    INSUFFICIENT_PERMISSION = "insufficient_permission"
    INVALID_STATE = "invalid_state"
    OPERATION_FAILED = "operation_failed"

    # 系统错误
    INTERNAL_ERROR = "internal_error"
    SERVICE_UNAVAILABLE = "service_unavailable"
    RATE_LIMITED = "rate_limited"

class ErrorResponse:
    """标准错误响应"""

    def __init__(
        self,
        error_type: ErrorType,
        message: str,
        details: Optional[Dict[str, Any]] = None,
        code: Optional[str] = None,
    ):
        self.error_type = error_type
        self.message = message
        self.details = details or {}
        self.code = code or error_type.value

    def to_dict(self) -> Dict[str, Any]:
        """转换为字典"""
        return {
            "error": {
                "type": self.error_type.value,
                "code": self.code,
                "message": self.message,
                "details": self.details,
            }
        }

# 预定义错误
class Errors:
    """预定义错误模板"""

    @staticmethod
    def unauthorized() -> ErrorResponse:
        return ErrorResponse(
            error_type=ErrorType.UNAUTHORIZED,
            message="需要登录才能访问此资源",
            details={"hint": "请先登录"},
        )

    @staticmethod
    def forbidden(resource: str) -> ErrorResponse:
        return ErrorResponse(
            error_type=ErrorType.FORBIDDEN,
            message=f"没有权限访问 {resource}",
            details={"resource": resource},
        )

    @staticmethod
    def not_found(resource: str, identifier: str) -> ErrorResponse:
        return ErrorResponse(
            error_type=ErrorType.NOT_FOUND,
            message=f"{resource} 不存在",
            details={"resource": resource, "identifier": identifier},
        )

    @staticmethod
    def validation_error(field: str, reason: str) -> ErrorResponse:
        return ErrorResponse(
            error_type=ErrorType.VALIDATION_ERROR,
            message=f"输入验证失败",
            details={"field": field, "reason": reason},
        )

    @staticmethod
    def room_full() -> ErrorResponse:
        return ErrorResponse(
            error_type=ErrorType.INVALID_STATE,
            message="房间已满，无法加入",
            details={"hint": "请尝试其他房间或联系房主"},
        )

    @staticmethod
    def character_not_found(character_id: str) -> ErrorResponse:
        return Errors.not_found("角色卡", character_id)
```

---

## 错误处理中间件

```python
# app/api/errors.py
from fastapi import Request, status
from fastapi.responses import JSONResponse
from fastapi.exceptions import RequestValidationError
from sqlalchemy.exc import IntegrityError

from app.core.errors import ErrorResponse, Errors

async def error_handler(request: Request, call_next):
    """全局错误处理中间件"""
    try:
        response = await call_next(request)
        return response
    except ErrorResponse as e:
        return JSONResponse(
            status_code=_get_status_code(e.error_type),
            content=e.to_dict(),
        )
    except RequestValidationError as e:
        return JSONResponse(
            status_code=status.HTTP_422_UNPROCESSABLE_ENTITY,
            content={
                "error": {
                    "type": "validation_error",
                    "message": "输入验证失败",
                    "details": e.errors(),
                }
            },
        )
    except IntegrityError as e:
        return JSONResponse(
            status_code=status.HTTP_409_CONFLICT,
            content={
                "error": {
                    "type": "conflict",
                    "message": "数据冲突",
                    "details": {"detail": str(e)},
                }
            },
        )
    except Exception as e:
        return JSONResponse(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            content={
                "error": {
                    "type": "internal_error",
                    "message": "服务器内部错误",
                }
            },
        )

def _get_status_code(error_type: ErrorType) -> int:
    """获取错误类型对应的 HTTP 状态码"""
    status_map = {
        ErrorType.UNAUTHORIZED: 401,
        ErrorType.FORBIDDEN: 403,
        ErrorType.TOKEN_EXPIRED: 401,
        ErrorType.INVALID_CREDENTIALS: 401,
        ErrorType.VALIDATION_ERROR: 422,
        ErrorType.INVALID_INPUT: 400,
        ErrorType.MISSING_REQUIRED: 400,
        ErrorType.NOT_FOUND: 404,
        ErrorType.ALREADY_EXISTS: 409,
        ErrorType.CONFLICT: 409,
        ErrorType.INSUFFICIENT_PERMISSION: 403,
        ErrorType.INVALID_STATE: 400,
        ErrorType.OPERATION_FAILED: 400,
        ErrorType.INTERNAL_ERROR: 500,
        ErrorType.SERVICE_UNAVAILABLE: 503,
        ErrorType.RATE_LIMITED: 429,
    }
    return status_map.get(error_type, 500)
```

---

## FastAPI 集成

```python
# app/main.py
from fastapi import FastAPI
from app.api.errors import error_handler

app = FastAPI()

# 添加错误处理中间件
app.middleware("http")(error_handler)
```

---

## 本地化支持

```python
# app/core/i18n.py
from typing import Dict

class ErrorMessages:
    """错误消息本地化"""

    MESSAGES: Dict[str, Dict[str, str]] = {
        "zh": {
            "unauthorized": "需要登录才能访问此资源",
            "forbidden": "没有权限访问此资源",
            "not_found": "资源不存在",
            "validation_error": "输入验证失败",
            # ... 更多消息
        },
        "en": {
            "unauthorized": "Authentication required",
            "forbidden": "Permission denied",
            "not_found": "Resource not found",
            "validation_error": "Input validation failed",
            # ... more messages
        },
    }

    @classmethod
    def get(cls, key: str, lang: str = "zh") -> str:
        """获取本地化消息"""
        return cls.MESSAGES.get(lang, cls.MESSAGES["zh"]).get(
            key,
            key
        )
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/core/errors.py` | 创建 | 错误定义 |
| `app/api/errors.py` | 创建 | 错误处理中间件 |
| `app/core/i18n.py` | 创建 | 本地化支持 |
| `docs/api/errors.md` | 创建 | 错误文档 |

---

## 验收标准

- [ ] 错误分类清晰
- [ ] 错误消息友好
- [ ] 响应格式统一
- [ ] 本地化支持完整
- [ ] 文档清晰

---

## 参考文档

- RFC 7807: Problem Details for HTTP APIs
- FastAPI 错误处理文档

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
