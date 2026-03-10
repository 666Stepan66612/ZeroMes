package service

import (
	"context"
	"encoding/json"

	realtimepb "github.com/666Stepan66612/ZeroMes/pkg/gen/realtimepb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RealtimeClientService struct {
	client realtimepb.ConnectionServiceClient
}

func NewRealtimeClient(addr string) (*RealtimeClientService, error){
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &RealtimeClientService{
		client: realtimepb.NewConnectionServiceClient(conn),
	}, nil
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
			data, err := json.Marshal(msg)
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