package service

import (
	"context"

	"api-gateway/internal/cores/domain"
)

type AuthClient interface {
	ValidateToken(token string) (userID string, err error)
	ChangePassword(ctx context.Context, login, oldHash, newHash, newPublicKey string) (string, error)
}

type MessageClient interface {
	SendMessage(ctx context.Context, chatID, senderID, recipientID, content, messageType string) (*domain.Message, error)
	GetMessages(ctx context.Context, chatID, userID, lastMessageID string, limit int32) (*domain.GetMessagesResponse, error)
	MarkAsRead(ctx context.Context, chatID, userID, lastMessageID string) error
	DeleteMessage(ctx context.Context, messageID, userID string) error
	AlterMessage(ctx context.Context, messageID, userID, newContent string) error
	GetChats(ctx context.Context, userID string) (*domain.GetChatsResponse, error)
	SaveChatKeys(ctx context.Context, userID, companionID, encryptedKey, keyIV string) error
	UpdateChatKeys(ctx context.Context, userID string, keys []domain.ChatKeyUpdate) (int, error)
}

type RealtimeClient interface {
	Connect(ctx context.Context, userID string, send chan<- []byte) error
}

type GatewayService interface {
	HandleWebSocket(ctx context.Context, userID string, send chan<- []byte, recv <-chan []byte) error
}

type Orchestrator interface {
	ChangePassword(ctx context.Context, req *domain.ChangePasswordRequest) (*domain.ChangePasswordResponse, error)
}