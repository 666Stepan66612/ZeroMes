package service

import (
	"context"

	messagepb "github.com/666Stepan66612/ZeroMes/pkg/gen/messagepb"
	"google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

type MessageClientService struct {
	client messagepb.MessageServiceClient
}

func NewMessageClient(addr string) (*MessageClientService, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &MessageClientService{
		client: messagepb.NewMessageServiceClient(conn),
	}, nil
}

func (c *MessageClientService) SendMessage(ctx context.Context, chatID, senderID, recipientID, encryptedContent, messageType string) (*messagepb.Message, error) {
	resp, err := c.client.SendMessage(ctx, &messagepb.SendMessageRequest{
		ChatId: chatID,
		SenderId: senderID,
		RecipientId: recipientID,
		EncryptedContent: encryptedContent,
		MessageType: messageType,
	})
	if err != nil {
		return nil, err
	}
	return resp.Message, nil
}

func (c *MessageClientService) GetMessages(ctx context.Context, chatID, userID string, limit int32, lastMessageID string) (*messagepb.GetMessagesResponse, error){
	return c.client.GetMessages(ctx, &messagepb.GetMessagesRequest{
		ChatId: chatID,
		UserId: userID,
		Limit: limit,
		LastMessageId: lastMessageID,
	})
}

func (c *MessageClientService) MarkAsRead(ctx context.Context, chatID, userID, lastMessageID string) error {
	_, err := c.client.MarkAsRead(ctx, &messagepb.MarkAsReadRequest{
		ChatId: chatID,
		UserId: userID,
		LastMessageId: lastMessageID,
	})
	return err
}

func (c *MessageClientService) DeleteMessage(ctx context.Context, messageID, userID string) error {
	_, err := c.client.DeleteMessage(ctx, &messagepb.DeleteMessageRequest{
		MessageId: messageID,
		UserId: userID,
	})
	return err
}

func (c *MessageClientService) AlterMessage(ctx context.Context, messageID, userID, newContent string) error {
	_, err := c.client.AlterMessage(ctx, &messagepb.AlterMessageRequest{
		MessageId: messageID,
		UserId: userID,
		NewContent: newContent,
	})
	return err
}