-- name: CreateTransaction :one
INSERT INTO transactions (user_id, subscription_id, type, amount, status, razorpay_payment_id, razorpay_order_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetTransactionByID :one
SELECT * FROM transactions WHERE id = $1;

-- name: ListTransactionsByUser :many
SELECT * FROM transactions WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: UpdateTransactionStatus :exec
UPDATE transactions SET status = $2 WHERE id = $1;

-- name: CreateSubscription :one
INSERT INTO subscriptions (user_id, parcel_id, plan, amount_per_cycle, current_period_start, current_period_end)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetSubscriptionByID :one
SELECT * FROM subscriptions WHERE id = $1;

-- name: GetActiveSubscription :one
SELECT * FROM subscriptions WHERE parcel_id = $1 AND status = 'active' LIMIT 1;

-- name: UpdateSubscriptionStatus :exec
UPDATE subscriptions SET status = $2, updated_at = NOW() WHERE id = $1;

-- name: CreateAgentPayout :one
INSERT INTO agent_payouts (agent_id, period_start, period_end, total_jobs, gross_amount, platform_commission, tds_deducted, net_amount)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListPayoutsByAgent :many
SELECT * FROM agent_payouts WHERE agent_id = $1 ORDER BY period_end DESC LIMIT $2 OFFSET $3;
