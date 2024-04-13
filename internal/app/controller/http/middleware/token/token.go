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

		authHeader := r.Header[usecase.AuthHeader]
		userCtx := processAuthUserID(authHeader)

		ctx := context.WithValue(r.Context(), entity.UserIDCtxKey{}, userCtx)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func processAuthUserID(authHeader []string) entity.UserIDCtx {
	if len(authHeader) == 0 {
		zap.L().Info("authorization header is empty")

		return entity.CreateUserIDCtx("", http.StatusUnauthorized)
	}

	userID, err := usecase.GetUserIDFromAuthHeader(authHeader[0])
	if err != nil {
		zap.L().Error("error while parsing auth header", zap.Error(err), zap.String("header", authHeader[0]))

		return entity.CreateUserIDCtx("", http.StatusUnauthorized)
	}

	if !userID.Valid() {
		zap.L().Error("empty user id in authorization header")
		
		return entity.CreateUserIDCtx("", http.StatusBadRequest)
	}
	
	return entity.CreateUserIDCtx(userID, http.StatusOK)
}
