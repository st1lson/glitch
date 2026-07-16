package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/st1lson/glitch/internal/config"
	"github.com/st1lson/glitch/internal/logging"
)

type dummyReporter struct{}

func (d *dummyReporter) Report(event logging.LogEvent) {}

func TestNewRouter(t *testing.T) {
	state := config.NewState(config.DefaultConfig())
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("api response"))
	})

	reporter := &dummyReporter{}
	router := NewRouter(state, apiHandler, reporter)

	t.Run("Normal Request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %v", rec.Code)
		}
		if rec.Body.String() != "api response" {
			t.Errorf("expected 'api response', got %q", rec.Body.String())
		}

		// Check CORS headers on normal request
		if origin := rec.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
			t.Errorf("expected CORS origin '*', got %q", origin)
		}
	})

	t.Run("CORS Preflight", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("expected 204 No Content for OPTIONS, got %v", rec.Code)
		}

		if origin := rec.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
			t.Errorf("expected CORS origin '*', got %q", origin)
		}
		if methods := rec.Header().Get("Access-Control-Allow-Methods"); methods == "" {
			t.Errorf("expected Access-Control-Allow-Methods header to be set")
		}
	})
}
