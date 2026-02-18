package kafka

import (
    "context"
    "encoding/json"

    "github.com/segmentio/kafka-go"
    "message-service/internal/messaging/service"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr: kafka.TCP(brokers...),
			Topic: topic,
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (p *Producer) PublishMessageSent(ctx context.Context, msg *service.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key: []byte(msg.ChatID),
		Value: data,
	})	
}

func (p *Producer) Close() error {
    return p.writer.Close()
}
