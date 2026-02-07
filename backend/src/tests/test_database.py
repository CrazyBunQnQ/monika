"""Tests for database models."""
from datetime import datetime

import pytest
from sqlalchemy import inspect

from src.tests.conftest import engine
from src.models.user import User
from src.models.character import Character


class TestUserModel:
    """Test User model."""

    def test_user_table_exists(self, client):
        """User table should exist in database."""
        inspector = inspect(engine)
        tables = inspector.get_table_names()
        assert "users" in tables

    def test_user_columns(self, client):
        """User table should have required columns."""
        inspector = inspect(engine)
        columns = {col["name"] for col in inspector.get_columns("users")}
        expected = {"id", "username", "email", "hashed_password", "created_at", "updated_at"}
        assert expected.issubset(columns)

    def test_user_create(self, client):
        """Should be able to create a user."""
        user = User(
            username="testuser",
            email="test@example.com",
            hashed_password="hashedpassword123",
        )
        # User should not be saved yet
        assert user.id is None


class TestCharacterModel:
    """Test Character model."""

    def test_character_table_exists(self, client):
        """Character table should exist in database."""
        inspector = inspect(engine)
        tables = inspector.get_table_names()
        assert "characters" in tables

    def test_character_columns(self, client):
        """Character table should have required columns."""
        inspector = inspect(engine)
        columns = {col["name"] for col in inspector.get_columns("characters")}
        expected = {
            "id",
            "owner_id",
            "name",
            "age",
            "gender",
            "occupation",
            "mental_illness",
            "backstory",
            "str",
            "con",
            "dex",
            "app",
            "pow",
            "int",
            "siz",
            "edu",
            "hp",
            "mp",
            "san",
            "max_san",
            "luck",
            "created_at",
            "updated_at",
        }
        assert expected.issubset(columns)

    def test_character_create(self, client):
        """Should be able to create a character."""
        character = Character(
            owner_id=1,
            name="Test Investigator",
            age=25,
            gender="Male",
            occupation="Private Investigator",
            mental_illness="",
            backstory="",
            str=50,
            con=60,
            dex=70,
            app=50,
            pow=60,
            int=70,
            siz=50,
            edu=80,
            hp=11,
            mp=12,
            san=60,
            max_san=60,
            luck=50,
        )
        # Character should not be saved yet
        assert character.id is None


class TestTableRelationships:
    """Test table relationships."""

    def test_character_has_owner_id(self, client):
        """Character should have owner_id foreign key column."""
        inspector = inspect(engine)
        columns = {col["name"] for col in inspector.get_columns("characters")}
        # SQLite doesn't enforce FK constraints, but column should exist
        assert "owner_id" in columns
