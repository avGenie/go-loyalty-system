package orders

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avGenie/go-loyalty-system/internal/app/controller/http/orders/mock"
	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Reader interface {
	Read(p []byte) (n int, err error)
}

type errReader int

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("test error")
}

func TestUploadOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	s := mock.NewMockOrderProcessor(ctrl)

	type want struct {
		statusCode int
		outputBody string
	}
	tests := []struct {
		name          string
		body          Reader
		uploadErr     error
		isUploadUser  bool
		isContext     bool
		userIDCtx     entity.UserIDCtx
		storageUserID entity.UserID

		want want
	}{
		{
			name:         "new order for user",
			body:         strings.NewReader("735584316112"),
			uploadErr:    nil,
			isUploadUser: true,
			isContext:    true,
			userIDCtx: entity.UserIDCtx{
				UserID:     "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusAccepted,
			},
		},
		{
			name:          "exist order for user",
			body:          strings.NewReader("735584316112"),
			uploadErr:     nil,
			isUploadUser:  true,
			isContext:     true,
			storageUserID: "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
			userIDCtx: entity.UserIDCtx{
				UserID:     "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusOK,
			},
		},
		{
			name:          "exist order for another user",
			body:          strings.NewReader("735584316112"),
			uploadErr:     nil,
			isUploadUser:  true,
			isContext:     true,
			storageUserID: "6f28a678-7eba-4a4e-966c-7fedc6420df7",
			userIDCtx: entity.UserIDCtx{
				UserID:     "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusConflict,
			},
		},
		{
			name:          "user id context undefined",
			body:          strings.NewReader("735584316112"),
			uploadErr:     nil,
			isUploadUser:  false,
			isContext:     false,
			storageUserID: "6f28a678-7eba-4a4e-966c-7fedc6420df7",

			want: want{
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:          "user id bad request",
			body:          strings.NewReader("735584316112"),
			uploadErr:     nil,
			isUploadUser:  false,
			isContext:     true,
			storageUserID: "6f28a678-7eba-4a4e-966c-7fedc6420df7",
			userIDCtx: entity.UserIDCtx{
				UserID:     "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
				StatusCode: http.StatusBadRequest,
			},

			want: want{
				statusCode: http.StatusUnauthorized,
				outputBody: ErrInvalidAuth,
			},
		},
		{
			name:          "user unauthorized",
			body:          strings.NewReader("735584316112"),
			uploadErr:     nil,
			isUploadUser:  false,
			isContext:     true,
			storageUserID: "6f28a678-7eba-4a4e-966c-7fedc6420df7",
			userIDCtx: entity.UserIDCtx{
				UserID:     "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
				StatusCode: http.StatusUnauthorized,
			},

			want: want{
				statusCode: http.StatusUnauthorized,
				outputBody: ErrTokenExpired,
			},
		},
		{
			name:          "user id is invalid",
			body:          strings.NewReader("735584316112"),
			uploadErr:     nil,
			isUploadUser:  false,
			isContext:     true,
			storageUserID: "6f28a678-7eba-4a4e-966c-7fedc6420df7",
			userIDCtx: entity.UserIDCtx{
				UserID:     "",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusUnauthorized,
				outputBody: ErrInvalidAuth,
			},
		},
		{
			name:          "read order number error",
			body:          errReader(0),
			uploadErr:     nil,
			isUploadUser:  false,
			isContext:     true,
			userIDCtx: entity.UserIDCtx{
				UserID:     "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:          "wrong order number",
			body:          strings.NewReader("1234"),
			uploadErr:     nil,
			isUploadUser:  false,
			isContext:     true,
			userIDCtx: entity.UserIDCtx{
				UserID:     "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusUnprocessableEntity,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/user/register", test.body)
			writer := httptest.NewRecorder()

			if test.isContext {
				request = request.WithContext(context.WithValue(request.Context(), entity.UserIDCtxKey{}, test.userIDCtx))
			}

			if test.isUploadUser {
				s.EXPECT().UploadOrder(gomock.Any(), gomock.Any(), gomock.Any()).Return(test.storageUserID, test.uploadErr)
			} else {
				s.EXPECT().UploadOrder(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			}

			orders := New(s)
			handler := orders.UploadOrder()
			handler(writer, request)

			res := writer.Result()

			assert.Equal(t, test.want.statusCode, res.StatusCode)

			if len(test.want.outputBody) != 0 {
				bodyResult, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				assert.Equal(t, test.want.outputBody, strings.TrimSuffix(string(bodyResult), "\n"))
			}

			err := res.Body.Close()
			require.NoError(t, err)
		})
	}
}
