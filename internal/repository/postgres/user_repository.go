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

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

// conn returns a DBTX (transaction if present in ctx, otherwise pool)
func (r *UserRepo) conn(ctx context.Context) DBTX {
	if tx := TxFromContext(ctx); tx != nil {
		return tx
	}
	return r.pool
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	query := `
	INSERT INTO users (username, email, hashed_password)
	VALUES ($1, $2, $3)
	RETURNING id, created_at, updated_at
	`
	row := r.conn(ctx).QueryRow(ctx, query,
		u.Username, u.Email, u.PasswordHash,
	)

	err := row.Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				switch pgErr.ConstraintName {
				case "uq_users_email":
					return domain.ErrDuplicateEmail
				case "uq_users_username":
					return domain.ErrDuplicateUsername
				}
			}
		}

		return fmt.Errorf("inserting user: %w", err)
	}

	return nil
}

func (r *UserRepo) Update(ctx context.Context, u *domain.User) error {
	query := `
    UPDATE users 
    SET username = $2, email = $3, hashed_password = $4
    WHERE id = $1
    RETURNING updated_at
    `
	row := r.conn(ctx).QueryRow(ctx, query,
		u.ID, u.Username, u.Email, u.PasswordHash,
	)

	if err := row.Scan(&u.UpdatedAt); err != nil {
		// Check for empty results first (driver-level error)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrUserNotFound
		}

		// Check for Postgres server-side constraint errors
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				switch pgErr.ConstraintName {
				case "uq_users_email":
					return domain.ErrDuplicateEmail
				case "uq_users_username":
					return domain.ErrDuplicateUsername
				}
			}
		}

		return fmt.Errorf("updating user: %w", err)
	}

	return nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User

	query := `
	SELECT id, username, email, hashed_password, created_at, updated_at
	FROM users
	WHERE email=$1
	`
	row := r.conn(ctx).QueryRow(ctx, query, email)

	if err := row.Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt,
	); err != nil {

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}

		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &user, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	var user domain.User

	query := `
	SELECT id, username, email, hashed_password, created_at, updated_at
	FROM users
	WHERE id=$1
	`
	row := r.conn(ctx).QueryRow(ctx, query, id)

	if err := row.Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt,
	); err != nil {

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}

		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return &user, nil
}

func (r *UserRepo) Delete(ctx context.Context, id int64) error {
	query := `
	DELETE FROM users
	WHERE id=$1
	`
	commandTag, err := r.conn(ctx).Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}
