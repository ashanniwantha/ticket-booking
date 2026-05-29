package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
)

type TokenGenerator struct {
	secret     []byte
	expiration time.Duration
}

func NewTokenGenerator(secret string, expiration time.Duration) *TokenGenerator {
	return &TokenGenerator{
		secret: []byte(secret), expiration: expiration,
	}
}

func (g *TokenGenerator) GenerateToken(userID int64, username string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  userID,
		"name": username,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(g.expiration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(g.secret)
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}
	return signed, nil
}
