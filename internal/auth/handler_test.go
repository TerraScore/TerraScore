package auth_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/terrascore/api/internal/auth"
)

// newTestHandler creates a handler with a zero-value service for validation tests.
// Validation checks in Service methods return errors before touching any dependencies.
func newTestHandler() *auth.Handler {
	svc := auth.NewServiceForTest()
	return auth.NewHandler(svc)
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
			body:       map[string]string{"full_name": "Test User"},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "missing full_name",
			body:       map[string]string{"phone": "+919876543210"},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "invalid role",
			body:       map[string]string{"phone": "+919876543210", "full_name": "Test", "role": "superadmin"},
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

func TestLoginValidation(t *testing.T) {
	handler := newTestHandler()
	router := handler.Routes()

	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("got status %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func TestVerifyOTPValidation(t *testing.T) {
	handler := newTestHandler()
	router := handler.Routes()

	body, _ := json.Marshal(map[string]string{"phone": "+919876543210"})
	req := httptest.NewRequest(http.MethodPost, "/verify-otp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("got status %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func TestRefreshValidation(t *testing.T) {
	handler := newTestHandler()
	router := handler.Routes()

	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("got status %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}
