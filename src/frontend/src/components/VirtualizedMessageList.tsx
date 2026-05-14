import { useRef, useEffect, useState } from 'react';
import { Virtuoso, type VirtuosoHandle } from 'react-virtuoso';
import type { Chat } from '@/types/api';
import { MessageStatus } from '@/types/api';
import { formatTime, formatDateSeparator, getDateKey } from '@/utils/dateUtils';

interface DecryptedMessage {
  id: string;
  chat_id: string;
  sender_id: string;
  recipient_id: string;
  encrypted_content: string;
  message_type: 'text' | 'image' | 'file';
  created_at: string;
  status: string | number;
  decryptedContent?: string;
  localStatus?: 'pending' | 'sent' | 'delivered' | 'read';
}

interface VirtualizedMessageListProps {
  messages: DecryptedMessage[];
  chat: Chat;
  onContextMenu: (e: React.MouseEvent, message: DecryptedMessage) => void;
  containerHeight: number;
  onAtBottomChange?: (atBottom: boolean) => void;
  scrollToBottom?: boolean;
  onLoadMore?: () => void;
  hasMore?: boolean;
  loadingMore?: boolean;
}

export function VirtualizedMessageList({
  messages,
  chat,
  onContextMenu,
  containerHeight,
  onAtBottomChange,
  scrollToBottom,
  onLoadMore,
  hasMore,
  loadingMore,
}: VirtualizedMessageListProps) {
  const listRef = useRef<VirtuosoHandle>(null);
  const [isAtBottom, setIsAtBottom] = useState(true);
  const prevMessagesLength = useRef(messages.length);

  // Group messages by date for separators
  const messagesWithSeparators = messages.reduce((acc, message, index) => {
    const currentDateKey = getDateKey(message.created_at);
    const prevDateKey = index > 0 ? getDateKey(messages[index - 1].created_at) : null;

    // Add date separator if day changed
    if (currentDateKey !== prevDateKey) {
      acc.push({
        type: 'separator' as const,
        date: message.created_at,
        id: `separator-${currentDateKey}`,
      });
    }

    acc.push({
      type: 'message' as const,
      data: message,
      id: message.id,
    });

    return acc;
  }, [] as Array<{ type: 'separator' | 'message'; date?: string; data?: DecryptedMessage; id: string }>);

  // Auto-scroll to bottom when new messages arrive and user is at bottom
  useEffect(() => {
    if (messagesWithSeparators.length > prevMessagesLength.current && isAtBottom) {
      listRef.current?.scrollToIndex({ index: messagesWithSeparators.length - 1, behavior: 'smooth' });
    }
    prevMessagesLength.current = messagesWithSeparators.length;
  }, [messagesWithSeparators.length, isAtBottom]);

  // Handle external scroll to bottom request
  useEffect(() => {
    if (scrollToBottom && messagesWithSeparators.length > 0) {
      listRef.current?.scrollToIndex({ index: messagesWithSeparators.length - 1, behavior: 'smooth' });
    }
  }, [scrollToBottom, messagesWithSeparators.length]);

  // Track if user is at bottom
  const handleAtBottomStateChange = (atBottom: boolean) => {
    setIsAtBottom(atBottom);
    onAtBottomChange?.(atBottom);
  };

  // Handle loading more messages when scrolling to top
  const handleStartReached = () => {
    if (hasMore && !loadingMore && onLoadMore) {
      onLoadMore();
    }
  };

  if (messages.length === 0) {
    return (
      <div className="no-messages">
        <p>No messages yet</p>
        <p className="help-text">Send a message to start the conversation</p>
      </div>
    );
  }

  if (messagesWithSeparators.length === 0) {
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
      data={messagesWithSeparators}
      followOutput="smooth"
      atBottomStateChange={handleAtBottomStateChange}
      atBottomThreshold={200}
      startReached={handleStartReached}
      overscan={200}
      components={{
        Header: () => loadingMore ? (
          <div className="loading-more">
            Loading older messages...
          </div>
        ) : null,
      }}
      itemContent={(_index, item) => {
        // Render date separator
        if (item.type === 'separator') {
          return (
            <div className="date-separator">
              <span className="date-separator-text">
                {formatDateSeparator(item.date!)}
              </span>
            </div>
          );
        }

        // Render message
        const message = item.data!;
        const isSent = message.sender_id !== chat.companion_id;
        const displayStatus = message.localStatus || message.status;

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
          >
            <div className="message-content">
              {message.decryptedContent || message.encrypted_content}
            </div>
            <div className="message-time">
              {formatTime(message.created_at)}
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
