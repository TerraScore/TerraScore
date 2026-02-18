package job

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// JobResponse is the API representation of a survey job.
type JobResponse struct {
	ID              uuid.UUID  `json:"id"`
	ParcelID        uuid.UUID  `json:"parcel_id"`
	UserID          uuid.UUID  `json:"user_id"`
	SurveyType      string     `json:"survey_type"`
	Priority        *string    `json:"priority,omitempty"`
	Deadline        time.Time  `json:"deadline"`
	Status          *string    `json:"status"`
	AssignedAgentID *uuid.UUID `json:"assigned_agent_id,omitempty"`
	AssignedAt      *time.Time `json:"assigned_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// OfferResponse is the API representation of a job offer.
type OfferResponse struct {
	ID           uuid.UUID  `json:"id"`
	JobID        uuid.UUID  `json:"job_id"`
	AgentID      uuid.UUID  `json:"agent_id"`
	CascadeRound int32      `json:"cascade_round"`
	OfferRank    int32      `json:"offer_rank"`
	DistanceKm   *float32   `json:"distance_km,omitempty"`
	Status       *string    `json:"status"`
	ExpiresAt    time.Time  `json:"expires_at"`
	SentAt       *time.Time `json:"sent_at,omitempty"`
}

// DeclineRequest is the payload for declining a job offer.
type DeclineRequest struct {
	Reason string `json:"reason,omitempty"`
}

// JobResponseFromSqlc maps sqlc.SurveyJob fields to JobResponse.
func JobResponseFromSqlc(id uuid.UUID, parcelID uuid.UUID, userID uuid.UUID, surveyType string, priority *string, deadline time.Time, status *string, assignedAgentID pgtype.UUID, assignedAt pgtype.Timestamptz, createdAt pgtype.Timestamptz) JobResponse {
	resp := JobResponse{
		ID:         id,
		ParcelID:   parcelID,
		UserID:     userID,
		SurveyType: surveyType,
		Priority:   priority,
		Deadline:   deadline,
		Status:     status,
	}
	if assignedAgentID.Valid {
		aid := uuid.UUID(assignedAgentID.Bytes)
		resp.AssignedAgentID = &aid
	}
	if assignedAt.Valid {
		t := assignedAt.Time
		resp.AssignedAt = &t
	}
	if createdAt.Valid {
		resp.CreatedAt = createdAt.Time
	}
	return resp
}
