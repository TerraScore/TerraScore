package notification

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/terrascore/api/db/sqlc"
)

// Repository wraps sqlc alert queries.
type Repository struct {
	q *sqlc.Queries
}

// NewRepository creates a notification repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{q: sqlc.New(db)}
}

// CreateAlert creates an in-app alert.
func (r *Repository) CreateAlert(ctx context.Context, userID uuid.UUID, alertType, title string, body *string, data []byte) (*sqlc.Alert, error) {
	alert, err := r.q.CreateAlert(ctx, sqlc.CreateAlertParams{
		UserID: userID,
		Type:   alertType,
		Title:  title,
		Body:   body,
		Data:   data,
	})
	if err != nil {
		return nil, fmt.Errorf("creating alert: %w", err)
	}
	return &alert, nil
}

// ListAlerts returns paginated alerts for a user.
func (r *Repository) ListAlerts(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]sqlc.Alert, error) {
	alerts, err := r.q.ListAlertsByUser(ctx, sqlc.ListAlertsByUserParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("listing alerts: %w", err)
	}
	return alerts, nil
}

// CountUnread returns the number of unread alerts for a user.
func (r *Repository) CountUnread(ctx context.Context, userID uuid.UUID) (int64, error) {
	count, err := r.q.CountUnreadAlerts(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("counting unread alerts: %w", err)
	}
	return count, nil
}

// MarkRead marks a single alert as read.
func (r *Repository) MarkRead(ctx context.Context, alertID uuid.UUID) error {
	if err := r.q.MarkAlertRead(ctx, alertID); err != nil {
		return fmt.Errorf("marking alert read: %w", err)
	}
	return nil
}

// MarkAllRead marks all alerts as read for a user.
func (r *Repository) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	if err := r.q.MarkAllAlertsRead(ctx, userID); err != nil {
		return fmt.Errorf("marking all alerts read: %w", err)
	}
	return nil
}
