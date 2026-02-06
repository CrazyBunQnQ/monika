from sqlalchemy import create_engine, select
from backend.core.database import Base
from backend.models.user import User

# Test database URL
TEST_DATABASE_URL = "sqlite:///:memory:"

# Create engine
engine = create_engine(
    TEST_DATABASE_URL,
    connect_args={"check_same_thread": False},
)

# Create tables
Base.metadata.create_all(engine)

# Check if table exists
from sqlalchemy import inspect
inspector = inspect(engine)
tables = inspector.get_table_names()
print(f"Tables created: {tables}")
print(f"Expected 'users' in tables: {'users' in tables}")

# Try to query
from sqlalchemy.orm import sessionmaker
SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)
session = SessionLocal()

# Count users (should be 0)
result = session.execute(select(User).where(User.username == "test"))
print(f"Query successful: {result}")
session.close()
