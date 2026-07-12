package chaos

import (
	"math/rand/v2"

	"github.com/st1lson/glitch/internal/config"
)

// ShouldFail determines whether to inject a failure for the current request.
// It returns whether to fail and the HTTP status code to use.
//
// Logic:
//  1. Check each specific StatusConfig — if its random roll hits, return that code.
//  2. If no specific status triggered but a general Rate is configured,
//     roll against that rate and return 500 on hit.
//  3. Otherwise, return false.
func ShouldFail(cfg config.FailureConfig) (bool, int) {
	// Phase 1: Check per-status-code failure rates.
	for _, sc := range cfg.Statuses {
		if sc.Rate > 0 && rand.Float64() < (sc.Rate/100.0) {
			return true, sc.Code
		}
	}

	// Phase 2: Check general failure rate.
	if cfg.Rate > 0 && rand.Float64() < (cfg.Rate/100.0) {
		return true, 500
	}

	return false, 0
}
