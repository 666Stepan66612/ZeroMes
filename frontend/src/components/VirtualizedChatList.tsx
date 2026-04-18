import { useRef, useEffect } from 'react';
// @ts-ignore - react-window types are problematic with React 19
import { List as FixedSizeList } from 'react-window';
import type { Chat } from '@/types/api';

interface VirtualizedChatListProps {
  chats: Chat[];
  selectedChat: Chat | null;
  onSelectChat: (chat: Chat) => void;
  containerHeight: number;
}

interface ChatRowData {
  chats: Chat[];
  selectedChat: Chat | null;
  onSelectChat: (chat: Chat) => void;
}

// Компонент для отдельного чата
const ChatRow = ({ index, style, data }: { index: number; style: React.CSSProperties; data: ChatRowData }) => {
  const { chats, selectedChat, onSelectChat } = data;
  const chat = chats[index];
  
  if (!chat) return null;
  
  return (
    <div style={style}>
      <div
        className={`chat-item ${selectedChat?.id === chat.id ? 'active' : ''}`}
        onClick={() => onSelectChat(chat)}
      >
        <div className={`chat-avatar ${chat.is_online ? 'online' : 'offline'}`}>
          {(chat.companion_login || chat.companion_id).substring(0, 2).toUpperCase()}
        </div>
        <div className="chat-info">
          <div className="chat-header">
            <span className="chat-name">{chat.companion_login || chat.companion_id}</span>
            <span className="chat-time">
              {chat.last_message_at && !isNaN(new Date(chat.last_message_at).getTime())
                ? new Date(chat.last_message_at).toLocaleTimeString([], {
                    hour: '2-digit',
                    minute: '2-digit',
                  })
                : ''}
            </span>
          </div>
          <div className="chat-preview">
            <span className="chat-last-message">
              {chat.last_message_preview || 'No messages yet'}
            </span>
            {chat.unread_count !== undefined && chat.unread_count > 0 && (
              <span className="unread-badge">{chat.unread_count}</span>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export function VirtualizedChatList({
  chats,
  selectedChat,
  onSelectChat,
  containerHeight,
}: VirtualizedChatListProps) {
  const listRef = useRef<any>(null);
  
  // Прокрутка к выбранному чату
  useEffect(() => {
    if (listRef.current && selectedChat) {
      const index = chats.findIndex(c => c.id === selectedChat.id);
      if (index !== -1) {
        listRef.current.scrollToItem(index, 'smart');
      }
    }
  }, [selectedChat, chats]);
  
  if (chats.length === 0) {
    return (
      <div className="chat-list-empty">
        <p>No chats yet</p>
        <p className="help-text">Start a new conversation</p>
      </div>
    );
  }
  
  return (
    <FixedSizeList
      ref={listRef}
      height={containerHeight}
      itemCount={chats.length}
      itemSize={88}
      width="100%"
      itemData={{ chats, selectedChat, onSelectChat }}
      overscanCount={3}
      // @ts-ignore - react-window types issue
      children={ChatRow}
    />
  );
}
