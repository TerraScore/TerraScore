package notification

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/terrascore/api/internal/auth"
	"github.com/terrascore/api/internal/platform"
)

// Handler handles notification/alert HTTP endpoints.
type Handler struct {
	repo     *Repository
	authRepo *auth.Repository
}

// NewHandler creates a notification handler.
func NewHandler(repo *Repository, authRepo *auth.Repository) *Handler {
	return &Handler{repo: repo, authRepo: authRepo}
}

// Routes returns the alerts router.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.ListAlerts)
	r.Get("/unread/count", h.UnreadCount)
	r.Put("/{id}/read", h.MarkRead)
	r.Put("/read-all", h.MarkAllRead)
	return r
}

// ListAlerts handles GET /v1/alerts.
func (h *Handler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	user, err := h.authRepo.GetUserByKeycloakID(r.Context(), userCtx.KeycloakID)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	pg := platform.ParsePagination(r)
	alerts, err := h.repo.ListAlerts(r.Context(), user.ID, int32(pg.PerPage), int32(pg.Offset))
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	result := make([]AlertResponse, len(alerts))
	for i, a := range alerts {
		result[i] = AlertResponse{
			ID:    a.ID.String(),
			Type:  a.Type,
			Title: a.Title,
			Body:  a.Body,
			IsRead: func() bool {
				if a.IsRead != nil {
					return *a.IsRead
				}
				return false
			}(),
			CreatedAt: a.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		}
		if len(a.Data) > 0 {
			var data map[string]string
			if err := json.Unmarshal(a.Data, &data); err == nil {
				result[i].Data = data
			}
		}
	}

	platform.JSON(w, http.StatusOK, result)
}

// UnreadCount handles GET /v1/alerts/unread/count.
func (h *Handler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	user, err := h.authRepo.GetUserByKeycloakID(r.Context(), userCtx.KeycloakID)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	count, err := h.repo.CountUnread(r.Context(), user.ID)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	platform.JSON(w, http.StatusOK, map[string]int64{"unread_count": count})
}

// MarkRead handles PUT /v1/alerts/{id}/read.
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	alertID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		platform.HandleError(w, platform.NewBadRequest("invalid alert ID"))
		return
	}

	if err := h.repo.MarkRead(r.Context(), alertID); err != nil {
		platform.HandleError(w, err)
		return
	}

	platform.JSON(w, http.StatusOK, map[string]string{"message": "alert marked as read"})
}

// MarkAllRead handles PUT /v1/alerts/read-all.
func (h *Handler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	userCtx := auth.GetUser(r.Context())
	if userCtx == nil {
		platform.JSONError(w, http.StatusUnauthorized, platform.CodeUnauthorized, "not authenticated")
		return
	}

	user, err := h.authRepo.GetUserByKeycloakID(r.Context(), userCtx.KeycloakID)
	if err != nil {
		platform.HandleError(w, err)
		return
	}

	if err := h.repo.MarkAllRead(r.Context(), user.ID); err != nil {
		platform.HandleError(w, err)
		return
	}

	platform.JSON(w, http.StatusOK, map[string]string{"message": "all alerts marked as read"})
}
