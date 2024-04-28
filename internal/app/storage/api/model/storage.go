package model

import (
	"context"

	"github.com/avGenie/go-loyalty-system/internal/app/entity"
)

type Storage interface {
	Close() error

	CreateUser(ctx context.Context, user entity.User) error
	GetUser(ctx context.Context, user entity.User) (entity.User, error)

	UploadOrder(ctx context.Context, userID entity.UserID, orderNumber entity.OrderNumber) (entity.UserID, error)
	GetUserOrders(ctx context.Context, userID entity.UserID) (entity.Orders, error)
	UpdateOrders(ctx context.Context, orders entity.Orders) error

	UpdateBalanceBatch(ctx context.Context, balances entity.UpdateUserBalances) error
	GetUserBalance(ctx context.Context, userID entity.UserID) (entity.UserBalance, error)
}
