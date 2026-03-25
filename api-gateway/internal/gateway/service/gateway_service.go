package service

import (
	"context"
	"encoding/json"
	"log/slog"

	"api-gateway/internal/cores/domain"
)

type gatewayService struct {
	messageClient  MessageClient
	realtimeClient RealtimeClient
}

func NewGatewayService(messageClient MessageClient, realtimeClient RealtimeClient) GatewayService {
	return &gatewayService{
		messageClient:  messageClient,
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
				slog.Debug("invalid websocket request", "user_id", userID, "err", err)
				sendResponse(send, "error", map[string]string{"error": "invalid request"})
				continue
			}
			switch req.Type {
			case "send_message":
				slog.Info("send_message request", "user_id", userID, "recipient_id", req.RecipientID)
				result, err := s.messageClient.SendMessage(ctx, req.ChatID, userID, req.RecipientID, req.Content, req.MessageType)
				if err != nil {
					slog.Warn("send_message failed", "user_id", userID, "err", err)
					sendResponse(send, "error", map[string]string{"error": "failed to send message"})
					continue
				}
				sendResponse(send, "message_sent", result)

			case "get_messages":
				result, err := s.messageClient.GetMessages(ctx, req.ChatID, userID, req.LastMessageID, int32(req.Limit))
				if err != nil {
					slog.Warn("get_messages failed", "user_id", userID, "err", err)
					sendResponse(send, "error", map[string]string{"error": "failed to fetch messages"})
					continue
				}
				sendResponse(send, "messages", result)

			case "mark_as_read":
				err := s.messageClient.MarkAsRead(ctx, req.ChatID, userID, req.LastMessageID)
				if err != nil {
					slog.Warn("mark_as_read failed", "user_id", userID, "err", err)
					sendResponse(send, "error", map[string]string{"error": "failed to mark as read"})
					continue
				}
				sendResponse(send, "marked_as_read", nil)

			case "delete_message":
				err := s.messageClient.DeleteMessage(ctx, req.MessageID, userID)
				if err != nil {
					slog.Warn("delete_message failed", "user_id", userID, "err", err)
					sendResponse(send, "error", map[string]string{"error": "failed to delete message"})
					continue
				}
				sendResponse(send, "message_deleted", nil)

			case "alter_message":
				err := s.messageClient.AlterMessage(ctx, req.MessageID, userID, req.NewContent)
				if err != nil {
					slog.Warn("alter_message failed", "user_id", userID, "err", err)
					sendResponse(send, "error", map[string]string{"error": "failed to update message"})
					continue
				}
				sendResponse(send, "message_altered", nil)

			case "get_chats":
				result, err := s.messageClient.GetChats(ctx, userID)
				if err != nil {
					slog.Warn("get_chats failed", "user_id", userID, "err", err)
					sendResponse(send, "error", map[string]string{"error": "failed to fetch chats"})
					continue
				} else {
					sendResponse(send, "chats", result)
					continue
				}

			default:
				slog.Warn("unknown websocket command", "user_id", userID, "type", req.Type)
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
		slog.Error("failed to marshal websocket response", "err", err)
		return
	}
	send <- data
}
