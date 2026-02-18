package job

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/terrascore/api/db/sqlc"
	"github.com/terrascore/api/internal/platform"
)

// Repository handles survey job and offer persistence.
type Repository struct {
	q  *sqlc.Queries
	db *pgxpool.Pool
}

// NewRepository creates a job repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		q:  sqlc.New(db),
		db: db,
	}
}

// CreateJob inserts a new survey job.
func (r *Repository) CreateJob(ctx context.Context, params sqlc.CreateSurveyJobParams) (*sqlc.SurveyJob, error) {
	job, err := r.q.CreateSurveyJob(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("creating survey job: %w", err)
	}
	return &job, nil
}

// GetJobByID returns a survey job by ID.
func (r *Repository) GetJobByID(ctx context.Context, id uuid.UUID) (*sqlc.SurveyJob, error) {
	job, err := r.q.GetSurveyJobByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, platform.NewNotFound("job not found")
		}
		return nil, fmt.Errorf("getting job by ID: %w", err)
	}
	return &job, nil
}

// UpdateJobStatus updates the status of a survey job.
func (r *Repository) UpdateJobStatus(ctx context.Context, id uuid.UUID, status string) (*sqlc.SurveyJob, error) {
	job, err := r.q.UpdateJobStatus(ctx, sqlc.UpdateJobStatusParams{
		ID:     id,
		Status: &status,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, platform.NewNotFound("job not found")
		}
		return nil, fmt.Errorf("updating job status: %w", err)
	}
	return &job, nil
}

// AssignAgent assigns an agent to a survey job.
func (r *Repository) AssignAgent(ctx context.Context, params sqlc.AssignAgentParams) (*sqlc.SurveyJob, error) {
	job, err := r.q.AssignAgent(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("assigning agent: %w", err)
	}
	return &job, nil
}

// ListPendingJobs returns jobs awaiting assignment.
func (r *Repository) ListPendingJobs(ctx context.Context, limit int32) ([]sqlc.SurveyJob, error) {
	jobs, err := r.q.ListPendingJobs(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("listing pending jobs: %w", err)
	}
	return jobs, nil
}

// ListJobsByAgent returns paginated jobs for an agent.
func (r *Repository) ListJobsByAgent(ctx context.Context, agentID uuid.UUID, limit, offset int32) ([]sqlc.SurveyJob, error) {
	jobs, err := r.q.ListJobsByAgent(ctx, sqlc.ListJobsByAgentParams{
		AssignedAgentID: pgtype.UUID{Bytes: agentID, Valid: true},
		Limit:           limit,
		Offset:          offset,
	})
	if err != nil {
		return nil, fmt.Errorf("listing jobs by agent: %w", err)
	}
	return jobs, nil
}

// CreateOffer inserts a new job offer.
func (r *Repository) CreateOffer(ctx context.Context, params sqlc.CreateJobOfferParams) (*sqlc.JobOffer, error) {
	offer, err := r.q.CreateJobOffer(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("creating job offer: %w", err)
	}
	return &offer, nil
}

// UpdateOfferStatus updates an offer's status and optional decline reason.
func (r *Repository) UpdateOfferStatus(ctx context.Context, id uuid.UUID, status string, reason *string) error {
	err := r.q.UpdateJobOfferStatus(ctx, sqlc.UpdateJobOfferStatusParams{
		ID:            id,
		Status:        &status,
		DeclineReason: reason,
	})
	if err != nil {
		return fmt.Errorf("updating offer status: %w", err)
	}
	return nil
}

// GetOfferByJobAndAgent returns a pending offer for a specific job+agent pair.
func (r *Repository) GetOfferByJobAndAgent(ctx context.Context, jobID, agentID uuid.UUID) (*sqlc.JobOffer, error) {
	offer, err := r.q.GetOfferByJobAndAgent(ctx, sqlc.GetOfferByJobAndAgentParams{
		JobID:   jobID,
		AgentID: agentID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, platform.NewNotFound("offer not found")
		}
		return nil, fmt.Errorf("getting offer by job and agent: %w", err)
	}
	return &offer, nil
}

// GetPendingOfferByID returns a pending offer by its ID.
func (r *Repository) GetPendingOfferByID(ctx context.Context, id uuid.UUID) (*sqlc.JobOffer, error) {
	offer, err := r.q.GetPendingOfferByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, platform.NewNotFound("offer not found or already responded")
		}
		return nil, fmt.Errorf("getting pending offer: %w", err)
	}
	return &offer, nil
}

// ListOffersByJob returns all offers for a job ordered by round and rank.
func (r *Repository) ListOffersByJob(ctx context.Context, jobID uuid.UUID) ([]sqlc.JobOffer, error) {
	offers, err := r.q.ListOffersByJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("listing offers by job: %w", err)
	}
	return offers, nil
}

// ListPendingOffersByAgent returns pending offers for an agent.
func (r *Repository) ListPendingOffersByAgent(ctx context.Context, agentID uuid.UUID) ([]sqlc.JobOffer, error) {
	offers, err := r.q.ListPendingOffersByAgent(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("listing pending offers by agent: %w", err)
	}
	return offers, nil
}

// ExpireOffers bulk-expires all offers past their deadline.
func (r *Repository) ExpireOffers(ctx context.Context) error {
	err := r.q.ExpireOffers(ctx)
	if err != nil {
		return fmt.Errorf("expiring offers: %w", err)
	}
	return nil
}

// CountActiveJobsByAgent returns the number of active (non-completed) jobs for an agent.
func (r *Repository) CountActiveJobsByAgent(ctx context.Context, agentID uuid.UUID) (int64, error) {
	count, err := r.q.CountActiveJobsByAgent(ctx, pgtype.UUID{Bytes: agentID, Valid: true})
	if err != nil {
		return 0, fmt.Errorf("counting active jobs: %w", err)
	}
	return count, nil
}
