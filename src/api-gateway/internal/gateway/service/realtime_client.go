package service

import (
	"context"
	"encoding/json"
	"fmt"

	"api-gateway/internal/cores/domain"

	realtimepb "github.com/666Stepan66612/ZeroMes/pkg/gen/realtimepb"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type RealtimeClientService struct {
	conn   *grpc.ClientConn
	client realtimepb.ConnectionServiceClient
	redis  *redis.Client
}

func NewRealtimeClient(addr string, redisClient *redis.Client) (*RealtimeClientService, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &RealtimeClientService{
		conn:   conn,
		client: realtimepb.NewConnectionServiceClient(conn),
		redis:  redisClient,
	}, nil
}

func (c *RealtimeClientService) Close() error {
	return c.conn.Close()
}

func (c *RealtimeClientService) Connect(ctx context.Context, userID string, send chan<- []byte) error {
	token, _ := ctx.Value(domain.AccessTokenKey).(string)
	md := metadata.Pairs("authorization", "Bearer "+token)
	ctx = metadata.NewOutgoingContext(ctx, md)

	stream, err := c.client.ConnectionStream(ctx)
	if err != nil {
		return err
	}

	if err := stream.Send(&realtimepb.ConnectionRequest{
		Payload: &realtimepb.ConnectionRequest_Register{
			Register: &realtimepb.RegisterClient{UserId: userID},
		},
	}); err != nil {
		return err
	}

	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				return
			}

			var msgType string
			var payload interface{}

			switch p := msg.Payload.(type) {
			case *realtimepb.ConnectionResponse_Status:
				msgType = "status"
				payload = map[string]interface{}{
					"user_id":   p.Status.UserId,
					"connected": p.Status.Connected,
				}
			case *realtimepb.ConnectionResponse_Message:
				// Try to parse content as JSON event
				var eventData map[string]interface{}
				if err := json.Unmarshal([]byte(p.Message.Content), &eventData); err == nil {
					// If content is valid JSON with type field, use it directly
					if eventType, ok := eventData["type"].(string); ok {
						msgType = eventType
						if eventPayload, ok := eventData["payload"].(map[string]interface{}); ok {
							payload = eventPayload
						} else {
							payload = eventData
						}
					} else {
						// Fallback to new_message
						msgType = "new_message"
						payload = map[string]interface{}{
							"message_id": p.Message.MessageId,
							"sender_id":  p.Message.SenderId,
							"content":    p.Message.Content,
							"timestamp":  p.Message.Timestamp,
						}
					}
				} else {
					// Not JSON, treat as regular message
					msgType = "new_message"
					payload = map[string]interface{}{
						"message_id": p.Message.MessageId,
						"sender_id":  p.Message.SenderId,
						"content":    p.Message.Content,
						"timestamp":  p.Message.Timestamp,
					}
				}
			default:
				continue
			}

			data, err := json.Marshal(map[string]interface{}{
				"type":    msgType,
				"payload": payload,
			})
			if err != nil {
				return
			}
			select {
			case send <- data:
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (c *RealtimeClientService) CheckOnlineStatus(ctx context.Context, userID string) (bool, error) {
	key := fmt.Sprintf("user:%s:online", userID)
	exists, err := c.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}
