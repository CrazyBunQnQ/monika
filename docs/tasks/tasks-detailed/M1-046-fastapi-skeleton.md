# M1-046: FastAPI 项目骨架

**任务类型**: backend
**预估工时**: 4h
**依赖**: M0
**状态**: [ ]

---

## 子任务拆解

### 1.1 项目目录结构 (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-046-01 | [ ] 创建项目根目录结构 | [ ] |
| M1-046-02 | [ ] 创建 `app/` 核心目录 | [ ] |
| M1-046-03 | [ ] 创建 `tests/` 测试目录 | [ ] |
| M1-046-04 | [ ] 创建 `scripts/` 脚本目录 | [ ] |

```
monika/
├── app/
│   ├── __init__.py
│   ├── main.py                 # FastAPI 应用入口
│   ├── api/                    # API 路由
│   │   ├── __init__.py
│   │   └── routes.py
│   ├── core/                   # 核心业务逻辑
│   │   ├── __init__.py
│   │   ├── config.py           # 配置
│   │   └── security.py         # 安全工具
│   ├── db/                     # 数据库层
│   │   ├── __init__.py
│   │   ├── connection.py       # 数据库连接
│   │   └── models.py            # SQLModel 模型
│   ├── services/               # 业务服务层
│   │   └── __init__.py
│   └── schemas/                # Pydantic schemas
│       └── __init__.py
├── tests/
│   ├── __init__.py
│   └── conftest.py
├── scripts/
│   └── init_db.py
├── pyproject.toml
├── .env.example
└── README.md
```

---

### 1.2 配置文件 (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-046-05 | [ ] 创建 `app/core/config.py` | [ ] |
| M1-046-06 | [ ] 实现环境变量加载 | [ ] |
| M1-046-07 | [ ] 实现开发/生产环境区分 | [ ] |

```python
# app/core/config.py
from functools import lru_cache
from typing import Optional
from pydantic import BaseModel
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """应用配置"""
    # 应用名称
    APP_NAME: str = "Monika - CoC TRPG Platform"
    APP_VERSION: str = "0.1.0"

    # 数据库配置
    DATABASE_URL: str = "postgresql://user:password@localhost:5432/monika"
    DATABASE_ECHO: bool = False

    # Redis 配置（可选）
    REDIS_URL: Optional[str] = None

    # 安全配置
    SECRET_KEY: str = "change-this-secret-key-in-production"
    ALGORITHM: str = "HS256"
    ACCESS_TOKEN_EXPIRE_MINUTES: int = 60 * 24 * 7  # 7天

    # LLM 配置
    LLM_PROVIDER: str = "openai"  # openai / anthropic
    LLM_MODEL: str = "gpt-4"
    LLM_API_KEY: Optional[str] = None

    # 日志配置
    LOG_LEVEL: str = "INFO"

    class Config:
        env_file = ".env"
        case_sensitive = True


@lru_cache()
def get_settings() -> Settings:
    """获取缓存的配置"""
    return Settings()


# 全局配置实例
settings = get_settings()
```

---

### 1.3 数据库连接 (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-046-08 | [ ] 创建 `app/db/connection.py` | [ ] |
| M1-046-09 | [ ] 实现 SQLModel 连接 | [ ] |
| M1-046-10 | [ ] 实现依赖注入 | [ ] |
| M1-046-11 | [ ] 实现 Session 管理 | [ ] |

```python
# app/db/connection.py
from typing import Generator, Annotated
from sqlmodel import Session, create_engine, SQLModel
from app.core.config import settings

# 创建数据库引擎
def get_database_url() -> str:
    """获取数据库 URL"""
    return settings.DATABASE_URL

# 创建引擎
engine = create_engine(
    get_database_url(),
    echo=settings.DATABASE_ECHO,
    pool_pre_ping=True,  # 连接池预ping
    pool_size=10,        # 连接池大小
    max_overflow=20      # 最大溢出
)

def init_db():
    """初始化数据库"""
    SQLModel.metadata.create_all(engine)

def get_session() -> Generator[Session, None, None]:
    """获取数据库会话（依赖注入）"""
    with Session(engine) as session:
        try:
            yield session
        finally:
            session.close()

# 简化的依赖注入类型
SessionDep = Annotated[Session, Depends(get_session)]
```

---

### 1.4 FastAPI 应用入口 (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-046-12 | [ ] 创建 `app/main.py` | [ ] |
| M1-046-13 | [ ] 实现应用实例配置 | [ ] |
| M1-046-14 | [ ] 添加中间件 | [ ] |
| M1-046-15 | [ ] 注册路由 | [ ] |
| M1-046-16 | [ ] 添加 CORS 配置 | [ ] |

```python
# app/main.py
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
from contextlib import asynccontextmanager
import logging

from app.core.config import settings
from app.db.connection import init_db
from app.api import routes

# 日志配置
logging.basicConfig(level=getattr(logging, settings.LOG_LEVEL))
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """应用生命周期管理"""
    # 启动时
    logger.info("正在启动 Monika CoC TRPG Platform...")
    init_db()
    logger.info("数据库初始化完成")
    yield
    # 关闭时
    logger.info("正在关闭 Monika CoC TRPG Platform...")


# 创建 FastAPI 应用
app = FastAPI(
    title=settings.APP_NAME,
    version=settings.APP_VERSION,
    description="一个基于 CoC 7e 规则的 TRPG AI 助手平台",
    lifespan=lifespan,
    docs_url="/docs",
    redoc_url="/redoc",
)

# CORS 中间件
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # 生产环境应限制为特定域名
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# 注册路由
app.include_router(routes.router, prefix="/api/v1")


@app.get("/health")
async def health_check():
    """健康检查"""
    return {
        "status": "healthy",
        "version": settings.APP_VERSION,
        "name": settings.APP_NAME
    }


@app.get("/")
async def root():
    """根路径"""
    return {
        "name": settings.APP_NAME,
        "version": settings.APP_VERSION,
        "docs": "/docs"
    }
```

---

### 1.5 路由注册 (20min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-046-17 | [ ] 创建 `app/api/__init__.py` | [ ] |
| M1-046-18 | [ ] 创建 `app/api/routes.py` | [ ] |
| M1-046-19 | [ ] 实现路由模块注册 | [ ] |

```python
# app/api/routes.py
from fastapi import APIRouter
from app.api import auth, health

# 主路由
router = APIRouter()

# 注册子路由
router.include_router(auth.router, prefix="/auth", tags=["认证"])
router.include_router(health.router, prefix="/health", tags=["健康检查"])

# 可以在此添加更多路由
# router.include_router(users.router, prefix="/users", tags=["用户"])
# router.include_router(players.router, prefix="/players", tags=["角色卡"])
```

```python
# app/api/__init__.py
from app.api.routes import router

__all__ = ["router"]
```

---

### 1.6 依赖文件 (30min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-046-20 | [ ] 创建 `pyproject.toml` | [ ] |
| M1-046-21 | [ ] 创建 `.env.example` | [ ] |
| M1-046-22 | [ ] 创建 `.gitignore` | [ ] |

```toml
# pyproject.toml
[project]
name = "monika"
version = "0.1.0"
description = "一个基于 CoC 7e 规则的 TRPG AI 助手平台"
readme = "README.md"
requires-python = ">=3.10"
license = {text = "MIT"}
authors = [
    {name = "Monika Team"}
]
dependencies = [
    "fastapi>=0.109.0",
    "uvicorn[standard]>=0.27.0",
    "sqlmodel>=0.0.19",
    "pydantic>=2.5.0",
    "pydantic-settings>=2.1.0",
    "python-multipart>=0.0.6",
    "passlib[bcrypt]>=1.7.4",
    "python-jose[cryptography]>=3.3.0",
    "aiofiles>=23.2.1",
    # 开发依赖
    "pytest>=7.4.0",
    "pytest-asyncio>=0.23.0",
    "httpx>=0.26.0",
]

[project.optional-dependencies]
dev = [
    "pytest>=7.4.0",
    "pytest-asyncio>=0.23.0",
    "httpx>=0.26.0",
    "black>=23.0.0",
    "ruff>=0.1.0",
    "mypy>=1.8.0",
]

[build-system]
requires = ["setuptools>=69.0.0"]
build-backend = "setuptools.build_meta"

[tool.pytest.ini_options]
asyncio_mode = "auto"
testpaths = ["tests"]

[tool.black]
line-length = 100
target-version = ["py310"]

[tool.ruff]
line-length = 100
target-version = "py310"
```

```bash
# .env.example
# 应用配置
APP_NAME=Monika - CoC TRPG Platform
APP_VERSION=0.1.0

# 数据库
DATABASE_URL=postgresql://user:password@localhost:5432/monika
DATABASE_ECHO=false

# Redis (可选)
REDIS_URL=

# 安全
SECRET_KEY=change-this-secret-key-in-production

# LLM 配置
LLM_PROVIDER=openai
LLM_MODEL=gpt-4
LLM_API_KEY=your-api-key-here

# 日志
LOG_LEVEL=INFO
```

---

### 1.7 测试配置 (20min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-046-23 | [ ] 创建 `tests/conftest.py` | [ ] |
| M1-046-24 | [ ] 实现测试 fixtures | [ ] |
| M1-046-25 | [ ] 创建 `tests/__init__.py` | [ ] |

```python
# tests/conftest.py
import pytest
from fastapi.testclient import TestClient
from sqlmodel import Session, SQLModel, create_engine
from app.main import app
from app.db.connection import get_session

# 测试数据库
TEST_DATABASE_URL = "sqlite:///./test.db"
engine = create_engine(TEST_DATABASE_URL)


@pytest.fixture(scope="session")
def db_engine():
    """测试数据库引擎"""
    SQLModel.metadata.create_all(engine)
    yield engine
    SQLModel.metadata.drop_all(engine)


@pytest.fixture
def session(db_engine):
    """测试数据库会话"""
    with Session(db_engine) as s:
        yield s


@pytest.fixture
def client(session):
    """测试客户端"""
    def override_get_session():
        yield session

    app.dependency_overrides[get_session] = override_get_session

    with TestClient(app) as c:
        yield c

    app.dependency_overrides.clear()


@pytest.fixture
def auth_headers(client):
    """认证请求头（如果有测试用户）"""
    # 可以在这里创建测试用户并返回 Token
    return {}
```

---

### 1.8 运行脚本 (20min)

| ID | 任务 | 状态 |
|----|------|------|
| M1-046-26 | [ ] 创建 `run.py` 启动脚本 | [ ] |
| M1-046-27 | [ ] 创建 `scripts/init_db.py` | [ ] |
| M1-046-28 | [ ] 创建 `scripts/seed_data.py` | [ ] |

```python
# run.py
#!/usr/bin/env python
"""启动脚本"""
import uvicorn
from app.core.config import settings

if __name__ == "__main__":
    uvicorn.run(
        "app.main:app",
        host="0.0.0.0",
        port=8000,
        reload=settings.LOG_LEVEL == "DEBUG",
        log_level=settings.LOG_LEVEL.lower()
    )
```

---

## 验收标准

- [ ] 项目结构清晰，符合最佳实践
- [ ] 支持环境变量配置
- [ ] 数据库连接正常
- [ ] API 文档可访问 (/docs)
- [ ] 健康检查端点正常
- [ ] 单元测试可通过
- [ ] CORS 配置正确
- [ ] 热重载支持（开发模式）

---

## 涉及文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `app/main.py` | 创建 | FastAPI 应用入口 |
| `app/core/config.py` | 创建 | 配置管理 |
| `app/db/connection.py` | 创建 | 数据库连接 |
| `app/api/routes.py` | 创建 | 路由注册 |
| `pyproject.toml` | 创建 | 项目依赖 |
| `tests/conftest.py` | 创建 | 测试配置 |
| `run.py` | 创建 | 启动脚本 |
| `.env.example` | 创建 | 环境变量示例 |

---

## 启动命令

```bash
# 开发模式
python run.py

# 或使用 uvicorn
uvicorn app.main:app --reload --log-level debug

# 运行测试
pytest tests/ -v

# 类型检查
mypy app/
```

---

## 项目结构预览

```
monika/
├── app/
│   ├── __init__.py
│   ├── main.py              # FastAPI 入口
│   ├── api/
│   │   ├── __init__.py
│   │   └── routes.py        # 路由
│   ├── core/
│   │   ├── __init__.py
│   │   └── config.py        # 配置
│   └── db/
│       ├── __init__.py
│       └── connection.py    # 数据库
├── tests/
│   ├── __init__.py
│   └── conftest.py          # 测试配置
├── pyproject.toml           # 项目配置
├── .env.example             # 环境变量示例
└── run.py                   # 启动脚本
```
