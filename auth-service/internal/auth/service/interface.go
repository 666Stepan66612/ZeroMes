package service

import "context"

type AuthService interface {
	Register(ctx context.Context, login, authHash, publicKey string) (*UserPublic, *TokenPair, error)
	Login(ctx context.Context, login, authHash string) (*UserPublic, *TokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)
	Logout(ctx context.Context, refreshToken, accessToken string) error
	Search(ctx context.Context, login string) ([]*UserPublic, error)
	ChangePassword(ctx context.Context, login, oldAuthHash, newAuthHash string) (string, error)
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByLogin(ctx context.Context, login string) (*User, error)
	SearchUsers(ctx context.Context, login string) ([]*UserPublic, error)
	UpdateAuthHash(ctx context.Context, userID, newAuthHash string) error
}

type TokenService interface {
	GenerateTokenPair(userID string) (*TokenPair, error)
	ValidateAccessToken(token string) (string, error)
	ValidateRefreshToken(token string) (string, error)
	InvalidateRefreshToken(token string) error
	InvalidateAccessToken(token string) error
}