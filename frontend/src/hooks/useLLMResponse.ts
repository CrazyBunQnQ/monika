/**
 * React hook for managing LLM streaming responses
 * Handles progressive text updates and response finalization
 */

import { useState, useCallback } from 'react';
import { LLMResponse } from '../types/websocket';

export function useLLMResponse() {
  const [streamingText, setStreamingText] = useState('');
  const [isStreaming, setIsStreaming] = useState(false);
  const [currentResponse, setCurrentResponse] = useState<LLMResponse | null>(null);

  /**
   * Process a streaming chunk of the LLM response
   * Updates the displayed text progressively as chunks arrive
   * @param chunk - A partial LLM response chunk
   */
  const processStream = useCallback((chunk: LLMResponse) => {
    setStreamingText(chunk.narrative);
    setIsStreaming(true);
  }, []);

  /**
   * Finalize the LLM response when streaming is complete
   * Sets the complete response and stops streaming state
   * @param response - The complete LLM response
   */
  const finalizeResponse = useCallback((response: LLMResponse) => {
    setCurrentResponse(response);
    setStreamingText(response.narrative);
    setIsStreaming(false);
  }, []);

  /**
   * Reset all response state
   */
  const reset = useCallback(() => {
    setStreamingText('');
    setIsStreaming(false);
    setCurrentResponse(null);
  }, []);

  return {
    streamingText,
    isStreaming,
    currentResponse,
    processStream,
    finalizeResponse,
    reset
  };
}
