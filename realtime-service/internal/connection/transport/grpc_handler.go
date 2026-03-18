package transport

import (
	"context"
	"time"

	"realtime-service/internal/connection/service"

	pb "github.com/666Stepan66612/ZeroMes/pkg/gen/realtimepb"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type ConnectionHandler struct {
	pb.UnimplementedConnectionServiceServer
	manager service.ConnectionManager
	jwtSecret []byte
}

func NewConnectionHandler(manager service.ConnectionManager, jwtSecret string) *ConnectionHandler {
	return &ConnectionHandler{
		manager: manager,
		jwtSecret: []byte(jwtSecret),
	}
}

func (h *ConnectionHandler) ConnectionStream(stream pb.ConnectionService_ConnectionStreamServer) error {
	claims, err := h.authenticate(stream.Context(), stream)
	if err != nil {
		return err
	}

	userID := claims.UserID
	_, err = stream.Recv()
	if err != nil {
		return err
	}

	if err := h.manager.RegisterConnection(stream.Context(), userID, stream); err != nil {
		return status.Error(codes.Internal, "conncetion failed")
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
        if err != nil {
            return err
        }

        switch msg.Payload.(type) {
        case *pb.ConnectionRequest_Disconnect:
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

func (h *ConnectionHandler) authenticate(ctx context.Context, stream pb.ConnectionService_ConnectionStreamServer) (*Claims, error) {
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

	return h.validateToken(tokenStr)
}