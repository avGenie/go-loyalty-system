package crypto

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost = 14
)

var (
	ErrWrongPassword = errors.New("wrong password")
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("error while hashing password: %w", err)
	}

	return string(bytes), nil
}

func CheckPasswordHash(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrWrongPassword
		}
		
		return fmt.Errorf("error while checking password hash: %w", err)
	}

	return nil
}
