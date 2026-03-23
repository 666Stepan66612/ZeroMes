package service

import (
	"time"
	"encoding/hex"
	"crypto/rand"
	"crypto/sha256"


	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        string
	Login     string
	AuthHash  string // PBKDF2/Argon2 hash for auth (not for encryption!)
	ServerSalt string //random salt, generate with register
	PublicKey string // for E2E encryption
	CreatedAt time.Time
	UpdatedAt time.Time
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
    combined := sha256.Sum256([]byte(clientAuthHash + serverSalt))
    hashed, err := bcrypt.GenerateFromPassword(combined[:], bcrypt.DefaultCost)
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
    combined := sha256.Sum256([]byte(clientAuthHash + u.ServerSalt))
    return bcrypt.CompareHashAndPassword([]byte(u.AuthHash), combined[:]) == nil
}

func NewUserID() string {
	return uuid.New().String()
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}
