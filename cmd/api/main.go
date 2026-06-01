package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ashanniwantha/ticket-booking/internal/auth"
	"github.com/ashanniwantha/ticket-booking/internal/config"
	"github.com/ashanniwantha/ticket-booking/internal/database"
	"github.com/ashanniwantha/ticket-booking/internal/handler"
	"github.com/ashanniwantha/ticket-booking/internal/logger"
	"github.com/ashanniwantha/ticket-booking/internal/redis"
	"github.com/ashanniwantha/ticket-booking/internal/repository/postgres"
	"github.com/ashanniwantha/ticket-booking/internal/service"
	"github.com/go-chi/chi/v5"
)

func main() {
	// load env config
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.AppEnv)
	log.Info("starting application", "env", cfg.AppEnv, "port", cfg.AppPort)
	fmt.Printf("Database URL target: postgres://%s:***@%s:%d/%s\n",
		cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)

	// Database pool
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	dbCfg := database.PoolConfig{
		DSN:               dsn,
		MaxConns:          cfg.DBMaxConns,
		MinConns:          cfg.DBMinConns,
		MaxConnLifetime:   cfg.DBMaxConnLifetime,
		MaxConnIdleTime:   cfg.DBMaxConnIdleTime,
		HealthCheckPeriod: cfg.DBHealthCheckPeriod,
		ConnectTimeout:    cfg.DBConnectTimeout,
		PingTimeout:       cfg.DBPingTimeout,
	}

	pool, err := database.NewPool(context.Background(), dbCfg)
	if err != nil {
		log.Error("failed to initialize database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	log.Info("Database connected", "mode", cfg.AppEnv)

	redisCfg := redis.RedisConfig{
		RedisHost:     cfg.RedisHost,
		RedisPort:     cfg.RedisPort,
		RedisPassword: cfg.RedisPassword,
		DB:            cfg.RedisDB,
		PingTimeout:   cfg.RedisPingTimeout,
	}

	rdb, err := redis.NewClient(context.Background(), redisCfg)
	if err != nil {
		log.Error("failed to initialize redi", "err", err)
		os.Exit(1)
	}
	defer rdb.Close()

	//Initialize the shared BaseHandler foundation once at the top
	baseHandler := handler.NewBaseHandler(log)

	// --- User components ---
	tokenGen := auth.NewTokenGenerator(cfg.JWTSecret, cfg.JWTExpiration)
	userRepo := postgres.NewUserRepo(pool)
	userSvc := service.NewUserService(userRepo, tokenGen, log)
	userHandler := handler.NewAuthHandler(userSvc, log)

	// -- Hall components --
	hallRepo := postgres.NewHallRepo(pool)
	hallSvc := service.NewHallService(hallRepo, log)
	hallHandler := handler.NewHallHandler(hallSvc, log)

	// -- Seat components --
	seatRepo := postgres.NewSeatRepo(pool)
	seatSvc := service.NewSeatService(seatRepo, log)
	seatHandler := handler.NewSeatHandler(seatSvc, log)

	// -- Movie components --
	movieRepo := postgres.NewMovieRepo(pool)
	movieSvc := service.NewMovieService(movieRepo, rdb, log)
	movieHandler := handler.NewMovieHandler(baseHandler, movieSvc)

	// -- Screening components --
	screeningRepo := postgres.NewScreeningRepo(pool)
	screeningSvc := service.NewScreeningService(screeningRepo, log)
	screeningHandler := handler.NewScreeningHandler(screeningSvc, log)

	// -- Ticket components --
	ticketRepo := postgres.NewTicketRepo(pool)

	// -- Booking components --
	bookingSvc := service.NewBookingService(
		ticketRepo,
		screeningRepo,
		seatRepo,
		log,
		pool,
	)
	bookingHandler := handler.NewBookingHandler(bookingSvc, log)

	//  Health Handler
	healthH := handler.NewHealthHandler(pool, log)

	// Chi Router
	r := chi.NewRouter()

	r.Get("/health", healthH.Ping())

	r.Route("/api/v1", func(r chi.Router) {
		// Publlic authentication
		r.Post("/register", userHandler.Register())
		r.Post("/login", userHandler.Login())

		// Protected admin routes (any authenticated user)
		r.Group(func(r chi.Router) {
			r.Use(auth.Authenticate(tokenGen))

			r.Route("/halls", func(r chi.Router) {
				r.Post("/", hallHandler.AddHall())
				r.Get("/{hall_id}", hallHandler.GetHall())
				r.Get("/{hall_id}/screenings", screeningHandler.ListScreeningsByHall())
				r.Patch("/{hall_id}", hallHandler.UpdateHall())
				r.Delete("/{hall_id}", hallHandler.RemoveHall())
				r.Get("/{hall_id}/seats", seatHandler.ListSeatsByHallID())
			})

			r.Route("/movies", func(r chi.Router) {
				r.Post("/", movieHandler.AddMovie())
				r.Get("/{movie_id}", movieHandler.GetMovieByID())
				r.Get("/{movie_id}/screenings", screeningHandler.ListScreeningsByMovie())
				r.Get("/", movieHandler.ListMovies())
				r.Patch("/{movie_id}", movieHandler.UpdateMovie())
				r.Delete("/{movie_id}", movieHandler.RemoveMovie())
			})

			r.Route("/seats", func(r chi.Router) {
				r.Post("/", seatHandler.AddSeat())
				r.Get("/", seatHandler.ListSeats()) // inside handler check query class
				r.Get("/{seat_id}", seatHandler.GetSeatByID())
				r.Patch("/{seat_id}", seatHandler.UpdateSeat())
				r.Delete("/{seat_id}", seatHandler.RemoveSeat())
			})

			r.Route("/screenings", func(r chi.Router) {
				r.Post("/", screeningHandler.AddScreening())
				r.Get("/{screening_id}", screeningHandler.GetScreeningByID())
				r.Get("/", screeningHandler.ListAllScreenings())
				r.Patch("/{screening_id}", screeningHandler.UpdateScreening())
				r.Delete("/{screening_id}", screeningHandler.RemoveScreening())
			})

			r.Route("/bookings", func(r chi.Router) {
				r.Post("/", bookingHandler.HoldSeat())
				r.Patch("/{ticket_id}/confirm", bookingHandler.ConfirmHold())
				r.Patch("/{ticket_id}/cancel", bookingHandler.CancelTicket())
			})
		})
	})

	// HTTP server with timeouts
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.AppPort),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		log.Info("server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("shutting down", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("forced shutdown", "err", err)
	}

	log.Info("server stopped")
}
