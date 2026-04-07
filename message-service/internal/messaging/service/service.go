package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"
	"errors"
	"encoding/json"

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