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
	movieVersionKey = "movies:version"
)

type AddMovieRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type UpdateMovieRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type MovieResponse struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type MovieService interface {
	AddMovie(ctx context.Context, req AddMovieRequest) (*MovieResponse, error)
	GetMovieByID(ctx context.Context, movieID int64) (*MovieResponse, error)
	ListMovieByTitle(ctx context.Context, movieTitle string) ([]MovieResponse, error)
	ListAllMovies(ctx context.Context) ([]MovieResponse, error)
	UpdateMovie(ctx context.Context, movieID int64, req UpdateMovieRequest) (*MovieResponse, error)
	RemoveMovie(ctx context.Context, movieID int64) error
}

type movieService struct {
	repo    domain.MovieRepository
	cache   *redis.Client
	logger  *slog.Logger
	sfGroup singleflight.Group
}

func NewMovieService(repo domain.MovieRepository, cache *redis.Client, logger *slog.Logger) MovieService {
	return &movieService{
		repo:    repo,
		cache:   cache,
		logger:  logger,
		sfGroup: singleflight.Group{},
	}
}

func (s *movieService) AddMovie(ctx context.Context, req AddMovieRequest) (*MovieResponse, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, domain.ErrMovieTitleEmpty
	}

	movie := &domain.Movie{
		Title:       req.Title,
		Description: req.Description,
	}

	if err := s.repo.Create(ctx, movie); err != nil {
		return nil, err
	}
	// Bumping master version
	s.incrementCollectionVersion(ctx, movieVersionKey)

	resp := &MovieResponse{
		ID:          movie.ID,
		Title:       movie.Title,
		Description: movie.Description,
		CreatedAt:   movie.CreatedAt,
		UpdatedAt:   movie.UpdatedAt,
	}

	s.logger.Info("movie added", "title", movie.Title)
	return resp, nil
}

func (s *movieService) GetMovieByID(ctx context.Context, movieID int64) (*MovieResponse, error) {
	if movieID <= 0 {
		return nil, ErrInvalidMovieID
	}

	// 1. -- Try cache first --
	cacheKey := fmt.Sprintf("movies:%d", movieID)
	if cache, err := s.cache.Get(ctx, cacheKey).Bytes(); err == nil {
		var moviesCache MovieResponse

		if err := json.Unmarshal(cache, &moviesCache); err == nil {
			return &moviesCache, nil
		} else {
			s.logger.Warn("failed to unmarshal cached data, falling back to db", "err", err)
		}

	} else if !errors.Is(err, redis.Nil) {
		s.logger.Warn("redis error fetching movies by ID, falling back to db", "err", err)
	}

	// 2. -- Cache Missed: Activate Sigleflight to prevent Thundering Herd --
	val, err, shared := s.sfGroup.Do(cacheKey, func() (interface{}, error) {

		// Detach cancellation from leader's context but enforces a timeout
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()

		// This block executes exactly ONCE for concurrent requests
		movie, err := s.repo.GetByID(dbCtx, movieID)
		if err != nil {
			return nil, err
		}

		movieResp := MovieResponse{
			ID:          movie.ID,
			Title:       movie.Title,
			Description: movie.Description,
			CreatedAt:   movie.CreatedAt,
			UpdatedAt:   movie.UpdatedAt,
		}

		// 3. -- Update Cache Asynchronously (Fire-and-Forget) --
		go s.writeToCacheBackground(cacheKey, movieResp, 5*time.Minute)

		return movieResp, nil
	})

	// 5. -- Handle SingleFlight processign results --
	if err != nil {
		return nil, err
	}

	movie, ok := val.(MovieResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected data type from Singleflight group")
	}
	if shared {
		s.logger.Info("Concurrency suppressed via Singleflight", "key", cacheKey)
	}
	return &movie, nil
}

func (s *movieService) ListMovieByTitle(ctx context.Context, movieTitle string) ([]MovieResponse, error) {
	if movieTitle == "" {
		return nil, domain.ErrMovieTitleEmpty
	}

	moviesList, err := s.repo.GetByTitle(ctx, movieTitle)
	if err != nil {
		return nil, err
	}

	moviesListResp := make([]MovieResponse, 0, len(moviesList))

	for _, movie := range moviesList {
		moviesListResp = append(moviesListResp, MovieResponse{
			ID:          movie.ID,
			Title:       movie.Title,
			Description: movie.Description,
			CreatedAt:   movie.CreatedAt,
			UpdatedAt:   movie.UpdatedAt,
		})
	}

	return moviesListResp, nil
}

func (s *movieService) ListAllMovies(ctx context.Context) ([]MovieResponse, error) {
	// 1. -- Fetch master collection versions (Defaults to 0 if not exists)
	version, err := s.cache.Get(ctx, movieVersionKey).Int64()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			s.logger.Warn("failed fetching movie cache collection version, falling back to 0", "err", err)
		}
		version = 0
	}

	// 2. -- Generate a version-scoped cache key which natively scales --
	cacheKey := fmt.Sprintf("movies:v%d:all", version)

	// 3. -- Try Cache First --
	if cache, err := s.cache.Get(ctx, cacheKey).Bytes(); err == nil {
		var cachedMovies []MovieResponse
		if err := json.Unmarshal(cache, &cachedMovies); err == nil {
			return cachedMovies, nil
		} else {
			s.logger.Warn("failed to unmarshal cached data, falling back to db", "err", err)
		}

	} else if !errors.Is(err, redis.Nil) {
		s.logger.Warn("redis error fetching all movies, falling back to db", "err", err)
	}

	// 4. -- Cache Miss: Activate Singleflight to prevent Thundering Herd --
	val, err, shared := s.sfGroup.Do(cacheKey, func() (interface{}, error) {

		// Detach cancellation from leader's context but enforce a safety timeout
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()

		// This inside block executes exactly ONCE for concurrent requests
		moviesList, err := s.repo.ListAll(dbCtx)
		if err != nil {
			return nil, err
		}

		moviesListResp := make([]MovieResponse, 0, len(moviesList))
		for _, movie := range moviesList {
			moviesListResp = append(moviesListResp, MovieResponse{
				ID:          movie.ID,
				Title:       movie.Title,
				Description: movie.Description,
				CreatedAt:   movie.CreatedAt,
				UpdatedAt:   movie.UpdatedAt,
			})
		}

		// 5. -- Cache Penetration Protection: Dynamic TTL --
		var ttl time.Duration
		if len(moviesListResp) == 0 {
			ttl = 1 * time.Minute
			s.logger.Info("database table is empty, caching empty result with short TTL")
		} else {
			ttl = 5 * time.Minute
		}

		// 6. -- Update Cache Asynchronously (Fire-and-Forget) --
		go s.writeToCacheBackground(cacheKey, moviesListResp, ttl)

		return moviesListResp, nil
	})

	// 7. -- Handle Singleflight Processing Results --
	if err != nil {
		return nil, err
	}
	if shared {
		s.logger.Info("concurrency suppressed via Singleflight", "key", cacheKey)
	}
	movies, ok := val.([]MovieResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected data type from singleflight group")
	}

	return movies, nil
}

func (s *movieService) UpdateMovie(ctx context.Context, movieID int64, req UpdateMovieRequest) (*MovieResponse, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, domain.ErrMovieTitleEmpty
	}

	movie := &domain.Movie{
		ID:          movieID,
		Title:       req.Title,
		Description: req.Description,
	}

	if err := s.repo.Update(ctx, movie); err != nil {
		return nil, err
	}

	// Evict specific resource pointer synchronously
	specificMovieKey := fmt.Sprintf("movies:%d", movie.ID)
	s.evictCache(ctx, specificMovieKey)

	// Increment collection state synchronously before returning status code
	s.incrementCollectionVersion(ctx, movieVersionKey)

	resp := &MovieResponse{
		ID:          movie.ID,
		Title:       movie.Title,
		Description: movie.Description,
		CreatedAt:   movie.CreatedAt,
		UpdatedAt:   movie.UpdatedAt,
	}

	s.logger.Info("movie updated successfully", "title", movie.Title)
	return resp, nil
}

func (s *movieService) RemoveMovie(ctx context.Context, movieID int64) error {
	if movieID <= 0 {
		return ErrInvalidMovieID
	}

	if err := s.repo.Delete(ctx, movieID); err != nil {
		return err
	}

	// Evict specific resource pointer synchronously
	specificMovieKey := fmt.Sprintf("movies:%d", movieID)
	s.evictCache(ctx, specificMovieKey)
	// Increment collection state synchronously
	s.incrementCollectionVersion(ctx, movieVersionKey)

	s.logger.Info("movie deleted", "movie_id", movieID)
	return nil
}

// Helper method for movie service to write cache
func (s *movieService) writeToCacheBackground(cacheKey string, data any, ttl time.Duration) {
	bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	bytes, err := json.Marshal(data)
	if err != nil {
		s.logger.Warn("failed to marshall cache movie data", "err", err, "key", cacheKey)
		return
	}
	if err := s.cache.Set(bgCtx, cacheKey, bytes, ttl).Err(); err != nil {
		s.logger.Warn("failed to cache data to redis in the background", "err", err, "key", cacheKey)
	}
}

// Helper method for movie service to remove stale cache synchronously
func (s *movieService) evictCache(ctx context.Context, cacheKeys ...string) {
	bgCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()

	if err := s.cache.Del(bgCtx, cacheKeys...).Err(); err != nil {
		s.logger.Warn("failed to evict stale cache from redis", "err", err, "keys", cacheKeys)
	}
}

// Help method for incrementing the master collection cache version
func (s *movieService) incrementCollectionVersion(ctx context.Context, collectionKey string) {
	bgCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()

	// INCR created key with value 1 if doesn't exists, or incrment by 1 natively
	if err := s.cache.Incr(bgCtx, collectionKey).Err(); err != nil {
		s.logger.Warn("failed to increment collection version pointer",
			"err", err,
			"collection", collectionKey,
		)
	}
}
