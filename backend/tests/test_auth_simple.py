"""Simplified tests for authentication API routes."""
import pytest
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker, Session

from backend.core.database import Base
from backend.core.security import verify_password, get_password_hash
from backend.models.user import User
from backend.schemas.user import UserCreate


# Test database URL
TEST_DATABASE_URL = "sqlite:///:memory:"


@pytest.fixture(scope="function")
def test_db():
    """Create a test database session."""
    engine = create_engine(
        TEST_DATABASE_URL,
        connect_args={"check_same_thread": False},
    )
    Base.metadata.create_all(engine)
    
    SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)
    session = SessionLocal()
    yield session
    session.close()
    engine.dispose()


class TestRegisterUserLogic:
    """Tests for user registration logic without FastAPI."""

    def test_check_existing_username(self, test_db):
        """Test checking for existing username."""
        import uuid
        # Create an existing user
        existing_user = User(
            user_id=str(uuid.uuid4()),
            username="existinguser",
            email="existing@example.com",
            password_hash=get_password_hash("password123"),
            is_active=True,
        )
        test_db.add(existing_user)
        test_db.commit()

        # Check if username exists
        found_user = test_db.query(User).filter(User.username == "existinguser").first()
        assert found_user is not None
        assert found_user.username == "existinguser"

    def test_check_existing_email(self, test_db):
        """Test checking for existing email."""
        import uuid
        # Create an existing user
        existing_user = User(
            user_id=str(uuid.uuid4()),
            username="existinguser2",
            email="existing2@example.com",
            password_hash=get_password_hash("password123"),
            is_active=True,
        )
        test_db.add(existing_user)
        test_db.commit()

        # Check if email exists
        found_user = test_db.query(User).filter(User.email == "existing2@example.com").first()
        assert found_user is not None
        assert found_user.email == "existing2@example.com"

    def test_create_new_user(self, test_db):
        """Test creating a new user."""
        import uuid
        user_data = {
            "user_id": str(uuid.uuid4()),
            "username": "newuser",
            "email": "newuser@example.com",
            "password_hash": get_password_hash("password123"),
            "is_active": True,
        }

        new_user = User(**user_data)
        test_db.add(new_user)
        test_db.commit()
        test_db.refresh(new_user)

        assert new_user.user_id is not None
        assert new_user.username == "newuser"
        assert new_user.email == "newuser@example.com"
        assert verify_password("password123", new_user.password_hash)
