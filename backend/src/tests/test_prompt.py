"""Tests for prompt template engine."""
import pytest
from src.services.prompt import PromptBuilder


@pytest.mark.asyncio
async def test_build_system_prompt():
    """Test system prompt generation contains required elements."""
    builder = PromptBuilder()
    prompt = await builder.build_system_prompt()

    assert "克苏鲁的呼唤" in prompt
    assert "守密人" in prompt
    assert "JSON 格式" in prompt
    assert "narrative" in prompt
    assert "tone" in prompt
    assert "urgency" in prompt


@pytest.mark.asyncio
async def test_build_context_messages_basic():
    """Test context message building with basic character and session."""
    from src.models.character import Character
    from src.models.session import GameSession

    builder = PromptBuilder()

    # Create test data
    character = Character(
        id=1,
        name="测试角色",
        hp=10,
        san=50,
        max_san=99
    )
    session = GameSession(
        id="123e4567-e89b-12d3-a456-426614174000",
        name="测试会话",
        current_scene_name="旧书房",
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


@pytest.mark.asyncio
async def test_build_context_messages_with_events():
    """Test context message building with multiple recent events."""
    from src.models.character import Character
    from src.models.session import GameSession

    builder = PromptBuilder()

    character = Character(
        id=1,
        name="调查员",
        hp=8,
        san=45,
        max_san=99
    )
    session = GameSession(
        id="123e4567-e89b-12d3-a456-426614174000",
        name="测试会话",
        current_scene_name="图书馆",
        world_state={"leads": ["古老书籍", "神秘符号"]}
    )

    recent_events = [
        {"description": "你听到了奇怪的声音"},
        {"description": "你发现了一本旧书"},
        {"description": "你感觉有人在注视你"},
        {"description": "你看到了一个影子"},
        {"description": "你闻到了腐烂的味道"},
        {"description": "这是第六个事件，应该被忽略"}  # Should be limited to 5
    ]

    messages = await builder.build_context_messages(
        character=character,
        session=session,
        recent_events=recent_events,
        user_message="我想离开这里"
    )

    # Should have context, events (limited to 5), and user message
    assert len(messages) == 3
    assert "最近发生的事情" in messages[1]["content"]
    # Should only include first 5 events
    assert "第六个事件" not in messages[1]["content"]
    assert messages[-1]["content"] == "我想离开这里"


@pytest.mark.asyncio
async def test_build_context_messages_no_leads():
    """Test context building when no leads are discovered."""
    from src.models.character import Character
    from src.models.session import GameSession

    builder = PromptBuilder()

    character = Character(
        id=1,
        name="测试角色",
        hp=10,
        san=50,
        max_san=99
    )
    session = GameSession(
        id="123e4567-e89b-12d3-a456-426614174000",
        name="测试会话",
        current_scene_name="空房间",
        world_state={}  # No leads
    )

    messages = await builder.build_context_messages(
        character=character,
        session=session,
        recent_events=[],
        user_message="环顾四周"
    )

    # Should not mention leads when none exist
    context_content = messages[0]["content"]
    assert "线索" not in context_content
    assert "空房间" in context_content


@pytest.mark.asyncio
async def test_build_context_messages_no_events():
    """Test context building with no recent events."""
    from src.models.character import Character
    from src.models.session import GameSession

    builder = PromptBuilder()

    character = Character(
        id=1,
        name="测试角色",
        hp=10,
        san=50,
        max_san=99
    )
    session = GameSession(
        id="123e4567-e89b-12d3-a456-426614174000",
        name="测试会话",
        current_scene_name="走廊",
        world_state={"leads": ["门后的声音"]}
    )

    messages = await builder.build_context_messages(
        character=character,
        session=session,
        recent_events=[],  # No events
        user_message="继续前进"
    )

    # Should only have context and user message, no events block
    assert len(messages) == 2
    assert messages[0]["role"] == "user"
    assert messages[1]["role"] == "user"
    assert messages[1]["content"] == "继续前进"
