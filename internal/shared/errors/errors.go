package errors

import "errors"

var (
	ErrValidation   = errors.New("validation error")
	ErrInvalidInput = errors.New("invalid input")

	ErrNotFound          = errors.New("not found")
	ErrBusinessRule      = errors.New("business rule violation")

	ErrExternalService = errors.New("external service error")
	ErrTimeout         = errors.New("timeout error")

	ErrInternal = errors.New("internal error")
)
