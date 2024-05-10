package crypto

import (
	"errors"
	"fmt"
	"time"

	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	err_usecase "github.com/avGenie/go-loyalty-system/internal/app/usecase/errors"
	"github.com/golang-jwt/jwt/v5"
)

const (
	tokenTimeout = time.Hour * 3
	secretKey    = "5269889d400bbf2dc66216f37b2839bb"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID entity.UserID
}

func BuildJWTString(userID entity.UserID) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTimeout)),
		},
		UserID: userID,
	})

	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func GetUserID(tokenString string) (entity.UserID, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method while parsing jwt: %v", t.Header["alg"])
			}
			return []byte(secretKey), nil
		})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return entity.UserID(""), err_usecase.ErrTokenExpired
		}

		return entity.UserID(""), fmt.Errorf("error while getting user id from token: %w", err)
	}

	if !token.Valid {
		return entity.UserID(""), err_usecase.ErrTokenNotValid
	}

	return claims.UserID, nil
}
