from typing import Optional

from pydantic import ConfigDict, Field
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    model_config = ConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )

    # Database
    DATABASE_URL: str = "postgresql+psycopg2://postgres:password@localhost:5432/monika"

    # Security
    SECRET_KEY: str = "change-this-secret-key-in-production"
    ALGORITHM: str = "HS256"
    ACCESS_TOKEN_EXPIRE_MINUTES: int = 30

    # Application
    DEBUG: bool = True

    # LLM Configuration
    llm_provider: str = Field(default="openai", description="LLM Provider: openai, claude, qwen")
    openai_api_key: Optional[str] = Field(default=None, description="OpenAI API Key")
    openai_model: str = Field(default="gpt-4", description="OpenAI Model")
    claude_api_key: Optional[str] = Field(default=None, description="Anthropic API Key")
    claude_model: str = Field(
        default="claude-3-sonnet-20240229", description="Claude Model"
    )

    # WebSocket Configuration
    ws_heartbeat_interval: int = Field(
        default=30, description="WebSocket heartbeat interval (seconds)"
    )
    ws_reconnect_max_attempts: int = Field(
        default=5, description="Max WebSocket reconnect attempts"
    )


settings = Settings()
