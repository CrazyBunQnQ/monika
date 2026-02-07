"""Character API routes."""
from typing import List, Dict, Any

from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.orm import Session

from src.core.auth import get_current_user
from src.core.database import get_db
from src.models.character import Character
from src.models.user import User
from src.schemas.character import CharacterCreate, CharacterUpdate

router = APIRouter(prefix="/characters", tags=["characters"])


def character_to_dict(character: Character) -> Dict[str, Any]:
    """Convert Character ORM object to dict."""
    return {
        "id": character.id,
        "owner_id": character.owner_id,
        "name": character.name,
        "age": character.age,
        "gender": character.gender,
        "occupation": character.occupation,
        "mental_illness": character.mental_illness,
        "backstory": character.backstory,
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
        "max_san": character.max_san,
        "luck": character.luck,
        "created_at": character.created_at.isoformat() if character.created_at else None,
        "updated_at": character.updated_at.isoformat() if character.updated_at else None,
    }


@router.post("")
def create_character(
    character_data: CharacterCreate,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """Create a new character."""
    character = Character(
        owner_id=current_user.id,
        name=character_data.name,
        age=character_data.age,
        gender=character_data.gender,
        occupation=character_data.occupation,
        mental_illness=character_data.mental_illness,
        backstory=character_data.backstory,
        str=character_data.str,
        con=character_data.con,
        dex=character_data.dex,
        app=character_data.app,
        pow=character_data.pow,
        int=character_data.intelligence,  # Map intelligence to int
        siz=character_data.siz,
        edu=character_data.edu,
        hp=character_data.hp,
        mp=character_data.mp,
        san=character_data.san,
        max_san=character_data.max_san,
        luck=character_data.luck,
    )
    db.add(character)
    db.commit()
    db.refresh(character)
    return character_to_dict(character)


@router.get("")
def list_characters(
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """List all characters for the current user."""
    characters = (
        db.query(Character)
        .filter(Character.owner_id == current_user.id)
        .all()
    )
    return [character_to_dict(c) for c in characters]


@router.get("/{character_id}")
def get_character(
    character_id: int,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """Get a specific character."""
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
    return character_to_dict(character)


@router.put("/{character_id}")
def update_character(
    character_id: int,
    character_data: CharacterUpdate,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """Update a character."""
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

    # Update fields
    update_data = character_data.model_dump(exclude_unset=True)
    for field, value in update_data.items():
        # Map intelligence to int
        if field == "intelligence":
            field = "int"
        setattr(character, field, value)

    db.commit()
    db.refresh(character)
    return character_to_dict(character)


@router.delete("/{character_id}")
def delete_character(
    character_id: int,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """Delete a character."""
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

    db.delete(character)
    db.commit()
    return {"message": "Character deleted successfully"}
