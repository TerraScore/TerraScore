package survey

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/terrascore/api/db/sqlc"
	"github.com/terrascore/api/internal/platform"
)

// Repository handles survey response and media persistence.
type Repository struct {
	q  *sqlc.Queries
	db *pgxpool.Pool
}

// NewRepository creates a survey repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		q:  sqlc.New(db),
		db: db,
	}
}

// CreateSurveyResponse inserts a new survey response.
func (r *Repository) CreateSurveyResponse(ctx context.Context, params sqlc.CreateSurveyResponseParams) (*sqlc.SurveyResponse, error) {
	resp, err := r.q.CreateSurveyResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("creating survey response: %w", err)
	}
	return &resp, nil
}

// GetSurveyResponseByJob returns the survey response for a job.
func (r *Repository) GetSurveyResponseByJob(ctx context.Context, jobID uuid.UUID) (*sqlc.SurveyResponse, error) {
	resp, err := r.q.GetSurveyResponseByJob(ctx, jobID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, platform.NewNotFound("survey response not found")
		}
		return nil, fmt.Errorf("getting survey response: %w", err)
	}
	return &resp, nil
}

// CreateSurveyMedia inserts a new media record.
func (r *Repository) CreateSurveyMedia(ctx context.Context, params sqlc.CreateSurveyMediaParams) (*sqlc.SurveyMedium, error) {
	media, err := r.q.CreateSurveyMedia(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("creating survey media: %w", err)
	}
	return &media, nil
}

// ListMediaByJob returns all media for a job.
func (r *Repository) ListMediaByJob(ctx context.Context, jobID uuid.UUID) ([]sqlc.SurveyMedium, error) {
	media, err := r.q.ListMediaByJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("listing media by job: %w", err)
	}
	return media, nil
}

// CountMediaByJob returns the count of media for a job.
func (r *Repository) CountMediaByJob(ctx context.Context, jobID uuid.UUID) (int64, error) {
	count, err := r.q.CountMediaByJob(ctx, jobID)
	if err != nil {
		return 0, fmt.Errorf("counting media by job: %w", err)
	}
	return count, nil
}

// GetActiveTemplate returns the active checklist template for a survey type.
func (r *Repository) GetActiveTemplate(ctx context.Context, surveyType string) (*sqlc.ChecklistTemplate, error) {
	tmpl, err := r.q.GetActiveTemplate(ctx, surveyType)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, platform.NewNotFound("no active template for survey type: " + surveyType)
		}
		return nil, fmt.Errorf("getting active template: %w", err)
	}
	return &tmpl, nil
}
