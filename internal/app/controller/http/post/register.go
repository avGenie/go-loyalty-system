package post

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	httputils "github.com/avGenie/go-loyalty-system/internal/app/controller/http/utils"
	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	"github.com/avGenie/go-loyalty-system/internal/app/model"
	err_storage "github.com/avGenie/go-loyalty-system/internal/app/storage/api/errors"
	usecase "github.com/avGenie/go-loyalty-system/internal/app/usecase/converter"
	"github.com/avGenie/go-loyalty-system/internal/app/usecase/crypto"
	"go.uber.org/zap"
)

type UserCreator interface {
	CreateUser(ctx context.Context, user entity.User) error
}

func CreateUser(creator UserCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := httputils.GetUserIDFromContext(r)
		if err != nil {
			zap.L().Error("error while parsing user id while user creation", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		createRequest := model.CreateUserRequest{}
		err = json.NewDecoder(r.Body).Decode(&createRequest)
		if err != nil {
			zap.L().Error("error while parsing create user request while user creation", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		user := entity.CreateUserFromCreateRequest(userID, createRequest)
		hashedPassword, err := crypto.HashPassword(user.Password)
		if err != nil {
			zap.L().Error("error while hashing password while user creation", zap.Error(err), zap.String("user_password", user.Password))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		user.Password = hashedPassword

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

		token, err := usecase.SetUserIDToAuthHeaderFormat(userID)
		if err != nil {
			zap.L().Error("error while preparing auth header while creating user", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Add("Authorization", token)
		w.WriteHeader(http.StatusOK)
	}
}
