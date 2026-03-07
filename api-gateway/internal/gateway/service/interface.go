package service

import "context"

type AuthClient interface {
	ValidateToken(ctx context.Context, token string) (userID string, err error)
}

type MessageClient interface {
	SendMessage(ctx context.Context, chatID, senderID, recipientID, content, messageType string) (interface{}, error)
	GetMessages(ctx context.Context, chatID, userID, lastMessageID string, limit int) (interface{}, error)
	MarkAsRead(ctx context.Context, chatID, userID, lastMessageID string) error
	DeleteMessage(ctx context.Context, messageID, userID string) error
	AlterMessage(ctx context.Context, messageID, userID, newContent string) error
}

type RealtimeClient interface {
	Connect(ctx context.Context, userID string, send chan<- []byte) error
}

type GatewayService interface {
	HandleWebSocket(ctx context.Context, userID string, send chan<- []byte, recv <-chan []byte) error
}