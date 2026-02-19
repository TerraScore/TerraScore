package auth

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/terrascore/api/db/sqlc"
	"github.com/terrascore/api/internal/platform"
)

// Repository handles user persistence beyond what Keycloak stores.
type Repository struct {
	q  *sqlc.Queries
	db *pgxpool.Pool
}

// NewRepository creates an auth repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		q:  sqlc.New(db),
		db: db,
	}
}

// CreateUser inserts a user into the local database.
func (r *Repository) CreateUser(ctx context.Context, params sqlc.CreateUserParams) (*sqlc.User, error) {
	user, err := r.q.CreateUser(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}
	return &user, nil
}

// GetUserByPhone finds a user by phone number.
func (r *Repository) GetUserByPhone(ctx context.Context, phone string) (*sqlc.User, error) {
	user, err := r.q.GetUserByPhone(ctx, phone)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, platform.NewNotFound("user not found")
		}
		return nil, fmt.Errorf("getting user by phone: %w", err)
	}
	return &user, nil
}

// GetKeycloakIDByPhone looks up the Keycloak ID by phone, checking users first then agents.
func (r *Repository) GetKeycloakIDByPhone(ctx context.Context, phone string) (string, error) {
	// Check users table first
	user, err := r.q.GetUserByPhone(ctx, phone)
	if err == nil && user.KeycloakID != nil {
		return *user.KeycloakID, nil
	}

	// Check agents table
	agent, err := r.q.GetAgentByPhone(ctx, phone)
	if err == nil && agent.KeycloakID != nil {
		return *agent.KeycloakID, nil
	}

	return "", platform.NewNotFound("user not found")
}

// GetUserByKeycloakID finds a user by their Keycloak ID.
func (r *Repository) GetUserByKeycloakID(ctx context.Context, keycloakID string) (*sqlc.User, error) {
	user, err := r.q.GetUserByKeycloakID(ctx, &keycloakID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, platform.NewNotFound("user not found")
		}
		return nil, fmt.Errorf("getting user by keycloak ID: %w", err)
	}
	return &user, nil
}

// GetUserByID finds a user by UUID.
func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*sqlc.User, error) {
	user, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, platform.NewNotFound("user not found")
		}
		return nil, fmt.Errorf("getting user by ID: %w", err)
	}
	return &user, nil
}
