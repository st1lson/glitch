package chaos

import (
	"context"
	"math"
	"math/rand/v2"
	"time"

	"github.com/st1lson/glitch/internal/config"
)

// LatencyInjector adds artificial latency to requests based on the configured
// latency profile (fixed, uniform range, or normal distribution).
type LatencyInjector struct {
	cfg config.LatencyConfig
}

// NewLatencyInjector creates a LatencyInjector from the given config.
func NewLatencyInjector(cfg config.LatencyConfig) *LatencyInjector {
	return &LatencyInjector{cfg: cfg}
}

// Inject computes the delay duration based on the configured mode, sleeps for
// that duration (respecting context cancellation), and returns the actual time slept.
func (l *LatencyInjector) Inject(ctx context.Context) time.Duration {
	delay := l.computeDelay()
	if delay <= 0 {
		return 0
	}

	start := time.Now()

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		// Full delay elapsed.
	case <-ctx.Done():
		// Context was cancelled; return however long we actually waited.
	}

	return time.Since(start)
}

// computeDelay returns the target delay duration based on the configured mode.
func (l *LatencyInjector) computeDelay() time.Duration {
	cfg := l.cfg

	// Mode 1: Fixed latency.
	if cfg.Fixed > 0 {
		return cfg.Fixed
	}

	// Range-based modes require both Min and Max.
	if cfg.Min <= 0 || cfg.Max <= 0 {
		return 0
	}

	minNs := float64(cfg.Min)
	maxNs := float64(cfg.Max)

	// Mode 2: Normal distribution.
	if cfg.Distribution == "normal" {
		mean := (minNs + maxNs) / 2
		stddev := (maxNs - minNs) / 4

		sample := rand.NormFloat64()*stddev + mean

		// Clamp to [0, Max*2] to prevent extreme outliers.
		sample = math.Max(0, math.Min(sample, float64(cfg.Max)*2))

		return time.Duration(sample)
	}

	// Mode 3 (default): Uniform random in [Min, Max].
	return time.Duration(minNs + rand.Float64()*(maxNs-minNs))
}
