package domain

import (
	"context"
	"time"
)

type Movie struct {
	ID          int64
	Title       string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type MovieRepository interface {
	CreateMovie(ctx context.Context, m *Movie) (*Movie, error)
	GetMovieByID(ctx context.Context, movieID int64) (*Movie, error)
	UpdateMovie(ctx context.Context, m *Movie) (*Movie, error)
	DeleteMovie(ctx context.Context, movieID int64) error
}
