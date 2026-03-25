package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/alexs/subscription-service/docs"
	"github.com/alexs/subscription-service/internal/config"
	"github.com/alexs/subscription-service/internal/handler"
	"github.com/alexs/subscription-service/internal/repository"
	"github.com/alexs/subscription-service/internal/service"
)

// @title           Subscription Service API
// @version         1.0
// @description     REST API для агрегации данных об онлайн-подписках пользователей

// @host      localhost:8080
// @BasePath  /api/v1
func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger := setupLogger(cfg.Log.Level)
	logger.Info("starting subscription service")

	// Run database migrations
	if err := runMigrations(cfg.Database.DSN(), logger); err != nil {
		logger.Error("failed to run migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Connect to database
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.Database.DSN())
	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Error("failed to ping database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("connected to database")

	// Wire up dependencies
	repo := repository.NewSubscriptionRepository(pool)
	svc := service.NewSubscriptionService(repo, logger)
	h := handler.NewSubscriptionHandler(svc, logger)

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(handler.LoggingMiddleware(logger))

	h.RegisterRoutes(r)

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("server listening", slog.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Info("shutting down server", slog.String("signal", sig.String()))

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("server stopped gracefully")
}

func setupLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

func runMigrations(dsn string, logger *slog.Logger) error {
	logger.Info("running database migrations")

	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		return fmt.Errorf("creating migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("applying migrations: %w", err)
	}

	logger.Info("migrations applied successfully")
	return nil
}
