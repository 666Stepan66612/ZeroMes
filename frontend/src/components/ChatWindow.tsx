import { useState, useEffect, useRef } from 'react';
import type { FormEvent, MouseEvent } from 'react';
import type { Chat, Message } from '@/types/api';
import { getMessages, sendMessage as sendMessageAPI, deleteMessage, editMessage } from '@/lib/api/messages';
import { encryptMessage, decryptChatKeyWithPrivateKey } from '@/lib/crypto/encryption';
import { restorePrivateKey } from '@/lib/crypto/keys';
import { getWebSocketClient } from '@/lib/api/websocket';
import { ContextMenu } from './ContextMenu';
import { ConfirmDialog } from './ConfirmDialog';
import type { EncryptedMessage } from '@/types/crypto';

interface ChatWindowProps {
  chat: Chat;
  onChatUpdate: () => void;
}

interface DecryptedMessage extends Message {
  decryptedContent?: string;
}

interface ContextMenuState {
  visible: boolean;
  x: number;
  y: number;
  messageId: string;
  messageContent: string;
}

export function ChatWindow({ chat, onChatUpdate }: ChatWindowProps) {
  const [messages, setMessages] = useState<DecryptedMessage[]>([]);
  const [messageText, setMessageText] = useState('');
  const [loading, setLoading] = useState(false);
  const [sending, setSending] = useState(false);
  const [chatKey, setChatKey] = useState<Uint8Array | null>(null);
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);
  const [editingMessage, setEditingMessage] = useState<{ id: string; content: string } | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Decrypt chat key when chat changes
  useEffect(() => {
    const decryptKey = async () => {
      try {
        const privateKey = restorePrivateKey();
        if (!privateKey || !chat.encrypted_key) {
          console.error('Missing private key or encrypted chat key');
          setChatKey(null);
          return;
        }
        const key = await decryptChatKeyWithPrivateKey(chat.encrypted_key, privateKey);
        setChatKey(key);
      } catch (error) {
        console.error('Failed to decrypt chat key:', error);
        setChatKey(null);
      }
    };
    decryptKey();
  }, [chat.id, chat.encrypted_key]);

  useEffect(() => {
    if (chat.id && chatKey) {
      loadMessages();
    }
  }, [chat.id, chatKey]);

  // Subscribe to WebSocket messages for real-time updates
  useEffect(() => {
    if (!chat.id || !chatKey) return;

    const ws = getWebSocketClient();
    
    const unsubscribe = ws.onMessage(async (message: any) => {
      console.log('[ChatWindow] WebSocket message received:', message.type, message);
      
      // Handle sent messages (our own messages)
      if (message.type === 'message_sent') {
        const msg = message.payload;
        
        // Only process messages for this chat
        if (msg.chat_id !== chat.id) return;
        
        console.log('[ChatWindow] Message sent confirmation:', msg);
        
        try {
          // Decrypt the message
          const encryptedMsg: EncryptedMessage = JSON.parse(msg.encrypted_content);
          
          // Import key for decryption
          const key = await crypto.subtle.importKey(
            'raw',
            chatKey as BufferSource,
            { name: 'AES-GCM' },
            false,
            ['decrypt']
          );
          
          // Decode base64
          const ciphertextBytes = Uint8Array.from(atob(encryptedMsg.ciphertext), c => c.charCodeAt(0));
          const nonceBytes = Uint8Array.from(atob(encryptedMsg.nonce), c => c.charCodeAt(0));
          
          // Decrypt
          const plaintext = await crypto.subtle.decrypt(
            { name: 'AES-GCM', iv: nonceBytes },
            key,
            ciphertextBytes
          );
          
          const decryptedText = new TextDecoder().decode(plaintext);
          
          // Create new message object
          const newMessage: DecryptedMessage = {
            id: msg.id,
            chat_id: msg.chat_id,
            sender_id: msg.sender_id,
            recipient_id: msg.recipient_id,
            encrypted_content: msg.encrypted_content,
            message_type: 'text',
            created_at: msg.created_at,
            status: 'delivered',
            decryptedContent: decryptedText,
          };
          
          // Add to messages if not already present
          setMessages(prev => {
            const exists = prev.some(m => m.id === newMessage.id);
            if (exists) return prev;
            return [...prev, newMessage];
          });
        } catch (error) {
          console.error('[ChatWindow] Failed to decrypt sent message:', error);
        }
      }
      
      // Handle new messages from others
      if (message.type === 'new_message') {
        const msg = message.payload;
        
        // Only process messages for this chat
        if (msg.chat_id !== chat.id) return;
        
        console.log('[ChatWindow] Received new message for current chat:', msg);
        
        try {
          // Decrypt the message
          const encryptedMsg: EncryptedMessage = JSON.parse(msg.encrypted_content);
          
          // Import key for decryption
          const key = await crypto.subtle.importKey(
            'raw',
            chatKey as BufferSource,
            { name: 'AES-GCM' },
            false,
            ['decrypt']
          );
          
          // Decode base64
          const ciphertextBytes = Uint8Array.from(atob(encryptedMsg.ciphertext), c => c.charCodeAt(0));
          const nonceBytes = Uint8Array.from(atob(encryptedMsg.nonce), c => c.charCodeAt(0));
          
          // Decrypt
          const plaintext = await crypto.subtle.decrypt(
            { name: 'AES-GCM', iv: nonceBytes },
            key,
            ciphertextBytes
          );
          
          const decryptedText = new TextDecoder().decode(plaintext);
          
          // Create new message object
          const newMessage: DecryptedMessage = {
            id: msg.message_id,
            chat_id: msg.chat_id,
            sender_id: msg.sender_id,
            recipient_id: chat.companion_id,
            encrypted_content: msg.encrypted_content,
            message_type: 'text',
            created_at: msg.timestamp,
            status: 'delivered',
            decryptedContent: decryptedText,
          };
          
          // Add to messages if not already present
          setMessages(prev => {
            const exists = prev.some(m => m.id === newMessage.id);
            if (exists) return prev;
            return [...prev, newMessage];
          });
        } catch (error) {
          console.error('[ChatWindow] Failed to decrypt incoming message:', error);
        }
      }
      
      // Handle message deletion
      if (message.type === 'message_deleted') {
        const messageId = message.payload?.message_id;
        if (messageId) {
          console.log('[ChatWindow] Message deleted:', messageId);
          setMessages(prev => prev.filter(m => m.id !== messageId));
        }
      }
      
      // Handle message edit
      if (message.type === 'message_altered') {
        const { message_id, new_content } = message.payload || {};
        if (message_id && new_content) {
          console.log('[ChatWindow] Message altered:', message_id);
          try {
            // Decrypt the new content
            const encryptedMsg: EncryptedMessage = JSON.parse(new_content);
            
            const key = await crypto.subtle.importKey(
              'raw',
              chatKey as BufferSource,
              { name: 'AES-GCM' },
              false,
              ['decrypt']
            );
            
            const ciphertextBytes = Uint8Array.from(atob(encryptedMsg.ciphertext), c => c.charCodeAt(0));
            const nonceBytes = Uint8Array.from(atob(encryptedMsg.nonce), c => c.charCodeAt(0));
            
            const plaintext = await crypto.subtle.decrypt(
              { name: 'AES-GCM', iv: nonceBytes },
              key,
              ciphertextBytes
            );
            
            const decryptedText = new TextDecoder().decode(plaintext);
            
            // Update message in state
            setMessages(prev => prev.map(m => 
              m.id === message_id 
                ? { ...m, encrypted_content: new_content, decryptedContent: decryptedText }
                : m
            ));
          } catch (error) {
            console.error('[ChatWindow] Failed to decrypt altered message:', error);
          }
        }
      }
    });

    return () => {
      unsubscribe();
    };
  }, [chat.id, chatKey]);

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const loadMessages = async () => {
    if (!chat.id || !chatKey) return;

    try {
      setLoading(true);
      const response = await getMessages({
        chat_id: chat.id,
        limit: 50,
      });
      
      // Decrypt all messages
      const decryptedMessages = await Promise.all(
        response.messages.map(async (msg) => {
          try {
            const encryptedMsg: EncryptedMessage = JSON.parse(msg.encrypted_content);
            
            // Import key for decryption
            const key = await crypto.subtle.importKey(
              'raw',
              chatKey as BufferSource,
              { name: 'AES-GCM' },
              false,
              ['decrypt']
            );
            
            // Decode base64
            const ciphertextBytes = Uint8Array.from(atob(encryptedMsg.ciphertext), c => c.charCodeAt(0));
            const nonceBytes = Uint8Array.from(atob(encryptedMsg.nonce), c => c.charCodeAt(0));
            
            // Decrypt
            const plaintext = await crypto.subtle.decrypt(
              { name: 'AES-GCM', iv: nonceBytes },
              key,
              ciphertextBytes
            );
            
            const decryptedText = new TextDecoder().decode(plaintext);
            return { ...msg, decryptedContent: decryptedText };
          } catch (error) {
            console.error('Failed to decrypt message:', error);
            return { ...msg, decryptedContent: '[Decryption failed]' };
          }
        })
      );
      
      setMessages(decryptedMessages);
    } catch (error) {
      console.error('Failed to load messages:', error);
    } finally {
      setLoading(false);
    }
  };

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  const handleSendMessage = async (e: FormEvent) => {
    e.preventDefault();

    if (!messageText.trim() || sending || !chatKey) return;

    try {
      setSending(true);

      // Encrypt message before sending
      const encrypted = await encryptMessage(messageText, chatKey);
      const encryptedContent = JSON.stringify(encrypted);

      console.log('[ChatWindow] Sending message:', {
        chat_id: chat.id,
        recipient_id: chat.companion_id,
        encrypted_content: encryptedContent,
        message_type: 'text',
      });

      const sentMessage = await sendMessageAPI({
        chat_id: chat.id,
        recipient_id: chat.companion_id,
        encrypted_content: encryptedContent,
        message_type: 'text',
      });

      console.log('[ChatWindow] Message sent, response:', sentMessage);

      // Add the sent message to the list immediately
      const newMessage: DecryptedMessage = {
        ...sentMessage,
        decryptedContent: messageText, // We already have the decrypted text
      };

      setMessages(prev => [...prev, newMessage]);
      setMessageText('');
      onChatUpdate();
    } catch (error) {
      console.error('Failed to send message:', error);
    } finally {
      setSending(false);
    }
  };

  const handleContextMenu = (e: MouseEvent, message: DecryptedMessage) => {
    e.preventDefault();
    setContextMenu({
      visible: true,
      x: e.clientX,
      y: e.clientY,
      messageId: message.id,
      messageContent: message.decryptedContent || '',
    });
  };

  const handleCopyMessage = () => {
    if (contextMenu) {
      navigator.clipboard.writeText(contextMenu.messageContent);
      setContextMenu(null);
    }
  };

  const handleEditMessage = () => {
    if (contextMenu) {
      setEditingMessage({
        id: contextMenu.messageId,
        content: contextMenu.messageContent,
      });
      setMessageText(contextMenu.messageContent);
      setContextMenu(null);
    }
  };

  const handleDeleteMessage = () => {
    if (contextMenu) {
      setDeleteConfirm(contextMenu.messageId);
      setContextMenu(null);
    }
  };

  const confirmDelete = async () => {
    if (!deleteConfirm) return;

    try {
      await deleteMessage(deleteConfirm);
      setMessages(prev => prev.filter(m => m.id !== deleteConfirm));
      setDeleteConfirm(null);
    } catch (error) {
      console.error('Failed to delete message:', error);
      alert('Failed to delete message');
    }
  };

  const handleSubmitEdit = async (e: FormEvent) => {
    e.preventDefault();

    if (!messageText.trim() || !editingMessage || !chatKey) return;

    try {
      setSending(true);

      // Encrypt the new content
      const encrypted = await encryptMessage(messageText, chatKey);
      const encryptedContent = JSON.stringify(encrypted);

      await editMessage(editingMessage.id, encryptedContent);

      // Update local state
      setMessages(prev => prev.map(m => 
        m.id === editingMessage.id 
          ? { ...m, encrypted_content: encryptedContent, decryptedContent: messageText }
          : m
      ));

      setMessageText('');
      setEditingMessage(null);
    } catch (error) {
      console.error('Failed to edit message:', error);
      alert('Failed to edit message');
    } finally {
      setSending(false);
    }
  };

  const cancelEdit = () => {
    setEditingMessage(null);
    setMessageText('');
  };

  return (
    <div className="chat-window">
      <div className="chat-window-header">
        <div className="chat-avatar">
          {(chat.companion_login || chat.companion_id).substring(0, 2).toUpperCase()}
        </div>
        <div className="chat-header-info">
          <h3>{chat.companion_login || chat.companion_id}</h3>
          <span className="chat-status">Online</span>
        </div>
      </div>

      <div className="chat-messages">
        {loading ? (
          <div className="loading">Loading messages...</div>
        ) : messages.length === 0 ? (
          <div className="no-messages">
            <p>No messages yet</p>
            <p className="help-text">Send a message to start the conversation</p>
          </div>
        ) : (
          messages.map((message) => (
            <div
              key={message.id}
              className={`message ${message.sender_id === chat.companion_id ? 'received' : 'sent'}`}
              onContextMenu={(e) => handleContextMenu(e, message)}
            >
              <div className="message-content">
                {message.decryptedContent || message.encrypted_content}
              </div>
              <div className="message-time">
                {new Date(message.created_at).toLocaleTimeString([], {
                  hour: '2-digit',
                  minute: '2-digit',
                })}
              </div>
            </div>
          ))
        )}
        <div ref={messagesEndRef} />
      </div>

      <form className="chat-input" onSubmit={editingMessage ? handleSubmitEdit : handleSendMessage}>
        {editingMessage && (
          <div className="edit-mode-banner">
            <span>✏️ Editing message</span>
            <button type="button" onClick={cancelEdit} className="btn-cancel-edit">
              Cancel
            </button>
          </div>
        )}
        <input
          type="text"
          value={messageText}
          onChange={(e) => setMessageText(e.target.value)}
          placeholder={editingMessage ? "Edit your message..." : "Type a message..."}
          disabled={sending || !chatKey}
        />
        <button
          type="submit"
          className="btn-send"
          disabled={!messageText.trim() || sending || !chatKey}
        >
          {sending ? '⏳' : editingMessage ? '✓' : '📤'}
        </button>
      </form>

      {contextMenu && (
        <ContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          onCopy={handleCopyMessage}
          onEdit={handleEditMessage}
          onDelete={handleDeleteMessage}
          onClose={() => setContextMenu(null)}
        />
      )}

      {deleteConfirm && (
        <ConfirmDialog
          title="Delete Message"
          message="Are you sure you want to delete this message? This action cannot be undone."
          confirmText="Delete"
          cancelText="Cancel"
          danger={true}
          onConfirm={confirmDelete}
          onCancel={() => setDeleteConfirm(null)}
        />
      )}
    </div>
  );
}
