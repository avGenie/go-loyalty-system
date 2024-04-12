package http

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	"github.com/avGenie/go-loyalty-system/internal/app/usecase/token"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	bearerHeader = "Bearer"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zap.L().Info("start user authentication")

		var userCtx entity.UserIDCtx

		authHeader := r.Header["Authorization"]
		if len(authHeader) == 0 {
			zap.L().Info("authorization header is empty")

			userCtx = entity.CreateUserIDCtx(createUserID(), http.StatusUnauthorized)
		} else {
			userID, err := parseAuthHeader(authHeader[0])
			if err != nil {
				zap.L().Error("error while parsing auth header", zap.Error(err), zap.String("header", authHeader[0]))

				userCtx = entity.CreateUserIDCtx(createUserID(), http.StatusUnauthorized)
			} else {
				userCtx = entity.CreateUserIDCtx(userID, http.StatusOK)
			}
		}

		ctx := context.WithValue(r.Context(), entity.UserIDCtxKey{}, userCtx)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func createUserID() entity.UserID {
	uuid := uuid.New()
	userID := entity.UserID(uuid.String())

	return userID
}

func parseAuthHeader(header string) (entity.UserID, error) {
	headerParts := strings.Split(header, " ")
	if len(headerParts) != 2 {
		return entity.UserID(""), fmt.Errorf("auth header doesn't contain two parts")
	}

	if headerParts[0] != bearerHeader {
		return entity.UserID(""), fmt.Errorf("first auth header part is invalid")
	}

	userID, err := token.GetUserID(headerParts[1])
	if err != nil {
		return entity.UserID(""), fmt.Errorf("error while getting user id from token: %w", err)
	}

	return userID, nil
}
