"""Tests for authentication."""
from datetime import datetime

import pytest
from fastapi.testclient import TestClient

from src.tests.conftest import TestingSessionLocal
from src.models.user import User
from src.core.security import get_password_hash, verify_password


class TestPasswordHashing:
    """Test password hashing functions."""

    def test_hash_password(self):
        """Should hash password correctly."""
        password = "testpassword123"
        hashed = get_password_hash(password)
        assert hashed != password
        assert len(hashed) > 50  # bcrypt hash is long

    def test_verify_password_correct(self):
        """Should verify correct password."""
        password = "testpassword123"
        hashed = get_password_hash(password)
        assert verify_password(password, hashed) is True

    def test_verify_password_incorrect(self):
        """Should reject incorrect password."""
        password = "testpassword123"
        hashed = get_password_hash(password)
        assert verify_password("wrongpassword", hashed) is False


class TestUserCreate:
    """Test user creation in database."""

    def test_create_user(self, test_db):
        """Should be able to create a user in database."""
        hashed = get_password_hash("password123")
        user = User(
            username="newuser",
            email="new@example.com",
            hashed_password=hashed,
        )
        test_db.add(user)
        test_db.commit()

        assert user.id is not None
        assert user.username == "newuser"
        assert user.email == "new@example.com"
        assert user.hashed_password == hashed

    def test_create_duplicate_username_fails(self, test_db):
        """Should fail when creating user with duplicate username."""
        hashed = get_password_hash("password123")
        user1 = User(username="duplicate", email="user1@example.com", hashed_password=hashed)
        test_db.add(user1)
        test_db.commit()

        user2 = User(username="duplicate", email="user2@example.com", hashed_password=hashed)
        test_db.add(user2)

        with pytest.raises(Exception):
            test_db.commit()

    def test_create_duplicate_email_fails(self, test_db):
        """Should fail when creating user with duplicate email."""
        hashed = get_password_hash("password123")
        user1 = User(username="user1", email="duplicate@example.com", hashed_password=hashed)
        test_db.add(user1)
        test_db.commit()

        user2 = User(username="user2", email="duplicate@example.com", hashed_password=hashed)
        test_db.add(user2)

        with pytest.raises(Exception):
            test_db.commit()


class TestAuthSchemas:
    """Test auth Pydantic schemas."""

    def test_user_create_schema(self):
        """Test UserCreate schema validation."""
        from src.schemas.user import UserCreate

        data = {
            "username": "testuser",
            "email": "test@example.com",
            "password": "securepassword123",
        }
        user = UserCreate(**data)
        assert user.username == "testuser"
        assert user.email == "test@example.com"
        assert user.password == "securepassword123"

    def test_user_login_schema(self):
        """Test UserLogin schema validation."""
        from src.schemas.user import UserLogin

        data = {"username": "testuser", "password": "password123"}
        login = UserLogin(**data)
        assert login.username == "testuser"
        assert login.password == "password123"

    def test_token_schema(self):
        """Test Token schema."""
        from src.schemas.user import Token

        data = {"access_token": "test_token", "token_type": "bearer"}
        token = Token(**data)
        assert token.access_token == "test_token"
        assert token.token_type == "bearer"
