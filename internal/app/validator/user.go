package validator

import "github.com/avGenie/go-loyalty-system/internal/app/model"

func CreateUserRequest(user model.CreateUserRequest) bool {
	return len(user.Login) > 0 && len(user.Password) > 0
}