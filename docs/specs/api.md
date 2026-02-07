# API Reference

This document describes all REST and WebSocket APIs for the Monika platform.

## Table of Contents

- [REST API](#rest-api)
  - [Authentication](#authentication)
  - [Characters](#characters)
  - [Game Sessions](#game-sessions)
  - [Combat](#combat)
  - [Chase](#chase)
- [WebSocket API](#websocket-api)
  - [Connection](#connection)
  - [Message Types](#message-types)
  - [Examples](#examples)

---

## REST API

Base URL: `http://localhost:8000`

All REST endpoints use JWT authentication. Include the token in the Authorization header:

```
Authorization: Bearer <your_jwt_token>
```

### Authentication

#### Register User
```http
POST /auth/register
Content-Type: application/json

{
  "username": "keeper",
  "email": "keeper@example.com",
  "password": "secure_password"
}
```

**Response:** `201 Created`
```json
{
  "id": "uuid",
  "username": "keeper",
  "email": "keeper@example.com",
  "created_at": "2024-01-01T00:00:00Z"
}
```

#### Login
```http
POST /auth/login
Content-Type: application/json

{
  "username": "keeper",
  "password": "secure_password"
}
```

**Response:** `200 OK`
```json
{
  "access_token": "eyJ...",
  "token_type": "bearer",
  "expires_in": 1800
}
```

#### Refresh Token
```http
POST /auth/refresh
Authorization: Bearer <your_jwt_token>
```

**Response:** `200 OK`
```json
{
  "access_token": "eyJ...",
  "token_type": "bearer",
  "expires_in": 1800
}
```

### Characters

#### Create Character
```http
POST /characters/
Authorization: Bearer <your_jwt_token>
Content-Type: application/json

{
  "name": "Investigator Smith",
  "age": 35,
  "occupation": "Private Detective",
  "str": 60,
  "dex": 70,
  "int": 75,
  "con": 60,
  "app": 50,
  "pow": 60,
  "siz": 55,
  "edu": 70,
  "san": 60,
  "hp": 12,
  "mp": 12,
  "skills": {
    "Spot Hidden": 60,
    "Listen": 50,
    "Library Use": 70,
    "Psychology": 40
  }
}
```

**Response:** `201 Created`
```json
{
  "id": "uuid",
  "name": "Investigator Smith",
  "age": 35,
  "occupation": "Private Detective",
  "str": 60,
  "dex": 70,
  "int": 75,
  "con": 60,
  "app": 50,
  "pow": 60,
  "siz": 55,
  "edu": 70,
  "san": 60,
  "hp": 12,
  "mp": 12,
  "skills": {
    "Spot Hidden": 60,
    "Listen": 50,
    "Library Use": 70,
    "Psychology": 40
  },
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

#### Get Character
```http
GET /characters/{character_id}
Authorization: Bearer <your_jwt_token>
```

#### List Characters
```http
GET /characters/
Authorization: Bearer <your_jwt_token>
```

#### Update Character
```http
PUT /characters/{character_id}
Authorization: Bearer <your_jwt_token>
Content-Type: application/json

{
  "hp": 10,
  "san": 45
}
```

#### Delete Character
```http
DELETE /characters/{character_id}
Authorization: Bearer <your_jwt_token>
```

### Game Sessions

#### Create Session
```http
POST /game/sessions
Authorization: Bearer <your_jwt_token>
Content-Type: application/json

{
  "character_id": "uuid",
  "scenario_id": "haunted_mansion",
  "current_scene": "entry_hall"
}
```

**Response:** `201 Created`
```json
{
  "id": "uuid",
  "character_id": "uuid",
  "scenario_id": "haunted_mansion",
  "current_scene": "entry_hall",
  "world_state": {},
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

#### Get Session
```http
GET /game/sessions/{session_id}
Authorization: Bearer <your_jwt_token>
```

#### List Sessions
```http
GET /game/sessions
Authorization: Bearer <your_jwt_token>
```

#### Update Session State
```http
PUT /game/sessions/{session_id}/state
Authorization: Bearer <your_jwt_token>
Content-Type: application/json

{
  "current_scene": "library",
  "world_state": {
    "discovered_secret": true
  }
}
```

### Combat

#### Start Combat
```http
POST /combat/start
Authorization: Bearer <your_jwt_token>
Content-Type: application/json

{
  "session_id": "uuid",
  "combatants": [
    {
      "name": "Cultist",
      "initiative": 45,
      "hp": 12,
      "damage_bonus": "1D4"
    }
  ]
}
```

#### Combat Action
```http
POST /combat/{combat_id}/action
Authorization: Bearer <your_jwt_token>
Content-Type: application/json

{
  "attacker_id": "uuid",
  "target_id": "uuid",
  "action_type": "attack",
  "skill_value": 65,
  "weapon_damage": "1D6+2"
}
```

#### End Combat
```http
POST /combat/{combat_id}/end
Authorization: Bearer <your_jwt_token>
```

### Chase

#### Start Chase
```http
POST /chase/start
Authorization: Bearer <your_jwt_token>
Content-Type: application/json

{
  "session_id": "uuid",
  "chase_type": "foot",
  "participants": [
    {
      "name": "Cultist Leader",
      "movement_rate": 8,
      "is_quarry": true
    }
  ]
}
```

#### Chase Action
```http
POST /chase/{chase_id}/action
Authorization: Bearer <your_jwt_token>
Content-Type: application/json

{
  "participant_id": "uuid",
  "action_type": "move",
  "distance_change": 2
}
```

---

## WebSocket API

The WebSocket API enables real-time communication with the AI Keeper for natural language interaction.

### Connection

#### WebSocket Endpoint
```
ws://localhost:8000/ws/game/{session_id}
```

**Parameters:**
- `session_id` (string, required): Game session UUID

**Connection Flow:**
1. Client connects to WebSocket endpoint
2. Server validates session and character
3. Server sends `connected` message
4. Client can send `user_message` messages
5. Server responds with streaming `keeper_message` messages
6. Server may send `state_update` messages when AI modifies game state
7. Connection persists until explicitly closed or error occurs

### Message Types

All WebSocket messages are JSON objects with a `type` field.

#### Client Messages

**User Message**
```json
{
  "type": "user_message",
  "content": "I want to examine the old bookshelf carefully",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

- `type` (string): Must be `"user_message"`
- `content` (string): Player's natural language input
- `timestamp` (string): ISO 8601 timestamp

#### Server Messages

**Connected**
```json
{
  "type": "connected",
  "content": {
    "session_id": "uuid",
    "character_name": "Investigator Smith"
  }
}
```

Sent upon successful connection.

**Keeper Message (Streaming)**
```json
{
  "type": "keeper_message",
  "content": {
    "narrative": "As you approach the bookshelf...",
    "tone": "mystery",
    "urgency": "low",
    "suggestions": ["Search for hidden compartments", "Read the book titles"]
  },
  "is_streaming": true
}
```

- `type` (string): `"keeper_message"`
- `content.narrative` (string): AI-generated narrative text
- `content.tone` (string): `"mystery" | "horror" | "action" | "calm"`
- `content.urgency` (string): `"low" | "medium" | "high"`
- `content.suggestions` (array of string, optional): Suggested player actions
- `content.audio_cue` (string, optional): Sound effect suggestion
- `content.requires_roll` (boolean, optional): Whether a skill check is suggested
- `is_streaming` (boolean): `true` while streaming, `false` for final message

**State Update**
```json
{
  "type": "state_update",
  "content": {
    "current_scene": "library",
    "world_state": {
      "discovered_secret": true,
      "bookshelf_searched": true
    }
  }
}
```

Sent when AI modifies game state (only whitelisted fields).

**Error**
```json
{
  "type": "error",
  "content": "Session not found"
}
```

Sent when an error occurs.

### Examples

#### Complete Conversation Flow

**1. Client Connects**
```
ws://localhost:8000/ws/game/a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

**2. Server Confirms Connection**
```json
{
  "type": "connected",
  "content": {
    "session_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "character_name": "Investigator Smith"
  }
}
```

**3. Client Sends Message**
```json
{
  "type": "user_message",
  "content": "I want to examine the old bookshelf carefully",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

**4. Server Streams Response (Multiple Messages)**
```json
{"type":"keeper_message","content":{"narrative":"A","tone":"mystery","urgency":"low"},"is_streaming":true}
```
```json
{"type":"keeper_message","content":{"narrative":"As you approach the bookshelf, your fingers trace the worn leather spines of ancient tomes.","tone":"mystery","urgency":"low"},"is_streaming":true}
```
```json
{"type":"keeper_message","content":{"narrative":"As you approach the bookshelf, your fingers trace the worn leather spines of ancient tomes. Dust motes dance in the pale light filtering through the boarded windows. Most books appear to be mundane late 19th-century texts, but one particularly thick volume catches your eye - it's bound in dark leather with strange symbols embossed on the spine.","tone":"mystery","urgency":"low","suggestions":["Examine the strange book","Continue searching the shelf","Check for hidden compartments"],"audio_cue":"creaking_wood"},"is_streaming":false}
```

**5. Server Sends State Update (if AI modified state)**
```json
{
  "type": "state_update",
  "content": {
    "current_scene": "library",
    "world_state": {
      "discovered_strange_book": true
    }
  }
}
```

### Error Handling

The WebSocket connection may close or send error messages in these cases:

**Invalid Session ID**
```json
{
  "type": "error",
  "content": "Invalid session ID format"
}
```
Connection closes immediately.

**Session Not Found**
```json
{
  "type": "error",
  "content": "Session not found"
}
```
Connection closes immediately.

**Character Not Found**
```json
{
  "type": "error",
  "content": "Character not found"
}
```
Connection closes immediately.

**Processing Error**
```json
{
  "type": "error",
  "content": "Failed to process message"
}
```
Connection remains open for new messages.

### State Change Whitelist

The AI Keeper can only modify specific whitelisted fields in the game state:

- `current_scene`: Scene name (string)
- `world_state`: Arbitrary key-value pairs (object)

The following fields **cannot** be modified by AI:
- `id`: Session ID
- `character_id`: Associated character
- `scenario_id`: Scenario identifier
- `created_at`: Creation timestamp
- `updated_at`: Update timestamp

This ensures data integrity while allowing narrative flexibility.

---

## Response Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not Found |
| 422 | Validation Error |
| 500 | Internal Server Error |

---

## Rate Limiting

API endpoints are rate-limited to prevent abuse:

- REST API: 100 requests per minute per IP
- WebSocket: 10 messages per minute per session

Rate limit headers are included in responses:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1609459200
```

---

## Testing

Use the provided test files to verify API functionality:

```bash
# Backend tests
cd backend
uv run pytest src/tests/

# Frontend tests (when available)
cd frontend
npm test
```
