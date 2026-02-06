# M3-007: 实现 AI 辅助功能

**任务ID**: M3-007
**标题**: 实现 AI 辅助功能
**类型**: backend (后端开发)
**预估工时**: 2.5h
**依赖**: M3-001

---

## 任务描述

实现 AI 辅助功能，帮助 KP 自动生成 NPC 对话、场景描述、任务线索等内容，提升创作效率。

---

## 子任务拆解

| ID | 子任务 | 描述 | 预估时间 |
|----|--------|------|----------|
| M3-007-01 | 设计 AI 接口层 | AI Interface | 25min |
| M3-007-02 | 实现 NPC 对话生成 | NPC Dialogue | 35min |
| M3-007-03 | 实现场景描述生成 | Scene Description | 30min |
| M3-007-04 | 实现线索生成 | Clue Generation | 25min |
| M3-007-05 | 实现任务生成 | Quest Generation | 30min |
| M3-007-06 | 编写 AI 测试 | 测试覆盖 | 20min |

---

## AI 服务接口

```python
# app/services/ai/ai_interface.py
from typing import Dict, Any, List, Optional
from abc import ABC, abstractmethod

class AIProvider(ABC):
    """AI 提供商接口"""

    @abstractmethod
    async def generate_text(
        self,
        prompt: str,
        max_tokens: int = 1000,
        temperature: float = 0.7,
    ) -> str:
        """生成文本"""
        pass

    @abstractmethod
    async def generate_json(
        self,
        prompt: str,
        schema: Dict,
    ) -> Dict[str, Any]:
        """生成 JSON"""
        pass

class OpenAIProvider(AIProvider):
    """OpenAI 提供商"""

    def __init__(self, api_key: str, model: str = "gpt-4"):
        self.api_key = api_key
        self.model = model

    async def generate_text(
        self,
        prompt: str,
        max_tokens: int = 1000,
        temperature: float = 0.7,
    ) -> str:
        import openai

        openai.api_key = self.api_key

        response = await openai.ChatCompletion.acreate(
            model=self.model,
            messages=[{"role": "user", "content": prompt}],
            max_tokens=max_tokens,
            temperature=temperature,
        )

        return response.choices[0].message.content

    async def generate_json(
        self,
        prompt: str,
        schema: Dict,
    ) -> Dict[str, Any]:
        import json

        # 在提示中指定 JSON 格式
        json_prompt = f"""{prompt}

请以 JSON 格式回复，格式如下：
{json.dumps(schema, ensure_ascii=False)}"""

        response = await self.generate_text(json_prompt)
        return json.loads(response)
```

---

## NPC 对话生成服务

```python
# app/services/ai/npc_dialogue.py
from typing import List, Dict, Any
from sqlalchemy.orm import Session

from app.services.ai.ai_interface import OpenAIProvider
from app.db.models.npc import NPC

class NPCDialogueService:
    """NPC 对话生成服务"""

    def __init__(
        self,
        db: Session,
        ai_provider: OpenAIProvider,
    ):
        self.db = db
        self.ai = ai_provider

    async def generate_dialogue(
        self,
        npc_id: str,
        player_input: str,
        context: Dict[str, Any] = None,
    ) -> str:
        """生成 NPC 回应"""
        npc = self.db.query(NPC).filter(NPC.id == npc_id).first()
        if not npc:
            raise ValueError("NPC 不存在")

        # 构建提示
        prompt = self._build_dialogue_prompt(
            npc=npc,
            player_input=player_input,
            context=context or {},
        )

        # 生成对话
        response = await self.ai.generate_text(
            prompt,
            max_tokens=500,
            temperature=0.8,
        )

        return response

    def _build_dialogue_prompt(
        self,
        npc: NPC,
        player_input: str,
        context: Dict,
    ) -> str:
        """构建对话提示"""
        prompt = f"""你是一个《克苏鲁的呼唤》(Call of Cthulhu) 的守密人(KP)。现在需要你扮演一个 NPC 与玩家互动。

NPC 信息：
- 姓名: {npc.name}
- 职业: {npc.occupation}
- 性格: {npc.personality or '普通'}
- 外貌: {npc.appearance or '未描述'}
- 背景: {npc.background or '无'}

玩家说: "{player_input}"

当前场景: {context.get('scene', '未知')}
在场角色: {context.get('characters', [])}
相关线索: {context.get('clues', [])}

请根据 NPC 的性格特点，生成一段符合克苏鲁风格的回应。回应应该：
1. 符合 NPC 的性格特征
2. 体现 1920 年代的时代背景
3. 带有神秘和不安的氛围
4. 回复长度在 50-100 字左右

请直接输出 NPC 的对话内容，不要包含任何说明。
"""

        return prompt

    async def generate_dialogue_options(
        self,
        npc_id: str,
        player_input: str,
        context: Dict = None,
    ) -> List[str]:
        """生成多个对话选项"""
        npc = self.db.query(NPC).filter(NPC.id == npc_id).first()
        if not npc:
            raise ValueError("NPC 不存在")

        prompt = f"""基于以下 NPC 信息和玩家的输入，生成 3-4 个可能的对话选项：

NPC: {npc.name} ({npc.occupation})
性格: {npc.personality}
玩家说: {player_input}

每个选项应该：
1. 反映 NPC 的不同情绪或态度
2. 提供不同的对话方向
3. 保持角色一致性
4. 简洁，每条不超过 20 字

请以 JSON 数组格式返回，格式：["选项1", "选项2", "选项3"]
"""

        response = await self.ai.generate_text(prompt, temperature=0.9)
        import json
        try:
            return json.loads(response)
        except:
            # 简单的回退方案
            return [
                f"{npc.name}: 当然...",
                f"{npc.name}: 我不清楚...",
                f"{npc.name}: 这很奇怪...",
            ]
```

---

## 场景描述生成服务

```python
# app/services/ai/scene_description.py
from typing import Dict, Any
from sqlalchemy.orm import Session

from app.services.ai.ai_interface import OpenAIProvider

class SceneDescriptionService:
    """场景描述生成服务"""

    def __init__(
        self,
        db: Session,
        ai_provider: OpenAIProvider,
    ):
        self.db = db
        self.ai = ai_provider

    async def generate_scene_description(
        self,
        scene_name: str,
        scene_type: str,
        location: str,
        mood: str = "neutral",
        additional_context: Dict = None,
    ) -> str:
        """生成场景描述"""
        prompt = f"""你是一个克苏鲁守密人(KP)。请为以下场景生成一段充满氛围感的描述：

场景名称: {scene_name}
类型: {scene_type}  (如: 住宅、森林、墓地、图书馆)
地点: {location}
氛围: {mood}  (如: 恐怖、神秘、压抑、诡异)

{"相关上下文: " + str(additional_context) if additional_context else ""}

描述要求：
1. 字数 150-250 字
2. 注重感官描写（视觉、听觉、嗅觉、触觉）
3. 营造克苏鲁式的恐怖氛围
4. 包含一些微妙的不祥之兆
5. 保持 1920 年代的时代感
6. 不要过于直白的恐怖，要留有想象空间

请直接输出场景描述，不要包含任何说明。
"""

        return await self.ai.generate_text(
            prompt,
            max_tokens=500,
            temperature=0.8,
        )

    async def expand_scene_description(
        self,
        brief_description: str,
        focus_areas: List[str] = None,
    ) -> Dict[str, str]:
        """扩展场景描述"""
        focus_areas = focus_areas or [
            "视觉细节",
            "声音",
            "气味",
            "温度/天气",
            "异常现象",
        ]

        prompt = f"""基于以下简短描述，为每个方面生成详细的描述：

原始描述: {brief_description}

请为以下方面生成描述，每个方面 2-3 句话：
{', '.join(focus_areas)}

要求：
1. 保持克苏鲁风格
2. 每个方面都要有具体、可感知的细节
3. 互相呼应，形成一个统一的整体

请以 JSON 格式返回：
{{
  "视觉细节": "...",
  "声音": "...",
  "气味": "...",
  "温度/天气": "...",
  "异常现象": "..."
}}
"""

        return await self.ai.generate_json(prompt, {})
```

---

## 线索生成服务

```python
# app/services/ai/clue_generation.py
from typing import List, Dict, Any
from sqlalchemy.orm import Session

from app.services.ai.ai_interface import OpenAIProvider

class ClueGenerationService:
    """线索生成服务"""

    def __init__(
        self,
        db: Session,
        ai_provider: OpenAIProvider,
    ):
        self.db = db
        self.ai = ai_provider

    async def generate_clues(
        self,
        scene_context: str,
        investigation_focus: str,
        difficulty: str = "medium",
        count: int = 3,
    ) -> List[Dict[str, Any]]:
        """生成线索"""
        prompt = f"""作为一个克苏鲁守密人，请为以下场景生成 {count} 条线索：

场景背景：
{scene_context}

调查重点：{investigation_focus}
难度：{difficulty}

每条线索应包含：
1. 线索名称
2. 线索类型 (physical, testimony, documentary, mystical)
3. 线索描述
4. 可能指向的真相
5. 获取方式

要求：
- 线索要有层次，从明显到隐晦
- 线索之间可以有关联，也可以是独立的
- 保持克苏鲁风格，不要太直白
- 线索类型多样化

请以 JSON 数组格式返回：
[
  {{
    "name": "线索名称",
    "type": "physical",
    "description": "描述",
    "implication": "暗示",
    "method": "获取方式"
  }}
]
"""

        response = await self.ai.generate_json(
            prompt,
            schema={
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "name": {"type": "string"},
                        "type": {"type": "string"},
                        "description": {"type": "string"},
                        "implication": {"type": "string"},
                        "method": {"type": "string"},
                    },
                },
            },
        )

        return response
```

---

## 任务生成服务

```python
# app/services/ai/quest_generation.py
from typing import List, Dict, Any
from sqlalchemy.orm import Session

from app.services.ai.ai_interface import OpenAIProvider

class QuestGenerationService:
    """任务生成服务"""

    def __init__(
        self,
        db: Session,
        ai_provider: OpenAIProvider,
    ):
        self.db = db
        self.ai = ai_provider

    async def generate_quest(
        self,
        theme: str,
        scope: str = "short",
        complexity: str = "medium",
        party_level: int = 3,
        player_count: int = 4,
    ) -> Dict[str, Any]:
        """生成任务"""
        prompt = f"""作为一个克苏鲁守密人，请生成一个{scope}调查任务：

主题：{theme}
规模：{scope} (short/medium/long)
复杂度：{complexity} (easy/medium/hard)
玩家等级：{party_level} (经验等级 1-10)
玩家数量：{player_count}

任务结构应包含：
1. 任务名称
2. 任务概述
3. 主要阶段 (3-5 个)
4. 每个阶段的关键场景
5. 可能的线索分布
6. 预期的调查方式
7. 危险点与 SAN 检定
8. 可能的结局

要求：
- 符合克苏鲁的恐怖氛围
- 调查方式多样化（访谈、现场勘查、文献检索等）
- 留有多个解决路径
- 包含道德困境或危险选择

请以 JSON 格式返回任务结构。
"""

        response = await self.ai.generate_json(
            prompt,
            schema={
                "type": "object",
                "properties": {
                    "name": {"type": "string"},
                    "overview": {"type": "string"},
                    "phases": {
                        "type": "array",
                        "items": {
                            "type": "object",
                            "properties": {
                                "title": {"type": "string"},
                                "scenes": {"type": "array", "items": {"type": "string"}},
                                "clues": {"type": "array", "items": {"type": "string"}},
                                "investigation_methods": {"type": "array", "items": {"type": "string"}},
                            },
                        },
                    },
                    "san_checks": {
                        "type": "array",
                        "items": {"type": "integer"},
                    },
                    "endings": {
                        "type": "array",
                        "items": {
                            "type": "object",
                            "properties": {
                                "condition": {"type": "string"},
                                "description": {"type": "string"},
                            },
                        },
                    },
                },
            },
        )

        return response

    async def generate_twist(
        self,
        current_quest: Dict[str, Any],
        twist_type: str = "revelation",
    ) -> Dict[str, Any]:
        """生成剧情反转"""
        prompt = f"""为以下克苏鲁任务生成一个剧情反转：

当前任务：
{current_quest}

反转类型：{twist_type}
(revelation: 真相揭露, betrayal: 背叛, deception: 欺骗, escalation: 升级)

反转要求：
- 出人意料但合乎情理
- 重新解释之前的线索
- 提供新的调查方向
- 保持恐怖氛围

请返回：
{{
  "trigger": "触发事件",
  "revelation": "揭露内容",
  "new_leads": ["新线索1", "新线索2"],
  "implications": "对任务的影响"
}}
"""

        return await self.ai.generate_json(prompt, {})
```

---

## AI 辅助 API

```python
# app/api/ai.py
from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel
from typing import List, Optional

from app.db.database import get_db
from app.api.deps.auth import get_current_user
from app.api.deps.permissions import require_room_role
from app.db.models.user import User
from app.services.ai.npc_dialogue import NPCDialogueService
from app.services.ai.scene_description import SceneDescriptionService
from app.services.ai.clue_generation import ClueGenerationService
from app.services.ai.quest_generation import QuestGenerationService
from app.services.ai.ai_interface import OpenAIProvider

router = APIRouter(prefix="/ai", tags=["ai"])

# 初始化 AI 提供商
ai_provider = OpenAIProvider(api_key=os.getenv("OPENAI_API_KEY"))

class NPCDialogueRequest(BaseModel):
    npc_id: str
    input: str
    context: dict = None

class SceneDescriptionRequest(BaseModel):
    scene_name: str
    scene_type: str
    location: str
    mood: str = "neutral"
    context: dict = None

class ClueGenerationRequest(BaseModel):
    scene_context: str
    investigation_focus: str
    difficulty: str = "medium"
    count: int = 3

class QuestGenerationRequest(BaseModel):
    theme: str
    scope: str = "short"
    complexity: str = "medium"
    party_level: int = 3
    player_count: int = 4

@router.post("/npc/dialogue")
async def generate_npc_dialogue(
    request: NPCDialogueRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """生成 NPC 对话"""
    service = NPCDialogueService(db, ai_provider)

    dialogue = await service.generate_dialogue(
        npc_id=request.npc_id,
        player_input=request.input,
        context=request.context,
    )

    return {"dialogue": dialogue}

@router.post("/scene/description")
async def generate_scene_description(
    request: SceneDescriptionRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """生成场景描述"""
    service = SceneDescriptionService(db, ai_provider)

    description = await service.generate_scene_description(
        scene_name=request.scene_name,
        scene_type=request.scene_type,
        location=request.location,
        mood=request.mood,
        additional_context=request.context,
    )

    return {"description": description}

@router.post("/clues/generate")
async def generate_clues(
    request: ClueGenerationRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """生成线索"""
    service = ClueGenerationService(db, ai_provider)

    clues = await service.generate_clues(
        scene_context=request.scene_context,
        investigation_focus=request.investigation_focus,
        difficulty=request.difficulty,
        count=request.count,
    )

    return {"clues": clues}

@router.post("/quest/generate")
async def generate_quest(
    request: QuestGenerationRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """生成任务"""
    service = QuestGenerationService(db, ai_provider)

    quest = await service.generate_quest(
        theme=request.theme,
        scope=request.scope,
        complexity=request.complexity,
        party_level=request.party_level,
        player_count=request.player_count,
    )

    return {"quest": quest}
```

---

## 涉及文件清单

| 文件路径 | 操作 | 说明 |
|----------|------|------|
| `app/services/ai/ai_interface.py` | 创建 | AI 接口层 |
| `app/services/ai/npc_dialogue.py` | 创建 | NPC 对话生成 |
| `app/services/ai/scene_description.py` | 创建 | 场景描述生成 |
| `app/services/ai/clue_generation.py` | 创建 | 线索生成 |
| `app/services/ai/quest_generation.py` | 创建 | 任务生成 |
| `app/api/ai.py` | 创建 | AI 辅助 API |
| `frontend/src/components/game/AIAssistant.tsx` | 创建 | AI 助手组件 |

---

## 验收标准

- [ ] NPC 对话自然
- [ ] 场景描述有氛围
- [ ] 线索生成合理
- [ ] 任务结构完整
- [ ] 剧情反转出人意料
- [ ] 生成速度可接受

---

## 参考文档

- M3-001: AI 总结服务
- OpenAI API 文档

---

**最后更新**: 2026-02-06
**状态**: [ ] 待开始
