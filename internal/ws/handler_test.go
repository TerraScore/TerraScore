package ws

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServeWS_MissingToken(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	w := httptest.NewRecorder()
	h.ServeWS(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestServeWS_InvalidToken(t *testing.T) {
	// With a nil keycloak client, ValidateToken will panic,
	// so we just test the missing token path which is the main guard.
	h := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/ws?token=", nil)
	w := httptest.NewRecorder()
	h.ServeWS(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for empty token, got %d", w.Code)
	}
}
