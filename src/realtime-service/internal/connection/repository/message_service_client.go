package repository

import (
	"context"
	"log/slog"

	messagepb "github.com/666Stepan66612/ZeroMes/pkg/gen/messagepb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type MessageServiceClient struct {
	conn   *grpc.ClientConn
	client messagepb.MessageServiceClient
}

func NewMessageServiceClient(address string) (*MessageServiceClient, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &MessageServiceClient{
		conn:   conn,
		client: messagepb.NewMessageServiceClient(conn),
	}, nil
}

func (c *MessageServiceClient) GetActiveGroupMemberIDs(ctx context.Context, groupID string) ([]string, error) {
	resp, err := c.client.GetGroupMembers(ctx, &messagepb.GetGroupMembersRequest{
		GroupId: groupID,
	})
	if err != nil {
		slog.Error("failed to get group members", "group_id", groupID, "err", err)
		return nil, err
	}

	ids := make([]string, 0, len(resp.Members))
	for _, m := range resp.Members {
		ids = append(ids, m.UserId)
	}
	return ids, nil
}

func (c *MessageServiceClient) Close() error {
	return c.conn.Close()
}
