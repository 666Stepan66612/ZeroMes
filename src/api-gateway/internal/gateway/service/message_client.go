package service

import (
	"api-gateway/internal/cores/domain"
	"context"
	"fmt"
	"log/slog"
	"time"

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
		ID:               resp.Message.Id,
		ChatID:           resp.Message.ChatId,
		SenderID:         resp.Message.SenderId,
		EncryptedContent: resp.Message.EncryptedContent,
		CreatedAt:        resp.Message.CreatedAt.AsTime().Format(time.RFC3339),
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
			ID:               m.Id,
			ChatID:           m.ChatId,
			SenderID:         m.SenderId,
			EncryptedContent: m.EncryptedContent,
			CreatedAt:        m.CreatedAt.AsTime().Format(time.RFC3339),
			Status:           int32(m.Status),
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
	slog.Info("calling message-service GetChats", "user_id", userID)
	resp, err := c.client.GetChats(ctx, &messagepb.GetChatsRequest{
		UserId: userID,
	})
	if err != nil {
		slog.Error("message-service GetChats failed", "err", err)
		return nil, err
	}
	slog.Info("message-service GetChats success", "chats_count", len(resp.Chats))

	chats := make([]*domain.Chat, len(resp.Chats))
	for i, ch := range resp.Chats {
		chats[i] = &domain.Chat{
			ID:            ch.Id,
			CompanionID:   ch.CompanionId,
			LastMessageAt: ch.LastMessageAt.AsTime().Format(time.RFC3339),
			EncryptedKey:  ch.EncryptedKey,
			KeyIV:         ch.KeyIv,
			LastMessage:   ch.LastMessage,
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

func (c *MessageClientService) UpdateChatKeys(ctx context.Context, userID string, keys []domain.ChatKeyUpdate) (int, error) {
	slog.Info("calling message-service UpdateChatKeys", "user_id", userID, "keys_count", len(keys))

	// Convert domain.ChatKeyUpdate to protobuf ChatKeyUpdate
	pbKeys := make([]*messagepb.ChatKeyUpdate, len(keys))
	for i, k := range keys {
		pbKeys[i] = &messagepb.ChatKeyUpdate{
			CompanionId:  k.CompanionID,
			EncryptedKey: k.EncryptedKey,
			KeyIv:        k.KeyIV,
		}
	}

	resp, err := c.client.UpdateChatKeys(ctx, &messagepb.UpdateChatKeysRequest{
		UserId: userID,
		Keys:   pbKeys,
	})
	if err != nil {
		slog.Error("message-service UpdateChatKeys failed", "err", err)
		return 0, err
	}

	if !resp.Success {
		slog.Error("message-service UpdateChatKeys returned failure", "error", resp.Error)
		return 0, fmt.Errorf("update failed: %s", resp.Error)
	}

	slog.Info("message-service UpdateChatKeys success", "updated_count", resp.UpdatedCount)
	return int(resp.UpdatedCount), nil
}

func (c *MessageClientService) CreateGroup(ctx context.Context, name, createdBy string, memberIDs []string, seedDistributions []domain.SeedDistribution) (*domain.GroupChat, error) {
	pbDistributions := make([]*messagepb.GroupKeySeedDistribution, len(seedDistributions))
	for i, sd := range seedDistributions {
		pbDistributions[i] = &messagepb.GroupKeySeedDistribution{
			UserId:        sd.UserID,
			EncryptedSeed: sd.EncryptedSeed,
			EncryptedBy:   sd.EncryptedBy,
		}
	}

	resp, err := c.client.CreateGroup(ctx, &messagepb.CreateGroupRequest{
		Name:              name,
		CreatedBy:         createdBy,
		MemberIds:         memberIDs,
		SeedDistributions: pbDistributions,
	})
	if err != nil {
		return nil, err
	}

	return &domain.GroupChat{
		ID:         resp.Group.Id,
		Name:       resp.Group.Name,
		AvatarURL:  resp.Group.AvatarUrl,
		CreatedBy:  resp.Group.CreatedBy,
		KeyVersion: resp.Group.KeyVersion,
		CreatedAt:  resp.Group.CreatedAt.AsTime().Format(time.RFC3339),
	}, nil
}

func (c *MessageClientService) AddGroupMember(ctx context.Context, groupID, userID, addedBy, encryptedSeed string) error {
	_, err := c.client.AddGroupMember(ctx, &messagepb.AddGroupMemberRequest{
		GroupId:       groupID,
		UserId:        userID,
		AddedBy:       addedBy,
		EncryptedSeed: encryptedSeed,
	})
	return err
}

func (c *MessageClientService) RemoveGroupMember(ctx context.Context, groupID, userID, removedBy string) (int32, error) {
	resp, err := c.client.RemoveGroupMember(ctx, &messagepb.RemoveGroupMemberRequest{
		GroupId:   groupID,
		UserId:    userID,
		RemovedBy: removedBy,
	})
	if err != nil {
		return 0, err
	}
	return resp.NewKeyVersion, nil
}

func (c *MessageClientService) LeaveGroup(ctx context.Context, groupID, userID string) error {
	_, err := c.client.LeaveGroup(ctx, &messagepb.LeaveGroupRequest{
		GroupId: groupID,
		UserId:  userID,
	})
	return err
}

func (c *MessageClientService) GetGroupChats(ctx context.Context, userID string) (*domain.GetGroupChatsResponse, error) {
	resp, err := c.client.GetGroupChats(ctx, &messagepb.GetGroupChatsRequest{
		UserId: userID,
	})
	if err != nil {
		return nil, err
	}

	groups := make([]*domain.GroupChat, len(resp.Groups))
	for i, g := range resp.Groups {
		groups[i] = &domain.GroupChat{
			ID:         g.Id,
			Name:       g.Name,
			AvatarURL:  g.AvatarUrl,
			CreatedBy:  g.CreatedBy,
			KeyVersion: g.KeyVersion,
			CreatedAt:  g.CreatedAt.AsTime().Format(time.RFC3339),
		}
	}

	return &domain.GetGroupChatsResponse{Groups: groups}, nil
}

func (c *MessageClientService) GetGroupMembers(ctx context.Context, groupID string) (*domain.GetGroupMembersResponse, error) {
	resp, err := c.client.GetGroupMembers(ctx, &messagepb.GetGroupMembersRequest{
		GroupId: groupID,
	})
	if err != nil {
		return nil, err
	}

	members := make([]*domain.GroupMember, len(resp.Members))
	for i, m := range resp.Members {
		gm := &domain.GroupMember{
			UserID:   m.UserId,
			Role:     m.Role,
			JoinedAt: m.JoinedAt.AsTime().Format(time.RFC3339),
		}
		if m.CanReadFromMessageId != "" {
			gm.CanReadFromMessageID = &m.CanReadFromMessageId
		}
		members[i] = gm
	}

	return &domain.GetGroupMembersResponse{Members: members}, nil
}

func (c *MessageClientService) SaveGroupKeySeed(ctx context.Context, userID, groupID, encryptedSeed, encryptedBy string, keyVersion int32) error {
	_, err := c.client.SaveGroupKeySeed(ctx, &messagepb.SaveGroupKeySeedRequest{
		UserId:        userID,
		GroupId:       groupID,
		EncryptedSeed: encryptedSeed,
		EncryptedBy:   encryptedBy,
		KeyVersion:    keyVersion,
	})
	return err
}

func (c *MessageClientService) GetGroupKeySeed(ctx context.Context, userID, groupID string) (*domain.GetGroupKeySeedResponse, error) {
	resp, err := c.client.GetGroupKeySeed(ctx, &messagepb.GetGroupKeySeedRequest{
		UserId:  userID,
		GroupId: groupID,
	})
	if err != nil {
		return nil, err
	}

	return &domain.GetGroupKeySeedResponse{
		Seed: &domain.GroupKeySeed{
			EncryptedSeed: resp.Seed.EncryptedSeed,
			EncryptedBy:   resp.Seed.EncryptedBy,
			KeyVersion:    resp.Seed.KeyVersion,
		},
		CurrentKeyVersion: resp.CurrentKeyVersion,
	}, nil
}
