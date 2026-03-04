package transport

import (
	"context"
	"errors"

    apperrors "message-service/internal/cores/errors"
	pb "github.com/666Stepan66612/ZeroMes/pkg/gen/messagepb"
	"message-service/internal/messaging/service"
    
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GRPCHandler struct {
    pb.UnimplementedMessageServiceServer
	messageService service.MessageService
}

func NewGRPCHandler(messageService service.MessageService) *GRPCHandler {
	return &GRPCHandler{
		messageService: messageService,
	}
}

func (h *GRPCHandler) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	msg, err := h.messageService.SendMessage(
		ctx,
		req.ChatId,
		req.SenderId,
		req.RecipientId,
		req.EncryptedContent,
		req.MessageType,
	)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.SendMessageResponse{
		Message: &pb.Message{
			Id:               msg.ID,
			ChatId:           msg.ChatID,
			SenderId:         msg.SenderID,
			RecipientId:      msg.RecipientID,
			EncryptedContent: msg.EncryptedContent,
			MessageType:      msg.MessageType,
			CreatedAt:        timestamppb.New(msg.CreatedAt),
			Status:           pb.MessageStatus(msg.Status),
		},
	}, nil
}

func (h *GRPCHandler) GetMessages(ctx context.Context, req *pb.GetMessagesRequest) (*pb.GetMessagesResponse, error) {
    messages, err := h.messageService.GetMessages(
        ctx,
        req.ChatId,
        int(req.Limit),
        req.LastMessageId,
    )
    if err != nil {
        return nil, toGRPCError(err)
    }

    pbMessages := make([]*pb.Message, 0, len(messages))
    for _, msg := range messages {
        pbMessages = append(pbMessages, &pb.Message{
            Id:               msg.ID,
            ChatId:           msg.ChatID,
            SenderId:         msg.SenderID,
            RecipientId:      msg.RecipientID,
            EncryptedContent: msg.EncryptedContent,
            MessageType:      msg.MessageType,
            CreatedAt:        timestamppb.New(msg.CreatedAt),
            Status:           pb.MessageStatus(msg.Status),
        })
    }

    var nextMessageID string
    var hasMore bool
    if len(messages) > 0 {
        nextMessageID = messages[len(messages)-1].ID
        hasMore = len(messages) == int(req.Limit)
    }

    return &pb.GetMessagesResponse{
        Messages:      pbMessages,
        NextMessageId: nextMessageID,
        HasMore:       hasMore,
    }, nil
}

func (h *GRPCHandler) MarkAsRead(ctx context.Context, req *pb.MarkAsReadRequest) (*pb.MarkAsReadResponse, error) {
    err := h.messageService.MarkAsRead(
        ctx,
        req.ChatId,
        req.UserId,
        req.LastMessageId,
    )
    if err != nil {
        return nil, toGRPCError(err)
	}

	return &pb.MarkAsReadResponse{
		Success: true,
	}, nil
}

func (h *GRPCHandler) AlterMessage(ctx context.Context, req *pb.AlterMessageRequest) (*pb.AlterMessageResponse, error) {
    err := h.messageService.AlterMessage(ctx, req.MessageId, req.UserId, req.NewContent)
    if err != nil {
        return nil, toGRPCError(err)
    }

    return &pb.AlterMessageResponse{
        Success: true,
    }, nil
}

func (h *GRPCHandler) DeleteMessage(ctx context.Context, req *pb.DeleteMessageRequest) (*pb.DeleteMessageResponse, error) {
    err := h.messageService.DeleteMessage(ctx, req.MessageId, req.UserId)
    if err != nil {
        return nil, toGRPCError(err)
    }

    return &pb.DeleteMessageResponse{
        Success: true,
    }, nil
}

func toGRPCError(err error) error {
    switch {
    case errors.Is(err, apperrors.ErrNotFound):
        return status.Error(codes.NotFound, err.Error())
    case errors.Is(err, apperrors.ErrNotYourMessage):
        return status.Error(codes.PermissionDenied, err.Error())
    case errors.Is(err, apperrors.ErrInvalidInput):
        return status.Error(codes.InvalidArgument, err.Error())
    default:
        return status.Error(codes.Internal, err.Error())
    }
}