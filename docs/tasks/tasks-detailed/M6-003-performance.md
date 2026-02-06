# M6-003: 实现性能优化

**任务ID**: M6-003
**标题**: 实现性能优化
**类型**: optimization (性能优化)
**预估工时**: 3h
**依赖**: M1

---

## 任务描述

实现系统性能优化，包括数据库查询优化、缓存策略、前端性能优化等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M6-003-01 | 性能分析 | Profiling | 30min |
| M6-003-02 | 数据库查询优化 | Query Optimization | 40min |
| M6-003-03 | 实现缓存策略 | Caching | 35min |
| M6-003-04 | 实现分页优化 | Pagination | 25min |
| M6-003-05 | 前端性能优化 | Frontend Perf | 35min |
| M6-003-06 | 实现资源压缩 | Compression | 20min |
| M6-003-07 | 性能监控 | Monitoring | 25min |

---

## 数据库查询优化

```python
# app/db/queries.py
from sqlalchemy.orm import Session, joinedload, selectinload
from sqlalchemy import func
from typing import List, Optional

class OptimizedQueries:
    """优化查询"""

    @staticmethod
    def get_character_with_skills(
        db: Session,
        character_id: str
    ) -> Optional[Character]:
        """获取角色及其技能（预加载）"""
        return db.query(Character)\
            .options(
                joinedload(Character.user),
                selectinload(Character.skills),
            )\
            .filter(Character.id == character_id)\
            .first()

    @staticmethod
    def list_characters_paginated(
        db: Session,
        skip: int = 0,
        limit: int = 20,
        user_id: Optional[str] = None,
    ) -> tuple[List[Character], int]:
        """分页查询角色，返回数据和总数"""
        query = db.query(Character)

        if user_id:
            query = query.filter(Character.user_id == user_id)

        # 获取总数
        total = query.count()

        # 分页
        characters = query.offset(skip).limit(limit).all()

        return characters, total
```

---

## Redis 缓存

```python
# app/core/cache.py
import redis
import json
from typing import Optional, Any
from datetime import timedelta

redis_client = redis.Redis(
    host="localhost",
    port=6379,
    db=0,
    decode_responses=True,
)

class CacheService:
    """缓存服务"""

    @staticmethod
    def get(key: str) -> Optional[Any]:
        """获取缓存"""
        data = redis_client.get(key)
        if data:
            return json.loads(data)
        return None

    @staticmethod
    def set(
        key: str,
        value: Any,
        ttl: int = 3600,
    ):
        """设置缓存"""
        redis_client.setex(
            key,
            ttl,
            json.dumps(value),
        )

    @staticmethod
    def delete(key: str):
        """删除缓存"""
        redis_client.delete(key)

    @staticmethod
    def invalidate_pattern(pattern: str):
        """按模式删除缓存"""
        for key in redis_client.scan_iter(match=pattern):
            redis_client.delete(key)

# 缓存装饰器
def cache_result(ttl: int = 3600, key_prefix: str = ""):
    """缓存结果装饰器"""
    def decorator(func):
        def wrapper(*args, **kwargs):
            # 生成缓存键
            cache_key = f"{key_prefix}:{func.__name__}:{str(args)}:{str(kwargs)}"

            # 尝试获取缓存
            cached = CacheService.get(cache_key)
            if cached is not None:
                return cached

            # 执行函数
            result = func(*args, **kwargs)

            # 设置缓存
            CacheService.set(cache_key, result, ttl)

            return result
        return wrapper
    return decorator
```

---

## 分页优化

```python
# app/api/pagination.py
from fastapi import Query
from typing import Generic, TypeVar, List
from pydantic import BaseModel

T = TypeVar('T')

class PaginatedResponse(BaseModel, Generic[T]):
    """分页响应"""
    items: List[T]
    total: int
    page: int
    page_size: int
    total_pages: int

    @classmethod
    def create(
        cls,
        items: List[T],
        total: int,
        page: int,
        page_size: int,
    ) -> "PaginatedResponse[T]":
        """创建分页响应"""
        total_pages = (total + page_size - 1) // page_size
        return cls(
            items=items,
            total=total,
            page=page,
            page_size=page_size,
            total_pages=total_pages,
        )

def get_pagination_params(
    page: int = Query(1, ge=1, description="页码"),
    page_size: int = Query(20, ge=1, le=100, description="每页数量"),
) -> tuple[int, int]:
    """获取分页参数"""
    skip = (page - 1) * page_size
    return skip, page_size
```

---

## 前端性能优化

```typescript
// frontend/src/utils/optimization.ts

// 防抖
export function debounce<T extends (...args: any[]) => any>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeout: NodeJS.Timeout | null = null

  return function executedFunction(...args: Parameters<T>) {
    const later = () => {
      timeout = null
      func(...args)
    }

    if (timeout) clearTimeout(timeout)
    timeout = setTimeout(later, wait)
  }
}

// 节流
export function throttle<T extends (...args: any[]) => any>(
  func: T,
  limit: number
): (...args: Parameters<T>) => void {
  let inThrottle: boolean

  return function executedFunction(...args: Parameters<T>) {
    if (!inThrottle) {
      func(...args)
      inThrottle = true
      setTimeout(() => (inThrottle = false), limit)
    }
  }
}

// 虚拟滚动
export function VirtualScroll<T>({
  items,
  itemHeight,
  containerHeight,
  renderItem,
}: {
  items: T[]
  itemHeight: number
  containerHeight: number
  renderItem: (item: T, index: number) => React.ReactNode
}) {
  const [scrollTop, setScrollTop] = useState(0)

  const visibleStart = Math.floor(scrollTop / itemHeight)
  const visibleEnd = Math.min(
    visibleStart + Math.ceil(containerHeight / itemHeight) + 1,
    items.length
  )

  const visibleItems = items.slice(visibleStart, visibleEnd)
  const totalHeight = items.length * itemHeight
  const offsetY = visibleStart * itemHeight

  return (
    <div
      style={{ height: containerHeight, overflow: 'auto' }}
      onScroll={(e) => setScrollTop(e.currentTarget.scrollTop)}
    >
      <div style={{ height: totalHeight, position: 'relative' }}>
        <div style={{ transform: `translateY(${offsetY}px)` }}>
          {visibleItems.map((item, index) =>
            renderItem(item, visibleStart + index)
          )}
        </div>
      </div>
    </div>
  )
}

// 图片懒加载
export function LazyImage({
  src,
  alt,
  placeholder,
}: {
  src: string
  alt: string
  placeholder?: string
}) {
  const [imageSrc, setImageSrc] = useState(placeholder || '')
  const imgRef = useRef<HTMLImageElement>(null)

  useEffect(() => {
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setImageSrc(src)
          observer.disconnect()
        }
      },
      { rootMargin: '100px' }
    )

    if (imgRef.current) {
      observer.observe(imgRef.current)
    }

    return () => observer.disconnect()
  }, [src])

  return (
    <img
      ref={imgRef}
      src={imageSrc}
      alt={alt}
      loading="lazy"
    />
  )
}
```

---

## 资源压缩

```python
# app/main.py
from fastapi.middleware.gzip import GZipMiddleware

app = FastAPI()

# 添加 Gzip 压缩
app.add_middleware(GZipMiddleware, minimum_size=1000)
```

---

## 性能监控

```python
# app/core/monitoring.py
import time
from functools import wraps
from contextlib import contextmanager

def timing(func):
    """计时装饰器"""
    @wraps(func)
    def wrapper(*args, **kwargs):
        start = time.time()
        result = func(*args, **kwargs)
        end = time.time()

        duration = (end - start) * 1000  # 毫秒
        print(f"{func.__name__} took {duration:.2f}ms")

        return result
    return wrapper

@contextmanager
def measure_time(name: str):
    """测量代码块执行时间"""
    start = time.time()
    yield
    end = time.time()
    duration = (end - start) * 1000
    print(f"{name} took {duration:.2f}ms")
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/db/queries.py` | 创建 | 优化查询 |
| `app/core/cache.py` | 创建 | 缓存服务 |
| `app/api/pagination.py` | 创建 | 分页工具 |
| `frontend/src/utils/optimization.ts` | 创建 | 前端优化 |
| `app/core/monitoring.py` | 创建 | 性能监控 |

---

## 验收标准

- [ ] 查询性能提升
- [ ] 缓存命中率良好
- [ ] 分页响应快速
- [ ] 前端加载优化
- [ ] 资源大小减小
- [ ] 监控数据可用

---

## 参考文档

- FastAPI 性能优化指南
- Redis 缓存最佳实践
- React 性能优化

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
