package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ashanniwantha/ticket-booking/internal/domain"
	"github.com/ashanniwantha/ticket-booking/internal/repository/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HoldSeatRequest struct {
	ScreeningID int64 `json:"screening_id"`
	SeatID      int64 `json:"seat_id"`
	UserID      int64 `json:"user_id"`
}

type TicketResponse struct {
	ID          int64               `json:"id"`
	ScreeningID int64               `json:"screening_id"`
	SeatID      int64               `json:"seat_id"`
	UserID      int64               `json:"user_id"`
	Status      domain.TicketStatus `json:"status"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

type BookingService interface {
	HoldSeat(ctx context.Context, req HoldSeatRequest) (*TicketResponse, error)
	ConfirmHold(ctx context.Context, ticketID int64, userID int64) error
	CancelTicket(ctx context.Context, ticketID int64, userID int64) error
}

type bookingService struct {
	ticketRepo    domain.TicketRepository
	screeningRepo domain.ScreeningRepository
	seatRepo      domain.SeatRepository
	logger        *slog.Logger
	pool          *pgxpool.Pool
}

func NewBookingService(
	ticketRepo domain.TicketRepository,
	screeningRepo domain.ScreeningRepository,
	seatRepo domain.SeatRepository,
	logger *slog.Logger,
	pool *pgxpool.Pool,
) BookingService {
	return &bookingService{
		ticketRepo:    ticketRepo,
		screeningRepo: screeningRepo,
		seatRepo:      seatRepo,
		logger:        logger,
		pool:          pool,
	}
}

func (s *bookingService) HoldSeat(ctx context.Context, req HoldSeatRequest) (*TicketResponse, error) {
	if req.ScreeningID <= 0 {
		return nil, ErrInvalidScreeningID
	}
	if req.SeatID <= 0 {
		return nil, ErrInvalidSeatID
	}
	if req.UserID <= 0 {
		return nil, ErrInvalidUserID
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("intializing hold seat transaction (service): %w", err)
	}
	defer tx.Rollback(ctx)

	//Embed transaction in context
	ctx = postgres.WithTx(ctx, tx)

	screening, err := s.screeningRepo.GetByID(ctx, req.ScreeningID)
	if err != nil {
		return nil, err
	}
	seat, err := s.seatRepo.GetByID(ctx, req.SeatID)
	if err != nil {
		return nil, err
	}
	if screening.HallID != seat.HallID {
		return nil, ErrSeatScreeningHallMismatch
	}

	ticket := &domain.Ticket{
		ScreeningID: req.ScreeningID,
		SeatID:      req.SeatID,
		UserID:      req.UserID,
		Status:      domain.TicketStatusHold,
	}

	if err := s.ticketRepo.Create(ctx, ticket); err != nil {
		return nil, err
	}

	ticketResp := &TicketResponse{
		ID:          ticket.ID,
		ScreeningID: ticket.ScreeningID,
		SeatID:      ticket.SeatID,
		UserID:      ticket.UserID,
		Status:      ticket.Status,
		CreatedAt:   ticket.CreatedAt,
		UpdatedAt:   ticket.UpdatedAt,
	}
	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil,
			fmt.Errorf("committing hold seat transaction (service): %w", err)
	}

	return ticketResp, nil
}

func (s *bookingService) ConfirmHold(ctx context.Context, ticketID int64, userID int64) error {
	if ticketID <= 0 {
		return ErrInvalidTicketID
	}
	if userID <= 0 {
		return ErrInvalidUserID
	}

	// Starts transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("intializing confirm hold transaction (service): %w", err)
	}
	defer tx.Rollback(ctx)

	// Embeds transaction in ctx
	ctx = postgres.WithTx(ctx, tx)

	ticket, err := s.ticketRepo.GetByIDForUpdate(ctx, ticketID)
	if err != nil {
		return err
	}

	if ticket.Status != domain.TicketStatusHold {
		return ErrTicketNotHold
	}

	if ticket.UserID != userID {
		return domain.ErrTicketNotFound
	}

	if err = s.ticketRepo.UpdateStatus(ctx, ticketID, domain.TicketStatusBooked); err != nil {
		return err
	}
	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing confirm hold transaction (service): %w", err)
	}

	return nil
}

func (s *bookingService) CancelTicket(ctx context.Context, ticketID int64, userID int64) error {
	if ticketID <= 0 {
		return ErrInvalidTicketID
	}
	if userID <= 0 {
		return ErrInvalidUserID
	}

	// Start transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("intializing cancel ticket transaction (service): %w", err)
	}
	defer tx.Rollback(ctx)

	// Embed transaction into ctx
	ctx = postgres.WithTx(ctx, tx)

	ticket, err := s.ticketRepo.GetByIDForUpdate(ctx, ticketID)
	if err != nil {
		return err
	}
	if ticket.Status == domain.TicketStatusCancelled {
		return ErrTicketAlreadyCancelled
	}
	if ticket.UserID != userID {
		return domain.ErrTicketNotFound
	}

	if err := s.ticketRepo.UpdateStatus(ctx, ticketID, domain.TicketStatusCancelled); err != nil {
		return err
	}
	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commiting cancel ticket transaction: %w", err)
	}

	return nil
}
