package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ashanniwantha/ticket-booking/internal/domain"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

const (
	seatVersionKey = "seats:version"
)

type AddSeatRequest struct {
	HallID     int64            `json:"hall_id"`
	SeatNumber string           `json:"seat_number"`
	Class      domain.SeatClass `json:"class"`
}

type UpdateSeatRequest struct {
	SeatNumber string           `json:"seat_number"`
	Class      domain.SeatClass `json:"class"`
}

type SeatResponse struct {
	ID         int64            `json:"id"`
	HallID     int64            `json:"hall_id"`
	SeatNumber string           `json:"seat_number"`
	Class      domain.SeatClass `json:"class"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

type SeatService interface {
	AddSeat(ctx context.Context, req AddSeatRequest) (*SeatResponse, error)
	GetSeatByID(ctx context.Context, seatID int64) (*SeatResponse, error)
	ListAllSeats(ctx context.Context) ([]SeatResponse, error)
	ListSeatsByHallID(ctx context.Context, hallID int64) ([]SeatResponse, error)
	ListSeatsByClass(ctx context.Context, class domain.SeatClass) ([]SeatResponse, error)
	UpdateSeat(ctx context.Context, seatID int64, req UpdateSeatRequest) (*SeatResponse, error)
	RemoveSeat(ctx context.Context, seatID int64) error
}

type seatService struct {
	repo    domain.SeatRepository
	cache   *redis.Client
	sfGroup singleflight.Group
	logger  *slog.Logger
}

func NewSeatService(repo domain.SeatRepository, cache *redis.Client, logger *slog.Logger) SeatService {
	return &seatService{
		repo:    repo,
		cache:   cache,
		sfGroup: singleflight.Group{},
		logger:  logger,
	}
}

func (s *seatService) AddSeat(ctx context.Context, req AddSeatRequest) (*SeatResponse, error) {
	if req.HallID <= 0 {
		return nil, ErrInvalidHallID
	}

	if req.Class != domain.SeatClassVIP && req.Class != domain.SeatClassBalcony && req.Class != domain.SeatClassRegular {
		return nil, ErrInvalidSeatClass
	}

	seatNumber := req.SeatNumber
	if seatNumber == "" {
		return nil, ErrEmptySeatNumber
	}

	seat := &domain.Seat{
		HallID:     req.HallID,
		SeatNumber: seatNumber,
		Class:      req.Class,
	}

	if err := s.repo.Create(ctx, seat); err != nil {
		return nil, err
	}

	// Increment master collection version
	s.incrementCollectionVersion(ctx, seatVersionKey)

	resp := &SeatResponse{
		ID:         seat.ID,
		HallID:     seat.HallID,
		SeatNumber: seat.SeatNumber,
		Class:      seat.Class,
		CreatedAt:  seat.CreatedAt,
		UpdatedAt:  seat.UpdatedAt,
	}

	s.logger.Info("seat created", "seat_number", seatNumber)
	return resp, nil
}

func (s *seatService) GetSeatByID(ctx context.Context, seatID int64) (*SeatResponse, error) {
	if seatID <= 0 {
		return nil, ErrInvalidSeatID
	}

	// -- Try cache first --
	cacheKey := fmt.Sprintf("seats:%d", seatID)
	if bytes, err := s.cache.Get(ctx, cacheKey).Bytes(); err == nil {
		var seatCache SeatResponse

		if err := json.Unmarshal(bytes, &seatCache); err != nil {
			s.logger.Warn("failed to unmarshal seat cache, falling back to db",
				"err", err,
				"key", cacheKey,
			)

		} else {
			// If the seat ID of the cache is 0, resolves as negative cache hit
			if seatCache.ID == 0 {
				return nil, domain.ErrSeatNotFound
			}

			return &seatCache, nil
		}
	} else if !errors.Is(err, redis.Nil) {
		s.logger.Warn("redis error fetching seats cache", "err", err, "key", cacheKey)
	}

	// -- Cache Miss: Activating Singleflight to prevent Thundering Herd
	val, sfErr, shared := s.sfGroup.Do(cacheKey, func() (interface{}, error) {

		// Detach cancellation from leader's context, but enforces a safety timeout
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()

		seat, err := s.repo.GetByID(dbCtx, seatID)
		if err != nil {
			if errors.Is(err, domain.ErrSeatNotFound) {
				s.logger.Warn("Seat not found in DB, caching negative cache hit to prevent penetration", "id", seatID)

				go s.writeToCacheBackground(ctx, cacheKey, SeatResponse{}, 1*time.Minute)
			}

			return nil, err
		}

		seatResp := &SeatResponse{
			ID:         seat.ID,
			HallID:     seat.HallID,
			SeatNumber: seat.SeatNumber,
			Class:      seat.Class,
			CreatedAt:  seat.CreatedAt,
			UpdatedAt:  seat.UpdatedAt,
		}

		go s.writeToCacheBackground(ctx, cacheKey, seatResp, 1*time.Hour)

		return seatResp, nil
	})

	// -- Handle Singleflight Group processing results --
	if sfErr != nil {
		return nil, sfErr
	}
	if shared {
		s.logger.Info("concurrency suppressed via SingleFlight", "key", cacheKey)
	}
	seat, ok := val.(SeatResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected data type from SingleFlight Group")
	}

	return &seat, nil
}

func (s *seatService) ListSeatsByHallID(ctx context.Context, hallID int64) ([]SeatResponse, error) {
	if hallID <= 0 {
		return nil, ErrInvalidHallID
	}
	// -- Fetch master collection key version (Defaults to 0 if not exists)
	version, err := s.cache.Get(ctx, seatVersionKey).Int64()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			s.logger.Warn("failed fetching seat collection version, falling back to 0",
				"err", err,
				"collection", seatVersionKey,
			)
		}
		version = 0
	}
	// Build a version-scoped cache key
	cacheKey := fmt.Sprintf("seats:h%d:v%d:all", hallID, version)

	// -- Check Cache first --
	if bytes, err := s.cache.Get(ctx, cacheKey).Bytes(); err == nil {
		seatListCache := make([]SeatResponse, 0)

		if err := json.Unmarshal(bytes, &seatListCache); err != nil {
			s.logger.Warn("failed to unmarshal seat list cache, falling back to DB", "err", err)

		} else {
			return seatListCache, nil
		}
	} else if !errors.Is(err, redis.Nil) {
		s.logger.Warn("redis error fetching seat list by hall ID", "err", err)
	}

	// -- Cache Missed: Activating SingleFlight to prevent Thundering Herd --
	val, sfErr, shared := s.sfGroup.Do(cacheKey, func() (interface{}, error) {

		// Detach cancellation from leader's context, but enforces safety timeouts
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()

		seatList, err := s.repo.ListByHallID(dbCtx, hallID)
		if err != nil {
			return nil, err
		}

		seatListResp := make([]SeatResponse, 0, len(seatList))

		for _, seat := range seatList {
			seatListResp = append(seatListResp, SeatResponse{
				ID:         seat.ID,
				HallID:     seat.HallID,
				SeatNumber: seat.SeatNumber,
				Class:      seat.Class,
				CreatedAt:  seat.CreatedAt,
				UpdatedAt:  seat.UpdatedAt,
			})
		}

		// -- Prevent cache penetration: Dynamic TTL --
		var ttl time.Duration
		if len(seatListResp) == 0 {
			ttl = 1 * time.Minute
		} else {
			ttl = 1 * time.Hour
		}

		// -- Update cache asynchronously (fire-and-forget) --
		go s.writeToCacheBackground(ctx, cacheKey, seatListResp, ttl)

		return seatListResp, nil
	})

	// -- Handle Singleflight group processing results --
	if sfErr != nil {
		return nil, sfErr
	}
	if shared {
		s.logger.Info("concurrency suppressed via Signleflight group", "key", cacheKey)
	}
	seats, ok := val.([]SeatResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected data type from SingleFlight group")
	}

	return seats, nil
}

func (s *seatService) ListAllSeats(ctx context.Context) ([]SeatResponse, error) {
	// -- Fetch master collection key version (Defaults to 0 if not exists)
	version, err := s.cache.Get(ctx, seatVersionKey).Int64()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			s.logger.Warn("failed to fetch seat cache collection version, falling back to 0", "err", err)
		}
		version = 0
	}

	// -- Generate a version-scoped cache key
	cacheKey := fmt.Sprintf("seats:v%d:all", version)

	// -- Try Cache first --
	if bytes, err := s.cache.Get(ctx, cacheKey).Bytes(); err == nil {
		seatListCache := make([]SeatResponse, 0)

		if err := json.Unmarshal(bytes, &seatListCache); err != nil {
			s.logger.Warn("failed to unmarshal seat list cache", "err", err)
		} else {
			return seatListCache, nil
		}

	}

	// -- Cache Miss: Activating Singleflight to prevent Thundering Herd
	val, sfErr, shared := s.sfGroup.Do(cacheKey, func() (interface{}, error) {

		// Detach cancellation from leader's context but enforces safety timeout
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()

		seatsList, err := s.repo.ListAll(dbCtx)

		if err != nil {
			return nil, err
		}

		seatListResp := make([]SeatResponse, 0, len(seatsList))

		for _, seat := range seatsList {
			seatListResp = append(seatListResp, SeatResponse{
				ID:         seat.ID,
				HallID:     seat.HallID,
				SeatNumber: seat.SeatNumber,
				Class:      seat.Class,
				CreatedAt:  seat.CreatedAt,
				UpdatedAt:  seat.UpdatedAt,
			})
		}

		// -- Prevent Cache Penetration: Dynamic TTL --
		var ttl time.Duration
		if len(seatListResp) == 0 {
			ttl = 1 * time.Minute
		} else {
			ttl = 1 * time.Hour
		}

		// -- Update cache asynchronously (fire-and-forget) --
		go s.writeToCacheBackground(ctx, cacheKey, seatListResp, ttl)

		return seatListResp, nil
	})

	// -- Handle Singleflight processing results --
	if sfErr != nil {
		return nil, sfErr
	}
	if shared {
		s.logger.Info("concurrency suppressed via Singleflight", "key", cacheKey)
	}
	seat, ok := val.([]SeatResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected data type from Singleflight group")
	}

	return seat, nil
}

func (s *seatService) ListSeatsByClass(ctx context.Context, class domain.SeatClass) ([]SeatResponse, error) {
	if class != domain.SeatClassVIP && class != domain.SeatClassBalcony && class != domain.SeatClassRegular {
		return nil, ErrInvalidSeatClass
	}

	seatList, err := s.repo.ListByClass(ctx, class)

	if err != nil {
		return nil, err
	}

	seatListResp := make([]SeatResponse, 0, len(seatList))
	for _, seat := range seatList {
		seatListResp = append(seatListResp, SeatResponse{
			ID:         seat.ID,
			HallID:     seat.HallID,
			SeatNumber: seat.SeatNumber,
			Class:      seat.Class,
			CreatedAt:  seat.CreatedAt,
			UpdatedAt:  seat.UpdatedAt,
		})
	}

	return seatListResp, nil
}

func (s *seatService) UpdateSeat(ctx context.Context, seatID int64, req UpdateSeatRequest) (*SeatResponse, error) {
	seatNumber := req.SeatNumber

	if seatNumber == "" {
		return nil, ErrEmptySeatNumber
	}

	if req.Class != domain.SeatClassVIP && req.Class != domain.SeatClassBalcony && req.Class != domain.SeatClassRegular {
		return nil, ErrInvalidSeatClass
	}

	seat := &domain.Seat{
		ID:         seatID,
		SeatNumber: seatNumber,
		Class:      req.Class,
	}

	if err := s.repo.Update(ctx, seat); err != nil {
		return nil, err
	}

	// Evict stale data
	specificCacheKey := fmt.Sprintf("seats:%d", seatID)
	s.evictCache(ctx, specificCacheKey)

	// Increment collection version stale older cache before returning status code
	s.incrementCollectionVersion(ctx, seatVersionKey)

	seatResp := &SeatResponse{
		ID:         seat.ID,
		HallID:     seat.HallID,
		SeatNumber: seat.SeatNumber,
		Class:      seat.Class,
		CreatedAt:  seat.CreatedAt,
		UpdatedAt:  seat.UpdatedAt,
	}

	s.logger.Info("seat updated", "seat_number", seat.SeatNumber)
	return seatResp, nil
}

func (s *seatService) RemoveSeat(ctx context.Context, seatID int64) error {
	if seatID <= 0 {
		return ErrInvalidSeatID
	}

	if err := s.repo.Delete(ctx, seatID); err != nil {
		return err
	}
	// Evict older specific cache key
	specificCacheKey := fmt.Sprintf("seats:%d", seatID)
	s.evictCache(ctx, specificCacheKey)

	// Increment master collection key to evict stale cache before returning status
	s.incrementCollectionVersion(ctx, seatVersionKey)

	s.logger.Info("seat deleted", "seat_id", seatID)
	return nil
}

// Helper method for writing to cache asynchronously
func (s *seatService) writeToCacheBackground(ctx context.Context, cachekey string, data any, ttl time.Duration) {
	bgCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()

	bytes, err := json.Marshal(data)

	if err != nil {
		s.logger.Warn("failed to marshall seats cache data", "err", err, "key", cachekey)
		return
	}

	if err := s.cache.Set(bgCtx, cachekey, bytes, ttl).Err(); err != nil {
		s.logger.Warn("failed to cache seats data to redis in the background", "err", err, "key", cachekey)
	}
}

// Helper method for eviction of stale cache synchronously
func (s *seatService) evictCache(ctx context.Context, cacheKeys ...string) {
	bgCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()

	if err := s.cache.Del(bgCtx, cacheKeys...).Err(); err != nil {
		s.logger.Warn("failed to evict stale cache from redis", "err", err, "keys", cacheKeys)
		return
	}
}

// Helper method for incrementing master collection cache key verisoning
func (s *seatService) incrementCollectionVersion(ctx context.Context, collectionKey string) {
	bgCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()

	if err := s.cache.Incr(bgCtx, collectionKey).Err(); err != nil {
		s.logger.Warn("failed to increment collection key version pointer",
			"err", err,
			"collection", collectionKey,
		)
	}
}
