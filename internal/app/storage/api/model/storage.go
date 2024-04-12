package model

import (
	"context"

	"github.com/avGenie/go-loyalty-system/internal/app/entity"
)

type Storage interface {
	Ping(ctx context.Context) error

	CreateUser(user entity.User) error
	GetUser(user entity.User) (entity.User, error)
}
