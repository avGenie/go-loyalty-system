package entity

import (
	"fmt"
	"net/http"
)

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

func CreateUserIDCtx(userID UserID, code int) UserIDCtx {
	return UserIDCtx{
		UserID:     userID,
		StatusCode: code,
	}
}

func ValidateCookieUserID(cookie *http.Cookie) (UserID, error) {
	rawUserID := cookie.Value

	if len(rawUserID) == 0 {
		return "", fmt.Errorf("cookie of user id is empty")
	}

	return UserID(rawUserID), nil
}
