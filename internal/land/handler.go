package land

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/terrascore/api/internal/auth"
	"github.com/terrascore/api/internal/platform"
)

// Handler handles land HTTP endpoints.
type Handler struct {
	service *Service
}

// NewHandler creates a land handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Routes returns the land router.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(auth.RequireRole("landowner"))
	r.Post("/", h.CreateParcel)
	r.Get("/", h.ListParcels)
	r.Get("/{id}", h.GetParcel)
	r.Put("/{id}/boundary", h.UpdateBoundary)
	r.Delete("/{id}", h.DeleteParcel)
	return r
}

// CreateParcel handles POST /v1/parcels.
func (h *Handler) CreateParcel(w http.ResponseWriter, r *http.Request) {
	var req CreateParcelRequest
	if err := platform.Decode(r, &req); err != nil {
		platform.HandleError(w, err)
		return
	}

	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	resp, err := h.service.CreateParcel(r.Context(), userCtx, req)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	platform.JSON(w, http.StatusCreated, resp)
}

// ListParcels handles GET /v1/parcels.
func (h *Handler) ListParcels(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	pg := platform.ParsePagination(r)

	parcels, total, err := h.service.ListParcels(r.Context(), userCtx, pg.Page, pg.PerPage)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	totalPages := int(total) / pg.PerPage
	if int(total)%pg.PerPage != 0 {
		totalPages++
	}

	platform.JSONList(w, http.StatusOK, parcels, platform.Meta{
		Page:       pg.Page,
		PerPage:    pg.PerPage,
		Total:      int(total),
		TotalPages: totalPages,
	})
}

// GetParcel handles GET /v1/parcels/{id}.
func (h *Handler) GetParcel(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.HandleError(w, platform.NewBadRequest("invalid parcel ID"))
		return
	}

	resp, err := h.service.GetParcel(r.Context(), userCtx, id)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	platform.JSON(w, http.StatusOK, resp)
}

// UpdateBoundary handles PUT /v1/parcels/{id}/boundary.
func (h *Handler) UpdateBoundary(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.HandleError(w, platform.NewBadRequest("invalid parcel ID"))
		return
	}

	var req UpdateBoundaryRequest
	if err := platform.Decode(r, &req); err != nil {
		platform.HandleError(w, err)
		return
	}

	if err := h.service.UpdateBoundary(r.Context(), userCtx, id, req); err != nil {
		platform.HandleError(w, err)
		return
	}

	platform.JSON(w, http.StatusOK, map[string]string{"message": "boundary updated"})
}

// DeleteParcel handles DELETE /v1/parcels/{id}.
func (h *Handler) DeleteParcel(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.HandleError(w, platform.NewBadRequest("invalid parcel ID"))
		return
	}

	if err := h.service.DeleteParcel(r.Context(), userCtx, id); err != nil {
		platform.HandleError(w, err)
		return
	}

	platform.JSON(w, http.StatusOK, map[string]string{"message": "parcel deleted"})
}
