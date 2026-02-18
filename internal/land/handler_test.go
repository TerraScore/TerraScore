package land_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/terrascore/api/internal/auth"
	"github.com/terrascore/api/internal/land"
)

func newTestHandler() *land.Handler {
	svc := land.NewServiceForTest()
	return land.NewHandler(svc)
}

// testRouter creates a router that injects a fake landowner user context,
// bypassing JWTAuth + RequireRole so we can test validation logic.
func testRouter() chi.Router {
	handler := newTestHandler()
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := auth.SetUser(r.Context(), &auth.UserContext{
				KeycloakID: "test-kc-id",
				Username:   "+919876543210",
				Roles:      []string{"landowner"},
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Mount("/", handler.Routes())
	return r
}

func TestCreateParcelValidation(t *testing.T) {
	router := testRouter()

	tests := []struct {
		name       string
		body       map[string]any
		wantStatus int
	}{
		{
			name:       "empty body",
			body:       map[string]any{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "missing boundary",
			body: map[string]any{
				"district":   "Bangalore Urban",
				"state":      "Karnataka",
				"state_code": "KA",
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "missing district",
			body: map[string]any{
				"state":      "Karnataka",
				"state_code": "KA",
				"boundary":   `{"type":"Polygon","coordinates":[[[77.0,12.0],[77.1,12.0],[77.1,12.1],[77.0,12.1],[77.0,12.0]]]}`,
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "invalid GeoJSON type",
			body: map[string]any{
				"district":   "Bangalore Urban",
				"state":      "Karnataka",
				"state_code": "KA",
				"boundary":   `{"type":"Point","coordinates":[77.0,12.0]}`,
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "coordinates outside India",
			body: map[string]any{
				"district":   "Test",
				"state":      "Test",
				"state_code": "XX",
				"boundary":   `{"type":"Polygon","coordinates":[[[0.0,0.0],[1.0,0.0],[1.0,1.0],[0.0,1.0],[0.0,0.0]]]}`,
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "too few coordinates",
			body: map[string]any{
				"district":   "Test",
				"state":      "Test",
				"state_code": "XX",
				"boundary":   `{"type":"Polygon","coordinates":[[[77.0,12.0],[77.1,12.0],[77.0,12.0]]]}`,
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}
