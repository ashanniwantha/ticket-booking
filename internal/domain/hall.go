package domain

import (
	"context"
	"errors"
	"time"
)

type Hall struct {
	ID        int64
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

var (
	ErrDuplicateHallName = errors.New("duplicate halls names")
	ErrHallNotFound      = errors.New("hall not found")
	ErrInvalidSeatData   = errors.New("invalid seat data")
)

type HallRepository interface {
	Create(ctx context.Context, h *Hall) error
	GetByID(ctx context.Context, hallID int64) (*Hall, error)
	Update(ctx context.Context, h *Hall) error
	Delete(ctx context.Context, hallID int64) error
}
