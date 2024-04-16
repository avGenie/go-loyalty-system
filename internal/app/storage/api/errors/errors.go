package storage

import "errors"

var (
	ErrLoginExists   = errors.New("given login already exists in storage")
	ErrLoginNotFound = errors.New("given login doesn't exist in storage")

	ErrOrderNumberExists = errors.New("order with given number already exists in storage")
	ErrOrderNumberNotFound = errors.New("order with given number doesn't exist in storage")
)
