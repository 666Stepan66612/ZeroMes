package outboxworker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"message-service/internal/messaging/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockOutboxRepository struct {
	mock.Mock
}

func (m *MockOutboxRepository) SaveToOutbox(ctx context.Context, event *service.OutboxEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockOutboxRepository) GetPendingEvents(ctx context.Context, limit int) ([]*service.OutboxEvent, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*service.OutboxEvent), args.Error(1)
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

type MockKafkaProducer struct {
	mock.Mock
}

func (m *MockKafkaProducer) PublishMessageSent(ctx context.Context, msg *service.Message) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockKafkaProducer) PublishMessageAltered(ctx context.Context, msg *service.Message, newContent string) error {
	args := m.Called(ctx, msg, newContent)
	return args.Error(0)
}

func (m *MockKafkaProducer) PublishMessageDeleted(ctx context.Context, msg *service.Message) error {
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

func TestPublishEvent_MessageSent(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	mockKafka := new(MockKafkaProducer)
	worker := NewOutboxWorker(mockRepo, mockKafka)

	ctx := context.Background()

	msg := service.Message{
		ID:               "msg-123",
		ChatID:           "chat-456",
		SenderID:         "user-1",
		RecipientID:      "user-2",
		EncryptedContent: "encrypted-content",
		MessageType:      "text",
		CreatedAt:        time.Now(),
		Status:           service.MessageStatusSent,
	}

	payload, _ := json.Marshal(msg)
	event := &service.OutboxEvent{
		ID:          "event-1",
		EventType:   "message_sent",
		AggregateID: msg.ID,
		Payload:     payload,
		CreatedAt:   time.Now(),
		Status:      "pending",
		RetryCount:  0,
	}

	mockKafka.On("PublishMessageSent", ctx, mock.AnythingOfType("*service.Message")).Return(nil)

	err := worker.publishEvent(ctx, event)

	assert.NoError(t, err)
	mockKafka.AssertExpectations(t)
}

func TestPublishEvent_MessageDeleted(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	mockKafka := new(MockKafkaProducer)
	worker := NewOutboxWorker(mockRepo, mockKafka)

	ctx := context.Background()

	msg := service.Message{
		ID:       "msg-123",
		ChatID:   "chat-456",
		SenderID: "user-1",
	}

	payload, _ := json.Marshal(msg)
	event := &service.OutboxEvent{
		ID:          "event-1",
		EventType:   "message_deleted",
		AggregateID: msg.ID,
		Payload:     payload,
		Status:      "pending",
	}

	mockKafka.On("PublishMessageDeleted", ctx, mock.AnythingOfType("*service.Message")).Return(nil)

	err := worker.publishEvent(ctx, event)

	assert.NoError(t, err)
	mockKafka.AssertExpectations(t)
}

func TestPublishEvent_MessageAltered(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	mockKafka := new(MockKafkaProducer)
	worker := NewOutboxWorker(mockRepo, mockKafka)

	ctx := context.Background()

	msg := service.Message{
		ID:               "msg-123",
		EncryptedContent: "old-content",
	}

	data := map[string]interface{}{
		"message":     msg,
		"new_content": "new-encrypted-content",
	}

	payload, _ := json.Marshal(data)
	event := &service.OutboxEvent{
		ID:          "event-1",
		EventType:   "message_altered",
		AggregateID: msg.ID,
		Payload:     payload,
		Status:      "pending",
	}

	mockKafka.On("PublishMessageAltered", ctx, mock.AnythingOfType("*service.Message"), "new-encrypted-content").Return(nil)

	err := worker.publishEvent(ctx, event)

	assert.NoError(t, err)
	mockKafka.AssertExpectations(t)
}

func TestPublishEvent_MessageRead(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	mockKafka := new(MockKafkaProducer)
	worker := NewOutboxWorker(mockRepo, mockKafka)

	ctx := context.Background()

	data := map[string]string{
		"chat_id":         "chat-123",
		"user_id":         "user-456",
		"sender_id":       "user-789",
		"last_message_id": "msg-999",
	}

	payload, _ := json.Marshal(data)
	event := &service.OutboxEvent{
		ID:          "event-1",
		EventType:   "message_read",
		AggregateID: "msg-999",
		Payload:     payload,
		Status:      "pending",
	}

	mockKafka.On("PublishMessageRead", ctx, "chat-123", "user-456", "user-789", "msg-999").Return(nil)

	err := worker.publishEvent(ctx, event)

	assert.NoError(t, err)
	mockKafka.AssertExpectations(t)
}

func TestPublishEvent_InvalidJSON(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	mockKafka := new(MockKafkaProducer)
	worker := NewOutboxWorker(mockRepo, mockKafka)

	ctx := context.Background()

	event := &service.OutboxEvent{
		ID:        "event-1",
		EventType: "message_sent",
		Payload:   []byte("invalid-json{{{"),
		Status:    "pending",
	}

	err := worker.publishEvent(ctx, event)

	assert.Error(t, err)
	mockKafka.AssertNotCalled(t, "PublishMessageSent")
}

func TestPublishEvent_UnknownEventType(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	mockKafka := new(MockKafkaProducer)
	worker := NewOutboxWorker(mockRepo, mockKafka)

	ctx := context.Background()

	event := &service.OutboxEvent{
		ID:        "event-1",
		EventType: "unknown_event_type",
		Payload:   []byte("{}"),
		Status:    "pending",
	}

	err := worker.publishEvent(ctx, event)

	assert.NoError(t, err) // Returns nil for unknown types
}

func TestProcessEvents_Success(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	mockKafka := new(MockKafkaProducer)
	worker := NewOutboxWorker(mockRepo, mockKafka)

	ctx := context.Background()

	msg := service.Message{ID: "msg-1", ChatID: "chat-1"}
	payload, _ := json.Marshal(msg)

	events := []*service.OutboxEvent{
		{
			ID:         "event-1",
			EventType:  "message_sent",
			Payload:    payload,
			RetryCount: 0,
		},
	}

	mockRepo.On("GetPendingEvents", ctx, 100).Return(events, nil)
	mockKafka.On("PublishMessageSent", ctx, mock.AnythingOfType("*service.Message")).Return(nil)
	mockRepo.On("MarkEventProcessed", ctx, "event-1").Return(nil)

	worker.processEvents(ctx)

	mockRepo.AssertExpectations(t)
	mockKafka.AssertExpectations(t)
}

func TestProcessEvents_MaxRetriesExceeded(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	mockKafka := new(MockKafkaProducer)
	worker := NewOutboxWorker(mockRepo, mockKafka)

	ctx := context.Background()

	events := []*service.OutboxEvent{
		{
			ID:         "event-1",
			EventType:  "message_sent",
			Payload:    []byte("{}"),
			RetryCount: 5, // Max retries
		},
	}

	mockRepo.On("GetPendingEvents", ctx, 100).Return(events, nil)
	mockRepo.On("MarkEventFailed", ctx, "event-1", "max retries exceeded").Return(nil)

	worker.processEvents(ctx)

	mockRepo.AssertExpectations(t)
	mockKafka.AssertNotCalled(t, "PublishMessageSent")
}

func TestProcessEvents_PublishFails(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	mockKafka := new(MockKafkaProducer)
	worker := NewOutboxWorker(mockRepo, mockKafka)

	ctx := context.Background()

	msg := service.Message{ID: "msg-1"}
	payload, _ := json.Marshal(msg)

	events := []*service.OutboxEvent{
		{
			ID:         "event-1",
			EventType:  "message_sent",
			Payload:    payload,
			RetryCount: 2,
		},
	}

	mockRepo.On("GetPendingEvents", ctx, 100).Return(events, nil)
	mockKafka.On("PublishMessageSent", ctx, mock.AnythingOfType("*service.Message")).Return(errors.New("kafka error"))
	mockRepo.On("IncrementRetryCount", ctx, "event-1").Return(nil)

	worker.processEvents(ctx)

	mockRepo.AssertExpectations(t)
	mockKafka.AssertExpectations(t)
}

func TestProcessEvents_NoEvents(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	mockKafka := new(MockKafkaProducer)
	worker := NewOutboxWorker(mockRepo, mockKafka)

	ctx := context.Background()

	mockRepo.On("GetPendingEvents", ctx, 100).Return([]*service.OutboxEvent{}, nil)

	worker.processEvents(ctx)

	mockRepo.AssertExpectations(t)
	mockKafka.AssertNotCalled(t, "PublishMessageSent")
}

func TestProcessEvents_GetPendingEventsFails(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	mockKafka := new(MockKafkaProducer)
	worker := NewOutboxWorker(mockRepo, mockKafka)

	ctx := context.Background()

	mockRepo.On("GetPendingEvents", ctx, 100).Return(nil, errors.New("database error"))

	worker.processEvents(ctx)

	mockRepo.AssertExpectations(t)
	mockKafka.AssertNotCalled(t, "PublishMessageSent")
}

func TestProcessEvents_MultipleEvents(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	mockKafka := new(MockKafkaProducer)
	worker := NewOutboxWorker(mockRepo, mockKafka)

	ctx := context.Background()

	msg1 := service.Message{ID: "msg-1"}
	msg2 := service.Message{ID: "msg-2"}
	payload1, _ := json.Marshal(msg1)
	payload2, _ := json.Marshal(msg2)

	events := []*service.OutboxEvent{
		{ID: "event-1", EventType: "message_sent", Payload: payload1, RetryCount: 0},
		{ID: "event-2", EventType: "message_sent", Payload: payload2, RetryCount: 0},
	}

	mockRepo.On("GetPendingEvents", ctx, 100).Return(events, nil)
	mockKafka.On("PublishMessageSent", ctx, mock.AnythingOfType("*service.Message")).Return(nil).Twice()
	mockRepo.On("MarkEventProcessed", ctx, "event-1").Return(nil)
	mockRepo.On("MarkEventProcessed", ctx, "event-2").Return(nil)

	worker.processEvents(ctx)

	mockRepo.AssertExpectations(t)
	mockKafka.AssertExpectations(t)
}

func TestNewOutboxWorker(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	mockKafka := new(MockKafkaProducer)

	worker := NewOutboxWorker(mockRepo, mockKafka)

	assert.NotNil(t, worker)
	assert.Equal(t, 5*time.Second, worker.interval)
	assert.Equal(t, 5, worker.maxRetries)
}
