import { useRef, useEffect } from 'react';
import { Virtuoso } from 'react-virtuoso';
import type { Chat } from '@/types/api';
import { MessageStatus } from '@/types/api';

interface DecryptedMessage {
  id: string;
  chat_id: string;
  sender_id: string;
  recipient_id: string;
  encrypted_content: string;
  message_type: 'text' | 'image' | 'file';
  created_at: string;
  status: any;
  decryptedContent?: string;
  localStatus?: 'pending' | 'sent' | 'delivered' | 'read';
}

interface VirtualizedMessageListProps {
  messages: DecryptedMessage[];
  chat: Chat;
  onContextMenu: (e: React.MouseEvent, message: DecryptedMessage) => void;
  containerHeight: number;
}

export function VirtualizedMessageList({
  messages,
  chat,
  onContextMenu,
  containerHeight,
}: VirtualizedMessageListProps) {
  const listRef = useRef<any>(null);

  // Прокрутка вниз при добавлении новых сообщений
  useEffect(() => {
    if (listRef.current && messages.length > 0) {
      listRef.current.scrollToIndex({ index: messages.length - 1, behavior: 'smooth' });
    }
  }, [messages.length]);

  if (messages.length === 0) {
    return (
      <div className="no-messages">
        <p>No messages yet</p>
        <p className="help-text">Send a message to start the conversation</p>
      </div>
    );
  }

  return (
    <Virtuoso
      ref={listRef}
      style={{ height: containerHeight }}
      data={messages}
      itemContent={(_index, message) => {
        const isSent = message.sender_id !== chat.companion_id;
        const displayStatus = message.localStatus || message.status;

        // Normalize status to string for consistent handling
        const getStatusIcon = () => {
          if (displayStatus === 'pending') return ' pending';

          const status = typeof displayStatus === 'number' ? displayStatus : displayStatus;

          if (status === 'sent' || status === MessageStatus.SENT) {
            return ' ✓';
          }
          if (status === 'delivered' || status === MessageStatus.DELIVERED) {
            return ' ✓';
          }
          if (status === 'read' || status === MessageStatus.READ) {
            return ' ✓✓';
          }

          return '';
        };

        return (
          <div
            className={`message ${isSent ? 'sent' : 'received'}`}
            onContextMenu={(e) => onContextMenu(e, message)}
            style={{ margin: '4px 0' }}
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
                <span className="message-status" title={`Status: ${displayStatus}`}>
                  {getStatusIcon()}
                </span>
              )}
            </div>
          </div>
        );
      }}
    />
  );
}
