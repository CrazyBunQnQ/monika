"""Streaming LLM response parser."""
import json
import logging
from typing import Optional, AsyncIterator

from src.schemas.llm_response import LLMResponse


logger = logging.getLogger(__name__)


class ResponseParser:
    """解析 LLM 流式响应"""

    def __init__(self):
        self.buffer = ""

    async def parse_stream(
        self,
        stream: AsyncIterator[str]
    ) -> AsyncIterator[LLMResponse]:
        """Parse a stream of chunks into LLMResponse objects.

        Buffers incoming chunks until a complete JSON object is detected,
        then yields a validated LLMResponse.
        """
        self.buffer = ""
        async for chunk in stream:
            self.buffer += chunk

            json_str = self._extract_json(self.buffer)
            if json_str:
                try:
                    response = LLMResponse.model_validate_json(json_str)
                    self.buffer = ""
                    yield response
                except Exception as e:
                    logger.warning(f"Failed to parse LLM response: {e}")
                    yield self._fallback_response(json_str)
            elif self.buffer and "{" not in self.buffer:
                # No JSON marker found, treat as plain text
                yield self._fallback_response(self.buffer)
                self.buffer = ""

    def _extract_json(self, text: str) -> Optional[str]:
        """Extract JSON object from text, handling surrounding content.

        Finds the first complete JSON object by matching braces.
        Returns None if no valid JSON object is found.
        """
        text = text.strip()
        if not text:
            return None
        start = text.find("{")
        end = text.rfind("}")
        if start != -1 and end != -1 and end > start:
            return text[start:end + 1]
        return None

    def _fallback_response(self, text: str) -> LLMResponse:
        """Create a fallback response when JSON parsing fails.

        Attempts to extract at least the narrative field from malformed JSON,
        otherwise uses the raw text truncated to 1000 characters.
        """
        json_str = self._extract_json(text)
        if json_str:
            try:
                data = json.loads(json_str)
                narrative = data.get("narrative", text)
            except Exception:
                narrative = text
        else:
            narrative = text

        return LLMResponse(
            narrative=narrative[:1000],
            tone="calm"
        )
