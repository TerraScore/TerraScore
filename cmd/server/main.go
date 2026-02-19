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
	"github.com/terrascore/api/db/sqlc"
	"github.com/terrascore/api/internal/agent"
	"github.com/terrascore/api/internal/auth"
	"github.com/terrascore/api/internal/job"
	"github.com/terrascore/api/internal/land"
	"github.com/terrascore/api/internal/notification"
	"github.com/terrascore/api/internal/platform"
	"github.com/terrascore/api/internal/qa"
	"github.com/terrascore/api/internal/report"
	"github.com/terrascore/api/internal/survey"
	"github.com/terrascore/api/internal/ws"
)

// Version is set at build time via -ldflags "-X main.Version=..."
var Version = "dev"

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

	// S3 client
	s3Client, err := platform.NewS3Client(cfg.AWS)
	if err != nil {
		return fmt.Errorf("creating S3 client: %w", err)
	}
	logger.Info("initialized S3 client", "bucket", cfg.AWS.S3Bucket)

	// Auth module
	keycloakClient := auth.NewKeycloakClient(cfg.Keycloak)
	otpService := auth.NewOTPService(rdb, cfg.OTP.Provider, cfg.OTP.AuthKey, logger)
	authRepo := auth.NewRepository(db)
	authService := auth.NewService(authRepo, keycloakClient, otpService, logger)
	authHandler := auth.NewHandler(authService)

	// Land module
	landRepo := land.NewRepository(db)
	landService := land.NewService(landRepo, authRepo, eventBus, logger)
	landHandler := land.NewHandler(landService)

	// Agent module
	agentRepo := agent.NewRepository(db)
	agentService := agent.NewService(agentRepo, rdb, keycloakClient, otpService, logger)
	agentHandler := agent.NewHandler(agentService)
	locationFlusher := agent.NewLocationFlusher(rdb, agentRepo, logger)
	go locationFlusher.Start(ctx)

	// Survey module
	surveyRepo := survey.NewRepository(db)

	// Job module
	jobRepo := job.NewRepository(db)
	agentQueries := sqlc.New(db)
	matcher := job.NewMatcher(agentQueries, jobRepo, logger)
	dispatcher := job.NewDispatcher(matcher, jobRepo, rdb, eventBus, logger)
	jobScheduler := job.NewScheduler(jobRepo, landRepo, eventBus, logger)
	jobHandler := job.NewHandler(jobRepo, agentRepo, surveyRepo, agentQueries, s3Client, rdb, eventBus, logger)

	// QA module
	qaRepo := qa.NewRepository(db)
	qaService := qa.NewService(qaRepo, surveyRepo, taskQueue, logger)

	// Notification module
	notifRepo := notification.NewRepository(db)
	mockPusher := notification.NewMockPusher(logger)
	mockEmailer := notification.NewMockEmailer(logger)
	mockSMS := notification.NewMockSMSSender(logger)
	notifService := notification.NewService(notifRepo, mockPusher, mockEmailer, mockSMS, logger)
	notifHandler := notification.NewHandler(notifRepo, authRepo)

	// Report module
	reportRepo := report.NewRepository(db)
	reportService := report.NewService(reportRepo, jobRepo, surveyRepo, authRepo, s3Client, taskQueue, logger)
	reportHandler := report.NewHandler(reportRepo, reportService)

	// Register task handlers
	taskQueue.Register("qa.score_survey", qaService.HandleTask)
	taskQueue.Register("report.generate", reportService.HandleTask)
	taskQueue.Register("notification.send", notifService.HandleTask)

	// Start task queue
	go taskQueue.Start(ctx)

	// WebSocket handler
	wsHandler := ws.NewHandler(rdb, keycloakClient, agentRepo, logger)

	// Subscribe dispatcher to job.created events
	eventBus.Subscribe("job.created", dispatcher.HandleJobCreated)

	// Subscribe to survey.submitted — enqueues QA scoring task
	eventBus.Subscribe("survey.submitted", func(ctx context.Context, event platform.Event) {
		payload, ok := event.Payload.(map[string]string)
		if !ok {
			logger.Error("invalid survey.submitted payload")
			return
		}
		if err := taskQueue.Enqueue(ctx, "qa.score_survey", qa.SurveyQAPayload{
			JobID:    payload["job_id"],
			ParcelID: payload["parcel_id"],
			UserID:   payload["user_id"],
		}); err != nil {
			logger.Error("failed to enqueue QA task", "error", err)
		}
	})

	// Start job scheduler
	go jobScheduler.Start(ctx)

	// Router
	r := chi.NewRouter()

	// Global middleware (applied to all routes)
	r.Use(chimw.RealIP)
	r.Use(platform.RequestID)
	r.Use(platform.Logging(logger))
	r.Use(platform.Recovery(logger))

	r.Use(chimw.Compress(5))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		platform.JSON(w, http.StatusOK, map[string]string{"status": "ok", "version": Version})
	})

	// WebSocket endpoint (outside /v1 prefix, auth via query param)
	r.Get("/ws", wsHandler.ServeWS)

	// API v1 routes
	r.Route("/v1", func(r chi.Router) {
		r.Mount("/auth", authHandler.Routes())

		// Public agent registration (no JWT required)
		r.Post("/agents/register", agentHandler.Register)

		// Protected routes (JWT required)
		r.Group(func(r chi.Router) {
			r.Use(auth.JWTAuth(keycloakClient))
			r.Mount("/parcels", landHandler.Routes())
			r.Mount("/agents", agentHandler.Routes())
			r.Mount("/jobs", jobHandler.Routes())
			r.Mount("/alerts", notifHandler.Routes())

			// Report routes
			r.Get("/parcels/{parcelId}/reports", reportHandler.ListByParcel)
			r.Get("/reports/{id}/download", reportHandler.Download)

			// Agent-specific job/offer routes (explicit to avoid mount conflicts)
			r.With(auth.RequireRole("agent")).Get("/agents/me/jobs", jobHandler.ListAgentJobs)
			r.With(auth.RequireRole("agent")).Get("/agents/me/offers", jobHandler.ListAgentOffers)
		})
	})

	_ = dispatcher    // Subscribed via eventBus, kept alive by event loop
	_ = agentService  // Used by agentHandler, kept alive by router
	_ = landService   // Used by landHandler, kept alive by router

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

	logger.Info("server starting", "version", Version, "addr", addr, "env", cfg.Server.Env)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
