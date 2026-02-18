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
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/terrascore/api/internal/platform"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	// Load config
	cfg, err := platform.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Logger
	logger := platform.NewLogger(cfg.Server.Env)
	slog.SetDefault(logger)

	// Database
	db, err := platform.NewDBPool(ctx, cfg.DB)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer db.Close()
	logger.Info("connected to database")

	// Redis
	rdb, err := platform.NewRedisClient(ctx, cfg.Redis)
	if err != nil {
		return fmt.Errorf("connecting to redis: %w", err)
	}
	defer rdb.Close()
	logger.Info("connected to redis")

	// Event bus
	eventBus := platform.NewEventBus(logger, 1000)
	go eventBus.Start(ctx)

	// Task queue
	taskQueue := platform.NewTaskQueue(db, logger)
	go taskQueue.Start(ctx)

	// Router
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RealIP)
	r.Use(platform.RequestID)
	r.Use(platform.Logging(logger))
	r.Use(platform.Recovery(logger))
	r.Use(chimw.Compress(5))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		platform.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// API v1 routes
	r.Route("/v1", func(r chi.Router) {
		// Auth routes will be wired in Milestone 5
		// r.Mount("/auth", auth.Routes(...))
	})

	// Suppress unused variable warnings — these are used in Milestone 5
	_ = cfg
	_ = rdb
	_ = eventBus
	_ = taskQueue

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		<-ctx.Done()
		logger.Info("shutting down server...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		srv.Shutdown(shutdownCtx)
	}()

	logger.Info("server starting", "addr", addr, "env", cfg.Server.Env)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
