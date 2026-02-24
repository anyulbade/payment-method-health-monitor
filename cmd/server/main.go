package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/anyulbade/payment-method-health-monitor/internal/config"
	"github.com/anyulbade/payment-method-health-monitor/internal/database"
	"github.com/anyulbade/payment-method-health-monitor/internal/handler"
	"github.com/anyulbade/payment-method-health-monitor/internal/middleware"
	"github.com/anyulbade/payment-method-health-monitor/internal/repository"
	"github.com/anyulbade/payment-method-health-monitor/internal/service"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()

	cfg := config.Load()
	gin.SetMode(cfg.GinMode)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer pool.Close()

	if cfg.AutoMigrate {
		if err := database.RunMigrations(cfg.DatabaseURL()); err != nil {
			log.Fatal().Err(err).Msg("failed to run migrations")
		}
		if err := database.SeedData(context.Background(), pool); err != nil {
			log.Fatal().Err(err).Msg("failed to seed data")
		}
	}

	router := gin.New()
	router.Use(middleware.Logger())
	router.Use(middleware.ErrorHandler())
	router.Use(gin.Recovery())

	healthHandler := handler.NewHealthHandler(pool)
	router.GET("/health", healthHandler.Health)

	setupAPIRoutes(router, pool)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().Str("port", cfg.Port).Msg("starting server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down server")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("server forced to shutdown")
	}

	log.Info().Msg("server exited")
}

func setupAPIRoutes(router *gin.Engine, pool *pgxpool.Pool) {
	txnRepo := repository.NewTransactionRepository(pool)
	pmRepo := repository.NewPaymentMethodRepository(pool)
	metricsRepo := repository.NewMetricsRepository(pool)

	txnService := service.NewTransactionService(txnRepo, pmRepo)
	metricsService := service.NewMetricsService(metricsRepo)

	txnHandler := handler.NewTransactionHandler(txnService)
	metricsHandler := handler.NewMetricsHandler(metricsService)

	api := router.Group("/api/v1")
	{
		api.POST("/transactions", txnHandler.Create)
		api.POST("/transactions/batch", txnHandler.CreateBatch)
		api.GET("/metrics", metricsHandler.GetMetrics)
	}
}
