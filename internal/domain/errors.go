package domain

import "errors"

var (
	// ---------- Hall ----------
	ErrHallNotFound      = errors.New("hall not found")
	ErrDuplicateHallName = errors.New("hall name already exists")

	// ---------- Movie ----------
	ErrMovieNotFound   = errors.New("movie not found")
	ErrMovieTitleEmpty = errors.New("movie title is required")

	// ---------- Screening ----------
	ErrScreeningNotFound            = errors.New("screening not found")
	ErrScreeningTimeConflict        = errors.New("screening time conflicts with an existing screening")
	ErrScreeningForeignKeyViolation = errors.New("screening foreign key violation")
	ErrInvalidScreeningData         = errors.New("invalid screening data")

	// ---------- Seat ----------
	ErrSeatNotFound            = errors.New("seat not found")
	ErrDuplicateSeatNumber     = errors.New("seat number already exists")
	ErrDuplicateSeat           = errors.New("seat already exists") // generic duplicate fallback
	ErrSeatForeignKeyViolation = errors.New("seat foreign key violation")
	ErrInvalidSeatData         = errors.New("invalid seat data")

	// ---------- User ----------
	ErrUserNotFound      = errors.New("user not found")
	ErrDuplicateEmail    = errors.New("duplicate email")
	ErrDuplicateUsername = errors.New("duplicate username")
)
