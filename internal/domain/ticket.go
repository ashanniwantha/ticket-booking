package domain

import (
	"context"
	"time"
)

type Ticket struct {
	ID          int64
	ScreeningID int64
	SeatID      int64
	UserID      int64
	Status      TicketStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type TicketStatus string

const (
	TicketStatusHold      TicketStatus = "hold"
	TicketStatusBooked    TicketStatus = "booked"
	TicketStatusCancelled TicketStatus = "cancelled"
)

// TicketRepository defines the contract for data access.
type TicketRepository interface {
	Create(ctx context.Context, t *Ticket) (*Ticket, error)
	GettByID(ctx context.Context, id int64) (*Ticket, error)
	ListByScreening(ctx context.Context, screeningID int64) ([]Ticket, error)
	ListByUser(ctx context.Context, userID int64) ([]Ticket, error)
	UpdateStatus(ctx context.Context, ticketID int64, status TicketStatus) error
}
