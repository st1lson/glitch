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
	state *config.State
}

// NewEngine constructs a chaos Engine from the application config state.
func NewEngine(state *config.State) *Engine {
	return &Engine{
		state: state,
	}
}

// Middleware returns an http.Handler middleware that applies chaos injection.
// Order: latency first (sleep), then failure check (may short-circuit), then next.
func (e *Engine) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read current configuration safely.
		cfg := e.state.Get()

		// Fast path: nothing enabled, skip all overhead.
		if !cfg.Latency.Enabled() && !cfg.Failure.Enabled() && cfg.Bandwidth == "" {
			next.ServeHTTP(w, r)
			return
		}

		info := &ChaosInfo{}

		// Wrap ResponseWriter for bandwidth throttling if configured
		var rw http.ResponseWriter = w
		if cfg.Bandwidth != "" {
			if bps, err := config.ParseBandwidth(cfg.Bandwidth); err == nil && bps > 0 {
				rw = newThrottledWriter(w, bps)
			}
		}

		// Phase 1: Latency injection.
		if cfg.Latency.Enabled() {
			info.LatencyAdded = InjectLatency(r.Context(), cfg.Latency)
		}

		// Phase 2: Failure injection — may short-circuit the request.
		if cfg.Failure.Enabled() {
			if fail, code := ShouldFail(cfg.Failure); fail {
				info.FailureCode = code
				r = setChaosInfo(r, info)

				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(code)
				// Use a raw write to avoid importing encoding/json just for this.
				fmt.Fprintf(rw, `{"error":"Injected by Glitch","status":%d}`, code)
				return
			}
		}

		// Attach chaos info for downstream consumers (e.g., the logger).
		r = setChaosInfo(r, info)
		next.ServeHTTP(rw, r)
	})
}
