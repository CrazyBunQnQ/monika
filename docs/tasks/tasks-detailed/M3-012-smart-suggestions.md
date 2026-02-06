# M3-012: 实现智能搜索建议

**任务ID**: M3-012
**标题**: 实现智能搜索建议
**类型**: backend + frontend (全栈开发)
**预估工时**: 3h
**依赖**: M3-027, M3-002

---

## 任务描述

实现智能搜索建议功能，在用户输入搜索关键词时，实时提供相关的搜索建议，包括：
- 自动补全关键词
- 相关搜索词推荐
- 热门搜索词
- 搜索历史记录
- 基于上下文的智能建议

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M3-012-01 | 设计建议数据结构 | Suggestion 表和索引 | 20min |
| M3-012-02 | 实现搜索日志记录 | 记录用户搜索行为 | 30min |
| M3-012-03 | 实现自动补全算法 | 前缀匹配 + TF-IDF | 45min |
| M3-012-04 | 实现热门词统计 | 统计热门搜索词 | 30min |
| M3-012-05 | 实现相关词推荐 | 协同过滤或词向量 | 45min |
| M3-012-06 | 实现建议 API | 实时建议接口 | 30min |
| M3-012-07 | 实现建议 UI 组件 | 下拉建议列表 | 30min |
| M3-012-08 | 编写建议测试 | 测试覆盖 | 10min |

---

## 后端代码示例

### 搜索建议服务

```python
# app/services/search_suggestion.py
from typing import List, Dict, Any, Optional
from datetime import datetime, timedelta
from collections import Counter
import re

from sqlalchemy.orm import Session
from app.db.models.search_log import SearchLog
from app.db.models.gamestate import GameEvent
from app.core.redis import redis_client

class SearchSuggestionService:
    """搜索建议服务"""

    def __init__(self, db: Session):
        self.db = db
        self.redis = redis_client

    async def get_suggestions(
        self,
        campaign_id: str,
        query: str,
        user_id: str,
        limit: int = 8,
    ) -> List[Dict[str, Any]]:
        """获取搜索建议

        Args:
            campaign_id: 战役 ID
            query: 查询字符串
            user_id: 用户 ID
            limit: 返回结果数量

        Returns:
            建议列表
        """
        suggestions = []

        # 1. 自动补全 (从事件关键词)
        completions = await self._get_completions(
            campaign_id=campaign_id,
            query=query,
            limit=limit // 2,
        )
        suggestions.extend(completions)

        # 2. 历史搜索记录 (用户个人)
        if len(query) >= 2:
            history = await self._get_search_history(
                user_id=user_id,
                query=query,
                limit=limit // 4,
            )
            suggestions.extend(history)

        # 3. 热门搜索词
        popular = await self._get_popular_searches(
            campaign_id=campaign_id,
            query=query,
            limit=limit // 4,
        )
        suggestions.extend(popular)

        # 4. 相关搜索词
        if len(query) >= 3:
            related = await self._get_related_searches(
                campaign_id=campaign_id,
                query=query,
                limit=limit // 4,
            )
            suggestions.extend(related)

        # 去重并限制数量
        seen = set()
        unique_suggestions = []
        for suggestion in suggestions:
            key = suggestion['text'].lower()
            if key not in seen and key != query.lower():
                seen.add(key)
                unique_suggestions.append(suggestion)
                if len(unique_suggestions) >= limit:
                    break

        return unique_suggestions

    async def _get_completions(
        self,
        campaign_id: str,
        query: str,
        limit: int,
    ) -> List[Dict[str, Any]]:
        """获取自动补全建议"""
        if not query:
            return []

        # 从事件中提取关键词
        events = self.db.query(GameEvent).filter(
            GameEvent.campaign_id == campaign_id,
            GameEvent.description.ilike(f"%{query}%"),
        ).limit(100).all()

        # 提取匹配的短语
        phrases = []
        for event in events:
            # 找到包含 query 的短语
            matches = re.finditer(
                rf'.{{0,20}}{re.escape(query)}.{{0,20}}',
                event.description,
                re.IGNORECASE
            )
            for match in matches:
                phrases.append({
                    'text': match.group(0),
                    'type': 'completion',
                    'source': 'event',
                })

        return phrases[:limit]

    async def _get_search_history(
        self,
        user_id: str,
        query: str,
        limit: int,
    ) -> List[Dict[str, Any]]:
        """获取用户搜索历史"""
        # 从 Redis 缓存获取
        cache_key = f"search_history:{user_id}"
        history = self.redis.lrange(cache_key, 0, -1)

        # 过滤匹配的历史记录
        matched = [
            {
                'text': h.decode('utf-8'),
                'type': 'history',
                'source': 'user',
            }
            for h in history
            if query.lower() in h.decode('utf-8').lower()
        ]

        return matched[:limit]

    async def _get_popular_searches(
        self,
        campaign_id: str,
        query: str,
        limit: int,
    ) -> List[Dict[str, Any]]:
        """获取热门搜索词"""
        # 从 Redis 缓存获取
        cache_key = f"popular_searches:{campaign_id}"
        popular = self.redis.zrevrange(cache_key, 0, 100, withscores=True)

        # 过滤匹配的热门词
        matched = [
            {
                'text': term.decode('utf-8'),
                'type': 'popular',
                'source': 'campaign',
                'count': int(score),
            }
            for term, score in popular
            if query.lower() in term.decode('utf-8').lower()
        ]

        return sorted(matched, key=lambda x: x['count'], reverse=True)[:limit]

    async def _get_related_searches(
        self,
        campaign_id: str,
        query: str,
        limit: int,
    ) -> List[Dict[str, Any]]:
        """获取相关搜索词"""
        # 找到搜索过 query 的用户也搜索过的词
        related_logs = self.db.query(SearchLog).filter(
            SearchLog.campaign_id == campaign_id,
            SearchLog.query.ilike(f"%{query}%"),
        ).all()

        # 提取相关搜索词
        related_queries = []
        for log in related_logs:
            # 找到同一用户的其他搜索
            user_logs = self.db.query(SearchLog.query).filter(
                SearchLog.user_id == log.user_id,
                SearchLog.campaign_id == campaign_id,
                SearchLog.id != log.id,
            ).all()

            for user_log in user_logs:
                if query.lower() not in user_log.query.lower():
                    related_queries.append(user_log.query)

        # 统计频率
        counter = Counter(related_queries)
        return [
            {
                'text': term,
                'type': 'related',
                'source': 'users',
                'count': count,
            }
            for term, count in counter.most_common(limit)
        ]

    async def log_search(
        self,
        campaign_id: str,
        user_id: str,
        query: str,
        results_count: int,
    ):
        """记录搜索行为"""
        # 记录到数据库
        log = SearchLog(
            campaign_id=campaign_id,
            user_id=user_id,
            query=query,
            results_count=results_count,
            created_at=datetime.utcnow(),
        )
        self.db.add(log)
        self.db.commit()

        # 更新用户搜索历史 (Redis)
        history_key = f"search_history:{user_id}"
        self.redis.lpush(history_key, query)
        self.redis.ltrim(history_key, 0, 19)  # 保留最近 20 条

        # 更新热门搜索词 (Redis)
        popular_key = f"popular_searches:{campaign_id}"
        self.redis.zincrby(popular_key, 1, query)

        # 设置过期时间
        self.redis.expire(history_key, timedelta(days=30))
        self.redis.expire(popular_key, timedelta(days=7))

    async def get_trending_queries(
        self,
        campaign_id: str,
        limit: int = 10,
    ) -> List[Dict[str, Any]]:
        """获取趋势搜索词"""
        cache_key = f"popular_searches:{campaign_id}"
        popular = self.redis.zrevrange(cache_key, 0, limit - 1, withscores=True)

        return [
            {
                'query': term.decode('utf-8'),
                'count': int(score),
            }
            for term, score in popular
        ]
```

### 搜索日志模型

```python
# app/db/models/search_log.py
from sqlalchemy import Column, String, DateTime, Integer, ForeignKey
from datetime import datetime

from app.db.database import Base

class SearchLog(Base):
    """搜索日志"""
    __tablename__ = "search_logs"

    id = Column(String, primary_key=True, index=True)
    campaign_id = Column(String, ForeignKey("campaigns.id"), nullable=False)
    user_id = Column(String, ForeignKey("users.id"), nullable=False)

    # 搜索信息
    query = Column(String, nullable=False)
    results_count = Column(Integer, default=0)

    # 时间
    created_at = Column(DateTime, default=datetime.utcnow, index=True)

    # 用于分析的元数据
    filters = Column(String)  # JSON 字符串
    selected_result = Column(String)  # 用户点击的结果 ID
```

### 搜索建议 API

```python
# app/api/search_suggestion.py
from fastapi import APIRouter, Depends, Query
from sqlalchemy.orm import Session
from pydantic import BaseModel
from typing import List

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.search_suggestion import SearchSuggestionService

router = APIRouter(prefix="/search/suggestions", tags=["search-suggestions"])

class SuggestionResponse(BaseModel):
    suggestions: List[dict]

@router.get("", response_model=SuggestionResponse)
async def get_suggestions(
    campaign_id: str,
    q: str = Query("", min_length=1),
    limit: int = 8,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取搜索建议"""
    service = SearchSuggestionService(db)

    suggestions = await service.get_suggestions(
        campaign_id=campaign_id,
        query=q,
        user_id=current_user.id,
        limit=limit,
    )

    return SuggestionResponse(suggestions=suggestions)

@router.get("/trending")
async def get_trending(
    campaign_id: str,
    limit: int = 10,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取趋势搜索词"""
    service = SearchSuggestionService(db)

    trending = await service.get_trending_queries(
        campaign_id=campaign_id,
        limit=limit,
    )

    return {"trending": trending}
```

---

## 前端代码示例

### 智能搜索输入组件

```typescript
// frontend/src/components/search/SmartSearchInput.tsx
import React, { useState, useEffect, useRef } from 'react';
import { Input } from '@/components/ui/input';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { Badge } from '@/components/ui/badge';
import { Search, Clock, TrendingUp, History } from 'lucide-react';
import { cn } from '@/lib/utils';

interface Suggestion {
  text: string;
  type: 'completion' | 'history' | 'popular' | 'related';
  source: string;
  count?: number;
}

interface SmartSearchInputProps {
  campaignId: string;
  onSearch: (query: string) => void;
  placeholder?: string;
}

export function SmartSearchInput({
  campaignId,
  onSearch,
  placeholder = "搜索事件、NPC、线索...",
}: SmartSearchInputProps) {
  const [query, setQuery] = useState('');
  const [suggestions, setSuggestions] = useState<Suggestion[]>([]);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const [loading, setLoading] = useState(false);

  const inputRef = useRef<HTMLInputElement>(null);
  const debounceRef = useRef<NodeJS.Timeout | null>(null);

  // 防抖获取建议
  useEffect(() => {
    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
    }

    if (query.length >= 1) {
      debounceRef.current = setTimeout(() => {
        fetchSuggestions();
      }, 200);
    } else {
      setSuggestions([]);
    }

    return () => {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
      }
    };
  }, [query, campaignId]);

  const fetchSuggestions = async () => {
    if (query.length < 1) return;

    setLoading(true);
    try {
      const response = await fetch(
        `/api/search/suggestions?campaign_id=${campaignId}&q=${encodeURIComponent(query)}`
      );
      const data = await response.json();
      setSuggestions(data.suggestions);
      setShowSuggestions(true);
    } catch (error) {
      console.error('获取建议失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleSelect = (suggestion: Suggestion) => {
    setQuery(suggestion.text);
    setShowSuggestions(false);
    onSearch(suggestion.text);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (!showSuggestions || suggestions.length === 0) return;

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setSelectedIndex(prev =>
          prev < suggestions.length - 1 ? prev + 1 : prev
        );
        break;
      case 'ArrowUp':
        e.preventDefault();
        setSelectedIndex(prev => (prev > 0 ? prev - 1 : 0));
        break;
      case 'Enter':
        e.preventDefault();
        if (selectedIndex >= 0 && selectedIndex < suggestions.length) {
          handleSelect(suggestions[selectedIndex]);
        } else {
          setShowSuggestions(false);
          onSearch(query);
        }
        break;
      case 'Escape':
        setShowSuggestions(false);
        setSelectedIndex(-1);
        break;
    }
  };

  const getSuggestionIcon = (type: Suggestion['type']) => {
    switch (type) {
      case 'history':
        return <Clock className="h-3 w-3" />;
      case 'popular':
        return <TrendingUp className="h-3 w-3" />;
      case 'related':
        return <History className="h-3 w-3" />;
      default:
        return <Search className="h-3 w-3" />;
    }
  };

  const getSuggestionTypeLabel = (type: Suggestion['type']) => {
    switch (type) {
      case 'history':
        return '历史';
      case 'popular':
        return '热门';
      case 'related':
        return '相关';
      default:
        return '自动补全';
    }
  };

  const highlightMatch = (text: string, query: string) => {
    if (!query) return text;

    const regex = new RegExp(`(${query})`, 'gi');
    const parts = text.split(regex);

    return parts.map((part, index) =>
      regex.test(part) ? (
        <mark key={index} className="bg-yellow-200 dark:bg-yellow-800 rounded">
          {part}
        </mark>
      ) : (
        part
      )
    );
  };

  return (
    <div className="relative">
      <Popover open={showSuggestions} onOpenChange={setShowSuggestions}>
        <PopoverTrigger asChild>
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              ref={inputRef}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={handleKeyDown}
              onFocus={() => {
                if (query.length >= 1 && suggestions.length > 0) {
                  setShowSuggestions(true);
                }
              }}
              placeholder={placeholder}
              className="pl-10"
            />
            {loading && (
              <div className="absolute right-3 top-1/2 -translate-y-1/2">
                <div className="h-4 w-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />
              </div>
            )}
          </div>
        </PopoverTrigger>

        <PopoverContent
          className="w-full p-0"
          align="start"
          onOpenAutoFocus={(e) => e.preventDefault()}
        >
          {suggestions.length > 0 ? (
            <div className="max-h-64 overflow-y-auto">
              {suggestions.map((suggestion, index) => (
                <div
                  key={index}
                  className={cn(
                    "flex items-center gap-3 px-3 py-2 cursor-pointer transition-colors",
                    index === selectedIndex && "bg-accent",
                    "hover:bg-accent/50"
                  )}
                  onClick={() => handleSelect(suggestion)}
                  onMouseEnter={() => setSelectedIndex(index)}
                >
                  <div className="text-muted-foreground">
                    {getSuggestionIcon(suggestion.type)}
                  </div>
                  <div className="flex-1 truncate">
                    {highlightMatch(suggestion.text, query)}
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant="outline" className="text-xs">
                      {getSuggestionTypeLabel(suggestion.type)}
                    </Badge>
                    {suggestion.count && (
                      <span className="text-xs text-muted-foreground">
                        {suggestion.count}
                      </span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="px-3 py-4 text-center text-sm text-muted-foreground">
              {query.length < 1 ? '输入关键词开始搜索' : '没有相关建议'}
            </div>
          )}
        </PopoverContent>
      </Popover>
    </div>
  );
}
```

### 趋势搜索组件

```typescript
// frontend/src/components/search/TrendingSearches.tsx
import React, { useEffect, useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { TrendingUp } from 'lucide-react';

interface TrendingQuery {
  query: string;
  count: number;
}

export function TrendingSearches({
  campaignId,
  onSelect,
}: {
  campaignId: string;
  onSelect: (query: string) => void;
}) {
  const [trending, setTrending] = useState<TrendingQuery[]>([]);

  useEffect(() => {
    loadTrending();
  }, [campaignId]);

  const loadTrending = async () => {
    try {
      const response = await fetch(
        `/api/search/suggestions/trending?campaign_id=${campaignId}`
      );
      const data = await response.json();
      setTrending(data.trending);
    } catch (error) {
      console.error('加载趋势搜索失败:', error);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base flex items-center gap-2">
          <TrendingUp className="h-4 w-4" />
          热门搜索
        </CardTitle>
      </CardHeader>
      <CardContent>
        {trending.length > 0 ? (
          <div className="flex flex-wrap gap-2">
            {trending.map((item, index) => (
              <Badge
                key={index}
                variant="secondary"
                className="cursor-pointer hover:bg-accent"
                onClick={() => onSelect(item.query)}
              >
                {item.query}
                <span className="ml-1 text-xs opacity-70">
                  ({item.count})
                </span>
              </Badge>
            ))}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">暂无热门搜索</p>
        )}
      </CardContent>
    </Card>
  );
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/search_suggestion.py` | 创建 | 搜索建议服务 |
| `app/db/models/search_log.py` | 创建 | 搜索日志模型 |
| `app/api/search_suggestion.py` | 创建 | 搜索建议 API |
| `frontend/src/components/search/SmartSearchInput.tsx` | 创建 | 智能搜索输入组件 |
| `frontend/src/components/search/TrendingSearches.tsx` | 创建 | 趋势搜索组件 |
| `tests/test_search_suggestion.py` | 创建 | 搜索建议测试 |

---

## 验收标准

- [ ] 自动补全准确匹配事件关键词
- [ ] 搜索历史正确记录和显示
- [ ] 热门搜索词统计准确
- [ ] 相关搜索推荐有价值
- [ ] 键盘导航流畅（上下选择、回车确认）
- [ ] 高亮匹配部分清晰可见
- [ ] 性能满足（防抖有效，延迟低）
- [ ] 趋势搜索实时更新

---

## 参考文档

- M3-027: 全文检索功能
- M3-002: 搜索功能实现
- Redis Sorted Set 文档
- 搜索建议最佳实践

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
