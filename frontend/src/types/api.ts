/**
 * API types and interfaces
 */

// User types
export interface User {
  id: string
  login: string
  public_key: string
  created_at: string
}

// Authentication requests
export interface RegisterRequest {
  login: string
  auth_hash: string
  public_key: string
}

export interface LoginRequest {
  login: string
  auth_hash: string
}

export interface ChatKeyUpdate {
  companion_id: string
  encrypted_key: string
  key_iv: string
}

export interface ChangePasswordRequest {
  login: string
  old_auth_hash: string
  new_auth_hash: string
  new_public_key: string
  chat_keys?: ChatKeyUpdate[]
}

// Authentication responses
export interface AuthResponse {
  user: User
  tokens: {
    access_token: string
    refresh_token: string
  }
}

export interface RefreshTokenResponse {
  access_token: string
  refresh_token: string
}

// Message status constants - backend sends numbers, frontend uses strings
export const MessageStatus = {
  SENT: 0,
  DELIVERED: 1,
  READ: 2
} as const;

export type MessageStatusValue = typeof MessageStatus[keyof typeof MessageStatus];
export type MessageStatusString = 'sent' | 'delivered' | 'read';

// Message types
export interface Message {
  id: string
  chat_id: string
  sender_id: string
  recipient_id: string
  encrypted_content: string
  message_type: 'text' | 'image' | 'file'
  created_at: string
  status: MessageStatusValue | MessageStatusString | number // Support both formats
}

export interface SendMessageRequest {
  chat_id: string
  recipient_id: string
  encrypted_content: string
  message_type: string
}

export interface GetMessagesRequest {
  chat_id: string
  limit?: number
  last_message_id?: string
}

export interface GetMessagesResponse {
  messages: Message[]
  next_message_id: string
  has_more: boolean
}

// Chat types
export interface Chat {
  id: string
  user_id: string
  companion_id: string
  companion_login?: string // For display purposes
  created_at: string
  last_message_at: string
  encrypted_key: string
  key_iv: string
  last_message?: string // Encrypted content of last message
  last_message_preview?: string // Decrypted preview for display
  is_online?: boolean // Online status of companion
  unread_count?: number // Number of unread messages
}

export interface SaveChatKeysRequest {
  user_id: string
  companion_id: string
  encrypted_key: string
  key_iv: string
}

// Error response
export interface ApiError {
  error: string
  message?: string
  status?: number
}
