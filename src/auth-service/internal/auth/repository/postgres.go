package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"auth-service/internal/auth/service"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// escapeILIKE escapes special characters in ILIKE patterns
func escapeILIKE(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

type postgresUserRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUserRepository(pool *pgxpool.Pool) service.UserRepository {
	return &postgresUserRepository{pool: pool}
}

func (r *postgresUserRepository) Create(ctx context.Context, user *service.User) error {
	query := `
		INSERT INTO users (id, login, auth_hash, server_salt, public_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.pool.Exec(ctx, query, user.ID, user.Login, user.AuthHash, user.ServerSalt, user.PublicKey, user.CreatedAt, user.UpdatedAt)

	return err
}

func (r *postgresUserRepository) GetByID(ctx context.Context, id string) (*service.User, error) {
	query := `
		SELECT id, login, auth_hash, server_salt, public_key, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &service.User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Login, &user.AuthHash, &user.ServerSalt, &user.PublicKey, &user.CreatedAt, &user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *postgresUserRepository) GetByLogin(ctx context.Context, login string) (*service.User, error) {
	query := `
		SELECT id, login, auth_hash, server_salt, public_key, created_at, updated_at
		FROM users
		WHERE login = $1
	`

	user := &service.User{}
	err := r.pool.QueryRow(ctx, query, login).Scan(
		&user.ID, &user.Login, &user.AuthHash, &user.ServerSalt, &user.PublicKey, &user.CreatedAt, &user.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *postgresUserRepository) SearchUsers(ctx context.Context, login string) ([]*service.UserPublic, error) {
	// Escape special ILIKE characters to prevent injection
	escapedLogin := escapeILIKE(login)

	query := `
		SELECT id, login, public_key, created_at
		FROM users
		WHERE login ILIKE $1
		LIMIT 10
	`

	rows, err := r.pool.Query(ctx, query, escapedLogin+"%")

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*service.UserPublic, 0)
	for rows.Next() {
		user := &service.UserPublic{}
		if err := rows.Scan(
			&user.ID, &user.Login, &user.PublicKey, &user.CreatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

func (r *postgresUserRepository) UpdateAuthHashAndPublicKey(ctx context.Context, userID, newAuthHash, newPublicKey string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		UPDATE users
		SET auth_hash = $1, public_key = $2
		WHERE id = $3
	`
	_, err = tx.Exec(ctx, query, newAuthHash, newPublicKey, userID)
	if err != nil {
		return fmt.Errorf("failed to update auth hash: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
