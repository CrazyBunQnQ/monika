# M3-002: 实现全文搜索功能

**任务ID**: M3-002
**标题**: 实现全文搜索功能
**类型**: backend (后端开发)
**预估工时**: 2.5h
**依赖**: M1-080

---

## 任务描述

实现游戏内容的全文搜索功能，支持搜索场景、NPC、线索、聊天记录等。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M3-002-01 | 设计搜索索引结构 | Index Schema | 20min |
| M3-002-02 | 实现 Elasticsearch 集成 | ES Setup | 35min |
| M3-002-03 | 实现索引同步 | Index Sync | 30min |
| M3-002-04 | 实现搜索 API | Search Endpoint | 30min |
| M3-002-05 | 实现搜索过滤 | Filters | 25min |
| M3-002-06 | 编写搜索测试 | 测试覆盖 | 20min |

---

## Elasticsearch 配置

```python
# app/core/elasticsearch.py
from elasticsearch import AsyncElasticsearch
import os

class ElasticsearchClient:
    """Elasticsearch 客户端"""

    def __init__(self):
        self.client = AsyncElasticsearch(
            hosts=[os.getenv("ELASTICSEARCH_URL", "http://localhost:9200")],
        )

    async def create_index(self, index_name: str, mapping: dict):
        """创建索引"""
        exists = await self.client.indices.exists(index=index_name)
        if not exists:
            await self.client.indices.create(
                index=index_name,
                body={"mappings": mapping}
            )

    async def index_document(
        self,
        index_name: str,
        doc_id: str,
        document: dict,
    ):
        """索引文档"""
        await self.client.index(
            index=index_name,
            id=doc_id,
            document=document,
        )

    async def search(
        self,
        index_name: str,
        query: dict,
        filters: dict = None,
        size: int = 20,
        from_: int = 0,
    ) -> dict:
        """搜索"""
        body = {
            "query": query,
            "size": size,
            "from": from_,
        }

        if filters:
            body["filter"] = filters

        response = await self.client.search(index=index_name, body=body)
        return response

    async def delete_document(self, index_name: str, doc_id: str):
        """删除文档"""
        await self.client.delete(index=index_name, id=doc_id)

# 全局客户端
es_client = ElasticsearchClient()
```

---

## 搜索索引映射

```python
# app/search/mappings.py

# 游戏事件索引
GAME_EVENTS_MAPPING = {
    "properties": {
        "content": {"type": "text", "analyzer": "ik_max_word"},
        "timestamp": {"type": "date"},
        "type": {"type": "keyword"},
        "campaign_id": {"type": "keyword"},
        "scene_id": {"type": "keyword"},
        "character_id": {"type": "keyword"},
        "user_id": {"type": "keyword"},
    }
}

# NPC 索引
NPCS_MAPPING = {
    "properties": {
        "name": {"type": "text", "analyzer": "ik_max_word"},
        "description": {"type": "text", "analyzer": "ik_max_word"},
        "background": {"type": "text", "analyzer": "ik_max_word"},
        "campaign_id": {"type": "keyword"},
        "scene_id": {"type": "keyword"},
    }
}

# 线索索引
CLUES_MAPPING = {
    "properties": {
        "title": {"type": "text", "analyzer": "ik_max_word"},
        "content": {"type": "text", "analyzer": "ik_max_word"},
        "tags": {"type": "keyword"},
        "campaign_id": {"type": "keyword"},
        "revealed_to": {"type": "keyword"},
    }
}
```

---

## 搜索服务

```python
# app/services/search.py
from typing import List, Dict, Any, Optional
from app.core.elasticsearch import es_client
from app.search.mappings import GAME_EVENTS_MAPPING, NPCS_MAPPING, CLUES_MAPPING

class SearchService:
    """搜索服务"""

    def __init__(self):
        self.es = es_client
        self.indices = {
            "events": "game_events",
            "npcs": "npcs",
            "clues": "clues",
        }

    async def initialize_indices(self):
        """初始化索引"""
        await self.es.create_index(
            self.indices["events"],
            GAME_EVENTS_MAPPING
        )
        await self.es.create_index(
            self.indices["npcs"],
            NPCS_MAPPING
        )
        await self.es.create_index(
            self.indices["clues"],
            CLUES_MAPPING
        )

    async def search_all(
        self,
        query: str,
        campaign_id: str,
        types: Optional[List[str]] = None,
        size: int = 20,
    ) -> Dict[str, Any]:
        """全局搜索"""
        results = {
            "events": [],
            "npcs": [],
            "clues": [],
        }

        # 搜索事件
        if not types or "events" in types:
            events = await self._search_events(query, campaign_id, size)
            results["events"] = events

        # 搜索 NPC
        if not types or "npcs" in types:
            npcs = await self._search_npcs(query, campaign_id, size)
            results["npcs"] = npcs

        # 搜索线索
        if not types or "clues" in types:
            clues = await self._search_clues(query, campaign_id, size)
            results["clues"] = clues

        return results

    async def _search_events(
        self,
        query: str,
        campaign_id: str,
        size: int = 20,
    ) -> List[Dict]:
        """搜索事件"""
        query_body = {
            "bool": {
                "must": [
                    {
                        "match": {
                            "content": query
                        }
                    },
                    {
                        "term": {
                            "campaign_id": campaign_id
                        }
                    }
                ]
            }
        }

        response = await self.es.search(
            self.indices["events"],
            query_body,
            size=size,
        )

        return [
            {
                "id": hit["_id"],
                "score": hit["_score"],
                **hit["_source"],
            }
            for hit in response["hits"]["hits"]
        ]

    async def _search_npcs(
        self,
        query: str,
        campaign_id: str,
        size: int = 20,
    ) -> List[Dict]:
        """搜索 NPC"""
        query_body = {
            "bool": {
                "must": [
                    {
                        "multi_match": {
                            "query": query,
                            "fields": ["name", "description", "background"],
                        }
                    },
                    {
                        "term": {
                            "campaign_id": campaign_id
                        }
                    }
                ]
            }
        }

        response = await self.es.search(
            self.indices["npcs"],
            query_body,
            size=size,
        )

        return [
            {
                "id": hit["_id"],
                "score": hit["_score"],
                **hit["_source"],
            }
            for hit in response["hits"]["hits"]
        ]

    async def _search_clues(
        self,
        query: str,
        campaign_id: str,
        size: int = 20,
    ) -> List[Dict]:
        """搜索线索"""
        query_body = {
            "bool": {
                "must": [
                    {
                        "multi_match": {
                            "query": query,
                            "fields": ["title", "content"],
                        }
                    },
                    {
                        "term": {
                            "campaign_id": campaign_id
                        }
                    }
                ]
            }
        }

        response = await self.es.search(
            self.indices["clues"],
            query_body,
            size=size,
        )

        return [
            {
                "id": hit["_id"],
                "score": hit["_score"],
                **hit["_source"],
            }
            for hit in response["hits"]["hits"]
        ]

    async def index_event(self, event: Dict[str, Any]):
        """索引事件"""
        await self.es.index_document(
            self.indices["events"],
            event["id"],
            {
                "content": event.get("description", ""),
                "timestamp": event.get("timestamp"),
                "type": event.get("type"),
                "campaign_id": event.get("campaign_id"),
                "scene_id": event.get("scene_id"),
                "character_id": event.get("character_id"),
                "user_id": event.get("user_id"),
            }
        )

    async def index_npc(self, npc: Dict[str, Any]):
        """索引 NPC"""
        await self.es.index_document(
            self.indices["npcs"],
            npc["id"],
            {
                "name": npc.get("name"),
                "description": npc.get("description", ""),
                "background": npc.get("background", ""),
                "campaign_id": npc.get("campaign_id"),
                "scene_id": npc.get("scene_id"),
            }
        )

    async def index_clue(self, clue: Dict[str, Any]):
        """索引线索"""
        await self.es.index_document(
            self.indices["clues"],
            clue["id"],
            {
                "title": clue.get("title"),
                "content": clue.get("content", ""),
                "tags": clue.get("tags", []),
                "campaign_id": clue.get("campaign_id"),
                "revealed_to": clue.get("revealed_to", []),
            }
        )
```

---

## 搜索 API

```python
# app/api/search.py
from fastapi import APIRouter, Depends, Query
from pydantic import BaseModel
from typing import Optional, List

from app.services.search import SearchService
from app.api.deps.auth import get_current_user
from app.db.models.user import User

router = APIRouter(prefix="/search", tags=["search"])

class SearchRequest(BaseModel):
    query: str
    campaign_id: str
    types: Optional[List[str]] = None
    size: int = 20

class SearchResponse(BaseModel):
    events: list
    npcs: list
    clues: list

@router.post("", response_model=SearchResponse)
async def search(
    request: SearchRequest,
    current_user: User = Depends(get_current_user),
):
    """搜索游戏内容"""
    service = SearchService()

    results = await service.search_all(
        query=request.query,
        campaign_id=request.campaign_id,
        types=request.types,
        size=request.size,
    )

    return SearchResponse(**results)
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/core/elasticsearch.py` | 创建 | ES 客户端 |
| `app/search/mappings.py` | 创建 | 索引映射 |
| `app/services/search.py` | 创建 | 搜索服务 |
| `app/api/search.py` | 创建 | 搜索 API |
| `tests/test_search.py` | 创建 | 搜索测试 |

---

## 验收标准

- [ ] 索引创建成功
- [ ] 文档索引正确
- [ ] 搜索结果准确
- [ ] 过滤功能有效
- [ ] 性能满足要求

---

## 参考文档

- Elasticsearch 文档
- M1-080: 事件日志系统

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
