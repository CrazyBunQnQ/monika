"""Tests for auth API endpoints."""
import pytest
from fastapi.testclient import TestClient


class TestAuthEndpoints:
    """Test authentication API endpoints."""

    def test_register_success(self, client):
        """Should register a new user successfully."""
        response = client.post(
            "/auth/register",
            json={
                "username": "newuser",
                "email": "new@example.com",
                "password": "securepassword123",
            },
        )
        assert response.status_code == 200
        data = response.json()
        assert data["username"] == "newuser"
        assert data["email"] == "new@example.com"
        assert "id" in data
        assert "hashed_password" not in data

    def test_register_duplicate_username(self, client):
        """Should fail when username already exists."""
        # First registration
        client.post(
            "/auth/register",
            json={
                "username": "duplicate",
                "email": "first@example.com",
                "password": "password123",
            },
        )
        # Second registration with same username
        response = client.post(
            "/auth/register",
            json={
                "username": "duplicate",
                "email": "second@example.com",
                "password": "password123",
            },
        )
        assert response.status_code == 400

    def test_register_duplicate_email(self, client):
        """Should fail when email already exists."""
        # First registration
        client.post(
            "/auth/register",
            json={
                "username": "firstuser",
                "email": "same@example.com",
                "password": "password123",
            },
        )
        # Second registration with same email
        response = client.post(
            "/auth/register",
            json={
                "username": "seconduser",
                "email": "same@example.com",
                "password": "password123",
            },
        )
        assert response.status_code == 400

    def test_register_invalid_email(self, client):
        """Should fail with invalid email format."""
        response = client.post(
            "/auth/register",
            json={
                "username": "testuser",
                "email": "not-an-email",
                "password": "password123",
            },
        )
        assert response.status_code == 422

    def test_register_short_password(self, client):
        """Should fail with short password."""
        response = client.post(
            "/auth/register",
            json={
                "username": "testuser",
                "email": "test@example.com",
                "password": "short",
            },
        )
        assert response.status_code == 422

    def test_login_success(self, client):
        """Should login successfully with correct credentials."""
        # Register first
        client.post(
            "/auth/register",
            json={
                "username": "loginuser",
                "email": "login@example.com",
                "password": "correctpassword",
            },
        )
        # Login
        response = client.post(
            "/auth/login",
            json={
                "username": "loginuser",
                "password": "correctpassword",
            },
        )
        assert response.status_code == 200
        data = response.json()
        assert "access_token" in data
        assert data["token_type"] == "bearer"

    def test_login_wrong_password(self, client):
        """Should fail with wrong password."""
        # Register first
        client.post(
            "/auth/login",
            json={
                "username": "testuser",
                "password": "correctpassword",
            },
        )

        # Register the user first
        client.post(
            "/auth/register",
            json={
                "username": "testuser",
                "email": "test@example.com",
                "password": "correctpassword",
            },
        )

        # Login with wrong password
        response = client.post(
            "/auth/login",
            json={
                "username": "testuser",
                "password": "wrongpassword",
            },
        )
        assert response.status_code == 401

    def test_login_nonexistent_user(self, client):
        """Should fail for non-existent user."""
        response = client.post(
            "/auth/login",
            json={
                "username": "nonexistent",
                "password": "password123",
            },
        )
        assert response.status_code == 401


class TestAuthSchemas:
    """Test auth schema validation in API."""

    def test_register_missing_username(self, client):
        """Should fail when username is missing."""
        response = client.post(
            "/auth/register",
            json={
                "email": "test@example.com",
                "password": "password123",
            },
        )
        assert response.status_code == 422

    def test_register_missing_password(self, client):
        """Should fail when password is missing."""
        response = client.post(
            "/auth/register",
            json={
                "username": "testuser",
                "email": "test@example.com",
            },
        )
        assert response.status_code == 422

    def test_login_missing_username(self, client):
        """Should fail when username is missing in login."""
        response = client.post(
            "/auth/login",
            json={
                "password": "password123",
            },
        )
        assert response.status_code == 422

    def test_login_missing_password(self, client):
        """Should fail when password is missing in login."""
        response = client.post(
            "/auth/login",
            json={
                "username": "testuser",
            },
        )
        assert response.status_code == 422
