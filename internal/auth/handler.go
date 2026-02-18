package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/terrascore/api/internal/platform"
)

// Handler handles auth HTTP endpoints.
type Handler struct {
	service *Service
}

// NewHandler creates an auth handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Routes returns the auth router.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/verify-otp", h.VerifyOTP)
	r.Post("/refresh", h.Refresh)
	return r
}

// Register handles user registration.
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

// Login handles login (sends OTP to existing user).
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := platform.Decode(r, &req); err != nil {
		platform.HandleError(w, err)
		return
	}

	resp, err := h.service.Login(r.Context(), req)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	platform.JSON(w, http.StatusOK, resp)
}

// VerifyOTP handles OTP verification and returns tokens.
func (h *Handler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req VerifyOTPRequest
	if err := platform.Decode(r, &req); err != nil {
		platform.HandleError(w, err)
		return
	}

	resp, err := h.service.VerifyOTP(r.Context(), req)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	platform.JSON(w, http.StatusOK, resp)
}

// Refresh handles token refresh.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := platform.Decode(r, &req); err != nil {
		platform.HandleError(w, err)
		return
	}

	resp, err := h.service.Refresh(r.Context(), req)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	platform.JSON(w, http.StatusOK, resp)
}
