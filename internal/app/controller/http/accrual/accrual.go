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
	ErrUnexpectedAccrualError = errors.New("unexpected accrual error")
	ErrAccrualAddressInvalid  = errors.New("accrual address invalid")
	ErrOrderNotRegister       = errors.New("order is not register in accrual system")
	ErrRequestsExceeded       = errors.New("number of requests to accrual has been exceeded")
)

type Accrual struct {
	client http.Client

	requestAddress string
	wg             sync.WaitGroup
	connector      *AccrualConnector
	storage        *AccrualStorage
	done           chan struct{}
}

func New(connector *AccrualConnector, config config.Config) (*Accrual, error) {
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
		storage:        NewStorage(),
		done:           make(chan struct{}),
	}

	instance.wg.Add(2)
	go func() {
		defer instance.wg.Done()
		instance.getRequest()
	}()

	go func() {
		defer instance.wg.Done()
		instance.processRequests()
	}()

	return instance, nil
}

func (a *Accrual) Stop() {
	sync.OnceFunc(func() {
		close(a.done)
	})()

	ready := make(chan bool)
	go func() {
		defer close(ready)
		a.wg.Wait()
	}()

	zap.L().Info("accrual service has been stopped")
}

func (a *Accrual) getRequest() {
	for {
		request, ok := a.connector.GetInput()
		if !ok {
			zap.L().Info("input channel has been closed for accrual service")
			return
		}

		zap.L().Debug("accrual get request", zap.String("number", string(request.Number)), zap.String("user_id", request.UserID.String()))

		err := a.storage.Add(request)
		if err != nil {
			if !errors.Is(err, ErrEmptyStorageSpace) {
				zap.L().Error("error while pushing number to accrual storage")
			}
		}
	}
}

func (a *Accrual) processRequests() {
	for {
		select {
		case <-a.done:
			a.connector.CloseOutput()
			return
		default:
			request, err := a.storage.Get()
			if err != nil {
				if !errors.Is(err, ErrEmptyStorage) {
					zap.L().Error("error while getting request from storage for accrual system")
				}
				continue
			}

			zap.L().Debug("accrual process request", zap.String("number", string(request.Number)), zap.String("user_id", request.UserID.String()))

			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s", a.requestAddress, request.Number), nil)
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
				if errors.Is(err, ErrOrderNotRegister) || errors.Is(err, ErrUnexpectedAccrualError) {
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

			a.connector.SetOutput(entity.CreateProcessingAccrualOrder(request.UserID, accrualResp.Order))
		}

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

	if http.StatusOK != status {
		zap.L().Error("unexpected status from accrual system", zap.Int("status", status))
		return entity.AccrualProcessingResponse{}, ErrUnexpectedAccrualError
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
