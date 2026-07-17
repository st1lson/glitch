package chaos

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/st1lson/glitch/internal/config"
)

func TestEngine_Middleware_NoChaos(t *testing.T) {
	// Empty config means no chaos
	cfg := config.Config{}
	engine := NewEngine(config.NewState(cfg))

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	mw := engine.Middleware(next)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	mw.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Error("expected next handler to be called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rr.Code)
	}
}

func TestEngine_Middleware_Failure(t *testing.T) {
	// 100% failure rate
	cfg := config.Config{
		Failure: config.FailureConfig{
			Rate: 100,
		},
	}
	engine := NewEngine(config.NewState(cfg))

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	mw := engine.Middleware(next)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	mw.ServeHTTP(rr, req)

	if handlerCalled {
		t.Error("expected next handler NOT to be called due to 100% failure rate")
	}
	if rr.Code != http.StatusInternalServerError { // default failure code
		t.Errorf("expected default 500 status, got %d", rr.Code)
	}

	info := GetChaosInfo(req)
	// info will be attached to the request passed to next, but since we short-circuit,
	// we'd need to extract it from a logger or something. This is fine.
	_ = info
}

func TestEngine_Middleware_Latency(t *testing.T) {
	// Fixed latency of 50ms
	cfg := config.Config{
		Latency: config.LatencyConfig{
			Fixed: config.Duration{Duration: 50 * time.Millisecond},
		},
	}
	engine := NewEngine(config.NewState(cfg))

	var ctxInfo *ChaosInfo
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxInfo = GetChaosInfo(r)
	})

	mw := engine.Middleware(next)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	start := time.Now()
	mw.ServeHTTP(rr, req)
	duration := time.Since(start)

	if duration < 50*time.Millisecond {
		t.Errorf("expected latency injection to delay at least 50ms, got %v", duration)
	}

	if ctxInfo == nil {
		t.Fatal("expected ChaosInfo to be injected into context")
	}
	if ctxInfo.LatencyAdded < 50*time.Millisecond {
		t.Errorf("expected LatencyAdded to be tracked in context")
	}
}



func TestEngine_Middleware_Routes(t *testing.T) {
	// Global failure is 0, but /fail fails 100% of the time.
	cfg := config.Config{
		Failure: config.FailureConfig{
			Rate: 0,
		},
		Routes: []config.RouteConfig{
			{
				Path: "/fail",
				Failure: &config.FailureConfig{
					Rate: 100,
				},
			},
		},
	}
	engine := NewEngine(config.NewState(cfg))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := engine.Middleware(next)

	// 1. Test the global stable route
	req1 := httptest.NewRequest("GET", "/stable", nil)
	rr1 := httptest.NewRecorder()
	mw.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Errorf("expected global route /stable to return 200 OK, got %d", rr1.Code)
	}

	// 2. Test the specific failing route
	req2 := httptest.NewRequest("GET", "/fail", nil)
	rr2 := httptest.NewRecorder()
	mw.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusInternalServerError { // default failure code
		t.Errorf("expected specific route /fail to return 500, got %d", rr2.Code)
	}
}
