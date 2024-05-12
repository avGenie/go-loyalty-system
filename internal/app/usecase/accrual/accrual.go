package accrual

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/avGenie/go-loyalty-system/internal/app/config"
	"github.com/avGenie/go-loyalty-system/internal/app/converter"
	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	"github.com/avGenie/go-loyalty-system/internal/app/model"
	"go.uber.org/zap"
)

const (
	accrualGetOrder   = `/api/orders/`
	retryAfterDefault = 60
)

var (
	ErrUnexpectedAccrualError = errors.New("unexpected accrual error")
	ErrAccrualAddressInvalid  = errors.New("accrual address invalid")
	ErrOrderNotRegister       = errors.New("order is not register in accrual system")
	ErrRequestsExceeded       = errors.New("number of requests to accrual has been exceeded")
)

type Accrual struct {
	client http.Client

	requestAddress string
}

func New(config config.Config) *Accrual {
	client := http.Client{
		Timeout: 3 * time.Second,
	}

	requestAddress := fmt.Sprintf("%s%s", config.AccrualAddr, accrualGetOrder)

	return &Accrual{
		client:         client,
		requestAddress: requestAddress,
	}
}

func (a *Accrual) MakeRequest(userID entity.UserID, number entity.OrderNumber) (entity.AccrualOrder, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", a.requestAddress, number), nil)
	if err != nil {
		return entity.AccrualOrder{}, fmt.Errorf("cannot create request for accrual service: %w", err)
	}
	res, err := a.client.Do(req)
	if err != nil {
		return entity.AccrualOrder{}, fmt.Errorf("cannot create request to accrual service: %w", err)
	}

	order, err := a.processAccrualResponse(res)
	if err != nil {
		return entity.AccrualOrder{}, fmt.Errorf("cannot process accrual response: %w", err)
	}

	order.UserID = userID
	
	return order, nil
}

func (a *Accrual) processAccrualResponse(res *http.Response) (entity.AccrualOrder, error) {
	status := res.StatusCode
	if status == http.StatusNoContent {
		return entity.AccrualOrder{
			Status: entity.StatusOrderNotRegistered,
		}, nil
	} else if status == http.StatusTooManyRequests {
		retryTime, err := strconv.Atoi(res.Header.Get("Retry-After"))
		if err != nil {
			zap.L().Error("error while parsing retry after time from accrual system", zap.Error(err))
			retryTime = retryAfterDefault
		}

		return entity.AccrualOrder{
			RetryAfter: time.Duration(retryTime) * time.Second,
			Status: entity.StatusPause,
		}, nil
	}

	if http.StatusOK != status {
		return entity.AccrualOrder{
			Status: entity.StatusError,
		}, fmt.Errorf("unexpected status from accrual system: %d", status)
	}

	var response model.AccrualResponse
	err := json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return entity.AccrualOrder{
			Status: entity.StatusError,
		}, fmt.Errorf("error while decoding accrual response: %w", err)
	}
	res.Body.Close()

	return entity.AccrualOrder{
		Order: converter.ConvertAccrualResponseToOrder(response),
		Status: entity.StatusOK,
	}, nil
}
