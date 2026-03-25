package errors

import "errors"

var (
    ErrNotFound       = errors.New("not found")
    ErrNotYourMessage = errors.New("forbidden: not your message")
    ErrInvalidInput   = errors.New("invalid input")
    ErrInternalServer = errors.New("internal server error")
    NilRequest        = errors.New("request is nil")
    ErrForbidden      = errors.New("Forbidden")
)