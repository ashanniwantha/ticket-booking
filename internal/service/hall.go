package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/ashanniwantha/ticket-booking/internal/domain"
)

type AddHallRequest struct {
	Name string `json:"name"`
}

type UpdateHallRequest struct {
	Name string `json:"name"`
}

type HallResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var (
	ErrInvalidHallID = errors.New("invalid hall id")
	ErrEmptyHallName = errors.New("hall is required")
)

type HallService interface {
	AddHall(ctx context.Context, req AddHallRequest) (*HallResponse, error)
	GetHall(ctx context.Context, hallID int64) (*HallResponse, error)
	UpdateHall(ctx context.Context, hallID int64, req UpdateHallRequest) (*HallResponse, error)
	RemoveHall(ctx context.Context, hallID int64) error
}

type hallService struct {
	repo   domain.HallRepository
	logger *slog.Logger
}

func NewHallService(repo domain.HallRepository, logger *slog.Logger) HallService {
	return &hallService{
		repo:   repo,
		logger: logger,
	}
}

func (s *hallService) AddHall(ctx context.Context, req AddHallRequest) (*HallResponse, error) {
	name := strings.TrimSpace(req.Name)

	if name == "" {
		return nil, ErrEmptyHallName
	}

	hall := &domain.Hall{
		Name: name,
	}

	if err := s.repo.Create(ctx, hall); err != nil {
		return nil, err
	}

	resp := &HallResponse{
		ID:        hall.ID,
		Name:      hall.Name,
		CreatedAt: hall.CreatedAt,
		UpdatedAt: hall.UpdatedAt,
	}

	s.logger.Info("hall created", "hall_name", hall.Name)
	return resp, nil
}

func (s *hallService) GetHall(ctx context.Context, hallID int64) (*HallResponse, error) {
	if hallID <= 0 {
		return nil, ErrInvalidHallID
	}

	hall, err := s.repo.GetByID(ctx, hallID)
	if err != nil {
		return nil, err
	}

	resp := &HallResponse{
		ID:        hall.ID,
		Name:      hall.Name,
		CreatedAt: hall.CreatedAt,
		UpdatedAt: hall.UpdatedAt,
	}

	return resp, nil
}

func (s *hallService) UpdateHall(ctx context.Context, hallID int64, req UpdateHallRequest) (*HallResponse, error) {
	if hallID <= 0 {
		return nil, ErrInvalidHallID
	}

	name := strings.TrimSpace(req.Name)

	if name == "" {
		return nil, ErrEmptyHallName
	}

	hall := &domain.Hall{
		ID:   hallID,
		Name: name,
	}

	if err := s.repo.Update(ctx, hall); err != nil {
		return nil, err
	}

	resp := &HallResponse{
		ID:        hall.ID,
		Name:      hall.Name,
		CreatedAt: hall.CreatedAt,
		UpdatedAt: hall.UpdatedAt,
	}

	s.logger.Info("hall updated", "hall_name", name)
	return resp, nil
}

func (s *hallService) RemoveHall(ctx context.Context, hallID int64) error {
	if hallID <= 0 {
		return ErrInvalidHallID
	}

	if err := s.repo.Delete(ctx, hallID); err != nil {
		return err
	}

	s.logger.Info("hall deleted", "hall_id", hallID)
	return nil
}
