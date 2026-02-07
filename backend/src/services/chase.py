"""Chase service for CoC 7e chase system."""
import uuid
import random
from typing import Optional, List, Dict, Any
from datetime import datetime

from sqlalchemy.orm import Session

from src.models.chase import (
    Chase,
    ChaseParticipant,
    ChaseObstacle,
    ChaseAction,
    ChaseState,
    ChaseEndReason,
    ChaseParticipantRole,
    ObstacleType,
)
from src.services.dice import roll_check, BonusPenalty, SuccessLevel


class ChaseService:
    """Service for managing chase sessions."""

    def __init__(self, db: Session):
        self.db = db

    def create_chase(
        self,
        session_id: uuid.UUID,
        location: str,
        setting: str = "city_streets",
    ) -> Chase:
        """Create a new chase session.

        Args:
            session_id: Game session UUID
            location: Where the chase takes place
            setting: Type of terrain/environment

        Returns:
            Created Chase instance
        """
        chase = Chase(
            session_id=session_id,
            location=location,
            setting=setting,
            state=ChaseState.ACTIVE.value,
        )
        self.db.add(chase)
        self.db.commit()
        self.db.refresh(chase)
        return chase

    def add_participant(
        self,
        chase_id: uuid.UUID,
        name: str,
        role: str,
        move_rate: int = 8,
        is_player: bool = False,
        character_id: Optional[int] = None,
    ) -> ChaseParticipant:
        """Add a participant to a chase.

        Args:
            chase_id: Chase UUID
            name: Participant name
            role: fugitive or pursuer
            move_rate: Movement rate (usually 8, or 9 if DEX >= SIZ and STR >= SIZ)
            is_player: Whether this is a player character
            character_id: Optional character reference

        Returns:
            Created ChaseParticipant instance
        """
        # Set initial position based on role
        # Fugitives start at positive position, pursuers at negative
        if role == ChaseParticipantRole.FUGITIVE.value:
            position_index = 0
        else:  # pursuer
            position_index = 0

        participant = ChaseParticipant(
            chase_id=chase_id,
            name=name,
            role=role,
            move_rate=move_rate,
            current_speed=move_rate,
            is_player=is_player,
            character_id=character_id,
            position_index=position_index,
        )
        self.db.add(participant)
        self.db.commit()
        self.db.refresh(participant)
        return participant

    def generate_obstacle(
        self,
        chase_id: uuid.UUID,
        round: int,
        distance_level: int,
    ) -> ChaseObstacle:
        """Generate a random obstacle based on chase setting and distance.

        Args:
            chase_id: Chase UUID
            round: Current round number
            distance_level: Current distance level (affects obstacle difficulty)

        Returns:
            Generated ChaseObstacle instance
        """
        chase = self.db.query(Chase).filter(Chase.id == chase_id).first()
        if not chase:
            raise ValueError(f"Chase {chase_id} not found")

        # Obstacle templates based on setting
        obstacle_templates = self._get_obstacle_templates(chase.setting)

        # Select an appropriate obstacle
        template = random.choice(obstacle_templates)

        # Difficulty scales with distance
        difficulty = self._get_difficulty_for_distance(distance_level)

        obstacle = ChaseObstacle(
            chase_id=chase_id,
            name=template["name"],
            description=template["description"],
            obstacle_type=template["type"],
            appears_at_round=round,
            appears_at_distance=distance_level,
            difficulty=difficulty,
            skill_required=template.get("skill"),
            failure_penalty=template.get("penalty", 1),
            failure_damage=template.get("damage"),
            fail_forward_result=template.get("fail_forward"),
        )
        self.db.add(obstacle)
        self.db.commit()
        self.db.refresh(obstacle)
        return obstacle

    def resolve_round(
        self,
        chase_id: uuid.UUID,
        actions: List[Dict[str, Any]],
    ) -> Dict[str, Any]:
        """Resolve a round of chase actions.

        Args:
            chase_id: Chase UUID
            actions: List of actions taken by participants

        Returns:
            Dict with round results and current chase state
        """
        chase = self.db.query(Chase).filter(Chase.id == chase_id).first()
        if not chase:
            raise ValueError(f"Chase {chase_id} not found")

        if chase.state != ChaseState.ACTIVE.value:
            raise ValueError(f"Chase is not active: {chase.state}")

        round_results = []

        for action_data in actions:
            action = self._resolve_action(chase, action_data, chase.current_round)
            round_results.append(action)

        # Calculate positions after all actions
        positions = self._calculate_positions(chase_id)

        # Update round
        chase.current_round += 1
        self.db.commit()

        # Check if chase should end
        end_check = self._check_chase_end(chase_id, positions)

        return {
            "chase_id": str(chase.id),
            "round": chase.current_round - 1,
            "actions": round_results,
            "positions": positions,
            "chase_ended": end_check["ended"],
            "end_reason": end_check.get("reason"),
        }

    def _resolve_action(
        self, chase: Chase, action_data: Dict[str, Any], round: int
    ) -> Dict[str, Any]:
        """Resolve a single chase action.

        Args:
            chase: Chase instance
            action_data: Action data from request
            round: Current round

        Returns:
            Action result dict
        """
        participant_id = action_data["participant_id"]
        action_type = action_data["action_type"]

        participant = (
            self.db.query(ChaseParticipant)
            .filter(ChaseParticipant.id == participant_id, ChaseParticipant.chase_id == chase.id)
            .first()
        )

        if not participant:
            raise ValueError(f"Participant {participant_id} not found")

        result: Dict[str, Any] = {
            "participant": participant.name,
            "action_type": action_type,
            "success": False,
        }

        action_record = ChaseAction(
            chase_id=chase.id,
            round=round,
            participant_id=participant_id,
            action_type=action_type,
        )

        if action_type == "accelerate":
            # Risk roll - success = +1 speed, failure = -1 speed and must roll next round
            roll = roll_check(skill=participant.current_speed)
            action_record.roll_value = roll.value
            action_record.skill_value = participant.current_speed
            action_record.success_level = roll.success_level.value

            if roll.success_level != SuccessLevel.FAILURE:
                participant.current_speed += 1
                result["success"] = True
                result["speed_change"] = 1
            else:
                participant.current_speed = max(1, participant.current_speed - 1)
                participant.consecutive_failures += 1
                result["speed_change"] = -1
                result["must_risk_next"] = True

        elif action_type == "decouple":
            # Or "slow down" - detach from pack (action must follow failed risk)
            participant.current_speed = max(1, participant.current_speed - 1)
            result["speed_change"] = -1

        elif action_type == "overcome_obstacle":
            obstacle_id = action_data.get("obstacle_id")
            if not obstacle_id:
                raise ValueError("obstacle_id required for overcome_obstacle action")

            obstacle = (
                self.db.query(ChaseObstacle)
                .filter(ChaseObstacle.id == obstacle_id, ChaseObstacle.chase_id == chase.id)
                .first()
            )

            if not obstacle:
                raise ValueError(f"Obstacle {obstacle_id} not found")

            # Roll skill check
            skill = action_data.get("skill", participant.current_speed)
            roll = roll_check(skill=skill)

            action_record.obstacle_id = obstacle_id
            action_record.roll_value = roll.value
            action_record.skill_value = skill
            action_record.success_level = roll.success_level.value

            if roll.success_level != SuccessLevel.FAILURE:
                result["success"] = True
                result["obstacle_overcome"] = True
            else:
                participant.failed_obstacle_count += 1
                participant.speed_penalty += obstacle.failure_penalty
                participant.current_speed = max(
                    1, participant.current_speed - obstacle.failure_penalty
                )
                result["obstacle_overcome"] = False
                result["penalty"] = obstacle.failure_penalty

                # Apply damage if specified
                if obstacle.failure_damage:
                    result["damage_taken"] = obstacle.failure_damage

                # Fail forward
                if obstacle.fail_forward_result:
                    result["fail_forward"] = obstacle.fail_forward_result

        elif action_type == "attack":
            # Brief combat in chase
            # For now, simplified - just reduce speed
            participant.current_speed = max(1, participant.current_speed - 2)
            result["speed_change"] = -2
            result["description"] = "Brief combat slows you down"

        action_record.speed_change = result.get("speed_change", 0)
        self.db.add(action_record)
        self.db.commit()

        result["action_id"] = str(action_record.id)
        return result

    def _calculate_positions(self, chase_id: uuid.UUID) -> List[Dict[str, Any]]:
        """Calculate relative positions of all participants.

        Args:
            chase_id: Chase UUID

        Returns:
            List of participants ordered by position
        """
        participants = (
            self.db.query(ChaseParticipant)
            .filter(ChaseParticipant.chase_id == chase_id, ChaseParticipant.is_active == True)
            .all()
        )

        # Calculate effective position based on speed and penalties
        for p in participants:
            effective_speed = max(1, p.current_speed - p.speed_penalty)
            # Update position index (simplified - full system uses more complex calc)
            # For each round, participants move based on their effective speed
            if p.role == ChaseParticipantRole.FUGITIVE.value:
                p.position_index += effective_speed
            else:
                p.position_index -= effective_speed

        self.db.commit()

        # Return ordered list
        ordered = sorted(participants, key=lambda p: p.position_index, reverse=True)
        return [
            {
                "id": str(p.id),
                "name": p.name,
                "role": p.role,
                "position": p.position_index,
                "speed": p.current_speed,
                "penalty": p.speed_penalty,
            }
            for p in ordered
        ]

    def _check_chase_end(
        self, chase_id: uuid.UUID, positions: List[Dict[str, Any]]
    ) -> Dict[str, Any]:
        """Check if the chase should end.

        Args:
            chase_id: Chase UUID
            positions: Current positions

        Returns:
            Dict with ended flag and optional reason
        """
        participants = (
            self.db.query(ChaseParticipant)
            .filter(ChaseParticipant.chase_id == chase_id, ChaseParticipant.is_active == True)
            .all()
        )

        fugitives = [p for p in participants if p.role == ChaseParticipantRole.FUGITIVE.value]
        pursuers = [p for p in participants if p.role == ChaseParticipantRole.PURSUER.value]

        if not fugitives or not pursuers:
            # One side eliminated
            chase = self.db.query(Chase).filter(Chase.id == chase_id).first()
            chase.state = ChaseState.ENDED.value
            chase.ended_at = datetime.utcnow()
            if not fugitives:
                chase.end_reason = ChaseEndReason.CAUGHT.value
            else:
                chase.end_reason = ChaseEndReason.ESCAPED.value
            self.db.commit()
            return {"ended": True, "reason": chase.end_reason}

        # Check distance - if fugitives are >20 ahead, they escape
        # If pursuers catch up (positions cross), chase may end
        max_fugitive_pos = max(p.position_index for p in fugitives)
        min_pursuer_pos = min(p.position_index for p in pursuers)

        # Fugitives escape if significantly ahead
        if max_fugitive_pos - min_pursuer_pos > 20:
            chase = self.db.query(Chase).filter(Chase.id == chase_id).first()
            chase.state = ChaseState.ENDED.value
            chase.ended_at = datetime.utcnow()
            chase.end_reason = ChaseEndReason.ESCAPED.value
            self.db.commit()
            return {"ended": True, "reason": ChaseEndReason.ESCAPED.value}

        # Pursuers catch fugitives
        if max_fugitive_pos <= min_pursuer_pos:
            chase = self.db.query(Chase).filter(Chase.id == chase_id).first()
            chase.state = ChaseState.ENDED.value
            chase.ended_at = datetime.utcnow()
            chase.end_reason = ChaseEndReason.CAUGHT.value
            self.db.commit()
            return {"ended": True, "reason": ChaseEndReason.CAUGHT.value}

        return {"ended": False}

    def end_chase(
        self,
        chase_id: uuid.UUID,
        reason: str,
        fail_forward_scene: Optional[str] = None,
    ) -> Chase:
        """Manually end a chase.

        Args:
            chase_id: Chase UUID
            reason: Why the chase ended
            fail_forward_scene: Optional description of what happens next

        Returns:
            Updated Chase instance
        """
        chase = self.db.query(Chase).filter(Chase.id == chase_id).first()
        if not chase:
            raise ValueError(f"Chase {chase_id} not found")

        chase.state = ChaseState.ENDED.value
        chase.ended_at = datetime.utcnow()
        chase.end_reason = reason
        if fail_forward_scene:
            chase.failed_forward_scene = fail_forward_scene

        self.db.commit()
        self.db.refresh(chase)
        return chase

    def get_chase_summary(self, chase_id: uuid.UUID) -> Dict[str, Any]:
        """Get a summary of the chase session.

        Args:
            chase_id: Chase UUID

        Returns:
            Chase summary dict
        """
        chase = self.db.query(Chase).filter(Chase.id == chase_id).first()
        if not chase:
            raise ValueError(f"Chase {chase_id} not found")

        participants = (
            self.db.query(ChaseParticipant)
            .filter(ChaseParticipant.chase_id == chase_id)
            .all()
        )

        obstacles = (
            self.db.query(ChaseObstacle)
            .filter(ChaseObstacle.chase_id == chase_id)
            .order_by(ChaseObstacle.appears_at_round.asc())
            .all()
        )

        return {
            "id": str(chase.id),
            "state": chase.state,
            "round": chase.current_round,
            "location": chase.location,
            "setting": chase.setting,
            "started_at": chase.started_at.isoformat() if chase.started_at else None,
            "ended_at": chase.ended_at.isoformat() if chase.ended_at else None,
            "end_reason": chase.end_reason,
            "failed_forward_scene": chase.failed_forward_scene,
            "participants": [self._participant_to_dict(p) for p in participants],
            "obstacles": [self._obstacle_to_dict(o) for o in obstacles],
        }

    def _participant_to_dict(self, participant: ChaseParticipant) -> Dict[str, Any]:
        """Convert participant to dictionary."""
        return {
            "id": str(participant.id),
            "name": participant.name,
            "role": participant.role,
            "is_player": participant.is_player,
            "position": participant.position_index,
            "move_rate": participant.move_rate,
            "current_speed": participant.current_speed,
            "speed_penalty": participant.speed_penalty,
            "is_active": participant.is_active,
            "is_exhausted": participant.is_exhausted,
        }

    def _obstacle_to_dict(self, obstacle: ChaseObstacle) -> Dict[str, Any]:
        """Convert obstacle to dictionary."""
        return {
            "id": str(obstacle.id),
            "name": obstacle.name,
            "description": obstacle.description,
            "type": obstacle.obstacle_type,
            "difficulty": obstacle.difficulty,
            "skill_required": obstacle.skill_required,
            "failure_penalty": obstacle.failure_penalty,
            "failure_damage": obstacle.failure_damage,
            "fail_forward_result": obstacle.fail_forward_result,
        }

    def _get_obstacle_templates(self, setting: str) -> List[Dict[str, Any]]:
        """Get obstacle templates for a setting.

        Args:
            setting: Chase setting type

        Returns:
            List of obstacle templates
        """
        templates = {
            "city_streets": [
                {
                    "name": "Traffic",
                    "description": "Heavy traffic blocks the path",
                    "type": ObstacleType.ENVIRONMENTAL.value,
                    "skill": "drive",
                    "penalty": 2,
                    "fail_forward": "Vehicle collision - take damage and fall behind",
                },
                {
                    "name": "Crowd",
                    "description": "A dense crowd slows movement",
                    "type": ObstacleType.PHYSICAL.value,
                    "skill": None,
                    "penalty": 1,
                    "fail_forward": "Lost in the crowd - separated from group",
                },
                {
                    "name": "Construction Zone",
                    "description": "Road construction creates barriers",
                    "type": ObstacleType.SKILL_CHECK.value,
                    "skill": "drive",
                    "penalty": 2,
                    "fail_forward": "Vehicle disabled - must continue on foot",
                },
            ],
            "forest": [
                {
                    "name": "Dense Undergrowth",
                    "description": "Thick vegetation blocks the way",
                    "type": ObstacleType.PHYSICAL.value,
                    "skill": "athletics",
                    "penalty": 1,
                    "fail_forward": "Tangled in thorns - minor injury and delay",
                },
                {
                    "name": "Ravine",
                    "description": "A deep ravine blocks the path",
                    "type": ObstacleType.SKILL_CHECK.value,
                    "skill": "jump",
                    "penalty": 3,
                    "fail_forward": "Fall into ravine - take damage and climb out",
                },
                {
                    "name": "Rough Terrain",
                    "description": "Rocky, uneven ground slows movement",
                    "type": ObstacleType.ENVIRONMENTAL.value,
                    "skill": None,
                    "penalty": 1,
                    "fail_forward": "Twist ankle - speed reduced until healed",
                },
            ],
            "corridor": [
                {
                    "name": "Locked Door",
                    "description": "A locked door blocks the passage",
                    "type": ObstacleType.SKILL_CHECK.value,
                    "skill": "locksmith",
                    "penalty": 2,
                    "fail_forward": "Must break door down - noisy and slow",
                },
                {
                    "name": "Debris",
                    "description": "Collapsed debris blocks the way",
                    "type": ObstacleType.PHYSICAL.value,
                    "skill": "athletics",
                    "penalty": 1,
                    "fail_forward": "Injury from shifting debris",
                },
            ],
        }

        return templates.get(setting, templates["city_streets"])

    def _get_difficulty_for_distance(self, distance: int) -> str:
        """Get obstacle difficulty based on distance level.

        Args:
            distance: Current distance level (0-4)

        Returns:
            Difficulty string
        """
        # Higher distance = harder obstacles
        if distance >= 4:
            return "extreme"
        elif distance >= 2:
            return "hard"
        else:
            return "regular"
