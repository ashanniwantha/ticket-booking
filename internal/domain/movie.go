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
	Create(ctx context.Context, m *Movie) error
	GetByID(ctx context.Context, movieID int64) (*Movie, error)
	GetByTitle(ctx context.Context, movieTitle string) ([]Movie, error)
	ListAll(ctx context.Context) ([]Movie, error)
	Update(ctx context.Context, m *Movie) error
	Delete(ctx context.Context, movieID int64) error
}
