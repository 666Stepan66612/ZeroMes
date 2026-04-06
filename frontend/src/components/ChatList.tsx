import type { Chat } from '@/types/api';

interface ChatListProps {
  chats: Chat[];
  selectedChat: Chat | null;
  onSelectChat: (chat: Chat) => void;
}

export function ChatList({ chats, selectedChat, onSelectChat }: ChatListProps) {
  if (chats.length === 0) {
    return (
      <div className="chat-list-empty">
        <p>No chats yet</p>
        <p className="help-text">Start a new conversation</p>
      </div>
    );
  }

  return (
    <div className="chat-list">
      {chats.map((chat) => (
        <div
          key={chat.id || chat.companion_id}
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
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
