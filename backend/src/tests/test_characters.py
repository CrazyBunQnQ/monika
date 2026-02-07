"""Tests for character API endpoints."""
import pytest
from fastapi.testclient import TestClient

from src.tests.conftest import TestingSessionLocal
from src.models.user import User
from src.core.security import get_password_hash


@pytest.fixture
def auth_headers(client):
    """Create authentication headers with a test user."""
    # Register user
    client.post(
        "/auth/register",
        json={
            "username": "testuser",
            "email": "test@example.com",
            "password": "testpassword123",
        },
    )

    # Login to get token
    response = client.post(
        "/auth/login",
        json={
            "username": "testuser",
            "password": "testpassword123",
        },
    )
    token = response.json()["access_token"]
    return {"Authorization": f"Bearer {token}"}


class TestCharacterCreate:
    """Test character creation endpoint."""

    def test_create_character_success(self, client, auth_headers):
        """Should create a character successfully."""
        response = client.post(
            "/characters",
            json={
                "name": "Test Investigator",
                "age": 25,
                "gender": "Male",
                "occupation": "Private Investigator",
                "mental_illness": "",
                "backstory": "",
                "str": 50,
                "con": 60,
                "dex": 70,
                "app": 50,
                "pow": 60,
                "int": 70,
                "siz": 50,
                "edu": 80,
                "hp": 11,
                "mp": 12,
                "san": 60,
                "max_san": 60,
                "luck": 50,
            },
            headers=auth_headers,
        )
        assert response.status_code == 200
        data = response.json()
        assert data["name"] == "Test Investigator"
        assert data["age"] == 25
        assert data["str"] == 50
        assert "id" in data
        assert "owner_id" in data

    def test_create_character_without_auth(self, client):
        """Should fail without authentication."""
        response = client.post(
            "/characters",
            json={
                "name": "Test Investigator",
                "age": 25,
                "gender": "Male",
                "occupation": "Private Investigator",
                "str": 50,
                "con": 60,
                "dex": 70,
                "app": 50,
                "pow": 60,
                "int": 70,
                "siz": 50,
                "edu": 80,
                "hp": 11,
                "mp": 12,
                "san": 60,
                "max_san": 60,
                "luck": 50,
            },
        )
        assert response.status_code == 401

    def test_create_character_without_name(self, client, auth_headers):
        """Should fail when name is missing."""
        response = client.post(
            "/characters",
            json={
                # Missing name - name is required
            },
            headers=auth_headers,
        )
        assert response.status_code == 422

    def test_create_character_minimal(self, client, auth_headers):
        """Should create character with minimal fields."""
        response = client.post(
            "/characters",
            json={
                "name": "Minimal Character",
            },
            headers=auth_headers,
        )
        assert response.status_code == 200
        data = response.json()
        assert data["name"] == "Minimal Character"


class TestCharacterRead:
    """Test character read endpoints."""

    def test_get_character(self, client, auth_headers):
        """Should get a character by ID."""
        # Create character first
        create_response = client.post(
            "/characters",
            json={
                "name": "Get Test Character",
                "age": 30,
                "str": 50,
                "con": 50,
                "dex": 50,
                "app": 50,
                "pow": 50,
                "int": 50,
                "siz": 50,
                "edu": 50,
                "hp": 10,
                "mp": 10,
                "san": 50,
                "max_san": 50,
                "luck": 50,
            },
            headers=auth_headers,
        )
        char_id = create_response.json()["id"]

        # Get character
        response = client.get(f"/characters/{char_id}", headers=auth_headers)
        assert response.status_code == 200
        data = response.json()
        assert data["name"] == "Get Test Character"
        assert data["id"] == char_id

    def test_get_character_not_found(self, client, auth_headers):
        """Should return 404 for non-existent character."""
        response = client.get("/characters/99999", headers=auth_headers)
        assert response.status_code == 404

    def test_list_characters(self, client, auth_headers):
        """Should list all characters for user."""
        # Create two characters
        client.post(
            "/characters",
            json={"name": "Character 1", "str": 50, "con": 50, "dex": 50, "app": 50, "pow": 50, "int": 50, "siz": 50, "edu": 50, "hp": 10, "mp": 10, "san": 50, "max_san": 50, "luck": 50},
            headers=auth_headers,
        )
        client.post(
            "/characters",
            json={"name": "Character 2", "str": 60, "con": 60, "dex": 60, "app": 60, "pow": 60, "int": 60, "siz": 60, "edu": 60, "hp": 12, "mp": 12, "san": 60, "max_san": 60, "luck": 60},
            headers=auth_headers,
        )

        response = client.get("/characters", headers=auth_headers)
        assert response.status_code == 200
        data = response.json()
        assert len(data) >= 2

    def test_list_characters_empty(self, client):
        """Should return empty list when no characters."""
        # Login as new user
        client.post(
            "/auth/register",
            json={
                "username": "emptyuser",
                "email": "empty@example.com",
                "password": "password123",
            },
        )
        login_response = client.post(
            "/auth/login",
            json={
                "username": "emptyuser",
                "password": "password123",
            },
        )
        token = login_response.json()["access_token"]
        headers = {"Authorization": f"Bearer {token}"}

        response = client.get("/characters", headers=headers)
        assert response.status_code == 200
        assert response.json() == []


class TestCharacterUpdate:
    """Test character update endpoint."""

    def test_update_character(self, client, auth_headers):
        """Should update a character."""
        # Create character
        create_response = client.post(
            "/characters",
            json={
                "name": "Original Name",
                "str": 50,
                "con": 50,
                "dex": 50,
                "app": 50,
                "pow": 50,
                "int": 50,
                "siz": 50,
                "edu": 50,
                "hp": 10,
                "mp": 10,
                "san": 50,
                "max_san": 50,
                "luck": 50,
            },
            headers=auth_headers,
        )
        char_id = create_response.json()["id"]

        # Update character
        response = client.put(
            f"/characters/{char_id}",
            json={
                "name": "Updated Name",
                "age": 35,
                "str": 60,
                "con": 60,
                "dex": 60,
                "app": 60,
                "pow": 60,
                "int": 60,
                "siz": 60,
                "edu": 60,
                "hp": 12,
                "mp": 12,
                "san": 60,
                "max_san": 60,
                "luck": 60,
            },
            headers=auth_headers,
        )
        assert response.status_code == 200
        data = response.json()
        assert data["name"] == "Updated Name"
        assert data["age"] == 35
        assert data["str"] == 60

    def test_update_character_not_found(self, client, auth_headers):
        """Should return 404 for non-existent character."""
        response = client.put(
            "/characters/99999",
            json={
                "name": "Updated",
                "str": 50,
                "con": 50,
                "dex": 50,
                "app": 50,
                "pow": 50,
                "int": 50,
                "siz": 50,
                "edu": 50,
                "hp": 10,
                "mp": 10,
                "san": 50,
                "max_san": 50,
                "luck": 50,
            },
            headers=auth_headers,
        )
        assert response.status_code == 404


class TestCharacterDelete:
    """Test character delete endpoint."""

    def test_delete_character(self, client, auth_headers):
        """Should delete a character."""
        # Create character
        create_response = client.post(
            "/characters",
            json={
                "name": "Delete Me",
                "str": 50,
                "con": 50,
                "dex": 50,
                "app": 50,
                "pow": 50,
                "int": 50,
                "siz": 50,
                "edu": 50,
                "hp": 10,
                "mp": 10,
                "san": 50,
                "max_san": 50,
                "luck": 50,
            },
            headers=auth_headers,
        )
        char_id = create_response.json()["id"]

        # Delete character
        response = client.delete(f"/characters/{char_id}", headers=auth_headers)
        assert response.status_code == 200

        # Verify deleted
        get_response = client.get(f"/characters/{char_id}", headers=auth_headers)
        assert get_response.status_code == 404

    def test_delete_character_not_found(self, client, auth_headers):
        """Should return 404 for non-existent character."""
        response = client.delete("/characters/99999", headers=auth_headers)
        assert response.status_code == 404


class TestCharacterOwnership:
    """Test character ownership enforcement."""

    def test_cannot_access_other_user_character(self, client):
        """Should not allow accessing another user's character."""
        # Create first user and character
        client.post(
            "/auth/register",
            json={
                "username": "user1",
                "email": "user1@example.com",
                "password": "password123",
            },
        )
        login1 = client.post(
            "/auth/login",
            json={"username": "user1", "password": "password123"},
        )
        token1 = login1.json()["access_token"]
        headers1 = {"Authorization": f"Bearer {token1}"}

        create_response = client.post(
            "/characters",
            json={"name": "User1 Character", "str": 50, "con": 50, "dex": 50, "app": 50, "pow": 50, "int": 50, "siz": 50, "edu": 50, "hp": 10, "mp": 10, "san": 50, "max_san": 50, "luck": 50},
            headers=headers1,
        )
        char_id = create_response.json()["id"]

        # Create second user
        client.post(
            "/auth/register",
            json={
                "username": "user2",
                "email": "user2@example.com",
                "password": "password123",
            },
        )
        login2 = client.post(
            "/auth/login",
            json={"username": "user2", "password": "password123"},
        )
        token2 = login2.json()["access_token"]
        headers2 = {"Authorization": f"Bearer {token2}"}

        # Try to access user1's character
        response = client.get(f"/characters/{char_id}", headers=headers2)
        assert response.status_code == 404
