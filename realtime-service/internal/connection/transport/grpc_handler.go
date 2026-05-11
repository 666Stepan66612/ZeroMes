package transport

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"time"

	"realtime-service/internal/connection/service"

	pb "github.com/666Stepan66612/ZeroMes/pkg/gen/realtimepb"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type ConnectionHandler struct {
	pb.UnimplementedConnectionServiceServer
	manager   service.ConnectionManager
	jwtSecret []byte
	redis     *redis.Client
}

func NewConnectionHandler(manager service.ConnectionManager, jwtSecret string, redisClient *redis.Client) *ConnectionHandler {
	return &ConnectionHandler{
		manager:   manager,
		jwtSecret: []byte(jwtSecret),
		redis:     redisClient,
	}
}

func (h *ConnectionHandler) ConnectionStream(stream pb.ConnectionService_ConnectionStreamServer) error {
	claims, err := h.authenticate(stream.Context())
	if err != nil {
		slog.Warn("authentication failed", "err", err)
		return err
	}

	userID := claims.UserID
	_, err = stream.Recv()
	if err != nil {
		slog.Debug("stream recv error during setup", "user_id", userID, "err", err)
		return err
	}

	if err := h.manager.RegisterConnection(stream.Context(), userID, stream); err != nil {
		slog.Error("failed to register connection", "user_id", userID, "err", err)
		return status.Error(codes.Internal, "connection failed")
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		h.manager.UnregisterConnection(cleanupCtx, userID)
	}()

	if err := stream.Send(&pb.ConnectionResponse{
		Payload: &pb.ConnectionResponse_Status{
			Status: &pb.ConnectionStatus{
				UserId:    userID,
				Connected: true,
			},
		},
	}); err != nil {
		return err
	}

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			slog.Debug("client closed stream", "user_id", userID)
			return err
		}
		if err != nil {
			slog.Warn("stream receive error", "user_id", userID, "err", err)
			return status.Error(codes.Internal, "connection error")
		}

		switch msg.Payload.(type) {
		case *pb.ConnectionRequest_Disconnect:
			slog.Debug("disconnect requested", "user_id", userID)
			return nil
		}
	}
}

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func (h *ConnectionHandler) validateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, status.Error(codes.Unauthenticated, "unexpected signing method")
		}
		return h.jwtSecret, nil
	})
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, status.Error(codes.Unauthenticated, "invalid token claims")
	}

	return claims, nil
}

func (h *ConnectionHandler) authenticate(ctx context.Context) (*Claims, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	token := md.Get("authorization")
	if len(token) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing token")
	}

	tokenStr := token[0]
	if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
		tokenStr = tokenStr[7:]
	}

	hash := sha256.Sum256([]byte(tokenStr))
	tokenHash := hex.EncodeToString(hash[:])
	val, _ := h.redis.Get(ctx, "blacklist:"+tokenHash).Result()
	if val != "" {
		return nil, status.Error(codes.Unauthenticated, "token revoked")
	}

	return h.validateToken(tokenStr)
}
