package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"time"

	apperrors "message-service/internal/cores/errors"

	"github.com/google/uuid"
)

type messageService struct {
	messageRepo   MessageRepository
	outboxRepo    OutboxRepository
	kafkaProducer KafkaProducer
}

func NewMessageService(messageRepo MessageRepository, kafkaProducer KafkaProducer, outboxRepo OutboxRepository) MessageService {
	return &messageService{
		messageRepo:   messageRepo,
		kafkaProducer: kafkaProducer,
		outboxRepo:    outboxRepo,
	}
}

func (s *messageService) SendMessage(ctx context.Context, chatID, senderID, recipientID, content, msgType string) (*Message, error) {
	if senderID == "" || recipientID == "" || content == "" {
		return nil, apperrors.ErrInvalidInput
	}

	ids := []string{senderID, recipientID}
	sort.Strings(ids)
	chatID = ids[0] + ":" + ids[1]

	if msgType == "" {
		msgType = "text"
	}

	newMessage := Message{
		ID:               uuid.New().String(),
		ChatID:           chatID,
		SenderID:         senderID,
		RecipientID:      recipientID,
		EncryptedContent: content,
		MessageType:      msgType,
		CreatedAt:        time.Now(),
		Status:           MessageStatusSent,
	}

	if err := s.messageRepo.CreateWithChats(ctx, &newMessage); err != nil {
		return nil, err
	}

	slog.Info("publishing to Kafka", "message_id", newMessage.ID, "chat_id", newMessage.ChatID)
	if err := s.kafkaProducer.PublishMessageSent(ctx, &newMessage); err != nil {
		slog.Warn("Failed to publish to Kafka",
			"chat_id", newMessage.ChatID,
			"msg_id", newMessage.ID,
			"error", err)
		payload, _ := json.Marshal(newMessage)
		outboxEvent := &OutboxEvent{
			ID:          uuid.New().String(),
			EventType:   "message_sent",
			AggregateID: newMessage.ID,
			Payload:     payload,
			CreatedAt:   time.Now(),
			Status:      "pending",
			RetryCount:  0,
		}
		if err := s.outboxRepo.SaveToOutbox(ctx, outboxEvent); err != nil {
			slog.Error("Failed to save to outbox", "error", err)
		} else {
			slog.Info("Saved to outbox for retry", "message_id", newMessage.ID)
		}
	} else {
		slog.Info("published to Kafka successfully", "message_id", newMessage.ID)
	}

	return &newMessage, nil
}

func (s *messageService) GetMessages(ctx context.Context, chatID, userID string, limit int, lastMessageID string) ([]*Message, error) {
	if chatID == "" {
		return nil, apperrors.ErrInvalidInput
	}

	if !strings.Contains(chatID, userID) {
		return nil, apperrors.ErrForbidden
	}

	if limit <= 0 || limit > 50 {
		limit = 50
	}

	messages, err := s.messageRepo.GetByChatID(ctx, chatID, limit, lastMessageID)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

func (s *messageService) DeleteMessage(ctx context.Context, messageID, userID string) error {
	if messageID == "" || userID == "" {
		return apperrors.ErrInvalidInput
	}
	msg, err := s.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return err
	}

	if msg.SenderID != userID {
		return apperrors.ErrNotYourMessage
	}

	if err := s.messageRepo.Delete(ctx, messageID); err != nil {
		return err
	}

	if err := s.kafkaProducer.PublishMessageDeleted(ctx, msg); err != nil {
		slog.Warn("failed to publish message_deleted", "msg_id", messageID, "err", err)
		payload, _ := json.Marshal(msg)
		outboxEvent := &OutboxEvent{
			ID:          uuid.New().String(),
			EventType:   "message_deleted",
			AggregateID: messageID,
			Payload:     payload,
			CreatedAt:   time.Now(),
			Status:      "pending",
			RetryCount:  0,
		}
		s.outboxRepo.SaveToOutbox(ctx, outboxEvent)
	}

	return nil
}

func (s *messageService) MarkAsRead(ctx context.Context, chatID, userID, lastMessageID string) error {
	if chatID == "" || userID == "" || lastMessageID == "" {
		return apperrors.ErrInvalidInput
	}

	msg, err := s.messageRepo.GetByID(ctx, lastMessageID)
	if err != nil {
		return err
	}

	if err := s.messageRepo.UpdateStatusBatch(ctx, chatID, userID, lastMessageID, MessageStatusRead); err != nil {
		return err
	}

	if err := s.kafkaProducer.PublishMessageRead(ctx, chatID, userID, msg.SenderID, lastMessageID); err != nil {
		slog.Warn("failed to publish message_read", "chat_id", chatID, "err", err)
		payload, _ := json.Marshal(map[string]interface{}{
			"chat_id":         chatID,
			"user_id":         userID,
			"sender_id":       msg.SenderID,
			"last_message_id": lastMessageID,
		})
		outboxEvent := &OutboxEvent{
			ID:          uuid.New().String(),
			EventType:   "message_read",
			AggregateID: lastMessageID,
			Payload:     payload,
			CreatedAt:   time.Now(),
			Status:      "pending",
			RetryCount:  0,
		}
		s.outboxRepo.SaveToOutbox(ctx, outboxEvent)
	}

	return nil
}

func (s *messageService) AlterMessage(ctx context.Context, messageID, userID, newContent string) error {
	if messageID == "" || userID == "" || newContent == "" {
		return apperrors.ErrInvalidInput
	}

	msg, err := s.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return err
	}

	if msg.SenderID != userID {
		return apperrors.ErrNotYourMessage
	}

	if err := s.messageRepo.Alter(ctx, messageID, newContent); err != nil {
		return err
	}

	if err := s.kafkaProducer.PublishMessageAltered(ctx, msg, newContent); err != nil {
		slog.Warn("failed to publish message_altered", "msg_id", messageID, "err", err)
		payload, _ := json.Marshal(map[string]interface{}{
			"message":     msg,
			"new_content": newContent,
		})
		outboxEvent := &OutboxEvent{
			ID:          uuid.New().String(),
			EventType:   "message_altered",
			AggregateID: messageID,
			Payload:     payload,
			CreatedAt:   time.Now(),
			Status:      "pending",
			RetryCount:  0,
		}
		s.outboxRepo.SaveToOutbox(ctx, outboxEvent)
	}

	return nil
}

func (s *messageService) GetChats(ctx context.Context, userID string) ([]*ChatsList, error) {
	if userID == "" {
		return nil, apperrors.ErrInvalidInput
	}

	chats, err := s.messageRepo.GetChats(ctx, userID)
	if err != nil {
		return nil, err
	}

	return chats, nil
}

func (s *messageService) SaveChatKeys(ctx context.Context, userID, companionID, encryptedKey, keyIV string) error {
	return s.messageRepo.SaveChatKeys(ctx, userID, companionID, encryptedKey, keyIV)
}

func (s *messageService) UpdateChatKeys(ctx context.Context, userID string, keys []ChatKeyUpdate) (int, error) {
	if userID == "" {
		return 0, errors.New("user_id is required")
	}
	if len(keys) == 0 {
		return 0, errors.New("keys array is empty")
	}

	count, err := s.messageRepo.UpdateChatKeys(ctx, userID, keys)
	if err != nil {
		slog.Error("failed to update chat keys", "user_id", userID, "error", err)
		return 0, err
	}

	slog.Info("chat keys updated", "user_id", userID, "count", count)
	return count, nil
}

func (s *messageService) CreateGroup(ctx context.Context, name, createdBy string, memberIDs []string, seedDistributions []SeedDistribution) (*GroupChat, error) {
	if name == "" || createdBy == "" {
		return nil, apperrors.ErrInvalidInput
	}

	groupID, err := s.messageRepo.CreateGroup(ctx, name, "", createdBy, 0)
	if err != nil {
		return nil, err
	}

	if err := s.messageRepo.AddGroupMember(ctx, groupID, createdBy, "admin", nil); err != nil {
		return nil, err
	}

	allMembers := append([]string{createdBy}, memberIDs...)
	seen := make(map[string]bool, len(allMembers))
	for _, uid := range allMembers {
		if uid == "" || seen[uid] {
			continue
		}
		seen[uid] = true
		if uid == createdBy {
			continue
		}
		if err := s.messageRepo.AddGroupMember(ctx, groupID, uid, "member", nil); err != nil {
			return nil, err
		}
	}

	seedMap := make(map[string]SeedDistribution, len(seedDistributions))
	for _, sd := range seedDistributions {
		seedMap[sd.UserID] = sd
	}
	for _, uid := range allMembers {
		sd, ok := seedMap[uid]
		if !ok {
			continue
		}
		if err := s.messageRepo.SaveGroupKeySeed(ctx, uid, groupID, sd.EncryptedSeed, sd.EncryptedBy, 0); err != nil {
			return nil, err
		}
	}

	group, err := s.messageRepo.GetGroupByID(ctx, groupID)
	if err != nil {
		return nil, err
	}

	slog.Info("group created", "group_id", groupID, "name", name, "created_by", createdBy)
	return group, nil
}

func (s *messageService) AddGroupMember(ctx context.Context, groupID, userID, addedBy, encryptedSeed string) error {
	if groupID == "" || userID == "" || addedBy == "" {
		return apperrors.ErrInvalidInput
	}

	isMember, _, err := s.messageRepo.CheckGroupMembership(ctx, groupID, addedBy)
	if err != nil {
		return err
	}
	if !isMember {
		return apperrors.ErrNotGroupMember
	}

	isAlreadyMember, _, err := s.messageRepo.CheckGroupMembership(ctx, groupID, userID)
	if err != nil {
		return err
	}
	if isAlreadyMember {
		return apperrors.ErrAlreadyMember
	}

	group, err := s.messageRepo.GetGroupByID(ctx, groupID)
	if err != nil {
		return err
	}

	if err := s.messageRepo.AddGroupMember(ctx, groupID, userID, "member", nil); err != nil {
		return err
	}

	keyVersion, err := s.messageRepo.GetCurrentKeyVersion(ctx, groupID)
	if err != nil {
		return err
	}

	if err := s.messageRepo.SaveGroupKeySeed(ctx, userID, groupID, encryptedSeed, addedBy, keyVersion); err != nil {
		return err
	}

	slog.Info("member added to group", "group_id", groupID, "user_id", userID, "added_by", addedBy, "key_version", keyVersion)
	_ = group
	return nil
}

func (s *messageService) RemoveGroupMember(ctx context.Context, groupID, userID, removedBy string) (int, error) {
	if groupID == "" || userID == "" || removedBy == "" {
		return 0, apperrors.ErrInvalidInput
	}

	isAdmin, _, err := s.messageRepo.CheckGroupMembership(ctx, groupID, removedBy)
	if err != nil {
		return 0, err
	}
	if !isAdmin {
		return 0, apperrors.ErrNotAdmin
	}

	isMember, _, err := s.messageRepo.CheckGroupMembership(ctx, groupID, userID)
	if err != nil {
		return 0, err
	}
	if !isMember {
		return 0, apperrors.ErrNotGroupMember
	}

	if err := s.messageRepo.RemoveGroupMember(ctx, groupID, userID); err != nil {
		return 0, err
	}

	newVersion, err := s.tryKeyRotation(ctx, groupID, removedBy)
	if err != nil {
		slog.Warn("key rotation failed after remove", "group_id", groupID, "error", err)
	}

	slog.Info("member removed from group", "group_id", groupID, "user_id", userID, "removed_by", removedBy, "new_key_version", newVersion)
	return newVersion, nil
}

func (s *messageService) LeaveGroup(ctx context.Context, groupID, userID string) error {
	if groupID == "" || userID == "" {
		return apperrors.ErrInvalidInput
	}

	isMember, _, err := s.messageRepo.CheckGroupMembership(ctx, groupID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return apperrors.ErrNotGroupMember
	}

	if err := s.messageRepo.RemoveGroupMember(ctx, groupID, userID); err != nil {
		return err
	}

	if _, err := s.tryKeyRotation(ctx, groupID, userID); err != nil {
		slog.Warn("key rotation failed after leave", "group_id", groupID, "error", err)
	}

	slog.Info("member left group", "group_id", groupID, "user_id", userID)
	return nil
}

func (s *messageService) GetGroupChats(ctx context.Context, userID string) ([]*GroupChat, error) {
	if userID == "" {
		return nil, apperrors.ErrInvalidInput
	}
	return s.messageRepo.GetGroupChats(ctx, userID)
}

func (s *messageService) GetGroupMembers(ctx context.Context, groupID string) ([]*GroupMember, error) {
	if groupID == "" {
		return nil, apperrors.ErrInvalidInput
	}
	return s.messageRepo.GetGroupMembers(ctx, groupID)
}

func (s *messageService) SaveGroupKeySeed(ctx context.Context, userID, groupID, encryptedSeed, encryptedBy string, keyVersion int) error {
	if userID == "" || groupID == "" || encryptedSeed == "" || encryptedBy == "" {
		return apperrors.ErrInvalidInput
	}
	return s.messageRepo.SaveGroupKeySeed(ctx, userID, groupID, encryptedSeed, encryptedBy, keyVersion)
}

func (s *messageService) GetGroupKeySeed(ctx context.Context, userID, groupID string) (*GroupKeySeed, int, error) {
	if userID == "" || groupID == "" {
		return nil, 0, apperrors.ErrInvalidInput
	}

	isMember, _, err := s.messageRepo.CheckGroupMembership(ctx, groupID, userID)
	if err != nil {
		return nil, 0, err
	}
	if !isMember {
		return nil, 0, apperrors.ErrNotGroupMember
	}

	seed, err := s.messageRepo.GetGroupKeySeed(ctx, userID, groupID)
	if err != nil {
		return nil, 0, err
	}

	keyVersion, err := s.messageRepo.GetCurrentKeyVersion(ctx, groupID)
	if err != nil {
		return nil, 0, err
	}

	return seed, keyVersion, nil
}

func (s *messageService) tryKeyRotation(ctx context.Context, groupID, userID string) (int, error) {
	acquired, err := s.messageRepo.TryAcquireRotationLock(ctx, groupID, userID)
	if err != nil {
		return 0, err
	}
	if !acquired {
		return s.messageRepo.GetCurrentKeyVersion(ctx, groupID)
	}

	newVersion, err := s.messageRepo.IncrementKeyVersion(ctx, groupID)
	if err != nil {
		s.messageRepo.ReleaseRotationLock(ctx, groupID)
		return 0, err
	}

	if err := s.messageRepo.ReleaseRotationLock(ctx, groupID); err != nil {
		slog.Error("failed to release rotation lock", "group_id", groupID, "error", err)
	}

	slog.Info("key rotation completed", "group_id", groupID, "new_version", newVersion)
	return newVersion, nil
}
