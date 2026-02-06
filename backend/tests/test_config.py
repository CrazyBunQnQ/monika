"""
Test configuration settings.

This test validates that the Settings class can be instantiated
and has the expected attributes.
"""
import os
from unittest.mock import patch

import pytest


def test_settings_has_database_url():
    """Test that settings can access DATABASE_URL."""
    # Mock environment variables
    with patch.dict(os.environ, {
        "JWT_SECRET_KEY": "test-secret-key",
        "OPENAI_API_KEY": "test-openai-key"
    }):
        from core.config import settings

        assert hasattr(settings, "DATABASE_URL")
        assert settings.DATABASE_URL == "postgresql://monika:monika_pass@localhost:5432/monika"


def test_settings_has_redis_url():
    """Test that settings can access REDIS_URL."""
    with patch.dict(os.environ, {
        "JWT_SECRET_KEY": "test-secret-key",
        "OPENAI_API_KEY": "test-openai-key"
    }):
        from core.config import settings

        assert hasattr(settings, "REDIS_URL")
        assert settings.REDIS_URL == "redis://localhost:6379/0"


def test_settings_has_jwt_config():
    """Test that settings has JWT configuration."""
    with patch.dict(os.environ, {
        "JWT_SECRET_KEY": "test-secret-key",
        "OPENAI_API_KEY": "test-openai-key"
    }):
        from core.config import settings

        assert hasattr(settings, "JWT_SECRET_KEY")
        assert hasattr(settings, "JWT_ALGORITHM")
        assert hasattr(settings, "ACCESS_TOKEN_EXPIRE_MINUTES")
        assert settings.JWT_ALGORITHM == "HS256"
        assert settings.ACCESS_TOKEN_EXPIRE_MINUTES == 30


def test_settings_has_openai_key():
    """Test that settings has OpenAI API key."""
    with patch.dict(os.environ, {
        "JWT_SECRET_KEY": "test-secret-key",
        "OPENAI_API_KEY": "test-openai-key"
    }):
        from core.config import settings

        assert hasattr(settings, "OPENAI_API_KEY")
