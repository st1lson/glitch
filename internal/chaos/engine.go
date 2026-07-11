package chaos

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/st1lson/glitch/internal/config"
)

// chaosContextKey is an unexported type used as the context key for ChaosInfo,
// avoiding collisions with other packages.
type chaosContextKey struct{}

// ChaosInfo carries information about chaos injected into a request,
// making it available to downstream handlers and the logger.
type ChaosInfo struct {
	LatencyAdded time.Duration
	FailureCode  int
}

// GetChaosInfo retrieves the ChaosInfo attached to the request, or nil if none.
func GetChaosInfo(r *http.Request) *ChaosInfo {
	if info, ok := r.Context().Value(chaosContextKey{}).(*ChaosInfo); ok {
		return info
	}
	return nil
}

// setChaosInfo returns a new request with ChaosInfo stored in its context.
func setChaosInfo(r *http.Request, info *ChaosInfo) *http.Request {
	ctx := context.WithValue(r.Context(), chaosContextKey{}, info)
	return r.WithContext(ctx)
}

// Engine is the central chaos-engineering component that orchestrates
// latency injection and failure injection.
type Engine struct {
	latency *LatencyInjector
	failure *FailureInjector
}

// NewEngine constructs a chaos Engine from the application config.
// Injectors are only created when their respective configs are enabled.
func NewEngine(cfg config.Config) *Engine {
	e := &Engine{}

	if cfg.Latency.Enabled() {
		e.latency = NewLatencyInjector(cfg.Latency)
	}

	if cfg.Failure.Enabled() {
		e.failure = NewFailureInjector(cfg.Failure)
	}

	return e
}

// Middleware returns an http.Handler middleware that applies chaos injection.
// Order: latency first (sleep), then failure check (may short-circuit), then next.
// If neither injector is configured, it's a zero-cost passthrough.
func (e *Engine) Middleware(next http.Handler) http.Handler {
	// Fast path: nothing enabled, skip all overhead.
	if e.latency == nil && e.failure == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := &ChaosInfo{}

		// Phase 1: Latency injection.
		if e.latency != nil {
			info.LatencyAdded = e.latency.Inject(r.Context())
		}

		// Phase 2: Failure injection — may short-circuit the request.
		if e.failure != nil {
			if shouldFail, code := e.failure.ShouldFail(); shouldFail {
				info.FailureCode = code
				r = setChaosInfo(r, info)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(code)
				// Use a raw write to avoid importing encoding/json just for this.
				fmt.Fprintf(w, `{"error":"Injected by Glitch","status":%d}`, code)
				return
			}
		}

		// Attach chaos info for downstream consumers (e.g., the logger).
		r = setChaosInfo(r, info)
		next.ServeHTTP(w, r)
	})
}
