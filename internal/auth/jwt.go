package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenGenerator creates and validates JWT access tokens.
type TokenGenerator struct {
	secret     []byte
	expiration time.Duration
}

// NewTokenGenerator returns a new TokenGenerator.
func NewTokenGenerator(secret string, expiration time.Duration) *TokenGenerator {
	return &TokenGenerator{
		secret:     []byte(secret),
		expiration: expiration,
	}
}

// GenerateToken signs a new JWT for the given user.
func (g *TokenGenerator) GenerateToken(userID int64, username string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":  userID,
		"name": username,
		"iat":  now.Unix(),
		"exp":  now.Add(g.expiration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(g.secret)
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}
	return signed, nil
}

// ValidateToken parses and validates a JWT string. It returns the user ID and
// username extracted from the token's claims.
func (g *TokenGenerator) ValidateToken(tokenString string) (userID int64, username string, err error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Ensure the signing method is HMAC.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return g.secret, nil
	})
	if err != nil {
		return 0, "", fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return 0, "", fmt.Errorf("invalid token claims")
	}

	// Extract user ID from "sub".
	idFloat, ok := claims["sub"].(float64)
	if !ok {
		return 0, "", fmt.Errorf("user ID claim missing or invalid")
	}
	userID = int64(idFloat)

	// Extract username from "name".
	username, _ = claims["name"].(string)

	return userID, username, nil
}
