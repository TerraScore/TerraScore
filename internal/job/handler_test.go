package job

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/terrascore/api/internal/auth"
	"github.com/terrascore/api/internal/platform"
)

func TestAcceptOffer_NoAuth(t *testing.T) {
	h := &Handler{}
	r := chi.NewRouter()
	r.Post("/jobs/{id}/accept", h.AcceptOffer)

	req := httptest.NewRequest(http.MethodPost, "/jobs/00000000-0000-0000-0000-000000000001/accept", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestDeclineOffer_NoAuth(t *testing.T) {
	h := &Handler{}
	r := chi.NewRouter()
	r.Post("/jobs/{id}/decline", h.DeclineOffer)

	req := httptest.NewRequest(http.MethodPost, "/jobs/00000000-0000-0000-0000-000000000001/decline", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAcceptOffer_InvalidJobID(t *testing.T) {
	h := &Handler{}
	r := chi.NewRouter()
	r.Post("/jobs/{id}/accept", h.AcceptOffer)

	ctx := auth.SetUser(context.Background(), &auth.UserContext{
		KeycloakID: "test-kc-id",
		Roles:      []string{"agent"},
	})

	req := httptest.NewRequest(http.MethodPost, "/jobs/not-a-uuid/accept", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDeclineOffer_InvalidJobID(t *testing.T) {
	h := &Handler{}
	r := chi.NewRouter()
	r.Post("/jobs/{id}/decline", h.DeclineOffer)

	ctx := auth.SetUser(context.Background(), &auth.UserContext{
		KeycloakID: "test-kc-id",
		Roles:      []string{"agent"},
	})

	req := httptest.NewRequest(http.MethodPost, "/jobs/not-a-uuid/decline", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestGetJob_NoAuth(t *testing.T) {
	h := &Handler{}
	r := chi.NewRouter()
	r.Get("/jobs/{id}", h.GetJob)

	req := httptest.NewRequest(http.MethodGet, "/jobs/00000000-0000-0000-0000-000000000001", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestListAgentJobs_NoAuth(t *testing.T) {
	h := &Handler{}
	r := chi.NewRouter()
	r.Get("/agents/me/jobs", h.ListAgentJobs)

	req := httptest.NewRequest(http.MethodGet, "/agents/me/jobs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestListAgentOffers_NoAuth(t *testing.T) {
	h := &Handler{}
	r := chi.NewRouter()
	r.Get("/agents/me/offers", h.ListAgentOffers)

	req := httptest.NewRequest(http.MethodGet, "/agents/me/offers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestDeclineRequest_JSON(t *testing.T) {
	body := DeclineRequest{Reason: "too far away"}
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	var decoded DeclineRequest
	if err := json.NewDecoder(bytes.NewReader(b)).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Reason != "too far away" {
		t.Errorf("expected reason 'too far away', got %q", decoded.Reason)
	}
}

func TestJobResponse_JSON(t *testing.T) {
	resp := JobResponse{
		SurveyType: "basic_check",
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}

	var envelope platform.Response
	if err := json.Unmarshal(b, &envelope); err != nil {
		// Direct marshal (not wrapped in envelope), just verify it's valid JSON
		var raw map[string]interface{}
		if err := json.Unmarshal(b, &raw); err != nil {
			t.Fatal("invalid JSON output")
		}
	}
}
