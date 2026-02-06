"""Character model for CoC TRPG character cards."""
import enum
from datetime import datetime
from typing import Dict, Any, List

from sqlalchemy import Column, String, Integer, JSON, DateTime, ForeignKey, Enum
from sqlalchemy.orm import relationship
from sqlalchemy.sql import func

from backend.core.database import Base


class CharacterType(str, enum.Enum):
    """Character type enumeration."""
    PLAYER = "player"
    NPC = "npc"


class Character(Base):
    """
    Character model for storing CoC 7th Edition character cards.

    Attributes:
        character_id: Unique identifier (UUID)
        player_id: Reference to the user who owns this character (nullable for NPCs)
        name: Character name
        type: Character type (player or NPC)
        core_attributes: Core attributes (STR, DEX, INT, EDU, APP, POW, SIZ, CON)
        derived_attributes: Derived attributes (HP, MP, SAN, Luck, Move, Build, BonusDamage)
        skills: Character skills organized by category
        inventory: List of items in character's possession
        clues: List of clues the character has discovered
        status: Character status (alive, conscious, dying, insane, conditions)
        created_at: Creation timestamp
        updated_at: Last update timestamp
    """
    __tablename__ = "characters"

    character_id = Column(String, primary_key=True, index=True)
    player_id = Column(String, ForeignKey("users.user_id"), nullable=True, index=True)
    name = Column(String, nullable=False, index=True)
    type = Column(Enum(CharacterType), nullable=False)

    # Core attributes (STR, DEX, INT, EDU, APP, POW, SIZ, CON)
    core_attributes = Column(JSON, nullable=False)

    # Derived attributes (HP, MP, SAN, Luck, Move, Build, BonusDamage)
    derived_attributes = Column(JSON, nullable=False)

    # Skills organized by category: {"common": {...}, "weapons": {...}, "skills": {...}}
    skills = Column(JSON, nullable=False)

    # Additional data
    inventory = Column(JSON, default=list)
    clues = Column(JSON, default=list)
    status = Column(JSON, nullable=False)

    # Timestamps
    created_at = Column(DateTime(timezone=True), server_default=func.now())
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())

    # Relationships
    player = relationship("User", back_populates="characters")

    def __repr__(self) -> str:
        return f"<Character(character_id={self.character_id}, name={self.name}, type={self.type})>"

    def to_dict(self) -> Dict[str, Any]:
        """Convert character to dictionary for serialization."""
        return {
            "character_id": self.character_id,
            "player_id": self.player_id,
            "name": self.name,
            "type": self.type.value if isinstance(self.type, CharacterType) else self.type,
            "core_attributes": self.core_attributes,
            "derived_attributes": self.derived_attributes,
            "skills": self.skills,
            "inventory": self.inventory,
            "clues": self.clues,
            "status": self.status,
            "created_at": self.created_at.isoformat() if self.created_at else None,
            "updated_at": self.updated_at.isoformat() if self.updated_at else None,
        }
