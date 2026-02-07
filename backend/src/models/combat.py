"""Combat state model for CoC 7e combat system."""
import uuid
from datetime import datetime
from enum import Enum

from sqlalchemy import Column, String, Integer, DateTime, JSON, ForeignKey, UUID, Boolean
from sqlalchemy.sql import func
from sqlalchemy.orm import relationship

from src.core.database import Base


class CombatState(str, Enum):
    """Combat states."""

    ACTIVE = "active"
    PAUSED = "paused"
    ENDED = "ended"


class CombatActionType(str, Enum):
    """Types of combat actions."""

    ATTACK = "attack"
    DODGE = "dodge"
    MANEUVER = "maneuver"
    FIGHT_BACK = "fight_back"
    FULL_AUTO = "full_auto"


class CombatantRole(str, Enum):
    """Role in combat."""

    PC = "pc"
    NPC = "npc"
    ALLY = "ally"


class DamageType(str, Enum):
    """Types of damage."""

    LETHAL = "lethal"
    NON_LETHAL = "non_lethal"


class Combat(Base):
    """Combat session model."""

    __tablename__ = "combats"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    session_id = Column(UUID(as_uuid=True), ForeignKey("game_sessions.id"), nullable=True, index=True)

    # Combat state
    state = Column(String(20), default=CombatState.ACTIVE.value, nullable=False)
    current_round = Column(Integer, default=1, nullable=False)
    current_turn_index = Column(Integer, default=0, nullable=False)

    # Combat metadata
    location = Column(String(200), nullable=True)
    description = Column(String(500), nullable=True)
    started_at = Column(DateTime(timezone=True), server_default=func.now(), nullable=False)
    ended_at = Column(DateTime(timezone=True), nullable=True)

    # Optional: JSON for storing complex combat state
    combat_metadata = Column(JSON, nullable=True, default=dict)

    # Relationships
    combatants = relationship("Combatant", back_populates="combat", cascade="all, delete-orphan")

    def __repr__(self) -> str:
        return f"<Combat {self.id} round={self.current_round} state={self.state}>"


class Combatant(Base):
    """Individual combatant in a combat session."""

    __tablename__ = "combatants"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    combat_id = Column(UUID(as_uuid=True), ForeignKey("combats.id"), nullable=False, index=True)

    # Reference to character/NPC (optional - can be anonymous enemies)
    character_id = Column(Integer, ForeignKey("characters.id"), nullable=True, index=True)

    # Combatant info
    name = Column(String(100), nullable=False)
    role = Column(String(20), nullable=False)  # PC, NPC, ALLY

    # Initiative - in CoC 7e, determined by DEX roll
    initiative = Column(Integer, nullable=False, default=0)
    dex = Column(Integer, nullable=False, default=50)

    # Current HP
    hp = Column(Integer, nullable=False)
    hp_max = Column(Integer, nullable=False)

    # Status flags
    is_active = Column(Boolean, default=True, nullable=False)
    is_dying = Column(Boolean, default=False, nullable=False)
    has_major_wound = Column(Boolean, default=False, nullable=False)
    is_unconscious = Column(Boolean, default=False, nullable=False)

    # Positioning
    position = Column(String(100), nullable=True)  # e.g. "front", "flank", "rear"

    # JSON for storing temporary modifiers, conditions, etc.
    combatant_metadata = Column(JSON, nullable=True, default=dict)

    # Timestamps
    created_at = Column(DateTime(timezone=True), server_default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())

    # Relationships
    combat = relationship("Combat", back_populates="combatants")

    def __repr__(self) -> str:
        return f"<Combatant {self.name} init={self.initiative} hp={self.hp}/{self.hp_max}>"


class CombatAction(Base):
    """Record of a combat action."""

    __tablename__ = "combat_actions"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    combat_id = Column(UUID(as_uuid=True), ForeignKey("combats.id"), nullable=False, index=True)
    round = Column(Integer, nullable=False)
    turn_order = Column(Integer, nullable=False)

    # Actor (can be null for system actions like healing)
    actor_id = Column(UUID(as_uuid=True), ForeignKey("combatants.id"), nullable=True)
    target_id = Column(UUID(as_uuid=True), ForeignKey("combatants.id"), nullable=True)

    # Action type and result
    action_type = Column(String(50), nullable=False)
    skill_used = Column(String(50), nullable=True)  # e.g. "fighting", "firearms"

    # Roll data
    roll_value = Column(Integer, nullable=True)
    skill_value = Column(Integer, nullable=True)
    success_level = Column(String(20), nullable=True)  # extreme, hard, regular, failure

    # Damage (if applicable)
    damage_amount = Column(Integer, nullable=True)
    damage_type = Column(String(20), nullable=True)
    target_hp_after = Column(Integer, nullable=True)

    # JSON for storing additional action details
    details = Column(JSON, nullable=True, default=dict)

    # Timestamp
    created_at = Column(DateTime(timezone=True), server_default=func.now(), nullable=False)

    def __repr__(self) -> str:
        return f"<CombatAction {self.action_type} round={self.round}>"
