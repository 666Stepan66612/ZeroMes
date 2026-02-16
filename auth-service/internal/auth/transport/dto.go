package transport

type RegisterRequest struct {
	Login     string `json:"login" validate:"required,min=3,max=32"`
	AuthHash  string `json:"auth_hash" validate:"required"`  // PBKDF2 от пароля для аутентификации
	PublicKey string `json:"public_key" validate:"required"` // для E2E шифрования
}

type LoginRequest struct {
	Login    string `json:"login" validate:"required"`
	AuthHash string `json:"auth_hash" validate:"required"` // PBKDF2 от пароля
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type RegisterResponse struct {
	User         UserDTO `json:"user"`
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
}

type LoginResponse struct {
	User         UserDTO `json:"user"`
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
}

type TokenPairResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
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
