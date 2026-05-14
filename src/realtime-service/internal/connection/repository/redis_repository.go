package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"realtime-service/internal/connection/service"
	apperrors "realtime-service/internal/cores/errors"

	"github.com/redis/go-redis/v9"
)

type RedisRepository struct {
	client *redis.Client
}

func NewRedisRepository(client *redis.Client) service.PresenceRepository {
	return &RedisRepository{
		client: client,
	}
}

func (r *RedisRepository) SetOnline(ctx context.Context, userID string, instanceID string, ttl time.Duration) error {
	key := fmt.Sprintf("user:%s:online", userID)
	pipe := r.client.Pipeline()
	pipe.Set(ctx, key, instanceID, ttl)
	pipe.SAdd(ctx, "online_users", userID)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}

	slog.Info("user set online", "userID", userID, "instanceID", instanceID)
	return nil
}

func (r *RedisRepository) SetOffline(ctx context.Context, userID string) error {
	key := fmt.Sprintf("user:%s:online", userID)
	pipe := r.client.Pipeline()
	pipe.Del(ctx, key)
	pipe.SRem(ctx, "online_users", userID)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}

	slog.Info("user set offline", "userID", userID)
	return nil
}

func (r *RedisRepository) IsOnline(ctx context.Context, userID string) (bool, error) {
	key := fmt.Sprintf("user:%s:online", userID)

	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		slog.Error("IsOnline check failed", "userID", userID, "err", err)
		return false, err
	}

	return exists > 0, nil
}

func (r *RedisRepository) GetUserInstance(ctx context.Context, userID string) (string, error) {
	key := fmt.Sprintf("user:%s:online", userID)

	instanceID, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", apperrors.ErrUserNotOnline
	}
	if err != nil {
		slog.Error("GetUserInstance failed", "userID", userID, "err", err)
		return "", err
	}

	return instanceID, nil
}

func (r *RedisRepository) GetOnlineCount(ctx context.Context) (int64, error) {
	count, err := r.client.SCard(ctx, "online_users").Result()
	if err != nil {
		slog.Error("GetOnlineCount failed", "err", err)
		return 0, err
	}

	return count, nil
}

func (r *RedisRepository) ExtendTTL(ctx context.Context, userID string, ttl time.Duration) error {
	key := fmt.Sprintf("user:%s:online", userID)

	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		slog.Error("ExtendTTL check failed", "userID", userID, "err", err)
		return err
	}

	if exists == 0 {
		return apperrors.ErrUserNotOnline
	}

	if err := r.client.Expire(ctx, key, ttl).Err(); err != nil {
		slog.Error("ExtendTTL failed", "userID", userID, "err", err)
		return err
	}

	return nil
}
