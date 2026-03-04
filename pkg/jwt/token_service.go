package jwt

import (
	"errors" 
    "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
    UserID string `json:"user_id"`
    jwt.RegisteredClaims
}

func ValidateAccessToken(tokenString, secret string) (string, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        return []byte(secret), nil
    })

    if err != nil || !token.Valid {
        return "", ErrInvalidToken
    }

    claims, ok := token.Claims.(*Claims)
    if !ok {
        return "", ErrInvalidClaims
    }

    return claims.UserID, nil
}

var (
    ErrInvalidToken  = errors.New("invalid token")
    ErrInvalidClaims = errors.New("invalid claims")
)