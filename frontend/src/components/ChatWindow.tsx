import { useState, useEffect, useRef, useCallback } from 'react';
import { checkOnlineStatus } from '@/lib/api/messages';
import type { FormEvent, MouseEvent } from 'react';
import type { Chat, Message } from '@/types/api';
import { getMessages, sendMessage as sendMessageAPI, deleteMessage, editMessage, markAsRead } from '@/lib/api/messages';
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
  localStatus?: 'pending' | 'sent' | 'delivered' | 'read';
}

interface ContextMenuState {
  visible: boolean;
  x: number;
  y: number;
  messageId: string;
  messageContent: string;
}

export function ChatWindow({ chat }: ChatWindowProps) {
  const [messages, setMessages] = useState<DecryptedMessage[]>([]);
  const [messageText, setMessageText] = useState('');
  const [loading, setLoading] = useState(false);
  const [chatKey, setChatKey] = useState<Uint8Array | null>(null);
  const [isOnline, setIsOnline] = useState(false);
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);
  const [editingMessage, setEditingMessage] = useState<{ id: string; content: string } | null>(null);
  const [hasMore, setHasMore] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messagesContainerRef = useRef<HTMLDivElement>(null);

  // Decrypt chat key when chat changes
  useEffect(() => {
    const decryptKey = async () => {
      try {
        const privateKey = await restorePrivateKey();
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
    // Reset pagination state when chat changes
    setMessages([]);
    setHasMore(true);
    setLoadingMore(false);
    
    if (chat.id && chatKey) {
      loadMessages();
      markMessagesAsRead();
    }
  }, [chat.id, chatKey]);

  // Subscribe to WebSocket messages for real-time updates
  useEffect(() => {
    if (!chat.id || !chatKey) return;

    const ws = getWebSocketClient();
    
    const unsubscribe = ws.onMessage(async (message: any) => {
      console.log('[ChatWindow] WebSocket message received:', message.type, message);
      
      // Handle sent messages (our own messages)
      // Note: We handle this in handleSendMessage directly, so we can skip this event
      // to avoid duplicate messages. The WebSocket event is already consumed by sendMessageAPI.
      if (message.type === 'message_sent') {
        // Skip - already handled in handleSendMessage
        return;
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
      
      // Handle message read status updates
      if (message.type === 'message_read') {
        const { chat_id, last_message_id } = message.payload || {};
        if (chat_id === chat.id && last_message_id) {
          console.log('[ChatWindow] Messages marked as read up to:', last_message_id);
          // Update all messages up to last_message_id to 'read' status
          setMessages(prev => {
            const lastReadIndex = prev.findIndex(msg => msg.id === last_message_id);
            if (lastReadIndex === -1) return prev;
            
            return prev.map((m, index) => {
              // Mark all our messages (sender is NOT companion) up to and including last_message_id as read
              const isOurMessage = m.sender_id !== chat.companion_id;
              if (index <= lastReadIndex && isOurMessage) {
                return { ...m, status: 'read' as const, localStatus: 'read' as const };
              }
              return m;
            });
          });
        }
      }
    });

    return () => {
      unsubscribe();
    };
  }, [chat.id, chatKey, chat.companion_id]);

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
      
      console.log('[ChatWindow] Loaded messages:', response.messages.map(m => ({
        id: m.id.substring(0, 8),
        status: m.status,
        sender_id: m.sender_id.substring(0, 8)
      })));
      
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
      
      // Reverse to show oldest first (backend returns newest first)
      setMessages(decryptedMessages.reverse());
      
      // Update hasMore flag from backend response
      setHasMore(response.has_more || false);
    } catch (error) {
      console.error('Failed to load messages:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadMoreMessages = useCallback(async () => {
    console.log('[ChatWindow] loadMoreMessages called', {
      hasChat: !!chat.id,
      hasChatKey: !!chatKey,
      hasMore,
      loadingMore,
      messagesLength: messages.length
    });
    
    if (!chat.id || !chatKey || !hasMore || loadingMore || messages.length === 0) {
      console.log('[ChatWindow] loadMoreMessages early return');
      return;
    }
    
    // Get ID of the oldest message (first in array after reverse)
    const oldestMessage = messages[0];
    if (!oldestMessage) {
      console.log('[ChatWindow] No oldest message found');
      return;
    }
    
    console.log('[ChatWindow] Loading more messages, oldest message ID:', oldestMessage.id);
    
    try {
      setLoadingMore(true);
      
      // Save current scroll position
      const container = messagesContainerRef.current;
      const oldScrollHeight = container?.scrollHeight || 0;
      
      console.log('[ChatWindow] Requesting messages with last_message_id:', oldestMessage.id);
      const response = await getMessages({
        chat_id: chat.id,
        limit: 50,
        last_message_id: oldestMessage.id,
      });
      
      console.log('[ChatWindow] Received response:', {
        messagesCount: response.messages.length,
        hasMore: response.has_more
      });
      
      if (response.messages.length === 0) {
        setHasMore(false);
        return;
      }
      
      // Decrypt messages
      const decryptedMessages = await Promise.all(
        response.messages.map(async (msg) => {
          try {
            const encryptedMsg: EncryptedMessage = JSON.parse(msg.encrypted_content);
            
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
            return { ...msg, decryptedContent: decryptedText };
          } catch (error) {
            console.error('Failed to decrypt message:', error);
            return { ...msg, decryptedContent: '[Decryption failed]' };
          }
        })
      );
      
      // Reverse to maintain order (backend returns DESC)
      const reversedMessages = decryptedMessages.reverse();
      
      // Add to the beginning of the array
      setMessages(prev => [...reversedMessages, ...prev]);
      
      // Restore scroll position
      setTimeout(() => {
        if (container) {
          const newScrollHeight = container.scrollHeight;
          container.scrollTop = newScrollHeight - oldScrollHeight;
        }
      }, 0);
      
      // Update hasMore flag
      setHasMore(response.has_more || false);
      
    } catch (error) {
      console.error('Failed to load more messages:', error);
    } finally {
      setLoadingMore(false);
    }
  }, [chat.id, chatKey, hasMore, loadingMore, messages]);

  // Handle scroll for pagination
  useEffect(() => {
    const container = messagesContainerRef.current;
    if (!container) {
      console.log('[ChatWindow] No messages container ref');
      return;
    }
    
    console.log('[ChatWindow] Setting up scroll handler');
    
    const handleScroll = () => {
      console.log('[ChatWindow] Scroll event:', {
        scrollTop: container.scrollTop,
        hasMore,
        loadingMore
      });
      
      // Load more when scrolled near the top
      if (container.scrollTop < 100 && hasMore && !loadingMore) {
        console.log('[ChatWindow] Triggering loadMoreMessages from scroll');
        loadMoreMessages();
      }
    };
    
    container.addEventListener('scroll', handleScroll);
    return () => container.removeEventListener('scroll', handleScroll);
  }, [hasMore, loadingMore, loadMoreMessages]);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  const markMessagesAsRead = async () => {
    if (!chat.id || messages.length === 0) return;

    // Find the last message from companion
    const lastCompanionMessage = [...messages]
      .reverse()
      .find(m => m.sender_id === chat.companion_id);

    if (lastCompanionMessage && lastCompanionMessage.status !== 'read') {
      try {
        await markAsRead(chat.id, lastCompanionMessage.id);
        console.log('[ChatWindow] Marked messages as read up to:', lastCompanionMessage.id);
      } catch (error) {
        console.error('[ChatWindow] Failed to mark messages as read:', error);
      }
    }
  };

  // Mark messages as read when new messages arrive
  useEffect(() => {
    if (messages.length > 0) {
      markMessagesAsRead();
    }
  }, [messages.length]);

  // Check online status when chat opens
  useEffect(() => {
    const checkStatus = async () => {
      try {
        const status = await checkOnlineStatus(chat.companion_id);
        setIsOnline(status);
      } catch (error) {
        console.error('Failed to check online status:', error);
      }
    };

    checkStatus();
    // Check every 2 seconds to match chat list
    const interval = setInterval(checkStatus, 2000);

    return () => clearInterval(interval);
  }, [chat.companion_id]);

  const handleSendMessage = async (e: FormEvent) => {
    e.preventDefault();

    if (!messageText.trim() || !chatKey) return;

    const startTime = performance.now();
    const tempId = `temp-${Date.now()}`;
    const tempMessage: DecryptedMessage = {
      id: tempId,
      chat_id: chat.id,
      sender_id: '', // Will be filled by server
      recipient_id: chat.companion_id,
      encrypted_content: '',
      message_type: 'text',
      created_at: new Date().toISOString(),
      status: 'sent',
      localStatus: 'pending',
      decryptedContent: messageText,
    };

    // Add pending message immediately
    setMessages(prev => [...prev, tempMessage]);
    const textToSend = messageText;
    setMessageText('');

    try {
      // Encrypt message before sending
      const encryptStart = performance.now();
      const encrypted = await encryptMessage(textToSend, chatKey);
      const encryptedContent = JSON.stringify(encrypted);
      const encryptTime = performance.now() - encryptStart;
      console.log(`[Performance] Encryption took: ${encryptTime.toFixed(2)}ms`);

      const sendStart = performance.now();
      const sentMessage = await sendMessageAPI({
        chat_id: chat.id,
        recipient_id: chat.companion_id,
        encrypted_content: encryptedContent,
        message_type: 'text',
      });
      const sendTime = performance.now() - sendStart;
      console.log(`[Performance] WebSocket send took: ${sendTime.toFixed(2)}ms`);
      console.log(`[Performance] Total time: ${(performance.now() - startTime).toFixed(2)}ms`);

      // Replace temp message with real one
      setMessages(prev => prev.map(m =>
        m.id === tempId
          ? { ...sentMessage, decryptedContent: textToSend, localStatus: 'sent' }
          : m
      ));
      
      // Don't call onChatUpdate() to avoid re-rendering the chat list
      // The chat list will be updated via WebSocket 'chats' event
    } catch (error) {
      console.error('Failed to send message:', error);
      // Remove temp message on error
      setMessages(prev => prev.filter(m => m.id !== tempId));
      setMessageText(textToSend); // Restore message text
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
          <span className="chat-status">{isOnline ? 'Online' : 'Offline'}</span>
        </div>
      </div>

      <div className="chat-messages" ref={messagesContainerRef}>
        {loading ? (
          <div className="loading">Loading messages...</div>
        ) : messages.length === 0 ? (
          <div className="no-messages">
            <p>No messages yet</p>
            <p className="help-text">Send a message to start the conversation</p>
          </div>
        ) : (
          <>
            {loadingMore && (
              <div className="loading-more" style={{ textAlign: 'center', padding: '10px', color: '#666' }}>
                Loading older messages...
              </div>
            )}
            {messages.map((message) => {
            const isSent = message.sender_id !== chat.companion_id;
            const displayStatus: string | number | undefined = message.localStatus || message.status;
            
            // Debug logging for status
            if (isSent && message.id) {
              console.log(`[ChatWindow] Message ${message.id.substring(0, 8)} status:`, {
                localStatus: message.localStatus,
                status: message.status,
                displayStatus,
                displayStatusType: typeof displayStatus,
                isDelivered: displayStatus === 'delivered',
                isRead: displayStatus === 'read'
              });
            }
            
            return (
              <div
                key={message.id}
                className={`message ${isSent ? 'sent' : 'received'}`}
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
                  {isSent && (
                    <span className="message-status">
                      {displayStatus === 'pending' && ' ⏰'}
                      {/* @ts-ignore - backend sends numbers, frontend uses strings */}
                      {(displayStatus === 'sent' || displayStatus == 0) && ' ✓'}
                      {/* @ts-ignore */}
                      {(displayStatus === 'delivered' || displayStatus == 1) && ' ✓'}
                      {/* @ts-ignore */}
                      {(displayStatus === 'read' || displayStatus == 2) && ' ✓✓'}
                    </span>
                  )}
                </div>
              </div>
            );
          })}
          </>
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
          disabled={!chatKey}
        />
        <button
          type="submit"
          className="btn-send"
          disabled={!messageText.trim() || !chatKey}
        >
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
            <line x1="12" y1="19" x2="12" y2="5"></line>
            <polyline points="5 12 12 5 19 12"></polyline>
          </svg>
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

