-- name: CreateSurveyJob :one
INSERT INTO survey_jobs (
    parcel_id, subscription_id, user_id, survey_type, priority, deadline, trigger, base_payout
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetSurveyJobByID :one
SELECT * FROM survey_jobs WHERE id = $1;

-- name: UpdateJobStatus :one
UPDATE survey_jobs SET status = $2, updated_at = NOW() WHERE id = $1 RETURNING *;

-- name: AssignAgent :one
UPDATE survey_jobs SET
    assigned_agent_id = $2,
    assigned_at = NOW(),
    status = 'assigned',
    cascade_round = $3,
    total_offers_sent = $4,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: RecordAgentArrival :exec
UPDATE survey_jobs SET
    agent_arrived_at = NOW(),
    arrival_location = ST_SetSRID(ST_MakePoint($2, $3), 4326),
    arrival_distance_m = $4,
    status = 'agent_on_site',
    updated_at = NOW()
WHERE id = $1;

-- name: CompleteJob :exec
UPDATE survey_jobs SET
    completed_at = NOW(),
    status = 'completed',
    qa_status = 'pending',
    total_payout = base_payout + distance_bonus + urgency_bonus,
    updated_at = NOW()
WHERE id = $1;

-- name: ListJobsByParcel :many
SELECT * FROM survey_jobs
WHERE parcel_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListJobsByAgent :many
SELECT * FROM survey_jobs
WHERE assigned_agent_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListPendingJobs :many
SELECT * FROM survey_jobs
WHERE status IN ('pending_assignment', 'offered')
ORDER BY deadline ASC
LIMIT $1;

-- name: CreateJobOffer :one
INSERT INTO job_offers (job_id, agent_id, cascade_round, offer_rank, distance_km, match_score, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateJobOfferStatus :exec
UPDATE job_offers SET status = $2, responded_at = NOW(), decline_reason = $3 WHERE id = $1;

-- name: ListOffersByJob :many
SELECT * FROM job_offers WHERE job_id = $1 ORDER BY cascade_round, offer_rank;

-- name: ListPendingOffersByAgent :many
SELECT * FROM job_offers
WHERE agent_id = $1 AND status = 'sent'
ORDER BY sent_at DESC;

-- name: CountActiveJobsByAgent :one
SELECT count(*) FROM survey_jobs
WHERE assigned_agent_id = $1
    AND status IN ('assigned', 'agent_on_site', 'survey_in_progress');

-- name: GetOfferByJobAndAgent :one
SELECT * FROM job_offers WHERE job_id = $1 AND agent_id = $2 AND status = 'sent';

-- name: GetPendingOfferByID :one
SELECT * FROM job_offers WHERE id = $1 AND status = 'sent';

-- name: ExpireOffers :exec
UPDATE job_offers SET status = 'expired' WHERE expires_at < NOW() AND status = 'sent';
