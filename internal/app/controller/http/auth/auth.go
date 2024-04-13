package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	httputils "github.com/avGenie/go-loyalty-system/internal/app/controller/http/utils"
	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	"github.com/avGenie/go-loyalty-system/internal/app/model"
	err_storage "github.com/avGenie/go-loyalty-system/internal/app/storage/api/errors"
	usecase "github.com/avGenie/go-loyalty-system/internal/app/usecase/converter"
	"github.com/avGenie/go-loyalty-system/internal/app/usecase/crypto"
	"github.com/avGenie/go-loyalty-system/internal/app/validator"
	"go.uber.org/zap"
)

type UserCreator interface {
	CreateUser(ctx context.Context, user entity.User) error
}

func CreateUser(creator UserCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := createUserFromRequest(createUserID(), w, r)
		if err != nil {
			zap.L().Error("error while parsing user credentials while creating user", zap.Error(err))
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), httputils.RequestTimeout)
		defer cancel()

		err = creator.CreateUser(ctx, user)
		if err != nil {
			if errors.Is(err, err_storage.ErrLoginExists) {
				zap.L().Error("error while creating user", zap.Error(err), zap.String("login", user.Login))
				w.WriteHeader(http.StatusConflict)
				return
			}

			zap.L().Error("error while creating user", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		token, err := usecase.SetUserIDToAuthHeaderFormat(user.ID)
		if err != nil {
			zap.L().Error("error while preparing auth header while creating user", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Add(usecase.AuthHeader, token)
		w.WriteHeader(http.StatusOK)
	}
}

func createUserFromRequest(userID entity.UserID, w http.ResponseWriter, r *http.Request) (entity.User, error) {
	var userCreds model.UserCredentialsRequest
	err := json.NewDecoder(r.Body).Decode(&userCreds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return entity.User{}, fmt.Errorf("error while decoding user credentials request: %w", err)
	}
	defer r.Body.Close()

	if !validator.ValidateCreateUserRequest(userCreds) {
		http.Error(w, ErrEmptyUserRequest, http.StatusBadRequest)
		return entity.User{}, fmt.Errorf(ErrEmptyUserRequest)
	}

	user := entity.CreateUserFromCreateRequest(userID, userCreds)

	hashedPassword, err := crypto.HashPassword(user.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return entity.User{}, fmt.Errorf("error while hashing password: %w", err)
	}
	user.Password = hashedPassword

	return user, nil
}
