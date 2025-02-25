package orders

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avGenie/go-loyalty-system/internal/app/config"
	"github.com/avGenie/go-loyalty-system/internal/app/controller/http/orders/mock"
	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	err_storage "github.com/avGenie/go-loyalty-system/internal/app/storage/api/errors"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	inputInvalid = `<invalid json>`
)

type Reader interface {
	Read(p []byte) (n int, err error)
}

type errReader int

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("test error")
}

func Config() config.Config {
	return config.Config{
		AccrualAddr: "localhost:64321",
	}
}

func TestUploadOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	orderProcessor := mock.NewMockOrderProcessor(ctrl)

	type want struct {
		statusCode int
		outputBody string
	}
	tests := []struct {
		name            string
		body            Reader
		uploadErr       error
		isUploadUser    bool
		isUpdateAccrual bool
		isContext       bool
		userIDCtx       entity.UserIDCtx
		storageUserID   entity.UserID

		want want
	}{
		{
			name:            "new order for user",
			body:            strings.NewReader("735584316112"),
			uploadErr:       nil,
			isUploadUser:    true,
			isUpdateAccrual: true,
			isContext:       true,
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
			uploadErr:     err_storage.ErrOrderNumberExists,
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
			uploadErr:     err_storage.ErrOrderNumberExists,
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
			name:         "read order number error",
			body:         errReader(0),
			uploadErr:    nil,
			isUploadUser: false,
			isContext:    true,
			userIDCtx: entity.UserIDCtx{
				UserID:     "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:         "wrong order number",
			body:         strings.NewReader("1234"),
			uploadErr:    nil,
			isUploadUser: false,
			isContext:    true,
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
				orderProcessor.EXPECT().UploadOrder(gomock.Any(), gomock.Any(), gomock.Any()).Return(test.storageUserID, test.uploadErr)
			} else {
				orderProcessor.EXPECT().UploadOrder(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			}

			orderProcessor.EXPECT().GetOrdersForUpdate(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

			orders := New(orderProcessor, Config())
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

func TestGetUserOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	orderProcessor := mock.NewMockOrderProcessor(ctrl)

	outputCorrect := strings.TrimSpace(`
	[
		{
			"number": "735584316112",
			"status": "NEW",
			"accrual": 0,
			"uploaded_at": "2024-04-16T12:40:29+03:00"
		},
		{
			"number": "527652728124",
			"status": "NEW",
			"accrual": 0,
			"uploaded_at": "2024-04-19T11:56:43+03:00"
		},
		{
			"number": "044606165247",
			"status": "NEW",
			"accrual": 0,
			"uploaded_at": "2024-04-19T11:58:34+03:00"
		}
	]`)

	correctDBOutput := entity.Orders{
		{
			Number:      entity.OrderNumber("735584316112"),
			Status:      "NEW",
			Accrual:     0,
			DateCreated: "2024-04-16T09:40:29.841538Z",
		},
		{
			Number:      entity.OrderNumber("527652728124"),
			Status:      "NEW",
			Accrual:     0,
			DateCreated: "2024-04-19T08:56:43.208729Z",
		},
		{
			Number:      entity.OrderNumber("044606165247"),
			Status:      "NEW",
			Accrual:     0,
			DateCreated: "2024-04-19T08:58:34.616336Z",
		},
	}

	type want struct {
		statusCode int
		outputBody string
	}
	tests := []struct {
		name            string
		storageErr      error
		isGetOrders     bool
		isUpdateAccrual bool
		isContext       bool
		isJSONBody      bool
		accrualCount    int
		dbOutput        entity.Orders
		userIDCtx       entity.UserIDCtx

		want want
	}{
		{
			name:            "correct input data",
			storageErr:      nil,
			isGetOrders:     true,
			isUpdateAccrual: true,
			isContext:       true,
			isJSONBody:      true,
			accrualCount:    3,
			dbOutput:        correctDBOutput,
			userIDCtx: entity.UserIDCtx{
				UserID:     "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusOK,
				outputBody: outputCorrect,
			},
		},
		{
			name:        "orders are not found for user",
			storageErr:  err_storage.ErrOrderForUserNotFound,
			isGetOrders: true,
			isContext:   true,
			userIDCtx: entity.UserIDCtx{
				UserID:     "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusNoContent,
			},
		},
		{
			name:        "storage error",
			storageErr:  fmt.Errorf(""),
			isGetOrders: true,
			isContext:   true,
			userIDCtx: entity.UserIDCtx{
				UserID:     "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:        "user id context undefined",
			storageErr:  nil,
			isGetOrders: false,
			isContext:   false,

			want: want{
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:        "user id bad request",
			storageErr:  nil,
			isGetOrders: false,
			isContext:   true,
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
			name:        "user unauthorized",
			storageErr:  nil,
			isGetOrders: false,
			isContext:   true,
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
			name:        "user id is invalid",
			storageErr:  nil,
			isGetOrders: false,
			isContext:   true,
			userIDCtx: entity.UserIDCtx{
				UserID:     "",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusUnauthorized,
				outputBody: ErrInvalidAuth,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
			writer := httptest.NewRecorder()

			if test.isContext {
				request = request.WithContext(context.WithValue(request.Context(), entity.UserIDCtxKey{}, test.userIDCtx))
			}

			if test.isGetOrders {
				orderProcessor.EXPECT().GetUserOrders(gomock.Any(), gomock.Any()).Return(test.dbOutput, test.storageErr)
			} else {
				orderProcessor.EXPECT().GetUserOrders(gomock.Any(), gomock.Any()).Times(0)
			}

			orderProcessor.EXPECT().GetOrdersForUpdate(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

			orders := New(orderProcessor, Config())
			handler := orders.GetUserOrders()
			handler(writer, request)

			res := writer.Result()

			assert.Equal(t, test.want.statusCode, res.StatusCode)

			if len(test.want.outputBody) != 0 {
				bodyResult, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				if test.isJSONBody {
					assert.JSONEq(t, test.want.outputBody, strings.TrimSuffix(string(bodyResult), "\n"))
				} else {
					assert.Equal(t, test.want.outputBody, strings.TrimSuffix(string(bodyResult), "\n"))
				}
			}

			err := res.Body.Close()
			require.NoError(t, err)
		})
	}
}

func TestGetUserBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	orderProcessor := mock.NewMockOrderProcessor(ctrl)

	outputCorrect := strings.TrimSpace(`
	{
		"current": 600.20,
		"withdrawn": 350.80
	}`)

	correctDBOutput := entity.UserBalance{
		UserID:      "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
		Balance:     600.20,
		Withdrawans: 350.80,
	}

	type want struct {
		statusCode int
		outputBody string
	}
	tests := []struct {
		name         string
		userID       string
		storageErr   error
		isGetBalance bool
		isContext    bool
		isJSONBody   bool
		dbOutput     entity.UserBalance
		userIDCtx    entity.UserIDCtx

		want want
	}{
		{
			name:         "get correct balance",
			userID:       "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
			storageErr:   nil,
			isGetBalance: true,
			isContext:    true,
			isJSONBody:   true,
			dbOutput:     correctDBOutput,
			userIDCtx: entity.UserIDCtx{
				UserID:     "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusOK,
				outputBody: outputCorrect,
			},
		},
		{
			name:         "storage error",
			storageErr:   fmt.Errorf(""),
			isGetBalance: true,
			isContext:    true,
			userIDCtx: entity.UserIDCtx{
				UserID:     "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:         "user id context undefined",
			storageErr:   nil,
			isGetBalance: false,
			isContext:    false,

			want: want{
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:         "user id bad request",
			storageErr:   nil,
			isGetBalance: false,
			isContext:    true,
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
			name:         "user unauthorized",
			storageErr:   nil,
			isGetBalance: false,
			isContext:    true,
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
			name:         "user id is invalid",
			storageErr:   nil,
			isGetBalance: false,
			isContext:    true,
			userIDCtx: entity.UserIDCtx{
				UserID:     "",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusUnauthorized,
				outputBody: ErrInvalidAuth,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
			writer := httptest.NewRecorder()

			if test.isContext {
				request = request.WithContext(context.WithValue(request.Context(), entity.UserIDCtxKey{}, test.userIDCtx))
			}

			if test.isGetBalance {
				orderProcessor.EXPECT().GetUserBalance(gomock.Any(), gomock.Any()).Return(test.dbOutput, test.storageErr)
			} else {
				orderProcessor.EXPECT().GetUserBalance(gomock.Any(), gomock.Any()).Times(0)
			}

			orderProcessor.EXPECT().GetOrdersForUpdate(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

			orders := New(orderProcessor, Config())
			handler := orders.GetUserBalance()
			handler(writer, request)

			res := writer.Result()

			assert.Equal(t, test.want.statusCode, res.StatusCode)

			if len(test.want.outputBody) != 0 {
				bodyResult, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				if test.isJSONBody {
					assert.JSONEq(t, test.want.outputBody, strings.TrimSuffix(string(bodyResult), "\n"))
				} else {
					assert.Equal(t, test.want.outputBody, strings.TrimSuffix(string(bodyResult), "\n"))
				}
			}

			err := res.Body.Close()
			require.NoError(t, err)
		})
	}
}

func TestWithdrawBonuses(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	orderProcessor := mock.NewMockOrderProcessor(ctrl)

	inputCorrect := strings.TrimSpace(`
	{
		"order": "221488416308",
		"sum": 751
	}`)

	inputIncorrect := strings.TrimSpace(`
	{
		"order": "1",
		"sum": 751
	}`)

	type want struct {
		statusCode int
		outputBody string
	}
	tests := []struct {
		name              string
		storageErr        error
		isWithdrawBonuses bool
		isContext         bool
		body              Reader
		userIDCtx         entity.UserIDCtx

		want want
	}{
		{
			name:              "correct input withdraw",
			storageErr:        nil,
			isWithdrawBonuses: true,
			isContext:         true,
			body:              strings.NewReader(inputCorrect),
			userIDCtx: entity.UserIDCtx{
				UserID:     "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusOK,
			},
		},
		{
			name:              "invalid order number",
			storageErr:        nil,
			isWithdrawBonuses: false,
			isContext:         true,
			body:              strings.NewReader(inputIncorrect),
			userIDCtx: entity.UserIDCtx{
				UserID:     "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusUnprocessableEntity,
			},
		},
		{
			name:              "invalid JSON",
			storageErr:        nil,
			isWithdrawBonuses: false,
			isContext:         true,
			body:              strings.NewReader(inputInvalid),
			userIDCtx: entity.UserIDCtx{
				UserID:     "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:              "read order number error",
			storageErr:        nil,
			isWithdrawBonuses: false,
			isContext:         true,
			body:              errReader(0),
			userIDCtx: entity.UserIDCtx{
				UserID:     "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:              "not enough money",
			storageErr:        err_storage.ErrNotEnoughSum,
			isWithdrawBonuses: true,
			isContext:         true,
			body:              strings.NewReader(inputCorrect),
			userIDCtx: entity.UserIDCtx{
				UserID:     "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusPaymentRequired,
			},
		},
		{
			name:              "database error",
			storageErr:        errors.New(""),
			isWithdrawBonuses: true,
			isContext:         true,
			body:              strings.NewReader(inputCorrect),
			userIDCtx: entity.UserIDCtx{
				UserID:     "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:              "user id context undefined",
			storageErr:        nil,
			isWithdrawBonuses: false,
			isContext:         false,
			body:              strings.NewReader(inputCorrect),

			want: want{
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:              "user id bad request",
			storageErr:        nil,
			isWithdrawBonuses: false,
			isContext:         true,
			body:              strings.NewReader(inputCorrect),
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
			name:              "user unauthorized",
			storageErr:        nil,
			isWithdrawBonuses: false,
			isContext:         true,
			body:              strings.NewReader(inputCorrect),
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
			name:              "user id is invalid",
			storageErr:        nil,
			isWithdrawBonuses: false,
			isContext:         true,
			body:              strings.NewReader(inputCorrect),
			userIDCtx: entity.UserIDCtx{
				UserID:     "",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusUnauthorized,
				outputBody: ErrInvalidAuth,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", test.body)
			writer := httptest.NewRecorder()

			if test.isContext {
				request = request.WithContext(context.WithValue(request.Context(), entity.UserIDCtxKey{}, test.userIDCtx))
			}

			if test.isWithdrawBonuses {
				orderProcessor.EXPECT().WithdrawUser(gomock.Any(), gomock.Any(), gomock.Any()).Return(test.storageErr)
			} else {
				orderProcessor.EXPECT().WithdrawUser(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			}

			orderProcessor.EXPECT().GetOrdersForUpdate(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

			orders := New(orderProcessor, Config())
			handler := orders.WithdrawBonuses()
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

func TestGetUserWithdrawals(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	orderProcessor := mock.NewMockOrderProcessor(ctrl)

	outputCorrect := strings.TrimSpace(`
	[
		{
			"order": "374311367329",
			"sum": 100,
			"processed_at": "2024-04-30T21:32:05+03:00"
		},
		{
			"order": "475622844086",
			"sum": 50,
			"processed_at": "2024-04-30T21:32:18+03:00"
		},
		{
			"order": "221488416308",
			"sum": 50,
			"processed_at": "2024-04-30T21:32:23+03:00"
		}
	]`)

	dbOutputCorrect := entity.Withdrawals{
		{
			OrderNumber: "374311367329",
			Sum:         100,
			DateCreated: "2024-04-30T18:32:05.187329Z",
		},
		{
			OrderNumber: "475622844086",
			Sum:         50,
			DateCreated: "2024-04-30T18:32:18.574718Z",
		},
		{
			OrderNumber: "221488416308",
			Sum:         50,
			DateCreated: "2024-04-30T18:32:23.250438Z",
		},
	}

	type want struct {
		statusCode int
		outputBody string
	}
	tests := []struct {
		name                 string
		storageErr           error
		isGetUserWithdrawals bool
		isContext            bool
		isJSONBody           bool
		dbOutput             entity.Withdrawals
		userIDCtx            entity.UserIDCtx

		want want
	}{
		{
			name:                 "correct get user withdrawals",
			storageErr:           nil,
			isGetUserWithdrawals: true,
			isContext:            true,
			isJSONBody:           true,
			dbOutput:             dbOutputCorrect,
			userIDCtx: entity.UserIDCtx{
				UserID:     "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusOK,
				outputBody: outputCorrect,
			},
		},
		{
			name:                 "withdrawals not found for user",
			storageErr:           err_storage.ErrWithdrawalsForUserNotFound,
			isGetUserWithdrawals: true,
			isContext:            true,
			dbOutput:             entity.Withdrawals{},
			userIDCtx: entity.UserIDCtx{
				UserID:     "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusNoContent,
			},
		},
		{
			name:                 "database error",
			storageErr:           errors.New(""),
			isGetUserWithdrawals: true,
			isContext:            true,
			dbOutput:             entity.Withdrawals{},
			userIDCtx: entity.UserIDCtx{
				UserID:     "ac2a4811-4f10-487f-bde3-e39a14af7cd8",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:                 "user id context undefined",
			storageErr:           nil,
			isGetUserWithdrawals: false,
			isContext:            false,

			want: want{
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:                 "user id bad request",
			storageErr:           nil,
			isGetUserWithdrawals: false,
			isContext:            true,
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
			name:                 "user unauthorized",
			storageErr:           nil,
			isGetUserWithdrawals: false,
			isContext:            true,
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
			name:                 "user id is invalid",
			storageErr:           nil,
			isGetUserWithdrawals: false,
			isContext:            true,
			userIDCtx: entity.UserIDCtx{
				UserID:     "",
				StatusCode: http.StatusOK,
			},

			want: want{
				statusCode: http.StatusUnauthorized,
				outputBody: ErrInvalidAuth,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
			writer := httptest.NewRecorder()

			if test.isContext {
				request = request.WithContext(context.WithValue(request.Context(), entity.UserIDCtxKey{}, test.userIDCtx))
			}

			if test.isGetUserWithdrawals {
				orderProcessor.EXPECT().GetUserWithdrawals(gomock.Any(), gomock.Any()).Return(test.dbOutput, test.storageErr)
			} else {
				orderProcessor.EXPECT().GetUserWithdrawals(gomock.Any(), gomock.Any()).Times(0)
			}

			orderProcessor.EXPECT().GetOrdersForUpdate(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

			orders := New(orderProcessor, Config())
			handler := orders.GetUserWithdrawals()
			handler(writer, request)

			res := writer.Result()

			assert.Equal(t, test.want.statusCode, res.StatusCode)

			if len(test.want.outputBody) != 0 {
				bodyResult, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				if test.isJSONBody {
					assert.JSONEq(t, test.want.outputBody, strings.TrimSuffix(string(bodyResult), "\n"))
				} else {
					assert.Equal(t, test.want.outputBody, strings.TrimSuffix(string(bodyResult), "\n"))
				}
			}

			err := res.Body.Close()
			require.NoError(t, err)
		})
	}
}
