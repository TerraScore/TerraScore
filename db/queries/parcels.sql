-- name: CreateParcel :one
INSERT INTO parcels (
    user_id, label, survey_number, village, taluk, district, state, state_code, pin_code,
    boundary, land_type, registered_area_sqm, title_deed_s3_key
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, ST_GeomFromGeoJSON($10), $11, $12, $13)
RETURNING *;

-- name: GetParcelByID :one
SELECT * FROM parcels WHERE id = $1;

-- name: ListParcelsByUser :many
SELECT * FROM parcels
WHERE user_id = $1 AND status = 'active'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountParcelsByUser :one
SELECT count(*) FROM parcels WHERE user_id = $1 AND status = 'active';

-- name: FindParcelsNeedingSurvey :many
SELECT p.* FROM parcels p
LEFT JOIN survey_jobs sj ON sj.parcel_id = p.id AND sj.status NOT IN ('completed', 'cancelled')
WHERE p.status = 'active' AND sj.id IS NULL
ORDER BY p.monitoring_since ASC
LIMIT $1;

-- name: UpdateParcelStatus :exec
UPDATE parcels SET status = $2, updated_at = NOW() WHERE id = $1;

-- name: DeleteParcel :exec
UPDATE parcels SET status = 'deleted', updated_at = NOW() WHERE id = $1;
