package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/st1lson/glitch/internal/config"
	"github.com/st1lson/glitch/internal/control"
	"github.com/st1lson/glitch/internal/logging"
)

type dummyReporter struct{}

func (d *dummyReporter) Report(event logging.LogEvent) {}

func TestNewRouter(t *testing.T) {
	state := config.NewManager(config.DefaultConfig())
	gate := control.NewGatekeeper()
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("api response"))
	})

	reporter := &dummyReporter{}
	router := NewRouter(state, gate, apiHandler, reporter)

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

		if origin := rec.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
			t.Errorf("expected CORS origin '*', got %q", origin)
		}
	})

	t.Run("CORS Preflight", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		req.Header.Set("Access-Control-Request-Headers", "X-Glitch-Scenario")
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
		if headers := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(headers, "X-Glitch-Scenario") {
			t.Errorf("expected scenario header to be allowed, got %q", headers)
		}
	})

	t.Run("CORS Scenario Header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		req.Header.Set("X-Glitch-Scenario", "browser-test")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %v", rec.Code)
		}
		if scenario := rec.Header().Get("X-Glitch-Scenario"); scenario != "browser-test" {
			t.Errorf("expected scenario header to be echoed, got %q", scenario)
		}
		if headers := rec.Header().Get("Access-Control-Expose-Headers"); !strings.Contains(headers, "X-Glitch-Scenario") {
			t.Errorf("expected scenario header to be exposed, got %q", headers)
		}
	})

	t.Run("No Scenario Header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if scenario := rec.Header().Get("X-Glitch-Scenario"); scenario != "" {
			t.Errorf("expected no scenario response header, got %q", scenario)
		}
	})
}
