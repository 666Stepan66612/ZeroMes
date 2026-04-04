import { useState, useEffect, useRef } from 'react';
import type { FormEvent } from 'react';
import type { Chat, Message } from '@/types/api';
import { getMessages, sendMessage as sendMessageAPI } from '@/lib/api/messages';

interface ChatWindowProps {
  chat: Chat;
  onChatUpdate: () => void;
}

export function ChatWindow({ chat, onChatUpdate }: ChatWindowProps) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [messageText, setMessageText] = useState('');
  const [loading, setLoading] = useState(false);
  const [sending, setSending] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (chat.id) {
      loadMessages();
    }
  }, [chat.id]);

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const loadMessages = async () => {
    if (!chat.id) return;

    try {
      setLoading(true);
      const response = await getMessages({
        chat_id: chat.id,
        limit: 50,
      });
      setMessages(response.messages);
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

    if (!messageText.trim() || sending) return;

    try {
      setSending(true);

      // TODO: Encrypt message before sending
      const encryptedContent = messageText; // Placeholder

      await sendMessageAPI({
        chat_id: chat.id,
        recipient_id: chat.companion_id,
        encrypted_content: encryptedContent,
        message_type: 'text',
      });

      setMessageText('');
      await loadMessages();
      onChatUpdate();
    } catch (error) {
      console.error('Failed to send message:', error);
    } finally {
      setSending(false);
    }
  };

  return (
    <div className="chat-window">
      <div className="chat-window-header">
        <div className="chat-avatar">
          {chat.companion_id.substring(0, 2).toUpperCase()}
        </div>
        <div className="chat-header-info">
          <h3>{chat.companion_id}</h3>
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
              className={`message ${message.sender_id === chat.user_id ? 'sent' : 'received'}`}
            >
              <div className="message-content">
                {message.encrypted_content}
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

      <form className="chat-input" onSubmit={handleSendMessage}>
        <input
          type="text"
          value={messageText}
          onChange={(e) => setMessageText(e.target.value)}
          placeholder="Type a message..."
          disabled={sending}
        />
        <button
          type="submit"
          className="btn-send"
          disabled={!messageText.trim() || sending}
        >
          {sending ? '⏳' : '📤'}
        </button>
      </form>
    </div>
  );
}
