package token

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	usecase "github.com/avGenie/go-loyalty-system/internal/app/usecase/converter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenParserMiddleware(t *testing.T) {
	type want struct {
		statusCode int
		userID     string
	}
	tests := []struct {
		name   string
		userID string

		want want
	}{
		{
			name:   "correct input data",
			userID: "00308dff-b6b1-4f1b-8515-d09d3db49951",

			want: want{
				statusCode: http.StatusOK,
				userID:     "00308dff-b6b1-4f1b-8515-d09d3db49951",
			},
		},
		{
			name:   "empty user id",
			userID: "",

			want: want{
				statusCode: http.StatusBadRequest,
				userID:     "",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", nil)
			writer := httptest.NewRecorder()

			bearerHash, err := usecase.SetUserIDToAuthHeaderFormat(entity.UserID(test.userID))
			assert.NoError(t, err)

			request.Header.Add(usecase.AuthHeader, bearerHash)

			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				userIDCtx, ok := r.Context().Value(entity.UserIDCtxKey{}).(entity.UserIDCtx)

				require.True(t, ok)
				assert.Equal(t, userIDCtx.UserID.String(), test.want.userID)
				assert.Equal(t, userIDCtx.StatusCode, test.want.statusCode)
			})

			handler := TokenParserMiddleware(nextHandler)
			handler.ServeHTTP(writer, request)
		})
	}
}

func TestInvalidTokenParserMiddleware(t *testing.T) {
	type want struct {
		statusCode int
	}
	tests := []struct {
		name string
		hash string

		want want
	}{
		{
			name: "undefined token",
			hash: "Bearer",

			want: want{
				statusCode: http.StatusUnauthorized,
			},
		},
		{
			name: "empty user id",
			hash: "",

			want: want{
				statusCode: http.StatusUnauthorized,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", nil)
			writer := httptest.NewRecorder()

			request.Header.Add(usecase.AuthHeader, test.hash)

			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				userIDCtx, ok := r.Context().Value(entity.UserIDCtxKey{}).(entity.UserIDCtx)

				require.True(t, ok)
				assert.Equal(t, userIDCtx.StatusCode, test.want.statusCode)
				assert.Empty(t, userIDCtx.UserID.String())
			})

			handler := TokenParserMiddleware(nextHandler)
			handler.ServeHTTP(writer, request)
		})
	}
}
