"""Test configuration and fixtures."""
import pytest
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker, Session

# Import these before anything else to avoid table conflicts
from backend.core.database import Base
from backend.models.user import User
from backend.core.security import get_password_hash


# Test database URL (use SQLite for tests)
TEST_DATABASE_URL = "sqlite:///:memory:"


@pytest.fixture(scope="function")
def engine():
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
def db_session(engine) -> Session:
    """Create a test database session."""
    SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)

    session = SessionLocal()
    yield session
    session.close()


@pytest.fixture(scope="function")
def test_user(db_session: Session) -> User:
    """Create a test user in the database."""
    import uuid

    user = User(
        user_id=str(uuid.uuid4()),
        username="testuser",
        email="testuser@example.com",
        password_hash=get_password_hash("testpassword123"),
        is_active=True,
    )

    db_session.add(user)
    db_session.commit()
    db_session.refresh(user)

    return user
