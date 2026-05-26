/**
 * WebSocket message types
 */

export interface WebSocketMessage {
  type: string;
  payload?: Record<string, unknown>;
}

export interface MessageSentPayload {
  id: string;
  chat_id: string;
  sender_id: string;
  recipient_id: string;
  encrypted_content: string;
  created_at: string;
  status: string;
}

export interface NewMessagePayload {
  id: string;
  message_id: string;
  chat_id: string;
  sender_id: string;
  encrypted_content: string;
  timestamp?: string;
  created_at?: string;
}

export interface ErrorPayload {
  error: string;
}
