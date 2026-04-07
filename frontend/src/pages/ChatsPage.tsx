import { useState, useEffect, useMemo, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { getChats, saveChatKeys, getUserPublicKey, checkOnlineStatus, getMessages } from '@/lib/api';
import { getWebSocketClient } from '@/lib/api/websocket';
import { restorePrivateKey, fromHex, clearKeys, isRememberMeEnabled } from '@/lib/crypto';
import { deriveChatKey, encryptChatKeyWithPrivateKey, decryptMessage } from '@/lib/crypto/encryption';
import { ChatList, ChatWindow, SearchModal } from '@/components';
import type { Chat, User } from '@/types/api';
import './ChatsPage.css';

export function ChatsPage() {
  const navigate = useNavigate();
  const [chats, setChats] = useState<Chat[]>([]);
  const [selectedChatId, setSelectedChatId] = useState<string | null>(null);
  const [showSearch, setShowSearch] = useState(false);
  const [loading, setLoading] = useState(true);
  const [wsConnected, setWsConnected] = useState(false);
  const [chatUpdateTrigger, setChatUpdateTrigger] = useState(0);
  const chatsRef = useRef<Chat[]>([]);
  const selectedChatIdRef = useRef<string | null>(null);
  const processedMessagesRef = useRef<Set<string>>(new Set());

  // Keep refs in sync with state
  useEffect(() => {
    chatsRef.current = chats;
  }, [chats]);

  useEffect(() => {
    selectedChatIdRef.current = selectedChatId;
  }, [selectedChatId]);

  // Clear keys on tab close if remember_me is false
  useEffect(() => {
    const handleBeforeUnload = () => {
      if (!isRememberMeEnabled()) {
        clearKeys(); // Fire and forget - async cleanup
      }
    };

    window.addEventListener('beforeunload', handleBeforeUnload);
    return () => {
      window.removeEventListener('beforeunload', handleBeforeUnload);
    };
  }, []);

  useEffect(() => {
    // Check if user is logged in and initialize
    const initializeConnection = async () => {
      const privateKey = await restorePrivateKey();
      if (!privateKey) {
        navigate('/login');
        return;
      }
      const ws = getWebSocketClient();
      
      const unsubscribeStatus = ws.onStatus((status) => {
        setWsConnected(status === 'connected');
      });

      const unsubscribeMessage = ws.onMessage(async (message: any) => {
        console.log('New message:', message);
        
        // Handle new message notification (incoming messages from others)
        if (message.type === 'new_message') {
          const msg = message.payload;
          console.log('[ChatsPage] new_message payload:', msg);
          
          if (!msg || !msg.chat_id) {
            console.warn('[ChatsPage] Invalid message payload:', msg);
            return;
          }
          
          // Skip if we already processed this message
          const messageKey = `${msg.id}_${msg.chat_id}`;
          if (processedMessagesRef.current.has(messageKey)) {
            console.log('[ChatsPage] Skipping duplicate message:', messageKey);
            return;
          }
          processedMessagesRef.current.add(messageKey);
          
          // Clean up old entries (keep only last 100)
          if (processedMessagesRef.current.size > 100) {
            const entries = Array.from(processedMessagesRef.current);
            processedMessagesRef.current = new Set(entries.slice(-100));
          }
          
          // Check if we have this chat (use ref to get current value)
          const existingChat = chatsRef.current.find(c => c.id === msg.chat_id);
          
          if (!existingChat) {
            // New chat - need to generate keys
            console.log('[ChatsPage] New chat detected, generating keys for:', msg.chat_id);
            
            try {
              // For new_message event, the sender is the companion (we are the recipient)
              const companionId = msg.sender_id;
              
              console.log('[ChatsPage] Companion ID (sender):', companionId);
              console.log('[ChatsPage] Chat ID:', msg.chat_id);
              
              // Get companion's public key
              const companionPublicKey = await getUserPublicKey(companionId);
              const companionPubKeyBytes = fromHex(companionPublicKey);
              
              // Derive chat key using ECDH
              const chatKey = deriveChatKey(privateKey, companionPubKeyBytes);
              
              // Encrypt chat key with private key for storage
              const { ciphertext } = await encryptChatKeyWithPrivateKey(chatKey, privateKey);
              
              // Save to server (user_id will be taken from JWT token on backend)
              await saveChatKeys({
                user_id: '', // Not used, backend gets it from JWT
                companion_id: companionId,
                encrypted_key: ciphertext,
                key_iv: '',
              });
              
              // Reload chats to get the new one
              await loadChats();
            } catch (error) {
              console.error('[ChatsPage] Failed to generate keys for new chat:', error);
            }
          } else {
            // Existing chat - update last message preview
            console.log('[ChatsPage] Message for existing chat:', msg.chat_id);
            console.log('[ChatsPage] Existing chat object:', existingChat);
            console.log('[ChatsPage] Message encrypted_content:', msg.encrypted_content);
            
            try {
              // Decrypt the message content
              const companionPublicKey = await getUserPublicKey(existingChat.companion_id);
              const companionPubKeyBytes = fromHex(companionPublicKey);
              const chatKey = deriveChatKey(privateKey, companionPubKeyBytes);
              
              const encryptedData = JSON.parse(msg.encrypted_content);
              const decrypted = await decryptMessage(
                encryptedData.ciphertext,
                encryptedData.nonce,
                chatKey
              );
              
              console.log('[ChatsPage] Decrypted message:', decrypted);
              
              const preview = decrypted.length > 50 ? decrypted.substring(0, 50) + '...' : decrypted;
              
              console.log('[ChatsPage] Updating chat with preview:', preview);
              console.log('[ChatsPage] msg.created_at:', msg.created_at, 'type:', typeof msg.created_at);
              
              // Use current time if created_at is invalid
              const messageTime = msg.created_at || new Date().toISOString();
              
              // Update chats state with new last message preview and increment unread count
              setChats(prevChats => {
                const updated = prevChats.map(chat => {
                  if (chat.id === msg.chat_id) {
                    // Increment unread count only if this chat is not currently selected (use ref for current value)
                    const isCurrentChat = selectedChatIdRef.current === chat.id;
                    const newUnreadCount = isCurrentChat ? 0 : (chat.unread_count || 0) + 1;
                    console.log('[ChatsPage] Updating unread count for chat', chat.id, 'from', chat.unread_count, 'to', newUnreadCount, 'isCurrentChat:', isCurrentChat, 'selectedChatIdRef:', selectedChatIdRef.current);
                    return {
                      ...chat,
                      last_message_preview: preview,
                      last_message_at: messageTime,
                      unread_count: newUnreadCount
                    };
                  }
                  return chat;
                }).sort((a, b) => new Date(b.last_message_at).getTime() - new Date(a.last_message_at).getTime());
                
                console.log('[ChatsPage] Updated chats:', updated);
                return updated;
              });
            } catch (error) {
              console.error('[ChatsPage] Failed to decrypt new message:', error);
            }
            
            // Trigger chat update if this chat is currently open
            if (selectedChat && selectedChat.id === msg.chat_id) {
              setChatUpdateTrigger(prev => prev + 1);
            }
          }
        }
        
        // Handle message_sent confirmation (update last message for sender)
        if (message.type === 'message_sent') {
          const msg = message.payload;
          
          if (!msg || !msg.chat_id) {
            return;
          }
          
          // Update last message preview for the chat we just sent to (use ref)
          const existingChat = chatsRef.current.find(c => c.id === msg.chat_id);
          
          if (existingChat) {
            try {
              // Decrypt the message content
              const companionPublicKey = await getUserPublicKey(existingChat.companion_id);
              const companionPubKeyBytes = fromHex(companionPublicKey);
              const chatKey = deriveChatKey(privateKey, companionPubKeyBytes);
              
              const encryptedData = JSON.parse(msg.encrypted_content);
              const decrypted = await decryptMessage(
                encryptedData.ciphertext,
                encryptedData.nonce,
                chatKey
              );
              
              const preview = decrypted.length > 50 ? decrypted.substring(0, 50) + '...' : decrypted;
              
              console.log('[ChatsPage] message_sent msg.created_at:', msg.created_at, 'type:', typeof msg.created_at);
              
              // Use current time if created_at is invalid
              const messageTime = msg.created_at || new Date().toISOString();
              
              // Update chats state with new last message preview
              setChats(prevChats => {
                const updated = prevChats.map(chat =>
                  chat.id === msg.chat_id
                    ? { ...chat, last_message_preview: preview, last_message_at: messageTime }
                    : chat
                ).sort((a, b) => new Date(b.last_message_at).getTime() - new Date(a.last_message_at).getTime());
                
                return updated;
              });
            } catch (error) {
              console.error('[ChatsPage] Failed to decrypt sent message:', error);
            }
          }
        }
      });

      ws.connect();
      
      // Wait for connection and then load chats
      try {
        await ws.waitForConnection();
        await loadChats();
      } catch (error) {
        console.error('Failed to initialize connection:', error);
      }

      return () => {
        unsubscribeStatus();
        unsubscribeMessage();
      };
    };

    let cleanup: (() => void) | undefined;
    initializeConnection().then(fn => { cleanup = fn; });

    return () => {
      cleanup?.();
    };
  }, [navigate]);

  // Periodically check online status for all chats
  useEffect(() => {
    const checkAllOnlineStatuses = async () => {
      if (chatsRef.current.length === 0) return;
      
      try {
        const updatedChats = await Promise.all(
          chatsRef.current.map(async (chat) => {
            try {
              const isOnline = await checkOnlineStatus(chat.companion_id);
              return { ...chat, is_online: isOnline };
            } catch (error) {
              return chat; // Keep existing status on error
            }
          })
        );
        setChats(updatedChats);
      } catch (error) {
        console.error('[ChatsPage] Failed to check online statuses:', error);
      }
    };

    // Check immediately and then every 2 seconds
    const interval = setInterval(checkAllOnlineStatuses, 2000);

    return () => clearInterval(interval);
  }, [chats.length]); // Only depend on length to avoid infinite loop

  // Memoize selectedChat to keep reference stable when only last_message_preview changes
  const selectedChat = useMemo(() => {
    if (!selectedChatId) return null;
    return chats.find(c => c.id === selectedChatId) || null;
  }, [selectedChatId, chats]);

  const loadChats = async () => {
    try {
      setLoading(true);
      const chatsData = await getChats();
      console.log('[ChatsPage] Loaded chats:', chatsData);
      console.log('[ChatsPage] First chat full object:', JSON.stringify(chatsData[0], null, 2));
      
      // Check for chats without encrypted_key and generate them
      // Also fetch companion logins and decrypt last message
      const privateKey = await restorePrivateKey();
      if (privateKey) {
        const updatedChats = await Promise.all(
          chatsData.map(async (chat) => {
            let updatedChat = { ...chat };
            
            // If chat has no encrypted_key, generate it
            if (!chat.encrypted_key || chat.encrypted_key === '') {
              try {
                console.log('[ChatsPage] Generating key for chat:', chat.companion_id);
                
                // Get companion's public key
                const companionPublicKey = await getUserPublicKey(chat.companion_id);
                const companionPubKeyBytes = fromHex(companionPublicKey);
                
                // Derive chat key using ECDH
                const chatKey = deriveChatKey(privateKey, companionPubKeyBytes);
                
                // Encrypt chat key with private key for storage
                const { ciphertext } = await encryptChatKeyWithPrivateKey(chatKey, privateKey);
                
                // Save to server
                await saveChatKeys({
                  user_id: chat.companion_id,
                  companion_id: chat.companion_id,
                  encrypted_key: ciphertext,
                  key_iv: '',
                });
                
                updatedChat.encrypted_key = ciphertext;
              } catch (error) {
                console.error('[ChatsPage] Failed to generate key for chat:', error);
              }
            }
            
            // Decrypt last message if available
            console.log('[ChatsPage] Processing last_message for chat:', chat.companion_id, 'last_message:', chat.last_message);
            if (chat.last_message && chat.encrypted_key) {
              try {
                const companionPublicKey = await getUserPublicKey(chat.companion_id);
                const companionPubKeyBytes = fromHex(companionPublicKey);
                const chatKey = deriveChatKey(privateKey, companionPubKeyBytes);
                
                // Parse encrypted message (format: {"ciphertext":"...","nonce":"..."})
                console.log('[ChatsPage] Parsing last_message:', chat.last_message);
                const encryptedData = JSON.parse(chat.last_message);
                const decrypted = await decryptMessage(
                  encryptedData.ciphertext,
                  encryptedData.nonce,
                  chatKey
                );
                console.log('[ChatsPage] Decrypted last message:', decrypted);
                updatedChat.last_message_preview = decrypted.length > 50
                  ? decrypted.substring(0, 50) + '...'
                  : decrypted;
              } catch (error) {
                console.error('[ChatsPage] Failed to decrypt last message:', error);
                updatedChat.last_message_preview = 'Message...';
              }
            }
            
            // Fetch companion login from auth-service
            try {
              console.log(`[ChatsPage] Fetching login for companion_id: ${chat.companion_id}`);
              const response = await fetch(`http://localhost/api/auth/search?id=${chat.companion_id}`, {
                credentials: 'include', // Include cookies for JWT
              });
              const data = await response.json();
              console.log(`[ChatsPage] Search response for ${chat.companion_id}:`, data);
              if (data.users && data.users.length > 0) {
                updatedChat.companion_login = data.users[0].login;
                console.log(`[ChatsPage] Set companion_login to: ${updatedChat.companion_login}`);
              } else {
                console.warn(`[ChatsPage] No user found for companion_id: ${chat.companion_id}`);
              }
            } catch (error) {
              console.error('[ChatsPage] Failed to fetch companion login:', error);
            }
            
            // Check online status
            try {
              const isOnline = await checkOnlineStatus(chat.companion_id);
              updatedChat.is_online = isOnline;
            } catch (error) {
              console.error('[ChatsPage] Failed to check online status:', error);
              updatedChat.is_online = false;
            }
            
            // Calculate unread count
            try {
              const userId = localStorage.getItem('user_id');
              if (userId) {
                const messagesResponse = await getMessages({ chat_id: chat.id, limit: 100 });
                const unreadCount = messagesResponse.messages.filter(
                  msg => msg.status !== 'read' && msg.sender_id !== userId
                ).length;
                updatedChat.unread_count = unreadCount;
              }
            } catch (error) {
              console.error('[ChatsPage] Failed to calculate unread count:', error);
              updatedChat.unread_count = 0;
            }
            
            return updatedChat;
          })
        );
        setChats(updatedChats);
      } else {
        setChats(chatsData);
      }
    } catch (error) {
      console.error('Failed to load chats:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleSelectChat = (chat: Chat) => {
    setSelectedChatId(chat.id);
    
    // Reset unread count for this chat
    setChats(prevChats =>
      prevChats.map(c =>
        c.id === chat.id ? { ...c, unread_count: 0 } : c
      )
    );
  };

  const handleNewChat = async (user: User) => {
    try {
      // Check if chat already exists
      const existingChat = chats.find(c => c.companion_id === user.id);
      if (existingChat) {
        setSelectedChatId(existingChat.id);
        setShowSearch(false);
        return;
      }

      // Get private key
      const privateKey = await restorePrivateKey();
      if (!privateKey) {
        navigate('/login');
        return;
      }

      // Derive chat key using ECDH
      const companionPublicKey = fromHex(user.public_key);
      const chatKey = deriveChatKey(privateKey, companionPublicKey);

      // Encrypt chat key with private key for storage
      const { ciphertext } = await encryptChatKeyWithPrivateKey(chatKey, privateKey);

      // Save chat keys to server
      await saveChatKeys({
        user_id: user.id, // This will be ignored by server, it uses JWT
        companion_id: user.id,
        encrypted_key: ciphertext,
        key_iv: '', // Not used with current encryption scheme
      });

      // Create new chat object with temporary ID
      const tempChatId = `temp-${user.id}`;
      const newChat: Chat = {
        id: tempChatId, // Temporary ID until first message is sent
        user_id: '', // Current user
        companion_id: user.id,
        companion_login: user.login,
        created_at: new Date().toISOString(),
        last_message_at: new Date().toISOString(),
        encrypted_key: ciphertext,
        key_iv: '',
      };

      // Add to chats list and select
      setChats([newChat, ...chats]);
      setSelectedChatId(tempChatId);
      setShowSearch(false);
    } catch (error) {
      console.error('Failed to create chat:', error);
      alert('Failed to create chat. Please try again.');
    }
  };

  return (
    <div className="chats-page">
      <div className="chats-sidebar">
        <div className="sidebar-header">
          <h2>Chats</h2>
          <div className="header-actions">
            <button
              className="btn-icon"
              onClick={() => setShowSearch(true)}
              title="Search"
            >
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                <circle cx="11" cy="11" r="8"></circle>
                <path d="m21 21-4.35-4.35"></path>
              </svg>
            </button>
            <button
              className="btn-icon"
              onClick={() => navigate('/change-password')}
              title="Settings"
            >
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"></path>
                <circle cx="12" cy="12" r="3"></circle>
              </svg>
            </button>
          </div>
        </div>

        <div className="connection-status">
          {wsConnected ? (
            <span className="status-connected">● Connected</span>
          ) : (
            <span className="status-disconnected">○ Connecting...</span>
          )}
        </div>

        {loading ? (
          <div className="loading">Loading chats...</div>
        ) : (
          <ChatList
            chats={chats}
            selectedChat={selectedChat}
            onSelectChat={handleSelectChat}
          />
        )}
      </div>

      <div className="chats-main">
        {selectedChat ? (
          <ChatWindow
            key={`${selectedChat.id}-${chatUpdateTrigger}`}
            chat={selectedChat}
            onChatUpdate={loadChats}
          />
        ) : (
          <div className="no-chat-selected">
            <div className="empty-state">
              <h3>Select a chat to start messaging</h3>
              <p>Choose from your existing conversations or start a new one</p>
              <button
                className="btn-primary"
                onClick={() => setShowSearch(true)}
              >
                Start New Chat
              </button>
            </div>
          </div>
        )}
      </div>

      {showSearch && (
        <SearchModal
          onClose={() => setShowSearch(false)}
          onSelectUser={handleNewChat}
        />
      )}
    </div>
  );
}
