/**
 * React hook for managing game WebSocket connection
 * Provides connection state and message sending functionality
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import { GameWebSocket } from '../services/websocket';
import type { UserMessage, ServerMessage, StateUpdate } from '../types/websocket';

export interface UseGameWebSocketOptions {
  onMessage?: (message: ServerMessage) => void;
  onStateUpdate?: (update: StateUpdate) => void;
}

export function useGameWebSocket(sessionId: string | null, options?: UseGameWebSocketOptions) {
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const wsRef = useRef<GameWebSocket | null>(null);
  const optionsRef = useRef(options);

  // Keep optionsRef updated without triggering reconnection
  useEffect(() => {
    optionsRef.current = options;
  }, [options]);

  useEffect(() => {
    if (!sessionId) return;

    const ws = new GameWebSocket({
      onMessage: (message: ServerMessage) => {
        // Forward to custom handler if provided
        if (optionsRef.current?.onMessage) {
          optionsRef.current.onMessage(message);
        }
      },
      onStateUpdate: (update: StateUpdate) => {
        // Forward to custom handler if provided
        if (optionsRef.current?.onStateUpdate) {
          optionsRef.current.onStateUpdate(update);
        }
      },
      onError: (err) => {
        setError(err);
      },
      onConnect: () => {
        setIsConnected(true);
        setError(null);
      },
      onDisconnect: () => {
        setIsConnected(false);
      }
    });

    ws.connect(sessionId);
    wsRef.current = ws;

    return () => {
      ws.disconnect();
    };
  }, [sessionId]);

  /**
   * Send a message to the WebSocket server
   * @param content - The message content to send
   */
  const sendMessage = useCallback((content: string) => {
    if (wsRef.current) {
      const message: UserMessage = {
        type: 'user_message',
        content,
        timestamp: new Date().toISOString()
      };
      wsRef.current.send(message);
    }
  }, []);

  /**
   * Manually disconnect the WebSocket connection
   */
  const disconnect = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.disconnect();
    }
  }, []);

  /**
   * Manually reconnect the WebSocket connection
   * Creates a new instance to avoid race conditions
   */
  const reconnect = useCallback(() => {
    if (!sessionId) return;

    // Disconnect existing instance
    if (wsRef.current) {
      wsRef.current.disconnect();
      wsRef.current = null;
    }

    // Create new instance to ensure clean state
    const ws = new GameWebSocket({
      onMessage: (message: ServerMessage) => {
        if (optionsRef.current?.onMessage) {
          optionsRef.current.onMessage(message);
        }
      },
      onStateUpdate: (update: StateUpdate) => {
        if (optionsRef.current?.onStateUpdate) {
          optionsRef.current.onStateUpdate(update);
        }
      },
      onError: (err) => {
        setError(err);
      },
      onConnect: () => {
        setIsConnected(true);
        setError(null);
      },
      onDisconnect: () => {
        setIsConnected(false);
      }
    });

    ws.connect(sessionId);
    wsRef.current = ws;
  }, [sessionId]);

  return {
    isConnected,
    error,
    sendMessage,
    disconnect,
    reconnect
  };
}
