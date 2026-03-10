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
                sendResponse(send, "error", map[string]string{"error": "invalid request"})
                continue
            }
            switch req.Type {
            case "send_message":
                result, err := s.messageClient.SendMessage(ctx, req.ChatID, userID, req.RecipientID, req.Content, req.MessageType)
                if err != nil {
                    sendResponse(send, "error", map[string]string{"error": err.Error()})
                    continue
                }
                sendResponse(send, "message_sent", result)

            case "get_messages":
                result, err := s.messageClient.GetMessages(ctx, req.ChatID, userID, req.LastMessageID, int32(req.Limit))
                if err != nil {
                    sendResponse(send, "error", map[string]string{"error": err.Error()})
                    continue
                }
                sendResponse(send, "messages", result)

            case "mark_as_read":
                err := s.messageClient.MarkAsRead(ctx, req.ChatID, userID, req.LastMessageID)
                if err != nil {
                    sendResponse(send, "error", map[string]string{"error": err.Error()})
                    continue
                }
                sendResponse(send, "marked_as_read", nil)

            case "delete_message":
                err := s.messageClient.DeleteMessage(ctx, req.MessageID, userID)
                if err != nil {
                    sendResponse(send, "error", map[string]string{"error": err.Error()})
                    continue
                }
                sendResponse(send, "message_deleted", nil)

            case "alter_message":
                err := s.messageClient.AlterMessage(ctx, req.MessageID, userID, req.NewContent)
                if err != nil {
                    sendResponse(send, "error", map[string]string{"error": err.Error()})
                    continue
                }
                sendResponse(send, "message_altered", nil)

            default:
                sendResponse(send, "error", map[string]string{"error": "unknown type"})
            }
		}
	}
}

func sendResponse(send chan<- []byte, msgType string, payload interface{}) {
    resp := domain.WSResponse{
        Type:    msgType,
        Payload: payload,
    }
    data, err := json.Marshal(resp)
    if err != nil {
        return
    }
    send <- data
}