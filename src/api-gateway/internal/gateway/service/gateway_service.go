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
				slog.Debug("invalid websocket request", "user_id", userID, "err", err, "data", string(data))
				sendResponse(send, "error", map[string]string{"error": "invalid request"})
				continue
			}
			slog.Info("websocket message received", "user_id", userID, "type", req.Type)
			switch req.Type {
			case "ping":
				sendResponse(send, "pong", nil)

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
				slog.Info("get_messages request", "user_id", userID, "chat_id", req.ChatID, "limit", req.Limit, "last_message_id", req.LastMessageID)
				result, err := s.messageClient.GetMessages(ctx, req.ChatID, userID, req.LastMessageID, int32(req.Limit))
				if err != nil {
					slog.Warn("get_messages failed", "user_id", userID, "err", err)
					sendResponse(send, "error", map[string]string{"error": "failed to fetch messages"})
					continue
				}
				slog.Info("get_messages response", "user_id", userID, "messages_count", len(result.Messages), "has_more", result.HasMore)
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
				}
				slog.Info("get_chats success", "user_id", userID, "chats_count", len(result.Chats))
				if len(result.Chats) > 0 {
					slog.Info("first chat debug", "chat_id", result.Chats[0].ID, "last_message", result.Chats[0].LastMessage)
				}
				sendResponse(send, "chats", result)

			case "save_chat_keys":
				err := s.messageClient.SaveChatKeys(ctx, userID, req.CompanionID, req.EncryptedKey, req.KeyIV)
				if err != nil {
					slog.Warn("save_chat_keys failed", "user_id", userID, "err", err)
					sendResponse(send, "error", map[string]string{"error": "failed to save chat keys"})
					continue
				}
				sendResponse(send, "chat_keys_saved", nil)

			case "check_online_status":
				isOnline, err := s.realtimeClient.CheckOnlineStatus(ctx, req.UserID)
				if err != nil {
					slog.Warn("check_online_status failed", "user_id", userID, "target_user", req.UserID, "err", err)
					sendResponse(send, "error", map[string]string{"error": "failed to check online status"})
					continue
				}
				sendResponse(send, "online_status", map[string]interface{}{
					"user_id":   req.UserID,
					"is_online": isOnline,
				})

			case "create_group":
				distributions := make([]domain.SeedDistribution, len(req.SeedDistributions))
				for i, sd := range req.SeedDistributions {
					distributions[i] = domain.SeedDistribution(sd)
				}
				result, err := s.messageClient.CreateGroup(ctx, req.GroupName, userID, req.MemberIDs, distributions)
				if err != nil {
					slog.Warn("create_group failed", "user_id", userID, "err", err)
					sendResponse(send, "error", map[string]string{"error": "failed to create group"})
					continue
				}
				sendResponse(send, "group_created", result)

			case "add_group_member":
				err := s.messageClient.AddGroupMember(ctx, req.GroupID, req.UserID, userID, req.EncryptedSeed)
				if err != nil {
					slog.Warn("add_group_member failed", "user_id", userID, "err", err)
					sendResponse(send, "error", map[string]string{"error": "failed to add group member"})
					continue
				}
				sendResponse(send, "group_member_added", nil)

			case "remove_group_member":
				newVersion, err := s.messageClient.RemoveGroupMember(ctx, req.GroupID, req.UserID, userID)
				if err != nil {
					slog.Warn("remove_group_member failed", "user_id", userID, "err", err)
					sendResponse(send, "error", map[string]string{"error": "failed to remove group member"})
					continue
				}
				sendResponse(send, "group_member_removed", map[string]interface{}{
					"group_id":       req.GroupID,
					"user_id":        req.UserID,
					"new_key_version": newVersion,
				})

			case "leave_group":
				err := s.messageClient.LeaveGroup(ctx, req.GroupID, userID)
				if err != nil {
					slog.Warn("leave_group failed", "user_id", userID, "err", err)
					sendResponse(send, "error", map[string]string{"error": "failed to leave group"})
					continue
				}
				sendResponse(send, "left_group", nil)

			case "get_group_chats":
				result, err := s.messageClient.GetGroupChats(ctx, userID)
				if err != nil {
					slog.Warn("get_group_chats failed", "user_id", userID, "err", err)
					sendResponse(send, "error", map[string]string{"error": "failed to fetch group chats"})
					continue
				}
				sendResponse(send, "group_chats", result)

			case "get_group_members":
				result, err := s.messageClient.GetGroupMembers(ctx, req.GroupID)
				if err != nil {
					slog.Warn("get_group_members failed", "user_id", userID, "err", err)
					sendResponse(send, "error", map[string]string{"error": "failed to fetch group members"})
					continue
				}
				sendResponse(send, "group_members", result)

			case "save_group_key_seed":
				err := s.messageClient.SaveGroupKeySeed(ctx, userID, req.GroupID, req.EncryptedSeed, req.AddedBy, req.KeyVersion)
				if err != nil {
					slog.Warn("save_group_key_seed failed", "user_id", userID, "err", err)
					sendResponse(send, "error", map[string]string{"error": "failed to save group key seed"})
					continue
				}
				sendResponse(send, "group_key_seed_saved", nil)

			case "get_group_key_seed":
				result, err := s.messageClient.GetGroupKeySeed(ctx, userID, req.GroupID)
				if err != nil {
					slog.Warn("get_group_key_seed failed", "user_id", userID, "err", err)
					sendResponse(send, "error", map[string]string{"error": "failed to fetch group key seed"})
					continue
				}
				sendResponse(send, "group_key_seed", result)

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
		slog.Error("failed to marshal websocket response", "err", err, "type", msgType)
		return
	}
	slog.Info("sending websocket response", "type", msgType)
	send <- data
}
