package chaos

import (
	"context"
	"net/http"
	"slices"
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
	Corrupted    bool
}

func GetChaosInfo(r *http.Request) *ChaosInfo {
	if info, ok := r.Context().Value(chaosContextKey{}).(*ChaosInfo); ok {
		return info
	}
	return nil
}



// Engine is the central chaos-engineering component that orchestrates
// latency injection and failure injection.
type Engine struct {
	state *config.State
	chain []func(http.Handler) http.Handler
}

// NewEngine constructs a chaos Engine from the application config state.
func NewEngine(state *config.State) *Engine {
	return &Engine{
		state: state,
		chain: []func(http.Handler) http.Handler{
			BandwidthMiddleware(),
			LatencyMiddleware(),
			FailureMiddleware(),
			StallMiddleware(),
			CorruptionMiddleware(),
			RealtimeMiddleware(),
		},
	}
}

// Middleware returns an http.Handler middleware that applies chaos injection.
func (e *Engine) Middleware(next http.Handler) http.Handler {
	chaosChain := next
	for _, v := range slices.Backward(e.chain) {
		chaosChain = v(chaosChain)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read current configuration safely.
		cfg := e.state.Get()

		// Fast path: nothing enabled, skip all overhead.
		if !cfg.HasChaos() {
			next.ServeHTTP(w, r)
			return
		}

		eff := evalChaos(cfg, r)

		// Secondary fast path: if specific route overrides disabled all chaos.
		if !eff.Latency.Enabled() && !eff.Failure.Enabled() && eff.Bandwidth.BytesPerSecond == 0 && !eff.Corruption.Enabled() && !eff.Stall.Enabled() && !eff.Realtime.Enabled() {
			next.ServeHTTP(w, r)
			return
		}

		// Inject effective chaos into context
		ctx := setEffectiveChaos(r.Context(), eff)

		// Setup chaos info for this request
		info := &ChaosInfo{}
		ctx = context.WithValue(ctx, chaosContextKey{}, info)

		// Pass down the chain
		chaosChain.ServeHTTP(w, r.WithContext(ctx))
	})
}
