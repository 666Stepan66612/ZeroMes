package service

import "context"

type MessageService interface {
	SendMessage(ctx context.Context, chatID, senderID, recipientID, content, msgType string) (*Message, error)
	GetMessages(ctx context.Context, chatID string, limit int, lastMessageID string) ([]*Message, error)
	MarkAsRead(ctx context.Context, chatID, userID, lastMessageID string) error
	AlterMessage(ctx context.Context, messageID, userID, newContent string) error
	DeleteMessage(ctx context.Context, messageID, userID string) error
	GetChats(ctx context.Context, userID string) ([]*ChatsList, error)
}

type MessageRepository interface {
	CreateWithChats(ctx context.Context, msg *Message) error
	GetByChatID(ctx context.Context, chatID string, limit int, lastMessageID string) ([]*Message, error)
	GetByID(ctx context.Context, messageID string) (*Message, error)
	Delete(ctx context.Context, messageID string) error
	Alter(ctx context.Context, messageID, newContent string) error
	UpdateStatusBatch(ctx context.Context, chatID, userID, lastMessageID string, status MessageStatus) error
	GetChats(ctx context.Context, userID string) ([]*ChatsList, error)
}

type KafkaProducer interface {
	PublishMessageSent(ctx context.Context, msg *Message) error
}