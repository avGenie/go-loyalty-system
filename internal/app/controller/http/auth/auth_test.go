package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avGenie/go-loyalty-system/internal/app/controller/http/auth/mock"
	err_storage "github.com/avGenie/go-loyalty-system/internal/app/storage/api/errors"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	inputCorrect = strings.TrimSpace(`
	{
		"login": "login",
		"password": "password"
	}`)

	inputEmptyLogin = strings.TrimSpace(`
	{
		"login": "",
		"password": "password"
	}`)

	inputEmptyPassword = strings.TrimSpace(`
	{
		"login": "login",
		"password": ""
	}`)

	inputEmptyLoginPassword = strings.TrimSpace(`
	{
		"login": "",
		"password": ""
	}`)

	inputInvalid = `<invalid json>`
)

func TestCreateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	s := mock.NewMockUserCreator(ctrl)

	type want struct {
		statusCode int
	}
	tests := []struct {
		name            string
		body            string
		createUserErr   error
		isCreateUser    bool
		authHeaderEmpty bool

		want want
	}{
		{
			name:            "correct input data",
			body:            inputCorrect,
			createUserErr:   nil,
			isCreateUser:    true,
			authHeaderEmpty: false,

			want: want{
				statusCode: http.StatusOK,
			},
		},
		{
			name:            "login exists in storage",
			body:            inputCorrect,
			createUserErr:   err_storage.ErrLoginExists,
			isCreateUser:    true,
			authHeaderEmpty: true,

			want: want{
				statusCode: http.StatusConflict,
			},
		},
		{
			name:            "storage error",
			body:            inputCorrect,
			createUserErr:   errors.New(""),
			isCreateUser:    true,
			authHeaderEmpty: true,

			want: want{
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:            "invalid user credentials",
			body:            inputInvalid,
			createUserErr:   nil,
			isCreateUser:    false,
			authHeaderEmpty: true,

			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:            "empty login in user credentials",
			body:            inputEmptyLogin,
			createUserErr:   nil,
			isCreateUser:    false,
			authHeaderEmpty: true,

			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:            "empty password in user credentials",
			body:            inputEmptyPassword,
			createUserErr:   nil,
			isCreateUser:    false,
			authHeaderEmpty: true,

			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:            "empty login and password in user credentials",
			body:            inputEmptyLoginPassword,
			createUserErr:   nil,
			isCreateUser:    false,
			authHeaderEmpty: true,

			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader(test.body))
			writer := httptest.NewRecorder()

			if test.isCreateUser {
				s.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(test.createUserErr)
			} else {
				s.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Times(0)
			}

			handler := CreateUser(s)
			handler(writer, request)

			res := writer.Result()

			assert.Equal(t, test.want.statusCode, res.StatusCode)

			err := res.Body.Close()
			require.NoError(t, err)

			if !test.authHeaderEmpty {
				authContent := res.Header.Get("Authorization")
				assert.NotEmpty(t, authContent)
			}
		})
	}
}
