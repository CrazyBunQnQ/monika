# M1-023 实现列表角色卡 GET /characters

## 概述
实现获取用户角色卡列表的 API 端点,支持分页、排序、筛选和搜索功能。

## 验收标准
- [ ] 实现 GET /characters 端点
- [ ] 支持分页(page/limit)
- [ ] 支持排序(sort_by/order)
- [ ] 支持筛选(status/campaign/created_at)
- [ ] 支持搜索(name)
- [ ] 返回总数和分页信息

## 技术方案

### API 端点

```python
from fastapi import APIRouter, Depends, Query
from sqlalchemy.orm import Session
from typing import Optional, List
from datetime import datetime

router = APIRouter(prefix="/characters", tags=["characters"])

class CharacterListQuery:
    def __init__(
        self,
        page: int = Query(1, ge=1, description="页码"),
        limit: int = Query(20, ge=1, le=100, description="每页数量"),
        sort_by: str = Query("created_at", description="排序字段"),
        order: str = Query("desc", regex="^(asc|desc)$", description="排序方向"),
        search: Optional[str] = Query(None, description="搜索名称"),
        status: Optional[str] = Query(None, description="状态筛选"),
        campaign_id: Optional[str] = Query(None, description="战役 ID"),
        include_deleted: bool = Query(False, description="包含已删除")
    ):
        self.page = page
        self.limit = limit
        self.sort_by = sort_by
        self.order = order
        self.search = search
        self.status = status
        self.campaign_id = campaign_id
        self.include_deleted = include_deleted

@router.get("")
async def list_characters(
    query: CharacterListQuery = Depends(),
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db)
):
    """
    获取角色卡列表

    返回:
    - characters: 角色卡列表
    - total: 总数
    - page: 当前页
    - limit: 每页数量
    - total_pages: 总页数
    """
    # 构建查询
    q = db.query(Character).filter(Character.user_id == current_user.id)

    # 软删除筛选
    if not query.include_deleted:
        q = q.filter(Character.deleted_at.is_(None))

    # 搜索
    if query.search:
        search_term = f"%{query.search}%"
        q = q.filter(Character.name.ilike(search_term))

    # 状态筛选
    if query.status:
        q = q.filter(Character.status == query.status)

    # 战役筛选
    if query.campaign_id:
        q = q.filter(Character.campaign_id == query.campaign_id)

    # 排序
    order_by = getattr(Character, query.sort_by, None)
    if order_by is not None:
        if query.order == "asc":
            q = q.order_by(order_by.asc())
        else:
            q = q.order_by(order_by.desc())

    # 总数
    total = q.count()

    # 分页
    offset = (query.page - 1) * query.limit
    characters = q.offset(offset).limit(query.limit).all()

    # 总页数
    total_pages = (total + query.limit - 1) // query.limit

    # 转换为响应格式
    character_responses = [
        {
            "id": char.id,
            "name": char.name,
            "status": char.status,
            "campaign_id": char.campaign_id,
            "created_at": char.created_at.isoformat(),
            "updated_at": char.updated_at.isoformat(),
            "deleted_at": char.deleted_at.isoformat() if char.deleted_at else None,
            # 简要属性
            "summary": {
                "age": char.age,
                "occupation": char.occupation,
                "hp": char.hp,
                "hp_max": char.hp_max,
                "san": char.san,
                "san_max": char.san_max
            }
        }
        for char in characters
    ]

    return {
        "characters": character_responses,
        "pagination": {
            "total": total,
            "page": query.page,
            "limit": query.limit,
            "total_pages": total_pages,
            "has_next": query.page < total_pages,
            "has_prev": query.page > 1
        }
    }
```

### 允许的排序字段

```python
ALLOWED_SORT_FIELDS = [
    "created_at",     # 创建时间
    "updated_at",     # 更新时间
    "name",           # 名称
    "age",            # 年龄
    "hp",             # HP
    "san",            # SAN
    "status"          # 状态
]

def validate_sort_field(sort_by: str) -> str:
    """验证排序字段"""
    if sort_by not in ALLOWED_SORT_FIELDS:
        raise HTTPException(
            status_code=400,
            detail=f"无效的排序字段。允许: {', '.join(ALLOWED_SORT_FIELDS)}"
        )
    return sort_by
```

### 搜索实现

```python
def build_search_query(
    query: Session,
    model: Character,
    search_term: str
) -> Session:
    """
    构建搜索查询

    支持:
    - 精确匹配
    - 模糊匹配
    - 多字段搜索
    """
    if not search_term:
        return query

    search_pattern = f"%{search_term}%"

    return query.filter(
        db.or_(
            model.name.ilike(search_pattern),
            model.occupation.ilike(search_pattern),
            model.notes.ilike(search_pattern)
        )
    )
```

### 高级筛选

```python
class AdvancedFilters:
    """高级筛选器"""

    @staticmethod
    def by_date_range(
        query: Session,
        model: Character,
        field: str,
        start: Optional[datetime],
        end: Optional[datetime]
    ) -> Session:
        """日期范围筛选"""
        column = getattr(model, field)

        if start:
            query = query.filter(column >= start)
        if end:
            query = query.filter(column <= end)

        return query

    @staticmethod
    def by_hp_range(
        query: Session,
        model: Character,
        min_hp: Optional[int],
        max_hp: Optional[int]
    ) -> Session:
        """HP 范围筛选"""
        if min_hp is not None:
            query = query.filter(model.hp >= min_hp)
        if max_hp is not None:
            query = query.filter(model.hp <= max_hp)

        return query

    @staticmethod
    def by_status_list(
        query: Session,
        model: Character,
        statuses: List[str]
    ) -> Session:
        """多状态筛选"""
        return query.filter(model.status.in_(statuses))
```

### 缓存策略

```python
from functools import lru_cache
from hashlib import sha256
import json

def cache_key(
    user_id: str,
    query: CharacterListQuery
) -> str:
    """生成缓存键"""
    data = {
        "user_id": user_id,
        "page": query.page,
        "limit": query.limit,
        "sort_by": query.sort_by,
        "order": query.order,
        "search": query.search,
        "status": query.status,
        "campaign_id": query.campaign_id
    }
    json_str = json.dumps(data, sort_keys=True)
    return sha256(json_str.encode()).hexdigest()

# 使用 Redis 缓存
async def get_characters_cached(
    query: CharacterListQuery,
    current_user: User,
    db: Session
):
    cache_key_val = cache_key(current_user.id, query)

    # 尝试从缓存获取
    cached = await redis.get(f"characters:{cache_key_val}")
    if cached:
        return json.loads(cached)

    # 从数据库获取
    result = await list_characters(query, current_user, db)

    # 缓存结果(5分钟)
    await redis.setex(
        f"characters:{cache_key_val}",
        300,
        json.dumps(result)
    )

    return result
```

### 响应格式

```python
# 成功响应
{
    "characters": [
        {
            "id": "char_001",
            "name": "侦探约翰",
            "status": "alive",
            "campaign_id": "campaign_001",
            "created_at": "2026-02-01T10:00:00Z",
            "updated_at": "2026-02-06T12:00:00Z",
            "deleted_at": null,
            "summary": {
                "age": 35,
                "occupation": "私家侦探",
                "hp": 12,
                "hp_max": 12,
                "san": 60,
                "san_max": 99
            }
        }
    ],
    "pagination": {
        "total": 45,
        "page": 1,
        "limit": 20,
        "total_pages": 3,
        "has_next": true,
        "has_prev": false
    }
}
```

### 性能优化

```python
# 1. 索引优化
class Character(Base):
    __tablename__ = "characters"

    # ... 字段定义 ...

    __table_args__ = (
        Index('idx_user_created', 'user_id', 'created_at'),
        Index('idx_user_status', 'user_id', 'status'),
        Index('idx_user_name', 'user_id', 'name'),
    )

# 2. 查询优化
# 只查询需要的字段
characters = q.with_entities(
    Character.id,
    Character.name,
    Character.status,
    Character.created_at
).all()

# 3. 批量加载
# 使用 joinedload 预加载关联数据
from sqlalchemy.orm import joinedload

characters = q.options(
    joinedload(Character.campaign),
    joinedload(Character.state)
).all()
```

## 依赖关系
- 前置任务: M1-019 实现创建角色卡 POST /characters
- 被依赖: M1-028 实现角色卡列表组件

## 预估工时
2h
