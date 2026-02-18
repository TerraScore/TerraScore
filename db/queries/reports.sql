-- name: CreateReport :one
INSERT INTO reports (parcel_id, job_id, s3_key, report_type, format, generated_at)
VALUES ($1, $2, $3, $4, $5, NOW())
RETURNING *;

-- name: GetReportByID :one
SELECT * FROM reports WHERE id = $1;

-- name: ListReportsByParcel :many
SELECT * FROM reports
WHERE parcel_id = $1
ORDER BY generated_at DESC
LIMIT $2 OFFSET $3;

-- name: CountReportsByParcel :one
SELECT count(*) FROM reports WHERE parcel_id = $1;
