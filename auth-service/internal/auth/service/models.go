package service

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        string
	Login     string
	AuthHash  string // PBKDF2/Argon2 хеш для аутентификации (НЕ для шифрования!)
	PublicKey string // Публичный ключ для E2E шифрования
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserPublic struct {
	ID        string
	Login     string
	PublicKey string
	CreatedAt time.Time
}

func (u *User) ToPublic() *UserPublic {
	return &UserPublic{
		ID:        u.ID,
		Login:     u.Login,
		PublicKey: u.PublicKey,
		CreatedAt: u.CreatedAt,
	}
}

func (u *User) ValidateAuthHash(authHash string) bool {
	return u.AuthHash == authHash
}

func NewUserID() string {
	return uuid.New().String()
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}
