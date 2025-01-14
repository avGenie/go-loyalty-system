package order

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	httputils "github.com/avGenie/go-loyalty-system/internal/app/usecase/utils"
	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	err_storage "github.com/avGenie/go-loyalty-system/internal/app/storage/api/errors"
	"go.uber.org/zap"
)

type OrderProcessor interface {
	UploadOrder(ctx context.Context, userID entity.UserID, orderNumber entity.OrderNumber) (entity.UserID, error)
	GetOrdersForUpdate(ctx context.Context, count, offset int) (entity.UpdateUserOrders, error)
	GetUserOrders(ctx context.Context, userID entity.UserID) (entity.Orders, error)
	UpdateOrders(ctx context.Context, orders entity.UpdateUserOrders) error
	GetUserBalance(ctx context.Context, userID entity.UserID) (entity.UserBalance, error)
	WithdrawUser(ctx context.Context, userID entity.UserID, withdraw entity.Withdraw) error
	GetUserWithdrawals(ctx context.Context, userID entity.UserID) (entity.Withdrawals, error)
}

func UploadOrder(userID entity.UserID, orderNumber entity.OrderNumber, processor OrderProcessor, w http.ResponseWriter) (entity.UserID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), httputils.RequestTimeout)
	defer cancel()

	storageUserID, err := processor.UploadOrder(ctx, userID, orderNumber)
	if err != nil {
		if errors.Is(err, err_storage.ErrOrderNumberExists) {
			if userID == storageUserID {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusConflict)
			}

			return storageUserID, err
		}

		w.WriteHeader(http.StatusInternalServerError)
		return entity.UserID(""), err
	}

	return storageUserID, nil
}

func GetUserOrders(userID entity.UserID, processor OrderProcessor, w http.ResponseWriter) (entity.Orders, error) {
	ctx, cancel := context.WithTimeout(context.Background(), httputils.RequestTimeout)
	defer cancel()

	orders, err := processor.GetUserOrders(ctx, userID)
	if err != nil {
		if errors.Is(err, err_storage.ErrOrderForUserNotFound) {
			zap.L().Info("orders for given user not found", zap.String("user_id", userID.String()))
			w.WriteHeader(http.StatusNoContent)
		} else {
			zap.L().Error("error while getting user orders", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
		}

		return entity.Orders{}, err
	}

	return orders, nil
}

func GetUserBalance(userID entity.UserID, processor OrderProcessor, w http.ResponseWriter) (entity.UserBalance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), httputils.RequestTimeout)
	defer cancel()

	balance, err := processor.GetUserBalance(ctx, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return entity.UserBalance{}, fmt.Errorf("error while getting user balance: %w", err)
	}

	return balance, nil
}

func WithdrawUserBonuses(userID entity.UserID, withdraw entity.Withdraw, processor OrderProcessor, w http.ResponseWriter) {
	ctx, cancel := context.WithTimeout(context.Background(), httputils.RequestTimeout)
	defer cancel()

	err := processor.WithdrawUser(ctx, userID, withdraw)
	if err != nil {
		if errors.Is(err, err_storage.ErrNotEnoughSum) {
			zap.L().Info("not enough money for withdrawing")
			w.WriteHeader(http.StatusPaymentRequired)
		} else {
			zap.L().Error("error while withdrawing user to storage", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
		}

		return
	}

	w.WriteHeader(http.StatusOK)
}

func GetUserWithdrawals(userID entity.UserID, processor OrderProcessor, w http.ResponseWriter) (entity.Withdrawals, error) {
	ctx, cancel := context.WithTimeout(context.Background(), httputils.RequestTimeout)
	defer cancel()

	withdrawals, err := processor.GetUserWithdrawals(ctx, userID)
	if err != nil {
		if errors.Is(err, err_storage.ErrWithdrawalsForUserNotFound) {
			zap.L().Info("withdrawals for given user not found", zap.String("user_id", userID.String()))
			w.WriteHeader(http.StatusNoContent)
		} else {
			zap.L().Error("error while getting user withdrawals", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
		}
		return nil, err
	}

	return withdrawals, nil
}
