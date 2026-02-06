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
│   │   ├── auth.py       # 认证路由
│   │   ├── characters.py # 角色路由
│   │   └── game.py       # 游戏路由
│   ├── core/             # 核心配置
│   │   ├── __init__.py
│   │   ├── config.py     # 环境变量 (pydantic-settings)
│   │   ├── security.py    # JWT、密码加密
│   │   └── database.py    # SQLAlchemy 连接
│   ├── models/           # SQLAlchemy 模型
│   │   ├── __init__.py
│   │   ├── user.py       # User 模型
│   │   ├── character.py  # Character 模型
│   │   └── session.py    # Session 模型
│   ├── schemas/          # Pydantic 模式
│   │   ├── __init__.py
│   │   ├── user.py       # User 模式
│   │   ├── character.py  # Character 模式
│   │   └── game.py       # 游戏模式
│   ├── services/        # 业务逻辑
│   │   ├── __init__.py
│   │   ├── auth.py      # 认证服务
│   │   ├── character.py # 角色服务
│   │   └── dice.py      # 掷骰引擎
│   └── tests/           # 测试
│       ├── __init__.py
│       ├── conftest.py  # pytest fixtures
│       ├── test_auth.py
│       ├── test_characters.py
│       └── test_dice.py
```

## TDD 开发顺序

| 阶段 | 任务 | 测试文件 | 说明 |
|------|------|----------|------|
| 0 | 项目初始化 | - | uv、依赖、配置 |
| 1 | 数据库层 | `test_database.py` | 连接、session |
| 2 | 用户认证 | `test_auth.py` | 注册、登录、JWT |
| 3 | 角色 CRUD | `test_characters.py` | 完整角色操作 |
| 4 | 掷骰引擎 | `test_dice.py` | d100、成功判定、奖惩骰 |
| 5 | API 集成 | `test_api.py` | 端到端测试 |

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

## 当前状态

- [x] 项目初始化完成
- [ ] 数据库层
- [ ] 用户认证
- [ ] 角色 CRUD
- [ ] 掷骰引擎
- [ ] API 集成测试

## 相关文档

- PRD: `docs/prd/PRD.md`
- M1 任务清单: `docs/tasks/02-m1-single-player-web.md`
