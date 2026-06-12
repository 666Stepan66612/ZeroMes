package repository

import (
	"context"
	"fmt"
	"log/slog"
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
	slog.Info("GetByChatID repository", "chat_id", chatID, "limit", limit, "last_message_id", lastMessageID)

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
		// First check if lastMessageID exists
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM messages WHERE id = $1)`
		if err := r.pool.QueryRow(ctx, checkQuery, lastMessageID).Scan(&exists); err != nil {
			return nil, err
		}

		if !exists {
			slog.Warn("lastMessageID not found, returning empty result", "last_message_id", lastMessageID)
			return []*service.Message{}, nil
		}

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

	slog.Info("GetByChatID repository result", "messages_count", len(messages))
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
	query := `
		SELECT c.user_id, c.companion_id, c.created_at, c.last_message_at,
		       COALESCE(c.encrypted_key, '') as encrypted_key,
		       COALESCE(c.key_iv, '') as key_iv,
		       COALESCE(m.encrypted_content, '') as last_message
		FROM chats c
		LEFT JOIN LATERAL (
			SELECT encrypted_content
			FROM messages
			WHERE chat_id = (
				CASE
					WHEN c.user_id < c.companion_id THEN c.user_id || ':' || c.companion_id
					ELSE c.companion_id || ':' || c.user_id
				END
			)
			ORDER BY created_at DESC
			LIMIT 1
		) m ON true
		WHERE c.user_id = $1
		ORDER BY c.last_message_at DESC
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
			&cht.LastMessage,
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
		INSERT INTO chats (user_id, companion_id, encrypted_key, key_iv, last_message_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (user_id, companion_id)
		DO UPDATE SET encrypted_key = $3, key_iv = $4
	`
	_, err := r.pool.Exec(ctx, query, userID, companionID, encryptedKey, keyIV)
	return err
}

func (r *postgresRepository) UpdateChatKeys(ctx context.Context, userID string, keys []service.ChatKeyUpdate) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction:  %w", err)
	}
	defer tx.Rollback(ctx)

	count := 0
	query := `
		UPDATE chats
		SET encrypted_key = $1, key_iv = $2
		WHERE user_id = $3 AND companion_id = $4
	`
	for _, key := range keys {
		result, err := tx.Exec(ctx, query, key.EncryptedKey, key.KeyIV, userID, key.CompanionID)
		if err != nil {
			return 0, fmt.Errorf("failed to update chat key for companion %s: %w", key.CompanionID, err)
		}
		count += int(result.RowsAffected())
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return count, nil
}

func (r *postgresRepository) CreateGroup(ctx context.Context, name, avatarURL, createdBy string, keyVersion int) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx,
		`INSERT INTO group_chats (name, avatar_url, created_by, key_version)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		name, avatarURL, createdBy, keyVersion,
	).Scan(&id)
	return id, err
}

func (r *postgresRepository) GetGroupByID(ctx context.Context, groupID string) (*service.GroupChat, error) {
	g := &service.GroupChat{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, COALESCE(avatar_url,''), created_by, key_version, created_at
		 FROM group_chats WHERE id = $1`, groupID,
	).Scan(&g.ID, &g.Name, &g.AvatarURL, &g.CreatedBy, &g.KeyVersion, &g.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, apperrors.ErrGroupNotFound
	}
	return g, err
}

func (r *postgresRepository) GetGroupChats(ctx context.Context, userID string) ([]*service.GroupChat, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT gc.id, gc.name, COALESCE(gc.avatar_url,''), gc.created_by, gc.key_version, gc.created_at
		 FROM group_chats gc
		 JOIN group_members gm ON gm.group_id = gc.id
		 WHERE gm.user_id = $1 AND gm.left_at IS NULL
		 ORDER BY gc.created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*service.GroupChat
	for rows.Next() {
		g := &service.GroupChat{}
		if err := rows.Scan(&g.ID, &g.Name, &g.AvatarURL, &g.CreatedBy, &g.KeyVersion, &g.CreatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func (r *postgresRepository) AddGroupMember(ctx context.Context, groupID, userID, role string, canReadFromMessageID *string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO group_members (group_id, user_id, role, can_read_from_message_id)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (group_id, user_id) DO NOTHING`,
		groupID, userID, role, canReadFromMessageID,
	)
	return err
}

func (r *postgresRepository) RemoveGroupMember(ctx context.Context, groupID, userID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE group_members SET left_at = NOW()
		 WHERE group_id = $1 AND user_id = $2 AND left_at IS NULL`,
		groupID, userID,
	)
	return err
}

func (r *postgresRepository) GetGroupMembers(ctx context.Context, groupID string) ([]*service.GroupMember, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT gm.user_id, COALESCE(gm.role,'member'), gm.joined_at, gm.can_read_from_message_id
		 FROM group_members gm
		 WHERE gm.group_id = $1 AND gm.left_at IS NULL
		 ORDER BY gm.joined_at`, groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*service.GroupMember
	for rows.Next() {
		m := &service.GroupMember{}
		if err := rows.Scan(&m.UserID, &m.Role, &m.JoinedAt, &m.CanReadFromMessageID); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *postgresRepository) CheckGroupMembership(ctx context.Context, groupID, userID string) (bool, string, error) {
	var role string
	err := r.pool.QueryRow(ctx,
		`SELECT role FROM group_members
		 WHERE group_id = $1 AND user_id = $2 AND left_at IS NULL`,
		groupID, userID,
	).Scan(&role)
	if err == pgx.ErrNoRows {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, role, nil
}

func (r *postgresRepository) GetActiveGroupMemberIDs(ctx context.Context, groupID string) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT user_id FROM group_members
		 WHERE group_id = $1 AND left_at IS NULL`, groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *postgresRepository) SaveGroupKeySeed(ctx context.Context, userID, groupID, encryptedSeed, encryptedBy string, keyVersion int) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_group_keys (user_id, group_id, encrypted_seed, encrypted_by, key_version)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (user_id, group_id)
		 DO UPDATE SET encrypted_seed = $3, encrypted_by = $4, key_version = $5`,
		userID, groupID, encryptedSeed, encryptedBy, keyVersion,
	)
	return err
}

func (r *postgresRepository) GetGroupKeySeed(ctx context.Context, userID, groupID string) (*service.GroupKeySeed, error) {
	s := &service.GroupKeySeed{}
	err := r.pool.QueryRow(ctx,
		`SELECT encrypted_seed, encrypted_by, key_version
		 FROM user_group_keys
		 WHERE user_id = $1 AND group_id = $2`,
		userID, groupID,
	).Scan(&s.EncryptedSeed, &s.EncryptedBy, &s.KeyVersion)
	if err == pgx.ErrNoRows {
		return nil, apperrors.ErrNotFound
	}
	return s, err
}

func (r *postgresRepository) GetCurrentKeyVersion(ctx context.Context, groupID string) (int, error) {
	var v int
	err := r.pool.QueryRow(ctx,
		`SELECT key_version FROM group_chats WHERE id = $1`, groupID,
	).Scan(&v)
	if err == pgx.ErrNoRows {
		return 0, apperrors.ErrGroupNotFound
	}
	return v, err
}

func (r *postgresRepository) TryAcquireRotationLock(ctx context.Context, groupID, userID string) (bool, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE group_chats
		 SET rotation_in_progress = TRUE, rotation_by = $2
		 WHERE id = $1 AND rotation_in_progress = FALSE AND needs_rotation = TRUE`,
		groupID, userID,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *postgresRepository) IncrementKeyVersion(ctx context.Context, groupID string) (int, error) {
	var newVersion int
	err := r.pool.QueryRow(ctx,
		`UPDATE group_chats
		 SET key_version = key_version + 1, needs_rotation = FALSE
		 WHERE id = $1
		 RETURNING key_version`,
		groupID,
	).Scan(&newVersion)
	return newVersion, err
}

func (r *postgresRepository) ReleaseRotationLock(ctx context.Context, groupID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE group_chats
		 SET rotation_in_progress = FALSE, rotation_by = NULL
		 WHERE id = $1`,
		groupID,
	)
	return err
}

func (r *postgresRepository) GetGroupMessages(ctx context.Context, groupID, userID string, limit int, lastMessageID string) ([]*service.Message, error) {
	var query string
	var args []interface{}

	if lastMessageID == "" {
		query = `
			SELECT id, chat_id, sender_id, recipient_id, encrypted_content, message_type, created_at, status
			FROM messages
			WHERE group_id = $1
			ORDER BY created_at DESC, id DESC
			LIMIT $2`
		args = []interface{}{groupID, limit}
	} else {
		query = `
			SELECT id, chat_id, sender_id, recipient_id, encrypted_content, message_type, created_at, status
			FROM messages
			WHERE group_id = $1 AND (created_at, id) < (SELECT created_at, id FROM messages WHERE id = $2)
			ORDER BY created_at DESC, id DESC
			LIMIT $3`
		args = []interface{}{groupID, lastMessageID, limit}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*service.Message
	for rows.Next() {
		msg := &service.Message{}
		if err := rows.Scan(&msg.ID, &msg.ChatID, &msg.SenderID, &msg.RecipientID,
			&msg.EncryptedContent, &msg.MessageType, &msg.CreatedAt, &msg.Status); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func (r *postgresRepository) CreateGroupMessage(ctx context.Context, msg *service.Message) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO messages (id, chat_id, sender_id, encrypted_content, message_type, created_at, status, group_id, key_version)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		msg.ID, msg.ChatID, msg.SenderID, msg.EncryptedContent,
		msg.MessageType, msg.CreatedAt, msg.Status, msg.ChatID, 0,
	)
	return err
}
