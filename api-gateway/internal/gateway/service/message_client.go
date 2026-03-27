package service

import (
	"api-gateway/internal/cores/domain"
	"context"
	"log/slog"

	messagepb "github.com/666Stepan66612/ZeroMes/pkg/gen/messagepb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type MessageClientService struct {
	conn   *grpc.ClientConn
	client messagepb.MessageServiceClient
}

func NewMessageClient(addr string) (*MessageClientService, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &MessageClientService{
		conn:   conn,
		client: messagepb.NewMessageServiceClient(conn),
	}, nil
}

func (c *MessageClientService) Close() error {
	return c.conn.Close()
}

func (c *MessageClientService) SendMessage(ctx context.Context, chatID, senderID, recipientID, encryptedContent, messageType string) (*domain.Message, error) {
	slog.Info("calling message-service SendMessage", "chat_id", chatID, "sender_id", senderID, "recipient_id", recipientID)
	resp, err := c.client.SendMessage(ctx, &messagepb.SendMessageRequest{
		ChatId:           chatID,
		SenderId:         senderID,
		RecipientId:      recipientID,
		EncryptedContent: encryptedContent,
		MessageType:      messageType,
	})
	if err != nil {
		slog.Error("message-service SendMessage failed", "err", err)
		return nil, err
	}
	slog.Info("message-service SendMessage success", "message_id", resp.Message.Id)
	return &domain.Message{
		ID:        resp.Message.Id,
		ChatID:    resp.Message.ChatId,
		SenderID:  resp.Message.SenderId,
		Content:   resp.Message.EncryptedContent,
		CreatedAt: resp.Message.CreatedAt.AsTime().String(),
	}, nil
}

func (c *MessageClientService) GetMessages(ctx context.Context, chatID, userID, lastMessageID string, limit int32) (*domain.GetMessagesResponse, error) {
	resp, err := c.client.GetMessages(ctx, &messagepb.GetMessagesRequest{
		ChatId:        chatID,
		UserId:        userID,
		Limit:         limit,
		LastMessageId: lastMessageID,
	})
	if err != nil {
		return nil, err
	}

	messages := make([]*domain.Message, len(resp.Messages))
	for i, m := range resp.Messages {
		messages[i] = &domain.Message{
			ID:        m.Id,
			ChatID:    m.ChatId,
			SenderID:  m.SenderId,
			Content:   m.EncryptedContent,
			CreatedAt: m.CreatedAt.AsTime().String(),
			Status:    int32(m.Status),
		}
	}

	return &domain.GetMessagesResponse{
		Messages:      messages,
		NextMessageId: resp.NextMessageId,
		HasMore:       resp.HasMore,
	}, nil
}

func (c *MessageClientService) MarkAsRead(ctx context.Context, chatID, userID, lastMessageID string) error {
	_, err := c.client.MarkAsRead(ctx, &messagepb.MarkAsReadRequest{
		ChatId:        chatID,
		UserId:        userID,
		LastMessageId: lastMessageID,
	})
	return err
}

func (c *MessageClientService) DeleteMessage(ctx context.Context, messageID, userID string) error {
	_, err := c.client.DeleteMessage(ctx, &messagepb.DeleteMessageRequest{
		MessageId: messageID,
		UserId:    userID,
	})
	return err
}

func (c *MessageClientService) AlterMessage(ctx context.Context, messageID, userID, newContent string) error {
	_, err := c.client.AlterMessage(ctx, &messagepb.AlterMessageRequest{
		MessageId:  messageID,
		UserId:     userID,
		NewContent: newContent,
	})
	return err
}

func (c *MessageClientService) GetChats(ctx context.Context, userID string) (*domain.GetChatsResponse, error) {
	resp, err := c.client.GetChats(ctx, &messagepb.GetChatsRequest{
		UserId: userID,
	})
	if err != nil {
		return nil, err
	}

	chats := make([]*domain.Chat, len(resp.Chats))
	for i, ch := range resp.Chats {
		chats[i] = &domain.Chat{
			ID:            ch.Id,
			CompanionID:   ch.CompanionId,
			LastMessageAt: ch.LastMessageAt.AsTime().String(),
			EncryptedKey:  ch.EncryptedKey,
        	KeyIV:         ch.KeyIv,   
		}
	}

	return &domain.GetChatsResponse{Chats: chats}, nil
}

func (c *MessageClientService) SaveChatKeys(ctx context.Context, userID, companionID, encryptedKey, keyIV string) error {
    _, err := c.client.SaveChatKeys(ctx, &messagepb.SaveChatKeysRequest{
        UserId:       userID,
        CompanionId:  companionID,
        EncryptedKey: encryptedKey,
        KeyIv:        keyIV,
    })
    return err
}