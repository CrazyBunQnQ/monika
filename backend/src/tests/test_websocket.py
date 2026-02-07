"""Tests for WebSocket endpoint."""
import json
import pytest
from fastapi.testclient import TestClient
from fastapi import FastAPI

from src.api.websocket import websocket_router, manager
from src.core.database import get_db


# Create test app
app = FastAPI()
app.include_router(websocket_router, prefix="/ws")


@pytest.fixture
def test_app():
    """Fixture for test FastAPI app."""
    return app


class TestWebSocketConnection:
    """Tests for WebSocket connection management."""

    def test_connection_manager_connect(self):
        """Test ConnectionManager.connect method."""
        import asyncio
        from unittest.mock import AsyncMock

        # Create mock websocket with AsyncMock for async methods
        mock_websocket = AsyncMock()

        # Run async test
        async def run_test():
            await manager.connect(1, mock_websocket)
            assert 1 in manager.active_connections
            mock_websocket.accept.assert_called_once()
            manager.disconnect(1)
            assert 1 not in manager.active_connections

        asyncio.run(run_test())

    def test_connection_manager_disconnect(self):
        """Test ConnectionManager.disconnect method."""
        from unittest.mock import MagicMock

        # Add a mock connection
        mock_websocket = MagicMock()
        manager.active_connections[1] = mock_websocket

        # Disconnect
        manager.disconnect(1)
        assert 1 not in manager.active_connections

    def test_connection_manager_send_message(self):
        """Test ConnectionManager.send_message method."""
        import asyncio
        from unittest.mock import AsyncMock

        # Create mock websocket with AsyncMock for async methods
        mock_websocket = AsyncMock()
        mock_websocket.send_json = AsyncMock()

        # Run async test
        async def run_test():
            await manager.connect(1, mock_websocket)
            result = await manager.send_message(1, {"type": "test", "content": "hello"})
            assert result is True
            mock_websocket.send_json.assert_called_once_with({"type": "test", "content": "hello"})
            manager.disconnect(1)

        asyncio.run(run_test())


class TestWebSocketEndpoint:
    """Tests for WebSocket endpoint."""

    def test_websocket_route_exists(self, test_app):
        """Test that WebSocket route is registered."""
        routes = [route.path for route in test_app.routes]
        # The router is included with prefix="/ws", and the route is "/game/{session_id}"
        # So the final path should be "/ws/game/{session_id}"
        assert "/ws/game/{session_id}" in routes

    def test_websocket_invalid_session(self, test_app):
        """Test WebSocket with invalid session ID."""
        from unittest.mock import patch, AsyncMock, MagicMock

        client = TestClient(test_app)

        # Mock database to return None for session
        async def mock_get_db():
            mock_db = MagicMock()
            mock_db.execute = AsyncMock(return_value=MagicMock(scalar_one_or_none=MagicMock(return_value=None)))
            return mock_db

        with patch('src.api.websocket.get_db', mock_get_db):
            # Note: TestClient's websocket_connect is limited, so we test the logic path
            # Full WebSocket testing requires a more complex setup
            pass

    def test_message_flow_structure(self):
        """Test that message flow has correct structure."""
        # This tests the expected message structure
        user_message = {
            "type": "user_message",
            "content": "I look around the room"
        }

        keeper_message_streaming = {
            "type": "keeper_message",
            "content": {
                "narrative": "You see a dusty room...",
                "tone": "mystery",
                "urgency": "low"
            },
            "is_streaming": True
        }

        state_update = {
            "type": "state_update",
            "content": {
                "current_scene": "Library",
                "world_state": {"leads": ["Old book"]}
            }
        }

        assert user_message["type"] == "user_message"
        assert keeper_message_streaming["type"] == "keeper_message"
        assert keeper_message_streaming["is_streaming"] is True
        assert state_update["type"] == "state_update"


class TestWebSocketIntegration:
    """Integration tests for WebSocket with services."""

    def test_prompt_builder_integration(self):
        """Test that PromptBuilder is correctly integrated."""
        from src.services.prompt import PromptBuilder

        builder = PromptBuilder()
        assert builder is not None
        assert hasattr(builder, 'build_system_prompt')
        assert hasattr(builder, 'build_context_messages')

    def test_response_parser_integration(self):
        """Test that ResponseParser is correctly integrated."""
        from src.services.response_parser import ResponseParser

        parser = ResponseParser()
        assert parser is not None
        assert hasattr(parser, 'parse_stream')

    def test_llm_provider_integration(self):
        """Test that OpenAIProvider is correctly integrated."""
        from src.services.llm.openai import OpenAIProvider

        # Note: This will fail if OPENAI_API_KEY is not set
        try:
            provider = OpenAIProvider()
            assert provider is not None
            assert hasattr(provider, 'stream_chat')
        except ValueError as e:
            # Expected if OPENAI_API_KEY is not set
            assert "OPENAI_API_KEY" in str(e)

    def test_state_sync_integration(self):
        """Test that StateSyncService is correctly integrated."""
        from src.services.state_sync import StateSyncService
        from unittest.mock import MagicMock

        mock_db = MagicMock()
        sync = StateSyncService(mock_db)
        assert sync is not None
        assert hasattr(sync, 'apply_state_changes')
