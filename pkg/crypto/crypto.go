package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// HashPassword hashes password with bcrypt (cost 12).
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword returns true if password matches hash.
func CheckPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// RandomToken returns n random bytes encoded as hex (string length 2*n).
func RandomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("random token: %w", err)
	}
	return hex.EncodeToString(b), nil
}
