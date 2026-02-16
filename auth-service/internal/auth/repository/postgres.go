package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"auth-service/internal/auth/service"
	"auth-service/internal/cores/errors"
)

type postgresUserRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUserRepository(pool *pgxpool.Pool) service.UserRepository {
	return &postgresUserRepository{pool: pool}
}

func (r *postgresUserRepository) Create(ctx context.Context, user *service.User) error {
	query := `
		INSERT INTO users (id, login, auth_hash, public_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.pool.Exec(ctx, query, user.ID, user.Login, user.AuthHash, user.PublicKey, user.CreatedAt, user.UpdatedAt)
	
	return err
}

func (r *postgresUserRepository) GetByID(ctx context.Context, id string) (*service.User, error) {
	query := `
		SELECT id, login, auth_hash, public_key, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &service.User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Login, &user.AuthHash, &user.PublicKey, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == errors.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *postgresUserRepository) GetByLogin(ctx context.Context, login string) (*service.User, error){
	query := `
		SELECT id, login, auth_hash, public_key, created_at, updated_at
		FROM users
		WHERE login = $1
	`

	user := &service.User{}
	err := r.pool.QueryRow(ctx, query, login).Scan(
		&user.ID, &user.Login, &user.AuthHash, &user.PublicKey, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == errors.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return user, nil
}