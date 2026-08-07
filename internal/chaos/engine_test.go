package chaos

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/st1lson/glitch/internal/config"
)

func TestEngine_Middleware_NoChaos(t *testing.T) {

	cfg := config.Config{}
	engine := NewEngine(config.NewManager(cfg))

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

	cfg := config.Config{
		Failure: config.FailureConfig{
			Rate: 100,
		},
	}
	engine := NewEngine(config.NewManager(cfg))

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
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected default 500 status, got %d", rr.Code)
	}

	info := GetChaosInfo(req)

	_ = info
}

func TestEngine_Middleware_Latency(t *testing.T) {

	cfg := config.Config{
		Latency: config.LatencyConfig{
			Fixed: config.Duration{Duration: 50 * time.Millisecond},
		},
	}
	engine := NewEngine(config.NewManager(cfg))

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
	engine := NewEngine(config.NewManager(cfg))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := engine.Middleware(next)

	req1 := httptest.NewRequest("GET", "/stable", nil)
	rr1 := httptest.NewRecorder()
	mw.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Errorf("expected global route /stable to return 200 OK, got %d", rr1.Code)
	}

	req2 := httptest.NewRequest("GET", "/fail", nil)
	rr2 := httptest.NewRecorder()
	mw.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusInternalServerError {
		t.Errorf("expected specific route /fail to return 500, got %d", rr2.Code)
	}
}

func TestEngine_Middleware_Seed(t *testing.T) {
	seed := int64(12345)
	cfg := config.Config{
		Seed: &seed,
		Failure: config.FailureConfig{
			Rate: 50,
		},
	}

	engine1 := NewEngine(config.NewManager(cfg))
	engine2 := NewEngine(config.NewManager(cfg))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw1 := engine1.Middleware(next)
	mw2 := engine2.Middleware(next)

	for i := 0; i < 20; i++ {
		req1 := httptest.NewRequest("GET", "/", nil)
		rr1 := httptest.NewRecorder()
		mw1.ServeHTTP(rr1, req1)

		req2 := httptest.NewRequest("GET", "/", nil)
		rr2 := httptest.NewRecorder()
		mw2.ServeHTTP(rr2, req2)

		if rr1.Code != rr2.Code {
			t.Fatalf("divergence at request %d: engine1 code %d, engine2 code %d", i, rr1.Code, rr2.Code)
		}
	}
}
