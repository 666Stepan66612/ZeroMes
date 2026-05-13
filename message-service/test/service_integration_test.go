//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/kafka"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"message-service/internal/messaging/repository"
	"message-service/internal/messaging/service"
	kafkapkg "message-service/pkg/kafka"
)

var (
	testDB           *pgxpool.Pool
	testKafka        service.KafkaProducer
	kafkaBrokers     []string
	pgContainer      *postgres.PostgresContainer
	kafkaContainer   *kafka.KafkaContainer
)

func TestMain(m *testing.M) {
	// Suppress testcontainers logs (including emojis)
	log.SetOutput(io.Discard)

	ctx := context.Background()

	// Setup PostgreSQL
	var err error
	pgContainer, err = postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to start PostgreSQL: %v", err))
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic(err)
	}

	// Retry connection to PostgreSQL (wait for it to be ready)
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		testDB, err = pgxpool.New(ctx, connStr)
		if err == nil {
			if err := testDB.Ping(ctx); err == nil {
				break
			}
			testDB.Close()
		}
		if i == maxRetries-1 {
			panic(fmt.Sprintf("Failed to connect to database after %d retries: %v", maxRetries, err))
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Run migrations
	migrator, err := migrate.New(
		"file://../migrations",
		connStr,
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to create migrator: %v", err))
	}

	if err := migrator.Up(); err != nil && err != migrate.ErrNoChange {
		panic(fmt.Sprintf("Failed to run migrations: %v", err))
	}

	// Setup Kafka
	kafkaContainer, err = kafka.RunContainer(ctx,
		testcontainers.WithImage("confluentinc/cp-kafka:7.5.0"),
		kafka.WithClusterID("test-cluster"),
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to start Kafka: %v", err))
	}

	kafkaBrokers, err = kafkaContainer.Brokers(ctx)
	if err != nil {
		panic(err)
	}

	// Create Kafka topics
	config := sarama.NewConfig()
	config.Version = sarama.V3_0_0_0
	admin, err := sarama.NewClusterAdmin(kafkaBrokers, config)
	if err != nil {
		panic(err)
	}
	defer admin.Close()

	err = admin.CreateTopic("messages.sent", &sarama.TopicDetail{
		NumPartitions:     1,
		ReplicationFactor: 1,
	}, false)
	if err != nil {
		panic(fmt.Sprintf("Failed to create topic: %v", err))
	}

	// Setup Kafka producer
	testKafka = kafkapkg.NewProducer(kafkaBrokers, "messages.sent")

	// Run tests
	code := m.Run()

	// Cleanup
	testDB.Close()
	testKafka.Close()
	pgContainer.Terminate(ctx)
	kafkaContainer.Terminate(ctx)

	os.Exit(code)
}

func cleanupAll(t *testing.T) {
	ctx := context.Background()

	_, err := testDB.Exec(ctx, "TRUNCATE TABLE messages, chats, outbox_events CASCADE")
	require.NoError(t, err)
}

func createTestConsumer(t *testing.T, topic string) sarama.PartitionConsumer {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true

	consumer, err := sarama.NewConsumer(kafkaBrokers, config)
	require.NoError(t, err)

	partitionConsumer, err := consumer.ConsumePartition(topic, 0, sarama.OffsetNewest)
	require.NoError(t, err)

	return partitionConsumer
}

// Test 1: Full flow - message saved to DB, event published to Kafka, chats created
func TestSendMessage_FullFlow_Integration(t *testing.T) {
	cleanupAll(t)
	ctx := context.Background()

	repo := repository.NewPostgresRepository(testDB)
	outboxRepo := repository.NewOutboxRepository(testDB)
	svc := service.NewMessageService(repo, testKafka, outboxRepo)

	consumer := createTestConsumer(t, "messages.sent")
	defer consumer.Close()

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()

	// Send message
	msg, err := svc.SendMessage(ctx, "", user1ID, user2ID, "encrypted-content", "text")
	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.NotEmpty(t, msg.ID)
	assert.Equal(t, user1ID, msg.SenderID)
	assert.Equal(t, user2ID, msg.RecipientID)
	// ChatID is sorted by service, so don't check exact format
	assert.NotEmpty(t, msg.ChatID)
	assert.Equal(t, "encrypted-content", msg.EncryptedContent)
	assert.Equal(t, service.MessageStatusSent, msg.Status)

	// Check message in DB
	saved, err := repo.GetByID(ctx, msg.ID)
	require.NoError(t, err)
	assert.Equal(t, msg.EncryptedContent, saved.EncryptedContent)
	assert.Equal(t, msg.SenderID, saved.SenderID)
	assert.Equal(t, msg.RecipientID, saved.RecipientID)

	// Check chats created
	chats, err := repo.GetChats(ctx, user1ID)
	require.NoError(t, err)
	assert.Len(t, chats, 1)
	assert.Equal(t, msg.ChatID, chats[0].ChatID)
	assert.Equal(t, user2ID, chats[0].CompanionID)

	chats, err = repo.GetChats(ctx, user2ID)
	require.NoError(t, err)
	assert.Len(t, chats, 1)
	assert.Equal(t, msg.ChatID, chats[0].ChatID)
	assert.Equal(t, user1ID, chats[0].CompanionID)

	// Check Kafka event
	select {
	case kafkaMsg := <-consumer.Messages():
		// Kafka producer uses protobuf, so we just check that message was sent
		assert.NotEmpty(t, kafkaMsg.Value)
		assert.Equal(t, msg.ChatID, string(kafkaMsg.Key))

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout: Kafka event not received")
	}
}

// Test 2: Kafka fails - message saved to outbox
func TestSendMessage_KafkaFails_SavesToOutbox_Integration(t *testing.T) {
	cleanupAll(t)
	ctx := context.Background()

	repo := repository.NewPostgresRepository(testDB)
	outboxRepo := repository.NewOutboxRepository(testDB)

	// Broken Kafka producer
	brokenKafka := &BrokenKafkaProducer{}

	svc := service.NewMessageService(repo, brokenKafka, outboxRepo)

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()

	// Send message
	msg, err := svc.SendMessage(ctx, "", user1ID, user2ID, "encrypted-content", "text")
	require.NoError(t, err)
	require.NotNil(t, msg)

	// Check message in DB
	saved, err := repo.GetByID(ctx, msg.ID)
	require.NoError(t, err)
	assert.NotNil(t, saved)

	// Check event in outbox
	events, err := outboxRepo.GetPendingEvents(ctx, 10)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "message_sent", events[0].EventType)
	assert.Equal(t, msg.ID, events[0].AggregateID)
	assert.Equal(t, 0, events[0].RetryCount)
	assert.Equal(t, "pending", events[0].Status)
}

// Test 3: Concurrent message sending to same chat
func TestSendMessage_Concurrent_Integration(t *testing.T) {
	cleanupAll(t)
	ctx := context.Background()

	repo := repository.NewPostgresRepository(testDB)
	outboxRepo := repository.NewOutboxRepository(testDB)
	svc := service.NewMessageService(repo, testKafka, outboxRepo)

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()

	var wg sync.WaitGroup
	errors := make(chan error, 10)
	messages := make(chan *service.Message, 10)

	// 10 goroutines send messages concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			msg, err := svc.SendMessage(ctx, "", user1ID, user2ID, fmt.Sprintf("content-%d", id), "text")
			if err != nil {
				errors <- err
			} else {
				messages <- msg
			}
		}(i)
	}

	wg.Wait()
	close(errors)
	close(messages)

	// Check no errors
	for err := range errors {
		t.Errorf("Concurrent send failed: %v", err)
	}

	// Get actual chatID from one of the messages
	var actualChatID string
	for msg := range messages {
		if actualChatID == "" {
			actualChatID = msg.ChatID
		}
	}

	// Check all messages saved
	var count int64
	err := testDB.QueryRow(ctx, "SELECT COUNT(*) FROM messages WHERE chat_id = $1", actualChatID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, int64(10), count)

	// Check chats created only once
	err = testDB.QueryRow(ctx, "SELECT COUNT(*) FROM chats WHERE user_id = $1 OR user_id = $2", user1ID, user2ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count) // One for user1, one for user2
}

// Test 4: GetMessages with pagination
func TestGetMessages_Pagination_Integration(t *testing.T) {
	cleanupAll(t)
	ctx := context.Background()

	repo := repository.NewPostgresRepository(testDB)
	svc := service.NewMessageService(repo, testKafka, nil)

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	chatID := fmt.Sprintf("%s:%s", user1ID, user2ID)

	// Create 100 messages
	messageIDs := make([]string, 100)
	for i := 0; i < 100; i++ {
		messageIDs[i] = uuid.New().String()
		msg := &service.Message{
			ID:               messageIDs[i],
			ChatID:           chatID,
			SenderID:         user1ID,
			RecipientID:      user2ID,
			EncryptedContent: fmt.Sprintf("content-%d", i),
			MessageType:      "text",
			Status:           service.MessageStatusSent,
			CreatedAt:        time.Now().Add(time.Duration(i) * time.Second),
		}
		err := repo.CreateWithChats(ctx, msg)
		require.NoError(t, err)
	}

	// Get first page (50 messages)
	messages, err := svc.GetMessages(ctx, chatID, user1ID, 50, "")
	require.NoError(t, err)
	assert.Len(t, messages, 50)

	// Check order (newest first)
	assert.Equal(t, messageIDs[99], messages[0].ID)
	assert.Equal(t, messageIDs[50], messages[49].ID)

	// Get second page
	lastID := messages[49].ID
	messages, err = svc.GetMessages(ctx, chatID, user1ID, 50, lastID)
	require.NoError(t, err)
	assert.Len(t, messages, 50)
	assert.Equal(t, messageIDs[49], messages[0].ID)
	assert.Equal(t, messageIDs[0], messages[49].ID)
}

// Test 5: MarkAsRead updates batch of messages
func TestMarkAsRead_UpdatesBatch_Integration(t *testing.T) {
	cleanupAll(t)
	ctx := context.Background()

	repo := repository.NewPostgresRepository(testDB)
	svc := service.NewMessageService(repo, testKafka, nil)

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	chatID := fmt.Sprintf("%s:%s", user1ID, user2ID)

	// Create 5 messages
	messageIDs := make([]string, 5)
	for i := 0; i < 5; i++ {
		messageIDs[i] = uuid.New().String()
		msg := &service.Message{
			ID:          messageIDs[i],
			ChatID:      chatID,
			SenderID:    user1ID,
			RecipientID: user2ID,
			Status:      service.MessageStatusDelivered,
			CreatedAt:   time.Now().Add(time.Duration(i) * time.Second),
		}
		err := repo.CreateWithChats(ctx, msg)
		require.NoError(t, err)
	}

	consumer := createTestConsumer(t, "messages.sent")
	defer consumer.Close()

	// Mark as read up to msg-3
	err := svc.MarkAsRead(ctx, chatID, user2ID, messageIDs[3])
	require.NoError(t, err)

	// Check statuses
	messages, err := repo.GetByChatID(ctx, chatID, 10, "")
	require.NoError(t, err)

	// msg-0, msg-1, msg-2, msg-3 should be READ
	for i := 0; i <= 3; i++ {
		assert.Equal(t, service.MessageStatusRead, messages[4-i].Status, "msg-%d should be READ", i)
	}

	// msg-4 should remain DELIVERED
	assert.Equal(t, service.MessageStatusDelivered, messages[0].Status)

	// Check Kafka event
	select {
	case kafkaMsg := <-consumer.Messages():
		var event map[string]interface{}
		err = json.Unmarshal(kafkaMsg.Value, &event)
		require.NoError(t, err)
		assert.Equal(t, "message_read", event["type"])
		assert.Equal(t, chatID, event["chat_id"])
		assert.Equal(t, messageIDs[3], event["last_message_id"])

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout: Kafka read event not received")
	}
}

// Test 6: DeleteMessage removes from DB and publishes event
func TestDeleteMessage_Integration(t *testing.T) {
	cleanupAll(t)
	ctx := context.Background()

	repo := repository.NewPostgresRepository(testDB)
	svc := service.NewMessageService(repo, testKafka, nil)

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	msgID := uuid.New().String()

	// Create message
	msg := &service.Message{
		ID:          msgID,
		ChatID:      fmt.Sprintf("%s:%s", user1ID, user2ID),
		SenderID:    user1ID,
		RecipientID: user2ID,
		Status:      service.MessageStatusSent,
		CreatedAt:   time.Now(),
	}
	err := repo.CreateWithChats(ctx, msg)
	require.NoError(t, err)

	consumer := createTestConsumer(t, "messages.sent")
	defer consumer.Close()

	// Delete message
	err = svc.DeleteMessage(ctx, msgID, user1ID)
	require.NoError(t, err)

	// Check message deleted from DB
	_, err = repo.GetByID(ctx, msgID)
	assert.Error(t, err)

	// Check Kafka event
	select {
	case kafkaMsg := <-consumer.Messages():
		var event map[string]interface{}
		err = json.Unmarshal(kafkaMsg.Value, &event)
		require.NoError(t, err)
		assert.Equal(t, "message_deleted", event["type"])
		assert.Equal(t, msgID, event["message_id"])

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout: Kafka delete event not received")
	}
}

// Test 7: AlterMessage updates content and publishes event
func TestAlterMessage_Integration(t *testing.T) {
	cleanupAll(t)
	ctx := context.Background()

	repo := repository.NewPostgresRepository(testDB)
	svc := service.NewMessageService(repo, testKafka, nil)

	user1ID := uuid.New().String()
	user2ID := uuid.New().String()
	msgID := uuid.New().String()

	// Create message
	msg := &service.Message{
		ID:               msgID,
		ChatID:           fmt.Sprintf("%s:%s", user1ID, user2ID),
		SenderID:         user1ID,
		RecipientID:      user2ID,
		EncryptedContent: "old-content",
		Status:           service.MessageStatusSent,
		CreatedAt:        time.Now(),
	}
	err := repo.CreateWithChats(ctx, msg)
	require.NoError(t, err)

	consumer := createTestConsumer(t, "messages.sent")
	defer consumer.Close()

	// Alter message
	err = svc.AlterMessage(ctx, msgID, user1ID, "new-content")
	require.NoError(t, err)

	// Check content updated in DB
	saved, err := repo.GetByID(ctx, msgID)
	require.NoError(t, err)
	assert.Equal(t, "new-content", saved.EncryptedContent)

	// Check Kafka event
	select {
	case kafkaMsg := <-consumer.Messages():
		var event map[string]interface{}
		err = json.Unmarshal(kafkaMsg.Value, &event)
		require.NoError(t, err)
		assert.Equal(t, "message_altered", event["type"])
		assert.Equal(t, msgID, event["message_id"])
		assert.Equal(t, "new-content", event["new_content"])

	case <-time.After(5 * time.Second):
		t.Fatal("Timeout: Kafka alter event not received")
	}
}

// Mock broken Kafka producer for testing outbox
type BrokenKafkaProducer struct{}

func (b *BrokenKafkaProducer) PublishMessageSent(ctx context.Context, msg *service.Message) error {
	return fmt.Errorf("kafka unavailable")
}

func (b *BrokenKafkaProducer) PublishMessageAltered(ctx context.Context, msg *service.Message, newContent string) error {
	return fmt.Errorf("kafka unavailable")
}

func (b *BrokenKafkaProducer) PublishMessageDeleted(ctx context.Context, msg *service.Message) error {
	return fmt.Errorf("kafka unavailable")
}

func (b *BrokenKafkaProducer) PublishMessageRead(ctx context.Context, chatID, readerID, senderID, lastMessageID string) error {
	return fmt.Errorf("kafka unavailable")
}

func (b *BrokenKafkaProducer) Close() error {
	return nil
}
