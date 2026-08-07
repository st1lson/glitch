package control

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/st1lson/glitch/internal/constants"
)

// Gatekeeper manages the paused state for requests, scoped by scenario.
type Gatekeeper struct {
	mu     sync.Mutex
	states map[string]*pauseState
}

type pauseState struct {
	gate  chan struct{}
	timer *time.Timer
}

// NewGatekeeper initializes a new Gatekeeper.
func NewGatekeeper() *Gatekeeper {
	return &Gatekeeper{
		states: make(map[string]*pauseState),
	}
}

// Pause blocks all incoming requests for the given scenario.
// If timeout is > 0, it automatically resumes after the duration.
func (g *Gatekeeper) Pause(scenario string, timeout time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()

	state, ok := g.states[scenario]
	if !ok || state.gate == nil {

		state = &pauseState{
			gate: make(chan struct{}),
		}
		g.states[scenario] = state
	} else if state.timer != nil {

		state.timer.Stop()
		state.timer = nil
	}

	if timeout > 0 {
		state.timer = time.AfterFunc(timeout, func() {
			g.Resume(scenario)
		})
	}
}

// Resume unblocks all pending and future requests for the given scenario.
func (g *Gatekeeper) Resume(scenario string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if state, ok := g.states[scenario]; ok {
		if state.timer != nil {
			state.timer.Stop()
			state.timer = nil
		}
		if state.gate != nil {
			close(state.gate)
			state.gate = nil
		}
		delete(g.states, scenario)
	}
}

// Wait blocks if the scenario is paused, until it's resumed or the context is canceled.
func (g *Gatekeeper) Wait(ctx context.Context, scenario string) error {
	g.mu.Lock()
	state, ok := g.states[scenario]
	var gate chan struct{}
	if ok {
		gate = state.gate
	}
	g.mu.Unlock()

	if gate == nil {
		return nil
	}

	select {
	case <-gate:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// IsPaused returns whether the scenario is currently paused.
func (g *Gatekeeper) IsPaused(scenario string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	state, ok := g.states[scenario]
	return ok && state.gate != nil
}

// PauseMiddleware returns a middleware that blocks requests if their scenario is paused.
func PauseMiddleware(gate *Gatekeeper) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scenario := r.Header.Get(constants.HeaderScenario)
			if err := gate.Wait(r.Context(), scenario); err != nil {
				http.Error(w, "request cancelled while paused", http.StatusGatewayTimeout)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
