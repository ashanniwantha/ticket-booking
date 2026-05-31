package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/ashanniwantha/ticket-booking/internal/domain"
)

type AddScreeningReq struct {
	MovieID   int64     `json:"movie_id"`
	HallID    int64     `json:"hall_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

type UpdateScreeningReq struct {
	MovieID   int64     `json:"movie_id"`
	HallID    int64     `json:"hall_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

type ScreeningResponse struct {
	ID        int64     `json:"id"`
	MovieID   int64     `json:"movie_id"`
	HallID    int64     `json:"hall_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ScreeningService interface {
	AddScreening(ctx context.Context, req AddScreeningReq) (*ScreeningResponse, error)
	GetScreeningByID(ctx context.Context, screeningID int64) (*ScreeningResponse, error)
	ListAllScreenings(ctx context.Context) ([]ScreeningResponse, error)
	ListScreeningsByMovie(ctx context.Context, movieID int64) ([]ScreeningResponse, error)
	ListScreeningsByHall(ctx context.Context, hallID int64) ([]ScreeningResponse, error)
	ListScreeningsByMovieAndHall(ctx context.Context, movieID int64, hallID int64) ([]ScreeningResponse, error)
	UpdateScreening(ctx context.Context, screeningID int64, req UpdateScreeningReq) (*ScreeningResponse, error)
	RemoveScreening(ctx context.Context, screeningID int64) error
}

type screeningService struct {
	repo   domain.ScreeningRepository
	logger *slog.Logger
}

func NewScreeningService(repo domain.ScreeningRepository, logger *slog.Logger) ScreeningService {
	return &screeningService{
		repo:   repo,
		logger: logger,
	}
}

func (s *screeningService) AddScreening(ctx context.Context, req AddScreeningReq) (*ScreeningResponse, error) {
	if req.MovieID <= 0 {
		return nil, ErrInvalidMovieID
	}

	if req.HallID <= 0 {
		return nil, ErrInvalidHallID
	}

	if (req.StartTime.Before(time.Now())) || !(req.EndTime.After(req.StartTime)) {
		return nil, ErrInvalidSchedule
	}

	screening := &domain.Screening{
		MovieID:   req.MovieID,
		HallID:    req.HallID,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
	}

	if err := s.repo.Create(ctx, screening); err != nil {
		return nil, err
	}

	screeningResp := &ScreeningResponse{
		ID:        screening.ID,
		MovieID:   screening.MovieID,
		HallID:    screening.HallID,
		StartTime: screening.StartTime,
		EndTime:   screening.EndTime,
		CreatedAt: screening.CreatedAt,
		UpdatedAt: screening.UpdatedAt,
	}

	s.logger.Info("screening created", "screening_id", screening.ID)
	return screeningResp, nil
}

func (s *screeningService) GetScreeningByID(ctx context.Context, screeningID int64) (*ScreeningResponse, error) {
	if screeningID <= 0 {
		return nil, ErrInvalidScreeningID
	}

	screening, err := s.repo.GetByID(ctx, screeningID)

	if err != nil {
		return nil, err
	}

	screeningResp := &ScreeningResponse{
		ID:        screening.ID,
		MovieID:   screening.MovieID,
		HallID:    screening.HallID,
		StartTime: screening.StartTime,
		EndTime:   screening.EndTime,
		CreatedAt: screening.CreatedAt,
		UpdatedAt: screening.UpdatedAt,
	}

	return screeningResp, nil
}

func (s *screeningService) ListAllScreenings(ctx context.Context) ([]ScreeningResponse, error) {
	screeningList, err := s.repo.ListAll(ctx)

	if err != nil {
		return nil, err
	}

	screeningListResp := make([]ScreeningResponse, 0, len(screeningList))

	for _, screening := range screeningList {
		screeningListResp = append(screeningListResp, ScreeningResponse{
			ID:        screening.ID,
			MovieID:   screening.MovieID,
			HallID:    screening.HallID,
			StartTime: screening.StartTime,
			EndTime:   screening.EndTime,
			CreatedAt: screening.CreatedAt,
			UpdatedAt: screening.UpdatedAt,
		})
	}

	return screeningListResp, nil
}

func (s *screeningService) ListScreeningsByMovie(ctx context.Context, movieID int64) ([]ScreeningResponse, error) {
	if movieID <= 0 {
		return nil, ErrInvalidMovieID
	}

	screeningList, err := s.repo.ListByMovie(ctx, movieID)

	if err != nil {
		return nil, err
	}

	screeningListResp := make([]ScreeningResponse, 0, len(screeningList))

	for _, screening := range screeningList {
		screeningListResp = append(screeningListResp, ScreeningResponse{
			ID:        screening.ID,
			MovieID:   screening.MovieID,
			HallID:    screening.HallID,
			StartTime: screening.StartTime,
			EndTime:   screening.EndTime,
			CreatedAt: screening.CreatedAt,
			UpdatedAt: screening.UpdatedAt,
		})
	}

	return screeningListResp, nil
}

func (s *screeningService) ListScreeningsByHall(ctx context.Context, hallID int64) ([]ScreeningResponse, error) {
	if hallID <= 0 {
		return nil, ErrInvalidHallID
	}

	screeningList, err := s.repo.ListByHall(ctx, hallID)

	if err != nil {
		return nil, err
	}

	screeningListResp := make([]ScreeningResponse, 0, len(screeningList))

	for _, screening := range screeningList {
		screeningListResp = append(screeningListResp, ScreeningResponse{
			ID:        screening.ID,
			MovieID:   screening.MovieID,
			HallID:    screening.HallID,
			StartTime: screening.StartTime,
			EndTime:   screening.EndTime,
			CreatedAt: screening.CreatedAt,
			UpdatedAt: screening.UpdatedAt,
		})
	}

	return screeningListResp, nil
}

func (s *screeningService) ListScreeningsByMovieAndHall(ctx context.Context, movieID int64, hallID int64) ([]ScreeningResponse, error) {
	if movieID <= 0 {
		return nil, ErrInvalidMovieID
	}

	if hallID <= 0 {
		return nil, ErrInvalidHallID
	}

	screeningList, err := s.repo.ListByMovieAndHall(ctx, movieID, hallID)
	if err != nil {
		return nil, err
	}

	screeningListResp := make([]ScreeningResponse, 0, len(screeningList))

	for _, screening := range screeningList {
		screeningListResp = append(screeningListResp, ScreeningResponse{
			ID:        screening.ID,
			MovieID:   screening.MovieID,
			HallID:    screening.HallID,
			StartTime: screening.StartTime,
			EndTime:   screening.EndTime,
			CreatedAt: screening.CreatedAt,
			UpdatedAt: screening.UpdatedAt,
		})
	}
	return screeningListResp, nil
}

func (s *screeningService) UpdateScreening(ctx context.Context, screeningID int64, req UpdateScreeningReq) (*ScreeningResponse, error) {
	if screeningID <= 0 {
		return nil, ErrInvalidScreeningID
	}

	if req.MovieID <= 0 {
		return nil, ErrInvalidMovieID
	}

	if req.HallID <= 0 {
		return nil, ErrInvalidHallID
	}

	if (req.StartTime.Before(time.Now())) || !(req.EndTime.After(req.StartTime)) {
		return nil, ErrInvalidSchedule
	}

	screening := &domain.Screening{
		ID:        screeningID,
		MovieID:   req.MovieID,
		HallID:    req.HallID,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
	}

	if err := s.repo.Update(ctx, screening); err != nil {
		return nil, err
	}

	screeningResp := &ScreeningResponse{
		ID:        screening.ID,
		MovieID:   screening.MovieID,
		HallID:    screening.HallID,
		StartTime: screening.StartTime,
		EndTime:   screening.EndTime,
		CreatedAt: screening.CreatedAt,
		UpdatedAt: screening.UpdatedAt,
	}

	s.logger.Info("screening updated", "screening_id", screeningID)
	return screeningResp, nil
}

func (s *screeningService) RemoveScreening(ctx context.Context, screeningID int64) error {
	if screeningID <= 0 {
		return ErrInvalidScreeningID
	}

	if err := s.repo.Delete(ctx, screeningID); err != nil {
		return err
	}

	s.logger.Info("screening removed", "screening_id", screeningID)
	return nil
}
