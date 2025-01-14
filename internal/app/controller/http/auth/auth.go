package auth

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	"github.com/avGenie/go-loyalty-system/internal/app/model"
	"github.com/avGenie/go-loyalty-system/internal/app/usecase/auth"
	usecase "github.com/avGenie/go-loyalty-system/internal/app/usecase/converter"
	"github.com/avGenie/go-loyalty-system/internal/app/usecase/crypto"
	"github.com/avGenie/go-loyalty-system/internal/app/validator"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	ErrEmptyUserRequest = "wrong user credentials format: empty login or password"
)

type AuthUser struct {
	storage auth.UserAuthenticator
}

func New(storage auth.UserAuthenticator) AuthUser {
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

		err = auth.CreateUser(user, a.storage, w)
		if err != nil {
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

		storageUser, err := auth.AuthUser(inputUser, a.storage, w)
		if err != nil {
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
