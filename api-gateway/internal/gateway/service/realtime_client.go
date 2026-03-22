package service

import (
	"context"
	"encoding/json"

	realtimepb "github.com/666Stepan66612/ZeroMes/pkg/gen/realtimepb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RealtimeClientService struct {
	conn *grpc.ClientConn
	client realtimepb.ConnectionServiceClient
}

func NewRealtimeClient(addr string) (*RealtimeClientService, error){
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &RealtimeClientService{
		conn: conn,
		client: realtimepb.NewConnectionServiceClient(conn),
	}, nil
}

func (c *RealtimeClientService) Close() error {
	return c.conn.Close()
}

func (c *RealtimeClientService) Connect(ctx context.Context, userID string, send chan<- []byte) error{
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
					"user_id": p.Status.UserId,
					"connected": p.Status.Connected,
				}
			case *realtimepb.ConnectionResponse_Message:
				msgType = "new_message"
				payload = map[string]interface{}{
					"message_id": p.Message.MessageId,
					"sender_id": p.Message.SenderId,
					"content": p.Message.Content,
					"timestamp": p.Message.Timestamp,
				}
			default:
				continue
			}

			data, err := json.Marshal(map[string]interface{}{
				"type": msgType,
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