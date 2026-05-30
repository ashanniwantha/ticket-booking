package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/ashanniwantha/ticket-booking/internal/domain"
)

type AddSeatRequest struct {
	HallID     int64            `json:"hall_id"`
	SeatNumber string           `json:"seat_number"`
	Class      domain.SeatClass `json:"class"`
}

type UpdateSeatRequest struct {
	SeatNumber string           `json:"seat_number"`
	Class      domain.SeatClass `json:"class"`
}

type SeatResponse struct {
	ID         int64            `json:"id"`
	HallID     int64            `json:"hall_id"`
	SeatNumber string           `json:"seat_number"`
	Class      domain.SeatClass `json:"class"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

var (
	ErrInvalidSeatID    = errors.New("invalid seat ID")
	ErrEmptySeatNumber  = errors.New("seat number is required")
	ErrInvalidSeatClass = errors.New("invalid seat class")
)

type SeatService interface {
	AddSeat(ctx context.Context, req AddSeatRequest) (*SeatResponse, error)
	GetSeatByID(ctx context.Context, seatID int64) (*SeatResponse, error)
	ListAllSeats(ctx context.Context) ([]SeatResponse, error)
	ListSeatsByHallID(ctx context.Context, hallID int64) ([]SeatResponse, error)
	ListSeatsByClass(ctx context.Context, class domain.SeatClass) ([]SeatResponse, error)
	UpdateSeat(ctx context.Context, seatID int64, req UpdateSeatRequest) (*SeatResponse, error)
	RemoveSeat(ctx context.Context, seatID int64) error
}

type seatService struct {
	repo   domain.SeatRepository
	logger *slog.Logger
}

func NewSeatService(repo domain.SeatRepository, logger *slog.Logger) SeatService {
	return &seatService{
		repo:   repo,
		logger: logger,
	}
}

func (s *seatService) AddSeat(ctx context.Context, req AddSeatRequest) (*SeatResponse, error) {
	if req.HallID <= 0 {
		return nil, ErrInvalidHallID
	}

	if req.Class != domain.SeatClassVIP && req.Class != domain.SeatClassBalcony && req.Class != domain.SeatClassRegular {
		return nil, ErrInvalidSeatClass
	}

	seatNumber := req.SeatNumber
	if seatNumber == "" {
		return nil, ErrEmptySeatNumber
	}

	seat := &domain.Seat{
		HallID:     req.HallID,
		SeatNumber: seatNumber,
		Class:      req.Class,
	}

	if err := s.repo.Create(ctx, seat); err != nil {
		return nil, err
	}

	resp := &SeatResponse{
		ID:         seat.ID,
		HallID:     seat.HallID,
		SeatNumber: seat.SeatNumber,
		Class:      seat.Class,
		CreatedAt:  seat.CreatedAt,
		UpdatedAt:  seat.UpdatedAt,
	}

	s.logger.Info("seat created", "seat_number", seatNumber)
	return resp, nil
}

func (s *seatService) GetSeatByID(ctx context.Context, seatID int64) (*SeatResponse, error) {
	if seatID <= 0 {
		return nil, ErrInvalidSeatID
	}

	seat, err := s.repo.GetByID(ctx, seatID)
	if err != nil {
		return nil, err
	}

	seatResp := &SeatResponse{
		ID:         seat.ID,
		HallID:     seat.HallID,
		SeatNumber: seat.SeatNumber,
		Class:      seat.Class,
		CreatedAt:  seat.CreatedAt,
		UpdatedAt:  seat.UpdatedAt,
	}

	return seatResp, nil
}

func (s *seatService) ListAllSeats(ctx context.Context) ([]SeatResponse, error) {
	seatsList, err := s.repo.ListAll(ctx)

	if err != nil {
		return nil, err
	}

	seatListResp := make([]SeatResponse, 0, len(seatsList))

	for _, seat := range seatsList {
		seatListResp = append(seatListResp, SeatResponse{
			ID:         seat.ID,
			HallID:     seat.HallID,
			SeatNumber: seat.SeatNumber,
			Class:      seat.Class,
			CreatedAt:  seat.CreatedAt,
			UpdatedAt:  seat.UpdatedAt,
		})
	}

	return seatListResp, nil
}

func (s *seatService) ListSeatsByHallID(ctx context.Context, hallID int64) ([]SeatResponse, error) {
	if hallID <= 0 {
		return nil, ErrInvalidHallID
	}

	seatList, err := s.repo.ListByHallID(ctx, hallID)
	if err != nil {
		return nil, err
	}

	seatListResp := make([]SeatResponse, 0, len(seatList))

	for _, seat := range seatList {
		seatListResp = append(seatListResp, SeatResponse{
			ID:         seat.ID,
			HallID:     seat.HallID,
			SeatNumber: seat.SeatNumber,
			Class:      seat.Class,
			CreatedAt:  seat.CreatedAt,
			UpdatedAt:  seat.UpdatedAt,
		})
	}

	return seatListResp, nil
}

func (s *seatService) ListSeatsByClass(ctx context.Context, class domain.SeatClass) ([]SeatResponse, error) {
	if class != domain.SeatClassVIP && class != domain.SeatClassBalcony && class != domain.SeatClassRegular {
		return nil, ErrInvalidSeatClass
	}

	seatList, err := s.repo.ListByClass(ctx, class)

	if err != nil {
		return nil, err
	}

	seatListResp := make([]SeatResponse, 0, len(seatList))
	for _, seat := range seatList {
		seatListResp = append(seatListResp, SeatResponse{
			ID:         seat.ID,
			HallID:     seat.HallID,
			SeatNumber: seat.SeatNumber,
			Class:      seat.Class,
			CreatedAt:  seat.CreatedAt,
			UpdatedAt:  seat.UpdatedAt,
		})
	}

	return seatListResp, nil
}

func (s *seatService) UpdateSeat(ctx context.Context, seatID int64, req UpdateSeatRequest) (*SeatResponse, error) {
	seatNumber := req.SeatNumber

	if seatNumber == "" {
		return nil, ErrEmptySeatNumber
	}

	if req.Class != domain.SeatClassVIP && req.Class != domain.SeatClassBalcony && req.Class != domain.SeatClassRegular {
		return nil, ErrInvalidSeatClass
	}

	seat := &domain.Seat{
		ID:         seatID,
		SeatNumber: seatNumber,
		Class:      req.Class,
	}

	if err := s.repo.Update(ctx, seat); err != nil {
		return nil, err
	}

	seatResp := &SeatResponse{
		ID:         seat.ID,
		HallID:     seat.HallID,
		SeatNumber: seat.SeatNumber,
		Class:      seat.Class,
		CreatedAt:  seat.CreatedAt,
		UpdatedAt:  seat.UpdatedAt,
	}

	s.logger.Info("seat updated", "seat_number", seat.SeatNumber)
	return seatResp, nil
}

func (s *seatService) RemoveSeat(ctx context.Context, seatID int64) error {
	if seatID <= 0 {
		return ErrInvalidSeatID
	}

	if err := s.repo.Delete(ctx, seatID); err != nil {
		return err
	}

	s.logger.Info("seat deleted", "seat_id", seatID)
	return nil
}
