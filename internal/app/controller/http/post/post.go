package post

import (
	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	"github.com/google/uuid"
)

const (
	ErrEmptyUserRequest = "wrong user credentials format: empty login or password"
)

func createUserID() entity.UserID {
	uuid := uuid.New()
	userID := entity.UserID(uuid.String())

	return userID
}
