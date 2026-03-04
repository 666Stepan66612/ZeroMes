package transport

type RegisterRequest struct {
	Login     string `json:"login" validate:"required,min=3,max=32"`
	AuthHash  string `json:"auth_hash" validate:"required"`  // PBKDF2 password for auth
	PublicKey string `json:"public_key" validate:"required"` // for E2E encryprion
}

type LoginRequest struct {
	Login    string `json:"login" validate:"required"`
	AuthHash string `json:"auth_hash" validate:"required"` // PBKDF2 of password
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
	CreatedAt string `json:"created_at"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
