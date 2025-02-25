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
	GetOrdersForUpdate(ctx context.Context, count, offset int) (entity.UpdateUserOrders, error)
	GetUserOrders(ctx context.Context, userID entity.UserID) (entity.Orders, error)
	UpdateOrders(ctx context.Context, orders entity.UpdateUserOrders) error

	GetUserBalance(ctx context.Context, userID entity.UserID) (entity.UserBalance, error)

	WithdrawUser(ctx context.Context, userID entity.UserID, withdraw entity.Withdraw) error
	GetUserWithdrawals(ctx context.Context, userID entity.UserID) (entity.Withdrawals, error)
}
