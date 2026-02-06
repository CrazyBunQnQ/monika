# Monika CoC TRPG Platform - Backend

Backend implementation for the Monika Call of Cthulhu TRPG platform using FastAPI, PostgreSQL, and Redis.

## Setup Instructions

### Prerequisites
- Python 3.9+
- Docker and Docker Compose
- pip

### Installation

1. Start PostgreSQL and Redis containers:
```bash
docker-compose up -d
```

2. Install Python dependencies:
```bash
pip install -r requirements.txt
```

3. Configure environment variables:
```bash
cp .env.example .env
# Edit .env with your actual values
```

4. Run database migrations:
```bash
cd backend
alembic upgrade head
```

### Project Structure

```
backend/
├── alembic/              # Database migrations
│   └── versions/         # Migration files
├── core/                 # Core functionality
│   ├── config.py         # Application settings
│   └── database.py       # Database configuration
├── tests/                # Test files
└── requirements.txt      # Python dependencies
```

### Environment Variables

- `DATABASE_URL`: PostgreSQL connection string
- `REDIS_URL`: Redis connection string
- `JWT_SECRET_KEY`: Secret key for JWT token signing
- `JWT_ALGORITHM`: JWT algorithm (default: HS256)
- `ACCESS_TOKEN_EXPIRE_MINUTES`: JWT token expiration time
- `OPENAI_API_KEY`: OpenAI API key for LLM features

### Running Tests

```bash
pytest
```

### Database Migrations

Create a new migration:
```bash
alembic revision --autogenerate -m "description"
```

Apply migrations:
```bash
alembic upgrade head
```

### Docker Services

- PostgreSQL 15 on port 5432
- Redis 7 on port 6379
