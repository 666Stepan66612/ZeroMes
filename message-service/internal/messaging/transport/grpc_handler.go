package transport

import (
	"context"
	"errors"
	"log/slog"

	apperrors "message-service/internal/cores/errors"
	"message-service/internal/messaging/service"

	pb "github.com/666Stepan66612/ZeroMes/pkg/gen/messagepb"

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
	slog.Info("gRPC SendMessage received", "chat_id", req.ChatId, "sender_id", req.SenderId, "recipient_id", req.RecipientId)
	msg, err := h.messageService.SendMessage(
		ctx,
		req.ChatId,
		req.SenderId,
		req.RecipientId,
		req.EncryptedContent,
		req.MessageType,
	)
	if err != nil {
		slog.Error("SendMessage service failed", "err", err)
		return nil, toGRPCError(err)
	}

	slog.Info("SendMessage success", "message_id", msg.ID)
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
        req.UserId,
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

func (h *GRPCHandler) GetChats(ctx context.Context, req *pb.GetChatsRequest) (*pb.GetChatsResponse, error) {
	chats, err := h.messageService.GetChats(ctx, req.UserId)
	if err != nil {
		return nil, toGRPCError(err)
	}

	pbChats := make([]*pb.Chat, 0, len(chats))
	for _, cht := range chats {
		pbChats = append(pbChats, &pb.Chat{
			Id:            cht.ChatID,
			UserId:        cht.UserID,
			CompanionId:   cht.CompanionID,
			CreatedAt:     timestamppb.New(cht.CreatedAt),
			LastMessageAt: timestamppb.New(cht.LastMessageAt),
			EncryptedKey:  cht.EncryptedKey,
			KeyIv:         cht.KeyIV, 
		})
	}

	return &pb.GetChatsResponse{
		Chats: pbChats,
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
