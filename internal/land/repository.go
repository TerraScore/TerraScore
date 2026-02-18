package land

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/terrascore/api/db/sqlc"
	"github.com/terrascore/api/internal/platform"
)

// Repository handles parcel persistence.
type Repository struct {
	q  *sqlc.Queries
	db *pgxpool.Pool
}

// NewRepository creates a land repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		q:  sqlc.New(db),
		db: db,
	}
}

// CreateParcel inserts a new parcel.
func (r *Repository) CreateParcel(ctx context.Context, params sqlc.CreateParcelParams) (*sqlc.Parcel, error) {
	parcel, err := r.q.CreateParcel(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("creating parcel: %w", err)
	}
	return &parcel, nil
}

// GetParcelByID returns a parcel by its ID.
func (r *Repository) GetParcelByID(ctx context.Context, id uuid.UUID) (*sqlc.Parcel, error) {
	parcel, err := r.q.GetParcelByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, platform.NewNotFound("parcel not found")
		}
		return nil, fmt.Errorf("getting parcel: %w", err)
	}
	return &parcel, nil
}

// GetParcelWithGeoJSON returns a parcel with the boundary as a GeoJSON string.
func (r *Repository) GetParcelWithGeoJSON(ctx context.Context, id uuid.UUID) (*sqlc.GetParcelWithGeoJSONRow, error) {
	row, err := r.q.GetParcelWithGeoJSON(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, platform.NewNotFound("parcel not found")
		}
		return nil, fmt.Errorf("getting parcel with geojson: %w", err)
	}
	return &row, nil
}

// ListParcelsByUser returns paginated parcels for a user.
func (r *Repository) ListParcelsByUser(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]sqlc.Parcel, error) {
	parcels, err := r.q.ListParcelsByUser(ctx, sqlc.ListParcelsByUserParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("listing parcels: %w", err)
	}
	return parcels, nil
}

// CountParcelsByUser returns the total count of active parcels for a user.
func (r *Repository) CountParcelsByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	count, err := r.q.CountParcelsByUser(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("counting parcels: %w", err)
	}
	return count, nil
}

// UpdateParcelBoundary updates the parcel's boundary geometry.
func (r *Repository) UpdateParcelBoundary(ctx context.Context, id uuid.UUID, geoJSON string) error {
	err := r.q.UpdateParcelBoundary(ctx, sqlc.UpdateParcelBoundaryParams{
		ID:                id,
		StGeomfromgeojson: geoJSON,
	})
	if err != nil {
		return fmt.Errorf("updating parcel boundary: %w", err)
	}
	return nil
}

// DeleteParcel soft-deletes a parcel by setting status to 'deleted'.
func (r *Repository) DeleteParcel(ctx context.Context, id uuid.UUID) error {
	err := r.q.DeleteParcel(ctx, id)
	if err != nil {
		return fmt.Errorf("deleting parcel: %w", err)
	}
	return nil
}
