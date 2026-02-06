# M5-049: 设计成长记录表

**任务ID**: M5-049
**标题**: 设计成长记录表
**类型**: db (数据库设计)
**预估工时**: 2h
**依赖**: M1-017

---

## 任务描述

设计角色成长记录的数据表结构，用于记录技能经验值、成长检定历史等信息。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M5-049-01 | 分析成长需求 | 确定需要记录的数据 | 20min |
| M5-049-02 | 设计 SkillGrowth 表 | 成长记录表 | 30min |
| M5-049-03 | 设计 SkillUsage 表 | 技能使用记录 | 25min |
| M5-049-04 | 编写迁移脚本 | Alembic 迁移 | 25min |
| M5-049-05 | 创建索引 | 查询优化 | 15min |
| M5-049-06 | 编写表文档 | 数据字典 | 10min |

---

## SkillGrowth 表结构

```sql
-- 技能成长记录表
CREATE TABLE skill_growth (
    -- 主键
    id BIGSERIAL PRIMARY KEY,

    -- 关联
    character_id INTEGER NOT NULL,
    skill_id VARCHAR(100) NOT NULL,

    -- 成长信息
    old_value INTEGER NOT NULL,           -- 成长前技能值
    new_value INTEGER NOT NULL,           -- 成长后技能值
    increase_amount INTEGER NOT NULL,     -- 增加的点数

    -- 检定信息
    roll_result INTEGER,                  -- 成长检定结果
    roll_success BOOLEAN NOT NULL,        -- 是否成功
    was_critical BOOLEAN DEFAULT FALSE,   -- 是否大成功

    -- 时间
    game_time VARCHAR(100),               -- 游戏内时间
    occurred_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- 触发
    trigger_event_id VARCHAR(100),        -- 触发成长的事件
    trigger_reason TEXT,                  -- 成功原因描述

    -- 元数据
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- 外键
    CONSTRAINT fk_growth_character
        FOREIGN KEY (character_id)
        REFERENCES characters(id)
        ON DELETE CASCADE,

    -- 索引
    CONSTRAINT idx_growth_character_skill
        UNIQUE (character_id, skill_id, occurred_at)
);

-- 索引
CREATE INDEX idx_growth_character ON skill_growth(character_id);
CREATE INDEX idx_growth_skill ON skill_growth(skill_id);
CREATE INDEX idx_growth_date ON skill_growth(occurred_at DESC);
CREATE INDEX idx_growth_success ON skill_growth(roll_success);

-- 注释
COMMENT ON TABLE skill_growth IS '角色技能成长记录';
COMMENT ON COLUMN skill_growth.increase_amount IS '技能增加的点数 (1d10 或 2d10)';
COMMENT ON COLUMN skill_growth.was_critical IS '大成功获得双倍增加';
```

---

## SkillUsage 表结构

```sql
-- 技能使用记录表
CREATE TABLE skill_usage (
    -- 主键
    id BIGSERIAL PRIMARY KEY,

    -- 关联
    character_id INTEGER NOT NULL,
    skill_id VARCHAR(100) NOT NULL,

    -- 使用信息
    use_type VARCHAR(50) NOT NULL,       -- check, combat, interaction 等
    difficulty VARCHAR(20),              -- regular, hard, extreme
    modifier INTEGER DEFAULT 0,          -- 修正值

    -- 结果
    roll_result INTEGER,                 -- 掷骰结果
    success_level VARCHAR(20),           -- critical, extreme, hard, regular, failure, fumble
    was_success BOOLEAN,

    -- 时间
    session_id VARCHAR(100),
    game_time VARCHAR(100),
    occurred_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- 元数据
    event_id VARCHAR(100),
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- 外键
    CONSTRAINT fk_usage_character
        FOREIGN KEY (character_id)
        REFERENCES characters(id)
        ON DELETE CASCADE,

    -- 索引
    CONSTRAINT idx_usage_character_skill
        UNIQUE (character_id, skill_id, event_id)
);

-- 索引
CREATE INDEX idx_usage_character ON skill_usage(character_id);
CREATE INDEX idx_usage_skill ON skill_usage(skill_id);
CREATE INDEX idx_usage_session ON skill_usage(session_id);
CREATE INDEX idx_usage_date ON skill_usage(occurred_at DESC);
CREATE INDEX idx_usage_success ON skill_usage(was_success);

-- 注释
COMMENT ON TABLE skill_usage IS '角色技能使用记录';
COMMENT ON COLUMN skill_usage.use_type IS '使用类型: check(检定), combat(战斗), interaction(互动)等';
COMMENT ON COLUMN skill_usage.success_level IS '成功等级: critical(大成功), extreme(极难), hard(困难), regular(普通), failure(失败), fumble(大失败)';
```

---

## 统计视图

```sql
-- 技能使用统计视图
CREATE VIEW v_skill_usage_stats AS
SELECT
    character_id,
    skill_id,
    COUNT(*) as total_uses,
    SUM(CASE WHEN was_success THEN 1 ELSE 0 END) as success_count,
    SUM(CASE WHEN was_success THEN 0 ELSE 1 END) as failure_count,
    ROUND(
        100.0 * SUM(CASE WHEN was_success THEN 1 ELSE 0 END) / NULLIF(COUNT(*), 0),
        2
    ) as success_rate,
    MAX(occurred_at) as last_used
FROM skill_usage
GROUP BY character_id, skill_id;

-- 技能成长统计视图
CREATE VIEW v_skill_growth_stats AS
SELECT
    character_id,
    skill_id,
    COUNT(*) as growth_attempts,
    SUM(CASE WHEN roll_success THEN 1 ELSE 0 END) as successful_growths,
    SUM(increase_amount) as total_increase,
    AVG(increase_amount) as avg_increase,
    MAX(new_value) as current_value
FROM skill_growth
GROUP BY character_id, skill_id;
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `alembic/versions/xxx_add_growth_tables.py` | 创建 | 迁移脚本 |
| `app/db/models/skill_growth.py` | 创建 | 数据模型 |
| `docs/database/growth-tables.md` | 创建 | 表文档 |

---

## 验收标准

- [ ] 表结构定义完整
- [ ] 外键关系正确
- [ ] 索引合理
- [ ] 视图有效
- [ ] 迁移脚本可执行

---

## 参考文档

- M1-017: Characters 表结构
- M1-005: 技能系统

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
