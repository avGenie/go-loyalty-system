package accrual

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
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
	ErrAccrualAddressInvalid = errors.New("accrual address invalid")
	ErrOrderNotRegister      = errors.New("order is not register in accrual system")
	ErrRequestsExceeded      = errors.New("number of requests to accrual has been exceeded")
)

type Accrual struct {
	client http.Client

	requestAddress string
	wg             sync.WaitGroup
	connector      AccrualConnector
}

func New(connector AccrualConnector, config config.Config) (*Accrual, error) {
	if len(config.AccrualAddr) == 0 {
		return nil, ErrAccrualAddressInvalid
	}

	client := http.Client{
		Timeout: 3 * time.Second,
	}

	requestAddress := fmt.Sprintf("%s%s", config.AccrualAddr, accrualGetOrder)

	instance := &Accrual{
		client:         client,
		requestAddress: requestAddress,
		wg:             sync.WaitGroup{},
		connector:      connector,
	}

	instance.wg.Add(1)
	go func() {
		defer instance.wg.Done()
		instance.getRequest()
	}()

	return instance, nil
}

func (a *Accrual) getRequest() {
	for {
		number, ok := a.connector.GetInput()
		if !ok {
			zap.L().Error("input channel has been closed for accrual service")
			return
		}

		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", a.requestAddress, number), nil)
		if err != nil {
			zap.L().Error("cannot create request for accrual service", zap.Error(err))
			continue
		}
		res, err := a.client.Do(req)
		if err != nil {
			zap.L().Error("cannot create request to accrual service", zap.Error(err))
			continue
		}

		accrualResp, err := a.processAccrualResponse(res)
		if err != nil {
			if errors.Is(err, ErrOrderNotRegister) {
				continue
			} else if !errors.Is(err, ErrRequestsExceeded) {
				zap.L().Error("error while processing accrual response", zap.Error(err))
				continue
			}

			zap.L().Info("accrual service exceeded", zap.String("retry_after", accrualResp.RetryAfter.String()))
			a.connector.SetOutput(entity.CreatePausedAccrualOrder())
			<-time.After(accrualResp.RetryAfter)
			continue
		}

		a.connector.SetOutput(entity.CreateProcessingAccrualOrder(accrualResp.Order))
	}
}

func (a *Accrual) processAccrualResponse(res *http.Response) (entity.AccrualProcessingResponse, error) {
	status := res.StatusCode
	if status == http.StatusNoContent {
		return entity.AccrualProcessingResponse{}, ErrOrderNotRegister
	} else if status == http.StatusTooManyRequests {
		retryTime, err := strconv.Atoi(res.Header.Get("Retry-After"))
		if err != nil {
			zap.L().Error("error while parsing retry after time from accrual system", zap.Error(err))
			retryTime = retryAfterDefault
		}

		return entity.AccrualProcessingResponse{
			RetryAfter: time.Duration(retryTime) * time.Second,
		}, ErrRequestsExceeded
	}

	var response model.AccrualResponse
	err := json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		err = fmt.Errorf("error while decoding accrual response: %w", err)
		return entity.AccrualProcessingResponse{}, err
	}
	res.Body.Close()

	return entity.AccrualProcessingResponse{
		Order: converter.ConvertAccrualResponseToOrder(response),
	}, nil
}
