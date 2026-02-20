package job

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
	"github.com/terrascore/api/db/sqlc"
	"github.com/terrascore/api/internal/agent"
	"github.com/terrascore/api/internal/auth"
	"github.com/terrascore/api/internal/platform"
	"github.com/terrascore/api/internal/survey"
)

// Handler handles job HTTP endpoints.
type Handler struct {
	jobRepo    *Repository
	agentRepo  *agent.Repository
	surveyRepo *survey.Repository
	queries    *sqlc.Queries
	scheduler  *Scheduler
	s3Client   *platform.S3Client
	rdb        *redis.Client
	eventBus   *platform.EventBus
	logger     *slog.Logger
}

// NewHandler creates a job handler.
func NewHandler(jobRepo *Repository, agentRepo *agent.Repository, surveyRepo *survey.Repository, queries *sqlc.Queries, scheduler *Scheduler, s3Client *platform.S3Client, rdb *redis.Client, eventBus *platform.EventBus, logger *slog.Logger) *Handler {
	return &Handler{
		jobRepo:    jobRepo,
		agentRepo:  agentRepo,
		surveyRepo: surveyRepo,
		queries:    queries,
		scheduler:  scheduler,
		s3Client:   s3Client,
		rdb:        rdb,
		eventBus:   eventBus,
		logger:     logger,
	}
}

// publishAgentEvent sends a real-time event to an agent's WebSocket via Redis pub/sub.
func (h *Handler) publishAgentEvent(ctx context.Context, agentID uuid.UUID, eventType string, data map[string]string) {
	payload, _ := json.Marshal(map[string]interface{}{
		"type": eventType,
		"data": data,
	})
	channel := "agent:" + agentID.String() + ":events"
	h.rdb.Publish(ctx, channel, string(payload))
}

// Routes returns the job router.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// All job routes require authentication
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireRole("agent"))
		r.Post("/{id}/accept", h.AcceptOffer)
		r.Post("/{id}/decline", h.DeclineOffer)
		r.Get("/{id}", h.GetJob)
		r.Post("/{id}/arrive", h.Arrive)
		r.Get("/{id}/media/presigned", h.PresignedURL)
		r.Post("/{id}/media/upload", h.UploadMedia)
		r.Post("/{id}/media", h.RecordMedia)
		r.Post("/{id}/survey", h.SubmitSurvey)
		r.Get("/{id}/template", h.GetTemplate)
	})

	return r
}

// AcceptOffer handles POST /v1/jobs/{id}/accept.
func (h *Handler) AcceptOffer(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.HandleError(w, platform.NewBadRequest("invalid job ID"))
		return
	}

	// Resolve agent
	ag, err := h.agentRepo.GetAgentByKeycloakID(r.Context(), userCtx.KeycloakID)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	// Find the pending offer for this job+agent
	offer, err := h.jobRepo.GetOfferByJobAndAgent(r.Context(), jobID, ag.ID)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	// Update offer status to accepted
	accepted := "accepted"
	if err := h.jobRepo.UpdateOfferStatus(r.Context(), offer.ID, accepted, nil); err != nil {
		platform.HandleError(w, err)
		return
	}

	// Publish response to Redis so the dispatcher goroutine picks it up
	channel := fmt.Sprintf("offer:%s:response", offer.ID)
	h.rdb.Publish(r.Context(), channel, "accepted")

	h.logger.Info("agent accepted offer",
		"agent_id", ag.ID,
		"job_id", jobID,
		"offer_id", offer.ID,
	)

	// Notify agent via WebSocket
	h.publishAgentEvent(r.Context(), ag.ID, "job.accepted", map[string]string{
		"job_id": jobID.String(),
	})

	// Return the job details
	job, err := h.jobRepo.GetJobByID(r.Context(), jobID)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	platform.JSON(w, http.StatusOK, JobResponseFromSqlc(
		job.ID, job.ParcelID, job.UserID, job.SurveyType, job.Priority,
		job.Deadline, job.Status, job.AssignedAgentID, job.AssignedAt, job.CreatedAt,
	))
}

// DeclineOffer handles POST /v1/jobs/{id}/decline.
func (h *Handler) DeclineOffer(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.HandleError(w, platform.NewBadRequest("invalid job ID"))
		return
	}

	var req DeclineRequest
	if err := platform.Decode(r, &req); err != nil {
		// Decline reason is optional, so allow empty body
		req = DeclineRequest{}
	}

	// Resolve agent
	ag, err := h.agentRepo.GetAgentByKeycloakID(r.Context(), userCtx.KeycloakID)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	// Find the pending offer
	offer, err := h.jobRepo.GetOfferByJobAndAgent(r.Context(), jobID, ag.ID)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	// Update offer status to declined
	declined := "declined"
	var reason *string
	if req.Reason != "" {
		reason = &req.Reason
	}
	if err := h.jobRepo.UpdateOfferStatus(r.Context(), offer.ID, declined, reason); err != nil {
		platform.HandleError(w, err)
		return
	}

	// Publish response to Redis
	channel := fmt.Sprintf("offer:%s:response", offer.ID)
	h.rdb.Publish(r.Context(), channel, "declined")

	h.logger.Info("agent declined offer",
		"agent_id", ag.ID,
		"job_id", jobID,
		"offer_id", offer.ID,
		"reason", req.Reason,
	)

	platform.JSON(w, http.StatusOK, map[string]string{"message": "offer declined"})
}

// GetJob handles GET /v1/jobs/{id}.
func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.HandleError(w, platform.NewBadRequest("invalid job ID"))
		return
	}

	job, err := h.jobRepo.GetJobByID(r.Context(), jobID)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	resp := JobResponseFromSqlc(
		job.ID, job.ParcelID, job.UserID, job.SurveyType, job.Priority,
		job.Deadline, job.Status, job.AssignedAgentID, job.AssignedAt, job.CreatedAt,
	)

	// Enrich with parcel data for navigation
	parcel, err := h.queries.GetParcelWithGeoJSON(r.Context(), job.ParcelID)
	if err == nil {
		resp.Parcel = &ParcelEmbed{
			ID:              parcel.ID,
			Label:           parcel.Label,
			Village:         parcel.Village,
			Taluk:           parcel.Taluk,
			District:        parcel.District,
			State:           parcel.State,
			BoundaryGeoJSON: parcel.BoundaryGeojson,
			AreaSqm:         parcel.AreaSqm,
		}
	}

	platform.JSON(w, http.StatusOK, resp)
}

// RequestSurvey handles POST /v1/parcels/{parcelId}/request-survey.
// Landowner explicitly triggers a survey for their parcel.
func (h *Handler) RequestSurvey(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	parcelID, err := uuid.Parse(chi.URLParam(r, "parcelId"))
	if err != nil {
		platform.HandleError(w, platform.NewBadRequest("invalid parcel ID"))
		return
	}

	// Resolve keycloak_id to local user
	user, err := h.queries.GetUserByKeycloakID(r.Context(), &userCtx.KeycloakID)
	if err != nil {
		platform.HandleError(w, platform.NewNotFound("user not found"))
		return
	}

	// Verify parcel exists and belongs to this user
	parcel, err := h.queries.GetParcelByID(r.Context(), parcelID)
	if err != nil {
		platform.HandleError(w, platform.NewNotFound("parcel not found"))
		return
	}
	if parcel.UserID != user.ID {
		platform.JSONError(w, http.StatusForbidden, "FORBIDDEN", "not your parcel")
		return
	}

	// Check if there's already an active job for this parcel
	existingJobs, err := h.jobRepo.GetActiveJobsByParcel(r.Context(), parcelID)
	if err == nil && len(existingJobs) > 0 {
		platform.JSONError(w, http.StatusConflict, "CONFLICT", "a survey is already in progress for this parcel")
		return
	}

	// Create job and dispatch
	job, err := h.scheduler.CreateJobForParcel(r.Context(), parcel)
	if err != nil {
		h.logger.Error("failed to create survey job", "parcel_id", parcelID, "error", err)
		platform.HandleError(w, err)
		return
	}

	platform.JSON(w, http.StatusCreated, JobResponseFromSqlc(
		job.ID, job.ParcelID, job.UserID, job.SurveyType, job.Priority,
		job.Deadline, job.Status, job.AssignedAgentID, job.AssignedAt, job.CreatedAt,
	))
}

// ListAgentJobs handles GET /v1/agents/me/jobs.
func (h *Handler) ListAgentJobs(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	ag, err := h.agentRepo.GetAgentByKeycloakID(r.Context(), userCtx.KeycloakID)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	pg := platform.ParsePagination(r)
	jobs, err := h.jobRepo.ListJobsByAgent(r.Context(), ag.ID, int32(pg.PerPage), int32(pg.Offset))
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	result := make([]JobResponse, len(jobs))
	for i, j := range jobs {
		result[i] = JobResponseFromSqlc(
			j.ID, j.ParcelID, j.UserID, j.SurveyType, j.Priority,
			j.Deadline, j.Status, j.AssignedAgentID, j.AssignedAt, j.CreatedAt,
		)
	}

	platform.JSON(w, http.StatusOK, result)
}

// ListAgentOffers handles GET /v1/agents/me/offers.
func (h *Handler) ListAgentOffers(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	ag, err := h.agentRepo.GetAgentByKeycloakID(r.Context(), userCtx.KeycloakID)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	offers, err := h.jobRepo.ListPendingOffersByAgent(r.Context(), ag.ID)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	result := make([]OfferResponse, len(offers))
	for i, o := range offers {
		resp := OfferResponse{
			ID:           o.ID,
			JobID:        o.JobID,
			AgentID:      o.AgentID,
			CascadeRound: o.CascadeRound,
			OfferRank:    o.OfferRank,
			DistanceKm:   o.DistanceKm,
			Status:       o.Status,
			ExpiresAt:    o.ExpiresAt,
		}
		if o.SentAt.Valid {
			t := o.SentAt.Time
			resp.SentAt = &t
		}
		result[i] = resp
	}

	platform.JSON(w, http.StatusOK, result)
}

// Arrive handles POST /v1/jobs/{id}/arrive.
// Validates geofence (agent must be within 500m of parcel centroid).
func (h *Handler) Arrive(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.HandleError(w, platform.NewBadRequest("invalid job ID"))
		return
	}

	var req ArriveRequest
	if err := platform.Decode(r, &req); err != nil {
		platform.HandleError(w, err)
		return
	}

	if req.Lat == 0 && req.Lng == 0 {
		platform.HandleError(w, platform.NewBadRequest("lat and lng are required"))
		return
	}

	// Resolve agent
	ag, err := h.agentRepo.GetAgentByKeycloakID(r.Context(), userCtx.KeycloakID)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	// Verify job exists and is assigned to this agent
	job, err := h.jobRepo.GetJobByID(r.Context(), jobID)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	if !job.AssignedAgentID.Valid || uuid.UUID(job.AssignedAgentID.Bytes) != ag.ID {
		platform.HandleError(w, platform.NewForbidden("job not assigned to you"))
		return
	}

	if job.Status == nil || *job.Status != "assigned" {
		platform.HandleError(w, platform.NewBadRequest("job is not in assigned status"))
		return
	}

	// Geofence check: distance from agent to parcel centroid
	distM, err := h.jobRepo.DistanceToParcelCentroid(r.Context(), job.ParcelID, req.Lng, req.Lat)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	const maxDistanceM = 500.0
	if distM > maxDistanceM {
		platform.HandleError(w, platform.NewBadRequest(
			fmt.Sprintf("too far from parcel (%.0fm away, max %0.fm)", distM, maxDistanceM),
		))
		return
	}

	arrivalDist := float32(math.Round(distM*100) / 100)
	err = h.jobRepo.RecordAgentArrival(r.Context(), sqlc.RecordAgentArrivalParams{
		ID:               jobID,
		StMakepoint:      req.Lng,
		StMakepoint_2:    req.Lat,
		ArrivalDistanceM: &arrivalDist,
	})
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	h.logger.Info("agent arrived at parcel",
		"agent_id", ag.ID,
		"job_id", jobID,
		"distance_m", distM,
	)

	// Notify agent via WebSocket
	h.publishAgentEvent(r.Context(), ag.ID, "job.arrived", map[string]string{
		"job_id": jobID.String(),
	})

	platform.JSON(w, http.StatusOK, map[string]any{
		"message":    "arrival confirmed",
		"distance_m": math.Round(distM*100) / 100,
	})
}

// PresignedURL handles GET /v1/jobs/{id}/media/presigned.
// Query params: content_type, step_id.
func (h *Handler) PresignedURL(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.HandleError(w, platform.NewBadRequest("invalid job ID"))
		return
	}

	contentType := r.URL.Query().Get("content_type")
	stepID := r.URL.Query().Get("step_id")
	if contentType == "" || stepID == "" {
		platform.HandleError(w, platform.NewBadRequest("content_type and step_id query params are required"))
		return
	}

	// Determine file extension from content type
	ext := extensionFromContentType(contentType)

	// Build S3 key: media/{job_id}/{step_id}/{uuid}.{ext}
	s3Key := fmt.Sprintf("media/%s/%s/%s.%s", jobID, stepID, uuid.New(), ext)

	ttl := 15 * time.Minute
	url, err := h.s3Client.GeneratePresignedPutURL(r.Context(), s3Key, contentType, ttl)
	if err != nil {
		h.logger.Error("failed to generate presigned URL", "error", err)
		platform.HandleError(w, platform.NewInternal("failed to generate upload URL", err))
		return
	}

	platform.JSON(w, http.StatusOK, PresignedURLResponse{
		UploadURL: url,
		S3Key:     s3Key,
		ExpiresIn: int(ttl.Seconds()),
	})
}

// UploadMedia handles POST /v1/jobs/{id}/media/upload.
// Accepts multipart form data, uploads to S3 server-side, and records metadata.
// This proxies the upload through the API to avoid mobile clients needing direct S3 access.
func (h *Handler) UploadMedia(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.HandleError(w, platform.NewBadRequest("invalid job ID"))
		return
	}

	// 32 MB max
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		platform.HandleError(w, platform.NewBadRequest("invalid multipart form"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		platform.HandleError(w, platform.NewBadRequest("file field is required"))
		return
	}
	defer file.Close()

	stepID := r.FormValue("step_id")
	mediaType := r.FormValue("media_type")
	latStr := r.FormValue("lat")
	lngStr := r.FormValue("lng")
	capturedAtStr := r.FormValue("captured_at")

	if stepID == "" || mediaType == "" {
		platform.HandleError(w, platform.NewBadRequest("step_id and media_type are required"))
		return
	}

	lat, _ := strconv.ParseFloat(latStr, 64)
	lng, _ := strconv.ParseFloat(lngStr, 64)

	capturedAt := time.Now()
	if capturedAtStr != "" {
		if t, err := time.Parse(time.RFC3339, capturedAtStr); err == nil {
			capturedAt = t
		}
	}

	// Read file into memory for hashing and upload
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		platform.HandleError(w, platform.NewInternal("failed to read uploaded file", err))
		return
	}

	// Compute SHA-256
	hash := sha256.Sum256(fileBytes)
	fileSHA256 := hex.EncodeToString(hash[:])

	// Determine content type and extension
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	ext := extensionFromContentType(contentType)

	// Build S3 key
	s3Key := fmt.Sprintf("media/%s/%s/%s.%s", jobID, stepID, uuid.New(), ext)

	// Upload to S3
	if err := h.s3Client.PutObject(r.Context(), s3Key, contentType, bytes.NewReader(fileBytes)); err != nil {
		h.logger.Error("failed to upload media to S3", "error", err)
		platform.HandleError(w, platform.NewInternal("failed to upload file", err))
		return
	}

	// Resolve agent
	ag, err := h.agentRepo.GetAgentByKeycloakID(r.Context(), userCtx.KeycloakID)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	fileSize := int64(len(fileBytes))
	media, err := h.surveyRepo.CreateSurveyMedia(r.Context(), sqlc.CreateSurveyMediaParams{
		JobID:          jobID,
		AgentID:        ag.ID,
		StepID:         stepID,
		MediaType:      mediaType,
		S3Key:          s3Key,
		FileSizeBytes:  &fileSize,
		StMakepoint:    lng,
		StMakepoint_2:  lat,
		CapturedAt:     capturedAt,
		FileHashSha256: fileSHA256,
	})
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	h.logger.Info("media uploaded via proxy",
		"agent_id", ag.ID,
		"job_id", jobID,
		"media_id", media.ID,
		"step_id", stepID,
		"size_bytes", fileSize,
	)

	platform.JSON(w, http.StatusCreated, MediaResponse{
		ID:        media.ID,
		S3Key:     media.S3Key,
		StepID:    media.StepID,
		MediaType: media.MediaType,
		UploadedAt: func() time.Time {
			if media.UploadedAt.Valid {
				return media.UploadedAt.Time
			}
			return time.Now()
		}(),
	})
}

// RecordMedia handles POST /v1/jobs/{id}/media.
// Records media metadata after a successful S3 upload.
func (h *Handler) RecordMedia(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.HandleError(w, platform.NewBadRequest("invalid job ID"))
		return
	}

	var req MediaRequest
	if err := platform.Decode(r, &req); err != nil {
		platform.HandleError(w, err)
		return
	}

	if req.S3Key == "" || req.StepID == "" || req.MediaType == "" || req.SHA256 == "" {
		platform.HandleError(w, platform.NewBadRequest("s3_key, step_id, media_type, and sha256 are required"))
		return
	}

	// Resolve agent
	ag, err := h.agentRepo.GetAgentByKeycloakID(r.Context(), userCtx.KeycloakID)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	media, err := h.surveyRepo.CreateSurveyMedia(r.Context(), sqlc.CreateSurveyMediaParams{
		JobID:          jobID,
		AgentID:        ag.ID,
		StepID:         req.StepID,
		MediaType:      req.MediaType,
		S3Key:          req.S3Key,
		FileSizeBytes:  req.FileSize,
		StMakepoint:    req.Lng,
		StMakepoint_2:  req.Lat,
		CapturedAt:     req.CapturedAt,
		FileHashSha256: req.SHA256,
	})
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	h.logger.Info("media metadata recorded",
		"agent_id", ag.ID,
		"job_id", jobID,
		"media_id", media.ID,
		"step_id", req.StepID,
	)

	platform.JSON(w, http.StatusCreated, MediaResponse{
		ID:        media.ID,
		S3Key:     media.S3Key,
		StepID:    media.StepID,
		MediaType: media.MediaType,
		UploadedAt: func() time.Time {
			if media.UploadedAt.Valid {
				return media.UploadedAt.Time
			}
			return time.Now()
		}(),
	})
}

// SubmitSurvey handles POST /v1/jobs/{id}/survey.
// Submits the survey response and updates job status to survey_submitted.
func (h *Handler) SubmitSurvey(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.HandleError(w, platform.NewBadRequest("invalid job ID"))
		return
	}

	var req SurveySubmitRequest
	if err := platform.Decode(r, &req); err != nil {
		platform.HandleError(w, err)
		return
	}

	if req.Responses == nil {
		platform.HandleError(w, platform.NewBadRequest("responses field is required"))
		return
	}

	// Resolve agent
	ag, err := h.agentRepo.GetAgentByKeycloakID(r.Context(), userCtx.KeycloakID)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	// Verify job is assigned to this agent
	job, err := h.jobRepo.GetJobByID(r.Context(), jobID)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	if !job.AssignedAgentID.Valid || uuid.UUID(job.AssignedAgentID.Bytes) != ag.ID {
		platform.HandleError(w, platform.NewForbidden("job not assigned to you"))
		return
	}

	// Build params
	params := sqlc.CreateSurveyResponseParams{
		JobID:     jobID,
		AgentID:   ag.ID,
		Responses: req.Responses,
	}

	if req.TemplateID != nil {
		params.TemplateID = pgtype.UUID{Bytes: *req.TemplateID, Valid: true}
	}

	if req.GPSTrailGeoJSON != "" {
		params.StGeomfromgeojson = req.GPSTrailGeoJSON
	}

	if req.DeviceInfo != nil {
		params.DeviceInfo = req.DeviceInfo
	}

	if req.StartedAt != nil {
		params.StartedAt = pgtype.Timestamptz{Time: *req.StartedAt, Valid: true}
	}

	params.DurationMinutes = req.DurationMinutes

	surveyResp, err := h.surveyRepo.CreateSurveyResponse(r.Context(), params)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	// Update job status to survey_submitted
	_, err = h.jobRepo.UpdateJobStatus(r.Context(), jobID, "survey_submitted")
	if err != nil {
		h.logger.Error("failed to update job status after survey submit",
			"job_id", jobID,
			"error", err,
		)
	}

	// Publish event to trigger QA pipeline
	h.eventBus.Publish(platform.Event{
		Type: "survey.submitted",
		Payload: map[string]string{
			"job_id":    jobID.String(),
			"parcel_id": job.ParcelID.String(),
			"user_id":   job.UserID.String(),
		},
	})

	h.logger.Info("survey submitted",
		"agent_id", ag.ID,
		"job_id", jobID,
		"survey_response_id", surveyResp.ID,
	)

	// Notify agent via WebSocket
	h.publishAgentEvent(r.Context(), ag.ID, "job.survey_submitted", map[string]string{
		"job_id": jobID.String(),
	})

	platform.JSON(w, http.StatusOK, map[string]any{
		"message":            "survey submitted",
		"survey_response_id": surveyResp.ID,
	})
}

// GetTemplate handles GET /v1/jobs/{id}/template.
// Returns the active checklist template for the job's survey type.
func (h *Handler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	jobID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.HandleError(w, platform.NewBadRequest("invalid job ID"))
		return
	}

	job, err := h.jobRepo.GetJobByID(r.Context(), jobID)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	tmpl, err := h.surveyRepo.GetActiveTemplate(r.Context(), job.SurveyType)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	platform.JSON(w, http.StatusOK, TemplateResponse{
		ID:         tmpl.ID,
		Name:       tmpl.Name,
		SurveyType: tmpl.SurveyType,
		Version:    tmpl.Version,
		Steps:      tmpl.Steps,
	})
}

// ListParcelSurveys handles GET /v1/parcels/{parcelId}/surveys.
// Returns jobs with survey/QA info for the landowner.
func (h *Handler) ListParcelSurveys(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	parcelID, err := uuid.Parse(chi.URLParam(r, "parcelId"))
	if err != nil {
		platform.HandleError(w, platform.NewBadRequest("invalid parcel ID"))
		return
	}

	// Verify ownership
	user, err := h.queries.GetUserByKeycloakID(r.Context(), &userCtx.KeycloakID)
	if err != nil {
		platform.HandleError(w, platform.NewNotFound("user not found"))
		return
	}
	parcel, err := h.queries.GetParcelByID(r.Context(), parcelID)
	if err != nil {
		platform.HandleError(w, platform.NewNotFound("parcel not found"))
		return
	}
	if parcel.UserID != user.ID {
		platform.JSONError(w, http.StatusForbidden, "FORBIDDEN", "not your parcel")
		return
	}

	jobs, err := h.jobRepo.ListJobsByParcel(r.Context(), parcelID, 50, 0)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	var results []ParcelSurveyResponse
	for _, j := range jobs {
		resp := ParcelSurveyResponse{
			ID:         j.ID,
			SurveyType: j.SurveyType,
			Status:     j.Status,
		}
		if j.QaScore.Valid {
			f, _ := j.QaScore.Float64Value()
			if f.Valid {
				resp.QAScore = &f.Float64
			}
		}
		resp.QAStatus = j.QaStatus
		resp.QANotes = j.QaNotes
		if j.CreatedAt.Valid {
			resp.CreatedAt = j.CreatedAt.Time
		}
		if j.CompletedAt.Valid {
			t := j.CompletedAt.Time
			resp.CompletedAt = &t
		}

		// Fetch survey responses if submitted
		sr, err := h.surveyRepo.GetSurveyResponseByJob(r.Context(), j.ID)
		if err == nil {
			resp.Responses = sr.Responses
		}

		results = append(results, resp)
	}

	platform.JSON(w, http.StatusOK, results)
}

// extensionFromContentType maps common content types to file extensions.
func extensionFromContentType(ct string) string {
	switch strings.ToLower(ct) {
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/webp":
		return "webp"
	case "video/mp4":
		return "mp4"
	case "video/quicktime":
		return "mov"
	case "audio/aac":
		return "aac"
	case "audio/mpeg":
		return "mp3"
	default:
		// Try to extract from content type
		parts := strings.Split(ct, "/")
		if len(parts) == 2 {
			return filepath.Base(parts[1])
		}
		return "bin"
	}
}
