from abc import ABC, abstractmethod
from typing import AsyncIterator

class LLMProvider(ABC):
    """LLM Provider 抽象基类"""

    @abstractmethod
    async def stream_chat(
        self,
        messages: list[dict],
        system_prompt: str
    ) -> AsyncIterator[str]:
        """流式聊天响应，返回 JSON 字符串片段的异步迭代器"""
        pass

    @abstractmethod
    async def get_context_limit(self) -> int:
        """返回模型的最大上下文长度"""
        pass

    @abstractmethod
    async def health_check(self) -> bool:
        """健康检查"""
        pass
