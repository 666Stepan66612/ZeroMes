package service

import (
	"context"
	
	domain "realtime-service/internal/cores/domain"
	messagepb "github.com/666Stepan66612/ZeroMes/pkg/gen/messagepb"

    "github.com/segmentio/kafka-go"
    "google.golang.org/protobuf/proto"
)

type KafkaConsumer struct {
	reader *kafka.Reader
	manager ConnectionManager
}

func NewKafkaConsumer(brokers []string, topic, groupID string, manager ConnectionManager) *KafkaConsumer {
	return &KafkaConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			Topic: topic,
			GroupID: groupID,
		}),
		manager: manager,
	}
}

func (c *KafkaConsumer) Start(ctx context.Context) error {
	for {
		kafkaMsg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			continue
		}

		var protoMsg messagepb.Message
		if err := proto.Unmarshal(kafkaMsg.Value, &protoMsg); err != nil {
			continue
		}

		msg := &domain.Message{
			MessageID: protoMsg.Id,
			SenderID: protoMsg.SenderId,
			RecipientID: protoMsg.RecipientId,
			Content: protoMsg.Content,
			Timestamp:   protoMsg.CreatedAt.AsTime().Unix(),
		}

		if err := c.manager.DeliverMessage(ctx, msg); err != nil {
			continue
		}
	}
}

func (c *KafkaConsumer) Close() error {
	return c.reader.Close()
}
