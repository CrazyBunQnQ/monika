"""Combat service for CoC 7e combat system."""
import uuid
from datetime import datetime
from typing import Optional, List, Dict, Any
from enum import Enum

from sqlalchemy.orm import Session

from src.models.combat import (
    Combat,
    Combatant,
    CombatAction,
    CombatState,
    CombatActionType,
    DamageType,
    CombatantRole,
)
from src.models.character import Character
from src.models.event import Event, EventType
from src.services.dice import roll_check, BonusPenalty, SuccessLevel


class CombatService:
    """Service for managing combat sessions."""

    def __init__(self, db: Session):
        self.db = db

    def create_combat(
        self,
        session_id: uuid.UUID,
        location: Optional[str] = None,
        description: Optional[str] = None,
    ) -> Combat:
        """Create a new combat session.

        Args:
            session_id: Game session UUID
            location: Combat location
            description: Combat description

        Returns:
            Created Combat instance
        """
        combat = Combat(
            session_id=session_id,
            location=location,
            description=description,
            state=CombatState.ACTIVE.value,
        )
        self.db.add(combat)
        self.db.commit()
        self.db.refresh(combat)
        return combat

    def add_combatant(
        self,
        combat_id: uuid.UUID,
        name: str,
        hp: int,
        hp_max: int,
        dex: int = 50,
        role: str = CombatantRole.NPC.value,
        character_id: Optional[int] = None,
    ) -> Combatant:
        """Add a combatant to a combat session.

        Args:
            combat_id: Combat UUID
            name: Combatant name
            hp: Current HP
            hp_max: Maximum HP
            dex: Dexterity for initiative
            role: PC, NPC, or ALLY
            character_id: Optional character reference

        Returns:
            Created Combatant instance
        """
        # Roll initiative (D100 roll, take higher roll as better in CoC 7e)
        initiative_roll = roll_check(skill=dex)
        initiative = initiative_roll.value

        combatant = Combatant(
            combat_id=combat_id,
            character_id=character_id,
            name=name,
            role=role,
            dex=dex,
            initiative=initiative,
            hp=hp,
            hp_max=hp_max,
        )
        self.db.add(combatant)
        self.db.commit()
        self.db.refresh(combatant)
        return combatant

    def start_combat(self, combat_id: uuid.UUID) -> Combat:
        """Start combat by rolling initiatives and setting turn order.

        Args:
            combat_id: Combat UUID

        Returns:
            Updated Combat instance
        """
        combat = self.db.query(Combat).filter(Combat.id == combat_id).first()
        if not combat:
            raise ValueError(f"Combat {combat_id} not found")

        # Roll initiative for all combatants who haven't rolled yet
        combatants = (
            self.db.query(Combatant)
            .filter(Combatant.combat_id == combat_id, Combatant.initiative == 0)
            .all()
        )

        for combatant in combatants:
            roll = roll_check(skill=combatant.dex)
            combatant.initiative = roll.value

        self.db.commit()
        self.db.refresh(combat)
        return combat

    def get_turn_order(self, combat_id: uuid.UUID) -> List[Combatant]:
        """Get combatants sorted by initiative (highest first).

        Args:
            combat_id: Combat UUID

        Returns:
            List of combatants in initiative order
        """
        combatants = (
            self.db.query(Combatant)
            .filter(Combatant.combat_id == combat_id, Combatant.is_active == True)
            .order_by(Combatant.initiative.desc())
            .all()
        )
        return combatants

    def get_current_combatant(self, combat_id: uuid.UUID) -> Optional[Combatant]:
        """Get the combatant whose turn it currently is.

        Args:
            combat_id: Combat UUID

        Returns:
            Current combatant or None
        """
        combat = self.db.query(Combat).filter(Combat.id == combat_id).first()
        if not combat:
            return None

        turn_order = self.get_turn_order(combat_id)
        if not turn_order:
            return None

        # Wrap turn index around
        idx = combat.current_turn_index % len(turn_order)
        return turn_order[idx]

    def next_turn(self, combat_id: uuid.UUID) -> Dict[str, Any]:
        """Advance to the next combatant's turn.

        Args:
            combat_id: Combat UUID

        Returns:
            Dict with current_round, current_combatant, and is_new_round
        """
        combat = self.db.query(Combat).filter(Combat.id == combat_id).first()
        if not combat:
            raise ValueError(f"Combat {combat_id} not found")

        turn_order = self.get_turn_order(combat_id)
        if not turn_order:
            raise ValueError("No active combatants")

        # Check if advancing will start a new round
        would_be_new_round = (combat.current_turn_index + 1) >= len(turn_order)

        if would_be_new_round:
            # New round: reset to first combatant
            combat.current_round += 1
            combat.current_turn_index = 0
            is_new_round = True
        else:
            # Same round: advance to next combatant
            combat.current_turn_index += 1
            is_new_round = False

        self.db.commit()
        self.db.refresh(combat)

        current = self.get_current_combatant(combat_id)

        return {
            "combat_id": str(combat.id),
            "current_round": combat.current_round,
            "current_turn_index": combat.current_turn_index,
            "current_combatant": current.to_dict() if current else None,
            "is_new_round": is_new_round,
            "turn_order": [c.to_dict() for c in turn_order],
        }

    def resolve_attack(
        self,
        combat_id: uuid.UUID,
        attacker_id: uuid.UUID,
        target_id: uuid.UUID,
        attack_skill: int,
        attack_roll: Optional[int] = None,
        damage_roll: Optional[int] = None,
        damage_bonus: int = 0,
    ) -> Dict[str, Any]:
        """Resolve an attack in combat.

        Args:
            combat_id: Combat UUID
            attacker_id: Attacker combatant UUID
            target_id: Target combatant UUID
            attack_skill: Attacker's attack skill value
            attack_roll: Optional fixed roll value
            damage_roll: Optional fixed damage roll
            damage_bonus: DB from attacker's strength

        Returns:
            Dict with attack result, damage, and updated HP
        """
        # Get combatant and target
        attacker = (
            self.db.query(Combatant)
            .filter(Combatant.id == attacker_id, Combatant.combat_id == combat_id)
            .first()
        )
        target = (
            self.db.query(Combatant)
            .filter(Combatant.id == target_id, Combatant.combat_id == combat_id)
            .first()
        )

        if not attacker or not target:
            raise ValueError("Attacker or target not found")

        if not attacker.is_active or not target.is_active:
            raise ValueError("Attacker or target is not active")

        combat = self.db.query(Combat).filter(Combat.id == combat_id).first()

        # Roll attack
        attack_result = roll_check(skill=attack_skill, roll=attack_roll)

        # Record the action
        action = CombatAction(
            combat_id=combat_id,
            round=combat.current_round,
            turn_order=combat.current_turn_index,
            actor_id=attacker_id,
            target_id=target_id,
            action_type=CombatActionType.ATTACK.value,
            roll_value=attack_result.value,
            skill_value=attack_skill,
            success_level=attack_result.success_level.value,
        )
        self.db.add(action)

        result: Dict[str, Any] = {
            "attacker": attacker.name,
            "target": target.name,
            "attack_roll": attack_result.value,
            "attack_skill": attack_skill,
            "success_level": attack_result.success_level.value,
            "hit": False,
            "damage": 0,
            "target_hp_before": target.hp,
            "target_hp_after": target.hp,
            "target_status": "active",
        }

        # Check if hit (attacker succeeded OR target failed)
        # For now, simple version: hit if attacker succeeds
        if attack_result.success_level != SuccessLevel.FAILURE:
            result["hit"] = True

            # Roll damage
            # In CoC, damage is usually: weapon roll + damage bonus
            # For simplicity, using 1d6 + DB
            if damage_roll is None:
                import random
                damage_roll = random.randint(1, 6)

            total_damage = max(1, damage_roll + damage_bonus)

            # Apply damage
            target.hp = max(0, target.hp - total_damage)
            result["damage_roll"] = damage_roll
            result["damage_bonus"] = damage_bonus
            result["damage"] = total_damage
            result["target_hp_after"] = target.hp

            action.damage_amount = total_damage
            action.damage_type = DamageType.LETHAL.value
            action.target_hp_after = target.hp

            # Check for major wound (HP <= 0 and took > max/2 damage in one hit)
            if total_damage > target.hp_max // 2:
                target.has_major_wound = True

            # Check for dying/death
            if target.hp <= 0:
                target.is_dying = True
                target.is_active = False
                result["target_status"] = "dying"
            elif target.hp <= -target.hp_max:
                # Dead
                target.hp = -target.hp_max
                result["target_status"] = "dead"

        self.db.commit()
        self.db.refresh(action)

        result["action_id"] = str(action.id)
        return result

    def heal(
        self,
        combat_id: uuid.UUID,
        target_id: uuid.UUID,
        heal_amount: int,
        first_aid_skill: int,
        first_aid_roll: Optional[int] = None,
    ) -> Dict[str, Any]:
        """Apply healing to a combatant.

        Args:
            combat_id: Combat UUID
            target_id: Target combatant UUID
            heal_amount: Base healing amount
            first_aid_skill: First Aid skill for healer
            first_aid_roll: Optional fixed roll

        Returns:
            Dict with healing result
        """
        target = (
            self.db.query(Combatant)
            .filter(Combatant.id == target_id, Combatant.combat_id == combat_id)
            .first()
        )

        if not target:
            raise ValueError("Target not found")

        combat = self.db.query(Combat).filter(Combat.id == combat_id).first()

        # Roll First Aid
        first_aid_result = roll_check(skill=first_aid_skill, roll=first_aid_roll)

        result: Dict[str, Any] = {
            "target": target.name,
            "first_aid_roll": first_aid_result.value,
            "first_aid_skill": first_aid_skill,
            "success_level": first_aid_result.success_level.value,
            "hp_before": target.hp,
            "healing": 0,
            "hp_after": target.hp,
        }

        # Success determines healing amount
        if first_aid_result.success_level != SuccessLevel.FAILURE:
            # Success: heal 1 HP, Hard: heal 1d3+1, Extreme: heal 1d6+2
            import random

            if first_aid_result.success_level == SuccessLevel.EXTREME_SUCCESS:
                healing = random.randint(1, 6) + 2
            elif first_aid_result.success_level == SuccessLevel.HARD_SUCCESS:
                healing = random.randint(1, 3) + 1
            else:
                healing = 1

            healing = max(healing, heal_amount)  # Use the better of roll or base
            target.hp = min(target.hp_max, target.hp + healing)
            result["healing"] = healing
            result["hp_after"] = target.hp

            # Recovery from major wound/dying
            if target.hp > 0:
                target.is_dying = False
                target.is_active = True

        # Record action
        action = CombatAction(
            combat_id=combat_id,
            round=combat.current_round,
            turn_order=combat.current_turn_index,
            target_id=target_id,
            action_type="heal",
            roll_value=first_aid_result.value,
            skill_value=first_aid_skill,
            success_level=first_aid_result.success_level.value,
            details={"healing": result.get("healing", 0)},
        )
        self.db.add(action)
        self.db.commit()

        result["action_id"] = str(action.id)
        return result

    def end_combat(self, combat_id: uuid.UUID) -> Combat:
        """End a combat session.

        Args:
            combat_id: Combat UUID

        Returns:
            Updated Combat instance
        """
        combat = self.db.query(Combat).filter(Combat.id == combat_id).first()
        if not combat:
            raise ValueError(f"Combat {combat_id} not found")

        combat.state = CombatState.ENDED.value
        combat.ended_at = datetime.utcnow()
        self.db.commit()
        self.db.refresh(combat)
        return combat

    def get_combat_summary(self, combat_id: uuid.UUID) -> Dict[str, Any]:
        """Get a summary of the combat session.

        Args:
            combat_id: Combat UUID

        Returns:
            Combat summary dict
        """
        combat = self.db.query(Combat).filter(Combat.id == combat_id).first()
        if not combat:
            raise ValueError(f"Combat {combat_id} not found")

        combatants = self.get_turn_order(combat_id)
        actions = (
            self.db.query(CombatAction)
            .filter(CombatAction.combat_id == combat_id)
            .order_by(CombatAction.round.asc(), CombatAction.turn_order.asc())
            .all()
        )

        return {
            "id": str(combat.id),
            "state": combat.state,
            "round": combat.current_round,
            "location": combat.location,
            "description": combat.description,
            "started_at": combat.started_at.isoformat() if combat.started_at else None,
            "ended_at": combat.ended_at.isoformat() if combat.ended_at else None,
            "combatants": [c.to_dict() for c in combatants],
            "total_actions": len(actions),
            "current_turn": self.get_current_combatant(combat_id),
        }


# Extend Combatant model with to_dict method
def combatant_to_dict(self) -> Dict[str, Any]:
    """Convert combatant to dictionary."""
    return {
        "id": str(self.id),
        "name": self.name,
        "role": self.role,
        "initiative": self.initiative,
        "dex": self.dex,
        "hp": self.hp,
        "hp_max": self.hp_max,
        "is_active": self.is_active,
        "is_dying": self.is_dying,
        "has_major_wound": self.has_major_wound,
        "is_unconscious": self.is_unconscious,
        "position": self.position,
        "character_id": self.character_id,
    }


# Monkey patch the to_dict method onto the Combatant model
Combatant.to_dict = combatant_to_dict
