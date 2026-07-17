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

func TestMatchPath(t *testing.T) {
	tests := []struct {
		pattern  string
		path     string
		expected bool
		score    int
	}{
		{"/api/checkout", "/api/checkout", true, 1013},
		{"/api/checkout", "/api/products", false, 0},
		{"/api/*", "/api/checkout", true, 5}, // len("/api/") == 5
		{"*", "/api/checkout", true, 0},      // len("") == 0
		{"/api/products/*", "/api/products/123", true, 14}, // len("/api/products/") == 14
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			matched, score := matchPath(tt.pattern, tt.path)
			if matched != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, matched)
			}
			if score != tt.score {
				t.Errorf("expected score %v, got %v", tt.score, score)
			}
		})
	}
}

func TestEngine_evalChaos(t *testing.T) {
	bwGlobal := "100kbps"
	bwOverride := "50kbps"

	cfg := config.Config{
		Bandwidth: bwGlobal,
		Failure: config.FailureConfig{
			Rate: 50,
		},
		Routes: []config.RouteConfig{
			{
				Path: "*",
				Failure: &config.FailureConfig{
					Rate: 10,
				},
			},
			{
				Path: "/api/products/*",
				Failure: &config.FailureConfig{
					Rate: 20,
				},
			},
			{
				Path:   "/api/checkout",
				Method: "POST",
				Failure: &config.FailureConfig{
					Rate: 100, // Very specific
				},
			},
			{
				Path:   "/api/checkout",
				Method: "GET",
				Bandwidth: &bwOverride,
				Failure: &config.FailureConfig{
					Rate: 0, // Specifically disabled
				},
			},
		},
	}
	engine := NewEngine(config.NewState(cfg))

	tests := []struct {
		name         string
		method       string
		path         string
		expectedFail float64
		expectedBw   string
	}{
		{
			name:         "Fallback to lowest wildcard",
			method:       "GET",
			path:         "/other",
			expectedFail: 10,
			expectedBw:   bwGlobal,
		},
		{
			name:         "Match more specific wildcard",
			method:       "GET",
			path:         "/api/products/1",
			expectedFail: 20,
			expectedBw:   bwGlobal,
		},
		{
			name:         "Match exact path and method POST",
			method:       "POST",
			path:         "/api/checkout",
			expectedFail: 100,
			expectedBw:   bwGlobal,
		},
		{
			name:         "Match exact path and method GET",
			method:       "GET",
			path:         "/api/checkout",
			expectedFail: 0,
			expectedBw:   bwOverride,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			eff := engine.evalChaos(cfg, req)

			if eff.Failure.Rate != tt.expectedFail {
				t.Errorf("expected failure rate %v, got %v", tt.expectedFail, eff.Failure.Rate)
			}
			if eff.Bandwidth != tt.expectedBw {
				t.Errorf("expected bandwidth %v, got %v", tt.expectedBw, eff.Bandwidth)
			}
		})
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
