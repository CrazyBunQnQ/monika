# 自然语言交互系统设计文档

**日期**: 2025-01-15
**作者**: AI Assistant
**状态**: 设计阶段
**相关任务**: M1-046 ~ M1-056

---

## 1. 概述

### 1.1 设计目标

为 Monika 项目实现自然语言交互系统，让玩家能够通过自然语言与 AI 守密人（Keeper）进行角色扮演交互，获得沉浸式的 TRPG 体验。

### 1.2 核心原则

- **职责分离**: 游戏机制（检定、战斗、追逐）通过独立的 UI 控件执行，AI 守密人只负责叙事和角色扮演
- **规则适应**: 以 CoC 7e 规则为基础，但可以根据玩家需求调整时代背景和规则解释
- **完整记忆**: AI 能够访问从游戏开始的所有对话历史和事件，理解长期情节发展
- **安全可控**: AI 只能修改预定义的状态字段，核心角色属性仍需通过检定系统

---

## 2. 系统架构

### 2.1 架构图

```
┌─────────────────┐         WebSocket          ┌─────────────────┐
│                 │ ◄─────────────────────────► │                 │
│  Frontend (React)│                             │  Backend (FastAPI)│
│                 │                              │                 │
├─────────────────┤                              ├─────────────────┤
│ • WebSocket     │                              │ • WebSocket     │
│   Client        │                              │   Server        │
│ • Message       │                              │ • LLM Abstraction│
│   Processor     │                              │   Layer         │
│ • State Hook    │                              │ • Prompt Engine │
└─────────────────┘                              │ • Response      │
                                                 │   Parser        │
                                                 │ • State Sync    │
                                                 └─────────┬───────┘
                                                           │
                                                 ┌─────────▼───────┐
                                                 │  LLM Providers  │
                                                 ├─────────────────┤
                                                 │ • OpenAI (GPT-4)│
                                                 │ • Anthropic     │
                                                 │ • Qwen (可自部署)│
                                                 └─────────────────┘
```

### 2.2 数据流

1. 玩家在前端输入自然语言消息
2. 通过 WebSocket 发送到后端
3. 后端构建 Prompt（包含会话历史、角色状态、场景信息）
4. 调用 LLM API，流式获取响应
5. 解析 JSON 响应
6. 执行状态变化（更新数据库，记录事件）
7. 通过 WebSocket 将响应推送到前端
8. 前端显示叙述并更新 UI

---

## 3. 后端设计

### 3.1 LLM 抽象层

**基础接口** (`backend/src/services/llm/base.py`):

```python
from abc import ABC, abstractmethod
from typing import AsyncIterator

class LLMProvider(ABC):
    @abstractmethod
    async def stream_chat(
        self,
        messages: list[dict],
        system_prompt: str
    ) -> AsyncIterator[str]:
        """流式响应，返回 JSON 字符串片段"""
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

**Provider 实现**:
- `OpenAIProvider` - 使用 GPT-4/GPT-3.5
- `ClaudeProvider` - 使用 Anthropic Claude
- `QwenProvider` - 使用通义千问（可自部署）

### 3.2 Prompt 模板引擎

**系统提示词模板**:

```
你是克苏鲁的呼唤 7 版（Call of Cthulhu 7th Edition）的守密人（Keeper）。

你的职责：
1. 描述场景、NPC 反应和故事发展
2. 保持神秘和恐怖的氛围
3. 尊重玩家的选择，推动故事发展
4. 可以修改场景信息和添加线索，但不能直接修改玩家角色的 HP/SAN 等核心属性

你当前知道的：
- 当前场景：{current_scene}
- 角色：{character_name}，{character_description}
- 最近事件：{recent_events}

请以 JSON 格式响应，包含以下字段：
- narrative: 叙述文本（第二人称"你"）
- tone: 语气 (mystery|horror|action|calm)
- urgency: 紧迫度 (low|medium|high)
- state_changes: 状态变化（可选）
- suggestions: 给玩家的建议（可选）
```

**上下文构建**:
- 会话历史：最近 50 条消息
- 当前场景信息
- 角色当前状态
- 世界状态

### 3.3 JSON 响应结构

**Schema 定义** (`backend/src/schemas/llm_response.py`):

```python
from pydantic import BaseModel
from typing import Optional, Dict, List

class StateChanges(BaseModel):
    current_scene: Optional[str] = None
    world_state: Optional[Dict] = None

class LLMResponse(BaseModel):
    narrative: str  # 必填：叙述文本
    tone: str = "calm"
    urgency: str = "low"
    state_changes: Optional[StateChanges] = None
    suggestions: Optional[List[str]] = None
    audio_cue: Optional[str] = None
    requires_roll: bool = False
```

### 3.4 状态同步机制

**可修改的状态字段**（白名单）:
- `GameSession.current_scene` - 当前场景
- `GameSession.world_state.leads` - 线索列表
- `GameSession.world_state.location` - 地点信息
- `GameSession.world_state.npcs` - NPC 状态

**不可修改的状态**（需通过检定系统）:
- `Character.hp`, `Character.san`, `Character.mp`
- `Character.skills`
- `Character.attributes`

**状态同步流程**:
1. LLM 响应返回后，解析 `state_changes` 字段
2. 验证每个字段是否在白名单中
3. 更新数据库中的对应记录
4. 创建 Event 记录
5. 通过 WebSocket 推送状态更新

### 3.5 WebSocket 服务

**路由**: `ws://localhost:8000/ws/game/{session_id}`

**消息处理器**:

```python
@websocket_router.websocket("/ws/game/{session_id}")
async def websocket_endpoint(websocket: WebSocket, session_id: str):
    await websocket.accept()
    try:
        while True:
            data = await websocket.receive_json()
            response = await process_user_message(data)
            await stream_response(websocket, response)
    except WebSocketDisconnect:
        logger.info(f"WebSocket disconnected: {session_id}")
```

---

## 4. 前端设计

### 4.1 WebSocket 客户端

**服务类** (`frontend/src/services/websocket.ts`):

```typescript
class GameWebSocket {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private messageQueue: ServerMessage[] = [];

  connect(sessionId: string): void
  disconnect(): void
  send(message: UserMessage): void
  onMessage(callback: (msg: ServerMessage) => void): void
  onStateUpdate(callback: (update: StateUpdate) => void): void
  private reconnect(): void
}
```

### 4.2 消息流处理器

**自定义 Hook** (`frontend/src/hooks/useLLMResponse.ts`):

```typescript
function useLLMResponse() {
  const [streamingText, setStreamingText] = useState("");
  const [isComplete, setIsComplete] = useState(false);
  const [response, setResponse] = useState<LLMResponse | null>(null);

  const processStream = (chunk: LLMResponseChunk) => {
    setStreamingText(prev => prev + chunk.narrative);
  };

  const finalizeResponse = (fullResponse: LLMResponse) => {
    setResponse(fullResponse);
    setIsComplete(true);
  };

  return { streamingText, isComplete, response, processStream, finalizeResponse };
}
```

### 4.3 消息协议

**客户端 → 服务器**:
```json
{
  "type": "user_message",
  "content": "我想仔细查看这个房间",
  "timestamp": "2025-01-15T10:30:00Z"
}
```

**服务器 → 客户端**:
```json
{
  "type": "keeper_message",
  "content": {
    "narrative": "你仔细观察这个房间...",
    "tone": "mystery",
    "urgency": "low"
  },
  "is_streaming": true,
  "timestamp": "2025-01-15T10:30:01Z"
}
```

**事件类型**:
- `user_message` - 用户消息
- `keeper_message` - Keeper 响应（流式）
- `state_update` - 状态更新
- `error` - 错误消息
- `heartbeat` - 心跳（每 30 秒）

---

## 5. 配置与安全

### 5.1 环境变量

```bash
# LLM Provider 配置
LLM_PROVIDER=openai
OPENAI_API_KEY=sk-xxx
OPENAI_MODEL=gpt-4
CLAUDE_API_KEY=sk-ant-xxx
QWEN_API_KEY=xxx

# WebSocket 配置
WS_HEARTBEAT_INTERVAL=30
WS_RECONNECT_MAX_ATTEMPTS=5
```

### 5.2 安全措施

1. **API Key 管理** - 使用环境变量，不提交到 Git
2. **输入验证** - 限制用户消息长度（最大 2000 字符）
3. **速率限制** - 每用户每分钟最多 10 条消息
4. **内容过滤** - 检测并拒绝不当内容

---

## 6. 测试策略

### 6.1 单元测试

**Mock LLM Provider**:
```python
class MockLLMProvider(LLMProvider):
    async def stream_chat(self, messages, system_prompt):
        yield '{"narrative": "测试响应", "tone": "calm"}'
```

**测试用例**:
- `test_llm_response_parsing()` - JSON 响应解析
- `test_state_sync_whitelist()` - 状态同步白名单验证
- `test_invalid_json_recovery()` - JSON 解析失败时的降级处理

### 6.2 集成测试

- `test_websocket_message_flow()` - 完整消息流测试
- `test_websocket_reconnect()` - 断线重连测试

---

## 7. 实现任务

| ID | 任务 | 预估工时 |
|----|------|----------|
| M1-048 | 实现意图识别服务 | 6h |
| M1-049 | 实现 TRPG 门禁检查 | 4h |
| M1-050 | 实现拒绝模板响应 | 4h |
| M1-051 | 集成 OpenAI API | 4h |
| M1-052 | 实现 LLM Prompt 模板 | 2h |
| M1-053 | 实现流式响应 Streaming | 4h |
| M1-054 | 实现响应解析器 | 2h |
| M1-055 | 实现 WebSocket 连接服务 | 2h |
| M1-056 | 实现消息发送/接收 | 2h |

---

## 8. 风险与应对

| 风险 | 应对措施 |
|------|----------|
| LLM 响应不稳定 | 降级策略 + 友好错误提示 |
| JSON 解析失败 | 提取纯文本作为降级方案 |
| WebSocket 断连 | 自动重连机制 |
| API 调用成本 | 支持开源模型作为备选 |

---

**文档版本**: 1.0
**最后更新**: 2025-01-15
