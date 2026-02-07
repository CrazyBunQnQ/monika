"""LLM response schema definitions."""
from pydantic import BaseModel, Field
from typing import Optional, Dict, List


class StateChanges(BaseModel):
    """AI 可以修改的状态字段（白名单）"""
    current_scene: Optional[str] = Field(None, description="当前场景名称")
    world_state: Optional[Dict] = Field(None, description="世界状态更新")


class LLMResponse(BaseModel):
    """LLM 响应格式"""
    narrative: str = Field(..., description="叙述文本，显示给玩家")
    tone: str = Field("calm", description="语气: mystery, horror, action, calm")
    urgency: str = Field("low", description="紧迫度: low, medium, high")
    state_changes: Optional[StateChanges] = Field(None, description="状态变化")
    suggestions: Optional[List[str]] = Field(None, description="给玩家的操作建议")
    audio_cue: Optional[str] = Field(None, description="音效提示")
    requires_roll: bool = Field(False, description="是否建议玩家进行检定")
