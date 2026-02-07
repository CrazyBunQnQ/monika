"""State synchronization service for applying AI state changes."""
import logging
from typing import Optional

from sqlalchemy.orm import Session
from sqlalchemy.orm.attributes import flag_modified

from src.models.session import GameSession
from src.models.event import Event, EventType
from src.schemas.llm_response import StateChanges

logger = logging.getLogger(__name__)

# State modification whitelist - only these fields can be modified by AI
ALLOWED_STATE_CHANGES = {
    "current_scene",
    "world_state.leads",
    "world_state.location",
    "world_state.npcs"
}


class StateSyncService:
    """State synchronization service - safely applies AI state changes."""

    def __init__(self, db: Session):
        self.db = db

    def apply_state_changes(
        self,
        session: GameSession,
        changes: StateChanges,
        source_description: str = "AI Keeper"
    ) -> GameSession:
        """
        Apply state changes to a game session.

        Args:
            session: Current game session
            changes: State changes from LLM response
            source_description: Source description for event logging

        Returns:
            Updated game session
        """
        if not changes:
            return session

        # Record original state for event logging
        original_state = {
            "current_scene": session.current_scene_name,
            "world_state": session.world_state.copy() if session.world_state else {}
        }

        # Apply scene changes
        if changes.current_scene is not None:
            session.current_scene_name = changes.current_scene

        # Apply world state changes
        if changes.world_state:
            if not session.world_state:
                session.world_state = {}

            world_state_modified = False

            for key, value in changes.world_state.items():
                # Check if field is in whitelist
                field_path = f"world_state.{key}"
                if field_path in ALLOWED_STATE_CHANGES:
                    if key == "leads" and isinstance(value, str):
                        # Handle lead addition: +新线索
                        if value.startswith("+"):
                            new_lead = value[1:].strip()
                            if "leads" not in session.world_state:
                                session.world_state["leads"] = []
                            if new_lead not in session.world_state["leads"]:
                                session.world_state["leads"].append(new_lead)
                        elif value.startswith("-"):
                            # Handle lead removal: -旧线索
                            lead_to_remove = value[1:].strip()
                            if "leads" in session.world_state:
                                session.world_state["leads"] = [
                                    l for l in session.world_state["leads"]
                                    if l != lead_to_remove
                                ]
                        # Mark that we modified the world_state in place
                        world_state_modified = True
                    else:
                        # Direct assignment for other fields
                        session.world_state[key] = value
                        # Direct assignment also modifies world_state
                        world_state_modified = True
                else:
                    logger.warning(f"Attempted to modify non-allowed state: {field_path}")

            # Flag the world_state as modified if we changed it
            if world_state_modified:
                flag_modified(session, "world_state")

        # Save to database
        self.db.add(session)
        self.db.commit()
        # Don't refresh - the session object already has the updated values

        # Log state change event
        self._log_state_change(
            session=session,
            original_state=original_state,
            new_state={
                "current_scene": session.current_scene_name,
                "world_state": session.world_state
            },
            source=source_description
        )

        return session

    def _log_state_change(
        self,
        session: GameSession,
        original_state: dict,
        new_state: dict,
        source: str
    ):
        """Log state change as an event."""
        changes = []

        # Check for scene changes
        if original_state["current_scene"] != new_state["current_scene"]:
            changes.append(
                f"场景: {original_state['current_scene']} → {new_state['current_scene']}"
            )

        # Check for lead changes
        original_leads = set(original_state["world_state"].get("leads", []))
        new_leads = set(new_state["world_state"].get("leads", []))
        if original_leads != new_leads:
            added = new_leads - original_leads
            removed = original_leads - new_leads
            if added:
                changes.append(f"新增线索: {', '.join(added)}")
            if removed:
                changes.append(f"移除线索: {', '.join(removed)}")

        # Check for location changes
        original_location = original_state["world_state"].get("location")
        new_location = new_state["world_state"].get("location")
        if original_location != new_location:
            if new_location:
                changes.append(f"位置: {original_location or '未知'} → {new_location}")

        # Check for NPC changes
        original_npcs = set(original_state["world_state"].get("npcs", []))
        new_npcs = set(new_state["world_state"].get("npcs", []))
        if original_npcs != new_npcs:
            added_npcs = new_npcs - original_npcs
            if added_npcs:
                changes.append(f"新增NPC: {', '.join(added_npcs)}")

        # Create event if there are changes
        if changes:
            event = Event(
                session_id=session.id,
                actor_role="system",  # System-generated event
                visibility="public",
                event_type=EventType.SCENE_CHANGE,
                description=f"[{source}] " + "; ".join(changes),
                payload={
                    "original": original_state,
                    "new": new_state
                }
            )
            self.db.add(event)
            self.db.commit()
