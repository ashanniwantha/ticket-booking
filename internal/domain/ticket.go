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
	Create(ctx context.Context, ticket *Ticket) error
	GetByID(ctx context.Context, ticketID int64) (*Ticket, error)
	ListByScreening(ctx context.Context, screeningID int64) ([]Ticket, error)
	ListByUser(ctx context.Context, userID int64) ([]Ticket, error)
	Update(ctx context.Context, ticket *Ticket) error
	UpdateStatus(ctx context.Context, ticketID int64, status TicketStatus) error
	// GetByIDForUpdate is used within a transaction to lock the row
	GetByIDForUpdate(ctx context.Context, ticketID int64) (*Ticket, error)
	Delete(ctx context.Context, TicketID int64) error
}
