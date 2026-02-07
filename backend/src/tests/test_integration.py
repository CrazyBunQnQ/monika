"""Integration tests for full API workflow."""
import pytest
from fastapi.testclient import TestClient


class TestAuthToCharacterWorkflow:
    """Test complete workflow: auth -> character -> game."""

    def test_full_workflow(self, client):
        """Test complete user -> character -> dice workflow."""
        # Register
        client.post(
            "/auth/register",
            json={
                "username": "workflowuser",
                "email": "workflow@example.com",
                "password": "testpassword123",
            },
        )
        # Login
        login_response = client.post(
            "/auth/login",
            json={
                "username": "workflowuser",
                "password": "testpassword123",
            },
        )
        token = login_response.json()["access_token"]
        client.headers.update({"Authorization": f"Bearer {token}"})

        # 1. Create a character
        create_response = client.post(
            "/characters",
            json={
                "name": "Test Investigator",
                "str": 60,
                "con": 70,
                "dex": 80,
                "app": 50,
                "pow": 60,
                "int": 70,
                "siz": 50,
                "edu": 80,
                "hp": 12,
                "mp": 12,
                "san": 60,
                "max_san": 60,
                "luck": 70,
            },
        )
        assert create_response.status_code == 200
        character = create_response.json()
        character_id = character["id"]
        assert character["name"] == "Test Investigator"
        assert character["str"] == 60

        # 2. Roll for the character
        roll_response = client.post(
            f"/game/characters/{character_id}/roll/luck",
            params={"roll_value": 50},
        )
        assert roll_response.status_code == 200
        roll_data = roll_response.json()
        assert roll_data["character_name"] == "Test Investigator"
        assert roll_data["skill_name"] == "luck"
        assert roll_data["skill_value"] == 70
        assert roll_data["roll"] == 50

        # 3. List characters
        list_response = client.get("/characters")
        assert list_response.status_code == 200
        characters = list_response.json()
        assert len(characters) >= 1

    def test_roll_without_auth(self, client):
        """Test that roll endpoint requires authentication."""
        response = client.post(
            "/game/roll",
            json={"skill": 50},
        )
        assert response.status_code == 401

    def test_character_skill_roll(self, client):
        """Test rolling a specific character skill."""
        # Register and login
        client.post(
            "/auth/register",
            json={
                "username": "dexuser",
                "email": "dex@example.com",
                "password": "password123",
            },
        )
        login_response = client.post(
            "/auth/login",
            json={"username": "dexuser", "password": "password123"},
        )
        token = login_response.json()["access_token"]
        client.headers.update({"Authorization": f"Bearer {token}"})

        # Create character
        create_response = client.post(
            "/characters",
            json={
                "name": "DEX Character",
                "dex": 85,
                "str": 50,
                "con": 50,
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
        )
        character = create_response.json()
        character_id = character["id"]

        # Roll DEX (should succeed with 85 skill)
        roll_response = client.post(
            f"/game/characters/{character_id}/roll/dex",
            params={"roll_value": 80},
        )
        assert roll_response.status_code == 200
        roll_data = roll_response.json()
        assert roll_data["success_level"] == "regular_success"

        # Roll DEX (should fail)
        roll_response = client.post(
            f"/game/characters/{character_id}/roll/dex",
            params={"roll_value": 90},
        )
        assert roll_response.status_code == 200
        roll_data = roll_response.json()
        assert roll_data["success_level"] == "failure"

    def test_roll_with_bonus_penalty(self, client):
        """Test rolling with bonus/penalty dice."""
        # Register and login
        client.post(
            "/auth/register",
            json={
                "username": "bonususer",
                "email": "bonus@example.com",
                "password": "password123",
            },
        )
        login_response = client.post(
            "/auth/login",
            json={"username": "bonususer", "password": "password123"},
        )
        token = login_response.json()["access_token"]
        client.headers.update({"Authorization": f"Bearer {token}"})

        # Create character
        create_response = client.post(
            "/characters",
            json={
                "name": "Bonus Character",
                "luck": 50,
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
            },
        )
        character = create_response.json()
        character_id = character["id"]

        # Roll with hard difficulty (bonus dice) - skill 50, roll 60 should succeed
        # Hard: effective skill is 50 * 2/3 = 33, so 60 > 33 = failure
        # Actually in CoC 7e, hard means roll 2d100 take lower, so we get bonus
        # Let's test with a roll that should succeed with hard
        roll_response = client.post(
            f"/game/characters/{character_id}/roll/luck",
            params={"roll_value": 20, "bonus_penalty": "hard"},
        )
        assert roll_response.status_code == 200
        roll_data = roll_response.json()
        # Hard roll: effective skill = 50 * 2/3 = 33 (floor)
        # Roll 20 <= 33 = success
        assert roll_data["success_level"] == "regular_success"


class TestGameHealth:
    """Test game API health."""

    def test_game_health(self, client):
        """Test game API health endpoint."""
        response = client.get("/game/health")
        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "ok"
        assert data["service"] == "game"

    def test_api_root(self, client):
        """Test API root endpoint."""
        response = client.get("/")
        assert response.status_code == 200
        assert "Monika API" in response.json()["message"]

    def test_api_health(self, client):
        """Test API health endpoint."""
        response = client.get("/health")
        assert response.status_code == 200
        assert response.json()["status"] == "ok"


class TestCharacterOwnership:
    """Test character ownership in game context."""

    def test_cannot_roll_other_user_character(self, client):
        """Test that users cannot roll other users' characters."""
        from src.main import app as fastapi_app

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
        client1 = TestClient(app=fastapi_app)
        client1.headers.update({"Authorization": f"Bearer {token1}"})

        create_response = client1.post(
            "/characters",
            json={"name": "User1 Character", "luck": 50, "str": 50, "con": 50, "dex": 50, "app": 50, "pow": 50, "int": 50, "siz": 50, "edu": 50, "hp": 10, "mp": 10, "san": 50, "max_san": 50},
        )
        character_id = create_response.json()["id"]

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
        client2 = TestClient(app=fastapi_app)
        client2.headers.update({"Authorization": f"Bearer {token2}"})

        # Try to roll user1's character
        roll_response = client2.post(
            f"/game/characters/{character_id}/roll/luck",
            params={"roll_value": 50},
        )
        assert roll_response.status_code == 404


class TestDiceAPIValidation:
    """Test dice API input validation."""

    def test_roll_invalid_skill(self, client):
        """Test that invalid skill values are rejected."""
        # Register and login
        client.post(
            "/auth/register",
            json={
                "username": "validationuser",
                "email": "validation@example.com",
                "password": "password123",
            },
        )
        login_response = client.post(
            "/auth/login",
            json={"username": "validationuser", "password": "password123"},
        )
        token = login_response.json()["access_token"]
        client.headers.update({"Authorization": f"Bearer {token}"})

        response = client.post(
            "/game/roll",
            json={"skill": 150},  # Invalid: > 100
        )
        assert response.status_code == 422

    def test_roll_invalid_bonus_penalty(self, client):
        """Test that invalid bonus_penalty is rejected."""
        # Register and login
        client.post(
            "/auth/register",
            json={
                "username": "validationuser2",
                "email": "validation2@example.com",
                "password": "password123",
            },
        )
        login_response = client.post(
            "/auth/login",
            json={"username": "validationuser2", "password": "password123"},
        )
        token = login_response.json()["access_token"]
        client.headers.update({"Authorization": f"Bearer {token}"})

        response = client.post(
            "/game/roll",
            json={"skill": 50, "bonus_penalty": "invalid"},
        )
        assert response.status_code == 400

    def test_roll_invalid_skill_name(self, client):
        """Test that invalid skill name is rejected."""
        # Register and login
        client.post(
            "/auth/register",
            json={
                "username": "validationuser3",
                "email": "validation3@example.com",
                "password": "password123",
            },
        )
        login_response = client.post(
            "/auth/login",
            json={"username": "validationuser3", "password": "password123"},
        )
        token = login_response.json()["access_token"]
        client.headers.update({"Authorization": f"Bearer {token}"})

        # Create character first
        create_response = client.post(
            "/characters",
            json={"name": "Test", "luck": 50, "str": 50, "con": 50, "dex": 50, "app": 50, "pow": 50, "int": 50, "siz": 50, "edu": 50, "hp": 10, "mp": 10, "san": 50, "max_san": 50},
        )
        character_id = create_response.json()["id"]

        response = client.post(
            f"/game/characters/{character_id}/roll/invalid_skill",
        )
        assert response.status_code == 400
