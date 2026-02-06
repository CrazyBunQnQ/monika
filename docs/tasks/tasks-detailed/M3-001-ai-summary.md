# M3-001: 实现 AI 总结服务

**任务ID**: M3-001
**标题**: 实现 AI 总结服务
**类型**: backend (后端开发)
**预估工时**: 3h
**依赖**: M1-080

---

## 任务描述

实现基于 LLM 的游戏内容总结服务，自动生成游戏会话摘要。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M3-001-01 | 设计总结 API 结构 | API 设计 | 20min |
| M3-001-02 | 实现 LLM 接口 | LLM 集成 | 35min |
| M3-001-03 | 实现消息提取 | 日志处理 | 25min |
| M3-001-04 | 实现总结生成 | AI 处理 | 40min |
| M3-001-05 | 实现格式化输出 | 结果处理 | 20min |
| M3-001-06 | 实现缓存机制 | 性能优化 | 25min |
| M3-001-07 | 编写总结测试 | 测试覆盖 | 25min |

---

## 总结服务

```python
# app/services/summary.py
from typing import List, Dict, Any, Optional
from datetime import datetime, timedelta
import json

from app.services.llm import LLMService
from app.db.models.gamestate import GameState
from app.core.logger import EventLogger

class SummaryService:
    """AI 总结服务"""

    def __init__(self, db: Session):
        self.db = db
        self.llm = LLMService()
        self.logger = EventLogger()

    async def generate_summary(
        self,
        campaign_id: str,
        start_time: Optional[datetime] = None,
        end_time: Optional[datetime] = None,
    ) -> Dict[str, Any]:
        """生成游戏总结

        Args:
            campaign_id: 战役 ID
            start_time: 开始时间（默认为今天开始）
            end_time: 结束时间（默认为现在）

        Returns:
            包含总结的字典
        """
        # 默认时间范围：今天
        if start_time is None:
            start_time = datetime.now().replace(hour=0, minute=0, second=0)
        if end_time is None:
            end_time = datetime.now()

        # 提取事件
        events = await self._extract_events(
            campaign_id,
            start_time,
            end_time
        )

        if not events:
            return {
                "summary": "本时段没有游戏活动",
                "key_events": [],
                "characters": [],
                "locations": [],
            }

        # 生成总结
        summary = await self._generate_summary(events)

        return summary

    async def _extract_events(
        self,
        campaign_id: str,
        start_time: datetime,
        end_time: datetime,
    ) -> List[Dict[str, Any]]:
        """提取事件日志"""
        events = await self.logger.get_events(
            campaign_id=campaign_id,
            start_time=start_time,
            end_time=end_time,
        )

        # 格式化事件
        formatted_events = []
        for event in events:
            formatted_events.append({
                "timestamp": event.timestamp.isoformat(),
                "type": event.type,
                "description": event.description,
                "data": event.data,
            })

        return formatted_events

    async def _generate_summary(
        self,
        events: List[Dict[str, Any]],
    ) -> Dict[str, Any]:
        """使用 LLM 生成总结"""

        # 构建提示词
        prompt = self._build_prompt(events)

        # 调用 LLM
        response = await self.llm.complete(
            prompt=prompt,
            max_tokens=1000,
            temperature=0.7,
        )

        # 解析响应
        summary = self._parse_response(response, events)

        return summary

    def _build_prompt(self, events: List[Dict[str, Any]]) -> str:
        """构建 LLM 提示词"""
        events_text = "\n".join([
            f"[{e['timestamp']}] {e['type']}: {e['description']}"
            for e in events
        ])

        return f"""你是一个 CoC（克苏鲁的呼唤）TRPG 游戏助手。请根据以下游戏事件，生成一份简洁的总结。

事件列表：
{events_text}

请按以下 JSON 格式返回总结：
{{
    "summary": "整体总结，2-3句话",
    "key_events": ["关键事件1", "关键事件2", ...],
    "characters": ["参与的角色1", "参与的角色2", ...],
    "locations": ["涉及的地点1", "涉及的地点2", ...],
    "clues_found": ["发现的线索1", "发现的线索2", ...],
    "next_steps": ["建议的下一步行动1", "建议的下一步行动2", ...]
}}

注意：
1. 总结要简洁明了
2. 关键事件要突出重要情节
3. 线索要保密（避免剧透给未参与的玩家）
4. 下一步建议要帮助 KP 推进剧情
"""

    def _parse_response(
        self,
        response: str,
        events: List[Dict[str, Any]],
    ) -> Dict[str, Any]:
        """解析 LLM 响应"""
        try:
            # 尝试解析 JSON
            data = json.loads(response)
            return data
        except json.JSONDecodeError:
            # 解析失败，返回基础总结
            return {
                "summary": response[:500],
                "key_events": [e["description"] for e in events[:5]],
                "characters": list(set([
                    e.get("data", {}).get("character")
                    for e in events
                    if e.get("data", {}).get("character")
                ])),
                "locations": [],
                "clues_found": [],
                "next_steps": [],
            }
```

---

## LLM 服务接口

```python
# app/services/llm.py
from abc import ABC, abstractmethod
import httpx
import os

class LLMProvider(ABC):
    """LLM 提供者接口"""

    @abstractmethod
    async def complete(
        self,
        prompt: str,
        max_tokens: int = 1000,
        temperature: float = 0.7,
    ) -> str:
        """完成文本"""
        pass

class OpenAIProvider(LLMProvider):
    """OpenAI 提供者"""

    def __init__(self, api_key: str):
        self.api_key = api_key
        self.base_url = "https://api.openai.com/v1"
        self.model = os.getenv("OPENAI_MODEL", "gpt-4")

    async def complete(
        self,
        prompt: str,
        max_tokens: int = 1000,
        temperature: float = 0.7,
    ) -> str:
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{self.base_url}/chat/completions",
                headers={
                    "Authorization": f"Bearer {self.api_key}",
                    "Content-Type": "application/json",
                },
                json={
                    "model": self.model,
                    "messages": [
                        {"role": "user", "content": prompt}
                    ],
                    "max_tokens": max_tokens,
                    "temperature": temperature,
                },
                timeout=30.0,
            )
            response.raise_for_status()
            data = response.json()
            return data["choices"][0]["message"]["content"]

class LLMService:
    """LLM 服务"""

    def __init__(self):
        provider_type = os.getenv("LLM_PROVIDER", "openai")

        if provider_type == "openai":
            api_key = os.getenv("OPENAI_API_KEY")
            if not api_key:
                raise ValueError("OPENAI_API_KEY not set")
            self.provider = OpenAIProvider(api_key)
        else:
            raise ValueError(f"Unknown LLM provider: {provider_type}")

    async def complete(
        self,
        prompt: str,
        max_tokens: int = 1000,
        temperature: float = 0.7,
    ) -> str:
        """完成文本"""
        return await self.provider.complete(
            prompt,
            max_tokens,
            temperature,
        )
```

---

## 总结 API

```python
# app/api/summary.py
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session
from pydantic import BaseModel

from app.db.database import get_db
from app.services.summary import SummaryService
from app.api.deps.auth import get_current_user
from app.db.models.user import User

router = APIRouter(prefix="/summary", tags=["summary"])

class SummaryRequest(BaseModel):
    campaign_id: str
    start_time: Optional[str] = None
    end_time: Optional[str] = None

class SummaryResponse(BaseModel):
    summary: str
    key_events: list
    characters: list
    locations: list
    clues_found: list
    next_steps: list

@router.post("", response_model=SummaryResponse)
async def generate_summary(
    request: SummaryRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """生成游戏总结"""
    service = SummaryService(db)

    summary = await service.generate_summary(
        campaign_id=request.campaign_id,
        start_time=request.start_time,
        end_time=request.end_time,
    )

    return SummaryResponse(**summary)
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/summary.py` | 创建 | 总结服务 |
| `app/services/llm.py` | 创建 | LLM 服务 |
| `app/api/summary.py` | 创建 | 总结 API |
| `tests/test_summary.py` | 创建 | 总结测试 |

---

## 验收标准

- [ ] 总结生成正确
- [ ] LLM 接口正常
- [ ] 事件提取完整
- [ ] 格式化输出清晰
- [ ] 缓存机制有效
- [ ] API 响应及时

---

## 参考文档

- M1-080: 事件日志系统
- OpenAI API 文档

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
