package monkey

import (
	"context"
	"time"

	"github.com/st1lson/glitch/internal/config"
)

// Run starts the continuous chaos monkey loop.
// It applies the configured phases sequentially, overriding the main config's
// chaos settings. The loop continues until the context is canceled.
func Run(ctx context.Context, state *config.State) {
	for {
		cfg := state.Get()
		if !cfg.Monkey.Enabled || len(cfg.Monkey.Phases) == 0 {
			// If disabled or misconfigured, wait and check again.
			// This allows monkey mode to be toggled dynamically.
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}

		for _, phase := range cfg.Monkey.Phases {
			// Apply the current phase's chaos configuration
			state.Update(func(c *config.Config) {
				c.Bandwidth = phase.Bandwidth
				c.Latency = phase.Latency
				c.Failure = phase.Failure
				c.Stall = phase.Stall
				c.Corruption = phase.Corruption
			})

			duration := phase.Duration.Duration
			if duration <= 0 {
				duration = 10 * time.Second // Fallback duration if not specified
			}

			// Wait for the duration of the phase or context cancellation
			select {
			case <-ctx.Done():
				return
			case <-time.After(duration):
				// Phase duration elapsed, proceed to next phase
			}

			// Re-check if monkey was disabled during the sleep
			if !state.Get().Monkey.Enabled {
				break // Exit the phases loop, go back to the outer loop to wait
			}
		}
	}
}
