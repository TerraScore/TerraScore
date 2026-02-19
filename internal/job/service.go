package job

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// ParcelEmbed is a lightweight parcel representation embedded in job responses.
type ParcelEmbed struct {
	ID              uuid.UUID   `json:"id"`
	Label           *string     `json:"label,omitempty"`
	Village         *string     `json:"village,omitempty"`
	Taluk           *string     `json:"taluk,omitempty"`
	District        string      `json:"district"`
	State           string      `json:"state"`
	BoundaryGeoJSON interface{} `json:"boundary_geojson,omitempty"`
	AreaSqm         *float32    `json:"area_sqm,omitempty"`
}

// JobResponse is the API representation of a survey job.
type JobResponse struct {
	ID              uuid.UUID    `json:"id"`
	ParcelID        uuid.UUID    `json:"parcel_id"`
	UserID          uuid.UUID    `json:"user_id"`
	SurveyType      string       `json:"survey_type"`
	Priority        *string      `json:"priority,omitempty"`
	Deadline        time.Time    `json:"deadline"`
	Status          *string      `json:"status"`
	AssignedAgentID *uuid.UUID   `json:"assigned_agent_id,omitempty"`
	AssignedAt      *time.Time   `json:"assigned_at,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`
	Parcel          *ParcelEmbed `json:"parcel,omitempty"`
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

// ArriveRequest is the payload for agent arrival confirmation.
type ArriveRequest struct {
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Accuracy float64 `json:"accuracy,omitempty"`
}

// PresignedURLResponse is the response for presigned URL generation.
type PresignedURLResponse struct {
	UploadURL string `json:"upload_url"`
	S3Key     string `json:"s3_key"`
	ExpiresIn int    `json:"expires_in"`
}

// MediaRequest is the payload for recording media metadata after S3 upload.
type MediaRequest struct {
	S3Key      string    `json:"s3_key"`
	StepID     string    `json:"step_id"`
	MediaType  string    `json:"media_type"`
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	Accuracy   float64   `json:"accuracy,omitempty"`
	SHA256     string    `json:"sha256"`
	FileSize   *int64    `json:"file_size,omitempty"`
	CapturedAt time.Time `json:"captured_at"`
}

// MediaResponse is the API representation of a survey media record.
type MediaResponse struct {
	ID        uuid.UUID `json:"id"`
	S3Key     string    `json:"s3_key"`
	StepID    string    `json:"step_id"`
	MediaType string    `json:"media_type"`
	UploadedAt time.Time `json:"uploaded_at"`
}

// ParcelSurveyResponse is the API representation of a job with survey/QA info for landowners.
type ParcelSurveyResponse struct {
	ID         uuid.UUID       `json:"id"`
	SurveyType string          `json:"survey_type"`
	Status     *string         `json:"status"`
	QAScore    *float64        `json:"qa_score,omitempty"`
	QAStatus   *string         `json:"qa_status,omitempty"`
	QANotes    *string         `json:"qa_notes,omitempty"`
	Responses  json.RawMessage `json:"responses,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
}

// SurveySubmitRequest is the payload for submitting a survey.
type SurveySubmitRequest struct {
	Responses       json.RawMessage `json:"responses"`
	GPSTrailGeoJSON string          `json:"gps_trail_geojson"`
	DeviceInfo      json.RawMessage `json:"device_info,omitempty"`
	StartedAt       *time.Time      `json:"started_at,omitempty"`
	DurationMinutes *float32        `json:"duration_minutes,omitempty"`
	TemplateID      *uuid.UUID      `json:"template_id,omitempty"`
}

// TemplateResponse is the API representation of a checklist template.
type TemplateResponse struct {
	ID         uuid.UUID       `json:"id"`
	Name       string          `json:"name"`
	SurveyType string          `json:"survey_type"`
	Version    *int32          `json:"version,omitempty"`
	Steps      json.RawMessage `json:"steps"`
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
