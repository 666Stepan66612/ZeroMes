import { useState, useEffect, useRef, useCallback } from 'react';
import { checkOnlineStatus } from '@/lib/api/messages';
import type { FormEvent, MouseEvent } from 'react';
import type { Chat, Message } from '@/types/api';
import { getMessages, sendMessage as sendMessageAPI, deleteMessage, editMessage, markAsRead } from '@/lib/api/messages';
import { encryptMessage, decryptChatKeyWithPrivateKey } from '@/lib/crypto/encryption';
import { restorePrivateKey } from '@/lib/crypto/keys';
import { getWebSocketClient } from '@/lib/api/websocket';
import { decryptMessageContent } from '@/lib/crypto/messageDecryption';
import { ContextMenu } from './ContextMenu';
import { ConfirmDialog } from './ConfirmDialog';
import { VirtualizedMessageList } from './VirtualizedMessageList';
import { useToast } from '@/hooks/useToast';

interface ChatWindowProps {
  chat: Chat;
  onChatUpdate: () => void;
  onBack?: () => void;
}

interface DecryptedMessage extends Omit<Message, 'status'> {
  decryptedContent?: string;
  localStatus?: 'pending' | 'sent' | 'delivered' | 'read';
  status: string | number;
}

interface ContextMenuState {
  visible: boolean;
  x: number;
  y: number;
  messageId: string;
  messageContent: string;
}

export function ChatWindow({ chat, onBack }: ChatWindowProps) {
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
  const [showScrollButton, setShowScrollButton] = useState(false);
  const [triggerScrollToBottom, setTriggerScrollToBottom] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messagesContainerRef = useRef<HTMLDivElement>(null);
  const initialLoadDone = useRef(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const { showError, showSuccess } = useToast();

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
    initialLoadDone.current = false;

    if (chat.id && chatKey) {
      loadMessages().then(() => {
        setTimeout(() => {
          markMessagesAsRead();
          initialLoadDone.current = true;
        }, 100);
      });
    }

    setTimeout(() => {
      inputRef.current?.focus();
    }, 100);
  }, [chat.id, chatKey]);

  // Subscribe to WebSocket messages for real-time updates
  useEffect(() => {
    if (!chat.id || !chatKey) return;

    const ws = getWebSocketClient();
    let isSubscribed = true; // Flag to prevent state updates after unmount
    
    const unsubscribe = ws.onMessage(async (message: any) => {
      // Check if still subscribed before updating state
      if (!isSubscribed) return;
      
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
        
        try {
          // Decrypt the message using helper
          const decryptedText = await decryptMessageContent(msg.encrypted_content, chatKey);
          
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
          setMessages(prev => prev.filter(m => m.id !== messageId));
        }
      }
      
      // Handle message edit
      if (message.type === 'message_altered') {
        const { message_id, new_content } = message.payload || {};
        if (message_id && new_content) {
          try {
            // Decrypt the new content using helper
            const decryptedText = await decryptMessageContent(new_content, chatKey);
            
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
      isSubscribed = false; // Prevent state updates after cleanup
      unsubscribe();
    };
  }, [chat.id, chatKey, chat.companion_id]);

  const scrollToBottom = () => {
    setTriggerScrollToBottom(true);
    setTimeout(() => setTriggerScrollToBottom(false), 100);
  };

  const handleAtBottomChange = useCallback((atBottom: boolean) => {
    setShowScrollButton(!atBottom);
  }, []);

  const loadMessages = async () => {
    if (!chat.id || !chatKey) return;

    try {
      setLoading(true);
      const response = await getMessages({
        chat_id: chat.id,
        limit: 50,
      });


      const decryptedMessages = await Promise.all(
        response.messages.map(async (msg) => {
          try {
            const decryptedText = await decryptMessageContent(msg.encrypted_content, chatKey);
            return { ...msg, decryptedContent: decryptedText };
          } catch (error) {
            console.error('Failed to decrypt message:', error);
            showError('Failed to decrypt some messages');
            return { ...msg, decryptedContent: '[Decryption failed]' };
          }
        })
      );
      
      setMessages(decryptedMessages.reverse());

      setHasMore(response.has_more || false);

      setTimeout(() => {
        initialLoadDone.current = true;
      }, 100);
    } catch (error) {
      console.error('Failed to load messages:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadMoreMessages = useCallback(async () => {
    if (!chat.id || !chatKey || !hasMore || loadingMore || messages.length === 0) {
      return;
    }

    const oldestMessage = messages[0];
    if (!oldestMessage) {
      return;
    }

    try {
      setLoadingMore(true);

      const response = await getMessages({
        chat_id: chat.id,
        limit: 50,
        last_message_id: oldestMessage.id,
      });

      if (response.messages.length === 0) {
        setHasMore(false);
        return;
      }

      const decryptedMessages = await Promise.all(
        response.messages.map(async (msg) => {
          try {
            const decryptedText = await decryptMessageContent(msg.encrypted_content, chatKey);
            return { ...msg, decryptedContent: decryptedText };
          } catch (error) {
            console.error('Failed to decrypt message:', error);
            return { ...msg, decryptedContent: '[Decryption failed]' };
          }
        })
      );

      const reversedMessages = decryptedMessages.reverse();

      setMessages(prev => [...reversedMessages, ...prev]);

      setHasMore(response.has_more || false);

    } catch (error) {
      console.error('Failed to load more messages:', error);
    } finally {
      setLoadingMore(false);
    }
  }, [chat.id, chatKey, hasMore, loadingMore, messages]);

  const markMessagesAsRead = async () => {
    if (!chat.id || messages.length === 0) return;

    const lastCompanionMessage = [...messages]
      .reverse()
      .find(m => m.sender_id === chat.companion_id);

    if (lastCompanionMessage && lastCompanionMessage.status !== 'read') {
      try {
        await markAsRead(chat.id, lastCompanionMessage.id);
      } catch (error) {
        console.error('Failed to mark messages as read:', error);
      }
    }
  };

  useEffect(() => {
    if (messages.length > 0) {
      markMessagesAsRead();
    }
  }, [messages.length]);

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
    const interval = setInterval(checkStatus, 2000);

    return () => clearInterval(interval);
  }, [chat.companion_id]);

  const handleSendMessage = async (e: FormEvent) => {
    e.preventDefault();

    if (!messageText.trim() || !chatKey) return;

    const startTime = performance.now();
    const tempId = `temp-${Date.now()}`;

    // Use a placeholder sender_id that's definitely not the companion
    // The real sender_id will come from the server response
    const currentUserId = chat.user_id || 'me';

    const tempMessage: DecryptedMessage = {
      id: tempId,
      chat_id: chat.id,
      sender_id: currentUserId,
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
      showSuccess('Message deleted');
    } catch (error) {
      console.error('Failed to delete message:', error);
      showError('Failed to delete message. Please try again.');
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
      showSuccess('Message updated');
    } catch (error) {
      console.error('Failed to edit message:', error);
      showError('Failed to edit message. Please try again.');
    }
  };

  const cancelEdit = () => {
    setEditingMessage(null);
    setMessageText('');
  };

  return (
    <div className="chat-window">
      <div className="chat-window-header">
        {onBack && (
          <button className="btn-back" onClick={onBack} aria-label="Back to chats">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <polyline points="15 18 9 12 15 6"></polyline>
            </svg>
          </button>
        )}
        <div className="chat-avatar">
          {(chat.companion_login || chat.companion_id).substring(0, 2).toUpperCase()}
        </div>
        <div className="chat-header-info">
          <h3>{chat.companion_login || chat.companion_id}</h3>
          <span className="chat-status">{isOnline ? 'Online' : 'Offline'}</span>
        </div>
      </div>

      <div className="chat-messages" ref={messagesContainerRef} style={{ padding: 0 }}>
        {loading ? (
          <div className="loading">Loading messages...</div>
        ) : (
          <VirtualizedMessageList
            messages={messages}
            chat={chat}
            onContextMenu={handleContextMenu}
            containerHeight={messagesContainerRef.current?.clientHeight || 500}
            onAtBottomChange={handleAtBottomChange}
            scrollToBottom={triggerScrollToBottom}
            onLoadMore={loadMoreMessages}
            hasMore={hasMore}
            loadingMore={loadingMore}
          />
        )}
        <div ref={messagesEndRef} />

        <button
          className={`scroll-to-bottom-btn ${showScrollButton ? 'visible' : ''}`}
          onClick={scrollToBottom}
          aria-label="Scroll to bottom"
        >
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
            <path d="M12 5v14M19 12l-7 7-7-7"/>
          </svg>
        </button>
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
          ref={inputRef}
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

