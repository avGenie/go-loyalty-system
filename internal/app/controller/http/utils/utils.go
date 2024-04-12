package httputils

import (
	"fmt"
	"net/http"
	"time"

	"github.com/avGenie/go-loyalty-system/internal/app/entity"
)

const (
	RequestTimeout = 3*time.Second
)

func GetUserIDFromContext(r *http.Request) (entity.UserID, error) {
	userIDCtx, ok := r.Context().Value(entity.UserIDCtxKey{}).(entity.UserIDCtx)
	if !ok {
		return entity.UserID(""), fmt.Errorf("user id couldn't obtain from context")
	}

	if userIDCtx.StatusCode == http.StatusOK && !userIDCtx.UserID.Valid() {
		return entity.UserID(""), fmt.Errorf("invalid user id with status ok")
	}

	return userIDCtx.UserID, nil
}
