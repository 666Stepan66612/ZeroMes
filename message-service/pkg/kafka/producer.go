package kafka

import (
	"context"
	"encoding/json"
	"time"

	"message-service/internal/messaging/service"

	pb "github.com/666Stepan66612/ZeroMes/pkg/gen/messagepb"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  topic,
			Balancer:               &kafka.LeastBytes{},
			Compression:            kafka.Lz4,
			RequiredAcks:           kafka.RequireOne,
			Async:                  false,
			AllowAutoTopicCreation: true,
			BatchSize:              100,
			BatchTimeout:           10 * time.Millisecond,
			ReadTimeout:            10 * time.Second,
			WriteTimeout:           10 * time.Second,
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

func (p *Producer) PublishMessageAltered(ctx context.Context, msg *service.Message, newContent string) error {
    data, err := json.Marshal(map[string]interface{}{
        "type":         "message_altered",
        "message_id":   msg.ID,
        "chat_id":      msg.ChatID,
        "sender_id":    msg.SenderID,
        "recipient_id": msg.RecipientID,
        "new_content":  newContent,
    })
    if err != nil {
        return err
    }
    return p.writer.WriteMessages(ctx, kafka.Message{
        Key:   []byte(msg.ChatID),
        Value: data,
    })
}

func (p *Producer) PublishMessageDeleted(ctx context.Context, msg *service.Message) error {
    data, err := json.Marshal(map[string]interface{}{
        "type":         "message_deleted",
        "message_id":   msg.ID,
        "chat_id":      msg.ChatID,
        "sender_id":    msg.SenderID,
        "recipient_id": msg.RecipientID,
    })
    if err != nil {
        return err
    }
    return p.writer.WriteMessages(ctx, kafka.Message{
        Key:   []byte(msg.ChatID),
        Value: data,
    })
}

func (p *Producer) PublishMessageRead(ctx context.Context, chatID, readerID, senderID, lastMessageID string) error {
    data, err := json.Marshal(map[string]interface{}{
        "type":            "message_read",
        "chat_id":         chatID,
        "last_message_id": lastMessageID,
        "sender_id":       readerID,
        "recipient_id":    senderID,
    })
    if err != nil {
        return err
    }
    return p.writer.WriteMessages(ctx, kafka.Message{
        Key:   []byte(chatID),
        Value: data,
    })
}

func (p *Producer) Close() error {
    return p.writer.Close()
}
