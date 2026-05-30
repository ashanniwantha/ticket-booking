package domain

import (
	"context"
	"errors"
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

var (
	ErrScreeningForeignKeyViolation = errors.New("screening foreign key violation")
	ErrInvalidScreeningData         = errors.New("invalid screening data")
	ErrScreeningTimeConflict        = errors.New("screening time conflicts with an existing screening")
	ErrScreeningNotFound            = errors.New("screening not found")
)

type ScreeningRepository interface {
	Create(ctx context.Context, s *Screening) error
	GetByID(ctx context.Context, screeningID int64) (*Screening, error)
	ListAll(ctx context.Context) ([]Screening, error)
	ListByMovie(ctx context.Context, movieID int64) ([]Screening, error)
	ListByHall(ctx context.Context, hallID int64) ([]Screening, error)
	Update(ctx context.Context, s *Screening) error
	Delete(ctx context.Context, screeningID int64) error
}
