"""Pydantic schemas for character data validation and serialization."""
from datetime import datetime
from typing import Dict, List, Optional, Any

from pydantic import BaseModel, Field, field_validator


class CoreAttributes(BaseModel):
    """Core attributes for a CoC 7th Edition character."""
    STR: int = Field(..., ge=0, le=100, description="Strength")
    DEX: int = Field(..., ge=0, le=100, description="Dexterity")
    INT: int = Field(..., ge=0, le=100, description="Intelligence")
    EDU: int = Field(..., ge=0, le=100, description="Education")
    APP: int = Field(..., ge=0, le=100, description="Appearance")
    POW: int = Field(..., ge=0, le=100, description="Power")
    SIZ: int = Field(..., ge=0, le=100, description="Size")
    CON: int = Field(..., ge=0, le=100, description="Constitution")

    class Config:
        extra = "allow"  # Allow additional attributes for future extensions


class DerivedAttributes(BaseModel):
    """Derived attributes calculated from core attributes."""
    HP: int = Field(..., ge=0, description="Hit Points")
    HP_max: int = Field(..., ge=0, description="Maximum Hit Points")
    MP: int = Field(..., ge=0, description="Magic Points")
    MP_max: int = Field(..., ge=0, description="Maximum Magic Points")
    SAN: int = Field(..., ge=0, description="Sanity")
    SAN_max: int = Field(..., ge=0, description="Maximum Sanity")
    Luck: int = Field(..., ge=0, description="Luck")
    Luck_max: int = Field(..., ge=0, description="Maximum Luck")
    Move: int = Field(..., ge=0, description="Move Rate")
    Build: int = Field(default=0, description="Build modifier")
    BonusDamage: int = Field(default=0, description="Bonus damage modifier")

    class Config:
        extra = "allow"


class CharacterStatus(BaseModel):
    """Character status and condition tracking."""
    alive: bool = Field(default=True, description="Whether character is alive")
    conscious: bool = Field(default=True, description="Whether character is conscious")
    dying: bool = Field(default=False, description="Whether character is dying")
    insane: bool = Field(default=False, description="Whether character is insane")
    conditions: List[str] = Field(default_factory=list, description="Active conditions")

    class Config:
        extra = "allow"


class CharacterBase(BaseModel):
    """Base character schema with common fields."""
    name: str = Field(..., min_length=1, max_length=100, description="Character name")
    type: str = Field(default="player", description="Character type (player or npc)")
    core_attributes: CoreAttributes = Field(..., description="Core attributes")
    derived_attributes: DerivedAttributes = Field(..., description="Derived attributes")
    skills: Dict[str, Dict[str, int]] = Field(..., description="Character skills by category")
    inventory: List[Dict[str, Any]] = Field(default_factory=list, description="Inventory items")
    clues: List[Dict[str, Any]] = Field(default_factory=list, description="Discovered clues")
    status: CharacterStatus = Field(..., description="Character status")


class CharacterCreate(CharacterBase):
    """Schema for creating a new character."""
    player_id: Optional[str] = Field(None, description="Player ID (optional, set by server)")

    class Config:
        json_schema_extra = {
            "example": {
                "name": "调查员阿尔伯特",
                "type": "player",
                "core_attributes": {
                    "STR": 50,
                    "DEX": 55,
                    "INT": 70,
                    "EDU": 65,
                    "APP": 40,
                    "POW": 60,
                    "SIZ": 50,
                    "CON": 45
                },
                "derived_attributes": {
                    "HP": 12,
                    "HP_max": 12,
                    "MP": 14,
                    "MP_max": 14,
                    "SAN": 60,
                    "SAN_max": 99,
                    "Luck": 50,
                    "Luck_max": 50,
                    "Move": 7,
                    "Build": 0,
                    "BonusDamage": 0
                },
                "skills": {
                    "common": {
                        "library_use": 60,
                        "spot_hidden": 50,
                        "library_use": 60
                    },
                    "combat": {
                        "firearms": 40
                    }
                },
                "status": {
                    "alive": True,
                    "conscious": True,
                    "dying": False,
                    "insane": False,
                    "conditions": []
                }
            }
        }


class CharacterUpdate(BaseModel):
    """Schema for updating an existing character."""
    name: Optional[str] = Field(None, min_length=1, max_length=100)
    type: Optional[str] = None
    core_attributes: Optional[CoreAttributes] = None
    derived_attributes: Optional[DerivedAttributes] = None
    skills: Optional[Dict[str, Dict[str, int]]] = None
    inventory: Optional[List[Dict[str, Any]]] = None
    clues: Optional[List[Dict[str, Any]]] = None
    status: Optional[CharacterStatus] = None


class CharacterResponse(CharacterBase):
    """Schema for character API responses."""
    character_id: str = Field(..., description="Character unique identifier")
    player_id: Optional[str] = Field(None, description="Owning player ID")
    created_at: datetime = Field(..., description="Creation timestamp")
    updated_at: datetime = Field(..., description="Last update timestamp")

    class Config:
        from_attributes = True

    @field_validator("type", mode="before")
    @classmethod
    def validate_type(cls, v):
        """Convert enum to string if needed."""
        if hasattr(v, "value"):
            return v.value
        return v


class CharacterListResponse(BaseModel):
    """Schema for listing characters."""
    characters: List[CharacterResponse]
    total: int
