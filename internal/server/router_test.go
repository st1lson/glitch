package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/st1lson/glitch/internal/chaos"
	"github.com/st1lson/glitch/internal/config"
	"github.com/st1lson/glitch/internal/constants"
	"github.com/st1lson/glitch/internal/control"
	"github.com/st1lson/glitch/internal/logging"
	"github.com/st1lson/glitch/internal/reporting"
)

type dummyReporter struct{}

func (d *dummyReporter) Report(event logging.LogEvent) {}

func TestNewRouter(t *testing.T) {
	state := config.NewManager(config.DefaultConfig())
	gate := control.NewGatekeeper()
	reporter := &dummyReporter{}
	reports := reporting.NewReportManager(state)
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("api response"))
	})

	router := NewRouter(state, gate, apiHandler, reporter, reports)

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
		req.Header.Set("Access-Control-Request-Headers", constants.HeaderScenario)
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
		if headers := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(headers, constants.HeaderScenario) {
			t.Errorf("expected scenario header to be allowed, got %q", headers)
		}
	})

	t.Run("CORS Scenario Header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		req.Header.Set(constants.HeaderScenario, "browser-test")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %v", rec.Code)
		}
		if scenario := rec.Header().Get(constants.HeaderScenario); scenario != "browser-test" {
			t.Errorf("expected scenario header to be echoed, got %q", scenario)
		}
		if headers := rec.Header().Get("Access-Control-Expose-Headers"); !strings.Contains(headers, constants.HeaderScenario) {
			t.Errorf("expected scenario header to be exposed, got %q", headers)
		}
	})

	t.Run("No Scenario Header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if scenario := rec.Header().Get(constants.HeaderScenario); scenario != "" {
			t.Errorf("expected no scenario response header, got %q", scenario)
		}
	})

	t.Run("With Operation Index Option", func(t *testing.T) {
		opIdx := chaos.NewOperationIndex()
		opIdx.Register("getTest", "GET", "/test-op")

		cfg := config.DefaultConfig()
		cfg.Routes = []config.RouteConfig{
			{
				OperationID: "getTest",
				Failure:     &config.FailureConfig{Rate: 100},
			},
		}
		st := config.NewManager(cfg)

		rWithOp := NewRouter(st, gate, apiHandler, reporter, reports, chaos.WithOperationIndex(opIdx))

		req := httptest.NewRequest(http.MethodGet, "/test-op", nil)
		rec := httptest.NewRecorder()
		rWithOp.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500 Internal Server Error via OperationID route, got %d", rec.Code)
		}
	})
}

type captureReporter struct {
	events []logging.LogEvent
}

func (c *captureReporter) Report(event logging.LogEvent) {
	c.events = append(c.events, event)
}

// The request logger wraps the chaos engine, and the engine hands its chain a
// derived request. Unless the logger seeds the ChaosInfo itself, everything the
// engine records is invisible from out there and every chaos metric stays zero.
func TestNewRouter_RecordsInjectedChaos(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Latency = config.LatencyConfig{Fixed: config.DurationFromGo(20 * time.Millisecond)}
	cfg.Failure = config.FailureConfig{Statuses: []config.StatusConfig{{Code: 503, Rate: 100}}}

	state := config.NewManager(cfg)
	reporter := &captureReporter{}
	reports := reporting.NewReportManager(state)
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router := NewRouter(state, control.NewGatekeeper(), apiHandler, logging.MultiReporter{reporter, reports}, reports)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected the injected 503, got %d", rec.Code)
	}
	if len(reporter.events) != 1 {
		t.Fatalf("expected 1 log event, got %d", len(reporter.events))
	}

	event := reporter.events[0]
	if event.ChaosFailure != 503 {
		t.Errorf("expected the injected status on the event, got %d", event.ChaosFailure)
	}
	if event.ChaosLatency < 20*time.Millisecond {
		t.Errorf("expected at least 20ms of injected latency on the event, got %v", event.ChaosLatency)
	}

	report, ok := reports.GetReport(constants.DefaultScenario)
	if !ok {
		t.Fatal("expected a report for the default scenario")
	}
	if report.Metrics.Failures != 1 {
		t.Errorf("expected 1 failure counted, got %d", report.Metrics.Failures)
	}
	if report.Metrics.TotalLatencyAddedMs < 20 {
		t.Errorf("expected at least 20ms of latency counted, got %d", report.Metrics.TotalLatencyAddedMs)
	}
	if len(report.RequestEvents) != 1 || report.RequestEvents[0].ChaosFailureCode != 503 {
		t.Errorf("expected the request event to carry the injected status, got %+v", report.RequestEvents)
	}
}

func TestNewRouter_RecordsCorruptionAndStalls(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Corruption = config.CorruptionConfig{Rate: 100, Strategies: []config.CorruptionStrategy{config.StrategyInjectNull}}
	cfg.Stall = config.StallConfig{Rate: 100, Mode: config.StallModeDrop, DropAt: 100}

	state := config.NewManager(cfg)
	reporter := &captureReporter{}
	reports := reporting.NewReportManager(state)
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"Alice"}`))
	})

	router := NewRouter(state, control.NewGatekeeper(), apiHandler, logging.MultiReporter{reporter, reports}, reports)

	// Stall drop aborts the handler by panicking, which is exactly the case the
	// logger has to survive for the metric to exist.
	func() {
		defer func() { _ = recover() }()
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/test", nil))
	}()

	if len(reporter.events) != 1 {
		t.Fatalf("expected 1 log event, got %d", len(reporter.events))
	}

	event := reporter.events[0]
	if !event.ChaosCorrupted {
		t.Error("expected the event to be marked corrupted")
	}
	if !event.ChaosStalled {
		t.Error("expected the event to be marked stalled")
	}

	report, _ := reports.GetReport(constants.DefaultScenario)
	if report.Metrics.CorruptedPayloads != 1 {
		t.Errorf("expected 1 corrupted payload counted, got %d", report.Metrics.CorruptedPayloads)
	}
	if report.Metrics.Stalls != 1 {
		t.Errorf("expected 1 stall counted, got %d", report.Metrics.Stalls)
	}
}
