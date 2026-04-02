package service

import (
	"auth-service/internal/cores/errors"
	"context"
	"fmt"
	"time"
    "log/slog"
    "strings"

	"github.com/google/uuid"
)

type authService struct {
	userRepo UserRepository
	tokenSvc TokenService
}


func NewAuthService(userRepo UserRepository, tokenSvc TokenService) AuthService {
    return &authService{
        userRepo: userRepo,
        tokenSvc: tokenSvc,
    }
}

func (s *authService) Register(ctx context.Context, login, authHash, publicKey string) (*UserPublic, *TokenPair, error) {
    existing, _ := s.userRepo.GetByLogin(ctx, login)
    if existing != nil {
        return nil, nil, errors.ErrUserAlreadyExists
    }

    serverSalt, err := GenerateServerSalt()
    if err != nil {
        return nil, nil, err
    }

    storedHash, err := HashAuthHash(authHash, serverSalt)
    if err != nil {
        return nil, nil, err
    }

    user := &User{
        ID: uuid.New().String(),
        Login: login,
        AuthHash: storedHash,
        ServerSalt: serverSalt,
        PublicKey: publicKey,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    if err := s.userRepo.Create(ctx, user); err != nil {
        return nil, nil, err
    }

    tokens, err := s.tokenSvc.GenerateTokenPair(user.ID)
    if err != nil {
        return nil, nil, err
    }

    return user.ToPublic(), tokens, nil
}

func (s *authService) Login(ctx context.Context, login, authHash string) (*UserPublic, *TokenPair, error) {
    user, err := s.userRepo.GetByLogin(ctx, login)
    if err != nil || user == nil {
        return nil, nil, errors.ErrInvalidCredentials
    }

    if !user.ValidateAuthHash(authHash) {
        return nil, nil, errors.ErrInvalidCredentials
    }

    tokens, err := s.tokenSvc.GenerateTokenPair(user.ID)
    if err != nil {
        return nil, nil, err
    }

    return user.ToPublic(), tokens, nil
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
    userID, err := s.tokenSvc.ValidateRefreshToken(refreshToken)
    if err != nil {
        return nil, errors.ErrInvalidToken
    }

    return s.tokenSvc.GenerateTokenPair(userID)
}

func (s *authService) Logout(ctx context.Context, refreshToken, accessToken string) error {
    if err := s.tokenSvc.InvalidateRefreshToken(refreshToken); err != nil {
        return err
    }
    return s.tokenSvc.InvalidateAccessToken(accessToken)
}

func (s *authService) Search(ctx context.Context, login string) ([]*UserPublic, error) {
    users, err := s.userRepo.SearchUsers(ctx, login)
    if err != nil {
        return nil, err
    }
    if users == nil {
        return []*UserPublic{}, nil
    }
    return users, nil
}

func (s *authService) ChangePassword(ctx context.Context, login, oldAuthHash, newAuthHash string) (string, error) {
    if strings.TrimSpace(login) == "" {
        return "", errors.ErrInvalidInput
    }
    if oldAuthHash == "" {
        return "", errors.ErrInvalidInput
    }
    if newAuthHash == "" {
        return "", errors.ErrInvalidInput
    }

    user, err := s.userRepo.GetByLogin(ctx, login)
    if err != nil {
        return "", errors.ErrUserNotFound
    }

    if !user.ValidateAuthHash(oldAuthHash) {
        return "", errors.ErrInvalidOldPassword
    }

    hashedNewAuthHash, err := HashAuthHash(newAuthHash, user.ServerSalt)
    if err != nil {
        return "", fmt.Errorf("failed to hash new password: %w", err)
    }

    err = s.userRepo.UpdateAuthHash(ctx, user.ID, hashedNewAuthHash)
    if err != nil {
        return "", fmt.Errorf("failed to update password: %w", err)
    }

    if err = s.tokenSvc.InvalidateAccessToken(user.ID); err != nil {
        slog.Warn("failed to invalidate access token", "user_id", user.ID, "error", err)
    }

    if err = s.tokenSvc.InvalidateRefreshToken(user.ID); err != nil {
        slog.Warn("failed to invalidate refresh token", "user_id", user.ID, "error", err)
    }

    return  user.ID, nil
}