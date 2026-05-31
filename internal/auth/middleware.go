package auth

import (
	"context"
	"net/http"
	"strings"
)

// contextKey is an unexported type to avoid key collisions in context.
type contextKey string

const (
	userIDCtxKey   contextKey = "userID"
	usernameCtxKey contextKey = "username"
)

// UserIDFromContext retrieves the authenticated user's ID from the request context.
// Returns the ID and true if present, otherwise 0 and false.
func UserIDFromContext(r *http.Request) (int64, bool) {
	id, ok := r.Context().Value(userIDCtxKey).(int64)
	return id, ok
}

// UsernameFromContext retrieves the authenticated user's name from the request context.
// Returns the name and true if present, otherwise "" and false.
func UsernameFromContext(r *http.Request) (string, bool) {
	name, ok := r.Context().Value(usernameCtxKey).(string)
	return name, ok
}

// Authenticate returns a middleware that validates a JWT from the Authorization header,
// extracts the user ID and username, and injects them into the request context.
// If the token is missing or invalid, it responds with 401 Unauthorized.
func Authenticate(tokenGen *TokenGenerator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract the token from the Authorization header.
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			// The header must use the Bearer scheme.
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
				return
			}

			tokenStr := parts[1]

			// Validate token and extract user info.
			userID, username, err := tokenGen.ValidateToken(tokenStr)
			if err != nil {
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Inject user details into the context.
			ctx := context.WithValue(r.Context(), userIDCtxKey, userID)
			ctx = context.WithValue(ctx, usernameCtxKey, username)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
