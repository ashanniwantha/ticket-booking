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
	sfGroup singleflight.Group
	logger  *slog.Logger
}

func NewMovieService(repo domain.MovieRepository, cache *redis.Client, logger *slog.Logger) MovieService {
	return &movieService{
		repo:   repo,
		cache:  cache,
		logger: logger,
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

	movie, err := s.repo.GetByID(ctx, movieID)

	if err != nil {
		return nil, err
	}

	resp := &MovieResponse{
		ID:          movie.ID,
		Title:       movie.Title,
		Description: movie.Description,
		CreatedAt:   movie.CreatedAt,
		UpdatedAt:   movie.UpdatedAt,
	}

	return resp, nil
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
	const cacheKey = "movies:all"

	// 1. -- Try Cache First --
	if cache, err := s.cache.Get(ctx, cacheKey).Bytes(); err == nil {
		var cachedMovies []MovieResponse
		if err := json.Unmarshal(cache, &cachedMovies); err != nil {
			s.logger.Warn("failed to unmarshal cached data, falling back to db", "err", err)
		} else {
			return cachedMovies, nil
		}
	} else if !errors.Is(err, redis.Nil) {
		s.logger.Warn("redis error fetching all movies, falling back to db", "err", err)
	}

	// 2. -- Cache Miss: Activate Singleflight to prevent Thundering Herd --
	val, err, _ := s.sfGroup.Do(cacheKey, func() (interface{}, error) {

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

		// 3. -- Cache Penetration Protection: Dynamic TTL --
		var ttl time.Duration
		if len(moviesListResp) == 0 {
			ttl = 1 * time.Minute
			s.logger.Info("database table is empty, caching empty result with short TTL")
		} else {
			ttl = 5 * time.Minute
		}

		// 4. -- Update Cache Asynchronously (Fire-and-Forget) --
		go func(dataToCache []MovieResponse, cacheTTL time.Duration) {
			bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			data, err := json.Marshal(dataToCache)
			if err != nil {
				s.logger.Warn("failed to marshal movies for caching", "err", err)
				return
			}

			if err := s.cache.Set(bgCtx, cacheKey, data, cacheTTL); err != nil {
				s.logger.Warn("failed to cache movies in background", "err", err)
			}
		}(moviesListResp, ttl)

		return moviesListResp, nil
	})

	// 5. -- Handle Singleflight Processing Results --
	if err != nil {
		return nil, err
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

	resp := &MovieResponse{
		ID:          movie.ID,
		Title:       movie.Title,
		Description: movie.Description,
		CreatedAt:   movie.CreatedAt,
		UpdatedAt:   movie.UpdatedAt,
	}

	s.logger.Info("movie updated", "title", movie.Title)
	return resp, nil
}

func (s *movieService) RemoveMovie(ctx context.Context, movieID int64) error {
	if movieID <= 0 {
		return ErrInvalidMovieID
	}

	if err := s.repo.Delete(ctx, movieID); err != nil {
		return err
	}

	s.logger.Info("movie deleted", "movied_id", movieID)
	return nil
}
