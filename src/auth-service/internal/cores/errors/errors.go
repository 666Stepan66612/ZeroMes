package errors

import "errors"

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrInvalidClaims      = errors.New("invalid claims")
	ErrUserNotFound       = errors.New("user not found")
	ErrInternalServer     = errors.New("internal server error")
	ErrInvalidPayload     = errors.New("invalid requerst payload")
	ErrNoResult           = errors.New("no results")
	ErrInvalidOldPassword = errors.New("invalid old password")
	ErrInvalidInput       = errors.New("invalid input")
)
