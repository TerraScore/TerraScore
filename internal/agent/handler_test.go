package agent_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/terrascore/api/internal/agent"
)

func newTestHandler() *agent.Handler {
	svc := agent.NewServiceForTest()
	return agent.NewHandler(svc)
}

func TestRegisterValidation(t *testing.T) {
	handler := newTestHandler()
	router := handler.Routes()

	tests := []struct {
		name       string
		body       map[string]string
		wantStatus int
	}{
		{
			name:       "empty body",
			body:       map[string]string{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "missing phone",
			body:       map[string]string{"full_name": "Test Agent"},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "missing full_name",
			body:       map[string]string{"phone": "+919876543210"},
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}
