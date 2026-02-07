"""Character Pydantic schemas."""
from datetime import datetime
from typing import Optional

from pydantic import BaseModel, ConfigDict, Field


class CharacterCreate(BaseModel):
    """Schema for creating a character."""

    name: str
    age: Optional[int] = Field(None, alias="age")
    gender: Optional[str] = Field(None, alias="gender")
    occupation: Optional[str] = Field(None, alias="occupation")
    mental_illness: Optional[str] = Field(None, alias="mental_illness")
    backstory: Optional[str] = Field(None, alias="backstory")
    str: Optional[int] = Field(None, alias="str")
    con: Optional[int] = Field(None, alias="con")
    dex: Optional[int] = Field(None, alias="dex")
    app: Optional[int] = Field(None, alias="app")
    pow: Optional[int] = Field(None, alias="pow")
    intelligence: Optional[int] = Field(None, alias="int")
    siz: Optional[int] = Field(None, alias="siz")
    edu: Optional[int] = Field(None, alias="edu")
    hp: Optional[int] = Field(None, alias="hp")
    mp: Optional[int] = Field(None, alias="mp")
    san: Optional[int] = Field(None, alias="san")
    max_san: Optional[int] = Field(None, alias="max_san")
    luck: Optional[int] = Field(None, alias="luck")

    model_config = ConfigDict(populate_by_name=True)


class CharacterUpdate(BaseModel):
    """Schema for updating a character."""

    name: Optional[str] = Field(None, alias="name")
    age: Optional[int] = Field(None, alias="age")
    gender: Optional[str] = Field(None, alias="gender")
    occupation: Optional[str] = Field(None, alias="occupation")
    mental_illness: Optional[str] = Field(None, alias="mental_illness")
    backstory: Optional[str] = Field(None, alias="backstory")
    str: Optional[int] = Field(None, alias="str")
    con: Optional[int] = Field(None, alias="con")
    dex: Optional[int] = Field(None, alias="dex")
    app: Optional[int] = Field(None, alias="app")
    pow: Optional[int] = Field(None, alias="pow")
    intelligence: Optional[int] = Field(None, alias="int")
    siz: Optional[int] = Field(None, alias="siz")
    edu: Optional[int] = Field(None, alias="edu")
    hp: Optional[int] = Field(None, alias="hp")
    mp: Optional[int] = Field(None, alias="mp")
    san: Optional[int] = Field(None, alias="san")
    max_san: Optional[int] = Field(None, alias="max_san")
    luck: Optional[int] = Field(None, alias="luck")

    model_config = ConfigDict(populate_by_name=True)
