# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Monika** is an AI-driven Call of Cthulhu 7th Edition (CoC 7e) Tabletop Role-Playing Game (TRPG) platform. It uses AI as the Keeper (KP/Game Master) to facilitate TRPG sessions with automated game mechanics including skill checks, combat, SAN checks, and chase systems.

### Architecture

**Frontend-Backend Separation:**
- **Backend**: Python 3.11+ with FastAPI, PostgreSQL, SQLAlchemy ORM
- **Frontend**: React 19 with TypeScript, Vite, shadcn/ui components, TailwindCSS
- **API Design**: RESTful endpoints with JWT authentication

### Directory Structure

```
monika/
├── backend/
│   ├── src/
│   │   ├── api/           # FastAPI route handlers (auth, characters, game, combat, chase, sessions)
│   │   ├── core/          # Core utilities (auth, database, config, security)
│   │   ├── models/        # SQLAlchemy database models
│   │   ├── schemas/       # Pydantic validation schemas
│   │   ├── services/      # Business logic (dice, combat, chase, events)
│   │   └── tests/         # pytest test suite
│   ├── pyproject.toml     # uv package configuration
│   └── main.py           # FastAPI app entry point (includes all routers)
├── frontend/
│   ├── src/
│   │   ├── components/    # React components
│   │   │   ├── ui/       # shadcn/ui base components
│   │   │   └── [game components]
│   │   ├── pages/        # Page components
│   │   └── main.tsx      # React entry point
│   ├── package.json
│   └── vite.config.ts
└── docs/                 # Comprehensive documentation (PRD, architecture, specs)
```

## Development Commands

### Backend

```bash
cd backend

# Install dependencies (uses uv package manager)
uv sync

# Run development server (auto-reload on port 8000)
uv run python -m uvicorn src.main:app --reload

# Run all tests
uv run pytest

# Run specific test file
uv run pytest src/tests/test_dice.py

# Run tests with coverage
uv run pytest --cov=src

# Code formatting
uv run black src/

# Linting
uv run ruff check src/
```

### Frontend

```bash
cd frontend

# Install dependencies
npm install

# Run development server (Vite dev server on port 5173)
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview

# Type checking
npx tsc --noEmit
```

## Core Game Mechanics

### CoC 7e Dice System (`backend/src/services/dice.py`)

- **Success Levels**: Regular, Hard, Extreme success thresholds based on skill/5 and skill/2
- **Bonus/Penalty Dice**: Roll multiple d100, take lowest (bonus) or highest (penalty)
- **Criticals**: Natural 1 is critical success, natural 100 is critical failure (or >96 for low skills)
- **Pushing**: Failed rolls can be re-rolled with narrative consequences
- **Luck Points**: Can be spent to improve rolls

### Combat System (`backend/src/services/combat.py`)

Turn-based combat with:
- Initiative order (DEX-based)
- Attack rolls vs opponent's dodge
- Damage rolls with weapon damage dice
- Critical hits and fumbles
- Dying and death mechanics at 0 HP

### Chase System (`backend/src/services/chase.py`)

Distance-based chase mechanics:
- Action points per round (based on movement rate)
- Obstacles that require skill checks
- Automatic failure at high pressure levels
- Distance tracking between pursuer and quarry

### Event System (`backend/src/services/events.py`)

Event-driven state changes with:
- Visibility levels (public, kp_only, private)
- State change tracking (character, session, world)
- SAN loss events with breakdown thresholds

## Database Models

Key models in `backend/src/models/`:
- **User**: Authentication and profile
- **Character**: PC stats, skills, SAN, HP, MP
- **GameSession**: Session state, scene tracking, world state
- **Combat**: Combat encounters with participants and rounds
- **Chase**: Chase sequences with movement and obstacles
- **Event**: Event log with visibility and state changes

Database uses SQLAlchemy ORM with Alembic migrations.

## API Structure

All API routes are registered in `backend/src/main.py`:
- `/auth/*` - Authentication (register, login, token refresh)
- `/characters/*` - Character CRUD operations
- `/game/*` - Game session management
- `/combat/*` - Combat system endpoints
- `/chase/*` - Chase system endpoints
- `/sessions/*` - Session management

CORS is configured for `http://localhost:5173` and `http://localhost:3000`.

## Frontend Components

Key components in `frontend/src/components/`:
- **GameConsole**: Main game interface with message display and controls
- **CharacterForm**: Character creation/editing
- **StatePanel**: Displays character stats, world state, and leads
- **MessageList**: Chat-like message interface
- **DiceRoll**: Interactive dice rolling

Uses shadcn/ui base components with TailwindCSS styling.

## Testing Strategy

- **Backend**: pytest with SQLite in-memory database for testing
- **Fixtures**: `client` (TestClient), `test_db` (database session) in `conftest.py`
- **Test Mode**: `asyncio_mode = "auto"` for async test support
- **Coverage**: Integration tests for API endpoints, unit tests for services

## Configuration

Backend configuration via environment variables or `.env` file:
- `DATABASE_URL`: PostgreSQL connection string
- `SECRET_KEY`: JWT signing key
- `ALGORITHM`: JWT algorithm (default: HS256)
- `ACCESS_TOKEN_EXPIRE_MINUTES`: Token expiry (default: 30)
- `DEBUG`: Debug mode flag

See `backend/src/core/config.py` for defaults.

## Task-Driven Development Workflow

This project follows a **strict task-driven development process**. All development work must be tracked and completed according to the task system in `docs/tasks/`.

### Task Structure

Tasks are organized hierarchically:
- **Milestone**: M0-M6 (e.g., M1 = Single-Player Web Version)
- **Main Task**: M{XX}-{NNN} (e.g., M1-006 = User Authentication API)
- **Subtask**: M{XX}-{NNN}-{NN} (e.g., M1-006-01 = Create user.py model)

Task files are located in `docs/tasks/tasks-detailed/` (e.g., `M1-006-user-auth.md`).

### Task Status Convention

Task status is marked using markdown checkboxes in task files:

| Status | Checkbox | Meaning |
|--------|-----------|---------|
| Not Started | `[ ]` | Task not yet started |
| In Progress | `[~]` | Task currently being worked on |
| Completed | `[x]` | Task completed and verified |

Example:
```markdown
| M1-006-01 | [x] Create `app/db/user.py` | [x] |
| M1-006-02 | [ ] Define `User` SQLModel | [ ] |
| M1-006-03 | [~] Implement password hashing | [~] |
```

### Mandatory Development Workflow

**Before starting any development work:**

1. **Read the task file** - Each detailed task file in `docs/tasks/tasks-detailed/` contains:
   - Subtask breakdown with time estimates
   - Code examples and templates
   - Test case requirements
   - File locations and structure
   - Acceptance criteria

2. **Check dependencies** - Each task lists dependencies (e.g., "Depends: M0"). Ensure dependent tasks are completed first.

3. **Mark task as in-progress** - Change the task checkbox from `[ ]` to `[~]` before starting work.

**During development:**

4. **Follow the subtask breakdown** - Complete subtasks in the order specified in the task file. Each subtask has:
   - Specific file to create/modify
   - Code to implement
   - Time estimate

5. **Write tests first (TDD)** - The project uses Test-Driven Development. Write test cases before implementing functionality.

6. **Run tests frequently** - After each subtask, run relevant tests to ensure nothing breaks.

**After completing development:**

7. **Verify acceptance criteria** - Each task file lists acceptance criteria. Ensure all are met before marking complete.

8. **Run full test suite** - Execute `uv run pytest` (backend) or `npm test` (frontend) to ensure no regressions.

9. **Update task status** - Change the task checkbox from `[~]` to `[x]` for:
   - The subtask(s) you completed
   - The parent task (when all subtasks are done)

10. **Commit with task reference** - Include task ID in commit messages:
    ```
    git commit -m "feat(M1-006): implement user registration API"
    ```

### Finding Tasks

- **Milestone overview**: `docs/tasks/02-m1-single-player-web.md` - Shows all tasks for current milestone with status
- **Task index**: `docs/tasks/tasks-detailed/INDEX.md` - Complete index of all detailed tasks
- **Detailed task**: `docs/tasks/tasks-detailed/M{XX}-{NNN}-{name}.md` - Full task breakdown

### Example Workflow

```bash
# 1. Find the next task to work on
cat docs/tasks/02-m1-single-player-web.md  # Look for [ ] tasks

# 2. Read the detailed task file
cat docs/tasks/tasks-detailed/M1-006-user-auth.md

# 3. Mark task as in-progress
# Edit the file: change [ ] to [~] for M1-006

# 4. Implement according to subtasks
# Follow M1-006-01, M1-006-02, etc. in order

# 5. Run tests
uv run pytest

# 6. Mark task as complete
# Edit the file: change [~] to [x] for all completed subtasks and parent task

# 7. Commit changes
git add .
git commit -m "feat(M1-006): implement user authentication"
```

### Current Milestone Progress

Check `docs/tasks/02-m1-single-player-web.md` for the current milestone status and remaining tasks.

### Important Rules

- **NEVER skip tasks** - Complete tasks in dependency order
- **ALWAYS mark status** - Keep task checkboxes up-to-date
- **READ task files first** - Each file contains crucial implementation details
- **Follow TDD** - Write tests before implementation
- **Reference task IDs in commits** - Enables traceability

## Documentation

Comprehensive documentation in `docs/`:
- `docs/prd/` - Product Requirements Document
- `docs/guides/developer/` - Architecture, API reference, data dictionary
- `docs/specs/` - Commands, components, event structure, state structure
- `docs/tasks/` - Milestone breakdown (M0-M6) and task tracking

Current milestone: **M1 - Single-Player Web Version** (in progress)

## Game Commands Reference

The platform uses a slash-command system (see `docs/specs/commands.md`):
- `/help [command]` - Display help
- `/status` - Show current state
- `/roll <skill>` - Skill/attribute check
- `/push` - Re-roll failed check
- `/luck [n]` - Spend luck points
- `/combat start/action/end` - Combat commands
- `/san check <value>` - SAN check
- `/leads` - Show available actions
- `/rule <query>` - Rules Q&A
