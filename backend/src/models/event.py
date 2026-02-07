"""Event database model for append-only audit logging."""
from datetime import datetime
import uuid
from enum import Enum

from sqlalchemy import Column, String, DateTime, Integer, Text, ForeignKey, JSON, Enum as SQLEnum
from sqlalchemy.sql import func
try:
    from sqlalchemy.dialects.postgresql import UUID
except ImportError:
    from sqlalchemy import String
    UUID = String

from src.core.database import Base


class VisibilityLevel(str, Enum):
    """Visibility levels for events and data."""

    PUBLIC = "public"
    KP_ONLY = "kp"
    PLAYER_PREFIX = "player:"


class EventType(str, Enum):
    """Types of game events for audit trail."""

    # Dice rolls
    ROLL = "roll"
    PUSH_ROLL = "push_roll"
    LUCK_SPEND = "luck_spend"

    # SAN and mental
    SAN_CHECK = "san_check"
    SAN_LOSS = "san_loss"
    INSANITY_GAIN = "insanity_gain"

    # Combat
    COMBAT_START = "combat_start"
    COMBAT_END = "combat_end"
    COMBAT_ROUND = "combat_round"
    DAMAGE = "damage"
    HEAL = "heal"

    # Chase
    CHASE_START = "chase_start"
    CHASE_END = "chase_end"
    CHASE_ROUND = "chase_round"
    CHASE_OBSTACLE = "chase_obstacle"

    # State changes
    HP_CHANGE = "hp_change"
    MP_CHANGE = "mp_change"
    SAN_CHANGE = "san_change"
    LUCK_CHANGE = "luck_change"

    # Narrative
    MESSAGE = "message"
    SCENE_CHANGE = "scene_change"
    NPC_APPEAR = "npc_appear"

    # System
    SESSION_START = "session_start"
    SESSION_END = "session_end"
    RETCON = "retcon"


class Event(Base):
    """Append-only event log for game audit trail.

    All game state changes must be recorded as events.
    Events are immutable - never updated, only appended.
    """

    __tablename__ = "events"

    # Primary key - UUID for distributed systems support
    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)

    # Session this event belongs to
    session_id = Column(UUID(as_uuid=True), ForeignKey("game_sessions.id"), nullable=True, index=True)

    # Who triggered the event
    actor_player_id = Column(Integer, ForeignKey("users.id"), nullable=True, index=True)
    actor_role = Column(SQLEnum("kp", "player", "system", name="actor_role"), nullable=False)

    # Which character was affected (if applicable)
    character_id = Column(Integer, ForeignKey("characters.id"), nullable=True, index=True)

    # Event type
    event_type = Column(SQLEnum(EventType, name="event_type", create_constraint=False, native_enum=False), nullable=False, index=True)

    # Event payload - structured data specific to event type
    # Examples:
    # - roll: {skill, target, roll_value, success_level, bonus_dice, penalty_dice}
    # - damage: {target_id, amount, source, damage_type}
    # - san_check: {reason, difficulty, roll, loss_amount}
    payload = Column(JSON, nullable=False, default=dict)

    # Visibility: who can see this event
    visibility = Column(
        SQLEnum(VisibilityLevel, name="visibility_level", create_constraint=False, native_enum=False),
        nullable=False,
        default=VisibilityLevel.PUBLIC
    )

    # Timestamp - immutable, set at creation
    timestamp = Column(DateTime(timezone=True), server_default=func.now(), nullable=False)

    # Optional: reference to related event (e.g., a push_roll references the original roll)
    parent_event_id = Column(UUID(as_uuid=True), nullable=True, index=True)

    # Optional: human-readable description for quick viewing
    description = Column(String(500), nullable=True)

    def __repr__(self) -> str:
        return f"<Event {self.event_type.value} {self.id} at {self.timestamp}>"

    def to_dict(self) -> dict:
        """Convert event to dictionary for API responses."""
        return {
            "id": str(self.id),
            "session_id": str(self.session_id) if self.session_id else None,
            "actor_player_id": self.actor_player_id,
            "actor_role": self.actor_role,
            "character_id": self.character_id,
            "event_type": self.event_type.value,
            "payload": self.payload,
            "visibility": self.visibility.value,
            "timestamp": self.timestamp.isoformat(),
            "parent_event_id": str(self.parent_event_id) if self.parent_event_id else None,
            "description": self.description,
        }
