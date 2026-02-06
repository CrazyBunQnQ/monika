import pytest
from backend.models.user import User
from backend.schemas.user import UserBase, UserCreate, UserResponse, Token


def test_user_model_exists():
    # Test that User model can be imported
    assert User is not None
    assert User.__tablename__ == "users"


def test_user_schemas():
    # Test schema creation
    user_base = UserBase(username="test", email="test@example.com")
    assert user_base.username == "test"

    user_create = UserCreate(username="test", email="test@example.com", password="pass123")
    assert user_create.password == "pass123"

    # Test Token schema
    token = Token(access_token="abc", refresh_token="def")
    assert token.token_type == "bearer"
