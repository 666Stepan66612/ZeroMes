package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	apperrors "auth-service/internal/cores/errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

type tokenService struct {
	accessSecret  string
	refreshSecret string
	redis         *redis.Client
}

func NewTokenService(accessSecret, refreshSecret string, redisClient *redis.Client) TokenService {
	return &tokenService{
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
		redis:         redisClient,
	}
}

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func (s *tokenService) GenerateTokenPair(userID string) (*TokenPair, error) {
	accessToken, err := s.generateToken(userID, s.accessSecret, 15*time.Minute)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateToken(userID, s.refreshSecret, 7*24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *tokenService) generateToken(userID, secret string, duration time.Duration) (string, error) {
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (s *tokenService) ValidateAccessToken(token string) (string, error) {
	return s.validateToken(token, s.accessSecret)
}

func (s *tokenService) ValidateRefreshToken(token string) (string, error) {
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	ctx := context.Background()
	exists, err := s.redis.Exists(ctx, "blacklist:"+tokenHash).Result()
	if err == nil && exists > 0 {
		return "", apperrors.ErrInvalidToken
	}
	return s.validateToken(token, s.refreshSecret)
}

func (s *tokenService) validateToken(tokenString, secret string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		return "", apperrors.ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return "", apperrors.ErrInvalidClaims
	}

	return claims.UserID, nil
}

func (s *tokenService) InvalidateRefreshToken(token string) error {
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	ctx := context.Background()
	err := s.redis.Set(ctx, "blacklist:"+tokenHash, "revoked", 7*24*time.Hour).Err()
	if err != nil {
		return apperrors.ErrInternalServer
	}

	return nil
}

func (s *tokenService) InvalidateAccessToken(token string) error {
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])
	return s.redis.Set(context.Background(), "blacklist:"+tokenHash, "1", 15*time.Minute).Err()
}
