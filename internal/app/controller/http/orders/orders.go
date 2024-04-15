package orders

import (
	"context"
	"fmt"
	"io"
	"net/http"

	httputils "github.com/avGenie/go-loyalty-system/internal/app/controller/http/utils"
	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	"github.com/avGenie/go-loyalty-system/internal/app/usecase/validator"
	"go.uber.org/zap"
)

const (
	ErrTokenExpired = "token has expired"
	ErrInvalidAuth  = "auth credentials are invalid"
)

type OrderProcessor interface {
	UploadOrder(ctx context.Context, userID entity.UserID, orderNumber entity.OrderNumber) (entity.UserID, error)
	GetUserOrders(ctx context.Context, userID entity.UserID) (entity.Orders, error)
}

type Order struct {
	storage OrderProcessor
}

func New(storage OrderProcessor) Order {
	return Order{
		storage: storage,
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

		storageUserID, err := p.uploadOrder(userID, orderNumber, w)
		if err != nil {
			zap.L().Error("error while uploading order number to storage while uploading order", zap.Error(err))
			return
		}

		p.validateUploadOrderResult(userID, storageUserID, w)
	}
}

func (p *Order) uploadOrder(userID entity.UserID, orderNumber entity.OrderNumber, w http.ResponseWriter) (entity.UserID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), httputils.RequestTimeout)
	defer cancel()

	storageUserID, err := p.storage.UploadOrder(ctx, userID, orderNumber)
	if err != nil {
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

	orderNumber := entity.OrderNumber(data)
	err = validator.OrderNumberValidation(orderNumber)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return entity.OrderNumber(""), fmt.Errorf("error while validating order number = %s: %w", orderNumber, err)
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
