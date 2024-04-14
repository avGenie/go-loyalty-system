package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avGenie/go-loyalty-system/internal/app/controller/http/auth/mock"
	"github.com/avGenie/go-loyalty-system/internal/app/entity"
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

	s := mock.NewMockUserAuthenticator(ctrl)

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

			authenticator := New(s)
			handler := authenticator.CreateUser()
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

func TestAuthenticateUser(t *testing.T) {
	validOutputUser := entity.User{
		ID:       "0b98bf79-833c-44e0-b979-2dae19dda46c",
		Login:    "login",
		Password: "$2a$14$x/4h3rb3YiVrlyR4w.Rme.cJmpvTwxwdS.kBJTzcsacKGIkBp0ITq",
	}

	invalidPasswordOutputUser := entity.User{
		ID:       "0b98bf79-833c-44e0-b979-2dae19dda46c",
		Login:    "login",
		Password: "$2a$14$1El0x4E9IRDBQauUxPNO1uQyUqRDI1MFCP47hOxb.NYuiiE1Sm1ei",
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	s := mock.NewMockUserAuthenticator(ctrl)

	type want struct {
		statusCode int
	}
	tests := []struct {
		name       string
		body       string
		getUser    entity.User
		getUserErr error
		isGetUser  bool

		want want
	}{
		{
			name:       "correct input data",
			body:       inputCorrect,
			getUser:    validOutputUser,
			getUserErr: nil,
			isGetUser:  true,

			want: want{
				statusCode: http.StatusOK,
			},
		},
		{
			name:       "wrong password",
			body:       inputCorrect,
			getUser:    invalidPasswordOutputUser,
			getUserErr: nil,
			isGetUser:  true,

			want: want{
				statusCode: http.StatusUnauthorized,
			},
		},
		{
			name:       "login not found in storage",
			body:       inputCorrect,
			getUser:    invalidPasswordOutputUser,
			getUserErr: err_storage.ErrLoginNotFound,
			isGetUser:  true,

			want: want{
				statusCode: http.StatusUnauthorized,
			},
		},
		{
			name:       "storage error",
			body:       inputCorrect,
			getUser:    invalidPasswordOutputUser,
			getUserErr: errors.New(""),
			isGetUser:  true,

			want: want{
				statusCode: http.StatusInternalServerError,
			},
		},
		{
			name:       "invalid user credentials",
			body:       inputInvalid,
			getUserErr: nil,
			isGetUser:  false,

			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:            "empty login in user credentials",
			body:            inputEmptyLogin,
			getUserErr:   nil,
			isGetUser:    false,

			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:            "empty password in user credentials",
			body:            inputEmptyPassword,
			getUserErr:   nil,
			isGetUser:    false,

			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:            "empty login and password in user credentials",
			body:            inputEmptyLoginPassword,
			getUserErr:   nil,
			isGetUser:    false,

			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader(test.body))
			writer := httptest.NewRecorder()

			if test.isGetUser {
				s.EXPECT().GetUser(gomock.Any(), gomock.Any()).Return(test.getUser, test.getUserErr)
			} else {
				s.EXPECT().GetUser(gomock.Any(), gomock.Any()).Times(0)
			}

			authenticator := New(s)
			handler := authenticator.AuthenticateUser()
			handler(writer, request)

			res := writer.Result()

			assert.Equal(t, test.want.statusCode, res.StatusCode)

			err := res.Body.Close()
			require.NoError(t, err)
		})
	}
}
