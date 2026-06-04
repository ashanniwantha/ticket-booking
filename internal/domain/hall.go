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
	Create(ctx context.Context, h *Hall) error
	GetByID(ctx context.Context, hallID int64) (*Hall, error)
	ListAll(ctx context.Context) ([]Hall, error)
	Update(ctx context.Context, h *Hall) error
	Delete(ctx context.Context, hallID int64) error
}
