import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { getChats, logout } from '@/lib/api';
import { getWebSocketClient } from '@/lib/api/websocket';
import { restorePrivateKey, clearKeys } from '@/lib/crypto';
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

  useEffect(() => {
    // Check if user is logged in
    const privateKey = restorePrivateKey();
    if (!privateKey) {
      navigate('/login');
      return;
    }

    // Load chats
    loadChats();

    // Connect to WebSocket
    const ws = getWebSocketClient();
    
    const unsubscribeStatus = ws.onStatus((status) => {
      setWsConnected(status === 'connected');
    });

    const unsubscribeMessage = ws.onMessage((message) => {
      // Handle incoming message
      console.log('New message:', message);
      // TODO: Decrypt and add to chat
    });

    ws.connect();

    return () => {
      unsubscribeStatus();
      unsubscribeMessage();
    };
  }, [navigate]);

  const loadChats = async () => {
    try {
      setLoading(true);
      const chatsData = await getChats();
      setChats(chatsData);
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

  const handleNewChat = (user: User) => {
    // Create or open chat with user
    const existingChat = chats.find(c => c.companion_id === user.id);
    if (existingChat) {
      setSelectedChat(existingChat);
    } else {
      // Create new chat (will be created when first message is sent)
      const newChat: Chat = {
        id: '', // Will be assigned by server
        user_id: '', // Current user
        companion_id: user.id,
        created_at: new Date().toISOString(),
        last_message_at: new Date().toISOString(),
        encrypted_key: '',
        key_iv: '',
      };
      setSelectedChat(newChat);
    }
    setShowSearch(false);
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
