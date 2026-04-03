/**
 * Messages and Chats API client
 */

import { api } from './client'
import type {
  Message,
  SendMessageRequest,
  GetMessagesRequest,
  GetMessagesResponse,
  Chat,
  SaveChatKeysRequest,
} from '../../types/api'

/**
 * Send message to chat
 * 
 * @param data - Message data (encrypted content)
 * @returns Created message
 */
export async function sendMessage(data: SendMessageRequest): Promise<Message> {
  return api.post<Message>('/messages/send', data)
}

/**
 * Get messages from chat
 * 
 * @param params - Query parameters (chat_id, limit, last_message_id)
 * @returns Messages and pagination info
 */
export async function getMessages(
  params: GetMessagesRequest
): Promise<GetMessagesResponse> {
  return api.get<GetMessagesResponse>('/messages', {
    params: {
      chat_id: params.chat_id,
      limit: params.limit?.toString() || '50',
      ...(params.last_message_id && { last_message_id: params.last_message_id }),
    },
  })
}

/**
 * Mark messages as read
 * 
 * @param chatId - Chat ID
 * @param lastMessageId - Last read message ID
 */
export async function markAsRead(
  chatId: string,
  lastMessageId: string
): Promise<{ success: boolean }> {
  return api.post<{ success: boolean }>('/messages/read', {
    chat_id: chatId,
    last_message_id: lastMessageId,
  })
}

/**
 * Delete message
 * 
 * @param messageId - Message ID to delete
 */
export async function deleteMessage(
  messageId: string
): Promise<{ success: boolean }> {
  return api.delete<{ success: boolean }>(`/messages/${messageId}`)
}

/**
 * Edit message
 * 
 * @param messageId - Message ID to edit
 * @param newContent - New encrypted content
 */
export async function editMessage(
  messageId: string,
  newContent: string
): Promise<{ success: boolean }> {
  return api.put<{ success: boolean }>(`/messages/${messageId}`, {
    new_content: newContent,
  })
}

/**
 * Get user's chats
 * 
 * @returns List of chats with encrypted keys
 */
export async function getChats(): Promise<Chat[]> {
  return api.get<Chat[]>('/chats')
}

/**
 * Save chat encryption keys
 * Used when creating new chat or rotating keys
 * 
 * @param data - Chat keys data
 */
export async function saveChatKeys(
  data: SaveChatKeysRequest
): Promise<{ success: boolean }> {
  return api.post<{ success: boolean }>('/chats/keys', data)
}

/**
 * Update multiple chat keys at once
 * Used when changing password (all keys need to be re-encrypted)
 * 
 * @param keys - Array of chat key updates
 */
export async function updateChatKeys(
  keys: Array<{
    companion_id: string
    encrypted_key: string
    key_iv: string
  }>
): Promise<{ success: boolean; updated_count: number }> {
  return api.put<{ success: boolean; updated_count: number }>('/chats/keys', {
    keys,
  })
}
