"""Tests for StateSyncService."""
import pytest

from src.services.state_sync import StateSyncService, ALLOWED_STATE_CHANGES
from src.models.session import GameSession
from src.schemas.llm_response import StateChanges


def test_apply_scene_change(test_db):
    """Test applying scene changes to a session."""
    service = StateSyncService(test_db)

    session = GameSession(
        name="Test Session",
        owner_id=1,
        current_scene_name="旧书房",
        world_state={}
    )
    test_db.add(session)
    test_db.commit()

    changes = StateChanges(current_scene="密室")
    updated = service.apply_state_changes(session, changes)

    assert updated.current_scene_name == "密室"



def test_apply_lead_addition(test_db):
    """Test adding leads using + syntax."""
    service = StateSyncService(test_db)

    session = GameSession(
        name="Test Session",
        owner_id=1,
        current_scene_name="旧书房",
        world_state={"leads": ["旧钥匙"]}
    )
    test_db.add(session)
    test_db.commit()

    changes = StateChanges(
        world_state={"leads": "+神秘笔记"}
    )
    updated = service.apply_state_changes(session, changes)

    assert "神秘笔记" in updated.world_state["leads"]
    assert "旧钥匙" in updated.world_state["leads"]



def test_apply_lead_removal(test_db):
    """Test removing leads using - syntax."""
    service = StateSyncService(test_db)

    session = GameSession(
        name="Test Session",
        owner_id=1,
        current_scene_name="旧书房",
        world_state={"leads": ["旧钥匙", "神秘笔记"]}
    )
    test_db.add(session)
    test_db.commit()

    changes = StateChanges(
        world_state={"leads": "-旧钥匙"}
    )
    updated = service.apply_state_changes(session, changes)

    assert "旧钥匙" not in updated.world_state["leads"]
    assert "神秘笔记" in updated.world_state["leads"]



def test_whitelist_enforcement(test_db):
    """Test that only whitelisted fields can be modified."""
    service = StateSyncService(test_db)

    session = GameSession(
        name="Test Session",
        owner_id=1,
        current_scene_name="旧书房",
        world_state={}
    )
    test_db.add(session)
    test_db.commit()

    # Test allowed field - leads
    changes = StateChanges(
        world_state={"leads": ["新线索"]}
    )
    updated = service.apply_state_changes(session, changes)
    assert "新线索" in updated.world_state.get("leads", [])

    # Test allowed field - location
    changes = StateChanges(
        world_state={"location": "图书馆"}
    )
    updated = service.apply_state_changes(session, changes)
    assert updated.world_state.get("location") == "图书馆"

    # Test allowed field - npcs
    changes = StateChanges(
        world_state={"npcs": ["守门人"]}
    )
    updated = service.apply_state_changes(session, changes)
    assert "守门人" in updated.world_state.get("npcs", [])

    # Verify whitelist contains correct fields
    assert "current_scene" in ALLOWED_STATE_CHANGES
    assert "world_state.leads" in ALLOWED_STATE_CHANGES
    assert "world_state.location" in ALLOWED_STATE_CHANGES
    assert "world_state.npcs" in ALLOWED_STATE_CHANGES



def test_non_whitelisted_field_ignored(test_db):
    """Test that non-whitelisted fields are ignored and logged."""
    service = StateSyncService(test_db)

    session = GameSession(
        name="Test Session",
        owner_id=1,
        current_scene_name="旧书房",
        world_state={"some_field": "original_value"}
    )
    test_db.add(session)
    test_db.commit()

    # Try to modify a non-whitelisted field
    changes = StateChanges(
        world_state={"some_field": "new_value"}
    )
    updated = service.apply_state_changes(session, changes)

    # The field should remain unchanged
    assert updated.world_state.get("some_field") == "original_value"



def test_empty_state_changes(test_db):
    """Test that empty state changes don't modify the session."""
    service = StateSyncService(test_db)

    session = GameSession(
        name="Test Session",
        owner_id=1,
        current_scene_name="旧书房",
        world_state={"leads": ["线索1"]}
    )
    test_db.add(session)
    test_db.commit()

    changes = StateChanges()
    updated = service.apply_state_changes(session, changes)

    assert updated.current_scene_name == "旧书房"
    assert updated.world_state["leads"] == ["线索1"]



def test_multiple_leads_modification(test_db):
    """Test adding and removing multiple leads in sequence."""
    service = StateSyncService(test_db)

    session = GameSession(
        name="Test Session",
        owner_id=1,
        current_scene_name="旧书房",
        world_state={"leads": ["线索A", "线索B", "线索C"]}
    )
    test_db.add(session)
    test_db.commit()

    # Add one lead
    changes = StateChanges(world_state={"leads": "+线索D"})
    updated = service.apply_state_changes(session, changes)
    assert "线索D" in updated.world_state["leads"]
    assert len(updated.world_state["leads"]) == 4

    # Remove another lead
    changes = StateChanges(world_state={"leads": "-线索B"})
    updated = service.apply_state_changes(session, changes)
    assert "线索B" not in updated.world_state["leads"]
    assert len(updated.world_state["leads"]) == 3



def test_duplicate_lead_not_added(test_db):
    """Test that duplicate leads are not added."""
    service = StateSyncService(test_db)

    session = GameSession(
        name="Test Session",
        owner_id=1,
        current_scene_name="旧书房",
        world_state={"leads": ["线索A"]}
    )
    test_db.add(session)
    test_db.commit()

    # Try to add a duplicate lead
    changes = StateChanges(world_state={"leads": "+线索A"})
    updated = service.apply_state_changes(session, changes)

    # Should only have one instance
    assert updated.world_state["leads"].count("线索A") == 1
    assert len(updated.world_state["leads"]) == 1
