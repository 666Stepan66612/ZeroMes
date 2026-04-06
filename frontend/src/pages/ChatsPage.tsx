import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { getChats, logout, saveChatKeys, getUserPublicKey } from '@/lib/api';
import { getWebSocketClient } from '@/lib/api/websocket';
import { restorePrivateKey, clearKeys, fromHex } from '@/lib/crypto';
import { deriveChatKey, encryptChatKeyWithPrivateKey } from '@/lib/crypto/encryption';
import { ChatList, ChatWindow, SearchModal } from '@/components';
import type { Chat, User } from '@/types/api';
import './ChatsPage.css';

export function ChatsPage() {
  const navigate = useNavigate();
  const [chats, setChats] = useState<Chat[]>([]);
  const [selectedChat, setSelectedChat] = useState<Chat | null>(null);
  const [showSearch, setShowSearch] = useState(false);
  const [loading, setLoading] = useState(true);
  const [wsConnected, setWsConnected] = useState(false);
  const [chatUpdateTrigger, setChatUpdateTrigger] = useState(0);

  useEffect(() => {
    // Check if user is logged in
    const privateKey = restorePrivateKey();
    if (!privateKey) {
      navigate('/login');
      return;
    }

    // Connect to WebSocket and load chats
    const initializeConnection = async () => {
      const ws = getWebSocketClient();
      
      const unsubscribeStatus = ws.onStatus((status) => {
        setWsConnected(status === 'connected');
      });

      const unsubscribeMessage = ws.onMessage(async (message: any) => {
        console.log('New message:', message);
        
        // Handle new message notification
        if (message.type === 'new_message') {
          const msg = message.payload;
          console.log('[ChatsPage] new_message payload:', msg);
          
          if (!msg || !msg.chat_id) {
            console.warn('[ChatsPage] Invalid message payload:', msg);
            return;
          }
          
          // Check if we have this chat
          const existingChat = chats.find(c => c.id === msg.chat_id);
          
          if (!existingChat) {
            // New chat - need to generate keys
            console.log('[ChatsPage] New chat detected, generating keys for:', msg.chat_id);
            
            try {
              // The companion is the sender of the message
              const companionId = msg.sender_id;
              
              console.log('[ChatsPage] Companion ID (sender):', companionId);
              
              // Get companion's public key
              const companionPublicKey = await getUserPublicKey(companionId);
              const companionPubKeyBytes = fromHex(companionPublicKey);
              
              // Derive chat key using ECDH
              const chatKey = deriveChatKey(privateKey, companionPubKeyBytes);
              
              // Encrypt chat key with private key for storage
              const { ciphertext } = await encryptChatKeyWithPrivateKey(chatKey, privateKey);
              
              // Save to server
              await saveChatKeys({
                user_id: companionId,
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
            // Existing chat - trigger reload if this chat is currently open
            console.log('[ChatsPage] Message for existing chat:', msg.chat_id);
            if (selectedChat && selectedChat.id === msg.chat_id) {
              // Trigger chat update to reload messages
              setChatUpdateTrigger(prev => prev + 1);
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

  const loadChats = async () => {
    try {
      setLoading(true);
      const chatsData = await getChats();
      
      // Check for chats without encrypted_key and generate them
      // Also fetch companion logins
      const privateKey = restorePrivateKey();
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

  const handleLogout = async () => {
    try {
      await logout();
      clearKeys();
      navigate('/login');
    } catch (error) {
      console.error('Logout error:', error);
      // Force logout anyway
      clearKeys();
      navigate('/login');
    }
  };

  const handleSelectChat = (chat: Chat) => {
    setSelectedChat(chat);
  };

  const handleNewChat = async (user: User) => {
    try {
      // Check if chat already exists
      const existingChat = chats.find(c => c.companion_id === user.id);
      if (existingChat) {
        setSelectedChat(existingChat);
        setShowSearch(false);
        return;
      }

      // Get private key
      const privateKey = restorePrivateKey();
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

      // Create new chat object
      const newChat: Chat = {
        id: '', // Will be assigned by server when first message is sent
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
      setSelectedChat(newChat);
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
              title="New chat"
            >
              ✏️
            </button>
            <button
              className="btn-icon"
              onClick={() => navigate('/change-password')}
              title="Settings"
            >
              ⚙️
            </button>
            <button
              className="btn-icon"
              onClick={handleLogout}
              title="Logout"
            >
              🚪
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
