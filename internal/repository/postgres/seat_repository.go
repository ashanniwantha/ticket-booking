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

type SeatRepo struct {
	pool *pgxpool.Pool
}

func NewSeatRepo(pool *pgxpool.Pool) *SeatRepo {
	return &SeatRepo{pool: pool}
}

func (r *SeatRepo) conn(ctx context.Context) DBTX {
	if tx := TxFromContext(ctx); tx != nil {
		return tx
	}
	return r.pool
}

func (r *SeatRepo) Create(ctx context.Context, s *domain.Seat) error {
	query := `
	INSERT INTO seats (hall_id, seat_number, class)
	VALUES ($1, $2, $3)
	RETURNING id, created_at, updated_at
	`
	row := r.conn(ctx).QueryRow(
		ctx, query, s.HallID, s.SeatNumber, s.Class,
	)

	if err := row.Scan(
		&s.ID, &s.CreatedAt, &s.UpdatedAt,
	); err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				switch pgErr.ConstraintName {
				case "uq_seat_number":
					return domain.ErrDuplicateSeatNumber
				default:
					return domain.ErrDuplicateSeat
				}
			case "23503":
				switch pgErr.ConstraintName {
				case "fk_seats_halls":
					return domain.ErrHallNotFound
				default:
					return domain.ErrSeatForeignKeyViolation
				}
			case "23514":
				return domain.ErrInvalidSeatData
			}
		}
		return fmt.Errorf("inserting seat (repo): %w", err)
	}
	return nil
}

func (r *SeatRepo) GetByID(ctx context.Context, seatID int64) (*domain.Seat, error) {
	var seat domain.Seat

	query := `
		SELECT id, hall_id, seat_number, class, created_at,updated_at
		FROM seats
		WHERE id=$1
	`
	row := r.conn(ctx).QueryRow(ctx, query, seatID)

	if err := row.Scan(
		&seat.ID, &seat.HallID, &seat.SeatNumber, &seat.Class, &seat.CreatedAt, &seat.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSeatNotFound
		}

		return nil, fmt.Errorf("get seat by id (repository): %w", err)
	}

	return &seat, nil
}

func (r *SeatRepo) ListByHallID(ctx context.Context, hallID int64) ([]domain.Seat, error) {
	seatList := make([]domain.Seat, 0)

	query := `
		SELECT id, hall_id, seat_number, class, created_at,updated_at
		FROM seats
		WHERE hall_id=$1
	`
	rows, err := r.conn(ctx).Query(ctx, query, hallID)
	if err != nil {
		return nil, fmt.Errorf("querying seats by hall id (repository): %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var seat domain.Seat
		if err := rows.Scan(
			&seat.ID, &seat.HallID, &seat.SeatNumber, &seat.Class, &seat.CreatedAt, &seat.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning seat row: %w", err)
		}
		seatList = append(seatList, seat)
	}
	// Catch mid stream connection or buffer errors
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating seat rows completed with errors: %w", err)
	}

	return seatList, nil
}

func (r *SeatRepo) ListByClass(ctx context.Context, class domain.SeatClass) ([]domain.Seat, error) {
	seatList := make([]domain.Seat, 0)
	query := `
		SELECT id, hall_id, seat_number, class, created_at,updated_at
		FROM seats
		WHERE class=$1
	`
	rows, err := r.conn(ctx).Query(ctx, query, class)
	if err != nil {
		return nil, fmt.Errorf("querying seats by class (repository): %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var seat domain.Seat
		if err := rows.Scan(
			&seat.ID, &seat.HallID, &seat.SeatNumber, &seat.Class, &seat.CreatedAt, &seat.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning seat error: %w", err)
		}
		seatList = append(seatList, seat)
	}

	// Catch mid stream connection or buffer errors
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating seat rows completed with errors: %w", err)
	}

	return seatList, nil
}

func (r *SeatRepo) Update(ctx context.Context, s *domain.Seat) error {
	query := `
	UPDATE seats
	SET seat_number=$2, class=$3
	WHERE id=$1
	RETURNING id, hall_id, created_at, updated_at
	`
	row := r.conn(ctx).QueryRow(
		ctx, query, s.ID, s.SeatNumber, s.Class,
	)

	if err := row.Scan(&s.ID, &s.HallID, &s.CreatedAt, &s.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrSeatNotFound
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				switch pgErr.ConstraintName {
				case "uq_seat_number":
					return domain.ErrDuplicateSeatNumber
				default:
					return domain.ErrDuplicateSeat
				}
			case "23503":
				switch pgErr.ConstraintName {
				case "fk_seats_halls":
					return domain.ErrHallNotFound
				default:
					return domain.ErrSeatForeignKeyViolation
				}
			case "23514":
				return domain.ErrInvalidSeatData
			}
		}

		return fmt.Errorf("updating seats (repository): %w", err)
	}

	return nil
}

func (r *SeatRepo) Delete(ctx context.Context, seatID int64) error {
	query := `
		DELETE FROM seats
		WHERE id=$1
	`
	commandTag, err := r.conn(ctx).Exec(ctx, query, seatID)
	if err != nil {
		return fmt.Errorf("deleting seat (repository): %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return domain.ErrSeatNotFound
	}

	return nil
}
