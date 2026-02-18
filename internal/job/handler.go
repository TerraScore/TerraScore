package job

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/terrascore/api/internal/agent"
	"github.com/terrascore/api/internal/auth"
	"github.com/terrascore/api/internal/platform"
)

// Handler handles job HTTP endpoints.
type Handler struct {
	jobRepo   *Repository
	agentRepo *agent.Repository
	rdb       *redis.Client
	logger    *slog.Logger
}

// NewHandler creates a job handler.
func NewHandler(jobRepo *Repository, agentRepo *agent.Repository, rdb *redis.Client, logger *slog.Logger) *Handler {
	return &Handler{
		jobRepo:   jobRepo,
		agentRepo: agentRepo,
		rdb:       rdb,
		logger:    logger,
	}
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

	platform.JSON(w, http.StatusOK, JobResponseFromSqlc(
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
