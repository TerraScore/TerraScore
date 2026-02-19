package job

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/terrascore/api/db/sqlc"
	"github.com/terrascore/api/internal/land"
	"github.com/terrascore/api/internal/platform"
)

// Scheduler creates survey jobs from parcels needing surveys on a periodic ticker.
type Scheduler struct {
	jobRepo  *Repository
	landRepo *land.Repository
	eventBus *platform.EventBus
	logger   *slog.Logger
	interval time.Duration
}

// NewScheduler creates a job scheduler that runs every hour.
func NewScheduler(jobRepo *Repository, landRepo *land.Repository, eventBus *platform.EventBus, logger *slog.Logger) *Scheduler {
	return &Scheduler{
		jobRepo:  jobRepo,
		landRepo: landRepo,
		eventBus: eventBus,
		logger:   logger,
		interval: 1 * time.Hour,
	}
}

// Start runs the scheduler loop. Call in a goroutine.
func (s *Scheduler) Start(ctx context.Context) {
	s.logger.Info("job scheduler started", "interval", s.interval)

	// Run once immediately on startup
	s.tick(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("job scheduler stopped")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	s.logger.Debug("scheduler tick: looking for parcels needing survey")

	parcels, err := s.landRepo.FindParcelsNeedingSurvey(ctx, 100)
	if err != nil {
		s.logger.Error("scheduler: failed to find parcels", "error", err)
		return
	}

	if len(parcels) == 0 {
		s.logger.Debug("scheduler: no parcels need surveys")
		return
	}

	created := 0
	for _, p := range parcels {
		job, err := s.createJobForParcel(ctx, p)
		if err != nil {
			s.logger.Error("scheduler: failed to create job",
				"parcel_id", p.ID,
				"error", err,
			)
			continue
		}

		s.eventBus.Publish(platform.Event{
			Type:    "job.created",
			Payload: job,
		})
		created++
	}

	s.logger.Info("scheduler tick complete",
		"parcels_found", len(parcels),
		"jobs_created", created,
	)
}

// CreateJobForParcel creates a basic_check survey job for a parcel and publishes a job.created event.
func (s *Scheduler) CreateJobForParcel(ctx context.Context, parcel sqlc.Parcel) (*sqlc.SurveyJob, error) {
	job, err := s.createJobForParcel(ctx, parcel)
	if err != nil {
		return nil, err
	}

	s.eventBus.Publish(platform.Event{
		Type:    "job.created",
		Payload: job,
	})

	return job, nil
}

// createJobForParcel creates a basic_check survey job with a 72-hour deadline.
// Phase 1 simplification: all parcels get basic_check, no subscription check.
func (s *Scheduler) createJobForParcel(ctx context.Context, p sqlc.Parcel) (*sqlc.SurveyJob, error) {
	surveyType := "basic_check"
	priority := "normal"
	trigger := "scheduled"
	deadline := time.Now().Add(72 * time.Hour)

	// Base payout: 500 INR (stored as numeric)
	basePayout := pgtype.Numeric{}
	basePayout.Scan("500.00")

	params := sqlc.CreateSurveyJobParams{
		ParcelID:   p.ID,
		UserID:     p.UserID,
		SurveyType: surveyType,
		Priority:   &priority,
		Deadline:   deadline,
		Trigger:    &trigger,
		BasePayout: basePayout,
	}

	job, err := s.jobRepo.CreateJob(ctx, params)
	if err != nil {
		return nil, err
	}

	s.logger.Info("created survey job",
		"job_id", job.ID,
		"parcel_id", p.ID,
		"survey_type", surveyType,
		"deadline", deadline,
	)

	return job, nil
}
