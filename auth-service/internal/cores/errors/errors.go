package errors

import "errors"

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
    ErrInvalidCredentials = errors.New("invalid credentials")
    ErrInvalidToken       = errors.New("invalid token")
    ErrInvalidClaims      = errors.New("invalid claims")
    ErrUserNotFound       = errors.New("user not found")
    ErrInternalServer     = errors.New("internal server error")
    ErrNoRows             = errors.New("no rows in result set")
    ErrInvalidPayload     = errors.New("invalid requerst payload")
)