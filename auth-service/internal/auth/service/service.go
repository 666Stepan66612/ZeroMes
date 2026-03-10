package service

import (
	"context"
	"auth-service/internal/cores/errors"
    "time"

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

    user := &User{
        ID: uuid.New().String(),
        Login: login,
        AuthHash: authHash,
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
    return s.tokenSvc.InvalidateRefreshToken(accessToken)
}