package platform_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/terrascore/api/internal/platform"
)

func TestRequestID_GeneratesID(t *testing.T) {
	handler := platform.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := platform.GetRequestID(r.Context())
		if id == "" {
			t.Error("expected request ID to be set")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID response header")
	}
}

func TestRequestID_UsesExisting(t *testing.T) {
	handler := platform.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := platform.GetRequestID(r.Context())
		if id != "test-id-123" {
			t.Errorf("expected request ID 'test-id-123', got '%s'", id)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "test-id-123")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Header().Get("X-Request-ID") != "test-id-123" {
		t.Errorf("expected X-Request-ID 'test-id-123', got '%s'", w.Header().Get("X-Request-ID"))
	}
}

func TestRecovery_HandlesPanic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	handler := platform.Recovery(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestLogging_SetsStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	handler := platform.Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}
}
