# Rules Knowledge Base Design Document

**Date:** 2025-01-15
**Status:** Design Complete
**Related Tasks:** M1-097 through M1-102

---

## Overview

The Rules Knowledge Base is a RAG (Retrieval-Augmented Generation) system that enables players to query Call of Cthulhu 7th Edition rules through natural language. It integrates with the existing NLI system, allowing both direct command queries (`/rule`) and AI-orchestrated rule citations.

---

## Architecture

### Design Decisions

| Aspect | Choice | Rationale |
|--------|--------|-----------|
| **Vector Database** | pgvector (PostgreSQL extension) | Reuses existing PostgreSQL, no additional infrastructure needed |
| **Data Organization** | Hybrid Index (rules + FAQs) | Covers both "what is X" and "how to do Y" queries |
| **Frontend Interaction** | Combined Mode | Inline summary + detail dialog, balances speed and depth |
| **NLI Integration** | Hybrid Mode | `/rule` direct query + AI tool call access |
| **Embedding Model** | OpenAI text-embedding-3-small | Lower cost than ada-002, sufficient for rules domain |

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                         Frontend                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ RuleSearch   │  │ RuleInline   │  │RuleDetail    │      │
│  │  Component   │  │  Result      │  │  Dialog      │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    REST API Layer                            │
│  GET /rules/search    GET /rules/{id}    POST /rules/import │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                     Service Layer                            │
│  ┌──────────────────┐  ┌──────────────────┐  ┌─────────────┐ │
│  │ RuleEmbedding    │  │ RuleSearch       │  │ RuleCitation│ │
│  │    Service       │  │    Service       │  │   Service   │ │
│  └──────────────────┘  └──────────────────┘  └─────────────┘ │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Data Layer                                │
│  PostgreSQL + pgvector                                      │
│  ┌──────────────┐  ┌──────────────┐                        │
│  │    rules     │  │  rule_faqs   │                        │
│  │   (1536-dim) │  │  (1536-dim)  │                        │
│  └──────────────┘  └──────────────┘                        │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│              External Services                               │
│  OpenAI API (text-embedding-3-small)                        │
└─────────────────────────────────────────────────────────────┘
```

---

## Database Schema

### rules Table

```sql
CREATE TABLE rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL,
    subcategory VARCHAR(100),
    content TEXT NOT NULL,
    example TEXT,
    mechanics JSONB,
    aliases TEXT[],
    tags TEXT[],
    related_rule_ids UUID[],
    embedding vector(1536),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX rules_fts ON rules USING gin(to_tsvector('english', title || ' ' || content));
CREATE INDEX rules_embedding_idx ON rules USING ivfflat(embedding vector_cosine_ops) WITH (lists = 100);
CREATE INDEX rules_category_idx ON rules(category);
```

### rule_faqs Table

```sql
CREATE TABLE rule_faqs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question TEXT NOT NULL,
    answer TEXT NOT NULL,
    category VARCHAR(100),
    related_rule_ids UUID[],
    embedding vector(1536),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX rule_faqs_embedding_idx ON rule_faqs USING ivfflat(embedding vector_cosine_ops) WITH (lists = 50);
```

---

## API Design

### Search Endpoint

```http
GET /rules/search?query=暗视&category=skill&limit=5
```

**Response:**
```json
{
  "results": [
    {
      "id": "uuid",
      "title": "暗视",
      "category": "skill",
      "content": "暗视允许调查员在...",
      "relevance_score": 0.92,
      "related_rules": [
        {"id": "uuid", "title": "盲视"}
      ]
    }
  ]
}
```

### Get Rule Detail

```http
GET /rules/{rule_id}
```

**Response:**
```json
{
  "id": "uuid",
  "title": "暗视",
  "category": "skill",
  "subcategory": "感知技能",
  "content": "完整规则内容...",
  "example": "例如，调查员在...",
  "mechanics": {
    "modifier": "-20 检定",
    "condition": "完全黑暗"
  },
  "aliases": ["夜视", "Darkness adaptation"],
  "tags": ["感知", "修正"],
  "related_rules": [...],
  "faqs": [
    {"question": "黑暗中如何修正检定", "answer": "..."}
  ]
}
```

---

## Frontend Components

### RuleSearch
- Input component for query entry
- Integrates with GameConsole command processing

### RuleInlineResult
- Displays inline summary in message list
- Shows title, category, content preview
- "View Detail" button triggers dialog

### RuleDetailDialog
- Full modal with complete rule content
- Shows example, mechanics, related rules
- Associated FAQs for "how to" questions

---

## NLI Integration

### Tool Call Format

AI Keeper can request rule searches:

```json
{
  "narrative": "让我查一下暗视的具体规则...",
  "tool_calls": [
    {
      "name": "search_rules",
      "parameters": {
        "query": "暗视",
        "limit": 3
      }
    }
  ]
}
```

### Citation Format

AI responses include formatted rule references:

```
根据【暗视】规则，你在完全黑暗中会有-20的检定修正...
```

---

## MVP Rule Categories

Priority order for initial rule import:

1. **Core Mechanics** - Dice rolling, success levels, push, luck
2. **Skills** - Common skills (Spot Hidden, Listen, Library Use, etc.)
3. **Combat** - Rounds, attacks, damage, dying
4. **Sanity** - SAN checks, madness symptoms
5. **Chase** - Distance, pressure, obstacles

---

## Error Handling

### Embedding Generation Failure
- Fallback to keyword-only search
- Log error for monitoring

### Database Query Failure
- Return empty results with error message
- Don't crash the request

### No Results Found
- Suggest alternative queries
- Show "Related rules" based on query terms

---

## Performance Considerations

### Caching
- LRU cache for embedding generation (API cost reduction)
- Consider Redis for high-traffic deployments

### Index Tuning
- `ivfflat` index with `lists = sqrt(rows)` for optimal recall
- Re-index when rule count grows significantly

### Query Optimization
- Vector search first, then keyword filter
- Limit candidate set before loading related rules

---

## Security

- `/rules/import` requires admin authentication
- Rate limiting on search endpoints (100 req/min per user)
- Input sanitization for query parameters

---

## Testing Strategy

### Unit Tests
- `test_search_by_keyword()` - Keyword-based search
- `test_search_by_embedding()` - Semantic similarity search
- `test_rule_citation_format()` - Citation formatting
- `test_nli_tool_call()` - NLI integration

### Integration Tests
- `test_rule_search_e2e()` - Full search flow
- `test_ai_citation_flow()` - AI tool call → search → citation

### Manual Testing
- Verify 50 core rules are importable
- Test semantic queries ("darkness penalty" → "暗视")
- Confirm inline + dialog interaction flow

---

## Dependencies

### Backend
- `pgvector` PostgreSQL extension
- `openai` Python SDK (for embeddings)
- Existing LLM abstraction layer

### Frontend
- New components in `frontend/src/components/rules/`
- Integration with existing `GameConsole`

---

## Migration Path

1. Install pgvector extension
2. Run migration to create tables
3. Import seed rules (JSON or CSV)
4. Generate embeddings for all rules
5. Test search functionality
6. Deploy frontend components

---

## Future Enhancements

- Rule versioning (if CoC 8e released)
- User-contributed rules (community features)
- Analytics on popular queries
- Multilingual support (English/Chinese)

---

**Document Version:** 1.0
**Last Updated:** 2025-01-15
