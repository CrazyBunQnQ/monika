/**
 * WebSocket service for real-time game communication
 * Manages connection, reconnection, and message handling
 */

import type { ServerMessage, UserMessage, StateUpdate } from '../types/websocket';

export interface WebSocketCallbacks {
  onMessage: (message: ServerMessage) => void;
  onStateUpdate: (update: StateUpdate) => void;
  onError: (error: string) => void;
  onConnect: () => void;
  onDisconnect: () => void;
}

export class GameWebSocket {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private sessionId: string | null = null;
  private callbacks: WebSocketCallbacks;
  private reconnectTimeout: ReturnType<typeof setTimeout> | null = null;

  constructor(callbacks: WebSocketCallbacks) {
    this.callbacks = callbacks;
  }

  /**
   * Connect to the WebSocket server for a specific game session
   * @param sessionId - The game session ID to connect to
   */
  connect(sessionId: string): void {
    this.sessionId = sessionId;
    const wsUrl = `ws://localhost:8000/ws/game/${sessionId}`;

    try {
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        console.log('WebSocket connected');
        this.reconnectAttempts = 0;
        this.callbacks.onConnect();
      };

      this.ws.onmessage = (event) => {
        try {
          const message: ServerMessage = JSON.parse(event.data);
          this.callbacks.onMessage(message);

          if (message.type === 'state_update') {
            this.callbacks.onStateUpdate(message);
          }
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error);
        }
      };

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        this.callbacks.onError('Connection error');
      };

      this.ws.onclose = () => {
        console.log('WebSocket closed');
        this.callbacks.onDisconnect();
        this.attemptReconnect();
      };
    } catch (error) {
      console.error('Failed to create WebSocket:', error);
      this.callbacks.onError('Failed to connect');
    }
  }

  /**
   * Disconnect from the WebSocket server
   * Clears session ID and reconnect attempts to prevent scheduled reconnects
   */
  disconnect(): void {
    // Clear any scheduled reconnection attempts
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }

    // Close the WebSocket connection
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }

    // Clear session ID and reset reconnect attempts to prevent future reconnects
    this.sessionId = null;
    this.reconnectAttempts = 0;
  }

  /**
   * Send a message to the WebSocket server
   * @param message - The user message to send
   */
  send(message: UserMessage): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    } else {
      console.error('WebSocket is not connected');
    }
  }

  /**
   * Attempt to reconnect with exponential backoff
   */
  private attemptReconnect(): void {
    if (this.reconnectAttempts < this.maxReconnectAttempts && this.sessionId) {
      this.reconnectAttempts++;
      const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

      console.log(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);

      this.reconnectTimeout = setTimeout(() => {
        this.connect(this.sessionId!);
      }, delay);
    }
  }

  /**
   * Get the current connection state
   */
  get isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }
}
