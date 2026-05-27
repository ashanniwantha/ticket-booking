package domain

import (
	"context"
	"time"
)

type Hall struct {
	ID        int64
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type HallRepository interface {
	CreateHall(ctx context.Context, h *Hall) (*Hall, error)
	GetHallByID(ctx context.Context, hallID int64) (*Hall, error)
	UpdateHall(ctx context.Context, h *Hall) (*Hall, error)
	DeleteHall(ctx context.Context, hallID int64) error
}
