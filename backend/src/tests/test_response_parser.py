import pytest
from src.services.response_parser import ResponseParser
from src.schemas.llm_response import LLMResponse


@pytest.mark.asyncio
async def test_parse_valid_json_response():
    """Test parsing a complete valid JSON response"""
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
    """Test parsing JSON that arrives in multiple chunks"""
    parser = ResponseParser()

    async def mock_stream():
        yield '{"narr'
        yield 'ative": "test"'
        yield ', "tone": "horror"}'

    responses = []
    async for response in parser.parse_stream(mock_stream()):
        responses.append(response)

    assert len(responses) == 1
    assert responses[0].narrative == "test"


@pytest.mark.asyncio
async def test_fallback_on_invalid_json():
    """Test fallback response when JSON is invalid"""
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
    """Test JSON extraction from text with extra content"""
    parser = ResponseParser()

    text = 'Some text before {"narrative": "test"} some text after'
    result = parser._extract_json(text)
    assert result == '{"narrative": "test"}'

    assert parser._extract_json("no json here") is None


@pytest.mark.asyncio
async def test_parse_full_llm_response():
    """Test parsing a complete LLMResponse with all fields"""
    parser = ResponseParser()

    async def mock_stream():
        yield '''{
            "narrative": "You enter a dark room",
            "tone": "horror",
            "urgency": "high",
            "state_changes": {
                "current_scene": "dark_room"
            },
            "suggestions": ["Search the room", "Turn on light"],
            "audio_cue": "creaking_door",
            "requires_roll": true
        }'''

    responses = []
    async for response in parser.parse_stream(mock_stream()):
        responses.append(response)

    assert len(responses) == 1
    assert responses[0].narrative == "You enter a dark room"
    assert responses[0].tone == "horror"
    assert responses[0].urgency == "high"
    assert responses[0].state_changes.current_scene == "dark_room"
    assert len(responses[0].suggestions) == 2
    assert responses[0].audio_cue == "creaking_door"
    assert responses[0].requires_roll is True


@pytest.mark.asyncio
async def test_minimal_response():
    """Test parsing minimal response with only required field"""
    parser = ResponseParser()

    async def mock_stream():
        yield '{"narrative": "Simple response"}'

    responses = []
    async for response in parser.parse_stream(mock_stream()):
        responses.append(response)

    assert len(responses) == 1
    assert responses[0].narrative == "Simple response"
    # Check defaults
    assert responses[0].tone == "calm"
    assert responses[0].urgency == "low"
    assert responses[0].requires_roll is False


@pytest.mark.asyncio
async def test_multiple_responses_in_stream():
    """Test parsing multiple complete JSON responses from stream"""
    parser = ResponseParser()

    async def mock_stream():
        yield '{"narrative": "First response"}'
        yield '{"narrative": "Second response", "tone": "action"}'

    responses = []
    async for response in parser.parse_stream(mock_stream()):
        responses.append(response)

    assert len(responses) == 2
    assert responses[0].narrative == "First response"
    assert responses[1].narrative == "Second response"
    assert responses[1].tone == "action"


@pytest.mark.asyncio
async def test_partial_json_buffering():
    """Test that incomplete JSON is buffered until complete"""
    parser = ResponseParser()

    async def mock_stream():
        yield '{"narrative": "'
        yield 'incomplete'
        yield '"}'

    responses = []
    async for response in parser.parse_stream(mock_stream()):
        responses.append(response)

    assert len(responses) == 1
    assert responses[0].narrative == "incomplete"


@pytest.mark.asyncio
async def test_state_changes_with_world_state():
    """Test state changes with world_state dictionary"""
    parser = ResponseParser()

    async def mock_stream():
        yield '''{
            "narrative": "The door is locked",
            "state_changes": {
                "world_state": {"door_unlocked": false, "keys_found": 0}
            }
        }'''

    responses = []
    async for response in parser.parse_stream(mock_stream()):
        responses.append(response)

    assert len(responses) == 1
    assert responses[0].state_changes.world_state["door_unlocked"] is False
    assert responses[0].state_changes.world_state["keys_found"] == 0


@pytest.mark.asyncio
async def test_empty_stream():
    """Test handling of empty stream"""
    parser = ResponseParser()

    async def mock_stream():
        return
        yield

    responses = []
    async for response in parser.parse_stream(mock_stream()):
        responses.append(response)

    assert len(responses) == 0


@pytest.mark.asyncio
async def test_nested_json_extraction():
    """Test JSON extraction with nested objects"""
    parser = ResponseParser()

    text = 'Prefix {"narrative": "test", "state_changes": {"current_scene": "room"}} suffix'
    result = parser._extract_json(text)
    assert result == '{"narrative": "test", "state_changes": {"current_scene": "room"}}'
