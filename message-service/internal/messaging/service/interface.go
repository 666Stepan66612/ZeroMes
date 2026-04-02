package service

import "context"

type MessageService interface {
	SendMessage(ctx context.Context, chatID, senderID, recipientID, content, msgType string) (*Message, error)
	GetMessages(ctx context.Context, chatID, userID string, limit int, lastMessageID string) ([]*Message, error)
	MarkAsRead(ctx context.Context, chatID, userID, lastMessageID string) error
	AlterMessage(ctx context.Context, messageID, userID, newContent string) error
	DeleteMessage(ctx context.Context, messageID, userID string) error
	GetChats(ctx context.Context, userID string) ([]*ChatsList, error)
	SaveChatKeys(ctx context.Context, userID, companionID, encryptedKey, keyIV string) error
	UpdateChatKeys(ctx context.Context, userID string, keys []ChatKeyUpdate) (int, error)
}

type MessageRepository interface {
	CreateWithChats(ctx context.Context, msg *Message) error
	GetByChatID(ctx context.Context, chatID string, limit int, lastMessageID string) ([]*Message, error)
	GetByID(ctx context.Context, messageID string) (*Message, error)
	Delete(ctx context.Context, messageID string) error
	Alter(ctx context.Context, messageID, newContent string) error
	UpdateStatusBatch(ctx context.Context, chatID, userID, lastMessageID string, status MessageStatus) error
	GetChats(ctx context.Context, userID string) ([]*ChatsList, error)
	SaveChatKeys(ctx context.Context, userID, companionID, encryptedKey, keyIV string) error
	UpdateChatKeys(ctx context.Context, userID string, keys []ChatKeyUpdate) (int, error)
}

type KafkaProducer interface {
	PublishMessageSent(ctx context.Context, msg *Message) error
	PublishMessageAltered(ctx context.Context, msg *Message, newContent string) error
    PublishMessageDeleted(ctx context.Context, msg *Message) error
    PublishMessageRead(ctx context.Context, chatID, readerID, senderID, lastMessageID string) error
    Close() error
}