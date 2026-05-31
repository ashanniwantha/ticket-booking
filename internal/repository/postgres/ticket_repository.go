package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/ashanniwantha/ticket-booking/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TicketRepo struct {
	pool *pgxpool.Pool
}

func NewTicketRepo(pool *pgxpool.Pool) *TicketRepo {
	return &TicketRepo{pool: pool}
}

func (r *TicketRepo) conn(ctx context.Context) DBTX {
	if tx := TxFromContext(ctx); tx != nil {
		return tx
	}
	return r.pool
}

func (r *TicketRepo) Create(ctx context.Context, ticket *domain.Ticket) error {
	query := `
	INSERT INTO tickets (screening_id, seat_id, user_id, status)
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at, updated_at
	`
	row := r.conn(ctx).QueryRow(
		ctx, query, ticket.ScreeningID, ticket.SeatID, ticket.UserID, ticket.Status,
	)

	if err := row.Scan(
		&ticket.ID, &ticket.CreatedAt, &ticket.UpdatedAt,
	); err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23503":
				switch pgErr.ConstraintName {
				case "fk_tickets_screenings":
					return domain.ErrScreeningNotFound
				case "fk_tickets_seats":
					return domain.ErrSeatNotFound
				case "fk_tickets_users":
					return domain.ErrUserNotFound
				}
			case "23505":
				if pgErr.ConstraintName == "uq_active_tickets" {
					return domain.ErrSeatUnavailable
				}
			case "23514":
				return domain.ErrInvalidTicketData
			}
		}

		return fmt.Errorf("creating ticket (repo): %w", err)
	}
	return nil
}

func (r *TicketRepo) GetByID(ctx context.Context, ticketID int64) (*domain.Ticket, error) {
	var ticket domain.Ticket

	query := `
	SELECT id, screening_id, seat_id, user_id, status, created_at, updated_at
	FROM tickets
	WHERE id=$1
	`
	row := r.conn(ctx).QueryRow(ctx, query, ticketID)

	if err := row.Scan(
		&ticket.ID, &ticket.ScreeningID, &ticket.SeatID, &ticket.UserID, &ticket.Status, &ticket.CreatedAt, &ticket.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTicketNotFound
		}

		return nil, fmt.Errorf("getting ticket by ID (repo): %w", err)
	}

	return &ticket, nil
}

func (r *TicketRepo) GetByIDForUpdate(ctx context.Context, ticketID int64) (*domain.Ticket, error) {
	var ticket domain.Ticket

	query := `
		SELECT id, screening_id, seat_id, user_id, status, created_at, updated_at
		FROM tickets
		WHERE id=$1
		FOR UPDATE
	`
	row := r.conn(ctx).QueryRow(ctx, query, ticketID)
	if err := row.Scan(
		&ticket.ID, &ticket.ScreeningID, &ticket.SeatID, &ticket.UserID, &ticket.Status, &ticket.CreatedAt, &ticket.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTicketNotFound
		}

		return nil, fmt.Errorf("getting ticket by ID for update (repo): %w", err)
	}
	return &ticket, nil
}

func (r *TicketRepo) ListByScreening(ctx context.Context, screeningID int64) ([]domain.Ticket, error) {
	query := `
	SELECT id, screening_id, seat_id, user_id, status, created_at, updated_at
	FROM tickets
	WHERE screening_id=$1
	`
	rows, err := r.conn(ctx).Query(ctx, query, screeningID)
	if err != nil {
		return nil, fmt.Errorf("querying ticket list by screening (repo): %w", err)
	}
	defer rows.Close()
	ticketList := make([]domain.Ticket, 0)

	for rows.Next() {
		var ticket domain.Ticket

		if err := rows.Scan(
			&ticket.ID, &ticket.ScreeningID, &ticket.SeatID, &ticket.UserID,
			&ticket.Status, &ticket.CreatedAt, &ticket.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning ticket list (repo): %w", err)
		}
		ticketList = append(ticketList, ticket)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scanning ticket list completed with errors (repo) %w", err)
	}

	return ticketList, nil
}

func (r *TicketRepo) ListByUser(ctx context.Context, userID int64) ([]domain.Ticket, error) {
	query := `
	SELECT id, screening_id, seat_id, user_id, status, created_at, updated_at
	FROM tickets
	WHERE user_id=$1
	`
	rows, err := r.conn(ctx).Query(ctx, query, userID)

	if err != nil {
		return nil, fmt.Errorf("querying ticket list by users (repo): %w", err)
	}
	defer rows.Close()
	ticketList := make([]domain.Ticket, 0)

	for rows.Next() {
		var ticket domain.Ticket

		if err := rows.Scan(
			&ticket.ID, &ticket.ScreeningID, &ticket.SeatID, &ticket.UserID,
			&ticket.Status, &ticket.CreatedAt, &ticket.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning ticket list (repo): %w", err)
		}
		ticketList = append(ticketList, ticket)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scanning ticket list completed with errors (repo): %w", err)
	}

	return ticketList, nil
}

func (r *TicketRepo) UpdateStatus(ctx context.Context, ticketID int64, status domain.TicketStatus) error {
	query := `
	UPDATE tickets
	SET status=$2
	WHERE id=$1
	`
	commandTag, err := r.conn(ctx).Exec(ctx, query, ticketID, status)
	if err != nil {
		return fmt.Errorf("updating ticket status (repo): %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return domain.ErrTicketNotFound
	}

	return nil
}

func (r *TicketRepo) Update(ctx context.Context, t *domain.Ticket) error {
	query := `
	UPDATE tickets
	SET screening_id=$2, seat_id=$3, user_id=$4, status=$5
	WHERE id=$1
	RETURNING screening_id, seat_id, user_id, status, created_at, updated_at
	`
	row := r.conn(ctx).QueryRow(
		ctx, query, t.ID,
		t.ScreeningID,
		t.SeatID,
		t.UserID,
		t.Status,
	)

	if err := row.Scan(
		&t.ScreeningID, &t.SeatID, &t.UserID,
		&t.Status, &t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrTicketNotFound
		}
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23503":
				switch pgErr.ConstraintName {
				case "fk_tickets_screenings":
					return domain.ErrScreeningNotFound
				case "fk_tickets_seats":
					return domain.ErrSeatNotFound
				case "fk_tickets_users":
					return domain.ErrUserNotFound
				}
			case "23505":
				if pgErr.ConstraintName == "uq_active_tickets" {
					return domain.ErrSeatUnavailable
				}
			case "23514":
				return domain.ErrInvalidTicketData
			}
		}
		return fmt.Errorf("updating tickets (repo): %w", err)
	}
	return nil
}

func (r *TicketRepo) Delete(ctx context.Context, ticketID int64) error {
	query := `
	DELETE FROM tickets
	WHERE id=$1
	`
	commandTag, err := r.conn(ctx).Exec(ctx, query, ticketID)
	if err != nil {
		return fmt.Errorf("deleting ticket (repo): %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return domain.ErrTicketNotFound
	}

	return nil
}
