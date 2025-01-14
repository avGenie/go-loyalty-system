package orders

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/avGenie/go-loyalty-system/internal/app/config"
	"github.com/avGenie/go-loyalty-system/internal/app/converter"
	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	"github.com/avGenie/go-loyalty-system/internal/app/model"
	err_storage "github.com/avGenie/go-loyalty-system/internal/app/storage/api/errors"
	"github.com/avGenie/go-loyalty-system/internal/app/usecase/order"
	"github.com/avGenie/go-loyalty-system/internal/app/usecase/validator"
	"go.uber.org/zap"
)

const (
	ErrTokenExpired = "token has expired"
	ErrInvalidAuth  = "auth credentials are invalid"
)

const (
	stopTimeout = 5 * time.Second
)

type AccrualOrderConnector interface {
	SetInput(number entity.AccrualOrderRequest)
	CloseInput()
	GetOutput() (entity.AccrualOrder, bool)
}

type Order struct {
	storage       order.OrderProcessor
	statusUpdater *order.StatusUpdater
	wg            *sync.WaitGroup
}

func New(storage order.OrderProcessor, config config.Config) Order {
	instance := Order{
		storage:       storage,
		statusUpdater: order.CreateStatusUpdater(storage, config),
		wg:            &sync.WaitGroup{},
	}

	instance.wg.Add(1)
	go func() {
		instance.wg.Done()
		instance.statusUpdater.Start()
	}()

	return instance
}

func (p *Order) Stop() {
	p.statusUpdater.Stop()

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

		storageUserID, err := order.UploadOrder(userID, orderNumber, p.storage, w)
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

		w.WriteHeader(http.StatusAccepted)
	}
}

func (p *Order) GetUserOrders() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := p.parseUserID(w, r)
		if err != nil {
			zap.L().Error("error while parsing user id while uploading order", zap.Error(err))
			return
		}

		orders, err := order.GetUserOrders(userID, p.storage, w)
		if err != nil {
			return
		}

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

		balance, err := order.GetUserBalance(userID, p.storage, w)
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
			return
		}

		order.WithdrawUserBonuses(userID, withdraw, p.storage, w)
	}
}

func (p *Order) GetUserWithdrawals() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := p.parseUserID(w, r)
		if err != nil {
			zap.L().Error("error while parsing user id while withdrawing user bonuses", zap.Error(err))
			return
		}

		withdrawals, err := order.GetUserWithdrawals(userID, p.storage, w)
		if err != nil {
			return
		}

		p.sendUserWithdrawals(withdrawals, w)
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
