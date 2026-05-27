package domain

import (
	"context"
	"time"
)

type Screening struct {
	ID            int64
	MovieID       int64
	HallID        int64
	ScreeningTime time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ScreeningRepository interface {
	Create(ctx context.Context, s *Screening) (*Screening, error)
	GetByID(ctx context.Context, id int64) (*Screening, error)
	Update(ctx context.Context, s *Screening) (*Screening, error)
	ListByMovie(ctx context.Context, movieID int64) ([]Screening, error)
	ListByHall(ctx context.Context, hallID int64) ([]Screening, error)
}
