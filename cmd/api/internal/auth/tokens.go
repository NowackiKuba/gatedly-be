package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// claims are used to create and parse JWTs (access and refresh).
type claims struct {
	jwt.RegisteredClaims
	UserID    string `json:"userId"`
	TokenType string `json:"tokenType"`
}

// CreateAccessToken signs a new access token for the user.
func CreateAccessToken(userID string, secret string, ttl time.Duration) (string, error) {
	return createToken(userID, TokenTypeAccess, secret, ttl)
}

// CreateRefreshToken signs a new refresh token for the user.
func CreateRefreshToken(userID string, secret string, ttl time.Duration) (string, error) {
	return createToken(userID, TokenTypeRefresh, secret, ttl)
}

func createToken(userID, tokenType, secret string, ttl time.Duration) (string, error) {
	now := time.Now()
	c := claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		UserID:    userID,
		TokenType: tokenType,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &c)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// ParseRefreshToken parses the token string and returns userID if it is a valid refresh token.
func ParseRefreshToken(tokenStr, secret string) (userID string, err error) {
	token, err := jwt.ParseWithClaims(tokenStr, &claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil {
		return "", fmt.Errorf("parse refresh token: %w", err)
	}
	c, ok := token.Claims.(*claims)
	if !ok || !token.Valid || c.TokenType != TokenTypeRefresh || c.UserID == "" {
		return "", fmt.Errorf("invalid refresh token")
	}
	return c.UserID, nil
}
