package order

import (
	"context"
	"errors"
	"time"

	"github.com/avGenie/go-loyalty-system/internal/app/config"
	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	err_storage "github.com/avGenie/go-loyalty-system/internal/app/storage/api/errors"
	"github.com/avGenie/go-loyalty-system/internal/app/usecase/accrual"
	"go.uber.org/zap"
)

const (
	flushBufLen = 10

	requestTimeout = 3 * time.Second
	updateTimeout  = 5 * time.Second
)

type OrdersUpdater interface {
	GetOrdersForUpdate(ctx context.Context, count, offset int) (entity.UpdateUserOrders, error)
	UpdateOrders(ctx context.Context, orders entity.UpdateUserOrders) error
}

type StatusUpdater struct {
	updater         OrdersUpdater
	accrual         *accrual.Accrual
	batchOrders     map[entity.OrderNumber]entity.UpdateUserOrder
	done            chan struct{}
	countForUpdate  int
	offsetForUpdate int
}

func CreateStatusUpdater(updater OrdersUpdater, config config.Config) *StatusUpdater {
	return &StatusUpdater{
		updater:         updater,
		accrual:         accrual.New(config),
		batchOrders:     make(map[entity.OrderNumber]entity.UpdateUserOrder, flushBufLen),
		done:            make(chan struct{}),
		countForUpdate:  flushBufLen,
		offsetForUpdate: 0,
	}
}

func (u *StatusUpdater) Start() {
	for {
		select {
		case <-u.done:
			zap.L().Info("status updater work has finished")
			return
		default:
			orders := u.getOrdersForUpdate()
			if len(orders) == 0 {
				continue
			}

			u.requestForUpdate(orders)
			u.flushUpdates()
		}
	}
}

func (u *StatusUpdater) Stop() {
	close(u.done)
}

func (u *StatusUpdater) getOrdersForUpdate() entity.UpdateUserOrders {
	ctx, close := context.WithTimeout(context.Background(), requestTimeout)
	defer close()

	orders, err := u.updater.GetOrdersForUpdate(ctx, u.countForUpdate, u.offsetForUpdate)
	if err != nil {
		if errors.Is(err, err_storage.ErrOrdersForUpdateNotFound) {
			u.offsetForUpdate = 0
			return entity.UpdateUserOrders{}
		}

		zap.L().Error("error while getting orders for update", zap.Error(err))
		return entity.UpdateUserOrders{}
	}

	return orders
}

func (u *StatusUpdater) requestForUpdate(orders entity.UpdateUserOrders) {
	for _, order := range orders {
		accrualOrder, err := u.accrual.MakeRequest(order.UserID, order.Order.Number)
		if err != nil {
			zap.L().Error("error while getting accrual update", zap.Error(err))
			continue
		}

		if entity.StatusPause == accrualOrder.Status {
			zap.L().Info("accrual update request returns retry after value", zap.Int("retry after", int(accrualOrder.RetryAfter)))
			u.flushUpdates()
			<-time.After(accrualOrder.RetryAfter)

			continue
		}

		if entity.StatusOrderNotRegistered == accrualOrder.Status {
			zap.L().Info("order not registered while getting accrual update")
			continue
		}

		updatedOrder := entity.UpdateUserOrder{
			UserID: accrualOrder.UserID,
			Order:  accrualOrder.Order,
		}

		u.batchOrders[accrualOrder.Order.Number] = updatedOrder
	}
}

func (u *StatusUpdater) flushUpdates() {
	if len(u.batchOrders) == 0 {
		return
	}

	dbOrders := make(entity.UpdateUserOrders, 0, len(u.batchOrders))
	for _, order := range u.batchOrders {
		dbOrders = append(dbOrders, order)
	}

	ctx, close := context.WithTimeout(context.Background(), updateTimeout)
	defer close()

	err := u.updater.UpdateOrders(ctx, dbOrders)
	if err != nil {
		zap.L().Error("error while updating orders", zap.Error(err))
	}

	clear(u.batchOrders)
}
