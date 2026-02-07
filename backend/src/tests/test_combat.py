"""Tests for combat service."""
import uuid
from datetime import datetime

import pytest
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker

from src.core.database import Base
from src.models.combat import Combat, Combatant, CombatAction, CombatState, CombatantRole
from src.services.combat import CombatService


@pytest.fixture
def test_db():
    """Create a test database."""
    engine = create_engine("sqlite:///:memory:")
    TestingSessionLocal = sessionmaker(bind=engine)
    Base.metadata.create_all(bind=engine)

    db = TestingSessionLocal()
    try:
        yield db
    finally:
        db.close()


@pytest.fixture
def session_id():
    """Return a test session UUID."""
    return uuid.uuid4()


class TestCombatService:
    """Test CombatService."""

    def test_create_combat(self, test_db, session_id):
        """Test creating a combat session."""
        service = CombatService(test_db)

        combat = service.create_combat(
            session_id=session_id,
            location="Dark Alley",
            description="Ambushed by cultists",
        )

        assert combat.id is not None
        assert combat.session_id == session_id
        assert combat.location == "Dark Alley"
        assert combat.state == CombatState.ACTIVE.value
        assert combat.current_round == 1

    def test_add_combatant(self, test_db, session_id):
        """Test adding combatants to combat."""
        service = CombatService(test_db)

        combat = service.create_combat(session_id=session_id)

        # Add player character
        pc = service.add_combatant(
            combat_id=combat.id,
            name="Investigator",
            hp=12,
            hp_max=12,
            dex=60,
            role=CombatantRole.PC.value,
        )

        assert pc.name == "Investigator"
        assert pc.hp == 12
        assert pc.initiative > 0  # Should have rolled

        # Add enemy
        enemy = service.add_combatant(
            combat_id=combat.id,
            name="Cultist",
            hp=10,
            hp_max=10,
            dex=50,
            role=CombatantRole.NPC.value,
        )

        assert enemy.name == "Cultist"
        assert enemy.role == CombatantRole.NPC.value

    def test_get_turn_order(self, test_db, session_id):
        """Test getting combatants in initiative order."""
        service = CombatService(test_db)

        combat = service.create_combat(session_id=session_id)

        combatants = []
        for i in range(3):
            c = service.add_combatant(
                combat_id=combat.id,
                name=f"Combatant{i}",
                hp=10,
                hp_max=10,
                dex=50 + i * 10,  # Different DEX for different initiative
            )
            combatants.append(c)

        turn_order = service.get_turn_order(combat.id)

        # Should be sorted by initiative (highest first)
        assert len(turn_order) == 3
        # Combatants should have different initiatives (random)
        initiatives = [c.initiative for c in turn_order]
        # Initiatives should be sorted in descending order
        assert initiatives == sorted(initiatives, reverse=True)

    def test_next_turn(self, test_db, session_id):
        """Test advancing turns."""
        service = CombatService(test_db)

        combat = service.create_combat(session_id=session_id)

        service.add_combatant(
            combat_id=combat.id, name="A", hp=10, hp_max=10, dex=60
        )
        service.add_combatant(
            combat_id=combat.id, name="B", hp=10, hp_max=10, dex=50
        )

        turn_order = service.get_turn_order(combat.id)
        assert len(turn_order) == 2

        # First turn: index 0 -> 1
        turn1 = service.next_turn(combat.id)
        assert turn1["current_turn_index"] == 1
        assert turn1["is_new_round"] is False
        assert turn1["current_round"] == 1

        # Second turn: index 1 -> wraps to 0, new round
        turn2 = service.next_turn(combat.id)
        assert turn2["current_turn_index"] == 0
        assert turn2["is_new_round"] is True
        assert turn2["current_round"] == 2

        # Third turn: index 0 -> 1
        turn3 = service.next_turn(combat.id)
        assert turn3["current_turn_index"] == 1
        assert turn3["is_new_round"] is False
        assert turn3["current_round"] == 2

    def test_resolve_attack_hit(self, test_db, session_id):
        """Test resolving a successful attack."""
        service = CombatService(test_db)

        combat = service.create_combat(session_id=session_id)

        attacker = service.add_combatant(
            combat_id=combat.id, name="Attacker", hp=12, hp_max=12, dex=60
        )
        target = service.add_combatant(
            combat_id=combat.id, name="Target", hp=10, hp_max=10, dex=50
        )

        # Force a hit with fixed roll
        result = service.resolve_attack(
            combat_id=combat.id,
            attacker_id=attacker.id,
            target_id=target.id,
            attack_skill=50,
            attack_roll=30,  # Hit
            damage_roll=4,
            damage_bonus=0,
        )

        assert result["hit"] is True
        assert result["damage"] == 4
        assert result["target_hp_after"] == 6

        # Check target was updated
        test_db.refresh(target)
        assert target.hp == 6

    def test_resolve_attack_miss(self, test_db, session_id):
        """Test resolving a failed attack."""
        service = CombatService(test_db)

        combat = service.create_combat(session_id=session_id)

        attacker = service.add_combatant(
            combat_id=combat.id, name="Attacker", hp=12, hp_max=12, dex=60
        )
        target = service.add_combatant(
            combat_id=combat.id, name="Target", hp=10, hp_max=10, dex=50
        )

        # Force a miss with fixed roll
        result = service.resolve_attack(
            combat_id=combat.id,
            attacker_id=attacker.id,
            target_id=target.id,
            attack_skill=50,
            attack_roll=70,  # Miss
        )

        assert result["hit"] is False
        assert result["damage"] == 0
        assert result["target_hp_after"] == result["target_hp_before"]

    def test_resolve_attack_major_wound(self, test_db, session_id):
        """Test major wound from high damage."""
        service = CombatService(test_db)

        combat = service.create_combat(session_id=session_id)

        attacker = service.add_combatant(
            combat_id=combat.id, name="Attacker", hp=12, hp_max=12, dex=60
        )
        target = service.add_combatant(
            combat_id=combat.id, name="Target", hp=10, hp_max=10, dex=50
        )

        # Deal massive damage (> max/2 = 5)
        result = service.resolve_attack(
            combat_id=combat.id,
            attacker_id=attacker.id,
            target_id=target.id,
            attack_skill=50,
            attack_roll=10,
            damage_roll=6,
            damage_bonus=0,
        )

        test_db.refresh(target)
        assert target.has_major_wound is True

    def test_resolve_attack_knockout(self, test_db, session_id):
        """Test knocking a combatant to 0 HP."""
        service = CombatService(test_db)

        combat = service.create_combat(session_id=session_id)

        attacker = service.add_combatant(
            combat_id=combat.id, name="Attacker", hp=12, hp_max=12, dex=60
        )
        target = service.add_combatant(
            combat_id=combat.id, name="Target", hp=5, hp_max=10, dex=50
        )

        # Deal lethal damage
        result = service.resolve_attack(
            combat_id=combat.id,
            attacker_id=attacker.id,
            target_id=target.id,
            attack_skill=50,
            attack_roll=10,
            damage_roll=5,
            damage_bonus=1,
        )

        test_db.refresh(target)
        assert target.hp == 0
        assert target.is_dying is True
        assert target.is_active is False

    def test_heal_success(self, test_db, session_id):
        """Test successful healing."""
        service = CombatService(test_db)

        combat = service.create_combat(session_id=session_id)

        target = service.add_combatant(
            combat_id=combat.id, name="Patient", hp=5, hp_max=12, dex=50
        )

        result = service.heal(
            combat_id=combat.id,
            target_id=target.id,
            heal_amount=1,
            first_aid_skill=50,
            first_aid_roll=30,  # Success
        )

        assert result["healing"] >= 1
        assert result["hp_after"] > result["hp_before"]

        test_db.refresh(target)
        assert target.hp > 5

    def test_heal_failure(self, test_db, session_id):
        """Test failed healing attempt."""
        service = CombatService(test_db)

        combat = service.create_combat(session_id=session_id)

        target = service.add_combatant(
            combat_id=combat.id, name="Patient", hp=5, hp_max=12, dex=50
        )

        result = service.heal(
            combat_id=combat.id,
            target_id=target.id,
            heal_amount=1,
            first_aid_skill=50,
            first_aid_roll=70,  # Failure
        )

        assert result["healing"] == 0
        assert result["hp_after"] == result["hp_before"]

    def test_end_combat(self, test_db, session_id):
        """Test ending combat."""
        service = CombatService(test_db)

        combat = service.create_combat(session_id=session_id)

        assert combat.state == CombatState.ACTIVE.value
        assert combat.ended_at is None

        ended = service.end_combat(combat.id)

        test_db.refresh(ended)
        assert ended.state == CombatState.ENDED.value
        assert ended.ended_at is not None

    def test_get_combat_summary(self, test_db, session_id):
        """Test getting combat summary."""
        service = CombatService(test_db)

        combat = service.create_combat(
            session_id=session_id,
            location="Library",
            description="Fight with a Deep One",
        )

        service.add_combatant(
            combat_id=combat.id, name="Investigator", hp=12, hp_max=12, dex=60
        )
        service.add_combatant(
            combat_id=combat.id, name="Deep One", hp=15, hp_max=15, dex=40
        )

        summary = service.get_combat_summary(combat.id)

        assert summary["id"] == str(combat.id)
        assert summary["location"] == "Library"
        assert summary["round"] == 1
        assert len(summary["combatants"]) == 2
