package rng

import (
	"context"
	"math/rand/v2"
	"sync"
)

// lockedSource wraps a rand.Source to make it safe for concurrent use.
// math/rand/v2's rand.Rand is not thread-safe by itself.
type lockedSource struct {
	mu  sync.Mutex
	src rand.Source
}

func (l *lockedSource) Uint64() uint64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.src.Uint64()
}

type contextKey struct{}

// WithRNG injects a rand.Rand into the context.
func WithRNG(ctx context.Context, r *rand.Rand) context.Context {
	return context.WithValue(ctx, contextKey{}, r)
}

// globalRand is the fallback global random generator for unseeded chaos.
var globalRand = rand.New(&lockedSource{src: rand.NewPCG(rand.Uint64(), rand.Uint64())})

// FromContext retrieves the rand.Rand from the context, or returns the thread-safe global one if none is set.
func FromContext(ctx context.Context) *rand.Rand {
	if r, ok := ctx.Value(contextKey{}).(*rand.Rand); ok {
		return r
	}
	return globalRand
}

// New creates a new thread-safe *rand.Rand initialized with the given seed.
func New(seed int64) *rand.Rand {

	src := rand.NewPCG(uint64(seed), uint64(seed)+1)
	return rand.New(&lockedSource{src: src})
}
