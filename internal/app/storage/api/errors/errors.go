package storage

import "errors"

var (
	ErrLoginExists   = errors.New("given login exists in storage")
	ErrLoginNotFound = errors.New("given login doesn't exist in storage")
)
