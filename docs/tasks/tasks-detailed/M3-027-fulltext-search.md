# M3-027: 创建事件全文索引

**任务ID**: M3-027
**标题**: 创建事件全文索引
**类型**: backend (后端开发)
**预估工时**: 4h
**依赖**: M3-001

---

## 任务描述

为事件日志创建全文索引，支持快速检索历史事件、对话内容、线索描述等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M3-027-01 | 选择全文索引方案 | PostgreSQL 全文搜索 | 20min |
| M3-027-02 | 设计索引结构 | 索引字段和权重 | 25min |
| M3-027-03 | 创建全文索引 | SQL DDL | 30min |
| M3-027-04 | 实现索引更新触发器 | 自动更新索引 | 30min |
| M3-027-05 | 实现搜索查询 | ts_query 查询 | 45min |
| M3-027-06 | 实现结果排序 | 相关性排序 | 30min |
| M3-027-07 | 编写搜索测试 | 性能和准确性测试 | 30min |
| M3-027-08 | 编写搜索文档 | API 说明 | 15min |

---

## 索引方案

使用 PostgreSQL 内置的全文搜索 (tsvector):

```sql
-- 在 events 表上添加全文索引列
ALTER TABLE events
ADD COLUMN search_vector tsvector
GENERATED ALWAYS AS (
  setweight(to_tsvector('english', coalesce(raw_message, '')), 'A') ||
  setweight(to_tsvector('english', coalesce(narration_text, '')), 'B') ||
  setweight(to_tsvector('english', coalesce(event_type, '')), 'C') ||
  setweight(to_tsvector('english', coalesce(description, '')), 'D')
) STORED;

-- 创建 GIN 索引
CREATE INDEX idx_events_search ON events USING GIN (search_vector);

-- 创建复合索引 (session_id + 全文搜索)
CREATE INDEX idx_events_session_search ON events USING GIN (session_id, search_vector);
```

---

## 权重说明

| 字段 | 权重 | 说明 |
|------|------|------|
| raw_message | A | 用户原始消息，最相关 |
| narration_text | B | 叙事文本，次相关 |
| event_type | C | 事件类型，辅助搜索 |
| description | D | 描述信息，相关性低 |

---

## 搜索查询实现

```python
# app/services/search.py
from sqlalchemy import text, func
from sqlalchemy.orm import Session
from typing import List, Dict, Optional

class EventSearchService:
    def __init__(self, db: Session):
        self.db = db

    def search_events(
        self,
        session_id: str,
        query: str,
        user_id: str,
        user_role: str,
        limit: int = 20,
        offset: int = 0
    ) -> List[Dict]:
        """搜索事件"""

        # 将查询转换为 tsquery
        ts_query = self._parse_query(query)

        # 构建基础查询
        sql = text("""
            SELECT
                event_id,
                session_id,
                event_type,
                raw_message,
                narration_text,
                timestamp,
                visibility,
                ts_rank(search_vector, :ts_query) AS rank
            FROM events
            WHERE
                session_id = :session_id
                AND search_vector @@ :ts_query
                AND :visibility_check
            ORDER BY rank DESC, timestamp DESC
            LIMIT :limit OFFSET :offset
        """)

        # 构建可见性检查
        visibility_check = self._build_visibility_check(user_id, user_role)

        # 执行查询
        result = self.db.execute(
            sql,
            {
                "session_id": session_id,
                "ts_query": ts_query,
                "visibility_check": visibility_check,
                "limit": limit,
                "offset": offset
            }
        )

        return [dict(row) for row in result]

    def _parse_query(self, query: str) -> str:
        """解析查询字符串为 tsquery"""
        # 简单实现：处理中文和英文
        import re

        # 移除特殊字符
        query = re.sub(r'[^\w\s\u4e00-\u9fff]', ' ', query)

        # 分词
        words = query.split()

        # 转换为 tsquery 格式
        # 中文使用 & 连接，英文使用词干
        tsquery_parts = []
        for word in words:
            if word.strip():
                # 检查是否为中文
                if '\u4e00' <= word[0] <= '\u9fff':
                    tsquery_parts.append(word)
                else:
                    # 英文词干化
                    tsquery_parts.append(f"{word}:*")

        return " & ".join(tsquery_parts)

    def _build_visibility_check(self, user_id: str, user_role: str) -> str:
        """构建可见性检查 SQL"""
        checks = ["visibility = 'public'"]

        if user_role == 'kp':
            checks.append("visibility = 'kp'")

        checks.append(f"visibility = 'player:{user_id}'")

        return " OR ".join([f"({c})" for c in checks])

    def get_search_suggestions(
        self,
        session_id: str,
        query: str,
        limit: int = 5
    ) -> List[str]:
        """获取搜索建议"""

        sql = text("""
            SELECT
                raw_message,
                ts_headline('english', raw_message, :ts_query) AS highlight
            FROM events
            WHERE
                session_id = :session_id
                AND search_vector @@ :ts_query
            ORDER BY ts_rank(search_vector, :ts_query) DESC
            LIMIT :limit
        """)

        ts_query = self._parse_query(query)

        result = self.db.execute(
            sql,
            {
                "session_id": session_id,
                "ts_query": ts_query,
                "limit": limit
            }
        )

        return [
            {
                "text": row.raw_message,
                "highlight": row.highlight
            }
            for row in result
        ]
```

---

## API 端点

```yaml
# POST /events/search
搜索事件

requestBody:
  content:
    application/json:
      schema:
        type: object
        required: [query]
        properties:
          query:
            type: string
            description: 搜索关键词
          session_id:
            type: string
          limit:
            type: integer
            default: 20
          offset:
            type: integer
            default: 0

responses:
  200:
    description: 搜索成功
    content:
      application/json:
        schema:
          type: object
          properties:
            total:
              type: integer
            results:
              type: array
              items:
                $ref: '#/components/schemas/Event'
            highlights:
              type: object
              description: 高亮匹配文本
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `alembic/versions/xxx_add_search_index.py` | 创建 | 迁移脚本 |
| `app/services/search.py` | 创建 | 搜索服务 |
| `app/api/search.py` | 创建 | 搜索 API |
| `tests/test_search.py` | 创建 | 搜索测试 |

---

## 搜索性能优化

```sql
-- 维护索引 (定期执行)
ANALYZE events;
REINDEX INDEX idx_events_search;

-- VACUUM 优化表
VACUUM ANALYZE events;

-- 监控索引使用情况
SELECT
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
WHERE tablename = 'events';
```

---

## 验收标准

- [ ] 全文索引创建成功
- [ ] 中文搜索支持
- [ ] 英文搜索支持
- [ ] 相关性排序合理
- [ ] 可见性过滤正确
- [ ] 搜索性能 < 100ms

---

## 参考文档

- PostgreSQL 全文搜索文档
- M3-001: Events 表结构扩展

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
