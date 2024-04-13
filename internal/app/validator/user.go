package validator

import "github.com/avGenie/go-loyalty-system/internal/app/model"

func ValidateCreateUserRequest(user model.UserCredentialsRequest) bool {
	return len(user.Login) > 0 && len(user.Password) > 0
}
