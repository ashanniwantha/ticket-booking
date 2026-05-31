package service

import "errors"

// ---------- Hall ----------
var (
	ErrInvalidHallID = errors.New("invalid hall ID")
	ErrEmptyHallName = errors.New("hall name is required")
)

// ---------- Movie ----------
var (
	ErrInvalidMovieID = errors.New("invalid movie ID")
)

// ---------- Screening ----------
var (
	ErrInvalidScreeningID = errors.New("invalid screening ID")
	ErrInvalidSchedule    = errors.New("start time must be in the future and before end time")
)

// ---------- Seat ----------
var (
	ErrInvalidSeatID    = errors.New("invalid seat ID")
	ErrEmptySeatNumber  = errors.New("seat number is required")
	ErrInvalidSeatClass = errors.New("invalid seat class")
)

// ---------- Authentication / User ----------
var (
	ErrUsernameOrEmailEmpty = errors.New("username or email is required")
	ErrPasswordEmpty        = errors.New("password is required")
	ErrPasswordTooLong      = errors.New("password too long")
	ErrInvalidUserID        = errors.New("invalid user ID")
	ErrMissingCredentials   = errors.New("email and password are required")
	ErrInvalidCredentials   = errors.New("invalid credentials")
)

// ---------- Ticket Booking ----------
var (
	ErrInvalidTicketID           = errors.New("invalid ticket ID")
	ErrTicketNotHold             = errors.New("ticket must be in 'hold' status to confirm")
	ErrTicketAlreadyCancelled    = errors.New("ticket is already cancelled")
	ErrSeatScreeningHallMismatch = errors.New("seat does not belong to the screening's hall")
)
