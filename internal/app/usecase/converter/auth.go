package usecase

import (
	"fmt"
	"strings"

	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	"github.com/avGenie/go-loyalty-system/internal/app/usecase/crypto"
)

const (
	bearerHeader = "Bearer"

	AuthHeader = "Authorization"
)

func GetUserIDFromAuthHeader(header string) (entity.UserID, error) {
	headerParts := strings.Split(header, " ")
	if len(headerParts) != 2 {
		return entity.UserID(""), fmt.Errorf("auth header doesn't contain two parts")
	}

	if headerParts[0] != bearerHeader {
		return entity.UserID(""), fmt.Errorf("first auth header part is invalid")
	}

	userID, err := crypto.GetUserID(headerParts[1])
	if err != nil {
		return entity.UserID(""), fmt.Errorf("error while getting user id from token: %w", err)
	}

	return userID, nil
}

func SetUserIDToAuthHeaderFormat(userID entity.UserID) (string, error) {
	token, err := crypto.BuildJWTString(userID)
	if err != nil {
		return "", fmt.Errorf("error while creating jwt token: %w", err)
	}

	return fmt.Sprintf("%s %s", bearerHeader, token), nil
}
