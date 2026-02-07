"""Game API routes for dice and gameplay."""
from typing import Optional

from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel, Field
from sqlalchemy.orm import Session

from src.core.auth import get_current_user
from src.core.database import get_db
from src.models.character import Character
from src.models.user import User
from src.services.dice import roll_check, BonusPenalty, RollResult

router = APIRouter(prefix="/game", tags=["game"])


class DiceRollRequest(BaseModel):
    """Request for a dice roll."""

    skill: int = Field(..., ge=0, le=100, description="Skill or attribute value (0-100)")
    roll: Optional[int] = Field(None, ge=1, le=100, description="Optional fixed roll value")
    bonus_penalty: str = Field("regular", description="Bonus/penalty level")


class DiceRollResponse(BaseModel):
    """Response for a dice roll."""

    value: int
    success_level: str
    raw_rolls: list[int]
    bonus_penalty: str
    skill: int
    result: str


@router.post("/roll", response_model=DiceRollResponse)
def roll_dice(
    request: DiceRollRequest,
    current_user: User = Depends(get_current_user),
):
    """Roll d100 against a skill or attribute.

    Args:
        request: Dice roll parameters.
        current_user: Authenticated user.

    Returns:
        Roll result with success level.
    """
    # Parse bonus/penalty
    try:
        bp = BonusPenalty(request.bonus_penalty)
    except ValueError:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Invalid bonus_penalty level",
        )

    # Perform the roll
    result = roll_check(
        skill=request.skill,
        roll=request.roll,
        bonus_penalty=bp,
    )

    # Format result message
    result_message = format_roll_result(result)

    return DiceRollResponse(
        value=result.value,
        success_level=result.success_level.value,
        raw_rolls=result.raw_rolls or [result.value],
        bonus_penalty=result.bonus_penalty.value,
        skill=request.skill,
        result=result_message,
    )


def format_roll_result(result: RollResult) -> str:
    """Format roll result as a message."""
    level = result.success_level.value.replace("_", " ").title()
    return f"{result.value} - {level}"


@router.post("/characters/{character_id}/roll/{skill_name}")
def roll_character_skill(
    character_id: int,
    skill_name: str,
    roll_value: Optional[int] = None,
    bonus_penalty: str = "regular",
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """Roll a specific skill for a character.

    Args:
        character_id: Character ID.
        skill_name: Name of the skill (str, con, dex, etc.).
        roll_value: Optional fixed roll value.
        bonus_penalty: Bonus/penalty level.
        current_user: Authenticated user.
        db: Database session.

    Returns:
        Roll result.
    """
    # Get character
    character = (
        db.query(Character)
        .filter(Character.id == character_id, Character.owner_id == current_user.id)
        .first()
    )
    if character is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Character not found",
        )

    # Get skill value
    skill_map = {
        "str": character.str,
        "con": character.con,
        "dex": character.dex,
        "app": character.app,
        "pow": character.pow,
        "int": character.int,
        "siz": character.siz,
        "edu": character.edu,
        "hp": character.hp,
        "mp": character.mp,
        "san": character.san,
        "luck": character.luck,
        "max_san": character.max_san,
    }

    skill_value = skill_map.get(skill_name.lower())
    if skill_value is None:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Invalid skill name: {skill_name}",
        )

    # Parse bonus/penalty
    try:
        bp = BonusPenalty(bonus_penalty)
    except ValueError:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Invalid bonus_penalty level",
        )

    # Perform the roll
    result = roll_check(
        skill=skill_value,
        roll=roll_value,
        bonus_penalty=bp,
    )

    # Format result
    result_message = format_roll_result(result)

    return {
        "character_name": character.name,
        "skill_name": skill_name,
        "skill_value": skill_value,
        "roll": result.value,
        "success_level": result.success_level.value,
        "result": result_message,
        "raw_rolls": result.raw_rolls,
    }


@router.get("/health")
def game_health():
    """Health check for game API."""
    return {"status": "ok", "service": "game"}
