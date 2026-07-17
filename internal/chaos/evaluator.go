package chaos

import (
	"context"
	"net/http"
	"strings"

	"github.com/st1lson/glitch/internal/config"
)

// effectiveChaosKey is used to store the evaluated EffectiveChaos in the request context.
type effectiveChaosKey struct{}

// getEffectiveChaos retrieves the EffectiveChaos from the request context.
func getEffectiveChaos(ctx context.Context) (EffectiveChaos, bool) {
	eff, ok := ctx.Value(effectiveChaosKey{}).(EffectiveChaos)
	return eff, ok
}

// setEffectiveChaos stores the EffectiveChaos in the request context.
func setEffectiveChaos(ctx context.Context, eff EffectiveChaos) context.Context {
	return context.WithValue(ctx, effectiveChaosKey{}, eff)
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
func evalChaos(cfg config.Config, r *http.Request) EffectiveChaos {
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
