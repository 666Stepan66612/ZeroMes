package service

import (
	"api-gateway/internal/cores/domain"
	"context"
	"fmt"
	"log/slog"
)

type SagaOrchestrator struct {
	authClient    AuthClient
	messageClient MessageClient
}

func NewSagaOrchestrator(authClient AuthClient, messageClient MessageClient) *SagaOrchestrator {
	return &SagaOrchestrator{
		authClient:    authClient,
		messageClient: messageClient,
	}
}

func (s *SagaOrchestrator) ChangePassword(ctx context.Context, req *domain.ChangePasswordRequest) (*domain.ChangePasswordResponse, error) {
	slog.Info("saga: starting change password", "login", req.Login)

	userID, err := s.authClient.ChangePassword(ctx, req.Login, req.OldAuthHash, req.NewAuthHash, req.NewPublicKey)
	if err != nil {
		slog.Error("saga: auth-service failed", "error", err)
		return nil, fmt.Errorf("failed to change password in auth service: %w", err)
	}
	slog.Info("saga: auth-service success", "user_id", userID)

	updatedCount := 0
	if len(req.ChatKeys) > 0 {
		updatedCount, err = s.messageClient.UpdateChatKeys(ctx, userID, req.ChatKeys)
		if err != nil {
			slog.Error("saga: message-service failed", "error", err, "user_id", userID)
			return nil, fmt.Errorf("password changed but failed to update chat keys: %w", err)
		}
		slog.Info("saga: message-service success", "updated_chats", updatedCount)
	}

	slog.Info("saga: change password completed", "user_id", userID, "updated_chats", updatedCount)
	return &domain.ChangePasswordResponse{
		Success:      true,
		UpdatedChats: updatedCount,
		Message:      "password changed successfully",
	}, nil
}
