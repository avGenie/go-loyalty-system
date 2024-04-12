package entity

import (
	"fmt"
	"net/http"
)

const (
	UserIDKey = "user_id"
)

type UserIDCtxKey struct{}

type UserID string

type User struct {
	ID       UserID
	Login    string
	Password string
}

type UserIDCtx struct {
	UserID     UserID
	StatusCode int
}

func ValidateCookieUserID(cookie *http.Cookie) (UserID, error) {
	rawUserID := cookie.Value

	if len(rawUserID) == 0 {
		return "", fmt.Errorf("cookie of user id is empty")
	}

	return UserID(rawUserID), nil
}
