package qa

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/terrascore/api/internal/platform"
	"github.com/terrascore/api/internal/survey"
)

// Service handles QA scoring for survey submissions.
type Service struct {
	qaRepo     *Repository
	surveyRepo *survey.Repository
	taskQueue  *platform.TaskQueue
	logger     *slog.Logger
}

// NewService creates a QA service.
func NewService(qaRepo *Repository, surveyRepo *survey.Repository, taskQueue *platform.TaskQueue, logger *slog.Logger) *Service {
	return &Service{
		qaRepo:     qaRepo,
		surveyRepo: surveyRepo,
		taskQueue:  taskQueue,
		logger:     logger,
	}
}

// ScoreSurvey runs all QA checks and returns a ScoreResult.
func (s *Service) ScoreSurvey(ctx context.Context, jobID, parcelID uuid.UUID) (*ScoreResult, error) {
	checks := make([]CheckScore, 0, 5)

	// 1. Geo check (25%): media within boundary
	geoScore, geoDetail := s.checkGeo(ctx, jobID)
	checks = append(checks, CheckScore{Name: "geo_within_boundary", Weight: WeightGeo, Score: geoScore, Detail: geoDetail})

	// 2. Completeness check (25%): media count + response completeness
	compScore, compDetail := s.checkCompleteness(ctx, jobID)
	checks = append(checks, CheckScore{Name: "completeness", Weight: WeightCompleteness, Score: compScore, Detail: compDetail})

	// 3. Boundary walk check (20%): Hausdorff distance
	walkScore, walkDetail := s.checkBoundaryWalk(ctx, jobID)
	checks = append(checks, CheckScore{Name: "boundary_walk", Weight: WeightBoundaryWalk, Score: walkScore, Detail: walkDetail})

	// 4. Timestamps check (15%): reasonable timing
	tsScore, tsDetail := s.checkTimestamps(ctx, jobID)
	checks = append(checks, CheckScore{Name: "timestamps", Weight: WeightTimestamps, Score: tsScore, Detail: tsDetail})

	// 5. Duplicate check (15%): hash-based
	dupScore, dupDetail := s.checkDuplicates(ctx, jobID, parcelID)
	checks = append(checks, CheckScore{Name: "duplicates", Weight: WeightDuplicate, Score: dupScore, Detail: dupDetail})

	// Calculate weighted overall score
	var overall float64
	for _, c := range checks {
		overall += c.Score * c.Weight
	}

	// Determine status
	status := StatusPassed
	var notes []string

	// Force reject if geo score below threshold
	if geoScore < ThresholdGeoReject {
		status = StatusFailed
		notes = append(notes, "geo check below threshold — possible location fraud")
	} else if overall < ThresholdFlagged {
		status = StatusFailed
		notes = append(notes, "overall score below minimum threshold")
	} else if overall < ThresholdAutoPass {
		status = StatusFlagged
		notes = append(notes, "score below auto-pass threshold — needs manual review")
	}

	// Random 20% flagging for quality assurance
	if status == StatusPassed {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		if rng.Float64() < ThresholdRandomFlag {
			status = StatusFlagged
			notes = append(notes, "randomly selected for manual review")
		}
	}

	noteStr := "all checks passed"
	if len(notes) > 0 {
		noteStr = strings.Join(notes, "; ")
	}

	return &ScoreResult{
		OverallScore: overall,
		Status:       status,
		Notes:        noteStr,
		Checks:       checks,
	}, nil
}

// HandleTask is the TaskHandler for "qa.score_survey".
func (s *Service) HandleTask(ctx context.Context, taskType string, payload json.RawMessage) error {
	var p SurveyQAPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("unmarshalling QA payload: %w", err)
	}

	jobID, err := uuid.Parse(p.JobID)
	if err != nil {
		return fmt.Errorf("invalid job ID: %w", err)
	}
	parcelID, err := uuid.Parse(p.ParcelID)
	if err != nil {
		return fmt.Errorf("invalid parcel ID: %w", err)
	}

	s.logger.Info("scoring survey", "job_id", jobID, "parcel_id", parcelID)

	result, err := s.ScoreSurvey(ctx, jobID, parcelID)
	if err != nil {
		return fmt.Errorf("scoring survey: %w", err)
	}

	// Update job QA columns
	if err := s.qaRepo.UpdateJobQA(ctx, jobID, result.OverallScore, result.Status, result.Notes); err != nil {
		return fmt.Errorf("updating QA result: %w", err)
	}

	s.logger.Info("QA scoring complete",
		"job_id", jobID,
		"score", result.OverallScore,
		"status", result.Status,
	)

	// Enqueue report generation
	if err := s.taskQueue.Enqueue(ctx, "report.generate", map[string]string{
		"job_id":    p.JobID,
		"parcel_id": p.ParcelID,
		"user_id":   p.UserID,
	}); err != nil {
		s.logger.Error("failed to enqueue report generation", "error", err)
	}

	return nil
}

func (s *Service) checkGeo(ctx context.Context, jobID uuid.UUID) (float64, string) {
	within, total, err := s.qaRepo.CheckMediaWithinBoundary(ctx, jobID)
	if err != nil || total == 0 {
		return 0.0, "no media found or error checking geo"
	}
	score := float64(within) / float64(total)
	return score, fmt.Sprintf("%d/%d media within boundary", within, total)
}

func (s *Service) checkCompleteness(ctx context.Context, jobID uuid.UUID) (float64, string) {
	// Check that survey response exists and has media
	_, err := s.surveyRepo.GetSurveyResponseByJob(ctx, jobID)
	if err != nil {
		return 0.0, "no survey response found"
	}

	mediaCount, err := s.surveyRepo.CountMediaByJob(ctx, jobID)
	if err != nil || mediaCount == 0 {
		return 0.3, "survey response exists but no media"
	}

	// Score based on media count: 1+ photos = minimum 0.5, 3+ = 0.8, 5+ = 1.0
	var score float64
	switch {
	case mediaCount >= 5:
		score = 1.0
	case mediaCount >= 3:
		score = 0.8
	case mediaCount >= 1:
		score = 0.5
	}

	return score, fmt.Sprintf("%d media files uploaded", mediaCount)
}

func (s *Service) checkBoundaryWalk(ctx context.Context, jobID uuid.UUID) (float64, string) {
	meters, err := s.qaRepo.CheckBoundaryWalkDistance(ctx, jobID)
	if err != nil {
		return 0.0, "error checking boundary walk"
	}

	var score float64
	switch {
	case meters < 50:
		score = 1.0
	case meters < 100:
		score = 0.5
	default:
		score = 0.0
	}

	return score, fmt.Sprintf("Hausdorff distance: %.0fm", meters)
}

func (s *Service) checkTimestamps(ctx context.Context, jobID uuid.UUID) (float64, string) {
	media, err := s.surveyRepo.ListMediaByJob(ctx, jobID)
	if err != nil || len(media) == 0 {
		return 0.5, "no media timestamps to validate"
	}

	// Check that all media was captured within a 2-hour window
	var earliest, latest time.Time
	for i, m := range media {
		if i == 0 || m.CapturedAt.Before(earliest) {
			earliest = m.CapturedAt
		}
		if i == 0 || m.CapturedAt.After(latest) {
			latest = m.CapturedAt
		}
	}

	duration := latest.Sub(earliest)
	score := 1.0

	// Penalize if session > 2 hours (suspicious)
	if duration > 2*time.Hour {
		score = 0.3
		return score, fmt.Sprintf("media span %.0f minutes — exceeds 2h window", duration.Minutes())
	}

	// Reward if >= 15 minutes on-site
	if duration >= 15*time.Minute {
		score = 1.0
	} else if duration >= 5*time.Minute {
		score = 0.7
	} else {
		score = 0.4
	}

	return score, fmt.Sprintf("on-site duration: %.0f minutes", duration.Minutes())
}

func (s *Service) checkDuplicates(ctx context.Context, jobID, parcelID uuid.UUID) (float64, string) {
	dupes, total, err := s.qaRepo.FindDuplicateHashes(ctx, jobID, parcelID)
	if err != nil || total == 0 {
		return 1.0, "no media to check for duplicates"
	}

	score := 1.0 - float64(dupes)/float64(total)
	if score < 0 {
		score = 0
	}

	return score, fmt.Sprintf("%d/%d media files are duplicates from other surveys", dupes, total)
}
