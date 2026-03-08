package errors

import "errors"

var (
	ErrUpdate = errors.New("failed to upgrade connection")
)