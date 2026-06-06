package test

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Creates a Redis client connected to the test Redis instance
func NewTestRedis(ctx context.Context) (*redis.Client, error) {
	host := envOrDeafault("TEST_REDIS_HOST", "localhost")
	port := envOrDeafault("TEST_REDIS_PORT", "6381")

	addr := fmt.Sprintf("%s:%s", host, port)

	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   0,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
		return nil, fmt.Errorf("ping test redis: %w", err)
	}
	return rdb, nil
}
