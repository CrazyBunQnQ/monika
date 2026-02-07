"""Chase API routes."""
import uuid
from typing import List, Optional

from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel, Field

from sqlalchemy.orm import Session

from src.core.auth import get_current_user
from src.core.database import get_db
from src.models.user import User
from src.services.chase import ChaseService

router = APIRouter(prefix="/chase", tags=["chase"])


# Request/Response Schemas
class ChaseCreateRequest(BaseModel):
    """Request to create a chase session."""

    session_id: str = Field(..., description="Game session UUID")
    location: str = Field(..., description="Where the chase takes place")
    setting: str = Field("city_streets", description="Type of environment")


class ChaseParticipantCreateRequest(BaseModel):
    """Request to add a chase participant."""

    name: str = Field(..., min_length=1, max_length=100, description="Participant name")
    role: str = Field(..., pattern="^(fugitive|pursuer)$", description="fugitive or pursuer")
    move_rate: int = Field(8, ge=1, le=20, description="Movement rate (usually 8 or 9)")
    is_player: bool = Field(False, description="Whether this is a player character")
    character_id: Optional[int] = Field(None, description="Optional character ID")


class ChaseRoundRequest(BaseModel):
    """Request to resolve a chase round."""

    actions: List[dict] = Field(
        ...,
        description="List of actions taken by participants this round",
        min_length=1,
    )


class ChaseActionRequest(BaseModel):
    """Request for a single chase action."""

    participant_id: str = Field(..., description="Participant UUID")
    action_type: str = Field(
        ...,
        pattern="^(accelerate|decelerate|overcome_obstacle|attack)$",
        description="Type of action",
    )
    obstacle_id: Optional[str] = Field(None, description="Obstacle UUID (for overcome_obstacle)")
    skill: Optional[int] = Field(None, ge=0, le=100, description="Skill value (for checks)")


class ChaseEndRequest(BaseModel):
    """Request to manually end a chase."""

    reason: str = Field(..., description="Why the chase ended")
    fail_forward_scene: Optional[str] = Field(
        None, description="What happens next (fail forward)"
    )


class ChaseResponse(BaseModel):
    """Response with chase data."""

    id: str
    state: str
    round: int
    location: str
    setting: str
    started_at: Optional[str]
    ended_at: Optional[str]
    end_reason: Optional[str]
    failed_forward_scene: Optional[str]
    participants: List[dict]
    obstacles: List[dict]


class ChaseRoundResponse(BaseModel):
    """Response for round resolution."""

    chase_id: str
    round: int
    actions: List[dict]
    positions: List[dict]
    chase_ended: bool
    end_reason: Optional[str]


class ObstacleResponse(BaseModel):
    """Response for obstacle data."""

    id: str
    name: str
    description: str
    type: str
    difficulty: str
    skill_required: Optional[str]
    failure_penalty: int
    failure_damage: Optional[int]
    fail_forward_result: Optional[str]


# Endpoints
@router.post("/start", response_model=ChaseResponse)
def start_chase(
    request: ChaseCreateRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """Start a new chase session.

    Creates a new chase with the given location and setting.
    """
    service = ChaseService(db)

    try:
        session_id = uuid.UUID(request.session_id)
    except ValueError:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Invalid session_id format",
        )

    chase = service.create_chase(
        session_id=session_id,
        location=request.location,
        setting=request.setting,
    )

    return service.get_chase_summary(chase.id)


@router.post("/{chase_id}/participants", response_model=dict)
def add_participant(
    chase_id: str,
    request: ChaseParticipantCreateRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """Add a participant to a chase session."""
    service = ChaseService(db)

    try:
        chase_uuid = uuid.UUID(chase_id)
    except ValueError:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Invalid chase_id format",
        )

    participant = service.add_participant(
        chase_id=chase_uuid,
        name=request.name,
        role=request.role,
        move_rate=request.move_rate,
        is_player=request.is_player,
        character_id=request.character_id,
    )

    return service._participant_to_dict(participant)


@router.get("/{chase_id}", response_model=ChaseResponse)
def get_chase(
    chase_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """Get chase session details."""
    service = ChaseService(db)

    try:
        chase_uuid = uuid.UUID(chase_id)
    except ValueError:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Invalid chase_id format",
        )

    return service.get_chase_summary(chase_uuid)


@router.post("/{chase_id}/round", response_model=ChaseRoundResponse)
def resolve_round(
    chase_id: str,
    request: ChaseRoundRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """Resolve a round of chase actions.

    Processes all participant actions for the current round,
    updates positions based on speeds, and checks if the chase should end.
    """
    service = ChaseService(db)

    try:
        chase_uuid = uuid.UUID(chase_id)
    except ValueError:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Invalid chase_id format",
        )

    return service.resolve_round(chase_uuid, request.actions)


@router.post("/{chase_id}/obstacles/generate", response_model=ObstacleResponse)
def generate_obstacle(
    chase_id: str,
    round: int = 1,
    distance_level: int = 0,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """Generate a random obstacle for the current chase round.

    Creates an appropriate obstacle based on chase setting and current distance.
    """
    service = ChaseService(db)

    try:
        chase_uuid = uuid.UUID(chase_id)
    except ValueError:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Invalid chase_id format",
        )

    obstacle = service.generate_obstacle(chase_uuid, round, distance_level)
    return service._obstacle_to_dict(obstacle)


@router.post("/{chase_id}/end", response_model=ChaseResponse)
def end_chase(
    chase_id: str,
    request: ChaseEndRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """Manually end a chase session.

    Use this when the chase should end early or you want to
    force a specific outcome with fail-forward.
    """
    service = ChaseService(db)

    try:
        chase_uuid = uuid.UUID(chase_id)
    except ValueError:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Invalid chase_id format",
        )

    chase = service.end_chase(
        chase_uuid, request.reason, request.fail_forward_scene
    )

    return service.get_chase_summary(chase.uuid)


@router.get("/health")
def chase_health():
    """Health check for chase API."""
    return {"status": "ok", "service": "chase"}
