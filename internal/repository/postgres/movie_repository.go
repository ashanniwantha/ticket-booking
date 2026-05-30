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

type MovieRepo struct {
	pool *pgxpool.Pool
}

func NewMovieRepo(pool *pgxpool.Pool) *MovieRepo {
	return &MovieRepo{pool: pool}
}

func (r *MovieRepo) conn(ctx context.Context) DBTX {
	if tx := TxFromContext(ctx); tx != nil {
		return tx
	}
	return r.pool
}

func (r *MovieRepo) Create(ctx context.Context, m *domain.Movie) error {
	query := `
	INSERT INTO movies (title, description)
	VALUES ($1, $2)
	RETURNING id, created_at, updated_at
	`
	row := r.conn(ctx).QueryRow(ctx, query, m.Title, m.Description)

	if err := row.Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt); err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) {
			if pgErr.Code == "23514" {
				switch pgErr.ConstraintName {
				case "chk_movies_title_not_empty":
					return domain.ErrMovieTitleEmpty
				}
			}
		}

		return fmt.Errorf("inserting movie (repo): %w", err)
	}

	return nil
}

func (r *MovieRepo) GetByID(ctx context.Context, movieID int64) (*domain.Movie, error) {
	var movie domain.Movie

	query := `
	SELECT id, title, description, created_at, updated_at
	FROM movies
	WHERE id=$1
	`
	row := r.conn(ctx).QueryRow(ctx, query, movieID)

	if err := row.Scan(&movie.ID, &movie.Title, &movie.Description, &movie.CreatedAt, &movie.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMovieNotFound
		}

		return nil, fmt.Errorf("getting movie by id (repo) :%w", err)
	}

	return &movie, nil
}

func (r *MovieRepo) GetByTitle(ctx context.Context, movieTitle string) (*domain.Movie, error) {
	var movie domain.Movie

	query := `
	SELECT id, title, description, created_at, updated_at
	FROM movies
	WHERE title=$1
	`
	row := r.conn(ctx).QueryRow(ctx, query, movieTitle)

	if err := row.Scan(&movie.ID, &movie.Title, &movie.Description, &movie.CreatedAt, &movie.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMovieNotFound
		}

		return nil, fmt.Errorf("getting movie by title: %w", err)
	}

	return &movie, nil
}

func (r *MovieRepo) Update(ctx context.Context, m *domain.Movie) error {
	query := `
	UPDATE movies
	SET title=$2, description=$3
	WHERE id=$1
	RETURNING created_at, updated_at
	`
	row := r.conn(ctx).QueryRow(ctx, query, m.ID, m.Title, m.Description)

	if err := row.Scan(&m.CreatedAt, &m.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrMovieNotFound
		}

		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) {
			if pgErr.Code == "23514" {
				switch pgErr.ConstraintName {
				case "chk_movies_title_not_empty":
					return domain.ErrMovieTitleEmpty
				}
			}
		}
		return fmt.Errorf("updating movie (repo): %w", err)
	}

	return nil
}

func (r *MovieRepo) Delete(ctx context.Context, movieID int64) error {
	query := `
	DELETE FROM movies
	WHERE id=$1
	`
	commandTag, err := r.conn(ctx).Exec(ctx, query, movieID)

	if err != nil {
		return fmt.Errorf("deleting movie (repo): %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return domain.ErrMovieNotFound
	}

	return nil
}
