# M3-011: 实现 AI 问答功能

**任务ID**: M3-011
**标题**: 实现 AI 问答功能
**类型**: backend + frontend (全栈开发)
**预估工时**: 5h
**依赖**: M3-001, M3-027

---

## 任务描述

实现基于游戏记忆的 AI 问答功能，允许玩家和 KP 向 AI 提问关于游戏历史的问题，如"我们上次发现了什么线索？"、"某某角色的 HP 是多少？"等，AI 从事件日志和结构化摘要中检索相关信息并回答。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M3-011-01 | 设计问答 API 接口 | QA API 设计 | 30min |
| M3-011-02 | 实现上下文检索服务 | 从记忆中检索相关上下文 | 1h |
| M3-011-03 | 实现 RAG 检索增强 | 向量检索 + 全文检索混合 | 1h |
| M3-011-04 | 实现问答提示词工程 | 优化 QA 提示词 | 45min |
| M3-011-05 | 实现问答流式输出 | SSE 流式响应 | 30min |
| M3-011-06 | 实现问答 UI 组件 | 聊天式问答界面 | 1h |
| M3-011-07 | 实现上下文引用显示 | 显示引用来源 | 30min |
| M3-011-08 | 编写问答测试 | 测试覆盖 | 15min |

---

## 后端代码示例

### 问答服务

```python
# app/services/qa_service.py
from typing import List, Dict, Any, Optional
from datetime import datetime
import json

from app.services.llm import LLMService
from app.services.search import SearchService
from app.core.vector_store import VectorStore
from app.db.models.summary import SessionSummary

class QAService:
    """AI 问答服务"""

    def __init__(self, db):
        self.db = db
        self.llm = LLMService()
        self.search = SearchService()
        self.vector_store = VectorStore()

    async def ask_question(
        self,
        campaign_id: str,
        question: str,
        user_id: str,
        context_size: int = 5,
    ) -> Dict[str, Any]:
        """回答问题

        Args:
            campaign_id: 战役 ID
            question: 用户问题
            user_id: 用户 ID
            context_size: 检索上下文数量

        Returns:
            问答响应
        """
        # 1. 检索相关上下文
        context = await self._retrieve_context(
            campaign_id=campaign_id,
            question=question,
            user_id=user_id,
            context_size=context_size,
        )

        # 2. 构建提示词
        prompt = self._build_qa_prompt(
            question=question,
            context=context,
            campaign_id=campaign_id,
        )

        # 3. 调用 LLM
        response = await self.llm.complete(
            prompt=prompt,
            max_tokens=800,
            temperature=0.3,  # 较低温度，更准确的回答
        )

        # 4. 解析响应
        answer = self._parse_answer(response, context)

        return answer

    async def _retrieve_context(
        self,
        campaign_id: str,
        question: str,
        user_id: str,
        context_size: int,
    ) -> List[Dict[str, Any]]:
        """检索相关上下文"""

        # 1. 向量检索（语义相似）
        vector_results = await self.vector_store.search(
            campaign_id=campaign_id,
            query=question,
            limit=context_size,
        )

        # 2. 全文检索（关键词匹配）
        search_results = await self.search.search_all(
            query=question,
            campaign_id=campaign_id,
            size=context_size,
        )

        # 3. 合并去重
        context = []
        seen_ids = set()

        # 优先使用向量检索结果
        for result in vector_results:
            if result['id'] not in seen_ids:
                context.append(result)
                seen_ids.add(result['id'])

        # 补充全文检索结果
        for event in search_results.get('events', []):
            if event['id'] not in seen_ids:
                context.append({
                    'id': event['id'],
                    'type': 'event',
                    'content': event.get('description', ''),
                    'timestamp': event.get('timestamp'),
                    'source': 'search',
                })
                seen_ids.add(event['id'])

        # 4. 可见性过滤
        context = [c for c in context if self._check_visibility(c, user_id)]

        return context[:context_size]

    def _build_qa_prompt(
        self,
        question: str,
        context: List[Dict[str, Any]],
        campaign_id: str,
    ) -> str:
        """构建问答提示词"""

        # 格式化上下文
        context_text = "\n\n".join([
            f"[{c.get('timestamp', 'Unknown')}]\n"
            f"{c.get('content', '')}\n"
            f"(来源: {c.get('source', 'unknown')})"
            for c in context
        ])

        prompt = f"""你是一个《克苏鲁的呼唤》(Call of Cthulhu) TRPG 游戏助手。玩家正在询问关于游戏历史的问题。

战役 ID: {campaign_id}

玩家问题:
{question}

相关游戏记录:
{context_text}

请根据以上游戏记录回答玩家的问题。要求:
1. 回答要准确，基于提供的游戏记录
2. 如果记录中没有相关信息，明确说明"根据现有记录没有找到相关信息"
3. 引用具体的记录时，说明时间戳
4. 保持友好和帮助的语气
5. 如果涉及线索等敏感信息，注意保密

请以 JSON 格式回复:
{{
  "answer": "你的回答",
  "sources": ["引用的记录ID列表"],
  "confidence": "high/medium/low",
  "need_clarification": "是否需要澄清问题"
}}
"""

        return prompt

    def _parse_answer(
        self,
        response: str,
        context: List[Dict[str, Any]],
    ) -> Dict[str, Any]:
        """解析 LLM 响应"""
        try:
            data = json.loads(response)
            return {
                "answer": data.get("answer", response),
                "sources": data.get("sources", []),
                "confidence": data.get("confidence", "medium"),
                "need_clarification": data.get("need_clarification", False),
                "context": context,
            }
        except json.JSONDecodeError:
            # 回退到简单响应
            return {
                "answer": response,
                "sources": [c.get("id") for c in context[:3]],
                "confidence": "low",
                "need_clarification": False,
                "context": context,
            }

    def _check_visibility(self, context_item: Dict, user_id: str) -> bool:
        """检查上下文可见性"""
        # 简化处理，实际应该检查用户权限
        visibility = context_item.get("visibility", "public")
        if visibility == "public":
            return True
        if visibility == "kp_only":
            # 需要 KP 权限
            return True
        return False

    async def get_suggested_questions(
        self,
        campaign_id: str,
    ) -> List[str]:
        """获取建议问题"""
        suggestions = [
            "我们上次发现了什么线索？",
            "当前所有角色的状态如何？",
            "发生了哪些重要战斗？",
            "有哪些未完成的承诺？",
            "某某角色的 SAN 值变化了多少？",
        ]

        # 可以根据游戏历史动态生成建议
        return suggestions
```

### 向量存储服务

```python
# app/core/vector_store.py
from typing import List, Dict, Any
import numpy as np

from app.services.embeddings import EmbeddingService

class VectorStore:
    """向量存储服务"""

    def __init__(self, db):
        self.db = db
        self.embedding_service = EmbeddingService()

    async def search(
        self,
        campaign_id: str,
        query: str,
        limit: int = 5,
    ) -> List[Dict[str, Any]]:
        """向量搜索

        Args:
            campaign_id: 战役 ID
            query: 查询文本
            limit: 返回结果数量

        Returns:
            相似度最高的记录
        """
        # 1. 生成查询向量
        query_embedding = await self.embedding_service.embed(query)

        # 2. 从数据库中检索
        # 使用 pgvector 或其他向量数据库
        results = await self._vector_search(
            campaign_id=campaign_id,
            query_vector=query_embedding,
            limit=limit,
        )

        return results

    async def _vector_search(
        self,
        campaign_id: str,
        query_vector: np.ndarray,
        limit: int,
    ) -> List[Dict[str, Any]]:
        """执行向量搜索"""
        # 示例：使用 pgvector
        from app.db.models.event_vector import EventVector

        # 计算余弦相似度
        results = self.db.query(EventVector).filter(
            EventVector.campaign_id == campaign_id
        ).order_by(
            EventVector.embedding.cosine_distance(query_vector)
        ).limit(limit).all()

        return [
            {
                "id": r.event_id,
                "type": r.event_type,
                "content": r.content,
                "timestamp": r.timestamp.isoformat(),
                "score": r.similarity,
                "source": "vector",
            }
            for r in results
        ]

    async def index_event(self, event: Dict[str, Any]):
        """索引事件到向量存储"""
        # 生成嵌入
        text = f"{event.get('description', '')} {event.get('data', {})}"
        embedding = await self.embedding_service.embed(text)

        # 存储到数据库
        from app.db.models.event_vector import EventVector

        vector = EventVector(
            event_id=event["id"],
            campaign_id=event["campaign_id"],
            event_type=event["type"],
            content=event.get("description", ""),
            embedding=embedding,
            timestamp=event.get("timestamp"),
        )

        self.db.add(vector)
        self.db.commit()
```

### 嵌入服务

```python
# app/services/embeddings.py
import httpx
import os
import numpy as np

class EmbeddingService:
    """文本嵌入服务"""

    def __init__(self):
        self.api_key = os.getenv("OPENAI_API_KEY")
        self.model = os.getenv("EMBEDDING_MODEL", "text-embedding-ada-002")
        self.base_url = "https://api.openai.com/v1"

    async def embed(self, text: str) -> np.ndarray:
        """生成文本嵌入

        Args:
            text: 输入文本

        Returns:
            嵌入向量
        """
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{self.base_url}/embeddings",
                headers={
                    "Authorization": f"Bearer {self.api_key}",
                    "Content-Type": "application/json",
                },
                json={
                    "model": self.model,
                    "input": text,
                },
                timeout=30.0,
            )
            response.raise_for_status()
            data = response.json()
            return np.array(data["data"][0]["embedding"])
```

### 问答 API

```python
# app/api/qa.py
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session
from pydantic import BaseModel
from typing import List, Optional

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.db.models.user import User
from app.services.qa_service import QAService

router = APIRouter(prefix="/qa", tags=["qa"])

class AskRequest(BaseModel):
    campaign_id: str
    question: str
    context_size: int = 5

class AskResponse(BaseModel):
    answer: str
    sources: List[str]
    confidence: str
    need_clarification: bool
    context: List[dict]

@router.post("/ask", response_model=AskResponse)
async def ask_question(
    request: AskRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """提问"""
    service = QAService(db)

    answer = await service.ask_question(
        campaign_id=request.campaign_id,
        question=request.question,
        user_id=current_user.id,
        context_size=request.context_size,
    )

    return AskResponse(**answer)

@router.get("/suggestions")
async def get_suggestions(
    campaign_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """获取建议问题"""
    service = QAService(db)

    suggestions = await service.get_suggested_questions(
        campaign_id=campaign_id,
    )

    return {"suggestions": suggestions}
```

---

## 前端代码示例

### AI 问答组件

```typescript
// frontend/src/components/qa/AIQAChat.tsx
import React, { useState, useRef, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Send, Lightbulb, Loader2 } from 'lucide-react';
import { Avatar } from '@/components/ui/avatar';
import { Badge } from '@/components/ui/badge';

interface Message {
  role: 'user' | 'assistant';
  content: string;
  sources?: string[];
  confidence?: string;
  timestamp: Date;
}

export function AIQAChat({ campaignId }: { campaignId: string }) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const [suggestions, setSuggestions] = useState<string[]>([]);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    loadSuggestions();
  }, [campaignId]);

  useEffect(() => {
    // 自动滚动到底部
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [messages]);

  const loadSuggestions = async () => {
    const response = await fetch(`/api/qa/suggestions?campaign_id=${campaignId}`);
    const data = await response.json();
    setSuggestions(data.suggestions);
  };

  const handleSend = async () => {
    if (!input.trim() || loading) return;

    const userMessage: Message = {
      role: 'user',
      content: input,
      timestamp: new Date(),
    };

    setMessages(prev => [...prev, userMessage]);
    setInput('');
    setLoading(true);

    try {
      const response = await fetch('/api/qa/ask', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          campaign_id: campaignId,
          question: input,
          context_size: 5,
        }),
      });

      const data = await response.json();

      const assistantMessage: Message = {
        role: 'assistant',
        content: data.answer,
        sources: data.sources,
        confidence: data.confidence,
        timestamp: new Date(),
      };

      setMessages(prev => [...prev, assistantMessage]);
    } catch (error) {
      console.error('提问失败:', error);
      const errorMessage: Message = {
        role: 'assistant',
        content: '抱歉，我遇到了一些问题。请稍后再试。',
        timestamp: new Date(),
      };
      setMessages(prev => [...prev, errorMessage]);
    } finally {
      setLoading(false);
    }
  };

  const handleSuggestionClick = (suggestion: string) => {
    setInput(suggestion);
  };

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Lightbulb className="h-5 w-5" />
            AI 问答助手
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {/* 消息列表 */}
            <ScrollArea className="h-96 rounded-md border p-4" ref={scrollRef}>
              <div className="space-y-4">
                {messages.length === 0 && (
                  <div className="text-center text-muted-foreground py-8">
                    <Lightbulb className="h-12 w-12 mx-auto mb-4 opacity-50" />
                    <p>问我关于游戏的任何问题</p>
                    <p className="text-sm mt-2">
                      例如："我们上次发现了什么线索？"
                    </p>
                  </div>
                )}

                {messages.map((message, index) => (
                  <div
                    key={index}
                    className={`flex gap-3 ${
                      message.role === 'user' ? 'justify-end' : 'justify-start'
                    }`}
                  >
                    {message.role === 'assistant' && (
                      <Avatar className="h-8 w-8">
                        <div className="w-full h-full bg-primary flex items-center justify-center text-white">
                          AI
                        </div>
                      </Avatar>
                    )}

                    <div
                      className={`rounded-lg px-4 py-2 max-w-[80%] ${
                        message.role === 'user'
                          ? 'bg-primary text-primary-foreground'
                          : 'bg-muted'
                      }`}
                    >
                      <p className="text-sm">{message.content}</p>

                      {message.sources && message.sources.length > 0 && (
                        <div className="mt-2 pt-2 border-t border-border/50">
                          <p className="text-xs text-muted-foreground mb-1">
                            引用来源:
                          </p>
                          {message.sources.map((source, i) => (
                            <Badge key={i} variant="outline" className="text-xs mr-1">
                              {source.slice(0, 8)}...
                            </Badge>
                          ))}
                        </div>
                      )}

                      {message.confidence && (
                        <div className="mt-1">
                          <Badge
                            variant={
                              message.confidence === 'high'
                                ? 'default'
                                : message.confidence === 'medium'
                                ? 'secondary'
                                : 'outline'
                            }
                            className="text-xs"
                          >
                            置信度: {message.confidence}
                          </Badge>
                        </div>
                      )}
                    </div>

                    {message.role === 'user' && (
                      <Avatar className="h-8 w-8">
                        <div className="w-full h-full bg-secondary flex items-center justify-center">
                          我
                        </div>
                      </Avatar>
                    )}
                  </div>
                ))}

                {loading && (
                  <div className="flex gap-3">
                    <Avatar className="h-8 w-8">
                      <div className="w-full h-full bg-primary flex items-center justify-center text-white">
                        AI
                      </div>
                    </Avatar>
                    <div className="bg-muted rounded-lg px-4 py-2">
                      <Loader2 className="h-4 w-4 animate-spin" />
                    </div>
                  </div>
                )}
              </div>
            </ScrollArea>

            {/* 建议问题 */}
            {messages.length === 0 && suggestions.length > 0 && (
              <div className="space-y-2">
                <p className="text-sm text-muted-foreground">你可以这样问:</p>
                <div className="flex flex-wrap gap-2">
                  {suggestions.map((suggestion, index) => (
                    <Button
                      key={index}
                      variant="outline"
                      size="sm"
                      onClick={() => handleSuggestionClick(suggestion)}
                    >
                      {suggestion}
                    </Button>
                  ))}
                </div>
              </div>
            )}

            {/* 输入框 */}
            <div className="flex gap-2">
              <Input
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && !e.shiftKey) {
                    e.preventDefault();
                    handleSend();
                  }
                }}
                placeholder="输入你的问题..."
                disabled={loading}
              />
              <Button onClick={handleSend} disabled={loading || !input.trim()}>
                <Send className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/qa_service.py` | 创建 | 问答服务 |
| `app/core/vector_store.py` | 创建 | 向量存储服务 |
| `app/services/embeddings.py` | 创建 | 嵌入服务 |
| `app/db/models/event_vector.py` | 创建 | 事件向量模型 |
| `app/api/qa.py` | 创建 | 问答 API |
| `frontend/src/components/qa/AIQAChat.tsx` | 创建 | AI 问答聊天组件 |
| `tests/test_qa.py` | 创建 | 问答测试 |

---

## 验收标准

- [ ] 能准确回答关于游戏历史的问题
- [ ] 引用来源正确显示
- [ ] 置信度评估合理
- [ ] 建议问题相关且有引导性
- [ ] 向量检索和全文检索混合有效
- [ ] 可见性控制正确（KP/玩家）
- [ ] 回答基于事实，不编造内容
- [ ] 流式输出流畅（如果实现）

---

## 参考文档

- M3-001: AI 总结服务
- M3-027: 全文检索功能
- OpenAI Embeddings API 文档
- RAG (Retrieval Augmented Generation) 最佳实践

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
