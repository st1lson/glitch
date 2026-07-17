package logging

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/st1lson/glitch/internal/chaos"
	"github.com/st1lson/glitch/internal/config"
)

type mockReporter struct {
	lastEvent LogEvent
}

func (m *mockReporter) Report(event LogEvent) {
	m.lastEvent = event
}

func TestRequestLogger(t *testing.T) {
	state := config.NewState(config.Config{})
	reporter := &mockReporter{}

	mw := RequestLogger(state, reporter)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest("POST", "/api/test", bytes.NewBuffer([]byte(`{"test":1}`)))
	req.Header.Set("Content-Type", "application/json")
	
	// To inject chaos info without exposing setChaosInfo from chaos pkg, 
	// actually the chaos info is passed through chaos package middleware.
	// But we can test buildChaosAnnotations directly.

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if reporter.lastEvent.StatusCode != http.StatusCreated {
		t.Errorf("expected 201 Created, got %d", reporter.lastEvent.StatusCode)
	}
	if reporter.lastEvent.Method != "POST" {
		t.Errorf("expected POST, got %s", reporter.lastEvent.Method)
	}
}

func TestRequestLogger_Verbose(t *testing.T) {
	state := config.NewState(config.Config{Verbose: true})

	mw := RequestLogger(state, nil) // no reporter, prints to stdout
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest("GET", "/verbose", bytes.NewBuffer([]byte(`{"test":1}`)))
	req.Header.Set("X-Test", "1")
	rr := httptest.NewRecorder()
	
	// This will just print to stdout, we just ensure it doesn't panic
	handler.ServeHTTP(rr, req)
}

func TestColorMethod(t *testing.T) {
	tests := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, "UNKNOWN"}
	for _, m := range tests {
		colorMethod(m) // just checking it runs without panic
	}
}

func TestColorStatus(t *testing.T) {
	tests := []int{200, 404, 500, 100}
	for _, s := range tests {
		colorStatus(s) // just checking it runs without panic
	}
}

func TestFormatDuration(t *testing.T) {
	formatDuration(500 * time.Microsecond)
	formatDuration(500 * time.Millisecond)
	formatDuration(2 * time.Second)
}

func TestBuildChaosAnnotations(t *testing.T) {
	info := &chaos.ChaosInfo{
		LatencyAdded: 100 * time.Millisecond,
		FailureCode:  503,
		Corrupted:    true,
	}
	ann := buildChaosAnnotations(info)
	if ann == "" {
		t.Errorf("expected annotations")
	}
}

func TestResponseWriter_Unwrap(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr}
	if rw.Unwrap() != rr {
		t.Errorf("expected unwrapped writer to match")
	}
}
