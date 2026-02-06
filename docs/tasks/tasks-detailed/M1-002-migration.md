# M1-002: 数据库迁移脚本

**任务ID**: M1-002
**标题**: 数据库迁移脚本
**类型**: db (数据库)
**预估工时**: 3h
**依赖**: M1-001

---

## 任务描述

使用 Alembic 创建数据库迁移脚本，实现数据库版本管理，支持升级和回滚。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M1-002-01 | 配置 Alembic | 初始化 Alembic 配置 | 30min |
| M1-002-02 | 创建初始迁移 | 生成基础表结构 | 30min |
| M1-002-03 | 创建 Users 表迁移 | 用户表 | 20min |
| M1-002-04 | 创建 Characters 表迁移 | 角色卡表 | 20min |
| M1-002-05 | 创建 Campaigns 表迁移 | 战役表 | 20min |
| M1-002-06 | 创建 Sessions 表迁移 | 会话表 | 15min |
| M1-002-07 | 创建 Events 表迁移 | 事件日志表 | 15min |
| M1-002-08 | 编写升级/降级脚本 | upgrade/downgrade 函数 | 20min |
| M1-002-09 | 测试迁移和回滚 | 验证迁移正确性 | 15min |
| M1-002-10 | 编写迁移文档 | 使用说明 | 10min |

---

## Alembic 配置

```ini
# alembic.ini
[alembic]
script_location = alembic
file_template = %%(year)d%%(month).2d%%(day).2d_%%(hour).2d%%(minute).2d_%%(rev)s_%%(slug)s
truncate_slug_length = 40
sqlalchemy.url = postgresql://user:pass@localhost/monika

[loggers]
keys = root,sqlalchemy,alembic

[handlers]
keys = console

[formatters]
keys = generic
```

---

## 迁移脚本示例

```python
# alembic/versions/20260206_initial.py
"""Initial migration

Revision ID: 001
Revises:
Create Date: 2026-02-06 10:00:00.000000

"""
from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects import postgresql

revision = '001'
down_revision = None
branch_labels = None
depends_on = None


def upgrade() -> None:
    # 创建 users 表
    op.create_table(
        'users',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('username', sa.String(length=50), nullable=False),
        sa.Column('email', sa.String(length=255), nullable=False),
        sa.Column('hashed_password', sa.String(length=255), nullable=False),
        sa.Column('created_at', sa.DateTime(), server_default=sa.text('now()'), nullable=False),
        sa.Column('updated_at', sa.DateTime(), server_default=sa.text('now()'), nullable=False),
        sa.PrimaryKeyConstraint('id'),
        sa.UniqueConstraint('username'),
        sa.UniqueConstraint('email'),
    )
    op.create_index('ix_users_id', 'users', ['id'])

    # 创建 characters 表
    op.create_table(
        'characters',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('user_id', sa.Integer(), nullable=False),
        sa.Column('name', sa.String(length=100), nullable=False),
        sa.Column('data', postgresql.JSONB(), nullable=False),
        sa.Column('created_at', sa.DateTime(), server_default=sa.text('now()'), nullable=False),
        sa.Column('updated_at', sa.DateTime(), server_default=sa.text('now()'), nullable=False),
        sa.PrimaryKeyConstraint('id'),
        sa.ForeignKeyConstraint(['user_id'], ['users.id']),
    )

    # 创建 campaigns 表
    op.create_table(
        'campaigns',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('name', sa.String(length=200), nullable=False),
        sa.Column('kp_id', sa.Integer(), nullable=False),
        sa.Column('status', sa.String(length=20), nullable=False),
        sa.Column('created_at', sa.DateTime(), server_default=sa.text('now()'), nullable=False),
        sa.PrimaryKeyConstraint('id'),
        sa.ForeignKeyConstraint(['kp_id'], ['users.id']),
    )

    # ... 其他表


def downgrade() -> None:
    op.drop_table('campaigns')
    op.drop_table('characters')
    op.drop_table('users')
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `alembic.ini` | 创建 | Alembic 配置 |
| `alembic/env.py` | 创建 | 环境配置 |
| `alembic/script.py.mako` | 创建 | 迁移模板 |
| `alembic/versions/001_initial.py` | 创建 | 初始迁移 |
| `docs/database/migrations.md` | 创建 | 迁移文档 |

---

## 常用命令

```bash
# 创建新迁移
alembic revision --autogenerate -m "description"

# 升级到最新版本
alembic upgrade head

# 升级到特定版本
alembic upgrade +1

# 降级一个版本
alembic downgrade -1

# 查看当前版本
alembic current

# 查看历史
alembic history
```

---

## 验收标准

- [ ] Alembic 配置正确
- [ ] 初始迁移包含所有表
- [ ] upgrade 函数正确执行
- [ ] downgrade 函数正确回滚
- [ ] 迁移文档完整

---

## 参考文档

- M1-001: 数据库表结构设计
- Alembic 官方文档

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
