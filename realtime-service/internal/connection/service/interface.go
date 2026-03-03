package service

import (
	"context"
	"time"

	domain "realtime-service/internal/cores/domain"
	pb "realtime-service/gen/realtimepb"
)

type ConnectionManager interface {
	RegisterConnection(ctx context.Context, userID string, stream pb.ConnectionService_ConnectionStreamServer) error
	UnregisterConnection(ctx context.Context, userID string) error
	GetConnection(userID string) (pb.ConnectionService_ConnectionStreamServer, error)
	GetAllUserIDs() []string
	GetConnectionCount() int
	CloseAll(ctx context.Context) error
	DeliverMessage(ctx context.Context, msg *domain.Message) error
}

type PresenceRepository interface {
	SetOnline(ctx context.Context, userID string, instanceID string, ttl time.Duration) error
	SetOffline(ctx context.Context, userID string) error
	IsOnline(ctx context.Context, userID string) (bool, error)
	GetUserInstance(ctx context.Context, userID string) (string, error)
	GetOnlineCount(ctx context.Context) (int64, error)
	ExtendTTL(ctx context.Context, userID string, ttl time.Duration) error
}