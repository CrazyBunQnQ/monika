"""Combat API routes."""
import uuid
from typing import List, Optional

from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel, Field

from sqlalchemy.orm import Session

from src.core.auth import get_current_user
from src.core.database import get_db
from src.models.user import User
from src.services.combat import CombatService

router = APIRouter(prefix="/combat", tags=["combat"])


# Request/Response Schemas
class CombatCreateRequest(BaseModel):
    """Request to create a combat session."""

    session_id: str = Field(..., description="Game session UUID")
    location: Optional[str] = Field(None, description="Combat location")
    description: Optional[str] = Field(None, description="Combat description")


class CombatantCreateRequest(BaseModel):
    """Request to add a combatant."""

    name: str = Field(..., min_length=1, max_length=100, description="Combatant name")
    hp: int = Field(..., ge=0, description="Current HP")
    hp_max: int = Field(..., gt=0, description="Maximum HP")
    dex: int = Field(..., ge=0, le=100, description="Dexterity for initiative")
    role: str = Field("npc", pattern="^(pc|npc|ally)$", description="Combatant role")
    character_id: Optional[int] = Field(None, description="Optional character ID")


class AttackRequest(BaseModel):
    """Request to make an attack."""

    attacker_id: str = Field(..., description="Attacker combatant UUID")
    target_id: str = Field(..., description="Target combatant UUID")
    attack_skill: int = Field(..., ge=0, le=100, description="Attack skill value")
    attack_roll: Optional[int] = Field(None, ge=1, le=100, description="Fixed roll (optional)")
    damage_roll: Optional[int] = Field(None, ge=1, description="Fixed damage roll (optional)")
    damage_bonus: int = Field(0, ge=0, description="Damage bonus from strength")


class HealRequest(BaseModel):
    """Request to heal a combatant."""

    target_id: str = Field(..., description="Target combatant UUID")
    heal_amount: int = Field(..., ge=0, description="Base healing amount")
    first_aid_skill: int = Field(..., ge=0, le=100, description="First Aid skill")
    first_aid_roll: Optional[int] = Field(None, ge=1, le=100, description="Fixed roll (optional)")


class CombatResponse(BaseModel):
    """Response with combat data."""

    id: str
    state: str
    round: int
    location: Optional[str]
    description: Optional[str]
    started_at: Optional[str]
    ended_at: Optional[str]
    combatants: List[dict]
    current_turn: Optional[dict]


class CombatantResponse(BaseModel):
    """Response with combatant data."""

    id: str
    name: str
    role: str
    initiative: int
    dex: int
    hp: int
    hp_max: int
    is_active: bool
    is_dying: bool
    has_major_wound: bool
    is_unconscious: bool


class TurnResponse(BaseModel):
    """Response for turn advancement."""

    combat_id: str
    current_round: int
    current_turn_index: int
    current_combatant: Optional[dict]
    is_new_round: bool
    turn_order: List[dict]


class AttackResponse(BaseModel):
    """Response for attack resolution."""

    attacker: str
    target: str
    attack_roll: int
    attack_skill: int
    success_level: str
    hit: bool
    damage: int
    target_hp_before: int
    target_hp_after: int
    target_status: str
    action_id: str


class HealResponse(BaseModel):
    """Response for healing."""

    target: str
    first_aid_roll: int
    first_aid_skill: int
    success_level: str
    hp_before: int
    healing: int
    hp_after: int
    action_id: str


# Endpoints
@router.post("/start", response_model=CombatResponse)
def start_combat(
    request: CombatCreateRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """Start a new combat session.

    Creates a new combat session and rolls initiative for all combatants.
    """
    service = CombatService(db)

    try:
        session_id = uuid.UUID(request.session_id)
    except ValueError:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Invalid session_id format",
        )

    combat = service.create_combat(
        session_id=session_id,
        location=request.location,
        description=request.description,
    )

    return service.get_combat_summary(combat.id)


@router.post("/{combat_id}/combatants", response_model=CombatantResponse)
def add_combatant(
    combat_id: str,
    request: CombatantCreateRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """Add a combatant to an existing combat session."""
    service = CombatService(db)

    try:
        combat_uuid = uuid.UUID(combat_id)
    except ValueError:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Invalid combat_id format",
        )

    combatant = service.add_combatant(
        combat_id=combat_uuid,
        name=request.name,
        hp=request.hp,
        hp_max=request.hp_max,
        dex=request.dex,
        role=request.role,
        character_id=request.character_id,
    )

    return combatant.to_dict()


@router.get("/{combat_id}", response_model=CombatResponse)
def get_combat(
    combat_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """Get combat session details."""
    service = CombatService(db)

    try:
        combat_uuid = uuid.UUID(combat_id)
    except ValueError:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Invalid combat_id format",
        )

    return service.get_combat_summary(combat_uuid)


@router.get("/{combat_id}/turn-order", response_model=List[CombatantResponse])
def get_turn_order(
    combat_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """Get combatants in initiative order."""
    service = CombatService(db)

    try:
        combat_uuid = uuid.UUID(combat_id)
    except ValueError:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Invalid combat_id format",
        )

    turn_order = service.get_turn_order(combat_uuid)
    return [c.to_dict() for c in turn_order]


@router.post("/{combat_id}/turn", response_model=TurnResponse)
def next_turn(
    combat_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """Advance to the next combatant's turn."""
    service = CombatService(db)

    try:
        combat_uuid = uuid.UUID(combat_id)
    except ValueError:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Invalid combat_id format",
        )

    return service.next_turn(combat_uuid)


@router.post("/{combat_id}/attack", response_model=AttackResponse)
def resolve_attack(
    combat_id: str,
    request: AttackRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """Resolve an attack in combat.

    Performs the attack roll, determines hit/miss based on success level,
    calculates damage, and updates target HP.
    """
    service = CombatService(db)

    try:
        combat_uuid = uuid.UUID(combat_id)
        attacker_uuid = uuid.UUID(request.attacker_id)
        target_uuid = uuid.UUID(request.target_id)
    except ValueError:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Invalid UUID format",
        )

    result = service.resolve_attack(
        combat_id=combat_uuid,
        attacker_id=attacker_uuid,
        target_id=target_uuid,
        attack_skill=request.attack_skill,
        attack_roll=request.attack_roll,
        damage_roll=request.damage_roll,
        damage_bonus=request.damage_bonus,
    )

    return result


@router.post("/{combat_id}/heal", response_model=HealResponse)
def heal_combatant(
    combat_id: str,
    request: HealRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """Apply healing to a combatant.

    Makes a First Aid roll and heals the target if successful.
    Success level determines healing amount:
    - Regular: 1 HP
    - Hard: 1d3 + 1 HP
    - Extreme: 1d6 + 2 HP
    """
    service = CombatService(db)

    try:
        combat_uuid = uuid.UUID(combat_id)
        target_uuid = uuid.UUID(request.target_id)
    except ValueError:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Invalid UUID format",
        )

    result = service.heal(
        combat_id=combat_uuid,
        target_id=target_uuid,
        heal_amount=request.heal_amount,
        first_aid_skill=request.first_aid_skill,
        first_aid_roll=request.first_aid_roll,
    )

    return result


@router.post("/{combat_id}/end", response_model=CombatResponse)
def end_combat(
    combat_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """End a combat session."""
    service = CombatService(db)

    try:
        combat_uuid = uuid.UUID(combat_id)
    except ValueError:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Invalid combat_id format",
        )

    combat = service.end_combat(combat_uuid)
    return service.get_combat_summary(combat_uuid)


@router.get("/health")
def combat_health():
    """Health check for combat API."""
    return {"status": "ok", "service": "combat"}
