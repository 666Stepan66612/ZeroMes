package kafka

import (
	"context"

    "github.com/segmentio/kafka-go"
    pb "github.com/666Stepan66612/ZeroMes/pkg/gen/messagepb"
    "message-service/internal/messaging/service"
    "google.golang.org/protobuf/proto"
    "google.golang.org/protobuf/types/known/timestamppb"
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
			Compression: kafka.Snappy,
			RequiredAcks: kafka.RequireAll,
			Async: false,
			AllowAutoTopicCreation: true,  
		},
	}
}

func (p *Producer) PublishMessageSent(ctx context.Context, msg *service.Message) error {
	pbMsg := &pb.Message{
		Id: msg.ID,
		ChatId: msg.ChatID,
		SenderId: msg.SenderID,
		RecipientId: msg.RecipientID,
		EncryptedContent: msg.EncryptedContent,
		MessageType: msg.MessageType,
		CreatedAt: timestamppb.New(msg.CreatedAt),
		Status: pb.MessageStatus(msg.Status),
	}

	data, err := proto.Marshal(pbMsg)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key: []byte(msg.ChatID),
		Value: data,
		Headers: []kafka.Header{
			{Key: "content-type", Value: []byte("application/protobuf")},
			{Key: "scheme-version", Value: []byte("1.0")},
		},
	})	
}

func (p *Producer) Close() error {
    return p.writer.Close()
}