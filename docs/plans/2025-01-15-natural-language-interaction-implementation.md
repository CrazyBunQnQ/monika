# 自然语言交互系统实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**目标:** 实现玩家与 AI 守密人之间的自然语言对话系统，支持流式响应和状态同步

**架构:** 分层架构设计，WebSocket 通信 + LLM 抽象层 + 状态同步服务

**技术栈:** FastAPI (WebSocket)、OpenAI/Claude API、React (WebSocket Client)、Pydantic

---

## 前置检查清单

**在开始之前，确认以下条件已满足：**

- [ ] PostgreSQL 数据库已启动并可连接
- [ ] 后端依赖已安装 (`cd backend && uv sync`)
- [ ] 前端依赖已安装 (`cd frontend && npm install`)
- [ ] 已有测试用户和角色卡数据
- [ ] 环境变量 `.env` 文件已配置

---

## 任务 1: LLM 抽象层 - 基础接口和 OpenAI Provider

**目标:** 创建 LLM Provider 抽象接口和 OpenAI 实现

**文件:**
- 创建: `backend/src/services/llm/__init__.py`
- 创建: `backend/src/services/llm/base.py`
- 创建: `backend/src/services/llm/openai.py`
- 创建: `backend/src/tests/test_llm_providers.py`

### 步骤 1: 创建包结构和基础接口

**创建 `backend/src/services/llm/__init__.py`:**

```python
from .base import LLMProvider
from .openai import OpenAIProvider

__all__ = ["LLMProvider", "OpenAIProvider"]
```

**创建 `backend/src/services/llm/base.py`:**

```python
from abc import ABC, abstractmethod
from typing import AsyncIterator

class LLMProvider(ABC):
    """LLM Provider 抽象基类"""

    @abstractmethod
    async def stream_chat(
        self,
        messages: list[dict],
        system_prompt: str
    ) -> AsyncIterator[str]:
        """
        流式聊天响应
        返回 JSON 字符串片段的异步迭代器
        """
        pass

    @abstractmethod
    async def get_context_limit(self) -> int:
        """返回模型的最大上下文长度"""
        pass

    @abstractmethod
    async def health_check(self) -> bool:
        """健康检查"""
        pass
```

**创建 `backend/src/services/llm/openai.py`:**

```python
import os
from typing import AsyncIterator
from openai import AsyncOpenAI
from .base import LLMProvider

class OpenAIProvider(LLMProvider):
    def __init__(self):
        api_key = os.getenv("OPENAI_API_KEY")
        if not api_key:
            raise ValueError("OPENAI_API_KEY environment variable is required")
        self.client = AsyncOpenAI(api_key=api_key)
        self.model = os.getenv("OPENAI_MODEL", "gpt-4")

    async def stream_chat(
        self,
        messages: list[dict],
        system_prompt: str
    ) -> AsyncIterator[str]:
        # 构建完整的消息列表
        full_messages = [{"role": "system", "content": system_prompt}]
        full_messages.extend(messages)

        stream = await self.client.chat.completions.create(
            model=self.model,
            messages=full_messages,
            stream=True,
            temperature=0.8,
            max_tokens=2000
        )

        async for chunk in stream:
            if chunk.choices and chunk.choices[0].delta.content:
                yield chunk.choices[0].delta.content

    async def get_context_limit(self) -> int:
        # GPT-4 的上下文限制
        return 8192

    async def health_check(self) -> bool:
        try:
            response = await self.client.chat.completions.create(
                model=self.model,
                messages=[{"role": "user", "content": "test"}],
                max_tokens=1
            )
            return True
        except Exception:
            return False
```

### 步骤 2: 添加 OpenAI 依赖

**修改 `backend/pyproject.toml`:**

在 dependencies 中添加:
```toml
openai = "^1.0.0"
```

运行:
```bash
cd d:/git/monika-nli/backend
uv sync
```

### 步骤 3: 编写测试

**创建 `backend/src/tests/test_llm_providers.py`:**

```python
import pytest
from backend.src.services.llm.openai import OpenAIProvider

@pytest.fixture
async def mock_openai_provider():
    # 使用 Mock 避免真实 API 调用
    from unittest.mock import AsyncMock, patch
    with patch("backend.src.services.llm.openai.AsyncOpenAI"):
        provider = OpenAIProvider()
        yield provider

@pytest.mark.asyncio
async def test_openai_provider_context_limit(mock_openai_provider):
    limit = await mock_openai_provider.get_context_limit()
    assert limit == 8192

@pytest.mark.asyncio
async def test_openai_provider_stream_chat(mock_openai_provider):
    # Mock stream_chat 方法
    async def mock_stream(messages, system_prompt):
        yield '{"narrative": "test"}'

    mock_openai_provider.stream_chat = mock_stream

    chunks = []
    async for chunk in mock_openai_provider.stream_chat([], ""):
        chunks.append(chunk)

    assert len(chunks) == 1
    assert chunks[0] == '{"narrative": "test"}'
```

### 步骤 4: 运行测试

```bash
cd d:/git/monika-nli/backend
uv run pytest src/tests/test_llm_providers.py -v
```

预期输出: `PASS`

### 步骤 5: 提交

```bash
cd d:/git/monika-nli
git add backend/src/services/llm/ backend/src/tests/test_llm_providers.py backend/pyproject.toml
git commit -m "feat(NLI-01): add LLM abstraction layer and OpenAI provider"
```

---

## 任务 2: LLM 响应 Schema 和解析器

**目标:** 定义 LLM 响应的 Pydantic Schema 并实现解析器

**文件:**
- 创建: `backend/src/schemas/llm_response.py`
- 创建: `backend/src/services/response_parser.py`
- 创建: `backend/src/tests/test_response_parser.py`

### 步骤 1: 创建响应 Schema

**创建 `backend/src/schemas/llm_response.py`:**

```python
from pydantic import BaseModel, Field
from typing import Optional, Dict, List

class StateChanges(BaseModel):
    """AI 可以修改的状态字段（白名单）"""
    current_scene: Optional[str] = Field(None, description="当前场景名称")
    world_state: Optional[Dict] = Field(None, description="世界状态更新")

class LLMResponse(BaseModel):
    """LLM 响应格式"""
    narrative: str = Field(..., description="叙述文本，显示给玩家")
    tone: str = Field("calm", description="语气: mystery, horror, action, calm")
    urgency: str = Field("low", description="紧迫度: low, medium, high")
    state_changes: Optional[StateChanges] = Field(None, description="状态变化")
    suggestions: Optional[List[str]] = Field(None, description="给玩家的操作建议")
    audio_cue: Optional[str] = Field(None, description="音效提示")
    requires_roll: bool = Field(False, description="是否建议玩家进行检定")
```

### 步骤 2: 创建响应解析器

**创建 `backend/src/services/response_parser.py`:**

```python
import json
import logging
from typing import Optional, AsyncIterator
from backend.src.schemas.llm_response import LLMResponse

logger = logging.getLogger(__name__)

class ResponseParser:
    """解析 LLM 流式响应"""

    def __init__(self):
        self.buffer = ""

    async def parse_stream(
        self,
        stream: AsyncIterator[str]
    ) -> AsyncIterator[LLMResponse]:
        """
        解析流式响应
        当检测到完整的 JSON 时 yield 解析后的 LLMResponse
        """
        self.buffer = ""
        async for chunk in stream:
            self.buffer += chunk

            # 尝试提取 JSON
            json_str = self._extract_json(self.buffer)
            if json_str:
                try:
                    response = LLMResponse.model_validate_json(json_str)
                    self.buffer = ""  # 清空缓冲区
                    yield response
                except Exception as e:
                    logger.warning(f"Failed to parse LLM response: {e}")
                    # 尝试降级处理
                    yield self._fallback_response(json_str)

    def _extract_json(self, text: str) -> Optional[str]:
        """从文本中提取 JSON 对象"""
        text = text.strip()
        if not text:
            return None

        # 查找第一个 { 和最后一个 }
        start = text.find("{")
        end = text.rfind("}")

        if start != -1 and end != -1 and end > start:
            return text[start:end + 1]
        return None

    def _fallback_response(self, text: str) -> LLMResponse:
        """JSON 解析失败时的降级处理"""
        # 尝试提取纯文本作为 narrative
        json_str = self._extract_json(text)
        if json_str:
            try:
                data = json.loads(json_str)
                narrative = data.get("narrative", text)
            except:
                narrative = text
        else:
            narrative = text

        return LLMResponse(
            narrative=narrative[:1000],  # 限制长度
            tone="calm"
        )
```

### 步骤 3: 编写测试

**创建 `backend/src/tests/test_response_parser.py`:**

```python
import pytest
from backend.src.services.response_parser import ResponseParser
from backend.src.schemas.llm_response import LLMResponse

@pytest.mark.asyncio
async def test_parse_valid_json_response():
    parser = ResponseParser()

    async def mock_stream():
        yield '{"narrative": "You see a door", "tone": "mystery"}'

    responses = []
    async for response in parser.parse_stream(mock_stream()):
        responses.append(response)

    assert len(responses) == 1
    assert responses[0].narrative == "You see a door"
    assert responses[0].tone == "mystery"

@pytest.mark.asyncio
async def test_parse_streaming_chunks():
    parser = ResponseParser()

    async def mock_stream():
        yield '{"narr":'
        yield 'ative": "test"'
        yield ', "tone": "horror"}'

    responses = []
    async for response in parser.parse_stream(mock_stream()):
        responses.append(response)

    assert len(responses) == 1
    assert responses[0].narrative == "test"

@pytest.mark.asyncio
async def test_fallback_on_invalid_json():
    parser = ResponseParser()

    async def mock_stream():
        yield 'This is plain text, not JSON'

    responses = []
    async for response in parser.parse_stream(mock_stream()):
        responses.append(response)

    assert len(responses) == 1
    assert "plain text" in responses[0].narrative

@pytest.mark.asyncio
async def test_extract_json():
    parser = ResponseParser()

    # 测试 JSON 提取
    text = 'Some text before {"narrative": "test"} some text after'
    result = parser._extract_json(text)
    assert result == '{"narrative": "test"}'

    # 测试无 JSON
    assert parser._extract_json("no json here") is None
```

### 步骤 4: 运行测试

```bash
cd d:/git/monika-nli/backend
uv run pytest src/tests/test_response_parser.py -v
```

预期输出: 全部 PASS

### 步骤 5: 提交

```bash
cd d:/git/monika-nli
git add backend/src/schemas/llm_response.py backend/src/services/response_parser.py backend/src/tests/test_response_parser.py
git commit -m "feat(NLI-02): add LLM response schema and parser"
```

---

## 任务 3: Prompt 模板引擎

**目标:** 实现构建 CoC 7e 守密人 Prompt 的模板引擎

**文件:**
- 创建: `backend/src/services/prompt.py`
- 创建: `backend/src/tests/test_prompt.py`

### 步骤 1: 创建 Prompt 服务

**创建 `backend/src/services/prompt.py`:**

```python
from typing import Optional
from backend.src.models.character import Character
from backend.src.models.session import GameSession

class PromptBuilder:
    """构建 LLM Prompt"""

    SYSTEM_TEMPLATE = """你是克苏鲁的呼唤 7 版（Call of Cthulhu 7th Edition）的守密人（Keeper）。

你的职责：
1. 描述场景、NPC 反应和故事发展
2. 保持神秘和恐怖的氛围
3. 尊重玩家的选择，推动故事发展
4. 可以修改场景信息和添加线索，但不能直接修改玩家角色的 HP/SAN 等核心属性

请以 JSON 格式响应，包含以下字段：
- narrative: 叙述文本（第二人称"你"）
- tone: 语气 (mystery|horror|action|calm)
- urgency: 紧迫度 (low|medium|high)
- state_changes: 状态变化（可选）- 只修改 current_scene 和 world_state
- suggestions: 给玩家的建议（可选）
"""

    async def build_system_prompt(self) -> str:
        """构建系统提示词"""
        return self.SYSTEM_TEMPLATE

    async def build_context_messages(
        self,
        character: Character,
        session: GameSession,
        recent_events: list[dict],
        user_message: str
    ) -> list[dict]:
        """
        构建上下文消息列表
        包含: 角色信息、当前场景、最近事件、用户消息
        """
        messages = []

        # 添加角色和场景上下文
        context = f"""当前游戏状态:
- 角色: {character.name}
- 当前场景: {session.current_scene}
- HP: {character.hp}/{character.max_hp}
- SAN: {character.san}/{character.max_san}
"""

        if session.world_state.get("leads"):
            context += f"- 已发现线索: {', '.join(session.world_state['leads'])}\n"

        messages.append({"role": "user", "content": context})

        # 添加最近事件（最近 5 条）
        if recent_events:
            events_text = "最近发生的事情:\n"
            for event in recent_events[-5:]:
                events_text += f"- {event.get('description', 'event')}\n"
            messages.append({"role": "user", "content": events_text})

        # 添加用户当前消息
        messages.append({"role": "user", "content": user_message})

        return messages
```

### 步骤 2: 编写测试

**创建 `backend/src/tests/test_prompt.py`:**

```python
import pytest
from backend.src.services.prompt import PromptBuilder

@pytest.mark.asyncio
async def test_build_system_prompt():
    builder = PromptBuilder()
    prompt = await builder.build_system_prompt()

    assert "克苏鲁的呼唤" in prompt
    assert "守密人" in prompt
    assert "JSON 格式" in prompt

@pytest.mark.asyncio
async def test_build_context_messages():
    from backend.src.models.character import Character
    from backend.src.models.session import GameSession

    builder = PromptBuilder()

    # 创建测试数据
    character = Character(
        id=1,
        name="测试角色",
        hp=10,
        max_hp=10,
        san=50,
        max_san=99
    )
    session = GameSession(
        id=1,
        current_scene="旧书房",
        world_state={"leads": ["神秘笔记"]}
    )

    messages = await builder.build_context_messages(
        character=character,
        session=session,
        recent_events=[{"description": "你进入了一个房间"}],
        user_message="我想查看书桌"
    )

    assert len(messages) == 3
    assert any("测试角色" in msg.get("content", "") for msg in messages)
    assert any("旧书房" in msg.get("content", "") for msg in messages)
    assert messages[-1]["content"] == "我想查看书桌"
```

### 步骤 3: 运行测试

```bash
cd d:/git/monika-nli/backend
uv run pytest src/tests/test_prompt.py -v
```

### 步骤 4: 提交

```bash
cd d:/git/monika-nli
git add backend/src/services/prompt.py backend/src/tests/test_prompt.py
git commit -m "feat(NLI-03): add prompt template engine"
```

---

## 任务 4: 状态同步服务

**目标:** 实现将 AI 的状态变化安全地同步到数据库

**文件:**
- 创建: `backend/src/services/state_sync.py`
- 创建: `backend/src/tests/test_state_sync.py`

### 步骤 1: 创建状态同步服务

**创建 `backend/src/services/state_sync.py`:**

```python
import logging
from typing import Optional
from sqlalchemy.ext.asyncio import AsyncSession
from backend.src.models.session import GameSession
from backend.src.models.event import Event
from backend.src.schemas.llm_response import StateChanges

logger = logging.getLogger(__name__)

# 状态修改白名单
ALLOWED_STATE_CHANGES = {
    "current_scene",
    "world_state.leads",
    "world_state.location",
    "world_state.npcs"
}

class StateSyncService:
    """状态同步服务 - 安全地应用 AI 的状态变化"""

    def __init__(self, db: AsyncSession):
        self.db = db

    async def apply_state_changes(
        self,
        session: GameSession,
        changes: StateChanges,
        source_description: str = "AI Keeper"
    ) -> GameSession:
        """
        应用状态变化到游戏会话

        Args:
            session: 当前游戏会话
            changes: LLM 响应中的状态变化
            source_description: 变化来源描述（用于事件记录）

        Returns:
            更新后的游戏会话
        """
        if not changes:
            return session

        # 记录原始状态用于事件日志
        original_state = {
            "current_scene": session.current_scene,
            "world_state": session.world_state.copy() if session.world_state else {}
        }

        # 应用场景变化
        if changes.current_scene is not None:
            session.current_scene = changes.current_scene

        # 应用世界状态变化
        if changes.world_state:
            if not session.world_state:
                session.world_state = {}

            for key, value in changes.world_state.items():
                # 检查是否在白名单中
                field_path = f"world_state.{key}"
                if field_path in ALLOWED_STATE_CHANGES:
                    if key == "leads" and isinstance(value, str):
                        # 处理线索添加: +新线索
                        if value.startswith("+"):
                            new_lead = value[1:].strip()
                            if "leads" not in session.world_state:
                                session.world_state["leads"] = []
                            if new_lead not in session.world_state["leads"]:
                                session.world_state["leads"].append(new_lead)
                        elif value.startswith("-"):
                            # 处理线索删除
                            lead_to_remove = value[1:].strip()
                            if "leads" in session.world_state:
                                session.world_state["leads"] = [
                                    l for l in session.world_state["leads"]
                                    if l != lead_to_remove
                                ]
                    else:
                        session.world_state[key] = value
                else:
                    logger.warning(f"Attempted to modify non-allowed state: {field_path}")

        # 保存到数据库
        self.db.add(session)
        await self.db.commit()
        await self.db.refresh(session)

        # 记录事件
        await self._log_state_change(
            session=session,
            original_state=original_state,
            new_state={
                "current_scene": session.current_scene,
                "world_state": session.world_state
            },
            source=source_description
        )

        return session

    async def _log_state_change(
        self,
        session: GameSession,
        original_state: dict,
        new_state: dict,
        source: str
    ):
        """记录状态变化事件"""
        changes = []
        if original_state["current_scene"] != new_state["current_scene"]:
            changes.append(
                f"场景: {original_state['current_scene']} → {new_state['current_scene']}"
            )

        original_leads = set(original_state["world_state"].get("leads", []))
        new_leads = set(new_state["world_state"].get("leads", []))
        if original_leads != new_leads:
            added = new_leads - original_leads
            removed = original_leads - new_leads
            if added:
                changes.append(f"新增线索: {', '.join(added)}")
            if removed:
                changes.append(f"移除线索: {', '.join(removed)}")

        if changes:
            event = Event(
                session_id=session.id,
                visibility="public",
                category="state_change",
                description=f"[{source}] " + "; ".join(changes),
                state_change={
                    "original": original_state,
                    "new": new_state
                }
            )
            self.db.add(event)
            await self.db.commit()
```

### 步骤 2: 编写测试

**创建 `backend/src/tests/test_state_sync.py`:**

```python
import pytest
from sqlalchemy.ext.asyncio import AsyncSession
from backend.src.services.state_sync import StateSyncService, ALLOWED_STATE_CHANGES
from backend.src.models.session import GameSession
from backend.src.models.event import Event
from backend.src.schemas.llm_response import StateChanges

@pytest.mark.asyncio
async def test_apply_scene_change(test_db: AsyncSession):
    service = StateSyncService(test_db)

    session = GameSession(
        id=1,
        current_scene="旧书房",
        world_state={}
    )
    test_db.add(session)
    await test_db.commit()

    changes = StateChanges(current_scene="密室")
    updated = await service.apply_state_changes(session, changes)

    assert updated.current_scene == "密室"

@pytest.mark.asyncio
async def test_apply_lead_addition(test_db: AsyncSession):
    service = StateSyncService(test_db)

    session = GameSession(
        id=1,
        current_scene="旧书房",
        world_state={"leads": ["旧钥匙"]}
    )
    test_db.add(session)
    await test_db.commit()

    changes = StateChanges(
        world_state={"leads": "+神秘笔记"}
    )
    updated = await service.apply_state_changes(session, changes)

    assert "神秘笔记" in updated.world_state["leads"]
    assert "旧钥匙" in updated.world_state["leads"]

@pytest.mark.asyncio
async def test_whitelist_enforcement(test_db: AsyncSession):
    service = StateSyncService(test_db)

    session = GameSession(
        id=1,
        current_scene="旧书房",
        world_state={}
    )
    test_db.add(session)
    await test_db.commit()

    # 尝试修改允许的字段
    changes = StateChanges(
        world_state={"leads": ["新线索"]}
    )
    updated = await service.apply_state_changes(session, changes)
    assert "新线索" in updated.world_state.get("leads", [])

    # 验证白名单包含正确的字段
    assert "current_scene" in ALLOWED_STATE_CHANGES
    assert "world_state.leads" in ALLOWED_STATE_CHANGES
```

### 步骤 3: 运行测试

```bash
cd d:/git/monika-nli/backend
uv run pytest src/tests/test_state_sync.py -v
```

### 步骤 4: 提交

```bash
cd d:/git/monika-nli
git add backend/src/services/state_sync.py backend/src/tests/test_state_sync.py
git commit -m "feat(NLI-04): add state sync service with whitelist"
```

---

## 任务 5: WebSocket 服务器

**目标:** 实现 FastAPI WebSocket 端点处理消息流

**文件:**
- 创建: `backend/src/api/websocket.py`
- 修改: `backend/src/main.py` (注册 WebSocket 路由)
- 创建: `backend/src/tests/test_websocket.py`

### 步骤 1: 创建 WebSocket 路由

**创建 `backend/src/api/websocket.py`:**

```python
import json
import logging
from typing import Dict
from fastapi import WebSocket, WebSocketDisconnect, Depends
from sqlalchemy.ext.asyncio import AsyncSession
from backend.src.core.database import get_db
from backend.src.services.llm.openai import OpenAIProvider
from backend.src.services.response_parser import ResponseParser
from backend.src.services.prompt import PromptBuilder
from backend.src.services.state_sync import StateSyncService
from backend.src.models.session import GameSession
from backend.src.models.character import Character

logger = logging.getLogger(__name__)

# 活跃连接管理器
class ConnectionManager:
    def __init__(self):
        self.active_connections: Dict[int, WebSocket] = {}

    async def connect(self, session_id: int, websocket: WebSocket):
        await websocket.accept()
        self.active_connections[session_id] = websocket
        logger.info(f"WebSocket connected for session {session_id}")

    def disconnect(self, session_id: int):
        if session_id in self.active_connections:
            del self.active_connections[session_id]
            logger.info(f"WebSocket disconnected for session {session_id}")

    async def send_message(self, session_id: int, message: dict):
        if session_id in self.active_connections:
            await self.active_connections[session_id].send_json(message)

manager = ConnectionManager()

# 初始化服务（生产环境应通过依赖注入）
llm_provider = OpenAIProvider()
response_parser = ResponseParser()
prompt_builder = PromptBuilder()

@websocket_router.websocket("/ws/game/{session_id}")
async def websocket_endpoint(
    websocket: WebSocket,
    session_id: int,
    db: AsyncSession = Depends(get_db)
):
    await manager.connect(session_id, websocket)
    state_sync = StateSyncService(db)

    try:
        # 加载会话和角色
        session_query = await db.execute(
            select(GameSession).where(GameSession.id == session_id)
        )
        session = session_query.scalar_one_or_none()

        if not session:
            await websocket.send_json({
                "type": "error",
                "content": "Session not found"
            })
            await websocket.close()
            return

        character_query = await db.execute(
            select(Character).where(Character.id == session.character_id)
        )
        character = character_query.scalar_one_or_none()

        if not character:
            await websocket.send_json({
                "type": "error",
                "content": "Character not found"
            })
            await websocket.close()
            return

        # 加载最近事件
        events_query = await db.execute(
            select(Event)
            .where(Event.session_id == session_id)
            .order_by(Event.created_at.desc())
            .limit(10)
        )
        recent_events = [
            {"description": e.description}
            for e in events_query.scalars().all()
        ]

        # 主消息循环
        while True:
            data = await websocket.receive_json()

            if data.get("type") != "user_message":
                continue

            user_message = data.get("content", "")

            # 构建 Prompt
            system_prompt = await prompt_builder.build_system_prompt()
            messages = await prompt_builder.build_context_messages(
                character=character,
                session=session,
                recent_events=recent_events,
                user_message=user_message
            )

            # 调用 LLM 并流式响应
            await manager.send_message(session_id, {
                "type": "keeper_message",
                "content": {"narrative": "", "tone": "calm", "urgency": "low"},
                "is_streaming": True
            })

            full_narrative = ""
            async for llm_response in response_parser.parse_stream(
                llm_provider.stream_chat(messages, system_prompt)
            ):
                # 发送流式片段
                full_narrative = llm_response.narrative
                await manager.send_message(session_id, {
                    "type": "keeper_message",
                    "content": {
                        "narrative": llm_response.narrative,
                        "tone": llm_response.tone,
                        "urgency": llm_response.urgency
                    },
                    "is_streaming": True
                })

            # 发送最终响应
            await manager.send_message(session_id, {
                "type": "keeper_message",
                "content": {
                    "narrative": full_narrative,
                    "tone": llm_response.tone,
                    "urgency": llm_response.urgency,
                    "suggestions": llm_response.suggestions
                },
                "is_streaming": False
            })

            # 应用状态变化
            if llm_response.state_changes:
                session = await state_sync.apply_state_changes(
                    session=session,
                    changes=llm_response.state_changes,
                    source_description="AI Keeper"
                )

                # 发送状态更新
                await manager.send_message(session_id, {
                    "type": "state_update",
                    "content": {
                        "current_scene": session.current_scene,
                        "world_state": session.world_state
                    }
                })

    except WebSocketDisconnect:
        manager.disconnect(session_id)
    except Exception as e:
        logger.error(f"WebSocket error for session {session_id}: {e}")
        await manager.send_message(session_id, {
            "type": "error",
            "content": "An error occurred"
        })
        manager.disconnect(session_id)
```

### 步骤 2: 注册 WebSocket 路由

**修改 `backend/src/main.py`:**

在文件顶部添加:
```python
from backend.src.api.websocket import websocket_router
```

在路由注册部分添加:
```python
app.include_router(websocket_router, prefix="/ws", tag="WebSocket")
```

### 步骤 3: 编写测试

**创建 `backend/src/tests/test_websocket.py`:**

```python
import pytest
from fastapi.testclient import TestClient
from fastapi import FastAPI
from backend.src.api.websocket import websocket_router

app = FastAPI()
app.include_router(websocket_router, prefix="/ws")

@pytest.mark.asyncio
async def test_websocket_connection():
    # 基础连接测试
    pass

@pytest.mark.asyncio
async def test_websocket_message_flow():
    # 消息流测试
    pass
```

### 步骤 4: 运行测试

```bash
cd d:/git/monika-nli/backend
uv run pytest src/tests/test_websocket.py -v
```

### 步骤 5: 提交

```bash
cd d:/git/monika-nli
git add backend/src/api/websocket.py backend/src/main.py backend/src/tests/test_websocket.py
git commit -m "feat(NLI-05): add WebSocket server endpoint"
```

---

## 任务 6: 前端 WebSocket 客户端

**目标:** 实现 React WebSocket 连接管理和消息处理

**文件:**
- 创建: `frontend/src/services/websocket.ts`
- 创建: `frontend/src/hooks/useLLMResponse.ts`
- 创建: `frontend/src/hooks/useGameWebSocket.ts`
- 创建: `frontend/src/types/websocket.ts`

### 步骤 1: 定义类型

**创建 `frontend/src/types/websocket.ts`:**

```typescript
export type ToneType = 'mystery' | 'horror' | 'action' | 'calm';
export type UrgencyType = 'low' | 'medium' | 'high';

export interface StateChanges {
  current_scene?: string;
  world_state?: {
    leads?: string[];
    location?: string;
    npcs?: Record<string, any>;
  };
}

export interface LLMResponse {
  narrative: string;
  tone: ToneType;
  urgency: UrgencyType;
  state_changes?: StateChanges;
  suggestions?: string[];
  audio_cue?: string;
  requires_roll: boolean;
}

export interface UserMessage {
  type: 'user_message';
  content: string;
  timestamp: string;
}

export interface KeeperMessage {
  type: 'keeper_message';
  content: LLMResponse;
  is_streaming: boolean;
  timestamp?: string;
}

export interface StateUpdate {
  type: 'state_update';
  content: {
    current_scene: string;
    world_state: Record<string, any>;
  };
}

export interface ErrorMessage {
  type: 'error';
  content: string;
}

export type ServerMessage = KeeperMessage | StateUpdate | ErrorMessage;
```

### 步骤 2: 创建 WebSocket 服务

**创建 `frontend/src/services/websocket.ts`:**

```typescript
import { ServerMessage, UserMessage } from '../types/websocket';

export interface WebSocketCallbacks {
  onMessage: (message: ServerMessage) => void;
  onStateUpdate: (update: StateUpdate) => void;
  onError: (error: string) => void;
  onConnect: () => void;
  onDisconnect: () => void;
}

export class GameWebSocket {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private sessionId: string | null = null;
  private callbacks: WebSocketCallbacks;

  constructor(callbacks: WebSocketCallbacks) {
    this.callbacks = callbacks;
  }

  connect(sessionId: string): void {
    this.sessionId = sessionId;
    const wsUrl = `ws://localhost:8000/ws/game/${sessionId}`;

    try {
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        console.log('WebSocket connected');
        this.reconnectAttempts = 0;
        this.callbacks.onConnect();
      };

      this.ws.onmessage = (event) => {
        try {
          const message: ServerMessage = JSON.parse(event.data);
          this.callbacks.onMessage(message);

          if (message.type === 'state_update') {
            this.callbacks.onStateUpdate(message);
          }
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error);
        }
      };

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        this.callbacks.onError('Connection error');
      };

      this.ws.onclose = () => {
        console.log('WebSocket closed');
        this.callbacks.onDisconnect();
        this.attemptReconnect();
      };
    } catch (error) {
      console.error('Failed to create WebSocket:', error);
      this.callbacks.onError('Failed to connect');
    }
  }

  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  send(message: UserMessage): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    } else {
      console.error('WebSocket is not connected');
    }
  }

  private attemptReconnect(): void {
    if (this.reconnectAttempts < this.maxReconnectAttempts && this.sessionId) {
      this.reconnectAttempts++;
      const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

      console.log(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);

      setTimeout(() => {
        this.connect(this.sessionId!);
      }, delay);
    }
  }
}
```

### 步骤 3: 创建 LLM 响应 Hook

**创建 `frontend/src/hooks/useLLMResponse.ts`:**

```typescript
import { useState, useCallback } from 'react';
import { LLMResponse } from '../types/websocket';

export function useLLMResponse() {
  const [streamingText, setStreamingText] = useState('');
  const [isStreaming, setIsStreaming] = useState(false);
  const [currentResponse, setCurrentResponse] = useState<LLMResponse | null>(null);

  const processStream = useCallback((chunk: LLMResponse) => {
    setStreamingText(chunk.narrative);
    setIsStreaming(true);
  }, []);

  const finalizeResponse = useCallback((response: LLMResponse) => {
    setCurrentResponse(response);
    setStreamingText(response.narrative);
    setIsStreaming(false);
  }, []);

  const reset = useCallback(() => {
    setStreamingText('');
    setIsStreaming(false);
    setCurrentResponse(null);
  }, []);

  return {
    streamingText,
    isStreaming,
    currentResponse,
    processStream,
    finalizeResponse,
    reset
  };
}
```

### 步骤 4: 创建游戏 WebSocket Hook

**创建 `frontend/src/hooks/useGameWebSocket.ts`:**

```typescript
import { useState, useEffect, useCallback, useRef } from 'react';
import { GameWebSocket } from '../services/websocket';
import { UserMessage, ServerMessage, LLMResponse } from '../types/websocket';

export function useGameWebSocket(sessionId: string | null) {
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const wsRef = useRef<GameWebSocket | null>(null);

  useEffect(() => {
    if (!sessionId) return;

    const ws = new GameWebSocket({
      onMessage: (message: ServerMessage) => {
        if (message.type === 'keeper_message') {
          // 由调用者处理
        }
      },
      onStateUpdate: (update) => {
        // 状态更新逻辑
      },
      onError: (err) => {
        setError(err);
      },
      onConnect: () => {
        setIsConnected(true);
        setError(null);
      },
      onDisconnect: () => {
        setIsConnected(false);
      }
    });

    ws.connect(sessionId);
    wsRef.current = ws;

    return () => {
      ws.disconnect();
    };
  }, [sessionId]);

  const sendMessage = useCallback((content: string) => {
    if (wsRef.current) {
      const message: UserMessage = {
        type: 'user_message',
        content,
        timestamp: new Date().toISOString()
      };
      wsRef.current.send(message);
    }
  }, []);

  return {
    isConnected,
    error,
    sendMessage
  };
}
```

### 步骤 5: 提交

```bash
cd d:/git/monika-nli
git add frontend/src/types/websocket.ts frontend/src/services/websocket.ts frontend/src/hooks/useLLMResponse.ts frontend/src/hooks/useGameWebSocket.ts
git commit -m "feat(NLI-06): add frontend WebSocket client and hooks"
```

---

## 任务 7: 集成到 GameConsole 组件

**目标:** 将自然语言输入集成到现有游戏控制台

**文件:**
- 修改: `frontend/src/components/GameConsole.tsx`

### 步骤 1: 更新 GameConsole 组件

**修改 `frontend/src/components/GameConsole.tsx`:**

在现有组件中添加自然语言输入支持。假设组件有消息输入区域，添加以下逻辑：

```typescript
import { useGameWebSocket } from '../hooks/useGameWebSocket';
import { useLLMResponse } from '../hooks/useLLMResponse';

// 在组件内部
const sessionId = useSelector(state => state.game.sessionId);
const { isConnected, error, sendMessage } = useGameWebSocket(sessionId);
const { streamingText, isStreaming, currentResponse, reset: resetLLM } = useLLMResponse();

// 处理用户消息发送
const handleSendMessage = useCallback((content: string) => {
  sendMessage(content);
  resetLLM();
}, [sendMessage, resetLLM]);
```

### 步骤 2: 提交

```bash
cd d:/git/monika-nli
git add frontend/src/components/GameConsole.tsx
git commit -m "feat(NLI-07): integrate WebSocket with GameConsole"
```

---

## 任务 8: 配置管理

**目标:** 添加环境变量配置

**文件:**
- 修改: `backend/src/core/config.py`
- 创建: `backend/.env.example`

### 步骤 1: 更新配置

**修改 `backend/src/core/config.py`:**

```python
from pydantic import Field

class Settings(BaseSettings):
    # ... 现有配置 ...

    # LLM 配置
    llm_provider: str = Field(default="openai", description="LLM Provider: openai, claude, qwen")
    openai_api_key: Optional[str] = Field(default=None, description="OpenAI API Key")
    openai_model: str = Field(default="gpt-4", description="OpenAI Model")
    claude_api_key: Optional[str] = Field(default=None, description="Anthropic API Key")
    claude_model: str = Field(default="claude-3-sonnet-20240229", description="Claude Model")

    # WebSocket 配置
    ws_heartbeat_interval: int = Field(default=30, description="WebSocket heartbeat interval (seconds)")
    ws_reconnect_max_attempts: int = Field(default=5, description="Max WebSocket reconnect attempts")

    class Config:
        env_file = ".env"
```

### 步骤 2: 创建环境变量示例

**创建 `backend/.env.example`:**

```bash
# LLM Provider 配置
LLM_PROVIDER=openai
OPENAI_API_KEY=sk-your-api-key-here
OPENAI_MODEL=gpt-4
CLAUDE_API_KEY=sk-ant-your-api-key-here

# WebSocket 配置
WS_HEARTBEAT_INTERVAL=30
WS_RECONNECT_MAX_ATTEMPTS=5
```

### 步骤 3: 提交

```bash
cd d:/git/monika-nli
git add backend/src/core/config.py backend/.env.example
git commit -m "feat(NLI-08): add LLM and WebSocket configuration"
```

---

## 任务 9: 端到端测试

**目标:** 完整的端到端测试

**文件:**
- 创建: `backend/src/tests/test_e2e_nli.py`

### 步骤 1: 创建 E2E 测试

**创建 `backend/src/tests/test_e2e_nli.py`:**

```python
import pytest
from httpx_ws import aconnect_ws
from backend.src.main import app

@pytest.mark.asyncio
async def test_nli_message_flow():
    """测试完整的自然语言交互流程"""
    async with aconnect_ws("/ws/game/1", app) as websocket:
        # 发送用户消息
        await websocket.send_json({
            "type": "user_message",
            "content": "我想查看这个房间",
            "timestamp": "2025-01-15T10:00:00Z"
        })

        # 接收流式响应
        response_count = 0
        while True:
            response = await websocket.receive_json()
            response_count += 1

            if response.get("type") == "keeper_message":
                if not response.get("is_streaming"):
                    # 流式结束
                    break

        assert response_count > 0
```

### 步骤 2: 运行完整测试套件

```bash
cd d:/git/monika-nli/backend
uv run pytest src/tests/ -v --cov=src
```

### 步骤 3: 提交

```bash
cd d:/git/monika-nli
git add backend/src/tests/test_e2e_nli.py
git commit -m "test(NLI-09): add end-to-end tests for NLI system"
```

---

## 任务 10: 文档和清理

**目标:** 更新文档，代码清理

**文件:**
- 修改: `docs/specs/api.md` (添加 WebSocket API 文档)
- 修改: `CLAUDE.md` (更新项目指南)

### 步骤 1: 更新 API 文档

**在 `docs/specs/api.md` 中添加:**

```markdown
## WebSocket API

### 连接端点
`ws://localhost:8000/ws/game/{session_id}`

### 消息格式
...
```

### 步骤 2: 最终测试和提交

```bash
cd d:/git/monika-nli/backend
uv run pytest

cd d:/git/monika-nli
git add docs/
git commit -m "docs(NLI-10): update API documentation"
```

---

## 验收标准

完成所有任务后，确认以下条件：

- [ ] 所有单元测试通过
- [ ] WebSocket 连接正常工作
- [ ] LLM 能够正确响应并返回 JSON
- [ ] 状态变化能够安全地同步到数据库
- [ ] 前端能够正确显示流式响应
- [ ] 环境变量配置正确
- [ ] 文档已更新

---

## 风险与缓解

| 风险 | 缓解措施 |
|------|----------|
| LLM API 限流 | 实现请求队列和重试逻辑 |
| JSON 解析失败 | 降级处理，提取纯文本 |
| WebSocket 断连 | 自动重连机制 |
| 状态同步冲突 | 使用白名单验证 |

---

**实施计划版本:** 1.0
**最后更新:** 2025-01-15
