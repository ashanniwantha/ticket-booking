package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/ashanniwantha/ticket-booking/internal/domain"
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

var (
	ErrInvalidMovieID = errors.New("invalid movie id")
)

type MovieService interface {
	AddMovie(ctx context.Context, req AddMovieRequest) (*MovieResponse, error)
	GetMovieByID(ctx context.Context, movieID int64) (*MovieResponse, error)
	GetMovieByTitle(ctx context.Context, movieTitle string) (*MovieResponse, error)
	UpdateMovie(ctx context.Context, movieID int64, req UpdateMovieRequest) (*MovieResponse, error)
	RemoveMovie(ctx context.Context, movieID int64) error
}

type movieService struct {
	repo   domain.MovieRepository
	logger *slog.Logger
}

func NewMovieService(repo domain.MovieRepository, logger *slog.Logger) MovieService {
	return &movieService{
		repo:   repo,
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

func (s *movieService) GetMovieByTitle(ctx context.Context, movieTitle string) (*MovieResponse, error) {
	if movieTitle == "" {
		return nil, domain.ErrMovieTitleEmpty
	}

	movie, err := s.repo.GetByTitle(ctx, movieTitle)

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
