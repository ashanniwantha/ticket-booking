package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

type BaseService struct {
	cache   *redis.Client
	sfGroup singleflight.Group
	logger  *slog.Logger
}

func NewBaseService(logger *slog.Logger) *BaseService {
	return &BaseService{logger: logger}
}

// Helper method that writes to the cache in the background
func (b *BaseService) WriteToCacheBackground(cacheKey string, dataToCache any, cacheTTL time.Duration) {
	bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data, err := json.Marshal(dataToCache)
	if err != nil {
		b.logger.Warn("failed to marshal payload for caching", "err", err, "key", cacheKey)
		return
	}
	if err := b.cache.Set(bgCtx, cacheKey, data, cacheTTL).Err(); err != nil {
		b.logger.Warn("failed to save payload into redis backing store", "err", err, "key", cacheKey)
	}
}
