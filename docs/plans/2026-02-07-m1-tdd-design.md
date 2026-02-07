# M1 TDD 重构设计文档

## 项目概述

- **项目名称**: Monika Backend
- **技术栈**: FastAPI + SQLAlchemy + PostgreSQL + uv
- **开发方式**: TDD（测试驱动开发）
- **分支**: `refactor/m1-tdd-backend`

## 项目结构

```
backend/
├── pyproject.toml          # 项目配置 + 依赖
├── .python-version         # Python 3.11
├── .env.example           # 环境变量模板
├── requirements.txt       # 依赖列表
├── migrations/           # Alembic 迁移脚本
├── src/
│   ├── __init__.py
│   ├── main.py           # FastAPI 入口
│   ├── api/              # API 路由
│   │   ├── __init__.py
│   │   ├── auth.py       # 认证路由 (POST /auth/register, POST /auth/login)
│   │   ├── characters.py  # 角色路由 (CRUD)
│   │   └── game.py       # 游戏路由 (POST /game/roll, POST /game/characters/{id}/roll/{skill})
│   ├── core/             # 核心配置
│   │   ├── __init__.py
│   │   ├── config.py     # 环境变量 (pydantic-settings)
│   │   ├── security.py    # JWT、密码加密
│   │   ├── database.py    # SQLAlchemy 连接
│   │   └── auth.py       # 认证依赖 (get_current_user)
│   ├── models/           # SQLAlchemy 模型
│   │   ├── __init__.py
│   │   ├── user.py       # User 模型
│   │   └── character.py  # Character 模型
│   ├── schemas/          # Pydantic 模式
│   │   ├── __init__.py
│   │   ├── user.py       # User 模式
│   │   └── character.py  # Character 模式
│   ├── services/        # 业务逻辑
│   │   ├── __init__.py
│   │   └── dice.py      # 掷骰引擎
│   └── tests/           # 测试
│       ├── __init__.py
│       ├── conftest.py  # pytest fixtures
│       ├── test_database.py
│       ├── test_auth.py
│       ├── test_api_auth.py
│       ├── test_characters.py
│       ├── test_dice.py
│       └── test_integration.py
```

## TDD 开发顺序

| 阶段 | 状态 | 测试数 | 说明 |
|------|------|--------|------|
| 0. 项目初始化 | ✅ | - | uv、依赖、配置 |
| 1. 数据库层 | ✅ | 7 | User/Character 模型、测试fixtures |
| 2. 用户认证 | ✅ | 21 | 注册、登录、JWT |
| 3. 角色 CRUD | ✅ | 13 | 完整角色操作、所有权验证 |
| 4. 掷骰引擎 | ✅ | 19 | d100、成功判定、奖惩骰 |
| 5. API 集成测试 | ✅ | 11 | 端到端测试、工作流测试 |
| **总计** | | **71** | |

## TDD 循环

```
1. 写测试 (Red)     → 运行测试 → 失败
2. 写实现 (Green)  → 运行测试 → 通过
3. 重构 (Refactor) → 运行测试 → 确保通过
```

## 关键决策

### 1. src-layout vs 平铺布局
- **选择**: src-layout（推荐）
- **原因**: 清晰分离源代码和配置文件，避免导入问题

### 2. pyproject.toml vs setup.py
- **选择**: pyproject.toml
- **原因**: Python 官方推荐，uv 原生支持

### 3. 测试数据库
- **选择**: SQLite in-memory
- **原因**: 速度快，无需外部依赖，适合 CI/CD

### 4. 依赖管理
- **选择**: uv
- **原因**: 比 pip 快，支持锁定文件，Python 版本管理

### 5. Pydantic V2 兼容
- **选择**: 使用 `model_config = ConfigDict(from_attributes=True)`
- **原因**: V2 新语法，废弃了 `Config` class

### 6. Python 3.11 `int` 字段名
- **选择**: Schema 使用 `intelligence`，数据库使用 `int`
- **原因**: Python 不允许 `int` 作为字段名（与内置类型冲突）

## API 端点

### 认证 (Auth)
- `POST /auth/register` - 用户注册
- `POST /auth/login` - 用户登录 (返回 JWT)

### 角色 (Characters)
- `POST /characters` - 创建角色
- `GET /characters` - 列出角色
- `GET /characters/{id}` - 获取角色
- `PUT /characters/{id}` - 更新角色
- `DELETE /characters/{id}` - 删除角色

### 游戏 (Game)
- `POST /game/roll` - 掷 d100
- `POST /game/characters/{id}/roll/{skill}` - 掷角色技能
- `GET /game/health` - 健康检查

## 运行命令

```bash
# 安装依赖
uv sync

# 运行开发服务器
uv run python -m uvicorn src.main:app --reload

# 运行测试
uv run pytest

# 运行特定测试
uv run pytest src/tests/test_integration.py -v
```

## 当前状态

- [x] 项目初始化完成
- [x] 数据库层
- [x] 用户认证
- [x] 角色 CRUD
- [x] 掷骰引擎
- [x] API 集成测试

## 相关文档

- PRD: `docs/prd/PRD.md`
- M1 任务清单: `docs/tasks/02-m1-single-player-web.md`
