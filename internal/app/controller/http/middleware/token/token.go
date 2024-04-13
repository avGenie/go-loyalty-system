package token

import (
	"context"
	"net/http"

	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	usecase "github.com/avGenie/go-loyalty-system/internal/app/usecase/converter"
	"go.uber.org/zap"
)

func TokenParserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zap.L().Info("start token parsing")

		var userCtx entity.UserIDCtx

		authHeader := r.Header["Authorization"]
		if len(authHeader) == 0 {
			zap.L().Info("authorization header is empty")

			userCtx = entity.CreateUserIDCtx("", http.StatusUnauthorized)
		} else {
			userID, err := usecase.GetUserIDFromAuthHeader(authHeader[0])
			if err != nil {
				zap.L().Error("error while parsing auth header", zap.Error(err), zap.String("header", authHeader[0]))

				userCtx = entity.CreateUserIDCtx("", http.StatusUnauthorized)
			} else {
				userCtx = entity.CreateUserIDCtx(userID, http.StatusOK)
			}
		}

		ctx := context.WithValue(r.Context(), entity.UserIDCtxKey{}, userCtx)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
