package report

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/terrascore/api/db/sqlc"
	"github.com/terrascore/api/internal/platform"
)

// Repository wraps sqlc report queries.
type Repository struct {
	q *sqlc.Queries
}

// NewRepository creates a report repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{q: sqlc.New(db)}
}

// Create inserts a new report record.
func (r *Repository) Create(ctx context.Context, params sqlc.CreateReportParams) (*sqlc.Report, error) {
	report, err := r.q.CreateReport(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("creating report: %w", err)
	}
	return &report, nil
}

// GetByID returns a report by ID.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*sqlc.Report, error) {
	report, err := r.q.GetReportByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, platform.NewNotFound("report not found")
		}
		return nil, fmt.Errorf("getting report: %w", err)
	}
	return &report, nil
}

// ListByParcel returns paginated reports for a parcel.
func (r *Repository) ListByParcel(ctx context.Context, parcelID uuid.UUID, limit, offset int32) ([]sqlc.Report, error) {
	reports, err := r.q.ListReportsByParcel(ctx, sqlc.ListReportsByParcelParams{
		ParcelID: parcelID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, fmt.Errorf("listing reports: %w", err)
	}
	return reports, nil
}

// CountByParcel returns the total number of reports for a parcel.
func (r *Repository) CountByParcel(ctx context.Context, parcelID uuid.UUID) (int64, error) {
	count, err := r.q.CountReportsByParcel(ctx, parcelID)
	if err != nil {
		return 0, fmt.Errorf("counting reports: %w", err)
	}
	return count, nil
}
