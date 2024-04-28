package storage

import "errors"

var (
	ErrLoginExists   = errors.New("given login already exists in storage")
	ErrLoginNotFound = errors.New("given login doesn't exist in storage")

	ErrOrderNumberExists    = errors.New("order with given number already exists in storage")
	ErrOrderNumberNotFound  = errors.New("order with given number doesn't exist in storage")
	ErrOrderForUserNotFound = errors.New("order with given number doesn't exist for given user in storage")

	ErrUserExistsTable   = errors.New("given user exist in table")
	ErrUserNotFoundTable = errors.New("given user doesn't exist in table")

	ErrNotEnoughSum = errors.New("not enough sum")
)
