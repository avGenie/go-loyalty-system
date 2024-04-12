package model

import (
	"context"

	"github.com/avGenie/go-loyalty-system/internal/app/entity"
)

type Storage interface {
	CreateUser(ctx context.Context, user entity.User) error
	GetUser(ctx context.Context, user entity.User) (entity.User, error)
}
