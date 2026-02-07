"""Chase state model for CoC 7e chase system."""
import uuid
from datetime import datetime
from enum import Enum

from sqlalchemy import Column, String, Integer, DateTime, JSON, ForeignKey, UUID, Boolean
from sqlalchemy.sql import func
from sqlalchemy.orm import relationship

from src.core.database import Base


class ChaseState(str, Enum):
    """Chase states."""

    ACTIVE = "active"
    PAUSED = "paused"
    ENDED = "ended"


class ChaseEndReason(str, Enum):
    """Why a chase ended."""

    ESCAPED = "escaped"  # Fugitives escaped
    CAUGHT = "caught"  # Fugitives were caught
    ABANDONED = "abandoned"  # Chase abandoned
    FAILED_FORWARD = "failed_forward"  # Failed to new situation


class ChaseParticipantRole(str, Enum):
    """Role in the chase."""

    FUGITIVE = "fugitive"  # Running away
    PURSUER = "pursuer"  # Chasing


class ObstacleType(str, Enum):
    """Types of obstacles in a chase."""

    PHYSICAL = "physical"  # Jump gap, climb wall
    ENVIRONMENTAL = "environmental"  # Crowd, traffic, weather
    SKILL_CHECK = "skill_check"  # Drive, swim, etc.
    COMBAT = "combat"  # Brief combat encounter


class Chase(Base):
    """Chase session model."""

    __tablename__ = "chases"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    session_id = Column(UUID(as_uuid=True), ForeignKey("game_sessions.id"), nullable=True, index=True)

    # Chase state
    state = Column(String(20), default=ChaseState.ACTIVE.value, nullable=False)
    current_round = Column(Integer, default=1, nullable=False)

    # Chase configuration
    location = Column(String(200), nullable=True)
    setting = Column(String(200), nullable=True)  # e.g. "city_streets", "forest", "corridor"

    # Chase outcome
    end_reason = Column(String(50), nullable=True)
    failed_forward_scene = Column(String(500), nullable=True)

    # Timestamps
    started_at = Column(DateTime(timezone=True), server_default=func.now(), nullable=False)
    ended_at = Column(DateTime(timezone=True), nullable=True)

    # JSON for storing complex chase state
    chase_metadata = Column(JSON, nullable=True, default=dict)

    # Relationships
    participants = relationship("ChaseParticipant", back_populates="chase", cascade="all, delete-orphan")
    obstacles = relationship("ChaseObstacle", back_populates="chase", cascade="all, delete-orphan")

    def __repr__(self) -> str:
        return f"<Chase {self.id} round={self.current_round} state={self.state}>"


class ChaseParticipant(Base):
    """Participant in a chase."""

    __tablename__ = "chase_participants"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    chase_id = Column(UUID(as_uuid=True), ForeignKey("chases.id"), nullable=False, index=True)

    # Reference to character/NPC
    character_id = Column(Integer, ForeignKey("characters.id"), nullable=True, index=True)

    # Participant info
    name = Column(String(100), nullable=False)
    role = Column(String(20), nullable=False)  # fugitive or pursuer
    is_player = Column(Boolean, default=False, nullable=False)

    # Movement capabilities
    move_rate = Column(Integer, nullable=False, default=8)
    current_speed = Column(Integer, nullable=False, default=8)

    # Position in chase (relative distance)
    position_index = Column(Integer, nullable=False, default=0)
    # Higher index = further ahead (positive for fugitives ahead, negative for pursuers)

    # Status
    is_active = Column(Boolean, default=True, nullable=False)
    is_exhausted = Column(Boolean, default=False, nullable=False)
    failed_obstacle_count = Column(Integer, default=0, nullable=False)

    # Injury/fatigue penalties
    speed_penalty = Column(Integer, default=0, nullable=False)
    consecutive_failures = Column(Integer, default=0, nullable=False)

    # JSON for storing temporary modifiers, conditions
    participant_metadata = Column(JSON, nullable=True, default=dict)

    # Timestamps
    created_at = Column(DateTime(timezone=True), server_default=func.now(), nullable=False)
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())

    # Relationships
    chase = relationship("Chase", back_populates="participants")

    def __repr__(self) -> str:
        return f"<ChaseParticipant {self.name} role={self.role} pos={self.position_index}>"


class ChaseObstacle(Base):
    """Obstacle encountered during a chase."""

    __tablename__ = "chase_obstacles"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    chase_id = Column(UUID(as_uuid=True), ForeignKey("chases.id"), nullable=False, index=True)

    # Obstacle info
    name = Column(String(200), nullable=False)
    description = Column(String(500), nullable=False)
    obstacle_type = Column(String(50), nullable=False)

    # When the obstacle appears
    appears_at_round = Column(Integer, nullable=False)
    appears_at_distance = Column(Integer, nullable=False)

    # Difficulty
    difficulty = Column(String(20), nullable=False)  # regular, hard, extreme
    skill_required = Column(String(50), nullable=True)  # e.g. "drive", "athletics", "swim"

    # Consequences of failure
    failure_penalty = Column(Integer, default=1, nullable=False)  # Speed penalty
    failure_damage = Column(Integer, nullable=True)
    failure_san_cost = Column(Integer, nullable=True)
    fail_forward_result = Column(String(500), nullable=True)

    # JSON for storing additional data
    details = Column(JSON, nullable=True, default=dict)

    # Timestamp
    created_at = Column(DateTime(timezone=True), server_default=func.now(), nullable=False)

    # Relationships
    chase = relationship("Chase", back_populates="obstacles")

    def __repr__(self) -> str:
        return f"<ChaseObstacle {self.name} type={self.obstacle_type}>"


class ChaseAction(Base):
    """Record of an action taken during a chase."""

    __tablename__ = "chase_actions"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    chase_id = Column(UUID(as_uuid=True), ForeignKey("chases.id"), nullable=False, index=True)
    round = Column(Integer, nullable=False)

    # Who acted
    participant_id = Column(UUID(as_uuid=True), ForeignKey("chase_participants.id"), nullable=False)
    obstacle_id = Column(UUID(as_uuid=True), ForeignKey("chase_obstacles.id"), nullable=True)

    # Action taken
    action_type = Column(String(50), nullable=False)  # accelerate, decelerate, overcome_obstacle, attack

    # Roll data
    skill_used = Column(String(50), nullable=True)
    roll_value = Column(Integer, nullable=True)
    skill_value = Column(Integer, nullable=True)
    success_level = Column(String(20), nullable=True)

    # Result
    speed_change = Column(Integer, nullable=False, default=0)
    position_change = Column(Integer, nullable=False, default=0)
    damage_taken = Column(Integer, nullable=True)
    san_lost = Column(Integer, nullable=True)

    # JSON for storing additional details
    details = Column(JSON, nullable=True, default=dict)

    # Timestamp
    created_at = Column(DateTime(timezone=True), server_default=func.now(), nullable=False)

    def __repr__(self) -> str:
        return f"<ChaseAction {self.action_type} round={self.round}>"
