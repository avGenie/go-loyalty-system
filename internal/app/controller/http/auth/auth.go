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
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	ErrEmptyUserRequest = "wrong user credentials format: empty login or password"
	ErrLoginNotExist    = "login doesn't exist"
	ErrWrongPassword    = "wrong password"
)

type UserAuthenticator interface {
	CreateUser(ctx context.Context, user entity.User) error
	GetUser(ctx context.Context, user entity.User) (entity.User, error)
}

type AuthUser struct {
	storage UserAuthenticator
}

func New(storage UserAuthenticator) AuthUser {
	return AuthUser{
		storage: storage,
	}
}

func (a *AuthUser) CreateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := a.createUserFromRequestPassHashed(a.createUserID(), w, r)
		if err != nil {
			zap.L().Error("error while parsing user credentials while creating user", zap.Error(err))
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), httputils.RequestTimeout)
		defer cancel()

		err = a.storage.CreateUser(ctx, user)
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

		a.setUserIDToHeader(user.ID, w)
	}
}

func (a *AuthUser) AuthenticateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inputUser, err := a.createUserFromRequest(entity.UserID(""), w, r)
		if err != nil {
			zap.L().Error("error while parsing user credentials while creating user", zap.Error(err))
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), httputils.RequestTimeout)
		defer cancel()

		storageUser, err := a.storage.GetUser(ctx, inputUser)
		if err != nil {
			zap.L().Error("error while getting user while authentication request", zap.Error(err))

			if errors.Is(err, err_storage.ErrLoginNotFound) {
				http.Error(w, ErrLoginNotExist, http.StatusUnauthorized)
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = crypto.CheckPasswordHash(inputUser.Password, storageUser.Password)
		if err != nil {
			zap.L().Error("error while checking user password while authentication request", zap.Error(err))
			if errors.Is(err, crypto.ErrWrongPassword) {
				http.Error(w, ErrLoginNotExist, http.StatusUnauthorized)
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		a.setUserIDToHeader(storageUser.ID, w)
	}
}

func (a *AuthUser) createUserFromRequestPassHashed(userID entity.UserID, w http.ResponseWriter, r *http.Request) (entity.User, error) {
	user, err := a.createUserFromRequest(userID, w, r)
	if err != nil {
		return user, err
	}

	hashedPassword, err := crypto.HashPassword(user.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return entity.User{}, fmt.Errorf("error while hashing password: %w", err)
	}
	user.Password = hashedPassword

	return user, nil
}

func (a *AuthUser) createUserFromRequest(userID entity.UserID, w http.ResponseWriter, r *http.Request) (entity.User, error) {
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

	return user, nil
}

func (a *AuthUser) setUserIDToHeader(userID entity.UserID, w http.ResponseWriter) {
	token, err := usecase.SetUserIDToAuthHeaderFormat(userID)
	if err != nil {
		zap.L().Error("error while preparing auth header", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Add(usecase.AuthHeader, token)
	w.WriteHeader(http.StatusOK)
}

func (a *AuthUser) createUserID() entity.UserID {
	uuid := uuid.New()
	userID := entity.UserID(uuid.String())

	return userID
}
