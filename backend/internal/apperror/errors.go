package apperror

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken = errors.New("email already registered")
	ErrInvalidAccessCode = errors.New("invalid or expired access code")
	ErrAccessCodeExhausted = errors.New("access code has reached its usage limit")
	ErrInvalidFavorite = errors.New("invalid favorite")
)
