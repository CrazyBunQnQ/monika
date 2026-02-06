"""Character API routes for CRUD operations on character cards."""
import uuid
from typing import List, Optional

from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.orm import Session

from backend.core.database import get_db
from backend.api.dependencies import get_current_user
from backend.models.character import Character, CharacterType
from backend.models.user import User
from backend.schemas.character import (
    CharacterCreate,
    CharacterUpdate,
    CharacterResponse,
    CharacterListResponse,
)

router = APIRouter(prefix="/characters", tags=["characters"])


@router.post("", response_model=CharacterResponse, status_code=status.HTTP_201_CREATED)
def create_character(
    character_data: CharacterCreate,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
) -> Character:
    """
    Create a new character card.

    Args:
        character_data: Character data from request body
        current_user: Authenticated user (from JWT token)
        db: Database session

    Returns:
        Created character object

    Raises:
        HTTPException: 400 if character name already exists for this user
    """
    # Check for duplicate character name for this user
    existing = db.query(Character).filter(
        Character.player_id == current_user.user_id,
        Character.name == character_data.name,
    ).first()

    if existing:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Character with name '{character_data.name}' already exists",
        )

    # Create character
    character_id = str(uuid.uuid4())

    # Convert enum to string value if needed
    char_type = character_data.type
    if hasattr(char_type, "value"):
        char_type = char_type.value

    db_character = Character(
        character_id=character_id,
        player_id=current_user.user_id,
        name=character_data.name,
        type=char_type,
        core_attributes=character_data.core_attributes.model_dump(),
        derived_attributes=character_data.derived_attributes.model_dump(),
        skills=character_data.skills,
        inventory=character_data.inventory,
        clues=character_data.clues,
        status=character_data.status.model_dump(),
    )

    db.add(db_character)
    db.commit()
    db.refresh(db_character)

    return db_character


@router.get("", response_model=CharacterListResponse)
def list_characters(
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
) -> CharacterListResponse:
    """
    List all characters for the authenticated user.

    Args:
        current_user: Authenticated user (from JWT token)
        db: Database session

    Returns:
        List of characters owned by the user
    """
    characters = db.query(Character).filter(
        Character.player_id == current_user.user_id
    ).all()

    return CharacterListResponse(
        characters=characters,
        total=len(characters),
    )


@router.get("/{character_id}", response_model=CharacterResponse)
def get_character(
    character_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
) -> Character:
    """
    Get a specific character by ID.

    Args:
        character_id: Character's unique identifier
        current_user: Authenticated user (from JWT token)
        db: Database session

    Returns:
        Character object

    Raises:
        HTTPException: 404 if character not found or doesn't belong to user
    """
    character = db.query(Character).filter(
        Character.character_id == character_id,
    ).first()

    if not character:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Character not found",
        )

    # Check ownership (player characters only, NPCs are shared)
    if character.player_id and character.player_id != current_user.user_id:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="You don't have permission to access this character",
        )

    return character


@router.put("/{character_id}", response_model=CharacterResponse)
def update_character(
    character_id: str,
    character_data: CharacterUpdate,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
) -> Character:
    """
    Update an existing character.

    Args:
        character_id: Character's unique identifier
        character_data: Updated character data
        current_user: Authenticated user (from JWT token)
        db: Database session

    Returns:
        Updated character object

    Raises:
        HTTPException: 404 if character not found
        HTTPException: 403 if user doesn't own this character
    """
    character = db.query(Character).filter(
        Character.character_id == character_id,
    ).first()

    if not character:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Character not found",
        )

    # Check ownership
    if character.player_id and character.player_id != current_user.user_id:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="You don't have permission to modify this character",
        )

    # Update fields
    update_data = character_data.model_dump(exclude_unset=True)

    for field, value in update_data.items():
        if field == "core_attributes" and value:
            setattr(character, field, value.model_dump() if hasattr(value, "model_dump") else value)
        elif field == "derived_attributes" and value:
            setattr(character, field, value.model_dump() if hasattr(value, "model_dump") else value)
        elif field == "status" and value:
            setattr(character, field, value.model_dump() if hasattr(value, "model_dump") else value)
        elif hasattr(value, "model_dump"):
            setattr(character, field, value.model_dump())
        else:
            setattr(character, field, value)

    db.commit()
    db.refresh(character)

    return character


@router.delete("/{character_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_character(
    character_id: str,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
) -> None:
    """
    Delete a character.

    Args:
        character_id: Character's unique identifier
        current_user: Authenticated user (from JWT token)
        db: Database session

    Raises:
        HTTPException: 404 if character not found
        HTTPException: 403 if user doesn't own this character
    """
    character = db.query(Character).filter(
        Character.character_id == character_id,
    ).first()

    if not character:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Character not found",
        )

    # Check ownership
    if character.player_id and character.player_id != current_user.user_id:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="You don't have permission to delete this character",
        )

    db.delete(character)
    db.commit()


@router.post("/{character_id}/inventory", response_model=CharacterResponse)
def add_inventory_item(
    character_id: str,
    item: dict,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
) -> Character:
    """
    Add an item to character's inventory.

    Args:
        character_id: Character's unique identifier
        item: Item data to add
        current_user: Authenticated user (from JWT token)
        db: Database session

    Returns:
        Updated character object
    """
    character = db.query(Character).filter(
        Character.character_id == character_id,
    ).first()

    if not character:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Character not found",
        )

    # Check ownership
    if character.player_id and character.player_id != current_user.user_id:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="You don't have permission to modify this character",
        )

    inventory = character.inventory or []
    inventory.append(item)

    character.inventory = inventory
    db.commit()
    db.refresh(character)

    return character


@router.delete("/{character_id}/inventory/{item_index}", response_model=CharacterResponse)
def remove_inventory_item(
    character_id: str,
    item_index: int,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
) -> Character:
    """
    Remove an item from character's inventory.

    Args:
        character_id: Character's unique identifier
        item_index: Index of item to remove
        current_user: Authenticated user (from JWT token)
        db: Database session

    Returns:
        Updated character object
    """
    character = db.query(Character).filter(
        Character.character_id == character_id,
    ).first()

    if not character:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Character not found",
        )

    # Check ownership
    if character.player_id and character.player_id != current_user.user_id:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="You don't have permission to modify this character",
        )

    inventory = character.inventory or []

    if item_index < 0 or item_index >= len(inventory):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Invalid item index",
        )

    inventory.pop(item_index)
    character.inventory = inventory
    db.commit()
    db.refresh(character)

    return character
