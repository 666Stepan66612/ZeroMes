package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	domain "realtime-service/internal/cores/domain"
	apperrors "realtime-service/internal/cores/errors"

	pb "github.com/666Stepan66612/ZeroMes/pkg/gen/realtimepb"
)

type Hub struct {
	streams          map[string]pb.ConnectionService_ConnectionStreamServer
	repo             PresenceRepository
	groupProvider    GroupMemberProvider
	instanceID       string
	mu               sync.RWMutex
	cancelFuncs      map[string]context.CancelFunc
	cancelMu         sync.Mutex
}

func NewHub(repo PresenceRepository, instanceID string, groupProvider GroupMemberProvider) *Hub {
	return &Hub{
		streams:       make(map[string]pb.ConnectionService_ConnectionStreamServer),
		cancelFuncs:   make(map[string]context.CancelFunc),
		repo:          repo,
		groupProvider: groupProvider,
		instanceID:    instanceID,
	}
}

func (h *Hub) RegisterConnection(ctx context.Context, userID string, stream pb.ConnectionService_ConnectionStreamServer) error {
	err := h.repo.SetOnline(ctx, userID, h.instanceID, 300*time.Second)
	if err != nil {
		return err
	}

	h.mu.Lock()
	h.streams[userID] = stream
	h.mu.Unlock()

	heartbeatCtx, cancel := context.WithCancel(context.Background())
	h.cancelMu.Lock()
	h.cancelFuncs[userID] = cancel
	h.cancelMu.Unlock()

	go h.heartbeat(heartbeatCtx, userID)

	return nil
}

func (h *Hub) UnregisterConnection(ctx context.Context, userID string) error {
	h.cancelMu.Lock()
	if cancel, exists := h.cancelFuncs[userID]; exists {
		cancel()
		delete(h.cancelFuncs, userID)
	}
	h.cancelMu.Unlock()

	h.mu.Lock()
	delete(h.streams, userID)
	h.mu.Unlock()

	return h.repo.SetOffline(ctx, userID)
}

func (h *Hub) GetStream(userID string) (pb.ConnectionService_ConnectionStreamServer, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stream, exists := h.streams[userID]
	if !exists {
		return nil, apperrors.ErrConNotFound
	}

	return stream, nil
}

func (h *Hub) GetAllUserIDs() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	userIDs := make([]string, 0, len(h.streams))
	for userID := range h.streams {
		userIDs = append(userIDs, userID)
	}
	return userIDs
}

func (h *Hub) GetConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.streams)
}

func (h *Hub) CloseAll(ctx context.Context) error {
	h.cancelMu.Lock()
	for _, cancel := range h.cancelFuncs {
		cancel()
	}
	h.cancelFuncs = make(map[string]context.CancelFunc)
	h.cancelMu.Unlock()

	h.mu.Lock()
	defer h.mu.Unlock()

	for userID := range h.streams {
		err := h.repo.SetOffline(ctx, userID)
		if err != nil {
			continue
		}
	}

	h.streams = make(map[string]pb.ConnectionService_ConnectionStreamServer)
	return nil
}

func (h *Hub) heartbeat(ctx context.Context, userID string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.mu.RLock()
			_, exists := h.streams[userID]
			h.mu.RUnlock()

			if !exists {
				return
			}

			h.repo.ExtendTTL(ctx, userID, 5*time.Minute)

		case <-ctx.Done():
			return
		}
	}
}

func (h *Hub) DeliverMessage(ctx context.Context, msg *domain.Message) error {
	if msg.GroupID != "" {
		return h.deliverGroupMessage(ctx, msg)
	}
	return h.deliverDirectMessage(ctx, msg)
}

func (h *Hub) deliverDirectMessage(ctx context.Context, msg *domain.Message) error {
	stream, err := h.GetStream(msg.RecipientID)
	if err != nil {
		return nil
	}

	payload := map[string]interface{}{
		"type": "new_message",
		"payload": map[string]interface{}{
			"message_id":        msg.MessageID,
			"chat_id":           msg.ChatID,
			"sender_id":         msg.SenderID,
			"encrypted_content": msg.Content,
			"timestamp":         msg.Timestamp,
		},
	}

	jsonData, _ := json.Marshal(payload)

	return stream.Send(&pb.ConnectionResponse{
		Payload: &pb.ConnectionResponse_Message{
			Message: &pb.IncomingMessage{
				MessageId: msg.MessageID,
				SenderId:  msg.SenderID,
				Content:   string(jsonData),
			},
		},
	})
}

func (h *Hub) deliverGroupMessage(ctx context.Context, msg *domain.Message) error {
	memberIDs, err := h.groupProvider.GetActiveGroupMemberIDs(ctx, msg.GroupID)
	if err != nil {
		slog.Error("failed to get group members", "group_id", msg.GroupID, "err", err)
		return err
	}

	payload := map[string]interface{}{
		"type": "new_message",
		"payload": map[string]interface{}{
			"message_id":        msg.MessageID,
			"chat_id":           msg.ChatID,
			"sender_id":         msg.SenderID,
			"encrypted_content": msg.Content,
			"timestamp":         msg.Timestamp,
			"group_id":          msg.GroupID,
			"key_version":       msg.KeyVersion,
		},
	}

	jsonData, _ := json.Marshal(payload)

	sent := 0
	for _, userID := range memberIDs {
		if userID == msg.SenderID {
			continue
		}

		stream, err := h.GetStream(userID)
		if err != nil {
			continue
		}

		if err := stream.Send(&pb.ConnectionResponse{
			Payload: &pb.ConnectionResponse_Message{
				Message: &pb.IncomingMessage{
					MessageId: msg.MessageID,
					SenderId:  msg.SenderID,
					ChatId:    msg.ChatID,
					GroupId:   msg.GroupID,
					Content:   string(jsonData),
				},
			},
		}); err != nil {
			slog.Warn("failed to send group message", "user_id", userID, "err", err)
			continue
		}
		sent++
	}

	slog.Info("group message delivered", "group_id", msg.GroupID, "total_members", len(memberIDs), "sent", sent)
	return nil
}
