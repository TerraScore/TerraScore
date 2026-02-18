package agent

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/terrascore/api/internal/auth"
	"github.com/terrascore/api/internal/platform"
)

// Handler handles agent HTTP endpoints.
type Handler struct {
	service *Service
}

// NewHandler creates an agent handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Routes returns the agent router.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// Public route — registration via phone+OTP
	r.Post("/register", h.Register)

	// Protected routes — agent role required
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireRole("agent"))
		r.Get("/me", h.GetProfile)
		r.Put("/me/profile", h.UpdateProfile)
		r.Post("/me/location", h.UpdateLocation)
		r.Put("/me/availability", h.UpdateAvailability)
		r.Put("/me/fcm-token", h.UpdateFCMToken)
	})

	return r
}

// Register handles POST /v1/agents/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := platform.Decode(r, &req); err != nil {
		platform.HandleError(w, err)
		return
	}

	resp, err := h.service.Register(r.Context(), req)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	platform.JSON(w, http.StatusCreated, resp)
}

// GetProfile handles GET /v1/agents/me.
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	resp, err := h.service.GetProfile(r.Context(), userCtx)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	platform.JSON(w, http.StatusOK, resp)
}

// UpdateProfile handles PUT /v1/agents/me/profile.
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	var req UpdateProfileRequest
	if err := platform.Decode(r, &req); err != nil {
		platform.HandleError(w, err)
		return
	}

	if err := h.service.UpdateProfile(r.Context(), userCtx, req); err != nil {
		platform.HandleError(w, err)
		return
	}

	platform.JSON(w, http.StatusOK, map[string]string{"message": "profile updated"})
}

// UpdateLocation handles POST /v1/agents/me/location.
func (h *Handler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	var req LocationRequest
	if err := platform.Decode(r, &req); err != nil {
		platform.HandleError(w, err)
		return
	}

	if err := h.service.UpdateLocation(r.Context(), userCtx, req); err != nil {
		platform.HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateAvailability handles PUT /v1/agents/me/availability.
func (h *Handler) UpdateAvailability(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	var req AvailabilityRequest
	if err := platform.Decode(r, &req); err != nil {
		platform.HandleError(w, err)
		return
	}

	if err := h.service.UpdateAvailability(r.Context(), userCtx, req); err != nil {
		platform.HandleError(w, err)
		return
	}

	platform.JSON(w, http.StatusOK, map[string]string{"message": "availability updated"})
}

// UpdateFCMToken handles PUT /v1/agents/me/fcm-token.
func (h *Handler) UpdateFCMToken(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	var req FCMTokenRequest
	if err := platform.Decode(r, &req); err != nil {
		platform.HandleError(w, err)
		return
	}

	if err := h.service.UpdateFCMToken(r.Context(), userCtx, req); err != nil {
		platform.HandleError(w, err)
		return
	}

	platform.JSON(w, http.StatusOK, map[string]string{"message": "fcm token updated"})
}
