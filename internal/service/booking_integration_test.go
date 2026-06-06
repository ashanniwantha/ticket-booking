//go:build integration

package service_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/ashanniwantha/ticket-booking/internal/auth"
	"github.com/ashanniwantha/ticket-booking/internal/domain"
	"github.com/ashanniwantha/ticket-booking/internal/repository/postgres"
	"github.com/ashanniwantha/ticket-booking/internal/service"
	"github.com/ashanniwantha/ticket-booking/internal/test"
	"github.com/stretchr/testify/require"
)

func TestConcurrentBooking(t *testing.T) {
	ctx := context.Background()

	// Setup test DB and Redis
	pool, err := test.NewTestPool(ctx)
	require.NoError(t, err, "failed to connect to test DB")
	t.Cleanup(pool.Close)

	rdb, err := test.NewTestRedis(ctx)
	require.NoError(t, err, "failed to connect to test redis")
	t.Cleanup(func() { rdb.Close() })

	// Initates necessary repositories
	userRepo := postgres.NewUserRepo(pool)
	hallRepo := postgres.NewHallRepo(pool)
	movieRepo := postgres.NewMovieRepo(pool)
	screeningRepo := postgres.NewScreeningRepo(pool)
	seatRepo := postgres.NewSeatRepo(pool)
	ticketRepo := postgres.NewTicketRepo(pool)

	// Create necessary services for seeding data
	logger := slog.New(slog.NewTextHandler(io.Discard, nil)) // silent loggers for test
	tokenGen := auth.NewTokenGenerator("super-secret-jwt-token-with-at-least-32-characters-long", 1*time.Minute)
	userSvc := service.NewUserService(userRepo, tokenGen, logger)
	hallSvc := service.NewHallService(hallRepo, rdb, logger)
	movieSvc := service.NewMovieService(movieRepo, rdb, logger)
	screeningSvc := service.NewScreeningService(screeningRepo, logger)
	seatSvc := service.NewSeatService(seatRepo, rdb, logger)

	user, err := userSvc.Register(ctx, service.RegisterRequest{
		Username: "test",
		Email:    "test@test.com",
		Password: "123456",
	})
	require.NoError(t, err, "failed to create a test user")

	hall, err := hallSvc.AddHall(ctx, service.AddHallRequest{Name: "Test Hall"})
	require.NoError(t, err, "failed to create a test hall")

	movie, err := movieSvc.AddMovie(ctx, service.AddMovieRequest{Title: "Test Movie", Description: "Testing"})
	require.NoError(t, err, "failed to create a test movie")

	screeningTime := time.Now().Add(10 * 24 * time.Hour)
	endTime := screeningTime.Add(2 * time.Hour)

	screening, err := screeningSvc.AddScreening(ctx, service.AddScreeningReq{
		MovieID:   movie.ID,
		HallID:    hall.ID,
		StartTime: screeningTime,
		EndTime:   endTime,
	})
	require.NoError(t, err, "failed to create a test screening")

	seat, err := seatSvc.AddSeat(ctx, service.AddSeatRequest{
		HallID:     hall.ID,
		SeatNumber: "A1",
		Class:      domain.SeatClassVIP,
	})
	require.NoError(t, err, "failed to create a test seat")

	// Create booking service
	bookingSvc := service.NewBookingService(ticketRepo, screeningRepo, seatRepo, logger, pool)

	const numRequest = 20
	var wg sync.WaitGroup
	result := make(chan error, numRequest)

	// All go routines try to book the same seat
	// We'll simulate the same authenticated user
	for range numRequest {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := service.HoldSeatRequest{
				ScreeningID: screening.ID,
				SeatID:      seat.ID,
				UserID:      user.ID,
			}

			_, err := bookingSvc.HoldSeat(ctx, req)
			result <- err
		}()
	}
	wg.Wait()
	close(result)

	// Count success and conflicts
	var successCount, conflictCount int
	for err := range result {
		if err == nil {
			successCount++
		} else if errors.Is(err, domain.ErrSeatUnavailable) {
			conflictCount++
		} else {
			// Unexpected error
			t.Errorf("unexpected error during hold: %v", err)
		}
	}
	require.Equal(t, 1, successCount, "exactly one booking should succeed")
	require.Equal(t, numRequest-1, conflictCount, "all other booking should conflict")
}
