package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID         string
	Login      string
	AuthHash   string // PBKDF2/Argon2 hash for auth (not for encryption!)
	ServerSalt string //random salt, generate with register
	PublicKey  string // for E2E encryption
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type UserPublic struct {
	ID        string
	Login     string
	PublicKey string
	CreatedAt time.Time
}

func GenerateServerSalt() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func HashAuthHash(clientAuthHash, serverSalt string) (string, error) {
	// Combine client auth hash with server salt
	combined := clientAuthHash + serverSalt

	// Hash the combination with SHA256 to reduce length to 32 bytes
	hash := sha256.Sum256([]byte(combined))

	// Apply bcrypt to the SHA256 hash (32 bytes < 72 bytes limit)
	hashed, err := bcrypt.GenerateFromPassword(hash[:], bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func (u *User) ToPublic() *UserPublic {
	return &UserPublic{
		ID:        u.ID,
		Login:     u.Login,
		PublicKey: u.PublicKey,
		CreatedAt: u.CreatedAt,
	}
}

func (u *User) ValidateAuthHash(clientAuthHash string) bool {
	// Combine client auth hash with server salt
	combined := clientAuthHash + u.ServerSalt

	// Hash the combination with SHA256 (same as in HashAuthHash)
	hash := sha256.Sum256([]byte(combined))

	// Compare with stored bcrypt hash
	return bcrypt.CompareHashAndPassword([]byte(u.AuthHash), hash[:]) == nil
}

func NewUserID() string {
	return uuid.New().String()
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}
