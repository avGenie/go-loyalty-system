package crypto

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost = 14
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
		return fmt.Errorf("error while checking password hash: %w", err)
	}

	return nil
}
