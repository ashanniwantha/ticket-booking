package domain

import (
	"context"
	"errors"
	"time"
)

type User struct {
	ID           int64
	Username     string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

var (
	ErrorDuplicateEmail    = errors.New("duplicate email")
	ErrorDuplicateUsername = errors.New("duplicate username")
	ErrorUserNotFound      = errors.New("user not found")
)

type UserRepository interface {
	Create(ctx context.Context, u *User) error
	Update(ctx context.Context, u *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id int64) (*User, error)
	Delete(ctx context.Context, id int64) error
}
