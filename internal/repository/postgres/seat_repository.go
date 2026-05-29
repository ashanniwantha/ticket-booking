package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/ashanniwantha/ticket-booking/internal/domain"
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
