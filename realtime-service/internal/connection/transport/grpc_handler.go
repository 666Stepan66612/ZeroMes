package transport

import (
	"context"

	apperrors "realtime-service/internal/cores/errors"
	pb "github.com/666Stepan66612/ZeroMes/pkg/gen/realtimepb"
    "realtime-service/internal/connection/service"
)

type ConnectionHandler struct {
	pb.UnimplementedConnectionServiceServer
	manager service.ConnectionManager
}

func NewConncetionHandler(manager service.ConnectionManager) *ConnectionHandler {
	return &ConnectionHandler{
		manager: manager,
	}
}

func (h *ConnectionHandler) ConnecionStream(stream pb.ConnectionService_ConnectionStreamServer) error {
	msg, err := stream.Recv()
	if err != nil {
		return err
	}

	reg, ok := msg.Payload.(*pb.ConnectionRequest_Register)
	if !ok {
		return apperrors.ErrUnexpectedMessage
	}

	userID := reg.Register.UserId

	if err := h.manager.RegisterConnection(stream.Context(), userID, stream); err != nil {
		return err
	}
	defer h.manager.UnregisterConnection(context.Background(), userID)

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