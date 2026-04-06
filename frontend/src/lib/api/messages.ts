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
 * Send message to chat via WebSocket
 *
 * @param data - Message data (encrypted content)
 * @returns Created message
 */
export async function sendMessage(data: SendMessageRequest): Promise<Message> {
  const ws = (await import('./websocket')).getWebSocketClient()
  
  // Wait for connection
  await ws.waitForConnection()
  
  return new Promise((resolve, reject) => {
    ws.send({
      type: 'send_message',
      chat_id: data.chat_id,
      recipient_id: data.recipient_id,
      content: data.encrypted_content,
      message_type: data.message_type || 'text'
    })
    
    // Listen for response
    const timeout = setTimeout(() => reject(new Error('Timeout')), 5000)
    const unsubscribe = ws.onMessage((msg: any) => {
      if (msg.type === 'message_sent') {
        clearTimeout(timeout)
        unsubscribe()
        resolve(msg.payload)
      } else if (msg.type === 'error') {
        clearTimeout(timeout)
        unsubscribe()
        reject(new Error(msg.payload?.error || 'Failed to send message'))
      }
    })
  })
}

/**
 * Get messages from chat via WebSocket
 *
 * @param params - Query parameters (chat_id, limit, last_message_id)
 * @returns Messages and pagination info
 */
export async function getMessages(
  params: GetMessagesRequest
): Promise<GetMessagesResponse> {
  const ws = (await import('./websocket')).getWebSocketClient()
  
  // Wait for connection
  await ws.waitForConnection()
  
  return new Promise((resolve, reject) => {
    ws.send({
      type: 'get_messages',
      chat_id: params.chat_id,
      limit: params.limit || 50,
      last_message_id: params.last_message_id || ''
    })
    
    // Listen for response
    const timeout = setTimeout(() => reject(new Error('Timeout')), 5000)
    const unsubscribe = ws.onMessage((msg: any) => {
      if (msg.type === 'messages') {
        clearTimeout(timeout)
        unsubscribe()
        resolve(msg.payload || { messages: [], next_message_id: '', has_more: false })
      } else if (msg.type === 'error') {
        clearTimeout(timeout)
        unsubscribe()
        reject(new Error(msg.payload?.error || 'Failed to get messages'))
      }
    })
  })
}

/**
 * Mark messages as read via WebSocket
 *
 * @param chatId - Chat ID
 * @param lastMessageId - Last read message ID
 */
export async function markAsRead(
  chatId: string,
  lastMessageId: string
): Promise<void> {
  const ws = (await import('./websocket')).getWebSocketClient()
  
  // Wait for connection
  await ws.waitForConnection()
  
  return new Promise((resolve, reject) => {
    ws.send({
      type: 'mark_as_read',
      chat_id: chatId,
      last_message_id: lastMessageId,
    })
    
    // Listen for response
    const timeout = setTimeout(() => reject(new Error('Timeout')), 5000)
    const unsubscribe = ws.onMessage((msg: any) => {
      if (msg.type === 'marked_as_read') {
        clearTimeout(timeout)
        unsubscribe()
        resolve()
      } else if (msg.type === 'error') {
        clearTimeout(timeout)
        unsubscribe()
        reject(new Error(msg.payload?.error || 'Failed to mark as read'))
      }
    })
  })
}

/**
 * Delete message via WebSocket
 *
 * @param messageId - Message ID to delete
 */
export async function deleteMessage(
  messageId: string
): Promise<void> {
  const ws = (await import('./websocket')).getWebSocketClient()
  
  // Wait for connection
  await ws.waitForConnection()
  
  return new Promise((resolve, reject) => {
    ws.send({
      type: 'delete_message',
      message_id: messageId
    })
    
    // Listen for response
    const timeout = setTimeout(() => reject(new Error('Timeout')), 5000)
    const unsubscribe = ws.onMessage((msg: any) => {
      if (msg.type === 'message_deleted') {
        clearTimeout(timeout)
        unsubscribe()
        resolve()
      } else if (msg.type === 'error') {
        clearTimeout(timeout)
        unsubscribe()
        reject(new Error(msg.payload?.error || 'Failed to delete message'))
      }
    })
  })
}

/**
 * Edit message via WebSocket
 *
 * @param messageId - Message ID to edit
 * @param newContent - New encrypted content
 */
export async function editMessage(
  messageId: string,
  newContent: string
): Promise<void> {
  const ws = (await import('./websocket')).getWebSocketClient()
  
  // Wait for connection
  await ws.waitForConnection()
  
  return new Promise((resolve, reject) => {
    ws.send({
      type: 'alter_message',
      message_id: messageId,
      new_content: newContent
    })
    
    // Listen for response
    const timeout = setTimeout(() => reject(new Error('Timeout')), 5000)
    const unsubscribe = ws.onMessage((msg: any) => {
      if (msg.type === 'message_altered') {
        clearTimeout(timeout)
        unsubscribe()
        resolve()
      } else if (msg.type === 'error') {
        clearTimeout(timeout)
        unsubscribe()
        reject(new Error(msg.payload?.error || 'Failed to edit message'))
      }
    })
  })
}

/**
 * Get user's chats via WebSocket
 *
 * @returns List of chats with encrypted keys
 */
export async function getChats(): Promise<Chat[]> {
  const ws = (await import('./websocket')).getWebSocketClient()
  
  // Wait for connection
  await ws.waitForConnection()
  
  return new Promise((resolve, reject) => {
    ws.send({
      type: 'get_chats'
    })
    
    // Listen for response
    const timeout = setTimeout(() => reject(new Error('Timeout')), 5000)
    const unsubscribe = ws.onMessage((msg: any) => {
      if (msg.type === 'chats') {
        clearTimeout(timeout)
        unsubscribe()
        resolve(msg.payload?.chats || [])
      }
    })
  })
}

/**
 * Save chat encryption keys via WebSocket
 * Used when creating new chat or rotating keys
 *
 * @param data - Chat keys data
 */
export async function saveChatKeys(
  data: SaveChatKeysRequest
): Promise<{ success: boolean }> {
  const ws = (await import('./websocket')).getWebSocketClient()
  
  // Wait for connection
  await ws.waitForConnection()
  
  return new Promise((resolve, reject) => {
    ws.send({
      type: 'save_chat_keys',
      user_id: data.user_id,
      companion_id: data.companion_id,
      encrypted_key: data.encrypted_key,
      key_iv: data.key_iv
    })
    
    // Listen for response
    const timeout = setTimeout(() => reject(new Error('Timeout')), 5000)
    const unsubscribe = ws.onMessage((msg: any) => {
      if (msg.type === 'chat_keys_saved') {
        clearTimeout(timeout)
        unsubscribe()
        resolve({ success: true })
      } else if (msg.type === 'error') {
        clearTimeout(timeout)
        unsubscribe()
        reject(new Error(msg.error || 'Failed to save chat keys'))
      }
    })
  })
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
