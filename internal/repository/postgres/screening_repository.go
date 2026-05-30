package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ashanniwantha/ticket-booking/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ScreeningRepo struct {
	pool *pgxpool.Pool
}

func NewScreeningRepo(pool *pgxpool.Pool) *ScreeningRepo {
	return &ScreeningRepo{pool: pool}
}

func (r *ScreeningRepo) conn(ctx context.Context) DBTX {
	if tx := TxFromContext(ctx); tx != nil {
		return tx
	}
	return r.pool
}

func (r *ScreeningRepo) Create(ctx context.Context, s *domain.Screening) error {
	// Transform domain timestamp into into an immutable pgx tstzrange object
	period := pgtype.Range[time.Time]{
		Lower:     s.StartTime,
		Upper:     s.EndTime,
		LowerType: pgtype.Inclusive,
		UpperType: pgtype.Exclusive, // Mathematical notation representing: [Start, End)
		Valid:     true,
	}

	query := `
	INSERT INTO screenings (movie_id, hall_id, screening_period)
	VALUES ($1, $2, $3)
	RETURNING id, created_at, updated_at
	`
	row := r.conn(ctx).QueryRow(ctx, query, s.MovieID, s.HallID, period)

	if err := row.Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt); err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) {

			switch pgErr.Code {
			case "23P01": // exclusion_violation
				return domain.ErrScreeningTimeConflict

			case "23503": // foreign_key_violation
				switch pgErr.ConstraintName {
				case "fk_screenings_movies":
					return domain.ErrMovieNotFound
				case "fk_screenings_halls":
					return domain.ErrHallNotFound
				default:
					return domain.ErrScreeningForeignKeyViolation
				}

			case "23514":
				if pgErr.ConstraintName == "chk_screening_period_future" {
					return domain.ErrInvalidScreeningData
				}
			}
		}

		return fmt.Errorf("inserting screenings (repo): %w", err)
	}

	return nil
}

func (r *ScreeningRepo) GetByID(ctx context.Context, screeningID int64) (*domain.Screening, error) {
	var screening domain.Screening
	var period pgtype.Range[time.Time]

	query := `
	SELECT id, movie_id, hall_id, screening_period, created_at, updated_at
	FROM screenings
	WHERE id=$1
	`
	row := r.conn(ctx).QueryRow(ctx, query, screeningID)

	if err := row.Scan(
		&screening.ID,
		&screening.MovieID,
		&screening.HallID,
		&period,
		&screening.CreatedAt,
		&screening.UpdatedAt,
	); err != nil {

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrScreeningNotFound
		}
		return nil, fmt.Errorf("getting screening by id (repo): %w", err)
	}

	// Map the range boundaries cleanly back to our pure domain fields
	screening.StartTime = period.Lower
	screening.EndTime = period.Upper

	return &screening, nil
}

func (r *ScreeningRepo) ListAll(ctx context.Context) ([]domain.Screening, error) {
	screeningList := make([]domain.Screening, 0)

	query := `
	SELECT id, movie_id, hall_id, screening_period, created_at, updated_at
	FROM screenings
	`
	rows, err := r.conn(ctx).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying all screens (repo): %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var screening domain.Screening
		var period pgtype.Range[time.Time]

		if err := rows.Scan(
			&screening.ID,
			&screening.MovieID,
			&screening.HallID,
			&period,
			&screening.CreatedAt,
			&screening.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning all screening list (repo): %w", err)
		}
		// Map the range boundaries back to pure domain fields
		screening.StartTime = period.Lower
		screening.EndTime = period.Upper

		screeningList = append(screeningList, screening)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scanning all screening list completed with errors (repo): %w", err)
	}

	return screeningList, nil
}

func (r *ScreeningRepo) ListByMovie(ctx context.Context, movieID int64) ([]domain.Screening, error) {
	screeningList := make([]domain.Screening, 0)

	query := `
	SELECT id, hall_id, screening_period, created_at, updated_at
	FROM screenings
	WHERE movie_id=$1
	`
	rows, err := r.conn(ctx).Query(ctx, query, movieID)
	if err != nil {
		return nil, fmt.Errorf("querying all screening by movie (repo): %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var screening domain.Screening
		var period pgtype.Range[time.Time]

		if err := rows.Scan(
			&screening.ID,
			&screening.HallID,
			&period,
			&screening.CreatedAt,
			&screening.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning screening list (repo): %w", err)
		}
		screening.MovieID = movieID
		// Map the range boundaries back to pure model fields
		screening.StartTime = period.Lower
		screening.EndTime = period.Upper

		screeningList = append(screeningList, screening)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scanning screening list completed with errors: %w", err)
	}

	return screeningList, nil
}

func (r *ScreeningRepo) ListByHall(ctx context.Context, hallID int64) ([]domain.Screening, error) {
	screeningList := make([]domain.Screening, 0)

	query := `
	SELECT id, movie_id, screening_period, created_at, updated_at
	FROM screenings
	WHERE hall_id=$1
	`
	rows, err := r.conn(ctx).Query(ctx, query, hallID)
	if err != nil {
		return nil, fmt.Errorf("querying all screenings by hall (repo): %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var screening domain.Screening
		var period pgtype.Range[time.Time]

		if err := rows.Scan(
			&screening.ID,
			&screening.MovieID,
			&period,
			&screening.CreatedAt,
			&screening.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning screening list (repo): %w", err)
		}
		screening.HallID = hallID
		// Map the range boundaries back to pure domain fields
		screening.StartTime = period.Lower
		screening.EndTime = period.Upper

		screeningList = append(screeningList, screening)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scanning screening list completed with errors (repo): %w", err)
	}

	return screeningList, nil
}

func (r *ScreeningRepo) Update(ctx context.Context, s *domain.Screening) error {
	period := pgtype.Range[time.Time]{
		Lower:     s.StartTime,
		Upper:     s.EndTime,
		LowerType: pgtype.Inclusive,
		UpperType: pgtype.Exclusive,
		Valid:     true,
	}

	query := `
	UPDATE screenings
	SET movie_id=$2, hall_id=$3, screening_period=$4
	WHERE id=$1
	RETURNING created_at, updated_at
	`
	row := r.conn(ctx).QueryRow(ctx, query, s.ID, s.MovieID, s.HallID, period)

	if err := row.Scan(&s.CreatedAt, &s.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrScreeningNotFound
		}
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) {

			switch pgErr.Code {
			case "23P01":
				return domain.ErrScreeningTimeConflict

			case "23503":
				switch pgErr.ConstraintName {
				case "fk_screenings_movies":
					return domain.ErrMovieNotFound
				case "fk_screenings_halls":
					return domain.ErrHallNotFound
				default:
					return domain.ErrScreeningForeignKeyViolation
				}

			case "23514":
				if pgErr.ConstraintName == "chk_screening_period_future" {
					return domain.ErrInvalidScreeningData
				}
			}
		}
		return fmt.Errorf("updating screening (repo): %w", err)
	}

	return nil
}

func (r *ScreeningRepo) Delete(ctx context.Context, screeningID int64) error {
	query := `
	DELETE FROM screenings
	WHERE id=$1
	`
	commandTag, err := r.conn(ctx).Exec(ctx, query, screeningID)
	if err != nil {
		return fmt.Errorf("deleting screening (repo): %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return domain.ErrScreeningNotFound
	}

	return nil
}
