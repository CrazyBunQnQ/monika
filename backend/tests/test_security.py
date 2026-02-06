"""Tests for security module."""
import pytest
from datetime import datetime, timedelta, timezone
from backend.core.security import (
    verify_password,
    get_password_hash,
    create_access_token,
    create_refresh_token,
    decode_token,
)


class TestPasswordHashing:
    """Tests for password hashing and verification."""

    def test_get_password_hash(self):
        """Test that password hashing returns a hash."""
        password = "test_password_123"
        hashed = get_password_hash(password)

        assert hashed is not None
        assert hashed != password
        assert isinstance(hashed, str)
        assert len(hashed) > 0

    def test_verify_password_correct(self):
        """Test that correct password verifies successfully."""
        password = "test_password_123"
        hashed = get_password_hash(password)

        assert verify_password(password, hashed) is True

    def test_verify_password_incorrect(self):
        """Test that incorrect password fails verification."""
        password = "test_password_123"
        wrong_password = "wrong_password"
        hashed = get_password_hash(password)

        assert verify_password(wrong_password, hashed) is False

    def test_hash_is_different_for_same_password(self):
        """Test that bcrypt generates different hashes for same password (salt)."""
        password = "test_password_123"
        hash1 = get_password_hash(password)
        hash2 = get_password_hash(password)

        assert hash1 != hash2
        assert verify_password(password, hash1) is True
        assert verify_password(password, hash2) is True


class TestJWTTokenCreation:
    """Tests for JWT token creation."""

    def test_create_access_token(self):
        """Test access token creation."""
        data = {"sub": "user123"}
        token = create_access_token(data)

        assert token is not None
        assert isinstance(token, str)
        assert len(token) > 0

    def test_create_access_token_with_expiry(self):
        """Test access token creation with custom expiry."""
        data = {"sub": "user123"}
        expires_delta = timedelta(minutes=60)
        token = create_access_token(data, expires_delta)

        assert token is not None
        assert isinstance(token, str)

    def test_create_refresh_token(self):
        """Test refresh token creation."""
        data = {"sub": "user123"}
        token = create_refresh_token(data)

        assert token is not None
        assert isinstance(token, str)
        assert len(token) > 0


class TestJWTTokenDecoding:
    """Tests for JWT token decoding."""

    def test_decode_valid_token(self):
        """Test decoding a valid token."""
        data = {"sub": "user123", "username": "testuser"}
        token = create_access_token(data)
        decoded = decode_token(token)

        assert decoded is not None
        assert decoded["sub"] == "user123"
        assert decoded["username"] == "testuser"

    def test_decode_invalid_token(self):
        """Test decoding an invalid token."""
        invalid_token = "invalid.token.string"
        decoded = decode_token(invalid_token)

        assert decoded is None

    def test_decode_expired_token(self):
        """Test decoding an expired token."""
        data = {"sub": "user123"}
        # Create token that expired 1 hour ago
        expires_delta = timedelta(hours=-1)
        token = create_access_token(data, expires_delta)
        decoded = decode_token(token)

        assert decoded is None

    def test_decode_access_token_contains_expiration(self):
        """Test that decoded access token contains expiration."""
        data = {"sub": "user123"}
        token = create_access_token(data)
        decoded = decode_token(token)

        assert decoded is not None
        assert "exp" in decoded

    def test_decode_refresh_token_expiration(self):
        """Test that refresh token has 7 day expiration."""
        data = {"sub": "user123"}
        token = create_refresh_token(data)
        decoded = decode_token(token)

        assert decoded is not None
        assert "exp" in decoded

        # Check that expiration is approximately 7 days from now
        exp_timestamp = decoded["exp"]
        exp_datetime = datetime.fromtimestamp(exp_timestamp, timezone.utc)
        now = datetime.now(timezone.utc)
        time_diff = exp_datetime - now

        # Should be approximately 7 days (give or take a few seconds)
        assert 6.9 * 24 * 3600 <= time_diff.total_seconds() <= 7.1 * 24 * 3600
