package domain

import (
	"context"
	"errors"
	"time"
)

type Seat struct {
	ID         int64
	HallID     int64
	SeatNumber string
	Class      SeatClass
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type SeatClass string

const (
	SeatClassVIP     SeatClass = "vip"
	SeatClassBalcony SeatClass = "balcony"
	SeatClassRegular SeatClass = "regular"
)

var (
	ErrSeatNotFound            = errors.New("seat not found")
	ErrDuplicateSeat           = errors.New("duplicate seat")
	ErrDuplicateSeatNumber     = errors.New("duplicate seat number")
	ErrSeatForeignKeyViolation = errors.New("seat foreign key violation")
)

type SeatRepository interface {
	Create(ctx context.Context, s *Seat) error
	GetByID(ctx context.Context, seatID int64) (*Seat, error)
	ListByHallID(ctx context.Context, hallID int64) ([]Seat, error)
	ListByClass(ctx context.Context, class SeatClass) ([]Seat, error)
	Update(ctx context.Context, s *Seat) error
	Delete(ctx context.Context, seatID int64) error
}
