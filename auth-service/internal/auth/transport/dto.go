package transport

import "time"

type RegisterRequest struct {
	Login     string `json:"login"`
	AuthHash  string `json:"auth_hash"`  // PBKDF2 password for auth
	PublicKey string `json:"public_key"` // for E2E encryprion
}

type LoginRequest struct {
	Login    string `json:"login"`
	AuthHash string `json:"auth_hash"` // PBKDF2 of password
}

type RegisterResponse struct {
	User         UserDTO `json:"user"`
}

type LoginResponse struct {
	User         UserDTO `json:"user"`
}

type UserDTO struct {
	ID        string `json:"id"`
	Login     string `json:"login"`
	PublicKey string `json:"public_key"`
	CreatedAt time.Time `json:"created_at"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type SearchUserResponse struct {
	Users []UserDTO `json:"users"`
}

type ChangePasswordRequest struct {
    Login       string `json:"login"`
    OldAuthHash string `json:"old_auth_hash"`
    NewAuthHash string `json:"new_auth_hash"`
	NewPublicKey string `json:"new_public_key"`
}