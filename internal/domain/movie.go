package domain

import (
	"context"
	"errors"
	"time"
)

type Movie struct {
	ID          int64
	Title       string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

var (
	ErrMovieNotFound   = errors.New("movie not found")
	ErrMovieTitleEmpty = errors.New("movie title empty")
)

type MovieRepository interface {
	Create(ctx context.Context, m *Movie) (*Movie, error)
	GetByID(ctx context.Context, movieID int64) (*Movie, error)
	GetByTitle(ctx context.Context, movieTitle string) (*Movie, error)
	Update(ctx context.Context, m *Movie) error
	Delete(ctx context.Context, movieID int64) error
}
