package qa

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/terrascore/api/db/sqlc"
)

// Repository handles QA-related spatial queries using raw SQL.
type Repository struct {
	q  *sqlc.Queries
	db *pgxpool.Pool
}

// NewRepository creates a QA repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		q:  sqlc.New(db),
		db: db,
	}
}

// CheckMediaWithinBoundary counts how many media items have GPS within the parcel boundary.
func (r *Repository) CheckMediaWithinBoundary(ctx context.Context, jobID uuid.UUID) (within, total int, err error) {
	err = r.db.QueryRow(ctx,
		`SELECT
			COUNT(*) FILTER (WHERE ST_Contains(p.boundary::geometry, sm.location::geometry)) as within_count,
			COUNT(*) as total_count
		FROM survey_media sm
		JOIN survey_jobs sj ON sm.job_id = sj.id
		JOIN parcels p ON sj.parcel_id = p.id
		WHERE sm.job_id = $1`,
		jobID,
	).Scan(&within, &total)
	if err != nil {
		return 0, 0, fmt.Errorf("checking media within boundary: %w", err)
	}
	return within, total, nil
}

// CheckBoundaryWalkDistance calculates the Hausdorff distance between the GPS trail and parcel boundary.
// Returns distance in meters. Lower is better (agent walked closer to boundary).
func (r *Repository) CheckBoundaryWalkDistance(ctx context.Context, jobID uuid.UUID) (meters float64, err error) {
	err = r.db.QueryRow(ctx,
		`SELECT COALESCE(
			ST_HausdorffDistance(
				p.boundary::geometry,
				sr.gps_trail::geometry
			) * 111320, -- approximate degrees to meters at equator
			999999
		)
		FROM survey_responses sr
		JOIN survey_jobs sj ON sr.job_id = sj.id
		JOIN parcels p ON sj.parcel_id = p.id
		WHERE sr.job_id = $1`,
		jobID,
	).Scan(&meters)
	if err != nil {
		return 999999, fmt.Errorf("checking boundary walk distance: %w", err)
	}
	return meters, nil
}

// FindDuplicateHashes checks how many media hashes in this job match media from other jobs on the same parcel.
func (r *Repository) FindDuplicateHashes(ctx context.Context, jobID, parcelID uuid.UUID) (dupes, total int, err error) {
	err = r.db.QueryRow(ctx,
		`SELECT
			COUNT(*) FILTER (WHERE sm.file_hash_sha256 IN (
				SELECT sm2.file_hash_sha256
				FROM survey_media sm2
				JOIN survey_jobs sj2 ON sm2.job_id = sj2.id
				WHERE sj2.parcel_id = $2
				  AND sm2.job_id != $1
				  AND sm2.file_hash_sha256 != ''
			)) as dupe_count,
			COUNT(*) as total_count
		FROM survey_media sm
		WHERE sm.job_id = $1 AND sm.file_hash_sha256 != ''`,
		jobID, parcelID,
	).Scan(&dupes, &total)
	if err != nil {
		return 0, 0, fmt.Errorf("finding duplicate hashes: %w", err)
	}
	return dupes, total, nil
}

// UpdateJobQA updates the QA score, status, and notes on a survey job.
func (r *Repository) UpdateJobQA(ctx context.Context, jobID uuid.UUID, score float64, status, notes string) error {
	scoreNum := pgtype.Numeric{}
	if err := scoreNum.Scan(fmt.Sprintf("%.4f", score)); err != nil {
		return fmt.Errorf("converting score to numeric: %w", err)
	}

	err := r.q.UpdateJobQA(ctx, sqlc.UpdateJobQAParams{
		ID:       jobID,
		QaScore:  scoreNum,
		QaStatus: &status,
		QaNotes:  &notes,
	})
	if err != nil {
		return fmt.Errorf("updating job QA: %w", err)
	}
	return nil
}
