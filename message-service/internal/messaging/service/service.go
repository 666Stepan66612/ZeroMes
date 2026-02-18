package service

import (
	"context"
	"time"

	apperrors "message-service/internal/cores/errors"
	"github.com/google/uuid"
)

type messageService struct {
	messageRepo MessageRepository
	kafkaProducer KafkaProducer
}

func NewMessageService(messageRepo MessageRepository, kafkaProducer KafkaProducer) MessageService {
	return &messageService{
		messageRepo: messageRepo,
		kafkaProducer: kafkaProducer,
	}
}

func (s *messageService) SendMessage(ctx context.Context, chatID, senderID, recipientID, content, msgType string) (*Message, error) {
	newMessage := Message{
		ID: uuid.New().String(),
		ChatID: chatID,
		SenderID: senderID,
		RecipientID: recipientID,
		EncryptedContent: content,
		MessageType: msgType,
		CreatedAt: time.Now(),
		Status: MessageStatusSent,
	}

	if err := s.messageRepo.Create(ctx, &newMessage); err != nil {
		return nil, err
	}

	if err := s.kafkaProducer.PublishMessageSent(ctx, &newMessage); err != nil {
		// Логировать ошибку, но не возвращать (сообщение уже в БД)
        // TODO: добавить retry или DLQ
	}

	return &newMessage, nil
}

func (s *messageService) GetMessages(ctx context.Context, chatID, userID string, limit int, lastMessageID string) ([]*Message, error) {
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

	return nil
}

func (s *messageService) MarkAsRead(ctx context.Context, chatID, userID, lastMessageID string) error {
	msg, err := s.messageRepo.GetByID(ctx, lastMessageID)
	if err != nil {
		return err
	}

	if msg.SenderID != userID {
        return apperrors.ErrNotYourMessage
    }

	if err := s.messageRepo.UpdateStatusBatch(ctx, chatID, userID, lastMessageID, MessageStatusRead); err != nil {
		return err
	}

	return nil
}

func (s *messageService) AlterMessage(ctx context.Context, messageID, userID, newContent string) error {
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

	return nil
}
