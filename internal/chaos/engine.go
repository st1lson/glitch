package chaos

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/st1lson/glitch/internal/chaos/corruption"
	"github.com/st1lson/glitch/internal/chaos/failure"
	"github.com/st1lson/glitch/internal/chaos/latency"
	"github.com/st1lson/glitch/internal/chaos/stall"
	"github.com/st1lson/glitch/internal/chaos/throttle"
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
	Corrupted    bool
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
		if !cfg.Latency.Enabled() && !cfg.Failure.Enabled() && cfg.Bandwidth == "" && !cfg.Corruption.Enabled() {
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

		// Phase 0: Bandwidth Throttling (wrapping the ResponseWriter)
		if cfg.BandwidthBps > 0 {
			rw = throttle.NewWriter(rw, cfg.BandwidthBps)
		}

		// Phase 1: Latency Injection
		if cfg.Latency.Enabled() {
			if applied, delay := latency.Inject(cfg.Latency); applied {
				info.LatencyAdded = delay
			}
		}

		// Phase 2: Failure Injection
		if cfg.Failure.Enabled() {
			if fail, code := failure.ShouldTrigger(cfg.Failure); fail {
				info.FailureCode = code
				http.Error(rw, http.StatusText(code), code)
				return
			}
		}

		// Phase 3: Stall Injection. Wrap the ResponseWriter.
		if cfg.Stall.Enabled() && stall.ShouldTrigger(cfg.Stall) {
			rw = stall.NewWriter(rw, cfg.Stall.Mode, cfg.Stall.DropAt)
		}

		// Phase 4: Payload corruption. Wrap the ResponseWriter to buffer and mutate.
		var cw *corruption.Writer
		if cfg.Corruption.Enabled() && corruption.ShouldTrigger(cfg.Corruption) {
			cw = corruption.NewWriter(rw, cfg.Corruption)
			rw = cw
			info.Corrupted = true
		}

		// Attach chaos info for downstream consumers (e.g., the logger).
		r = setChaosInfo(r, info)
		next.ServeHTTP(rw, r)

		// Flush corruption buffer if active
		if cw != nil {
			cw.Flush()
		}
	})
}
