package chaos

import (
	"context"
	"math/rand/v2"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/st1lson/glitch/internal/chaos/rng"
	"github.com/st1lson/glitch/internal/config"
	"github.com/st1lson/glitch/internal/constants"
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

type scenarioRNG struct {
	seed int64
	r    *rand.Rand
}

// Engine is the central chaos-engineering component that orchestrates
// latency injection and failure injection.
type Engine struct {
	state *config.Manager
	chain []func(http.Handler) http.Handler

	rngMu sync.RWMutex
	rngs  map[string]*scenarioRNG
}

// NewEngine constructs a chaos Engine from the application config state.
func NewEngine(state *config.Manager) *Engine {
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
		rngs: make(map[string]*scenarioRNG),
	}
}

func (e *Engine) getRNG(scenario string, seed int64) *rand.Rand {
	e.rngMu.RLock()
	entry, ok := e.rngs[scenario]
	e.rngMu.RUnlock()

	if ok && entry.seed == seed {
		return entry.r
	}

	e.rngMu.Lock()
	defer e.rngMu.Unlock()

	// Double check after acquiring write lock
	if entry, ok := e.rngs[scenario]; ok && entry.seed == seed {
		return entry.r
	}

	r := rng.New(seed)
	e.rngs[scenario] = &scenarioRNG{seed: seed, r: r}
	return r
}

// Middleware returns an http.Handler middleware that applies chaos injection.
func (e *Engine) Middleware(next http.Handler) http.Handler {
	chaosChain := next
	for _, v := range slices.Backward(e.chain) {
		chaosChain = v(chaosChain)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read current configuration safely.
		scenario := r.Header.Get(constants.HeaderScenario)
		cfg := e.state.Resolve(scenario)

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

		// Setup RNG for this request
		if cfg.Seed != nil {
			// Map unknown scenarios to default cache key to prevent unbounded map growth (DoS)
			cacheKey := scenario
			if scenario != "" && !e.state.Has(scenario) {
				cacheKey = ""
			}
			scenarioRNG := e.getRNG(cacheKey, *cfg.Seed)
			ctx = rng.WithRNG(ctx, scenarioRNG)
		}

		// Setup chaos info for this request
		info := &ChaosInfo{}
		ctx = context.WithValue(ctx, chaosContextKey{}, info)

		// Pass down the chain
		chaosChain.ServeHTTP(w, r.WithContext(ctx))
	})
}
