package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	err_storage "github.com/avGenie/go-loyalty-system/internal/app/storage/api/errors"
	"github.com/avGenie/go-loyalty-system/internal/app/usecase/crypto"
	httputils "github.com/avGenie/go-loyalty-system/internal/app/usecase/utils"
	"go.uber.org/zap"
)

const (
	ErrLoginNotExist    = "login doesn't exist"
	ErrWrongPassword    = "wrong password"
)

type UserAuthenticator interface {
	CreateUser(ctx context.Context, user entity.User) error
	GetUser(ctx context.Context, user entity.User) (entity.User, error)
}

func CreateUser(user entity.User, authenticator UserAuthenticator, w http.ResponseWriter) error {
	ctx, cancel := context.WithTimeout(context.Background(), httputils.RequestTimeout)
	defer cancel()

	err := authenticator.CreateUser(ctx, user)
	if err != nil {
		if errors.Is(err, err_storage.ErrLoginExists) {
			zap.L().Error("error while creating user", zap.Error(err), zap.String("login", user.Login))
			w.WriteHeader(http.StatusConflict)
			return fmt.Errorf("error while creating user: %w", err)
		}

		zap.L().Error("error while creating user", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return fmt.Errorf("error while creating user: %w", err)
	}

	return nil
}

func AuthUser(inputUser entity.User, authenticator UserAuthenticator, w http.ResponseWriter) (entity.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), httputils.RequestTimeout)
	defer cancel()

	storageUser, err := authenticator.GetUser(ctx, inputUser)
	if err != nil {
		zap.L().Error("error while getting user while authentication request", zap.Error(err))

		if errors.Is(err, err_storage.ErrLoginNotFound) {
			http.Error(w, ErrLoginNotExist, http.StatusUnauthorized)
			return entity.User{}, err
		}

		w.WriteHeader(http.StatusInternalServerError)
		return entity.User{}, err
	}

	err = crypto.CheckPasswordHash(inputUser.Password, storageUser.Password)
	if err != nil {
		zap.L().Error("error while checking user password while authentication request", zap.Error(err))
		if errors.Is(err, crypto.ErrWrongPassword) {
			http.Error(w, ErrLoginNotExist, http.StatusUnauthorized)
			return entity.User{}, err
		}

		w.WriteHeader(http.StatusInternalServerError)
		return entity.User{}, err
	}

	return storageUser, nil
}
