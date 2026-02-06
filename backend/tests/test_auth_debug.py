from backend.core.database import Base
from backend.models.user import User

print("Tables in Base.metadata:")
print(list(Base.metadata.tables.keys()))
print(f"User table: {User.__tablename__}")
