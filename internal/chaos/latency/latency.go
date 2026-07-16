package latency

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/st1lson/glitch/internal/config"
)

// InjectLatency computes the delay duration based on the configured mode, sleeps for
// that duration (respecting context cancellation), and returns the actual time slept.
func Inject(ctx context.Context, cfg config.LatencyConfig) time.Duration {
	delay := computeDelay(cfg)
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
func computeDelay(cfg config.LatencyConfig) time.Duration {
	if cfg.Fixed.Duration > 0 {
		return cfg.Fixed.Duration
	}

	// If no fixed or range is provided, return 0
	if cfg.Min.Duration <= 0 && cfg.Max.Duration <= 0 {
		return 0
	}

	minF := float64(cfg.Min.Duration)
	maxF := float64(cfg.Max.Duration)

	var delay time.Duration
	if cfg.Distribution == "normal" {
		// Use a normal distribution centered between min and max
		mean := (minF + maxF) / 2
		stdDev := (maxF - minF) / 6 // 99.7% of values fall within 3 stdDevs
		val := rand.NormFloat64()*stdDev + mean

		// Clamp the result to [min, max]
		if val < minF {
			val = minF
		} else if val > maxF {
			val = maxF
		}
		delay = time.Duration(val)
	} else {
		// Default to uniform distribution
		diff := cfg.Max.Duration - cfg.Min.Duration
		if diff <= 0 {
			return cfg.Min.Duration
		}
		delay = cfg.Min.Duration + time.Duration(rand.Int64N(int64(diff)))
	}

	return delay
}
