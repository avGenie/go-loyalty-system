package token

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	usecase "github.com/avGenie/go-loyalty-system/internal/app/usecase/converter"
	err_usecase "github.com/avGenie/go-loyalty-system/internal/app/usecase/errors"
	"go.uber.org/zap"
)

func TokenParserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zap.L().Info("start token parsing")

		authHeader := r.Header[usecase.AuthHeader]
		userCtx := processAuthUserID(authHeader)

		fmt.Println(userCtx)

		ctx := context.WithValue(r.Context(), entity.UserIDCtxKey{}, userCtx)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func processAuthUserID(authHeader []string) entity.UserIDCtx {
	if len(authHeader) == 0 {
		zap.L().Info("authorization header is empty")

		return entity.CreateUserIDCtx("", http.StatusBadRequest)
	}

	userID, err := usecase.GetUserIDFromAuthHeader(authHeader[0])
	if err != nil {
		zap.L().Error("error while parsing auth header", zap.Error(err), zap.String("header", authHeader[0]))
		if errors.Is(err, err_usecase.ErrTokenExpired) {
			return entity.CreateUserIDCtx("", http.StatusUnauthorized)
		}

		return entity.CreateUserIDCtx("", http.StatusBadRequest)
	}

	if !userID.Valid() {
		zap.L().Error("empty user id in authorization header")

		return entity.CreateUserIDCtx("", http.StatusBadRequest)
	}

	return entity.CreateUserIDCtx(userID, http.StatusOK)
}
