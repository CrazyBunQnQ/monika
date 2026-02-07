import pytest
from unittest.mock import AsyncMock, MagicMock, patch
from src.services.llm.openai import OpenAIProvider


@pytest.fixture
async def mock_openai_provider():
    """Fixture that provides an OpenAIProvider with a mocked AsyncOpenAI client."""
    with patch.dict("os.environ", {"OPENAI_API_KEY": "test-key", "OPENAI_MODEL": "gpt-4"}):
        provider = OpenAIProvider()

        # Mock the client's chat.completions.create method
        mock_create = AsyncMock()
        provider.client.chat.completions.create = mock_create

        yield provider, mock_create


@pytest.mark.asyncio
async def test_openai_provider_context_limit(mock_openai_provider):
    """Test that get_context_limit returns the correct limit."""
    provider, _ = mock_openai_provider
    limit = await provider.get_context_limit()
    assert limit == 8192


@pytest.mark.asyncio
async def test_openai_provider_stream_chat(mock_openai_provider):
    """Test that stream_chat properly calls the OpenAI API and yields chunks."""
    provider, mock_create = mock_openai_provider

    # Create mock stream chunks
    async def mock_stream():
        class MockChunk:
            def __init__(self, content):
                self.choices = [MagicMock()]
                self.choices[0].delta.content = content

        yield MockChunk('{"narrative": "test"}')
        yield MockChunk(' more content')

    # Make create return an async iterator
    mock_create.return_value = mock_stream()

    # Call stream_chat and collect chunks
    chunks = []
    messages = [{"role": "user", "content": "test"}]
    system_prompt = "You are a test system."

    async for chunk in provider.stream_chat(messages, system_prompt):
        chunks.append(chunk)

    # Verify the API was called correctly
    mock_create.assert_called_once_with(
        model="gpt-4",
        messages=[
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": "test"}
        ],
        stream=True,
        temperature=0.8,
        max_tokens=2000
    )

    # Verify chunks were yielded
    assert len(chunks) == 2
    assert chunks[0] == '{"narrative": "test"}'
    assert chunks[1] == ' more content'


@pytest.mark.asyncio
async def test_openai_provider_health_check_success(mock_openai_provider):
    """Test that health_check returns True when API call succeeds."""
    provider, mock_create = mock_openai_provider

    # Mock successful response
    mock_response = MagicMock()
    mock_create.return_value = mock_response

    result = await provider.health_check()

    # Verify the API was called correctly
    mock_create.assert_called_once_with(
        model="gpt-4",
        messages=[{"role": "user", "content": "test"}],
        max_tokens=1
    )

    assert result is True


@pytest.mark.asyncio
async def test_openai_provider_health_check_failure(mock_openai_provider):
    """Test that health_check returns False when API call fails."""
    provider, mock_create = mock_openai_provider

    # Mock API failure
    mock_create.side_effect = Exception("API error")

    result = await provider.health_check()

    assert result is False
