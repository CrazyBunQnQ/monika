"""Tests for authentication dependencies."""
import pytest
from fastapi import HTTPException, status
from fastapi.security import HTTPAuthorizationCredentials

from backend.api.dependencies import (
    get_current_user,
    get_optional_user,
)
from backend.core.security import create_access_token
from backend.models.user import User


class TestGetCurrentUser:
    """Tests for get_current_user dependency."""

    def test_get_current_user_valid_token(self, db_session, test_user):
        """Test getting current user with valid token."""
        token = create_access_token({"sub": test_user.user_id})
        credentials = HTTPAuthorizationCredentials(
            scheme="bearer",
            credentials=token
        )

        user = get_current_user(credentials, db_session)
        assert user is not None
        assert user.user_id == test_user.user_id

    def test_get_current_user_no_token(self, db_session):
        """Test getting current user with no token raises 401."""
        with pytest.raises(HTTPException) as exc_info:
            get_current_user(None, db_session)

        assert exc_info.value.status_code == status.HTTP_401_UNAUTHORIZED
        assert exc_info.value.detail == "Not authenticated"

    def test_get_current_user_invalid_token(self, db_session):
        """Test getting current user with invalid token raises 401."""
        credentials = HTTPAuthorizationCredentials(
            scheme="bearer",
            credentials="invalid.token.here"
        )

        with pytest.raises(HTTPException) as exc_info:
            get_current_user(credentials, db_session)

        assert exc_info.value.status_code == status.HTTP_401_UNAUTHORIZED
        assert exc_info.value.detail == "Could not validate credentials"

    def test_get_current_user_nonexistent_user(self, db_session):
        """Test getting current user with valid token but non-existent user raises 401."""
        token = create_access_token({"sub": "nonexistent_user_id"})
        credentials = HTTPAuthorizationCredentials(
            scheme="bearer",
            credentials=token
        )

        with pytest.raises(HTTPException) as exc_info:
            get_current_user(credentials, db_session)

        assert exc_info.value.status_code == status.HTTP_401_UNAUTHORIZED
        assert exc_info.value.detail == "User not found"


class TestGetOptionalUser:
    """Tests for get_optional_user dependency."""

    def test_get_optional_user_valid_token(self, db_session, test_user):
        """Test getting optional user with valid token returns user."""
        token = create_access_token({"sub": test_user.user_id})
        credentials = HTTPAuthorizationCredentials(
            scheme="bearer",
            credentials=token
        )

        user = get_optional_user(credentials, db_session)
        assert user is not None
        assert user.user_id == test_user.user_id

    def test_get_optional_user_no_token(self, db_session):
        """Test getting optional user with no token returns None."""
        user = get_optional_user(None, db_session)
        assert user is None

    def test_get_optional_user_invalid_token(self, db_session):
        """Test getting optional user with invalid token returns None."""
        credentials = HTTPAuthorizationCredentials(
            scheme="bearer",
            credentials="invalid.token.here"
        )

        user = get_optional_user(credentials, db_session)
        assert user is None

    def test_get_optional_user_nonexistent_user(self, db_session):
        """Test getting optional user with valid token but non-existent user returns None."""
        token = create_access_token({"sub": "nonexistent_user_id"})
        credentials = HTTPAuthorizationCredentials(
            scheme="bearer",
            credentials=token
        )

        user = get_optional_user(credentials, db_session)
        assert user is None
