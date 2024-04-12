package entity

import "github.com/avGenie/go-loyalty-system/internal/app/model"

type UserID string

type User struct {
	ID       UserID
	Login    string
	Password string
}

type UserIDCtxKey struct{}

type UserIDCtx struct {
	UserID     UserID
	StatusCode int
}

func (u UserID) String() string {
	return string(u)
}

func (u *UserID) Valid() bool {
	return len(u.String()) != 0
}

func CreateUserIDCtx(userID UserID, code int) UserIDCtx {
	return UserIDCtx{
		UserID:     userID,
		StatusCode: code,
	}
}

func CreateUserFromCreateRequest(userID UserID, request model.CreateUserRequest) User {
	return User{
		ID: userID,
		Login: request.Login,
		Password: request.Password,
	}
}
