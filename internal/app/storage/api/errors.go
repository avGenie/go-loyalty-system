package storage

import "errors"

var (
	ErrLoginExists = errors.New("given login exists in storage")
)