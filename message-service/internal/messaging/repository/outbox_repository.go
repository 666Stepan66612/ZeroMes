package repository

import (
	"context"
	"time"

	"message-service/internal/messaging/service"

	"github.com/jackc/pgx/v5/pgxpool"
)

type outboxRepository struct {
	pool *pgxpool.Pool
}

func NewOutboxRepository(pool *pgxpool.Pool) service.OutboxRepository {
	return &outboxRepository{
		pool: pool,
	}
}

func (r *outboxRepository) SaveToOutbox(ctx context.Context, event *service.OutboxEvent) error {
	query := `
		INSERT INTO outbox_events (id, event_type, aggregate_id, payload, created_at, retry_count, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	
	_, err := r.pool.Exec(ctx, query,
		event.ID,
		event.EventType,
		event.AggregateID,
		event.Payload,
		event.CreatedAt,
		event.RetryCount,
		event.Status,
	)
	
	return err
}

func (r *outboxRepository) GetPendingEvents(ctx context.Context, limit int) ([]*service.OutboxEvent, error) {
	query := `
		SELECT id, event_type, aggregate_id, payload, created_at, processed_at, 
		       retry_count, last_error, status
		FROM outbox_events
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT $1
	`
	
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	events := make([]*service.OutboxEvent, 0)
	for rows.Next() {
		event := &service.OutboxEvent{}
		err := rows.Scan(
			&event.ID,
			&event.EventType,
			&event.AggregateID,
			&event.Payload,
			&event.CreatedAt,
			&event.ProcessedAt,
			&event.RetryCount,
			&event.LastError,
			&event.Status,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	
	return events, rows.Err()
}

func (r *outboxRepository) MarkEventProcessed(ctx context.Context, eventID string) error {
	query := `
		UPDATE outbox_events
		SET status = 'completed', processed_at = $1
		WHERE id = $2
	`
	
	_, err := r.pool.Exec(ctx, query, time.Now(), eventID)
	return err
}

func (r *outboxRepository) MarkEventFailed(ctx context.Context, eventID string, errorMsg string) error {
	query := `
		UPDATE outbox_events
		SET status = 'failed', last_error = $1
		WHERE id = $2
	`
	
	_, err := r.pool.Exec(ctx, query, errorMsg, eventID)
	return err
}

func (r *outboxRepository) IncrementRetryCount(ctx context.Context, eventID string) error {
	query := `
		UPDATE outbox_events
		SET retry_count = retry_count + 1
		WHERE id = $1
	`
	
	_, err := r.pool.Exec(ctx, query, eventID)
	return err
}

func (r *outboxRepository) DeleteProcessedEvents(ctx context.Context, olderThan time.Time) error {
	query := `
		DELETE FROM outbox_events
		WHERE status = 'completed' AND processed_at < $1
	`
	
	_, err := r.pool.Exec(ctx, query, olderThan)
	return err
}

func (r *outboxRepository) GetFailedEvents(ctx context.Context, maxRetries int) ([]*service.OutboxEvent, error) {
	query := `
		SELECT id, event_type, aggregate_id, payload, created_at, processed_at,
		       retry_count, last_error, status
		FROM outbox_events
		WHERE status = 'pending' AND retry_count >= $1
		ORDER BY created_at ASC
	`
	
	rows, err := r.pool.Query(ctx, query, maxRetries)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	events := make([]*service.OutboxEvent, 0)
	for rows.Next() {
		event := &service.OutboxEvent{}
		err := rows.Scan(
			&event.ID,
			&event.EventType,
			&event.AggregateID,
			&event.Payload,
			&event.CreatedAt,
			&event.ProcessedAt,
			&event.RetryCount,
			&event.LastError,
			&event.Status,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	
	return events, rows.Err()
}
