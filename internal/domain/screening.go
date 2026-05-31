package domain

import (
	"context"
	"time"
)

type Screening struct {
	ID        int64
	MovieID   int64
	HallID    int64
	StartTime time.Time
	EndTime   time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ScreeningRepository interface {
	Create(ctx context.Context, s *Screening) error
	GetByID(ctx context.Context, screeningID int64) (*Screening, error)
	ListAll(ctx context.Context) ([]Screening, error)
	ListByMovie(ctx context.Context, movieID int64) ([]Screening, error)
	ListByHall(ctx context.Context, hallID int64) ([]Screening, error)
	ListByMovieAndHall(ctx context.Context, movieID int64, hallID int64) ([]Screening, error)
	Update(ctx context.Context, s *Screening) error
	Delete(ctx context.Context, screeningID int64) error
}
