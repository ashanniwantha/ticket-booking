package test

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Returns value of the environment variable or the default
func envOrDeafault(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func NewTestPool(ctx context.Context) (*pgxpool.Pool, error) {
	host := envOrDeafault("TEST_DB_HOST", "localhost")
	port := envOrDeafault("TEST_DB_PORT", "5434")
	user := envOrDeafault("TEST_DB_USER", "testuser")
	password := envOrDeafault("TEST_DB_PASSWORD", "testpass")
	name := envOrDeafault("TEST_DB_NAME", "ticketbook_test")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, name)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("open test db: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping test db: %w", err)
	}
	return pool, nil
}
