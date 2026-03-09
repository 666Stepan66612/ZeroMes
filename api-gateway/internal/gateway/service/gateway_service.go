package service

import (
	"context"
	"encoding/json"

	"api-gateway/internal/cores/domain"
)

type gatewayService struct {
	messageClient MessageClient
	realtimeClient RealtimeClient
}

func NewGatewayService(messageClient MessageClient, realtimeClient RealtimeClient) GatewayService {
	return &gatewayService{
		messageClient: messageClient,
		realtimeClient: realtimeClient,
	}
}

func (s *gatewayService) HandleWebSocket(ctx context.Context, userID string, send chan<- []byte, recv <-chan []byte) error {
	if err := s.realtimeClient.Connect(ctx, userID, send); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case data, ok := <-recv:
			if !ok {
				return nil
			}
			var req domain.WSRequest
			if err := json.Unmarshal(data, &req); err != nil {
				continue
			}
			switch req.Type {
			case "send_message":
				s.messageClient.SendMessage(ctx, req.ChatID, userID, req.RecipientID, req.Content, req.MessageType)
			case "get_messages":
				s.messageClient.GetMessages(ctx, req.ChatID, userID, req.LastMessageID, int32(req.Limit))
			case "mark_as_read":
				s.messageClient.MarkAsRead(ctx, req.ChatID, userID, req.LastMessageID)
			case "delete_message":
				s.messageClient.DeleteMessage(ctx, req.MessageID, userID)
			case "alter_message":
				s.messageClient.AlterMessage(ctx, req.MessageID, userID, req.NewContent)
			}
		}
	}
}