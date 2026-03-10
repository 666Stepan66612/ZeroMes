package service

import (
	pkgjwt "github.com/666Stepan66612/ZeroMes/pkg/jwt"
)

type AuthClientService struct {
	secret string
}

func NewAuthClient(secret string) *AuthClientService {
	return &AuthClientService{secret: secret}
}

func (c *AuthClientService) ValidateToken(token string) (string, error) {
	return pkgjwt.ValidateAccessToken(token, c.secret)
}
