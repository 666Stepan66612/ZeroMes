/**
 * WebSocket client for real-time messaging
 * Handles connection, reconnection, and message delivery
 */

export type WebSocketStatus = 'connecting' | 'connected' | 'disconnected' | 'error';

export interface IncomingMessage {
  message_id: string;
  sender_id: string;
  content: string;
  timestamp: number;
}

export interface WebSocketMessage {
  type: 'message' | 'ping' | 'pong';
  data?: IncomingMessage;
}

type MessageHandler = (message: IncomingMessage) => void;
type StatusHandler = (status: WebSocketStatus) => void;
type ErrorHandler = (error: Error) => void;

export class WebSocketClient {
  private ws: WebSocket | null = null;
  private url: string;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000; // Start with 1 second
  private maxReconnectDelay = 30000; // Max 30 seconds
  private reconnectTimer: number | null = null;
  private pingInterval: number | null = null;
  private status: WebSocketStatus = 'disconnected';
  
  private messageHandlers: Set<MessageHandler> = new Set();
  private statusHandlers: Set<StatusHandler> = new Set();
  private errorHandlers: Set<ErrorHandler> = new Set();

  constructor(baseUrl: string = 'ws://localhost') {
    this.url = `${baseUrl}/ws`;
  }

  /**
   * Connect to WebSocket server
   */
  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return;
    }

    this.setStatus('connecting');

    try {
      this.ws = new WebSocket(this.url);
      
      this.ws.onopen = () => this.handleOpen();
      this.ws.onmessage = (event) => this.handleMessage(event);
      this.ws.onerror = (event) => this.handleError(event);
      this.ws.onclose = (event) => this.handleClose(event);
    } catch (error) {
      this.handleError(error as Error);
    }
  }

  /**
   * Disconnect from WebSocket server
   */
  disconnect(): void {
    this.clearReconnectTimer();
    this.clearPingInterval();
    
    if (this.ws) {
      this.ws.close(1000, 'Client disconnect');
      this.ws = null;
    }
    
    this.setStatus('disconnected');
  }

  /**
   * Send a message through WebSocket
   * Note: In this architecture, messages are sent via HTTP API,
   * WebSocket is only for receiving
   */
  send(data: unknown): void {
    if (this.status === 'connected' && this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data));
    } else {
      throw new Error('WebSocket is not connected');
    }
  }

  /**
   * Subscribe to incoming messages
   */
  onMessage(handler: MessageHandler): () => void {
    this.messageHandlers.add(handler);
    return () => this.messageHandlers.delete(handler);
  }

  /**
   * Subscribe to connection status changes
   */
  onStatus(handler: StatusHandler): () => void {
    this.statusHandlers.add(handler);
    // Immediately call with current status
    handler(this.status);
    return () => this.statusHandlers.delete(handler);
  }

  /**
   * Subscribe to errors
   */
  onError(handler: ErrorHandler): () => void {
    this.errorHandlers.add(handler);
    return () => this.errorHandlers.delete(handler);
  }

  /**
   * Get current connection status
   */
  getStatus(): WebSocketStatus {
    return this.status;
  }

  /**
   * Wait for WebSocket to be connected
   */
  async waitForConnection(timeout = 5000): Promise<void> {
    if (this.status === 'connected' && this.ws?.readyState === WebSocket.OPEN) {
      return;
    }

    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.statusHandlers.delete(handler);
        reject(new Error('Connection timeout'));
      }, timeout);

      const handler: StatusHandler = (status) => {
        if (status === 'connected') {
          clearTimeout(timer);
          this.statusHandlers.delete(handler);
          // Add small delay to ensure WebSocket is fully ready
          setTimeout(() => resolve(), 50);
        } else if (status === 'error') {
          clearTimeout(timer);
          this.statusHandlers.delete(handler);
          reject(new Error('Connection failed'));
        }
      };
      
      // Add handler without calling it immediately
      this.statusHandlers.add(handler);
    });
  }

  /**
   * Check if connected
   */
  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  private handleOpen(): void {
    console.log('[WebSocket] Connected');
    this.reconnectAttempts = 0;
    this.reconnectDelay = 1000;
    this.setStatus('connected');
    this.startPing();
  }

  private handleMessage(event: MessageEvent): void {
    try {
      const data = JSON.parse(event.data);
      
      // Special handling for pong
      if (data.type === 'pong') {
        console.log('[WebSocket] Pong received');
        return;
      }
      
      // Notify all message handlers with the full message
      this.messageHandlers.forEach(handler => {
        try {
          handler(data);
        } catch (error) {
          console.error('[WebSocket] Error in message handler:', error);
        }
      });
    } catch (error) {
      console.error('[WebSocket] Failed to parse message:', error);
      this.notifyError(new Error('Failed to parse WebSocket message'));
    }
  }

  private handleError(error: Event | Error): void {
    console.error('[WebSocket] Error:', error);
    this.setStatus('error');
    
    const err = error instanceof Error 
      ? error 
      : new Error('WebSocket error occurred');
    
    this.notifyError(err);
  }

  private handleClose(event: CloseEvent): void {
    console.log(`[WebSocket] Closed: ${event.code} ${event.reason}`);
    this.clearPingInterval();
    this.setStatus('disconnected');
    
    // Attempt reconnection if not a normal closure
    if (event.code !== 1000 && this.reconnectAttempts < this.maxReconnectAttempts) {
      this.scheduleReconnect();
    }
  }

  private scheduleReconnect(): void {
    this.clearReconnectTimer();
    
    this.reconnectAttempts++;
    const delay = Math.min(
      this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1),
      this.maxReconnectDelay
    );
    
    console.log(`[WebSocket] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
    
    this.reconnectTimer = window.setTimeout(() => {
      this.connect();
    }, delay);
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer !== null) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  private startPing(): void {
    this.clearPingInterval();
    
    // Send ping every 30 seconds to keep connection alive
    this.pingInterval = window.setInterval(() => {
      if (this.isConnected()) {
        try {
          this.send({ type: 'ping' });
        } catch (error) {
          console.error('[WebSocket] Failed to send ping:', error);
        }
      }
    }, 30000);
  }

  private clearPingInterval(): void {
    if (this.pingInterval !== null) {
      clearInterval(this.pingInterval);
      this.pingInterval = null;
    }
  }

  private setStatus(status: WebSocketStatus): void {
    if (this.status !== status) {
      this.status = status;
      this.statusHandlers.forEach(handler => {
        try {
          handler(status);
        } catch (error) {
          console.error('[WebSocket] Error in status handler:', error);
        }
      });
    }
  }

  private notifyError(error: Error): void {
    this.errorHandlers.forEach(handler => {
      try {
        handler(error);
      } catch (err) {
        console.error('[WebSocket] Error in error handler:', err);
      }
    });
  }
}

// Singleton instance
let wsClient: WebSocketClient | null = null;

/**
 * Get or create WebSocket client instance
 */
export function getWebSocketClient(baseUrl?: string): WebSocketClient {
  if (!wsClient) {
    wsClient = new WebSocketClient(baseUrl);
  }
  return wsClient;
}

/**
 * Reset WebSocket client (useful for testing or logout)
 */
export function resetWebSocketClient(): void {
  if (wsClient) {
    wsClient.disconnect();
    wsClient = null;
  }
}
