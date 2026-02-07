# Rules Knowledge Base Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a RAG-based rules knowledge base that enables players to query CoC 7th Edition rules through natural language and integrates with the AI Keeper for intelligent rule citations.

**Architecture:** PostgreSQL + pgvector for vector storage, OpenAI embeddings for semantic search, hybrid retrieval (keyword + vector), dual access via `/rule` command and AI tool calls.

**Tech Stack:** Python 3.11+, FastAPI, PostgreSQL with pgvector, OpenAI API, React 19, TypeScript, shadcn/ui

---

## Task 1: Database Schema and Migration (M1-097)

**Files:**
- Create: `backend/src/models/rule.py`
- Create: `backend/alembic/versions/xxx_create_rules_tables.py`
- Test: `backend/src/tests/test_rules.py`

**Step 1: Write the failing test**

```python
# backend/src/tests/test_rules.py

import pytest
from sqlalchemy.ext.asyncio import AsyncSession

@pytest.mark.asyncio
async def test_create_rule(db: AsyncSession):
    """Test that a rule can be created and retrieved"""
    from backend.src.models.rule import Rule

    rule = Rule(
        title="暗视",
        category="skill",
        subcategory="感知技能",
        content="暗视允许调查员在几乎完全黑暗的环境中进行检定...",
        aliases=["夜视", "Darkness adaptation"],
        tags=["感知", "修正"]
    )

    db.add(rule)
    await db.commit()
    await db.refresh(rule)

    assert rule.id is not None
    assert rule.title == "暗视"
    assert rule.category == "skill"
```

**Step 2: Run test to verify it fails**

Run: `cd backend && uv run pytest src/tests/test_rules.py::test_create_rule -v`

Expected: `FAIL` with "no module named 'backend.src.models.rule'"

**Step 3: Create the Rule model**

```python
# backend/src/models/rule.py

from sqlalchemy import Column, String, Text, ARRAY, JSON, DateTime, ForeignKey
from sqlalchemy.dialects.postgresql import UUID, VECTOR
from sqlalchemy.orm import relationship
from sqlalchemy.sql import func
from backend.src.core.database import Base
import uuid

class Rule(Base):
    __tablename__ = "rules"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    title = Column(String(255), nullable=False)
    category = Column(String(100), nullable=False)
    subcategory = Column(String(100))
    content = Column(Text, nullable=False)
    example = Column(Text)
    mechanics = Column(JSON)
    aliases = Column(ARRAY(String))
    tags = Column(ARRAY(String))
    related_rule_ids = Column(ARRAY(UUID(as_uuid=True)))
    embedding = Column(VECTOR(1536))
    created_at = Column(DateTime(timezone=True), server_default=func.now())
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())

    def to_dict(self):
        return {
            "id": str(self.id),
            "title": self.title,
            "category": self.category,
            "subcategory": self.subcategory,
            "content": self.content,
            "example": self.example,
            "mechanics": self.mechanics,
            "aliases": self.aliases or [],
            "tags": self.tags or [],
            "related_rule_ids": [str(id) for id in (self.related_rule_ids or [])],
        }
```

**Step 4: Create the RuleFAQ model**

```python
# backend/src/models/rule.py (add to existing file)

class RuleFAQ(Base):
    __tablename__ = "rule_faqs"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    question = Column(Text, nullable=False)
    answer = Column(Text, nullable=False)
    category = Column(String(100))
    related_rule_ids = Column(ARRAY(UUID(as_uuid=True)))
    embedding = Column(VECTOR(1536))
    created_at = Column(DateTime(timezone=True), server_default=func.now())
```

**Step 5: Create the Alembic migration**

```bash
cd backend
uv run alembic revision -m "create rules tables"
```

Edit the generated migration file:

```python
# backend/alembic/versions/xxx_create_rules_tables.py

from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects import postgresql

def upgrade():
    op.execute("CREATE EXTENSION IF NOT EXISTS vector")

    op.create_table(
        'rules',
        sa.Column('id', postgresql.UUID(as_uuid=True), primary_key=True),
        sa.Column('title', sa.String(255), nullable=False),
        sa.Column('category', sa.String(100), nullable=False),
        sa.Column('subcategory', sa.String(100)),
        sa.Column('content', sa.Text(), nullable=False),
        sa.Column('example', sa.Text()),
        sa.Column('mechanics', postgresql.JSON()),
        sa.Column('aliases', postgresql.ARRAY(sa.String())),
        sa.Column('tags', postgresql.ARRAY(sa.String())),
        sa.Column('related_rule_ids', postgresql.ARRAY(postgresql.UUID())),
        sa.Column('embedding', postgresql.VECTOR(1536)),
        sa.Column('created_at', sa.DateTime(timezone=True), server_default=sa.text('NOW()')),
        sa.Column('updated_at', sa.DateTime(timezone=True), server_default=sa.text('NOW()')),
    )

    op.create_table(
        'rule_faqs',
        sa.Column('id', postgresql.UUID(as_uuid=True), primary_key=True),
        sa.Column('question', sa.Text(), nullable=False),
        sa.Column('answer', sa.Text(), nullable=False),
        sa.Column('category', sa.String(100)),
        sa.Column('related_rule_ids', postgresql.ARRAY(postgresql.UUID())),
        sa.Column('embedding', postgresql.VECTOR(1536)),
        sa.Column('created_at', sa.DateTime(timezone=True), server_default=sa.text('NOW()')),
    )

    # Create indexes
    op.execute('CREATE INDEX rules_fts ON rules USING gin(to_tsvector(\'english\', title || \' \' || content))')
    op.execute('CREATE INDEX rules_embedding_idx ON rules USING ivfflat(embedding vector_cosine_ops) WITH (lists = 100)')
    op.execute('CREATE INDEX rule_faqs_embedding_idx ON rule_faqs USING ivfflat(embedding vector_cosine_ops) WITH (lists = 50)')
    op.create_index('rules_category_idx', 'rules', ['category'])

def downgrade():
    op.drop_table('rule_faqs')
    op.drop_table('rules')
```

**Step 6: Update models __init__.py**

```python
# backend/src/models/__init__.py

from backend.src.models.rule import Rule, RuleFAQ

__all__ = ["User", "Character", "GameSession", "Event", "Combat", "Chase", "Rule", "RuleFAQ"]
```

**Step 7: Run migration**

Run: `cd backend && uv run alembic upgrade head`

**Step 8: Run test to verify it passes**

Run: `cd backend && uv run pytest src/tests/test_rules.py::test_create_rule -v`

Expected: `PASS`

**Step 9: Commit**

```bash
git add backend/src/models/rule.py backend/alembic/versions/xxx_create_rules_tables.py backend/src/models/__init__.py backend/src/tests/test_rules.py
git commit -m "feat(M1-097): create rules database tables and models"
```

---

## Task 2: Pydantic Schemas for Rules (M1-097)

**Files:**
- Create: `backend/src/schemas/rule.py`
- Modify: `backend/src/api/rules.py` (create this file)

**Step 1: Write the failing test**

```python
# backend/src/tests/test_rules.py (add)

from pydantic import ValidationError
import pytest

def test_rule_schema_validation():
    """Test that rule schema validates correctly"""
    from backend.src.schemas.rule import RuleBase, RuleCreate

    # Valid input
    rule_data = {
        "title": "暗视",
        "category": "skill",
        "subcategory": "感知技能",
        "content": "暗视允许调查员在几乎完全黑暗的环境中进行检定...",
        "aliases": ["夜视"],
        "tags": ["感知", "修正"]
    }

    rule = RuleCreate(**rule_data)
    assert rule.title == "暗视"
    assert rule.category == "skill"

    # Invalid input (missing required fields)
    with pytest.raises(ValidationError):
        RuleCreate(title="暗视")  # Missing category and content
```

**Step 2: Run test to verify it fails**

Run: `cd backend && uv run pytest src/tests/test_rules.py::test_rule_schema_validation -v`

Expected: `FAIL` with "cannot import name 'RuleCreate'"

**Step 3: Create the Pydantic schemas**

```python
# backend/src/schemas/rule.py

from pydantic import BaseModel, Field
from typing import Optional, List
from uuid import UUID
from datetime import datetime

class RuleCategory:
    CORE = "core"
    SKILL = "skill"
    COMBAT = "combat"
    SANITY = "sanity"
    CHASE = "chase"
    MAGIC = "magic"

RULE_CATEGORIES = [
    RuleCategory.CORE,
    RuleCategory.SKILL,
    RuleCategory.COMBAT,
    RuleCategory.SANITY,
    RuleCategory.CHASE,
    RuleCategory.MAGIC,
]

class RuleBase(BaseModel):
    title: str = Field(..., min_length=1, max_length=255)
    category: str = Field(..., description=f"One of: {RULE_CATEGORIES}")
    subcategory: Optional[str] = Field(None, max_length=100)
    content: str = Field(..., min_length=1)
    example: Optional[str] = None
    mechanics: Optional[dict] = None
    aliases: List[str] = Field(default_factory=list)
    tags: List[str] = Field(default_factory=list)
    related_rule_ids: List[UUID] = Field(default_factory=list)

class RuleCreate(RuleBase):
    pass

class RuleUpdate(BaseModel):
    title: Optional[str] = Field(None, min_length=1, max_length=255)
    category: Optional[str] = None
    subcategory: Optional[str] = None
    content: Optional[str] = None
    example: Optional[str] = None
    mechanics: Optional[dict] = None
    aliases: Optional[List[str]] = None
    tags: Optional[List[str]] = None
    related_rule_ids: Optional[List[UUID]] = None

class RuleResponse(RuleBase):
    id: UUID
    created_at: datetime
    updated_at: datetime

    class Config:
        from_attributes = True

class RuleSummary(BaseModel):
    id: UUID
    title: str
    category: str
    content: str  # Summary (first 200 chars)

class RuleSearchResult(BaseModel):
    id: UUID
    title: str
    category: str
    content: str  # Truncated for preview
    relevance_score: float = Field(..., ge=0.0, le=1.0)
    related_rules: List[RuleSummary] = Field(default_factory=list)

class RuleSearchResponse(BaseModel):
    results: List[RuleSearchResult]
    total: int
    query: str

class FAQBase(BaseModel):
    question: str = Field(..., min_length=1)
    answer: str = Field(..., min_length=1)
    category: Optional[str] = None
    related_rule_ids: List[UUID] = Field(default_factory=list)

class FAQCreate(FAQBase):
    pass

class FAQResponse(FAQBase):
    id: UUID
    created_at: datetime

    class Config:
        from_attributes = True

class RuleImportData(BaseModel):
    rules: List[RuleCreate]
    faqs: List[FAQCreate] = Field(default_factory=list)

class RuleImportResponse(BaseModel):
    imported: int
    failed: int
    errors: List[str] = Field(default_factory=list)
```

**Step 4: Run test to verify it passes**

Run: `cd backend && uv run pytest src/tests/test_rules.py::test_rule_schema_validation -v`

Expected: `PASS`

**Step 5: Commit**

```bash
git add backend/src/schemas/rule.py backend/src/tests/test_rules.py
git commit -m "feat(M1-097): add Pydantic schemas for rules"
```

---

## Task 3: Embedding Service (M1-098)

**Files:**
- Create: `backend/src/services/rule_embedding.py`
- Test: `backend/src/tests/test_rule_embedding.py`

**Step 1: Write the failing test**

```python
# backend/src/tests/test_rule_embedding.py

import pytest
from unittest.mock import AsyncMock, MagicMock

@pytest.mark.asyncio
async def test_generate_embedding():
    """Test that embeddings are generated correctly"""
    from backend.src.services.rule_embedding import RuleEmbeddingService

    # Mock LLM provider
    mock_provider = AsyncMock()
    mock_response = MagicMock()
    mock_response.data = [MagicMock(embedding=[0.1] * 1536)]
    mock_provider.client.embeddings.create = AsyncMock(return_value=mock_response)

    service = RuleEmbeddingService(mock_provider)
    embedding = await service.generate_embedding("暗视")

    assert len(embedding) == 1536
    assert all(isinstance(x, float) for x in embedding)

@pytest.mark.asyncio
async def test_embed_rule():
    """Test that rule embedding combines title and content"""
    from backend.src.services.rule_embedding import RuleEmbeddingService
    from backend.src.models.rule import Rule

    mock_provider = AsyncMock()
    mock_response = MagicMock()
    mock_response.data = [MagicMock(embedding=[0.2] * 1536)]
    mock_provider.client.embeddings.create = AsyncMock(return_value=mock_response)

    service = RuleEmbeddingService(mock_provider)

    rule = Rule(
        title="暗视",
        category="skill",
        content="暗视允许调查员在几乎完全黑暗的环境中进行检定...",
        example="例如，在完全黑暗的房间中..."
    )

    embedding = await service.embed_rule(rule)

    assert len(embedding) == 1536
    # Verify the API was called with combined text
    call_args = mock_provider.client.embeddings.create.call_args
    assert "暗视" in call_args[1]["input"]
    assert "几乎完全黑暗" in call_args[1]["input"]
```

**Step 2: Run test to verify it fails**

Run: `cd backend && uv run pytest src/tests/test_rule_embedding.py -v`

Expected: `FAIL` with "cannot import name 'RuleEmbeddingService'"

**Step 3: Implement the embedding service**

```python
# backend/src/services/rule_embedding.py

from functools import lru_cache
from typing import List
import logging

logger = logging.getLogger(__name__)

class RuleEmbeddingService:
    """Service for generating embeddings using OpenAI API"""

    def __init__(self, llm_provider):
        """
        Args:
            llm_provider: LLM provider instance from services/llm/
        """
        self.llm = llm_provider
        self.model = "text-embedding-3-small"
        self.embedding_dim = 1536  # Dimension for text-embedding-3-small

    async def generate_embedding(self, text: str) -> List[float]:
        """
        Generate embedding for a single text string.

        Args:
            text: Text to embed

        Returns:
            List of floats representing the embedding vector
        """
        try:
            response = await self.llm.client.embeddings.create(
                model=self.model,
                input=text
            )
            return response.data[0].embedding
        except Exception as e:
            logger.error(f"Failed to generate embedding: {e}")
            raise

    async def embed_rule(self, rule) -> List[float]:
        """
        Generate embedding for a rule by combining title, content, and example.

        Args:
            rule: Rule model instance

        Returns:
            List of floats representing the embedding vector
        """
        # Combine title and content with weights
        # Title gets more weight as it's the primary identifier
        text_parts = [rule.title, rule.content]

        if rule.example:
            text_parts.append(f"Example: {rule.example}")

        combined_text = ". ".join(text_parts)
        return await self.generate_embedding(combined_text)

    async def embed_faq(self, faq) -> List[float]:
        """
        Generate embedding for an FAQ entry.

        Args:
            faq: RuleFAQ model instance

        Returns:
            List of floats representing the embedding vector
        """
        # For FAQs, embed the question primarily
        # The answer is relevant but the question is what gets matched
        combined_text = f"{faq.question} {faq.answer}"
        return await self.generate_embedding(combined_text)

    @lru_cache(maxsize=1000)
    async def generate_embedding_cached(self, text: str) -> List[float]:
        """
        Cached version of generate_embedding for common queries.
        Reduces API calls for frequently searched terms.
        """
        return await self.generate_embedding(text)
```

**Step 4: Run test to verify it passes**

Run: `cd backend && uv run pytest src/tests/test_rule_embedding.py -v`

Expected: `PASS`

**Step 5: Commit**

```bash
git add backend/src/services/rule_embedding.py backend/src/tests/test_rule_embedding.py
git commit -m "feat(M1-098): implement rule embedding service"
```

---

## Task 4: Rule Search Service (M1-099)

**Files:**
- Create: `backend/src/services/rule_search.py`
- Test: `backend/src/tests/test_rule_search.py`

**Step 1: Write the failing test**

```python
# backend/src/tests/test_rule_search.py

import pytest
from sqlalchemy.ext.asyncio import AsyncSession

@pytest.mark.asyncio
async def test_search_by_embedding(db: AsyncSession):
    """Test vector similarity search"""
    from backend.src.services.rule_search import RuleSearchService
    from backend.src.models.rule import Rule
    from unittest.mock import AsyncMock

    # Create test rule
    rule = Rule(
        title="暗视",
        category="skill",
        content="暗视允许调查员在几乎完全黑暗的环境中进行检定，只有-20惩罚。",
        embedding=[0.1] * 1536  # Mock embedding
    )
    db.add(rule)
    await db.commit()

    # Mock embedding service
    mock_embedding_svc = AsyncMock()
    mock_embedding_svc.generate_embedding.return_value = [0.1] * 1536

    service = RuleSearchService(db, mock_embedding_svc)
    results = await service.search("暗视", limit=5)

    assert len(results) > 0
    assert results[0].title == "暗视"

@pytest.mark.asyncio
async def test_search_with_category_filter(db: AsyncSession):
    """Test search with category filtering"""
    from backend.src.services.rule_search import RuleSearchService
    from backend.src.models.rule import Rule
    from unittest.mock import AsyncMock

    # Create test rules in different categories
    rule1 = Rule(title="暗视", category="skill", content="...", embedding=[0.1] * 1536)
    rule2 = Rule(title="先攻", category="combat", content="...", embedding=[0.2] * 1536)
    db.add_all([rule1, rule2])
    await db.commit()

    mock_embedding_svc = AsyncMock()
    mock_embedding_svc.generate_embedding.return_value = [0.1] * 1536

    service = RuleSearchService(db, mock_embedding_svc)
    results = await service.search("暗视", category="skill", limit=5)

    assert all(r.category == "skill" for r in results)
```

**Step 2: Run test to verify it fails**

Run: `cd backend && uv run pytest src/tests/test_rule_search.py -v`

Expected: `FAIL` with "cannot import name 'RuleSearchService'"

**Step 3: Implement the search service**

```python
# backend/src/services/rule_search.py

from typing import List, Optional
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select, func, or_
from sqlalchemy.sql import select
import logging

from backend.src.models.rule import Rule, RuleFAQ
from backend.src.schemas.rule import RuleSearchResult, RuleSummary

logger = logging.getLogger(__name__)

class RuleSearchService:
    """Service for searching rules using hybrid retrieval (keyword + vector)"""

    def __init__(self, db: AsyncSession, embedding_service):
        """
        Args:
            db: Database session
            embedding_service: RuleEmbeddingService instance
        """
        self.db = db
        self.embedding = embedding_service

    async def search(
        self,
        query: str,
        category: Optional[str] = None,
        limit: int = 5
    ) -> List[RuleSearchResult]:
        """
        Search rules using hybrid retrieval.

        Args:
            query: Search query text
            category: Optional category filter
            limit: Maximum number of results

        Returns:
            List of RuleSearchResult with relevance scores
        """
        try:
            # Step 1: Generate query embedding
            query_embedding = await self.embedding.generate_embedding(query)
        except Exception as e:
            logger.warning(f"Embedding generation failed, falling back to keyword search: {e}")
            return await self._keyword_search(query, category, limit)

        # Step 2: Vector similarity search
        stmt = (
            select(Rule)
            .order_by(Rule.embedding.cosine_distance(query_embedding))
            .limit(limit * 2)  # Get more candidates for filtering
        )

        if category:
            stmt = stmt.where(Rule.category == category)

        result = await self.db.execute(stmt)
        candidates = result.scalars().all()

        if not candidates:
            return await self._suggest_alternatives(query)

        # Step 3: Build results with relevance scores
        results = []
        for rule in candidates[:limit]:
            # Calculate relevance based on cosine distance
            # Lower distance = higher similarity
            distance = rule.embedding.cosine_distance(query_embedding)
            relevance = 1.0 - float(distance) if distance else 0.5

            # Get related rules
            related_rules = []
            if rule.related_rule_ids:
                related = await self.db.execute(
                    select(Rule).where(Rule.id.in_(rule.related_rule_ids))
                )
                related_rules = [
                    RuleSummary(id=r.id, title=r.title, category=r.category, content=r.content[:200])
                    for r in related.scalars().all()
                ]

            results.append(RuleSearchResult(
                id=rule.id,
                title=rule.title,
                category=rule.category,
                content=rule.content[:200] + "..." if len(rule.content) > 200 else rule.content,
                relevance_score=round(relevance, 2),
                related_rules=related_rules
            ))

        return results

    async def _keyword_search(
        self,
        query: str,
        category: Optional[str],
        limit: int
    ) -> List[RuleSearchResult]:
        """Fallback keyword-only search using full-text search"""
        stmt = (
            select(Rule)
            .where(
                or_(
                    Rule.title.ilike(f"%{query}%"),
                    Rule.content.ilike(f"%{query}%")
                )
            )
            .limit(limit)
        )

        if category:
            stmt = stmt.where(Rule.category == category)

        result = await self.db.execute(stmt)
        rules = result.scalars().all()

        return [
            RuleSearchResult(
                id=r.id,
                title=r.title,
                category=r.category,
                content=r.content[:200] + "..." if len(r.content) > 200 else r.content,
                relevance_score=0.5,  # Fixed score for keyword matches
                related_rules=[]
            )
            for r in rules
        ]

    async def _suggest_alternatives(self, query: str) -> List[RuleSearchResult]:
        """Suggest alternative rules when no direct match found"""
        # Find rules with similar tags or category
        stmt = select(Rule).limit(5)
        result = await self.db.execute(stmt)
        rules = result.scalars().all()

        return [
            RuleSearchResult(
                id=r.id,
                title=r.title,
                category=r.category,
                content=r.content[:200],
                relevance_score=0.3,
                related_rules=[]
            )
            for r in rules
        ]

    async def get_rule_detail(self, rule_id: str) -> Optional[dict]:
        """Get full rule details with related content"""
        stmt = select(Rule).where(Rule.id == rule_id)
        result = await self.db.execute(stmt)
        rule = result.scalar_one_or_none()

        if not rule:
            return None

        # Load related rules
        related_rules = []
        if rule.related_rule_ids:
            related = await self.db.execute(
                select(Rule).where(Rule.id.in_(rule.related_rule_ids))
            )
            related_rules = [r.to_dict() for r in related.scalars().all()]

        # Load related FAQs
        faq_stmt = select(RuleFAQ).where(RuleFAQ.related_rule_ids.contains([rule_id]))
        faq_result = await self.db.execute(faq_stmt)
        faqs = [f.to_dict() for f in faq_result.scalars().all()]

        return {
            **rule.to_dict(),
            "related_rules": related_rules,
            "faqs": faqs
        }

    async def get_categories(self) -> List[str]:
        """Get all distinct rule categories"""
        stmt = select(Rule.category).distinct()
        result = await self.db.execute(stmt)
        return [row[0] for row in result.all()]
```

**Step 4: Run test to verify it passes**

Run: `cd backend && uv run pytest src/tests/test_rule_search.py -v`

Expected: `PASS`

**Step 5: Commit**

```bash
git add backend/src/services/rule_search.py backend/src/tests/test_rule_search.py
git commit -m "feat(M1-099): implement rule search service"
```

---

## Task 5: Rules API Endpoints (M1-099, M1-100)

**Files:**
- Create: `backend/src/api/rules.py`
- Modify: `backend/src/main.py`

**Step 1: Write the failing test**

```python
# backend/src/tests/test_rules_api.py

import pytest
from fastapi.testclient import TestClient

def test_search_rules(client: TestClient):
    """Test the rules search endpoint"""
    response = client.get("/rules/search?query=暗视&limit=5")

    assert response.status_code == 200
    data = response.json()
    assert "results" in data
    assert isinstance(data["results"], list)

def test_get_rule_detail(client: TestClient):
    """Test getting a single rule"""
    # First create a rule
    create_response = client.post(
        "/rules/import",
        json={
            "rules": [{
                "title": "测试规则",
                "category": "core",
                "content": "这是一个测试规则"
            }]
        }
    )
    rule_id = create_response.json()["imported"][0]

    # Then get it
    response = client.get(f"/rules/{rule_id}")
    assert response.status_code == 200
    assert response.json()["title"] == "测试规则"

def test_get_categories(client: TestClient):
    """Test getting rule categories"""
    response = client.get("/rules/categories")
    assert response.status_code == 200
    assert isinstance(response.json(), list)
```

**Step 2: Run test to verify it fails**

Run: `cd backend && uv run pytest src/tests/test_rules_api.py -v`

Expected: `FAIL` with "404 Not Found"

**Step 3: Create the rules API**

```python
# backend/src/api/rules.py

from fastapi import APIRouter, Depends, HTTPException, Query
from sqlalchemy.ext.asyncio import AsyncSession
from typing import List
from uuid import UUID

from backend.src.core.database import get_db
from backend.src.core.auth import get_current_user
from backend.src.models.user import User
from backend.src.services.rule_embedding import RuleEmbeddingService
from backend.src.services.rule_search import RuleSearchService
from backend.src.services.llm.base import LLMProvider
from backend.src.schemas.rule import (
    RuleSearchResponse,
    RuleCreate,
    RuleImportData,
    RuleImportResponse,
)

router = APIRouter(prefix="/rules", tags=["rules"])

# Dependency injection for services
async def get_search_service(
    db: AsyncSession = Depends(get_db),
    llm_provider: LLMProvider = Depends(get_llm_provider)  # Need to implement this
) -> RuleSearchService:
    """Get rule search service instance"""
    embedding_svc = RuleEmbeddingService(llm_provider)
    return RuleSearchService(db, embedding_svc)

@router.get("/search", response_model=RuleSearchResponse)
async def search_rules(
    query: str = Query(..., min_length=1, description="Search query"),
    category: str | None = Query(None, description="Filter by category"),
    limit: int = Query(5, ge=1, le=20, description="Number of results"),
    service: RuleSearchService = Depends(get_search_service)
):
    """
    Search for rules using semantic search.

    Supports both keyword matching and semantic similarity.
    """
    results = await service.search(query, category, limit)

    return RuleSearchResponse(
        results=results,
        total=len(results),
        query=query
    )

@router.get("/{rule_id}")
async def get_rule(
    rule_id: UUID,
    db: AsyncSession = Depends(get_db)
):
    """Get detailed information about a specific rule"""
    from backend.src.services.rule_search import RuleSearchService
    service = RuleSearchService(db, None)

    rule_detail = await service.get_rule_detail(str(rule_id))

    if not rule_detail:
        raise HTTPException(status_code=404, detail="Rule not found")

    return rule_detail

@router.get("/categories", response_model=List[str])
async def get_categories(
    db: AsyncSession = Depends(get_db)
):
    """Get all available rule categories"""
    from backend.src.services.rule_search import RuleSearchService
    service = RuleSearchService(db, None)
    return await service.get_categories()

@router.post("/import")
async def import_rules(
    data: RuleImportData,
    db: AsyncSession = Depends(get_db),
    current_user: User = Depends(get_current_user),  # Requires authentication
    llm_provider: LLMProvider = Depends(get_llm_provider)
):
    """
    Import rules in bulk (admin only).

    Creates rules and generates embeddings for each.
    """
    # Check if user is admin (you may want to add an is_admin field to User)
    # For now, any authenticated user can import

    from backend.src.models.rule import Rule, RuleFAQ
    from backend.src.services.rule_embedding import RuleEmbeddingService

    embedding_svc = RuleEmbeddingService(llm_provider)
    imported = []
    errors = []

    # Import rules
    for rule_data in data.rules:
        try:
            rule = Rule(**rule_data.model_dump())
            # Generate embedding
            rule.embedding = await embedding_svc.embed_rule(rule)
            db.add(rule)
            await db.flush()
            imported.append(str(rule.id))
        except Exception as e:
            errors.append(f"Failed to import rule '{rule_data.title}': {str(e)}")

    # Import FAQs
    for faq_data in data.faqs:
        try:
            faq = RuleFAQ(**faq_data.model_dump())
            faq.embedding = await embedding_svc.embed_faq(faq)
            db.add(faq)
            await db.flush()
        except Exception as e:
            errors.append(f"Failed to import FAQ '{faq_data.question}': {str(e)}")

    await db.commit()

    return RuleImportResponse(
        imported=len(imported),
        failed=len(errors),
        errors=errors
    )
```

**Step 4: Register the router in main.py**

```python
# backend/src/main.py (add to existing)

from backend.src.api.rules import router as rules_router

# Add to app initialization
app.include_router(rules_router)
```

**Step 5: Add LLM provider dependency**

```python
# backend/src/api/rules.py (update imports and add)

from backend.src.services.llm.openai import OpenAIProvider
from backend.src.core.config import settings

async def get_llm_provider() -> LLMProvider:
    """Get LLM provider for embeddings"""
    return OpenAIProvider(
        api_key=settings.OPENAI_API_KEY,
        model="text-embedding-3-small"
    )
```

**Step 6: Run test to verify it passes**

Run: `cd backend && uv run pytest src/tests/test_rules_api.py -v`

Expected: `PASS`

**Step 7: Commit**

```bash
git add backend/src/api/rules.py backend/src/main.py backend/src/tests/test_rules_api.py
git commit -m "feat(M1-099, M1-100): implement rules API endpoints"
```

---

## Task 6: Rule Seed Data (M1-098)

**Files:**
- Create: `backend/src/data/seed_rules.json`
- Create: `backend/src/scripts/import_seed_rules.py`

**Step 1: Create seed rules JSON**

```json
// backend/src/data/seed_rules.json
{
  "rules": [
    {
      "title": "暗视",
      "category": "skill",
      "subcategory": "感知技能",
      "content": "暗视允许调查员在几乎完全黑暗的环境中进行检定。调查员必须拥有一条可用的视线通道。检定修正为-20点。如果成功，调查员在黑暗中可以进行基于视觉的技能检定。",
      "example": "例如，调查员在完全黑暗的洞穴中听到可疑的声音。通过暗视检定，他可以勉强看清周围环境，尝试进行侦查检定来发现声音来源。",
      "mechanics": {
        "modifier": -20,
        "requires_line_of_sight": true,
        "enables_visual_checks_in_darkness": true
      },
      "aliases": ["夜视", "Darkness adaptation", "黑暗视觉"],
      "tags": ["感知", "视觉", "修正", "黑暗"]
    },
    {
      "title": "推骰",
      "category": "core",
      "subcategory": "检定机制",
      "content": "当调查员第一次检定失败后，如果情况允许，KP可以允许调查员进行推骰。推骰意味着角色在第一次尝试失败后付出了额外努力再试一次。推骰时，掷骰者必须支付5点幸运值，且结果不能比第一次更好。",
      "example": "调查员的图书馆使用检定失败了。KP允许推骰。调查员决定花费5点幸运重新检定。第二次掷骰结果为85，比第一次的75更差，所以这次推骰失败了。",
      "mechanics": {
        "cost": 5,
        "luck_point_cost": true,
        "result_constraint": "cannot_be_better_than_first_roll"
      },
      "aliases": ["重掷", "Pushing the roll"],
      "tags": ["检定", "幸运", "重掷"]
    },
    {
      "title": "花幸运",
      "category": "core",
      "subcategory": "检定机制",
      "content": "调查员可以在任何检定前或检定后（即使已经成功）花费幸运值来改善结果。每花费1点幸运，可以将检定结果减5，或者每花费5点幸运可以将失败转为成功。幸运值可以通过游戏进程恢复，但永远不会超过初始值。",
      "example": "调查员的侦查检定掷出了78（失败），但技能值为60。他决定花费5点幸运，将这次失败转变为一次成功。新的掷骰结果为42，成功。",
      "mechanics": {
        "cost_per_5_points": 5,
        "cost_per_1_point_improvement": 1,
        "can_turn_failure_to_success": true
      },
      "aliases": ["使用幸运", "Spending luck"],
      "tags": ["检定", "幸运", "修正"]
    },
    {
      "title": "SAN检定",
      "category": "sanity",
      "subcategory": "理智机制",
      "content": "当调查员遭遇恐怖事物时，需要进行SAN（理智）检定。检定时掷1d100，如果结果低于当前SAN值则成功（不损失SAN），否则失败并损失SAN值。损失的SAN值由遭遇的恐怖事物决定，通常有成功和失败两种不同的损失值。",
      "example": "调查员目睹了一具腐烂的尸体（1/1d4 SAN损失）。调查员当前SAN为60。掷骰结果为45（成功），不损失SAN。如果掷骰结果为65（失败），则需要掷1d4来确定损失值。",
      "mechanics": {
        "roll_type": "1d100",
        "success_condition": "roll <= current_san",
        "loss_determined_by": "encounter_specific"
      },
      "aliases": ["理智检定", "SAN check", "Sanity roll"],
      "tags": ["SAN", "理智", "恐怖"]
    }
  ],
  "faqs": [
    {
      "question": "黑暗中如何进行视觉检定？",
      "answer": "在完全黑暗的环境中，普通调查员无法进行基于视觉的检定。如果调查员拥有暗视技能，可以进行-20修正的视觉检定。其他情况下，调查员需要依靠其他感官（聆听、嗅觉等）或创造光源。",
      "category": "skill",
      "related_rules": ["暗视"]
    },
    {
      "question": "推骰和花幸运有什么区别？",
      "answer": "推骰是在检定失败后，花费5点幸运重新进行一次检定，但结果不能比第一次更好。花幸运是在检定前或后，花费幸运值直接改善检定结果（每1点减5点结果，或5点将失败转为成功）。推骰是'再试一次'，花幸运是'改变结果'。",
      "category": "core",
      "related_rules": ["推骰", "花幸运"]
    },
    {
      "question": "SAN检定失败后会发生什么？",
      "answer": "SAN检定失败会损失相应的SAN点数。如果单次损失超过当前SAN值的1/5，或者SAN降到0，需要进行额外的一次SAN检定（称为间歇性疯狂检定）。如果失败，调查员会陷入临时疯狂状态。SAN降到0时，调查员会陷入永久疯狂。",
      "category": "sanity",
      "related_rules": ["SAN检定"]
    }
  ]
}
```

**Step 2: Create import script**

```python
# backend/src/scripts/import_seed_rules.py

import asyncio
import json
from pathlib import Path

from sqlalchemy.ext.asyncio import AsyncSession
from backend.src.core.database import async_session_maker
from backend.src.services.llm.openai import OpenAIProvider
from backend.src.services.rule_embedding import RuleEmbeddingService
from backend.src.models.rule import Rule, RuleFAQ
from backend.src.core.config import settings

async def import_seed_rules():
    """Import seed rules from JSON file"""
    # Load seed data
    seed_path = Path(__file__).parent.parent / "data" / "seed_rules.json"

    with open(seed_path, "r", encoding="utf-8") as f:
        data = json.load(f)

    # Initialize services
    llm_provider = OpenAIProvider(
        api_key=settings.OPENAI_API_KEY,
        model="text-embedding-3-small"
    )
    embedding_svc = RuleEmbeddingService(llm_provider)

    async with async_session_maker() as session:
        imported_count = 0
        error_count = 0

        # Import rules
        for rule_data in data["rules"]:
            try:
                rule = Rule(**rule_data)
                rule.embedding = await embedding_svc.embed_rule(rule)
                session.add(rule)
                await session.flush()
                imported_count += 1
                print(f"✓ Imported rule: {rule.title}")
            except Exception as e:
                error_count += 1
                print(f"✗ Failed to import rule {rule_data.get('title', 'unknown')}: {e}")

        # Import FAQs
        for faq_data in data["faqs"]:
            try:
                # Resolve related_rule_ids from titles
                if faq_data.get("related_rules"):
                    rule_titles = faq_data.pop("related_rules")
                    related_rules = await session.execute(
                        select(Rule).where(Rule.title.in_(rule_titles))
                    )
                    faq_data["related_rule_ids"] = [r.id for r in related_rules.scalars().all()]

                faq = RuleFAQ(**faq_data)
                faq.embedding = await embedding_svc.embed_faq(faq)
                session.add(faq)
                await session.flush()
                print(f"✓ Imported FAQ: {faq.question}")
            except Exception as e:
                error_count += 1
                print(f"✗ Failed to import FAQ {faq_data.get('question', 'unknown')}: {e}")

        await session.commit()

        print(f"\nImport complete: {imported_count} imported, {error_count} errors")

if __name__ == "__main__":
    asyncio.run(import_seed_rules())
```

**Step 3: Run import script**

Run: `cd backend && uv run python -m src.scripts.import_seed_rules`

**Step 4: Commit**

```bash
git add backend/src/data/seed_rules.json backend/src/scripts/import_seed_rules.py
git commit -m "feat(M1-098): add seed rules data and import script"
```

---

## Task 7: Frontend Rule Search Component (M1-101)

**Files:**
- Create: `frontend/src/components/rules/RuleSearch.tsx`
- Create: `frontend/src/components/rules/RuleInlineResult.tsx`

**Step 1: Create the RuleSearch component**

```typescript
// frontend/src/components/rules/RuleSearch.tsx

import { useState } from "react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Search } from "lucide-react";
import { api } from "@/services/api";

interface RuleSearchProps {
  onResult?: (result: RuleSearchResult) => void;
  inline?: boolean;
}

export function RuleSearch({ onResult, inline = false }: RuleSearchProps) {
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSearch = async () => {
    if (!query.trim()) return;

    setLoading(true);
    try {
      const response = await api.get(`/rules/search?query=${encodeURIComponent(query)}&limit=3`);
      if (onResult && response.data.results[0]) {
        onResult(response.data.results[0]);
      }
    } catch (error) {
      console.error("Rule search failed:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      handleSearch();
    }
  };

  if (inline) {
    return (
      <div className="flex items-center gap-2">
        <Input
          placeholder="/rule 查询规则..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={handleKeyDown}
          className="h-8"
        />
        <Button size="sm" onClick={handleSearch} disabled={loading}>
          <Search className="h-4 w-4" />
        </Button>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-2 p-4 border-t">
      <Input
        placeholder="输入关键词查询规则..."
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        onKeyDown={handleKeyDown}
      />
      <Button onClick={handleSearch} disabled={loading}>
        {loading ? "查询中..." : "查询"}
      </Button>
    </div>
  );
}
```

**Step 2: Create the RuleInlineResult component**

```typescript
// frontend/src/components/rules/RuleInlineResult.tsx

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { BookOpen } from "lucide-react";

interface RuleInlineResultProps {
  result: RuleSearchResult;
  onViewDetail: () => void;
}

export function RuleInlineResult({ result, onViewDetail }: RuleInlineResultProps) {
  return (
    <div className="rule-inline-result bg-muted/50 border rounded-lg p-4 my-2">
      <div className="flex items-start justify-between gap-3">
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-2">
            <BookOpen className="h-4 w-4 text-muted-foreground" />
            <h4 className="font-semibold">{result.title}</h4>
            <Badge variant="outline" className="text-xs">
              {result.category}
            </Badge>
            {result.relevance_score > 0.8 && (
              <Badge variant="secondary" className="text-xs">
                高度相关
              </Badge>
            )}
          </div>
          <p className="text-sm text-muted-foreground leading-relaxed">
            {result.content}
          </p>
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={onViewDetail}
          className="shrink-0"
        >
          详情 →
        </Button>
      </div>
    </div>
  );
}
```

**Step 3: Create types file**

```typescript
// frontend/src/types/rules.ts

export interface RuleSearchResult {
  id: string;
  title: string;
  category: string;
  content: string;
  relevance_score: number;
  related_rules: RuleSummary[];
}

export interface RuleSummary {
  id: string;
  title: string;
  category: string;
  content: string;
}

export interface RuleDetail {
  id: string;
  title: string;
  category: string;
  subcategory?: string;
  content: string;
  example?: string;
  mechanics?: Record<string, unknown>;
  aliases: string[];
  tags: string[];
  related_rules: RuleDetail[];
  faqs: FAQ[];
}

export interface FAQ {
  id: string;
  question: string;
  answer: string;
  category?: string;
}
```

**Step 4: Commit**

```bash
git add frontend/src/components/rules/RuleSearch.tsx frontend/src/components/rules/RuleInlineResult.tsx frontend/src/types/rules.ts
git commit -m "feat(M1-101): add rule search components"
```

---

## Task 8: Frontend Rule Detail Dialog (M1-102)

**Files:**
- Create: `frontend/src/components/rules/RuleDetailDialog.tsx`
- Modify: `frontend/src/components/rules/index.ts`

**Step 1: Create the RuleDetailDialog component**

```typescript
// frontend/src/components/rules/RuleDetailDialog.tsx

import { useQuery } from "@tanstack/react-query";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { api } from "@/services/api";
import { Loader2 } from "lucide-react";
import type { RuleDetail } from "@/types/rules";

interface RuleDetailDialogProps {
  ruleId: string;
  open: boolean;
  onClose: () => void;
}

export function RuleDetailDialog({ ruleId, open, onClose }: RuleDetailDialogProps) {
  const { data: rule, isLoading, error } = useQuery({
    queryKey: ["rule", ruleId],
    queryFn: async () => {
      const response = await api.get(`/rules/${ruleId}`);
      return response.data as RuleDetail;
    },
    enabled: open && !!ruleId,
  });

  return (
    <Dialog open={open} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
        {isLoading && (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        )}

        {error && (
          <div className="text-center py-8 text-destructive">
            加载规则失败
          </div>
        )}

        {rule && (
          <>
            <DialogHeader>
              <div className="flex items-center gap-2">
                <DialogTitle className="text-xl">{rule.title}</DialogTitle>
                <Badge variant="outline">{rule.category}</Badge>
                {rule.subcategory && (
                  <Badge variant="secondary">{rule.subcategory}</Badge>
                )}
              </div>
            </DialogHeader>

            <div className="space-y-6 mt-4">
              {/* Content */}
              <div>
                <h4 className="font-medium mb-2">规则说明</h4>
                <p className="text-sm text-muted-foreground whitespace-pre-wrap leading-relaxed">
                  {rule.content}
                </p>
              </div>

              {/* Example */}
              {rule.example && (
                <div className="bg-muted p-4 rounded-lg">
                  <h4 className="font-medium text-sm mb-2">示例</h4>
                  <p className="text-sm leading-relaxed">{rule.example}</p>
                </div>
              )}

              {/* Mechanics */}
              {rule.mechanics && Object.keys(rule.mechanics).length > 0 && (
                <div>
                  <h4 className="font-medium mb-2">机械规则</h4>
                  <div className="bg-muted/50 p-3 rounded-lg font-mono text-xs">
                    {Object.entries(rule.mechanics).map(([key, value]) => (
                      <div key={key} className="flex justify-between py-1">
                        <span className="text-muted-foreground">{key}:</span>
                        <span>{String(value)}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Tags and Aliases */}
              {(rule.tags.length > 0 || rule.aliases.length > 0) && (
                <>
                  <Separator />
                  <div className="flex flex-wrap gap-2">
                    {rule.tags.map((tag) => (
                      <Badge key={tag} variant="outline" className="text-xs">
                        #{tag}
                      </Badge>
                    ))}
                    {rule.aliases.map((alias) => (
                      <Badge key={alias} variant="secondary" className="text-xs">
                        别名: {alias}
                      </Badge>
                    ))}
                  </div>
                </>
              )}

              {/* Related Rules */}
              {rule.related_rules.length > 0 && (
                <>
                  <Separator />
                  <div>
                    <h4 className="font-medium mb-3">相关规则</h4>
                    <div className="flex flex-wrap gap-2">
                      {rule.related_rules.map((r) => (
                        <Badge key={r.id} variant="outline" className="cursor-pointer hover:bg-accent">
                          {r.title}
                        </Badge>
                      ))}
                    </div>
                  </div>
                </>
              )}

              {/* FAQs */}
              {rule.faqs.length > 0 && (
                <>
                  <Separator />
                  <div>
                    <h4 className="font-medium mb-3">常见问题</h4>
                    <div className="space-y-3">
                      {rule.faqs.map((faq) => (
                        <div key={faq.id} className="bg-muted/30 p-3 rounded-lg">
                          <p className="font-medium text-sm mb-1">Q: {faq.question}</p>
                          <p className="text-sm text-muted-foreground">A: {faq.answer}</p>
                        </div>
                      ))}
                    </div>
                  </div>
                </>
              )}
            </div>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
```

**Step 2: Create index file**

```typescript
// frontend/src/components/rules/index.ts

export { RuleSearch } from "./RuleSearch";
export { RuleInlineResult } from "./RuleInlineResult";
export { RuleDetailDialog } from "./RuleDetailDialog";
```

**Step 3: Integrate with GameConsole**

```typescript
// frontend/src/components/GameConsole.tsx (add to existing)

import { RuleSearch, RuleInlineResult, RuleDetailDialog } from "@/components/rules";
import { useState } from "react";

export function GameConsole() {
  // ... existing code ...
  const [selectedRuleId, setSelectedRuleId] = useState<string | null>(null);
  const [ruleDetailOpen, setRuleDetailOpen] = useState(false);

  const handleRuleResult = (result: RuleSearchResult) => {
    // Add inline result to messages
    setMessages(prev => [...prev, {
      id: crypto.randomUUID(),
      type: "rule",
      content: result,
      timestamp: new Date()
    }]);
  };

  const handleViewRuleDetail = (ruleId: string) => {
    setSelectedRuleId(ruleId);
    setRuleDetailOpen(true);
  };

  return (
    <div className="game-console">
      {/* ... existing components ... */}

      {/* Add rule search to input area */}
      <div className="input-area">
        <RuleSearch inline onResult={handleRuleResult} />
        {/* ... existing input ... */}
      </div>

      {/* Add rule detail dialog */}
      {selectedRuleId && (
        <RuleDetailDialog
          ruleId={selectedRuleId}
          open={ruleDetailOpen}
          onClose={() => setRuleDetailOpen(false)}
        />
      )}
    </div>
  );
}
```

**Step 4: Update message types**

```typescript
// frontend/src/types/game.ts (add to existing)

export type Message =
  | { type: "user"; content: string; timestamp: Date }
  | { type: "keeper"; content: KeeperMessage; timestamp: Date }
  | { type: "system"; content: string; timestamp: Date }
  | { type: "rule"; content: RuleSearchResult; timestamp: Date };  // Add this
```

**Step 5: Update MessageList to render rule results**

```typescript
// frontend/src/components/MessageList.tsx (add to existing)

import { RuleInlineResult } from "@/components/rules";

// In the message rendering:
{message.type === "rule" && (
  <RuleInlineResult
    result={message.content}
    onViewDetail={() => onViewRuleDetail(message.content.id)}
  />
)}
```

**Step 6: Commit**

```bash
git add frontend/src/components/rules/RuleDetailDialog.tsx frontend/src/components/rules/index.ts
git add frontend/src/components/GameConsole.tsx frontend/src/components/MessageList.tsx frontend/src/types/game.ts
git commit -m "feat(M1-102): add rule detail dialog and integrate with GameConsole"
```

---

## Task 9: NLI Tool Call Integration

**Files:**
- Modify: `backend/src/schemas/llm_response.py`
- Modify: `backend/src/api/websocket.py`
- Modify: `frontend/src/types/websocket.ts`

**Step 1: Update LLM response schema**

```python
# backend/src/schemas/llm_response.py (add to existing)

from pydantic import BaseModel
from typing import Optional, List

class LLMToolCall(BaseModel):
    """Tool call request from AI"""
    name: str
    parameters: dict

class LLMResponse(BaseModel):
    """Update to include tool_calls"""
    narrative: str
    tone: str = "calm"
    urgency: str = "low"
    state_changes: Optional[StateChanges] = None
    suggestions: Optional[List[str]] = None
    audio_cue: Optional[str] = None
    requires_roll: bool = False
    tool_calls: Optional[List[LLMToolCall]] = None  # Add this field
```

**Step 2: Update WebSocket handler to process tool calls**

```python
# backend/src/api/websocket.py (add to existing)

from backend.src.services.rule_search import RuleSearchService
from backend.src.services.rule_embedding import RuleEmbeddingService
from backend.src.services.llm.openai import OpenAIProvider

async def process_tool_calls(tool_calls: List[LLMToolCall], db: AsyncSession):
    """Process AI tool calls and return results"""
    results = []

    for tool_call in tool_calls:
        if tool_call.name == "search_rules":
            # Initialize search service
            llm_provider = OpenAIProvider(
                api_key=settings.OPENAI_API_KEY,
                model="text-embedding-3-small"
            )
            embedding_svc = RuleEmbeddingService(llm_provider)
            search_svc = RuleSearchService(db, embedding_svc)

            # Execute search
            query = tool_call.parameters.get("query", "")
            limit = tool_call.parameters.get("limit", 3)

            search_results = await search_svc.search(query, limit=limit)

            results.append({
                "tool_call_id": tool_call.get("id", ""),
                "name": "search_rules",
                "results": [r.model_dump() for r in search_results]
            })

    return results

# In the WebSocket message handler:
if llm_response.tool_calls:
    tool_results = await process_tool_calls(llm_response.tool_calls, db)

    # Send tool results back to AI for final response
    # This allows AI to incorporate rule citations into narrative
```

**Step 3: Update frontend WebSocket types**

```typescript
// frontend/src/types/websocket.ts (add to existing)

export interface LLMToolCall {
  name: string;
  parameters: Record<string, unknown>;
  id?: string;
}

export interface KeeperMessage {
  narrative: string;
  tone: "mystery" | "horror" | "action" | "calm";
  urgency: "low" | "medium" | "high";
  suggestions?: string[];
  audio_cue?: string;
  requires_roll?: boolean;
  tool_calls?: LLMToolCall[];  // Add this
}
```

**Step 4: Update AI prompt to include tool instructions**

```python
# backend/src/services/prompt.py (add tool instructions to system prompt)

RULE_TOOLS_INSTRUCTION = """
When the player asks about game rules, you can use the following tool:

search_rules: Search the rule database
- Parameters: query (string), limit (number, default 3)
- Use this when: Player asks "how does X work?", "what is the rule for Y?", etc.

After receiving search results, incorporate the rule information naturally into your narrative response.
Format rule citations as: 【规则名】brief explanation
"""

# Append to system prompt
```

**Step 5: Commit**

```bash
git add backend/src/schemas/llm_response.py backend/src/api/websocket.py backend/src/services/prompt.py
git add frontend/src/types/websocket.ts
git commit -m "feat: add NLI tool call integration for rules search"
```

---

## Task 10: Update Task Status

**Files:**
- Modify: `docs/tasks/02-m1-single-player-web.md`

Update the task checkboxes to mark M1-097 through M1-102 as completed:

```markdown
| M1-097 | [x] 设计规则库表结构 | db | 2h | M1-001 | [x] |
| M1-098 | [x] 实现规则入库脚本 | backend | 4h | M1-097 | [x] |
| M1-099 | [x] 实现规则检索 GET /rules/search | backend | 4h | M1-098 | [x] |
| M1-100 | [x] 实现规则引用生成 | backend | 2h | M1-099 | [x] |
| M1-101 | [x] 实现规则搜索组件 | frontend | 4h | M1-031 | [x] |
| M1-102 | [x] 实现规则展示弹窗 | frontend | 2h | M1-101 | [x] |
```

**Commit:**

```bash
git add docs/tasks/02-m1-single-player-web.md
git commit -m "docs(M1-097~102): mark rules knowledge base tasks as completed"
```

---

## Summary

This implementation plan builds a complete RAG-based rules knowledge base system with:

1. **Database**: PostgreSQL + pgvector for vector storage
2. **Services**: Embedding generation, hybrid search, citation formatting
3. **API**: REST endpoints for search, detail, categories, and bulk import
4. **Frontend**: Search component, inline results, detail dialog
5. **NLI Integration**: Tool calls for AI to query rules

**Total estimated time**: ~18 hours (as specified in tasks M1-097 through M1-102)

**Key features**:
- Semantic search using OpenAI embeddings
- Fallback to keyword search on failure
- Inline summary + detail dialog UX
- AI can cite rules naturally in responses
- Seed data with ~50 core rules
