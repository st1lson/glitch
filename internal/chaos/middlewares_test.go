package chaos

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/st1lson/glitch/internal/config"
)

func TestBandwidthMiddleware(t *testing.T) {
	mw := BandwidthMiddleware()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Case 1: Disabled
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Case 2: Enabled
	eff := EffectiveChaos{Bandwidth: config.Bandwidth{StringValue: "100kbps", BytesPerSecond: 102400}}
	ctx := setEffectiveChaos(req.Context(), eff)
	req2 := req.WithContext(ctx)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
}

func TestStallMiddleware(t *testing.T) {
	mw := StallMiddleware()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Case 1: Enabled
	eff := EffectiveChaos{Stall: config.StallConfig{Rate: 100, Mode: "drop", DropAt: 10}}
	req := httptest.NewRequest("GET", "/", nil)
	ctx := setEffectiveChaos(req.Context(), eff)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req.WithContext(ctx))
}

func TestCorruptionMiddleware(t *testing.T) {
	mw := CorruptionMiddleware()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello world"))
	}))

	// Case 1: Enabled
	eff := EffectiveChaos{Corruption: config.CorruptionConfig{Rate: 100, Strategies: []config.CorruptionStrategy{config.StrategyDropField}}}
	req := httptest.NewRequest("GET", "/", nil)
	ctx := setEffectiveChaos(req.Context(), eff)
	
	// Set ChaosInfo to test coverage of info injection
	info := &ChaosInfo{}
	ctx = context.WithValue(ctx, chaosContextKey{}, info)
	
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req.WithContext(ctx))
	
	if !info.Corrupted {
		t.Errorf("expected info.Corrupted to be true")
	}
}

func TestLatencyMiddleware_WithInfo(t *testing.T) {
	mw := LatencyMiddleware()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Case 1: Enabled
	eff := EffectiveChaos{Latency: config.LatencyConfig{Fixed: config.Duration{Duration: 10 * time.Millisecond}}}
	req := httptest.NewRequest("GET", "/", nil)
	ctx := setEffectiveChaos(req.Context(), eff)
	
	// Set ChaosInfo to test coverage of info injection
	info := &ChaosInfo{}
	ctx = context.WithValue(ctx, chaosContextKey{}, info)
	
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req.WithContext(ctx))
	
	if info.LatencyAdded < 10*time.Millisecond {
		t.Errorf("expected info.LatencyAdded to be >= 10ms")
	}
}

func TestFailureMiddleware_WithInfo(t *testing.T) {
	mw := FailureMiddleware()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Case 1: Enabled
	eff := EffectiveChaos{Failure: config.FailureConfig{Rate: 100}}
	req := httptest.NewRequest("GET", "/", nil)
	ctx := setEffectiveChaos(req.Context(), eff)
	
	// Set ChaosInfo to test coverage of info injection
	info := &ChaosInfo{}
	ctx = context.WithValue(ctx, chaosContextKey{}, info)
	
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req.WithContext(ctx))
	
	if info.FailureCode == 0 {
		t.Errorf("expected info.FailureCode to be set")
	}
}
