package outboxworker

import (
    "context"
    "encoding/json"
    "log/slog"
    "time"
    
    "message-service/internal/messaging/service"
)

type OutboxWorker struct {
    outboxRepo    service.OutboxRepository
    kafkaProducer service.KafkaProducer
    interval      time.Duration
    maxRetries    int
}

func NewOutboxWorker(outboxRepo service.OutboxRepository, kafkaProducer service.KafkaProducer) *OutboxWorker {
    return &OutboxWorker{
        outboxRepo:    outboxRepo,
        kafkaProducer: kafkaProducer,
        interval:      5 * time.Second,
        maxRetries:    5,
    }
}

func (w *OutboxWorker) Start(ctx context.Context) {
    ticker := time.NewTicker(w.interval)
    defer ticker.Stop()
    
    slog.Info("Outbox worker started")
    
    for {
        select {
        case <-ticker.C:
            w.processEvents(ctx)
        case <-ctx.Done():
            slog.Info("Outbox worker stopped")
            return
        }
    }
}

func (w *OutboxWorker) processEvents(ctx context.Context) {
    events, err := w.outboxRepo.GetPendingEvents(ctx, 100)
    if err != nil {
        slog.Error("Failed to get pending events", "error", err)
        return
    }
    
    if len(events) == 0 {
        return
    }
    
    slog.Info("Processing outbox events", "count", len(events))
    
    for _, event := range events {
        if event.RetryCount >= w.maxRetries {
            w.outboxRepo.MarkEventFailed(ctx, event.ID, "max retries exceeded")
            continue
        }
        
        if err := w.publishEvent(ctx, event); err != nil {
            slog.Warn("Failed to publish event", "event_id", event.ID, "error", err)
            w.outboxRepo.IncrementRetryCount(ctx, event.ID)
        } else {
            w.outboxRepo.MarkEventProcessed(ctx, event.ID)
            slog.Info("Event published successfully", "event_id", event.ID)
        }
    }
}

func (w *OutboxWorker) publishEvent(ctx context.Context, event *service.OutboxEvent) error {
    switch event.EventType {
    case "message_sent":
        var msg service.Message
        if err := json.Unmarshal(event.Payload, &msg); err != nil {
            return err
        }
        return w.kafkaProducer.PublishMessageSent(ctx, &msg)
        
    case "message_deleted":
        var msg service.Message
        if err := json.Unmarshal(event.Payload, &msg); err != nil {
            return err
        }
        return w.kafkaProducer.PublishMessageDeleted(ctx, &msg)
        
    case "message_altered":
        var data struct {
            Message    service.Message `json:"message"`
            NewContent string          `json:"new_content"`
        }
        if err := json.Unmarshal(event.Payload, &data); err != nil {
            return err
        }
        return w.kafkaProducer.PublishMessageAltered(ctx, &data.Message, data.NewContent)
        
    case "message_read":
        var data map[string]string
        if err := json.Unmarshal(event.Payload, &data); err != nil {
            return err
        }
        return w.kafkaProducer.PublishMessageRead(ctx, 
            data["chat_id"], data["user_id"], data["sender_id"], data["last_message_id"])
    }
    
    return nil
}