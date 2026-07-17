package chaos

import (
	"context"
	"net/http"
	"strings"
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

// EffectiveChaos represents the final chaos settings for a specific request.
type EffectiveChaos struct {
	Bandwidth  string
	Latency    config.LatencyConfig
	Failure    config.FailureConfig
	Stall      config.StallConfig
	Corruption config.CorruptionConfig
}

// evalChaos overlays route-specific chaos on top of global chaos, selecting the most specific match.
func (e *Engine) evalChaos(cfg config.Config, r *http.Request) EffectiveChaos {
	eff := EffectiveChaos{
		Bandwidth:  cfg.Bandwidth,
		Latency:    cfg.Latency,
		Failure:    cfg.Failure,
		Stall:      cfg.Stall,
		Corruption: cfg.Corruption,
	}

	if len(cfg.Routes) == 0 {
		return eff
	}

	var bestMatch *config.RouteConfig
	bestScore := -1

	for i := range cfg.Routes {
		route := &cfg.Routes[i]

		// Method check
		if route.Method != "" && !strings.EqualFold(route.Method, r.Method) {
			continue
		}

		matched, score := matchPath(route.Path, r.URL.Path)
		if !matched {
			continue
		}

		if route.Method != "" {
			score += 100 // Method match increases specificity
		}

		if score > bestScore {
			bestScore = score
			bestMatch = route
		}
	}

	if bestMatch != nil {
		if bestMatch.Bandwidth != nil {
			eff.Bandwidth = *bestMatch.Bandwidth
		}
		if bestMatch.Latency != nil {
			eff.Latency = *bestMatch.Latency
		}
		if bestMatch.Failure != nil {
			eff.Failure = *bestMatch.Failure
		}
		if bestMatch.Stall != nil {
			eff.Stall = *bestMatch.Stall
		}
		if bestMatch.Corruption != nil {
			eff.Corruption = *bestMatch.Corruption
		}
	}

	return eff
}

// matchPath checks if a pattern matches a path and returns a specificity score.
// Exact matches get +1000 score. Prefix matches (ending in *) get score based on prefix length.
func matchPath(pattern, path string) (bool, int) {
	if pattern == path {
		return true, 1000 + len(pattern)
	}
	if before, ok := strings.CutSuffix(pattern, "*"); ok {
		prefix := before
		if strings.HasPrefix(path, prefix) {
			return true, len(prefix)
		}
	}
	return false, 0
}

// Middleware returns an http.Handler middleware that applies chaos injection.
// Order: latency first (sleep), then failure check (may short-circuit), then next.
func (e *Engine) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read current configuration safely.
		cfg := e.state.Get()

		// Fast path: nothing enabled, skip all overhead.
		if !cfg.HasChaos() {
			next.ServeHTTP(w, r)
			return
		}

		eff := e.evalChaos(cfg, r)

		// Secondary fast path: if specific route overrides disabled all chaos.
		if !eff.Latency.Enabled() && !eff.Failure.Enabled() && eff.Bandwidth == "" && !eff.Corruption.Enabled() && !eff.Stall.Enabled() {
			next.ServeHTTP(w, r)
			return
		}

		info := &ChaosInfo{}

		// Wrap ResponseWriter for bandwidth throttling if configured
		var rw http.ResponseWriter = w
		if eff.Bandwidth != "" {
			if bps, err := config.ParseBandwidth(eff.Bandwidth); err == nil && bps > 0 {
				rw = throttle.NewWriter(w, bps)
			}
		}

		// Phase 1: Latency Injection
		if eff.Latency.Enabled() {
			info.LatencyAdded = latency.Inject(r.Context(), eff.Latency)
		}

		// Phase 2: Failure Injection
		if eff.Failure.Enabled() {
			if fail, code := failure.ShouldTrigger(eff.Failure); fail {
				info.FailureCode = code
				http.Error(rw, http.StatusText(code), code)
				return
			}
		}

		// Phase 3: Stall Injection. Wrap the ResponseWriter.
		if eff.Stall.Enabled() && stall.ShouldTrigger(eff.Stall) {
			rw = stall.NewWriter(rw, eff.Stall.Mode, eff.Stall.DropAt)
		}

		// Phase 4: Payload corruption. Wrap the ResponseWriter to buffer and mutate.
		var cw *corruption.Writer
		if eff.Corruption.Enabled() && corruption.ShouldTrigger(eff.Corruption) {
			cw = corruption.NewWriter(rw, eff.Corruption)
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
