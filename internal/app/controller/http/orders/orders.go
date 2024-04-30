package orders

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	httputils "github.com/avGenie/go-loyalty-system/internal/app/controller/http/utils"
	"github.com/avGenie/go-loyalty-system/internal/app/converter"
	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	"github.com/avGenie/go-loyalty-system/internal/app/model"
	err_storage "github.com/avGenie/go-loyalty-system/internal/app/storage/api/errors"
	"github.com/avGenie/go-loyalty-system/internal/app/usecase/validator"
	"go.uber.org/zap"
)

const (
	ErrTokenExpired = "token has expired"
	ErrInvalidAuth  = "auth credentials are invalid"
)

const (
	flushBufLen = 10

	tickerTime  = 5 * time.Second
	stopTimeout = 5 * time.Second
)

type OrderProcessor interface {
	UploadOrder(ctx context.Context, userID entity.UserID, orderNumber entity.OrderNumber) (entity.UserID, error)
	GetUserOrders(ctx context.Context, userID entity.UserID) (entity.Orders, error)
	UpdateOrders(ctx context.Context, orders entity.Orders) error
	UpdateBalanceBatch(ctx context.Context, balances entity.UpdateUserBalances) error
	GetUserBalance(ctx context.Context, userID entity.UserID) (entity.UserBalance, error)
	WithdrawUser(ctx context.Context, userID entity.UserID, withdraw entity.Withdraw) error
	GetUserWithdrawals(ctx context.Context, userID entity.UserID) (entity.Withdrawals, error)
}

type AccrualOrderConnector interface {
	SetInput(number entity.AccrualOrderRequest)
	CloseInput()
	GetOutput() (entity.AccrualOrder, bool)
}

type Order struct {
	storage          OrderProcessor
	accrualConnector AccrualOrderConnector
	wg               *sync.WaitGroup
}

func New(storage OrderProcessor, accrualConnector AccrualOrderConnector) Order {
	instance := Order{
		storage:          storage,
		accrualConnector: accrualConnector,
		wg:               &sync.WaitGroup{},
	}

	instance.wg.Add(1)
	go func() {
		instance.wg.Done()
		instance.updateOrders()
	}()

	return instance
}

func (p *Order) UploadOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := p.parseUserID(w, r)
		if err != nil {
			zap.L().Error("error while parsing user id while uploading order", zap.Error(err))
			return
		}

		orderNumber, err := p.parseOrderNumber(w, r)
		if err != nil {
			zap.L().Error("error while parsing order number while uploading order", zap.Error(err))
			return
		}

		storageUserID, err := p.uploadOrder(userID, orderNumber, w)
		if err != nil {
			if errors.Is(err, err_storage.ErrOrderNumberExists) {
				zap.L().Info(
					"order number exists in storage while uploading one",
					zap.String("user_id", userID.String()),
					zap.String("order_user_id", storageUserID.String()),
					zap.String("order_number", string(orderNumber)),
				)
			} else {
				zap.L().Error("error while uploading order number to storage while uploading order", zap.Error(err))
			}
			return
		}

		zap.L().Debug("UploadOrder number", zap.String("number", string(orderNumber)))
		zap.L().Info(
			"upload order number to storage",
			zap.String("user_id", userID.String()),
			zap.String("order_user_id", storageUserID.String()),
			zap.String("order_number", string(orderNumber)),
		)

		p.accrualConnector.SetInput(entity.CreateAccrualRequest(userID, orderNumber))

		p.validateUploadOrderResult(userID, storageUserID, w)
	}
}

func (p *Order) GetUserOrders() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := p.parseUserID(w, r)
		if err != nil {
			zap.L().Error("error while parsing user id while uploading order", zap.Error(err))
			return
		}

		orders, err := p.getUserOrders(userID, w)
		if err != nil {
			return
		}

		p.updateAccrualState(userID, orders)

		p.sendUserOrders(orders, w)
	}
}

func (p *Order) GetUserBalance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := p.parseUserID(w, r)
		if err != nil {
			zap.L().Error("error while parsing user id while getting user balance", zap.Error(err))
			return
		}

		balance, err := p.getUserBalance(userID, w)
		if err != nil {
			zap.L().Error("error while getting user balance", zap.Error(err))
			return
		}

		p.sendUserBalance(balance, w)
	}
}

func (p *Order) WithdrawBonuses() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := p.parseUserID(w, r)
		if err != nil {
			zap.L().Error("error while parsing user id while withdrawing user bonuses", zap.Error(err))
			return
		}

		withdraw, err := p.parseUserWithdraw(w, r)
		if err != nil {
			zap.L().Error("error while parsing user withdraw", zap.Error(err))
		}

		p.withdrawUserBonuses(userID, withdraw, w)
	}
}

func (p *Order) GetUserWithdrawals() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := p.parseUserID(w, r)
		if err != nil {
			zap.L().Error("error while parsing user id while withdrawing user bonuses", zap.Error(err))
			return
		}

		withdrawals, err := p.getUserWithdrawals(userID, w)
		if err != nil {
			return
		}

		p.sendUserWithdrawals(withdrawals, w)
	}
}

func (p *Order) Stop() {
	ready := make(chan bool)
	go func() {
		defer close(ready)
		p.wg.Wait()
	}()

	// устанавливаем таймаут на ожидание сброса в БД последней порции
	select {
	case <-time.After(stopTimeout):
		zap.L().Error("timeout stopped while sending data for update orders to the storage while shutting down")
		return
	case <-ready:
		zap.L().Info("succsessful sending data for update orders to the storage while shutting down")
		return
	}
}

func (p *Order) sendUserWithdrawals(withdrawals entity.Withdrawals, w http.ResponseWriter) {
	outWithdrawals := converter.ConvertWithdrawToWithdrawResponse(withdrawals)

	out, err := json.Marshal(outWithdrawals)
	if err != nil {
		zap.L().Error("error while marshalling user withdrawals", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(out)
}

func (p *Order) getUserWithdrawals(userID entity.UserID, w http.ResponseWriter) (entity.Withdrawals, error) {
	ctx, cancel := context.WithTimeout(context.Background(), httputils.RequestTimeout)
	defer cancel()

	withdrawals, err := p.storage.GetUserWithdrawals(ctx, userID)
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

func (p *Order) withdrawUserBonuses(userID entity.UserID, withdraw entity.Withdraw, w http.ResponseWriter) {
	ctx, cancel := context.WithTimeout(context.Background(), httputils.RequestTimeout)
	defer cancel()

	err := p.storage.WithdrawUser(ctx, userID, withdraw)
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

func (p *Order) parseUserWithdraw(w http.ResponseWriter, r *http.Request) (entity.Withdraw, error) {
	bodyResult, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return entity.Withdraw{}, fmt.Errorf("error while reading request body :%w", err)
	}
	defer r.Body.Close()

	var withdraw model.WithdrawRequest
	err = json.Unmarshal(bodyResult, &withdraw)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return entity.Withdraw{}, fmt.Errorf("error while unmarshal request body :%w", err)
	}

	orderNumber := entity.OrderNumber(withdraw.Order)
	isValid := validator.OrderNumberValidation(orderNumber)
	if !isValid {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return entity.Withdraw{}, fmt.Errorf("order number = %s is invalid while parse user withdraw", orderNumber)
	}

	return converter.ConvertRequestWithdrawToEntity(withdraw), nil
}

func (p *Order) updateOrders() {
	ticker := time.NewTicker(tickerTime)
	orders := make(entity.Orders, 0, flushBufLen)
	balance := make(map[entity.UserID]float64, flushBufLen)

	flushOrders := func() {
		if len(orders) != 0 {
			p.updateOrdersStorage(orders)
			orders = orders[:0]
		}
		if len(balance) != 0 {
			err := p.updateBalanceStorage(balance)
			if err != nil {
				zap.L().Error("error while updating balance batch", zap.Error(err))
				return
			}

			clear(balance)
		}
	}

	for {
		select {
		case <-ticker.C:
			flushOrders()
		default:
			accrualOrder, ok := p.accrualConnector.GetOutput()
			if !ok {
				zap.L().Info("output channel from accrual connector has been closed")
				flushOrders()
				p.accrualConnector.CloseInput()
				return
			}

			if entity.StatusPause == accrualOrder.Status {
				zap.L().Debug("accrual paused")
				flushOrders()
				continue
			}

			zap.L().Debug(
				"accrual order status",
				zap.String("order_number", string(accrualOrder.Order.Number)),
				zap.String("status", accrualOrder.Order.Status),
				zap.String("user_id", string(accrualOrder.UserID.String())),
			)

			if string(model.StatusProcessedAccrual) == accrualOrder.Order.Status {
				balance[accrualOrder.UserID] += accrualOrder.Order.Accrual
			}

			if model.StatusRegisteredAccrual == model.AccrualOrderStatus(accrualOrder.Order.Status) {
				zap.L().Debug("accrual order registered", zap.String("number", string(accrualOrder.Order.Number)))
				continue
			}

			zap.L().Debug("accrual order successfully appended", zap.String("number", string(accrualOrder.Order.Number)))

			orders = append(orders, accrualOrder.Order)
			if len(orders) == cap(orders) {
				flushOrders()
			}
		}
	}
}

func (p *Order) updateBalanceStorage(userBalances map[entity.UserID]float64) error {
	var balances entity.UpdateUserBalances
	for userID, balance := range userBalances {
		balances = append(balances, entity.UserBalance{
			UserID: userID,
			Balance: balance,
		})
	}

	ctx, close := context.WithTimeout(context.Background(), httputils.UpdateTimeout)
	defer close()

	err := p.storage.UpdateBalanceBatch(ctx, balances)
	if err != nil {
		return fmt.Errorf("error while updating balance batch: %w", err)
	}

	return nil
}

func (p *Order) updateOrdersStorage(orders entity.Orders) {
	ctx, close := context.WithTimeout(context.Background(), httputils.UpdateTimeout)
	defer close()

	err := p.storage.UpdateOrders(ctx, orders)
	if err != nil {
		zap.L().Error("error while updating orders", zap.Error(err))
	}
}

func (p *Order) updateAccrualState(userID entity.UserID, orders entity.Orders) {
	for _, order := range orders {
		if order.Status == string(model.StatusNewOrder) || order.Status == string(model.StatusProcessingOrder) {
			p.accrualConnector.SetInput(entity.CreateAccrualRequest(userID, order.Number))
		}
	}
}

func (p *Order) getUserBalance(userID entity.UserID, w http.ResponseWriter) (entity.UserBalance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), httputils.RequestTimeout)
	defer cancel()

	balance, err := p.storage.GetUserBalance(ctx, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return entity.UserBalance{}, fmt.Errorf("error while getting user balance: %w", err)
	}

	return balance, nil
}

func (p *Order) getUserOrders(userID entity.UserID, w http.ResponseWriter) (entity.Orders, error) {
	ctx, cancel := context.WithTimeout(context.Background(), httputils.RequestTimeout)
	defer cancel()

	orders, err := p.storage.GetUserOrders(ctx, userID)
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

func (p *Order) sendUserBalance(balance entity.UserBalance, w http.ResponseWriter) {
	outBalance := converter.ConvertStorageBalanceToOutput(balance)

	out, err := json.Marshal(outBalance)
	if err != nil {
		zap.L().Error("error while marshalling user balance", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(out)
}

func (p *Order) sendUserOrders(orders entity.Orders, w http.ResponseWriter) {
	outOrders, err := converter.ConvertStorageOrdersToOutputUploadedOrders(orders)
	if err != nil {
		zap.L().Error("error while converting user orders to output model", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	out, err := json.Marshal(outOrders)
	if err != nil {
		zap.L().Error("error while marshalling user orders", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(out)
}

func (p *Order) uploadOrder(userID entity.UserID, orderNumber entity.OrderNumber, w http.ResponseWriter) (entity.UserID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), httputils.RequestTimeout)
	defer cancel()

	storageUserID, err := p.storage.UploadOrder(ctx, userID, orderNumber)
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

func (p *Order) validateUploadOrderResult(userID entity.UserID, storageUserID entity.UserID, w http.ResponseWriter) {
	if len(storageUserID.String()) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	if storageUserID == userID {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusConflict)
}

func (p *Order) parseOrderNumber(w http.ResponseWriter, r *http.Request) (entity.OrderNumber, error) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return entity.OrderNumber(""), fmt.Errorf("error while request body parsing: %w", err)
	}
	defer r.Body.Close()

	orderNumber := entity.OrderNumber(string(data))
	isValid := validator.OrderNumberValidation(orderNumber)
	if !isValid {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return entity.OrderNumber(""), fmt.Errorf("order number = %s is invalid", orderNumber)
	}

	return orderNumber, nil
}

func (p *Order) parseUserID(w http.ResponseWriter, r *http.Request) (entity.UserID, error) {
	userIDCtx, ok := r.Context().Value(entity.UserIDCtxKey{}).(entity.UserIDCtx)

	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return entity.UserID(""), fmt.Errorf("user id couldn't obtain from context")
	}

	if userIDCtx.StatusCode == http.StatusBadRequest {
		http.Error(w, ErrInvalidAuth, http.StatusUnauthorized)
		return entity.UserID(""), fmt.Errorf("failed auth credentials")
	}

	if userIDCtx.StatusCode == http.StatusUnauthorized {
		http.Error(w, ErrTokenExpired, http.StatusUnauthorized)
		return entity.UserID(""), fmt.Errorf(ErrTokenExpired)
	}

	if userIDCtx.StatusCode == http.StatusOK && !userIDCtx.UserID.Valid() {
		http.Error(w, ErrInvalidAuth, http.StatusUnauthorized)
		return entity.UserID(""), fmt.Errorf("invalid user id with status ok")
	}

	return userIDCtx.UserID, nil
}
