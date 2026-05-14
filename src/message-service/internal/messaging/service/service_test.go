package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock MessageRepository
type MockMessageRepository struct {
	mock.Mock
}

func (m *MockMessageRepository) CreateWithChats(ctx context.Context, msg *Message) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockMessageRepository) GetByChatID(ctx context.Context, chatID string, limit int, lastMessageID string) ([]*Message, error) {
	args := m.Called(ctx, chatID, limit, lastMessageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Message), args.Error(1)
}

func (m *MockMessageRepository) GetByID(ctx context.Context, messageID string) (*Message, error) {
	args := m.Called(ctx, messageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Message), args.Error(1)
}

func (m *MockMessageRepository) Delete(ctx context.Context, messageID string) error {
	args := m.Called(ctx, messageID)
	return args.Error(0)
}

func (m *MockMessageRepository) Alter(ctx context.Context, messageID, newContent string) error {
	args := m.Called(ctx, messageID, newContent)
	return args.Error(0)
}

func (m *MockMessageRepository) UpdateStatusBatch(ctx context.Context, chatID, userID, lastMessageID string, status MessageStatus) error {
	args := m.Called(ctx, chatID, userID, lastMessageID, status)
	return args.Error(0)
}

func (m *MockMessageRepository) GetChats(ctx context.Context, userID string) ([]*ChatsList, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ChatsList), args.Error(1)
}

func (m *MockMessageRepository) SaveChatKeys(ctx context.Context, userID, companionID, encryptedKey, keyIV string) error {
	args := m.Called(ctx, userID, companionID, encryptedKey, keyIV)
	return args.Error(0)
}

func (m *MockMessageRepository) UpdateChatKeys(ctx context.Context, userID string, keys []ChatKeyUpdate) (int, error) {
	args := m.Called(ctx, userID, keys)
	return args.Int(0), args.Error(1)
}

// Mock KafkaProducer
type MockKafkaProducer struct {
	mock.Mock
}

func (m *MockKafkaProducer) PublishMessageSent(ctx context.Context, msg *Message) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockKafkaProducer) PublishMessageAltered(ctx context.Context, msg *Message, newContent string) error {
	args := m.Called(ctx, msg, newContent)
	return args.Error(0)
}

func (m *MockKafkaProducer) PublishMessageDeleted(ctx context.Context, msg *Message) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockKafkaProducer) PublishMessageRead(ctx context.Context, chatID, readerID, senderID, lastMessageID string) error {
	args := m.Called(ctx, chatID, readerID, senderID, lastMessageID)
	return args.Error(0)
}

func (m *MockKafkaProducer) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Mock OutboxRepository
type MockOutboxRepository struct {
	mock.Mock
}

func (m *MockOutboxRepository) SaveToOutbox(ctx context.Context, event *OutboxEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockOutboxRepository) GetPendingEvents(ctx context.Context, limit int) ([]*OutboxEvent, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*OutboxEvent), args.Error(1)
}

func (m *MockOutboxRepository) MarkEventProcessed(ctx context.Context, eventID string) error {
	args := m.Called(ctx, eventID)
	return args.Error(0)
}

func (m *MockOutboxRepository) MarkEventFailed(ctx context.Context, eventID string, errorMsg string) error {
	args := m.Called(ctx, eventID, errorMsg)
	return args.Error(0)
}

func (m *MockOutboxRepository) IncrementRetryCount(ctx context.Context, eventID string) error {
	args := m.Called(ctx, eventID)
	return args.Error(0)
}

func TestSendMessage_Success(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockKafkaProd := new(MockKafkaProducer)
	mockOutboxRepo := new(MockOutboxRepository)
	service := NewMessageService(mockRepo, mockKafkaProd, mockOutboxRepo)

	ctx := context.Background()

	senderID := "user-123"
	recipientID := "user-456"
	content := "encrypted-content"
	msgType := "text"

	mockRepo.On("CreateWithChats", ctx, mock.AnythingOfType("*service.Message")).Return(nil)
	mockKafkaProd.On("PublishMessageSent", ctx, mock.AnythingOfType("*service.Message")).Return(nil)

	msg, err := service.SendMessage(ctx, "", senderID, recipientID, content, msgType)

	assert.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, senderID, msg.SenderID)
	assert.Equal(t, recipientID, msg.RecipientID)
	assert.Equal(t, "user-123:user-456", msg.ChatID)
	assert.Equal(t, content, msg.EncryptedContent)
	assert.Equal(t, msgType, msg.MessageType)

	mockKafkaProd.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestSendMessage_CreateChatsWrong(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockKafkaProd := new(MockKafkaProducer)
	mockOutboxRepo := new(MockOutboxRepository)
	service := NewMessageService(mockRepo, mockKafkaProd, mockOutboxRepo)

	ctx := context.Background()

	senderID := "user-123"
	recipientID := "user-456"
	content := "encrypted-content"
	msgType := "text"

	mockRepo.On("CreateWithChats", ctx, mock.AnythingOfType("*service.Message")).Return(errors.ErrUnsupported)

	msg, err := service.SendMessage(ctx, "", senderID, recipientID, content, msgType)

	assert.Error(t, err)
	assert.Nil(t, msg)

	mockKafkaProd.AssertNotCalled(t, "PublishMessageSent")
	mockRepo.AssertExpectations(t)
}

func TestSendMessage_OutBoxSuccess(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockKafkaProd := new(MockKafkaProducer)
	mockOutboxRepo := new(MockOutboxRepository)
	service := NewMessageService(mockRepo, mockKafkaProd, mockOutboxRepo)

	ctx := context.Background()

	senderID := "user-123"
	recipientID := "user-456"
	content := "encrypted-content"
	msgType := "text"

	mockRepo.On("CreateWithChats", ctx, mock.AnythingOfType("*service.Message")).Return(nil)
	mockKafkaProd.On("PublishMessageSent", ctx, mock.AnythingOfType("*service.Message")).Return(errors.ErrUnsupported)
	mockOutboxRepo.On("SaveToOutbox", ctx, mock.AnythingOfType("*service.OutboxEvent")).Return(nil)

	msg, err := service.SendMessage(ctx, "", senderID, recipientID, content, msgType)

	assert.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, senderID, msg.SenderID)
	assert.Equal(t, recipientID, msg.RecipientID)
	assert.Equal(t, "user-123:user-456", msg.ChatID)
	assert.Equal(t, content, msg.EncryptedContent)
	assert.Equal(t, msgType, msg.MessageType)

	mockKafkaProd.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockOutboxRepo.AssertExpectations(t)
}

func TestSendMessage_OutBoxWrong(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockKafkaProd := new(MockKafkaProducer)
	mockOutboxRepo := new(MockOutboxRepository)
	service := NewMessageService(mockRepo, mockKafkaProd, mockOutboxRepo)

	ctx := context.Background()

	senderID := "user-123"
	recipientID := "user-456"
	content := "encrypted-content"
	msgType := "text"

	mockRepo.On("CreateWithChats", ctx, mock.AnythingOfType("*service.Message")).Return(nil)
	mockKafkaProd.On("PublishMessageSent", ctx, mock.AnythingOfType("*service.Message")).Return(errors.ErrUnsupported)
	mockOutboxRepo.On("SaveToOutbox", ctx, mock.AnythingOfType("*service.OutboxEvent")).Return(errors.ErrUnsupported)

	msg, err := service.SendMessage(ctx, "", senderID, recipientID, content, msgType)

	assert.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, senderID, msg.SenderID)
	assert.Equal(t, recipientID, msg.RecipientID)
	assert.Equal(t, "user-123:user-456", msg.ChatID)
	assert.Equal(t, content, msg.EncryptedContent)
	assert.Equal(t, msgType, msg.MessageType)
	mockKafkaProd.AssertExpectations(t)

	mockRepo.AssertExpectations(t)
	mockOutboxRepo.AssertExpectations(t)
}

func TestGetMessages_Success(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockKafkaProd := new(MockKafkaProducer)
	mockOutboxRepo := new(MockOutboxRepository)
	service := NewMessageService(mockRepo, mockKafkaProd, mockOutboxRepo)

	ctx := context.Background()
	chatID := "user-123:user-456"
	userID := "user-123"
	limit := 20

	expectedMessages := []*Message{
		{ID: "msg-1", ChatID: chatID, SenderID: "user-123", RecipientID: "user-456", EncryptedContent: "content-1"},
		{ID: "msg-2", ChatID: chatID, SenderID: "user-456", RecipientID: "user-123", EncryptedContent: "content-2"},
	}

	mockRepo.On("GetByChatID", ctx, chatID, limit, "").Return(expectedMessages, nil)

	messages, err := service.GetMessages(ctx, chatID, userID, limit, "")

	assert.NoError(t, err)
	assert.Equal(t, expectedMessages, messages)
	assert.Len(t, messages, 2)

	mockRepo.AssertExpectations(t)
}

func TestGetMessages_EmptyChatID(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockKafkaProd := new(MockKafkaProducer)
	mockOutboxRepo := new(MockOutboxRepository)
	service := NewMessageService(mockRepo, mockKafkaProd, mockOutboxRepo)

	ctx := context.Background()

	messages, err := service.GetMessages(ctx, "", "user-123", 20, "")

	assert.Error(t, err)
	assert.Nil(t, messages)

	mockRepo.AssertNotCalled(t, "GetByChatID")
}

func TestGetMessages_Forbidden(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockKafkaProd := new(MockKafkaProducer)
	mockOutboxRepo := new(MockOutboxRepository)
	service := NewMessageService(mockRepo, mockKafkaProd, mockOutboxRepo)

	ctx := context.Background()
	chatID := "user-123:user-456"
	userID := "user-789"

	messages, err := service.GetMessages(ctx, chatID, userID, 20, "")

	assert.Error(t, err)
	assert.Nil(t, messages)

	mockRepo.AssertNotCalled(t, "GetByChatID")
}

func TestGetMessages_LimitAdjustment(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockKafkaProd := new(MockKafkaProducer)
	mockOutboxRepo := new(MockOutboxRepository)
	service := NewMessageService(mockRepo, mockKafkaProd, mockOutboxRepo)

	ctx := context.Background()
	chatID := "user-123:user-456"
	userID := "user-123"

	testCases := []struct {
		name          string
		inputLimit    int
		expectedLimit int
	}{
		{"zero limit", 0, 50},
		{"negative limit", -10, 50},
		{"over max limit", 100, 50},
		{"valid limit", 25, 25},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo.On("GetByChatID", ctx, chatID, tc.expectedLimit, "").Return([]*Message{}, nil).Once()

			_, err := service.GetMessages(ctx, chatID, userID, tc.inputLimit, "")

			assert.NoError(t, err)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestDeleteMessage_Success(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockKafkaProd := new(MockKafkaProducer)
	mockOutboxRepo := new(MockOutboxRepository)
	service := NewMessageService(mockRepo, mockKafkaProd, mockOutboxRepo)

	ctx := context.Background()
	messageID := "msg-123"
	userID := "user-123"

	existingMessage := &Message{
		ID:       messageID,
		SenderID: userID,
		ChatID:   "user-123:user-456",
	}

	mockRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)
	mockRepo.On("Delete", ctx, messageID).Return(nil)
	mockKafkaProd.On("PublishMessageDeleted", ctx, existingMessage).Return(nil)

	err := service.DeleteMessage(ctx, messageID, userID)

	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
	mockKafkaProd.AssertExpectations(t)
}

func TestDeleteMessage_NotYourMessage(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockKafkaProd := new(MockKafkaProducer)
	mockOutboxRepo := new(MockOutboxRepository)
	service := NewMessageService(mockRepo, mockKafkaProd, mockOutboxRepo)

	ctx := context.Background()
	messageID := "msg-123"
	userID := "user-456"

	existingMessage := &Message{
		ID:       messageID,
		SenderID: "user-123",
		ChatID:   "user-123:user-456",
	}

	mockRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)

	err := service.DeleteMessage(ctx, messageID, userID)

	assert.Error(t, err)

	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Delete")
	mockKafkaProd.AssertNotCalled(t, "PublishMessageDeleted")
}

func TestDeleteMessage_EmptyInputs(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockKafkaProd := new(MockKafkaProducer)
	mockOutboxRepo := new(MockOutboxRepository)
	service := NewMessageService(mockRepo, mockKafkaProd, mockOutboxRepo)

	ctx := context.Background()

	testCases := []struct {
		name      string
		messageID string
		userID    string
	}{
		{"empty messageID", "", "user-123"},
		{"empty userID", "msg-123", ""},
		{"both empty", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := service.DeleteMessage(ctx, tc.messageID, tc.userID)

			assert.Error(t, err)
			mockRepo.AssertNotCalled(t, "GetByID")
		})
	}
}

func TestAlterMessage_Success(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockKafkaProd := new(MockKafkaProducer)
	mockOutboxRepo := new(MockOutboxRepository)
	service := NewMessageService(mockRepo, mockKafkaProd, mockOutboxRepo)

	ctx := context.Background()
	messageID := "msg-123"
	userID := "user-123"
	newContent := "new-encrypted-content"

	existingMessage := &Message{
		ID:               messageID,
		SenderID:         userID,
		EncryptedContent: "old-content",
	}

	mockRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)
	mockRepo.On("Alter", ctx, messageID, newContent).Return(nil)
	mockKafkaProd.On("PublishMessageAltered", ctx, existingMessage, newContent).Return(nil)

	err := service.AlterMessage(ctx, messageID, userID, newContent)

	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
	mockKafkaProd.AssertExpectations(t)
}

func TestAlterMessage_NotYourMessage(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockKafkaProd := new(MockKafkaProducer)
	mockOutboxRepo := new(MockOutboxRepository)
	service := NewMessageService(mockRepo, mockKafkaProd, mockOutboxRepo)

	ctx := context.Background()
	messageID := "msg-123"
	userID := "user-456"
	newContent := "new-content"

	existingMessage := &Message{
		ID:       messageID,
		SenderID: "user-123",
	}

	mockRepo.On("GetByID", ctx, messageID).Return(existingMessage, nil)

	err := service.AlterMessage(ctx, messageID, userID, newContent)

	assert.Error(t, err)

	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "Alter")
}

func TestAlterMessage_EmptyInputs(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockKafkaProd := new(MockKafkaProducer)
	mockOutboxRepo := new(MockOutboxRepository)
	service := NewMessageService(mockRepo, mockKafkaProd, mockOutboxRepo)

	ctx := context.Background()

	testCases := []struct {
		name       string
		messageID  string
		userID     string
		newContent string
	}{
		{"empty messageID", "", "user-123", "content"},
		{"empty userID", "msg-123", "", "content"},
		{"empty newContent", "msg-123", "user-123", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := service.AlterMessage(ctx, tc.messageID, tc.userID, tc.newContent)

			assert.Error(t, err)
			mockRepo.AssertNotCalled(t, "GetByID")
		})
	}
}

func TestMarkAsRead_Success(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockKafkaProd := new(MockKafkaProducer)
	mockOutboxRepo := new(MockOutboxRepository)
	service := NewMessageService(mockRepo, mockKafkaProd, mockOutboxRepo)

	ctx := context.Background()
	chatID := "user-123:user-456"
	userID := "user-456"
	lastMessageID := "msg-123"

	existingMessage := &Message{
		ID:       lastMessageID,
		SenderID: "user-123",
		ChatID:   chatID,
	}

	mockRepo.On("GetByID", ctx, lastMessageID).Return(existingMessage, nil)
	mockRepo.On("UpdateStatusBatch", ctx, chatID, userID, lastMessageID, MessageStatusRead).Return(nil)
	mockKafkaProd.On("PublishMessageRead", ctx, chatID, userID, existingMessage.SenderID, lastMessageID).Return(nil)

	err := service.MarkAsRead(ctx, chatID, userID, lastMessageID)

	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
	mockKafkaProd.AssertExpectations(t)
}

func TestMarkAsRead_EmptyInputs(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockKafkaProd := new(MockKafkaProducer)
	mockOutboxRepo := new(MockOutboxRepository)
	service := NewMessageService(mockRepo, mockKafkaProd, mockOutboxRepo)

	ctx := context.Background()

	testCases := []struct {
		name          string
		chatID        string
		userID        string
		lastMessageID string
	}{
		{"empty chatID", "", "user-123", "msg-123"},
		{"empty userID", "chat-123", "", "msg-123"},
		{"empty lastMessageID", "chat-123", "user-123", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := service.MarkAsRead(ctx, tc.chatID, tc.userID, tc.lastMessageID)

			assert.Error(t, err)
			mockRepo.AssertNotCalled(t, "GetByID")
		})
	}
}

func TestGetChats_Success(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockKafkaProd := new(MockKafkaProducer)
	mockOutboxRepo := new(MockOutboxRepository)
	service := NewMessageService(mockRepo, mockKafkaProd, mockOutboxRepo)

	ctx := context.Background()
	userID := "user-123"

	expectedChats := []*ChatsList{
		{ChatID: "user-123:user-456", CompanionID: "user-456"},
		{ChatID: "user-123:user-789", CompanionID: "user-789"},
	}

	mockRepo.On("GetChats", ctx, userID).Return(expectedChats, nil)

	chats, err := service.GetChats(ctx, userID)

	assert.NoError(t, err)
	assert.Equal(t, expectedChats, chats)
	assert.Len(t, chats, 2)

	mockRepo.AssertExpectations(t)
}

func TestGetChats_EmptyUserID(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockKafkaProd := new(MockKafkaProducer)
	mockOutboxRepo := new(MockOutboxRepository)
	service := NewMessageService(mockRepo, mockKafkaProd, mockOutboxRepo)

	ctx := context.Background()

	chats, err := service.GetChats(ctx, "")

	assert.Error(t, err)
	assert.Nil(t, chats)

	mockRepo.AssertNotCalled(t, "GetChats")
}

func TestUpdateChatKeys_Success(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockKafkaProd := new(MockKafkaProducer)
	mockOutboxRepo := new(MockOutboxRepository)
	service := NewMessageService(mockRepo, mockKafkaProd, mockOutboxRepo)

	ctx := context.Background()
	userID := "user-123"
	keys := []ChatKeyUpdate{
		{CompanionID: "user-456", EncryptedKey: "key-1"},
		{CompanionID: "user-789", EncryptedKey: "key-2"},
	}

	mockRepo.On("UpdateChatKeys", ctx, userID, keys).Return(2, nil)

	count, err := service.UpdateChatKeys(ctx, userID, keys)

	assert.NoError(t, err)
	assert.Equal(t, 2, count)

	mockRepo.AssertExpectations(t)
}

func TestUpdateChatKeys_EmptyUserID(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockKafkaProd := new(MockKafkaProducer)
	mockOutboxRepo := new(MockOutboxRepository)
	service := NewMessageService(mockRepo, mockKafkaProd, mockOutboxRepo)

	ctx := context.Background()
	keys := []ChatKeyUpdate{{CompanionID: "user-456", EncryptedKey: "key-1"}}

	count, err := service.UpdateChatKeys(ctx, "", keys)

	assert.Error(t, err)
	assert.Equal(t, 0, count)
	assert.Contains(t, err.Error(), "user_id is required")

	mockRepo.AssertNotCalled(t, "UpdateChatKeys")
}

func TestUpdateChatKeys_EmptyKeys(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockKafkaProd := new(MockKafkaProducer)
	mockOutboxRepo := new(MockOutboxRepository)
	service := NewMessageService(mockRepo, mockKafkaProd, mockOutboxRepo)

	ctx := context.Background()
	userID := "user-123"

	count, err := service.UpdateChatKeys(ctx, userID, []ChatKeyUpdate{})

	assert.Error(t, err)
	assert.Equal(t, 0, count)
	assert.Contains(t, err.Error(), "keys array is empty")

	mockRepo.AssertNotCalled(t, "UpdateChatKeys")
}
