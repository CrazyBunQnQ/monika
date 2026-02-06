"""Tests for authentication API routes."""
import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker, Session

# Import these first to ensure Base is properly set up
from backend.core.database import Base, get_db
from backend.core.security import verify_password, get_password_hash
from backend.models.user import User
from backend.schemas.user import UserCreate
from backend.api.auth import router as auth_router


# Test database URL
TEST_DATABASE_URL = "sqlite:///:memory:"


@pytest.fixture(scope="function")
def test_engine():
    """Create a test database engine."""
    engine = create_engine(
        TEST_DATABASE_URL,
        connect_args={"check_same_thread": False},
    )
    Base.metadata.create_all(engine)
    yield engine
    Base.metadata.drop_all(engine)
    engine.dispose()


@pytest.fixture(scope="function")
def test_db(test_engine):
    """Create a test database session."""
    SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=test_engine)
    session = SessionLocal()
    yield session
    session.close()


@pytest.fixture(scope="function")
def client(test_db):
    """Create a test client with database override."""
    # Create test app
    app = FastAPI()
    app.include_router(auth_router)

    # Override the get_db dependency
    def override_get_db():
        yield test_db

    app.dependency_overrides[get_db] = override_get_db
    
    # Use sync client
    test_client = TestClient(app, raise_server_exceptions=False)
    yield test_client
    app.dependency_overrides.clear()


class TestRegisterUser:
    """Tests for user registration endpoint."""

    def test_register_user_success(self, client):
        """Test successful user registration."""
        user_data = {
            "username": "newuser",
            "email": "newuser@example.com",
            "password": "securepassword123"
        }

        response = client.post("/auth/register", json=user_data)

        print(f"Response status: {response.status_code}")
        print(f"Response content: {response.content}")
        
        assert response.status_code == 201
        data = response.json()
        assert data["username"] == "newuser"
        assert data["email"] == "newuser@example.com"
        assert "user_id" in data
        assert data["is_active"] is True
        assert "password_hash" not in data  # Should not expose password hash

    def test_register_user_duplicate_username(self, client, test_db):
        """Test registration with duplicate username fails."""
        # Create an existing user
        import uuid
        existing_user = User(
            user_id=str(uuid.uuid4()),
            username="existinguser",
            email="existing@example.com",
            password_hash=get_password_hash("password123"),
            is_active=True,
        )
        test_db.add(existing_user)
        test_db.commit()

        user_data = {
            "username": "existinguser",  # Already exists
            "email": "different@example.com",
            "password": "password123"
        }

        response = client.post("/auth/register", json=user_data)

        assert response.status_code == 400
        assert "already taken" in response.json()["detail"]

    def test_register_user_duplicate_email(self, client, test_db):
        """Test registration with duplicate email fails."""
        # Create an existing user
        import uuid
        existing_user = User(
            user_id=str(uuid.uuid4()),
            username="existinguser2",
            email="existing2@example.com",
            password_hash=get_password_hash("password123"),
            is_active=True,
        )
        test_db.add(existing_user)
        test_db.commit()

        user_data = {
            "username": "differentuser",
            "email": "existing2@example.com",  # Already exists
            "password": "password123"
        }

        response = client.post("/auth/register", json=user_data)

        assert response.status_code == 400
        assert "already registered" in response.json()["detail"]

    def test_register_user_password_hashed(self, client, test_db):
        """Test that password is properly hashed."""
        user_data = {
            "username": "hashuser",
            "email": "hashuser@example.com",
            "password": "plaintext123"
        }

        response = client.post("/auth/register", json=user_data)
        assert response.status_code == 201

        # Verify password is hashed in database
        user = test_db.query(User).filter(User.username == "hashuser").first()
        assert user is not None
        assert user.password_hash != "plaintext123"
        assert verify_password("plaintext123", user.password_hash) is True

    def test_register_user_creates_uuid(self, client):
        """Test that user is created with UUID."""
        user_data = {
            "username": "uuiduser",
            "email": "uuiduser@example.com",
            "password": "password123"
        }

        response = client.post("/auth/register", json=user_data)
        assert response.status_code == 201

        data = response.json()
        import uuid
        # Verify user_id is a valid UUID
        uuid.UUID(data["user_id"])  # Will raise if invalid
