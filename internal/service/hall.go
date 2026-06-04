package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ashanniwantha/ticket-booking/internal/domain"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

const (
	hallVersionKey = "halls:version"
)

type AddHallRequest struct {
	Name string `json:"name"`
}

type UpdateHallRequest struct {
	Name string `json:"name"`
}

type HallResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type HallService interface {
	AddHall(ctx context.Context, req AddHallRequest) (*HallResponse, error)
	GetHallByID(ctx context.Context, hallID int64) (*HallResponse, error)
	ListAllHalls(ctx context.Context) ([]HallResponse, error)
	UpdateHall(ctx context.Context, hallID int64, req UpdateHallRequest) (*HallResponse, error)
	RemoveHall(ctx context.Context, hallID int64) error
}

type hallService struct {
	repo    domain.HallRepository
	cache   *redis.Client
	sfGroup singleflight.Group
	logger  *slog.Logger
}

func NewHallService(repo domain.HallRepository, cache *redis.Client, logger *slog.Logger) HallService {
	return &hallService{
		repo:    repo,
		cache:   cache,
		logger:  logger,
		sfGroup: singleflight.Group{},
	}
}

func (s *hallService) AddHall(ctx context.Context, req AddHallRequest) (*HallResponse, error) {
	name := strings.TrimSpace(req.Name)

	if name == "" {
		return nil, ErrEmptyHallName
	}

	hall := &domain.Hall{
		Name: name,
	}

	if err := s.repo.Create(ctx, hall); err != nil {
		return nil, err
	}

	// Increment the master collection version
	s.incrementCollectionVersion(ctx, hallVersionKey)

	resp := &HallResponse{
		ID:        hall.ID,
		Name:      hall.Name,
		CreatedAt: hall.CreatedAt,
		UpdatedAt: hall.UpdatedAt,
	}

	s.logger.Info("hall created", "hall_name", hall.Name)
	return resp, nil
}

func (s *hallService) GetHallByID(ctx context.Context, hallID int64) (*HallResponse, error) {
	if hallID <= 0 {
		return nil, ErrInvalidHallID
	}

	//  -- Try Cache First --
	cacheKey := fmt.Sprintf("halls:%d", hallID)
	if data, err := s.cache.Get(ctx, cacheKey).Bytes(); err == nil {
		var hallCache HallResponse

		if err := json.Unmarshal(data, &hallCache); err != nil {
			s.logger.Warn("failed to unmarshal hall cache, falling back to db", "err", err, "key", cacheKey)

		} else {
			// If the cache ID is 0, this is resolved as negative cache hit
			if hallCache.ID == 0 {
				return nil, domain.ErrHallNotFound
			}

			return &hallCache, nil
		}

	} else if !errors.Is(err, redis.Nil) {
		s.logger.Warn("redis error fetching hall cache", "err", err, "key", cacheKey)
	}

	// -- Cache Miss: Activating Singleflight to prevent Thundering herd --
	val, sfErr, shared := s.sfGroup.Do(cacheKey, func() (interface{}, error) {

		// Detach cancellation from leader's context but enforces a timeout
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()

		hall, err := s.repo.GetByID(dbCtx, hallID)
		if err != nil {

			if errors.Is(err, domain.ErrHallNotFound) {
				s.logger.Info("hall not found in DB, caching negative hit to prevent penetration", "id", hallID)
				go s.writeToCacheBackground(ctx, cacheKey, HallResponse{}, 1*time.Minute)
			}

			return nil, err
		}

		hallResp := HallResponse{
			ID:        hall.ID,
			Name:      hall.Name,
			CreatedAt: hall.CreatedAt,
			UpdatedAt: hall.UpdatedAt,
		}

		// -- Update Cache asynchronously (fire-and-forget)
		go s.writeToCacheBackground(ctx, cacheKey, hallResp, 5*time.Minute)

		return hallResp, nil
	})

	// -- Handle Singleflight processing results --
	if sfErr != nil {
		return nil, sfErr
	}
	if shared {
		s.logger.Info("concurrency suppressed via Singleflight", "key", cacheKey)
	}
	hall, ok := val.(HallResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected data type from singleflight group")
	}
	return &hall, nil
}

func (s *hallService) ListAllHalls(ctx context.Context) ([]HallResponse, error) {
	// 1. -- Fetch master collection key version (Defaults to 0 if not exists)
	version, err := s.cache.Get(ctx, hallVersionKey).Int64()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			s.logger.Warn("failed to fetch hall cache collection version, falling back to 0", "err", err)
		}
		version = 0
	}

	// 2. -- Generate a version scoped cache key --
	cacheKey := fmt.Sprintf("halls:v%d:all", version)

	// 3. -- Try cache first --
	if data, err := s.cache.Get(ctx, cacheKey).Bytes(); err == nil {
		cacheHallList := make([]HallResponse, 0)

		if err := json.Unmarshal(data, &cacheHallList); err != nil {
			s.logger.Warn("failed to unmarshal hall list cache, falling back to db",
				"err", err,
			)
		} else {
			return cacheHallList, nil
		}
	} else if !errors.Is(err, redis.Nil) {
		s.logger.Warn("redis error fetching all halls, falling back to db", "err", err)
	}

	// 4. Cache Miss: Activate SingleFlight to prevent thundering herd
	val, sfErr, shared := s.sfGroup.Do(cacheKey, func() (interface{}, error) {
		// Detach cancellation from leader's context but enforces a safety timeout
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()

		hallList, err := s.repo.ListAll(dbCtx)
		if err != nil {
			return nil, err
		}

		hallListResp := make([]HallResponse, 0, len(hallList))
		for _, hall := range hallList {
			hallListResp = append(hallListResp, HallResponse{
				ID:        hall.ID,
				Name:      hall.Name,
				CreatedAt: hall.CreatedAt,
				UpdatedAt: hall.UpdatedAt,
			})
		}

		// Cache penetration protection: Dynamic TTL --
		var ttl time.Duration
		if len(hallListResp) == 0 {
			ttl = 1 * time.Minute
		} else {
			ttl = 5 * time.Minute
		}

		// -- Update cache asynchronously (fire-and-forget) --
		go s.writeToCacheBackground(ctx, cacheKey, hallListResp, ttl)

		return hallListResp, nil
	})

	// -- Handle singleflight processing results --
	if sfErr != nil {
		return nil, sfErr
	}
	if shared {
		s.logger.Info("concurrency suppressed via Singleflight", "key", cacheKey)
	}
	halls, ok := val.([]HallResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected data type from Singleflight group")
	}

	return halls, nil
}

func (s *hallService) UpdateHall(ctx context.Context, hallID int64, req UpdateHallRequest) (*HallResponse, error) {
	if hallID <= 0 {
		return nil, ErrInvalidHallID
	}

	name := strings.TrimSpace(req.Name)

	if name == "" {
		return nil, ErrEmptyHallName
	}

	hall := &domain.Hall{
		ID:   hallID,
		Name: name,
	}

	if err := s.repo.Update(ctx, hall); err != nil {
		return nil, err
	}
	// Evict specific cache
	specificCacheKey := fmt.Sprintf("halls:%d", hall.ID)
	s.evictCache(ctx, specificCacheKey)

	// Increment master collection version before returning status code
	s.incrementCollectionVersion(ctx, hallVersionKey)

	resp := &HallResponse{
		ID:        hall.ID,
		Name:      hall.Name,
		CreatedAt: hall.CreatedAt,
		UpdatedAt: hall.UpdatedAt,
	}

	s.logger.Info("hall updated", "hall_name", name)
	return resp, nil
}

func (s *hallService) RemoveHall(ctx context.Context, hallID int64) error {
	if hallID <= 0 {
		return ErrInvalidHallID
	}

	if err := s.repo.Delete(ctx, hallID); err != nil {
		return err
	}
	// Evict specific stale cache
	specificCacheKey := fmt.Sprintf("halls:%d", hallID)
	s.evictCache(ctx, specificCacheKey)

	// Increment master collection version before returning status code
	s.incrementCollectionVersion(ctx, hallVersionKey)

	s.logger.Info("hall deleted", "hall_id", hallID)
	return nil
}

// Write to cache asynchronously
func (s *hallService) writeToCacheBackground(ctx context.Context, cacheKey string, data any, ttl time.Duration) {
	bgCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()

	bytes, err := json.Marshal(data)

	if err != nil {
		s.logger.Warn("failed to marhsal hall cache data", "err", err, "key", cacheKey)
		return
	}

	if err := s.cache.Set(bgCtx, cacheKey, bytes, ttl).Err(); err != nil {
		s.logger.Warn("failed to cache data to redis in the background", "err", err)
	}
}

// Remove stale cache synchronously
func (s *hallService) evictCache(ctx context.Context, cacheKeys ...string) {
	bgCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()

	if err := s.cache.Del(bgCtx, cacheKeys...).Err(); err != nil {
		s.logger.Warn("failed to evict stale cache from redis", "err", err, "keys", cacheKeys)
	}
}

// INCrement the master collection cache key versioning
func (s *hallService) incrementCollectionVersion(ctx context.Context, collectionKey string) {
	bgCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()

	if err := s.cache.Incr(bgCtx, collectionKey).Err(); err != nil {
		s.logger.Warn("failed to increment master collection version pointer",
			"err", err,
			"collection", collectionKey,
		)
	}
}
