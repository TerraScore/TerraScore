package agent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/terrascore/api/db/sqlc"
	"github.com/terrascore/api/internal/platform"
)

// Repository handles agent persistence.
type Repository struct {
	q  *sqlc.Queries
	db *pgxpool.Pool
}

// NewRepository creates an agent repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		q:  sqlc.New(db),
		db: db,
	}
}

// CreateAgent inserts a new agent.
func (r *Repository) CreateAgent(ctx context.Context, params sqlc.CreateAgentParams) (*sqlc.Agent, error) {
	agent, err := r.q.CreateAgent(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("creating agent: %w", err)
	}
	return &agent, nil
}

// GetAgentByID returns an agent by ID.
func (r *Repository) GetAgentByID(ctx context.Context, id uuid.UUID) (*sqlc.Agent, error) {
	agent, err := r.q.GetAgentByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, platform.NewNotFound("agent not found")
		}
		return nil, fmt.Errorf("getting agent by ID: %w", err)
	}
	return &agent, nil
}

// GetAgentByKeycloakID returns an agent by Keycloak ID.
func (r *Repository) GetAgentByKeycloakID(ctx context.Context, keycloakID string) (*sqlc.Agent, error) {
	agent, err := r.q.GetAgentByKeycloakID(ctx, &keycloakID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, platform.NewNotFound("agent not found")
		}
		return nil, fmt.Errorf("getting agent by keycloak ID: %w", err)
	}
	return &agent, nil
}

// GetAgentByPhone returns an agent by phone.
func (r *Repository) GetAgentByPhone(ctx context.Context, phone string) (*sqlc.Agent, error) {
	agent, err := r.q.GetAgentByPhone(ctx, phone)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, platform.NewNotFound("agent not found")
		}
		return nil, fmt.Errorf("getting agent by phone: %w", err)
	}
	return &agent, nil
}

// UpdateAgentProfile updates agent profile fields.
func (r *Repository) UpdateAgentProfile(ctx context.Context, params sqlc.UpdateAgentProfileParams) error {
	err := r.q.UpdateAgentProfile(ctx, params)
	if err != nil {
		return fmt.Errorf("updating agent profile: %w", err)
	}
	return nil
}

// UpdateAgentLocation updates agent's last known location in PostGIS.
func (r *Repository) UpdateAgentLocation(ctx context.Context, id uuid.UUID, lng, lat float64) error {
	err := r.q.UpdateAgentLocation(ctx, sqlc.UpdateAgentLocationParams{
		ID:            id,
		StMakepoint:   lng,
		StMakepoint_2: lat,
	})
	if err != nil {
		return fmt.Errorf("updating agent location: %w", err)
	}
	return nil
}

// UpdateAgentOnlineStatus updates the is_online flag.
func (r *Repository) UpdateAgentOnlineStatus(ctx context.Context, id uuid.UUID, isOnline bool) error {
	err := r.q.UpdateAgentOnlineStatus(ctx, sqlc.UpdateAgentOnlineStatusParams{
		ID:       id,
		IsOnline: &isOnline,
	})
	if err != nil {
		return fmt.Errorf("updating agent online status: %w", err)
	}
	return nil
}

// UpdateAgentFCMToken updates the FCM token and device info.
func (r *Repository) UpdateAgentFCMToken(ctx context.Context, params sqlc.UpdateAgentFCMTokenParams) error {
	err := r.q.UpdateAgentFCMToken(ctx, params)
	if err != nil {
		return fmt.Errorf("updating agent FCM token: %w", err)
	}
	return nil
}
