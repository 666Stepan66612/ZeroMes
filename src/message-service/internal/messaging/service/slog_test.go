package service

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type TestLogHandler struct {
	logs []map[string]interface{}
	buf  *bytes.Buffer
}

func NewTestLogHandler() *TestLogHandler {
	buf := &bytes.Buffer{}
	return &TestLogHandler{
		logs: make([]map[string]interface{}, 0),
		buf:  buf,
	}
}

func (h *TestLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (h *TestLogHandler) Handle(ctx context.Context, record slog.Record) error {
	logEntry := map[string]interface{}{
		"level": record.Level.String(),
		"msg":   record.Message,
		"time":  record.Time,
	}

	record.Attrs(func(attr slog.Attr) bool {
		logEntry[attr.Key] = attr.Value.Any()
		return true
	})

	h.logs = append(h.logs, logEntry)
	return nil
}

func (h *TestLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *TestLogHandler) WithGroup(name string) slog.Handler {
	return h
}

func (h *TestLogHandler) GetLogs() []map[string]interface{} {
	return h.logs
}

func (h *TestLogHandler) FindLog(msg string) (map[string]interface{}, bool) {
	for _, log := range h.logs {
		if log["msg"] == msg {
			return log, true
		}
	}
	return nil, false
}

func TestSendMessage_LogsKafkaFailure(t *testing.T) {
	testHandler := NewTestLogHandler()
	oldLogger := slog.Default()
	defer slog.SetDefault(oldLogger)

	logger := slog.New(testHandler)
	slog.SetDefault(logger)

	mockRepo := new(MockMessageRepository)
	mockKafkaProd := new(MockKafkaProducer)
	mockOutboxRepo := new(MockOutboxRepository)
	service := NewMessageService(mockRepo, mockKafkaProd, mockOutboxRepo)

	ctx := context.Background()
	senderID := "user-123"
	recipientID := "user-456"
	content := "encrypted-content"

	mockRepo.On("CreateWithChats", ctx, mock.AnythingOfType("*service.Message")).Return(nil)
	mockKafkaProd.On("PublishMessageSent", ctx, mock.AnythingOfType("*service.Message")).Return(assert.AnError)
	mockOutboxRepo.On("SaveToOutbox", ctx, mock.AnythingOfType("*service.OutboxEvent")).Return(nil)

	msg, err := service.SendMessage(ctx, "", senderID, recipientID, content, "text")

	assert.NoError(t, err)
	assert.NotNil(t, msg)

	warnLog, found := testHandler.FindLog("Failed to publish to Kafka")
	assert.True(t, found, "Should log Kafka failure warning")
	assert.Equal(t, "WARN", warnLog["level"])
	assert.Equal(t, msg.ID, warnLog["msg_id"])
	assert.Equal(t, msg.ChatID, warnLog["chat_id"])

	infoLog, found := testHandler.FindLog("Saved to outbox for retry")
	assert.True(t, found, "Should log outbox save")
	assert.Equal(t, "INFO", infoLog["level"])
	assert.Equal(t, msg.ID, infoLog["message_id"])

	mockRepo.AssertExpectations(t)
	mockKafkaProd.AssertExpectations(t)
	mockOutboxRepo.AssertExpectations(t)
}

func TestSendMessage_LogsWithJSONHandler(t *testing.T) {
	var buf bytes.Buffer

	jsonHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	oldLogger := slog.Default()
	defer slog.SetDefault(oldLogger)

	logger := slog.New(jsonHandler)
	slog.SetDefault(logger)

	mockRepo := new(MockMessageRepository)
	mockKafkaProd := new(MockKafkaProducer)
	mockOutboxRepo := new(MockOutboxRepository)
	service := NewMessageService(mockRepo, mockKafkaProd, mockOutboxRepo)

	ctx := context.Background()
	senderID := "user-123"
	recipientID := "user-456"

	mockRepo.On("CreateWithChats", ctx, mock.AnythingOfType("*service.Message")).Return(nil)
	mockKafkaProd.On("PublishMessageSent", ctx, mock.AnythingOfType("*service.Message")).Return(nil)

	msg, err := service.SendMessage(ctx, "", senderID, recipientID, "content", "text")

	assert.NoError(t, err)
	assert.NotNil(t, msg)

	logs := buf.String()
	assert.Contains(t, logs, "publishing to Kafka")
	assert.Contains(t, logs, msg.ID)
	assert.Contains(t, logs, msg.ChatID)

	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var logEntry map[string]interface{}
		err := json.Unmarshal(line, &logEntry)
		assert.NoError(t, err)

		if logEntry["msg"] == "publishing to Kafka" {
			assert.Equal(t, msg.ID, logEntry["message_id"])
			assert.Equal(t, msg.ChatID, logEntry["chat_id"])
		}
	}
}

func TestSendMessage_DoesNotLogDebugInProduction(t *testing.T) {
	var buf bytes.Buffer

	jsonHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	oldLogger := slog.Default()
	defer slog.SetDefault(oldLogger)

	logger := slog.New(jsonHandler)
	slog.SetDefault(logger)

	slog.Debug("debug message")
	slog.Info("info message")
	slog.Warn("warn message")

	logs := buf.String()

	assert.NotContains(t, logs, "debug message")
	assert.Contains(t, logs, "info message")
	assert.Contains(t, logs, "warn message")
}
