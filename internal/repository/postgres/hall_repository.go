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

type HallRepo struct {
	pool *pgxpool.Pool
}

func NewHallRepo(pool *pgxpool.Pool) *HallRepo {
	return &HallRepo{pool: pool}
}

func (r *HallRepo) conn(ctx context.Context) DBTX {
	if tx := TxFromContext(ctx); tx != nil {
		return tx
	}
	return r.pool
}

func (r *HallRepo) Create(ctx context.Context, h *domain.Hall) error {
	query := `
	INSERT INTO halls (name)
	VALUES ($1)
	RETURNING id, name, created_at, updated_at
	`
	row := r.conn(ctx).QueryRow(ctx, query, h.Name)
	if err := row.Scan(&h.ID, &h.Name, &h.CreatedAt, &h.UpdatedAt); err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				switch pgErr.ConstraintName {
				case "uq_halls_name":
					return domain.ErrDuplicateHallName
				}
			}
		}

		return fmt.Errorf("inserting hall: %w", err)
	}

	return nil
}

func (r *HallRepo) Update(ctx context.Context, h *domain.Hall) error {
	query := `
	UPDATE halls
	SET name=$2
	WHERE id=$1
	RETURNING id, name, created_at, updated_at
	`
	row := r.conn(ctx).QueryRow(ctx, query, h.ID, h.Name)

	if err := row.Scan(&h.ID, h.Name, h.CreatedAt, h.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrHallNotFound
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				switch pgErr.ConstraintName {
				case "uq_halls_name":
					return domain.ErrDuplicateHallName
				}
			}
		}

		return fmt.Errorf("updating hall: %w", err)
	}
	return nil
}

func (r *HallRepo) GetByID(ctx context.Context, hallID int64) (*domain.Hall, error) {
	var hall domain.Hall

	query := `
	SELECT id, name, created_at, updated_at
	WHERE id=$1
	`
	row := r.conn(ctx).QueryRow(ctx, query, hallID)

	if err := row.Scan(&hall.ID, &hall.Name, &hall.CreatedAt, &hall.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrHallNotFound
		}

		return nil, fmt.Errorf("get hall by id: %w", err)
	}

	return &hall, nil
}

func (r *HallRepo) Delete(ctx context.Context, halldID int64) error {
	query := `
	DELETE FROM halls
	WHERE id=$1
	`
	commandTag, err := r.conn(ctx).Exec(ctx, query, halldID)
	if err != nil {
		return fmt.Errorf("deleting hall: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return domain.ErrHallNotFound
	}
	return nil
}
