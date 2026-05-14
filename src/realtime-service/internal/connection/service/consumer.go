package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	domain "realtime-service/internal/cores/domain"

	messagepb "github.com/666Stepan66612/ZeroMes/src/pkg/gen/messagepb"
	realtimepb "github.com/666Stepan66612/ZeroMes/src/pkg/gen/realtimepb"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

type KafkaConsumer struct {
	reader  *kafka.Reader
	manager ConnectionManager
}

func NewKafkaConsumer(brokers []string, topic, groupID string, manager ConnectionManager) *KafkaConsumer {
	return &KafkaConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:          brokers,
			Topic:            topic,
			GroupID:          groupID,
			StartOffset:      kafka.LastOffset,
			MinBytes:         1,
			MaxBytes:         10e6,
			MaxWait:          500 * time.Millisecond,
			CommitInterval:   time.Second,
			ReadBatchTimeout: 10 * time.Second,
		}),
		manager: manager,
	}
}

func (c *KafkaConsumer) Start(ctx context.Context) error {
	slog.Info("Kafka consumer started",
		"brokers", c.reader.Config().Brokers,
		"topic", c.reader.Config().Topic,
		"groupID", c.reader.Config().GroupID)
	slog.Info("Waiting for messages from Kafka...")

	for {
		slog.Info("Attempting to read message from Kafka...")
		kafkaMsg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				slog.Info("Context cancelled, stopping consumer")
				return nil
			}
			slog.Error("kafka read error", "err", err)
			continue
		}

		slog.Info("Kafka message received", "topic", kafkaMsg.Topic, "partition", kafkaMsg.Partition, "offset", kafkaMsg.Offset)

		contentType := ""
		for _, h := range kafkaMsg.Headers {
			if h.Key == "content-type" {
				contentType = string(h.Value)
				break
			}
		}

		if contentType == "application/protobuf" {
			var protoMsg messagepb.Message
			if err := proto.Unmarshal(kafkaMsg.Value, &protoMsg); err != nil {
				slog.Error("failed to unmarshal proto", "err", err)
				continue
			}

			slog.Info("new message received", "msg_id", protoMsg.Id)

			msg := &domain.Message{
				MessageID:   protoMsg.Id,
				ChatID:      protoMsg.ChatId,
				SenderID:    protoMsg.SenderId,
				RecipientID: protoMsg.RecipientId,
				Content:     protoMsg.EncryptedContent,
				Timestamp:   protoMsg.CreatedAt.AsTime().Format(time.RFC3339),
			}

			if err := c.manager.DeliverMessage(ctx, msg); err != nil {
				slog.Warn("failed to deliver message", "msg_id", msg.MessageID, "err", err)
			}

		} else {
			var event struct {
				Type          string `json:"type"`
				MessageID     string `json:"message_id"`
				ChatID        string `json:"chat_id"`
				SenderID      string `json:"sender_id"`
				RecipientID   string `json:"recipient_id"`
				NewContent    string `json:"new_content,omitempty"`
				LastMessageID string `json:"last_message_id,omitempty"`
			}
			if err := json.Unmarshal(kafkaMsg.Value, &event); err != nil {
				slog.Error("failed to unmarshal event", "err", err)
				continue
			}

			slog.Info("event received", "type", event.Type, "msg_id", event.MessageID)

			data, _ := json.Marshal(map[string]interface{}{
				"type":    event.Type,
				"payload": event,
			})

			recipientStream, err := c.manager.GetStream(event.RecipientID)
			if err == nil {
				if err := recipientStream.Send(&realtimepb.ConnectionResponse{
					Payload: &realtimepb.ConnectionResponse_Message{
						Message: &realtimepb.IncomingMessage{
							MessageId: event.MessageID,
							SenderId:  event.SenderID,
							Content:   string(data),
						},
					},
				}); err != nil {
					slog.Warn("failed to send event to recipient", "recipient_id", event.RecipientID, "err", err)
				}
			} else {
				slog.Debug("recipient offline", "recipient_id", event.RecipientID)
			}

			if event.Type == "message_read" || event.Type == "message_altered" || event.Type == "message_deleted" {
				senderStream, err := c.manager.GetStream(event.SenderID)
				if err == nil {
					if err := senderStream.Send(&realtimepb.ConnectionResponse{
						Payload: &realtimepb.ConnectionResponse_Message{
							Message: &realtimepb.IncomingMessage{
								MessageId: event.MessageID,
								SenderId:  event.SenderID,
								Content:   string(data),
							},
						},
					}); err != nil {
						slog.Warn("failed to send event to sender", "sender_id", event.SenderID, "err", err)
					}
				} else {
					slog.Debug("sender offline", "sender_id", event.SenderID)
				}
			}
		}
	}
}

func (c *KafkaConsumer) Close() error {
	return c.reader.Close()
}
