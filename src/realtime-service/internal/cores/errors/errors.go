package errors

import "errors"

var (
	ErrConNotFound       = errors.New("connection not found")
	ErrUserNotOnline     = errors.New("user not online")
	ErrUnexpectedMessage = errors.New("unexpected message")
)
