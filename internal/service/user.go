package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ashanniwantha/ticket-booking/internal/auth"
	"github.com/ashanniwantha/ticket-booking/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var (
	ErrUsernameOrEmailEmpty = errors.New("username or email is required")
	ErrPasswordEmpty        = errors.New("password is required")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrMissingCredentials   = errors.New("email and password are required")
	ErrInvalidPassword      = errors.New("invalid password")
	ErrPasswordTooLong      = errors.New("password too long")
)

type UserService interface {
	Register(ctx context.Context, req RegisterRequest) (*UserResponse, error)
	Login(ctx context.Context, req LoginRequest) (string, error)
}

type userService struct {
	repo     domain.UserRepository
	tokenGen *auth.TokenGenerator
	logger   *slog.Logger
}

func NewUserService(repo domain.UserRepository, tokenGen *auth.TokenGenerator, logger *slog.Logger) UserService {
	return &userService{
		repo:     repo,
		tokenGen: tokenGen,
		logger:   logger,
	}
}

func (s *userService) Register(ctx context.Context, req RegisterRequest) (*UserResponse, error) {
	username := strings.TrimSpace(req.Username)
	email := strings.TrimSpace(req.Email)

	if username == "" || email == "" {
		return nil, ErrUsernameOrEmailEmpty
	}

	if req.Password == "" {
		return nil, ErrPasswordEmpty
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		if errors.Is(err, bcrypt.ErrPasswordTooLong) {
			return nil, ErrPasswordTooLong
		}

		s.logger.Error("hashing password failed", "err", err)
		return nil, fmt.Errorf("internal error")
	}

	user := &domain.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	res := &UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	s.logger.Info("user registered", "user_id", user.ID)
	return res, nil

}

func (s *userService) Login(ctx context.Context, req LoginRequest) (string, error) {
	email := strings.TrimSpace(req.Email)
	password := strings.TrimSpace(req.Password)

	if email == "" || password == "" {
		return "", ErrMissingCredentials
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", ErrInvalidCredentials
		}

		s.logger.Error("user get by email failed", "err", err)
		return "", fmt.Errorf("internal error")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	token, err := s.tokenGen.GenerateToken(user.ID, user.Username)
	if err != nil {
		s.logger.Error("generating token failed", "err", err)
		return "", fmt.Errorf("internal error")
	}
	return token, nil
}
