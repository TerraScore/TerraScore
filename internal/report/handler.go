package report

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/terrascore/api/internal/auth"
	"github.com/terrascore/api/internal/platform"
)

// Handler handles report HTTP endpoints.
type Handler struct {
	repo    *Repository
	service *Service
}

// NewHandler creates a report handler.
func NewHandler(repo *Repository, service *Service) *Handler {
	return &Handler{repo: repo, service: service}
}

// ListByParcel handles GET /v1/parcels/{parcelId}/reports.
func (h *Handler) ListByParcel(w http.ResponseWriter, r *http.Request) {
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

	pg := platform.ParsePagination(r)
	reports, err := h.repo.ListByParcel(r.Context(), parcelID, int32(pg.PerPage), int32(pg.Offset))
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	result := make([]ReportResponse, len(reports))
	for i, rpt := range reports {
		result[i] = ReportResponse{
			ID:          rpt.ID.String(),
			ParcelID:    rpt.ParcelID.String(),
			JobID:       rpt.JobID.String(),
			ReportType:  rpt.ReportType,
			Format:      rpt.Format,
			GeneratedAt: rpt.GeneratedAt,
		}
	}

	platform.JSON(w, http.StatusOK, result)
}

// Download handles GET /v1/reports/{id}/download.
func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	reportID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.HandleError(w, platform.NewBadRequest("invalid report ID"))
		return
	}

	rpt, err := h.repo.GetByID(r.Context(), reportID)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	url, err := h.service.GetDownloadURL(r.Context(), rpt.S3Key)
	if err != nil {
		platform.HandleError(w, platform.NewInternal("failed to generate download URL", err))
		return
	}

	platform.JSON(w, http.StatusOK, DownloadResponse{DownloadURL: url})
}
