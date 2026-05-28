package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ashanniwantha/ticket-booking/internal/config"
	"github.com/ashanniwantha/ticket-booking/internal/database"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	dbCfg := database.PoolConfig{
		DSN:               dsn,
		MaxConns:          cfg.DBMaxConns,
		MinConns:          cfg.DBMinConns,
		MaxConnLifetime:   cfg.DBMaxConnLifetime,
		MaxConnIdleTime:   cfg.DBMaxConnIdleTime,
		HealthCheckPeriod: cfg.DBHealthCheckPeriod,
	}

	pool, err := database.NewPool(context.Background(), dbCfg)
	if err != nil {
		log.Fatalf("failed initialize database: %s", err)
	}
	defer pool.Close()

	log.Printf("Database connected - %s mode", cfg.AppEnv)

	log.Printf("Starting application in [%s] mode...", cfg.AppEnv)
	fmt.Printf("Database URL target: postgres://%s:***@%s:%d/%s\n",
		cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)
}
