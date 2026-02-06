"""User Pydantic schemas."""
from pydantic import BaseModel, ConfigDict, EmailStr


class UserCreate(BaseModel):
    """Schema for creating a new user."""

    username: str
    email: EmailStr
    password: str


class UserLogin(BaseModel):
    """Schema for user login."""

    username: str
    password: str


class Token(BaseModel):
    """Schema for JWT token response."""

    access_token: str
    token_type: str = "bearer"


class TokenData(BaseModel):
    """Schema for token payload data."""

    username: str | None = None
    user_id: int | None = None


class UserResponse(BaseModel):
    """Schema for user response (without password)."""

    id: int
    username: str
    email: str

    model_config = ConfigDict(from_attributes=True)
