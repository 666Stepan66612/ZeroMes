package repository

import (
	"context"
	"sort"

	apperrors "message-service/internal/cores/errors"
	"message-service/internal/messaging/service"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) service.MessageRepository {
	return &postgresRepository{
		pool: pool,
	}
}

func (r *postgresRepository) CreateWithChats(ctx context.Context, msg *service.Message) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
 
	_, err = tx.Exec(ctx, `
		INSERT INTO messages (id, chat_id, sender_id, recipient_id, encrypted_content, message_type, created_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, msg.ID, msg.ChatID, msg.SenderID, msg.RecipientID,
		msg.EncryptedContent, msg.MessageType, msg.CreatedAt, msg.Status)
	if err != nil {
		return err
	}
 
	upsert := `
		INSERT INTO chats (user_id, companion_id, last_message_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id, companion_id) DO UPDATE SET last_message_at = NOW()
	`
	if _, err = tx.Exec(ctx, upsert, msg.SenderID, msg.RecipientID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, upsert, msg.RecipientID, msg.SenderID); err != nil {
		return err
	}
 
	return tx.Commit(ctx)
}

func (r *postgresRepository) GetByChatID(ctx context.Context, chatID string, limit int, lastMessageID string) ([]*service.Message, error) {
	var query string
	var args []interface{}

	if lastMessageID == "" {
		query = `
			SELECT id, chat_id, sender_id, recipient_id, encrypted_content, message_type, created_at, status
			FROM messages
			WHERE chat_id = $1
			ORDER BY created_at DESC, id DESC
			LIMIT $2
		`

		args = []interface{}{chatID, limit}
	} else {
		query = `
			SELECT id, chat_id, sender_id, recipient_id, encrypted_content, message_type, created_at, status
			FROM messages
			WHERE chat_id = $1 AND (created_at, id) < (SELECT created_at, id FROM messages WHERE id = $2)
			ORDER BY created_at DESC, id DESC
			LIMIT $3
		`
		args = []interface{}{chatID, lastMessageID, limit}
	}
	
	rows, err := r.pool.Query(ctx, query, args...)
	if err == pgx.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]*service.Message, 0)
	for rows.Next() {
		msg := &service.Message{}
		err := rows.Scan(
			&msg.ID,
			&msg.ChatID,
    		&msg.SenderID,
    		&msg.RecipientID,
			&msg.EncryptedContent,
    		&msg.MessageType,
    		&msg.CreatedAt,
    		&msg.Status,
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

func (r *postgresRepository) GetByID(ctx context.Context, messageID string) (*service.Message, error) {
	query := `
		SELECT id, chat_id, sender_id, recipient_id, encrypted_content, message_type, created_at, status
		FROM messages
		WHERE id = $1
	`

	msg := &service.Message{}
	err := r.pool.QueryRow(ctx, query, messageID).Scan(
		&msg.ID,
    	&msg.ChatID,
    	&msg.SenderID,
    	&msg.RecipientID,
    	&msg.EncryptedContent,
    	&msg.MessageType,
    	&msg.CreatedAt,
    	&msg.Status,
	)

	if err == pgx.ErrNoRows {
		return nil, apperrors.ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (r *postgresRepository) Delete(ctx context.Context, messageID string) error {
	query := `
		DELETE FROM messages WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, messageID)
	return err
}

func (r *postgresRepository) Alter(ctx context.Context, messageID, newContent string) error {
	query := `
		UPDATE messages SET encrypted_content = $1 WHERE id = $2
	`
	_, err := r.pool.Exec(ctx, query, newContent, messageID)
	return err
}

func (r *postgresRepository) UpdateStatusBatch(ctx context.Context, chatID, userID, lastMessageID string, status service.MessageStatus) error {
	query := `
		UPDATE messages
		SET status = $1
		WHERE chat_id = $2
			AND recipient_id = $3
			AND created_at <= (SELECT created_at FROM messages WHERE id = $4)
			AND status < $1
	`

	_, err := r.pool.Exec(ctx, query, status, chatID, userID, lastMessageID)
	return err
}

func (r *postgresRepository) GetChats(ctx context.Context, userID string) ([]*service.ChatsList, error) {
	query :=  `
		SELECT user_id, companion_id, created_at, last_message_at,
		       COALESCE(encrypted_key, '') as encrypted_key,
		       COALESCE(key_iv, '') as key_iv
		FROM chats
		WHERE user_id = $1
		ORDER BY last_message_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err == pgx.ErrNoRows {
		return nil, apperrors.ErrNotFound
	}
	defer rows.Close()

	if err != nil {
		return nil, err
	}

	chatList := make([]*service.ChatsList, 0)
	for rows.Next() {
		cht := &service.ChatsList{}
		err := rows.Scan(
    		&cht.UserID,
    		&cht.CompanionID,
    		&cht.CreatedAt,
			&cht.LastMessageAt,
			&cht.EncryptedKey,
			&cht.KeyIV,
		)
		if err != nil {
			return nil, err
		}

		ids := []string{cht.UserID, cht.CompanionID}
		sort.Strings(ids)
		cht.ChatID = ids[0] + ":" + ids[1]

		chatList = append(chatList, cht)
	}

	return chatList, rows.Err()
}

func (r *postgresRepository) SaveChatKeys(ctx context.Context, userID, companionID, encryptedKey, keyIV string) error {
	query := `
		UPDATE chats
		SET encrypted_key = $1, key_iv = $2
		WHERE user_id = $3 AND companion_id = $4
	`
	_, err := r.pool.Exec(ctx, query, userID, companionID, encryptedKey, keyIV)
	return err
}