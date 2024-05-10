package usecase

import "errors"

var (
	ErrTokenNotValid = errors.New("token is not valid")
	ErrTokenExpired  = errors.New("token is expired")
)
