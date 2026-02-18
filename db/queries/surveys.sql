-- name: CreateChecklistTemplate :one
INSERT INTO checklist_templates (name, survey_type, version, steps)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetActiveTemplate :one
SELECT * FROM checklist_templates
WHERE survey_type = $1 AND is_active = TRUE
ORDER BY version DESC
LIMIT 1;

-- name: ListTemplates :many
SELECT * FROM checklist_templates ORDER BY survey_type, version DESC;

-- name: CreateSurveyResponse :one
INSERT INTO survey_responses (job_id, agent_id, template_id, responses, gps_trail, device_info, started_at, duration_minutes)
VALUES ($1, $2, $3, $4, ST_GeomFromGeoJSON($5), $6, $7, $8)
RETURNING *;

-- name: GetSurveyResponseByJob :one
SELECT * FROM survey_responses WHERE job_id = $1;

-- name: CreateSurveyMedia :one
INSERT INTO survey_media (
    job_id, agent_id, step_id, media_type, s3_key, file_size_bytes, duration_sec,
    location, captured_at, file_hash_sha256, device_id, within_boundary
)
VALUES ($1, $2, $3, $4, $5, $6, $7, ST_SetSRID(ST_MakePoint($8, $9), 4326), $10, $11, $12, $13)
RETURNING *;

-- name: ListMediaByJob :many
SELECT * FROM survey_media WHERE job_id = $1 ORDER BY captured_at;

-- name: CountMediaByJob :one
SELECT count(*) FROM survey_media WHERE job_id = $1;

-- name: ListDuplicateHashesForParcel :many
SELECT sm.file_hash_sha256, count(*) as hash_count
FROM survey_media sm
JOIN survey_jobs sj ON sm.job_id = sj.id
WHERE sj.parcel_id = $1
  AND sm.file_hash_sha256 != ''
GROUP BY sm.file_hash_sha256
HAVING count(*) > 1;
